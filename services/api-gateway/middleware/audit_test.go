package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetClientIP_XForwardedFor(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	req.RemoteAddr = "192.168.1.1:1234"

	ip := getClientIP(req)
	if ip != "10.0.0.1" {
		t.Errorf("getClientIP() = %q, want %q", ip, "10.0.0.1")
	}
}

func TestGetClientIP_XRealIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Real-IP", "10.0.0.2")
	req.RemoteAddr = "192.168.1.1:1234"

	ip := getClientIP(req)
	if ip != "10.0.0.2" {
		t.Errorf("getClientIP() = %q, want %q", ip, "10.0.0.2")
	}
}

func TestGetClientIP_FallbackToRemoteAddr(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"

	ip := getClientIP(req)
	if ip != "192.168.1.1:1234" {
		t.Errorf("getClientIP() = %q, want %q", ip, "192.168.1.1:1234")
	}
}

func TestDetermineActionAndResource(t *testing.T) {
	tests := []struct {
		name             string
		method           string
		path             string
		expectedAction   string
		expectedResource string
	}{
		{"GET request", http.MethodGet, "/api/v1/jobs", "read", "api"},
		{"POST request", http.MethodPost, "/api/v1/jobs", "write", "api"},
		{"PUT request", http.MethodPut, "/api/v1/jobs/123", "write", "api"},
		{"PATCH request", http.MethodPatch, "/api/v1/jobs/123", "write", "api"},
		{"DELETE request", http.MethodDelete, "/api/v1/jobs/123", "delete", "api"},
		{"empty path", http.MethodGet, "/", "read", "unknown"},
		{"simple path", http.MethodGet, "/health", "read", "health"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, resource := determineActionAndResource(tt.method, tt.path)
			if action != tt.expectedAction {
				t.Errorf("action = %q, want %q", action, tt.expectedAction)
			}
			if resource != tt.expectedResource {
				t.Errorf("resource = %q, want %q", resource, tt.expectedResource)
			}
		})
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected []string
	}{
		{"simple path", "/api/v1/jobs", []string{"api", "v1", "jobs"}},
		{"root path", "/", nil},
		{"no leading slash", "api/v1", []string{"api", "v1"}},
		{"trailing slash", "/api/v1/", []string{"api", "v1"}},
		{"double slash", "/api//v1", []string{"api", "v1"}},
		{"single segment", "/health", []string{"health"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitPath(tt.path)
			if len(got) != len(tt.expected) {
				t.Errorf("splitPath(%q) length = %d, want %d (got %v)", tt.path, len(got), len(tt.expected), got)
				return
			}
			for i, v := range got {
				if v != tt.expected[i] {
					t.Errorf("splitPath(%q)[%d] = %q, want %q", tt.path, i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestNullIfEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		isNil    bool
	}{
		{"empty string returns nil", "", true},
		{"non-empty string returns value", "hello", false},
		{"zero int returns nil", 0, true},
		{"non-zero int returns value", 42, false},
		{"other type returns value", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nullIfEmpty(tt.input)
			if tt.isNil && got != nil {
				t.Errorf("nullIfEmpty(%v) = %v, want nil", tt.input, got)
			}
			if !tt.isNil && got == nil {
				t.Errorf("nullIfEmpty(%v) = nil, want non-nil", tt.input)
			}
		})
	}
}

func TestAuditResponseWriter_WriteHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	arw := &auditResponseWriter{
		ResponseWriter: rr,
		status:         http.StatusOK,
	}

	arw.WriteHeader(http.StatusCreated)
	if arw.status != http.StatusCreated {
		t.Errorf("status = %d, want %d", arw.status, http.StatusCreated)
	}
	if !arw.wroteHeader {
		t.Error("wroteHeader should be true")
	}

	// Second call should be ignored
	arw.WriteHeader(http.StatusNotFound)
	if arw.status != http.StatusCreated {
		t.Errorf("status after second WriteHeader = %d, want %d", arw.status, http.StatusCreated)
	}
}

func TestGenerateRequestID(t *testing.T) {
	id1 := generateRequestID()
	id2 := generateRequestID()

	if id1 == "" {
		t.Error("generateRequestID returned empty string")
	}
	if id1 == id2 {
		t.Error("generateRequestID should generate unique IDs")
	}
}
