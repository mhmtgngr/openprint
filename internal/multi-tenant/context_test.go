// Package multitenant provides tests for tenant context management.
package multitenant

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTenantContext(t *testing.T) {
	tests := []struct {
		name            string
		tenantID        string
		tenantName      string
		role            Role
		isPlatformAdmin bool
	}{
		{
			name:            "platform admin",
			tenantID:        "",
			tenantName:      "",
			role:            RolePlatformAdmin,
			isPlatformAdmin: true,
		},
		{
			name:            "org admin",
			tenantID:        "tenant-123",
			tenantName:      "Acme Corp",
			role:            RoleOrgAdmin,
			isPlatformAdmin: false,
		},
		{
			name:            "org user",
			tenantID:        "tenant-456",
			tenantName:      "Globex Inc",
			role:            RoleOrgUser,
			isPlatformAdmin: false,
		},
		{
			name:            "org viewer",
			tenantID:        "tenant-789",
			tenantName:      "Soylent Corp",
			role:            RoleOrgViewer,
			isPlatformAdmin: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := NewTenantContext(tt.tenantID, tt.tenantName, tt.role, tt.isPlatformAdmin)

			assert.Equal(t, tt.tenantID, tc.TenantID)
			assert.Equal(t, tt.tenantName, tc.TenantName)
			assert.Equal(t, tt.role, tc.UserRole)
			assert.Equal(t, tt.isPlatformAdmin, tc.IsPlatformAdmin)
		})
	}
}

func TestWithTenantContext(t *testing.T) {
	ctx := context.Background()

	tc := &TenantContext{
		TenantID:        "tenant-123",
		TenantName:      "Test Org",
		UserRole:        RoleOrgAdmin,
		IsPlatformAdmin: false,
		Quota: &QuotaInfo{
			MaxPrinters:     10,
			CurrentPrinters: 5,
		},
	}

	result := WithTenantContext(ctx, tc)

	// Verify tenant ID can be retrieved
	tenantID, err := GetTenantID(result)
	require.NoError(t, err)
	assert.Equal(t, "tenant-123", tenantID)

	// Verify tenant name can be retrieved
	tenantName := GetTenantName(result)
	assert.Equal(t, "Test Org", tenantName)

	// Verify role can be retrieved
	role := GetTenantRole(result)
	assert.Equal(t, RoleOrgAdmin, role)

	// Verify platform admin flag
	isAdmin := IsPlatformAdmin(result)
	assert.False(t, isAdmin)

	// Verify quota can be retrieved
	quota, ok := GetQuota(result)
	require.True(t, ok)
	require.NotNil(t, quota)
	assert.Equal(t, int32(10), quota.MaxPrinters)
}

func TestGetTenantID(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx func() context.Context
		wantID   string
		wantErr  error
	}{
		{
			name: "valid tenant ID",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), TenantIDKey, "tenant-123")
			},
			wantID:  "tenant-123",
			wantErr: nil,
		},
		{
			name: "missing tenant ID",
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantID:  "",
			wantErr: ErrNoTenantContext,
		},
		{
			name: "empty string tenant ID",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), TenantIDKey, "")
			},
			wantID:  "",
			wantErr: ErrNoTenantContext,
		},
		{
			name: "wrong type in context",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), TenantIDKey, 123)
			},
			wantID:  "",
			wantErr: ErrNoTenantContext,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			id, err := GetTenantID(ctx)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantID, id)
			}
		})
	}
}

func TestMustGetTenantID(t *testing.T) {
	t.Run("valid tenant ID returns ID", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), TenantIDKey, "tenant-123")
		id := MustGetTenantID(ctx)
		assert.Equal(t, "tenant-123", id)
	})

	t.Run("missing tenant ID panics", func(t *testing.T) {
		ctx := context.Background()
		assert.Panics(t, func() {
			MustGetTenantID(ctx)
		})
	})
}

func TestGetTenantName(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx func() context.Context
		wantName string
	}{
		{
			name: "valid tenant name",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), TenantNameKey, "Acme Corp")
			},
			wantName: "Acme Corp",
		},
		{
			name: "missing tenant name",
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantName: "",
		},
		{
			name: "wrong type in context",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), TenantNameKey, 123)
			},
			wantName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			name := GetTenantName(ctx)
			assert.Equal(t, tt.wantName, name)
		})
	}
}

func TestGetTenantRole(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx func() context.Context
		wantRole Role
	}{
		{
			name: "platform admin role",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), TenantRoleKey, RolePlatformAdmin)
			},
			wantRole: RolePlatformAdmin,
		},
		{
			name: "org admin role",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), TenantRoleKey, RoleOrgAdmin)
			},
			wantRole: RoleOrgAdmin,
		},
		{
			name: "org user role",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), TenantRoleKey, RoleOrgUser)
			},
			wantRole: RoleOrgUser,
		},
		{
			name: "org viewer role",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), TenantRoleKey, RoleOrgViewer)
			},
			wantRole: RoleOrgViewer,
		},
		{
			name: "missing role",
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantRole: "",
		},
		{
			name: "wrong type in context",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), TenantRoleKey, 123)
			},
			wantRole: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			role := GetTenantRole(ctx)
			assert.Equal(t, tt.wantRole, role)
		})
	}
}

