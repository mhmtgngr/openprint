// Package telemetry provides OpenTelemetry tracing and instrumentation for all services.
package telemetry

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.31.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	// ServiceNameKey is the attribute key for service name.
	ServiceNameKey = attribute.Key("service.name")
	// ServiceVersionKey is the attribute key for service version.
	ServiceVersionKey = attribute.Key("service.version")
	// ServiceInstanceIDKey is the attribute key for service instance ID.
	ServiceInstanceIDKey = attribute.Key("service.instance.id")
)

// Tracer is a global tracer instance.
var Tracer trace.Tracer

// InitTracer initializes the OpenTelemetry tracer with OTLP/Jaeger exporter.
// In production, configure OTEL_EXPORTER_OTLP_ENDPOINT or JAEGER_ENDPOINT.
// For development without a collector, tracing is disabled (noop tracer).
func InitTracer(serviceName, serviceVersion, jaegerEndpoint string) (func(context.Context) error, error) {
	ctx := context.Background()

	// Check for OTLP endpoint environment variable
	otlpEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otlpEndpoint == "" {
		otlpEndpoint = os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
	}

	var exporter tracesdk.SpanExporter
	var err error

	// Try OTLP exporter first (recommended for production)
	if otlpEndpoint != "" {
		exporter, err = otlptracegrpc.New(ctx,
			otlptracegrpc.WithInsecure(),
			otlptracegrpc.WithEndpoint(otlpEndpoint),
		)
		if err != nil {
			// Log error but don't fail - use noop tracer
			fmt.Printf("Warning: failed to create OTLP exporter: %v (using noop tracer)\n", err)
			Tracer = trace.NewNoopTracerProvider().Tracer(serviceName)
			return func(ctx context.Context) error { return nil }, nil
		}
	} else if jaegerEndpoint != "" {
		// Legacy Jaeger endpoint support
		// Jaeger accepts OTLP format directly on port 4317
		exporter, err = otlptracegrpc.New(ctx,
			otlptracegrpc.WithInsecure(),
			otlptracegrpc.WithEndpoint(jaegerEndpoint),
		)
		if err != nil {
			fmt.Printf("Warning: failed to create Jaeger exporter: %v (using noop tracer)\n", err)
			Tracer = trace.NewNoopTracerProvider().Tracer(serviceName)
			return func(ctx context.Context) error { return nil }, nil
		}
	} else {
		// No tracing endpoint configured - use noop tracer for security
		// Stdout exporter is intentionally NOT used in production to prevent
		// sensitive data (tokens, credentials, user data) from being logged
		Tracer = trace.NewNoopTracerProvider().Tracer(serviceName)
		return func(ctx context.Context) error { return nil }, nil
	}

	// Get deployment environment
	env := os.Getenv("OTEL_ENVIRONMENT_NAME")
	if env == "" {
		env = os.Getenv("ENVIRONMENT")
	}
	if env == "" {
		env = "production" // Default to production for security
	}

	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			ServiceVersionKey.String(serviceVersion),
			semconv.DeploymentEnvironmentNameKey.String(env),
		),
		resource.WithFromEnv(), // Also use OTEL_RESOURCE_ATTRIBUTES
	)
	if err != nil {
		exporter.Shutdown(ctx)
		return nil, err
	}

	// Configure sampling based on environment
	var sampler tracesdk.Sampler
	switch env {
	case "production", "prod":
		// Sample 1% in production to reduce costs while maintaining observability
		sampler = tracesdk.TraceIDRatioBased(0.01)
	case "staging":
		// Sample 10% in staging
		sampler = tracesdk.TraceIDRatioBased(0.1)
	default:
		// Always sample in development
		sampler = tracesdk.AlwaysSample()
	}

	tracerProvider := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exporter),
		tracesdk.WithResource(res),
		tracesdk.WithSampler(sampler),
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	Tracer = tracerProvider.Tracer(serviceName)

	return tracerProvider.Shutdown, nil
}

// Middleware returns HTTP middleware for tracing requests.
func Middleware(serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(next, serviceName,
			otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
				return fmtSpanName(r)
			}),
		)
	}
}

// fmtSpanName formats a span name from the HTTP request.
func fmtSpanName(r *http.Request) string {
	// Remove leading slash and split path
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) == 0 {
		return r.Method
	}

	// Use first meaningful path segment
	for i, part := range parts {
		if part != "" && !strings.HasPrefix(part, "_") {
			if i+1 < len(parts) {
				// Check if next part might be an ID
				if looksLikeID(parts[i+1]) {
					return r.Method + " /" + part + "/:id"
				}
			}
			return r.Method + " /" + part
		}
	}
	return r.Method + " " + r.URL.Path
}

// looksLikeID checks if a string looks like a resource ID.
func looksLikeID(s string) bool {
	if s == "" {
		return false
	}
	// UUID-like or numeric ID
	if len(s) == 36 && strings.Count(s, "-") == 4 {
		return true
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

// HTTPMiddleware wraps the standard OpenTelemetry HTTP middleware with
// additional request logging attributes.
func HTTPMiddleware(serviceName string) func(http.Handler) http.Handler {
	mw := Middleware(serviceName)
	return func(next http.Handler) http.Handler {
		return mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create custom response writer to capture status
			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

			// Call the wrapped handler
			mw(next).ServeHTTP(rw, r)

			// Record request duration as a span attribute
			duration := time.Since(start)
			span := trace.SpanFromContext(r.Context())
			span.SetAttributes(
				attribute.Int("http.status_code", rw.status),
				attribute.Int64("http.duration_ms", duration.Milliseconds()),
			)
		}))
	}
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

// Hijack implements http.Hijacker for WebSocket support.
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// WriteHeader captures the status code.
func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.status = code
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

// AddUserID adds the user ID to the current span as an attribute.
func AddUserID(ctx context.Context, userID string) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("user.id", userID))
}

// AddOrgID adds the organization ID to the current span as an attribute.
func AddOrgID(ctx context.Context, orgID string) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("org.id", orgID))
}

// AddPrinterID adds the printer ID to the current span as an attribute.
func AddPrinterID(ctx context.Context, printerID string) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("printer.id", printerID))
}

// AddJobID adds the job ID to the current span as an attribute.
func AddJobID(ctx context.Context, jobID string) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("job.id", jobID))
}

// WithSpan creates a new span for the given operation.
func WithSpan(ctx context.Context, name string, fn func(context.Context) error) error {
	ctx, span := Tracer.Start(
		ctx,
		name,
		trace.WithSpanKind(trace.SpanKindInternal),
	)
	defer span.End()

	if err := fn(ctx); err != nil {
		span.RecordError(err)
		return err
	}
	return nil
}

// ExtractUserID extracts the user ID from span attributes.
func ExtractUserID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	var userID string
	span.SpanContext().IsValid()
	// Note: In practice, you'd retrieve from span attributes
	// This is a placeholder for implementation
	return userID
}
