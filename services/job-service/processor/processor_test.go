// Package processor provides tests for background job processing.
package processor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/openprint/openprint/services/job-service/repository"
)

// mockJobRepo is a mock implementation of JobRepository for testing
type mockJobRepo struct {
	jobs      map[string]*repository.PrintJob
	createErr error
	findErr   error
	updateErr error
}

func (m *mockJobRepo) Create(ctx context.Context, job *repository.PrintJob) error {
	if m.jobs == nil {
		m.jobs = make(map[string]*repository.PrintJob)
	}
	m.jobs[job.ID] = job
	return m.createErr
}

func (m *mockJobRepo) FindByID(ctx context.Context, id string) (*repository.PrintJob, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	job, ok := m.jobs[id]
	if !ok {
		return nil, &testError{"job not found"}
	}
	return job, nil
}

func (m *mockJobRepo) Update(ctx context.Context, job *repository.PrintJob) error {
	m.jobs[job.ID] = job
	return m.updateErr
}

func (m *mockJobRepo) FindByStatus(ctx context.Context, status string, limit int) ([]*repository.PrintJob, error) {
	var result []*repository.PrintJob
	for _, job := range m.jobs {
		if job.Status == status {
			result = append(result, job)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *mockJobRepo) CountByStatus(ctx context.Context, status string) (int64, error) {
	var count int64
	for _, job := range m.jobs {
		if job.Status == status {
			count++
		}
	}
	return count, nil
}

// Implement other required methods with no-ops
func (m *mockJobRepo) UpdateStatus(ctx context.Context, jobID, status string) error { return nil }
func (m *mockJobRepo) Delete(ctx context.Context, id string) error                  { return nil }
func (m *mockJobRepo) FindByPrinter(ctx context.Context, printerID string, limit, offset int) ([]*repository.PrintJob, error) {
	return nil, nil
}
func (m *mockJobRepo) FindByUser(ctx context.Context, userEmail string, limit, offset int) ([]*repository.PrintJob, error) {
	return nil, nil
}
func (m *mockJobRepo) ListWithFilters(ctx context.Context, limit, offset int, printerID, status, userEmail string) ([]*repository.PrintJob, int, error) {
	return nil, 0, nil
}
func (m *mockJobRepo) GetNextPendingJob(ctx context.Context, printerID string) (*repository.PrintJob, error) {
	return nil, nil
}
func (m *mockJobRepo) UpdateJobProgress(ctx context.Context, jobID string, progress int) error {
	return nil
}
func (m *mockJobRepo) GetJobsNeedingRetry(ctx context.Context, maxRetries, limit int) ([]*repository.PrintJob, error) {
	return nil, nil
}
func (m *mockJobRepo) AssignAgent(ctx context.Context, jobID, agentID string) error { return nil }

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// mockHistoryRepo is a mock implementation of JobHistoryRepository
type mockHistoryRepo struct {
	entries []*repository.JobHistory
}

func (m *mockHistoryRepo) Create(ctx context.Context, history *repository.JobHistory) error {
	m.entries = append(m.entries, history)
	return nil
}

// Implement other required methods
func (m *mockHistoryRepo) FindByID(ctx context.Context, id string) (*repository.JobHistory, error) {
	return nil, nil
}
func (m *mockHistoryRepo) FindByJobID(ctx context.Context, jobID string) ([]*repository.JobHistory, error) {
	return nil, nil
}
func (m *mockHistoryRepo) FindByStatus(ctx context.Context, status string, limit, offset int) ([]*repository.JobHistory, error) {
	return nil, nil
}
func (m *mockHistoryRepo) DeleteByJobID(ctx context.Context, jobID string) error { return nil }
func (m *mockHistoryRepo) DeleteOld(ctx context.Context, olderThan time.Duration) (int64, error) {
	return 0, nil
}
func (m *mockHistoryRepo) GetLatestByJobID(ctx context.Context, jobID string) (*repository.JobHistory, error) {
	return nil, nil
}
func (m *mockHistoryRepo) CountByJobID(ctx context.Context, jobID string) (int, error) { return 0, nil }
func (m *mockHistoryRepo) List(ctx context.Context, limit, offset int) ([]*repository.JobHistory, int, error) {
	return nil, 0, nil
}
func (m *mockHistoryRepo) CreateBatch(ctx context.Context, entries []*repository.JobHistory) error {
	return nil
}

func TestNewProcessor(t *testing.T) {
	jobRepo := &mockJobRepo{jobs: make(map[string]*repository.PrintJob)}
	historyRepo := &mockHistoryRepo{}

	cfg := Config{
		JobRepo:      jobRepo,
		HistoryRepo:  historyRepo,
		Redis:        nil,
		Workers:      2,
		PollInterval: time.Second,
	}

	proc := New(cfg)

	if proc == nil {
		t.Fatal("New() returned nil")
	}
	if proc.workers != 2 {
		t.Errorf("Expected 2 workers, got %d", proc.workers)
	}
	if proc.pollInterval != time.Second {
		t.Errorf("Expected 1s poll interval, got %v", proc.pollInterval)
	}
	if proc.jobQueue == nil {
		t.Error("jobQueue should be initialized")
	}
	if proc.processing == nil {
		t.Error("processing map should be initialized")
	}
	if proc.cancelled == nil {
		t.Error("cancelled map should be initialized")
	}
}

func TestProcessor_Stats(t *testing.T) {
	jobRepo := &mockJobRepo{jobs: make(map[string]*repository.PrintJob)}
	historyRepo := &mockHistoryRepo{}

	// Add some test jobs
	now := time.Now()
	jobRepo.jobs["job-1"] = &repository.PrintJob{ID: "job-1", Status: "queued", CreatedAt: now}
	jobRepo.jobs["job-2"] = &repository.PrintJob{ID: "job-2", Status: "processing", CreatedAt: now}
	jobRepo.jobs["job-3"] = &repository.PrintJob{ID: "job-3", Status: "completed", CreatedAt: now}
	jobRepo.jobs["job-4"] = &repository.PrintJob{ID: "job-4", Status: "failed", CreatedAt: now}
	jobRepo.jobs["job-5"] = &repository.PrintJob{ID: "job-5", Status: "pending_agent", CreatedAt: now}

	cfg := Config{
		JobRepo:      jobRepo,
		HistoryRepo:  historyRepo,
		Redis:        nil,
		Workers:      2,
		PollInterval: time.Second,
	}

	proc := New(cfg)
	ctx := context.Background()

	stats, err := proc.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats() returned error: %v", err)
	}

	if stats.Workers != 2 {
		t.Errorf("Expected 2 workers, got %d", stats.Workers)
	}
	if stats.Queued != 1 {
		t.Errorf("Expected 1 queued job, got %d", stats.Queued)
	}
	if stats.Processing != 2 { // processing + pending_agent
		t.Errorf("Expected 2 processing jobs, got %d", stats.Processing)
	}
	if stats.Completed != 1 {
		t.Errorf("Expected 1 completed job, got %d", stats.Completed)
	}
	if stats.Failed != 1 {
		t.Errorf("Expected 1 failed job, got %d", stats.Failed)
	}
}

func TestProcessor_Cancel(t *testing.T) {
	jobRepo := &mockJobRepo{jobs: make(map[string]*repository.PrintJob)}
	historyRepo := &mockHistoryRepo{}

	cfg := Config{
		JobRepo:      jobRepo,
		HistoryRepo:  historyRepo,
		Redis:        nil,
		Workers:      2,
		PollInterval: time.Second,
	}

	proc := New(cfg)
	ctx := context.Background()

	// Add a job to processing
	job := &repository.PrintJob{
		ID:        "job-to-cancel",
		Status:    "processing",
		CreatedAt: time.Now(),
	}
	proc.processing[job.ID] = job

	// Cancel the job
	proc.Cancel(ctx, job.ID)

	// Check if marked for cancellation
	proc.mu.Lock()
	_, stillProcessing := proc.processing[job.ID]
	_, wasCancelled := proc.cancelled[job.ID]
	proc.mu.Unlock()

	if stillProcessing && wasCancelled {
		t.Log("Job marked for cancellation while still in processing map")
	}
}

func TestProcessor_CompleteJob(t *testing.T) {
	jobRepo := &mockJobRepo{jobs: make(map[string]*repository.PrintJob)}
	historyRepo := &mockHistoryRepo{}

	now := time.Now()
	job := &repository.PrintJob{
		ID:        "job-123",
		Status:    "processing",
		CreatedAt: now,
	}
	jobRepo.jobs[job.ID] = job

	cfg := Config{
		JobRepo:      jobRepo,
		HistoryRepo:  historyRepo,
		Redis:        nil,
		Workers:      2,
		PollInterval: time.Second,
	}

	proc := New(cfg)
	ctx := context.Background()

	// Test successful completion
	err := proc.CompleteJob(ctx, job.ID, true, "Job completed successfully")
	if err != nil {
		t.Errorf("CompleteJob() with success=true returned error: %v", err)
	}

	// Check job was updated
	updatedJob, _ := jobRepo.FindByID(ctx, job.ID)
	if updatedJob.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", updatedJob.Status)
	}
	if updatedJob.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}
}

