package service

import (
	"context"
	"fmt"
	"time"

	"ride-sharing/services/payment-service/internal/domain"
	"ride-sharing/services/payment-service/pkg/types"

	"github.com/google/uuid"
)

type paymentService struct {
	paymentProcessor domain.PaymentProcessor
	sessionStore     domain.SessionStore
}

// NewPaymentService creates a new instance of the payment service
func NewPaymentService(paymentProcessor domain.PaymentProcessor, sessionStore domain.SessionStore) domain.Service {
	return &paymentService{
		paymentProcessor: paymentProcessor,
		sessionStore:     sessionStore,
	}
}

// CreatePaymentSession creates a new payment session for a trip.
// It is idempotent: if a session was already created for the given tripID it
// returns the existing session instead of calling Stripe again.
func (s *paymentService) CreatePaymentSession(
	ctx context.Context,
	tripID string,
	userID string,
	driverID string,
	amount int64,
	currency string,
) (*types.PaymentIntent, error) {
	// Check Redis first to avoid creating a duplicate Stripe session.
	existing, err := s.sessionStore.Get(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("failed to check session store: %w", err)
	}
	if existing != "" {
		return &types.PaymentIntent{
			TripID:          tripID,
			UserID:          userID,
			DriverID:        driverID,
			Amount:          amount,
			Currency:        currency,
			StripeSessionID: existing,
		}, nil
	}

	metadata := map[string]string{
		"trip_id":   tripID,
		"user_id":   userID,
		"driver_id": driverID,
	}

	sessionID, err := s.paymentProcessor.CreatePaymentSession(ctx, amount, currency, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment session: %w", err)
	}

	if err := s.sessionStore.Set(ctx, tripID, sessionID); err != nil {
		// Non-fatal: session was created on Stripe; log but don't fail the request.
		fmt.Printf("warning: failed to persist session ID in store for trip %s: %v\n", tripID, err)
	}

	return &types.PaymentIntent{
		ID:              uuid.New().String(),
		TripID:          tripID,
		UserID:          userID,
		DriverID:        driverID,
		Amount:          amount,
		Currency:        currency,
		StripeSessionID: sessionID,
		CreatedAt:       time.Now(),
	}, nil
}
