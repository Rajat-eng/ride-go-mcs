package repository

import (
	"context"
	"fmt"
	"ride-sharing/services/trip-service/internal/domain"
)

type InMemoryRepository struct {
	trips     map[string]*domain.TripModel
	rideFares map[string]*domain.RideFareModel
}

func NewInmemoryRepository() *InMemoryRepository {
	// this is trip repository which gets injected into the service layer
	// it uses in-memory storage for simplicity
	// in real-world applications, this would be replaced with a database connection
	return &InMemoryRepository{
		trips:     make(map[string]*domain.TripModel),
		rideFares: make(map[string]*domain.RideFareModel),
	}
}

func (r *InMemoryRepository) CreateTrip(ctx context.Context, trip *domain.TripModel) (*domain.TripModel, error) {
	r.trips[trip.ID.Hex()] = trip
	return trip, nil
}

func (r *InMemoryRepository) SaveRideFare(ctx context.Context, f *domain.RideFareModel) error {
	r.rideFares[f.ID.Hex()] = f
	return nil
}

func (r *InMemoryRepository) GetRideFareByID(ctx context.Context, id string) (*domain.RideFareModel, error) {
	fare, exist := r.rideFares[id]
	if !exist {
		return nil, fmt.Errorf("fare does not exist with ID: %s", id)
	}

	return fare, nil
}