func TestProcessor_CompleteJobFailure(t *testing.T) {
	jobRepo := &mockJobRepo{jobs: make(map[string]*repository.PrintJob)}
	historyRepo := &mockHistoryRepo{}

	job := &repository.PrintJob{
		ID:        "job-fail-123",
		Status:    "processing",
		Retries:   0,
		CreatedAt: time.Now(),
	}
	jobRepo.jobs[job.ID] = job

	cfg := Config{
		JobRepo:      jobRepo,
		HistoryRepo:  historyRepo,
		Redis:        nil,
		Workers:      2,
		PollInterval: time.Second,
	}

	proc := New(cfg)
	ctx := context.Background()

	// Test failure completion
	err := proc.CompleteJob(ctx, job.ID, false, "Printer offline")
	if err != nil {
		t.Errorf("CompleteJob() with success=false returned error: %v", err)
	}

	// Check job was updated
	updatedJob, _ := jobRepo.FindByID(ctx, job.ID)
	if updatedJob.Status != "failed" {
		t.Errorf("Expected status 'failed', got '%s'", updatedJob.Status)
	}
	if updatedJob.Retries != 1 {
		t.Errorf("Expected Retries=1, got %d", updatedJob.Retries)
	}
}

func TestProcessor_UpdateJobProgress(t *testing.T) {
	jobRepo := &mockJobRepo{jobs: make(map[string]*repository.PrintJob)}
	historyRepo := &mockHistoryRepo{}

	job := &repository.PrintJob{
		ID:        "job-progress-123",
		Status:    "processing",
		CreatedAt: time.Now(),
	}
	jobRepo.jobs[job.ID] = job

	cfg := Config{
		JobRepo:      jobRepo,
		HistoryRepo:  historyRepo,
		Redis:        nil,
		Workers:      2,
		PollInterval: time.Second,
	}

	proc := New(cfg)
	ctx := context.Background()

	// Update progress
	err := proc.UpdateJobProgress(ctx, job.ID, "processing", 50, "Printing page 5 of 10")
	if err != nil {
		t.Errorf("UpdateJobProgress() returned error: %v", err)
	}

	// Check history was created
	if len(historyRepo.entries) != 1 {
		t.Errorf("Expected 1 history entry, got %d", len(historyRepo.entries))
	}
}

