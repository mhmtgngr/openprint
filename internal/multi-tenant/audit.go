// Package multitenant provides multi-tenancy support for OpenPrint services.
// This file contains tenant-scoped audit logging functionality.
package multitenant

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
)

// AuditLevel represents the severity/level of an audit event.
type AuditLevel string

const (
	// AuditLevelInfo is for informational events.
	AuditLevelInfo AuditLevel = "info"
	// AuditLevelWarn is for warning events.
	AuditLevelWarn AuditLevel = "warn"
	// AuditLevelError is for error events.
	AuditLevelError AuditLevel = "error"
	// AuditLevelCritical is for critical events.
	AuditLevelCritical AuditLevel = "critical"
)

// AuditAction represents the type of action being audited.
type AuditAction string

const (
	// ActionCreate represents resource creation.
	ActionCreate AuditAction = "create"
	// ActionRead represents resource access/viewing.
	ActionRead AuditAction = "read"
	// ActionUpdate represents resource modification.
	ActionUpdate AuditAction = "update"
	// ActionDelete represents resource deletion.
	ActionDelete AuditAction = "delete"
	// ActionLogin represents user login.
	ActionLogin AuditAction = "login"
	// ActionLogout represents user logout.
	ActionLogout AuditAction = "logout"
	// ActionExport represents data export.
	ActionExport AuditAction = "export"
	// ActionImport represents data import.
	ActionImport AuditAction = "import"
	// ActionShare represents resource sharing.
	ActionShare AuditAction = "share"
	// ActionPermissionChange represents permission changes.
	ActionPermissionChange AuditAction = "permission_change"
	// ActionQuotaExceeded represents quota exceeded events.
	ActionQuotaExceeded AuditAction = "quota_exceeded"
)

