// Package middleware provides rate limiting middleware integration for OpenPrint services.
package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	sharedcontext "github.com/openprint/openprint/internal/shared/context"
	"github.com/openprint/openprint/internal/shared/ratelimit"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
)

// RateLimitConfig holds configuration for rate limit middleware.
type RateLimitConfig struct {
	Limiter           *ratelimit.Limiter
	SkipPaths         []string
	SkipIPs           []string
	EnableByDefault   bool
	// FailClosed determines behavior when rate limiter fails. If true, requests
	// are blocked when the rate limiter is unavailable (more secure). If false,
	// requests are allowed with degraded rate limiting (fail-open with logging).
	FailClosed        bool
	// DegradedLimit is the request limit per minute to apply when rate limiter
	// is unavailable and FailClosed is false. Set to 0 to disable degraded limiting.
	DegradedLimit     int
}

// RateLimit returns a middleware function that enforces rate limits.
// It integrates with the existing OpenPrint middleware chain and uses
// the shared ratelimit package for Redis-based sliding window rate limiting.
func RateLimit(cfg *RateLimitConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip if limiter not configured
			if cfg.Limiter == nil || !cfg.EnableByDefault {
				next.ServeHTTP(w, r)
				return
			}

			// Check skip paths
			for _, path := range cfg.SkipPaths {
				if strings.HasPrefix(r.URL.Path, path) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Build rate limit request from HTTP request
			req := buildRateLimitRequest(r)

			// Check if IP should be skipped
			clientIP := getClientIP(r)
			userID := sharedcontext.GetUserID(r.Context())
			for _, skipIP := range cfg.SkipIPs {
				if clientIP == skipIP {
					setRateLimitHeaders(w, &ratelimit.CheckResult{
						Allowed:    true,
						IsBypassed: true,
					})
					next.ServeHTTP(w, r)
					return
				}
			}

			// Check rate limit
			result, err := cfg.Limiter.Check(r.Context(), req)
			if err != nil {
				// SECURE FIX: Log rate limiter error for security monitoring
				slog.Error("Rate limiter error",
					"error", err,
					"client_ip", clientIP,
					"path", r.URL.Path,
					"method", r.Method,
					"user_id", userID,
					"identifier", req.Identifier,
				)

				// Fail-closed mode: block requests when rate limiter is unavailable
				if cfg.FailClosed {
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("Retry-After", "60")
					w.WriteHeader(http.StatusServiceUnavailable)
					w.Write([]byte(`{"code":"RATE_LIMITER_UNAVAILABLE","message":"Service temporarily unavailable. Please retry later."}`))
					return
				}

				// Fail-open with degraded mode: apply basic rate limiting
				if cfg.DegradedLimit > 0 {
					if !applyDegradedRateLimit(r, w, clientIP, cfg.DegradedLimit) {
						return
					}
				}

				// Apply degraded mode headers to indicate rate limiter is down
				w.Header().Set("X-RateLimit-Degraded", "true")
				next.ServeHTTP(w, r)
				return
			}

			// Set rate limit headers
			setRateLimitHeaders(w, result)

			// Handle rate limit exceeded
			if !result.Allowed {
				if result.IsQueued {
					respondQueued(w, result)
					return
				}
				respondRateLimited(w, result)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// buildRateLimitRequest constructs a rate limit Request from an HTTP request.
// It extracts user information from context and determines the appropriate
// rate limit scope (user, API key, or IP).
func buildRateLimitRequest(r *http.Request) *ratelimit.Request {
	req := &ratelimit.Request{
		Method:    r.Method,
		Path:      r.URL.Path,
		Timestamp: time.Now(),
	}

	// Get API key from header
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		apiKey = r.Header.Get("Authorization")
		if strings.HasPrefix(apiKey, "Bearer ") {
			apiKey = strings.TrimPrefix(apiKey, "Bearer ")
		}
	}

	// Get user info from context
	ctx := r.Context()
	userID := sharedcontext.GetUserID(ctx)
	orgID := sharedcontext.GetOrgID(ctx)
	role := sharedcontext.GetRole(ctx)
	scopes := sharedcontext.GetScopes(ctx)

	// Determine request type and identifier
	if apiKey != "" && isValidAPIKey(ctx, apiKey) {
		req.Type = "api_key"
		req.Identifier = apiKey
		req.APIKey = apiKey
	} else if userID != "" {
		req.Type = "user"
		req.Identifier = userID
		req.Role = role
		req.OrgID = orgID
	} else {
		// Fall back to IP-based limiting
		req.Type = "ip"
		req.Identifier = getClientIP(r)
	}

	// Extract priority for queuing (optional)
	if priorityHeader := r.Header.Get("X-Rate-Limit-Priority"); priorityHeader != "" {
		if priority, err := strconv.Atoi(priorityHeader); err == nil {
			req.Priority = priority
		}
	}

	// Check for burst flag
	if r.Header.Get("X-Rate-Limit-Burst") == "true" {
		req.IsBurst = true
	}

	// Check if admin (admins get higher limits)
	if hasAdminScope(scopes) {
		req.Priority = 1000 // High priority for admins
	}

	return req
}

// getClientIP extracts the client IP address from the request.
// It checks X-Forwarded-For, X-Real-IP headers before falling back to RemoteAddr.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies/load balancers)
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
	return r.RemoteAddr
}

// setRateLimitHeaders sets standard rate limit headers on the response.
// These headers follow the IETF draft spec for rate limit headers.
func setRateLimitHeaders(w http.ResponseWriter, result *ratelimit.CheckResult) {
	if result.IsBypassed {
		w.Header().Set("X-RateLimit-Bypassed", "true")
		return
	}

	if result.Limit > 0 {
		w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(result.Limit, 10))
		w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(result.Remaining, 10))
	}

	if !result.ResetAt.IsZero() {
		resetUnix := result.ResetAt.Unix()
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetUnix, 10))
	}

	if result.Policy != nil {
		w.Header().Set("X-RateLimit-Policy", result.Policy.Name)
	}
}

