package finland

import (
	"strconv"
	"strings"

	apierrors "github.com/olgasafonova/nordic-registry-mcp-server/internal/errors"
)

// ValidateBusinessID validates a Finnish business ID (Y-tunnus).
// Format: 7 digits + hyphen + check digit (e.g., 0112038-9)
// Uses NormalizeBusinessID for format validation and adds check digit verification.
func ValidateBusinessID(businessID string) error {
	if businessID == "" {
		return apierrors.NewValidationError("business_id", "", "is required")
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
		return apierrors.NewValidationError("business_id", businessID, "invalid format")
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
		return apierrors.NewValidationError("business_id", businessID, "invalid check digit")
	} else {
		expectedCheck = 11 - remainder
	}

	if checkDigit != expectedCheck {
		return apierrors.NewValidationError("business_id", businessID, "incorrect check digit")
	}

	return nil
}

// ValidateSearchQuery validates a search query.
func ValidateSearchQuery(query string) error {
	if query == "" {
		return apierrors.NewValidationError("query", "", "is required")
	}
	if len(query) < 2 {
		return apierrors.NewValidationError("query", query, "must be at least 2 characters")
	}
	return nil
}
