// Package multitenant provides tests for quota enforcement logic.
package multitenant

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockQuotaRepository is a mock implementation of QuotaRepository for testing.
type mockQuotaRepository struct {
	quota  *QuotaInfo
	getErr error
}

func (m *mockQuotaRepository) GetQuota(ctx context.Context, tenantID string) (*QuotaInfo, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.quota, nil
}

func (m *mockQuotaRepository) GetCurrentUsage(ctx context.Context, tenantID string) (*QuotaInfo, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.quota, nil
}

func (m *mockQuotaRepository) UpdateUsage(ctx context.Context, tenantID string, resourceType ResourceType, delta int64) error {
	return nil
}

func TestNewQuotaEnforcer(t *testing.T) {
	repo := &mockQuotaRepository{}
	enforcer := NewQuotaEnforcer(repo)

	require.NotNil(t, enforcer)
	assert.Equal(t, repo, enforcer.repo)
	assert.Equal(t, 80.0, enforcer.warningThreshold)
}

func TestQuotaEnforcer_SetWarningThreshold(t *testing.T) {
	repo := &mockQuotaRepository{}
	enforcer := NewQuotaEnforcer(repo)

	enforcer.SetWarningThreshold(90.0)
	assert.Equal(t, 90.0, enforcer.warningThreshold)
}

func TestQuotaEnforcer_CheckPrinterQuota(t *testing.T) {
	tests := []struct {
		name          string
		quota         *QuotaInfo
		tenantID      string
		count         int
		threshold     float64
		wantStatus    QuotaStatus
		wantAllowed   bool
		wantCurrent   int64
		wantMaximum   int64
		wantPercent   float64
		wantRemaining int64
		wantErr       error
	}{
		{
			name: "unlimited quota",
			quota: &QuotaInfo{
				MaxPrinters:     0, // 0 = unlimited
				CurrentPrinters: 5,
			},
			tenantID:      "tenant-123",
			count:         1,
			threshold:     80.0,
			wantStatus:    QuotaStatusOK,
			wantAllowed:   true,
			wantCurrent:   5,
			wantMaximum:   -1, // -1 indicates unlimited
			wantPercent:   0,
			wantRemaining: -1,
		},
		{
			name: "under limit",
			quota: &QuotaInfo{
				MaxPrinters:     100,
				CurrentPrinters: 50,
			},
			tenantID:      "tenant-123",
			count:         1,
			threshold:     80.0,
			wantStatus:    QuotaStatusOK,
			wantAllowed:   true,
			wantCurrent:   50,
			wantMaximum:   100,
			wantPercent:   50.0,
			wantRemaining: 50,
		},
		{
			name: "near limit",
			quota: &QuotaInfo{
				MaxPrinters:     100,
				CurrentPrinters: 85,
			},
			tenantID:      "tenant-123",
			count:         1,
			threshold:     80.0,
			wantStatus:    QuotaStatusNearLimit,
			wantAllowed:   true,
			wantCurrent:   85,
			wantMaximum:   100,
			wantPercent:   85.0,
			wantRemaining: 15,
		},
		{
			name: "would exceed quota",
			quota: &QuotaInfo{
				MaxPrinters:     100,
				CurrentPrinters: 99,
			},
			tenantID:      "tenant-123",
			count:         2, // Adding 2 would exceed
			threshold:     80.0,
			wantStatus:    QuotaStatusExceeded,
			wantAllowed:   false,
			wantCurrent:   99,
			wantMaximum:   100,
			wantPercent:   99.0,
			wantRemaining: 1,
		},
		{
			name: "exactly at limit",
			quota: &QuotaInfo{
				MaxPrinters:     100,
				CurrentPrinters: 100,
			},
			tenantID:      "tenant-123",
			count:         1,
			threshold:     80.0,
			wantStatus:    QuotaStatusExceeded,
			wantAllowed:   false,
			wantCurrent:   100,
			wantMaximum:   100,
			wantPercent:   100.0,
			wantRemaining: 0,
		},
		{
			name: "near limit when at threshold",
			quota: &QuotaInfo{
				MaxPrinters:     100,
				CurrentPrinters: 80,
			},
			tenantID:      "tenant-123",
			count:         5,
			threshold:     80.0,
			wantStatus:    QuotaStatusNearLimit,
			wantAllowed:   true,
			wantCurrent:   80,
			wantMaximum:   100,
			wantPercent:   80.0,
			wantRemaining: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockQuotaRepository{quota: tt.quota}
			enforcer := NewQuotaEnforcer(repo)
			enforcer.SetWarningThreshold(tt.threshold)

			ctx := WithTenantID(context.Background(), tt.tenantID)
			result, err := enforcer.CheckPrinterQuota(ctx, tt.count)

			if tt.wantErr != nil {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, ResourcePrinters, result.ResourceType)
			assert.Equal(t, tt.wantStatus, result.Status)
			assert.Equal(t, tt.wantCurrent, result.Current)
			assert.Equal(t, tt.wantMaximum, result.Maximum)
			assert.Equal(t, tt.wantPercent, result.UsagePercent)
			assert.Equal(t, tt.wantRemaining, result.Remaining)
			assert.Equal(t, tt.wantAllowed, result.IsAllowed())
		})
	}
}

