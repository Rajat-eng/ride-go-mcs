package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"ride-sharing/services/trip-service/internal/domain"
	tripTypes "ride-sharing/services/trip-service/pkg/types"
	pb "ride-sharing/shared/proto/trip"
	"ride-sharing/shared/types"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TripService struct {
	repo     domain.TripRepository
	fareRepo domain.RideFareRepository
}

func NewTripService(repo domain.TripRepository, fareRepo domain.RideFareRepository) *TripService {
	return &TripService{repo: repo, fareRepo: fareRepo}
}

// Implement service methods here bcoz NewTripSerice return tripservice --> it should implement all methods of tripService defined in domain

func (s *TripService) CreateTrip(ctx context.Context, fare *domain.RideFareModel) (*domain.TripModel, error) {
	t := &domain.TripModel{
		ID:       primitive.NewObjectID(),
		UserID:   fare.UserID,
		RideFare: fare, // * gives value stored at address fare
		Status:   "pending",
		// Keep driver nil until a driver is actually assigned.
		Driver: nil,
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
		// convert to go struct osrmapiresponse from json data received from osrm api
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &routeResponse, nil
}

func (s *TripService) GetAndValidateFare(ctx context.Context, fareID, userID string) (*domain.RideFareModel, error) {
	fare, err := s.fareRepo.GetRideFareByID(ctx, fareID, userID)
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

func (s *TripService) GenerateTripFares(ctx context.Context, rideFares []*domain.RideFareModel, userID string, route *tripTypes.OSRMApiResponse) (*domain.FarePreview, error) {
	options := make([]*domain.FarePreviewOption, len(rideFares))
	for i, f := range rideFares {
		options[i] = &domain.FarePreviewOption{
			ID:                primitive.NewObjectID(),
			PackageSlug:       f.PackageSlug,
			TotalPriceInCents: f.TotalPriceInCents,
		}
	}

	// single write — overwrites any previous preview (cancel + retry safe)
	preview := &domain.FarePreview{
		UserID: userID,
		Route:  route,
		Fares:  options,
	}
	if err := s.fareRepo.SaveFarePreview(ctx, preview); err != nil {
		return nil, fmt.Errorf("failed to save fare preview: %w", err)
	}

	return preview, nil
}

func getBaseFares() []*domain.RideFareModel {
	return []*domain.RideFareModel{
		{
			PackageSlug:       "van",
			TotalPriceInCents: 100,
		},
		{
			PackageSlug:       "suv",
			TotalPriceInCents: 150,
		},
		{
			PackageSlug:       "sedan",
			TotalPriceInCents: 250,
		},
		{
			PackageSlug:       "luxury",
			TotalPriceInCents: 400,
		},
	}
}

func (s *TripService) GetTripByID(ctx context.Context, id string) (*domain.TripModel, error) {
	return s.repo.GetTripByID(ctx, id)
}

func (s *TripService) UpdateTrip(ctx context.Context, tripID string, status string, driver *pb.TripDriver) error {
	return s.repo.UpdateTrip(ctx, tripID, status, driver)
}

func (s *TripService) CancelTrip(ctx context.Context, tripID, requesterUserID string) (*domain.TripModel, error) {
	trip, err := s.repo.GetTripByID(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("trip not found: %w", err)
	}

	isRider := trip.UserID == requesterUserID
	isAccepted := trip.Status == "accepted"
	driverID := ""
	if trip.Driver != nil {
		driverID = trip.Driver.Id
	}
	isAcceptedDriver := isAccepted && driverID != ""
	isAssignedDriver := isAcceptedDriver && driverID == requesterUserID

	if !isRider && !isAssignedDriver {
		return nil, fmt.Errorf("only the rider can cancel before acceptance; after acceptance rider or assigned driver can cancel")
	}

	if trip.Status == "cancelled" {
		return trip, nil // idempotent
	}
	if err := s.repo.UpdateTrip(ctx, tripID, "cancelled", nil); err != nil {
		return nil, fmt.Errorf("failed to cancel trip: %w", err)
	}
	trip.Status = "cancelled"
	return trip, nil
}
