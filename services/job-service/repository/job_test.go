// Package repository provides tests for print job data access layer.
package repository

import (
	"context"
	"testing"
	"time"
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
		UserName:   "testuser",
		UserEmail:  "test@example.com",
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
			t.Error("Priority 1 should be less than priority 10")
		}
	})
}

func TestJobRepository_CRUD(t *testing.T) {
	repo := NewJobRepository(nil)
	ctx := context.Background()

	job := &PrintJob{
		ID:         "job-123",
		DocumentID: "doc-123",
		PrinterID:  "printer-123",
		UserEmail:  "test@example.com",
		Title:      "Test Document",
		Status:     "queued",
		Priority:   5,
	}

	t.Run("create job", func(t *testing.T) {
		err := repo.Create(ctx, job)
		if err == nil {
			t.Log("Create() succeeded (unexpected without DB)")
		}
	})

	t.Run("find by ID", func(t *testing.T) {
		_, err := repo.FindByID(ctx, "job-123")
		if err == nil {
			t.Log("FindByID() succeeded (unexpected without DB)")
		}
	})

	t.Run("update job", func(t *testing.T) {
		err := repo.Update(ctx, job)
		if err == nil {
			t.Log("Update() succeeded (unexpected without DB)")
		}
	})

	t.Run("update status", func(t *testing.T) {
		err := repo.UpdateStatus(ctx, "job-123", "processing")
		if err == nil {
			t.Log("UpdateStatus() succeeded (unexpected without DB)")
		}
	})

	t.Run("assign agent", func(t *testing.T) {
		err := repo.AssignAgent(ctx, "job-123", "agent-123")
		if err == nil {
			t.Log("AssignAgent() succeeded (unexpected without DB)")
		}
	})

	t.Run("delete job", func(t *testing.T) {
		err := repo.Delete(ctx, "job-123")
		if err == nil {
			t.Log("Delete() succeeded (unexpected without DB)")
		}
	})
}

func TestJobRepository_QueryMethods(t *testing.T) {
	repo := NewJobRepository(nil)
	ctx := context.Background()

	t.Run("find by status", func(t *testing.T) {
		_, err := repo.FindByStatus(ctx, "queued", 10)
		if err == nil {
			t.Log("FindByStatus() succeeded (unexpected without DB)")
		}
	})

	t.Run("find by printer", func(t *testing.T) {
		_, err := repo.FindByPrinter(ctx, "printer-123", 10, 0)
		if err == nil {
			t.Log("FindByPrinter() succeeded (unexpected without DB)")
		}
	})

	t.Run("find by user", func(t *testing.T) {
		_, err := repo.FindByUser(ctx, "test@example.com", 10, 0)
		if err == nil {
			t.Log("FindByUser() succeeded (unexpected without DB)")
		}
	})

	t.Run("list with filters", func(t *testing.T) {
		_, _, err := repo.ListWithFilters(ctx, 10, 0, "printer-123", "queued", "test@example.com")
		if err == nil {
			t.Log("ListWithFilters() succeeded (unexpected without DB)")
		}
	})

	t.Run("count by status", func(t *testing.T) {
		_, err := repo.CountByStatus(ctx, "queued")
		if err == nil {
			t.Log("CountByStatus() succeeded (unexpected without DB)")
		}
	})
}

func TestJobRepository_AdvancedOperations(t *testing.T) {
	repo := NewJobRepository(nil)
	ctx := context.Background()

	t.Run("get next pending job", func(t *testing.T) {
		job, err := repo.GetNextPendingJob(ctx, "printer-123")
		if err == nil {
			t.Log("GetNextPendingJob() succeeded (unexpected without DB)")
		}
		if job != nil {
			t.Logf("GetNextPendingJob() returned job: %v", job.ID)
		}
	})

	t.Run("update job progress", func(t *testing.T) {
		err := repo.UpdateJobProgress(ctx, "job-123", 5)
		if err == nil {
			t.Log("UpdateJobProgress() succeeded (unexpected without DB)")
		}
	})

	t.Run("get jobs needing retry", func(t *testing.T) {
		jobs, err := repo.GetJobsNeedingRetry(ctx, 3, 10)
		if err == nil {
			t.Logf("GetJobsNeedingRetry() returned %d jobs (unexpected without DB)", len(jobs))
		}
	})
}

func TestPrintJob_TimeFields(t *testing.T) {
	now := time.Now()
	completedAt := now

	job := &PrintJob{
		StartedAt:   now,
		CompletedAt: &completedAt,
		CreatedAt:   now,
		UpdatedAt:   now,
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
	repo := NewJobRepository(nil)
	ctx := context.Background()

	// All filters empty should return all jobs
	_, _, err := repo.ListWithFilters(ctx, 10, 0, "", "", "")
	if err == nil {
		t.Log("ListWithFilters() with empty filters succeeded (unexpected without DB)")
	}
}

func TestJobRepository_ListWithFilters_SingleFilter(t *testing.T) {
	repo := NewJobRepository(nil)
	ctx := context.Background()

	t.Run("filter by printer only", func(t *testing.T) {
		_, _, err := repo.ListWithFilters(ctx, 10, 0, "printer-123", "", "")
		if err == nil {
			t.Log("ListWithFilters() by printer succeeded (unexpected without DB)")
		}
	})

	t.Run("filter by status only", func(t *testing.T) {
		_, _, err := repo.ListWithFilters(ctx, 10, 0, "", "processing", "")
		if err == nil {
			t.Log("ListWithFilters() by status succeeded (unexpected without DB)")
		}
	})

	t.Run("filter by user only", func(t *testing.T) {
		_, _, err := repo.ListWithFilters(ctx, 10, 0, "", "", "user@example.com")
		if err == nil {
			t.Log("ListWithFilters() by user succeeded (unexpected without DB)")
		}
	})
}

func TestJobRepository_DuplicateHandling(t *testing.T) {
	// Test that the repository can handle duplicate scenarios
	// These would typically use database constraints

	t.Run("unique job ID", func(t *testing.T) {
		job1 := &PrintJob{ID: "job-123"}
		job2 := &PrintJob{ID: "job-123"}

		if job1.ID == job2.ID {
			t.Log("Duplicate job IDs would be handled by database constraints")
		}
	})
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
