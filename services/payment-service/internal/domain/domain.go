package domain

import (
	"context"

	"ride-sharing/services/payment-service/pkg/types"
)

type Service interface {
	CreatePaymentSession(ctx context.Context, tripID, userID, driverID string, amount int64, currency string) (*types.PaymentIntent, error)
}

type PaymentProcessor interface {
	CreatePaymentSession(ctx context.Context, amount int64, currency string, metadata map[string]string) (string, error)
	// GetSessionStatus(ctx context.Context, sessionID string) (types.PaymentStatus, error)
}

// SessionStore provides idempotency for payment session creation.
// Get returns ("", nil) when no session exists for the key yet.
type SessionStore interface {
	Get(ctx context.Context, tripID string) (sessionID string, err error)
	Set(ctx context.Context, tripID string, sessionID string) error
}
