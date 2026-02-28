// Package repository provides tests for print job data access layer.
package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/openprint/openprint/internal/testutil"
)

func TestNewJobRepository(t *testing.T) {
	repo := NewJobRepository(nil)

	if repo == nil {
		t.Fatal("NewJobRepository() returned nil")
	}
}

func TestPrintJob_Struct(t *testing.T) {
	now := time.Now()
	completedAt := now

	job := &PrintJob{
		ID:          "job-123",
		DocumentID:  "doc-123",
		PrinterID:   "printer-123",
		UserName:    "testuser",
		UserEmail:   "test@example.com",
		Title:       "Test Document",
		Copies:      2,
		ColorMode:   "color",
		Duplex:      true,
		MediaType:   "a4",
		Quality:     "normal",
		Pages:       10,
		Status:      "queued",
		Priority:    5,
		Retries:     0,
		AgentID:     "agent-123",
		StartedAt:   now,
		CompletedAt: &completedAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if job.ID != "job-123" {
		t.Error("PrintJob ID not set correctly")
	}
	if job.Status != "queued" {
		t.Error("PrintJob Status should be queued")
	}
	if job.Copies != 2 {
		t.Error("PrintJob Copies should be 2")
	}
	if job.Priority != 5 {
		t.Error("PrintJob Priority should be 5")
	}
	if job.CompletedAt == nil {
		t.Error("PrintJob CompletedAt should not be nil")
	}
}

func TestPrintJob_StatusValues(t *testing.T) {
	validStatuses := []string{
		"queued", "processing", "pending_agent", "completed", "failed", "cancelled", "paused",
	}

	for _, status := range validStatuses {
		job := &PrintJob{Status: status}
		if job.Status != status {
			t.Errorf("PrintJob status not set correctly to %s", status)
		}
	}
}

func TestPrintJob_ColorMode(t *testing.T) {
	validModes := []string{"color", "monochrome"}

	for _, mode := range validModes {
		job := &PrintJob{ColorMode: mode}
		if job.ColorMode != mode {
			t.Errorf("PrintJob ColorMode not set correctly to %s", mode)
		}
	}
}

func TestPrintJob_PriorityRange(t *testing.T) {
	t.Run("valid priorities", func(t *testing.T) {
		for priority := 1; priority <= 10; priority++ {
			job := &PrintJob{Priority: priority}
			if job.Priority < 1 || job.Priority > 10 {
				t.Errorf("Priority %d is out of range [1-10]", priority)
			}
		}
	})

	t.Run("priority 1 is lowest", func(t *testing.T) {
		job1 := &PrintJob{Priority: 1}
		job10 := &PrintJob{Priority: 10}

		if job1.Priority >= job10.Priority {
			t.Error("Priority 1 should be less than Priority 10")
		}
	})
}

// Database-backed tests using testcontainers

func TestJobRepository_CRUD(t *testing.T) {
	repo := NewJobRepository(testDB.Pool)
	ctx := context.Background()

	// Setup test data
	orgID, err := testutil.CreateTestOrganization(ctx, testDB.Pool)
	require.NoError(t, err)

	_, _, _, _, documentID, _, err := testutil.CreateFullTestSetup(ctx, testDB.Pool)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, testDB.Pool, orgID)
	require.NoError(t, err)

	printerID, err := testutil.CreateTestPrinter(ctx, testDB.Pool, agentID)
	require.NoError(t, err)

	// Get a user email
	var userEmail string
	err = testDB.Pool.QueryRow(ctx, "SELECT email FROM users LIMIT 1").Scan(&userEmail)
	require.NoError(t, err)

	job := &PrintJob{
		DocumentID: documentID,
		PrinterID:  printerID,
		UserEmail:  userEmail,
		Title:      "Test Document",
		Status:     "queued",
		Priority:   5,
		Options:    "{}",
	}

	t.Run("create job", func(t *testing.T) {
		err := repo.Create(ctx, job)
		require.NoError(t, err)
		assert.NotEmpty(t, job.ID)
	})

	t.Run("find by ID", func(t *testing.T) {
		found, err := repo.FindByID(ctx, job.ID)
		require.NoError(t, err)
		assert.Equal(t, job.ID, found.ID)
		assert.Equal(t, job.Title, found.Title)
		assert.Equal(t, job.UserEmail, found.UserEmail)
	})

	t.Run("update job", func(t *testing.T) {
		job.Title = "Updated Title"
		job.Copies = 2
		err := repo.Update(ctx, job)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, job.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Title", found.Title)
		assert.Equal(t, 2, found.Copies)
	})

	t.Run("update status", func(t *testing.T) {
		err := repo.UpdateStatus(ctx, job.ID, "processing")
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, job.ID)
		require.NoError(t, err)
		assert.Equal(t, "processing", found.Status)
	})

	t.Run("assign agent", func(t *testing.T) {
		err := repo.AssignAgent(ctx, job.ID, agentID)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, job.ID)
		require.NoError(t, err)
		assert.Equal(t, agentID, found.AgentID)
	})

	t.Run("delete job", func(t *testing.T) {
		err := repo.Delete(ctx, job.ID)
		require.NoError(t, err)

		_, err = repo.FindByID(ctx, job.ID)
		assert.Error(t, err)
	})
}

