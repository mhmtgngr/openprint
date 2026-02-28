// Package repository provides tests for job history data access layer.
package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/openprint/openprint/internal/testutil"
)

func TestNewJobHistoryRepository(t *testing.T) {
	repo := NewJobHistoryRepository(nil)

	if repo == nil {
		t.Fatal("NewJobHistoryRepository() returned nil")
	}
}

func TestJobHistory_Struct(t *testing.T) {
	now := time.Now()

	history := &JobHistory{
		ID:        "hist-123",
		JobID:     "job-123",
		Status:    "queued",
		Message:   "Job created",
		CreatedAt: now,
	}

	if history.ID != "hist-123" {
		t.Error("JobHistory ID not set correctly")
	}
	if history.JobID != "job-123" {
		t.Error("JobHistory JobID not set correctly")
	}
	if history.Status != "queued" {
		t.Error("JobHistory Status should be queued")
	}
	if history.Message != "Job created" {
		t.Error("JobHistory Message not set correctly")
	}
	if history.CreatedAt.IsZero() {
		t.Error("JobHistory CreatedAt should not be zero")
	}
}

// Database-backed tests using testcontainers

func TestJobHistoryRepository_CRUD(t *testing.T) {
	repo := NewJobHistoryRepository(testDB.Pool)
	ctx := context.Background()

	// Create a test job first
	_, err := testutil.CreateTestOrganization(ctx, testDB.Pool)
	require.NoError(t, err)

	_, _, _, _, documentID, _, err := testutil.CreateFullTestSetup(ctx, testDB.Pool)
	require.NoError(t, err)

	_, _, _, printerID, _, _, err := testutil.CreateFullTestSetup(ctx, testDB.Pool)
	require.NoError(t, err)

	// Get a user email
	var userEmail string
	err = testDB.Pool.QueryRow(ctx, "SELECT email FROM users LIMIT 1").Scan(&userEmail)
	require.NoError(t, err)

	jobID, err := testutil.CreateTestPrintJob(ctx, testDB.Pool, documentID, printerID, userEmail)
	require.NoError(t, err)

	history := &JobHistory{
		JobID:     jobID,
		Status:    "queued",
		Message:   "Job created",
		CreatedAt: time.Now(),
	}

	t.Run("create history", func(t *testing.T) {
		err := repo.Create(ctx, history)
		require.NoError(t, err)
		assert.NotEmpty(t, history.ID)
	})

	t.Run("find by ID", func(t *testing.T) {
		found, err := repo.FindByID(ctx, history.ID)
		require.NoError(t, err)
		assert.Equal(t, history.ID, found.ID)
		assert.Equal(t, history.JobID, found.JobID)
		assert.Equal(t, history.Status, found.Status)
		assert.Equal(t, history.Message, found.Message)
	})

	t.Run("find by job ID", func(t *testing.T) {
		entries, err := repo.FindByJobID(ctx, jobID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(entries), 1)
		if len(entries) > 0 {
			assert.Equal(t, history.ID, entries[0].ID)
		}
	})

	t.Run("find by status", func(t *testing.T) {
		entries, err := repo.FindByStatus(ctx, "queued", 10, 0)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(entries), 1)
	})

	t.Run("delete by job ID", func(t *testing.T) {
		err := repo.DeleteByJobID(ctx, jobID)
		require.NoError(t, err)

		// Verify deletion
		_, err = repo.FindByID(ctx, history.ID)
		assert.Error(t, err)
	})
}

func TestJobHistoryRepository_QueryMethods(t *testing.T) {
	repo := NewJobHistoryRepository(testDB.Pool)
	ctx := context.Background()

	// Setup test data
	_, err := testutil.CreateTestOrganization(ctx, testDB.Pool)
	require.NoError(t, err)

	_, _, _, _, documentID, _, err := testutil.CreateFullTestSetup(ctx, testDB.Pool)
	require.NoError(t, err)

	_, _, _, printerID, _, _, err := testutil.CreateFullTestSetup(ctx, testDB.Pool)
	require.NoError(t, err)

	// Get a user email
	var userEmail string
	err = testDB.Pool.QueryRow(ctx, "SELECT email FROM users LIMIT 1").Scan(&userEmail)
	require.NoError(t, err)

	jobID, err := testutil.CreateTestPrintJob(ctx, testDB.Pool, documentID, printerID, userEmail)
	require.NoError(t, err)

	// Create multiple history entries
	entries := []*JobHistory{
		{JobID: jobID, Status: "queued", Message: "Job queued"},
		{JobID: jobID, Status: "processing", Message: "Job started"},
		{JobID: jobID, Status: "completed", Message: "Job finished"},
	}
	for _, e := range entries {
		e.CreatedAt = time.Now()
		err := repo.Create(ctx, e)
		require.NoError(t, err)
	}

	t.Run("get latest by job ID", func(t *testing.T) {
		history, err := repo.GetLatestByJobID(ctx, jobID)
		require.NoError(t, err)
		assert.NotNil(t, history)
		assert.Equal(t, jobID, history.JobID)
	})

	t.Run("count by job ID", func(t *testing.T) {
		count, err := repo.CountByJobID(ctx, jobID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, 3)
	})

	t.Run("list with pagination", func(t *testing.T) {
		list, total, err := repo.List(ctx, 10, 0)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 3)
		assert.GreaterOrEqual(t, len(list), 3)
	})
}

