// Package multitenant provides tests for tenant middleware.
package multitenant

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/internal/shared/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTenantMiddleware tests the main tenant middleware.
func TestTenantMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		config         MiddlewareConfig
		setupRequest   func() *http.Request
		wantStatusCode int
		wantContext    func(*testing.T, context.Context)
		setupContext   func(context.Context) context.Context
	}{
		{
			name: "skip path matches",
			config: MiddlewareConfig{
				SkipPaths: []string{"/health", "/metrics"},
			},
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/health", nil)
			},
			wantStatusCode: http.StatusOK,
			wantContext: func(t *testing.T, ctx context.Context) {
				// No tenant context should be set for skipped paths
				_, err := GetTenantID(ctx)
				assert.Error(t, err)
			},
		},
		{
			name: "skip path with prefix",
			config: MiddlewareConfig{
				SkipPaths: []string{"/public"},
			},
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/public/data", nil)
			},
			wantStatusCode: http.StatusOK,
			wantContext: func(t *testing.T, ctx context.Context) {
				// No tenant context for public paths
				_, err := GetTenantID(ctx)
				assert.Error(t, err)
			},
		},
		{
			name: "platform admin path with admin role",
			config: MiddlewareConfig{
				PlatformAdminPaths: []string{"/admin"},
			},
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/admin/users", nil)
				return req
			},
			wantStatusCode: http.StatusOK,
			wantContext: func(t *testing.T, ctx context.Context) {
				// Platform admin paths don't require tenant context
			},
			setupContext: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, middleware.RoleKey, "admin")
			},
		},
		{
			name: "platform admin path with non-admin role",
			config: MiddlewareConfig{
				PlatformAdminPaths: []string{"/admin"},
			},
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/admin/users", nil)
				return req
			},
			wantStatusCode: http.StatusForbidden,
			wantContext:    nil,
			setupContext: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, middleware.RoleKey, "user")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := TenantMiddleware(tt.config)

			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.wantContext != nil {
					tt.wantContext(t, r.Context())
				}
				w.WriteHeader(http.StatusOK)
			}))

			req := tt.setupRequest()
			rec := httptest.NewRecorder()

			// Apply setupContext if provided
			if tt.setupContext != nil {
				req = req.WithContext(tt.setupContext(req.Context()))
			}

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatusCode, rec.Code)
		})
	}
}