func TestJobRepository_QueryMethods(t *testing.T) {
	repo := NewJobRepository(testDB.Pool)
	ctx := context.Background()

	// Setup test data
	orgID, err := testutil.CreateTestOrganization(ctx, testDB.Pool)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, testDB.Pool, orgID)
	require.NoError(t, err)

	printerID, err := testutil.CreateTestPrinter(ctx, testDB.Pool, agentID)
	require.NoError(t, err)

	// Get a user email
	var userEmail string
	err = testDB.Pool.QueryRow(ctx, "SELECT email FROM users LIMIT 1").Scan(&userEmail)
	require.NoError(t, err)

	// Create multiple jobs with different statuses
	statuses := []string{"queued", "processing", "completed"}
	var jobIDs []string

	for _, status := range statuses {
		_, _, _, _, documentID, _, err := testutil.CreateFullTestSetup(ctx, testDB.Pool)
		require.NoError(t, err)

		job := &PrintJob{
			DocumentID: documentID,
			PrinterID:  printerID,
			UserEmail:  userEmail,
			Title:      "Test Job " + status,
			Status:     status,
			Priority:   5,
		}
		err = repo.Create(ctx, job)
		require.NoError(t, err)
		jobIDs = append(jobIDs, job.ID)
	}

	t.Run("find by status", func(t *testing.T) {
		jobs, err := repo.FindByStatus(ctx, "queued", 10)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(jobs), 1)
		if len(jobs) > 0 {
			assert.Equal(t, "queued", jobs[0].Status)
		}
	})

	t.Run("find by printer", func(t *testing.T) {
		jobs, err := repo.FindByPrinter(ctx, printerID, 10, 0)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(jobs), 1)
		if len(jobs) > 0 {
			assert.Equal(t, printerID, jobs[0].PrinterID)
		}
	})

	t.Run("find by user", func(t *testing.T) {
		jobs, err := repo.FindByUser(ctx, userEmail, 10, 0)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(jobs), 1)
		if len(jobs) > 0 {
			assert.Equal(t, userEmail, jobs[0].UserEmail)
		}
	})

	t.Run("list with filters", func(t *testing.T) {
		jobs, total, err := repo.ListWithFilters(ctx, 10, 0, printerID, "", "")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(1))
		assert.GreaterOrEqual(t, len(jobs), 1)
	})

	t.Run("count by status", func(t *testing.T) {
		count, err := repo.CountByStatus(ctx, "queued")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(1))
	})
}

