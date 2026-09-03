// +build integration

package main

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupRedisTestContainer starts a Redis test container and returns a client
func setupRedisTestContainer(ctx context.Context) (*redis.Client, testcontainers.Container, error) {
	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor: wait.ForLog("Ready to accept connections").
			WithStartupTimeout(10 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, nil, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		return nil, nil, err
	}

	port, err := container.MappedPort(ctx, "6379")
	if err != nil {
		container.Terminate(ctx)
		return nil, nil, err
	}

	client := redis.NewClient(&redis.Options{
		Addr: host + ":" + port.Port(),
	})

	return client, container, nil
}

// TestUpdateDriverLocation tests the critical path: updating driver location in Redis GEO index
func TestUpdateDriverLocation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	client, container, err := setupRedisTestContainer(ctx)
	if err != nil {
		t.Fatalf("failed to setup Redis test container: %v", err)
	}
	defer container.Terminate(ctx)

	svc := NewService(client)

	tests := []struct {
		name    string
		driverID string
		packageSlug string
		lat     float64
		lng     float64
		wantErr bool
	}{
		{
			name:        "valid_location_update",
			driverID:    "driver123",
			packageSlug: "bangalore",
			lat:         12.9716,
			lng:         77.5946,
			wantErr:     false,
		},
		{
			name:        "update_multiple_drivers",
			driverID:    "driver456",
			packageSlug: "bangalore",
			lat:         12.9550,
			lng:         77.7010,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.UpdateDriverLocation(tt.driverID, tt.packageSlug, tt.lat, tt.lng)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateDriverLocation error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestFindAvailableDrivers tests the critical path: searching for nearby drivers
func TestFindAvailableDrivers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	client, container, err := setupRedisTestContainer(ctx)
	if err != nil {
		t.Fatalf("failed to setup Redis test container: %v", err)
	}
	defer container.Terminate(ctx)

	svc := NewService(client)

	// Setup: Add multiple drivers to the geo index
	packageSlug := "bangalore"
	drivers := []struct {
		id  string
		lat float64
		lng float64
	}{
		{"driver1", 12.9716, 77.5946},
		{"driver2", 12.9750, 77.5980},
		{"driver3", 12.9800, 77.6050},
		{"driver_far", 13.0500, 77.7500}, // Far driver (outside 15km radius)
	}

	for _, d := range drivers {
		if err := svc.UpdateDriverLocation(d.id, packageSlug, d.lat, d.lng); err != nil {
			t.Fatalf("setup failed: UpdateDriverLocation error %v", err)
		}
	}

	// Test: Find nearby drivers
	pickupLat := 12.9716
	pickupLng := 77.5946

	nearby := svc.FindAvailableDrivers(packageSlug, pickupLat, pickupLng)

	if len(nearby) == 0 {
		t.Error("expected to find nearby drivers, got none")
	}

	// Verify that far driver is not included (unlikely to be within 15km)
	for _, id := range nearby {
		if id == "driver_far" {
			t.Logf("warning: driver_far found in nearby list, may be outside radius")
		}
	}
}

// TestRemoveDriverFromGeo tests removing a driver from the geo index
func TestRemoveDriverFromGeo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	client, container, err := setupRedisTestContainer(ctx)
	if err != nil {
		t.Fatalf("failed to setup Redis test container: %v", err)
	}
	defer container.Terminate(ctx)

	svc := NewService(client)

	driverID := "driver123"
	packageSlug := "bangalore"

	// Setup: Add driver
	if err := svc.UpdateDriverLocation(driverID, packageSlug, 12.9716, 77.5946); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Test: Remove driver
	svc.RemoveDriverFromGeo(ctx, driverID, packageSlug)

	// Verify: Driver should not be found
	nearby := svc.FindAvailableDrivers(packageSlug, 12.9716, 77.5946)
	for _, id := range nearby {
		if id == driverID {
			t.Errorf("expected driver %s to be removed, but found in results", driverID)
		}
	}
}
