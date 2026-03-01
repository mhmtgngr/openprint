// Package repository provides data access layer for job assignment operations.
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// JobAssignment represents the assignment of a job to a specific agent.
type JobAssignment struct {
	ID           string
	JobID        string
	AgentID      string
	AssignedAt   time.Time
	StartedAt    *time.Time
	CompletedAt  *time.Time
	Status       string // "assigned", "in_progress", "completed", "failed", "cancelled"
	RetryCount   int
	LastHeartbeat time.Time
	Error        string
	DocumentETag string // For resume support
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// AssignmentCriteria defines criteria for assigning jobs.
type AssignmentCriteria struct {
	UserEmail     string
	AgentID       string
	PrinterID     string
	Status        string
	Limit         int
	Offset        int
	ExcludeJobIDs []string
}

// JobAssignmentRepository handles job assignment data operations.
type JobAssignmentRepository struct {
	db *pgxpool.Pool
}

// NewJobAssignmentRepository creates a new job assignment repository.
func NewJobAssignmentRepository(db *pgxpool.Pool) *JobAssignmentRepository {
	return &JobAssignmentRepository{db: db}
}

// AssignJob assigns a job to an agent.
func (r *JobAssignmentRepository) AssignJob(ctx context.Context, assignment *JobAssignment) error {
	now := time.Now()
	assignment.ID = uuid.New().String()
	assignment.AssignedAt = now
	assignment.CreatedAt = now
	assignment.UpdatedAt = now
	assignment.LastHeartbeat = now

	query := `
		INSERT INTO job_assignments (id, job_id, agent_id, assigned_at, started_at,
			completed_at, status, retry_count, last_heartbeat, error, document_etag, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id
	`

	err := r.db.QueryRow(ctx, query,
		assignment.ID,
		assignment.JobID,
		assignment.AgentID,
		assignment.AssignedAt,
		assignment.StartedAt,
		assignment.CompletedAt,
		assignment.Status,
		assignment.RetryCount,
		assignment.LastHeartbeat,
		assignment.Error,
		assignment.DocumentETag,
		assignment.CreatedAt,
		assignment.UpdatedAt,
	).Scan(&assignment.ID)

	if err != nil {
		return fmt.Errorf("assign job: %w", err)
	}

	// Update the job's agent_id in print_jobs table
	_, err = r.db.Exec(ctx, "UPDATE print_jobs SET agent_id = $1, updated_at = $2 WHERE id = $3",
		assignment.AgentID, now, assignment.JobID)
	if err != nil {
		// Log but don't fail - the assignment is still valid
		fmt.Printf("Warning: failed to update job agent_id: %v", err)
	}

	return nil
}

// FindByID retrieves a job assignment by ID.
func (r *JobAssignmentRepository) FindByID(ctx context.Context, id string) (*JobAssignment, error) {
	query := `
		SELECT id, job_id, agent_id, assigned_at, started_at, completed_at,
		       status, retry_count, last_heartbeat, error, document_etag, created_at, updated_at
		FROM job_assignments
		WHERE id = $1
	`

	assignment, err := r.scanAssignment(r.db.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("assignment not found")
		}
		return nil, fmt.Errorf("find assignment by id: %w", err)
	}

	return assignment, nil
}

// FindByJobID retrieves all assignments for a job.
func (r *JobAssignmentRepository) FindByJobID(ctx context.Context, jobID string) ([]*JobAssignment, error) {
	query := `
		SELECT id, job_id, agent_id, assigned_at, started_at, completed_at,
		       status, retry_count, last_heartbeat, error, document_etag, created_at, updated_at
		FROM job_assignments
		WHERE job_id = $1
		ORDER BY assigned_at DESC
	`

	rows, err := r.db.Query(ctx, query, jobID)
	if err != nil {
		return nil, fmt.Errorf("find assignments by job id: %w", err)
	}
	defer rows.Close()

	var assignments []*JobAssignment
	for rows.Next() {
		assignment, err := r.scanAssignment(rows)
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, assignment)
	}

	return assignments, rows.Err()
}

