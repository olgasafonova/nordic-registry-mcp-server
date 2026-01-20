package norway

import (
	"fmt"
	"regexp"
)

var orgNumberRegex = regexp.MustCompile(`^\d{9}$`)

// ValidateOrgNumber validates a Norwegian organization number.
// Norwegian org numbers are exactly 9 digits.
func ValidateOrgNumber(orgNumber string) error {
	if orgNumber == "" {
		return fmt.Errorf("organization number is required")
	}
	if !orgNumberRegex.MatchString(orgNumber) {
		return fmt.Errorf("invalid Norwegian organization number %q: must be exactly 9 digits", orgNumber)
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

// ValidateSize validates the page size parameter.
func ValidateSize(size int) error {
	if size < 0 {
		return fmt.Errorf("size cannot be negative")
	}
	if size > 100 {
		return fmt.Errorf("size cannot exceed 100")
	}
	return nil
}
