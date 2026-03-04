// Package handler provides HTTP handlers for the job service.
package handler

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/internal/shared/telemetry/prometheus"
	"github.com/openprint/openprint/services/job-service/processor"
	"github.com/openprint/openprint/services/job-service/repository"
)

// Repository defines the interface for job repository operations used by handlers.
type Repository interface {
	Create(ctx context.Context, job *repository.PrintJob) error
	FindByID(ctx context.Context, id string) (*repository.PrintJob, error)
	Update(ctx context.Context, job *repository.PrintJob) error
	FindByStatus(ctx context.Context, status string, limit int) ([]*repository.PrintJob, error)
	CountByStatus(ctx context.Context, status string) (int64, error)
	UpdateStatus(ctx context.Context, jobID, status string) error
	Delete(ctx context.Context, id string) error
	FindByPrinter(ctx context.Context, printerID string, limit, offset int) ([]*repository.PrintJob, error)
	FindByUser(ctx context.Context, userEmail string, limit, offset int) ([]*repository.PrintJob, error)
	ListWithFilters(ctx context.Context, limit, offset int, printerID, status, userEmail string) ([]*repository.PrintJob, int, error)
}

// HistoryRepository defines the interface for job history repository operations used by handlers.
type HistoryRepository interface {
	FindByJobID(ctx context.Context, jobID string) ([]*repository.JobHistory, error)
	Create(ctx context.Context, history *repository.JobHistory) error
}

// Processor defines the interface for processor operations used by handlers.
type Processor interface {
	Cancel(ctx context.Context, jobID string)
	GetStats(ctx context.Context) (*processor.Stats, error)
	Enqueue(ctx context.Context, job *repository.PrintJob) error
}

// Config holds handler dependencies.
type Config struct {
	JobRepo     Repository
	HistoryRepo HistoryRepository
	Processor   Processor
	Metrics     *prometheus.Metrics
	ServiceName string
}

// Handler provides job service HTTP handlers.
type Handler struct {
	jobRepo     Repository
	historyRepo HistoryRepository
	processor   Processor
	metrics     *prometheus.Metrics
	serviceName string
}

// New creates a new handler instance.
func New(cfg Config) *Handler {
	serviceName := cfg.ServiceName
	if serviceName == "" {
		serviceName = "job-service"
	}
	return &Handler{
		jobRepo:     cfg.JobRepo,
		historyRepo: cfg.HistoryRepo,
		processor:   cfg.Processor,
		metrics:     cfg.Metrics,
		serviceName: serviceName,
	}
}

// CreateJobRequest represents a print job creation request.
type CreateJobRequest struct {
	DocumentID string            `json:"document_id"`
	PrinterID  string            `json:"printer_id"`
	UserName   string            `json:"user_name"`
	UserEmail  string            `json:"user_email"`
	Title      string            `json:"title"`
	Copies     int               `json:"copies"`
	ColorMode  string            `json:"color_mode"` // "color" or "monochrome"
	Duplex     bool              `json:"duplex"`
	MediaType  string            `json:"media_type"` // e.g., "a4", "letter"
	Quality    string            `json:"quality"`    // e.g., "draft", "normal", "high"
	Pages      int               `json:"pages,omitempty"`
	Options    map[string]string `json:"options,omitempty"`
}

// JobsHandler handles job list and creation.
func (h *Handler) JobsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		h.listJobs(w, r, ctx)
	case http.MethodPost:
		h.createJob(w, r, ctx)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) createJob(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	var req CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate request
	if req.DocumentID == "" {
		respondError(w, apperrors.New("document_id is required", http.StatusBadRequest))
		return
	}
	if req.PrinterID == "" {
		respondError(w, apperrors.New("printer_id is required", http.StatusBadRequest))
		return
	}
	if req.UserEmail == "" {
		respondError(w, apperrors.New("user_email is required", http.StatusBadRequest))
		return
	}

	// Set defaults
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

	// Create job
	job := &repository.PrintJob{
		ID:         uuid.New().String(),
		DocumentID: req.DocumentID,
		PrinterID:  req.PrinterID,
		UserName:   req.UserName,
		UserEmail:  req.UserEmail,
		Title:      req.Title,
		Copies:     req.Copies,
		ColorMode:  req.ColorMode,
		Duplex:     req.Duplex,
		MediaType:  req.MediaType,
		Quality:    req.Quality,
		Pages:      req.Pages,
		Status:     "queued",
		Priority:   5,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Serialize options
	if len(req.Options) > 0 {
		optionsJSON, _ := json.Marshal(req.Options)
		job.Options = string(optionsJSON)
	}

	if err := h.jobRepo.Create(ctx, job); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to create job", http.StatusInternalServerError))
		return
	}

	// Record job creation metric
	orgID := ""
	if job != nil {
		// Extract org ID from job context if available
		orgID = "" // Would need to be set on job
	}
	if h.metrics != nil {
		h.metrics.Business.JobsCreatedTotal.WithLabelValues(
			h.serviceName,
			orgID,
		).Inc()
	}

	// Add to processor queue
	if err := h.processor.Enqueue(ctx, job); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to enqueue job: %v", err)
	}

	// Create history entry
	history := &repository.JobHistory{
		JobID:     job.ID,
		Status:    job.Status,
		Message:   "Job created and queued",
		CreatedAt: time.Now(),
	}
	h.historyRepo.Create(ctx, history)

	respondJSON(w, http.StatusCreated, jobToResponse(job))
}