func TestJobHistoryRepository_BatchOperations(t *testing.T) {
	repo := NewJobHistoryRepository(testDB.Pool)
	ctx := context.Background()

	// Setup test data
	_, _, _, _, _, jobID, err := testutil.CreateFullTestSetup(ctx, testDB.Pool)
	require.NoError(t, err)

	entries := []*JobHistory{
		{JobID: jobID, Status: "queued", Message: "Job queued", CreatedAt: time.Now()},
		{JobID: jobID, Status: "processing", Message: "Job started", CreatedAt: time.Now()},
		{JobID: jobID, Status: "completed", Message: "Job finished", CreatedAt: time.Now()},
	}

	err = repo.CreateBatch(ctx, entries)
	require.NoError(t, err)

	// Verify all entries were created
	count, err := repo.CountByJobID(ctx, jobID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, 3)
}

func TestJobHistoryRepository_DeleteOld(t *testing.T) {
	repo := NewJobHistoryRepository(testDB.Pool)
	ctx := context.Background()

	// Setup test data
	_, _, _, _, _, jobID, err := testutil.CreateFullTestSetup(ctx, testDB.Pool)
	require.NoError(t, err)

	// Create old history entry (manually set created_at to past)
	oldEntry := &JobHistory{
		JobID:     jobID,
		Status:    "queued",
		Message:   "Old entry",
		CreatedAt: time.Now().Add(-48 * time.Hour),
	}
	err = repo.Create(ctx, oldEntry)
	require.NoError(t, err)

	// Create recent entry
	recentEntry := &JobHistory{
		JobID:     jobID,
		Status:    "processing",
		Message:   "Recent entry",
		CreatedAt: time.Now(),
	}
	err = repo.Create(ctx, recentEntry)
	require.NoError(t, err)

	// Delete entries older than 24 hours
	deleted, err := repo.DeleteOld(ctx, 24*time.Hour)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, deleted, int64(1))

	// Verify old entry is gone
	_, err = repo.FindByID(ctx, oldEntry.ID)
	assert.Error(t, err)

	// Verify recent entry still exists
	found, err := repo.FindByID(ctx, recentEntry.ID)
	require.NoError(t, err)
	assert.Equal(t, recentEntry.ID, found.ID)
}

func TestJobHistoryRepository_FindByID_NotFound(t *testing.T) {
	repo := NewJobHistoryRepository(testDB.Pool)
	ctx := context.Background()

	_, err := repo.FindByID(ctx, "non-existent-id")
	assert.Error(t, err)
}

func TestJobHistoryRepository_GetLatestByJobID_NotFound(t *testing.T) {
	repo := NewJobHistoryRepository(testDB.Pool)
	ctx := context.Background()

	history, err := repo.GetLatestByJobID(ctx, "00000000-0000-0000-0000-000000000001")
	assert.NoError(t, err)
	assert.Nil(t, history)
}

