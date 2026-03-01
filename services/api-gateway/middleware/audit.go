// Package middleware provides audit logging middleware for the API gateway.
package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AuditLogger provides audit logging functionality.
type AuditLogger struct {
	db *pgxpool.Pool
}

// NewAuditLogger creates a new audit logger.
func NewAuditLogger(db *pgxpool.Pool) *AuditLogger {
	return &AuditLogger{db: db}
}

// AuditEntry represents an audit log entry.
type AuditEntry struct {
	ID           string
	OrganizationID string
	UserID       string
	APIKeyID     string
	Action       string
	Resource     string
	Method       string
	Path         string
	StatusCode   int
	IPAddress    string
	UserAgent    string
	RequestID    string
	LatencyMs    int
	CreatedAt    time.Time
}

// AuditMiddleware creates middleware that logs all API requests for audit purposes.
func AuditMiddleware(db *pgxpool.Pool) func(http.Handler) http.Handler {
	auditor := NewAuditLogger(db)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Generate request ID
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = generateRequestID()
			}

			// Wrap response writer to capture status code
			rw := &auditResponseWriter{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			// Read and restore body for logging
			var bodyBytes []byte
			if r.Body != nil && r.Method != http.MethodGet && r.Method != http.MethodHead {
				bodyBytes, _ = io.ReadAll(r.Body)
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}

			// Call next handler
			next.ServeHTTP(rw, r)

			// Log audit entry asynchronously
			latency := time.Since(start)
			go auditor.logEntry(context.Background(), &AuditEntry{
				RequestID:  requestID,
				Method:     r.Method,
				Path:      r.URL.Path,
				StatusCode: rw.status,
				IPAddress:  getClientIP(r),
				UserAgent:  r.UserAgent(),
				LatencyMs:  int(latency.Milliseconds()),
				CreatedAt:  time.Now(),
				// UserID, OrgID, APIKeyID will be extracted from context
			}, r, bodyBytes)
		})
	}
}

// logEntry stores an audit log entry in the database.
func (a *AuditLogger) logEntry(ctx context.Context, entry *AuditEntry, r *http.Request, body []byte) {
	// Extract values from context if available
	if userID := ctx.Value("user_id"); userID != nil {
		entry.UserID = fmt.Sprintf("%v", userID)
	}
	if orgID := ctx.Value("org_id"); orgID != nil {
		entry.OrganizationID = fmt.Sprintf("%v", orgID)
	}
	if apiKeyID := ctx.Value("api_key_id"); apiKeyID != nil {
		entry.APIKeyID = fmt.Sprintf("%v", apiKeyID)
	}

	// Create table if not exists
	initQuery := `
		CREATE TABLE IF NOT EXISTS audit_logs (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			organization_id UUID,
			user_id UUID,
			api_key_id UUID,
			action VARCHAR(100),
			resource VARCHAR(500),
			method VARCHAR(10),
			path TEXT,
			status_code INTEGER,
			ip_address INET,
			user_agent TEXT,
			request_id VARCHAR(100) UNIQUE,
			latency_ms INTEGER,
			request_body JSONB,
			response_size INTEGER,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_audit_logs_org ON audit_logs(organization_id);
		CREATE INDEX IF NOT EXISTS idx_audit_logs_user ON audit_logs(user_id);
		CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at);
	`
	a.db.Exec(ctx, initQuery)

	// Determine action and resource from path
	entry.Action, entry.Resource = determineActionAndResource(entry.Method, entry.Path)

	// Truncate body if too large
	var bodyJSON interface{}
	if len(body) > 0 && len(body) < 10000 {
		json.Unmarshal(body, &bodyJSON)
	}

	query := `
		INSERT INTO audit_logs (
			organization_id, user_id, api_key_id, action, resource,
			method, path, status_code, ip_address, user_agent, request_id,
			latency_ms, request_body, created_at
		) VALUES (
			$1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8, $9::inet, $10, $11, $12, $13, NOW()
		)
		ON CONFLICT (request_id) DO UPDATE SET
			status_code = EXCLUDED.status_code,
			latency_ms = EXCLUDED.latency_ms
	`

	_, err := a.db.Exec(ctx, query,
		nullIfEmpty(entry.OrganizationID), nullIfEmpty(entry.UserID), nullIfEmpty(entry.APIKeyID),
		entry.Action, entry.Resource, entry.Method, entry.Path, entry.StatusCode,
		entry.IPAddress, entry.UserAgent, entry.RequestID, entry.LatencyMs,
		bodyJSON,
	)

	if err != nil {
		log.Printf("Failed to log audit entry: %v", err)
	}
}

