package errors

import (
	"fmt"
	"net/http"
)

// StandardError represents a standardized error response
type StandardError struct {
	Code    string `json:"error"`   // Error code/type (e.g., "InvalidRequest", "ItemNotFound")
	Message string `json:"message"` // Human-readable error message
	Details string `json:"details"` // Additional details (field name, validation info, etc.)
}

// Error implements the error interface
func (e *StandardError) Error() string {
	return e.Message
}

// HTTPStatus returns the appropriate HTTP status code for the error
func (e *StandardError) HTTPStatus() int {
	switch e.Code {
	case "InvalidRequest", "ValidationError":
		return http.StatusBadRequest
	case "ItemNotFound", "ResourceNotFound":
		return http.StatusNotFound
	case "DatabaseError", "InternalError":
		return http.StatusInternalServerError
	case "CacheError":
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// NewStandardError creates a new StandardError
func NewStandardError(errorCode, message, details string) *StandardError {
	return &StandardError{
		Code:    errorCode,
		Message: message,
		Details: details,
	}
}

// Common error constructors
func NewInvalidRequest(message, details string) *StandardError {
	return NewStandardError("InvalidRequest", message, details)
}

func NewValidationError(message, field string) *StandardError {
	return NewStandardError("ValidationError", message, fmt.Sprintf("Field: %s", field))
}

func NewItemNotFound(itemID string) *StandardError {
	return NewStandardError("ItemNotFound", "item not found", fmt.Sprintf("Item ID: %s", itemID))
}

func NewDatabaseError(operation string, err error) *StandardError {
	return NewStandardError("DatabaseError", fmt.Sprintf("database operation failed: %s", operation), err.Error())
}

func NewCacheError(operation string, err error) *StandardError {
	return NewStandardError("CacheError", fmt.Sprintf("cache operation failed: %s", operation), err.Error())
}

func NewInternalError(message string, err error) *StandardError {
	details := ""
	if err != nil {
		details = err.Error()
	}
	return NewStandardError("InternalError", message, details)
}
