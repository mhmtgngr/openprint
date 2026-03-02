// Package repository provides data access layer for auth service.
// This file contains quota repository for resource quota management.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
	multitenant "github.com/openprint/openprint/internal/multi-tenant"
)

// QuotaConfig represents the quota configuration for a tenant.
type QuotaConfig struct {
	ID           string        `json:"id" db:"id"`
	TenantID     string        `json:"tenant_id" db:"tenant_id"`
	MaxPrinters  int32         `json:"max_printers" db:"max_printers"`
	MaxStorageGB int32         `json:"max_storage_gb" db:"max_storage_gb"`
	MaxJobsPerMonth int32      `json:"max_jobs_per_month" db:"max_jobs_per_month"`
	MaxUsers     int32         `json:"max_users" db:"max_users"`
	AlertThreshold int32       `json:"alert_threshold" db:"alert_threshold"`
	CreatedAt    time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at" db:"updated_at"`
}

// QuotaUsage represents the current resource usage for a tenant.
type QuotaUsage struct {
	ID              string    `json:"id" db:"id"`
	TenantID        string    `json:"tenant_id" db:"tenant_id"`
	PrintersCount   int32     `json:"printers_count" db:"printers_count"`
	StorageUsedGB   int64     `json:"storage_used_gb" db:"storage_used_gb"` // Stored as bytes, reported as GB
	JobsThisMonth   int32     `json:"jobs_this_month" db:"jobs_this_month"`
	UsersCount      int32     `json:"users_count" db:"users_count"`
	Month           time.Time `json:"month" db:"month"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// QuotaRepository provides data access for quota management.
type QuotaRepository struct {
	db *pgxpool.Pool
}

// NewQuotaRepository creates a new quota repository.
func NewQuotaRepository(db *pgxpool.Pool) *QuotaRepository {
	return &QuotaRepository{db: db}
}

// CreateConfig creates a new quota configuration for a tenant.
func (r *QuotaRepository) CreateConfig(ctx context.Context, config *QuotaConfig) error {
	config.ID = uuid.New().String()
	config.CreatedAt = time.Now().UTC()
	config.UpdatedAt = config.CreatedAt

	// Set default alert threshold to 80%
	if config.AlertThreshold == 0 {
		config.AlertThreshold = 80
	}

	query := `
		INSERT INTO quota_configs (id, tenant_id, max_printers, max_storage_gb, max_jobs_per_month, max_users, alert_threshold, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (tenant_id) DO UPDATE SET
			max_printers = EXCLUDED.max_printers,
			max_storage_gb = EXCLUDED.max_storage_gb,
			max_jobs_per_month = EXCLUDED.max_jobs_per_month,
			max_users = EXCLUDED.max_users,
			alert_threshold = EXCLUDED.alert_threshold,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.db.Exec(ctx, query,
		config.ID, config.TenantID, config.MaxPrinters, config.MaxStorageGB,
		config.MaxJobsPerMonth, config.MaxUsers, config.AlertThreshold,
		config.CreatedAt, config.UpdatedAt,
	)

	if err != nil {
		return apperrors.Wrap(err, "failed to create quota config", 500)
	}

	return nil
}

// GetConfig retrieves the quota configuration for a tenant.
func (r *QuotaRepository) GetConfig(ctx context.Context, tenantID string) (*QuotaConfig, error) {
	query := `
		SELECT id, tenant_id, max_printers, max_storage_gb, max_jobs_per_month, max_users, alert_threshold, created_at, updated_at
		FROM quota_configs
		WHERE tenant_id = $1
	`

	config := &QuotaConfig{}
	err := r.db.QueryRow(ctx, query, tenantID).Scan(
		&config.ID, &config.TenantID, &config.MaxPrinters, &config.MaxStorageGB,
		&config.MaxJobsPerMonth, &config.MaxUsers, &config.AlertThreshold,
		&config.CreatedAt, &config.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Return default quotas if none configured
			return r.getDefaultConfig(tenantID), nil
		}
		return nil, apperrors.Wrap(err, "failed to get quota config", 500)
	}

	return config, nil
}

