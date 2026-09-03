package service

import (
	"testing"
)

// TestValidateMessageContent validates message content constraints
func TestValidateMessageContent(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		maxLength  int
		wantErr    bool
	}{
		{
			name:      "valid_message",
			content:   "Hello, this is a test message",
			maxLength: 1000,
			wantErr:   false,
		},
		{
			name:      "empty_message",
			content:   "",
			maxLength: 1000,
			wantErr:   true,
		},
		{
			name:      "message_exceeds_max_length",
			content:   "This is a very long message that exceeds the maximum allowed length",
			maxLength: 10,
			wantErr:   true,
		},
		{
			name:      "message_at_max_length",
			content:   "12345",
			maxLength: 5,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMessageContent(tt.content, tt.maxLength)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMessageContent error = %v, wantErr %v",
					err, tt.wantErr)
			}
		})
	}
}

// TestValidateTripID validates trip ID format
func TestValidateTripID(t *testing.T) {
	tests := []struct {
		name    string
		tripID  string
		wantErr bool
	}{
		{
			name:    "valid_trip_id",
			tripID:  "507f1f77bcf86cd799439011",
			wantErr: false,
		},
		{
			name:    "empty_trip_id",
			tripID:  "",
			wantErr: true,
		},
		{
			name:    "invalid_trip_id_format",
			tripID:  "invalid-id",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTripID(tt.tripID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTripID error = %v, wantErr %v",
					err, tt.wantErr)
			}
		})
	}
}

// TestValidateParticipant validates participant ID format
func TestValidateParticipant(t *testing.T) {
	tests := []struct {
		name         string
		participantID string
		wantErr      bool
	}{
		{
			name:         "valid_participant",
			participantID: "user123",
			wantErr:      false,
		},
		{
			name:         "empty_participant",
			participantID: "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateParticipant(tt.participantID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateParticipant error = %v, wantErr %v",
					err, tt.wantErr)
			}
		})
	}
}
