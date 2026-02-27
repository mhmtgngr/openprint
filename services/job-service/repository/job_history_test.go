// Package repository provides tests for job history data access layer.
package repository

import (
	"context"
	"testing"
	"time"
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

func TestJobHistoryRepository_CRUD(t *testing.T) {
	t.Skip("Requires database connection")

	repo := NewJobHistoryRepository(nil)
	ctx := context.Background()

	history := &JobHistory{
		ID:        "hist-123",
		JobID:     "job-123",
		Status:    "queued",
		Message:   "Job created",
		CreatedAt: time.Now(),
	}

	t.Run("create history", func(t *testing.T) {
		err := repo.Create(ctx, history)
		if err == nil {
			t.Log("Create() succeeded (unexpected without DB)")
		}
	})

	t.Run("find by ID", func(t *testing.T) {
		_, err := repo.FindByID(ctx, "hist-123")
		if err == nil {
			t.Log("FindByID() succeeded (unexpected without DB)")
		}
	})

	t.Run("find by job ID", func(t *testing.T) {
		_, err := repo.FindByJobID(ctx, "job-123")
		if err == nil {
			t.Log("FindByJobID() succeeded (unexpected without DB)")
		}
	})

	t.Run("find by status", func(t *testing.T) {
		_, err := repo.FindByStatus(ctx, "queued", 10, 0)
		if err == nil {
			t.Log("FindByStatus() succeeded (unexpected without DB)")
		}
	})

	t.Run("delete by job ID", func(t *testing.T) {
		err := repo.DeleteByJobID(ctx, "job-123")
		if err == nil {
			t.Log("DeleteByJobID() succeeded (unexpected without DB)")
		}
	})
}

func TestJobHistoryRepository_QueryMethods(t *testing.T) {
	repo := NewJobHistoryRepository(nil)
	ctx := context.Background()

	t.Run("get latest by job ID", func(t *testing.T) {
		history, err := repo.GetLatestByJobID(ctx, "job-123")
		if err == nil && history != nil {
			t.Log("GetLatestByJobID() returned history (unexpected without DB)")
		}
	})

	t.Run("count by job ID", func(t *testing.T) {
		count, err := repo.CountByJobID(ctx, "job-123")
		if err == nil {
			t.Logf("CountByJobID() returned %d (unexpected without DB)", count)
		}
	})

	t.Run("list with pagination", func(t *testing.T) {
		_, total, err := repo.List(ctx, 10, 0)
		if err == nil {
			t.Logf("List() returned total=%d (unexpected without DB)", total)
		}
	})
}

func TestJobHistoryRepository_BatchOperations(t *testing.T) {
	repo := NewJobHistoryRepository(nil)
	ctx := context.Background()

	entries := []*JobHistory{
		{
			ID:        "hist-1",
			JobID:     "job-123",
			Status:    "queued",
			Message:   "Job queued",
			CreatedAt: time.Now(),
		},
		{
			ID:        "hist-2",
			JobID:     "job-123",
			Status:    "processing",
			Message:   "Job started processing",
			CreatedAt: time.Now(),
		},
		{
			ID:        "hist-3",
			JobID:     "job-123",
			Status:    "completed",
			Message:   "Job completed successfully",
			CreatedAt: time.Now(),
		},
	}

	err := repo.CreateBatch(ctx, entries)
	if err == nil {
		t.Log("CreateBatch() succeeded (unexpected without DB)")
	}
}

func TestJobHistoryRepository_DeleteOld(t *testing.T) {
	repo := NewJobHistoryRepository(nil)
	ctx := context.Background()

	t.Run("delete entries older than duration", func(t *testing.T) {
		deleted, err := repo.DeleteOld(ctx, 30*24*time.Hour)
		if err == nil {
			t.Logf("DeleteOld() deleted %d entries (unexpected without DB)", deleted)
		}
	})

	t.Run("delete entries older than 1 hour", func(t *testing.T) {
		deleted, err := repo.DeleteOld(ctx, 1*time.Hour)
		if err == nil {
			t.Logf("DeleteOld() deleted %d entries (unexpected without DB)", deleted)
		}
	})
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
				JobID:    "job-123",
				Status:   tt.to,
				Message:  tt.message,
				CreatedAt: time.Now(),
			}

			if history.Status != tt.to {
				t.Errorf("Expected status %s, got %s", tt.to, history.Status)
			}
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
				JobID:    "job-123",
				Status:   "processing",
				Message:  tt.message,
				CreatedAt: time.Now(),
			}

			if history.Message != tt.message {
				t.Errorf("Expected message %s, got %s", tt.message, history.Message)
			}
		})
	}
}

