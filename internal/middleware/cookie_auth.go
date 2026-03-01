// Package middleware provides HTTP middleware for cookie-based authentication.
//
// SECURITY CONSIDERATIONS FOR COOKIES:
//
// This package provides secure cookie helpers that enforce the following security attributes:
//
// 1. Secure Flag - Cookies are only sent over HTTPS connections, preventing
//    interception on the network. Always use Secure=true in production.
//
// 2. HttpOnly Flag - Cookies cannot be accessed via JavaScript document.cookie,
//    protecting against XSS attacks that attempt to steal session tokens.
//
// 3. SameSite Attribute - Controls cross-site cookie behavior:
//    - SameSiteStrict: Strongest CSRF protection, but may break legitimate navigation
//    - SameSiteLax: Balanced protection, allows top-level navigations (recommended)
//    - SameSiteNone: Allows cross-site requests; requires Secure=true
//
// 4. Cookie Expiration - Session cookies have limited lifetime (15 minutes default
//    for access tokens, 7 days for refresh tokens) to reduce the window of abuse.
//
// USAGE EXAMPLE:
//
//	security := middleware.DefaultCookieSecurity() // Use DevelopmentCookieSecurity() for local dev
//	middleware.SetSessionCookie(w, middleware.SessionCookieName, tokenValue, 15*time.Minute, security)
//
//	// On logout:
//	middleware.ClearSessionCookie(w, middleware.SessionCookieName, security)
package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

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

// CookieSecurityConfig defines security attributes for session cookies.
// These settings help prevent XSS and CSRF attacks.
type CookieSecurityConfig struct {
	// Secure ensures the cookie is only sent over HTTPS.
	// Should always be true in production.
	Secure bool

	// HttpOnly prevents JavaScript from accessing the cookie,
	// protecting against XSS attacks.
	HttpOnly bool

	// SameSite controls cross-site request handling.
	// - SameSiteStrict: Prevents CSRF, may break some navigation flows
	// - SameSiteLax: Balanced security, allows navigation from external sites
	// - SameSiteNone: Allows cross-site (requires Secure=true)
	SameSite http.SameSite

	// Domain specifies the cookie domain. If empty, defaults to current host.
	Domain string

	// Path limits the cookie to a specific path. If empty, applies to entire site.
	Path string
}

// DefaultCookieSecurity returns secure cookie settings for production use.
// These settings prioritize security while maintaining reasonable usability.
func DefaultCookieSecurity() *CookieSecurityConfig {
	return &CookieSecurityConfig{
		Secure:   true,  // Always use HTTPS in production
		HttpOnly: true,  // Prevent XSS access
		SameSite: http.SameSiteLaxMode, // Balance CSRF protection with usability
		Path:     "/",
	}
}

// DevelopmentCookieSecurity returns settings for local development.
// These settings allow HTTP and relaxed same-site policies for testing.
func DevelopmentCookieSecurity() *CookieSecurityConfig {
	return &CookieSecurityConfig{
		Secure:   false, // Allow HTTP locally
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	}
}

// SetSessionCookie sets a session cookie with appropriate security flags.
// This helper ensures consistent security attributes across all cookie operations.
//
// Example:
//
//	cfg := middleware.DefaultCookieSecurity()
//	middleware.SetSessionCookie(w, "session_token", tokenValue, 15*time.Minute, cfg)
func SetSessionCookie(w http.ResponseWriter, name, value string, maxAge time.Duration, security *CookieSecurityConfig) {
	if security == nil {
		security = DefaultCookieSecurity()
	}

	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     security.Path,
		Domain:   security.Domain,
		Expires:  time.Now().Add(maxAge),
		MaxAge:   int(maxAge.Seconds()),
		Secure:   security.Secure,
		HttpOnly: security.HttpOnly,
		SameSite: security.SameSite,
	}

	http.SetCookie(w, cookie)
}

// ClearSessionCookie removes a session cookie by setting it to expire immediately.
// This should be called on logout to invalidate the session.
func ClearSessionCookie(w http.ResponseWriter, name string, security *CookieSecurityConfig) {
	if security == nil {
		security = DefaultCookieSecurity()
	}

	cookie := &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     security.Path,
		Domain:   security.Domain,
		Expires:  time.Unix(1, 0), // Far past date
		MaxAge:   -1,
		Secure:   security.Secure,
		HttpOnly: security.HttpOnly,
		SameSite: security.SameSite,
	}

	http.SetCookie(w, cookie)
}

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
