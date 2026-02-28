// Package middleware provides rate limiting middleware for the API gateway.
package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiterConfig holds rate limiting configuration.
type RateLimiterConfig struct {
	RequestsPerMinute int
	CleanupInterval   time.Duration
}

// DefaultRateLimiterConfig returns sensible defaults for rate limiting.
func DefaultRateLimiterConfig() *RateLimiterConfig {
	return &RateLimiterConfig{
		RequestsPerMinute: 100,
		CleanupInterval:   5 * time.Minute,
	}
}

// IPRateLimiter tracks rate limiters for each IP address.
type IPRateLimiter struct {
	limiters map[string]*limiterEntry
	mu       sync.RWMutex
	config   *RateLimiterConfig
}

// limiterEntry holds a rate limiter and its last access time.
type limiterEntry struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

// NewIPRateLimiter creates a new IP-based rate limiter.
func NewIPRateLimiter(config *RateLimiterConfig) *IPRateLimiter {
	if config == nil {
		config = DefaultRateLimiterConfig()
	}

	rl := &IPRateLimiter{
		limiters: make(map[string]*limiterEntry),
		config:   config,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// getLimiter returns the rate limiter for the given IP address.
func (rl *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, exists := rl.limiters[ip]
	if !exists {
		// Create new limiter: requestsPerMinute requests per minute
		// rate.Every converts to duration per request
		interval := rate.Every(time.Minute / time.Duration(rl.config.RequestsPerMinute))
		limiter := rate.NewLimiter(interval, rl.config.RequestsPerMinute)
		rl.limiters[ip] = &limiterEntry{
			limiter:    limiter,
			lastAccess: time.Now(),
		}
		return limiter
	}

	entry.lastAccess = time.Now()
	return entry.limiter
}

// Allow checks if a request from the given IP should be allowed.
func (rl *IPRateLimiter) Allow(ip string) bool {
	limiter := rl.getLimiter(ip)
	return limiter.Allow()
}

// cleanup removes stale entries from the rate limiter map.
func (rl *IPRateLimiter) cleanup() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, entry := range rl.limiters {
			// Remove entries not accessed in 2x the cleanup interval
			if now.Sub(entry.lastAccess) > 2*rl.config.CleanupInterval {
				delete(rl.limiters, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// GetIP extracts the IP address from the request.
// It checks X-Forwarded-For and X-Real-IP headers for proxied requests.
func GetIP(r *http.Request) string {
	// Check X-Forwarded-For header (can contain multiple IPs)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP (original client)
		if idx := len(xff); idx > 0 {
			for i, c := range xff {
				if c == ',' {
					return xff[:i]
				}
			}
			return xff
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// RateLimitMiddleware creates a rate limiting middleware using golang.org/x/time/rate.
func RateLimitMiddleware(config *RateLimiterConfig) func(http.Handler) http.Handler {
	limiter := NewIPRateLimiter(config)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := GetIP(r)

			if !limiter.Allow(ip) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-RateLimit-Limit", string(rune(config.RequestsPerMinute)))
				w.Header().Set("Retry-After", "60")
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

// PerUserRateLimiter tracks rate limiters for each user.
type PerUserRateLimiter struct {
	limiters map[string]*limiterEntry
	mu       sync.RWMutex
	config   *RateLimiterConfig
}

// NewPerUserRateLimiter creates a new user-based rate limiter.
func NewPerUserRateLimiter(config *RateLimiterConfig) *PerUserRateLimiter {
	if config == nil {
		config = DefaultRateLimiterConfig()
	}

	rl := &PerUserRateLimiter{
		limiters: make(map[string]*limiterEntry),
		config:   config,
	}

	go rl.cleanup()

	return rl
}

// getLimiter returns the rate limiter for the given user.
func (rl *PerUserRateLimiter) getLimiter(userID string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, exists := rl.limiters[userID]
	if !exists {
		interval := rate.Every(time.Minute / time.Duration(rl.config.RequestsPerMinute))
		limiter := rate.NewLimiter(interval, rl.config.RequestsPerMinute)
		rl.limiters[userID] = &limiterEntry{
			limiter:    limiter,
			lastAccess: time.Now(),
		}
		return limiter
	}

	entry.lastAccess = time.Now()
	return entry.limiter
}

// Allow checks if a request from the given user should be allowed.
func (rl *PerUserRateLimiter) Allow(userID string) bool {
	limiter := rl.getLimiter(userID)
	return limiter.Allow()
}

func (rl *PerUserRateLimiter) cleanup() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for userID, entry := range rl.limiters {
			if now.Sub(entry.lastAccess) > 2*rl.config.CleanupInterval {
				delete(rl.limiters, userID)
			}
		}
		rl.mu.Unlock()
	}
}

// PerUserRateLimitMiddleware creates rate limiting middleware per user.
// Falls back to IP-based limiting if user is not authenticated.
func PerUserRateLimitMiddleware(config *RateLimiterConfig) func(http.Handler) http.Handler {
	limiter := NewPerUserRateLimiter(config)
	ipLimiter := NewIPRateLimiter(config)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try user-based limiting first
			userID := GetUserID(r)
			allowed := false

			if userID != "" {
				allowed = limiter.Allow(userID)
			} else {
				// Fall back to IP-based limiting for unauthenticated requests
				ip := GetIP(r)
				allowed = ipLimiter.Allow(ip)
			}

			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-RateLimit-Limit", string(rune(config.RequestsPerMinute)))
				w.Header().Set("Retry-After", "60")
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

// BurstRateLimitMiddleware creates a rate limiter with burst capacity.
// Useful for handling short spikes in traffic.
type BurstRateLimitMiddleware struct {
	limiter *rate.Limiter
}

// NewBurstRateLimitMiddleware creates a new burst rate limiter.
// rps is requests per second, burst is the maximum burst size.
func NewBurstRateLimitMiddleware(rps int, burst int) *BurstRateLimitMiddleware {
	return &BurstRateLimitMiddleware{
		limiter: rate.NewLimiter(rate.Limit(rps), burst),
	}
}

// Middleware returns the middleware function.
func (m *BurstRateLimitMiddleware) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !m.limiter.Allow() {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "Server is experiencing high load. Please try again later.",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