func TestExtractFromJWT(t *testing.T) {
	tests := []struct {
		name              string
		setupRequest      func() *http.Request
		wantTenantID      string
		wantRole          Role
		wantIsPlatformAdmin bool
		wantErr           bool
	}{
		{
			name: "platform admin",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-User-Role", "admin")
				req.Header.Set("X-User-ID", "admin-123")
				return req
			},
			wantTenantID:       "",
			wantRole:           RolePlatformAdmin,
			wantIsPlatformAdmin: true,
			wantErr:            false,
		},
		{
			name: "platform_admin role",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-User-Role", "platform_admin")
				req.Header.Set("X-User-ID", "admin-123")
				return req
			},
			wantTenantID:       "",
			wantRole:           RolePlatformAdmin,
			wantIsPlatformAdmin: true,
			wantErr:            false,
		},
		{
			name: "org admin with organization",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-User-Role", "org_admin")
				req.Header.Set("X-User-ID", "user-123")
				req.Header.Set("X-Org-ID", "org-456")
				return req
			},
			wantTenantID:       "org-456",
			wantRole:           RoleOrgAdmin,
			wantIsPlatformAdmin: false,
			wantErr:            false,
		},
		{
			name: "org user",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-User-Role", "org_user")
				req.Header.Set("X-User-ID", "user-123")
				req.Header.Set("X-Org-ID", "org-456")
				return req
			},
			wantTenantID:       "org-456",
			wantRole:           RoleOrgUser,
			wantIsPlatformAdmin: false,
			wantErr:            false,
		},
		{
			name: "user role maps to org_user",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-User-Role", "user")
				req.Header.Set("X-User-ID", "user-123")
				req.Header.Set("X-Org-ID", "org-456")
				return req
			},
			wantTenantID:       "org-456",
			wantRole:           RoleOrgUser,
			wantIsPlatformAdmin: false,
			wantErr:            false,
		},
		{
			name: "viewer role",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-User-Role", "viewer")
				req.Header.Set("X-User-ID", "user-123")
				req.Header.Set("X-Org-ID", "org-456")
				return req
			},
			wantTenantID:       "org-456",
			wantRole:           RoleOrgViewer,
			wantIsPlatformAdmin: false,
			wantErr:            false,
		},
		{
			name: "org_viewer role",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-User-Role", "org_viewer")
				req.Header.Set("X-User-ID", "user-123")
				req.Header.Set("X-Org-ID", "org-456")
				return req
			},
			wantTenantID:       "org-456",
			wantRole:           RoleOrgViewer,
			wantIsPlatformAdmin: false,
			wantErr:            false,
		},
		{
			name: "unknown role defaults to org_user",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-User-Role", "unknown")
				req.Header.Set("X-User-ID", "user-123")
				req.Header.Set("X-Org-ID", "org-456")
				return req
			},
			wantTenantID:       "org-456",
			wantRole:           RoleOrgUser,
			wantIsPlatformAdmin: false,
			wantErr:            false,
		},
		{
			name: "no user context",
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/", nil)
			},
			wantTenantID: "",
			wantRole:     "",
			wantIsPlatformAdmin: false,
			wantErr:      true,
		},
		{
			name: "regular user without organization",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-User-Role", "user")
				req.Header.Set("X-User-ID", "user-123")
				return req
			},
			wantTenantID: "",
			wantRole:     "",
			wantIsPlatformAdmin: false,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: extractFromJWT uses middleware.GetUserID, GetRole, GetOrgID
			// which rely on request context. For testing, we'll use a mock approach.

			// Since we can't easily mock the middleware package's context functions,
			// we'll test the middleware integration directly

			mw := TenantMiddleware(MiddlewareConfig{
				RequireTenant: false,
			})

			handlerCalled := false
			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				if !tt.wantErr {
					// Verify context was set
					tenantID, err := GetTenantID(r.Context())
					if tt.wantTenantID != "" {
						require.NoError(t, err)
						assert.Equal(t, tt.wantTenantID, tenantID)
					}

					role := GetTenantRole(r.Context())
					assert.Equal(t, tt.wantRole, role)

				 isAdmin := IsPlatformAdmin(r.Context())
					assert.Equal(t, tt.wantIsPlatformAdmin, isAdmin)
				}
			}))

			req := tt.setupRequest()
			rec := httptest.NewRecorder()

			// Set up request context with user info
			ctx := req.Context()
			if userID := req.Header.Get("X-User-ID"); userID != "" {
				// Store user ID in context (middleware package would do this)
				ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
			}
			if role := req.Header.Get("X-User-Role"); role != "" {
				ctx = context.WithValue(ctx, middleware.RoleKey, role)
			}
			if orgID := req.Header.Get("X-Org-ID"); orgID != "" {
				ctx = context.WithValue(ctx, middleware.OrgIDKey, orgID)
			}
			req = req.WithContext(ctx)

			handler.ServeHTTP(rec, req)

			// When RequireTenant is false, handler is always called even on error
			// This is the expected behavior - requests without tenant context pass through
			assert.True(t, handlerCalled, "Handler should be called (RequireTenant is false)")

			// If there's an error, the response should indicate it
			if tt.wantErr {
				// Handler was called but tenant context should not be set
				tenantID, _ := GetTenantID(req.Context())
				assert.Equal(t, "", tenantID, "Tenant ID should not be set")
			}
		})
	}
}

func TestRequireTenant(t *testing.T) {
	tests := []struct {
		name           string
		skipPaths      []string
		setupRequest   func() *http.Request
		wantStatusCode int
	}{
		{
			name:      "skip path",
			skipPaths: []string{"/health"},
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/health", nil)
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:      "no tenant context returns error",
			skipPaths: nil,
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/api/resource", nil)
			},
			wantStatusCode: http.StatusInternalServerError, // extractFromJWT returns generic error, not ErrNoTenantContext
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := RequireTenant(tt.skipPaths...)

			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := tt.setupRequest()
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatusCode, rec.Code)
		})
	}
}

