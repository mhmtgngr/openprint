// Package repository provides data access layer for the job service.
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PrintJob represents a print job.
type PrintJob struct {
	ID          string
	DocumentID  string
	PrinterID   string
	UserName    string
	UserEmail   string
	Title       string
	Copies      int
	ColorMode   string
	Duplex      bool
	MediaType   string
	Quality     string
	Pages       int
	Status      string // "queued", "processing", "pending_agent", "completed", "failed", "cancelled", "paused"
	Priority    int    // 1-10, higher = more important
	Retries     int
	Options     string // JSON string
	AgentID     string
	StartedAt   time.Time
	CompletedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// JobRepository handles print job data operations.
type JobRepository struct {
	db *pgxpool.Pool
}

// NewJobRepository creates a new job repository.
func NewJobRepository(db *pgxpool.Pool) *JobRepository {
	return &JobRepository{db: db}
}

// Create inserts a new print job.
func (r *JobRepository) Create(ctx context.Context, job *PrintJob) error {
	now := time.Now()
	job.CreatedAt = now
	job.UpdatedAt = now

	if job.ID == "" {
		job.ID = uuid.New().String()
	}

	// Handle empty options by setting it to a valid empty JSON object
	options := job.Options
	if options == "" {
		options = "{}"
	}

	query := `
		INSERT INTO print_jobs (id, document_id, printer_id, user_name, user_email, title, copies,
		                       color_mode, duplex, media_type, quality, pages, status, priority,
		                       retries, options, agent_id, started_at, completed_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16::jsonb, NULLIF($17, '')::uuid, $18, $19, $20, $21)
		RETURNING id
	`

	err := r.db.QueryRow(ctx, query,
		job.ID, job.DocumentID, job.PrinterID, job.UserName, job.UserEmail, job.Title, job.Copies,
		job.ColorMode, job.Duplex, job.MediaType, job.Quality, job.Pages, job.Status, job.Priority,
		job.Retries, options, job.AgentID, job.StartedAt, job.CompletedAt, job.CreatedAt, job.UpdatedAt,
	).Scan(&job.ID)

	if err != nil {
		return fmt.Errorf("create print job: %w", err)
	}

	return nil
}

// FindByID retrieves a print job by ID.
func (r *JobRepository) FindByID(ctx context.Context, id string) (*PrintJob, error) {
	query := `
		SELECT id, document_id, printer_id, user_name, user_email, title, copies,
		       color_mode, duplex, media_type, quality, pages, status, priority,
		       retries, options, agent_id, started_at, completed_at, created_at, updated_at
		FROM print_jobs
		WHERE id = $1
	`

	job, err := r.scanJob(r.db.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("print job not found")
		}
		return nil, fmt.Errorf("find print job by id: %w", err)
	}

	return job, nil
}