func TestProcessor_RequeueFailedJobs(t *testing.T) {
	jobRepo := &mockJobRepo{jobs: make(map[string]*repository.PrintJob)}
	historyRepo := &mockHistoryRepo{}

	now := time.Now()

	// Add failed jobs with different retry counts
	jobRepo.jobs["job-retry-1"] = &repository.PrintJob{ID: "job-retry-1", Status: "failed", Retries: 0, CreatedAt: now}
	jobRepo.jobs["job-retry-2"] = &repository.PrintJob{ID: "job-retry-2", Status: "failed", Retries: 1, CreatedAt: now}
	jobRepo.jobs["job-retry-3"] = &repository.PrintJob{ID: "job-retry-3", Status: "failed", Retries: 5, CreatedAt: now}

	cfg := Config{
		JobRepo:      jobRepo,
		HistoryRepo:  historyRepo,
		Redis:        nil,
		Workers:      2,
		PollInterval: time.Second,
	}

	proc := New(cfg)
	ctx := context.Background()

	// Requeue jobs with maxRetries=3
	requeued, err := proc.RequeueFailedJobs(ctx, 3)
	if err != nil {
		t.Errorf("RequeueFailedJobs() returned error: %v", err)
	}

	// Should requeue 2 jobs (job-retry-1 and job-retry-2)
	if requeued != 2 {
		t.Errorf("Expected 2 requeued jobs, got %d", requeued)
	}

	// Check the jobs were updated
	job1, _ := jobRepo.FindByID(ctx, "job-retry-1")
	if job1.Status != "queued" {
		t.Errorf("job-retry-1 should be queued, got %s", job1.Status)
	}

	job3, _ := jobRepo.FindByID(ctx, "job-retry-3")
	if job3.Status != "failed" {
		t.Errorf("job-retry-3 should still be failed, got %s", job3.Status)
	}
}

