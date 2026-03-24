package http

import (
	"errors"
	"fmt"
	"net/http"
)

// DefaultOperationName is the default name for operations.
const DefaultOperationName = "operation"

// ErrUnauthorized is returned when authentication fails (e.g., invalid JWT token).
var ErrUnauthorized = errors.New("unauthorized")

// errorFactory is a helper for creating errors with optional ErrUnauthorized wrapping.
type errorFactory struct {
	wrapErr bool
	message string
}

// Error creates the actual error.
func (ef errorFactory) Error() error {
	if ef.wrapErr {
		return fmt.Errorf("%w: %s", ErrUnauthorized, ef.message)
	}
	return fmt.Errorf("%s", ef.message)
}

// authErr creates an error factory that wraps with ErrUnauthorized.
func authErr(msg string) errorFactory {
	return errorFactory{wrapErr: true, message: msg}
}

// plainErr creates an error factory without wrapping.
func plainErr(msg string) errorFactory {
	return errorFactory{wrapErr: false, message: msg}
}

// ErrorMessages maps HTTP status codes to error messages for operations.
type ErrorMessages map[int]map[int]errorFactory

// OpHandler handles operation ID-based error mapping.
type OpHandler struct {
	Messages   ErrorMessages
	Names      map[int]string
	Default    string
}

// NewOpHandler creates a new operation handler.
func NewOpHandler() *OpHandler {
	return &OpHandler{
		Messages: make(ErrorMessages),
		Names:   make(map[int]string),
		Default: DefaultOperationName,
	}
}

// AddOperation adds error messages for an operation.
func (oh *OpHandler) AddOperation(opID int, errorMap map[int]errorFactory) {
	oh.Messages[opID] = errorMap
}

// SetName sets the name for an operation ID.
func (oh *OpHandler) SetName(opID int, name string) {
	oh.Names[opID] = name
}

// HandleResponse handles an HTTP response, returning an error if appropriate.
func (oh *OpHandler) HandleResponse(statusCode int, body []byte, opID int, successCodes []int) error {
	// Check if status code is in success codes
	for _, code := range successCodes {
		if statusCode == code {
			return nil
		}
	}

	// Check for custom error message in global map
	if errorMessages, ok := oh.Messages[opID]; ok {
		if factory, ok := errorMessages[statusCode]; ok {
			return factory.Error()
		}
	}

	// Get operation name for generic error
	opName := oh.Names[opID]
	if opName == "" {
		opName = oh.Default
	}

	// Generic error with body
	return fmt.Errorf("%s failed with status %d: %s", opName, statusCode, string(body))
}

// AuthError creates an unauthorized error factory.
func AuthError(msg string) errorFactory {
	return authErr(msg)
}

// PlainError creates a plain error factory.
func PlainError(msg string) errorFactory {
	return plainErr(msg)
}

// ValidateJSON200 validates the HTTP status code and JSON200 field.
func ValidateJSON200[T any](statusCode int, body []byte, json200 *T, nilMsg string) (*T, error) {
	if statusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("%w: authentication required", ErrUnauthorized)
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("failed with status %d: %s", statusCode, string(body))
	}
	if json200 == nil {
		return nil, fmt.Errorf("%s", nilMsg)
	}
	return json200, nil
}