func TestJobRepository_AdvancedOperations(t *testing.T) {
	repo := NewJobRepository(testDB.Pool)
	ctx := context.Background()

	// Setup test data
	orgID, err := testutil.CreateTestOrganization(ctx, testDB.Pool)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, testDB.Pool, orgID)
	require.NoError(t, err)

	printerID, err := testutil.CreateTestPrinter(ctx, testDB.Pool, agentID)
	require.NoError(t, err)

	// Get a user email
	var userEmail string
	err = testDB.Pool.QueryRow(ctx, "SELECT email FROM users LIMIT 1").Scan(&userEmail)
	require.NoError(t, err)

	// Create a pending job
	_, _, _, _, documentID, _, err := testutil.CreateFullTestSetup(ctx, testDB.Pool)
	require.NoError(t, err)

	job := &PrintJob{
		DocumentID: documentID,
		PrinterID:  printerID,
		UserEmail:  userEmail,
		Title:      "Pending Job",
		Status:     "queued",
		Priority:   5,
	}
	err = repo.Create(ctx, job)
	require.NoError(t, err)

	t.Run("get next pending job", func(t *testing.T) {
		pending, err := repo.GetNextPendingJob(ctx, printerID)
		require.NoError(t, err)
		if pending != nil {
			assert.Equal(t, "queued", pending.Status)
		}
	})

	t.Run("update job progress", func(t *testing.T) {
		err := repo.UpdateJobProgress(ctx, job.ID, 5)
		require.NoError(t, err)

		updated, err := repo.FindByID(ctx, job.ID)
		require.NoError(t, err)
		assert.Equal(t, 5, updated.Pages)
	})

	t.Run("create failed job for retry", func(t *testing.T) {
		_, _, _, _, documentID2, _, err := testutil.CreateFullTestSetup(ctx, testDB.Pool)
		require.NoError(t, err)

		failedJob := &PrintJob{
			DocumentID: documentID2,
			PrinterID:  printerID,
			UserEmail:  userEmail,
			Title:      "Failed Job",
			Status:     "failed",
			Priority:   5,
			Retries:    1,
		}
		err = repo.Create(ctx, failedJob)
		require.NoError(t, err)
	})

	t.Run("get jobs needing retry", func(t *testing.T) {
		jobs, err := repo.GetJobsNeedingRetry(ctx, 3, 10)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(jobs), 1)
		if len(jobs) > 0 {
			assert.Equal(t, "failed", jobs[0].Status)
		}
	})
}

func TestPrintJob_TimeFields(t *testing.T) {
	now := time.Now()
	completedAt := now

	job := &PrintJob{
		StartedAt:    now,
		CompletedAt:  &completedAt,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if job.StartedAt.IsZero() {
		t.Error("StartedAt should not be zero")
	}
	if job.CompletedAt == nil {
		t.Error("CompletedAt should not be nil")
	}
	if job.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
	if job.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero")
	}
}

func TestPrintJob_CompletionTracking(t *testing.T) {
	t.Run("completed job", func(t *testing.T) {
		now := time.Now()
		job := &PrintJob{
			Status:      "completed",
			CompletedAt: &now,
		}

		if job.CompletedAt == nil {
			t.Error("Completed job should have CompletedAt set")
		}
	})

	t.Run("incomplete job", func(t *testing.T) {
		job := &PrintJob{
			Status:      "queued",
			CompletedAt: nil,
		}

		if job.CompletedAt != nil {
			t.Error("Incomplete job should not have CompletedAt set")
		}
	})
}

func TestPrintJob_Retries(t *testing.T) {
	job := &PrintJob{
		Retries: 0,
	}

	for i := 0; i < 5; i++ {
		job.Retries++
		if job.Retries != i+1 {
			t.Errorf("Retries increment failed: expected %d, got %d", i+1, job.Retries)
		}
	}

	if job.Retries != 5 {
		t.Errorf("Final Retries count should be 5, got %d", job.Retries)
	}
}

func TestPrintJob_Options(t *testing.T) {
	t.Run("empty options", func(t *testing.T) {
		job := &PrintJob{
			Options: "",
		}

		if job.Options != "" {
			t.Error("Options should be empty")
		}
	})

	t.Run("JSON options", func(t *testing.T) {
		options := `{"two_sided": true, "staple": true}`
		job := &PrintJob{
			Options: options,
		}

		if job.Options != options {
			t.Error("Options not set correctly")
		}
	})
}

func TestPrintJob_UserFields(t *testing.T) {
	job := &PrintJob{
		UserName:  "John Doe",
		UserEmail: "john@example.com",
	}

	if job.UserName == "" {
		t.Error("UserName should not be empty")
	}
	if job.UserEmail == "" {
		t.Error("UserEmail should not be empty")
	}
}

