// Package middleware provides authentication and authorization middleware for the API gateway.
package middleware

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/openprint/openprint/internal/auth/jwt"
)

// Context keys for storing request-scoped data.
type contextKey string

const (
	UserIDKey contextKey = "user_id"
	EmailKey  contextKey = "email"
	OrgIDKey  contextKey = "org_id"
	RoleKey   contextKey = "role"
	TokenKey  contextKey = "token"
)

// JWTAuthConfig holds JWT authentication configuration.
type JWTAuthConfig struct {
	SecretKey  string
	JWTManager *jwt.Manager
	SkipPaths  []string
}

// JWTAuthMiddleware creates JWT authentication middleware for the gateway.
// It validates JWT tokens from the Authorization header and adds user info to the request context.
func JWTAuthMiddleware(cfg JWTAuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path should be skipped (public endpoints)
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

			// Check Bearer prefix
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				respondAuthError(w, "invalid authorization header format")
				return
			}

			// Validate token using provided manager or create temporary one
			var claims *jwt.Claims
			var err error

			if cfg.JWTManager != nil {
				claims, err = cfg.JWTManager.ValidateAccessToken(tokenString)
			} else {
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

			// Add user info to context for downstream middleware/handlers
			ctx := r.Context()
			ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, EmailKey, claims.Email)
			ctx = context.WithValue(ctx, OrgIDKey, claims.OrgID)
			ctx = context.WithValue(ctx, RoleKey, claims.Role)
			ctx = context.WithValue(ctx, TokenKey, tokenString)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID extracts the user ID from the request context.
func GetUserID(r *http.Request) string {
	return getStringFromContext(r.Context(), UserIDKey)
}

// GetEmail extracts the email from the request context.
func GetEmail(r *http.Request) string {
	return getStringFromContext(r.Context(), EmailKey)
}

// GetOrgID extracts the organization ID from the request context.
func GetOrgID(r *http.Request) string {
	return getStringFromContext(r.Context(), OrgIDKey)
}

// GetRole extracts the role from the request context.
func GetRole(r *http.Request) string {
	return getStringFromContext(r.Context(), RoleKey)
}

// getStringFromContext safely extracts a string value from context.
func getStringFromContext(ctx context.Context, key contextKey) string {
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

// OptionalAuthMiddleware attempts to authenticate but doesn't require it.
// If authentication fails, the handler is called without user context.
func OptionalAuthMiddleware(cfg JWTAuthConfig) func(http.Handler) http.Handler {
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
						ctx = context.WithValue(ctx, TokenKey, tokenString)
					}
				}
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
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

// Chain chains multiple middleware together.
func Chain(middleware ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(final http.Handler) http.Handler {
		for i := len(middleware) - 1; i >= 0; i-- {
			final = middleware[i](final)
		}
		return final
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

// LoggingMiddleware creates a middleware that logs HTTP requests.
func LoggingMiddleware(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := r.Context().Value("start_time")
			if start == nil {
				start = struct{}{}
			}

			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, r)

			logger.Printf("%s %s %d", r.Method, r.URL.Path, rw.status)
		})
	}
}
