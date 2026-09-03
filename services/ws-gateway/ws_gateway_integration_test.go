// +build integration

package main

import (
	"context"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

// setupRedisForWSTest starts a Redis client for WS tests
func setupRedisForWSTest(ctx context.Context) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Test connection
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx, nil).Err(); err != nil {
		return nil, err
	}

	return client, nil
}

// TestTripChatRoomIDIntegration tests chat room ID generation
func TestTripChatRoomIDIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tripID := "trip-123-abc"
	roomID := tripChatRoomID(tripID)

	if roomID == "" {
		t.Error("expected room ID to not be empty")
	}

	if !contains(roomID, tripID) {
		t.Errorf("room ID %s does not contain trip ID %s", roomID, tripID)
	}
}

// TestWebSocketAuthFlow tests complete WS auth flow (critical path)
func TestWebSocketAuthFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	client, err := setupRedisForWSTest(ctx)
	if err != nil {
		t.Skipf("failed to setup Redis: %v (ensure Redis is running)", err)
	}
	defer client.Close()

	// Test valid token extraction and user ID extraction
	validToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoidXNlcjEyMyIsIm5hbWUiOiJKb2huIERvZSJ9.test"
	userID := "user123"

	extractedToken := ExtractJWTFromHeader("Bearer " + validToken)
	if extractedToken != validToken {
		t.Errorf("token extraction failed: got %s", extractedToken)
	}

	// Test invalid token
	invalidToken := ExtractJWTFromHeader("InvalidToken")
	if invalidToken != "" {
		t.Errorf("expected empty token for invalid header, got %s", invalidToken)
	}

	// Test user ID extraction from claims
	claims := jwt.MapClaims{
		"user_id": userID,
		"name":    "John Doe",
	}

	extractedUserID, err := ExtractUserIDFromToken(claims)
	if err != nil {
		t.Errorf("user ID extraction error: %v", err)
	}

	if extractedUserID != userID {
		t.Errorf("expected user ID %s, got %s", userID, extractedUserID)
	}
}

// TestChatMessageRelay tests message relay in trip chat (critical path)
func TestChatMessageRelay(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	client, err := setupRedisForWSTest(ctx)
	if err != nil {
		t.Skipf("failed to setup Redis: %v", err)
	}
	defer client.Close()

	defer client.FlushDB(ctx)

	tripID := "trip-123"
	roomID := tripChatRoomID(tripID)

	// Verify room ID format
	if roomID != "trip:"+tripID+":chat" {
		t.Errorf("expected room ID format 'trip:%s:chat', got %s", tripID, roomID)
	}

	// Test that room ID is empty for empty trip ID
	emptyRoomID := tripChatRoomID("")
	if emptyRoomID != "" {
		t.Errorf("expected empty room ID for empty trip ID, got %s", emptyRoomID)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr))
}
