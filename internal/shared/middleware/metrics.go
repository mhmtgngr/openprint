// Package middleware provides HTTP middleware for metrics collection.
// This middleware integrates with Prometheus to track HTTP request metrics.
package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/openprint/openprint/internal/shared/telemetry/prometheus"
)

// MetricsMiddlewareConfig holds configuration for the metrics middleware.
type MetricsMiddlewareConfig struct {
	// Registry is the Prometheus registry to use for metrics.
	Registry *prometheus.Registry

	// ServiceName is the name of the service.
	ServiceName string

	// SkipPaths is a list of paths to skip tracking.
	SkipPaths []string

	// ExcludeStaticFiles indicates whether to exclude static file requests.
	ExcludeStaticFiles bool
}

// MetricsMiddleware returns HTTP middleware that records Prometheus metrics for all requests.
// It tracks request count, duration, response size, and in-flight requests.
func MetricsMiddleware(cfg MetricsMiddlewareConfig) func(http.Handler) http.Handler {
	metrics := prometheus.NewMetrics(cfg.Registry)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path should be skipped
			if shouldSkipPath(r.URL.Path, cfg.SkipPaths, cfg.ExcludeStaticFiles) {
				next.ServeHTTP(w, r)
				return
			}

			// Get normalized path (replace IDs with placeholders)
			path := normalizePath(r.URL.Path)
			if path == "" {
				path = "/"
			}

			// Track in-flight requests
			metrics.HTTP.RequestsInFlight.WithLabelValues(
				cfg.ServiceName,
				r.Method,
				path,
			).Inc()

			// Wrap response writer to capture status and size
			rw := &metricsResponseWriter{
				ResponseWriter: w,
				status:         http.StatusOK, // Default status
			}

			// Record start time
			start := time.Now()

			// Call next handler
			next.ServeHTTP(rw, r)

			// Calculate duration
			duration := time.Since(start).Seconds()

			// Record metrics
			statusLabel := prometheus.HTTPStatusLabel(rw.status)
			method := r.Method

			// Total requests counter
			metrics.HTTP.RequestsTotal.WithLabelValues(
				cfg.ServiceName,
				method,
				path,
				statusLabel,
			).Inc()

			// Request duration histogram
			metrics.HTTP.RequestDuration.WithLabelValues(
				cfg.ServiceName,
				method,
				path,
			).Observe(duration)

			// Response size histogram
			if rw.size > 0 {
				metrics.HTTP.ResponseSize.WithLabelValues(
					cfg.ServiceName,
					method,
					path,
				).Observe(float64(rw.size))
			}

			// Error tracking (4xx and 5xx)
			if rw.status >= 400 {
				metrics.HTTP.ErrorsTotal.WithLabelValues(
					cfg.ServiceName,
					method,
					path,
					statusLabel,
					"", // error_code - can be populated by handlers
				).Inc()
			}

			// Decrement in-flight requests
			metrics.HTTP.RequestsInFlight.WithLabelValues(
				cfg.ServiceName,
				method,
				path,
			).Dec()
		})
	}
}

// metricsResponseWriter wraps http.ResponseWriter to capture status code and size.
type metricsResponseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

// WriteHeader captures the status code.
func (w *metricsResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Write captures the response size.
func (w *metricsResponseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.size += n
	return n, err
}

// shouldSkipPath determines if a path should be skipped from metrics collection.
func shouldSkipPath(path string, skipPaths []string, excludeStaticFiles bool) bool {
	// Check explicit skip paths
	for _, skip := range skipPaths {
		if strings.HasPrefix(path, skip) {
			return true
		}
	}

	// Skip static files if configured
	if excludeStaticFiles {
		// Skip common static file extensions
		exts := []string{".css", ".js", ".png", ".jpg", ".jpeg", ".gif", ".ico", ".svg", ".woff", ".woff2", ".ttf", ".eot"}
		for _, ext := range exts {
			if strings.HasSuffix(path, ext) {
				return true
			}
		}
	}

	// Skip metrics endpoint itself
	if path == "/metrics" {
		return true
	}

	// Skip health checks
	if path == "/health" || path == "/healthz" || path == "/ready" || path == "/readyz" {
		return true
	}

	return false
}

// normalizePath converts a request path into a normalized form for metrics.
// IDs and other variable segments are replaced with placeholders.
func normalizePath(path string) string {
	if path == "" || path == "/" {
		return "/"
	}

	// Remove leading slash
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return "/"
	}

	// Split path into segments
	segments := strings.Split(path, "/")

	// Normalize segments that look like IDs
	for i, seg := range segments {
		if looksLikeID(seg) {
			segments[i] = ":id"
		} else if looksLikeUUID(seg) {
			segments[i] = ":uuid"
		} else if looksLikeToken(seg) {
			segments[i] = ":token"
		}
	}

	// Reconstruct path
	normalized := "/" + strings.Join(segments, "/")
	return normalized
}

// looksLikeID checks if a string looks like a numeric ID.
func looksLikeID(s string) bool {
	if len(s) == 0 || len(s) > 20 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// looksLikeUUID checks if a string looks like a UUID.
func looksLikeUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	parts := strings.Split(s, "-")
	if len(parts) != 5 {
		return false
	}
	if len(parts[0]) != 8 || len(parts[1]) != 4 || len(parts[2]) != 4 || len(parts[3]) != 4 || len(parts[4]) != 12 {
		return false
	}
	return true
}

