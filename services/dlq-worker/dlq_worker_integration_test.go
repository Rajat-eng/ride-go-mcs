// +build integration

package main

import (
	"testing"
)

// TestValidateEventTypeIntegration tests event type validation
func TestValidateEventTypeIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tests := []struct {
		name     string
		eventType string
		wantErr  bool
	}{
		{
			name:     "valid_payment_failed",
			eventType: "payment_failed",
			wantErr:  false,
		},
		{
			name:     "valid_trip_cancelled",
			eventType: "trip_cancelled",
			wantErr:  false,
		},
		{
			name:     "invalid_event",
			eventType: "invalid_event",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEventType(tt.eventType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEventType error = %v, wantErr %v",
					err, tt.wantErr)
			}
		})
	}
}

// TestValidateMaxRetriesIntegration tests retry count validation
func TestValidateMaxRetriesIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tests := []struct {
		name    string
		retries int
		wantErr bool
	}{
		{
			name:    "valid_retries_3",
			retries: 3,
			wantErr: false,
		},
		{
			name:    "valid_retries_0",
			retries: 0,
			wantErr: false,
		},
		{
			name:    "valid_retries_50",
			retries: 50,
			wantErr: false,
		},
		{
			name:    "invalid_retries_negative",
			retries: -1,
			wantErr: true,
		},
		{
			name:    "invalid_retries_excessive",
			retries: 1000,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMaxRetries(tt.retries)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMaxRetries error = %v, wantErr %v",
					err, tt.wantErr)
			}
		})
	}
}

// TestIsRetryableErrorIntegration tests error classification
func TestIsRetryableErrorIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tests := []struct {
		name       string
		errMsg     string
		isRetryable bool
	}{
		{
			name:        "retryable_timeout",
			errMsg:      "connection timeout",
			isRetryable: true,
		},
		{
			name:        "retryable_unavailable",
			errMsg:      "service unavailable",
			isRetryable: true,
		},
		{
			name:        "retryable_io_timeout",
			errMsg:      "i/o timeout",
			isRetryable: true,
		},
		{
			name:        "non_retryable_invalid",
			errMsg:      "invalid request format",
			isRetryable: false,
		},
		{
			name:        "non_retryable_not_found",
			errMsg:      "resource not found",
			isRetryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retryable := IsRetryableError(tt.errMsg)
			if retryable != tt.isRetryable {
				t.Errorf("IsRetryableError(%s) = %v, expected %v",
					tt.errMsg, retryable, tt.isRetryable)
			}
		})
	}
}

// TestDLQMessageProcessingFlow tests the complete DLQ processing flow (critical)
func TestDLQMessageProcessingFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Test: Validate event type
	eventType := "payment_failed"
	if err := ValidateEventType(eventType); err != nil {
		t.Errorf("event type validation failed: %v", err)
	}

	// Test: Validate message ID
	messageID := "msg-123-abc"
	if err := ValidateMessageID(messageID); err != nil {
		t.Errorf("message ID validation failed: %v", err)
	}

	// Test: Validate retry count
	retryCount := 2
	if err := ValidateMaxRetries(retryCount); err != nil {
		t.Errorf("retry count validation failed: %v", err)
	}

	// Test: Classify error as retryable
	errMsg := "temporary network timeout"
	if !IsRetryableError(errMsg) {
		t.Errorf("error should be classified as retryable")
	}
}
