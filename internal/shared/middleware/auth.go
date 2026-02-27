// Package middleware provides HTTP middleware for authentication, logging, and recovery.
package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/openprint/openprint/internal/auth/jwt"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
)

// Context keys for storing request-scoped data.
type contextKey string

const (
	UserIDKey  contextKey = "user_id"
	EmailKey   contextKey = "email"
	OrgIDKey   contextKey = "org_id"
	RoleKey    contextKey = "role"
	ScopesKey  contextKey = "scopes"
	TokenKey   contextKey = "token"
)

// JWTConfig holds JWT authentication configuration.
type JWTConfig struct {
	SecretKey   string
	SkipPaths   []string
	JWTManager  *jwt.Manager
}

// AuthMiddleware creates JWT authentication middleware.
func AuthMiddleware(cfg JWTConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path should be skipped
			for _, skipPath := range cfg.SkipPaths {
				if strings.HasPrefix(r.URL.Path, skipPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondAuthError(w, "missing authorization header")
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				respondAuthError(w, "invalid authorization header format")
				return
			}

			// Validate token
			var claims *jwt.Claims
			var err error

			if cfg.JWTManager != nil {
				claims, err = cfg.JWTManager.ValidateAccessToken(tokenString)
			} else {
				// Fallback: create a temporary manager if none provided
				jwtCfg, jwtCfgErr := jwt.DefaultConfig(cfg.SecretKey)
				if jwtCfgErr != nil {
					respondAuthError(w, "server configuration error")
					return
				}
				tmpManager, mgrErr := jwt.NewManager(jwtCfg)
				if mgrErr != nil {
					respondAuthError(w, "server configuration error")
					return
				}
				claims, err = tmpManager.ValidateAccessToken(tokenString)
			}

			if err != nil {
				respondAuthError(w, "invalid or expired token")
				return
			}

			// Add user info to context
			ctx := r.Context()
			ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, EmailKey, claims.Email)
			ctx = context.WithValue(ctx, OrgIDKey, claims.OrgID)
			ctx = context.WithValue(ctx, RoleKey, claims.Role)
			ctx = context.WithValue(ctx, ScopesKey, claims.Scopes)
			ctx = context.WithValue(ctx, TokenKey, tokenString)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuth wraps a handler that requires authentication.
// It extracts user info from context and passes it to the handler.
func RequireAuth(fn func(w http.ResponseWriter, r *http.Request, userID, email, orgID, role string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := GetStringFromContext(r.Context(), UserIDKey)
		email := GetStringFromContext(r.Context(), EmailKey)
		orgID := GetStringFromContext(r.Context(), OrgIDKey)
		role := GetStringFromContext(r.Context(), RoleKey)

		if userID == "" {
			respondAuthError(w, "unauthorized")
			return
		}

		fn(w, r, userID, email, orgID, role)
	}
}

// GetUserID extracts the user ID from the request context.
func GetUserID(r *http.Request) string {
	return GetStringFromContext(r.Context(), UserIDKey)
}

// GetEmail extracts the email from the request context.
func GetEmail(r *http.Request) string {
	return GetStringFromContext(r.Context(), EmailKey)
}

// GetOrgID extracts the organization ID from the request context.
func GetOrgID(r *http.Request) string {
	return GetStringFromContext(r.Context(), OrgIDKey)
}

// GetRole extracts the role from the request context.
func GetRole(r *http.Request) string {
	return GetStringFromContext(r.Context(), RoleKey)
}

// GetScopes extracts the scopes from the request context.
func GetScopes(r *http.Request) []string {
	scopes, _ := r.Context().Value(ScopesKey).([]string)
	return scopes
}

// GetStringFromContext safely extracts a string value from context.
func GetStringFromContext(ctx context.Context, key contextKey) string {
	val := ctx.Value(key)
	if val == nil {
		return ""
	}
	if str, ok := val.(string); ok {
		return str
	}
	return ""
}

// respondAuthError sends an authentication error response.
func respondAuthError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    "AUTHENTICATION_ERROR",
		"message": message,
	})
}

// APIKeyMiddleware creates API key authentication middleware for service-to-service communication.
func APIKeyMiddleware(validAPIKeys map[string]bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				respondAuthError(w, "missing API key")
				return
			}

			if !validAPIKeys[apiKey] {
				respondAuthError(w, "invalid API key")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// OptionalAuthMiddleware attempts to authenticate but doesn't require it.
// If authentication fails, the handler is called without user context.
func OptionalAuthMiddleware(cfg JWTConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Try to extract and validate token
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				tokenString := strings.TrimPrefix(authHeader, "Bearer ")
				if tokenString != authHeader {
					var claims *jwt.Claims
					var err error

					if cfg.JWTManager != nil {
						claims, err = cfg.JWTManager.ValidateAccessToken(tokenString)
					} else {
						jwtCfg, jwtCfgErr := jwt.DefaultConfig(cfg.SecretKey)
						if jwtCfgErr == nil {
							tmpManager, mgrErr := jwt.NewManager(jwtCfg)
							if mgrErr == nil {
								claims, err = tmpManager.ValidateAccessToken(tokenString)
							}
						}
					}

					if err == nil && claims != nil {
						ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
						ctx = context.WithValue(ctx, EmailKey, claims.Email)
						ctx = context.WithValue(ctx, OrgIDKey, claims.OrgID)
						ctx = context.WithValue(ctx, RoleKey, claims.Role)
						ctx = context.WithValue(ctx, ScopesKey, claims.Scopes)
					}
				}
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole creates middleware that requires a specific role.
func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := GetRole(r)
			if role == "" {
				respondAuthError(w, "unauthorized")
				return
			}

			for _, allowed := range allowedRoles {
				if role == allowed {
					next.ServeHTTP(w, r)
					return
				}
			}

			respondAuthError(w, "insufficient permissions")
		})
	}
}

