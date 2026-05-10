package token

import (
	"fmt"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

type Manager struct {
	secretKey          []byte
	accessTokenExpiry  time.Duration
	refreshTokenExpiry time.Duration
}

type AccessClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type RefreshClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func NewManager(secretKey string, accessExpiry, refreshExpiry time.Duration) *Manager {
	return &Manager{
		secretKey:          []byte(secretKey),
		accessTokenExpiry:  accessExpiry,
		refreshTokenExpiry: refreshExpiry,
	}
}

func (m *Manager) GenerateAccessToken(userID, email, role string) (string, error) {
	// Create claims with user ID, email, and standard JWT claims like expiration and issued at
	claims := AccessClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.accessTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

func (m *Manager) GenerateRefreshToken(userID string) (string, error) {
	claims := RefreshClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.refreshTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

func (m *Manager) ValidateAccessToken(tokenStr string) (*AccessClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &AccessClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secretKey, nil // return the secret key for validation
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*AccessClaims) // token returns the claims if valid and we assert it to our AccessClaims struct
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

func (m *Manager) ValidateRefreshToken(tokenStr string) (*RefreshClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &RefreshClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secretKey, nil // return the secret key for validation to ensure the token is valid and was signed with our secret
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*RefreshClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}