// FindByAgent retrieves pending/active assignments for an agent.
func (r *JobAssignmentRepository) FindByAgent(ctx context.Context, agentID string, limit int) ([]*JobAssignment, error) {
	if limit == 0 {
		limit = 50
	}

	query := `
		SELECT id, job_id, agent_id, assigned_at, started_at, completed_at,
		       status, retry_count, last_heartbeat, error, document_etag, created_at, updated_at
		FROM job_assignments
		WHERE agent_id = $1 AND status IN ('assigned', 'in_progress')
		ORDER BY assigned_at ASC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, agentID, limit)
	if err != nil {
		return nil, fmt.Errorf("find assignments by agent: %w", err)
	}
	defer rows.Close()

	var assignments []*JobAssignment
	for rows.Next() {
		assignment, err := r.scanAssignment(rows)
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, assignment)
	}

	return assignments, rows.Err()
}

// FindByJobAndAgent retrieves an assignment for a specific job and agent.
func (r *JobAssignmentRepository) FindByJobAndAgent(ctx context.Context, jobID, agentID string) (*JobAssignment, error) {
	query := `
		SELECT id, job_id, agent_id, assigned_at, started_at, completed_at,
		       status, retry_count, last_heartbeat, error, document_etag, created_at, updated_at
		FROM job_assignments
		WHERE job_id = $1 AND agent_id = $2
		ORDER BY assigned_at DESC
		LIMIT 1
	`

	assignment, err := r.scanAssignment(r.db.QueryRow(ctx, query, jobID, agentID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("assignment not found")
		}
		return nil, fmt.Errorf("find assignment by job and agent: %w", err)
	}

	return assignment, nil
}

// FindPendingJobs retrieves jobs pending assignment for a user.
func (r *JobAssignmentRepository) FindPendingJobs(ctx context.Context, criteria AssignmentCriteria) ([]*PrintJob, error) {
	if criteria.Limit == 0 {
		criteria.Limit = 50
	}

	query := `
		SELECT j.id, j.document_id, j.printer_id, j.user_name, j.user_email, j.title,
		       j.copies, j.color_mode, j.duplex, j.media_type, j.quality, j.pages,
		       j.status, j.priority, j.retries, j.options, j.agent_id,
		       j.started_at, j.completed_at, j.created_at, j.updated_at
		FROM print_jobs j
		WHERE j.status = 'queued'
			AND ($1 = '' OR j.user_email = $1)
			AND ($2 = '' OR j.printer_id = $2)
			AND ($3 = '' OR j.agent_id IS NULL)
		ORDER BY j.priority DESC, j.created_at ASC
		LIMIT $4 OFFSET $5
	`

	rows, err := r.db.Query(ctx, query,
		criteria.UserEmail,
		criteria.PrinterID,
		criteria.AgentID,
		criteria.Limit,
		criteria.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf("find pending jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*PrintJob
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

// UpdateStatus updates an assignment's status.
func (r *JobAssignmentRepository) UpdateStatus(ctx context.Context, assignmentID, status string) error {
	now := time.Now()

	// Use parameterized queries to prevent SQL injection
	// Set timestamps based on status
	var query string
	var args []interface{}

	switch status {
	case "in_progress":
		query = `
			UPDATE job_assignments
			SET status = $2, updated_at = $3, started_at = $3
			WHERE id = $1
		`
		args = []interface{}{assignmentID, status, now}
	case "completed", "failed", "cancelled":
		query = `
			UPDATE job_assignments
			SET status = $2, updated_at = $3, completed_at = $3
			WHERE id = $1
		`
		args = []interface{}{assignmentID, status, now}
	default:
		query = `
			UPDATE job_assignments
			SET status = $2, updated_at = $3
			WHERE id = $1
		`
		args = []interface{}{assignmentID, status, now}
	}

	cmdTag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update assignment status: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("assignment not found")
	}

	return nil
}

// UpdateHeartbeat updates the last heartbeat time for an assignment.
func (r *JobAssignmentRepository) UpdateHeartbeat(ctx context.Context, assignmentID string, heartbeat time.Time) error {
	query := `
		UPDATE job_assignments
		SET last_heartbeat = $2, updated_at = $3
		WHERE id = $1
	`

	cmdTag, err := r.db.Exec(ctx, query, assignmentID, heartbeat, heartbeat)
	if err != nil {
		return fmt.Errorf("update assignment heartbeat: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("assignment not found")
	}

	return nil
}

// IncrementRetry increments the retry count for an assignment.
func (r *JobAssignmentRepository) IncrementRetry(ctx context.Context, assignmentID string) error {
	query := `
		UPDATE job_assignments
		SET retry_count = retry_count + 1, updated_at = $2
		WHERE id = $1
		RETURNING retry_count
	`

	var retryCount int
	err := r.db.QueryRow(ctx, query, assignmentID, time.Now()).Scan(&retryCount)
	if err != nil {
		return fmt.Errorf("increment retry count: %w", err)
	}

	return nil
}

// SetError sets an error message for an assignment.
func (r *JobAssignmentRepository) SetError(ctx context.Context, assignmentID, errorMsg string) error {
	query := `
		UPDATE job_assignments
		SET error = $2, status = 'failed', completed_at = $3, updated_at = $3
		WHERE id = $1
	`

	cmdTag, err := r.db.Exec(ctx, query, assignmentID, errorMsg, time.Now())
	if err != nil {
		return fmt.Errorf("set assignment error: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("assignment not found")
	}

	return nil
}

// Delete removes an assignment.
func (r *JobAssignmentRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM job_assignments WHERE id = $1`

	cmdTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete assignment: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("assignment not found")
	}

	return nil
}

