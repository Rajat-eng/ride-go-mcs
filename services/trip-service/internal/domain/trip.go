package domain

import (
	"context"
	tripTypes "ride-sharing/services/trip-service/pkg/types"
	pb "ride-sharing/shared/proto/trip"
	"ride-sharing/shared/types"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TripModel struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	UserID   string             `bson:"userID"`
	Status   string             `bson:"status"`
	RideFare *RideFareModel     `bson:"rideFare"`
	Driver   *pb.TripDriver     `bson:"driver"`
}

func (t *TripModel) ToProto() *pb.Trip {
	var driver *pb.TripDriver
	if t.Driver != nil && t.Driver.Id != "" && t.Driver.Name != "" {
		driver = t.Driver
	}

	return &pb.Trip{
		Id:           t.ID.Hex(),
		UserID:       t.UserID,
		SelectedFare: t.RideFare.ToProto(),
		Status:       t.Status,
		Driver:       driver,
		Route:        t.RideFare.Route.ToProto(),
	}
}

type TripRepository interface {
	CreateTrip(ctx context.Context, trip *TripModel) (*TripModel, error)
	GetTripByID(ctx context.Context, id string) (*TripModel, error)
	UpdateTrip(ctx context.Context, tripID string, status string, driver *pb.TripDriver) error
}

type RideFareRepository interface {
	SaveFarePreview(ctx context.Context, preview *FarePreview) error
	GetRideFareByID(ctx context.Context, fareID, userID string) (*RideFareModel, error)
}
type TripService interface {
	CreateTrip(ctx context.Context, fare *RideFareModel) (*TripModel, error)
	GetRoute(ctx context.Context, pickup, destination *types.Coordinate) (*tripTypes.OSRMApiResponse, error)
	EstimatePackagePriceWithRoute(route *tripTypes.OSRMApiResponse) ([]*RideFareModel, error)
	GenerateTripFares(ctx context.Context, fares []*RideFareModel, userID string, route *tripTypes.OSRMApiResponse) (*FarePreview, error)
	GetAndValidateFare(ctx context.Context, fareID, userID string) (*RideFareModel, error)
	GetTripByID(ctx context.Context, id string) (*TripModel, error)
	UpdateTrip(ctx context.Context, tripID string, status string, driver *pb.TripDriver) error
	CancelTrip(ctx context.Context, tripID, requesterUserID string) (*TripModel, error)
}
