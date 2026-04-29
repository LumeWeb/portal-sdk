package http

import (
	"errors"
	"fmt"
	"net/http"
)

// DefaultOperationName is the default name for operations.
const DefaultOperationName = "operation"

// Common HTTP error sentinels for programmatic error checking.
var (
	// ErrUnauthorized is returned when authentication fails (e.g., invalid JWT token).
	ErrUnauthorized = errors.New("unauthorized")

	// ErrNotFound is returned when a requested resource is not found.
	ErrNotFound = errors.New("not found")

	// ErrForbidden is returned when the user lacks permission for the operation.
	ErrForbidden = errors.New("forbidden")

	// ErrBadRequest is returned when the request is invalid or malformed.
	ErrBadRequest = errors.New("bad request")

	// ErrConflict is returned when the request conflicts with the current state.
	ErrConflict = errors.New("conflict")

	// ErrInternalServer is returned when the server encounters an unexpected error.
	ErrInternalServer = errors.New("internal server error")

	// ErrUnavailable is returned when the service is temporarily unavailable.
	ErrUnavailable = errors.New("service unavailable")
)

// ErrorFactoryError sentinels for HTTP response error mapping.
// These are used by SDK clients to map HTTP status codes to typed errors.
var (
	// FactoryErrAuthRequired is a sentinel for authentication required errors.
	FactoryErrAuthRequired = AuthError("authentication required")

	// FactoryErrInsufficientPermissions is a sentinel for insufficient permissions errors.
	FactoryErrInsufficientPermissions = ForbiddenError("insufficient permissions")

	// FactoryErrBadRequest is a sentinel for bad request errors.
	FactoryErrBadRequest = BadRequestError("bad request")

	// FactoryErrNotFound is a sentinel for not found errors.
	FactoryErrNotFound = NotFoundError("not found")

	// FactoryErrInternalServerError is a sentinel for internal server errors.
	FactoryErrInternalServerError = PlainError("internal server error")

	// FactoryErrUserNotFound is a sentinel for user not found errors.
	FactoryErrUserNotFound = NotFoundError("user not found")
)

// sentinelMap maps HTTP status codes to their corresponding sentinel errors.
var sentinelMap = map[int]error{
	http.StatusUnauthorized:         ErrUnauthorized,
	http.StatusForbidden:            ErrForbidden,
	http.StatusNotFound:             ErrNotFound,
	http.StatusBadRequest:           ErrBadRequest,
	http.StatusConflict:             ErrConflict,
	http.StatusInternalServerError:  ErrInternalServer,
	http.StatusServiceUnavailable:   ErrUnavailable,
}

// ErrorFactoryError is a helper for creating errors with optional sentinel wrapping.
type ErrorFactoryError struct {
	wrapErr  bool
	sentinel error
	message  string
}

// Error creates the actual error.
func (ef ErrorFactoryError) Error() error {
	if ef.wrapErr && ef.sentinel != nil {
		return fmt.Errorf("%w: %s", ef.sentinel, ef.message)
	}
	return fmt.Errorf("%s", ef.message)
}

// SentinelError returns the underlying sentinel error, if any.
func (ef ErrorFactoryError) SentinelError() error {
	if ef.wrapErr {
		return ef.sentinel
	}
	return nil
}

// authErr creates an error factory that wraps with ErrUnauthorized.
func authErr(msg string) ErrorFactoryError {
	return ErrorFactoryError{wrapErr: true, sentinel: ErrUnauthorized, message: msg}
}

// plainErr creates an error factory without wrapping.
func plainErr(msg string) ErrorFactoryError {
	return ErrorFactoryError{wrapErr: false, message: msg}
}

// sentinelWrappedErr creates an error factory that wraps with a specific sentinel.
func sentinelWrappedErr(sentinel error, msg string) ErrorFactoryError {
	return ErrorFactoryError{wrapErr: true, sentinel: sentinel, message: msg}
}

// ErrorMessages maps HTTP status codes to error messages for operations.
type ErrorMessages map[int]map[int]ErrorFactoryError

// OpHandler handles operation ID-based error mapping.
type OpHandler struct {
	Messages ErrorMessages
	Names    map[int]string
	Default  string
}

// NewOpHandler creates a new operation handler.
func NewOpHandler() *OpHandler {
	return &OpHandler{
		Messages: make(ErrorMessages),
		Names:    make(map[int]string),
		Default:  DefaultOperationName,
	}
}

