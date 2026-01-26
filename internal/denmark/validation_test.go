package denmark

import "testing"

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