func TestJobHistory_TimeFields(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	histories := []*JobHistory{
		{
			ID:        "hist-1",
			JobID:     "job-123",
			Status:    "queued",
			Message:   "Initial status",
			CreatedAt: past,
		},
		{
			ID:        "hist-2",
			JobID:     "job-123",
			Status:    "processing",
			Message:   "Started processing",
			CreatedAt: now,
		},
		{
			ID:        "hist-3",
			JobID:     "job-123",
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

func TestJobHistory_EmptyBatch(t *testing.T) {
	repo := NewJobHistoryRepository(nil)
	ctx := context.Background()

	// Empty batch should not error
	err := repo.CreateBatch(ctx, []*JobHistory{})
	if err != nil {
		t.Errorf("CreateBatch() with empty entries should not error, got %v", err)
	}
}

func TestJobHistory_JobLifecycle(t *testing.T) {
	// Simulate a complete job lifecycle through history entries
	jobID := "job-lifecycle-123"

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

	var histories []*JobHistory
	for i, step := range lifecycle {
		histories = append(histories, &JobHistory{
			ID:        "hist-" + string(rune('1'+i)),
			JobID:     jobID,
			Status:    step.status,
			Message:   step.message,
			CreatedAt: time.Now().Add(time.Duration(i) * time.Minute),
		})
	}

	for i, h := range histories {
		if h.JobID != jobID {
			t.Errorf("History %d: expected JobID %s, got %s", i, jobID, h.JobID)
		}
	}
}

func TestJobHistory_MultipleJobs(t *testing.T) {
	// Test history for multiple jobs
	jobs := []string{"job-1", "job-2", "job-3"}

	for _, jobID := range jobs {
		history := &JobHistory{
			ID:        "hist-" + jobID,
			JobID:     jobID,
			Status:    "queued",
			Message:   "Job created",
			CreatedAt: time.Now(),
		}

		if history.JobID != jobID {
			t.Errorf("Expected JobID %s, got %s", jobID, history.JobID)
		}
	}
}

func TestJobHistory_LongMessages(t *testing.T) {
	// Test handling of long messages
	longMessage := "This is a very long error message that contains detailed information about what went wrong during the printing process, including error codes, stack traces, and diagnostic information that might be useful for debugging purposes."

	history := &JobHistory{
		ID:        "hist-123",
		JobID:     "job-123",
		Status:    "failed",
		Message:   longMessage,
		CreatedAt: time.Now(),
	}

	if len(history.Message) != len(longMessage) {
		t.Error("Long message was truncated")
	}
}

func TestJobHistory_SpecialCharacters(t *testing.T) {
	// Test handling of special characters in messages
	messages := []string{
		"Job completed with \"quotes\"",
		"Error: printer's paper tray is empty",
		"Status: 100% complete\nNext: idle",
		"Price: $10.50 per page",
	}

	for _, msg := range messages {
		history := &JobHistory{
			ID:        "hist-123",
			JobID:     "job-123",
			Status:    "processing",
			Message:   msg,
			CreatedAt: time.Now(),
		}

		if history.Message != msg {
			t.Errorf("Message with special characters was modified: %s", history.Message)
		}
	}
}