// AddOperation adds error messages for an operation.
func (oh *OpHandler) AddOperation(opID int, errorMap map[int]ErrorFactoryError) {
	oh.Messages[opID] = errorMap
}

// SetName sets the name for an operation ID.
func (oh *OpHandler) SetName(opID int, name string) {
	oh.Names[opID] = name
}

// HandleResponse handles an HTTP response, returning an error if appropriate.
// op: the operation ID (used to lookup custom error messages)
// successCodes: status codes that indicate success (e.g., []int{http.StatusOK})
// Returns nil for success codes, custom error from global map, or generic error with body.
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

	// Check for standard HTTP sentinel to wrap
	if sentinel, ok := sentinelMap[statusCode]; ok {
		return fmt.Errorf("%w: %s", sentinel, string(body))
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
func AuthError(msg string) ErrorFactoryError {
	return authErr(msg)
}

// PlainError creates a plain error factory without sentinel wrapping.
func PlainError(msg string) ErrorFactoryError {
	return plainErr(msg)
}

// NotFoundError creates a not-found error factory wrapping ErrNotFound.
func NotFoundError(msg string) ErrorFactoryError {
	return sentinelWrappedErr(ErrNotFound, msg)
}

// ForbiddenError creates a forbidden error factory wrapping ErrForbidden.
func ForbiddenError(msg string) ErrorFactoryError {
	return sentinelWrappedErr(ErrForbidden, msg)
}

// BadRequestError creates a bad-request error factory wrapping ErrBadRequest.
func BadRequestError(msg string) ErrorFactoryError {
	return sentinelWrappedErr(ErrBadRequest, msg)
}

// ConflictError creates a conflict error factory wrapping ErrConflict.
func ConflictError(msg string) ErrorFactoryError {
	return sentinelWrappedErr(ErrConflict, msg)
}

// Predefined ErrorFactoryError sentinels for commonly used error messages.
// These eliminate repetitive string literals in error mappings.
var (
	// Authentication/authorization errors
	ErrAuthRequired            = AuthError("authentication required")
	ErrInsufficientPermissions = ForbiddenError("insufficient permissions")
)

// JSON200Validator is a functional interface for validating JSON 200 responses using an OpHandler.
type JSON200Validator[T any] func(oh *OpHandler, statusCode int, body []byte, json200 *T, opID int) (*T, error)

// JSON201Validator is a functional interface for validating JSON 201 responses using an OpHandler.
type JSON201Validator[T any] func(oh *OpHandler, statusCode int, body []byte, json201 *T, opID int) (*T, error)

// ValidateJSON200 validates the HTTP status code and JSON200 field using the OpHandler for error mapping.
// Returns the JSON data on success, or an error with proper sentinel wrapping.
func ValidateJSON200[T any](oh *OpHandler, statusCode int, body []byte, json200 *T, opID int) (*T, error) {
	if statusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("%w: authentication required", ErrUnauthorized)
	}

	// For non-OK responses, check the error map first
	if statusCode != http.StatusOK {
		if errorMessages, ok := oh.Messages[opID]; ok {
			if factory, ok := errorMessages[statusCode]; ok {
				return nil, factory.Error()
			}
		}

		// Check for standard HTTP sentinel to wrap
		if sentinel, ok := sentinelMap[statusCode]; ok {
			return nil, fmt.Errorf("%w: %s", sentinel, string(body))
		}

		return nil, fmt.Errorf("expected status 200, got %d: %s", statusCode, string(body))
	}

	if json200 == nil {
		return nil, fmt.Errorf("response body is required")
	}
	return json200, nil
}

// ValidateJSON201 validates the HTTP status code and JSON201 field using the OpHandler for error mapping.
// Returns the JSON data on success, or an error with proper sentinel wrapping.
func ValidateJSON201[T any](oh *OpHandler, statusCode int, body []byte, json201 *T, opID int) (*T, error) {
	if statusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("%w: authentication required", ErrUnauthorized)
	}

	// For non-Created responses, check the error map first
	if statusCode != http.StatusCreated {
		if errorMessages, ok := oh.Messages[opID]; ok {
			if factory, ok := errorMessages[statusCode]; ok {
				return nil, factory.Error()
			}
		}

		// Check for standard HTTP sentinel to wrap
		if sentinel, ok := sentinelMap[statusCode]; ok {
			return nil, fmt.Errorf("%w: %s", sentinel, string(body))
		}

		return nil, fmt.Errorf("expected status 201, got %d: %s", statusCode, string(body))
	}

	if json201 == nil {
		return nil, fmt.Errorf("response body is required")
	}
	return json201, nil
}