func TestQuotaEnforcer_CheckPrinterQuota_NoTenantContext(t *testing.T) {
	repo := &mockQuotaRepository{}
	enforcer := NewQuotaEnforcer(repo)

	ctx := context.Background()
	_, err := enforcer.CheckPrinterQuota(ctx, 1)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoTenantContext)
}

func TestQuotaEnforcer_CheckStorageQuota(t *testing.T) {
	tests := []struct {
		name          string
		quota         *QuotaInfo
		tenantID      string
		bytes         int64
		threshold     float64
		wantStatus    QuotaStatus
		wantAllowed   bool
		wantCurrent   int64
		wantMaximum   int64
		wantPercent   float64
		wantRemaining int64
	}{
		{
			name: "unlimited storage",
			quota: &QuotaInfo{
				MaxStorageGB:     0,                      // Unlimited
				CurrentStorageGB: 5 * 1024 * 1024 * 1024, // 5GB
			},
			tenantID:      "tenant-123",
			bytes:         1024 * 1024 * 1024, // 1GB
			threshold:     80.0,
			wantStatus:    QuotaStatusOK,
			wantAllowed:   true,
			wantCurrent:   5 * 1024 * 1024 * 1024,
			wantMaximum:   -1,
			wantPercent:   0,
			wantRemaining: -1,
		},
		{
			name: "under storage limit",
			quota: &QuotaInfo{
				MaxStorageGB:     10,                     // 10GB
				CurrentStorageGB: 5 * 1024 * 1024 * 1024, // 5GB
			},
			tenantID:      "tenant-123",
			bytes:         1024 * 1024 * 1024, // 1GB
			threshold:     80.0,
			wantStatus:    QuotaStatusOK,
			wantAllowed:   true,
			wantCurrent:   5 * 1024 * 1024 * 1024,
			wantMaximum:   10 * 1024 * 1024 * 1024,
			wantPercent:   50.0,
			wantRemaining: 5 * 1024 * 1024 * 1024,
		},
		{
			name: "near storage limit",
			quota: &QuotaInfo{
				MaxStorageGB:     10,
				CurrentStorageGB: 9 * 1024 * 1024 * 1024, // 9GB
			},
			tenantID:      "tenant-123",
			bytes:         512 * 1024 * 1024, // 512MB
			threshold:     80.0,
			wantStatus:    QuotaStatusNearLimit,
			wantAllowed:   true,
			wantCurrent:   9 * 1024 * 1024 * 1024,
			wantMaximum:   10 * 1024 * 1024 * 1024,
			wantPercent:   90.0,
			wantRemaining: 1 * 1024 * 1024 * 1024,
		},
		{
			name: "would exceed storage quota",
			quota: &QuotaInfo{
				MaxStorageGB:     10,
				CurrentStorageGB: 9*1024*1024*1024 + 500*1024*1024, // 9.5GB
			},
			tenantID:      "tenant-123",
			bytes:         1024 * 1024 * 1024, // 1GB
			threshold:     80.0,
			wantStatus:    QuotaStatusExceeded,
			wantAllowed:   false,
			wantCurrent:   9*1024*1024*1024 + 500*1024*1024,
			wantMaximum:   10 * 1024 * 1024 * 1024,
			wantPercent:   95.0,
			wantRemaining: 512 * 1024 * 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockQuotaRepository{quota: tt.quota}
			enforcer := NewQuotaEnforcer(repo)
			enforcer.SetWarningThreshold(tt.threshold)

			ctx := WithTenantID(context.Background(), tt.tenantID)
			result, err := enforcer.CheckStorageQuota(ctx, tt.bytes)

			require.NoError(t, err)
			assert.Equal(t, ResourceStorage, result.ResourceType)
			assert.Equal(t, tt.wantStatus, result.Status)
			assert.Equal(t, tt.wantAllowed, result.IsAllowed())
		})
	}
}

