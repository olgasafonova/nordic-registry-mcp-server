package finland

import (
	"fmt"
	"strconv"
	"strings"
)

// ValidateBusinessID validates a Finnish business ID (Y-tunnus).
// Format: 7 digits + hyphen + check digit (e.g., 0112038-9)
// Uses NormalizeBusinessID for format validation and adds check digit verification.
func ValidateBusinessID(businessID string) error {
	if businessID == "" {
		return fmt.Errorf("business ID is required")
	}

	// Use existing normalization which validates format
	normalized, err := NormalizeBusinessID(businessID)
	if err != nil {
		return err
	}

	// Additional check digit validation using the official algorithm
	// Weights: 7, 9, 10, 5, 8, 4, 2
	parts := strings.Split(normalized, "-")
	if len(parts) != 2 {
		return fmt.Errorf("invalid Finnish business ID format: %s", businessID)
	}

	digits := parts[0]
	checkDigit, _ := strconv.Atoi(parts[1])

	weights := []int{7, 9, 10, 5, 8, 4, 2}
	sum := 0
	for i, w := range weights {
		d, _ := strconv.Atoi(string(digits[i]))
		sum += d * w
	}

	remainder := sum % 11
	var expectedCheck int
	if remainder == 0 {
		expectedCheck = 0
	} else if remainder == 1 {
		// Check digit 1 is not valid, this business ID would not be issued
		return fmt.Errorf("invalid Finnish business ID %q: invalid check digit", businessID)
	} else {
		expectedCheck = 11 - remainder
	}

	if checkDigit != expectedCheck {
		return fmt.Errorf("invalid Finnish business ID %q: check digit should be %d", businessID, expectedCheck)
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
