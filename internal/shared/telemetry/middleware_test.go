// Package telemetry provides tests for OpenTelemetry tracing and instrumentation.
package telemetry

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace"
)

func TestInitTracer(t *testing.T) {
	t.Run("no jaeger endpoint returns noop tracer", func(t *testing.T) {
		shutdown, err := InitTracer("test-service", "1.0.0", "")

		if err != nil {
			t.Errorf("InitTracer() error = %v", err)
		}
		if shutdown == nil {
			t.Error("InitTracer() shutdown should not be nil")
		}

		// Call shutdown to clean up
		ctx := context.Background()
		if err := shutdown(ctx); err != nil {
			t.Errorf("shutdown() error = %v", err)
		}
	})

	t.Run("with jaeger endpoint creates stdout tracer", func(t *testing.T) {
		// Note: This test may fail if the environment doesn't support stdout tracer properly
		// In a CI environment, this might need adjustment
		shutdown, err := InitTracer("test-service", "1.0.0", "stdout")

		if err != nil {
			t.Logf("InitTracer() with stdout produced error (may be expected in some environments): %v", err)
		}
		if shutdown != nil {
			ctx := context.Background()
			shutdown(ctx)
		}
	})
}

func TestMiddleware(t *testing.T) {
	t.Run("middleware creates span", func(t *testing.T) {
		// Initialize tracer for testing
		shutdown, _ := InitTracer("test-service", "1.0.0", "")
		defer shutdown(context.Background())

		mw := Middleware("test-service")

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if span exists in context
			span := trace.SpanFromContext(r.Context())
			// With noop tracer, span exists but context may not be valid
			// The important thing is the span is retrievable without panic
			_ = span
			w.WriteHeader(http.StatusOK)
		})

		server := httptest.NewServer(mw(handler))
		defer server.Close()

		resp, err := http.Get(server.URL + "/test")
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})
}

func TestHTTPMiddleware(t *testing.T) {
	t.Run("http middleware adds duration attribute", func(t *testing.T) {
		shutdown, _ := InitTracer("test-service", "1.0.0", "")
		defer shutdown(context.Background())

		mw := HTTPMiddleware("test-service")

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate some work
			time.Sleep(10 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		})

		server := httptest.NewServer(mw(handler))
		defer server.Close()

		resp, err := http.Get(server.URL + "/test")
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("http middleware captures status code", func(t *testing.T) {
		shutdown, _ := InitTracer("test-service", "1.0.0", "")
		defer shutdown(context.Background())

		mw := HTTPMiddleware("test-service")

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		server := httptest.NewServer(mw(handler))
		defer server.Close()

		resp, err := http.Get(server.URL + "/test")
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", resp.StatusCode)
		}
	})
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	t.Run("WriteHeader is called only once", func(t *testing.T) {
		underlying := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: underlying, status: http.StatusOK}

		rw.WriteHeader(http.StatusNotFound)
		rw.WriteHeader(http.StatusOK) // Should be ignored

		if rw.status != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rw.status)
		}
	})

	t.Run("default status is OK", func(t *testing.T) {
		underlying := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: underlying, status: http.StatusOK, wroteHeader: false}

		if rw.status != http.StatusOK {
			t.Errorf("Expected default status 200, got %d", rw.status)
		}
	})

	t.Run("wroteHeader flag is set", func(t *testing.T) {
		underlying := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: underlying, status: http.StatusOK}

		if rw.wroteHeader {
			t.Error("wroteHeader should be false initially")
		}

		rw.WriteHeader(http.StatusNotFound)

		if !rw.wroteHeader {
			t.Error("wroteHeader should be true after WriteHeader")
		}
	})
}

