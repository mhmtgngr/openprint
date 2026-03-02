// Package multitenant provides multi-tenancy support for OpenPrint services.
// This file contains quota enforcement logic.
package multitenant

import (
	"context"
	"fmt"

	apperrors "github.com/openprint/openprint/internal/shared/errors"
)

// ResourceType represents a quota-controlled resource type.
type ResourceType string

const (
	// ResourcePrinters represents the printer resource quota.
	ResourcePrinters ResourceType = "printers"
	// ResourceStorage represents the storage quota in bytes.
	ResourceStorage ResourceType = "storage"
	// ResourceJobs represents the monthly job throughput quota.
	ResourceJobs ResourceType = "jobs"
	// ResourceUsers represents the user count quota.
	ResourceUsers ResourceType = "users"
)

// QuotaStatus represents the status of a quota check.
type QuotaStatus string

const (
	// QuotaStatusOK means the quota is available.
	QuotaStatusOK QuotaStatus = "ok"
	// QuotaStatusNearLimit means the quota is above the warning threshold.
	QuotaStatusNearLimit QuotaStatus = "near_limit"
	// QuotaStatusExceeded means the quota has been exceeded.
	QuotaStatusExceeded QuotaStatus = "exceeded"
)

// QuotaCheckResult represents the result of a quota check.
type QuotaCheckResult struct {
	ResourceType  ResourceType
	Status        QuotaStatus
	Current       int64
	Maximum       int64
	UsagePercent  float64
	Remaining     int64
}

// IsAllowed returns true if the operation is allowed under quota.
func (r *QuotaCheckResult) IsAllowed() bool {
	return r.Status == QuotaStatusOK || r.Status == QuotaStatusNearLimit
}

// QuotaRepository defines the interface for quota data access.
type QuotaRepository interface {
	// GetQuota retrieves the quota configuration for a tenant.
	GetQuota(ctx context.Context, tenantID string) (*QuotaInfo, error)
	// GetCurrentUsage retrieves the current usage for a tenant.
	GetCurrentUsage(ctx context.Context, tenantID string) (*QuotaInfo, error)
	// UpdateUsage updates the usage counter for a resource.
	UpdateUsage(ctx context.Context, tenantID string, resourceType ResourceType, delta int64) error
}

// QuotaEnforcer handles quota checking and enforcement.
type QuotaEnforcer struct {
	repo           QuotaRepository
	warningThreshold float64 // Percentage threshold for warnings (default 0.8 = 80%)
}

// NewQuotaEnforcer creates a new quota enforcer.
func NewQuotaEnforcer(repo QuotaRepository) *QuotaEnforcer {
	return &QuotaEnforcer{
		repo:             repo,
		warningThreshold: 80.0,
	}
}

// SetWarningThreshold sets the warning threshold percentage.
func (e *QuotaEnforcer) SetWarningThreshold(threshold float64) {
	e.warningThreshold = threshold
}