// respondRateLimited sends a rate limit exceeded response.
func respondRateLimited(w http.ResponseWriter, result *ratelimit.CheckResult) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-RateLimit-Remaining", "0")

	if result.RetryAfter > 0 {
		w.Header().Set("Retry-After", strconv.FormatInt(int64(result.RetryAfter.Seconds()), 10))
	}

	w.WriteHeader(http.StatusTooManyRequests)

	response := map[string]interface{}{
		"code":    "RATE_LIMIT_EXCEEDED",
		"message": "Rate limit exceeded. Please retry later.",
		"limit":   result.Limit,
	}

	if !result.ResetAt.IsZero() {
		response["reset_at"] = result.ResetAt.Format(time.RFC3339Nano)
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

	w.Write([]byte(`{"code":"RATE_LIMIT_EXCEEDED","message":"Rate limit exceeded. Please retry later."}`))
}

// respondQueued sends a request queued response.
func respondQueued(w http.ResponseWriter, result *ratelimit.CheckResult) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-RateLimit-Queued", "true")
	w.Header().Set("X-RateLimit-Queue-Position", strconv.Itoa(result.QueuePosition))
	w.WriteHeader(http.StatusAccepted)

	response := map[string]interface{}{
		"code":           "REQUEST_QUEUED",
		"message":        "Request has been queued due to rate limiting",
		"queue_position": result.QueuePosition,
	}

	if result.EstimatedWait > 0 {
		response["estimated_wait"] = result.EstimatedWait.String()
	}

	w.Write([]byte(`{"code":"REQUEST_QUEUED","message":"Request has been queued"}`))
}

// isValidAPIKey checks if an API key is valid.
// In production, this would validate against the database.
func isValidAPIKey(ctx context.Context, apiKey string) bool {
	// Basic validation - check if it looks like a UUID or has required format
	if len(apiKey) < 16 {
		return false
	}
	return true
}

// hasAdminScope checks if the request has admin scope.
func hasAdminScope(scopes []string) bool {
	for _, scope := range scopes {
		if scope == "admin" || scope == "system:admin" {
			return true
		}
	}
	return false
}