func TestProcessor_ConfigDefaults(t *testing.T) {
	jobRepo := &mockJobRepo{jobs: make(map[string]*repository.PrintJob)}
	historyRepo := &mockHistoryRepo{}

	tests := []struct {
		name         string
		workers      int
		pollInterval time.Duration
	}{
		{"default config", 4, 5 * time.Second},
		{"single worker", 1, 100 * time.Millisecond},
		{"many workers", 10, 1 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				JobRepo:      jobRepo,
				HistoryRepo:  historyRepo,
				Workers:      tt.workers,
				PollInterval: tt.pollInterval,
			}

			proc := New(cfg)

			if proc.workers != tt.workers {
				t.Errorf("Expected %d workers, got %d", tt.workers, proc.workers)
			}
			if proc.pollInterval != tt.pollInterval {
				t.Errorf("Expected poll interval %v, got %v", tt.pollInterval, proc.pollInterval)
			}
		})
	}
}

func TestProcessor_JobQueue(t *testing.T) {
	jobRepo := &mockJobRepo{jobs: make(map[string]*repository.PrintJob)}
	historyRepo := &mockHistoryRepo{}

	cfg := Config{
		JobRepo:      jobRepo,
		HistoryRepo:  historyRepo,
		Redis:        nil,
		Workers:      2,
		PollInterval: time.Second,
	}

	proc := New(cfg)

	// Check jobQueue capacity
	if cap(proc.jobQueue) != 1000 {
		t.Errorf("Expected jobQueue capacity of 1000, got %d", cap(proc.jobQueue))
	}

	// Try to add a job
	job := &repository.PrintJob{
		ID:        "queue-test-job",
		Status:    "queued",
		CreatedAt: time.Now(),
	}

	select {
	case proc.jobQueue <- job:
		// Success
	default:
		t.Error("Failed to add job to queue")
	}
}

