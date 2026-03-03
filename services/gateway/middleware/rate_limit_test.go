package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDefaultRateLimiterConfig(t *testing.T) {
	cfg := DefaultRateLimiterConfig()
	if cfg.RequestsPerMinute != 100 {
		t.Errorf("RequestsPerMinute = %d, want 100", cfg.RequestsPerMinute)
	}
	if cfg.CleanupInterval != 5*time.Minute {
		t.Errorf("CleanupInterval = %v, want %v", cfg.CleanupInterval, 5*time.Minute)
	}
}

func TestGetIP_XForwardedFor(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	req.RemoteAddr = "192.168.1.1:1234"

	ip := GetIP(req)
	if ip != "10.0.0.1" {
		t.Errorf("GetIP() = %q, want %q", ip, "10.0.0.1")
	}
}

func TestGetIP_XForwardedForSingle(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	req.RemoteAddr = "192.168.1.1:1234"

	ip := GetIP(req)
	if ip != "10.0.0.1" {
		t.Errorf("GetIP() = %q, want %q", ip, "10.0.0.1")
	}
}

func TestGetIP_XRealIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Real-IP", "10.0.0.3")
	req.RemoteAddr = "192.168.1.1:1234"

	ip := GetIP(req)
	if ip != "10.0.0.3" {
		t.Errorf("GetIP() = %q, want %q", ip, "10.0.0.3")
	}
}

func TestGetIP_RemoteAddr(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"

	ip := GetIP(req)
	if ip != "192.168.1.1" {
		t.Errorf("GetIP() = %q, want %q", ip, "192.168.1.1")
	}
}

func TestGetIP_RemoteAddrNoPort(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1"

	ip := GetIP(req)
	// SplitHostPort will fail, returns the raw RemoteAddr
	if ip != "192.168.1.1" {
		t.Errorf("GetIP() = %q, want %q", ip, "192.168.1.1")
	}
}

func TestIPRateLimiter_Allow(t *testing.T) {
	cfg := &RateLimiterConfig{
		RequestsPerMinute: 5,
		CleanupInterval:   1 * time.Hour, // Long interval so cleanup doesn't interfere
	}

	limiter := NewIPRateLimiter(cfg)

	// First request should be allowed
	if !limiter.Allow("10.0.0.1") {
		t.Error("first request should be allowed")
	}

	// Different IPs should be independent
	if !limiter.Allow("10.0.0.2") {
		t.Error("first request from different IP should be allowed")
	}
}

func TestIPRateLimiter_NilConfig(t *testing.T) {
	limiter := NewIPRateLimiter(nil)
	if limiter.config.RequestsPerMinute != 100 {
		t.Errorf("RequestsPerMinute = %d, want 100 (default)", limiter.config.RequestsPerMinute)
	}
}

func TestPerUserRateLimiter_Allow(t *testing.T) {
	cfg := &RateLimiterConfig{
		RequestsPerMinute: 5,
		CleanupInterval:   1 * time.Hour,
	}

	limiter := NewPerUserRateLimiter(cfg)

	if !limiter.Allow("user-1") {
		t.Error("first request should be allowed")
	}

	if !limiter.Allow("user-2") {
		t.Error("first request from different user should be allowed")
	}
}

func TestPerUserRateLimiter_NilConfig(t *testing.T) {
	limiter := NewPerUserRateLimiter(nil)
	if limiter.config.RequestsPerMinute != 100 {
		t.Errorf("RequestsPerMinute = %d, want 100 (default)", limiter.config.RequestsPerMinute)
	}
}

func TestNewBurstRateLimitMiddleware(t *testing.T) {
	m := NewBurstRateLimitMiddleware(10, 20)
	if m == nil {
		t.Fatal("NewBurstRateLimitMiddleware returned nil")
	}
	if m.limiter == nil {
		t.Fatal("limiter should be initialized")
	}
}

func TestBurstRateLimitMiddleware_AllowsRequests(t *testing.T) {
	m := NewBurstRateLimitMiddleware(100, 100)
	middleware := m.Middleware()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}