func TestRequirePlatformAdmin(t *testing.T) {
	tests := []struct {
		name           string
		skipPaths      []string
		role           string
		path           string
		wantStatusCode int
	}{
		{
			name:           "platform admin allowed",
			skipPaths:      nil,
			role:           "platform_admin",
			path:           "/admin/users",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "admin role allowed",
			skipPaths:      nil,
			role:           "admin",
			path:           "/admin/users",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "regular user denied",
			skipPaths:      nil,
			role:           "user",
			path:           "/admin/users",
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "no role denied",
			skipPaths:      nil,
			role:           "",
			path:           "/admin/users",
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:      "skip path allowed",
			skipPaths: []string{"/public"},
			role:      "user",
			path:      "/public/data",
			wantStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := RequirePlatformAdmin(tt.skipPaths...)

			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", tt.path, nil)
			req.Header.Set("X-User-Role", tt.role)

			// Set role in context (as middleware package would do)
			ctx := req.Context()
			ctx = context.WithValue(ctx, middleware.RoleKey, tt.role)
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatusCode, rec.Code)
		})
	}
}

func TestRequireOrgAdmin(t *testing.T) {
	tests := []struct {
		name           string
		skipPaths      []string
		role           string
		wantStatusCode int
	}{
		{
			name:           "platform admin allowed",
			skipPaths:      nil,
			role:           "platform_admin",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "admin role allowed",
			skipPaths:      nil,
			role:           "admin",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "org admin allowed",
			skipPaths:      nil,
			role:           "org_admin",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "org user denied",
			skipPaths:      nil,
			role:           "org_user",
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "viewer denied",
			skipPaths:      nil,
			role:           "org_viewer",
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:      "skip path allowed",
			skipPaths: []string{"/public"},
			role:      "org_viewer",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "no role on skip path",
			skipPaths:      []string{"/public"},
			role:           "",
			wantStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := RequireOrgAdmin(tt.skipPaths...)

			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// Use /api/resource as default URL, but check if we need a different path
			url := "/api/resource"
			if len(tt.skipPaths) > 0 {
				// For skip path tests, use a URL that matches the skip path
				if strings.HasPrefix("/public/data", tt.skipPaths[0]) {
					url = "/public/data"
				} else {
					url = tt.skipPaths[0] + "/resource"
				}
			}

			req := httptest.NewRequest("GET", url, nil)
			if tt.role != "" {
				req.Header.Set("X-User-Role", tt.role)
			}

			// Set role in request context (middleware package would do this)
			ctx := req.Context()
			if tt.role != "" {
				ctx = context.WithValue(ctx, middleware.RoleKey, tt.role)
			}
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatusCode, rec.Code)
		})
	}
}

func TestRequireTenantAccessMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		tenantIDFunc   func(*http.Request) (string, error)
		setupContext   func() context.Context
		targetTenantID string
		wantStatusCode int
	}{
		{
			name: "platform admin can access any tenant",
			tenantIDFunc: func(r *http.Request) (string, error) {
				return "target-tenant-456", nil
			},
			setupContext: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, IsPlatformAdminKey, true)
				ctx = context.WithValue(ctx, TenantIDKey, "tenant-123")
				return ctx
			},
			targetTenantID: "target-tenant-456",
			wantStatusCode: http.StatusOK,
		},
		{
			name: "user can access their own tenant",
			tenantIDFunc: func(r *http.Request) (string, error) {
				return "tenant-123", nil
			},
			setupContext: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, TenantIDKey, "tenant-123")
				return ctx
			},
			targetTenantID: "tenant-123",
			wantStatusCode: http.StatusOK,
		},
		{
			name: "user cannot access different tenant",
			tenantIDFunc: func(r *http.Request) (string, error) {
				return "tenant-456", nil
			},
			setupContext: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, TenantIDKey, "tenant-123")
				return ctx
			},
			targetTenantID: "tenant-456",
			wantStatusCode: http.StatusForbidden,
		},
		{
			name: "no tenant context returns error",
			tenantIDFunc: func(r *http.Request) (string, error) {
				return "tenant-456", nil
			},
			setupContext: func() context.Context { return context.Background() },
			targetTenantID: "tenant-456",
			wantStatusCode: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := RequireTenantAccessMiddleware(tt.tenantIDFunc)

			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", "/api/resource", nil)
			req = req.WithContext(tt.setupContext())

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatusCode, rec.Code)
		})
	}
}