// RateLimitByPath creates rate limit middleware for specific paths.
// This allows different rate limits for different endpoint groups.
func RateLimitByPath(cfg *RateLimitConfig, pathLimits map[string]int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Find matching path limit
			var limit int64
			for path, pathLimit := range pathLimits {
				if strings.HasPrefix(r.URL.Path, path) {
					limit = pathLimit
					break
				}
			}

			if limit == 0 {
				// No specific limit for this path, use default middleware
				next.ServeHTTP(w, r)
				return
			}

			// Apply path-specific limit
			req := buildRateLimitRequest(r)
			result, err := cfg.Limiter.Check(r.Context(), req)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			setRateLimitHeaders(w, result)

			if !result.Allowed {
				respondRateLimited(w, result)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// IPOnlyRateLimit creates a simple IP-only rate limiting middleware.
// This is useful for protecting against DDoS attacks before authentication.
func IPOnlyRateLimit(cfg *RateLimitConfig, requestsPerMinute int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.Limiter == nil {
				next.ServeHTTP(w, r)
				return
			}

			ip := getClientIP(r)

			req := &ratelimit.Request{
				Type:      "ip",
				Identifier: ip,
				Method:    r.Method,
				Path:      r.URL.Path,
				Timestamp: time.Now(),
			}

			result, err := cfg.Limiter.Check(r.Context(), req)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(result.Limit, 10))
			w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(result.Remaining, 10))
			if !result.ResetAt.IsZero() {
				w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))
			}

			if !result.Allowed {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", strconv.FormatInt(int64(result.RetryAfter.Seconds()), 10))
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"code":"RATE_LIMIT_EXCEEDED","message":"Too many requests from this IP"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// // WithUserID returns a context with the user ID set.
// // This helper function can be used to inject user context for testing.
// func WithUserID(ctx context.Context, userID string) context.Context {
// 	return context.WithValue(ctx, context.UserIDKey, userID)
// }

// // WithOrgID returns a context with the organization ID set.
// func WithOrgID(ctx context.Context, orgID string) context.Context {
// 	return context.WithValue(ctx, context.OrgIDKey, orgID)
// }

// // WithRole returns a context with the role set.
// func WithRole(ctx context.Context, role string) context.Context {
// 	return context.WithValue(ctx, context.RoleKey, role)
// }

// // WithScopes returns a context with scopes set.
// func WithScopes(ctx context.Context, scopes []string) context.Context {
// 	return context.WithValue(ctx, context.ScopesKey, scopes)
// }

// DefaultRateLimitConfig returns default rate limit configuration.
func DefaultRateLimitConfig(redisAddr string) *RateLimitConfig {
	cfg := &ratelimit.Config{
		RedisAddr:     redisAddr,
		RedisPassword: "",
		RedisDB:       0,
		DefaultPolicy: ratelimit.DefaultGlobalPolicy(),
		EnableMetrics: true,
		EnableAlerts:  true,
	}

	limiter, err := ratelimit.NewLimiter(cfg)
	if err != nil {
		// Log error but return config with nil limiter
		return &RateLimitConfig{
			Limiter:         nil,
			SkipPaths:       []string{"/health", "/metrics"},
			SkipIPs:         []string{},
			EnableByDefault: false,
			FailClosed:      false,    // Default to fail-open for backward compatibility
			DegradedLimit:   10,       // 10 requests per minute when degraded
		}
	}

	return &RateLimitConfig{
		Limiter:         limiter,
		SkipPaths:       []string{"/health", "/metrics", "/api/v1/docs"},
		SkipIPs:         []string{"127.0.0.1", "::1"},
		EnableByDefault: true,
		FailClosed:      false,       // Default to fail-open for backward compatibility
		DegradedLimit:   10,          // 10 requests per minute when degraded
	}
}

// GetCircuitBreakerStatus returns circuit breaker status for all paths.
func GetCircuitBreakerStatus(cfg *RateLimitConfig) map[string]interface{} {
	if cfg.Limiter == nil {
		return map[string]interface{}{
			"enabled": false,
		}
	}

	cb := cfg.Limiter.GetCircuitBreaker()
	if cb == nil {
		return map[string]interface{}{
			"enabled": false,
		}
	}

	states := cb.GetAllStates()
	circuitStates := make([]map[string]interface{}, len(states))

	for i, state := range states {
		circuitStates[i] = map[string]interface{}{
			"path":          state.Path,
			"state":         state.State,
			"failure_count": state.FailureCount,
		}
		if !state.LastFailureAt.IsZero() {
			circuitStates[i]["last_failure_at"] = state.LastFailureAt.Format(time.RFC3339)
		}
	}

	return map[string]interface{}{
		"enabled":  true,
		"circuits": circuitStates,
	}
}

// ResetCircuitBreakerForPath resets the circuit breaker for a specific path.
func ResetCircuitBreakerForPath(cfg *RateLimitConfig, path string) error {
	if cfg.Limiter == nil {
		return apperrors.New("rate limiter not configured", http.StatusInternalServerError)
	}

	cb := cfg.Limiter.GetCircuitBreaker()
	if cb == nil {
		return apperrors.New("circuit breaker not enabled", http.StatusServiceUnavailable)
	}

	return cb.Reset(path)
}

// OpenCircuitBreaker forcibly opens a circuit breaker.
func OpenCircuitBreaker(cfg *RateLimitConfig, path string, duration time.Duration) error {
	if cfg.Limiter == nil {
		return apperrors.New("rate limiter not configured", http.StatusInternalServerError)
	}

	cb := cfg.Limiter.GetCircuitBreaker()
	if cb == nil {
		return apperrors.New("circuit breaker not enabled", http.StatusServiceUnavailable)
	}

	cb.ForceOpen(path, duration)
	return nil
}

// degradedLimiter tracks degraded mode rate limiting per IP.
// This is a simple in-memory limiter used only when Redis is unavailable.
var degradedLimiter = &simpleRateLimiter{
	clients: make(map[string]*rateLimitClient),
	mu:      &sync.Mutex{},
}

// simpleRateLimiter provides basic in-memory rate limiting.
type simpleRateLimiter struct {
	clients map[string]*rateLimitClient
	mu      *sync.Mutex
}

// rateLimitClient tracks request counts for a single client.
type rateLimitClient struct {
	count     int
	windowEnd time.Time
}

// applyDegradedRateLimit applies basic rate limiting during Redis outages.
func applyDegradedRateLimit(r *http.Request, w http.ResponseWriter, clientIP string, limitPerMinute int) bool {
	now := time.Now()

	degradedLimiter.mu.Lock()
	defer degradedLimiter.mu.Unlock()

	client, exists := degradedLimiter.clients[clientIP]
	if !exists || now.After(client.windowEnd) {
		client = &rateLimitClient{
			count:     0,
			windowEnd: now.Add(time.Minute),
		}
		degradedLimiter.clients[clientIP] = client
	}

	client.count++

	if client.count > limitPerMinute {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"code":"RATE_LIMIT_EXCEEDED","message":"Service degraded. Too many requests."}`))
		return false
	}

	return true
}
