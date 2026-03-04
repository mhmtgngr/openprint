// Package context provides shared context keys for use across all OpenPrint services.
// This is the single source of truth for all context key definitions.
package context

import (
	"context"
	"errors"
)

// ContextKey is a custom type for context keys to prevent collisions.
// Using a custom type prevents collisions with packages using string keys.
// This type is exported to allow other packages to create type aliases.
type ContextKey string

// contextKey is the unexported alias for internal use.
// For backward compatibility, the exported type is ContextKey.
type contextKey = ContextKey

// Context key constants for all OpenPrint services.
// These are the single source of truth - all other packages should import these.
const (
	// User authentication keys
	UserIDKey contextKey = "user_id"
	EmailKey  contextKey = "email"
	OrgIDKey  contextKey = "org_id"
	RoleKey   contextKey = "role"
	ScopesKey contextKey = "scopes"
	TokenKey  contextKey = "token"

	// Tenant/Multi-tenancy keys
	TenantIDKey        contextKey = "tenant_id"
	TenantNameKey      contextKey = "tenant_name"
	TenantRoleKey      contextKey = "tenant_role"
	IsPlatformAdminKey contextKey = "is_platform_admin"
	QuotaKey           contextKey = "quota"

	// Request metadata keys
	RequestIDKey contextKey = "request_id"
	TraceIDKey   contextKey = "trace_id"
)

// Common errors for context operations.
var (
	// ErrNoTenantContext is returned when tenant context is not available.
	ErrNoTenantContext = errors.New("tenant context not available")
	// ErrNoUserContext is returned when user context is not available.
	ErrNoUserContext = errors.New("user context not available")
)

// UserContext holds user-specific context for a request.
type UserContext struct {
	UserID string
	Email  string
	OrgID  string
	Role   string
	Scopes []string
	Token  string
}

// GetUserID extracts the user ID from context.
func GetUserID(ctx context.Context) string {
	val := ctx.Value(UserIDKey)
	if val == nil {
		return ""
	}
	if str, ok := val.(string); ok {
		return str
	}
	return ""
}

// GetEmail extracts the email from context.
func GetEmail(ctx context.Context) string {
	val := ctx.Value(EmailKey)
	if val == nil {
		return ""
	}
	if str, ok := val.(string); ok {
		return str
	}
	return ""
}

// GetOrgID extracts the organization ID from context.
func GetOrgID(ctx context.Context) string {
	val := ctx.Value(OrgIDKey)
	if val == nil {
		return ""
	}
	if str, ok := val.(string); ok {
		return str
	}
	return ""
}

// GetRole extracts the role from context.
func GetRole(ctx context.Context) string {
	val := ctx.Value(RoleKey)
	if val == nil {
		return ""
	}
	if str, ok := val.(string); ok {
		return str
	}
	return ""
}

// GetScopes extracts the scopes from context.
func GetScopes(ctx context.Context) []string {
	val := ctx.Value(ScopesKey)
	if val == nil {
		return nil
	}
	if scopes, ok := val.([]string); ok {
		return scopes
	}
	return nil
}

// WithUserContext adds user information to a context.
func WithUserContext(ctx context.Context, userCtx *UserContext) context.Context {
	ctx = context.WithValue(ctx, UserIDKey, userCtx.UserID)
	ctx = context.WithValue(ctx, EmailKey, userCtx.Email)
	ctx = context.WithValue(ctx, OrgIDKey, userCtx.OrgID)
	ctx = context.WithValue(ctx, RoleKey, userCtx.Role)
	ctx = context.WithValue(ctx, ScopesKey, userCtx.Scopes)
	ctx = context.WithValue(ctx, TokenKey, userCtx.Token)
	return ctx
}

// WithUserID adds a user ID to a context.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// WithEmail adds an email to a context.
func WithEmail(ctx context.Context, email string) context.Context {
	return context.WithValue(ctx, EmailKey, email)
}

// WithOrgID adds an organization ID to a context.
func WithOrgID(ctx context.Context, orgID string) context.Context {
	return context.WithValue(ctx, OrgIDKey, orgID)
}

// WithRole adds a role to a context.
func WithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, RoleKey, role)
}

// WithScopes adds scopes to a context.
func WithScopes(ctx context.Context, scopes []string) context.Context {
	return context.WithValue(ctx, ScopesKey, scopes)
}

// WithToken adds a token to a context.
func WithToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, TokenKey, token)
}

// WithRequestID adds a request ID to a context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// GetRequestID extracts the request ID from context.
func GetRequestID(ctx context.Context) string {
	val := ctx.Value(RequestIDKey)
	if val == nil {
		return ""
	}
	if str, ok := val.(string); ok {
		return str
	}
	return ""
}
