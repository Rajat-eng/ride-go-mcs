// +build integration

package service

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// setupMongoForChatTest connects to MongoDB for testing
func setupMongoForChatTest(ctx context.Context) (*mongo.Client, error) {
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

// TestValidateMessageContentIntegration tests message content validation
func TestValidateMessageContentIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tests := []struct {
		name       string
		content    string
		maxLength  int
		wantErr    bool
	}{
		{
			name:      "valid_short_message",
			content:   "Hello!",
			maxLength: 100,
			wantErr:   false,
		},
		{
			name:      "valid_max_length_message",
			content:   "a",
			maxLength: 1,
			wantErr:   false,
		},
		{
			name:      "empty_message",
			content:   "",
			maxLength: 100,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMessageContent(tt.content, tt.maxLength)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMessageContent error = %v, wantErr %v",
					err, tt.wantErr)
			}
		})
	}
}

// TestValidateTripIDIntegration tests trip ID validation
func TestValidateTripIDIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	client, err := setupMongoForChatTest(ctx)
	if err != nil {
		t.Skipf("failed to setup MongoDB: %v (ensure MongoDB is running)", err)
	}
	defer client.Disconnect(ctx)

	// Test valid and invalid trip IDs
	validTripID := "507f1f77bcf86cd799439011"
	invalidTripID := "invalid-trip-id"

	if err := ValidateTripID(validTripID); err != nil {
		t.Errorf("valid trip ID should not error: %v", err)
	}

	if err := ValidateTripID(invalidTripID); err == nil {
		t.Error("invalid trip ID should error")
	}
}
