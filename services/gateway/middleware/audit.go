// Package middleware provides audit logging middleware for the API gateway.
package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

// AuditLogEntry represents a single audit log entry.
type AuditLogEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	Method      string    `json:"method"`
	Path        string    `json:"path"`
	QueryString string    `json:"query_string,omitempty"`
	UserAgent   string    `json:"user_agent,omitempty"`
	ClientIP    string    `json:"client_ip"`
	UserID      string    `json:"user_id,omitempty"`
	Email       string    `json:"email,omitempty"`
	Role        string    `json:"role,omitempty"`
	StatusCode  int       `json:"status_code"`
	DurationMs  int64     `json:"duration_ms"`
	RequestID   string    `json:"request_id,omitempty"`
}

// AuditLogger handles writing audit logs.
type AuditLogger struct {
	logger *log.Logger
}

// NewAuditLogger creates a new audit logger.
func NewAuditLogger(logger *log.Logger) *AuditLogger {
	return &AuditLogger{
		logger: logger,
	}
}

// Log writes an audit log entry.
func (a *AuditLogger) Log(entry AuditLogEntry) {
	// Convert to JSON for structured logging
	data, err := json.Marshal(entry)
	if err != nil {
		a.logger.Printf("AUDIT_ERROR: failed to marshal audit entry: %v", err)
		return
	}
	a.logger.Printf("AUDIT: %s", string(data))
}

// LogWriter writes audit logs to a custom writer.
type LogWriter interface {
	Write(entry AuditLogEntry) error
}

// JSONLogWriter writes audit logs as JSON to an io.Writer.
type JSONLogWriter struct {
	writer io.Writer
}

// NewJSONLogWriter creates a new JSON log writer.
func NewJSONLogWriter(writer io.Writer) *JSONLogWriter {
	return &JSONLogWriter{writer: writer}
}

// Write writes an audit log entry as JSON.
func (j *JSONLogWriter) Write(entry AuditLogEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = j.writer.Write(append(data, '\n'))
	return err
}

// AuditMiddleware creates middleware that logs all requests with audit information.
func AuditMiddleware(auditLogger *AuditLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Set request ID if not present
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = generateRequestID()
			}

			// Store start time and request ID in context
			ctx := r.Context()
			ctx = context.WithValue(ctx, "start_time", start)
			ctx = context.WithValue(ctx, "request_id", requestID)
			r = r.WithContext(ctx)

			// Wrap response writer to capture status code
			rw := &auditResponseWriter{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			// Call next handler
			next.ServeHTTP(rw, r)

			// Calculate duration
			duration := time.Since(start)

			// Create audit log entry
			entry := AuditLogEntry{
				Timestamp:   start,
				Method:      r.Method,
				Path:        r.URL.Path,
				QueryString: r.URL.RawQuery,
				UserAgent:   r.UserAgent(),
				ClientIP:    GetIP(r),
				UserID:      GetUserID(r),
				Email:       GetEmail(r),
				Role:        GetRole(r),
				StatusCode:  rw.status,
				DurationMs:  duration.Milliseconds(),
				RequestID:   requestID,
			}

			auditLogger.Log(entry)
		})
	}
}

// auditResponseWriter wraps http.ResponseWriter to capture status code and response size.
type auditResponseWriter struct {
	http.ResponseWriter
	status      int
	size        int
	wroteHeader bool
}

// WriteHeader captures the status code.
func (rw *auditResponseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.status = code
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

// Write captures the response size.
func (rw *auditResponseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// generateRequestID generates a unique request ID.
func generateRequestID() string {
	// Simple request ID generation using timestamp and random component
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString generates a random string of given length.
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().Nanosecond()%len(letters)]
	}
	return string(b)
}

// DetailedAuditMiddleware creates middleware that logs detailed request information
// including request and response bodies for debugging.
func DetailedAuditMiddleware(auditLogger *AuditLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = generateRequestID()
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, "start_time", start)
			ctx = context.WithValue(ctx, "request_id", requestID)
			r = r.WithContext(ctx)

			// Capture request body if present
			var requestBody []byte
			if r.Body != nil && r.Method != http.MethodGet && r.Method != http.MethodHead {
				requestBody, _ = io.ReadAll(r.Body)
				// Restore body for further reading
				r.Body = io.NopCloser(bytes.NewReader(requestBody))
			}

			rw := &auditResponseWriter{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			next.ServeHTTP(rw, r)

			duration := time.Since(start)

			entry := AuditLogEntry{
				Timestamp:   start,
				Method:      r.Method,
				Path:        r.URL.Path,
				QueryString: r.URL.RawQuery,
				UserAgent:   r.UserAgent(),
				ClientIP:    GetIP(r),
				UserID:      GetUserID(r),
				Email:       GetEmail(r),
				Role:        GetRole(r),
				StatusCode:  rw.status,
				DurationMs:  duration.Milliseconds(),
				RequestID:   requestID,
			}

			auditLogger.Log(entry)

			// Optionally log request body for non-GET requests
			if len(requestBody) > 0 && len(requestBody) < 1024 {
				// Only log small request bodies to avoid flooding logs
				auditLogger.logger.Printf("REQUEST_BODY: %s", string(requestBody))
			}
		})
	}
}

