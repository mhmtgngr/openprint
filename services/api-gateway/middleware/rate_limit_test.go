package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewInMemoryRateLimiter(t *testing.T) {
	limiter := NewInMemoryRateLimiter()
	if limiter == nil {
		t.Fatal("NewInMemoryRateLimiter returned nil")
	}
	if limiter.limiters == nil {
		t.Fatal("limiters map should be initialized")
	}
}

func TestInMemoryRateLimiter_Allow_FirstRequest(t *testing.T) {
	limiter := NewInMemoryRateLimiter()
	allowed := limiter.Allow("test-key", 10, time.Minute)
	if !allowed {
		t.Error("first request should be allowed")
	}
}

func TestInMemoryRateLimiter_Allow_WithinLimit(t *testing.T) {
	limiter := NewInMemoryRateLimiter()
	limit := 5

	for i := 0; i < limit; i++ {
		allowed := limiter.Allow("test-key", limit, time.Minute)
		if !allowed {
			t.Errorf("request %d should be allowed (limit=%d)", i+1, limit)
		}
	}
}

func TestInMemoryRateLimiter_Allow_ExceedsLimit(t *testing.T) {
	limiter := NewInMemoryRateLimiter()
	limit := 3

	// Exhaust the limit
	for i := 0; i < limit; i++ {
		limiter.Allow("test-key", limit, time.Minute)
	}

	// Next request should be denied
	allowed := limiter.Allow("test-key", limit, time.Minute)
	if allowed {
		t.Error("request exceeding limit should be denied")
	}
}

func TestInMemoryRateLimiter_Allow_DifferentKeys(t *testing.T) {
	limiter := NewInMemoryRateLimiter()

	// Each key gets its own limit
	if !limiter.Allow("key-a", 1, time.Minute) {
		t.Error("key-a first request should be allowed")
	}
	if !limiter.Allow("key-b", 1, time.Minute) {
		t.Error("key-b first request should be allowed")
	}

	// key-a exhausted, key-b still available
	if limiter.Allow("key-a", 1, time.Minute) {
		t.Error("key-a second request should be denied")
	}
}

func TestInMemoryRateLimiter_Cleanup(t *testing.T) {
	limiter := NewInMemoryRateLimiter()

	// Add an entry
	limiter.Allow("test-key", 10, time.Minute)

	// Manually set lastRefill to be old
	limiter.mu.RLock()
	entry := limiter.limiters["test-key"]
	limiter.mu.RUnlock()

	entry.mu.Lock()
	entry.lastRefill = time.Now().Add(-15 * time.Minute)
	entry.mu.Unlock()

	// Run cleanup
	limiter.cleanup()

	// Entry should be removed
	limiter.mu.RLock()
	_, exists := limiter.limiters["test-key"]
	limiter.mu.RUnlock()

	if exists {
		t.Error("stale entry should have been cleaned up")
	}
}

func TestRateLimitMiddleware_AllowsRequests(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RateLimitMiddleware(100, 5*time.Minute)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestRateLimitMiddleware_RejectsExcessiveRequests(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RateLimitMiddleware(2, 5*time.Minute)
	wrappedHandler := middleware(handler)

	// Send requests exceeding the limit
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		rr := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rr, req)

		if i < 2 && rr.Code != http.StatusOK {
			t.Errorf("request %d: status = %d, want %d", i+1, rr.Code, http.StatusOK)
		}
		if i == 2 && rr.Code != http.StatusTooManyRequests {
			t.Errorf("request %d: status = %d, want %d", i+1, rr.Code, http.StatusTooManyRequests)
		}
	}
}

func TestRateLimitMiddleware_UsesAPIKey(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RateLimitMiddleware(1, 5*time.Minute)
	wrappedHandler := middleware(handler)

	// First request with API key A
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.Header.Set("X-API-Key", "key-a")
	rr1 := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Errorf("key-a first request: status = %d, want %d", rr1.Code, http.StatusOK)
	}

	// First request with API key B should still work (different key)
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set("X-API-Key", "key-b")
	rr2 := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Errorf("key-b first request: status = %d, want %d", rr2.Code, http.StatusOK)
	}
}

func TestRateLimitMiddleware_SetsHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RateLimitMiddleware(1, 5*time.Minute)
	wrappedHandler := middleware(handler)

	// Exhaust limit
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "10.0.0.2:1234"
	wrappedHandler.ServeHTTP(httptest.NewRecorder(), req1)

	// This request should be rate limited
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "10.0.0.2:1234"
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req2)

	if rr.Header().Get("X-RateLimit-Limit") != "1" {
		t.Errorf("X-RateLimit-Limit = %q, want %q", rr.Header().Get("X-RateLimit-Limit"), "1")
	}
	if rr.Header().Get("X-RateLimit-Remaining") != "0" {
		t.Errorf("X-RateLimit-Remaining = %q, want %q", rr.Header().Get("X-RateLimit-Remaining"), "0")
	}
	if rr.Header().Get("X-RateLimit-Reset") == "" {
		t.Error("X-RateLimit-Reset should be set")
	}
}