func TestPrintJob_PrinterAssociation(t *testing.T) {
	job := &PrintJob{
		PrinterID: "printer-123",
		AgentID:   "agent-123",
	}

	if job.PrinterID == "" {
		t.Error("PrinterID should be set")
	}
	if job.AgentID == "" {
		t.Error("AgentID should be set")
	}
}

func TestJobRepository_ListWithFilters_EmptyFilters(t *testing.T) {
	repo := NewJobRepository(testDB.Pool)
	ctx := context.Background()

	// All filters empty should return all jobs
	_, _, err := repo.ListWithFilters(ctx, 10, 0, "", "", "")
	require.NoError(t, err)
}

func TestJobRepository_ListWithFilters_SingleFilter(t *testing.T) {
	repo := NewJobRepository(testDB.Pool)
	ctx := context.Background()

	// Setup test data
	orgID, err := testutil.CreateTestOrganization(ctx, testDB.Pool)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, testDB.Pool, orgID)
	require.NoError(t, err)

	printerID, err := testutil.CreateTestPrinter(ctx, testDB.Pool, agentID)
	require.NoError(t, err)

	// Get a user email
	var userEmail string
	err = testDB.Pool.QueryRow(ctx, "SELECT email FROM users LIMIT 1").Scan(&userEmail)
	require.NoError(t, err)

	_, _, _, _, documentID, _, err := testutil.CreateFullTestSetup(ctx, testDB.Pool)
	require.NoError(t, err)

	job := &PrintJob{
		DocumentID: documentID,
		PrinterID:  printerID,
		UserEmail:  userEmail,
		Title:      "Test Job",
		Status:     "queued",
		Priority:   5,
	}
	err = repo.Create(ctx, job)
	require.NoError(t, err)

	t.Run("filter by printer only", func(t *testing.T) {
		jobs, total, err := repo.ListWithFilters(ctx, 10, 0, printerID, "", "")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(1))
		assert.GreaterOrEqual(t, len(jobs), 1)
	})

	t.Run("filter by status only", func(t *testing.T) {
		jobs, total, err := repo.ListWithFilters(ctx, 10, 0, "", "queued", "")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(1))
		assert.GreaterOrEqual(t, len(jobs), 1)
	})

	t.Run("filter by user only", func(t *testing.T) {
		jobs, total, err := repo.ListWithFilters(ctx, 10, 0, "", "", userEmail)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(1))
		assert.GreaterOrEqual(t, len(jobs), 1)
	})
}

func TestJobRepository_DuplicateHandling(t *testing.T) {
	repo := NewJobRepository(testDB.Pool)
	ctx := context.Background()

	// Setup test data
	orgID, err := testutil.CreateTestOrganization(ctx, testDB.Pool)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, testDB.Pool, orgID)
	require.NoError(t, err)

	printerID, err := testutil.CreateTestPrinter(ctx, testDB.Pool, agentID)
	require.NoError(t, err)

	// Get a user email
	var userEmail string
	err = testDB.Pool.QueryRow(ctx, "SELECT email FROM users LIMIT 1").Scan(&userEmail)
	require.NoError(t, err)

	_, _, _, _, documentID, _, err := testutil.CreateFullTestSetup(ctx, testDB.Pool)
	require.NoError(t, err)

	job := &PrintJob{
		DocumentID: documentID,
		PrinterID:  printerID,
		UserEmail:  userEmail,
		Title:      "Test Job",
		Status:     "queued",
		Priority:   5,
	}
	err = repo.Create(ctx, job)
	require.NoError(t, err)

	// Try to create a job with the same ID - should fail
	duplicate := &PrintJob{
		ID:         job.ID,
		DocumentID: documentID,
		PrinterID:  printerID,
		UserEmail:  userEmail,
		Title:      "Duplicate Job",
		Status:     "queued",
		Priority:   5,
	}
	err = repo.Create(ctx, duplicate)
	assert.Error(t, err)
}

func TestPrintJob_MediaTypes(t *testing.T) {
	validMediaTypes := []string{"a4", "a3", "letter", "legal", "tabloid"}

	for _, mediaType := range validMediaTypes {
		job := &PrintJob{MediaType: mediaType}
		if job.MediaType != mediaType {
			t.Errorf("MediaType not set correctly to %s", mediaType)
		}
	}
}

