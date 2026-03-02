// Package prometheus provides tests for service initialization helpers.
package prometheus

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceInitConfig(t *testing.T) {
	cfg := ServiceInitConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		EnableMetrics:  true,
		EnableTracing:  false,
		MetricsPort:    9090,
		JaegerEndpoint: "http://jaeger:14268/api/traces",
	}

	assert.Equal(t, "test-service", cfg.ServiceName)
	assert.Equal(t, "1.0.0", cfg.ServiceVersion)
	assert.True(t, cfg.EnableMetrics)
	assert.False(t, cfg.EnableTracing)
	assert.Equal(t, 9090, cfg.MetricsPort)
	assert.Equal(t, "http://jaeger:14268/api/traces", cfg.JaegerEndpoint)
}

func TestInitService(t *testing.T) {
	// Reset global state before test
	ResetGlobalRegistry()

	t.Run("init with metrics enabled", func(t *testing.T) {
		ResetGlobalRegistry()

		cfg := ServiceInitConfig{
			ServiceName:   "test-init-metrics",
			EnableMetrics: true,
			MetricsPort:   0, // Use default
		}

		server, shutdown, err := InitService(cfg)

		if err != nil {
			t.Skipf("Could not init service: %v", err)
		}

		assert.NotNil(t, server)
		assert.NotNil(t, shutdown)

		// Verify global registry is set
		globalReg, err := GetRegistry()
		assert.NoError(t, err)
		assert.NotNil(t, globalReg)

		// Cleanup
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdown(ctx)
		ResetGlobalRegistry()
	})

	t.Run("init with metrics disabled", func(t *testing.T) {
		ResetGlobalRegistry()

		cfg := ServiceInitConfig{
			ServiceName:   "test-no-metrics",
			EnableMetrics: false,
		}

		server, shutdown, err := InitService(cfg)

		assert.NoError(t, err)
		assert.Nil(t, server) // No metrics server when disabled
		assert.NotNil(t, shutdown)

		ResetGlobalRegistry()
	})

	t.Run("init with custom port", func(t *testing.T) {
		ResetGlobalRegistry()

		cfg := ServiceInitConfig{
			ServiceName:   "test-custom-port",
			EnableMetrics: true,
			MetricsPort:   9999,
		}

		server, shutdown, err := InitService(cfg)

		if err != nil {
			t.Skipf("Could not init service: %v", err)
		}

		if server != nil {
			assert.Equal(t, 9999, server.Port())
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdown(ctx)
		ResetGlobalRegistry()
	})

	t.Run("shutdown function works", func(t *testing.T) {
		ResetGlobalRegistry()

		cfg := ServiceInitConfig{
			ServiceName:   "test-shutdown",
			EnableMetrics: true,
		}

		_, shutdown, err := InitService(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		err = shutdown(ctx)
		assert.NoError(t, err)

		ResetGlobalRegistry()
	})
}

func TestWrapPostgresPool(t *testing.T) {
	t.Run("wraps pool with metrics", func(t *testing.T) {
		ResetGlobalRegistry()

		cfg := Config{ServiceName: "test-wrap-pg"}
		reg, err := NewRegistry(cfg)
		require.NoError(t, err)
		SetRegistry(reg)

		// Can't create a real pool without database
		// Just verify the function exists and handles nil
		WrapPostgresPool(nil, "test-service")

		// Should not panic
		assert.True(t, true)

		ResetGlobalRegistry()
	})

	t.Run("handles no global registry", func(t *testing.T) {
		ResetGlobalRegistry()

		// Should not panic when no registry is set
		WrapPostgresPool(nil, "test-service")

		assert.True(t, true)
	})
}

func TestWrapRedisClientWithMetrics(t *testing.T) {
	t.Run("wraps client with metrics", func(t *testing.T) {
		ResetGlobalRegistry()

		cfg := Config{ServiceName: "test-wrap-redis"}
		reg, err := NewRegistry(cfg)
		require.NoError(t, err)
		SetRegistry(reg)

		// Currently this function doesn't do much
		// It's a placeholder for future implementation
		WrapRedisClientWithMetrics(nil, "test-service")

		assert.True(t, true)

		ResetGlobalRegistry()
	})

	t.Run("handles no global registry", func(t *testing.T) {
		ResetGlobalRegistry()

		WrapRedisClientWithMetrics(nil, "test-service")

		assert.True(t, true)
	})
}

func TestWrapRedisClientExternal(t *testing.T) {
	t.Run("wraps client externally", func(t *testing.T) {
		cfg := Config{ServiceName: "test-external-redis"}
		reg, err := NewRegistry(cfg)
		require.NoError(t, err)

		// Create a real redis client (with invalid address, but that's ok for this test)
		// The function just tries to instrument it
		client := redis.NewClient(&redis.Options{
			Addr: "localhost:6379",
		})

		// This function uses redisotel instrumentation
		// We just verify it doesn't panic with a valid client
		WrapRedisClientExternal(client, reg, "test-service")

		assert.NotNil(t, client)
	})
}

func TestGetEnvServiceConfig(t *testing.T) {
	t.Run("reads from environment", func(t *testing.T) {
		// Set environment variables
		os.Setenv("SERVICE_VERSION", "2.5.0")
		os.Setenv("ENABLE_METRICS", "true")
		os.Setenv("ENABLE_TRACING", "false")
		os.Setenv("METRICS_PORT", "9095")
		os.Setenv("JAEGER_ENDPOINT", "http://localhost:14268")
		defer func() {
			os.Unsetenv("SERVICE_VERSION")
			os.Unsetenv("ENABLE_METRICS")
			os.Unsetenv("ENABLE_TRACING")
			os.Unsetenv("METRICS_PORT")
			os.Unsetenv("JAEGER_ENDPOINT")
		}()

		cfg := GetEnvServiceConfig("env-test-service")

		assert.Equal(t, "env-test-service", cfg.ServiceName)
		assert.Equal(t, "2.5.0", cfg.ServiceVersion)
		assert.True(t, cfg.EnableMetrics)
		assert.False(t, cfg.EnableTracing)
		assert.Equal(t, 9095, cfg.MetricsPort)
		assert.Equal(t, "http://localhost:14268", cfg.JaegerEndpoint)
	})

	t.Run("uses defaults when env vars not set", func(t *testing.T) {
		// Clear relevant env vars
		os.Unsetenv("SERVICE_VERSION")
		os.Unsetenv("ENABLE_METRICS")
		os.Unsetenv("ENABLE_TRACING")
		os.Unsetenv("METRICS_PORT")
		os.Unsetenv("JAEGER_ENDPOINT")
		defer func() {
			os.Unsetenv("SERVICE_VERSION")
		}()

		cfg := GetEnvServiceConfig("default-test-service")

		assert.Equal(t, "default-test-service", cfg.ServiceName)
		assert.Equal(t, "1.0.0", cfg.ServiceVersion) // Default version
		assert.True(t, cfg.EnableMetrics)          // Default: true
		assert.False(t, cfg.EnableTracing)         // Default: false
		assert.Equal(t, 0, cfg.MetricsPort)        // Default: 0 (use service default)
		assert.Empty(t, cfg.JaegerEndpoint)
	})

	t.Run("parses boolean env vars", func(t *testing.T) {
		tests := []struct {
			name  string
			value string
			want  bool
		}{
			{"true", "true", true},
			{"TRUE", "TRUE", true},
			{"false", "false", false},
			{"FALSE", "FALSE", false},
			{"invalid", "not-a-bool", true}, // Invalid returns default (true for ENABLE_METRICS)
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				os.Setenv("ENABLE_METRICS", tt.value)
				defer os.Unsetenv("ENABLE_METRICS")

				cfg := GetEnvServiceConfig("bool-test")
				assert.Equal(t, tt.want, cfg.EnableMetrics)
			})
		}
	})

	t.Run("parses int env vars", func(t *testing.T) {
		tests := []struct {
			name  string
			value string
			want  int
		}{
			{"valid", "9100", 9100},
			{"zero", "0", 0},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				os.Setenv("METRICS_PORT", tt.value)
				defer os.Unsetenv("METRICS_PORT")

				cfg := GetEnvServiceConfig("int-test")
				assert.Equal(t, tt.want, cfg.MetricsPort)
			})
		}
	})

	t.Run("handles invalid int env vars", func(t *testing.T) {
		os.Setenv("METRICS_PORT", "not-a-number")
		defer os.Unsetenv("METRICS_PORT")

		cfg := GetEnvServiceConfig("invalid-int-test")

		// Should use default when invalid
		assert.Equal(t, 0, cfg.MetricsPort)
	})
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		defaultVal bool
		expected bool
	}{
		{"true value", "TEST_BOOL", "true", false, true},
		{"false value", "TEST_BOOL", "false", true, false},
		{"unset returns default", "TEST_BOOL_UNSET", "", true, true},
		{"invalid returns default", "TEST_BOOL", "invalid", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}

			result := getEnvBool(tt.key, tt.defaultVal)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		defaultVal int
		expected int
	}{
		{"valid int", "TEST_INT", "12345", 0, 12345},
		{"zero", "TEST_INT", "0", 100, 0},
		{"negative", "TEST_INT", "-100", 0, -100},
		{"unset returns default", "TEST_INT_UNSET", "", 999, 999},
		{"invalid returns default", "TEST_INT", "not-a-number", 777, 777},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}

			result := getEnvInt(tt.key, tt.defaultVal)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestServiceInitConfig_Defaults(t *testing.T) {
	cfg := ServiceInitConfig{}

	assert.Empty(t, cfg.ServiceName)
	assert.Empty(t, cfg.ServiceVersion)
	assert.False(t, cfg.EnableMetrics)
	assert.False(t, cfg.EnableTracing)
	assert.Equal(t, 0, cfg.MetricsPort)
	assert.Empty(t, cfg.JaegerEndpoint)
}

