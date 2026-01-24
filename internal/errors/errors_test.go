package errors

import (
	"errors"
	"testing"
)

func TestNotFoundError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *NotFoundError
		expected string
	}{
		{
			name: "with entity type",
			err: &NotFoundError{
				Country:    "norway",
				EntityType: "company",
				Identifier: "923609016",
			},
			expected: "company not found in norway registry: 923609016",
		},
		{
			name: "without entity type",
			err: &NotFoundError{
				Country:    "denmark",
				Identifier: "10150817",
			},
			expected: "not found in denmark registry: 10150817",
		},
		{
			name: "with search query",
			err: &NotFoundError{
				Country:    "finland",
				EntityType: "company",
				Identifier: "Nokia",
			},
			expected: "company not found in finland registry: Nokia",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("NotFoundError.Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNewNotFoundError(t *testing.T) {
	err := NewNotFoundError("norway", "923609016")

	if err.Country != "norway" {
		t.Errorf("Country = %q, want %q", err.Country, "norway")
	}
	if err.EntityType != "company" {
		t.Errorf("EntityType = %q, want %q", err.EntityType, "company")
	}
	if err.Identifier != "923609016" {
		t.Errorf("Identifier = %q, want %q", err.Identifier, "923609016")
	}
}

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ValidationError
		expected string
	}{
		{
			name: "with field and value",
			err: &ValidationError{
				Field:   "org_number",
				Value:   "123",
				Message: "must be 9 digits",
			},
			expected: "validation failed for org_number=\"123\": must be 9 digits",
		},
		{
			name: "with field only",
			err: &ValidationError{
				Field:   "business_id",
				Message: "is required",
			},
			expected: "validation failed for business_id: is required",
		},
		{
			name: "message only",
			err: &ValidationError{
				Message: "invalid input",
			},
			expected: "validation failed: invalid input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("ValidationError.Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("cvr", "abc", "must be 8 digits")

	if err.Field != "cvr" {
		t.Errorf("Field = %q, want %q", err.Field, "cvr")
	}
	if err.Value != "abc" {
		t.Errorf("Value = %q, want %q", err.Value, "abc")
	}
	if err.Message != "must be 8 digits" {
		t.Errorf("Message = %q, want %q", err.Message, "must be 8 digits")
	}
}

func TestIsNotFound(t *testing.T) {
	notFoundErr := &NotFoundError{Country: "norway", Identifier: "123"}
	validationErr := &ValidationError{Message: "test"}
	plainErr := errors.New("plain error")

	if !IsNotFound(notFoundErr) {
		t.Error("IsNotFound should return true for NotFoundError")
	}
	if IsNotFound(validationErr) {
		t.Error("IsNotFound should return false for ValidationError")
	}
	if IsNotFound(plainErr) {
		t.Error("IsNotFound should return false for plain error")
	}
	if IsNotFound(nil) {
		t.Error("IsNotFound should return false for nil")
	}
}

func TestIsValidation(t *testing.T) {
	notFoundErr := &NotFoundError{Country: "norway", Identifier: "123"}
	validationErr := &ValidationError{Message: "test"}
	plainErr := errors.New("plain error")

	if IsValidation(notFoundErr) {
		t.Error("IsValidation should return false for NotFoundError")
	}
	if !IsValidation(validationErr) {
		t.Error("IsValidation should return true for ValidationError")
	}
	if IsValidation(plainErr) {
		t.Error("IsValidation should return false for plain error")
	}
	if IsValidation(nil) {
		t.Error("IsValidation should return false for nil")
	}
}
