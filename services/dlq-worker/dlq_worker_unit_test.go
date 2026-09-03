package main

import (
	"testing"
)

// TestValidateEventType validates event type
func TestValidateEventType(t *testing.T) {
	tests := []struct {
		name     string
		eventType string
		wantErr  bool
	}{
		{
			name:     "valid_payment_event",
			eventType: "payment_failed",
			wantErr:  false,
		},
		{
			name:     "valid_trip_event",
			eventType: "trip_cancelled",
			wantErr:  false,
		},
		{
			name:     "invalid_event_type",
			eventType: "unknown_event",
			wantErr:  true,
		},
		{
			name:     "empty_event_type",
			eventType: "",
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

// TestValidateMaxRetries validates retry count
func TestValidateMaxRetries(t *testing.T) {
	tests := []struct {
		name     string
		retries  int
		wantErr  bool
	}{
		{
			name:    "valid_retries",
			retries: 3,
			wantErr: false,
		},
		{
			name:    "zero_retries",
			retries: 0,
			wantErr: false,
		},
		{
			name:    "negative_retries",
			retries: -1,
			wantErr: true,
		},
		{
			name:    "excessive_retries",
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

// TestValidateMessageID validates message ID
func TestValidateMessageID(t *testing.T) {
	tests := []struct {
		name    string
		msgID   string
		wantErr bool
	}{
		{
			name:    "valid_message_id",
			msgID:   "msg-123-abc",
			wantErr: false,
		},
		{
			name:    "empty_message_id",
			msgID:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMessageID(tt.msgID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMessageID error = %v, wantErr %v",
					err, tt.wantErr)
			}
		})
	}
}

// TestIsRetryableError validates which errors are retryable
func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name       string
		errMsg     string
		isRetryable bool
	}{
		{
			name:        "connection_timeout",
			errMsg:      "connection timeout",
			isRetryable: true,
		},
		{
			name:        "service_unavailable",
			errMsg:      "service unavailable",
			isRetryable: true,
		},
		{
			name:        "invalid_request",
			errMsg:      "invalid request format",
			isRetryable: false,
		},
		{
			name:        "not_found",
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