// DeleteByJob removes all assignments for a job.
func (r *JobAssignmentRepository) DeleteByJob(ctx context.Context, jobID string) error {
	query := `DELETE FROM job_assignments WHERE job_id = $1`

	_, err := r.db.Exec(ctx, query, jobID)
	if err != nil {
		return fmt.Errorf("delete assignments by job: %w", err)
	}

	return nil
}

// GetStaleAssignments returns assignments that haven't sent a heartbeat recently.
func (r *JobAssignmentRepository) GetStaleAssignments(ctx context.Context, since time.Time) ([]*JobAssignment, error) {
	query := `
		SELECT id, job_id, agent_id, assigned_at, started_at, completed_at,
		       status, retry_count, last_heartbeat, error, document_etag, created_at, updated_at
		FROM job_assignments
		WHERE last_heartbeat < $1 AND status IN ('assigned', 'in_progress')
		ORDER BY last_heartbeat ASC
	`

	rows, err := r.db.Query(ctx, query, since)
	if err != nil {
		return nil, fmt.Errorf("get stale assignments: %w", err)
	}
	defer rows.Close()

	var assignments []*JobAssignment
	for rows.Next() {
		assignment, err := r.scanAssignment(rows)
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, assignment)
	}

	return assignments, rows.Err()
}

// GetAssignmentStats returns statistics for assignments.
func (r *JobAssignmentRepository) GetAssignmentStats(ctx context.Context, agentID string) (map[string]int64, error) {
	var query string
	var args []interface{}

	if agentID == "" {
		query = `
			SELECT status, COUNT(*) as count
			FROM job_assignments
			GROUP BY status
		`
		args = []interface{}{}
	} else {
		query = `
			SELECT status, COUNT(*) as count
			FROM job_assignments
			WHERE agent_id = $1
			GROUP BY status
		`
		args = []interface{}{agentID}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get assignment stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]int64)
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats[status] = count
	}

	return stats, rows.Err()
}

