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

// ValidateSearchQuery validates a search query.
func ValidateSearchQuery(query string) error {
	if query == "" {
		return fmt.Errorf("search query is required")
	}
	if len(query) < 2 {
		return fmt.Errorf("search query must be at least 2 characters")
	}
	return nil
}