// SecurityAuditMiddleware creates middleware focused on security-relevant events.
// It logs authentication failures, authorization failures, and suspicious activities.
type SecurityAuditMiddleware struct {
	auditLogger *AuditLogger
}

// NewSecurityAuditMiddleware creates a new security audit middleware.
func NewSecurityAuditMiddleware(auditLogger *AuditLogger) *SecurityAuditMiddleware {
	return &SecurityAuditMiddleware{
		auditLogger: auditLogger,
	}
}

// Middleware returns the middleware function.
func (s *SecurityAuditMiddleware) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := &auditResponseWriter{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			next.ServeHTTP(rw, r)

			duration := time.Since(start)

			// Log security-relevant events
			if rw.status == http.StatusUnauthorized || rw.status == http.StatusForbidden {
				entry := AuditLogEntry{
					Timestamp:   start,
					Method:      r.Method,
					Path:        r.URL.Path,
					QueryString: r.URL.RawQuery,
					UserAgent:   r.UserAgent(),
					ClientIP:    GetIP(r),
					UserID:      GetUserID(r),
					Email:       GetEmail(r),
					Role:        GetRole(r),
					StatusCode:  rw.status,
					DurationMs:  duration.Milliseconds(),
				}
				s.auditLogger.Log(entry)
				s.auditLogger.logger.Printf("SECURITY_ALERT: %s %d from %s", r.URL.Path, rw.status, GetIP(r))
			}
		})
	}
}

// AsyncAuditLogger writes audit logs asynchronously to avoid blocking requests.
type AsyncAuditLogger struct {
	entryChan chan AuditLogEntry
	writer    LogWriter
	done      chan struct{}
}

// NewAsyncAuditLogger creates a new async audit logger.
func NewAsyncAuditLogger(writer LogWriter, bufferSize int) *AsyncAuditLogger {
	a := &AsyncAuditLogger{
		entryChan: make(chan AuditLogEntry, bufferSize),
		writer:    writer,
		done:      make(chan struct{}),
	}
	go a.processEntries()
	return a
}

// Log queues an audit log entry for async writing.
func (a *AsyncAuditLogger) Log(entry AuditLogEntry) {
	select {
	case a.entryChan <- entry:
	default:
		// Channel full, drop the entry to avoid blocking
		log.Printf("AUDIT_WARNING: audit log channel full, dropping entry")
	}
}

// Close closes the async audit logger.
func (a *AsyncAuditLogger) Close() {
	close(a.entryChan)
	<-a.done
}

// processEntries processes audit log entries from the channel.
func (a *AsyncAuditLogger) processEntries() {
	defer close(a.done)
	for entry := range a.entryChan {
		if err := a.writer.Write(entry); err != nil {
			log.Printf("AUDIT_ERROR: failed to write audit entry: %v", err)
		}
	}
}

// SlowQueryAuditMiddleware logs requests that take longer than the specified threshold.
func SlowQueryAuditMiddleware(auditLogger *AuditLogger, threshold time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := &auditResponseWriter{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			next.ServeHTTP(rw, r)

			duration := time.Since(start)

			if duration > threshold {
				entry := AuditLogEntry{
					Timestamp:   start,
					Method:      r.Method,
					Path:        r.URL.Path,
					QueryString: r.URL.RawQuery,
					UserAgent:   r.UserAgent(),
					ClientIP:    GetIP(r),
					UserID:      GetUserID(r),
					Email:       GetEmail(r),
					Role:        GetRole(r),
					StatusCode:  rw.status,
					DurationMs:  duration.Milliseconds(),
				}
				auditLogger.Log(entry)
				auditLogger.logger.Printf("SLOW_REQUEST: %s %s took %dms", r.Method, r.URL.Path, duration.Milliseconds())
			}
		})
	}
}
