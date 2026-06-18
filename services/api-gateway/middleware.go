package main

import (
	"context"
	"net/http"
	"strings"

	"ride-sharing/shared/env"

	jwt "github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	ctxKeyUserID contextKey = "userID"
	ctxKeyToken  contextKey = "token"
	ctxKeyName   contextKey = "name"
)

func wsAuthMiddleware(next http.Handler) http.Handler {
	jwtSecret := []byte(env.GetString("JWT_SECRET", "change-me-in-production"))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr := ""

		// 1. Try Authorization header
		if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			tokenStr = strings.TrimPrefix(auth, "Bearer ")
		}

		// 2. Fall back to query param (required for browser WebSocket clients)
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
	})
}
