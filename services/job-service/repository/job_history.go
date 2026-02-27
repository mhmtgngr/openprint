// Package repository provides job history data access for the job service.
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// JobHistory represents a status change in a print job's lifecycle.
type JobHistory struct {
	ID        string
	JobID     string
	Status    string
	Message   string
	CreatedAt time.Time
}

// JobHistoryRepository handles job history data operations.
type JobHistoryRepository struct {
	db *pgxpool.Pool
}

// NewJobHistoryRepository creates a new job history repository.
func NewJobHistoryRepository(db *pgxpool.Pool) *JobHistoryRepository {
	return &JobHistoryRepository{db: db}
}

// Create inserts a new job history entry.
func (r *JobHistoryRepository) Create(ctx context.Context, history *JobHistory) error {
	if history.ID == "" {
		history.ID = uuid.New().String()
	}
	if history.CreatedAt.IsZero() {
		history.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO job_history (id, job_id, status, message, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	err := r.db.QueryRow(ctx, query,
		history.ID,
		history.JobID,
		history.Status,
		history.Message,
		history.CreatedAt,
	).Scan(&history.ID)

	if err != nil {
		return fmt.Errorf("create job history: %w", err)
	}

	return nil
}

// FindByID retrieves a job history entry by ID.
func (r *JobHistoryRepository) FindByID(ctx context.Context, id string) (*JobHistory, error) {
	query := `
		SELECT id, job_id, status, message, created_at
		FROM job_history
		WHERE id = $1
	`

	history, err := r.scanHistory(r.db.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("job history not found")
		}
		return nil, fmt.Errorf("find history by id: %w", err)
	}

	return history, nil
}

// FindByJobID retrieves all history entries for a job.
func (r *JobHistoryRepository) FindByJobID(ctx context.Context, jobID string) ([]*JobHistory, error) {
	query := `
		SELECT id, job_id, status, message, created_at
		FROM job_history
		WHERE job_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(ctx, query, jobID)
	if err != nil {
		return nil, fmt.Errorf("find history by job id: %w", err)
	}
	defer rows.Close()

	var history []*JobHistory
	for rows.Next() {
		h, err := r.scanHistory(rows)
		if err != nil {
			return nil, err
		}
		history = append(history, h)
	}

	return history, rows.Err()
}

// FindByStatus retrieves all history entries with a given status.
func (r *JobHistoryRepository) FindByStatus(ctx context.Context, status string, limit, offset int) ([]*JobHistory, error) {
	query := `
		SELECT id, job_id, status, message, created_at
		FROM job_history
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("find history by status: %w", err)
	}
	defer rows.Close()

	var history []*JobHistory
	for rows.Next() {
		h, err := r.scanHistory(rows)
		if err != nil {
			return nil, err
		}
		history = append(history, h)
	}

	return history, rows.Err()
}

// DeleteByJobID removes all history entries for a job.
func (r *JobHistoryRepository) DeleteByJobID(ctx context.Context, jobID string) error {
	query := `DELETE FROM job_history WHERE job_id = $1`

	_, err := r.db.Exec(ctx, query, jobID)
	if err != nil {
		return fmt.Errorf("delete history by job id: %w", err)
	}

	return nil
}

// DeleteOld removes history entries older than the given duration.
func (r *JobHistoryRepository) DeleteOld(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)

	query := `DELETE FROM job_history WHERE created_at < $1`

	cmdTag, err := r.db.Exec(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("delete old history: %w", err)
	}

	return cmdTag.RowsAffected(), nil
}

// GetLatestByJobID retrieves the latest history entry for a job.
func (r *JobHistoryRepository) GetLatestByJobID(ctx context.Context, jobID string) (*JobHistory, error) {
	query := `
		SELECT id, job_id, status, message, created_at
		FROM job_history
		WHERE job_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	history, err := r.scanHistory(r.db.QueryRow(ctx, query, jobID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get latest history: %w", err)
	}

	return history, nil
}

// CountByJobID returns the number of history entries for a job.
func (r *JobHistoryRepository) CountByJobID(ctx context.Context, jobID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM job_history WHERE job_id = $1", jobID).Scan(&count)
	return count, err
}

// List retrieves all history entries with pagination.
func (r *JobHistoryRepository) List(ctx context.Context, limit, offset int) ([]*JobHistory, int, error) {
	// Get total count
	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM job_history").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count history: %w", err)
	}

	// Get history
	query := `
		SELECT id, job_id, status, message, created_at
		FROM job_history
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list history: %w", err)
	}
	defer rows.Close()

	var history []*JobHistory
	for rows.Next() {
		h, err := r.scanHistory(rows)
		if err != nil {
			return nil, 0, err
		}
		history = append(history, h)
	}

	return history, total, rows.Err()
}

// CreateBatch inserts multiple history entries efficiently.
func (r *JobHistoryRepository) CreateBatch(ctx context.Context, entries []*JobHistory) error {
	if len(entries) == 0 {
		return nil
	}

	now := time.Now()
	for _, entry := range entries {
		if entry.ID == "" {
			entry.ID = uuid.New().String()
		}
		if entry.CreatedAt.IsZero() {
			entry.CreatedAt = now
		}
	}

	// Use a transaction for batch insert
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	stmt := `
		INSERT INTO job_history (id, job_id, status, message, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	for _, entry := range entries {
		if _, err := tx.Exec(ctx, stmt, entry.ID, entry.JobID, entry.Status, entry.Message, entry.CreatedAt); err != nil {
			return fmt.Errorf("insert history entry: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// scanHistory scans a job history entry from a database row.
func (r *JobHistoryRepository) scanHistory(row interface{ Scan(...interface{}) error }) (*JobHistory, error) {
	var history JobHistory
	err := row.Scan(
		&history.ID,
		&history.JobID,
		&history.Status,
		&history.Message,
		&history.CreatedAt,
	)
	return &history, err
}
