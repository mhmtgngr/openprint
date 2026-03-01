// Package middleware provides HTTP middleware for cookie-based authentication.
package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/openprint/openprint/internal/auth/jwt"
)

const (
	// SessionCookieName is the name of the session cookie
	SessionCookieName = "openprint_session"
	// RefreshCookieName is the name of the refresh token cookie
	RefreshCookieName = "openprint_refresh"

	UserIDKey  contextKey = "user_id"
	EmailKey   contextKey = "email"
	OrgIDKey   contextKey = "org_id"
	RoleKey    contextKey = "role"
	ScopesKey  contextKey = "scopes"
	TokenKey   contextKey = "token"
)

// CookieAuthConfig holds cookie authentication configuration.
type CookieAuthConfig struct {
	SecretKey  string
	SkipPaths  []string
	JWTManager *jwt.Manager
}

// CookieAuthMiddleware creates middleware that authenticates using httpOnly cookies.
// This is an alternative to AuthMiddleware for production environments with cookie-based auth.
func CookieAuthMiddleware(cfg CookieAuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path should be skipped
			for _, skipPath := range cfg.SkipPaths {
				if strings.HasPrefix(r.URL.Path, skipPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Try to get token from cookie first
			sessionCookie, err := r.Cookie(SessionCookieName)
			tokenString := ""

			if err == nil {
				tokenString = sessionCookie.Value
			} else {
				// Fall back to Authorization header
				authHeader := r.Header.Get("Authorization")
				if authHeader != "" {
					tokenString = strings.TrimPrefix(authHeader, "Bearer ")
				}
			}

			if tokenString == "" {
				respondAuthError(w, "missing authentication")
				return
			}

			// Validate token
			var claims *jwt.Claims

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

// OptionalAuthMiddleware attempts to authenticate from cookies or headers but doesn't require it.
// If authentication fails, the handler is called without user context.
func OptionalCookieAuthMiddleware(cfg CookieAuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			var tokenString string

			// Try cookie first
			if sessionCookie, err := r.Cookie(SessionCookieName); err == nil {
				tokenString = sessionCookie.Value
			}

			// Fall back to Authorization header
			if tokenString == "" {
				if authHeader := r.Header.Get("Authorization"); authHeader != "" {
					tokenString = strings.TrimPrefix(authHeader, "Bearer ")
				}
			}

			// Validate token if present
			if tokenString != "" {
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
					ctx = context.WithValue(ctx, TokenKey, tokenString)
				}
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CORSMiddlewareWithCredentials creates CORS middleware optimized for cookie credentials.
// This sets proper CORS headers for cookie-based authentication.
func CORSMiddlewareWithCredentials(allowedOrigins []string) func(http.Handler) http.Handler {
	allowedMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}
	allowedHeaders := []string{"Content-Type", "Authorization", "X-Requested-With"}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			for _, allowedOrigin := range allowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					// When using credentials, must use exact origin, not wildcard
					if origin != "" {
						w.Header().Set("Access-Control-Allow-Origin", origin)
					}
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

// respondAuthError sends an authentication error response.
func respondAuthError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    "AUTHENTICATION_ERROR",
		"message": message,
	})
}