// FindByStatus retrieves all jobs with a given status.
func (r *JobRepository) FindByStatus(ctx context.Context, status string, limit int) ([]*PrintJob, error) {
	query := `
		SELECT id, document_id, printer_id, user_name, user_email, title, copies,
		       color_mode, duplex, media_type, quality, pages, status, priority,
		       retries, options, agent_id, started_at, completed_at, created_at, updated_at
		FROM print_jobs
		WHERE status = $1
		ORDER BY priority DESC, created_at ASC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, status, limit)
	if err != nil {
		return nil, fmt.Errorf("find jobs by status: %w", err)
	}
	defer rows.Close()

	var jobs []*PrintJob
	for rows.Next() {
		job, err := r.scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

// FindByPrinter retrieves jobs for a specific printer.
func (r *JobRepository) FindByPrinter(ctx context.Context, printerID string, limit, offset int) ([]*PrintJob, error) {
	query := `
		SELECT id, document_id, printer_id, user_name, user_email, title, copies,
		       color_mode, duplex, media_type, quality, pages, status, priority,
		       retries, options, agent_id, started_at, completed_at, created_at, updated_at
		FROM print_jobs
		WHERE printer_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, printerID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("find jobs by printer: %w", err)
	}
	defer rows.Close()

	var jobs []*PrintJob
	for rows.Next() {
		job, err := r.scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

// FindByUser retrieves jobs for a specific user.
func (r *JobRepository) FindByUser(ctx context.Context, userEmail string, limit, offset int) ([]*PrintJob, error) {
	query := `
		SELECT id, document_id, printer_id, user_name, user_email, title, copies,
		       color_mode, duplex, media_type, quality, pages, status, priority,
		       retries, options, agent_id, started_at, completed_at, created_at, updated_at
		FROM print_jobs
		WHERE user_email = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, userEmail, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("find jobs by user: %w", err)
	}
	defer rows.Close()

	var jobs []*PrintJob
	for rows.Next() {
		job, err := r.scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

// ListWithFilters retrieves jobs with optional filters.
func (r *JobRepository) ListWithFilters(ctx context.Context, limit, offset int, printerID, status, userEmail string) ([]*PrintJob, int, error) {
	// Build query with filters
	baseQuery := `SELECT id, document_id, printer_id, user_name, user_email, title, copies,
		       color_mode, duplex, media_type, quality, pages, status, priority,
		       retries, options, agent_id, started_at, completed_at, created_at, updated_at
		       FROM print_jobs WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM print_jobs WHERE 1=1`
	args := []interface{}{}
	argNum := 1

	if printerID != "" {
		baseQuery += fmt.Sprintf(" AND printer_id = $%d", argNum)
		countQuery += fmt.Sprintf(" AND printer_id = $%d", argNum)
		args = append(args, printerID)
		argNum++
	}
	if status != "" {
		baseQuery += fmt.Sprintf(" AND status = $%d", argNum)
		countQuery += fmt.Sprintf(" AND status = $%d", argNum)
		args = append(args, status)
		argNum++
	}
	if userEmail != "" {
		baseQuery += fmt.Sprintf(" AND user_email = $%d", argNum)
		countQuery += fmt.Sprintf(" AND user_email = $%d", argNum)
		args = append(args, userEmail)
		argNum++
	}

	// Get total count
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count jobs: %w", err)
	}

	// Add ordering and pagination
	baseQuery += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*PrintJob
	for rows.Next() {
		job, err := r.scanJob(rows)
		if err != nil {
			return nil, 0, err
		}
		jobs = append(jobs, job)
	}

	return jobs, total, rows.Err()
}

// Update updates a print job.
func (r *JobRepository) Update(ctx context.Context, job *PrintJob) error {
	job.UpdatedAt = time.Now()

	query := `
		UPDATE print_jobs
		SET document_id = $2, printer_id = $3, user_name = $4, user_email = $5, title = $6,
		    copies = $7, color_mode = $8, duplex = $9, media_type = $10, quality = $11,
		    pages = $12, status = $13, priority = $14, retries = $15, options = $16,
		    agent_id = NULLIF($17, '')::uuid, started_at = $18, completed_at = $19, updated_at = $20
		WHERE id = $1
	`

	cmdTag, err := r.db.Exec(ctx, query,
		job.ID, job.DocumentID, job.PrinterID, job.UserName, job.UserEmail, job.Title, job.Copies,
		job.ColorMode, job.Duplex, job.MediaType, job.Quality, job.Pages, job.Status, job.Priority,
		job.Retries, job.Options, job.AgentID, job.StartedAt, job.CompletedAt, job.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("update print job: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("print job not found")
	}

	return nil
}

// UpdateStatus updates only the status of a job.
func (r *JobRepository) UpdateStatus(ctx context.Context, id, status string) error {
	query := `UPDATE print_jobs SET status = $2, updated_at = $3 WHERE id = $1`

	cmdTag, err := r.db.Exec(ctx, query, id, status, time.Now())
	if err != nil {
		return fmt.Errorf("update job status: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("print job not found")
	}

	return nil
}

// AssignAgent assigns an agent to a job.
func (r *JobRepository) AssignAgent(ctx context.Context, jobID, agentID string) error {
	query := `UPDATE print_jobs SET agent_id = $2, status = 'processing', updated_at = $3 WHERE id = $1`

	cmdTag, err := r.db.Exec(ctx, query, jobID, agentID, time.Now())
	if err != nil {
		return fmt.Errorf("assign agent: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("print job not found")
	}

	return nil
}

// CountByStatus returns the count of jobs by status.
func (r *JobRepository) CountByStatus(ctx context.Context, status string) (int64, error) {
	var count int64
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM print_jobs WHERE status = $1", status).Scan(&count)
	return count, err
}

// Delete removes a print job.
func (r *JobRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM print_jobs WHERE id = $1`

	cmdTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete print job: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("print job not found")
	}

	return nil
}

// GetNextPendingJob retrieves the next pending job for a printer/agent.
func (r *JobRepository) GetNextPendingJob(ctx context.Context, printerID string) (*PrintJob, error) {
	query := `
		SELECT id, document_id, printer_id, user_name, user_email, title, copies,
		       color_mode, duplex, media_type, quality, pages, status, priority,
		       retries, options, agent_id, started_at, completed_at, created_at, updated_at
		FROM print_jobs
		WHERE printer_id = $1 AND status IN ('queued', 'pending_agent')
		ORDER BY priority DESC, created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`

	job, err := r.scanJob(r.db.QueryRow(ctx, query, printerID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get next pending job: %w", err)
	}

	return job, nil
}

// UpdateJobProgress updates job progress information.
func (r *JobRepository) UpdateJobProgress(ctx context.Context, jobID string, pagesPrinted int) error {
	query := `UPDATE print_jobs SET pages = $2, updated_at = $3 WHERE id = $1`

	cmdTag, err := r.db.Exec(ctx, query, jobID, pagesPrinted, time.Now())
	if err != nil {
		return fmt.Errorf("update job progress: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("print job not found")
	}

	return nil
}

// GetJobsNeedingRetry finds failed jobs that can be retried.
func (r *JobRepository) GetJobsNeedingRetry(ctx context.Context, maxRetries int, limit int) ([]*PrintJob, error) {
	query := `
		SELECT id, document_id, printer_id, user_name, user_email, title, copies,
		       color_mode, duplex, media_type, quality, pages, status, priority,
		       retries, options, agent_id, started_at, completed_at, created_at, updated_at
		FROM print_jobs
		WHERE status = 'failed' AND retries < $1
		ORDER BY created_at ASC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, maxRetries, limit)
	if err != nil {
		return nil, fmt.Errorf("find retry jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*PrintJob
	for rows.Next() {
		job, err := r.scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

// scanJob scans a print job from a database row.
func (r *JobRepository) scanJob(row interface{ Scan(...interface{}) error }) (*PrintJob, error) {
	var job PrintJob
	// Use pointers for fields that can be NULL
	var options *string
	var agentID *string
	err := row.Scan(
		&job.ID, &job.DocumentID, &job.PrinterID, &job.UserName, &job.UserEmail, &job.Title, &job.Copies,
		&job.ColorMode, &job.Duplex, &job.MediaType, &job.Quality, &job.Pages, &job.Status, &job.Priority,
		&job.Retries, &options, &agentID, &job.StartedAt, &job.CompletedAt, &job.CreatedAt, &job.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	// Convert pointers to strings, defaulting to empty string if NULL
	if options != nil {
		job.Options = *options
	}
	if agentID != nil {
		job.AgentID = *agentID
	}
	return &job, nil
}
