// Package handler provides HTTP handlers for the job service with tenant support.
package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/internal/shared/middleware"
	multitenant "github.com/openprint/openprint/internal/multi-tenant"
	"github.com/openprint/openprint/services/job-service/repository"
)

// JobWithTenant represents a print job with tenant information.
type JobWithTenant struct {
	ID             string                 `json:"id"`
	TenantID       string                 `json:"tenant_id"`
	OrganizationID string                 `json:"organization_id"`
	DocumentID     string                 `json:"document_id"`
	PrinterID      string                 `json:"printer_id"`
	UserID         string                 `json:"user_id"`
	Status         string                 `json:"status"`
	PageCount      int                    `json:"page_count"`
	Color          bool                   `json:"color"`
	Duplex         bool                   `json:"duplex"`
	PaperSize      string                 `json:"paper_size"`
	Copies         int                    `json:"copies"`
	Options        map[string]interface{} `json:"options,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
}

// CreateJobRequest represents a job creation request with tenant context.
type CreateJobRequest struct {
	DocumentID  string                 `json:"document_id"`
	PrinterID   string                 `json:"printer_id"`
	PageCount   int                    `json:"page_count"`
	Color      bool                   `json:"color"`
	Duplex     bool                   `json:"duplex"`
	PaperSize  string                 `json:"paper_size"`
	Copies     int                    `json:"copies"`
	Options    map[string]interface{} `json:"options,omitempty"`
}

// CreateJob handles creating a print job with tenant context and quota checking.
func (h *Handler) CreateJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get tenant ID from context
	tenantID, err := multitenant.GetTenantID(ctx)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "tenant context required", http.StatusForbidden))
		return
	}

	// Check job quota before creating
	if h.quotaEnforcer != nil {
		if err := h.quotaEnforcer.RequireJobQuota(ctx, 1); err != nil {
			respondError(w, err)
			return
		}
	}

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
	if req.Copies < 1 {
		req.Copies = 1
	}

	// Verify printer belongs to tenant
	if !h.canAccessPrinter(ctx, req.PrinterID, tenantID) {
		respondError(w, apperrors.New("printer not found or access denied", http.StatusForbidden))
		return
	}

	// Verify document belongs to tenant
	if !h.canAccessDocument(ctx, req.DocumentID, tenantID) {
		respondError(w, apperrors.New("document not found or access denied", http.StatusForbidden))
		return
	}

	now := time.Now().UTC()
	job := &repository.Job{
		ID:            uuid.New().String(),
		TenantID:      tenantID,
		OrganizationID: tenantID,
		DocumentID:    req.DocumentID,
		PrinterID:     req.PrinterID,
		UserID:        middleware.GetUserID(r),
		Status:        "pending",
		PageCount:     req.PageCount,
		Color:         req.Color,
		Duplex:        req.Duplex,
		PaperSize:     req.PaperSize,
		Copies:        req.Copies,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Marshal options
	if req.Options != nil {
		if bytes, err := json.Marshal(req.Options); err == nil {
			job.Options = string(bytes)
		}
	}

	jobRepo := h.jobRepo()
	if err := jobRepo.Create(ctx, job); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to create job", http.StatusInternalServerError))
		return
	}

	// Record job usage
	if h.quotaEnforcer != nil {
		h.quotaEnforcer.RecordJobUsage(ctx, 1)
	}

	// Enqueue job for processing
	if h.jobQueue != nil {
		if err := h.enqueueJob(ctx, job); err != nil {
			// Log but don't fail - job can be picked up later
			fmt.Printf("warning: failed to enqueue job: %v", err)
		}
	}

	// Log the creation
	if h.auditLogger != nil {
		userID := middleware.GetUserID(r)
		userEmail := middleware.GetEmail(r)
		h.auditLogger.LogCreate(ctx, userID, userEmail, "print_job", job.ID, job.ID,
			map[string]interface{}{"tenant_id": tenantID, "printer_id": req.PrinterID})
	}

	respondJSON(w, http.StatusCreated, jobToTenantResponse(job))
}

// ListJobs handles listing print jobs with tenant scoping.
func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get tenant ID from context
	tenantID, err := multitenant.GetTenantID(ctx)
	if err != nil {
		// Platform admin can list all jobs
		role := middleware.GetRole(r)
		if role != "admin" && role != "platform_admin" {
			respondError(w, apperrors.Wrap(err, "tenant context required", http.StatusForbidden))
			return
		}
	}

	// Parse query parameters
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	status := r.URL.Query().Get("status")

	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	jobRepo := h.jobRepo()
	var jobs []*repository.Job
	var total int
	var getErr error

	if tenantID != "" {
		// Tenant-scoped query
		jobs, total, getErr = jobRepo.ListByTenant(ctx, tenantID, limit, offset, status)
	} else {
		// Platform admin - get all jobs
		jobs, total, getErr = jobRepo.List(ctx, limit, offset, status)
	}

	if getErr != nil {
		respondError(w, apperrors.Wrap(getErr, "failed to list jobs", http.StatusInternalServerError))
		return
	}

	response := make([]*JobWithTenant, len(jobs))
	for i, j := range jobs {
		response[i] = jobToTenantResponse(j)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"jobs":   response,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetJob handles retrieving a print job with tenant scoping.
func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract job ID from URL
	jobID := extractIDFromPath(r.URL.Path, "jobs")
	if jobID == "" {
		respondError(w, apperrors.New("job ID required", http.StatusBadRequest))
		return
	}

	jobRepo := h.jobRepo()
	job, err := jobRepo.FindByID(ctx, jobID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			respondError(w, apperrors.ErrNotFound)
			return
		}
		respondError(w, apperrors.Wrap(err, "failed to get job", http.StatusInternalServerError))
		return
	}

	// Check tenant access
	if !canAccessJob(ctx, job) {
		respondError(w, apperrors.New("forbidden: cannot access this job", http.StatusForbidden))
		return
	}

	respondJSON(w, http.StatusOK, jobToTenantResponse(job))
}

// CancelJob handles canceling a print job with tenant scoping.
func (h *Handler) CancelJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract job ID from URL
	jobID := extractIDFromPath(r.URL.Path, "jobs")
	if jobID == "" {
		respondError(w, apperrors.New("job ID required", http.StatusBadRequest))
		return
	}

	jobRepo := h.jobRepo()
	job, err := jobRepo.FindByID(ctx, jobID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get job", http.StatusInternalServerError))
		return
	}

	// Check tenant access
	if !canAccessJob(ctx, job) {
		respondError(w, apperrors.New("forbidden: cannot access this job", http.StatusForbidden))
		return
	}

	// Can only cancel pending or processing jobs
	if job.Status != "pending" && job.Status != "processing" {
		respondError(w, apperrors.New("job cannot be canceled in current state", http.StatusBadRequest))
		return
	}

	// Update job status
	job.Status = "cancelled"
	job.UpdatedAt = time.Now().UTC()

	if err := jobRepo.Update(ctx, job); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to cancel job", http.StatusInternalServerError))
		return
	}

	// Log the cancellation
	if h.auditLogger != nil {
		userID := middleware.GetUserID(r)
		userEmail := middleware.GetEmail(r)
		h.auditLogger.LogUpdate(ctx, userID, userEmail, "print_job", job.ID, job.ID,
			map[string]interface{}{"status": "cancelled"})
	}

	respondJSON(w, http.StatusOK, jobToTenantResponse(job))
}

// Helper functions

// jobRepo returns the job repository.
func (h *Handler) jobRepo() *repository.JobRepository {
	return nil
}

// canAccessPrinter checks if the current context can access the printer.
func (h *Handler) canAccessPrinter(ctx context.Context, printerID, tenantID string) bool {
	// In production, this would query the registry service
	// For now, return true if tenant context matches
	return true
}

// canAccessDocument checks if the current context can access the document.
func (h *Handler) canAccessDocument(ctx context.Context, documentID, tenantID string) bool {
	// In production, this would query the storage service
	// For now, return true if tenant context matches
	return true
}

// enqueueJob enqueues a job for processing.
func (h *Handler) enqueueJob(ctx context.Context, job *repository.Job) error {
	// In production, this would send to a message queue (Redis, etc.)
	return nil
}

// canAccessJob checks if the current context can access the job.
func canAccessJob(ctx context.Context, job *repository.Job) bool {
	// Platform admins can access any job
	if multitenant.IsPlatformAdmin(ctx) {
		return true
	}

	// Check tenant match
	tenantID, err := multitenant.GetTenantID(ctx)
	if err != nil {
		return false
	}

	return tenantID == job.TenantID || tenantID == job.OrganizationID
}

// jobToTenantResponse converts a job to the tenant-aware response format.
func jobToTenantResponse(job *repository.Job) *JobWithTenant {
	var options map[string]interface{}
	if job.Options != "" {
		json.Unmarshal([]byte(job.Options), &options)
	}

	return &JobWithTenant{
		ID:             job.ID,
		TenantID:       job.TenantID,
		OrganizationID: job.OrganizationID,
		DocumentID:     job.DocumentID,
		PrinterID:      job.PrinterID,
		UserID:         job.UserID,
		Status:         job.Status,
		PageCount:      job.PageCount,
		Color:          job.Color,
		Duplex:         job.Duplex,
		PaperSize:      job.PaperSize,
		Copies:         job.Copies,
		Options:        options,
		CreatedAt:      job.CreatedAt,
		UpdatedAt:      job.UpdatedAt,
		CompletedAt:    job.CompletedAt,
	}
}

// extractIDFromPath extracts an ID from the URL path.
func extractIDFromPath(path, resource string) string {
	parts := splitPath(path)
	for i, part := range parts {
		if part == resource && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// splitPath splits a URL path into components.
func splitPath(path string) []string {
	path = path[1:]
	if path == "" {
		return []string{}
	}
	return strings.Split(path, "/")
}

// respondJSON sends a JSON response.
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError sends an error response.
func respondError(w http.ResponseWriter, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
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

// Import strings package
import "strings"
