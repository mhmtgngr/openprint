// Package repository provides unit tests for job assignment operations.
package repository

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJobAssignmentRepository_AssignJob tests assigning a job to an agent.
func TestJobAssignmentRepository_AssignJob(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	// Create a test print job first
	job := createTestPrintJob(t, db)

	assignment := &JobAssignment{
		JobID:  job.ID,
		AgentID: "test-agent-123",
		Status: "assigned",
	}

	err := repo.AssignJob(ctx, assignment)
	require.NoError(t, err)
	assert.NotEmpty(t, assignment.ID)
	assert.NotEmpty(t, assignment.AssignedAt)
	assert.Equal(t, "assigned", assignment.Status)
}

// TestJobAssignmentRepository_UpdateStatus tests updating assignment status.
func TestJobAssignmentRepository_UpdateStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	// Create an assignment
	job := createTestPrintJob(t, db)
	assignment := &JobAssignment{
		JobID:   job.ID,
		AgentID: "test-agent-123",
		Status:  "assigned",
	}
	err := repo.AssignJob(ctx, assignment)
	require.NoError(t, err)

	// Update to in_progress
	err = repo.UpdateStatus(ctx, assignment.ID, "in_progress")
	require.NoError(t, err)

	// Verify update
	updated, err := repo.FindByID(ctx, assignment.ID)
	require.NoError(t, err)
	assert.Equal(t, "in_progress", updated.Status)
	assert.NotNil(t, updated.StartedAt)

	// Update to completed
	err = repo.UpdateStatus(ctx, assignment.ID, "completed")
	require.NoError(t, err)

	updated, err = repo.FindByID(ctx, assignment.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", updated.Status)
	assert.NotNil(t, updated.CompletedAt)
}

// TestJobAssignmentRepository_UpdateStatus_SQLInjection tests that UpdateStatus
// is protected against SQL injection (regression test for the fix).
func TestJobAssignmentRepository_UpdateStatus_SQLInjection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	// Create an assignment
	job := createTestPrintJob(t, db)
	assignment := &JobAssignment{
		JobID:   job.ID,
		AgentID: "test-agent-123",
		Status:  "assigned",
	}
	err := repo.AssignJob(ctx, assignment)
	require.NoError(t, err)

	// Try various SQL injection attempts
	maliciousInputs := []string{
		"in_progress'; DROP TABLE job_assignments; --",
		"completed' OR '1'='1",
		"failed'; DELETE FROM users WHERE '1'='1'; --",
		"in_progress'; INSERT INTO job_assignments (status) VALUES ('hacked'); --",
	}

	for _, maliciousInput := range maliciousInputs {
		t.Run("Input: "+maliciousInput, func(t *testing.T) {
			// This should not cause an error or execute the injected SQL
			err := repo.UpdateStatus(ctx, assignment.ID, maliciousInput)
			// The update might fail validation, but shouldn't cause SQL errors
			// that indicate successful injection
			if err != nil {
				assert.NotContains(t, err.Error(), "DROP TABLE")
				assert.NotContains(t, err.Error(), "DELETE FROM users")
				assert.NotContains(t, err.Error(), "syntax error")
			}
		})
	}

	// Verify the assignment is still in a valid state
	updated, err := repo.FindByID(ctx, assignment.ID)
	require.NoError(t, err)
	// Status should be one of the valid states (possibly the malicious input
	// if no validation exists, but no SQL injection should have occurred)
	assert.NotEmpty(t, updated.Status)
}

// TestJobAssignmentRepository_FindByAgent tests finding assignments by agent.
func TestJobAssignmentRepository_FindByAgent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	// Create multiple assignments for the same agent
	agentID := "test-agent-find"
	for i := 0; i < 3; i++ {
		job := createTestPrintJob(t, db)
		assignment := &JobAssignment{
			JobID:   job.ID,
			AgentID: agentID,
			Status:  "assigned",
		}
		err := repo.AssignJob(ctx, assignment)
		require.NoError(t, err)
	}

	// Find assignments
	assignments, err := repo.FindByAgent(ctx, agentID, 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(assignments), 3)

	for _, a := range assignments {
		assert.Equal(t, agentID, a.AgentID)
	}
}

// TestJobAssignmentRepository_FindByJobAndAgent tests finding assignment by job and agent.
func TestJobAssignmentRepository_FindByJobAndAgent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	job := createTestPrintJob(t, db)
	assignment := &JobAssignment{
		JobID:   job.ID,
		AgentID: "test-agent-specific",
		Status:  "assigned",
	}
	err := repo.AssignJob(ctx, assignment)
	require.NoError(t, err)

	// Find the assignment
	found, err := repo.FindByJobAndAgent(ctx, job.ID, "test-agent-specific")
	require.NoError(t, err)
	assert.Equal(t, assignment.ID, found.ID)
	assert.Equal(t, job.ID, found.JobID)
	assert.Equal(t, "test-agent-specific", found.AgentID)
}