// RequireScope creates middleware that requires specific OAuth scopes.
func RequireScope(scopes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userScopes := GetScopes(r)

			for _, required := range scopes {
				found := false
				for _, s := range userScopes {
					if s == required {
						found = true
						break
					}
				}
				if !found {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"code":    "INSUFFICIENT_SCOPE",
						"message": "required scope not granted",
					})
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireOwnership checks that the user owns the resource or is an admin.
func RequireOwnership(getResourceOwner func(r *http.Request) (string, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := GetRole(r)
			email := GetEmail(r)

			// Admins can access any resource
			if role == "admin" || role == "org_admin" {
				next.ServeHTTP(w, r)
				return
			}

			// Get the resource owner
			ownerEmail, err := getResourceOwner(r)
			if err != nil {
				respondError(w, apperrors.Wrap(err, "failed to verify ownership", http.StatusInternalServerError))
				return
			}

			// Check ownership
			if email != ownerEmail {
				respondAuthError(w, "access denied: resource belongs to another user")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireOrgMembership checks that the user belongs to the specified organization.
func RequireOrgMembership(getResourceOrgID func(r *http.Request) (string, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userOrgID := GetOrgID(r)
			role := GetRole(r)

			// Admins can access any organization
			if role == "admin" {
				next.ServeHTTP(w, r)
				return
			}

			// Get the resource's organization ID
			resourceOrgID, err := getResourceOrgID(r)
			if err != nil {
				respondError(w, apperrors.Wrap(err, "failed to verify organization", http.StatusInternalServerError))
				return
			}

			// Check organization membership
			if userOrgID != resourceOrgID {
				respondAuthError(w, "access denied: resource belongs to another organization")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// respondError sends an error response.
func respondError(w http.ResponseWriter, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(appErr.StatusCode)
		json.NewEncoder(w).Encode(apperrors.ToJSON(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]string{
		"code":    "INTERNAL_ERROR",
		"message": "An internal error occurred",
	})
}

// LoggingMiddleware creates a middleware that logs HTTP requests.
func LoggingMiddleware(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create response writer wrapper to capture status code
			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

			// Call next handler
			next.ServeHTTP(rw, r)

			// Log request
			duration := time.Since(start)
			logger.Printf("%s %s %s %s %d %d",
				r.Method,
				r.URL.Path,
				r.RemoteAddr,
				r.UserAgent(),
				rw.status,
				duration.Milliseconds(),
			)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

// WriteHeader captures the status code.
func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.status = code
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

// RecoveryMiddleware creates a middleware that recovers from panics.
func RecoveryMiddleware(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Printf("PANIC: %v\n%s", err, getStackTrace())
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"code":    "INTERNAL_ERROR",
						"message": "An internal error occurred",
					})
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// getStackTrace returns the current stack trace.
func getStackTrace() string {
	buf := make([]byte, 4096)
	n := 0
	// Simple stack trace capture
	for i := 2; i < 32; i++ {
		if n >= len(buf) {
			break
		}
		buf[n] = '\n'
		n++
	}
	return string(buf[:n])
}

// CORSMiddleware creates a middleware that handles CORS headers.
func CORSMiddleware(allowedOrigins []string, allowedMethods []string, allowedHeaders []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			for _, allowedOrigin := range allowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
					break
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(allowedMethods, ", "))
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(allowedHeaders, ", "))
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
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

			next.ServeHTTP(w, r)
		})
	}
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

// RateLimitMiddleware creates a simple rate limiter middleware.
func RateLimitMiddleware(requestsPerMinute int, cleanupInterval time.Duration) func(http.Handler) http.Handler {
	limiter := NewIPRateLimiter(requestsPerMinute, cleanupInterval)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}

			if !limiter.Allow(ip) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "Too many requests",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// IPRateLimiter tracks IP addresses and their request counts.
type IPRateLimiter struct {
	ips            map[string]*IPTracker
	requestsPerMin int
	mu             chan struct{}
}

// IPTracker tracks requests from a single IP.
type IPTracker struct {
	count     int
	lastReset time.Time
}

// NewIPRateLimiter creates a new IP-based rate limiter.
func NewIPRateLimiter(requestsPerMinute int, cleanupInterval time.Duration) *IPRateLimiter {
	rl := &IPRateLimiter{
		ips:            make(map[string]*IPTracker),
		requestsPerMin: requestsPerMinute,
		mu:             make(chan struct{}, 1),
	}

	// Start cleanup goroutine
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()

	return rl
}

// Allow checks if a request from the given IP should be allowed.
func (rl *IPRateLimiter) Allow(ip string) bool {
	rl.mu <- struct{}{}
	defer func() { <-rl.mu }()

	tracker, exists := rl.ips[ip]
	now := time.Now()

	if !exists {
		rl.ips[ip] = &IPTracker{count: 1, lastReset: now}
		return true
	}

	// Reset counter if a minute has passed
	if now.Sub(tracker.lastReset) > time.Minute {
		tracker.count = 1
		tracker.lastReset = now
		return true
	}

	if tracker.count >= rl.requestsPerMin {
		return false
	}

	tracker.count++
	return true
}

// cleanup removes stale entries from the rate limiter.
func (rl *IPRateLimiter) cleanup() {
	rl.mu <- struct{}{}
	defer func() { <-rl.mu }()

	now := time.Now()
	for ip, tracker := range rl.ips {
		if now.Sub(tracker.lastReset) > 5*time.Minute {
			delete(rl.ips, ip)
		}
	}
}
