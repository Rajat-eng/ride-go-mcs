// +build integration

package service

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/services/trip-service/internal/infrastructure"
	"ride-sharing/shared/types"
)

// setupMongoTestContainer starts a MongoDB test container and returns a client
// Note: This assumes you have testcontainers-go set up for MongoDB
func setupMongoTestDB(ctx context.Context) (*mongo.Client, error) {
	// Connect to local MongoDB for testing (ensure MongoDB is running)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		return nil, err
	}

	// Verify connection
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	return client, nil
}

// TestCreateTripIntegration tests the critical path: creating a trip in MongoDB
func TestCreateTripIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	client, err := setupMongoTestDB(ctx)
	if err != nil {
		t.Skipf("failed to setup MongoDB: %v (ensure MongoDB is running)", err)
	}
	defer client.Disconnect(ctx)

	// Setup repositories
	db := client.Database("ride_sharing_test")
	tripRepo := infrastructure.NewTripRepository(db)
	fareRepo := infrastructure.NewRideFareRepository(db)

	svc := NewTripService(tripRepo, fareRepo)

	// Create test fare
	testFare := &domain.RideFareModel{
		UserID:           "user123",
		PickupLatitude:   12.9716,
		PickupLongitude:  77.5946,
		DropLatitude:     12.9550,
		DropLongitude:    77.7010,
		EstimatedFare:    250.0,
		Currency:         "INR",
	}

	// Test: Create trip
	trip, err := svc.CreateTrip(ctx, testFare)

	if err != nil {
		t.Errorf("CreateTrip error = %v", err)
	}

	if trip == nil {
		t.Fatal("expected trip to not be nil")
	}

	if trip.UserID != testFare.UserID {
		t.Errorf("expected trip.UserID = %s, got %s", testFare.UserID, trip.UserID)
	}

	if trip.Status != "pending" {
		t.Errorf("expected trip.Status = pending, got %s", trip.Status)
	}

	// Cleanup
	db.Drop(ctx)
}

// TestGetRouteIntegration tests fetching route from OSRM API
func TestGetRouteIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	client, err := setupMongoTestDB(ctx)
	if err != nil {
		t.Skipf("failed to setup MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("ride_sharing_test")
	tripRepo := infrastructure.NewTripRepository(db)
	fareRepo := infrastructure.NewRideFareRepository(db)

	svc := NewTripService(tripRepo, fareRepo)

	// Test: Get route from OSRM
	pickup := &types.Coordinate{
		Latitude:  12.9716,
		Longitude: 77.5946,
	}

	destination := &types.Coordinate{
		Latitude:  12.9550,
		Longitude: 77.7010,
	}

	route, err := svc.GetRoute(ctx, pickup, destination)

	// OSRM API might not be available in test environment
	if err != nil {
		t.Logf("GetRoute skipped (OSRM API not available): %v", err)
		return
	}

	if route == nil {
		t.Fatal("expected route to not be nil")
	}

	if len(route.Routes) == 0 {
		t.Error("expected at least one route in response")
	}
}

// TestGetAndValidateFareIntegration tests fare validation (critical path)
func TestGetAndValidateFareIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	client, err := setupMongoTestDB(ctx)
	if err != nil {
		t.Skipf("failed to setup MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("ride_sharing_test")
	tripRepo := infrastructure.NewTripRepository(db)
	fareRepo := infrastructure.NewRideFareRepository(db)

	svc := NewTripService(tripRepo, fareRepo)

	userID := "user123"
	testFare := &domain.RideFareModel{
		ID:               primitive.NewObjectID(),
		UserID:           userID,
		PickupLatitude:   12.9716,
		PickupLongitude:  77.5946,
		DropLatitude:     12.9550,
		DropLongitude:    77.7010,
		EstimatedFare:    250.0,
		Currency:         "INR",
	}

	// Setup: Insert test fare
	if _, err := fareRepo.CreateRideFare(ctx, testFare); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Test: Get and validate fare
	fare, err := svc.GetAndValidateFare(ctx, testFare.ID.Hex(), userID)

	if err != nil {
		t.Errorf("GetAndValidateFare error = %v", err)
	}

	if fare == nil {
		t.Fatal("expected fare to not be nil")
	}

	if fare.UserID != userID {
		t.Errorf("expected fare.UserID = %s, got %s", userID, fare.UserID)
	}

	// Cleanup
	db.Drop(ctx)
}
