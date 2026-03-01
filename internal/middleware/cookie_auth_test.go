// Package middleware provides tests for cookie authentication middleware.
package middleware

import (
	"net/http"
	"os"
	"testing"
	"time"
)

func TestDefaultCookieSecurity(t *testing.T) {
	security := DefaultCookieSecurity()

	if !security.Secure {
		t.Error("DefaultCookieSecurity() should set Secure=true")
	}
	if !security.HttpOnly {
		t.Error("DefaultCookieSecurity() should set HttpOnly=true")
	}
	if security.SameSite != http.SameSiteLaxMode {
		t.Errorf("DefaultCookieSecurity() SameSite = %v, want SameSiteLaxMode", security.SameSite)
	}
	if security.Path != "/" {
		t.Errorf("DefaultCookieSecurity() Path = %v, want /", security.Path)
	}
}

func TestProductionCookieSecurity(t *testing.T) {
	security := ProductionCookieSecurity()

	if !security.Secure {
		t.Error("ProductionCookieSecurity() MUST set Secure=true")
	}
	if !security.HttpOnly {
		t.Error("ProductionCookieSecurity() MUST set HttpOnly=true")
	}
	if security.SameSite != http.SameSiteStrictMode {
		t.Errorf("ProductionCookieSecurity() SameSite = %v, want SameSiteStrictMode for stricter CSRF protection", security.SameSite)
	}
	if security.Path != "/" {
		t.Errorf("ProductionCookieSecurity() Path = %v, want /", security.Path)
	}
}

func TestDevelopmentCookieSecurity(t *testing.T) {
	security := DevelopmentCookieSecurity()

	if security.Secure {
		t.Error("DevelopmentCookieSecurity() should allow HTTP (Secure=false)")
	}
	if !security.HttpOnly {
		t.Error("DevelopmentCookieSecurity() should still set HttpOnly=true")
	}
	if security.SameSite != http.SameSiteLaxMode {
		t.Errorf("DevelopmentCookieSecurity() SameSite = %v, want SameSiteLaxMode", security.SameSite)
	}
}

func TestEnvIsProduction(t *testing.T) {
	// Save original env values
	oldEnv := os.Getenv("ENV")
	oldGoEnv := os.Getenv("GO_ENV")
	defer func() {
		if oldEnv != "" {
			os.Setenv("ENV", oldEnv)
		} else {
			os.Unsetenv("ENV")
		}
		if oldGoEnv != "" {
			os.Setenv("GO_ENV", oldGoEnv)
		} else {
			os.Unsetenv("GO_ENV")
		}
	}()

	tests := []struct {
		name     string
		env      string
		goEnv    string
		expected bool
	}{
		{
			name:     "production via ENV",
			env:      "production",
			goEnv:    "",
			expected: true,
		},
		{
			name:     "production via GO_ENV",
			env:      "",
			goEnv:    "production",
			expected: true,
		},
		{
			name:     "production via both",
			env:      "production",
			goEnv:    "production",
			expected: true,
		},
		{
			name:     "development via ENV",
			env:      "development",
			goEnv:    "",
			expected: false,
		},
		{
			name:     "staging via ENV",
			env:      "staging",
			goEnv:    "",
			expected: false,
		},
		{
			name:     "no env set",
			env:      "",
			goEnv:    "",
			expected: false,
		},
		{
			name:     "production uppercase",
			env:      "PRODUCTION",
			goEnv:    "",
			expected: true,
		},
		{
			name:     "production mixed case with spaces",
			env:      "  Production  ",
			goEnv:    "",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear and set new values
			os.Unsetenv("ENV")
			os.Unsetenv("GO_ENV")
			if tt.env != "" {
				os.Setenv("ENV", tt.env)
			}
			if tt.goEnv != "" {
				os.Setenv("GO_ENV", tt.goEnv)
			}

			result := EnvIsProduction()
			if result != tt.expected {
				t.Errorf("EnvIsProduction() = %v, want %v (ENV=%q, GO_ENV=%q)", result, tt.expected, tt.env, tt.goEnv)
			}
		})
	}
}

