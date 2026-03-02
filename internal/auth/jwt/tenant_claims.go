// Package jwt provides JWT token generation and validation for OpenPrint authentication.
// This file contains tenant-specific JWT claims and utilities.
package jwt

import (
	"context"
	"errors"
)

var (
	// ErrMissingTenantClaim is returned when the tenant claim is missing.
	ErrMissingTenantClaim = errors.New("missing tenant claim in token")
	// ErrInvalidTenantID is returned when the tenant ID is invalid.
	ErrInvalidTenantID = errors.New("invalid tenant ID")
)

// TenantClaims extends the base Claims with tenant-specific information.
type TenantClaims struct {
	Claims
	// TenantRole represents the user's role within the tenant.
	TenantRole string `json:"tenant_role"`
	// IsPlatformAdmin indicates if the user is a platform admin.
	IsPlatformAdmin bool `json:"is_platform_admin"`
	// TenantName is the human-readable name of the tenant.
	TenantName string `json:"tenant_name,omitempty"`
	// TenantStatus indicates the current status of the tenant.
	TenantStatus string `json:"tenant_status,omitempty"`
}

// ContextKey is the type used for context keys.
type contextKey string

const (
	// TenantIDKey is the context key for tenant ID.
	TenantIDKey contextKey = "tenant_id"
	// TenantRoleKey is the context key for tenant role.
	TenantRoleKey contextKey = "tenant_role"
	// IsPlatformAdminKey is the context key for platform admin flag.
	IsPlatformAdminKey contextKey = "is_platform_admin"
)

// GenerateTenantTokenPair generates access and refresh tokens with tenant claims.
func (m *Manager) GenerateTenantTokenPair(userID, email, role string, orgID string, tenantRole string, isPlatformAdmin bool, scopes []string) (accessToken, refreshToken string, err error) {
	accessToken, err = m.GenerateTenantToken(userID, email, role, orgID, tenantRole, isPlatformAdmin, scopes, AccessTokenType)
	if err != nil {
		return "", "", err
	}

	refreshToken, err = m.GenerateTenantToken(userID, email, role, orgID, tenantRole, isPlatformAdmin, nil, RefreshTokenType)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// GenerateTenantToken generates a JWT token with tenant-specific claims.
func (m *Manager) GenerateTenantToken(userID, email, role string, orgID string, tenantRole string, isPlatformAdmin bool, scopes []string, tokenType TokenType) (string, error) {
	// Use the base GenerateToken method which already supports org_id
	return m.GenerateToken(userID, email, role, orgID, scopes, tokenType)
}

// ValidateTenantToken validates a JWT token and returns the tenant claims.
func (m *Manager) ValidateTenantToken(tokenString string) (*TenantClaims, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	tenantClaims := &TenantClaims{
		Claims:          *claims,
		TenantRole:      claims.Role, // Map role to tenant_role
		IsPlatformAdmin: claims.Role == "admin" || claims.Role == "platform_admin",
	}

	return tenantClaims, nil
}

// TenantIDFromContext extracts tenant ID from context.
func TenantIDFromContext(ctx context.Context) (string, error) {
	tenantID, ok := ctx.Value(TenantIDKey).(string)
	if !ok || tenantID == "" {
		return "", ErrMissingTenantClaim
	}
	return tenantID, nil
}

// MustTenantIDFromContext extracts tenant ID or panics.
func MustTenantIDFromContext(ctx context.Context) string {
	tenantID, err := TenantIDFromContext(ctx)
	if err != nil {
		panic("tenant context required but not available")
	}
	return tenantID
}

// TenantRoleFromContext extracts tenant role from context.
func TenantRoleFromContext(ctx context.Context) (string, error) {
	role, ok := ctx.Value(TenantRoleKey).(string)
	if !ok || role == "" {
		return "", ErrMissingTenantClaim
	}
	return role, nil
}

// IsPlatformAdminFromContext checks if user is platform admin from context.
func IsPlatformAdminFromContext(ctx context.Context) bool {
	isAdmin, ok := ctx.Value(IsPlatformAdminKey).(bool)
	return ok && isAdmin
}

// WithTenantContext adds tenant information to a context.
func WithTenantContext(ctx context.Context, tenantID, tenantRole string, isPlatformAdmin bool) context.Context {
	ctx = context.WithValue(ctx, TenantIDKey, tenantID)
	ctx = context.WithValue(ctx, TenantRoleKey, tenantRole)
	ctx = context.WithValue(ctx, IsPlatformAdminKey, isPlatformAdmin)
	return ctx
}

// TenantRole constants for multi-tenant users.
const (
	TenantRoleOwner      = "owner"
	TenantRoleAdmin      = "admin"
	TenantRoleUser       = "user"
	TenantRoleViewer     = "viewer"
	TenantRoleBilling    = "billing"
)

// IsValidTenantRole checks if a role is a valid tenant role.
func IsValidTenantRole(role string) bool {
	switch role {
	case TenantRoleOwner, TenantRoleAdmin, TenantRoleUser, TenantRoleViewer, TenantRoleBilling:
		return true
	default:
		return false
	}
}

// TenantRolePermissions returns the permissions for a given tenant role.
func TenantRolePermissions(role string) []string {
	switch role {
	case TenantRoleOwner, TenantRoleAdmin:
		return []string{
			"tenant:read", "tenant:write", "tenant:delete",
			"user:read", "user:write", "user:delete", "user:invite",
			"printer:read", "printer:write", "printer:delete",
			"job:read", "job:write", "job:delete",
			"document:read", "document:write", "document:delete",
			"quota:read", "quota:write",
			"audit:read",
			"billing:read", "billing:write",
		}
	case TenantRoleUser:
		return []string{
			"tenant:read",
			"printer:read", "printer:write",
			"job:read", "job:write",
			"document:read", "document:write",
		}
	case TenantRoleViewer:
		return []string{
			"tenant:read",
			"printer:read",
			"job:read",
			"document:read",
		}
	case TenantRoleBilling:
		return []string{
			"tenant:read",
			"billing:read", "billing:write",
			"quota:read",
			"audit:read",
		}
	default:
		return []string{}
	}
}

// HasTenantPermission checks if a tenant role has a specific permission.
func HasTenantPermission(role string, permission string) bool {
	permissions := TenantRolePermissions(role)
	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}
