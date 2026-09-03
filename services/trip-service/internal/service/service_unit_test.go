package service

import (
	"context"
	"testing"

	"ride-sharing/services/trip-service/internal/domain"
	pb "ride-sharing/shared/proto/trip"
)

// MockTripRepository mocks the TripRepository for unit testing
type MockTripRepository struct {
	trips map[string]*domain.TripModel
}

func NewMockTripRepository() *MockTripRepository {
	return &MockTripRepository{
		trips: make(map[string]*domain.TripModel),
	}
}

func (m *MockTripRepository) CreateTrip(ctx context.Context, trip *domain.TripModel) (*domain.TripModel, error) {
	m.trips[trip.ID.Hex()] = trip
	return trip, nil
}

func (m *MockTripRepository) GetTripByID(ctx context.Context, id string) (*domain.TripModel, error) {
	if trip, ok := m.trips[id]; ok {
		return trip, nil
	}
	return nil, nil
}

func (m *MockTripRepository) UpdateTrip(ctx context.Context, tripID string, status string, driver *pb.TripDriver) error {
	if trip, ok := m.trips[tripID]; ok {
		trip.Status = status
		return nil
	}
	return nil
}

// MockRideFareRepository mocks the RideFareRepository for unit testing
type MockRideFareRepository struct {
	fares map[string]*domain.RideFareModel
}

func NewMockRideFareRepository() *MockRideFareRepository {
	return &MockRideFareRepository{
		fares: make(map[string]*domain.RideFareModel),
	}
}

func (m *MockRideFareRepository) SaveFarePreview(ctx context.Context, preview *domain.FarePreview) error {
	return nil
}

func (m *MockRideFareRepository) GetRideFareByID(ctx context.Context, fareID, userID string) (*domain.RideFareModel, error) {
	if fare, ok := m.fares[fareID]; ok {
		if fare.UserID == userID {
			return fare, nil
		}
	}
	return nil, nil
}

// TestNewTripService validates that TripService is created correctly
func TestNewTripService(t *testing.T) {
	tripRepo := NewMockTripRepository()
	fareRepo := NewMockRideFareRepository()

	svc := NewTripService(tripRepo, fareRepo)

	if svc == nil {
		t.Fatal("expected TripService to not be nil")
	}
}

// TestValidateTripStatus validates trip status constraints
func TestValidateTripStatus(t *testing.T) {
	tests := []struct {
		name    string
		status  string
		wantErr bool
	}{
		{
			name:    "valid_pending_status",
			status:  "pending",
			wantErr: false,
		},
		{
			name:    "valid_accepted_status",
			status:  "accepted",
			wantErr: false,
		},
		{
			name:    "valid_completed_status",
			status:  "completed",
			wantErr: false,
		},
		{
			name:    "valid_cancelled_status",
			status:  "cancelled",
			wantErr: false,
		},
		{
			name:    "invalid_status",
			status:  "invalid_status",
			wantErr: true,
		},
		{
			name:    "empty_status",
			status:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTripStatus(tt.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTripStatus(%s) error = %v, wantErr %v",
					tt.status, err, tt.wantErr)
			}
		})
	}
}

// TestValidateCoordinate validates coordinate constraints
func TestValidateCoordinate(t *testing.T) {
	tests := []struct {
		name    string
		lat     float64
		lng     float64
		wantErr bool
	}{
		{
			name:    "valid_coordinate",
			lat:     12.9716,
			lng:     77.5946,
			wantErr: false,
		},
		{
			name:    "invalid_latitude",
			lat:     91.0,
			lng:     77.5946,
			wantErr: true,
		},
		{
			name:    "invalid_longitude",
			lat:     12.9716,
			lng:     181.0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCoordinate(tt.lat, tt.lng)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCoordinate(%f, %f) error = %v, wantErr %v",
					tt.lat, tt.lng, err, tt.wantErr)
			}
		})
	}
}