// CheckPrinterQuota checks if a tenant can add more printers.
func (e *QuotaEnforcer) CheckPrinterQuota(ctx context.Context, count int) (*QuotaCheckResult, error) {
	tenantID, err := GetTenantID(ctx)
	if err != nil {
		return nil, err
	}

	quota, err := e.repo.GetQuota(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	current := int64(quota.CurrentPrinters)
	maximum := int64(quota.MaxPrinters)

	if maximum <= 0 {
		// Unlimited quota
		return &QuotaCheckResult{
			ResourceType: ResourcePrinters,
			Status:       QuotaStatusOK,
			Current:      current,
			Maximum:      -1, // -1 indicates unlimited
			Remaining:    -1,
		}, nil
	}

	usagePercent := float64(current) / float64(maximum) * 100
	remaining := maximum - current

	status := QuotaStatusOK
	if current+int64(count) > maximum {
		status = QuotaStatusExceeded
	} else if usagePercent >= e.warningThreshold {
		status = QuotaStatusNearLimit
	}

	return &QuotaCheckResult{
		ResourceType: ResourcePrinters,
		Status:       status,
		Current:      current,
		Maximum:      maximum,
		UsagePercent: usagePercent,
		Remaining:    remaining,
	}, nil
}

// CheckStorageQuota checks if a tenant can store more data.
func (e *QuotaEnforcer) CheckStorageQuota(ctx context.Context, bytes int64) (*QuotaCheckResult, error) {
	tenantID, err := GetTenantID(ctx)
	if err != nil {
		return nil, err
	}

	quota, err := e.repo.GetQuota(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	current := quota.CurrentStorageGB
	maximum := int64(quota.MaxStorageGB) * 1024 * 1024 * 1024 // Convert GB to bytes

	if maximum <= 0 {
		// Unlimited quota
		return &QuotaCheckResult{
			ResourceType: ResourceStorage,
			Status:       QuotaStatusOK,
			Current:      current,
			Maximum:      -1,
			Remaining:    -1,
		}, nil
	}

	usagePercent := float64(current) / float64(maximum) * 100
	remaining := maximum - current

	status := QuotaStatusOK
	if current+bytes > maximum {
		status = QuotaStatusExceeded
	} else if usagePercent >= e.warningThreshold {
		status = QuotaStatusNearLimit
	}

	return &QuotaCheckResult{
		ResourceType: ResourceStorage,
		Status:       status,
		Current:      current,
		Maximum:      maximum,
		UsagePercent: usagePercent,
		Remaining:    remaining,
	}, nil
}

// CheckJobQuota checks if a tenant can submit more jobs this month.
func (e *QuotaEnforcer) CheckJobQuota(ctx context.Context, count int) (*QuotaCheckResult, error) {
	tenantID, err := GetTenantID(ctx)
	if err != nil {
		return nil, err
	}

	quota, err := e.repo.GetQuota(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	current := int64(quota.CurrentJobs)
	maximum := int64(quota.MaxJobsPerMonth)

	if maximum <= 0 {
		// Unlimited quota
		return &QuotaCheckResult{
			ResourceType: ResourceJobs,
			Status:       QuotaStatusOK,
			Current:      current,
			Maximum:      -1,
			Remaining:    -1,
		}, nil
	}

	usagePercent := float64(current) / float64(maximum) * 100
	remaining := maximum - current

	status := QuotaStatusOK
	if current+int64(count) > maximum {
		status = QuotaStatusExceeded
	} else if usagePercent >= e.warningThreshold {
		status = QuotaStatusNearLimit
	}

	return &QuotaCheckResult{
		ResourceType: ResourceJobs,
		Status:       status,
		Current:      current,
		Maximum:      maximum,
		UsagePercent: usagePercent,
		Remaining:    remaining,
	}, nil
}

// CheckUserQuota checks if a tenant can add more users.
func (e *QuotaEnforcer) CheckUserQuota(ctx context.Context, count int) (*QuotaCheckResult, error) {
	tenantID, err := GetTenantID(ctx)
	if err != nil {
		return nil, err
	}

	quota, err := e.repo.GetQuota(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	current := int64(quota.CurrentUsers)
	maximum := int64(quota.MaxUsers)

	if maximum <= 0 {
		// Unlimited quota
		return &QuotaCheckResult{
			ResourceType: ResourceUsers,
			Status:       QuotaStatusOK,
			Current:      current,
			Maximum:      -1,
			Remaining:    -1,
		}, nil
	}

	usagePercent := float64(current) / float64(maximum) * 100
	remaining := maximum - current

	status := QuotaStatusOK
	if current+int64(count) > maximum {
		status = QuotaStatusExceeded
	} else if usagePercent >= e.warningThreshold {
		status = QuotaStatusNearLimit
	}

	return &QuotaCheckResult{
		ResourceType: ResourceUsers,
		Status:       status,
		Current:      current,
		Maximum:      maximum,
		UsagePercent: usagePercent,
		Remaining:    remaining,
	}, nil
}

// RequirePrinterQuota checks printer quota and returns an error if exceeded.
func (e *QuotaEnforcer) RequirePrinterQuota(ctx context.Context, count int) error {
	result, err := e.CheckPrinterQuota(ctx, count)
	if err != nil {
		return err
	}

	if !result.IsAllowed() {
		return apperrors.New(
			fmt.Sprintf("printer quota exceeded: %d/%d printers used", result.Current, result.Maximum),
			apperrors.HTTPStatus(ErrQuotaExceeded),
		).WithCode("QUOTA_EXCEEDED").WithDetail("resource", "printers")
	}

	return nil
}

// RequireStorageQuota checks storage quota and returns an error if exceeded.
func (e *QuotaEnforcer) RequireStorageQuota(ctx context.Context, bytes int64) error {
	result, err := e.CheckStorageQuota(ctx, bytes)
	if err != nil {
		return err
	}

	if !result.IsAllowed() {
		return apperrors.New(
			fmt.Sprintf("storage quota exceeded: %.2f/%.2f GB used",
				float64(result.Current)/(1024*1024*1024),
				float64(result.Maximum)/(1024*1024*1024)),
			apperrors.HTTPStatus(ErrQuotaExceeded),
		).WithCode("QUOTA_EXCEEDED").WithDetail("resource", "storage")
	}

	return nil
}

// RequireJobQuota checks job quota and returns an error if exceeded.
func (e *QuotaEnforcer) RequireJobQuota(ctx context.Context, count int) error {
	result, err := e.CheckJobQuota(ctx, count)
	if err != nil {
		return err
	}

	if !result.IsAllowed() {
		return apperrors.New(
			fmt.Sprintf("monthly job quota exceeded: %d/%d jobs used", result.Current, result.Maximum),
			apperrors.HTTPStatus(ErrQuotaExceeded),
		).WithCode("QUOTA_EXCEEDED").WithDetail("resource", "jobs")
	}

	return nil
}

// RequireUserQuota checks user quota and returns an error if exceeded.
func (e *QuotaEnforcer) RequireUserQuota(ctx context.Context, count int) error {
	result, err := e.CheckUserQuota(ctx, count)
	if err != nil {
		return err
	}

	if !result.IsAllowed() {
		return apperrors.New(
			fmt.Sprintf("user quota exceeded: %d/%d users", result.Current, result.Maximum),
			apperrors.HTTPStatus(ErrQuotaExceeded),
		).WithCode("QUOTA_EXCEEDED").WithDetail("resource", "users")
	}

	return nil
}

// CheckAllQuotas checks all quota types and returns the results.
func (e *QuotaEnforcer) CheckAllQuotas(ctx context.Context) (map[ResourceType]*QuotaCheckResult, error) {
	printerResult, err := e.CheckPrinterQuota(ctx, 0)
	if err != nil {
		return nil, err
	}

	storageResult, err := e.CheckStorageQuota(ctx, 0)
	if err != nil {
		return nil, err
	}

	jobResult, err := e.CheckJobQuota(ctx, 0)
	if err != nil {
		return nil, err
	}

	userResult, err := e.CheckUserQuota(ctx, 0)
	if err != nil {
		return nil, err
	}

	return map[ResourceType]*QuotaCheckResult{
		ResourcePrinters: printerResult,
		ResourceStorage:  storageResult,
		ResourceJobs:     jobResult,
		ResourceUsers:    userResult,
	}, nil
}

// RecordPrinterUsage records a change in printer count.
func (e *QuotaEnforcer) RecordPrinterUsage(ctx context.Context, delta int64) error {
	tenantID, err := GetTenantID(ctx)
	if err != nil {
		return err
	}
	return e.repo.UpdateUsage(ctx, tenantID, ResourcePrinters, delta)
}

// RecordStorageUsage records a change in storage usage.
func (e *QuotaEnforcer) RecordStorageUsage(ctx context.Context, delta int64) error {
	tenantID, err := GetTenantID(ctx)
	if err != nil {
		return err
	}
	return e.repo.UpdateUsage(ctx, tenantID, ResourceStorage, delta)
}

// RecordJobUsage records a change in job count.
func (e *QuotaEnforcer) RecordJobUsage(ctx context.Context, delta int64) error {
	tenantID, err := GetTenantID(ctx)
	if err != nil {
		return err
	}
	return e.repo.UpdateUsage(ctx, tenantID, ResourceJobs, delta)
}

// RecordUserUsage records a change in user count.
func (e *QuotaEnforcer) RecordUserUsage(ctx context.Context, delta int64) error {
	tenantID, err := GetTenantID(ctx)
	if err != nil {
		return err
	}
	return e.repo.UpdateUsage(ctx, tenantID, ResourceUsers, delta)
}

// QuotaExceededError is the error for quota exceeded.
var ErrQuotaExceeded = apperrors.New("quota exceeded", 429)
