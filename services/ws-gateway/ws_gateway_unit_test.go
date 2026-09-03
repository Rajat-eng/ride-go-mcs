package main

import (
	"context"
	"net/http/httptest"
	"testing"

	jwt "github.com/golang-jwt/jwt/v5"
)

// TestTripChatRoomID validates chat room ID generation
func TestTripChatRoomID(t *testing.T) {
	tests := []struct {
		name     string
		tripID   string
		expected string
	}{
		{
			name:     "valid_trip_id",
			tripID:   "trip123",
			expected: "trip:trip123:chat",
		},
		{
			name:     "empty_trip_id",
			tripID:   "",
			expected: "",
		},
		{
			name:     "trip_id_with_special_chars",
			tripID:   "trip-456-abc",
			expected: "trip:trip-456-abc:chat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID := tripChatRoomID(tt.tripID)
			if roomID != tt.expected {
				t.Errorf("tripChatRoomID(%s) = %s, expected %s",
					tt.tripID, roomID, tt.expected)
			}
		})
	}
}

// TestExtractJWTFromHeader validates JWT extraction from Bearer token
func TestExtractJWTFromHeader(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		wantToken  string
		wantErr    bool
	}{
		{
			name:       "valid_bearer_token",
			authHeader: "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			wantToken:  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			wantErr:    false,
		},
		{
			name:       "missing_bearer_prefix",
			authHeader: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			wantToken:  "",
			wantErr:    true,
		},
		{
			name:       "empty_auth_header",
			authHeader: "",
			wantToken:  "",
			wantErr:    true,
		},
		{
			name:       "bearer_with_no_token",
			authHeader: "Bearer ",
			wantToken:  "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := ExtractJWTFromHeader(tt.authHeader)
			if (token == "") != tt.wantErr && token != tt.wantToken {
				t.Errorf("ExtractJWTFromHeader(%s) = %s, expected %s",
					tt.authHeader, token, tt.wantToken)
			}
		})
	}
}

// TestExtractUserIDFromToken validates user ID extraction from JWT claims
func TestExtractUserIDFromToken(t *testing.T) {
	tests := []struct {
		name       string
		buildClaim func() jwt.MapClaims
		wantUserID string
		wantErr    bool
	}{
		{
			name: "valid_user_id",
			buildClaim: func() jwt.MapClaims {
				return jwt.MapClaims{
					"user_id": "user123",
					"name":    "John Doe",
				}
			},
			wantUserID: "user123",
			wantErr:    false,
		},
		{
			name: "missing_user_id",
			buildClaim: func() jwt.MapClaims {
				return jwt.MapClaims{
					"name": "John Doe",
				}
			},
			wantUserID: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := tt.buildClaim()
			userID, err := ExtractUserIDFromToken(claims)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractUserIDFromToken error = %v, wantErr %v",
					err, tt.wantErr)
			}
			if userID != tt.wantUserID {
				t.Errorf("ExtractUserIDFromToken = %s, expected %s",
					userID, tt.wantUserID)
			}
		})
	}
}

// TestValidateWebSocketContext validates context has required auth fields
func TestValidateWebSocketContext(t *testing.T) {
	tests := []struct {
		name     string
		userID   interface{}
		socketID interface{}
		wantErr  bool
	}{
		{
			name:     "valid_context",
			userID:   "user123",
			socketID: "socket-abc",
			wantErr:  false,
		},
		{
			name:     "missing_user_id",
			userID:   nil,
			socketID: "socket-abc",
			wantErr:  true,
		},
		{
			name:     "missing_socket_id",
			userID:   "user123",
			socketID: nil,
			wantErr:  true,
		},
		{
			name:     "invalid_user_id_type",
			userID:   123,
			socketID: "socket-abc",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			ctx := req.Context()

			if tt.userID != nil {
				req = req.WithContext(setContextValue(ctx, ctxKeyUserID, tt.userID))
			}
			ctx = req.Context()

			if tt.socketID != nil {
				req = req.WithContext(setContextValue(ctx, ctxKeySocketID, tt.socketID))
			}

			err := ValidateWebSocketContext(req.Context())
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWebSocketContext error = %v, wantErr %v",
					err, tt.wantErr)
			}
		})
	}
}

// Helper function to set context value
func setContextValue(ctx context.Context, key contextKey, value interface{}) context.Context {
	return context.WithValue(ctx, key, value)
}
