package jaeger

import (
	"context"
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	// Clear relevant env vars for deterministic test
	os.Unsetenv("SERVICE_VERSION")
	os.Unsetenv("JAEGER_ENDPOINT")
	os.Unsetenv("ENVIRONMENT")
	os.Unsetenv("ENABLE_TRACING")

	cfg := DefaultConfig("test-service")

	if cfg.ServiceName != "test-service" {
		t.Errorf("ServiceName = %q, want %q", cfg.ServiceName, "test-service")
	}
	if cfg.EnableTracing {
		t.Error("EnableTracing should be false by default")
	}
	if cfg.SampleRate != 0 {
		t.Errorf("SampleRate = %f, want 0", cfg.SampleRate)
	}
}

func TestDefaultConfig_WithEnvVars(t *testing.T) {
	os.Setenv("SERVICE_VERSION", "2.0.0")
	os.Setenv("JAEGER_ENDPOINT", "localhost:4317")
	os.Setenv("ENVIRONMENT", "staging")
	os.Setenv("ENABLE_TRACING", "true")
	defer func() {
		os.Unsetenv("SERVICE_VERSION")
		os.Unsetenv("JAEGER_ENDPOINT")
		os.Unsetenv("ENVIRONMENT")
		os.Unsetenv("ENABLE_TRACING")
	}()

	cfg := DefaultConfig("test-service")

	if cfg.ServiceVersion != "2.0.0" {
		t.Errorf("ServiceVersion = %q, want %q", cfg.ServiceVersion, "2.0.0")
	}
	if cfg.Endpoint != "localhost:4317" {
		t.Errorf("Endpoint = %q, want %q", cfg.Endpoint, "localhost:4317")
	}
	if cfg.Environment != "staging" {
		t.Errorf("Environment = %q, want %q", cfg.Environment, "staging")
	}
	if !cfg.EnableTracing {
		t.Error("EnableTracing should be true")
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name     string
		envVal   string
		defVal   bool
		expected bool
	}{
		{"true string", "true", false, true},
		{"1 string", "1", false, true},
		{"TRUE string", "TRUE", false, true},
		{"false string", "false", true, false},
		{"empty uses default true", "", true, true},
		{"empty uses default false", "", false, false},
		{"random string", "random", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("TEST_BOOL_VAR", tt.envVal)
			if tt.envVal == "" {
				os.Unsetenv("TEST_BOOL_VAR")
			}
			defer os.Unsetenv("TEST_BOOL_VAR")

			got := getEnvBool("TEST_BOOL_VAR", tt.defVal)
			if got != tt.expected {
				t.Errorf("getEnvBool(%q, %v) = %v, want %v", tt.envVal, tt.defVal, got, tt.expected)
			}
		})
	}
}

func TestInitTracer_Disabled(t *testing.T) {
	cfg := Config{
		ServiceName:   "test-service",
		EnableTracing: false,
	}

	shutdown, err := InitTracer(cfg)
	if err != nil {
		t.Fatalf("InitTracer error: %v", err)
	}
	if shutdown == nil {
		t.Fatal("shutdown function should not be nil")
	}

	// Shutdown should succeed without error
	if err := shutdown(context.Background()); err != nil {
		t.Errorf("shutdown error: %v", err)
	}
}

func TestInitTracer_NoEndpoint(t *testing.T) {
	os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	os.Unsetenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")

	cfg := Config{
		ServiceName:   "test-service",
		EnableTracing: true,
		Endpoint:      "",
	}

	shutdown, err := InitTracer(cfg)
	if err != nil {
		t.Fatalf("InitTracer error: %v", err)
	}
	if shutdown == nil {
		t.Fatal("shutdown function should not be nil")
	}

	if err := shutdown(context.Background()); err != nil {
		t.Errorf("shutdown error: %v", err)
	}
}

