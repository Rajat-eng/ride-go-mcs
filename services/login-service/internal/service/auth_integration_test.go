// +build integration

package service

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// setupMongoForLoginTest connects to MongoDB for testing
func setupMongoForLoginTest(ctx context.Context) (*mongo.Client, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	return client, nil
}

// TestValidateSignupEmailIntegration tests email validation in signup flow
func TestValidateSignupEmailIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	client, err := setupMongoForLoginTest(ctx)
	if err != nil {
		t.Skipf("failed to setup MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{
			name:    "valid_email",
			email:   "test@example.com",
			wantErr: false,
		},
		{
			name:    "invalid_email",
			email:   "invalid-email",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSignupEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSignupEmail error = %v, wantErr %v",
					err, tt.wantErr)
			}
		})
	}
}

// TestValidateUserRoleIntegration tests user role validation
func TestValidateUserRoleIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tests := []struct {
		name    string
		role    string
		wantErr bool
	}{
		{
			name:    "valid_rider_role",
			role:    "rider",
			wantErr: false,
		},
		{
			name:    "valid_driver_role",
			role:    "driver",
			wantErr: false,
		},
		{
			name:    "invalid_role",
			role:    "invalid_role",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUserRole(tt.role)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUserRole error = %v, wantErr %v",
					err, tt.wantErr)
			}
		})
	}
}

// TestSignupValidationFlow tests complete signup validation (critical path)
func TestSignupValidationFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	client, err := setupMongoForLoginTest(ctx)
	if err != nil {
		t.Skipf("failed to setup MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	// Test complete signup validation
	email := "newuser@example.com"
	password := "securepassword123"
	role := "rider"

	if err := ValidateSignupEmail(email); err != nil {
		t.Errorf("email validation failed: %v", err)
	}

	if err := ValidatePasswordStrength(password); err != nil {
		t.Errorf("password validation failed: %v", err)
	}

	if err := ValidateUserRole(role); err != nil {
		t.Errorf("role validation failed: %v", err)
	}
}
