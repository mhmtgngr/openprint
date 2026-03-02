// Package repository provides job data access with tenant support.
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// JobRepositoryWithTenant extends JobRepository with tenant-aware methods.
type JobRepositoryWithTenant struct {
	db *pgxpool.Pool
}

// NewJobRepositoryWithTenant creates a new tenant-aware job repository.
func NewJobRepositoryWithTenant(db *pgxpool.Pool) *JobRepositoryWithTenant {
	return &JobRepositoryWithTenant{db: db}
}

// ListByTenant retrieves jobs for a specific tenant with pagination.
func (r *JobRepositoryWithTenant) ListByTenant(ctx context.Context, tenantID string, limit, offset int, status string) ([]*Job, int, error) {
	whereClause := "WHERE tenant_id = $1"
	args := []interface{}{tenantID}
	argCount := 2

	if status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
		argCount++
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM print_jobs " + whereClause
	var total int
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count jobs by tenant: %w", err)
	}

	// Get paginated results
	query := `
		SELECT id, tenant_id, organization_id, document_id, printer_id, user_id, status,
		       page_count, color, duplex, paper_size, copies, options, created_at, updated_at, completed_at
		FROM print_jobs
		` + whereClause + `
		ORDER BY created_at DESC
		LIMIT $` + fmt.Sprint(argCount) + ` OFFSET $` + fmt.Sprint(argCount+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list jobs by tenant: %w", err)
	}
	defer rows.Close()

	jobs := []*Job{}
	for rows.Next() {
		job, err := r.scanJob(rows)
		if err != nil {
			return nil, 0, err
		}
		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate jobs: %w", err)
	}

	return jobs, total, nil
}

// CreateWithTenant creates a job with explicit tenant context.
func (r *JobRepositoryWithTenant) CreateWithTenant(ctx context.Context, job *Job, tenantID string) error {
	// Set tenant context for RLS
	if _, err := r.db.Exec(ctx, "SET LOCAL app.tenant_id = $1", tenantID); err != nil {
		return fmt.Errorf("set tenant context: %w", err)
	}

	defer func() {
		_, _ = r.db.Exec(ctx, "SET LOCAL app.tenant_id = NULL")
	}()

	now := time.Now().UTC()
	job.CreatedAt = now
	job.UpdatedAt = now
	job.TenantID = tenantID
	job.OrganizationID = tenantID

	query := `
		INSERT INTO print_jobs (
			id, tenant_id, organization_id, document_id, printer_id, user_id, status,
			page_count, color, duplex, paper_size, copies, options, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id
	`

	err := r.db.QueryRow(ctx, query,
		job.ID,
		job.TenantID,
		job.OrganizationID,
		job.DocumentID,
		job.PrinterID,
		job.UserID,
		job.Status,
		job.PageCount,
		job.Color,
		job.Duplex,
		job.PaperSize,
		job.Copies,
		job.Options,
		job.CreatedAt,
		job.UpdatedAt,
	).Scan(&job.ID)

	if err != nil {
		return fmt.Errorf("create job with tenant: %w", err)
	}

	return nil
}

// FindByTenant retrieves a job by ID within a tenant context.
func (r *JobRepositoryWithTenant) FindByTenant(ctx context.Context, jobID, tenantID string) (*Job, error) {
	// Set tenant context for RLS
	if _, err := r.db.Exec(ctx, "SET LOCAL app.tenant_id = $1", tenantID); err != nil {
		return nil, fmt.Errorf("set tenant context: %w", err)
	}

	defer func() {
		_, _ = r.db.Exec(ctx, "SET LOCAL app.tenant_id = NULL")
	}()

	query := `
		SELECT id, tenant_id, organization_id, document_id, printer_id, user_id, status,
		       page_count, color, duplex, paper_size, copies, options, created_at, updated_at, completed_at
		FROM print_jobs
		WHERE id = $1
	`

	job, err := r.scanJob(r.db.QueryRow(ctx, query, jobID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find job by tenant: %w", err)
	}

	// Verify the job belongs to the tenant
	if job.TenantID != tenantID && job.OrganizationID != tenantID {
		return nil, ErrNotFound
	}

	return job, nil
}

// UpdateWithTenant updates a job within a tenant context.
func (r *JobRepositoryWithTenant) UpdateWithTenant(ctx context.Context, job *Job, tenantID string) error {
	// Set tenant context for RLS
	if _, err := r.db.Exec(ctx, "SET LOCAL app.tenant_id = $1", tenantID); err != nil {
		return fmt.Errorf("set tenant context: %w", err)
	}

	defer func() {
		_, _ = r.db.Exec(ctx, "SET LOCAL app.tenant_id = NULL")
	}()

	job.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE print_jobs
		SET status = $2, completed_at = $3, updated_at = $4
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query,
		job.ID,
		job.Status,
		job.CompletedAt,
		job.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("update job with tenant: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// CountByTenant returns the number of jobs for a tenant in the current month.
func (r *JobRepositoryWithTenant) CountByTenant(ctx context.Context, tenantID string) (int, error) {
	monthStart := time.Now().UTC().Truncate(24 * time.Hour).AddDate(0, 0, -time.Now().UTC().Day()+1)

	query := `
		SELECT COUNT(*)
		FROM print_jobs
		WHERE tenant_id = $1 AND created_at >= $2
	`

	var count int
	err := r.db.QueryRow(ctx, query, tenantID, monthStart).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count jobs by tenant: %w", err)
	}

	return count, nil
}

// scanJob scans a row into a Job struct.
func (r *JobRepositoryWithTenant) scanJob(row pgx.Row) (*Job, error) {
	var j Job
	err := row.Scan(
		&j.ID,
		&j.TenantID,
		&j.OrganizationID,
		&j.DocumentID,
		&j.PrinterID,
		&j.UserID,
		&j.Status,
		&j.PageCount,
		&j.Color,
		&j.Duplex,
		&j.PaperSize,
		&j.Copies,
		&j.Options,
		&j.CreatedAt,
		&j.UpdatedAt,
		&j.CompletedAt,
	)
	if err != nil {
		return nil, err
	}
	return &j, nil
}