func TestJobHistoryRepository_CountByJobID_Zero(t *testing.T) {
	repo := NewJobHistoryRepository(testDB.Pool)
	ctx := context.Background()

	count, err := repo.CountByJobID(ctx, "00000000-0000-0000-0000-000000000001")
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestJobHistoryRepository_EmptyBatch(t *testing.T) {
	repo := NewJobHistoryRepository(testDB.Pool)
	ctx := context.Background()

	// Empty batch should not error
	err := repo.CreateBatch(ctx, []*JobHistory{})
	assert.NoError(t, err)
}

func TestJobHistory_StatusTransitions(t *testing.T) {
	validTransitions := []struct {
		from    string
		to      string
		message string
	}{
		{"queued", "processing", "Job assigned to processor"},
		{"processing", "pending_agent", "Job waiting for agent"},
		{"pending_agent", "processing", "Agent picked up job"},
		{"processing", "completed", "Job completed successfully"},
		{"processing", "failed", "Job failed: printer error"},
		{"queued", "cancelled", "Job cancelled by user"},
		{"paused", "queued", "Job resumed"},
	}

	for _, tt := range validTransitions {
		t.Run(tt.from+" to "+tt.to, func(t *testing.T) {
			history := &JobHistory{
				JobID:     "test-job-id",
				Status:    tt.to,
				Message:   tt.message,
				CreatedAt: time.Now(),
			}

			assert.Equal(t, tt.to, history.Status)
		})
	}
}

func TestJobHistory_Messages(t *testing.T) {
	testMessages := []struct {
		name    string
		message string
	}{
		{"simple message", "Job created"},
		{"message with detail", "Job failed: printer offline"},
		{"progress message", "Processing page 5 of 10"},
		{"completion message", "Job completed successfully in 2 minutes"},
		{"error message", "Error: insufficient paper in tray"},
	}

	for _, tt := range testMessages {
		t.Run(tt.name, func(t *testing.T) {
			history := &JobHistory{
				JobID:     "test-job-id",
				Status:    "processing",
				Message:   tt.message,
				CreatedAt: time.Now(),
			}

			assert.Equal(t, tt.message, history.Message)
		})
	}
}

func TestJobHistory_TimeFields(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

histories := []*JobHistory{
		{
			JobID:     "test-job-id",
			Status:    "queued",
			Message:   "Initial status",
			CreatedAt: past,
		},
		{
			JobID:     "test-job-id",
			Status:    "processing",
			Message:   "Started processing",
			CreatedAt: now,
		},
		{
			JobID:     "test-job-id",
			Status:    "completed",
			Message:   "Completed",
			CreatedAt: future,
		},
	}

	for i, h := range histories {
		if h.CreatedAt.IsZero() {
			t.Errorf("History %d: CreatedAt should not be zero", i)
		}
	}
}

func TestJobHistory_JobLifecycle(t *testing.T) {
	repo := NewJobHistoryRepository(testDB.Pool)
	ctx := context.Background()

	// Setup test data
	_, _, _, _, _, jobID, err := testutil.CreateFullTestSetup(ctx, testDB.Pool)
	require.NoError(t, err)

	// Simulate a complete job lifecycle through history entries
	lifecycle := []struct {
		status  string
		message string
	}{
		{"queued", "Job created and queued"},
		{"processing", "Job assigned to processor"},
		{"pending_agent", "Waiting for agent to accept"},
		{"processing", "Agent started printing"},
		{"completed", "Job finished successfully"},
	}

	var historyIDs []string
	for _, step := range lifecycle {
		history := &JobHistory{
				JobID:     jobID,
			Status:    step.status,
			Message:   step.message,
			CreatedAt: time.Now(),
		}
		err := repo.Create(ctx, history)
		require.NoError(t, err)
		historyIDs = append(historyIDs, history.ID)
	}

	// Verify all entries were created
	count, err := repo.CountByJobID(ctx, jobID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, len(lifecycle))

	// Verify we can retrieve all history entries
	entries, err := repo.FindByJobID(ctx, jobID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(entries), len(lifecycle))
}

func TestJobHistory_MultipleJobs(t *testing.T) {
	repo := NewJobHistoryRepository(testDB.Pool)
	ctx := context.Background()

	// Create multiple jobs with history
	var jobIDs []string
	for i := 0; i < 3; i++ {
		_, _, _, _, _, jobID, err := testutil.CreateFullTestSetup(ctx, testDB.Pool)
		require.NoError(t, err)
		jobIDs = append(jobIDs, jobID)

		history := &JobHistory{
				JobID:     jobID,
			Status:    "queued",
			Message:   "Job created",
			CreatedAt: time.Now(),
		}
		err = repo.Create(ctx, history)
		require.NoError(t, err)
	}

	// Verify each job has history
	for _, jobID := range jobIDs {
		count, err := repo.CountByJobID(ctx, jobID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, 1)
	}
}

func TestJobHistory_LongMessages(t *testing.T) {
	repo := NewJobHistoryRepository(testDB.Pool)
	ctx := context.Background()

	// Test handling of long messages
	longMessage := "This is a very long error message that contains detailed information about what went wrong during the printing process, including error codes, stack traces, and diagnostic information that might be useful for debugging purposes."

	// Setup test data
	_, _, _, _, _, jobID, err := testutil.CreateFullTestSetup(ctx, testDB.Pool)
	require.NoError(t, err)

	history := &JobHistory{
		JobID:     jobID,
		Status:    "failed",
		Message:   longMessage,
		CreatedAt: time.Now(),
	}

	err = repo.Create(ctx, history)
	require.NoError(t, err)

	// Retrieve and verify
	found, err := repo.FindByID(ctx, history.ID)
	require.NoError(t, err)
	assert.Equal(t, len(longMessage), len(found.Message))
}

func TestJobHistory_SpecialCharacters(t *testing.T) {
	repo := NewJobHistoryRepository(testDB.Pool)
	ctx := context.Background()

	messages := []string{
		"Job completed with \"quotes\"",
		"Error: printer's paper tray is empty",
		"Status: 100% complete\nNext: idle",
		"Price: $10.50 per page",
	}

	// Setup test data
	_, _, _, _, _, jobID, err := testutil.CreateFullTestSetup(ctx, testDB.Pool)
	require.NoError(t, err)

	for _, msg := range messages {
		history := &JobHistory{
				JobID:     jobID,
			Status:    "processing",
			Message:   msg,
			CreatedAt: time.Now(),
		}

		err := repo.Create(ctx, history)
		require.NoError(t, err)

		// Retrieve and verify
		found, err := repo.FindByID(ctx, history.ID)
		require.NoError(t, err)
		assert.Equal(t, msg, found.Message)
	}
}

func TestJobHistoryRepository_FindByStatus_Pagination(t *testing.T) {
	repo := NewJobHistoryRepository(testDB.Pool)
	ctx := context.Background()

	// Setup test data
	_, _, _, _, _, jobID, err := testutil.CreateFullTestSetup(ctx, testDB.Pool)
	require.NoError(t, err)

	// Create multiple entries with same status
	status := "queued"
	for i := 0; i < 5; i++ {
		history := &JobHistory{
				JobID:     jobID,
			Status:    status,
			Message:   "Test entry",
			CreatedAt: time.Now(),
		}
		err := repo.Create(ctx, history)
		require.NoError(t, err)
	}

	// Test pagination
	t.Run("first page", func(t *testing.T) {
		entries, err := repo.FindByStatus(ctx, status, 2, 0)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(entries), 2)
	})

	t.Run("second page", func(t *testing.T) {
		entries, err := repo.FindByStatus(ctx, status, 2, 2)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(entries), 2)
	})
}

