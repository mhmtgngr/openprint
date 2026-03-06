package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetUserID_FromContext(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), UserIDKey, "user-123")
	req = req.WithContext(ctx)

	got := GetUserID(req)
	if got != "user-123" {
		t.Errorf("GetUserID() = %q, want %q", got, "user-123")
	}
}

func TestGetUserID_Empty(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	got := GetUserID(req)
	if got != "" {
		t.Errorf("GetUserID() = %q, want empty", got)
	}
}

func TestGetEmail_FromContext(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), EmailKey, "user@example.com")
	req = req.WithContext(ctx)

	got := GetEmail(req)
	if got != "user@example.com" {
		t.Errorf("GetEmail() = %q, want %q", got, "user@example.com")
	}
}

func TestGetOrgID_FromContext(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), OrgIDKey, "org-456")
	req = req.WithContext(ctx)

	got := GetOrgID(req)
	if got != "org-456" {
		t.Errorf("GetOrgID() = %q, want %q", got, "org-456")
	}
}

func TestGetRole_FromContext(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), RoleKey, "admin")
	req = req.WithContext(ctx)

	got := GetRole(req)
	if got != "admin" {
		t.Errorf("GetRole() = %q, want %q", got, "admin")
	}
}

func TestGetStringFromContext_NonStringValue(t *testing.T) {
	ctx := context.WithValue(context.Background(), UserIDKey, 123)
	got := getStringFromContext(ctx, UserIDKey)
	if got != "" {
		t.Errorf("getStringFromContext with non-string value = %q, want empty", got)
	}
}

func TestJWTAuthMiddleware_SkipPaths(t *testing.T) {
	cfg := JWTAuthConfig{
		SecretKey: "test-secret-key-that-is-32-chars!",
		SkipPaths: []string{"/health", "/api/v1/public"},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := JWTAuthMiddleware(cfg)
	wrappedHandler := middleware(handler)

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{"health path skipped", "/health", http.StatusOK},
		{"public path skipped", "/api/v1/public/docs", http.StatusOK},
		{"protected path without auth", "/api/v1/jobs", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rr := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
		})
	}
}

func TestJWTAuthMiddleware_MissingAuthHeader(t *testing.T) {
	cfg := JWTAuthConfig{
		SecretKey: "test-secret-key-that-is-32-chars!",
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := JWTAuthMiddleware(cfg)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/api/v1/jobs", nil)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestJWTAuthMiddleware_InvalidFormat(t *testing.T) {
	cfg := JWTAuthConfig{
		SecretKey: "test-secret-key-that-is-32-chars!",
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := JWTAuthMiddleware(cfg)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/api/v1/jobs", nil)
	req.Header.Set("Authorization", "Basic token123")
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestAPIKeyMiddleware_MissingKey(t *testing.T) {
	validKeys := map[string]bool{"valid-key": true}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := APIKeyMiddleware(validKeys)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestAPIKeyMiddleware_ValidKey(t *testing.T) {
	validKeys := map[string]bool{"valid-key": true}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := APIKeyMiddleware(validKeys)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "valid-key")
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestAPIKeyMiddleware_InvalidKey(t *testing.T) {
	validKeys := map[string]bool{"valid-key": true}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := APIKeyMiddleware(validKeys)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "invalid-key")
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestRequireRole_Authorized(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequireRole("admin", "editor")
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), RoleKey, "admin")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestRequireRole_Unauthorized(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequireRole("admin")
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), RoleKey, "viewer")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestRequireRole_NoRole(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequireRole("admin")
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestChain(t *testing.T) {
	var callOrder []string

	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callOrder = append(callOrder, "mw1")
			next.ServeHTTP(w, r)
		})
	}

	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callOrder = append(callOrder, "mw2")
			next.ServeHTTP(w, r)
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callOrder = append(callOrder, "handler")
		w.WriteHeader(http.StatusOK)
	})

	chained := Chain(mw1, mw2)(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	chained.ServeHTTP(rr, req)

	expected := []string{"mw1", "mw2", "handler"}
	if len(callOrder) != len(expected) {
		t.Fatalf("call order length = %d, want %d", len(callOrder), len(expected))
	}
	for i, v := range callOrder {
		if v != expected[i] {
			t.Errorf("callOrder[%d] = %q, want %q", i, v, expected[i])
		}
	}
}

func TestRespondAuthError(t *testing.T) {
	rr := httptest.NewRecorder()
	respondAuthError(rr, "test error message")

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rr, status: http.StatusOK}

	rw.WriteHeader(http.StatusCreated)
	if rw.status != http.StatusCreated {
		t.Errorf("status = %d, want %d", rw.status, http.StatusCreated)
	}

	// Second call should be ignored
	rw.WriteHeader(http.StatusNotFound)
	if rw.status != http.StatusCreated {
		t.Errorf("status after second call = %d, want %d", rw.status, http.StatusCreated)
	}
}

func TestOptionalAuthMiddleware_NoAuth(t *testing.T) {
	cfg := JWTAuthConfig{
		SecretKey: "test-secret-key-that-is-32-chars!",
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should still be called even without auth
		w.WriteHeader(http.StatusOK)
	})

	middleware := OptionalAuthMiddleware(cfg)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}