// AuditEvent represents a tenant-scoped audit log entry.
type AuditEvent struct {
	ID           string                 `json:"id"`
	TenantID     string                 `json:"tenant_id"`
	UserID       string                 `json:"user_id"`
	UserName     string                 `json:"user_name,omitempty"`
	UserEmail    string                 `json:"user_email"`
	Action       AuditAction            `json:"action"`
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id,omitempty"`
	ResourceName string                 `json:"resource_name,omitempty"`
	Level        AuditLevel             `json:"level"`
	Message      string                 `json:"message"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	IPAddress    string                 `json:"ip_address,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	SessionID    string                 `json:"session_id,omitempty"`
	RequestID    string                 `json:"request_id,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
}

// AuditRepository defines the interface for audit log persistence.
type AuditRepository interface {
	// Store saves an audit event to persistent storage.
	Store(ctx context.Context, event *AuditEvent) error
	// Query retrieves audit events for a tenant.
	Query(ctx context.Context, tenantID string, filter *AuditFilter) ([]*AuditEvent, error)
	// QueryByID retrieves a specific audit event.
	QueryByID(ctx context.Context, tenantID, eventID string) (*AuditEvent, error)
}

// AuditFilter defines filter criteria for audit log queries.
type AuditFilter struct {
	StartTime    *time.Time
	EndTime      *time.Time
	UserID       string
	Action       AuditAction
	ResourceType string
	Level        AuditLevel
	Limit        int
	Offset       int
}

// Logger handles tenant-scoped audit logging.
type Logger struct {
	repo        AuditRepository
	serviceName string
}

// NewLogger creates a new audit logger.
func NewLogger(repo AuditRepository, serviceName string) *Logger {
	return &Logger{
		repo:        repo,
		serviceName: serviceName,
	}
}

// Log records an audit event.
func (l *Logger) Log(ctx context.Context, event *AuditEvent) error {
	// Enrich event with tenant context from context
	tenantID, err := GetTenantID(ctx)
	if err != nil && event.TenantID == "" {
		// If no tenant in context and none set, this might be a platform event
		// Allow it through with empty tenant ID for platform-level auditing
	} else if event.TenantID == "" {
		event.TenantID = tenantID
	}

	// Generate ID if not set
	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	// Set timestamp if not set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	// Add service name to metadata
	if event.Metadata == nil {
		event.Metadata = make(map[string]interface{})
	}
	event.Metadata["service"] = l.serviceName

	return l.repo.Store(ctx, event)
}

// LogCreate logs a resource creation event.
func (l *Logger) LogCreate(ctx context.Context, userID, userEmail, resourceType, resourceID, resourceName string, metadata map[string]interface{}) error {
	return l.Log(ctx, &AuditEvent{
		UserID:       userID,
		UserEmail:    userEmail,
		Action:       ActionCreate,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ResourceName: resourceName,
		Level:        AuditLevelInfo,
		Message:      fmt.Sprintf("Created %s: %s", resourceType, resourceName),
		Metadata:     metadata,
	})
}

// LogRead logs a resource access event.
func (l *Logger) LogRead(ctx context.Context, userID, userEmail, resourceType, resourceID, resourceName string, metadata map[string]interface{}) error {
	return l.Log(ctx, &AuditEvent{
		UserID:       userID,
		UserEmail:    userEmail,
		Action:       ActionRead,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ResourceName: resourceName,
		Level:        AuditLevelInfo,
		Message:      fmt.Sprintf("Accessed %s: %s", resourceType, resourceName),
		Metadata:     metadata,
	})
}

// LogUpdate logs a resource update event.
func (l *Logger) LogUpdate(ctx context.Context, userID, userEmail, resourceType, resourceID, resourceName string, metadata map[string]interface{}) error {
	return l.Log(ctx, &AuditEvent{
		UserID:       userID,
		UserEmail:    userEmail,
		Action:       ActionUpdate,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ResourceName: resourceName,
		Level:        AuditLevelInfo,
		Message:      fmt.Sprintf("Updated %s: %s", resourceType, resourceName),
		Metadata:     metadata,
	})
}

// LogDelete logs a resource deletion event.
func (l *Logger) LogDelete(ctx context.Context, userID, userEmail, resourceType, resourceID, resourceName string, metadata map[string]interface{}) error {
	return l.Log(ctx, &AuditEvent{
		UserID:       userID,
		UserEmail:    userEmail,
		Action:       ActionDelete,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ResourceName: resourceName,
		Level:        AuditLevelWarn,
		Message:      fmt.Sprintf("Deleted %s: %s", resourceType, resourceName),
		Metadata:     metadata,
	})
}

// LogLogin logs a user login event.
func (l *Logger) LogLogin(ctx context.Context, userID, userEmail, ipAddress, userAgent string) error {
	return l.Log(ctx, &AuditEvent{
		UserID:    userID,
		UserEmail: userEmail,
		Action:    ActionLogin,
		Level:     AuditLevelInfo,
		Message:   "User logged in",
		IPAddress: ipAddress,
		UserAgent: userAgent,
	})
}

// LogLogout logs a user logout event.
func (l *Logger) LogLogout(ctx context.Context, userID, userEmail, ipAddress, userAgent string) error {
	return l.Log(ctx, &AuditEvent{
		UserID:    userID,
		UserEmail: userEmail,
		Action:    ActionLogout,
		Level:     AuditLevelInfo,
		Message:   "User logged out",
		IPAddress: ipAddress,
		UserAgent: userAgent,
	})
}

// LogPermissionChange logs a permission change event.
func (l *Logger) LogPermissionChange(ctx context.Context, userID, userEmail, targetUserID, permission string, granted bool) error {
	action := "granted"
	if !granted {
		action = "revoked"
	}

	return l.Log(ctx, &AuditEvent{
		UserID:       userID,
		UserEmail:    userEmail,
		Action:       ActionPermissionChange,
		ResourceType: "user_permission",
		ResourceID:   targetUserID,
		Level:        AuditLevelWarn,
		Message:      fmt.Sprintf("Permission %s: %s to user %s", action, permission, targetUserID),
		Metadata: map[string]interface{}{
			"permission": permission,
			"granted":    granted,
		},
	})
}

// LogQuotaExceeded logs a quota exceeded event.
func (l *Logger) LogQuotaExceeded(ctx context.Context, resourceType ResourceType, current, maximum int64) error {
	tenantID, _ := GetTenantID(ctx)
	userID := ""
	userEmail := ""

	return l.Log(ctx, &AuditEvent{
		TenantID:     tenantID,
		UserID:       userID,
		UserEmail:    userEmail,
		Action:       ActionQuotaExceeded,
		ResourceType: string(resourceType),
		Level:        AuditLevelWarn,
		Message:      fmt.Sprintf("Quota exceeded for %s: %d/%d", resourceType, current, maximum),
		Metadata: map[string]interface{}{
			"resource_type": resourceType,
			"current":       current,
			"maximum":       maximum,
		},
	})
}

// LogSecurityEvent logs a security-related event.
func (l *Logger) LogSecurityEvent(ctx context.Context, level AuditLevel, message string, metadata map[string]interface{}) error {
	return l.Log(ctx, &AuditEvent{
		Action:       ActionRead,
		ResourceType: "security",
		Level:        level,
		Message:      message,
		Metadata:     metadata,
	})
}

// Query retrieves audit events for the current tenant.
func (l *Logger) Query(ctx context.Context, filter *AuditFilter) ([]*AuditEvent, error) {
	tenantID, err := GetTenantID(ctx)
	if err != nil {
		// Platform admin can query all tenants
		if !IsPlatformAdmin(ctx) {
			return nil, ErrNoTenantContext
		}
		// For platform admin, filter will include tenant_id
	}

	return l.repo.Query(ctx, tenantID, filter)
}

// GetEvent retrieves a specific audit event by ID.
func (l *Logger) GetEvent(ctx context.Context, eventID string) (*AuditEvent, error) {
	tenantID, err := GetTenantID(ctx)
	if err != nil {
		if !IsPlatformAdmin(ctx) {
			return nil, ErrNoTenantContext
		}
	}

	return l.repo.QueryByID(ctx, tenantID, eventID)
}

// HTTPMiddleware creates middleware that adds request context to audit events.
func (l *Logger) HTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Store request context for potential audit logging
			ctx := r.Context()

			// Extract common request info
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = uuid.New().String()
			}

			// Store in context for audit events
			ctx = context.WithValue(ctx, contextKey("request_id"), requestID)
			ctx = context.WithValue(ctx, contextKey("ip_address"), r.RemoteAddr)
			ctx = context.WithValue(ctx, contextKey("user_agent"), r.UserAgent())

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// InMemoryAuditRepository is an in-memory implementation for testing/fallback.
type InMemoryAuditRepository struct {
	events map[string]map[string]*AuditEvent // tenantID -> eventID -> event
	mu     chan struct{}
}

// NewInMemoryAuditRepository creates a new in-memory audit repository.
func NewInMemoryAuditRepository() *InMemoryAuditRepository {
	return &InMemoryAuditRepository{
		events: make(map[string]map[string]*AuditEvent),
		mu:     make(chan struct{}, 1),
	}
}

// Store saves an audit event.
func (r *InMemoryAuditRepository) Store(ctx context.Context, event *AuditEvent) error {
	r.mu <- struct{}{}
	defer func() { <-r.mu }()

	if r.events[event.TenantID] == nil {
		r.events[event.TenantID] = make(map[string]*AuditEvent)
	}
	r.events[event.TenantID][event.ID] = event
	return nil
}

// Query retrieves audit events for a tenant.
func (r *InMemoryAuditRepository) Query(ctx context.Context, tenantID string, filter *AuditFilter) ([]*AuditEvent, error) {
	r.mu <- struct{}{}
	defer func() { <-r.mu }()

	events, ok := r.events[tenantID]
	if !ok {
		return []*AuditEvent{}, nil
	}

	result := make([]*AuditEvent, 0, len(events))
	for _, event := range events {
		if r.matchesFilter(event, filter) {
			result = append(result, event)
		}
	}

	// Apply pagination
	if filter != nil && filter.Limit > 0 {
		start := filter.Offset
		if start >= len(result) {
			return []*AuditEvent{}, nil
		}
		end := start + filter.Limit
		if end > len(result) {
			end = len(result)
		}
		result = result[start:end]
	}

	return result, nil
}

// QueryByID retrieves a specific audit event.
func (r *InMemoryAuditRepository) QueryByID(ctx context.Context, tenantID, eventID string) (*AuditEvent, error) {
	r.mu <- struct{}{}
	defer func() { <-r.mu }()

	events, ok := r.events[tenantID]
	if !ok {
		return nil, apperrors.ErrNotFound
	}

	event, ok := events[eventID]
	if !ok {
		return nil, apperrors.ErrNotFound
	}

	return event, nil
}

// matchesFilter checks if an event matches the filter criteria.
func (r *InMemoryAuditRepository) matchesFilter(event *AuditEvent, filter *AuditFilter) bool {
	if filter == nil {
		return true
	}

	if filter.StartTime != nil && event.Timestamp.Before(*filter.StartTime) {
		return false
	}

	if filter.EndTime != nil && event.Timestamp.After(*filter.EndTime) {
		return false
	}

	if filter.UserID != "" && event.UserID != filter.UserID {
		return false
	}

	if filter.Action != "" && event.Action != filter.Action {
		return false
	}

	if filter.ResourceType != "" && event.ResourceType != filter.ResourceType {
		return false
	}

	if filter.Level != "" && event.Level != filter.Level {
		return false
	}

	return true
}