func TestJobHistoryRepository_List_Pagination(t *testing.T) {
	repo := NewJobHistoryRepository(testDB.Pool)
	ctx := context.Background()

	// Create a bunch of history entries
	for i := 0; i < 15; i++ {
		_, _, _, _, _, jobID, err := testutil.CreateFullTestSetup(ctx, testDB.Pool)
		require.NoError(t, err)

		history := &JobHistory{
				JobID:     jobID,
			Status:    "queued",
			Message:   "Test entry",
			CreatedAt: time.Now(),
		}
		err = repo.Create(ctx, history)
		require.NoError(t, err)
	}

	t.Run("first page", func(t *testing.T) {
		list, total, err := repo.List(ctx, 10, 0)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 15)
		assert.LessOrEqual(t, len(list), 10)
	})

	t.Run("second page", func(t *testing.T) {
		list, total, err := repo.List(ctx, 10, 10)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 15)
		assert.LessOrEqual(t, len(list), 10)
	})
}

func TestJobHistoryRepository_Update_Status(t *testing.T) {
	repo := NewJobHistoryRepository(testDB.Pool)
	ctx := context.Background()

	// Setup test data
	_, _, _, _, _, jobID, err := testutil.CreateFullTestSetup(ctx, testDB.Pool)
	require.NoError(t, err)

	history := &JobHistory{
		JobID:     jobID,
		Status:    "queued",
		Message:   "Initial status",
		CreatedAt: time.Now(),
	}
	err = repo.Create(ctx, history)
	require.NoError(t, err)

	// History entries are immutable - we create new entries for status changes
	newHistory := &JobHistory{
		JobID:     jobID,
		Status:    "processing",
		Message:   "Status changed to processing",
		CreatedAt: time.Now(),
	}
	err = repo.Create(ctx, newHistory)
	require.NoError(t, err)

	// Verify we have two history entries
	count, err := repo.CountByJobID(ctx, jobID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, 2)
}
