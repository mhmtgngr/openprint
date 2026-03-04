// Package middleware provides HTTP middleware for authentication, logging, and recovery.
package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	apperrors "github.com/openprint/openprint/internal/shared/errors"
)

// PaginationParams holds validated pagination parameters.
type PaginationParams struct {
	Limit   int
	Offset  int
	Page    int
	PerPage int
}

// ValidatePagination extracts and validates pagination parameters from the request.
func ValidatePagination(defaultLimit, maxLimit int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Parse limit
			limit := defaultLimit
			if l := r.URL.Query().Get("limit"); l != "" {
				parsed, err := strconv.Atoi(l)
				if err != nil || parsed < 0 {
					respondError(w, apperrors.New("invalid limit parameter", http.StatusBadRequest))
					return
				}
				if parsed > maxLimit {
					limit = maxLimit
				} else {
					limit = parsed
				}
			}

			// Parse offset
			offset := 0
			if o := r.URL.Query().Get("offset"); o != "" {
				parsed, err := strconv.Atoi(o)
				if err != nil || parsed < 0 {
					respondError(w, apperrors.New("invalid offset parameter", http.StatusBadRequest))
					return
				}
				offset = parsed
			}

			// Parse page (alternative pagination style)
			page := 1
			if p := r.URL.Query().Get("page"); p != "" {
				parsed, err := strconv.Atoi(p)
				if err != nil || parsed < 1 {
					respondError(w, apperrors.New("invalid page parameter", http.StatusBadRequest))
					return
				}
				page = parsed
			}

			// Parse per_page
			perPage := defaultLimit
			if pp := r.URL.Query().Get("per_page"); pp != "" {
				parsed, err := strconv.Atoi(pp)
				if err != nil || parsed < 1 {
					respondError(w, apperrors.New("invalid per_page parameter", http.StatusBadRequest))
					return
				}
				if parsed > maxLimit {
					perPage = maxLimit
				} else {
					perPage = parsed
				}
			}

			// Store pagination params in context
			params := PaginationParams{
				Limit:   limit,
				Offset:  offset,
				Page:    page,
				PerPage: perPage,
			}

			ctx = contextWithValue(ctx, "pagination", params)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetPaginationParams retrieves pagination params from context.
func GetPaginationParams(r *http.Request) PaginationParams {
	params, ok := r.Context().Value("pagination").(PaginationParams)
	if !ok {
		return PaginationParams{
			Limit:   50,
			Offset:  0,
			Page:    1,
			PerPage: 50,
		}
	}
	return params
}

// ValidateContentType ensures the request has a valid content type.
func ValidateContentType(allowedTypes []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip validation for GET, HEAD, DELETE
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodDelete {
				next.ServeHTTP(w, r)
				return
			}

			contentType := r.Header.Get("Content-Type")
			if contentType == "" {
				respondError(w, apperrors.New("missing Content-Type header", http.StatusUnsupportedMediaType))
				return
			}

			// Check if content type is allowed (strip charset etc.)
			baseType := strings.Split(contentType, ";")[0]
			baseType = strings.TrimSpace(baseType)

			allowed := false
			for _, allowedType := range allowedTypes {
				if baseType == allowedType {
					allowed = true
					break
				}
			}

			if !allowed {
				respondError(w, apperrors.New("unsupported Content-Type", http.StatusUnsupportedMediaType))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ValidateJSONBody validates that the request body is valid JSON.
func ValidateJSONBody(maxSize int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only validate for POST, PUT, PATCH
			if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch {
				next.ServeHTTP(w, r)
				return
			}

			// Enforce max size
			r.Body = http.MaxBytesReader(w, r.Body, maxSize)

			// Check content type
			contentType := r.Header.Get("Content-Type")
			if !strings.Contains(contentType, "application/json") {
				next.ServeHTTP(w, r)
				return
			}

			// Try to decode JSON to validate it
			var js json.RawMessage
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&js); err != nil {
				respondError(w, apperrors.Wrap(err, "invalid JSON body", http.StatusBadRequest))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// SanitizeInput middleware sanitizes query parameters to prevent injection attacks.
func SanitizeInput() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Sanitize query parameters
			for _, values := range r.URL.Query() {
				// Remove potentially dangerous characters
				for i, v := range values {
					values[i] = sanitizeString(v)
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// sanitizeString removes potentially dangerous characters from a string.
func sanitizeString(s string) string {
	// Remove null bytes and control characters
	result := strings.Builder{}
	for _, r := range s {
		if r >= 32 && r != 127 || r == '\t' || r == '\n' || r == '\r' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// RequireHeaders middleware ensures required headers are present.
func RequireHeaders(headers ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, header := range headers {
				if r.Header.Get(header) == "" {
					respondError(w, apperrors.New("missing required header: "+header, http.StatusBadRequest))
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// MaxBodySize limits the maximum size of request body.
func MaxBodySize(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only limit for methods that typically have bodies
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// contextWithValue is a helper to set values in context.
func contextWithValue(ctx context.Context, key string, value interface{}) context.Context {
	return context.WithValue(ctx, contextKey(key), value)
}
