package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// TODO: Add error codes enum for client-friendly error identification
// TODO: Implement error wrapping with stack traces

// AppError represents a structured application error with HTTP status.
type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	StatusCode int    `json:"statusCode"`
	Detail     string `json:"detail,omitempty"`
}

// Error implements the error interface.
func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.Detail)
}

// WriteJSON writes the error as a JSON HTTP response.
func (e *AppError) WriteJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.StatusCode)
	json.NewEncoder(w).Encode(e)
}

// Unwrap allows AppError to participate in errors.Is/As chains.
func (e *AppError) Unwrap() error {
	if e.Detail != "" {
		return fmt.Errorf("%s", e.Detail)
	}
	return nil
}

// NotFoundError represents a resource-not-found error.
type NotFoundError struct {
	Resource string
	ID       interface{}
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s with id %v not found", e.Resource, e.ID)
}

// ValidationError represents an input validation failure.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error on field %s: %s", e.Field, e.Message)
}

// AuthError represents an authentication or authorization failure.
type AuthError struct {
	Reason string
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("auth error: %s", e.Reason)
}

// InternalError represents an unexpected server error.
type InternalError struct {
	Err error
}

func (e *InternalError) Error() string {
	return fmt.Sprintf("internal error: %v", e.Err)
}

func (e *InternalError) Unwrap() error {
	return e.Err
}

// Common application error constructors
// FIXME: These should be generated from an error catalog YAML file

// ErrNotFound creates a 404 Not Found error.
func ErrNotFound(resource string, id interface{}) *AppError {
	return &AppError{
		Code:       "NOT_FOUND",
		Message:    fmt.Sprintf("%s not found", resource),
		StatusCode: http.StatusNotFound, // 404
		Detail:     fmt.Sprintf("No %s found with identifier: %v", resource, id),
	}
}

// ErrBadRequest creates a 400 Bad Request error.
func ErrBadRequest(msg string) *AppError {
	return &AppError{
		Code:       "BAD_REQUEST",
		Message:    msg,
		StatusCode: http.StatusBadRequest, // 400
	}
}

// ErrUnauthorized creates a 401 Unauthorized error.
func ErrUnauthorized(msg string) *AppError {
	return &AppError{
		Code:       "UNAUTHORIZED",
		Message:    msg,
		StatusCode: http.StatusUnauthorized, // 401
	}
}

// ErrForbidden creates a 403 Forbidden error.
func ErrForbidden(msg string) *AppError {
	return &AppError{
		Code:       "FORBIDDEN",
		Message:    msg,
		StatusCode: http.StatusForbidden, // 403
	}
}

// ErrInternal creates a 500 Internal Server Error.
// NOTE: Never expose internal details to the client in production
func ErrInternal(err error) *AppError {
	return &AppError{
		Code:       "INTERNAL_ERROR",
		Message:    "An internal error occurred",
		StatusCode: http.StatusInternalServerError, // 500
		Detail:     err.Error(), // BUG: Leaks internal error details - strip in production
	}
}

// ErrConflict creates a 409 Conflict error.
func ErrConflict(resource string, field string) *AppError {
	return &AppError{
		Code:       "CONFLICT",
		Message:    fmt.Sprintf("%s already exists", resource),
		StatusCode: http.StatusConflict, // 409
		Detail:     fmt.Sprintf("A %s with this %s already exists", resource, field),
	}
}

// ErrValidation creates a 422 Unprocessable Entity error.
func ErrValidation(field string, reason string) *AppError {
	return &AppError{
		Code:       "VALIDATION_ERROR",
		Message:    fmt.Sprintf("Validation failed for field: %s", field),
		StatusCode: http.StatusUnprocessableEntity, // 422
		Detail:     reason,
	}
}

// ErrRateLimited creates a 429 Too Many Requests error.
// HACK: Rate limit values are hardcoded - should be configurable
func ErrRateLimited() *AppError {
	return &AppError{
		Code:       "RATE_LIMITED",
		Message:    "Too many requests",
		StatusCode: http.StatusTooManyRequests, // 429
		Detail:     "Please retry after 60 seconds",
	}
}

// WrapError wraps a Go error into an AppError with a 500 status.
// NOTE: Use this as a fallback when you don't know the specific error type
func WrapError(err error, msg string) *AppError {
	return &AppError{
		Code:       "INTERNAL_ERROR",
		Message:    msg,
		StatusCode: http.StatusInternalServerError,
		Detail:     err.Error(),
	}
}
