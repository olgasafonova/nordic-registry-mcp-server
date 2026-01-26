package norway

import "testing"

func TestValidateOrgNumber(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid 9 digits", "923609016", false},
		{"valid with leading zeros", "012345678", false},
		{"empty", "", true},
		{"too short", "12345678", true},
		{"too long", "1234567890", true},
		{"contains letters", "92360901A", true},
		// Spaces and dashes are now normalized (removed) before validation
		{"contains spaces", "923 609 016", false},
		{"contains dash", "923-609-016", false},
		{"mixed format", "923-609 016", false},
		{"with leading/trailing spaces", "  923609016  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOrgNumber(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOrgNumber(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
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
		{"valid query", "Equinor", false},
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

func TestValidateSize(t *testing.T) {
	tests := []struct {
		name    string
		input   int
		wantErr bool
	}{
		{"valid size", 50, false},
		{"zero", 0, false},
		{"max size", 100, false},
		{"negative", -1, true},
		{"too large", 101, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSize(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSize(%d) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}
