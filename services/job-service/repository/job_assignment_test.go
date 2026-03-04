// Package repository provides unit tests for job assignment operations.
package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openprint/openprint/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupAssignmentTestDB creates a test database connection for assignment tests.
// It now uses the shared testDB from TestMain.
func setupAssignmentTestDB(t *testing.T) *pgxpool.Pool {
	return testDB.Pool
}

// createTestPrintJob creates a test print job for testing.
// It now uses the testutil fixtures for complete setup.
func createTestPrintJob(t *testing.T, db *pgxpool.Pool) *PrintJob {
	ctx := context.Background()

	// Create a complete test setup
	_, _, _, _, documentID, _, err := testutil.CreateFullTestSetup(ctx, db)
	require.NoError(t, err, "Failed to create test setup")

	// Create an additional printer for the job
	orgID, _, _, printerID, _, _, err := testutil.CreateFullTestSetup(ctx, db)
	require.NoError(t, err, "Failed to create printer setup")

	// Get a user email
	var userEmail string
	err = db.QueryRow(ctx, "SELECT email FROM users WHERE organization_id = $1 LIMIT 1", orgID).Scan(&userEmail)
	require.NoError(t, err)

	job := &PrintJob{
		DocumentID: documentID,
		PrinterID:  printerID,
		UserName:   "Test User",
		UserEmail:  userEmail,
		Title:      "Test Job",
		Copies:     1,
		ColorMode:  "color",
		Duplex:     true,
		MediaType:  "a4",
		Quality:    "normal",
		Status:     "queued",
		Priority:   5,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	query := `
		INSERT INTO print_jobs (document_id, printer_id, user_name, user_email, title,
			copies, color_mode, duplex, media_type, quality, status, priority, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id
	`

	err = db.QueryRow(ctx, query,
		job.DocumentID, job.PrinterID, job.UserName, job.UserEmail, job.Title,
		job.Copies, job.ColorMode, job.Duplex, job.MediaType, job.Quality, job.Status,
		job.Priority, job.CreatedAt, job.UpdatedAt).Scan(&job.ID)

	require.NoError(t, err, "Failed to create test print job")

	return job
}

// TestJobAssignmentRepository_AssignJob tests assigning a job to an agent.
func TestJobAssignmentRepository_AssignJob(t *testing.T) {
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	// Create a test print job first
	job := createTestPrintJob(t, db)

	// Create a test agent
	orgID, err := testutil.CreateTestOrganization(ctx, db)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, db, orgID)
	require.NoError(t, err)

	assignment := &JobAssignment{
		JobID:   job.ID,
		AgentID: agentID,
		Status:  "assigned",
	}

	err = repo.AssignJob(ctx, assignment)
	require.NoError(t, err)
	assert.NotEmpty(t, assignment.ID)
	assert.NotEmpty(t, assignment.AssignedAt)
	assert.Equal(t, "assigned", assignment.Status)
}

// TestJobAssignmentRepository_UpdateStatus tests updating assignment status.
func TestJobAssignmentRepository_UpdateStatus(t *testing.T) {
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	// Create an assignment
	job := createTestPrintJob(t, db)

	orgID, err := testutil.CreateTestOrganization(ctx, db)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, db, orgID)
	require.NoError(t, err)

	assignment := &JobAssignment{
		JobID:   job.ID,
		AgentID: agentID,
		Status:  "assigned",
	}
	err = repo.AssignJob(ctx, assignment)
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
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	// Create an assignment
	job := createTestPrintJob(t, db)

	orgID, err := testutil.CreateTestOrganization(ctx, db)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, db, orgID)
	require.NoError(t, err)

	assignment := &JobAssignment{
		JobID:   job.ID,
		AgentID: agentID,
		Status:  "assigned",
	}
	err = repo.AssignJob(ctx, assignment)
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
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	// Create multiple assignments for the same agent
	orgID, err := testutil.CreateTestOrganization(ctx, db)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, db, orgID)
	require.NoError(t, err)

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
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	job := createTestPrintJob(t, db)

	orgID, err := testutil.CreateTestOrganization(ctx, db)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, db, orgID)
	require.NoError(t, err)

	assignment := &JobAssignment{
		JobID:   job.ID,
		AgentID: agentID,
		Status:  "assigned",
	}
	err = repo.AssignJob(ctx, assignment)
	require.NoError(t, err)

	// Find the assignment
	found, err := repo.FindByJobAndAgent(ctx, job.ID, agentID)
	require.NoError(t, err)
	assert.Equal(t, assignment.ID, found.ID)
	assert.Equal(t, job.ID, found.JobID)
	assert.Equal(t, agentID, found.AgentID)
}