func (h *Handler) listJobs(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	// Parse pagination and filter params
	limit := 50
	offset := 0
	printerID := r.URL.Query().Get("printer_id")
	status := r.URL.Query().Get("status")
	userEmail := r.URL.Query().Get("user_email")

	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	// Get jobs
	jobs, total, err := h.jobRepo.ListWithFilters(ctx, limit, offset, printerID, status, userEmail)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to list jobs", http.StatusInternalServerError))
		return
	}

	// Convert to response
	response := make([]map[string]interface{}, len(jobs))
	for i, job := range jobs {
		response[i] = jobToResponse(job)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":   response,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// JobHandler handles individual job operations.
func (h *Handler) JobHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract job ID from path
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		respondError(w, apperrors.New("invalid job path", http.StatusBadRequest))
		return
	}
	jobID := parts[1]

	switch r.Method {
	case http.MethodGet:
		h.getJob(w, r, ctx, jobID)
	case http.MethodDelete:
		h.cancelJob(w, r, ctx, jobID)
	case http.MethodPost:
		// Check for specific actions
		if strings.HasSuffix(r.URL.Path, "/retry") {
			h.retryJob(w, r, ctx, jobID)
		} else if strings.HasSuffix(r.URL.Path, "/pause") {
			h.pauseJob(w, r, ctx, jobID)
		} else if strings.HasSuffix(r.URL.Path, "/resume") {
			h.resumeJob(w, r, ctx, jobID)
		} else {
			http.Error(w, "unknown action", http.StatusNotFound)
		}
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) getJob(w http.ResponseWriter, r *http.Request, ctx context.Context, jobID string) {
	job, err := h.jobRepo.FindByID(ctx, jobID)
	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	respondJSON(w, http.StatusOK, jobToResponse(job))
}

func (h *Handler) cancelJob(w http.ResponseWriter, r *http.Request, ctx context.Context, jobID string) {
	job, err := h.jobRepo.FindByID(ctx, jobID)
	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	if job.Status == "completed" || job.Status == "cancelled" {
		respondError(w, apperrors.New("cannot cancel job in current status", http.StatusBadRequest))
		return
	}

	// Update status
	job.Status = "cancelled"
	if err := h.jobRepo.UpdateStatus(ctx, jobID, "cancelled"); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to cancel job", http.StatusInternalServerError))
		return
	}

	// Create history entry
	history := &repository.JobHistory{
		JobID:     jobID,
		Status:    "cancelled",
		Message:   "Job cancelled by user",
		CreatedAt: time.Now(),
	}
	h.historyRepo.Create(ctx, history)

	// Notify processor
	h.processor.Cancel(ctx, jobID)

	respondJSON(w, http.StatusOK, jobToResponse(job))
}

func (h *Handler) retryJob(w http.ResponseWriter, r *http.Request, ctx context.Context, jobID string) {
	job, err := h.jobRepo.FindByID(ctx, jobID)
	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	if job.Status != "failed" {
		respondError(w, apperrors.New("can only retry failed jobs", http.StatusBadRequest))
		return
	}

	// Reset job for retry
	job.Status = "queued"
	job.Retries++
	job.UpdatedAt = time.Now()

	if err := h.jobRepo.Update(ctx, job); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to retry job", http.StatusInternalServerError))
		return
	}

	// Re-enqueue
	h.processor.Enqueue(ctx, job)

	// Create history entry
	history := &repository.JobHistory{
		JobID:     jobID,
		Status:    "queued",
		Message:   fmt.Sprintf("Job retry #%d", job.Retries),
		CreatedAt: time.Now(),
	}
	h.historyRepo.Create(ctx, history)

	respondJSON(w, http.StatusOK, jobToResponse(job))
}

func (h *Handler) pauseJob(w http.ResponseWriter, r *http.Request, ctx context.Context, jobID string) {
	job, err := h.jobRepo.FindByID(ctx, jobID)
	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	if job.Status != "queued" {
		respondError(w, apperrors.New("can only pause queued jobs", http.StatusBadRequest))
		return
	}

	if err := h.jobRepo.UpdateStatus(ctx, jobID, "paused"); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to pause job", http.StatusInternalServerError))
		return
	}

	history := &repository.JobHistory{
		JobID:     jobID,
		Status:    "paused",
		Message:   "Job paused",
		CreatedAt: time.Now(),
	}
	h.historyRepo.Create(ctx, history)

	respondJSON(w, http.StatusOK, jobToResponse(job))
}

