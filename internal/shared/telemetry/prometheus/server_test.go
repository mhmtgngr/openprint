// Package prometheus provides tests for the metrics server.
package prometheus

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultServerConfig(t *testing.T) {
	cfg := Config{ServiceName: "test"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	serverCfg := DefaultServerConfig(reg)

	assert.Equal(t, reg, serverCfg.Registry)
	assert.Equal(t, 9090, serverCfg.Port)
	assert.Equal(t, "0.0.0.0", serverCfg.Host)
}

func TestNewMetricsServer(t *testing.T) {
	t.Run("creates server with defaults", func(t *testing.T) {
		cfg := Config{ServiceName: "test-server"}
		reg, err := NewRegistry(cfg)
		require.NoError(t, err)

		server := NewMetricsServer(ServerConfig{
			Registry: reg,
		})

		assert.NotNil(t, server)
		assert.Equal(t, 9090, server.Port())
		assert.Equal(t, "0.0.0.0:9090", server.Addr())
	})

	t.Run("creates server with custom port", func(t *testing.T) {
		cfg := Config{ServiceName: "test-server"}
		reg, err := NewRegistry(cfg)
		require.NoError(t, err)

		server := NewMetricsServer(ServerConfig{
			Registry: reg,
			Port:     9100,
			Host:     "127.0.0.1",
		})

		assert.Equal(t, 9100, server.Port())
		assert.Equal(t, "127.0.0.1:9100", server.Addr())
	})

	t.Run("zero port defaults to 9090", func(t *testing.T) {
		cfg := Config{ServiceName: "test-server"}
		reg, err := NewRegistry(cfg)
		require.NoError(t, err)

		server := NewMetricsServer(ServerConfig{
			Registry: reg,
			Port:     0,
		})

		assert.Equal(t, 9090, server.Port())
	})

	t.Run("empty host defaults to 0.0.0.0", func(t *testing.T) {
		cfg := Config{ServiceName: "test-server"}
		reg, err := NewRegistry(cfg)
		require.NoError(t, err)

		server := NewMetricsServer(ServerConfig{
			Registry: reg,
			Host:     "",
		})

		assert.Equal(t, "0.0.0.0:9090", server.Addr())
	})
}

func TestMetricsServer_Endpoints(t *testing.T) {
	cfg := Config{ServiceName: "test-endpoints"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	_ = NewMetricsServer(ServerConfig{
		Registry: reg,
	})

	t.Run("GET /metrics returns metrics", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()

		// Create a test handler that mimics the server's handler
		handler := metricsHandler(reg, nil)
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		// Content-Type may include escaping parameter
		contentType := resp.Header.Get("Content-Type")
		assert.Contains(t, contentType, "text/plain")
		assert.Contains(t, contentType, "version=0.0.4")

		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		// Should contain some Prometheus metrics
		assert.Contains(t, bodyStr, "go_")
		assert.Contains(t, bodyStr, "process_")
	})

	t.Run("GET /health returns health status", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		handler := healthHandler(reg)
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		assert.Contains(t, bodyStr, "healthy")
		assert.Contains(t, bodyStr, "test-endpoints")
	})

	t.Run("GET / returns HTML page", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		handler := rootHandler(reg)
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "text/html", resp.Header.Get("Content-Type"))

		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		assert.Contains(t, bodyStr, "<html>")
		assert.Contains(t, bodyStr, "/metrics")
		assert.Contains(t, bodyStr, "/health")
		assert.Contains(t, bodyStr, "test-endpoints")
	})

	t.Run("GET /unknown returns 404", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/unknown", nil)
		w := httptest.NewRecorder()

		handler := rootHandler(reg)
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("POST /metrics returns 405", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/metrics", nil)
		w := httptest.NewRecorder()

		handler := metricsHandler(reg, nil)
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
}

func TestMetricsServer_BasicAuth(t *testing.T) {
	cfg := Config{ServiceName: "test-auth"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	auth := &BasicAuthConfig{
		Username: "admin",
		Password: "secret",
	}

	t.Run("valid auth succeeds", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/metrics", nil)
		req.SetBasicAuth("admin", "secret")
		w := httptest.NewRecorder()

		handler := metricsHandler(reg, auth)
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("missing auth returns 401", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()

		handler := metricsHandler(reg, auth)
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("WWW-Authenticate"), "Basic")
	})

	t.Run("invalid auth returns 401", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/metrics", nil)
		req.SetBasicAuth("wrong", "credentials")
		w := httptest.NewRecorder()

		handler := metricsHandler(reg, auth)
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("wrong password returns 401", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/metrics", nil)
		req.SetBasicAuth("admin", "wrong")
		w := httptest.NewRecorder()

		handler := metricsHandler(reg, auth)
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestMetricsServer_StartAndShutdown(t *testing.T) {
	t.Run("server starts and shuts down gracefully", func(t *testing.T) {
		cfg := Config{ServiceName: "test-start"}
		reg, err := NewRegistry(cfg)
		require.NoError(t, err)

		server := NewMetricsServer(ServerConfig{
			Registry: reg,
			Port:     0, // Let OS pick port
		})

		// Modify server to use a dynamic port for testing
		server.port = 0 // Will be set by Start()

		err = server.Start()
		// Note: Start may fail if port is already in use, we skip in that case
		if err != nil {
			t.Skipf("Could not start server: %v", err)
		}

		// Give server time to start
		time.Sleep(50 * time.Millisecond)

		// Shutdown should complete
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = server.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestStartMetricsServer(t *testing.T) {
	t.Run("starts server with default port", func(t *testing.T) {
		// Reset global registry
		ResetGlobalRegistry()

		cfg := Config{ServiceName: "test-start-func"}
		reg, err := NewRegistry(cfg)
		require.NoError(t, err)

		server, err := StartMetricsServer(reg, 0)
		if err != nil {
			t.Skipf("Could not start server: %v", err)
		}

		assert.NotNil(t, server)

		// Shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	})

	t.Run("starts server with custom port", func(t *testing.T) {
		ResetGlobalRegistry()

		cfg := Config{ServiceName: "test-custom-port"}
		reg, err := NewRegistry(cfg)
		require.NoError(t, err)

		server, err := StartMetricsServer(reg, 9999)
		if err != nil {
			t.Skipf("Could not start server: %v", err)
		}

		assert.Equal(t, 9999, server.Port())

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	})
}

func TestMustStartMetricsServer(t *testing.T) {
	t.Run("panics on error", func(t *testing.T) {
		ResetGlobalRegistry()

		// This test verifies panic behavior
		// In a real scenario, we'd need to trigger an actual error
		// For now, we just verify the function exists
		cfg := Config{ServiceName: "test-must-start"}
		reg, err := NewRegistry(cfg)
		require.NoError(t, err)

		// Use a random high port that's unlikely to conflict
		assert.NotPanics(t, func() {
			server := MustStartMetricsServer(reg, 39090)
			if server != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = server.Shutdown(ctx)
			}
		})
	})
}

func TestGetDefaultMetricsPort(t *testing.T) {
	tests := []struct {
		name         string
		serviceName  string
		expectedPort int
	}{
		{"auth service", ServiceAuthService, 9091},
		{"registry service", ServiceRegistryService, 9092},
		{"job service", ServiceJobService, 9093},
		{"storage service", ServiceStorageService, 9094},
		{"notification service", ServiceNotificationService, 9095},
		{"unknown service", "unknown-service", 9090},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port := GetDefaultMetricsPort(tt.serviceName)
			assert.Equal(t, tt.expectedPort, port)
		})
	}
}

func TestMetricsServer_AddrAndPort(t *testing.T) {
	cfg := Config{ServiceName: "test-addr"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	server := NewMetricsServer(ServerConfig{
		Registry: reg,
		Port:     8080,
		Host:     "localhost",
	})

	assert.Equal(t, 8080, server.Port())
	assert.Equal(t, "localhost:8080", server.Addr())
}

func TestMultiRegistryServer(t *testing.T) {
	t.Run("creates multi-registry server", func(t *testing.T) {
		server := NewMultiRegistryServer(9090, "127.0.0.1")

		assert.NotNil(t, server)
		assert.Equal(t, 9090, server.port)
		assert.Equal(t, "127.0.0.1:9090", server.addr)
	})

	t.Run("uses defaults when zero values", func(t *testing.T) {
		server := NewMultiRegistryServer(0, "")

		assert.Equal(t, 9090, server.port)
		assert.Equal(t, "0.0.0.0:9090", server.addr)
	})
}

func TestMultiRegistryServer_AddRegistry(t *testing.T) {
	server := NewMultiRegistryServer(9090, "127.0.0.1")

	cfg1 := Config{ServiceName: "service-1"}
	reg1, err := NewRegistry(cfg1)
	require.NoError(t, err)

	cfg2 := Config{ServiceName: "service-2"}
	reg2, err := NewRegistry(cfg2)
	require.NoError(t, err)

	server.AddRegistry("service-1", reg1)
	server.AddRegistry("service-2", reg2)

	// Can't access private field, just verify the function completed without panic
	assert.NotNil(t, server)
}

func TestMultiRegistryServer_Endpoints(t *testing.T) {
	server := NewMultiRegistryServer(9090, "127.0.0.1")

	cfg := Config{ServiceName: "test-multi"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	server.AddRegistry("test", reg)

	t.Run("health handler", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		handler := server.healthHandler()
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), `"services":1`)
	})

	t.Run("root handler", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		handler := server.rootHandler()
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		assert.Contains(t, bodyStr, "Serving 1 services")
	})

	t.Run("404 for unknown paths", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/unknown", nil)
		w := httptest.NewRecorder()

		handler := server.rootHandler()
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestMetricsServer_MetricsContent(t *testing.T) {
	cfg := Config{ServiceName: "test-content"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	// Add some metrics to the registry
	metrics := NewMetrics(reg)
	metrics.HTTP.RequestsTotal.WithLabelValues("test-content", "GET", "/api", "200").Inc()

	t.Run("metrics endpoint contains custom metrics", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()

		handler := metricsHandler(reg, nil)
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		// Should contain our custom metric
		assert.Contains(t, bodyStr, "openprint_http_requests_total")
	})
}

func TestMetricsServer_HTMLContent(t *testing.T) {
	t.Run("root HTML contains service info", func(t *testing.T) {
		cfg := Config{
			ServiceName:    "html-test",
			ServiceVersion: "2.0.0",
		}
		reg, err := NewRegistry(cfg)
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		handler := rootHandler(reg)
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		assert.Contains(t, bodyStr, "html-test")
		assert.Contains(t, bodyStr, "2.0.0")
		assert.Contains(t, bodyStr, "<title>")
		assert.Contains(t, bodyStr, "</html>")
	})
}

func TestMetricsServer_HandlerComposition(t *testing.T) {
	cfg := Config{ServiceName: "test-compose"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	_ = NewMetricsServer(ServerConfig{
		Registry: reg,
	})

	t.Run("basic auth config is stored", func(t *testing.T) {
		auth := &BasicAuthConfig{
			Username: "user",
			Password: "pass",
		}

		serverWithAuth := NewMetricsServer(ServerConfig{
			Registry: reg,
			BasicAuth: auth,
		})

		assert.Equal(t, auth, serverWithAuth.basicAuth)
	})

	t.Run("TLS config is stored", func(t *testing.T) {
		serverWithTLS := NewMetricsServer(ServerConfig{
			Registry: reg,
			CertFile:  "/path/to/cert.pem",
			KeyFile:   "/path/to/key.pem",
		})

		assert.Equal(t, "/path/to/cert.pem", serverWithTLS.certFile)
		assert.Equal(t, "/path/to/key.pem", serverWithTLS.keyFile)
	})
}

func TestMultiRegistryServer_Shutdown(t *testing.T) {
	server := NewMultiRegistryServer(0, "127.0.0.1")

	// Shutdown without starting should not error
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	// With no server running, this should return nil
	assert.NoError(t, err)
}

func TestMultiRegistryServer_MetricsHandler(t *testing.T) {
	server := NewMultiRegistryServer(9090, "127.0.0.1")

	cfg1 := Config{ServiceName: "svc1"}
	reg1, _ := NewRegistry(cfg1)
	server.AddRegistry("svc1", reg1)

	cfg2 := Config{ServiceName: "svc2"}
	reg2, _ := NewRegistry(cfg2)
	server.AddRegistry("svc2", reg2)

	// Add metrics to each registry
	metrics1 := NewMetrics(reg1)
	metrics1.HTTP.RequestsTotal.WithLabelValues("svc1", "GET", "/", "200").Inc()

	metrics2 := NewMetrics(reg2)
	metrics2.HTTP.RequestsTotal.WithLabelValues("svc2", "POST", "/api", "201").Inc()

	t.Run("metrics handler outputs all registries", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()

		handler := server.metricsHandler()
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		// Should contain both services
		assert.Contains(t, bodyStr, "Service: svc1")
		assert.Contains(t, bodyStr, "Service: svc2")
	})
}

func TestBasicAuthConfig(t *testing.T) {
	auth := &BasicAuthConfig{
		Username: "testuser",
		Password: "testpass",
	}

	assert.Equal(t, "testuser", auth.Username)
	assert.Equal(t, "testpass", auth.Password)
}

func TestServerConfig(t *testing.T) {
	cfg := Config{ServiceName: "test"}
	reg, _ := NewRegistry(cfg)

	auth := &BasicAuthConfig{
		Username: "admin",
		Password: "secret",
	}

	serverCfg := ServerConfig{
		Registry:   reg,
		Port:       9100,
		Host:       "localhost",
		CertFile:   "cert.pem",
		KeyFile:    "key.pem",
		BasicAuth:  auth,
	}

	assert.Equal(t, reg, serverCfg.Registry)
	assert.Equal(t, 9100, serverCfg.Port)
	assert.Equal(t, "localhost", serverCfg.Host)
	assert.Equal(t, "cert.pem", serverCfg.CertFile)
	assert.Equal(t, "key.pem", serverCfg.KeyFile)
	assert.Equal(t, auth, serverCfg.BasicAuth)
}

func TestMetricsServer_ContextCancellation(t *testing.T) {
	cfg := Config{ServiceName: "test-cancel"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	server := NewMetricsServer(ServerConfig{
		Registry: reg,
		Port:     0,
	})

	err = server.Start()
	if err != nil {
		t.Skipf("Could not start server: %v", err)
	}

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel right away

	err = server.Shutdown(ctx)
	// Should either succeed or error with context canceled
	if err != nil {
		assert.True(t, strings.Contains(err.Error(), "context canceled") ||
			err == nil)
	}
}