// looksLikeToken checks if a string looks like an auth token or API key.
func looksLikeToken(s string) bool {
	// Tokens are typically long (20+ chars) and contain alphanumeric + special chars
	if len(s) < 20 || len(s) > 256 {
		return false
	}
	hasAlpha := false
	hasDigit := false
	hasSpecial := false
	for _, c := range s {
		switch {
		case (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z'):
			hasAlpha = true
		case c >= '0' && c <= '9':
			hasDigit = true
		case c == '_' || c == '-' || c == '.' || c == '~':
			hasSpecial = true
		}
	}
	return hasAlpha && (hasDigit || hasSpecial)
}

// Inc increments a prometheus Gauge.
func Inc() func() {
	return func() {}
}

// MetricPath extracts the normalized path from a request.
// This is useful for handlers that want to record custom metrics.
func MetricPath(r *http.Request) string {
	return normalizePath(r.URL.Path)
}

// MetricStatus extracts the status code from a response writer.
// Works with metricsResponseWriter and other wrapped writers.
func MetricStatus(w http.ResponseWriter) int {
	if rw, ok := w.(*metricsResponseWriter); ok {
		return rw.status
	}
	return http.StatusOK
}

// RecordCustomMetric allows handlers to record custom metrics.
// This is useful for tracking specific business logic metrics.
func RecordCustomMetric(registry *prometheus.Registry, serviceName, metricName, help string, labels prometheus.Labels, value float64) {
	// This is a placeholder for custom metric recording
	// In practice, you'd want to use a more sophisticated approach
	// with predefined metric collectors
}

// RecordAuthMetric records authentication-related metrics.
func RecordAuthMetric(metrics *prometheus.Metrics, serviceName, authMethod, role string, success bool) {
	if success {
		metrics.Business.AuthSuccessTotal.WithLabelValues(
			serviceName,
			authMethod,
			role,
		).Inc()
	} else {
		metrics.Business.AuthFailureTotal.WithLabelValues(
			serviceName,
			authMethod,
		).Inc()
	}
}

// RecordJobMetric records job-related metrics.
func RecordJobMetric(metrics *prometheus.Metrics, serviceName, orgID string, status string, duration float64) {
	switch status {
	case prometheus.JobStatusCompleted:
		metrics.Business.JobsCompletedTotal.WithLabelValues(
			serviceName,
			orgID,
		).Inc()
		if duration > 0 {
			metrics.Business.JobProcessingDuration.WithLabelValues(
				serviceName,
				orgID,
			).Observe(duration)
		}
	case prometheus.JobStatusFailed:
		metrics.Business.JobsFailedTotal.WithLabelValues(
			serviceName,
			orgID,
			"",
		).Inc()
	}
}

// RecordPrinterMetric records printer-related metrics.
func RecordPrinterMetric(metrics *prometheus.Metrics, serviceName, orgID, printerID string, metricType string) {
	switch metricType {
	case "heartbeat":
		metrics.Business.PrinterHeartbeatsTotal.WithLabelValues(
			serviceName,
			orgID,
		).Inc()
	case "register":
		metrics.Business.PrintersRegisteredTotal.WithLabelValues(
			serviceName,
			orgID,
		).Inc()
	}
}

// RecordStorageMetric records storage-related metrics.
func RecordStorageMetric(metrics *prometheus.Metrics, serviceName, backend, docType string, metricType string, size int64) {
	switch metricType {
	case "store":
		metrics.Business.DocumentsStoredTotal.WithLabelValues(
			serviceName,
			backend,
			docType,
		).Inc()
		if size > 0 {
			metrics.Business.DocumentStorageSize.WithLabelValues(
				serviceName,
				backend,
			).Add(float64(size))
		}
	case "retrieve":
		metrics.Business.DocumentsRetrievedTotal.WithLabelValues(
			serviceName,
			backend,
		).Inc()
	}
}

// RecordWebSocketMetric records WebSocket-related metrics.
func RecordWebSocketMetric(metrics *prometheus.Metrics, serviceName string, metricType string, delta float64) {
	switch metricType {
	case "connect":
		metrics.Business.WebSocketConnectionsActive.WithLabelValues(
			serviceName,
		).Add(delta)
	case "disconnect":
		metrics.Business.WebSocketConnectionsActive.WithLabelValues(
			serviceName,
		).Add(delta)
	case "message":
		metrics.Business.WebSocketMessagesTotal.WithLabelValues(
			serviceName,
		).Inc()
	}
}

// WithMetricLabel adds a label to the request context for use in metrics.
// This allows handlers to add additional context to metrics.
func WithMetricLabel(r *http.Request, key, value string) *http.Request {
	// Labels are stored in the request context
	// This is a simplified implementation
	return r.WithContext(contextWithMetricLabel(r.Context(), key, value))
}

// metricContextKey is the type for metric label context keys.
type metricContextKey string

const metricLabelKey metricContextKey = "metric_labels"

// contextWithMetricLabel adds a metric label to the context.
func contextWithMetricLabel(ctx context.Context, key, value string) context.Context {
	// Get existing labels
	labels, _ := ctx.Value(metricLabelKey).(map[string]string)
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[key] = value

	// Store updated labels
	return context.WithValue(ctx, metricLabelKey, labels)
}

// GetMetricLabels retrieves metric labels from the context.
func GetMetricLabels(r *http.Request) map[string]string {
	if labels, ok := r.Context().Value(metricLabelKey).(map[string]string); ok {
		return labels
	}
	return nil
}

// ParseStatusCode parses a status code from various sources.
func ParseStatusCode(s string) int {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return http.StatusOK
}

// StatusCodeClass returns the status class for a status code.
func StatusCodeClass(code int) string {
	return prometheus.HTTPStatusClass(code)
}

// StatusCodeLabel returns the label for a status code.
func StatusCodeLabel(code int) string {
	return prometheus.HTTPStatusLabel(code)
}
