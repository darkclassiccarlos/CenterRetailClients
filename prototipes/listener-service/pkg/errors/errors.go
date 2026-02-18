package errors

import (
	"fmt"
	"net/http"
)

// StandardError represents a standardized error response
type StandardError struct {
	Code    string `json:"error"`   // Error code/type (e.g., "InvalidRequest", "DatabaseError")
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
	case "DatabaseError", "InternalError":
		return http.StatusInternalServerError
	case "BrokerConnectionError", "ServiceUnavailable":
		return http.StatusServiceUnavailable
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
func NewDatabaseError(operation string, err error) *StandardError {
	return NewStandardError("DatabaseError", fmt.Sprintf("database operation failed: %s", operation), err.Error())
}

func NewBrokerConnectionError(err error) *StandardError {
	return NewStandardError("BrokerConnectionError", "failed to connect to event broker", err.Error())
}

func NewInternalError(message string, err error) *StandardError {
	details := ""
	if err != nil {
		details = err.Error()
	}
	return NewStandardError("InternalError", message, details)
}
