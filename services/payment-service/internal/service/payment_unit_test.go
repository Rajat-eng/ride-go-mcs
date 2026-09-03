package service

import (
	"testing"
)

// TestValidatePaymentAmount validates payment amount constraints
func TestValidatePaymentAmount(t *testing.T) {
	tests := []struct {
		name    string
		amount  float64
		wantErr bool
	}{
		{
			name:    "valid_amount",
			amount:  250.50,
			wantErr: false,
		},
		{
			name:    "zero_amount",
			amount:  0.0,
			wantErr: true,
		},
		{
			name:    "negative_amount",
			amount:  -100.0,
			wantErr: true,
		},
		{
			name:    "very_small_amount",
			amount:  0.01,
			wantErr: false,
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

// TestValidatePaymentMethod validates payment method
func TestValidatePaymentMethod(t *testing.T) {
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
			name:    "valid_wallet",
			method:  "wallet",
			wantErr: false,
		},
		{
			name:    "invalid_method",
			method:  "invalid_payment_method",
			wantErr: true,
		},
		{
			name:    "empty_method",
			method:  "",
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

// TestValidateCurrency validates currency code
func TestValidateCurrency(t *testing.T) {
	tests := []struct {
		name    string
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
		{
			name:     "empty_currency",
			currency: "",
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

// TestValidateTransactionID validates transaction ID format
func TestValidateTransactionID(t *testing.T) {
	tests := []struct {
		name      string
		txnID     string
		wantErr   bool
	}{
		{
			name:    "valid_transaction_id",
			txnID:   "TXN-123456789",
			wantErr: false,
		},
		{
			name:    "empty_transaction_id",
			txnID:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTransactionID(tt.txnID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTransactionID error = %v, wantErr %v",
					err, tt.wantErr)
			}
		})
	}
}
