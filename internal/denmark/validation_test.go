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
		{"contains spaces", "10 150 817", true},
		{"contains dash", "10-150-817", true},
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