func TestQuotaEnforcer_CheckJobQuota(t *testing.T) {
	tests := []struct {
		name        string
		quota       *QuotaInfo
		tenantID    string
		count       int
		wantStatus  QuotaStatus
		wantAllowed bool
	}{
		{
			name: "unlimited jobs",
			quota: &QuotaInfo{
				MaxJobsPerMonth: 0,
				CurrentJobs:     1000,
			},
			tenantID:    "tenant-123",
			count:       100,
			wantStatus:  QuotaStatusOK,
			wantAllowed: true,
		},
		{
			name: "under job limit",
			quota: &QuotaInfo{
				MaxJobsPerMonth: 1000,
				CurrentJobs:     500,
			},
			tenantID:    "tenant-123",
			count:       100,
			wantStatus:  QuotaStatusOK,
			wantAllowed: true,
		},
		{
			name: "would exceed job quota",
			quota: &QuotaInfo{
				MaxJobsPerMonth: 1000,
				CurrentJobs:     950,
			},
			tenantID:    "tenant-123",
			count:       100,
			wantStatus:  QuotaStatusExceeded,
			wantAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockQuotaRepository{quota: tt.quota}
			enforcer := NewQuotaEnforcer(repo)

			ctx := WithTenantID(context.Background(), tt.tenantID)
			result, err := enforcer.CheckJobQuota(ctx, tt.count)

			require.NoError(t, err)
			assert.Equal(t, ResourceJobs, result.ResourceType)
			assert.Equal(t, tt.wantStatus, result.Status)
			assert.Equal(t, tt.wantAllowed, result.IsAllowed())
		})
	}
}

func TestQuotaEnforcer_CheckUserQuota(t *testing.T) {
	tests := []struct {
		name        string
		quota       *QuotaInfo
		tenantID    string
		count       int
		wantStatus  QuotaStatus
		wantAllowed bool
	}{
		{
			name: "unlimited users",
			quota: &QuotaInfo{
				MaxUsers:     0,
				CurrentUsers: 100,
			},
			tenantID:    "tenant-123",
			count:       10,
			wantStatus:  QuotaStatusOK,
			wantAllowed: true,
		},
		{
			name: "under user limit",
			quota: &QuotaInfo{
				MaxUsers:     50,
				CurrentUsers: 25,
			},
			tenantID:    "tenant-123",
			count:       5,
			wantStatus:  QuotaStatusOK,
			wantAllowed: true,
		},
		{
			name: "would exceed user quota",
			quota: &QuotaInfo{
				MaxUsers:     50,
				CurrentUsers: 48,
			},
			tenantID:    "tenant-123",
			count:       5,
			wantStatus:  QuotaStatusExceeded,
			wantAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockQuotaRepository{quota: tt.quota}
			enforcer := NewQuotaEnforcer(repo)

			ctx := WithTenantID(context.Background(), tt.tenantID)
			result, err := enforcer.CheckUserQuota(ctx, tt.count)

			require.NoError(t, err)
			assert.Equal(t, ResourceUsers, result.ResourceType)
			assert.Equal(t, tt.wantStatus, result.Status)
			assert.Equal(t, tt.wantAllowed, result.IsAllowed())
		})
	}
}