func TestAutoCookieSecurity(t *testing.T) {
	// Save original env values
	oldEnv := os.Getenv("ENV")
	oldGoEnv := os.Getenv("GO_ENV")
	defer func() {
		if oldEnv != "" {
			os.Setenv("ENV", oldEnv)
		} else {
			os.Unsetenv("ENV")
		}
		if oldGoEnv != "" {
			os.Setenv("GO_ENV", oldGoEnv)
		} else {
			os.Unsetenv("GO_ENV")
		}
	}()

	t.Run("production environment returns secure settings", func(t *testing.T) {
		os.Setenv("ENV", "production")
		security := AutoCookieSecurity()

		if !security.Secure {
			t.Error("AutoCookieSecurity() in production MUST set Secure=true")
		}
		if !security.HttpOnly {
			t.Error("AutoCookieSecurity() in production MUST set HttpOnly=true")
		}
		if security.SameSite != http.SameSiteStrictMode {
			t.Errorf("AutoCookieSecurity() in production should use SameSiteStrictMode, got %v", security.SameSite)
		}
	})

	t.Run("development environment returns relaxed settings", func(t *testing.T) {
		os.Unsetenv("ENV")
		os.Unsetenv("GO_ENV")
		security := AutoCookieSecurity()

		if security.Secure {
			t.Error("AutoCookieSecurity() in development should allow HTTP (Secure=false)")
		}
		if !security.HttpOnly {
			t.Error("AutoCookieSecurity() should always set HttpOnly=true")
		}
	})
}

func TestSetSessionCookie(t *testing.T) {
	security := &CookieSecurityConfig{
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	}

	t.Run("sets cookie with all security flags", func(t *testing.T) {
		recorder := &responseRecorder{header: make(http.Header)}
		SetSessionCookie(recorder, SessionCookieName, "test-token", 15*time.Minute, security)

		cookies := recorder.Header()["Set-Cookie"]
		if len(cookies) != 1 {
			t.Fatalf("Expected 1 cookie, got %d", len(cookies))
		}

		cookieHeader := cookies[0]
		// Check for security attributes in the cookie header
		if !contains(cookieHeader, "Secure") {
			t.Error("Cookie should have Secure flag")
		}
		if !contains(cookieHeader, "HttpOnly") {
			t.Error("Cookie should have HttpOnly flag")
		}
		if !contains(cookieHeader, "Path=/") {
			t.Error("Cookie should have Path=/")
		}
		if !contains(cookieHeader, "SameSite=Strict") {
			t.Error("Cookie should have SameSite=Strict attribute")
		}
	})

	t.Run("uses default security when nil", func(t *testing.T) {
		recorder := &responseRecorder{header: make(http.Header)}
		SetSessionCookie(recorder, SessionCookieName, "test-token", 15*time.Minute, nil)

		cookies := recorder.Header()["Set-Cookie"]
		if len(cookies) != 1 {
			t.Fatalf("Expected 1 cookie, got %d", len(cookies))
		}

		cookieHeader := cookies[0]
		if !contains(cookieHeader, "Secure") {
			t.Error("Default cookie should have Secure flag")
		}
		if !contains(cookieHeader, "HttpOnly") {
			t.Error("Default cookie should have HttpOnly flag")
		}
	})
}

func TestClearSessionCookie(t *testing.T) {
	security := &CookieSecurityConfig{
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	}

	t.Run("clears cookie properly", func(t *testing.T) {
		recorder := &responseRecorder{header: make(http.Header)}
		ClearSessionCookie(recorder, SessionCookieName, security)

		cookies := recorder.Header()["Set-Cookie"]
		if len(cookies) != 1 {
			t.Fatalf("Expected 1 cookie, got %d", len(cookies))
		}

		cookieHeader := cookies[0]
		// Check for security attributes even when clearing
		if !contains(cookieHeader, "Secure") {
			t.Error("Cleared cookie should still have Secure flag")
		}
		if !contains(cookieHeader, "HttpOnly") {
			t.Error("Cleared cookie should still have HttpOnly flag")
		}
		// Check for past expiration date
		if !contains(cookieHeader, "1970") {
			t.Errorf("Cleared cookie should have past expiration, got: %s", cookieHeader)
		}
	})
}

// Helper functions for testing

type responseRecorder struct {
	header http.Header
}

func (r *responseRecorder) Header() http.Header {
	return r.header
}

func (r *responseRecorder) Write([]byte) (int, error) {
	return 0, nil
}

func (r *responseRecorder) WriteHeader(int) {}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > len(substr) && (
			s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