func TestPrintJob_QualityLevels(t *testing.T) {
	validQualities := []string{"draft", "normal", "high"}

	for _, quality := range validQualities {
		job := &PrintJob{Quality: quality}
		if job.Quality != quality {
			t.Errorf("Quality not set correctly to %s", quality)
		}
	}
}

// Additional edge case tests

func TestJobRepository_FindByID_NotFound(t *testing.T) {
	repo := NewJobRepository(testDB.Pool)
	ctx := context.Background()

	_, err := repo.FindByID(ctx, "00000000-0000-0000-0000-000000000001")
	assert.Error(t, err)
}

func TestJobRepository_Update_NotFound(t *testing.T) {
	repo := NewJobRepository(testDB.Pool)
	ctx := context.Background()

	job := &PrintJob{
		ID:        "00000000-0000-0000-0000-000000000001",
		DocumentID: "doc-123",
		PrinterID:  "printer-123",
		UserEmail: "test@example.com",
		Status:    "queued",
	}

	err := repo.Update(ctx, job)
	assert.Error(t, err)
}

func TestJobRepository_UpdateStatus_NotFound(t *testing.T) {
	repo := NewJobRepository(testDB.Pool)
	ctx := context.Background()

	err := repo.UpdateStatus(ctx, "00000000-0000-0000-0000-000000000001", "processing")
	assert.Error(t, err)
}

func TestJobRepository_AssignAgent_NotFound(t *testing.T) {
	repo := NewJobRepository(testDB.Pool)
	ctx := context.Background()

	err := repo.AssignAgent(ctx, "00000000-0000-0000-0000-000000000001", "agent-123")
	assert.Error(t, err)
}

func TestJobRepository_Delete_NotFound(t *testing.T) {
	repo := NewJobRepository(testDB.Pool)
	ctx := context.Background()

	err := repo.Delete(ctx, "00000000-0000-0000-0000-000000000001")
	assert.Error(t, err)
}

func TestJobRepository_UpdateJobProgress_NotFound(t *testing.T) {
	repo := NewJobRepository(testDB.Pool)
	ctx := context.Background()

	err := repo.UpdateJobProgress(ctx, "00000000-0000-0000-0000-000000000001", 5)
	assert.Error(t, err)
}

func TestJobRepository_PriorityOrdering(t *testing.T) {
	repo := NewJobRepository(testDB.Pool)
	ctx := context.Background()

	// Setup test data
	orgID, err := testutil.CreateTestOrganization(ctx, testDB.Pool)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, testDB.Pool, orgID)
	require.NoError(t, err)

	printerID, err := testutil.CreateTestPrinter(ctx, testDB.Pool, agentID)
	require.NoError(t, err)

	// Get a user email
	var userEmail string
	err = testDB.Pool.QueryRow(ctx, "SELECT email FROM users LIMIT 1").Scan(&userEmail)
	require.NoError(t, err)

	// Create jobs with different priorities
	for priority := 1; priority <= 10; priority++ {
		_, _, _, _, documentID, _, err := testutil.CreateFullTestSetup(ctx, testDB.Pool)
		require.NoError(t, err)

		job := &PrintJob{
			DocumentID: documentID,
			PrinterID:  printerID,
			UserEmail:  userEmail,
			Title:      "Job with priority",
			Status:     "queued",
			Priority:   priority,
		}
		err = repo.Create(ctx, job)
		require.NoError(t, err)
	}

	// Jobs should be ordered by priority (desc)
	jobs, err := repo.FindByStatus(ctx, "queued", 20)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(jobs), 10)

	// Check that higher priority jobs come first
	for i := 0; i < len(jobs)-1; i++ {
		assert.GreaterOrEqual(t, jobs[i].Priority, jobs[i+1].Priority)
	}
}

