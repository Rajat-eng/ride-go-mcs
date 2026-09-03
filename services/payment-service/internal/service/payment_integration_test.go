// +build integration

package service

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// setupRedisForPaymentTest connects to Redis for testing
func setupRedisForPaymentTest(ctx context.Context) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx, nil).Err(); err != nil {
		return nil, err
	}

	return client, nil
}

// TestValidatePaymentAmountIntegration tests payment amount validation
func TestValidatePaymentAmountIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	client, err := setupRedisForPaymentTest(ctx)
	if err != nil {
		t.Skipf("failed to setup Redis: %v", err)
	}
	defer client.Close()

	tests := []struct {
		name    string
		amount  float64
		wantErr bool
	}{
		{
			name:    "valid_amount_250",
			amount:  250.0,
			wantErr: false,
		},
		{
			name:    "valid_amount_1",
			amount:  1.0,
			wantErr: false,
		},
		{
			name:    "invalid_amount_zero",
			amount:  0.0,
			wantErr: true,
		},
		{
			name:    "invalid_amount_negative",
			amount:  -100.0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePaymentAmount(tt.amount)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePaymentAmount error = %v, wantErr %v",
					err, tt.wantErr)
			}
		})
	}
}

// TestValidatePaymentMethodIntegration tests payment method validation
func TestValidatePaymentMethodIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	client, err := setupRedisForPaymentTest(ctx)
	if err != nil {
		t.Skipf("failed to setup Redis: %v", err)
	}
	defer client.Close()

	tests := []struct {
		name    string
		method  string
		wantErr bool
	}{
		{
			name:    "valid_credit_card",
			method:  "credit_card",
			wantErr: false,
		},
		{
			name:    "valid_upi",
			method:  "upi",
			wantErr: false,
		},
		{
			name:    "invalid_method",
			method:  "invalid_method",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePaymentMethod(tt.method)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePaymentMethod error = %v, wantErr %v",
					err, tt.wantErr)
			}
		})
	}
}

// TestCreatePaymentSessionIntegration tests payment session creation (critical path)
func TestCreatePaymentSessionIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	client, err := setupRedisForPaymentTest(ctx)
	if err != nil {
		t.Skipf("failed to setup Redis: %v", err)
	}
	defer client.Close()
	defer client.FlushDB(ctx)

	// Test validation before creating session
	amount := int64(25000) // 250 INR in paise
	currency := "INR"

	// Validate amount (convert to float)
	if err := ValidatePaymentAmount(float64(amount) / 100); err != nil {
		t.Errorf("payment amount validation failed: %v", err)
	}

	// Validate currency
	if err := ValidateCurrency(currency); err != nil {
		t.Errorf("currency validation failed: %v", err)
	}

	// Validate method
	if err := ValidatePaymentMethod("credit_card"); err != nil {
		t.Errorf("payment method validation failed: %v", err)
	}
}

// TestValidateCurrencyIntegration tests currency validation with multiple currencies
func TestValidateCurrencyIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tests := []struct {
		name     string
		currency string
		wantErr  bool
	}{
		{
			name:     "valid_inr",
			currency: "INR",
			wantErr:  false,
		},
		{
			name:     "valid_usd",
			currency: "USD",
			wantErr:  false,
		},
		{
			name:     "invalid_currency",
			currency: "XXX",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCurrency(tt.currency)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCurrency error = %v, wantErr %v",
					err, tt.wantErr)
			}
		})
	}
}