func TestIsPlatformAdmin(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx func() context.Context
		wantBool bool
	}{
		{
			name: "is platform admin",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), IsPlatformAdminKey, true)
			},
			wantBool: true,
		},
		{
			name: "is not platform admin",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), IsPlatformAdminKey, false)
			},
			wantBool: false,
		},
		{
			name:     "missing platform admin flag",
			setupCtx: func() context.Context { return context.Background() },
			wantBool: false,
		},
		{
			name: "wrong type in context",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), IsPlatformAdminKey, "true")
			},
			wantBool: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			isAdmin := IsPlatformAdmin(ctx)
			assert.Equal(t, tt.wantBool, isAdmin)
		})
	}
}

func TestIsOrgAdmin(t *testing.T) {
	tests := []struct {
		name     string
		role     Role
		wantBool bool
	}{
		{
			name:     "org admin",
			role:     RoleOrgAdmin,
			wantBool: true,
		},
		{
			name:     "platform admin",
			role:     RolePlatformAdmin,
			wantBool: false,
		},
		{
			name:     "org user",
			role:     RoleOrgUser,
			wantBool: false,
		},
		{
			name:     "org viewer",
			role:     RoleOrgViewer,
			wantBool: false,
		},
		{
			name:     "empty role",
			role:     "",
			wantBool: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), TenantRoleKey, tt.role)
			isAdmin := IsOrgAdmin(ctx)
			assert.Equal(t, tt.wantBool, isAdmin)
		})
	}
}

func TestGetQuota(t *testing.T) {
	tests := []struct {
		name      string
		setupCtx  func() context.Context
		wantQuota *QuotaInfo
		wantOK    bool
	}{
		{
			name: "quota exists",
			setupCtx: func() context.Context {
				quota := &QuotaInfo{
					MaxPrinters:     100,
					CurrentPrinters: 50,
				}
				return context.WithValue(context.Background(), QuotaKey, quota)
			},
			wantQuota: &QuotaInfo{
				MaxPrinters:     100,
				CurrentPrinters: 50,
			},
			wantOK: true,
		},
		{
			name:      "quota missing",
			setupCtx:  func() context.Context { return context.Background() },
			wantQuota: nil,
			wantOK:    false,
		},
		{
			name: "wrong type in context",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), QuotaKey, "not a quota")
			},
			wantQuota: nil,
			wantOK:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			quota, ok := GetQuota(ctx)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantQuota != nil {
				require.NotNil(t, quota)
				assert.Equal(t, tt.wantQuota.MaxPrinters, quota.MaxPrinters)
				assert.Equal(t, tt.wantQuota.CurrentPrinters, quota.CurrentPrinters)
			} else {
				assert.Nil(t, quota)
			}
		})
	}
}

func TestCanAccessTenant(t *testing.T) {
	tests := []struct {
		name           string
		setupCtx       func() context.Context
		targetTenantID string
		wantBool       bool
	}{
		{
			name: "platform admin can access any tenant",
			setupCtx: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, IsPlatformAdminKey, true)
				return ctx
			},
			targetTenantID: "any-tenant-123",
			wantBool:       true,
		},
		{
			name: "user can access their own tenant",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), TenantIDKey, "tenant-123")
			},
			targetTenantID: "tenant-123",
			wantBool:       true,
		},
		{
			name: "user cannot access different tenant",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), TenantIDKey, "tenant-123")
			},
			targetTenantID: "tenant-456",
			wantBool:       false,
		},
		{
			name:           "user without tenant cannot access",
			setupCtx:       func() context.Context { return context.Background() },
			targetTenantID: "tenant-123",
			wantBool:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			result := CanAccessTenant(ctx, tt.targetTenantID)
			assert.Equal(t, tt.wantBool, result)
		})
	}
}

func TestRequireTenantAccess(t *testing.T) {
	tests := []struct {
		name           string
		setupCtx       func() context.Context
		targetTenantID string
		wantErr        error
	}{
		{
			name: "platform admin can access any tenant",
			setupCtx: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, IsPlatformAdminKey, true)
				return ctx
			},
			targetTenantID: "any-tenant-123",
			wantErr:        nil,
		},
		{
			name: "user can access their own tenant",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), TenantIDKey, "tenant-123")
			},
			targetTenantID: "tenant-123",
			wantErr:        nil,
		},
		{
			name: "user cannot access different tenant",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), TenantIDKey, "tenant-123")
			},
			targetTenantID: "tenant-456",
			wantErr:        ErrUnauthorizedTenant,
		},
		{
			name:           "user without tenant cannot access",
			setupCtx:       func() context.Context { return context.Background() },
			targetTenantID: "tenant-123",
			wantErr:        ErrUnauthorizedTenant,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			err := RequireTenantAccess(ctx, tt.targetTenantID)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestWithTenantID(t *testing.T) {
	ctx := context.Background()
	result := WithTenantID(ctx, "tenant-123")

	tenantID, err := GetTenantID(result)
	require.NoError(t, err)
	assert.Equal(t, "tenant-123", tenantID)
}

func TestClearTenantContext(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, TenantIDKey, "tenant-123")

	// Verify tenant ID exists
	tenantID, err := GetTenantID(ctx)
	require.NoError(t, err)
	assert.Equal(t, "tenant-123", tenantID)

	// Clear tenant context
	result := ClearTenantContext(ctx)

	// Verify tenant ID is cleared (set to nil)
	resultTenantID, err := GetTenantID(result)
	assert.Error(t, err)
	assert.Equal(t, "", resultTenantID)
}

