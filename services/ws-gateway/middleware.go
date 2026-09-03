package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"ride-sharing/shared/env"

	jwt "github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	ctxKeyUserID   contextKey = "userID"
	ctxKeyName     contextKey = "name"
	ctxKeyToken    contextKey = "token"
	ctxKeySocketID contextKey = "socketID"
)

// wsAuthMiddleware validates the JWT supplied either as a Bearer header or a
// ?token= query parameter (required for browser WebSocket clients).
func wsAuthMiddleware(next http.Handler) http.Handler {
	jwtSecret := []byte(env.GetString("JWT_SECRET", "change-me-in-production"))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr := ""

		if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			tokenStr = strings.TrimPrefix(auth, "Bearer ")
		}
		if tokenStr == "" {
			tokenStr = r.URL.Query().Get("token")
		}
		if tokenStr == "" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}

		token, err := jwt.ParseWithClaims(tokenStr, &jwt.MapClaims{}, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(*jwt.MapClaims)
		if !ok {
			http.Error(w, "invalid token claims", http.StatusUnauthorized)
			return
		}

		userID, _ := (*claims)["user_id"].(string)
		if userID == "" {
			http.Error(w, "invalid token claims", http.StatusUnauthorized)
			return
		}
		name, _ := (*claims)["name"].(string)

		ctx := context.WithValue(r.Context(), ctxKeyUserID, userID)
		ctx = context.WithValue(ctx, ctxKeyToken, tokenStr)
		ctx = context.WithValue(ctx, ctxKeyName, name)
		next.ServeHTTP(w, r.WithContext(ctx))
	})}

// ExtractJWTFromHeader extracts JWT from Authorization Bearer header
func ExtractJWTFromHeader(authHeader string) string {
	if authHeader == "" {
		return ""
	}
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return ""
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return ""
	}
	return token
}

// ExtractUserIDFromToken extracts user_id from JWT claims
func ExtractUserIDFromToken(claims jwt.MapClaims) (string, error) {
	userID, ok := claims["user_id"].(string)
	if !ok || userID == "" {
		return "", fmt.Errorf("user_id not found or invalid in token claims")
	}
	return userID, nil
}

// ValidateWebSocketContext checks if context has required auth fields
func ValidateWebSocketContext(ctx context.Context) error {
	userID, ok := ctx.Value(ctxKeyUserID).(string)
	if !ok || userID == "" {
		return fmt.Errorf("missing or invalid userID in context")
	}

	socketID, ok := ctx.Value(ctxKeySocketID).(string)
	if !ok || socketID == "" {
		return fmt.Errorf("missing or invalid socketID in context")
	}

	return nil}