// TestJobAssignmentRepository_UpdateHeartbeat tests updating heartbeat timestamp.
func TestJobAssignmentRepository_UpdateHeartbeat(t *testing.T) {
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	job := createTestPrintJob(t, db)

	orgID, err := testutil.CreateTestOrganization(ctx, db)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, db, orgID)
	require.NoError(t, err)

	assignment := &JobAssignment{
		JobID:   job.ID,
		AgentID: agentID,
		Status:  "in_progress",
	}
	err = repo.AssignJob(ctx, assignment)
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
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	job := createTestPrintJob(t, db)

	orgID, err := testutil.CreateTestOrganization(ctx, db)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, db, orgID)
	require.NoError(t, err)

	assignment := &JobAssignment{
		JobID:      job.ID,
		AgentID:    agentID,
		Status:     "assigned",
		RetryCount: 0,
	}
	err = repo.AssignJob(ctx, assignment)
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
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	job := createTestPrintJob(t, db)

	orgID, err := testutil.CreateTestOrganization(ctx, db)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, db, orgID)
	require.NoError(t, err)

	assignment := &JobAssignment{
		JobID:   job.ID,
		AgentID: agentID,
		Status:  "in_progress",
	}
	err = repo.AssignJob(ctx, assignment)
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
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	// Create an old assignment
	job := createTestPrintJob(t, db)

	orgID, err := testutil.CreateTestOrganization(ctx, db)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, db, orgID)
	require.NoError(t, err)

	assignment := &JobAssignment{
		JobID:         job.ID,
		AgentID:       agentID,
		Status:        "in_progress",
		LastHeartbeat: time.Now().Add(-1 * time.Hour),
	}
	err = repo.AssignJob(ctx, assignment)
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

// Additional tests for edge cases and error conditions

func TestJobAssignmentRepository_AssignJob_Conflict(t *testing.T) {
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	// Create test data
	job := createTestPrintJob(t, db)

	orgID, err := testutil.CreateTestOrganization(ctx, db)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, db, orgID)
	require.NoError(t, err)

	// Create first assignment
	assignment1 := &JobAssignment{
		JobID:   job.ID,
		AgentID: agentID,
		Status:  "assigned",
	}
	err = repo.AssignJob(ctx, assignment1)
	require.NoError(t, err)

	// Try to create duplicate assignment (should fail due to unique constraint)
	assignment2 := &JobAssignment{
		JobID:   job.ID,
		AgentID: agentID,
		Status:  "assigned",
	}
	err = repo.AssignJob(ctx, assignment2)
	assert.Error(t, err)
}

func TestJobAssignmentRepository_FindByID_NotFound(t *testing.T) {
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	// Try to find non-existent assignment
	_, err := repo.FindByID(ctx, "00000000-0000-0000-0000-000000000001")
	assert.Error(t, err)
}

func TestJobAssignmentRepository_Delete(t *testing.T) {
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	// Create test data
	job := createTestPrintJob(t, db)

	orgID, err := testutil.CreateTestOrganization(ctx, db)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, db, orgID)
	require.NoError(t, err)

	// Create assignment
	assignment := &JobAssignment{
		JobID:   job.ID,
		AgentID: agentID,
		Status:  "assigned",
	}
	err = repo.AssignJob(ctx, assignment)
	require.NoError(t, err)

	// Delete assignment
	err = repo.Delete(ctx, assignment.ID)
	require.NoError(t, err)

	// Verify deletion
	_, err = repo.FindByID(ctx, assignment.ID)
	assert.Error(t, err)
}

func TestJobAssignmentRepository_DeleteByJob(t *testing.T) {
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	// Create test data
	job := createTestPrintJob(t, db)

	orgID, err := testutil.CreateTestOrganization(ctx, db)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, db, orgID)
	require.NoError(t, err)

	// Create multiple assignments for the same job
	agentID2, err := testutil.CreateTestAgent(ctx, db, orgID)
	require.NoError(t, err)

	assignment1 := &JobAssignment{
		JobID:   job.ID,
		AgentID: agentID,
		Status:  "assigned",
	}
	err = repo.AssignJob(ctx, assignment1)
	require.NoError(t, err)

	// Cancel first assignment to allow second
	err = repo.UpdateStatus(ctx, assignment1.ID, "cancelled")
	require.NoError(t, err)

	assignment2 := &JobAssignment{
		JobID:   job.ID,
		AgentID: agentID2,
		Status:  "assigned",
	}
	err = repo.AssignJob(ctx, assignment2)
	require.NoError(t, err)

	// Delete all assignments for the job
	err = repo.DeleteByJob(ctx, job.ID)
	require.NoError(t, err)

	// Verify deletion
	assignments, err := repo.FindByJobID(ctx, job.ID)
	require.NoError(t, err)
	assert.Empty(t, assignments)
}

