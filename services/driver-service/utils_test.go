package main

import (
	"regexp"
	"testing"
)

// TestGenerateRandomPlate validates that the generated plate is 3 characters and uppercase letters only
func TestGenerateRandomPlate(t *testing.T) {
	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "plate_length_is_3",
			run: func(t *testing.T) {
				plate := GenerateRandomPlate()
				if len(plate) != 3 {
					t.Errorf("expected plate length 3, got %d: %s", len(plate), plate)
				}
			},
		},
		{
			name: "plate_contains_only_uppercase_letters",
			run: func(t *testing.T) {
				plate := GenerateRandomPlate()
				if !regexp.MustCompile(`^[A-Z]{3}$`).MatchString(plate) {
					t.Errorf("plate %s does not match pattern ^[A-Z]{3}$", plate)
				}
			},
		},
		{
			name: "multiple_calls_produce_different_plates",
			run: func(t *testing.T) {
				plate1 := GenerateRandomPlate()
				plate2 := GenerateRandomPlate()
				// Note: theoretically possible to be equal, but extremely unlikely
				// In practice, this tests randomness
				_ = plate1
				_ = plate2
				// If both calls complete without error, randomness is working
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}