func TestProcessor_TrackingMaps(t *testing.T) {
	jobRepo := &mockJobRepo{jobs: make(map[string]*repository.PrintJob)}
	historyRepo := &mockHistoryRepo{}

	cfg := Config{
		JobRepo:      jobRepo,
		HistoryRepo:  historyRepo,
		Redis:        nil,
		Workers:      2,
		PollInterval: time.Second,
	}

	proc := New(cfg)

	// Test processing map
	job1 := &repository.PrintJob{ID: "job-1", Status: "processing"}
	proc.processing[job1.ID] = job1

	if len(proc.processing) != 1 {
		t.Errorf("Expected 1 job in processing map, got %d", len(proc.processing))
	}

	// Test cancelled map
	proc.cancelled["job-2"] = struct{}{}

	if len(proc.cancelled) != 1 {
		t.Errorf("Expected 1 job in cancelled map, got %d", len(proc.cancelled))
	}

	// Test removal from processing
	delete(proc.processing, job1.ID)
	if len(proc.processing) != 0 {
		t.Errorf("Expected 0 jobs in processing map after deletion, got %d", len(proc.processing))
	}
}

func TestProcessor_Stop(t *testing.T) {
	jobRepo := &mockJobRepo{jobs: make(map[string]*repository.PrintJob)}
	historyRepo := &mockHistoryRepo{}

	cfg := Config{
		JobRepo:      jobRepo,
		HistoryRepo:  historyRepo,
		Redis:        nil,
		Workers:      2,
		PollInterval: time.Second,
	}

	proc := New(cfg)

	// Stop should not panic
	proc.Stop()

	// After stop, the stop channel should be closed
	select {
	case <-proc.workerStop:
		// Channel closed as expected
	default:
		t.Error("workerStop channel should be closed after Stop()")
	}
}

func TestStats_Struct(t *testing.T) {
	stats := &Stats{
		Queued:     10,
		Processing: 5,
		Completed:  100,
		Failed:     2,
		Workers:    4,
	}

	if stats.Queued != 10 {
		t.Errorf("Expected Queued=10, got %d", stats.Queued)
	}
	if stats.Processing != 5 {
		t.Errorf("Expected Processing=5, got %d", stats.Processing)
	}
	if stats.Completed != 100 {
		t.Errorf("Expected Completed=100, got %d", stats.Completed)
	}
	if stats.Failed != 2 {
		t.Errorf("Expected Failed=2, got %d", stats.Failed)
	}
	if stats.Workers != 4 {
		t.Errorf("Expected Workers=4, got %d", stats.Workers)
	}
}

func TestProcessor_ConcurrentAccess(t *testing.T) {
	jobRepo := &mockJobRepo{jobs: make(map[string]*repository.PrintJob)}
	historyRepo := &mockHistoryRepo{}

	cfg := Config{
		JobRepo:      jobRepo,
		HistoryRepo:  historyRepo,
		Redis:        nil,
		Workers:      2,
		PollInterval: time.Second,
	}

	proc := New(cfg)
	ctx := context.Background()

	// Test concurrent access to tracking maps
	done := make(chan bool)

	// Goroutine adding to processing
	go func() {
		for i := 0; i < 10; i++ {
			jobID := fmt.Sprintf("job-%d", i)
			proc.mu.Lock()
			proc.processing[jobID] = &repository.PrintJob{ID: jobID}
			proc.mu.Unlock()
		}
		done <- true
	}()

	// Goroutine adding to cancelled
	go func() {
		for i := 0; i < 10; i++ {
			jobID := fmt.Sprintf("job-cancel-%d", i)
			proc.mu.Lock()
			proc.cancelled[jobID] = struct{}{}
			proc.mu.Unlock()
		}
		done <- true
	}()

	// Goroutine getting stats
	go func() {
		for i := 0; i < 10; i++ {
			proc.GetStats(ctx)
		}
		done <- true
	}()

	// Wait for all goroutines
	<-done
	<-done
	<-done

	// If we got here without race condition or deadlock, test passed
	t.Log("Concurrent access test passed")
}