func TestJobAssignmentRepository_AssignmentStatuses(t *testing.T) {
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	// Test all valid status transitions
	validStatuses := []string{"assigned", "in_progress", "completed", "failed", "cancelled"}

	for _, status := range validStatuses {
		t.Run("Status_"+status, func(t *testing.T) {
			// Create test data for each status
			job := createTestPrintJob(t, db)

			orgID, err := testutil.CreateTestOrganization(ctx, db)
			require.NoError(t, err)

			agentID, err := testutil.CreateTestAgent(ctx, db, orgID)
			require.NoError(t, err)

			assignment := &JobAssignment{
				JobID:   job.ID,
				AgentID: agentID,
				Status:  "assigned",
			}
			err = repo.AssignJob(ctx, assignment)
			require.NoError(t, err)

			// Update to the target status
			err = repo.UpdateStatus(ctx, assignment.ID, status)
			require.NoError(t, err)

			// Verify the status
			updated, err := repo.FindByID(ctx, assignment.ID)
			require.NoError(t, err)
			assert.Equal(t, status, updated.Status)
		})
	}
}

func TestJobAssignmentRepository_GetAssignmentStats(t *testing.T) {
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	// Create test data
	orgID, err := testutil.CreateTestOrganization(ctx, db)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, db, orgID)
	require.NoError(t, err)

	// Create assignments with different statuses
	for _, status := range []string{"assigned", "in_progress", "completed"} {
		job := createTestPrintJob(t, db)
		assignment := &JobAssignment{
			JobID:   job.ID,
			AgentID: agentID,
			Status:  status,
		}
		err = repo.AssignJob(ctx, assignment)
		require.NoError(t, err)
	}

	// Get stats for the agent
	stats, err := repo.GetAssignmentStats(ctx, agentID)
	require.NoError(t, err)
	assert.NotEmpty(t, stats)
	assert.Equal(t, int64(1), stats["assigned"])
	assert.Equal(t, int64(1), stats["in_progress"])
	assert.Equal(t, int64(1), stats["completed"])
}

func TestJobAssignmentRepository_FindByJobID(t *testing.T) {
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	// Create test data
	job := createTestPrintJob(t, db)

	orgID, err := testutil.CreateTestOrganization(ctx, db)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, db, orgID)
	require.NoError(t, err)

	// Create multiple assignments for the same job
	agentID2, err := testutil.CreateTestAgent(ctx, db, orgID)
	require.NoError(t, err)

	assignment1 := &JobAssignment{
		JobID:   job.ID,
		AgentID: agentID,
		Status:  "assigned",
	}
	err = repo.AssignJob(ctx, assignment1)
	require.NoError(t, err)

	// Cancel first assignment
	err = repo.UpdateStatus(ctx, assignment1.ID, "cancelled")
	require.NoError(t, err)

	assignment2 := &JobAssignment{
		JobID:   job.ID,
		AgentID: agentID2,
		Status:  "assigned",
	}
	err = repo.AssignJob(ctx, assignment2)
	require.NoError(t, err)

	// Find all assignments for the job
	assignments, err := repo.FindByJobID(ctx, job.ID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(assignments), 2)
}

func TestJobAssignmentRepository_UpdateStatus_NotFound(t *testing.T) {
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	// Try to update non-existent assignment
	err := repo.UpdateStatus(ctx, "00000000-0000-0000-0000-000000000001", "in_progress")
	assert.Error(t, err)
}

func TestJobAssignmentRepository_UpdateHeartbeat_NotFound(t *testing.T) {
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	// Try to update heartbeat for non-existent assignment
	err := repo.UpdateHeartbeat(ctx, "00000000-0000-0000-0000-000000000001", time.Now())
	assert.Error(t, err)
}

func TestJobAssignmentRepository_IncrementRetry_NotFound(t *testing.T) {
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	// Try to increment retry for non-existent assignment
	err := repo.IncrementRetry(ctx, "00000000-0000-0000-0000-000000000001")
	assert.Error(t, err)
}

func TestJobAssignmentRepository_SetError_NotFound(t *testing.T) {
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	// Try to set error for non-existent assignment
	err := repo.SetError(ctx, "00000000-0000-0000-0000-000000000001", "Test error")
	assert.Error(t, err)
}

func TestJobAssignmentRepository_Delete_NotFound(t *testing.T) {
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	// Try to delete non-existent assignment
	err := repo.Delete(ctx, "00000000-0000-0000-0000-000000000001")
	assert.Error(t, err)
}

