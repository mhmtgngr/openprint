// Package errors provides shared error types and handling across all OpenPrint services.
package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Standard error variables for common error conditions.
var (
	ErrNotFound       = New("resource not found", http.StatusNotFound)
	ErrUnauthorized   = New("unauthorized access", http.StatusUnauthorized)
	ErrForbidden      = New("forbidden", http.StatusForbidden)
	ErrConflict       = New("resource conflict", http.StatusConflict)
	ErrInvalidInput   = New("invalid input", http.StatusBadRequest)
	ErrInternal       = New("internal server error", http.StatusInternalServerError)
	ErrServiceUnavailable = New("service unavailable", http.StatusServiceUnavailable)
	ErrDuplicate      = New("duplicate resource", http.StatusConflict)
	ErrExpired        = New("resource expired", http.StatusUnauthorized)
)

// AppError represents an application error with HTTP status code and optional details.
type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	StatusCode int    `json:"-"`
	Err        error  `json:"-"`
	Details    map[string]interface{} `json:"details,omitempty"`
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the underlying error.
func (e *AppError) Unwrap() error {
	return e.Err
}

// New creates a new AppError with the given message and HTTP status code.
func New(message string, statusCode int) *AppError {
	return &AppError{
		Message:    message,
		StatusCode: statusCode,
		Code:       codeFromStatus(statusCode),
	}
}

// Wrap wraps an existing error with additional context.
func Wrap(err error, message string, statusCode int) *AppError {
	if err == nil {
		return nil
	}
	return &AppError{
		Message:    message,
		StatusCode: statusCode,
		Err:        err,
		Code:       codeFromStatus(statusCode),
	}
}

// WithCode adds an error code to the AppError.
func (e *AppError) WithCode(code string) *AppError {
	e.Code = code
	return e
}

// WithDetail adds a detail to the AppError.
func (e *AppError) WithDetail(key string, value interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// IsNotFound checks if an error is a not found error.
func IsNotFound(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.StatusCode == http.StatusNotFound
	}
	return errors.Is(err, ErrNotFound)
}

// IsUnauthorized checks if an error is an unauthorized error.
func IsUnauthorized(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.StatusCode == http.StatusUnauthorized
	}
	return errors.Is(err, ErrUnauthorized)
}

// IsConflict checks if an error is a conflict error.
func IsConflict(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.StatusCode == http.StatusConflict
	}
	return errors.Is(err, ErrConflict)
}

// codeFromStatus generates a standard error code from HTTP status code.
func codeFromStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "BAD_REQUEST"
	case http.StatusUnauthorized:
		return "UNAUTHORIZED"
	case http.StatusForbidden:
		return "FORBIDDEN"
	case http.StatusNotFound:
		return "NOT_FOUND"
	case http.StatusConflict:
		return "CONFLICT"
	case http.StatusTooManyRequests:
		return "TOO_MANY_REQUESTS"
	case http.StatusInternalServerError:
		return "INTERNAL_ERROR"
	case http.StatusServiceUnavailable:
		return "SERVICE_UNAVAILABLE"
	default:
		return "UNKNOWN_ERROR"
	}
}

// ValidationError represents field validation errors.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// NewValidationError creates a validation error for a specific field.
func NewValidationError(field, message string) *AppError {
	return &AppError{
		Message:    "validation failed",
		StatusCode: http.StatusBadRequest,
		Code:       "VALIDATION_ERROR",
		Details: map[string]interface{}{
			"field":   field,
			"message": message,
		},
	}
}

// HTTPStatus returns the HTTP status code for an error, defaulting to 500.
func HTTPStatus(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.StatusCode
	}
	return http.StatusInternalServerError
}

// ToJSON converts an error to a JSON-serializable map.
func ToJSON(err error) map[string]interface{} {
	var appErr *AppError
	if errors.As(err, &appErr) {
		result := map[string]interface{}{
			"code":    appErr.Code,
			"message": appErr.Message,
		}
		if len(appErr.Details) > 0 {
			result["details"] = appErr.Details
		}
		return result
	}
	return map[string]interface{}{
		"code":    "INTERNAL_ERROR",
		"message": "An internal error occurred",
	}
}
