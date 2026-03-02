// Package multitenant provides multi-tenancy support for OpenPrint services.
// This file contains middleware for tenant context propagation.
package multitenant

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

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
					role := middleware.GetRole(r)
					if role != "admin" && role != "platform_admin" {
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
	ctx := r.Context()

	// Get user info from JWT middleware
	userID := middleware.GetUserID(r)
	userRole := middleware.GetRole(r)
	orgID := middleware.GetOrgID(r)

	if userID == "" {
		return "", "", "", false, errors.New("unauthorized: no user context")
	}

	// Check if platform admin
	if userRole == "admin" || userRole == "platform_admin" {
		// For platform admins, org_id may be empty for platform-level operations
		// or may contain the org they're currently managing
		return "", "", RolePlatformAdmin, true, nil
	}

	// Regular user - require organization context
	if orgID == "" {
		return "", "", "", false, errors.New("unauthorized: no organization context")
	}

	// Map JWT role to tenant role
	switch userRole {
	case "admin", "platform_admin":
		role = RolePlatformAdmin
		isPlatformAdmin = true
	case "org_admin":
		role = RoleOrgAdmin
	case "user", "org_user":
		role = RoleOrgUser
	case "viewer", "org_viewer":
		role = RoleOrgViewer
	default:
		role = RoleOrgUser
	}

	return orgID, "", role, isPlatformAdmin, nil
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

			// Check if user is platform admin
			role := middleware.GetRole(r)
			if role != "admin" && role != "platform_admin" {
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

			role := middleware.GetRole(r)

			// Platform admins can access
			if role == "admin" || role == "platform_admin" {
				next.ServeHTTP(w, r)
				return
			}

			// Org admins can access
			if role == "org_admin" {
				next.ServeHTTP(w, r)
				return
			}

			respondForbidden(w, "organization admin access required")
		})
	}
}

// RequireTenantAccess ensures the user can access the specified tenant.
// The tenant ID is extracted from the URL using the provided function.
func RequireTenantAccess(tenantIDFunc func(*http.Request) (string, error)) func(http.Handler) http.Handler {
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