func TestQuotaInfo_UsagePercentage(t *testing.T) {
	tests := []struct {
		name         string
		quota        *QuotaInfo
		resourceType string
		wantPercent  float64
	}{
		{
			name: "printers usage",
			quota: &QuotaInfo{
				MaxPrinters:     100,
				CurrentPrinters: 50,
			},
			resourceType: "printers",
			wantPercent:  50.0,
		},
		{
			name: "printers at limit",
			quota: &QuotaInfo{
				MaxPrinters:     100,
				CurrentPrinters: 100,
			},
			resourceType: "printers",
			wantPercent:  100.0,
		},
		{
			name: "printers unlimited",
			quota: &QuotaInfo{
				MaxPrinters:     0,
				CurrentPrinters: 50,
			},
			resourceType: "printers",
			wantPercent:  0,
		},
		{
			name: "storage usage",
			quota: &QuotaInfo{
				MaxStorageGB:     10,                     // 10 GB
				CurrentStorageGB: 5 * 1024 * 1024 * 1024, // 5 GB in bytes
			},
			resourceType: "storage",
			wantPercent:  50.0,
		},
		{
			name: "storage unlimited",
			quota: &QuotaInfo{
				MaxStorageGB:     0,
				CurrentStorageGB: 5 * 1024 * 1024 * 1024,
			},
			resourceType: "storage",
			wantPercent:  0,
		},
		{
			name: "jobs usage",
			quota: &QuotaInfo{
				MaxJobsPerMonth: 1000,
				CurrentJobs:     250,
			},
			resourceType: "jobs",
			wantPercent:  25.0,
		},
		{
			name: "users usage",
			quota: &QuotaInfo{
				MaxUsers:     50,
				CurrentUsers: 25,
			},
			resourceType: "users",
			wantPercent:  50.0,
		},
		{
			name: "unknown resource type",
			quota: &QuotaInfo{
				MaxUsers:     50,
				CurrentUsers: 25,
			},
			resourceType: "unknown",
			wantPercent:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			percent := tt.quota.UsagePercentage(tt.resourceType)
			assert.Equal(t, tt.wantPercent, percent)
		})
	}
}

func TestQuotaInfo_IsNearLimit(t *testing.T) {
	tests := []struct {
		name          string
		quota         *QuotaInfo
		resourceType  string
		threshold     float64
		wantNearLimit bool
	}{
		{
			name: "below threshold",
			quota: &QuotaInfo{
				MaxPrinters:     100,
				CurrentPrinters: 50,
			},
			resourceType:  "printers",
			threshold:     80.0,
			wantNearLimit: false,
		},
		{
			name: "at threshold",
			quota: &QuotaInfo{
				MaxPrinters:     100,
				CurrentPrinters: 80,
			},
			resourceType:  "printers",
			threshold:     80.0,
			wantNearLimit: true,
		},
		{
			name: "above threshold",
			quota: &QuotaInfo{
				MaxPrinters:     100,
				CurrentPrinters: 90,
			},
			resourceType:  "printers",
			threshold:     80.0,
			wantNearLimit: true,
		},
		{
			name: "unknown resource type",
			quota: &QuotaInfo{
				MaxPrinters:     100,
				CurrentPrinters: 90,
			},
			resourceType:  "unknown",
			threshold:     80.0,
			wantNearLimit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.quota.IsNearLimit(tt.resourceType, tt.threshold)
			assert.Equal(t, tt.wantNearLimit, result)
		})
	}
}

func TestRoleValues(t *testing.T) {
	tests := []struct {
		role    Role
		wantVal string
	}{
		{RolePlatformAdmin, "platform_admin"},
		{RoleOrgAdmin, "org_admin"},
		{RoleOrgUser, "org_user"},
		{RoleOrgViewer, "org_viewer"},
	}

	for _, tt := range tests {
		t.Run(tt.wantVal, func(t *testing.T) {
			assert.Equal(t, tt.wantVal, string(tt.role))
		})
	}
}

func TestErrorValues(t *testing.T) {
	assert.Equal(t, "tenant context not available", ErrNoTenantContext.Error())
	assert.Equal(t, "unauthorized: tenant mismatch", ErrUnauthorizedTenant.Error())
}
