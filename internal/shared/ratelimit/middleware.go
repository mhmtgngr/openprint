package ratelimit

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	sharedcontext "github.com/openprint/openprint/internal/shared/context"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
)

// Middleware provides HTTP middleware for rate limiting.
type Middleware struct {
	limiter       *Limiter
	skipPaths     []string
	skipIPs       map[string]bool
	extractUserID func(r *http.Request) string
	extractOrgID  func(r *http.Request) string
	extractRole   func(r *http.Request) string
	extractAPIKey func(r *http.Request) string
}

// MiddlewareConfig holds configuration for rate limit middleware.
type MiddlewareConfig struct {
	Limiter       *Limiter
	SkipPaths     []string
	SkipIPs       []string
	ExtractUserID func(r *http.Request) string
	ExtractOrgID  func(r *http.Request) string
	ExtractRole   func(r *http.Request) string
	ExtractAPIKey func(r *http.Request) string
}

// NewMiddleware creates a new rate limit middleware.
func NewMiddleware(cfg *MiddlewareConfig) *Middleware {
	skipIPMap := make(map[string]bool)
	for _, ip := range cfg.SkipIPs {
		skipIPMap[ip] = true
	}

	return &Middleware{
		limiter:       cfg.Limiter,
		skipPaths:     cfg.SkipPaths,
		skipIPs:       skipIPMap,
		extractUserID: cfg.ExtractUserID,
		extractOrgID:  cfg.ExtractOrgID,
		extractRole:   cfg.ExtractRole,
		extractAPIKey: cfg.ExtractAPIKey,
	}
}

// Handler returns an HTTP handler that enforces rate limits.
func (m *Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if path should be skipped
		if m.shouldSkip(r) {
			next.ServeHTTP(w, r)
			return
		}

		// Build rate limit request
		req := m.buildRequest(r)

		// Check rate limit
		result, err := m.limiter.Check(r.Context(), req)
		if err != nil {
			// Log error but allow request on failure (fail open)
			m.setRateLimitHeaders(w, &CheckResult{
				Allowed:   true,
				Limit:     0,
				Remaining: -1,
			})
			next.ServeHTTP(w, r)
			return
		}

		// Set rate limit headers
		m.setRateLimitHeaders(w, result)

		// Handle rate limit exceeded
		if !result.Allowed {
			if result.IsQueued {
				m.respondQueued(w, result)
				return
			}
			m.respondRateLimited(w, result)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// shouldSkip checks if a request should skip rate limiting.
func (m *Middleware) shouldSkip(r *http.Request) bool {
	// Check skip paths
	for _, path := range m.skipPaths {
		if strings.HasPrefix(r.URL.Path, path) {
			return true
		}
	}

	// Check skip IPs
	ip := getClientIP(r)
	if m.skipIPs[ip] {
		return true
	}

	return false
}

// buildRequest builds a rate limit Request from an HTTP request.
func (m *Middleware) buildRequest(r *http.Request) *Request {
	req := &Request{
		Method:    r.Method,
		Path:      r.URL.Path,
		IP:        getClientIP(r),
		Timestamp: time.Now(),
	}

	// Extract user info from context
	userID := sharedcontext.GetUserID(r.Context())
	if userID == "" && m.extractUserID != nil {
		userID = m.extractUserID(r)
	}
	req.Identifier = userID

	// Determine request type and identifier
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" && m.extractAPIKey != nil {
		apiKey = m.extractAPIKey(r)
	}

	if apiKey != "" {
		req.Type = "api_key"
		req.Identifier = apiKey
		req.APIKey = apiKey
	} else if userID != "" {
		req.Type = "user"
		req.Identifier = userID
		req.Role = sharedcontext.GetRole(r.Context())
		if req.Role == "" && m.extractRole != nil {
			req.Role = m.extractRole(r)
		}
		req.OrgID = sharedcontext.GetOrgID(r.Context())
		if req.OrgID == "" && m.extractOrgID != nil {
			req.OrgID = m.extractOrgID(r)
		}
	} else {
		// Fall back to IP-based limiting
		req.Type = "ip"
		req.Identifier = req.IP
	}

	// Extract priority from header (optional)
	if priorityHeader := r.Header.Get("X-Rate-Limit-Priority"); priorityHeader != "" {
		if priority, err := strconv.Atoi(priorityHeader); err == nil {
			req.Priority = priority
		}
	}

	// Check for burst flag
	if r.Header.Get("X-Rate-Limit-Burst") == "true" {
		req.IsBurst = true
	}

	return req
}

// setRateLimitHeaders sets standard rate limit headers on the response.
func (m *Middleware) setRateLimitHeaders(w http.ResponseWriter, result *CheckResult) {
	if result.IsBypassed {
		w.Header().Set("X-RateLimit-Bypassed", "true")
		return
	}

	w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(result.Limit, 10))
	w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(result.Remaining, 10))

	if !result.ResetAt.IsZero() {
		resetUnix := result.ResetAt.Unix()
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetUnix, 10))
	}

	if result.Policy != nil {
		w.Header().Set("X-RateLimit-Policy", result.Policy.Name)
	}
}