func TestJobRepository_StatusTransitions(t *testing.T) {
	repo := NewJobRepository(testDB.Pool)
	ctx := context.Background()

	// Setup test data
	orgID, err := testutil.CreateTestOrganization(ctx, testDB.Pool)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, testDB.Pool, orgID)
	require.NoError(t, err)

	printerID, err := testutil.CreateTestPrinter(ctx, testDB.Pool, agentID)
	require.NoError(t, err)

	// Get a user email
	var userEmail string
	err = testDB.Pool.QueryRow(ctx, "SELECT email FROM users LIMIT 1").Scan(&userEmail)
	require.NoError(t, err)

	_, _, _, _, documentID, _, err := testutil.CreateFullTestSetup(ctx, testDB.Pool)
	require.NoError(t, err)

	job := &PrintJob{
		DocumentID: documentID,
		PrinterID:  printerID,
		UserEmail:  userEmail,
		Title:      "Status Test Job",
		Status:     "queued",
		Priority:   5,
	}
	err = repo.Create(ctx, job)
	require.NoError(t, err)

	// Test status transitions
	transitions := []string{"processing", "pending_agent", "processing", "completed"}
	for _, newStatus := range transitions {
		err := repo.UpdateStatus(ctx, job.ID, newStatus)
		require.NoError(t, err)

		updated, err := repo.FindByID(ctx, job.ID)
		require.NoError(t, err)
		assert.Equal(t, newStatus, updated.Status)
	}
}

func TestJobRepository_AgentAssignment(t *testing.T) {
	repo := NewJobRepository(testDB.Pool)
	ctx := context.Background()

	// Setup test data
	orgID, err := testutil.CreateTestOrganization(ctx, testDB.Pool)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, testDB.Pool, orgID)
	require.NoError(t, err)

	printerID, err := testutil.CreateTestPrinter(ctx, testDB.Pool, agentID)
	require.NoError(t, err)

	// Get a user email
	var userEmail string
	err = testDB.Pool.QueryRow(ctx, "SELECT email FROM users LIMIT 1").Scan(&userEmail)
	require.NoError(t, err)

	_, _, _, _, documentID, _, err := testutil.CreateFullTestSetup(ctx, testDB.Pool)
	require.NoError(t, err)

	job := &PrintJob{
		DocumentID: documentID,
		PrinterID:  printerID,
		UserEmail:  userEmail,
		Title:      "Agent Test Job",
		Status:     "queued",
		Priority:   5,
	}
	err = repo.Create(ctx, job)
	require.NoError(t, err)

	// Initially no agent assigned
	found, err := repo.FindByID(ctx, job.ID)
	require.NoError(t, err)
	assert.Empty(t, found.AgentID)

	// Assign agent
	err = repo.AssignAgent(ctx, job.ID, agentID)
	require.NoError(t, err)

	// Verify assignment
	found, err = repo.FindByID(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, agentID, found.AgentID)
	assert.Equal(t, "processing", found.Status)
}

func TestJobRepository_OptionsJSON(t *testing.T) {
	repo := NewJobRepository(testDB.Pool)
	ctx := context.Background()

	// Setup test data
	orgID, err := testutil.CreateTestOrganization(ctx, testDB.Pool)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, testDB.Pool, orgID)
	require.NoError(t, err)

	printerID, err := testutil.CreateTestPrinter(ctx, testDB.Pool, agentID)
	require.NoError(t, err)

	// Get a user email
	var userEmail string
	err = testDB.Pool.QueryRow(ctx, "SELECT email FROM users LIMIT 1").Scan(&userEmail)
	require.NoError(t, err)

	_, _, _, _, documentID, _, err := testutil.CreateFullTestSetup(ctx, testDB.Pool)
	require.NoError(t, err)

	options := `{"two_sided": true, "staple": true, "color": true}`

	job := &PrintJob{
		DocumentID: documentID,
		PrinterID:  printerID,
		UserEmail:  userEmail,
		Title:      "Options Test Job",
		Options:    options,
		Status:     "queued",
		Priority:   5,
	}
	err = repo.Create(ctx, job)
	require.NoError(t, err)

	// Verify options were stored - JSON field order may vary
	found, err := repo.FindByID(ctx, job.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, found.Options)
	assert.Contains(t, found.Options, "two_sided")
	assert.Contains(t, found.Options, "staple")
	assert.Contains(t, found.Options, "color")
}

