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

		// Valid business ID with check digit = 0 (remainder = 0 case)
		// 2255546-0: 2*7 + 2*9 + 5*10 + 5*5 + 5*8 + 4*4 + 6*2 = 14+18+50+25+40+16+12 = 175
		// 175 mod 11 = 10, so 11-10 = 1... not 0. Let me find another.
		// 0000007-0: 0*7+0*9+0*10+0*5+0*8+0*4+7*2 = 14, 14 mod 11 = 3, 11-3=8 (not 0)
		// 1111111-0: 1*7+1*9+1*10+1*5+1*8+1*4+1*2 = 45, 45 mod 11 = 1 (invalid case!)
		// 2222222-0: 2*7+2*9+2*10+2*5+2*8+2*4+2*2 = 90, 90 mod 11 = 2, 11-2=9 (not 0)
		// 1100077-0: 1*7+1*9+0*10+0*5+0*8+7*4+7*2 = 7+9+0+0+0+28+14 = 58, 58 mod 11 = 3, 11-3=8 (not 0)
		// 0000011-0: 0+0+0+0+0+4+2 = 6, 6 mod 11 = 6, 11-6=5 (not 0)
		// To get remainder=0, sum must be divisible by 11
		// 1234561-0: 7+18+30+20+40+24+2 = 141, 141 mod 11 = 9, 11-9=2 (not 0)
		// 2234567-0: 14+18+30+20+40+24+14 = 160, 160 mod 11 = 6, 11-6=5 (not 0)
		// 3333333-0: 3*(7+9+10+5+8+4+2) = 3*45 = 135, 135 mod 11 = 3, 11-3=8 (not 0)
		// 5555555-0: 5*45 = 225, 225 mod 11 = 5, 11-5=6 (not 0)
		// Let's find one: sum mod 11 = 0 -> sum = 0, 11, 22, 33, 44, 55, 66, 77, 88...
		// Minimum sum for 7 digits (each 0-9): 0. Maximum: 9*(7+9+10+5+8+4+2) = 9*45 = 405
		// 0000000-0: sum=0, 0 mod 11 = 0, check = 0 -> VALID
		{"valid check digit 0", "0000000-0", false},

		// Invalid formats
		{"empty", "", true},
		{"no hyphen", "01120389", true},
		{"too few digits", "112038-9", true},
		{"too many digits", "01120380-9", true},
		{"letters in digits", "011203A-9", true},

		// Invalid check digits
		{"wrong check digit", "0112038-1", true},
		{"wrong check digit 2", "1927400-5", true},

		// Check digit that would be 10 (impossible, remainder=1 case)
		// Need sum mod 11 = 1 -> sum = 1, 12, 23, 34, 45, 56...
		// 1111111-X: 45 mod 11 = 1 -> would need check digit 10 (invalid)
		{"remainder 1 case", "1111111-0", true}, // Any check digit is wrong for this
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
