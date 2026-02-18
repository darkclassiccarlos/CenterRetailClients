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
	case "DuplicateSKU", "Conflict":
		return http.StatusConflict
	case "InsufficientStock", "InvalidOperation":
		return http.StatusBadRequest
	case "BrokerConnectionError", "ServiceUnavailable":
		return http.StatusServiceUnavailable
	case "SerializationError", "DatabaseError", "InternalError":
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

func NewDuplicateSKU(sku string) *StandardError {
	return NewStandardError("DuplicateSKU", "sku already exists", fmt.Sprintf("SKU: %s", sku))
}

func NewInsufficientStock(available, requested int) *StandardError {
	return NewStandardError("InsufficientStock", "insufficient stock available",
		fmt.Sprintf("Available: %d, Requested: %d", available, requested))
}

func NewInvalidReleaseQuantity(reserved, requested int) *StandardError {
	return NewStandardError("InvalidOperation", "invalid release quantity",
		fmt.Sprintf("Reserved: %d, Requested: %d", reserved, requested))
}

func NewSerializationError(err error) *StandardError {
	return NewStandardError("SerializationError", "failed to serialize data", err.Error())
}

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