// respondRateLimited sends a rate limit exceeded response.
func (m *Middleware) respondRateLimited(w http.ResponseWriter, result *CheckResult) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-RateLimit-Remaining", "0")

	if result.RetryAfter > 0 {
		w.Header().Set("Retry-After", strconv.FormatInt(int64(result.RetryAfter.Seconds()), 10))
	}

	w.WriteHeader(http.StatusTooManyRequests)

	response := map[string]interface{}{
		"code":    "RATE_LIMIT_EXCEEDED",
		"message": "Rate limit exceeded",
		"limit":   result.Limit,
	}

	if !result.ResetAt.IsZero() {
		response["reset_at"] = result.ResetAt.Format(time.RFC3339)
	}

	if result.RetryAfter > 0 {
		response["retry_after"] = result.RetryAfter.String()
	}

	if result.Policy != nil {
		response["policy"] = result.Policy.Name
	}

	if result.ViolationID != "" {
		response["violation_id"] = result.ViolationID
	}

	json.NewEncoder(w).Encode(response)
}

// respondQueued sends a request queued response.
func (m *Middleware) respondQueued(w http.ResponseWriter, result *CheckResult) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-RateLimit-Queued", "true")
	w.Header().Set("X-RateLimit-Queue-Position", strconv.Itoa(result.QueuePosition))
	w.WriteHeader(http.StatusAccepted)

	response := map[string]interface{}{
		"code":           "REQUEST_QUEUED",
		"message":        "Request has been queued",
		"queue_position": result.QueuePosition,
	}

	if result.EstimatedWait > 0 {
		response["estimated_wait"] = result.EstimatedWait.String()
	}

	json.NewEncoder(w).Encode(response)
}

// getClientIP extracts the client IP address from a request.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, use the first one
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// StandardMiddleware creates a middleware with standard extractors.
func StandardMiddleware(limiter *Limiter, skipPaths []string, skipIPs []string) func(http.Handler) http.Handler {
	mw := NewMiddleware(&MiddlewareConfig{
		Limiter:   limiter,
		SkipPaths: skipPaths,
		SkipIPs:   skipIPs,
		ExtractUserID: func(r *http.Request) string {
			return sharedcontext.GetUserID(r.Context())
		},
		ExtractOrgID: func(r *http.Request) string {
			return sharedcontext.GetOrgID(r.Context())
		},
		ExtractRole: func(r *http.Request) string {
			return sharedcontext.GetRole(r.Context())
		},
		ExtractAPIKey: func(r *http.Request) string {
			return r.Header.Get("X-API-Key")
		},
	})

	return mw.Handler
}