func TestDetermineSampler(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		description string
	}{
		{
			name:        "explicit sample rate",
			config:      Config{SampleRate: 0.5, Environment: "production"},
			description: "should use ratio-based sampler",
		},
		{
			name:        "production environment",
			config:      Config{SampleRate: 0, Environment: "production"},
			description: "should use 1% sampling",
		},
		{
			name:        "prod environment",
			config:      Config{SampleRate: 0, Environment: "prod"},
			description: "should use 1% sampling",
		},
		{
			name:        "staging environment",
			config:      Config{SampleRate: 0, Environment: "staging"},
			description: "should use 10% sampling",
		},
		{
			name:        "development environment",
			config:      Config{SampleRate: 0, Environment: "development"},
			description: "should always sample",
		},
		{
			name:        "dev environment",
			config:      Config{SampleRate: 0, Environment: "dev"},
			description: "should always sample",
		},
		{
			name:        "local environment",
			config:      Config{SampleRate: 0, Environment: "local"},
			description: "should always sample",
		},
		{
			name:        "unknown environment",
			config:      Config{SampleRate: 0, Environment: "custom"},
			description: "should use 10% sampling",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sampler := tt.config.determineSampler()
			if sampler == nil {
				t.Error("determineSampler returned nil")
			}
		})
	}
}

func TestGetTraceID_NoSpan(t *testing.T) {
	ctx := context.Background()
	traceID := GetTraceID(ctx)
	if traceID != "" {
		t.Errorf("GetTraceID with no span = %q, want empty", traceID)
	}
}

func TestGetSpanID_NoSpan(t *testing.T) {
	ctx := context.Background()
	spanID := GetSpanID(ctx)
	if spanID != "" {
		t.Errorf("GetSpanID with no span = %q, want empty", spanID)
	}
}

func TestIsTracingEnabled_NoSpan(t *testing.T) {
	ctx := context.Background()
	if IsTracingEnabled(ctx) {
		t.Error("IsTracingEnabled with no span should be false")
	}
}

func TestConfigFromEnv(t *testing.T) {
	os.Unsetenv("SERVICE_VERSION")
	os.Unsetenv("JAEGER_ENDPOINT")
	os.Unsetenv("ENVIRONMENT")
	os.Unsetenv("ENABLE_TRACING")

	overrides := map[string]string{
		"endpoint":       "custom:4317",
		"environment":    "staging",
		"version":        "3.0.0",
		"sample_rate":    "0.5",
		"enable_tracing": "true",
	}

	cfg := ConfigFromEnv("override-service", overrides)

	if cfg.ServiceName != "override-service" {
		t.Errorf("ServiceName = %q, want %q", cfg.ServiceName, "override-service")
	}
	if cfg.Endpoint != "custom:4317" {
		t.Errorf("Endpoint = %q, want %q", cfg.Endpoint, "custom:4317")
	}
	if cfg.Environment != "staging" {
		t.Errorf("Environment = %q, want %q", cfg.Environment, "staging")
	}
	if cfg.ServiceVersion != "3.0.0" {
		t.Errorf("ServiceVersion = %q, want %q", cfg.ServiceVersion, "3.0.0")
	}
	if cfg.SampleRate != 0.5 {
		t.Errorf("SampleRate = %f, want 0.5", cfg.SampleRate)
	}
	if !cfg.EnableTracing {
		t.Error("EnableTracing should be true")
	}
}

func TestConfigFromEnv_EmptyOverrides(t *testing.T) {
	os.Unsetenv("SERVICE_VERSION")
	os.Unsetenv("JAEGER_ENDPOINT")
	os.Unsetenv("ENVIRONMENT")
	os.Unsetenv("ENABLE_TRACING")

	cfg := ConfigFromEnv("no-override", map[string]string{})

	if cfg.ServiceName != "no-override" {
		t.Errorf("ServiceName = %q, want %q", cfg.ServiceName, "no-override")
	}
	if cfg.Endpoint != "" {
		t.Errorf("Endpoint should be empty, got %q", cfg.Endpoint)
	}
}

func TestMustInitTracer_NoEndpoint(t *testing.T) {
	os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	os.Unsetenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")

	cfg := Config{
		ServiceName:   "test-service",
		EnableTracing: false,
	}

	// Should not panic when tracing is disabled
	shutdown := MustInitTracer(cfg)
	if shutdown == nil {
		t.Fatal("MustInitTracer should return non-nil shutdown function")
	}
}

func TestInitTracerFromEnv(t *testing.T) {
	os.Unsetenv("JAEGER_ENDPOINT")
	os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	os.Unsetenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
	os.Unsetenv("ENABLE_TRACING")

	shutdown, err := InitTracerFromEnv("env-service")
	if err != nil {
		t.Fatalf("InitTracerFromEnv error: %v", err)
	}
	if shutdown == nil {
		t.Fatal("shutdown should not be nil")
	}
}
