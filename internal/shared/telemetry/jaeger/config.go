// Package jaeger provides Jaeger distributed tracing configuration and initialization.
// This package integrates with OpenTelemetry to provide tracing capabilities.
package jaeger

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.31.0"
)

// Config holds Jaeger tracing configuration.
type Config struct {
	// ServiceName is the name of the service (e.g., "auth-service").
	ServiceName string

	// ServiceVersion is the version of the service.
	ServiceVersion string

	// Endpoint is the Jaeger or OTLP endpoint.
	// Examples: "localhost:4317", "jaeger:4317", "http://localhost:14268/api/traces"
	Endpoint string

	// Environment is the deployment environment (production, staging, development).
	Environment string

	// SampleRate is the trace sampling rate (0.0 to 1.0).
	// If 0, uses defaults based on environment.
	SampleRate float64

	// EnableTracing enables or disables tracing.
	EnableTracing bool
}

// DefaultConfig returns a default Jaeger configuration.
func DefaultConfig(serviceName string) Config {
	return Config{
		ServiceName:    serviceName,
		ServiceVersion: os.Getenv("SERVICE_VERSION"),
		Endpoint:       os.Getenv("JAEGER_ENDPOINT"),
		Environment:    os.Getenv("ENVIRONMENT"),
		SampleRate:     0, // Use environment-based defaults
		EnableTracing:  getEnvBool("ENABLE_TRACING", false),
	}
}

// getEnvBool reads a boolean environment variable.
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return strings.ToLower(value) == "true" || value == "1"
	}
	return defaultValue
}

// InitTracer initializes the OpenTelemetry tracer with Jaeger/OTLP exporter.
// Returns a shutdown function that should be called during graceful shutdown.
func InitTracer(cfg Config) (func(context.Context) error, error) {
	ctx := context.Background()

	// If tracing is disabled, return a no-op shutdown function
	if !cfg.EnableTracing {
		return func(ctx context.Context) error { return nil }, nil
	}

	// Set default values
	if cfg.ServiceVersion == "" {
		cfg.ServiceVersion = "1.0.0"
	}
	if cfg.Environment == "" {
		cfg.Environment = "production"
	}

	// Build the exporter endpoint
	endpoint := cfg.Endpoint
	if endpoint == "" {
		// Try OTLP endpoint environment variable
		endpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
		if endpoint == "" {
			endpoint = os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
		}
	}

	// If still no endpoint, return no-op tracer
	if endpoint == "" {
		return func(ctx context.Context) error { return nil }, nil
	}

	// Create OTLP exporter (works with Jaeger in OTLP mode)
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(endpoint),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Build resource attributes
	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.ServiceVersion),
			semconv.DeploymentEnvironmentNameKey.String(cfg.Environment),
		),
		resource.WithFromEnv(), // Also use OTEL_RESOURCE_ATTRIBUTES
	)
	if err != nil {
		exporter.Shutdown(ctx)
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Determine sampling rate
	sampler := cfg.determineSampler()

	// Create tracer provider
	tracerProvider := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exporter),
		tracesdk.WithResource(res),
		tracesdk.WithSampler(sampler),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Return shutdown function
	return tracerProvider.Shutdown, nil
}

// determineSampler returns the appropriate sampler based on configuration.
func (c Config) determineSampler() tracesdk.Sampler {
	// Use explicit sample rate if provided
	if c.SampleRate > 0 {
		return tracesdk.TraceIDRatioBased(c.SampleRate)
	}

	// Otherwise, use environment-based defaults
	switch strings.ToLower(c.Environment) {
	case "production", "prod":
		// Sample 1% in production to reduce costs
		return tracesdk.TraceIDRatioBased(0.01)
	case "staging", "stage":
		// Sample 10% in staging
		return tracesdk.TraceIDRatioBased(0.1)
	case "development", "dev", "local":
		// Always sample in development
		return tracesdk.AlwaysSample()
	default:
		// Default to 10% for unknown environments
		return tracesdk.TraceIDRatioBased(0.1)
	}
}

// MustInitTracer initializes the tracer or panics.
func MustInitTracer(cfg Config) func(context.Context) error {
	shutdown, err := InitTracer(cfg)
	if err != nil {
		panic(err)
	}
	return shutdown
}

// InitTracerFromEnv initializes tracing from environment variables.
// This is a convenience function for simple service initialization.
func InitTracerFromEnv(serviceName string) (func(context.Context) error, error) {
	return InitTracer(DefaultConfig(serviceName))
}

// GetTraceID extracts the trace ID from the context.
func GetTraceID(ctx context.Context) string {
	span := otel.SpanFromContext(ctx)
	spanContext := span.SpanContext()
	if spanContext.IsValid() {
		return spanContext.TraceID().String()
	}
	return ""
}

// GetSpanID extracts the span ID from the context.
func GetSpanID(ctx context.Context) string {
	span := otel.SpanFromContext(ctx)
	spanContext := span.SpanContext()
	if spanContext.IsValid() {
		return spanContext.SpanID().String()
	}
	return ""
}

// IsTracingEnabled returns true if tracing is enabled for the given context.
func IsTracingEnabled(ctx context.Context) bool {
	span := otel.SpanFromContext(ctx)
	return span.SpanContext().IsValid()
}

// ConfigFromEnv creates a Config from environment variables with overrides.
func ConfigFromEnv(serviceName string, overrides map[string]string) Config {
	cfg := DefaultConfig(serviceName)

	// Apply overrides
	if v, ok := overrides["endpoint"]; ok && v != "" {
		cfg.Endpoint = v
	}
	if v, ok := overrides["environment"]; ok && v != "" {
		cfg.Environment = v
	}
	if v, ok := overrides["version"]; ok && v != "" {
		cfg.ServiceVersion = v
	}
	if v, ok := overrides["sample_rate"]; ok && v != "" {
		var rate float64
		if _, err := fmt.Sscanf(v, "%f", &rate); err == nil {
			cfg.SampleRate = rate
		}
	}
	if v, ok := overrides["enable_tracing"]; ok && v != "" {
		cfg.EnableTracing = strings.ToLower(v) == "true" || v == "1"
	}

	return cfg
}