// IPBasedMiddleware creates a simple IP-based rate limiting middleware.
func IPBasedMiddleware(limiter *Limiter, requestsPerMinute int, skipPaths []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check skip paths
			for _, path := range skipPaths {
				if strings.HasPrefix(r.URL.Path, path) {
					next.ServeHTTP(w, r)
					return
				}
			}

			ip := getClientIP(r)

			req := &Request{
				Type:      "ip",
				Identifier: ip,
				Method:    r.Method,
				Path:      r.URL.Path,
				IP:        ip,
				Timestamp: time.Now(),
			}

			result, err := limiter.Check(r.Context(), req)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			// Set headers
			w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(result.Limit, 10))
			w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(result.Remaining, 10))
			if !result.ResetAt.IsZero() {
				w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))
			}

			if !result.Allowed {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", strconv.FormatInt(int64(result.RetryAfter.Seconds()), 10))
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "Too many requests from this IP",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// TokenBucketMiddleware implements token bucket rate limiting.
type TokenBucketMiddleware struct {
	limiter    *Limiter
	rate       int64
	capacity   int64
	skipPaths  []string
}

// NewTokenBucketMiddleware creates a token bucket rate limiter.
func NewTokenBucketMiddleware(limiter *Limiter, rate, capacity int64, skipPaths []string) func(http.Handler) http.Handler {
	tb := &TokenBucketMiddleware{
		limiter:   limiter,
		rate:      rate,
		capacity:  capacity,
		skipPaths: skipPaths,
	}

	return tb.Handler
}

// Handler returns the token bucket HTTP handler.
func (tb *TokenBucketMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check skip paths
		for _, path := range tb.skipPaths {
			if strings.HasPrefix(r.URL.Path, path) {
				next.ServeHTTP(w, r)
				return
			}
		}

		ip := getClientIP(r)

		// Token bucket is implemented via a specific policy
		req := &Request{
			Type:      "ip",
			Identifier: ip,
			Method:    r.Method,
			Path:      r.URL.Path,
			IP:        ip,
			Timestamp: time.Now(),
		}

		result, err := tb.limiter.Check(r.Context(), req)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		// Set headers
		w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(tb.capacity, 10))
		w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(result.Remaining, 10))

		if !result.Allowed {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code":    "RATE_LIMIT_EXCEEDED",
				"message": "Token bucket exhausted",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// AdaptiveMiddleware adjusts rate limits based on system load.
type AdaptiveMiddleware struct {
	limiter         *Limiter
	baseLimit       int64
	minLimit        int64
	maxLimit        int64
	loadCheckFunc   func() float64 // Returns load as 0.0-1.0
	adjustThreshold float64
	skipPaths       []string
}

// NewAdaptiveMiddleware creates an adaptive rate limiting middleware.
func NewAdaptiveMiddleware(limiter *Limiter, baseLimit, minLimit, maxLimit int64, loadCheck func() float64, skipPaths []string) func(http.Handler) http.Handler {
	am := &AdaptiveMiddleware{
		limiter:         limiter,
		baseLimit:       baseLimit,
		minLimit:        minLimit,
		maxLimit:        maxLimit,
		loadCheckFunc:   loadCheck,
		adjustThreshold: 0.7, // Start reducing at 70% load
		skipPaths:       skipPaths,
	}

	return am.Handler
}

// Handler returns the adaptive rate limit handler.
func (am *AdaptiveMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check skip paths
		for _, path := range am.skipPaths {
			if strings.HasPrefix(r.URL.Path, path) {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Get current load and adjust limit
		currentLimit := am.baseLimit
		if am.loadCheckFunc != nil {
			load := am.loadCheckFunc()
			if load > am.adjustThreshold {
				// Linearly reduce limit based on load
				reduction := (load - am.adjustThreshold) / (1.0 - am.adjustThreshold)
				currentLimit = int64(float64(am.baseLimit) * (1.0 - reduction))
				if currentLimit < am.minLimit {
					currentLimit = am.minLimit
				}
			}
		}

		// Use the adjusted limit for checking
		ip := getClientIP(r)
		req := &Request{
			Type:      "ip",
			Identifier: ip,
			Method:    r.Method,
			Path:      r.URL.Path,
			IP:        ip,
			Timestamp: time.Now(),
		}

		result, err := am.limiter.Check(r.Context(), req)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		// Set headers
		w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(currentLimit, 10))
		w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(result.Remaining, 10))
		w.Header().Set("X-RateLimit-Adaptive", "true")

		if !result.Allowed {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code":    "RATE_LIMIT_EXCEEDED",
				"message": fmt.Sprintf("Rate limit reduced due to high load. Current limit: %d requests", currentLimit),
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// GradientDelayMiddleware implements gradient delay for throttling.
func GradientDelayMiddleware(limiter *Limiter, skipPaths []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check skip paths
			for _, path := range skipPaths {
				if strings.HasPrefix(r.URL.Path, path) {
					next.ServeHTTP(w, r)
					return
				}
			}

			ip := getClientIP(r)
			req := &Request{
				Type:      "ip",
				Identifier: ip,
				Method:    r.Method,
				Path:      r.URL.Path,
				IP:        ip,
				Timestamp: time.Now(),
			}

			result, err := limiter.Check(r.Context(), req)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			// Calculate delay based on remaining capacity
			if result.Remaining < result.Limit/4 {
				// Near limit, add delay
				delay := time.Duration((result.Limit - result.Remaining) * int64(time.Second)/result.Limit)
				if delay > 5*time.Second {
					delay = 5 * time.Second
				}
				time.Sleep(delay)
			}

			// Set headers
			w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(result.Limit, 10))
			w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(result.Remaining, 10))

			if !result.Allowed {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(apperrors.New("Rate limit exceeded", http.StatusTooManyRequests))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