func TestQuotaEnforcer_RequirePrinterQuota(t *testing.T) {
	tests := []struct {
		name    string
		quota   *QuotaInfo
		count   int
		wantErr bool
	}{
		{
			name: "allowed - under limit",
			quota: &QuotaInfo{
				MaxPrinters:     100,
				CurrentPrinters: 50,
			},
			count:   1,
			wantErr: false,
		},
		{
			name: "allowed - near limit",
			quota: &QuotaInfo{
				MaxPrinters:     100,
				CurrentPrinters: 85,
			},
			count:   1,
			wantErr: false,
		},
		{
			name: "denied - exceeded",
			quota: &QuotaInfo{
				MaxPrinters:     100,
				CurrentPrinters: 100,
			},
			count:   1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockQuotaRepository{quota: tt.quota}
			enforcer := NewQuotaEnforcer(repo)

			ctx := WithTenantID(context.Background(), "tenant-123")
			err := enforcer.RequirePrinterQuota(ctx, tt.count)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "printer quota exceeded")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestQuotaEnforcer_RequireStorageQuota(t *testing.T) {
	tests := []struct {
		name    string
		quota   *QuotaInfo
		bytes   int64
		wantErr bool
	}{
		{
			name: "allowed - under limit",
			quota: &QuotaInfo{
				MaxStorageGB:     10,
				CurrentStorageGB: 5 * 1024 * 1024 * 1024,
			},
			bytes:   1024 * 1024 * 1024,
			wantErr: false,
		},
		{
			name: "denied - exceeded",
			quota: &QuotaInfo{
				MaxStorageGB:     10,
				CurrentStorageGB: 10 * 1024 * 1024 * 1024,
			},
			bytes:   1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockQuotaRepository{quota: tt.quota}
			enforcer := NewQuotaEnforcer(repo)

			ctx := WithTenantID(context.Background(), "tenant-123")
			err := enforcer.RequireStorageQuota(ctx, tt.bytes)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "storage quota exceeded")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestQuotaEnforcer_RequireJobQuota(t *testing.T) {
	tests := []struct {
		name    string
		quota   *QuotaInfo
		count   int
		wantErr bool
	}{
		{
			name: "allowed",
			quota: &QuotaInfo{
				MaxJobsPerMonth: 1000,
				CurrentJobs:     500,
			},
			count:   100,
			wantErr: false,
		},
		{
			name: "denied",
			quota: &QuotaInfo{
				MaxJobsPerMonth: 1000,
				CurrentJobs:     1000,
			},
			count:   1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockQuotaRepository{quota: tt.quota}
			enforcer := NewQuotaEnforcer(repo)

			ctx := WithTenantID(context.Background(), "tenant-123")
			err := enforcer.RequireJobQuota(ctx, tt.count)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "monthly job quota exceeded")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestQuotaEnforcer_RequireUserQuota(t *testing.T) {
	tests := []struct {
		name    string
		quota   *QuotaInfo
		count   int
		wantErr bool
	}{
		{
			name: "allowed",
			quota: &QuotaInfo{
				MaxUsers:     50,
				CurrentUsers: 25,
			},
			count:   5,
			wantErr: false,
		},
		{
			name: "denied",
			quota: &QuotaInfo{
				MaxUsers:     50,
				CurrentUsers: 50,
			},
			count:   1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockQuotaRepository{quota: tt.quota}
			enforcer := NewQuotaEnforcer(repo)

			ctx := WithTenantID(context.Background(), "tenant-123")
			err := enforcer.RequireUserQuota(ctx, tt.count)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "user quota exceeded")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestQuotaEnforcer_CheckAllQuotas(t *testing.T) {
	quota := &QuotaInfo{
		MaxPrinters:      100,
		MaxStorageGB:     10,
		MaxJobsPerMonth:  1000,
		MaxUsers:         50,
		CurrentPrinters:  50,
		CurrentStorageGB: 5 * 1024 * 1024 * 1024,
		CurrentJobs:      500,
		CurrentUsers:     25,
	}

	repo := &mockQuotaRepository{quota: quota}
	enforcer := NewQuotaEnforcer(repo)

	ctx := WithTenantID(context.Background(), "tenant-123")
	results, err := enforcer.CheckAllQuotas(ctx)

	require.NoError(t, err)
	assert.Len(t, results, 4)

	// Check each resource type
	assert.Contains(t, results, ResourcePrinters)
	assert.Contains(t, results, ResourceStorage)
	assert.Contains(t, results, ResourceJobs)
	assert.Contains(t, results, ResourceUsers)

	// All should be allowed
	for _, result := range results {
		assert.True(t, result.IsAllowed())
	}
}

func TestQuotaEnforcer_CheckAllQuotas_Exceeded(t *testing.T) {
	quota := &QuotaInfo{
		MaxPrinters:      100,
		MaxStorageGB:     10,
		MaxJobsPerMonth:  1000,
		MaxUsers:         50,
		CurrentPrinters:  100, // At limit
		CurrentStorageGB: 5 * 1024 * 1024 * 1024,
		CurrentJobs:      500,
		CurrentUsers:     25,
	}

	repo := &mockQuotaRepository{quota: quota}
	enforcer := NewQuotaEnforcer(repo)

	ctx := WithTenantID(context.Background(), "tenant-123")
	results, err := enforcer.CheckAllQuotas(ctx)

	require.NoError(t, err)
	assert.Len(t, results, 4)

	// Printers at limit with count=0 returns near_limit (100% usage >= 80% threshold)
	assert.Equal(t, QuotaStatusNearLimit, results[ResourcePrinters].Status)
	assert.True(t, results[ResourcePrinters].IsAllowed())
}

func TestQuotaEnforcer_RecordPrinterUsage(t *testing.T) {
	repo := &mockQuotaRepository{}
	enforcer := NewQuotaEnforcer(repo)

	ctx := WithTenantID(context.Background(), "tenant-123")
	err := enforcer.RecordPrinterUsage(ctx, 1)

	require.NoError(t, err)
}

func TestQuotaEnforcer_RecordStorageUsage(t *testing.T) {
	repo := &mockQuotaRepository{}
	enforcer := NewQuotaEnforcer(repo)

	ctx := WithTenantID(context.Background(), "tenant-123")
	err := enforcer.RecordStorageUsage(ctx, 1024*1024*1024)

	require.NoError(t, err)
}

func TestQuotaEnforcer_RecordJobUsage(t *testing.T) {
	repo := &mockQuotaRepository{}
	enforcer := NewQuotaEnforcer(repo)

	ctx := WithTenantID(context.Background(), "tenant-123")
	err := enforcer.RecordJobUsage(ctx, 1)

	require.NoError(t, err)
}

func TestQuotaEnforcer_RecordUserUsage(t *testing.T) {
	repo := &mockQuotaRepository{}
	enforcer := NewQuotaEnforcer(repo)

	ctx := WithTenantID(context.Background(), "tenant-123")
	err := enforcer.RecordUserUsage(ctx, 1)

	require.NoError(t, err)
}

func TestQuotaEnforcer_Usage_NoTenantContext(t *testing.T) {
	repo := &mockQuotaRepository{}
	enforcer := NewQuotaEnforcer(repo)

	ctx := context.Background()

	// All usage recording methods should fail without tenant context
	err := enforcer.RecordPrinterUsage(ctx, 1)
	assert.Error(t, err)

	err = enforcer.RecordStorageUsage(ctx, 1024)
	assert.Error(t, err)

	err = enforcer.RecordJobUsage(ctx, 1)
	assert.Error(t, err)

	err = enforcer.RecordUserUsage(ctx, 1)
	assert.Error(t, err)
}

func TestQuotaEnforcer_GetQuotaError(t *testing.T) {
	tests := []struct {
		name    string
		getErr  error
		wantErr error
	}{
		{
			name:    "repository error",
			getErr:  errors.New("database error"),
			wantErr: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockQuotaRepository{getErr: tt.getErr}
			enforcer := NewQuotaEnforcer(repo)

			ctx := WithTenantID(context.Background(), "tenant-123")
			_, err := enforcer.CheckPrinterQuota(ctx, 1)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr.Error())
		})
	}
}

func TestQuotaCheckResult_IsAllowed(t *testing.T) {
	tests := []struct {
		name   string
		status QuotaStatus
		want   bool
	}{
		{
			name:   "OK status is allowed",
			status: QuotaStatusOK,
			want:   true,
		},
		{
			name:   "NearLimit status is allowed",
			status: QuotaStatusNearLimit,
			want:   true,
		},
		{
			name:   "Exceeded status is not allowed",
			status: QuotaStatusExceeded,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &QuotaCheckResult{Status: tt.status}
			assert.Equal(t, tt.want, result.IsAllowed())
		})
	}
}

func TestResourceTypeValues(t *testing.T) {
	tests := []struct {
		resourceType ResourceType
		wantVal      string
	}{
		{ResourcePrinters, "printers"},
		{ResourceStorage, "storage"},
		{ResourceJobs, "jobs"},
		{ResourceUsers, "users"},
	}

	for _, tt := range tests {
		t.Run(tt.wantVal, func(t *testing.T) {
			assert.Equal(t, tt.wantVal, string(tt.resourceType))
		})
	}
}

func TestQuotaStatusValues(t *testing.T) {
	tests := []struct {
		status  QuotaStatus
		wantVal string
	}{
		{QuotaStatusOK, "ok"},
		{QuotaStatusNearLimit, "near_limit"},
		{QuotaStatusExceeded, "exceeded"},
	}

	for _, tt := range tests {
		t.Run(tt.wantVal, func(t *testing.T) {
			assert.Equal(t, tt.wantVal, string(tt.status))
		})
	}
}

func TestQuotaExceededError(t *testing.T) {
	err := ErrQuotaExceeded
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "quota exceeded")
}

func TestQuotaEnforcer_CustomWarningThreshold(t *testing.T) {
	quota := &QuotaInfo{
		MaxPrinters:     100,
		CurrentPrinters: 75,
	}

	repo := &mockQuotaRepository{quota: quota}
	enforcer := NewQuotaEnforcer(repo)
	enforcer.SetWarningThreshold(70.0) // Lower threshold

	ctx := WithTenantID(context.Background(), "tenant-123")
	result, err := enforcer.CheckPrinterQuota(ctx, 1)

	require.NoError(t, err)
	// 75% is above 70% threshold
	assert.Equal(t, QuotaStatusNearLimit, result.Status)
}

func TestQuotaEnforcer_CheckPrinterQuota_ZeroCount(t *testing.T) {
	quota := &QuotaInfo{
		MaxPrinters:     100,
		CurrentPrinters: 85,
	}

	repo := &mockQuotaRepository{quota: quota}
	enforcer := NewQuotaEnforcer(repo)

	ctx := WithTenantID(context.Background(), "tenant-123")
	result, err := enforcer.CheckPrinterQuota(ctx, 0)

	require.NoError(t, err)
	// With 0 count, current (85) < max (100), so should be near limit
	assert.Equal(t, QuotaStatusNearLimit, result.Status)
	assert.True(t, result.IsAllowed())
}

func TestQuotaEnforcer_CheckStorageQuota_PreciseBytes(t *testing.T) {
	// Test exact byte calculations
	oneGB := int64(1024 * 1024 * 1024)
	quota := &QuotaInfo{
		MaxStorageGB:     10,
		CurrentStorageGB: 7 * oneGB,
	}

	repo := &mockQuotaRepository{quota: quota}
	enforcer := NewQuotaEnforcer(repo)

	ctx := WithTenantID(context.Background(), "tenant-123")
	result, err := enforcer.CheckStorageQuota(ctx, 2*oneGB)

	require.NoError(t, err)
	// 7GB + 2GB = 9GB < 10GB, should be OK
	assert.Equal(t, QuotaStatusOK, result.Status)
	assert.Equal(t, float64(70), result.UsagePercent)
	assert.Equal(t, 3*oneGB, result.Remaining)
}

func TestQuotaEnforcer_MultipleQuotaChecksConsistency(t *testing.T) {
	quota := &QuotaInfo{
		MaxPrinters:      100,
		MaxStorageGB:     10,
		MaxJobsPerMonth:  1000,
		MaxUsers:         50,
		CurrentPrinters:  50,
		CurrentStorageGB: 5 * 1024 * 1024 * 1024,
		CurrentJobs:      500,
		CurrentUsers:     25,
	}

	repo := &mockQuotaRepository{quota: quota}
	enforcer := NewQuotaEnforcer(repo)

	ctx := WithTenantID(context.Background(), "tenant-123")

	// Multiple checks should return consistent results
	result1, _ := enforcer.CheckPrinterQuota(ctx, 1)
	result2, _ := enforcer.CheckPrinterQuota(ctx, 1)

	assert.Equal(t, result1.Current, result2.Current)
	assert.Equal(t, result1.Maximum, result2.Maximum)
	assert.Equal(t, result1.UsagePercent, result2.UsagePercent)
}

func TestQuotaEnforcer_Formatting(t *testing.T) {
	// Test that error messages are properly formatted
	quota := &QuotaInfo{
		MaxPrinters:     100,
		CurrentPrinters: 100,
	}

	repo := &mockQuotaRepository{quota: quota}
	enforcer := NewQuotaEnforcer(repo)

	ctx := WithTenantID(context.Background(), "tenant-123")
	err := enforcer.RequirePrinterQuota(ctx, 1)

	require.Error(t, err)
	errMsg := fmt.Sprintf("%v", err)
	assert.Contains(t, errMsg, "100/100")
}

func TestQuotaEnforcer_NegativeDeltaRecording(t *testing.T) {
	// Test recording negative usage (reducing usage)
	repo := &mockQuotaRepository{}
	enforcer := NewQuotaEnforcer(repo)

	ctx := WithTenantID(context.Background(), "tenant-123")

	// Should not error on negative delta
	err := enforcer.RecordPrinterUsage(ctx, -1)
	require.NoError(t, err)
}
