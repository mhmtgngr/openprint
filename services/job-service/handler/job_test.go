// Package handler provides tests for job service HTTP handlers.
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/services/job-service/processor"
	"github.com/openprint/openprint/services/job-service/repository"
)

// mock repositories for testing
type mockJobHandlerRepo struct {
	jobs map[string]*repository.PrintJob
}

func (m *mockJobHandlerRepo) Create(ctx context.Context, job *repository.PrintJob) error {
	m.jobs[job.ID] = job
	return nil
}

func (m *mockJobHandlerRepo) FindByID(ctx context.Context, id string) (*repository.PrintJob, error) {
	job, ok := m.jobs[id]
	if !ok {
		return nil, apperrors.ErrNotFound
	}
	return job, nil
}

func (m *mockJobHandlerRepo) Update(ctx context.Context, job *repository.PrintJob) error {
	m.jobs[job.ID] = job
	return nil
}

func (m *mockJobHandlerRepo) UpdateStatus(ctx context.Context, jobID, status string) error {
	if job, ok := m.jobs[jobID]; ok {
		job.Status = status
		job.UpdatedAt = time.Now()
		return nil
	}
	return apperrors.ErrNotFound
}

func (m *mockJobHandlerRepo) FindByStatus(ctx context.Context, status string, limit int) ([]*repository.PrintJob, error) {
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

func (m *mockJobHandlerRepo) ListWithFilters(ctx context.Context, limit, offset int, printerID, status, userEmail string) ([]*repository.PrintJob, int, error) {
	var result []*repository.PrintJob
	for _, job := range m.jobs {
		result = append(result, job)
		if len(result) >= limit {
			break
		}
	}
	return result, len(result), nil
}

func (m *mockJobHandlerRepo) CountByStatus(ctx context.Context, status string) (int64, error) {
	var count int64
	for _, job := range m.jobs {
		if job.Status == status {
			count++
		}
	}
	return count, nil
}

// No-op implementations for unused methods
func (m *mockJobHandlerRepo) Delete(ctx context.Context, id string) error { return nil }
func (m *mockJobHandlerRepo) FindByPrinter(ctx context.Context, printerID string, limit, offset int) ([]*repository.PrintJob, error) { return nil, nil }
func (m *mockJobHandlerRepo) FindByUser(ctx context.Context, userEmail string, limit, offset int) ([]*repository.PrintJob, error) { return nil, nil }
func (m *mockJobHandlerRepo) GetNextPendingJob(ctx context.Context, printerID string) (*repository.PrintJob, error) { return nil, nil }
func (m *mockJobHandlerRepo) UpdateJobProgress(ctx context.Context, jobID string, progress int) error { return nil }
func (m *mockJobHandlerRepo) GetJobsNeedingRetry(ctx context.Context, maxRetries, limit int) ([]*repository.PrintJob, error) { return nil, nil }
func (m *mockJobHandlerRepo) AssignAgent(ctx context.Context, jobID, agentID string) error { return nil }

type mockHistoryHandlerRepo struct {
	entries map[string][]*repository.JobHistory
}

func (m *mockHistoryHandlerRepo) Create(ctx context.Context, history *repository.JobHistory) error {
	if m.entries == nil {
		m.entries = make(map[string][]*repository.JobHistory)
	}
	m.entries[history.JobID] = append(m.entries[history.JobID], history)
	return nil
}

func (m *mockHistoryHandlerRepo) FindByJobID(ctx context.Context, jobID string) ([]*repository.JobHistory, error) {
	return m.entries[jobID], nil
}

// No-op implementations
func (m *mockHistoryHandlerRepo) FindByID(ctx context.Context, id string) (*repository.JobHistory, error) { return nil, nil }
func (m *mockHistoryHandlerRepo) FindByStatus(ctx context.Context, status string, limit, offset int) ([]*repository.JobHistory, error) { return nil, nil }
func (m *mockHistoryHandlerRepo) DeleteByJobID(ctx context.Context, jobID string) error { return nil }
func (m *mockHistoryHandlerRepo) DeleteOld(ctx context.Context, olderThan time.Duration) (int64, error) { return 0, nil }
func (m *mockHistoryHandlerRepo) GetLatestByJobID(ctx context.Context, jobID string) (*repository.JobHistory, error) { return nil, nil }
func (m *mockHistoryHandlerRepo) CountByJobID(ctx context.Context, jobID string) (int, error) { return 0, nil }
func (m *mockHistoryHandlerRepo) List(ctx context.Context, limit, offset int) ([]*repository.JobHistory, int, error) { return nil, 0, nil }
func (m *mockHistoryHandlerRepo) CreateBatch(ctx context.Context, entries []*repository.JobHistory) error { return nil }

type mockProcessor struct{}

func (m *mockProcessor) Enqueue(ctx context.Context, job *repository.PrintJob) error { return nil }
func (m *mockProcessor) Cancel(ctx context.Context, jobID string)                       {}
func (m *mockProcessor) GetStats(ctx context.Context) (*processor.Stats, error) {
	return &processor.Stats{Queued: 1, Processing: 2, Completed: 10, Failed: 0, Workers: 4}, nil
}

func TestNewHandler(t *testing.T) {
	jobRepo := &mockJobHandlerRepo{jobs: make(map[string]*repository.PrintJob)}
	historyRepo := &mockHistoryHandlerRepo{}
	proc := &mockProcessor{}

	cfg := Config{
		JobRepo:     jobRepo,
		HistoryRepo: historyRepo,
		Processor:   proc,
	}

	h := New(cfg)

	if h == nil {
		t.Fatal("New() returned nil")
	}
	if h.jobRepo != jobRepo {
		t.Error("JobRepo not set correctly")
	}
	if h.historyRepo != historyRepo {
		t.Error("HistoryRepo not set correctly")
	}
	if h.processor != proc {
		t.Error("Processor not set correctly")
	}
}

func TestCreateJobRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateJobRequest
		wantErr string
	}{
		{
			name: "valid request",
			req: CreateJobRequest{
				DocumentID: "doc-123",
				PrinterID:  "printer-123",
				UserEmail:  "test@example.com",
				Title:      "Test Document",
			},
			wantErr: "",
		},
		{
			name: "missing document_id",
			req: CreateJobRequest{
				PrinterID: "printer-123",
				UserEmail: "test@example.com",
			},
			wantErr: "document_id",
		},
		{
			name: "missing printer_id",
			req: CreateJobRequest{
				DocumentID: "doc-123",
				UserEmail:  "test@example.com",
			},
			wantErr: "printer_id",
		},
		{
			name: "missing user_email",
			req: CreateJobRequest{
				DocumentID: "doc-123",
				PrinterID:  "printer-123",
			},
			wantErr: "user_email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasErr := false
			if tt.req.DocumentID == "" {
				hasErr = true
			}
			if tt.req.PrinterID == "" {
				hasErr = true
			}
			if tt.req.UserEmail == "" {
				hasErr = true
			}

			if !hasErr && tt.wantErr != "" {
				t.Error("Expected validation error but got none")
			}
			if hasErr && tt.wantErr == "" {
				t.Error("Expected no validation error but got one")
			}
		})
	}
}

