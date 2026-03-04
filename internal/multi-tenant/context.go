// Package multitenant provides multi-tenancy support for OpenPrint services.
// It handles tenant context propagation, resource isolation, and quota enforcement.
package multitenant

import (
	"context"
	"errors"

	sharedcontext "github.com/openprint/openprint/internal/shared/context"
)

var (
	// ErrNoTenantContext is returned when tenant context is not available.
	ErrNoTenantContext = sharedcontext.ErrNoTenantContext
	// ErrUnauthorizedTenant is returned when user tries to access a different tenant.
	ErrUnauthorizedTenant = errors.New("unauthorized: tenant mismatch")
)

// ContextKey type alias for compatibility with existing code.
// All new code should use sharedcontext package directly.
type contextKey = sharedcontext.ContextKey

// Context key constants - delegated to shared context package.
const (
	TenantIDKey        = sharedcontext.TenantIDKey
	TenantNameKey      = sharedcontext.TenantNameKey
	TenantRoleKey      = sharedcontext.TenantRoleKey
	IsPlatformAdminKey = sharedcontext.IsPlatformAdminKey
	QuotaKey           = sharedcontext.QuotaKey
)

// Role represents user roles within the system.
type Role string

const (
	// RolePlatformAdmin is the platform administrator with full system access.
	RolePlatformAdmin Role = "platform_admin"
	// RoleOrgAdmin is the organization administrator with full org access.
	RoleOrgAdmin Role = "org_admin"
	// RoleOrgUser is a regular organization user.
	RoleOrgUser Role = "org_user"
	// RoleOrgViewer is a read-only organization user.
	RoleOrgViewer Role = "org_viewer"
)

// TenantContext holds tenant-specific context for a request.
type TenantContext struct {
	TenantID        string
	TenantName      string
	UserRole        Role
	IsPlatformAdmin bool
	Quota           *QuotaInfo
}

// QuotaInfo holds quota information for the current tenant.
type QuotaInfo struct {
	MaxPrinters      int32
	MaxStorageGB     int32
	MaxJobsPerMonth  int32
	MaxUsers         int32
	CurrentPrinters  int32
	CurrentStorageGB int64
	CurrentJobs      int32
	CurrentUsers     int32
}

// UsagePercentage returns the usage percentage for the given resource type.
func (q *QuotaInfo) UsagePercentage(resourceType string) float64 {
	switch resourceType {
	case "printers":
		if q.MaxPrinters == 0 {
			return 0
		}
		return float64(q.CurrentPrinters) / float64(q.MaxPrinters) * 100
	case "storage":
		if q.MaxStorageGB == 0 {
			return 0
		}
		return float64(q.CurrentStorageGB) / float64(int64(q.MaxStorageGB)*1024*1024*1024) * 100
	case "jobs":
		if q.MaxJobsPerMonth == 0 {
			return 0
		}
		return float64(q.CurrentJobs) / float64(q.MaxJobsPerMonth) * 100
	case "users":
		if q.MaxUsers == 0 {
			return 0
		}
		return float64(q.CurrentUsers) / float64(q.MaxUsers) * 100
	default:
		return 0
	}
}

// IsNearLimit returns true if usage is above the threshold percentage.
func (q *QuotaInfo) IsNearLimit(resourceType string, threshold float64) bool {
	return q.UsagePercentage(resourceType) >= threshold
}

// NewTenantContext creates a new TenantContext.
func NewTenantContext(tenantID, tenantName string, role Role, isPlatformAdmin bool) *TenantContext {
	return &TenantContext{
		TenantID:        tenantID,
		TenantName:      tenantName,
		UserRole:        role,
		IsPlatformAdmin: isPlatformAdmin,
	}
}

// WithTenantContext adds tenant context to a context.
func WithTenantContext(ctx context.Context, tc *TenantContext) context.Context {
	ctx = context.WithValue(ctx, TenantIDKey, tc.TenantID)
	ctx = context.WithValue(ctx, TenantNameKey, tc.TenantName)
	ctx = context.WithValue(ctx, TenantRoleKey, tc.UserRole)
	ctx = context.WithValue(ctx, IsPlatformAdminKey, tc.IsPlatformAdmin)
	if tc.Quota != nil {
		ctx = context.WithValue(ctx, QuotaKey, tc.Quota)
	}
	return ctx
}

// GetTenantID extracts the tenant ID from context.
func GetTenantID(ctx context.Context) (string, error) {
	tenantID, ok := ctx.Value(TenantIDKey).(string)
	if !ok || tenantID == "" {
		return "", ErrNoTenantContext
	}
	return tenantID, nil
}

// MustGetTenantID extracts the tenant ID or panics.
// Use this only when you're certain tenant context exists.
func MustGetTenantID(ctx context.Context) string {
	tenantID, err := GetTenantID(ctx)
	if err != nil {
		panic("tenant context required but not available")
	}
	return tenantID
}

// GetTenantName extracts the tenant name from context.
func GetTenantName(ctx context.Context) string {
	name, ok := ctx.Value(TenantNameKey).(string)
	if !ok {
		return ""
	}
	return name
}

// GetTenantRole extracts the tenant role from context.
func GetTenantRole(ctx context.Context) Role {
	role, ok := ctx.Value(TenantRoleKey).(Role)
	if !ok {
		return ""
	}
	return role
}

// IsPlatformAdmin checks if the user is a platform admin.
func IsPlatformAdmin(ctx context.Context) bool {
	isAdmin, ok := ctx.Value(IsPlatformAdminKey).(bool)
	return ok && isAdmin
}

// IsOrgAdmin checks if the user is an org admin.
func IsOrgAdmin(ctx context.Context) bool {
	return GetTenantRole(ctx) == RoleOrgAdmin
}

// GetQuota extracts quota information from context.
func GetQuota(ctx context.Context) (*QuotaInfo, bool) {
	quota, ok := ctx.Value(QuotaKey).(*QuotaInfo)
	return quota, ok
}

// CanAccessTenant checks if the current user can access the specified tenant.
func CanAccessTenant(ctx context.Context, targetTenantID string) bool {
	// Platform admins can access any tenant
	if IsPlatformAdmin(ctx) {
		return true
	}

	// Users can access their own tenant
	tenantID, err := GetTenantID(ctx)
	if err != nil {
		return false
	}

	return tenantID == targetTenantID
}

// RequireTenantAccess returns an error if the user cannot access the specified tenant.
func RequireTenantAccess(ctx context.Context, targetTenantID string) error {
	if !CanAccessTenant(ctx, targetTenantID) {
		return ErrUnauthorizedTenant
	}
	return nil
}

// WithTenantID creates a context with just a tenant ID.
// This is useful for background jobs or internal service calls.
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, TenantIDKey, tenantID)
}

// ClearTenantContext removes tenant context from the context.
// This is useful for platform-level operations that should not be tenant-scoped.
func ClearTenantContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, TenantIDKey, nil)
}
