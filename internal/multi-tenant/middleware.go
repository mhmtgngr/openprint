// Package multitenant provides multi-tenancy support for OpenPrint services.
// This file contains middleware for tenant context propagation.
package multitenant

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/openprint/openprint/internal/auth/roles"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/internal/shared/middleware"
)

// TenantExtractor is a function that extracts tenant information from a request.
type TenantExtractor func(r *http.Request) (tenantID, tenantName string, role Role, isPlatformAdmin bool, err error)

// MiddlewareConfig holds configuration for tenant middleware.
type MiddlewareConfig struct {
	// RequireTenant, when true, returns an error if tenant context cannot be established.
	RequireTenant bool
	// SkipPaths are URL paths that skip tenant validation.
	SkipPaths []string
	// PlatformAdminPaths are paths that require platform admin access.
	PlatformAdminPaths []string
	// TenantExtractor is a function to extract tenant information from the request.
	TenantExtractor TenantExtractor
}

// TenantMiddleware creates middleware that extracts and validates tenant context from JWT claims.
func TenantMiddleware(cfg MiddlewareConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path should be skipped
			for _, skipPath := range cfg.SkipPaths {
				if strings.HasPrefix(r.URL.Path, skipPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			ctx := r.Context()

			// Check if this is a platform admin only path
			for _, adminPath := range cfg.PlatformAdminPaths {
				if strings.HasPrefix(r.URL.Path, adminPath) {
					userRole := middleware.GetRole(r)
					// Use centralized role validation for security
					parsedRole, err := roles.Parse(userRole)
					if err != nil || !parsedRole.IsPlatformAdmin() {
						respondForbidden(w, "platform admin access required")
						return
					}
					// Platform admin paths don't require tenant context
					next.ServeHTTP(w, r)
					return
				}
			}

			// Extract tenant information using the provided extractor or default JWT-based extraction
			var tenantID, tenantName string
			var userRole Role
			var isPlatformAdmin bool
			var err error

			if cfg.TenantExtractor != nil {
				tenantID, tenantName, userRole, isPlatformAdmin, err = cfg.TenantExtractor(r)
			} else {
				// Default extraction from JWT claims in context
				tenantID, tenantName, userRole, isPlatformAdmin, err = extractFromJWT(r)
			}

			if err != nil {
				if cfg.RequireTenant {
					respondError(w, err)
					return
				}
				// Continue without tenant context
				next.ServeHTTP(w, r)
				return
			}

			// Create and set tenant context
			tc := NewTenantContext(tenantID, tenantName, userRole, isPlatformAdmin)
			ctx = WithTenantContext(ctx, tc)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractFromJWT extracts tenant information from JWT claims stored in request context.
func extractFromJWT(r *http.Request) (tenantID, tenantName string, role Role, isPlatformAdmin bool, err error) {
	// Get user info from JWT middleware
	userID := middleware.GetUserID(r)
	userRole := middleware.GetRole(r)
	orgID := middleware.GetOrgID(r)

	if userID == "" {
		return "", "", "", false, errors.New("unauthorized: no user context")
	}

	// Map JWT role to tenant role - do this first for all users
	// Use centralized role validation to prevent authorization bypass
	parsedRole, err := roles.Parse(userRole)
	if err != nil {
		// If role parsing fails, default to least privilege
		parsedRole = roles.RoleOrgUser
	}

	switch parsedRole {
	case roles.RoleAdmin, roles.RolePlatformAdmin:
		role = RolePlatformAdmin
		isPlatformAdmin = true
		// For platform admins, org_id may be empty for platform-level operations
		// or may contain the org they're currently managing
		if orgID == "" {
			// Platform admin with no org context - accessing platform-level resources
			return "", "", RolePlatformAdmin, true, nil
		}
		// Platform admin accessing a specific organization - continue with tenant context
		tenantID = orgID
	case roles.RoleOrgAdmin:
		role = RoleOrgAdmin
		isPlatformAdmin = false
	case roles.RoleUser, roles.RoleOrgUser:
		role = RoleOrgUser
		isPlatformAdmin = false
	case roles.RoleViewer, roles.RoleOrgViewer:
		role = RoleOrgViewer
		isPlatformAdmin = false
	default:
		role = RoleOrgUser
		isPlatformAdmin = false
	}

	// Regular users require organization context
	if tenantID == "" && orgID == "" {
		return "", "", "", false, errors.New("unauthorized: no organization context")
	}

	// Use orgID if tenantID wasn't set
	if tenantID == "" {
		tenantID = orgID
	}

	return tenantID, "", role, isPlatformAdmin, nil
}

// RequireTenant middleware ensures tenant context is present.
func RequireTenant(skipPaths ...string) func(http.Handler) http.Handler {
	return TenantMiddleware(MiddlewareConfig{
		RequireTenant: true,
		SkipPaths:     skipPaths,
	})
}

// RequirePlatformAdmin middleware ensures the user is a platform admin.
func RequirePlatformAdmin(skipPaths ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check skip paths first
			for _, skipPath := range skipPaths {
				if strings.HasPrefix(r.URL.Path, skipPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			ctx := r.Context()

			// Check if user is platform admin using centralized role validation
			userRole := middleware.GetRole(r)
			parsedRole, err := roles.Parse(userRole)
			if err != nil || !parsedRole.IsPlatformAdmin() {
				respondForbidden(w, "platform admin access required")
				return
			}

			// Set platform admin flag in context
			ctx = context.WithValue(ctx, IsPlatformAdminKey, true)
			ctx = context.WithValue(ctx, TenantRoleKey, RolePlatformAdmin)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireOrgAdmin middleware ensures the user is either a platform admin or org admin.
func RequireOrgAdmin(skipPaths ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check skip paths first
			for _, skipPath := range skipPaths {
				if strings.HasPrefix(r.URL.Path, skipPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Use centralized role validation
			userRole := middleware.GetRole(r)
			parsedRole, err := roles.Parse(userRole)
			if err != nil {
				respondForbidden(w, "organization admin access required")
				return
			}

			// Check if user has org admin privileges (includes platform admins)
			if !parsedRole.IsOrgAdmin() {
				respondForbidden(w, "organization admin access required")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireTenantAccessMiddleware ensures the user can access the specified tenant.
// The tenant ID is extracted from the URL using the provided function.
func RequireTenantAccessMiddleware(tenantIDFunc func(*http.Request) (string, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Platform admins can access any tenant
			if IsPlatformAdmin(ctx) {
				next.ServeHTTP(w, r)
				return
			}

			// Get the target tenant ID
			targetTenantID, err := tenantIDFunc(r)
			if err != nil {
				respondError(w, apperrors.Wrap(err, "failed to determine target tenant", http.StatusBadRequest))
				return
			}

			// Check if user can access this tenant
			userTenantID, err := GetTenantID(ctx)
			if err != nil {
				respondError(w, apperrors.Wrap(err, "no tenant context", http.StatusForbidden))
				return
			}

			if userTenantID != targetTenantID {
				respondForbidden(w, "access denied: tenant mismatch")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// TenantFromHeader extracts tenant ID from X-Tenant-ID header.
// This is useful for service-to-service communication where JWT may not be present.
func TenantFromHeader(headerName string) TenantExtractor {
	return func(r *http.Request) (tenantID, tenantName string, role Role, isPlatformAdmin bool, err error) {
		tenantID = r.Header.Get(headerName)
		if tenantID == "" {
			return "", "", "", false, ErrNoTenantContext
		}
		return tenantID, "", RoleOrgUser, false, nil
	}
}

// TenantFromQuery extracts tenant ID from URL query parameter.
func TenantFromQuery(paramName string) TenantExtractor {
	return func(r *http.Request) (tenantID, tenantName string, role Role, isPlatformAdmin bool, err error) {
		tenantID = r.URL.Query().Get(paramName)
		if tenantID == "" {
			return "", "", "", false, ErrNoTenantContext
		}
		return tenantID, "", RoleOrgUser, false, nil
	}
}

// TenantFromURL extracts tenant ID from URL path parameter.
func TenantFromURL(paramName string) TenantExtractor {
	return func(r *http.Request) (tenantID, tenantName string, role Role, isPlatformAdmin bool, err error) {
		// Extract from path using chi or similar URL parameter
		// This is a simplified version - in practice you'd use your router's param extraction
		return "", "", "", false, errors.New("use router-specific param extraction")
	}
}

// respondError sends an error response.
func respondError(w http.ResponseWriter, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(appErr.StatusCode)
		json.NewEncoder(w).Encode(apperrors.ToJSON(err))
		return
	}

	if errors.Is(err, ErrNoTenantContext) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    "TENANT_REQUIRED",
			"message": "Tenant context is required for this request",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]string{
		"code":    "INTERNAL_ERROR",
		"message": "An internal error occurred",
	})
}

// respondForbidden sends a forbidden response.
func respondForbidden(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    "FORBIDDEN",
		"message": message,
	})
}