// GetActiveAgentsForUser returns agents that have active jobs for a user.
func (r *JobAssignmentRepository) GetActiveAgentsForUser(ctx context.Context, userEmail string) ([]string, error) {
	query := `
		SELECT DISTINCT a.agent_id
		FROM job_assignments a
		INNER JOIN print_jobs j ON j.id = a.job_id
		WHERE j.user_email = $1 AND a.status IN ('assigned', 'in_progress')
	`

	rows, err := r.db.Query(ctx, query, userEmail)
	if err != nil {
		return nil, fmt.Errorf("get active agents: %w", err)
	}
	defer rows.Close()

	var agents []string
	for rows.Next() {
		var agentID string
		if err := rows.Scan(&agentID); err != nil {
			return nil, err
		}
		agents = append(agents, agentID)
	}

	return agents, rows.Err()
}

// scanAssignment scans an assignment from a database row.
func (r *JobAssignmentRepository) scanAssignment(row interface{ Scan(...interface{}) error }) (*JobAssignment, error) {
	var assignment JobAssignment
	// Use pointers for fields that can be NULL in the database
	var errorMsg *string
	var documentETag *string
	err := row.Scan(
		&assignment.ID,
		&assignment.JobID,
		&assignment.AgentID,
		&assignment.AssignedAt,
		&assignment.StartedAt,
		&assignment.CompletedAt,
		&assignment.Status,
		&assignment.RetryCount,
		&assignment.LastHeartbeat,
		&errorMsg,
		&documentETag,
		&assignment.CreatedAt,
		&assignment.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	// Convert pointers to strings, defaulting to empty string if NULL
	if errorMsg != nil {
		assignment.Error = *errorMsg
	} else {
		assignment.Error = ""
	}
	if documentETag != nil {
		assignment.DocumentETag = *documentETag
	} else {
		assignment.DocumentETag = ""
	}
	return &assignment, nil
}

// scanJob is a helper to scan a PrintJob from a row (copied from job.go)
func scanJob(row interface{ Scan(...interface{}) error }) (*PrintJob, error) {
	var job PrintJob
	// Use pointers for fields that can be NULL
	var options *string
	var agentID *string
	var pages *int
	var startedAt *time.Time
	err := row.Scan(
		&job.ID,
		&job.DocumentID,
		&job.PrinterID,
		&job.UserName,
		&job.UserEmail,
		&job.Title,
		&job.Copies,
		&job.ColorMode,
		&job.Duplex,
		&job.MediaType,
		&job.Quality,
		&pages,
		&job.Status,
		&job.Priority,
		&job.Retries,
		&options,
		&agentID,
		&startedAt,
		&job.CompletedAt,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	// Convert pointers to values, defaulting to zero/empty if NULL
	if pages != nil {
		job.Pages = *pages
	}
	if options != nil {
		job.Options = *options
	}
	if agentID != nil {
		job.AgentID = *agentID
	}
	if startedAt != nil {
		job.StartedAt = *startedAt
	} else {
		job.StartedAt = time.Time{} // Zero time if NULL
	}
	return &job, nil
}

// GetJobsForAgentPolling retrieves jobs for an agent to poll.
// This is an optimized query that joins print_jobs with discovered_printers
// to match jobs with the agent's available printers.
// It also supports user-based routing: jobs with printer_id='__user_default__'
// are routed via user_printer_mappings to the correct client agent.
func (r *JobAssignmentRepository) GetJobsForAgentPolling(ctx context.Context, agentID, userEmail string, limit int) ([]*PrintJob, error) {
	if limit == 0 {
		limit = 50
	}

	// This query finds jobs that:
	// 1. Are in 'queued' status
	// 2. Match the user email (if provided)
	// 3. Have printers available to this agent (direct match via discovered_printers)
	// 4. OR are user-routed jobs (__user_default__) mapped to this agent via user_printer_mappings
	query := `
		SELECT DISTINCT j.id, j.document_id, j.printer_id, j.user_name, j.user_email, j.title,
		       j.copies, j.color_mode, j.duplex, j.media_type, j.quality, j.pages,
		       j.status, j.priority, j.retries, j.options, j.agent_id,
		       j.started_at, j.completed_at, j.created_at, j.updated_at
		FROM print_jobs j
		LEFT JOIN discovered_printers dp ON (dp.name = j.printer_id OR dp.id::text = j.printer_id) AND dp.agent_id = $1
		LEFT JOIN user_printer_mappings upm ON upm.user_email = j.user_email AND upm.client_agent_id = $1 AND upm.is_active = true
		WHERE j.status = 'queued'
			AND (j.agent_id IS NULL OR j.agent_id = $1)
			AND ($2 = '' OR j.user_email = $2)
			AND (
				dp.id IS NOT NULL
				OR (j.printer_id = '__user_default__' AND upm.id IS NOT NULL)
			)
		ORDER BY j.priority DESC, j.created_at ASC
		LIMIT $3
	`

	rows, err := r.db.Query(ctx, query, agentID, userEmail, limit)
	if err != nil {
		return nil, fmt.Errorf("find jobs for agent polling: %w", err)
	}
	defer rows.Close()

	var jobs []*PrintJob
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

// ResolveUserDefaultPrinter resolves __user_default__ printer_id to the actual target printer
// using user_printer_mappings.
func (r *JobAssignmentRepository) ResolveUserDefaultPrinter(ctx context.Context, userEmail, clientAgentID string) (string, string, error) {
	query := `
		SELECT COALESCE(upm.target_printer_name, dp.name, ''), COALESCE(upm.target_printer_id::text, dp.id::text, '')
		FROM user_printer_mappings upm
		LEFT JOIN discovered_printers dp ON dp.id = upm.target_printer_id
		WHERE upm.user_email = $1
			AND upm.client_agent_id = $2
			AND upm.is_active = true
		ORDER BY upm.is_default DESC
		LIMIT 1
	`

	var printerName, printerID string
	err := r.db.QueryRow(ctx, query, userEmail, clientAgentID).Scan(&printerName, &printerID)
	if err != nil {
		return "", "", fmt.Errorf("resolve user default printer: %w", err)
	}

	return printerName, printerID, nil
}

// GetJobWithPrinter retrieves a job with its associated printer information.
func (r *JobAssignmentRepository) GetJobWithPrinter(ctx context.Context, jobID string) (*PrintJob, map[string]interface{}, error) {
	query := `
		SELECT j.id, j.document_id, j.printer_id, j.user_name, j.user_email, j.title,
		       j.copies, j.color_mode, j.duplex, j.media_type, j.quality, j.pages,
		       j.status, j.priority, j.retries, j.options, j.agent_id,
		       j.started_at, j.completed_at, j.created_at, j.updated_at,
		       p.name as printer_name, p.status as printer_status, p.agent_id as printer_agent_id
		FROM print_jobs j
		LEFT JOIN printers p ON p.id = j.printer_id
		WHERE j.id = $1
	`

	var job PrintJob
	var printerName, printerStatus, printerAgentID *string
	var optionsJSON string

	err := r.db.QueryRow(ctx, query, jobID).Scan(
		&job.ID,
		&job.DocumentID,
		&job.PrinterID,
		&job.UserName,
		&job.UserEmail,
		&job.Title,
		&job.Copies,
		&job.ColorMode,
		&job.Duplex,
		&job.MediaType,
		&job.Quality,
		&job.Pages,
		&job.Status,
		&job.Priority,
		&job.Retries,
		&optionsJSON,
		&job.AgentID,
		&job.StartedAt,
		&job.CompletedAt,
		&job.CreatedAt,
		&job.UpdatedAt,
		&printerName,
		&printerStatus,
		&printerAgentID,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil, fmt.Errorf("job not found")
		}
		return nil, nil, fmt.Errorf("get job with printer: %w", err)
	}

	job.Options = optionsJSON

	printerInfo := make(map[string]interface{})
	if printerName != nil {
		printerInfo["name"] = *printerName
	}
	if printerStatus != nil {
		printerInfo["status"] = *printerStatus
	}
	if printerAgentID != nil {
		printerInfo["agent_id"] = *printerAgentID
	}

	return &job, printerInfo, nil
}
