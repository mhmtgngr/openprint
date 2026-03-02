// Package middleware provides HTTP middleware for input validation, rate limiting, and security.
package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	apperrors "github.com/openprint/openprint/internal/shared/errors"
)

// Context key type for storing values in request context.
type contextKey string

const (
	// ValidationContextKey is the key used to store validated data in context.
	ValidationContextKey contextKey = "validated_data"
)

// SanitizedString represents a string that has been sanitized.
type SanitizedString string

// ValidateEmail validates an email address format.
func ValidateEmail(email string) error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return apperrors.New("invalid email format", http.StatusBadRequest)
	}
	return nil
}

// ValidateUUID validates a UUID string.
func ValidateUUID(id string) error {
	uuidRegex := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	if !uuidRegex.MatchString(id) {
		return apperrors.New("invalid UUID format", http.StatusBadRequest)
	}
	return nil
}

// SanitizeString removes potentially dangerous characters from input.
// It preserves alphanumeric characters, spaces, and common safe punctuation.
func SanitizeString(input string) SanitizedString {
	// Remove null bytes and control characters except tab, newline, carriage return
	var result strings.Builder
	for _, r := range input {
		if r >= 32 && r != 127 || r == '\t' || r == '\n' || r == '\r' {
			result.WriteRune(r)
		}
	}

	// Remove potential SQL injection patterns
	sanitized := result.String()
	sanitized = strings.ReplaceAll(sanitized, "'", "''")
	sanitized = strings.ReplaceAll(sanitized, ";", "")

	// Remove potential XSS patterns
	sanitized = stripScriptTags(sanitized)

	return SanitizedString(sanitized)
}

// stripScriptTags removes script tags and related JavaScript patterns.
func stripScriptTags(input string) string {
	// Remove script tags
	scriptRegex := regexp.MustCompile(`(?i)<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>`)
	input = scriptRegex.ReplaceAllString(input, "")

	// Remove on* event handlers (onclick, onload, etc.)
	eventRegex := regexp.MustCompile(`(?i)\s+on\w+\s*=\s*["'][^"']*["']`)
	input = eventRegex.ReplaceAllString(input, "")

	// Remove javascript: protocol
	jsProtocolRegex := regexp.MustCompile(`(?i)javascript:\s*\S*`)
	input = jsProtocolRegex.ReplaceAllString(input, "")

	return input
}

// SanitizeHTML sanitizes HTML input by removing dangerous tags and attributes.
func SanitizeHTML(input string) SanitizedString {
	// Define dangerous tags
	dangerousTags := []string{
		"script", "iframe", "object", "embed", "form", "input", "button",
		"link", "meta", "style", "applet", "body", "html", "head",
	}

	sanitized := input
	for _, tag := range dangerousTags {
		tagRegex := regexp.MustCompile(fmt.Sprintf(`(?i)</?%s\b[^>]*>`, tag))
		sanitized = tagRegex.ReplaceAllString(sanitized, "")
	}

	// Remove dangerous attributes
	dangerousAttrs := []string{
		"onload", "onerror", "onclick", "onmouseover", "onmouseout",
		"onfocus", "onblur", "onchange", "onsubmit", "onreset",
	}
	for _, attr := range dangerousAttrs {
		attrRegex := regexp.MustCompile(fmt.Sprintf(`(?i)\s+%s\s*=\s*["'][^"']*["']`, attr))
		sanitized = attrRegex.ReplaceAllString(sanitized, "")
	}

	return SanitizedString(sanitized)
}

// ValidateStringLength validates string length constraints.
func ValidateStringLength(input string, minLength, maxLength int) error {
	length := len(input)
	if minLength > 0 && length < minLength {
		return apperrors.New(fmt.Sprintf("input must be at least %d characters", minLength), http.StatusBadRequest)
	}
	if maxLength > 0 && length > maxLength {
		return apperrors.New(fmt.Sprintf("input must not exceed %d characters", maxLength), http.StatusBadRequest)
	}
	return nil
}

// ValidateInteger validates an integer within a range.
func ValidateInteger(input string, min, max int64) (int64, error) {
	val, err := strconv.ParseInt(input, 10, 64)
	if err != nil {
		return 0, apperrors.New("invalid integer format", http.StatusBadRequest)
	}
	if min > 0 && val < min {
		return 0, apperrors.New(fmt.Sprintf("value must be at least %d", min), http.StatusBadRequest)
	}
	if max > 0 && val > max {
		return 0, apperrors.New(fmt.Sprintf("value must not exceed %d", max), http.StatusBadRequest)
	}
	return val, nil
}

// ValidateURL validates a URL string.
func ValidateURL(urlStr string) error {
	if urlStr == "" {
		return apperrors.New("URL cannot be empty", http.StatusBadRequest)
	}

	// Basic URL validation
	urlRegex := regexp.MustCompile(`^https?://[a-zA-Z0-9\-._~:/?#\[\]@!$&'()*+,;=]+$`)
	if !urlRegex.MatchString(urlStr) {
		return apperrors.New("invalid URL format", http.StatusBadRequest)
	}

	// Check for javascript: protocol
	if strings.Contains(strings.ToLower(urlStr), "javascript:") {
		return apperrors.New("invalid URL protocol", http.StatusBadRequest)
	}

	return nil
}

