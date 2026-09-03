package main

import (
	"testing"

	"github.com/redis/go-redis/v9"
)

// TestNewService creates a service instance and validates it
func TestNewService(t *testing.T) {
	client := &redis.Client{}
	svc := NewService(client)

	if svc == nil {
		t.Fatal("expected service to not be nil")
	}

	if svc.rdb != client {
		t.Fatal("expected service.rdb to reference the provided redis client")
	}
}

// TestValidateLocationCoordinates validates latitude and longitude constraints
func TestValidateLocationCoordinates(t *testing.T) {
	tests := []struct {
		name    string
		lat     float64
		lng     float64
		wantErr bool
	}{
		{
			name:    "valid_bangalore_coordinates",
			lat:     12.9716,
			lng:     77.5946,
			wantErr: false,
		},
		{
			name:    "latitude_too_high",
			lat:     90.1,
			lng:     77.5946,
			wantErr: true,
		},
		{
			name:    "latitude_too_low",
			lat:     -90.1,
			lng:     77.5946,
			wantErr: true,
		},
		{
			name:    "longitude_too_high",
			lat:     12.9716,
			lng:     180.1,
			wantErr: true,
		},
		{
			name:    "longitude_too_low",
			lat:     12.9716,
			lng:     -180.1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLocationCoordinates(tt.lat, tt.lng)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateLocationCoordinates(%f, %f) error = %v, wantErr %v",
					tt.lat, tt.lng, err, tt.wantErr)
			}
		})
	}
}

// TestValidateDriverID validates driver ID constraints
func TestValidateDriverID(t *testing.T) {
	tests := []struct {
		name      string
		driverID  string
		wantErr   bool
		errReason string
	}{
		{
			name:      "valid_driver_id",
			driverID:  "driver123",
			wantErr:   false,
			errReason: "",
		},
		{
			name:      "empty_driver_id",
			driverID:  "",
			wantErr:   true,
			errReason: "empty driver ID",
		},
		{
			name:      "driver_id_with_special_chars",
			driverID:  "driver@123",
			wantErr:   true,
			errReason: "invalid characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDriverID(tt.driverID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDriverID(%s) error = %v, wantErr %v (reason: %s)",
					tt.driverID, err, tt.wantErr, tt.errReason)
			}
		})
	}
}
