package service

import (
	"testing"
)

// TestValidateSignupEmail validates email in signup
func TestValidateSignupEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{
			name:    "valid_email",
			email:   "user@example.com",
			wantErr: false,
		},
		{
			name:    "empty_email",
			email:   "",
			wantErr: true,
		},
		{
			name:    "invalid_email_no_at",
			email:   "userexample.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSignupEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSignupEmail error = %v, wantErr %v",
					err, tt.wantErr)
			}
		})
	}
}

// TestValidatePasswordStrength validates password strength requirements
func TestValidatePasswordStrength(t *testing.T) {
	tests := []struct {
		name    string
		pwd     string
		wantErr bool
	}{
		{
			name:    "valid_strong_password",
			pwd:     "SecurePass123!",
			wantErr: false,
		},
		{
			name:    "password_too_short",
			pwd:     "Short1",
			wantErr: true,
		},
		{
			name:    "empty_password",
			pwd:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePasswordStrength(tt.pwd)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePasswordStrength error = %v, wantErr %v",
					err, tt.wantErr)
			}
		})
	}
}

// TestValidateUserRole validates user role
func TestValidateUserRole(t *testing.T) {
	tests := []struct {
		name    string
		role    string
		wantErr bool
	}{
		{
			name:    "valid_role_rider",
			role:    "rider",
			wantErr: false,
		},
		{
			name:    "valid_role_driver",
			role:    "driver",
			wantErr: false,
		},
		{
			name:    "invalid_role",
			role:    "admin",
			wantErr: true,
		},
		{
			name:    "empty_role",
			role:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUserRole(tt.role)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUserRole error = %v, wantErr %v",
					err, tt.wantErr)
			}
		})
	}
}

// TestGenerateTokens validates JWT token generation
func TestGenerateTokens(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		email   string
		role    string
		wantErr bool
	}{
		{
			name:    "valid_token_generation",
			userID:  "user123",
			email:   "user@example.com",
			role:    "rider",
			wantErr: false,
		},
		{
			name:    "empty_user_id",
			userID:  "",
			email:   "user@example.com",
			role:    "rider",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accessToken, refreshToken, err := GenerateTokens(tt.userID, tt.email, tt.role)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateTokens error = %v, wantErr %v",
					err, tt.wantErr)
			}
			if !tt.wantErr {
				if accessToken == "" || refreshToken == "" {
					t.Error("expected non-empty access and refresh tokens")
				}
			}
		})
	}
}