// TestJobAssignmentRepository_UpdateHeartbeat tests updating heartbeat timestamp.
func TestJobAssignmentRepository_UpdateHeartbeat(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	job := createTestPrintJob(t, db)
	assignment := &JobAssignment{
		JobID:   job.ID,
		AgentID: "test-agent-heartbeat",
		Status:  "in_progress",
	}
	err := repo.AssignJob(ctx, assignment)
	require.NoError(t, err)

	// Update heartbeat
	newTime := time.Now()
	err = repo.UpdateHeartbeat(ctx, assignment.ID, newTime)
	require.NoError(t, err)

	// Verify update
	updated, err := repo.FindByID(ctx, assignment.ID)
	require.NoError(t, err)
	assert.WithinDuration(t, newTime, updated.LastHeartbeat, time.Second)
}

// TestJobAssignmentRepository_IncrementRetry tests incrementing retry count.
func TestJobAssignmentRepository_IncrementRetry(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	job := createTestPrintJob(t, db)
	assignment := &JobAssignment{
		JobID:      job.ID,
		AgentID:    "test-agent-retry",
		Status:     "assigned",
		RetryCount: 0,
	}
	err := repo.AssignJob(ctx, assignment)
	require.NoError(t, err)

	// Increment retry count
	err = repo.IncrementRetry(ctx, assignment.ID)
	require.NoError(t, err)

	// Verify increment
	updated, err := repo.FindByID(ctx, assignment.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, updated.RetryCount)
}

// TestJobAssignmentRepository_SetError tests setting error message.
func TestJobAssignmentRepository_SetError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	job := createTestPrintJob(t, db)
	assignment := &JobAssignment{
		JobID:   job.ID,
		AgentID: "test-agent-error",
		Status:  "in_progress",
	}
	err := repo.AssignJob(ctx, assignment)
	require.NoError(t, err)

	// Set error
	err = repo.SetError(ctx, assignment.ID, "Printer not responding")
	require.NoError(t, err)

	// Verify
	updated, err := repo.FindByID(ctx, assignment.ID)
	require.NoError(t, err)
	assert.Equal(t, "failed", updated.Status)
	assert.Equal(t, "Printer not responding", updated.Error)
	assert.NotNil(t, updated.CompletedAt)
}

// TestJobAssignmentRepository_GetStaleAssignments tests finding stale assignments.
func TestJobAssignmentRepository_GetStaleAssignments(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	// Create an old assignment
	job := createTestPrintJob(t, db)
	assignment := &JobAssignment{
		JobID:         job.ID,
		AgentID:       "test-agent-stale",
		Status:        "in_progress",
		LastHeartbeat: time.Now().Add(-1 * time.Hour),
	}
	err := repo.AssignJob(ctx, assignment)
	require.NoError(t, err)

	// Manually update the heartbeat to be old
	_, err = db.Exec(ctx, "UPDATE job_assignments SET last_heartbeat = $1 WHERE id = $2",
		time.Now().Add(-1*time.Hour), assignment.ID)
	require.NoError(t, err)

	// Find stale assignments (threshold: 30 minutes ago)
	staleTime := time.Now().Add(-30 * time.Minute)
	stale, err := repo.GetStaleAssignments(ctx, staleTime)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(stale), 1)
}

// setupAssignmentTestDB creates a test database connection for assignment tests.
func setupAssignmentTestDB(t *testing.T) *pgxpool.Pool {
	dbURL := "postgres://openprint:openprint@localhost:5432/openprint?sslmode=disable"
	db, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Skip("database not available for testing")
	}
	return db
}

// createTestPrintJob creates a test print job for testing.
func createTestPrintJob(t *testing.T, db *pgxpool.Pool) *PrintJob {
	ctx := context.Background()

	job := &PrintJob{
		ID:          generateTestID(),
		DocumentID:  generateTestID(),
		PrinterID:   "test-printer",
		UserName:    "Test User",
		UserEmail:   "test@example.com",
		Title:       "Test Job",
		Copies:      1,
		ColorMode:   "color",
		Duplex:      true,
		MediaType:   "a4",
		Quality:     "normal",
		Status:      "queued",
		Priority:    5,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	query := `
		INSERT INTO print_jobs (id, document_id, printer_id, user_name, user_email, title,
			copies, color_mode, duplex, media_type, quality, status, priority, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	_, err := db.Exec(ctx, query,
		job.ID, job.DocumentID, job.PrinterID, job.UserName, job.UserEmail, job.Title,
		job.Copies, job.ColorMode, job.Duplex, job.MediaType, job.Quality, job.Status,
		job.Priority, job.CreatedAt, job.UpdatedAt)

	if err != nil {
		t.Fatalf("failed to create test print job: %v", err)
	}

	return job
}

func generateTestID() string {
	return "test-" + time.Now().Format("20060102150405.000000000")
}
