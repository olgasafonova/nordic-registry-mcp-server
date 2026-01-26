package denmark

import (
	"strings"
	"testing"
)

func TestValidateCVR(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid 8 digits", "10150817", false},
		{"valid with leading zeros", "01234567", false},
		{"empty", "", true},
		{"too short", "1234567", true},
		{"too long", "123456789", true},
		{"contains letters", "1015081A", true},
		// Spaces, dashes, and DK prefix are now normalized (removed) before validation
		{"contains spaces", "10 150 817", false},
		{"contains dash", "10-150-817", false},
		{"with DK prefix", "DK10150817", false},
		{"with DK prefix and dash", "DK-10150817", false},
		{"lowercase dk prefix", "dk10150817", false},
		{"mixed format", "DK 10-15-08-17", false},
		{"with leading/trailing spaces", "  10150817  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCVR(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCVR(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateAndNormalizeCVR(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantNormalized string
		wantErr        bool
	}{
		{"valid 8 digits", "10150817", "10150817", false},
		{"with DK prefix", "DK10150817", "10150817", false},
		{"lowercase dk prefix", "dk10150817", "10150817", false},
		{"with spaces", "10 150 817", "10150817", false},
		{"with dashes", "10-150-817", "10150817", false},
		{"with DK prefix and dash", "DK-10150817", "10150817", false},
		{"mixed format", "DK 10-15-08-17", "10150817", false},
		{"with leading/trailing spaces", "  10150817  ", "10150817", false},
		{"empty", "", "", true},
		{"too short", "1234567", "", true},
		{"too long", "123456789", "", true},
		{"contains letters", "1015081A", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized, err := ValidateAndNormalizeCVR(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAndNormalizeCVR(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && normalized != tt.wantNormalized {
				t.Errorf("ValidateAndNormalizeCVR(%q) = %q, want %q", tt.input, normalized, tt.wantNormalized)
			}
		})
	}
}

func TestValidateSearchQuery(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid query", "Novo Nordisk", false},
		{"short query", "AB", false},
		{"empty", "", true},
		{"single char", "A", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSearchQuery(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSearchQuery(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateSearchQuery_MaxLength(t *testing.T) {
	// Test query at exactly max length - should pass
	exactMaxQuery := strings.Repeat("a", MaxQueryLength)
	if err := ValidateSearchQuery(exactMaxQuery); err != nil {
		t.Errorf("ValidateSearchQuery with %d chars should pass, got error: %v", MaxQueryLength, err)
	}

	// Test query exceeding max length - should fail
	overMaxQuery := strings.Repeat("a", MaxQueryLength+1)
	if err := ValidateSearchQuery(overMaxQuery); err == nil {
		t.Errorf("ValidateSearchQuery with %d chars should fail", MaxQueryLength+1)
	}
}

func TestValidatePhone(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid 8 digits", "44448888", false},
		{"valid 10 digits", "4412345678", false},
		{"valid 15 digits (max)", "123456789012345", false},
		{"empty", "", true},
		{"too short (7 digits)", "1234567", true},
		{"too long (16 digits)", "1234567890123456", true},
		{"contains letters", "4444888A", true},
		{"contains spaces (not normalized)", "4444 8888", true},
		{"contains dashes (not normalized)", "4444-8888", true},
		{"contains plus (not normalized)", "+4544448888", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePhone(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePhone(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePNumber(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid 10 digits", "1234567890", false},
		{"valid with leading zeros", "0123456789", false},
		{"empty", "", true},
		{"too short (9 digits)", "123456789", true},
		{"too long (11 digits)", "12345678901", true},
		{"contains letters", "123456789A", true},
		{"contains spaces (not normalized)", "1234 567890", true},
		{"contains dashes (not normalized)", "1234-567890", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePNumber(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePNumber(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}