// AuditQuery provides methods to query audit logs.
type AuditQuery struct {
	db *pgxpool.Pool
}

// NewAuditQuery creates a new audit query instance.
func NewAuditQuery(db *pgxpool.Pool) *AuditQuery {
	return &AuditQuery{db: db}
}

// GetAuditLogs retrieves audit logs with optional filters.
func (q *AuditQuery) GetAuditLogs(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*AuditEntry, int, error) {
	baseQuery := `
		SELECT id, organization_id, user_id, api_key_id, action, resource,
		       method, path, status_code, ip_address, user_agent, request_id,
		       latency_ms, created_at
		FROM audit_logs
		WHERE 1=1
	`

	countQuery := `SELECT COUNT(*) FROM audit_logs WHERE 1=1`

	args := []interface{}{}
	argNum := 1

	// Apply filters
	if orgID, ok := filters["organization_id"]; ok && orgID != "" {
		baseQuery += fmt.Sprintf(" AND organization_id = $%d", argNum)
		countQuery += fmt.Sprintf(" AND organization_id = $%d", argNum)
		args = append(args, orgID)
		argNum++
	}
	if userID, ok := filters["user_id"]; ok && userID != "" {
		baseQuery += fmt.Sprintf(" AND user_id = $%d", argNum)
		countQuery += fmt.Sprintf(" AND user_id = $%d", argNum)
		args = append(args, userID)
		argNum++
	}
	if action, ok := filters["action"]; ok && action != "" {
		baseQuery += fmt.Sprintf(" AND action = $%d", argNum)
		countQuery += fmt.Sprintf(" AND action = $%d", argNum)
		args = append(args, action)
		argNum++
	}
	if startDate, ok := filters["start_date"]; ok && startDate != "" {
		baseQuery += fmt.Sprintf(" AND created_at >= $%d::date", argNum)
		countQuery += fmt.Sprintf(" AND created_at >= $%d::date", argNum)
		args = append(args, startDate)
		argNum++
	}
	if endDate, ok := filters["end_date"]; ok && endDate != "" {
		baseQuery += fmt.Sprintf(" AND created_at <= $%d::date", argNum)
		countQuery += fmt.Sprintf(" AND created_at <= $%d::date", argNum)
		args = append(args, endDate)
		argNum++
	}

	// Get total count
	var total int
	if err := q.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Add ordering and pagination
	baseQuery += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := q.db.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var entries []*AuditEntry
	for rows.Next() {
		var entry AuditEntry
		if err := rows.Scan(
			&entry.ID, &entry.OrganizationID, &entry.UserID, &entry.APIKeyID,
			&entry.Action, &entry.Resource, &entry.Method, &entry.Path,
			&entry.StatusCode, &entry.IPAddress, &entry.UserAgent,
			&entry.RequestID, &entry.LatencyMs, &entry.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		entries = append(entries, &entry)
	}

	return entries, total, nil
}

// auditResponseWriter wraps http.ResponseWriter to capture status code.
type auditResponseWriter struct {
	http.ResponseWriter
	status      int
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

// Helper functions

func generateRequestID() string {
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), randomString(8))
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}

func determineActionAndResource(method, path string) (string, string) {
	// Determine action from HTTP method
	action := method
	if method == http.MethodGet {
		action = "read"
	} else if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
		action = "write"
	} else if method == http.MethodDelete {
		action = "delete"
	}

	// Extract resource from path
	parts := splitPath(path)
	resource := "unknown"
	if len(parts) > 0 {
		resource = parts[0]
	}

	return action, resource
}

func splitPath(path string) []string {
	parts := make([]string, 0)
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func nullIfEmpty(s interface{}) interface{} {
	switch v := s.(type) {
	case string:
		if v == "" {
			return nil
		}
	case int:
		if v == 0 {
			return nil
		}
	}
	return s
}