func TestJobRepository_CompletionTimestamps(t *testing.T) {
	repo := NewJobRepository(testDB.Pool)
	ctx := context.Background()

	// Setup test data
	orgID, err := testutil.CreateTestOrganization(ctx, testDB.Pool)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, testDB.Pool, orgID)
	require.NoError(t, err)

	printerID, err := testutil.CreateTestPrinter(ctx, testDB.Pool, agentID)
	require.NoError(t, err)

	// Get a user email
	var userEmail string
	err = testDB.Pool.QueryRow(ctx, "SELECT email FROM users LIMIT 1").Scan(&userEmail)
	require.NoError(t, err)

	_, _, _, _, documentID, _, err := testutil.CreateFullTestSetup(ctx, testDB.Pool)
	require.NoError(t, err)

	job := &PrintJob{
		DocumentID: documentID,
		PrinterID:  printerID,
		UserEmail:  userEmail,
		Title:      "Completion Test Job",
		Status:     "queued",
		Priority:   5,
	}
	err = repo.Create(ctx, job)
	require.NoError(t, err)

	// Initially no completion timestamp
	found, err := repo.FindByID(ctx, job.ID)
	require.NoError(t, err)
	assert.Nil(t, found.CompletedAt)

	// Update to completed
	err = repo.UpdateStatus(ctx, job.ID, "completed")
	require.NoError(t, err)

	// Completion timestamp should still be nil (not auto-set by repository)
	found, err = repo.FindByID(ctx, job.ID)
	require.NoError(t, err)
	// The repository doesn't auto-set completed_at, so we need to update manually
	now := time.Now()
	found.CompletedAt = &now
	err = repo.Update(ctx, found)
	require.NoError(t, err)

	// Verify completion timestamp
	found, err = repo.FindByID(ctx, job.ID)
	require.NoError(t, err)
	assert.NotNil(t, found.CompletedAt)
}

func TestJobRepository_RetryTracking(t *testing.T) {
	repo := NewJobRepository(testDB.Pool)
	ctx := context.Background()

	// Setup test data
	orgID, err := testutil.CreateTestOrganization(ctx, testDB.Pool)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, testDB.Pool, orgID)
	require.NoError(t, err)

	printerID, err := testutil.CreateTestPrinter(ctx, testDB.Pool, agentID)
	require.NoError(t, err)

	// Get a user email
	var userEmail string
	err = testDB.Pool.QueryRow(ctx, "SELECT email FROM users LIMIT 1").Scan(&userEmail)
	require.NoError(t, err)

	_, _, _, _, documentID, _, err := testutil.CreateFullTestSetup(ctx, testDB.Pool)
	require.NoError(t, err)

	job := &PrintJob{
		DocumentID: documentID,
		PrinterID:  printerID,
		UserEmail:  userEmail,
		Title:      "Retry Test Job",
		Status:     "queued",
		Priority:   5,
		Retries:    0,
		Options:    "{}",
	}
	err = repo.Create(ctx, job)
	require.NoError(t, err)

	// Increment retries
	job.Retries = 1
	job.Status = "failed"
	err = repo.Update(ctx, job)
	require.NoError(t, err)

	// Verify retries
	found, err := repo.FindByID(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, found.Retries)
	assert.Equal(t, "failed", found.Status)

	// Job should appear in GetJobsNeedingRetry
	retryJobs, err := repo.GetJobsNeedingRetry(ctx, 3, 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(retryJobs), 1)
}

func TestJobRepository_MultiFilterQueries(t *testing.T) {
	repo := NewJobRepository(testDB.Pool)
	ctx := context.Background()

	// Setup test data
	orgID, err := testutil.CreateTestOrganization(ctx, testDB.Pool)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, testDB.Pool, orgID)
	require.NoError(t, err)

	printerID, err := testutil.CreateTestPrinter(ctx, testDB.Pool, agentID)
	require.NoError(t, err)

	// Get a user email
	var userEmail string
	err = testDB.Pool.QueryRow(ctx, "SELECT email FROM users LIMIT 1").Scan(&userEmail)
	require.NoError(t, err)

	_, _, _, _, documentID, _, err := testutil.CreateFullTestSetup(ctx, testDB.Pool)
	require.NoError(t, err)

	job := &PrintJob{
		DocumentID: documentID,
		PrinterID:  printerID,
		UserEmail:  userEmail,
		Title:      "Multi-filter Test Job",
		Status:     "queued",
		Priority:   5,
	}
	err = repo.Create(ctx, job)
	require.NoError(t, err)

	// Test with all filters
	jobs, total, err := repo.ListWithFilters(ctx, 10, 0, printerID, "queued", userEmail)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, int64(1))
	assert.GreaterOrEqual(t, len(jobs), 1)
	if len(jobs) > 0 {
		assert.Equal(t, printerID, jobs[0].PrinterID)
		assert.Equal(t, "queued", jobs[0].Status)
		assert.Equal(t, userEmail, jobs[0].UserEmail)
	}
}

