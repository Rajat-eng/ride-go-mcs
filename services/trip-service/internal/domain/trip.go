package domain

import (
	"context"
	tripTypes "ride-sharing/services/trip-service/pkg/types"
	"ride-sharing/shared/types"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TripModel struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	UserID   string             // user who created the trip
	From     string
	To       string
	Status   string
	RideFare *RideFareModel
	Driver   *TripDriver //*pb.TripDriver
}

type TripDriver struct {
	id             string
	name           string
	profilePicture string
	carPlate       string
}

type TripRepository interface {
	CreateTrip(ctx context.Context, trip *TripModel) (*TripModel, error)
	SaveRideFare(ctx context.Context, f *RideFareModel) error
	GetRideFareByID(ctx context.Context, id string) (*RideFareModel, error)
}
type TripService interface {
	CreateTrip(ctx context.Context, fare *RideFareModel) (*TripModel, error)
	GetRoute(ctx context.Context, pickup, destination *types.Coordinate) (*tripTypes.OSRMApiResponse, error)
	EstimatePackagePriceWithRoute(route *tripTypes.OSRMApiResponse) ([]*RideFareModel, error)
	GenerateTripFares(ctx context.Context, fares []*RideFareModel, userID string, route *tripTypes.OSRMApiResponse) ([]*RideFareModel, error)
	GetAndValidateFare(ctx context.Context, fareID, userID string) (*RideFareModel, error)
}
