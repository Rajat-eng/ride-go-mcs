package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"ride-sharing/services/trip-service/internal/domain"
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
		DriverID: "",
		From:     "",
		To:       "",
		RideFare: fare, // * gives value stored at address fare
		Status:   "pending",
	}
	return s.repo.CreateTrip(ctx, t)
}

func (s *TripService) GetRoute(ctx context.Context, pickup, destination *types.Coordinate) (*types.OSRMApiResponse, error) {

	url := fmt.Sprintf("http://router.project-osrm.org/route/v1/driving/%f,%f;%f,%f?overview=full&geometries=geojson",
		pickup.Latitude,
		pickup.Longitude,
		destination.Latitude,
		destination.Longitude,
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
	var routeResponse types.OSRMApiResponse
	if err := json.Unmarshal(body, &routeResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &routeResponse, nil
}