// getDefaultConfig returns default quota configuration.
func (r *QuotaRepository) getDefaultConfig(tenantID string) *QuotaConfig {
	now := time.Now().UTC()
	return &QuotaConfig{
		ID:              uuid.New().String(),
		TenantID:        tenantID,
		MaxPrinters:     100,
		MaxStorageGB:    100,
		MaxJobsPerMonth: 10000,
		MaxUsers:        50,
		AlertThreshold:  80,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// UpdateConfig updates the quota configuration for a tenant.
func (r *QuotaRepository) UpdateConfig(ctx context.Context, config *QuotaConfig) error {
	config.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE quota_configs
		SET max_printers = $2, max_storage_gb = $3, max_jobs_per_month = $4, max_users = $5, alert_threshold = $6, updated_at = $7
		WHERE tenant_id = $1
	`

	result, err := r.db.Exec(ctx, query,
		config.TenantID, config.MaxPrinters, config.MaxStorageGB,
		config.MaxJobsPerMonth, config.MaxUsers, config.AlertThreshold, config.UpdatedAt,
	)

	if err != nil {
		return apperrors.Wrap(err, "failed to update quota config", 500)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		// Config doesn't exist, create it
		return r.CreateConfig(ctx, config)
	}

	return nil
}

// GetUsage retrieves the current usage for a tenant.
func (r *QuotaRepository) GetUsage(ctx context.Context, tenantID string) (*QuotaUsage, error) {
	// Get current month
	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	query := `
		SELECT id, tenant_id, printers_count, storage_used_gb, jobs_this_month, users_count, month, updated_at
		FROM quota_usage
		WHERE tenant_id = $1 AND month = $2
	`

	usage := &QuotaUsage{}
	err := r.db.QueryRow(ctx, query, tenantID, monthStart).Scan(
		&usage.ID, &usage.TenantID, &usage.PrintersCount, &usage.StorageUsedGB,
		&usage.JobsThisMonth, &usage.UsersCount, &usage.Month, &usage.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Create initial usage record
			return r.initializeUsage(ctx, tenantID, monthStart)
		}
		return nil, apperrors.Wrap(err, "failed to get quota usage", 500)
	}

	return usage, nil
}

// initializeUsage creates an initial usage record for a tenant.
func (r *QuotaRepository) initializeUsage(ctx context.Context, tenantID string, month time.Time) (*QuotaUsage, error) {
	usage := &QuotaUsage{
		ID:            uuid.New().String(),
		TenantID:      tenantID,
		Month:         month,
		PrintersCount: 0,
		StorageUsedGB: 0,
		JobsThisMonth: 0,
		UsersCount:    0,
		UpdatedAt:     time.Now().UTC(),
	}

	// Count actual resources
	countQuery := `
		SELECT
			(SELECT COUNT(*) FROM printers WHERE tenant_id = $1 AND deleted_at IS NULL),
			(SELECT COALESCE(SUM(page_count * 0.00001), 0) FROM print_jobs WHERE tenant_id = $1 AND created_at >= $2),
			(SELECT COUNT(*) FROM print_jobs WHERE tenant_id = $1 AND created_at >= $2),
			(SELECT COUNT(*) FROM organization_users WHERE tenant_id = $1 AND deleted_at IS NULL)
	`

	err := r.db.QueryRow(ctx, countQuery, tenantID, month).Scan(
		&usage.PrintersCount, &usage.StorageUsedGB, &usage.JobsThisMonth, &usage.UsersCount,
	)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to initialize usage", 500)
	}

	// Convert storage from approximate GB to bytes
	usage.StorageUsedGB = usage.StorageUsedGB * 1024 * 1024 * 1024

	insertQuery := `
		INSERT INTO quota_usage (id, tenant_id, printers_count, storage_used_gb, jobs_this_month, users_count, month, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = r.db.Exec(ctx, insertQuery,
		usage.ID, usage.TenantID, usage.PrintersCount, usage.StorageUsedGB,
		usage.JobsThisMonth, usage.UsersCount, usage.Month, usage.UpdatedAt,
	)

	if err != nil {
		return nil, apperrors.Wrap(err, "failed to create usage record", 500)
	}

	return usage, nil
}

// UpdateUsage updates a single usage counter for a tenant.
func (r *QuotaRepository) UpdateUsage(ctx context.Context, tenantID string, resourceType multitenant.ResourceType, delta int64) error {
	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	var column string
	switch resourceType {
	case multitenant.ResourcePrinters:
		column = "printers_count"
	case multitenant.ResourceStorage:
		column = "storage_used_gb"
	case multitenant.ResourceJobs:
		column = "jobs_this_month"
	case multitenant.ResourceUsers:
		column = "users_count"
	default:
		return apperrors.New("invalid resource type", 400)
	}

	query := fmt.Sprintf(`
		INSERT INTO quota_usage (id, tenant_id, %s, month, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tenant_id, month) DO UPDATE SET
			%s = GREATEST(quota_usage.%s + EXCLUDED.%s, 0),
			updated_at = EXCLUDED.updated_at
	`, column, column, column, column)

	_, err := r.db.Exec(ctx, query, uuid.New().String(), tenantID, delta, monthStart, now)
	if err != nil {
		return apperrors.Wrap(err, "failed to update usage", 500)
	}

	return nil
}

// GetQuotaInfo returns combined quota configuration and usage information.
func (r *QuotaRepository) GetQuotaInfo(ctx context.Context, tenantID string) (*multitenant.QuotaInfo, error) {
	config, err := r.GetConfig(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	usage, err := r.GetUsage(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return &multitenant.QuotaInfo{
		MaxPrinters:      config.MaxPrinters,
		MaxStorageGB:     config.MaxStorageGB,
		MaxJobsPerMonth:  config.MaxJobsPerMonth,
		MaxUsers:         config.MaxUsers,
		CurrentPrinters:  usage.PrintersCount,
		CurrentStorageGB: usage.StorageUsedGB,
		CurrentJobs:      usage.JobsThisMonth,
		CurrentUsers:     usage.UsersCount,
	}, nil
}

// GetTenantUsageForMonth retrieves usage for a specific tenant and month.
func (r *QuotaRepository) GetTenantUsageForMonth(ctx context.Context, tenantID string, month time.Time) (*QuotaUsage, error) {
	monthStart := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)

	query := `
		SELECT id, tenant_id, printers_count, storage_used_gb, jobs_this_month, users_count, month, updated_at
		FROM quota_usage
		WHERE tenant_id = $1 AND month = $2
	`

	usage := &QuotaUsage{}
	err := r.db.QueryRow(ctx, query, tenantID, monthStart).Scan(
		&usage.ID, &usage.TenantID, &usage.PrintersCount, &usage.StorageUsedGB,
		&usage.JobsThisMonth, &usage.UsersCount, &usage.Month, &usage.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get usage for month", 500)
	}

	return usage, nil
}

// ListQuotaConfigs retrieves quota configurations for multiple tenants.
func (r *QuotaRepository) ListQuotaConfigs(ctx context.Context, tenantIDs []string) ([]*QuotaConfig, error) {
	if len(tenantIDs) == 0 {
		return []*QuotaConfig{}, nil
	}

	query := `
		SELECT id, tenant_id, max_printers, max_storage_gb, max_jobs_per_month, max_users, alert_threshold, created_at, updated_at
		FROM quota_configs
		WHERE tenant_id = ANY($1)
	`

	rows, err := r.db.Query(ctx, query, tenantIDs)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to list quota configs", 500)
	}
	defer rows.Close()

	configs := []*QuotaConfig{}
	for rows.Next() {
		config := &QuotaConfig{}
		err := rows.Scan(
			&config.ID, &config.TenantID, &config.MaxPrinters, &config.MaxStorageGB,
			&config.MaxJobsPerMonth, &config.MaxUsers, &config.AlertThreshold,
			&config.CreatedAt, &config.UpdatedAt,
		)
		if err != nil {
			return nil, apperrors.Wrap(err, "failed to scan quota config", 500)
		}
		configs = append(configs, config)
	}

	return configs, nil
}

// DeleteConfig deletes quota configuration for a tenant.
func (r *QuotaRepository) DeleteConfig(ctx context.Context, tenantID string) error {
	query := `DELETE FROM quota_configs WHERE tenant_id = $1`
	_, err := r.db.Exec(ctx, query, tenantID)
	return err
}

// ResetMonthlyUsage resets job counters for a new month.
// This should be called by a scheduled job.
func (r *QuotaRepository) ResetMonthlyUsage(ctx context.Context, tenantID string) error {
	now := time.Now().UTC()
	newMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	query := `
		INSERT INTO quota_usage (id, tenant_id, printers_count, storage_used_gb, jobs_this_month, users_count, month, updated_at)
		SELECT
			$1,
			tenant_id,
			printers_count,
			storage_used_gb,
			0,
			users_count,
			$2,
			$3
		FROM quota_usage
		WHERE tenant_id = $4 AND month = $5
		ON CONFLICT (tenant_id, month) DO UPDATE SET
			jobs_this_month = 0,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.db.Exec(ctx, query, uuid.New().String(), newMonthStart, now, tenantID, newMonthStart.AddDate(0, -1, 1))
	return err
}