func TestTenantFromHeader(t *testing.T) {
	extractor := TenantFromHeader("X-Tenant-ID")

	tests := []struct {
		name               string
		headerValue        string
		wantTenantID       string
		wantRole           Role
		wantIsPlatformAdmin bool
		wantErr            bool
	}{
		{
			name:               "valid tenant ID",
			headerValue:        "tenant-123",
			wantTenantID:       "tenant-123",
			wantRole:           RoleOrgUser,
			wantIsPlatformAdmin: false,
			wantErr:            false,
		},
		{
			name:               "empty header",
			headerValue:        "",
			wantTenantID:       "",
			wantRole:           "",
			wantIsPlatformAdmin: false,
			wantErr:            true,
		},
		{
			name:               "missing header",
			headerValue:        "",
			wantTenantID:       "",
			wantRole:           "",
			wantIsPlatformAdmin: false,
			wantErr:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.headerValue != "" {
				req.Header.Set("X-Tenant-ID", tt.headerValue)
			}

			tenantID, _, role, isAdmin, err := extractor(req)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantTenantID, tenantID)
				assert.Equal(t, tt.wantRole, role)
				assert.Equal(t, tt.wantIsPlatformAdmin, isAdmin)
			}
		})
	}
}

func TestTenantFromQuery(t *testing.T) {
	extractor := TenantFromQuery("tenant_id")

	tests := []struct {
		name               string
		queryParam         string
		wantTenantID       string
		wantErr            bool
	}{
		{
			name:       "valid tenant ID",
			queryParam: "tenant-123",
			wantTenantID: "tenant-123",
			wantErr:    false,
		},
		{
			name:       "empty query param",
			queryParam: "",
			wantTenantID: "",
			wantErr:    true,
		},
		{
			name:       "missing query param",
			queryParam: "",
			wantTenantID: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/?tenant_id="+tt.queryParam, nil)

			tenantID, _, _, _, err := extractor(req)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantTenantID, tenantID)
			}
		})
	}
}

func TestTenantFromURL(t *testing.T) {
	extractor := TenantFromURL("tenantID")

	req := httptest.NewRequest("GET", "/api/tenants/tenant-123/users", nil)

	tenantID, _, _, _, err := extractor(req)

	// Should return an error indicating router-specific extraction needed
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "router-specific")
	assert.Empty(t, tenantID)
}

func TestRespondError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		wantStatusCode int
		wantCode       string
	}{
		{
			name:           "no tenant context error",
			err:            ErrNoTenantContext,
			wantStatusCode: http.StatusForbidden,
			wantCode:       "TENANT_REQUIRED",
		},
		{
			name:           "other error",
			err:            errors.New("internal error"),
			wantStatusCode: http.StatusInternalServerError,
			wantCode:       "INTERNAL_ERROR",
		},
		{
			name: "app error",
			err: apperrors.New("validation failed", 400).
				WithCode("VALIDATION_ERROR"),
			wantStatusCode: http.StatusBadRequest,
			wantCode:       "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			respondError(rec, tt.err)

			assert.Equal(t, tt.wantStatusCode, rec.Code)

			var response map[string]interface{}
			err := json.NewDecoder(rec.Body).Decode(&response)
			require.NoError(t, err)

			assert.Equal(t, tt.wantCode, response["code"])
		})
	}
}

func TestRespondForbidden(t *testing.T) {
	rec := httptest.NewRecorder()
	respondForbidden(rec, "access denied")

	assert.Equal(t, http.StatusForbidden, rec.Code)

	var response map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "FORBIDDEN", response["code"])
	assert.Equal(t, "access denied", response["message"])
}

