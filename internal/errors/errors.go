// Package errors provides shared error types for Nordic registry clients.
package errors

import (
	"fmt"
)

// NotFoundError indicates an entity was not found in a registry.
type NotFoundError struct {
	Country    string // "norway", "denmark", "finland"
	EntityType string // "company", "subunit", "production_unit"
	Identifier string // org number, CVR, business ID, or search query
}

func (e *NotFoundError) Error() string {
	if e.EntityType != "" {
		return fmt.Sprintf("%s not found in %s registry: %s", e.EntityType, e.Country, e.Identifier)
	}
	return fmt.Sprintf("not found in %s registry: %s", e.Country, e.Identifier)
}

// NewNotFoundError creates a NotFoundError for a company lookup.
func NewNotFoundError(country, identifier string) *NotFoundError {
	return &NotFoundError{
		Country:    country,
		EntityType: "company",
		Identifier: identifier,
	}
}

// ValidationError indicates invalid input parameters.
type ValidationError struct {
	Field   string // field name that failed validation
	Value   string // the invalid value (may be empty for sensitive data)
	Message string // human-readable error message
}

func (e *ValidationError) Error() string {
	if e.Field != "" && e.Value != "" {
		return fmt.Sprintf("validation failed for %s=%q: %s", e.Field, e.Value, e.Message)
	}
	if e.Field != "" {
		return fmt.Sprintf("validation failed for %s: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation failed: %s", e.Message)
}

// NewValidationError creates a ValidationError.
func NewValidationError(field, value, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// IsNotFound returns true if the error is a NotFoundError.
func IsNotFound(err error) bool {
	_, ok := err.(*NotFoundError)
	return ok
}

// IsValidation returns true if the error is a ValidationError.
func IsValidation(err error) bool {
	_, ok := err.(*ValidationError)
	return ok
}