func TestJobRepository_NullAgentHandling(t *testing.T) {
	repo := NewJobRepository(testDB.Pool)
	ctx := context.Background()

	// Setup test data
	orgID, err := testutil.CreateTestOrganization(ctx, testDB.Pool)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, testDB.Pool, orgID)
	require.NoError(t, err)

	printerID, err := testutil.CreateTestPrinter(ctx, testDB.Pool, agentID)
	require.NoError(t, err)

	// Get a user email
	var userEmail string
	err = testDB.Pool.QueryRow(ctx, "SELECT email FROM users LIMIT 1").Scan(&userEmail)
	require.NoError(t, err)

	_, _, _, _, documentID, _, err := testutil.CreateFullTestSetup(ctx, testDB.Pool)
	require.NoError(t, err)

	// Create job with no agent
	job := &PrintJob{
		DocumentID: documentID,
		PrinterID:  printerID,
		UserEmail:  userEmail,
		Title:      "No Agent Job",
		Status:     "queued",
		Priority:   5,
		AgentID:    "",
	}
	err = repo.Create(ctx, job)
	require.NoError(t, err)

	// Verify agent is null/empty
	found, err := repo.FindByID(ctx, job.ID)
	require.NoError(t, err)
	assert.Empty(t, found.AgentID)

	// Assign agent
	err = repo.AssignAgent(ctx, job.ID, agentID)
	require.NoError(t, err)

	// Verify agent is now set
	found, err = repo.FindByID(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, agentID, found.AgentID)
}

func TestJobRepository_Pagination(t *testing.T) {
	repo := NewJobRepository(testDB.Pool)
	ctx := context.Background()

	// Setup test data
	orgID, err := testutil.CreateTestOrganization(ctx, testDB.Pool)
	require.NoError(t, err)

	agentID, err := testutil.CreateTestAgent(ctx, testDB.Pool, orgID)
	require.NoError(t, err)

	printerID, err := testutil.CreateTestPrinter(ctx, testDB.Pool, agentID)
	require.NoError(t, err)

	// Get a user email
	var userEmail string
	err = testDB.Pool.QueryRow(ctx, "SELECT email FROM users LIMIT 1").Scan(&userEmail)
	require.NoError(t, err)

	// Create multiple jobs
	for i := 0; i < 15; i++ {
		_, _, _, _, documentID, _, err := testutil.CreateFullTestSetup(ctx, testDB.Pool)
		require.NoError(t, err)

		job := &PrintJob{
			DocumentID: documentID,
			PrinterID:  printerID,
			UserEmail:  userEmail,
			Title:      "Pagination Test Job",
			Status:     "queued",
			Priority:   5,
		}
		err = repo.Create(ctx, job)
		require.NoError(t, err)
	}

	t.Run("first page", func(t *testing.T) {
		jobs, total, err := repo.ListWithFilters(ctx, 10, 0, "", "", "")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(15))
		assert.LessOrEqual(t, len(jobs), 10)
	})

	t.Run("second page", func(t *testing.T) {
		jobs, total, err := repo.ListWithFilters(ctx, 10, 10, "", "", "")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(15))
		assert.LessOrEqual(t, len(jobs), 10)
	})

	t.Run("offset beyond results", func(t *testing.T) {
		jobs, total, err := repo.ListWithFilters(ctx, 10, 1000, "", "", "")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(15))
		assert.Empty(t, jobs)
	})
}

// Helper function to generate unique test IDs
func generateTestID() string {
	return "test-" + time.Now().Format("20060102150405.000000000")
}
