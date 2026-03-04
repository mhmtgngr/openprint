// Package errors provides tests for shared error types and handling across all OpenPrint services.
package errors

import (
	"errors"
	"net/http"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name   string
		appErr *AppError
		want   string
	}{
		{
			name: "error without underlying error",
			appErr: &AppError{
				Message: "something went wrong",
			},
			want: "something went wrong",
		},
		{
			name: "error with underlying error",
			appErr: &AppError{
				Message: "something went wrong",
				Err:     errors.New("underlying error"),
			},
			want: "something went wrong: underlying error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.appErr.Error(); got != tt.want {
				t.Errorf("AppError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	appErr := &AppError{
		Message: "wrapper error",
		Err:     underlyingErr,
	}

	if got := appErr.Unwrap(); got != underlyingErr {
		t.Errorf("AppError.Unwrap() = %v, want %v", got, underlyingErr)
	}
}

func TestAppError_Unwrap_nil(t *testing.T) {
	appErr := &AppError{
		Message: "wrapper error",
		Err:     nil,
	}

	if got := appErr.Unwrap(); got != nil {
		t.Errorf("AppError.Unwrap() = %v, want nil", got)
	}
}

func TestNew(t *testing.T) {
	msg := "test error"
	status := http.StatusBadRequest
	appErr := New(msg, status)

	if appErr.Message != msg {
		t.Errorf("New() Message = %v, want %v", appErr.Message, msg)
	}
	if appErr.StatusCode != status {
		t.Errorf("New() StatusCode = %v, want %v", appErr.StatusCode, status)
	}
	if appErr.Code != "BAD_REQUEST" {
		t.Errorf("New() Code = %v, want BAD_REQUEST", appErr.Code)
	}
	if appErr.Err != nil {
		t.Errorf("New() Err = %v, want nil", appErr.Err)
	}
}

func TestWrap(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	msg := "wrapper message"
	status := http.StatusInternalServerError

	appErr := Wrap(underlyingErr, msg, status)

	if appErr.Message != msg {
		t.Errorf("Wrap() Message = %v, want %v", appErr.Message, msg)
	}
	if appErr.StatusCode != status {
		t.Errorf("Wrap() StatusCode = %v, want %v", appErr.StatusCode, status)
	}
	if appErr.Err != underlyingErr {
		t.Errorf("Wrap() Err = %v, want %v", appErr.Err, underlyingErr)
	}
}

func TestWrap_nil(t *testing.T) {
	appErr := Wrap(nil, "message", http.StatusBadRequest)
	if appErr != nil {
		t.Errorf("Wrap(nil) = %v, want nil", appErr)
	}
}

func TestAppError_WithCode(t *testing.T) {
	appErr := New("message", http.StatusBadRequest)
	customCode := "CUSTOM_CODE"
	result := appErr.WithCode(customCode)

	if result.Code != customCode {
		t.Errorf("WithCode() Code = %v, want %v", result.Code, customCode)
	}
	if result != appErr {
		t.Errorf("WithCode() should return same instance")
	}
}

func TestAppError_WithDetail(t *testing.T) {
	appErr := New("message", http.StatusBadRequest)
	key := "field"
	value := "email"
	result := appErr.WithDetail(key, value)

	if result.Details == nil {
		t.Fatal("WithDetail() Details should not be nil")
	}
	if result.Details[key] != value {
		t.Errorf("WithDetail() Details[%s] = %v, want %v", key, result.Details[key], value)
	}
	if result != appErr {
		t.Errorf("WithDetail() should return same instance")
	}
}

func TestAppError_WithDetail_multiple(t *testing.T) {
	appErr := New("message", http.StatusBadRequest)
	appErr.WithDetail("field1", "value1")
	appErr.WithDetail("field2", "value2")

	if len(appErr.Details) != 2 {
		t.Errorf("WithDetail() Details count = %d, want 2", len(appErr.Details))
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "ErrNotFound standard error",
			err:  ErrNotFound,
			want: true,
		},
		{
			name: "AppError with 404 status",
			err:  &AppError{StatusCode: http.StatusNotFound},
			want: true,
		},
		{
			name: "AppError with 500 status",
			err:  &AppError{StatusCode: http.StatusInternalServerError},
			want: false,
		},
		{
			name: "other error",
			err:  errors.New("other error"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFound(tt.err); got != tt.want {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsUnauthorized(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "ErrUnauthorized standard error",
			err:  ErrUnauthorized,
			want: true,
		},
		{
			name: "AppError with 401 status",
			err:  &AppError{StatusCode: http.StatusUnauthorized},
			want: true,
		},
		{
			name: "AppError with 500 status",
			err:  &AppError{StatusCode: http.StatusInternalServerError},
			want: false,
		},
		{
			name: "other error",
			err:  errors.New("other error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUnauthorized(tt.err); got != tt.want {
				t.Errorf("IsUnauthorized() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsConflict(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "ErrConflict standard error",
			err:  ErrConflict,
			want: true,
		},
		{
			name: "AppError with 409 status",
			err:  &AppError{StatusCode: http.StatusConflict},
			want: true,
		},
		{
			name: "AppError with 500 status",
			err:  &AppError{StatusCode: http.StatusInternalServerError},
			want: false,
		},
		{
			name: "other error",
			err:  errors.New("other error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsConflict(tt.err); got != tt.want {
				t.Errorf("IsConflict() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCodeFromStatus(t *testing.T) {
	tests := []struct {
		status   int
		expected string
	}{
		{http.StatusBadRequest, "BAD_REQUEST"},
		{http.StatusUnauthorized, "UNAUTHORIZED"},
		{http.StatusForbidden, "FORBIDDEN"},
		{http.StatusNotFound, "NOT_FOUND"},
		{http.StatusConflict, "CONFLICT"},
		{http.StatusTooManyRequests, "TOO_MANY_REQUESTS"},
		{http.StatusInternalServerError, "INTERNAL_ERROR"},
		{http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE"},
		{http.StatusCreated, "UNKNOWN_ERROR"},
		{999, "UNKNOWN_ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := codeFromStatus(tt.status); got != tt.expected {
				t.Errorf("codeFromStatus(%d) = %v, want %v", tt.status, got, tt.expected)
			}
		})
	}
}

func TestNewValidationError(t *testing.T) {
	field := "email"
	message := "invalid email format"

	appErr := NewValidationError(field, message)

	if appErr.Code != "VALIDATION_ERROR" {
		t.Errorf("NewValidationError() Code = %v, want VALIDATION_ERROR", appErr.Code)
	}
	if appErr.StatusCode != http.StatusBadRequest {
		t.Errorf("NewValidationError() StatusCode = %v, want %d", appErr.StatusCode, http.StatusBadRequest)
	}
	if appErr.Details == nil {
		t.Fatal("NewValidationError() Details should not be nil")
	}
	if appErr.Details["field"] != field {
		t.Errorf("NewValidationError() Details[field] = %v, want %v", appErr.Details["field"], field)
	}
	if appErr.Details["message"] != message {
		t.Errorf("NewValidationError() Details[message] = %v, want %v", appErr.Details["message"], message)
	}
}

func TestHTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "AppError with custom status",
			err:      &AppError{StatusCode: http.StatusNotFound},
			expected: http.StatusNotFound,
		},
		{
			name:     "standard error",
			err:      errors.New("standard error"),
			expected: http.StatusInternalServerError,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HTTPStatus(tt.err); got != tt.expected {
				t.Errorf("HTTPStatus() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestToJSON(t *testing.T) {
	t.Run("AppError without details", func(t *testing.T) {
		appErr := &AppError{
			Code:       "NOT_FOUND",
			Message:    "resource not found",
			StatusCode: http.StatusNotFound,
		}

		result := ToJSON(appErr)

		if result["code"] != "NOT_FOUND" {
			t.Errorf("ToJSON()[code] = %v, want NOT_FOUND", result["code"])
		}
		if result["message"] != "resource not found" {
			t.Errorf("ToJSON()[message] = %v, want 'resource not found'", result["message"])
		}
		if _, ok := result["details"]; ok {
			t.Error("ToJSON() should not have details key when Details is empty")
		}
	})

	t.Run("AppError with details", func(t *testing.T) {
		appErr := &AppError{
			Code:       "VALIDATION_ERROR",
			Message:    "validation failed",
			StatusCode: http.StatusBadRequest,
			Details: map[string]interface{}{
				"field":   "email",
				"message": "invalid format",
			},
		}

		result := ToJSON(appErr)

		if result["code"] != "VALIDATION_ERROR" {
			t.Errorf("ToJSON()[code] = %v, want VALIDATION_ERROR", result["code"])
		}
		if result["details"] == nil {
			t.Error("ToJSON() should have details key when Details is not empty")
		}
	})

	t.Run("standard error", func(t *testing.T) {
		err := errors.New("standard error")
		result := ToJSON(err)

		if result["code"] != "INTERNAL_ERROR" {
			t.Errorf("ToJSON()[code] = %v, want INTERNAL_ERROR", result["code"])
		}
		if result["message"] != "An internal error occurred" {
			t.Errorf("ToJSON()[message] = %v, want 'An internal error occurred'", result["message"])
		}
	})
}

func TestStandardErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        *AppError
		wantCode   string
		wantStatus int
	}{
		{"ErrNotFound", ErrNotFound, "NOT_FOUND", http.StatusNotFound},
		{"ErrUnauthorized", ErrUnauthorized, "UNAUTHORIZED", http.StatusUnauthorized},
		{"ErrForbidden", ErrForbidden, "FORBIDDEN", http.StatusForbidden},
		{"ErrConflict", ErrConflict, "CONFLICT", http.StatusConflict},
		{"ErrInvalidInput", ErrInvalidInput, "BAD_REQUEST", http.StatusBadRequest},
		{"ErrInternal", ErrInternal, "INTERNAL_ERROR", http.StatusInternalServerError},
		{"ErrServiceUnavailable", ErrServiceUnavailable, "SERVICE_UNAVAILABLE", http.StatusServiceUnavailable},
		{"ErrDuplicate", ErrDuplicate, "CONFLICT", http.StatusConflict},
		{"ErrExpired", ErrExpired, "UNAUTHORIZED", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.wantCode {
				t.Errorf("error Code = %v, want %v", tt.err.Code, tt.wantCode)
			}
			if tt.err.StatusCode != tt.wantStatus {
				t.Errorf("error StatusCode = %v, want %v", tt.err.StatusCode, tt.wantStatus)
			}
		})
	}
}

func TestAppError_Chaining(t *testing.T) {
	// Test error chaining with errors.Is
	baseErr := errors.New("base error")
	appErr := Wrap(baseErr, "wrapper", http.StatusBadRequest)

	if !errors.Is(appErr, baseErr) {
		t.Error("errors.Is should return true for base error")
	}

	// Test unwrapping
	unwrapped := errors.Unwrap(appErr)
	if unwrapped != baseErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, baseErr)
	}
}

func TestValidationErrorType(t *testing.T) {
	ve := ValidationError{
		Field:   "email",
		Message: "invalid format",
	}

	// This is a type check to ensure ValidationError struct exists
	// and has the expected fields
	if ve.Field != "email" {
		t.Errorf("ValidationError.Field = %v, want email", ve.Field)
	}
	if ve.Message != "invalid format" {
		t.Errorf("ValidationError.Message = %v, want 'invalid format'", ve.Message)
	}
}
