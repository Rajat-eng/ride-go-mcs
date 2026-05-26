package main

import (
	"context"
	pb "ride-sharing/shared/proto/driver"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type driverGrpcHandler struct {
	pb.UnimplementedDriverServiceServer

	service *Service // this is our domain service which has business logic
}

func NewGrpcHandler(s *grpc.Server, service *Service) {
	handler := &driverGrpcHandler{
		service: service,
	}

	pb.RegisterDriverServiceServer(s, handler) // register our handler with grpc server
}

func (h *driverGrpcHandler) UnregisterDriver(ctx context.Context, req *pb.RegisterDriverRequest) (*pb.RegisterDriverResponse, error) {
	h.service.RemoveDriverFromGeo(ctx, req.GetDriverID(), req.GetPackageSlug())

	return &pb.RegisterDriverResponse{
		Driver: &pb.Driver{
			Id: req.GetDriverID(),
		},
	}, nil
}

func (h *driverGrpcHandler) UpdateLocation(ctx context.Context, req *pb.UpdateLocationRequest) (*pb.UpdateLocationResponse, error) {
	loc := req.GetLocation()
	if loc == nil {
		return nil, status.Errorf(codes.InvalidArgument, "location is required")
	}
	if err := h.service.UpdateDriverLocation(req.GetDriverID(), req.GetPackageSlug(), loc.GetLatitude(), loc.GetLongitude()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update driver location: %v", err)
	}
	return &pb.UpdateLocationResponse{}, nil
}
