package models

import "fmt"

// ValidationError represents a validation failure for a model.
type ValidationError struct {
	Entity  string // Which entity type (e.g., "agent", "fragment")
	Message string // What went wrong
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s validation error: %s", e.Entity, e.Message)
}

// NewValidationError creates a new ValidationError.
func NewValidationError(entity, format string, args ...interface{}) *ValidationError {
	return &ValidationError{
		Entity:  entity,
		Message: fmt.Sprintf(format, args...),
	}
}

// NotFoundError represents an entity that was not found.
type NotFoundError struct {
	Entity string
	ID     string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.Entity, e.ID)
}

// NewNotFoundError creates a new NotFoundError.
func NewNotFoundError(entity, id string) *NotFoundError {
	return &NotFoundError{
		Entity: entity,
		ID:     id,
	}
}

// ConflictError represents a conflict (e.g., duplicate entry).
type ConflictError struct {
	Entity  string
	Message string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("%s conflict: %s", e.Entity, e.Message)
}

// NewConflictError creates a new ConflictError.
func NewConflictError(entity, format string, args ...interface{}) *ConflictError {
	return &ConflictError{
		Entity:  entity,
		Message: fmt.Sprintf(format, args...),
	}
}
