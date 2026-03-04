// Package middleware provides rate limiting middleware for the API gateway.
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RateLimiter interface defines rate limiting behavior.
type RateLimiter interface {
	Allow(key string, limit int, window time.Duration) bool
}

// InMemoryRateLimiter provides an in-memory rate limiter.
type InMemoryRateLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*tokenBucketLimiter
}

// tokenBucketLimiter implements token bucket rate limiting.
type tokenBucketLimiter struct {
	tokens     int
	maxTokens  int
	refillRate time.Duration
	lastRefill time.Time
	mu         sync.Mutex
}

// NewInMemoryRateLimiter creates a new in-memory rate limiter.
func NewInMemoryRateLimiter() *InMemoryRateLimiter {
	return &InMemoryRateLimiter{
		limiters: make(map[string]*tokenBucketLimiter),
	}
}

// Allow checks if a request should be allowed based on rate limit.
func (r *InMemoryRateLimiter) Allow(key string, limit int, window time.Duration) bool {
	r.mu.RLock()
	limiter, exists := r.limiters[key]
	r.mu.RUnlock()

	if !exists {
		r.mu.Lock()
		limiter = &tokenBucketLimiter{
			tokens:     limit - 1,
			maxTokens:  limit,
			refillRate: window / time.Duration(limit),
			lastRefill: time.Now(),
		}
		r.limiters[key] = limiter
		r.mu.Unlock()
		return true
	}

	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(limiter.lastRefill)
	tokensToAdd := int(elapsed / limiter.refillRate)

	if tokensToAdd > 0 {
		limiter.tokens += tokensToAdd
		if limiter.tokens > limiter.maxTokens {
			limiter.tokens = limiter.maxTokens
		}
		limiter.lastRefill = now
	}

	// Check if request is allowed
	if limiter.tokens > 0 {
		limiter.tokens--
		return true
	}

	return false
}

// DatabaseRateLimiter provides a database-backed rate limiter.
type DatabaseRateLimiter struct {
	db *pgxpool.Pool
}

// NewDatabaseRateLimiter creates a new database-backed rate limiter.
func NewDatabaseRateLimiter(db *pgxpool.Pool) *DatabaseRateLimiter {
	return &DatabaseRateLimiter{db: db}
}

// Allow checks if a request should be allowed based on rate limit.
func (r *DatabaseRateLimiter) Allow(key string, limit int, window time.Duration) bool {
	ctx := context.Background()

	// Check rate limit using database
	var count int
	query := `
		SELECT COUNT(*)
		FROM api_usage_logs
		WHERE api_key_id = $1
		  AND created_at > NOW() - $2::interval
	`

	err := r.db.QueryRow(ctx, query, key, window).Scan(&count)
	if err != nil {
		// Log error but allow request on failure
		return true
	}

	return count < limit
}

// RateLimitMiddleware creates a rate limiting middleware.
func RateLimitMiddleware(requestsPerMinute int, cleanupInterval time.Duration) func(http.Handler) http.Handler {
	limiter := NewInMemoryRateLimiter()

	// Start cleanup goroutine
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			limiter.cleanup()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get API key from header
			apiKey := r.Header.Get("X-API-Key")

			// Fall back to getting user ID from context
			key := apiKey
			if key == "" {
				if userID := r.Context().Value("user_id"); userID != nil {
					key = fmt.Sprintf("user:%v", userID)
				} else {
					// Use IP address as fallback
					key = r.RemoteAddr
				}
			}

			window := 1 * time.Minute

			if !limiter.Allow(key, requestsPerMinute, window) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", requestsPerMinute))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(window).Unix()))
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"code":"RATE_LIMIT_EXCEEDED","message":"Too many requests"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// cleanup removes stale entries from the rate limiter.
func (r *InMemoryRateLimiter) cleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Remove entries that haven't been used recently
	now := time.Now()
	for key, limiter := range r.limiters {
		limiter.mu.Lock()
		stale := now.Sub(limiter.lastRefill) > 10*time.Minute
		limiter.mu.Unlock()

		if stale {
			delete(r.limiters, key)
		}
	}
}

// APIKeyMiddleware creates middleware that validates and tracks API keys.
func APIKeyMiddleware(db *pgxpool.Pool, skipPaths []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path should be skipped
			for _, skipPath := range skipPaths {
				if len(r.URL.Path) >= len(skipPath) && r.URL.Path[:len(skipPath)] == skipPath {
					next.ServeHTTP(w, r)
					return
				}
			}

			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				// No API key, continue to auth middleware
				next.ServeHTTP(w, r)
				return
			}

			// Validate API key against database
			ctx := r.Context()
			var orgID, userID string
			var isActive bool
			var scopes []string
			var keyID string

			err := db.QueryRow(ctx, "SELECT id, organization_id, created_by, is_active, scopes FROM api_keys WHERE key_hash = $1", apiKey).
				Scan(&keyID, &orgID, &userID, &isActive, &scopes)

			if err == nil && isActive {
				// API key is valid, set context values
				ctx = context.WithValue(ctx, "api_key_id", keyID)
				ctx = context.WithValue(ctx, "org_id", orgID)
				ctx = context.WithValue(ctx, "user_id", userID)
				ctx = context.WithValue(ctx, "scopes", scopes)

				// Log API usage
				go logAPIUsage(context.Background(), db, keyID, r)

				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Invalid API key
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"code":"INVALID_API_KEY","message":"Invalid or expired API key"}`))
		})
	}
}

// logAPIUsage logs an API request for rate limiting and analytics.
func logAPIUsage(ctx context.Context, db *pgxpool.Pool, keyID string, r *http.Request) {
	query := `
		INSERT INTO api_usage_logs (
			api_key_id, organization_id, method, path, status_code,
			latency_ms, created_at
		) VALUES (
			$1::uuid,
			(SELECT organization_id FROM api_keys WHERE id = $1::uuid),
			$2, $3, $4, $5, NOW()
		)
	`

	// Execute without blocking
	db.Exec(ctx, query, keyID, r.Method, r.URL.Path, 200, 0)
}