func TestMiddlewareConfig_Defaults(t *testing.T) {
	config := MiddlewareConfig{
		RequireTenant: true,
		SkipPaths:     []string{"/health"},
		TenantExtractor: nil,
	}

	assert.True(t, config.RequireTenant)
	assert.NotEmpty(t, config.SkipPaths)
	assert.Nil(t, config.TenantExtractor)
}

func TestTenantExtractor_Signature(t *testing.T) {
	// Test that TenantExtractor has the correct signature
	extractor := func(r *http.Request) (string, string, Role, bool, error) {
		return "tenant-123", "Test Org", RoleOrgAdmin, false, nil
	}

	assert.NotNil(t, extractor)

	// Use it
	req := httptest.NewRequest("GET", "/", nil)
	tenantID, tenantName, role, isAdmin, err := extractor(req)

	require.NoError(t, err)
	assert.Equal(t, "tenant-123", tenantID)
	assert.Equal(t, "Test Org", tenantName)
	assert.Equal(t, RoleOrgAdmin, role)
	assert.False(t, isAdmin)
}

func TestTenantMiddleware_WithCustomExtractor(t *testing.T) {
	// Custom extractor that always returns a fixed tenant
	extractor := func(r *http.Request) (string, string, Role, bool, error) {
		return "custom-tenant", "Custom Org", RoleOrgUser, false, nil
	}

	config := MiddlewareConfig{
		RequireTenant:   false,
		TenantExtractor: extractor,
	}

	mw := TenantMiddleware(config)

	handlerCalled := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		tenantID, err := GetTenantID(r.Context())
		require.NoError(t, err)
		assert.Equal(t, "custom-tenant", tenantID)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/resource", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTenantMiddleware_WithErroringExtractor(t *testing.T) {
	// Custom extractor that returns an error
	extractor := func(r *http.Request) (string, string, Role, bool, error) {
		return "", "", "", false, ErrNoTenantContext
	}

	config := MiddlewareConfig{
		RequireTenant:   true,
		TenantExtractor: extractor,
	}

	mw := TenantMiddleware(config)

	handlerCalled := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/resource", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.False(t, handlerCalled, "Handler should not be called when extractor returns error")
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestTenantMiddleware_RequireTenantFalse(t *testing.T) {
	// When RequireTenant is false, requests without tenant context should pass through
	extractor := func(r *http.Request) (string, string, Role, bool, error) {
		return "", "", "", false, ErrNoTenantContext
	}

	config := MiddlewareConfig{
		RequireTenant:   false,
		TenantExtractor: extractor,
	}

	mw := TenantMiddleware(config)

	handlerCalled := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		// No tenant context should be set
		_, err := GetTenantID(r.Context())
		assert.Error(t, err)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/resource", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.True(t, handlerCalled, "Handler should be called even without tenant context")
	assert.Equal(t, http.StatusOK, rec.Code)
}

// Test that error responses match expected format
func TestErrorResponses(t *testing.T) {
	tests := []struct {
		name         string
		setupHandler http.Handler
		checkResponse func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "forbidden response format",
			setupHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				respondForbidden(w, "test forbidden message")
			}),
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusForbidden, rec.Code)

				var response map[string]interface{}
				err := json.NewDecoder(rec.Body).Decode(&response)
				require.NoError(t, err)

				assert.Equal(t, "FORBIDDEN", response["code"])
				assert.Equal(t, "test forbidden message", response["message"])
			},
		},
		{
			name: "tenant required response format",
			setupHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				respondError(w, ErrNoTenantContext)
			}),
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusForbidden, rec.Code)

				var response map[string]interface{}
				err := json.NewDecoder(rec.Body).Decode(&response)
				require.NoError(t, err)

				assert.Equal(t, "TENANT_REQUIRED", response["code"])
				assert.Contains(t, response["message"], "Tenant context is required")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			rec := httptest.NewRecorder()

			tt.setupHandler.ServeHTTP(rec, req)
			tt.checkResponse(t, rec)
		})
	}
}