// RateLimiterConfig holds rate limiter configuration.
type RateLimiterConfig struct {
	RequestsPerMinute int           // Maximum requests per time window
	TimeWindow        time.Duration // Time window for rate limiting
	CleanupInterval   time.Duration // How often to clean up expired entries
	EnableByIP        bool          // Enable IP-based limiting
	EnableByUser      bool          // Enable user-based limiting
	SkipPaths         []string      // Paths to skip rate limiting
}

// DefaultRateLimiterConfig returns default rate limiter configuration.
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		RequestsPerMinute: 60,
		TimeWindow:        time.Minute,
		CleanupInterval:   5 * time.Minute,
		EnableByIP:        true,
		EnableByUser:      false,
		SkipPaths:         []string{"/health", "/metrics"},
	}
}

// RateLimiterTracker tracks requests for a single client.
type RateLimiterTracker struct {
	count     int
	windowEnd time.Time
	mu        sync.Mutex
}

// Allow checks if a request should be allowed.
func (t *RateLimiterTracker) Allow(maxRequests int, window time.Duration) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()

	// Reset if window has expired
	if now.After(t.windowEnd) {
		t.count = 0
		t.windowEnd = now.Add(window)
	}

	if t.count >= maxRequests {
		return false
	}

	t.count++
	return true
}

// RateLimiter provides rate limiting using a sliding window algorithm.
type RateLimiter struct {
	ips      map[string]*RateLimiterTracker
	users    map[string]*RateLimiterTracker
	mu       sync.RWMutex
	config   RateLimiterConfig
	stopChan chan struct{}
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	rl := &RateLimiter{
		ips:      make(map[string]*RateLimiterTracker),
		users:    make(map[string]*RateLimiterTracker),
		config:   config,
		stopChan: make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// getTracker returns the tracker for the given identifier.
func (rl *RateLimiter) getTracker(identifier string, userBased bool) *RateLimiterTracker {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	var trackerMap map[string]*RateLimiterTracker
	if userBased {
		trackerMap = rl.users
	} else {
		trackerMap = rl.ips
	}

	tracker, exists := trackerMap[identifier]
	if !exists {
		tracker = &RateLimiterTracker{
			windowEnd: time.Now().Add(rl.config.TimeWindow),
		}
		trackerMap[identifier] = tracker
	}

	return tracker
}

// Allow checks if a request should be allowed for the given identifier.
func (rl *RateLimiter) Allow(identifier string) bool {
	tracker := rl.getTracker(identifier, false)
	return tracker.Allow(rl.config.RequestsPerMinute, rl.config.TimeWindow)
}

// AllowUser checks if a request should be allowed for the given user.
func (rl *RateLimiter) AllowUser(userID string) bool {
	tracker := rl.getTracker(userID, true)
	return tracker.Allow(rl.config.RequestsPerMinute, rl.config.TimeWindow)
}

// cleanup removes stale entries from the rate limiter maps.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			// Remove expired entries
			for id, tracker := range rl.ips {
				tracker.mu.Lock()
				if now.After(tracker.windowEnd.Add(rl.config.TimeWindow)) {
					delete(rl.ips, id)
				}
				tracker.mu.Unlock()
			}
			for id, tracker := range rl.users {
				tracker.mu.Lock()
				if now.After(tracker.windowEnd.Add(rl.config.TimeWindow)) {
					delete(rl.users, id)
				}
				tracker.mu.Unlock()
			}
			rl.mu.Unlock()
		case <-rl.stopChan:
			return
		}
	}
}

// Stop stops the cleanup goroutine.
func (rl *RateLimiter) Stop() {
	close(rl.stopChan)
}

