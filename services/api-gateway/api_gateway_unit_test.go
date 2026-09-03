package main

import (
	"testing"

	"github.com/redis/go-redis/v9"
)

// TestNewRateLimiter validates rate limiter initialization
func TestNewRateLimiter(t *testing.T) {
	client := &redis.Client{}
	rl := NewRateLimiter(client)

	if rl == nil {
		t.Fatal("expected RateLimiter to not be nil")
	}

	if rl.rdb != client {
		t.Fatal("expected RateLimiter.rdb to reference the provided redis client")
	}
}

// TestExtractIPAddress validates IP extraction from various request formats
func TestExtractIPAddress(t *testing.T) {
	tests := []struct {
		name           string
		xForwardedFor  string
		remoteAddr     string
		expectedIP     string
	}{
		{
			name:           "x-forwarded-for_single_ip",
			xForwardedFor:  "192.168.1.100",
			remoteAddr:     "10.0.0.1:8080",
			expectedIP:     "192.168.1.100",
		},
		{
			name:           "x-forwarded-for_multiple_ips",
			xForwardedFor:  "192.168.1.100, 10.0.0.1, 172.16.0.1",
			remoteAddr:     "10.0.0.2:8080",
			expectedIP:     "192.168.1.100",
		},
		{
			name:           "no_x-forwarded-for",
			xForwardedFor:  "",
			remoteAddr:     "192.168.1.50:9090",
			expectedIP:     "192.168.1.50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := ExtractIPAddress(tt.xForwardedFor, tt.remoteAddr)
			if ip != tt.expectedIP {
				t.Errorf("ExtractIPAddress got %s, expected %s", ip, tt.expectedIP)
			}
		})
	}
}

// TestValidateEmail validates email format
func TestValidateEmail(t *testing.T) {
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
			name:    "valid_email_subdomain",
			email:   "user.name@mail.example.co.uk",
			wantErr: false,
		},
		{
			name:    "invalid_email_no_at",
			email:   "userexample.com",
			wantErr: true,
		},
		{
			name:    "invalid_email_no_domain",
			email:   "user@",
			wantErr: true,
		},
		{
			name:    "empty_email",
			email:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEmail(%s) error = %v, wantErr %v",
					tt.email, err, tt.wantErr)
			}
		})
	}
}

// TestValidatePassword validates password requirements
func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name      string
		password  string
		wantErr   bool
	}{
		{
			name:     "valid_password_8_chars",
			password: "SecureP@ss123",
			wantErr:  false,
		},
		{
			name:     "valid_password_long",
			password: "VeryLongSecurePassword123!@#",
			wantErr:  false,
		},
		{
			name:     "invalid_password_too_short",
			password: "Short1!",
			wantErr:  true,
		},
		{
			name:     "invalid_password_empty",
			password: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword error = %v, wantErr %v",
					err, tt.wantErr)
			}
		})
	}
}

// TestValidatePhoneNumber validates phone number format
func TestValidatePhoneNumber(t *testing.T) {
	tests := []struct {
		name      string
		phone     string
		wantErr   bool
	}{
		{
			name:    "valid_indian_phone",
			phone:   "9876543210",
			wantErr: false,
		},
		{
			name:    "valid_phone_with_country",
			phone:   "+919876543210",
			wantErr: false,
		},
		{
			name:    "invalid_phone_too_short",
			phone:   "98765432",
			wantErr: true,
		},
		{
			name:    "invalid_phone_letters",
			phone:   "98765432abc",
			wantErr: true,
		},
		{
			name:    "empty_phone",
			phone:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePhoneNumber(tt.phone)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePhoneNumber error = %v, wantErr %v",
					err, tt.wantErr)
			}
		})
	}
}
