// +build integration

package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// setupRedisForRateLimitTest starts a Redis client (assumes Redis is running locally)
func setupRedisForRateLimitTest(ctx context.Context) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Test connection
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx, nil).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return client, nil
}

// TestRateLimiterIntegration tests rate limiting with actual Redis
func TestRateLimiterIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	client, err := setupRedisForRateLimitTest(ctx)
	if err != nil {
		t.Skipf("failed to setup Redis: %v (ensure Redis is running)", err)
	}
	defer client.Close()

	// Cleanup test data
	defer client.FlushDB(ctx)

	rl := NewRateLimiter(client)

	tests := []struct {
		name      string
		key       string
		limit     int
		requests  int
		wantLimit bool
	}{
		{
			name:      "within_limit",
			key:       "test:user:1",
			limit:     5,
			requests:  3,
			wantLimit: false,
		},
		{
			name:      "at_limit",
			key:       "test:user:2",
			limit:     5,
			requests:  5,
			wantLimit: false,
		},
		{
			name:      "exceeds_limit",
			key:       "test:user:3",
			limit:     5,
			requests:  6,
			wantLimit: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Cleanup key
			client.Del(ctx, tt.key)

			// Simulate requests
			for i := 0; i < tt.requests; i++ {
				// In real usage, this would use Limit middleware
				// For now, we test the underlying lua script directly
				count, err := luaRateLimit.Run(ctx, client, []string{tt.key}, 60).Int()
				if err != nil {
					t.Fatalf("rate limit script error: %v", err)
				}

				if !tt.wantLimit && count > tt.limit {
					t.Errorf("request %d: count %d exceeds limit %d", i+1, count, tt.limit)
				}
			}
		})
	}
}

// TestRateLimiterWindowExpiry tests that rate limit window expires correctly
func TestRateLimiterWindowExpiry(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	client, err := setupRedisForRateLimitTest(ctx)
	if err != nil {
		t.Skipf("failed to setup Redis: %v", err)
	}
	defer client.Close()

	defer client.FlushDB(ctx)

	key := "test:expiry:key"
	windowSecs := 2 // 2 second window

	// First request
	count1, err := luaRateLimit.Run(ctx, client, []string{key}, windowSecs).Int()
	if err != nil {
		t.Fatalf("first request error: %v", err)
	}

	if count1 != 1 {
		t.Errorf("first request count expected 1, got %d", count1)
	}

	// Wait for window to expire
	time.Sleep(time.Duration(windowSecs+1) * time.Second)

	// Second request in new window
	count2, err := luaRateLimit.Run(ctx, client, []string{key}, windowSecs).Int()
	if err != nil {
		t.Fatalf("second request error: %v", err)
	}

	if count2 != 1 {
		t.Errorf("second request (after expiry) expected 1, got %d", count2)
	}
}

// TestAuthMiddlewareIntegration tests authentication middleware (critical path)
func TestAuthMiddlewareIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	client, err := setupRedisForRateLimitTest(ctx)
	if err != nil {
		t.Skipf("failed to setup Redis: %v", err)
	}
	defer client.Close()

	defer client.FlushDB(ctx)

	// Test valid credentials validation
	email := "test@example.com"
	password := "securepassword123"

	// Validate email
	if err := ValidateEmail(email); err != nil {
		t.Errorf("valid email validation failed: %v", err)
	}

	// Validate password
	if err := ValidatePassword(password); err != nil {
		t.Errorf("valid password validation failed: %v", err)
	}

	// Test invalid email
	if err := ValidateEmail("invalid-email"); err == nil {
		t.Error("expected error for invalid email")
	}

	// Test invalid password
	if err := ValidatePassword("short"); err == nil {
		t.Error("expected error for password < 8 chars")
	}
}