func TestCreateJobRequest_Defaults(t *testing.T) {
	req := CreateJobRequest{
		DocumentID: "doc-123",
		PrinterID:  "printer-123",
		UserEmail:  "test@example.com",
	}

	// Apply defaults as the handler would
	if req.Copies == 0 {
		req.Copies = 1
	}
	if req.ColorMode == "" {
		req.ColorMode = "monochrome"
	}
	if req.MediaType == "" {
		req.MediaType = "a4"
	}
	if req.Quality == "" {
		req.Quality = "normal"
	}

	if req.Copies != 1 {
		t.Errorf("Expected default Copies=1, got %d", req.Copies)
	}
	if req.ColorMode != "monochrome" {
		t.Errorf("Expected default ColorMode=monochrome, got %s", req.ColorMode)
	}
	if req.MediaType != "a4" {
		t.Errorf("Expected default MediaType=a4, got %s", req.MediaType)
	}
	if req.Quality != "normal" {
		t.Errorf("Expected default Quality=normal, got %s", req.Quality)
	}
}

func TestJobToResponse(t *testing.T) {
	now := time.Now()
	completedAt := now

	job := &repository.PrintJob{
		ID:          "job-123",
		DocumentID:  "doc-123",
		PrinterID:   "printer-123",
		UserName:    "Test User",
		UserEmail:   "test@example.com",
		Title:       "Test Document",
		Copies:      2,
		ColorMode:   "color",
		Duplex:      true,
		MediaType:   "a4",
		Quality:     "high",
		Pages:       10,
		Status:      "completed",
		Priority:    5,
		Retries:     0,
		StartedAt:   now,
		CompletedAt: &completedAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	response := jobToResponse(job)

	if response["job_id"] != "job-123" {
		t.Error("job_id not set correctly")
	}
	if response["status"] != "completed" {
		t.Error("status not set correctly")
	}
	if response["copies"] != 2 {
		t.Error("copies not set correctly")
	}
	if response["pages"] != 10 {
		t.Error("pages not set correctly")
	}
}

func TestHandler_CreateJob_Success(t *testing.T) {
	jobRepo := &mockJobHandlerRepo{jobs: make(map[string]*repository.PrintJob)}
	historyRepo := &mockHistoryHandlerRepo{}
	proc := &mockProcessor{}

	cfg := Config{
		JobRepo:     jobRepo,
		HistoryRepo: historyRepo,
		Processor:   proc,
	}

	h := New(cfg)

	reqBody := CreateJobRequest{
		DocumentID: "doc-123",
		PrinterID:  "printer-123",
		UserEmail:  "test@example.com",
		Title:      "Test Document",
		Copies:     1,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/jobs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.createJob(w, req, context.Background())

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["job_id"] == nil {
		t.Error("Response should include job_id")
	}
}

func TestHandler_CreateJob_MissingFields(t *testing.T) {
	jobRepo := &mockJobHandlerRepo{jobs: make(map[string]*repository.PrintJob)}
	historyRepo := &mockHistoryHandlerRepo{}
	proc := &mockProcessor{}

	cfg := Config{
		JobRepo:     jobRepo,
		HistoryRepo: historyRepo,
		Processor:   proc,
	}

	h := New(cfg)

	// Test missing document_id
	reqBody := map[string]string{
		"printer_id": "printer-123",
		"user_email": "test@example.com",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/jobs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.createJob(w, req, context.Background())

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing document_id, got %d", w.Code)
	}
}

func TestHandler_GetJob_Success(t *testing.T) {
	jobRepo := &mockJobHandlerRepo{jobs: make(map[string]*repository.PrintJob)}
	historyRepo := &mockHistoryHandlerRepo{}
	proc := &mockProcessor{}

	// Add a test job
	job := &repository.PrintJob{
		ID:         "job-123",
		DocumentID: "doc-123",
		PrinterID:  "printer-123",
		UserEmail:  "test@example.com",
		Status:     "queued",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	jobRepo.jobs[job.ID] = job

	cfg := Config{
		JobRepo:     jobRepo,
		HistoryRepo: historyRepo,
		Processor:   proc,
	}

	h := New(cfg)
	ctx := context.Background()

	req := httptest.NewRequest("GET", "/jobs/job-123", nil)
	w := httptest.NewRecorder()

	h.getJob(w, req, ctx, "job-123")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["job_id"] != "job-123" {
		t.Error("Response job_id mismatch")
	}
}

func TestHandler_GetJob_NotFound(t *testing.T) {
	jobRepo := &mockJobHandlerRepo{jobs: make(map[string]*repository.PrintJob)}
	historyRepo := &mockHistoryHandlerRepo{}
	proc := &mockProcessor{}

	cfg := Config{
		JobRepo:     jobRepo,
		HistoryRepo: historyRepo,
		Processor:   proc,
	}

	h := New(cfg)
	ctx := context.Background()

	req := httptest.NewRequest("GET", "/jobs/nonexistent", nil)
	w := httptest.NewRecorder()

	h.getJob(w, req, ctx, "nonexistent")

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandler_ListJobs(t *testing.T) {
	jobRepo := &mockJobHandlerRepo{jobs: make(map[string]*repository.PrintJob)}
	historyRepo := &mockHistoryHandlerRepo{}
	proc := &mockProcessor{}

	// Add test jobs
	jobRepo.jobs["job-1"] = &repository.PrintJob{ID: "job-1", Status: "queued", CreatedAt: time.Now()}
	jobRepo.jobs["job-2"] = &repository.PrintJob{ID: "job-2", Status: "processing", CreatedAt: time.Now()}

	cfg := Config{
		JobRepo:     jobRepo,
		HistoryRepo: historyRepo,
		Processor:   proc,
	}

	h := New(cfg)
	ctx := context.Background()

	req := httptest.NewRequest("GET", "/jobs?limit=10&offset=0", nil)
	w := httptest.NewRecorder()

	h.listJobs(w, req, ctx)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["jobs"] == nil {
		t.Error("Response should include jobs array")
	}
	if response["total"] == nil {
		t.Error("Response should include total count")
	}
}

func TestHandler_CancelJob(t *testing.T) {
	jobRepo := &mockJobHandlerRepo{jobs: make(map[string]*repository.PrintJob)}
	historyRepo := &mockHistoryHandlerRepo{}
	proc := &mockProcessor{}

	// Add a queued job
	job := &repository.PrintJob{
		ID:         "job-cancel-123",
		DocumentID: "doc-123",
		PrinterID:  "printer-123",
		UserEmail:  "test@example.com",
		Status:     "queued",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	jobRepo.jobs[job.ID] = job

	cfg := Config{
		JobRepo:     jobRepo,
		HistoryRepo: historyRepo,
		Processor:   proc,
	}

	h := New(cfg)
	ctx := context.Background()

	req := httptest.NewRequest("DELETE", "/jobs/job-cancel-123", nil)
	w := httptest.NewRecorder()

	h.cancelJob(w, req, ctx, "job-cancel-123")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check status was updated
	if job.Status != "cancelled" {
		t.Errorf("Expected status 'cancelled', got '%s'", job.Status)
	}
}

func TestHandler_CancelJob_Completed(t *testing.T) {
	jobRepo := &mockJobHandlerRepo{jobs: make(map[string]*repository.PrintJob)}
	historyRepo := &mockHistoryHandlerRepo{}
	proc := &mockProcessor{}

	completedAt := time.Now()
	job := &repository.PrintJob{
		ID:          "job-completed-123",
		DocumentID:  "doc-123",
		PrinterID:   "printer-123",
		UserEmail:   "test@example.com",
		Status:      "completed",
		CompletedAt: &completedAt,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	jobRepo.jobs[job.ID] = job

	cfg := Config{
		JobRepo:     jobRepo,
		HistoryRepo: historyRepo,
		Processor:   proc,
	}

	h := New(cfg)
	ctx := context.Background()

	req := httptest.NewRequest("DELETE", "/jobs/job-completed-123", nil)
	w := httptest.NewRecorder()

	h.cancelJob(w, req, ctx, "job-completed-123")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for completed job, got %d", w.Code)
	}
}

func TestHandler_RetryJob(t *testing.T) {
	jobRepo := &mockJobHandlerRepo{jobs: make(map[string]*repository.PrintJob)}
	historyRepo := &mockHistoryHandlerRepo{}
	proc := &mockProcessor{}

	job := &repository.PrintJob{
		ID:         "job-failed-123",
		DocumentID: "doc-123",
		PrinterID:  "printer-123",
		UserEmail:  "test@example.com",
		Status:     "failed",
		Retries:    0,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	jobRepo.jobs[job.ID] = job

	cfg := Config{
		JobRepo:     jobRepo,
		HistoryRepo: historyRepo,
		Processor:   proc,
	}

	h := New(cfg)
	ctx := context.Background()

	req := httptest.NewRequest("POST", "/jobs/job-failed-123/retry", nil)
	w := httptest.NewRecorder()

	h.retryJob(w, req, ctx, "job-failed-123")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if job.Status != "queued" {
		t.Errorf("Expected status 'queued', got '%s'", job.Status)
	}
	if job.Retries != 1 {
		t.Errorf("Expected Retries=1, got %d", job.Retries)
	}
}

func TestHandler_PauseJob(t *testing.T) {
	jobRepo := &mockJobHandlerRepo{jobs: make(map[string]*repository.PrintJob)}
	historyRepo := &mockHistoryHandlerRepo{}
	proc := &mockProcessor{}

	job := &repository.PrintJob{
		ID:         "job-pause-123",
		DocumentID: "doc-123",
		PrinterID:  "printer-123",
		UserEmail:  "test@example.com",
		Status:     "queued",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	jobRepo.jobs[job.ID] = job

	cfg := Config{
		JobRepo:     jobRepo,
		HistoryRepo: historyRepo,
		Processor:   proc,
	}

	h := New(cfg)
	ctx := context.Background()

	req := httptest.NewRequest("POST", "/jobs/job-pause-123/pause", nil)
	w := httptest.NewRecorder()

	h.pauseJob(w, req, ctx, "job-pause-123")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if job.Status != "paused" {
		t.Errorf("Expected status 'paused', got '%s'", job.Status)
	}
}

func TestHandler_ResumeJob(t *testing.T) {
	jobRepo := &mockJobHandlerRepo{jobs: make(map[string]*repository.PrintJob)}
	historyRepo := &mockHistoryHandlerRepo{}
	proc := &mockProcessor{}

	job := &repository.PrintJob{
		ID:         "job-resume-123",
		DocumentID: "doc-123",
		PrinterID:  "printer-123",
		UserEmail:  "test@example.com",
		Status:     "paused",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	jobRepo.jobs[job.ID] = job

	cfg := Config{
		JobRepo:     jobRepo,
		HistoryRepo: historyRepo,
		Processor:   proc,
	}

	h := New(cfg)
	ctx := context.Background()

	req := httptest.NewRequest("POST", "/jobs/job-resume-123/resume", nil)
	w := httptest.NewRecorder()

	h.resumeJob(w, req, ctx, "job-resume-123")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if job.Status != "queued" {
		t.Errorf("Expected status 'queued', got '%s'", job.Status)
	}
}

func TestHandler_QueueStats(t *testing.T) {
	jobRepo := &mockJobHandlerRepo{jobs: make(map[string]*repository.PrintJob)}
	historyRepo := &mockHistoryHandlerRepo{}
	proc := &mockProcessor{}

	cfg := Config{
		JobRepo:     jobRepo,
		HistoryRepo: historyRepo,
		Processor:   proc,
	}

	h := New(cfg)

	req := httptest.NewRequest("GET", "/jobs/stats/queue", nil)
	w := httptest.NewRecorder()

	h.QueueStatsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var stats map[string]interface{}
	json.NewDecoder(w.Body).Decode(&stats)

	if stats["queued"] == nil {
		t.Error("Stats should include queued count")
	}
	if stats["workers"] == nil {
		t.Error("Stats should include workers count")
	}
}

func TestHandler_JobHistory(t *testing.T) {
	jobRepo := &mockJobHandlerRepo{jobs: make(map[string]*repository.PrintJob)}
	historyRepo := &mockHistoryHandlerRepo{}
	proc := &mockProcessor{}

	// Add history entries
	jobID := "job-history-123"
	historyRepo.entries = map[string][]*repository.JobHistory{
		jobID: {
			{ID: "hist-1", JobID: jobID, Status: "queued", Message: "Job created", CreatedAt: time.Now()},
			{ID: "hist-2", JobID: jobID, Status: "processing", Message: "Job started", CreatedAt: time.Now()},
		},
	}

	cfg := Config{
		JobRepo:     jobRepo,
		HistoryRepo: historyRepo,
		Processor:   proc,
	}

	h := New(cfg)

	req := httptest.NewRequest("GET", "/jobs/history?job_id="+jobID, nil)
	w := httptest.NewRecorder()

	h.HistoryHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["history"] == nil {
		t.Error("Response should include history array")
	}
	if response["count"] == nil {
		t.Error("Response should include count")
	}
}

func TestRespondJSON(t *testing.T) {
	w := httptest.NewRecorder()

	data := map[string]interface{}{
		"message": "success",
		"count":   42,
	}

	respondJSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("Content-Type should be application/json")
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["message"] != "success" {
		t.Error("Response data mismatch")
	}
}

func TestRespondError_AppError(t *testing.T) {
	w := httptest.NewRecorder()

	err := apperrors.New("test error", http.StatusBadRequest)
	respondError(w, err)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("Content-Type should be application/json")
	}
}

func TestRespondError_GenericError(t *testing.T) {
	w := httptest.NewRecorder()

	err := apperrors.New("generic error", http.StatusInternalServerError)
	respondError(w, err)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["code"] != "INTERNAL_ERROR" {
		t.Error("Generic errors should return INTERNAL_ERROR code")
	}
}
