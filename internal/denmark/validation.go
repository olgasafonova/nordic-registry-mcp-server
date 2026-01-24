package denmark

import (
	"fmt"
	"regexp"
)

var cvrRegex = regexp.MustCompile(`^\d{8}$`)

// ValidateCVR validates a Danish CVR number.
// Danish CVR numbers are exactly 8 digits.
func ValidateCVR(cvr string) error {
	if cvr == "" {
		return fmt.Errorf("CVR number is required")
	}
	if !cvrRegex.MatchString(cvr) {
		return fmt.Errorf("invalid Danish CVR number %q: must be exactly 8 digits", cvr)
	}
	return nil
}

// MaxQueryLength is the maximum allowed search query length
const MaxQueryLength = 500

// ValidateSearchQuery validates a search query.
func ValidateSearchQuery(query string) error {
	if query == "" {
		return fmt.Errorf("search query is required")
	}
	if len(query) < 2 {
		return fmt.Errorf("search query must be at least 2 characters")
	}
	if len(query) > MaxQueryLength {
		return fmt.Errorf("search query exceeds maximum length of %d characters", MaxQueryLength)
	}
	return nil
}

// phoneRegex matches valid phone numbers (digits only after normalization)
var phoneRegex = regexp.MustCompile(`^\d{8,15}$`)

// ValidatePhone validates a Danish phone number.
// After normalization, phone numbers should be 8-15 digits.
func ValidatePhone(phone string) error {
	if phone == "" {
		return fmt.Errorf("phone number is required")
	}
	if !phoneRegex.MatchString(phone) {
		return fmt.Errorf("invalid phone number %q: must be 8-15 digits", phone)
	}
	return nil
}

// pNumberRegex matches valid P-numbers (production unit numbers)
var pNumberRegex = regexp.MustCompile(`^\d{10}$`)

// ValidatePNumber validates a Danish P-number (production unit number).
// P-numbers are exactly 10 digits.
func ValidatePNumber(pnumber string) error {
	if pnumber == "" {
		return fmt.Errorf("P-number is required")
	}
	if !pNumberRegex.MatchString(pnumber) {
		return fmt.Errorf("invalid P-number %q: must be exactly 10 digits", pnumber)
	}
	return nil
}