func TestJobAssignmentRepository_UpdateNilTimestamp(t *testing.T) {
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	// Create test data
	job := createTestPrintJob(t, db)

	orgID, err := testutil.CreateTestOrganization(ctx, db)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, db, orgID)
	require.NoError(t, err)

	assignment := &JobAssignment{
		JobID:       job.ID,
		AgentID:     agentID,
		Status:      "assigned",
		StartedAt:   nil,
		CompletedAt: nil,
	}
	err = repo.AssignJob(ctx, assignment)
	require.NoError(t, err)

	// Verify nil timestamps
	fetched, err := repo.FindByID(ctx, assignment.ID)
	require.NoError(t, err)
	assert.Nil(t, fetched.StartedAt)
	assert.Nil(t, fetched.CompletedAt)

	// Update to in_progress should set StartedAt
	err = repo.UpdateStatus(ctx, assignment.ID, "in_progress")
	require.NoError(t, err)

	fetched, err = repo.FindByID(ctx, assignment.ID)
	require.NoError(t, err)
	assert.NotNil(t, fetched.StartedAt)
	assert.Nil(t, fetched.CompletedAt)
}

func TestJobAssignmentRepository_ScanAssignmentWithNulls(t *testing.T) {
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	// Create test data
	job := createTestPrintJob(t, db)

	orgID, err := testutil.CreateTestOrganization(ctx, db)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, db, orgID)
	require.NoError(t, err)

	// Insert assignment with NULL fields directly
	query := `
		INSERT INTO job_assignments (id, job_id, agent_id, status, assigned_at, created_at, updated_at, last_heartbeat)
		VALUES ($1, $2, $3, $4, NOW(), NOW(), NOW(), NOW())
	`
	testID := uuid.New().String()
	_, err = db.Exec(ctx, query, testID, job.ID, agentID, "assigned")
	require.NoError(t, err)

	// Scan the assignment
	assignment, err := repo.FindByID(ctx, testID)
	require.NoError(t, err)
	assert.NotNil(t, assignment)
	assert.Equal(t, testID, assignment.ID)
	assert.Nil(t, assignment.StartedAt)
	assert.Nil(t, assignment.CompletedAt)
	assert.Equal(t, "", assignment.Error)
	assert.Equal(t, "", assignment.DocumentETag)
}

// TestJobAssignmentRepository_DocumentETag tests the document ETag field for resume support.
func TestJobAssignmentRepository_DocumentETag(t *testing.T) {
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	// Create test data
	job := createTestPrintJob(t, db)

	orgID, err := testutil.CreateTestOrganization(ctx, db)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, db, orgID)
	require.NoError(t, err)

	// Create assignment with ETag
	assignment := &JobAssignment{
		JobID:        job.ID,
		AgentID:      agentID,
		Status:       "assigned",
		DocumentETag: "abc123etag",
	}
	err = repo.AssignJob(ctx, assignment)
	require.NoError(t, err)

	// Update the ETag directly
	_, err = db.Exec(ctx, "UPDATE job_assignments SET document_etag = $1 WHERE id = $2",
		"updated-etag-456", assignment.ID)
	require.NoError(t, err)

	// Verify the ETag was updated
	fetched, err := repo.FindByID(ctx, assignment.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated-etag-456", fetched.DocumentETag)
}

// TestJobAssignmentRepository_TransactionIsolation tests that transactions work correctly.
func TestJobAssignmentRepository_TransactionIsolation(t *testing.T) {
	db := setupAssignmentTestDB(t)
	repo := NewJobAssignmentRepository(db)

	// Create test data
	job := createTestPrintJob(t, db)

	orgID, err := testutil.CreateTestOrganization(ctx, db)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, db, orgID)
	require.NoError(t, err)

	// Start a transaction but don't commit
	tx, err := db.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	// Insert assignment within transaction
	testID := uuid.New().String()
	_, err = tx.Exec(ctx, `
		INSERT INTO job_assignments (id, job_id, agent_id, status, assigned_at, created_at, updated_at, last_heartbeat)
		VALUES ($1, $2, $3, $4, NOW(), NOW(), NOW(), NOW())
	`, testID, job.ID, agentID, "assigned")
	require.NoError(t, err)

	// Assignment should not be visible outside transaction
	_, err = repo.FindByID(ctx, testID)
	assert.Error(t, err)

	// Commit transaction
	err = tx.Commit(ctx)
	require.NoError(t, err)

	// Now assignment should be visible
	_, err = repo.FindByID(ctx, testID)
	assert.NoError(t, err)
}
