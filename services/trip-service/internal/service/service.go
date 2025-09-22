package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"ride-sharing/services/trip-service/internal/domain"
	tripTypes "ride-sharing/services/trip-service/pkg/types"
	"ride-sharing/shared/types"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TripService struct {
	repo domain.TripRepository
}

func NewTripService(repo domain.TripRepository) *TripService {
	return &TripService{repo: repo}
}

// Implement service methods here

func (s *TripService) CreateTrip(ctx context.Context, fare *domain.RideFareModel) (*domain.TripModel, error) {
	t := &domain.TripModel{
		ID:       primitive.NewObjectID(),
		UserID:   fare.UserID,
		From:     "",
		To:       "",
		RideFare: fare, // * gives value stored at address fare
		Status:   "pending",
		Driver:   &domain.TripDriver{},
	}
	return s.repo.CreateTrip(ctx, t)
}

func (s *TripService) GetRoute(ctx context.Context, pickup, destination *types.Coordinate) (*tripTypes.OSRMApiResponse, error) {

	url := fmt.Sprintf("http://router.project-osrm.org/route/v1/driving/%f,%f;%f,%f?overview=full&geometries=geojson",
		pickup.Longitude,
		pickup.Latitude,
		destination.Longitude,
		destination.Latitude,
	)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get route: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body) // read raw bytes
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	var routeResponse tripTypes.OSRMApiResponse
	if err := json.Unmarshal(body, &routeResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	// it gives unmarshalled osrm  --> like Route[Distance,Duration,Geometry[Coordinates]]
	// need to convert to pb response
	//  return routeResponse.ToProto(), nil

	return &routeResponse, nil
}

func (s *TripService) GetAndValidateFare(ctx context.Context, fareID, userID string) (*domain.RideFareModel, error) {
	fare, err := s.repo.GetRideFareByID(ctx, fareID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trip fare: %w", err)
	}

	if fare == nil {
		return nil, fmt.Errorf("fare does not exist")
	}

	// User fare validation (user is owner of this fare?)
	if userID != fare.UserID {
		return nil, fmt.Errorf("fare does not belong to the user")
	}

	return fare, nil
}

func (s *TripService) EstimatePackagePriceWithRoute(route *tripTypes.OSRMApiResponse) ([]*domain.RideFareModel, error) {
	if len(route.Routes) == 0 {
		return nil, fmt.Errorf("no routes found")
	}
	baseFares := getBaseFares()
	estimatedFares := make([]*domain.RideFareModel, len(baseFares))
	for i, f := range baseFares {
		estimatedFares[i] = estimateFareRoute(f, route)
	}
	// estimatedFares is slice of RideFareModel pointers which gives fare for various packages
	return estimatedFares, nil
}

func estimateFareRoute(f *domain.RideFareModel, route *tripTypes.OSRMApiResponse) *domain.RideFareModel {
	pricingCfg := tripTypes.DefaultPricingConfig()
	carPackagePrice := f.TotalPriceInCents

	distanceKm := route.Routes[0].Distance
	durationInMinutes := route.Routes[0].Duration

	distanceFare := distanceKm * pricingCfg.PricePerUnitOfDistance
	timeFare := durationInMinutes * pricingCfg.PricingPerMinute
	totalPrice := carPackagePrice + distanceFare + timeFare

	return &domain.RideFareModel{
		TotalPriceInCents: totalPrice,
		PackageSlug:       f.PackageSlug,
	}
}

func (s *TripService) GenerateTripFares(ctx context.Context, rideFares []*domain.RideFareModel, userID string, route *tripTypes.OSRMApiResponse) ([]*domain.RideFareModel, error) {
	// for each estimate route package fare create a RideFareModel and save to db on preview trip with stored userID
	fares := make([]*domain.RideFareModel, len(rideFares))

	for i, f := range rideFares {
		id := primitive.NewObjectID() // id for RideFareModel

		fare := &domain.RideFareModel{
			UserID:            userID,
			ID:                id,
			TotalPriceInCents: f.TotalPriceInCents,
			PackageSlug:       f.PackageSlug,
			Route:             route,
		}

		if err := s.repo.SaveRideFare(ctx, fare); err != nil {
			return nil, fmt.Errorf("failed to save trip fare: %w", err)
		}

		fares[i] = fare
	}

	return fares, nil
}

func getBaseFares() []*domain.RideFareModel {
	return []*domain.RideFareModel{
		{
			PackageSlug:       "Bike",
			TotalPriceInCents: 100,
		},
		{
			PackageSlug:       "Auto",
			TotalPriceInCents: 150,
		},
		{
			PackageSlug:       "Mini",
			TotalPriceInCents: 250,
		},
		{
			PackageSlug:       "Luxury",
			TotalPriceInCents: 400,
		},
	}
}