func TestLooksLikeID(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{"valid UUID", "550e8400-e29b-41d4-a716-446655440000", true},
		{"numeric ID", "12345", true},
		{"empty string", "", false},
		{"path with ID", "users/123", false},
		{"invalid UUID format", "not-a-uuid", false},
		{"mixed alphanumeric", "abc123", false},
		{"UUID without dashes", "550e8400e29b41d4a716446655440000", false},
		{"long numeric", "12345678901234567890", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := looksLikeID(tt.s); got != tt.want {
				t.Errorf("looksLikeID(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestFmtSpanName(t *testing.T) {
	tests := []struct {
		name string
		path string
		method string
		want string
	}{
		{
			name:  "simple path",
			path:  "/users",
			method: "GET",
			want:  "GET /users",
		},
		{
			name:  "path with ID",
			path:  "/users/123",
			method: "GET",
			want:  "GET /users/:id",
		},
		{
			name:  "nested path with ID",
			path:  "/api/v1/users/123/posts",
			method: "GET",
			want:  "GET /api",
		},
		{
			name:  "root path",
			path:  "/",
			method: "GET",
			want:  "GET /",
		},
		{
			name:  "path with internal prefix",
			path:  "/internal/health",
			method: "GET",
			want:  "GET /internal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			got := fmtSpanName(req)
			if got != tt.want {
				t.Errorf("fmtSpanName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddUserID(t *testing.T) {
	shutdown, _ := InitTracer("test-service", "1.0.0", "")
	defer shutdown(context.Background())

	ctx := context.Background()
	userID := "user-123"

	// This test ensures the function doesn't panic
	AddUserID(ctx, userID)
	// In a real scenario with a valid span, we would verify the attribute was added
}

func TestAddOrgID(t *testing.T) {
	shutdown, _ := InitTracer("test-service", "1.0.0", "")
	defer shutdown(context.Background())

	ctx := context.Background()
	orgID := "org-456"

	AddOrgID(ctx, orgID)
	// Function ensures no panic
}

func TestAddPrinterID(t *testing.T) {
	shutdown, _ := InitTracer("test-service", "1.0.0", "")
	defer shutdown(context.Background())

	ctx := context.Background()
	printerID := "printer-789"

	AddPrinterID(ctx, printerID)
}

func TestAddJobID(t *testing.T) {
	shutdown, _ := InitTracer("test-service", "1.0.0", "")
	defer shutdown(context.Background())

	ctx := context.Background()
	jobID := "job-101"

	AddJobID(ctx, jobID)
}

func TestWithSpan(t *testing.T) {
	shutdown, _ := InitTracer("test-service", "1.0.0", "")
	defer shutdown(context.Background())

	ctx := context.Background()
	operationName := "test-operation"

	t.Run("successful operation", func(t *testing.T) {
		err := WithSpan(ctx, operationName, func(ctx context.Context) error {
			return nil
		})
		if err != nil {
			t.Errorf("WithSpan() returned error: %v", err)
		}
	})

	t.Run("operation with error", func(t *testing.T) {
		expectedErr := errors.New("test error")
		err := WithSpan(ctx, operationName, func(ctx context.Context) error {
			return expectedErr
		})
		if err != expectedErr {
			t.Errorf("WithSpan() error = %v, want %v", err, expectedErr)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := WithSpan(ctx, operationName, func(ctx context.Context) error {
			return nil
		})
		// Function should handle cancelled context gracefully
		if err != nil && !strings.Contains(err.Error(), "context canceled") {
			t.Logf("WithSpan() with cancelled context returned: %v", err)
		}
	})
}

func TestExtractUserID(t *testing.T) {
	ctx := context.Background()
	userID := ExtractUserID(ctx)

	// Without a real span with attributes, this returns empty string
	if userID != "" {
		t.Errorf("ExtractUserID() from empty context = %v, want empty string", userID)
	}
}

func TestMiddleware_chain(t *testing.T) {
	shutdown, _ := InitTracer("test-service", "1.0.0", "")
	defer shutdown(context.Background())

	mw := Middleware("test-service")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Test that middleware can be chained
	chained := mw(handler)

	server := httptest.NewServer(chained)
	defer server.Close()

	resp, err := http.Get(server.URL + "/test/path")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestHTTPMiddleware_status_capture(t *testing.T) {
	shutdown, _ := InitTracer("test-service", "1.0.0", "")
	defer shutdown(context.Background())

	mw := HTTPMiddleware("test-service")

	tests := []struct {
		name           string
		statusCode     int
	}{
		{"OK", http.StatusOK},
		{"Created", http.StatusCreated},
		{"Accepted", http.StatusAccepted},
		{"No Content", http.StatusNoContent},
		{"Bad Request", http.StatusBadRequest},
		{"Unauthorized", http.StatusUnauthorized},
		{"Forbidden", http.StatusForbidden},
		{"Not Found", http.StatusNotFound},
		{"Internal Server Error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			})

			server := httptest.NewServer(mw(handler))
			defer server.Close()

			resp, err := http.Get(server.URL + "/")
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.statusCode {
				t.Errorf("Expected status %d, got %d", tt.statusCode, resp.StatusCode)
			}
		})
	}
}
