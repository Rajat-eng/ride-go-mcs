package grpc

import (
	// "context"
	// "log"
	"context"
	"log"
	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/services/trip-service/internal/infrastructure/events"
	pb "ride-sharing/shared/proto/trip"
	"ride-sharing/shared/types"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	// 	"ride-sharing/shared/types"
	"google.golang.org/grpc"
)

type gRPCHandler struct {
	pb.UnimplementedTripServiceServer // this is for forward compatibility meaning if we add new methods in future it will not break existing implementations
	// we need to implement all methods of TripServiceServer interface
	// so we embed UnimplementedTripServiceServer struct which has default implementations of all methods
	//like CreateTrip and PreviewTrip
	service   domain.TripService // this is our domain service which has business logic
	publisher *events.TripEventPublisher
}

func NewGRPCHandler(server *grpc.Server, service domain.TripService, publisher *events.TripEventPublisher) *gRPCHandler {
	handler := &gRPCHandler{
		service:   service,
		publisher: publisher,
	}

	pb.RegisterTripServiceServer(server, handler) // register our handler with grpc server
	return handler
}

func (h *gRPCHandler) CreateTrip(ctx context.Context, req *pb.CreateTripRequest) (*pb.CreateTripResponse, error) {
	fareID := req.GetRideFareID()
	userID := req.GetUserID()

	rideFare, err := h.service.GetAndValidateFare(ctx, fareID, userID) // get fare from db and validate userID matches
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to validate the fare: %v", err)
	}

	trip, err := h.service.CreateTrip(ctx, rideFare)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create the trip: %v", err)
	}

	if err := h.publisher.PublishTripCreated(ctx, trip); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to publish the trip created event: %v", err)
	}

	return &pb.CreateTripResponse{
		TripID: trip.ID.Hex(),
	}, nil
}

func (h *gRPCHandler) PreviewTrip(ctx context.Context, req *pb.PreviewTripRequest) (*pb.PreviewTripResponse, error) {
	pickup := req.GetStartLocation()
	destination := req.GetEndLocation()

	pickupCoord := &types.Coordinate{
		Latitude:  pickup.Latitude,
		Longitude: pickup.Longitude,
	}
	destinationCoord := &types.Coordinate{
		Latitude:  destination.Latitude,
		Longitude: destination.Longitude,
	}

	userID := req.GetUserID()

	route, err := h.service.GetRoute(ctx, pickupCoord, destinationCoord)
	// it gives osrm response--> need to convert to pb response
	if err != nil {
		log.Println(err)
		return nil, status.Errorf(codes.Internal, "failed to get route: %v", err)
	}

	// estimate ride fares based on the route
	// store the route and ride fares in the response
	estimatedFares, _ := h.service.EstimatePackagePriceWithRoute(route)

	fares, faresErr := h.service.GenerateTripFares(ctx, estimatedFares, userID, route) // store in db and return with trip

	if faresErr != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate the ride fares: %v", err)
	}

	return &pb.PreviewTripResponse{
		Route:     route.ToProto(),
		RideFares: domain.ToRideFaresProto(fares),
	}, nil
}
