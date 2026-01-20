package finland

import "testing"

func TestValidateBusinessID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid business IDs (with correct check digits)
		{"valid Nokia", "0112038-9", false},
		{"valid Kone", "1927400-1", false},

		// Invalid formats
		{"empty", "", true},
		{"no hyphen", "01120389", true},
		{"too few digits", "112038-9", true},
		{"too many digits", "01120380-9", true},
		{"letters in digits", "011203A-9", true},

		// Invalid check digits
		{"wrong check digit", "0112038-1", true},
		{"wrong check digit 2", "1927400-5", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBusinessID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBusinessID(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
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
		{"valid query", "Nokia", false},
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