func (h *Handler) resumeJob(w http.ResponseWriter, r *http.Request, ctx context.Context, jobID string) {
	job, err := h.jobRepo.FindByID(ctx, jobID)
	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	if job.Status != "paused" {
		respondError(w, apperrors.New("can only resume paused jobs", http.StatusBadRequest))
		return
	}

	job.Status = "queued"
	if err := h.jobRepo.Update(ctx, job); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to resume job", http.StatusInternalServerError))
		return
	}

	// Re-enqueue
	h.processor.Enqueue(ctx, job)

	history := &repository.JobHistory{
		JobID:     jobID,
		Status:    "queued",
		Message:   "Job resumed",
		CreatedAt: time.Now(),
	}
	h.historyRepo.Create(ctx, history)

	respondJSON(w, http.StatusOK, jobToResponse(job))
}

// JobStatusHandler handles job status updates from agents.
func (h *Handler) JobStatusHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract job ID from path
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		respondError(w, apperrors.New("invalid job path", http.StatusBadRequest))
		return
	}
	jobID := parts[2]

	var req struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Pages   int    `json:"pages"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Get job
	job, err := h.jobRepo.FindByID(ctx, jobID)
	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	// Update status
	job.Status = req.Status
	if req.Pages > 0 {
		job.Pages = req.Pages
	}
	job.UpdatedAt = time.Now()

	if req.Status == "processing" && job.StartedAt.IsZero() {
		job.StartedAt = time.Now()
	} else if req.Status == "completed" || req.Status == "failed" || req.Status == "cancelled" {
		job.CompletedAt = &[]time.Time{time.Now()}[0]
	}

	if err := h.jobRepo.Update(ctx, job); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to update job", http.StatusInternalServerError))
		return
	}

	// Create history entry
	history := &repository.JobHistory{
		JobID:     jobID,
		Status:    req.Status,
		Message:   req.Message,
		CreatedAt: time.Now(),
	}
	h.historyRepo.Create(ctx, history)

	// Record job status metrics
	if h.metrics != nil {
		orgID := "" // Would extract from job
		duration := float64(0)
		if !job.StartedAt.IsZero() {
			duration = time.Since(job.StartedAt).Seconds()
		}

		switch req.Status {
		case "completed":
			h.metrics.Business.JobsCompletedTotal.WithLabelValues(
				h.serviceName,
				orgID,
			).Inc()
			if duration > 0 {
				h.metrics.Business.JobProcessingDuration.WithLabelValues(
					h.serviceName,
					orgID,
				).Observe(duration)
			}
		case "failed":
			h.metrics.Business.JobsFailedTotal.WithLabelValues(
				h.serviceName,
				orgID,
				"",
			).Inc()
		}
	}

	respondJSON(w, http.StatusOK, jobToResponse(job))
}

// HistoryHandler handles job history requests.
func (h *Handler) HistoryHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract job ID from path
	jobID := r.URL.Query().Get("job_id")
	if jobID == "" {
		respondError(w, apperrors.New("job_id is required", http.StatusBadRequest))
		return
	}

	history, err := h.historyRepo.FindByJobID(ctx, jobID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get history", http.StatusInternalServerError))
		return
	}

	response := make([]map[string]interface{}, len(history))
	for i, h := range history {
		response[i] = map[string]interface{}{
			"id":         h.ID,
			"job_id":     h.JobID,
			"status":     h.Status,
			"message":    h.Message,
			"created_at": h.CreatedAt.Format(time.RFC3339),
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"history": response,
		"count":   len(response),
	})
}

// QueueStatsHandler returns queue statistics.
func (h *Handler) QueueStatsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := h.processor.GetStats(ctx)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get stats", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, stats)
}

// Helper functions

func jobToResponse(job *repository.PrintJob) map[string]interface{} {
	return map[string]interface{}{
		"job_id":      job.ID,
		"document_id": job.DocumentID,
		"printer_id":  job.PrinterID,
		"user_name":   job.UserName,
		"user_email":  job.UserEmail,
		"title":       job.Title,
		"copies":      job.Copies,
		"color_mode":  job.ColorMode,
		"duplex":      job.Duplex,
		"media_type":  job.MediaType,
		"quality":     job.Quality,
		"pages":       job.Pages,
		"status":      job.Status,
		"priority":    job.Priority,
		"retries":     job.Retries,
		"created_at":  job.CreatedAt.Format(time.RFC3339),
		"started_at":  job.StartedAt.Format(time.RFC3339),
		"completed_at": func() string {
			if job.CompletedAt != nil {
				return job.CompletedAt.Format(time.RFC3339)
			}
			return ""
		}(),
	}
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, err error) {
	var appErr *apperrors.AppError
	if stderrors.As(err, &appErr) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(appErr.StatusCode)
		json.NewEncoder(w).Encode(apperrors.ToJSON(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]string{
		"code":    "INTERNAL_ERROR",
		"message": "An internal error occurred",
	})
}