// RateLimitMiddleware creates a rate limiting middleware.
func RateLimitMiddleware(config RateLimiterConfig) func(http.Handler) http.Handler {
	limiter := NewRateLimiter(config)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path should be skipped
			for _, skipPath := range config.SkipPaths {
				if strings.HasPrefix(r.URL.Path, skipPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			allowed := false

			// Try user-based limiting first
			if config.EnableByUser {
				if userID := r.Header.Get("X-User-ID"); userID != "" {
					allowed = limiter.AllowUser(userID)
				}
			}

			// Fall back to IP-based limiting
			if !allowed && config.EnableByIP {
				ip := getClientIP(r)
				allowed = limiter.Allow(ip)
			}

			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "Too many requests. Please try again later.",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequestSizeLimitConfig holds request size limits configuration.
type RequestSizeLimitConfig struct {
	MaxBodySize   int64    // Maximum request body size in bytes
	MaxHeaderSize int64    // Maximum header size in bytes
	MaxQuerySize  int      // Maximum query string length
	SkipPaths     []string // Paths to skip size limits
	MaxURLLength  int      // Maximum URL length
}

// DefaultRequestSizeLimitConfig returns default request size limit configuration.
func DefaultRequestSizeLimitConfig() RequestSizeLimitConfig {
	return RequestSizeLimitConfig{
		MaxBodySize:   10 * 1024 * 1024, // 10MB
		MaxHeaderSize: 10 * 1024,        // 10KB
		MaxQuerySize:  2048,             // 2048 characters
		MaxURLLength:  2048,             // 2048 characters
		SkipPaths:     []string{"/health", "/metrics"},
	}
}

// RequestSizeLimitMiddleware creates middleware that limits request sizes.
func RequestSizeLimitMiddleware(config RequestSizeLimitConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path should be skipped
			for _, skipPath := range config.SkipPaths {
				if strings.HasPrefix(r.URL.Path, skipPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Check URL length
			if config.MaxURLLength > 0 && len(r.URL.String()) > config.MaxURLLength {
				respondValidationError(w, "URL too long")
				return
			}

			// Check query string length
			if config.MaxQuerySize > 0 && len(r.URL.RawQuery) > config.MaxQuerySize {
				respondValidationError(w, "query string too long")
				return
			}

			// Limit body size for methods that typically have bodies
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
				if config.MaxBodySize > 0 {
					r.Body = http.MaxBytesReader(w, r.Body, config.MaxBodySize)
				}
			}

			// Check total header size
			if config.MaxHeaderSize > 0 {
				headerSize := int64(0)
				for key, values := range r.Header {
					headerSize += int64(len(key))
					for _, val := range values {
						headerSize += int64(len(val))
					}
				}
				if headerSize > config.MaxHeaderSize {
					respondValidationError(w, "headers too large")
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// InputSanitizationMiddleware creates middleware that sanitizes user input.
func InputSanitizationMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Sanitize query parameters
			if r.URL.Query() != nil {
				for key, values := range r.URL.Query() {
					for i, v := range values {
						r.URL.Query()[key][i] = string(SanitizeString(v))
					}
				}
			}

			// Sanitize headers (prevent header injection)
			for key, values := range r.Header {
				for i, v := range values {
					// Remove newlines and carriage returns from headers
					cleaned := strings.ReplaceAll(v, "\n", "")
					cleaned = strings.ReplaceAll(cleaned, "\r", "")
					r.Header[key][i] = cleaned
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ValidateJSONSchema validates JSON request body against a schema.
type SchemaValidator struct {
	RequiredFields  []string
	MaxFields       int
	FieldValidators map[string]func(interface{}) error
}

// NewSchemaValidator creates a new JSON schema validator.
func NewSchemaValidator() *SchemaValidator {
	return &SchemaValidator{
		RequiredFields:  []string{},
		MaxFields:       50,
		FieldValidators: make(map[string]func(interface{}) error),
	}
}

// ValidateJSONBody validates JSON request body against the schema.
func (sv *SchemaValidator) ValidateJSONBody() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only validate for POST, PUT, PATCH
			if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch {
				next.ServeHTTP(w, r)
				return
			}

			contentType := r.Header.Get("Content-Type")
			if !strings.Contains(contentType, "application/json") {
				next.ServeHTTP(w, r)
				return
			}

			// Decode JSON
			var data map[string]interface{}
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&data); err != nil {
				respondValidationError(w, "invalid JSON: "+err.Error())
				return
			}

			// Check field count
			if sv.MaxFields > 0 && len(data) > sv.MaxFields {
				respondValidationError(w, fmt.Sprintf("too many fields (max %d)", sv.MaxFields))
				return
			}

			// Check required fields
			for _, field := range sv.RequiredFields {
				if _, exists := data[field]; !exists {
					respondValidationError(w, fmt.Sprintf("missing required field: %s", field))
					return
				}
			}

			// Run field validators
			for field, validator := range sv.FieldValidators {
				if value, exists := data[field]; exists {
					if err := validator(value); err != nil {
						respondValidationError(w, fmt.Sprintf("invalid field %s: %s", field, err.Error()))
						return
					}
				}
			}

			// Store validated data in context
			ctx := context.WithValue(r.Context(), ValidationContextKey, data)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetValidatedData retrieves validated JSON data from context.
func GetValidatedData(r *http.Request) map[string]interface{} {
	data, _ := r.Context().Value(ValidationContextKey).(map[string]interface{})
	return data
}

// AddRequiredField adds a required field to the schema.
func (sv *SchemaValidator) AddRequiredField(field string) *SchemaValidator {
	sv.RequiredFields = append(sv.RequiredFields, field)
	return sv
}

// AddFieldValidator adds a validator for a specific field.
func (sv *SchemaValidator) AddFieldValidator(field string, validator func(interface{}) error) *SchemaValidator {
	sv.FieldValidators[field] = validator
	return sv
}

// respondValidationError sends a validation error response.
func respondValidationError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    "VALIDATION_ERROR",
		"message": message,
	})
}

// Chain chains multiple middleware together.
func Chain(middleware ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(final http.Handler) http.Handler {
		for i := len(middleware) - 1; i >= 0; i-- {
			final = middleware[i](final)
		}
		return final
	}
}

// SecurityHeadersMiddleware adds security headers to responses.
func SecurityHeadersMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; object-src 'none'; base-uri 'self'")

			next.ServeHTTP(w, r)
		})
	}
}