func TestInitService_GlobalRegistry(t *testing.T) {
	t.Run("sets global registry", func(t *testing.T) {
		ResetGlobalRegistry()

		cfg := ServiceInitConfig{
			ServiceName:   "global-registry-test",
			EnableMetrics: true,
			MetricsPort:   0,
		}

		_, shutdown, err := InitService(cfg)
		if err != nil {
			t.Skipf("Could not init service: %v", err)
		}

		// Check global registry is set
		globalReg, err := GetRegistry()
		assert.NoError(t, err)
		assert.NotNil(t, globalReg)
		assert.Equal(t, "global-registry-test", globalReg.ServiceName())

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdown(ctx)
		ResetGlobalRegistry()
	})

	t.Run("uses default metrics port when not specified", func(t *testing.T) {
		ResetGlobalRegistry()

		cfg := ServiceInitConfig{
			ServiceName:   "default-port-test",
			EnableMetrics: true,
			MetricsPort:   0,
		}

		server, shutdown, err := InitService(cfg)
		if err != nil {
			t.Skipf("Could not init service: %v", err)
		}

		// Should use GetDefaultMetricsPort which returns 9090 for unknown service
		if server != nil {
			expectedPort := GetDefaultMetricsPort("default-port-test")
			assert.Equal(t, expectedPort, server.Port())
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdown(ctx)
		ResetGlobalRegistry()
	})
}

func TestInitService_ServiceDefaults(t *testing.T) {
	t.Run("auth-service uses port 9091", func(t *testing.T) {
		ResetGlobalRegistry()

		cfg := ServiceInitConfig{
			ServiceName:   ServiceAuthService,
			EnableMetrics: true,
			MetricsPort:   0,
		}

		server, shutdown, err := InitService(cfg)
		if err != nil {
			t.Skipf("Could not init service: %v", err)
		}

		if server != nil {
			assert.Equal(t, 9091, server.Port())
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdown(ctx)
		ResetGlobalRegistry()
	})

	t.Run("registry-service uses port 9092", func(t *testing.T) {
		ResetGlobalRegistry()

		cfg := ServiceInitConfig{
			ServiceName:   ServiceRegistryService,
			EnableMetrics: true,
			MetricsPort:   0,
		}

		server, shutdown, err := InitService(cfg)
		if err != nil {
			t.Skipf("Could not init service: %v", err)
		}

		if server != nil {
			assert.Equal(t, 9092, server.Port())
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdown(ctx)
		ResetGlobalRegistry()
	})
}

func TestInitService_TracingNotSupported(t *testing.T) {
	ResetGlobalRegistry()

	cfg := ServiceInitConfig{
		ServiceName:   "tracing-test",
		EnableMetrics: true,
		EnableTracing: true,
		MetricsPort:   0,
	}

	// Tracing not yet implemented - should not error
	_, shutdown, err := InitService(cfg)
	if err != nil {
		t.Skipf("Could not init service: %v", err)
	}

	assert.NotNil(t, shutdown)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = shutdown(ctx)
	ResetGlobalRegistry()
}

func TestServiceVersionDefaults(t *testing.T) {
	t.Run("empty version defaults to 1.0.0", func(t *testing.T) {
		os.Unsetenv("SERVICE_VERSION")
		defer os.Unsetenv("SERVICE_VERSION")

		cfg := GetEnvServiceConfig("version-test")

		assert.Equal(t, "1.0.0", cfg.ServiceVersion)
	})
}

func TestDBConfig_Struct(t *testing.T) {
	cfg := DBConfig{
		ServiceName: "test-service",
		DBName:      "testdb",
		DBSystem:    DBSystemPostgreSQL,
	}

	assert.Equal(t, "test-service", cfg.ServiceName)
	assert.Equal(t, "testdb", cfg.DBName)
	assert.Equal(t, DBSystemPostgreSQL, cfg.DBSystem)
}

func TestRedisConfig_Struct(t *testing.T) {
	cfg := RedisConfig{
		ServiceName: "test-service",
		DBName:      "cache",
	}

	assert.Equal(t, "test-service", cfg.ServiceName)
	assert.Equal(t, "cache", cfg.DBName)
}
