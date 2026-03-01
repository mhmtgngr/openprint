// Package handler provides HTTP handlers for agent job polling.
package handler

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/internal/agent"
	"github.com/openprint/openprint/services/job-service/repository"
)

// AssignmentRepository defines the interface for job assignment operations.
type AssignmentRepository interface {
	GetJobsForAgentPolling(ctx context.Context, agentID, userEmail string, limit int) ([]*repository.PrintJob, error)
	AssignJob(ctx context.Context, assignment *repository.JobAssignment) error
	FindByAgent(ctx context.Context, agentID string, limit int) ([]*repository.JobAssignment, error)
	UpdateStatus(ctx context.Context, assignmentID, status string) error
	FindByJobAndAgent(ctx context.Context, jobID, agentID string) (*repository.JobAssignment, error)
	UpdateHeartbeat(ctx context.Context, assignmentID string, heartbeat time.Time) error
	GetJobWithPrinter(ctx context.Context, jobID string) (*repository.PrintJob, map[string]interface{}, error)
	ResolveUserDefaultPrinter(ctx context.Context, userEmail, clientAgentID string) (string, string, error)
}

// AgentPollConfig holds agent poll handler dependencies.
type AgentPollConfig struct {
	AssignmentRepo AssignmentRepository
	DocumentBaseURL string // Base URL for document downloads
	AssignmentTimeout time.Duration
}

// AgentPollHandler handles job polling from agents.
type AgentPollHandler struct {
	assignmentRepo   AssignmentRepository
	documentBaseURL  string
	assignmentTimeout time.Duration
}

// NewAgentPollHandler creates a new agent poll handler.
func NewAgentPollHandler(cfg AgentPollConfig) *AgentPollHandler {
	return &AgentPollHandler{
		assignmentRepo:   cfg.AssignmentRepo,
		documentBaseURL:  cfg.DocumentBaseURL,
		assignmentTimeout: cfg.AssignmentTimeout,
	}
}

// PollJobs handles job polling requests from agents.
// POST /agents/jobs/poll
func (h *AgentPollHandler) PollJobs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req agent.JobPollRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondPollError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate request
	if req.AgentID == "" {
		respondPollError(w, apperrors.New("agent_id is required", http.StatusBadRequest))
		return
	}

	// Set default limit
	if req.Limit == 0 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100 // Max limit
	}

	// Get pending jobs for this agent
	jobs, err := h.assignmentRepo.GetJobsForAgentPolling(ctx, req.AgentID, req.UserEmail, req.Limit)
	if err != nil {
		respondPollError(w, apperrors.Wrap(err, "failed to fetch jobs", http.StatusInternalServerError))
		return
	}

	// Convert to assigned jobs format
	assignedJobs := make([]agent.AssignedJob, 0, len(jobs))
	now := time.Now()

	for _, job := range jobs {
		// Skip nil jobs
		if job == nil {
			continue
		}

		// Create assignment record
		assignment := &repository.JobAssignment{
			JobID:    job.ID,
			AgentID:  req.AgentID,
			Status:   "assigned",
		}

		if err := h.assignmentRepo.AssignJob(ctx, assignment); err != nil {
			// Log error but continue with other jobs
			fmt.Printf("Failed to create assignment for job %s: %v", job.ID, err)
			continue
		}

		// Parse options
		var options map[string]string
		if job.Options != "" {
			json.Unmarshal([]byte(job.Options), &options)
		}

		// Build document URL
		documentURL := fmt.Sprintf("%s/documents/%s", h.documentBaseURL, job.DocumentID)
		if req.UserEmail != "" {
			documentURL += fmt.Sprintf("?user_email=%s", req.UserEmail)
		}

		// Resolve __user_default__ printer to actual target printer
		printerID := job.PrinterID
		printerName := job.PrinterID
		if job.PrinterID == "__user_default__" {
			resolvedName, resolvedID, err := h.assignmentRepo.ResolveUserDefaultPrinter(ctx, job.UserEmail, req.AgentID)
			if err == nil && resolvedName != "" {
				printerName = resolvedName
				if resolvedID != "" {
					printerID = resolvedID
				}
			}
		}

		assignedJob := agent.AssignedJob{
			JobID:               job.ID,
			DocumentID:          job.DocumentID,
			DocumentURL:         documentURL,
			DocumentChecksum:    "", // Would be populated from documents table
			DocumentSize:        0,  // Would be populated from documents table
			DocumentContentType: "application/pdf", // Default
			PrinterID:           printerID,
			PrinterName:         printerName,
			UserName:            job.UserName,
			UserEmail:           job.UserEmail,
			Title:               job.Title,
			Copies:              job.Copies,
			ColorMode:           job.ColorMode,
			Duplex:              job.Duplex,
			MediaType:           job.MediaType,
			Quality:             job.Quality,
			Priority:            job.Priority,
			Options:             options,
			CreatedAt:           job.CreatedAt,
		}

		assignedJobs = append(assignedJobs, assignedJob)
	}

	response := agent.JobPollResponse{
		Jobs:       assignedJobs,
		HasMore:    len(jobs) >= req.Limit,
		ServerTime: now,
	}

	respondPollJSON(w, http.StatusOK, response)
}

// GetPendingJobs retrieves pending jobs for an agent without assigning them.
// GET /agents/{agent_id}/jobs/pending
func (h *AgentPollHandler) GetPendingJobs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract agent ID from path
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		respondPollError(w, apperrors.New("invalid path", http.StatusBadRequest))
		return
	}
	agentID := parts[1]

	userEmail := r.URL.Query().Get("user_email")
	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	jobs, err := h.assignmentRepo.GetJobsForAgentPolling(ctx, agentID, userEmail, limit)
	if err != nil {
		respondPollError(w, apperrors.Wrap(err, "failed to fetch jobs", http.StatusInternalServerError))
		return
	}

	// Convert to response format
	jobResponses := make([]map[string]interface{}, len(jobs))
	for i, job := range jobs {
		jobResponses[i] = map[string]interface{}{
			"job_id":      job.ID,
			"document_id": job.DocumentID,
			"printer_id":  job.PrinterID,
			"user_email":  job.UserEmail,
			"title":       job.Title,
			"copies":      job.Copies,
			"priority":    job.Priority,
			"created_at":  job.CreatedAt.Format(time.RFC3339),
		}
	}

	respondPollJSON(w, http.StatusOK, map[string]interface{}{
		"jobs":    jobResponses,
		"count":   len(jobResponses),
		"has_more": len(jobs) >= limit,
	})
}

// GetActiveJobs retrieves active/assigned jobs for an agent.
// GET /agents/{agent_id}/jobs/active
func (h *AgentPollHandler) GetActiveJobs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract agent ID from path
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		respondPollError(w, apperrors.New("invalid path", http.StatusBadRequest))
		return
	}
	agentID := parts[1]

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	assignments, err := h.assignmentRepo.FindByAgent(ctx, agentID, limit)
	if err != nil {
		respondPollError(w, apperrors.Wrap(err, "failed to fetch active jobs", http.StatusInternalServerError))
		return
	}

	// Get job details for each assignment
	jobResponses := make([]map[string]interface{}, 0)
	for _, assignment := range assignments {
		job, printerInfo, err := h.assignmentRepo.GetJobWithPrinter(ctx, assignment.JobID)
		if err != nil {
			continue
		}

		jobResponse := map[string]interface{}{
			"job_id":          job.ID,
			"document_id":     job.DocumentID,
			"printer_id":      job.PrinterID,
			"printer_name":    printerInfo["name"],
			"user_email":      job.UserEmail,
			"title":           job.Title,
			"status":          job.Status,
			"assignment_id":   assignment.ID,
			"assignment_status": assignment.Status,
			"assigned_at":     assignment.AssignedAt.Format(time.RFC3339),
			"retry_count":     assignment.RetryCount,
		}

		jobResponses = append(jobResponses, jobResponse)
	}

	respondPollJSON(w, http.StatusOK, map[string]interface{}{
		"jobs":  jobResponses,
		"count": len(jobResponses),
	})
}

// UpdateJobStatus handles job status updates from agents.
// PUT /agents/{agent_id}/jobs/{job_id}/status
func (h *AgentPollHandler) UpdateJobStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract agent and job IDs from path
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 5 {
		respondPollError(w, apperrors.New("invalid path", http.StatusBadRequest))
		return
	}
	agentID := parts[1]
	jobID := parts[3]

	var req agent.JobStatusUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondPollError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Find the assignment
	assignment, err := h.assignmentRepo.FindByJobAndAgent(ctx, jobID, agentID)
	if err != nil {
		respondPollError(w, apperrors.ErrNotFound)
		return
	}

	// Update assignment status
	if err := h.assignmentRepo.UpdateStatus(ctx, assignment.ID, req.Status); err != nil {
		respondPollError(w, apperrors.Wrap(err, "failed to update status", http.StatusInternalServerError))
		return
	}

	// Update heartbeat
	if err := h.assignmentRepo.UpdateHeartbeat(ctx, assignment.ID, req.Timestamp); err != nil {
		fmt.Printf("Failed to update heartbeat: %v", err)
	}

	// Build response
	response := map[string]interface{}{
		"job_id":    jobID,
		"status":    req.Status,
		"updated_at": time.Now().Format(time.RFC3339),
	}

	if req.Message != "" {
		response["message"] = req.Message
	}
	if req.PagesPrinted > 0 {
		response["pages_printed"] = req.PagesPrinted
	}

	respondPollJSON(w, http.StatusOK, response)
}

// JobHeartbeat sends a heartbeat for an in-progress job.
// POST /agents/{agent_id}/jobs/{job_id}/heartbeat
func (h *AgentPollHandler) JobHeartbeat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract agent and job IDs from path
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 5 {
		respondPollError(w, apperrors.New("invalid path", http.StatusBadRequest))
		return
	}
	agentID := parts[1]
	jobID := parts[3]

	// Find the assignment
	assignment, err := h.assignmentRepo.FindByJobAndAgent(ctx, jobID, agentID)
	if err != nil {
		respondPollError(w, apperrors.ErrNotFound)
		return
	}

	// Update heartbeat
	now := time.Now()
	if err := h.assignmentRepo.UpdateHeartbeat(ctx, assignment.ID, now); err != nil {
		respondPollError(w, apperrors.Wrap(err, "failed to update heartbeat", http.StatusInternalServerError))
		return
	}

	respondPollJSON(w, http.StatusOK, map[string]interface{}{
		"job_id":       jobID,
		"assignment_id": assignment.ID,
		"server_time":  now.Format(time.RFC3339),
	})
}

// CompleteJob marks a job as completed by an agent.
// POST /agents/{agent_id}/jobs/{job_id}/complete
func (h *AgentPollHandler) CompleteJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract agent and job IDs from path
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 5 {
		respondPollError(w, apperrors.New("invalid path", http.StatusBadRequest))
		return
	}
	agentID := parts[1]
	jobID := parts[3]

	var req struct {
		PagesPrinted int    `json:"pages_printed"`
		Message      string `json:"message"`
		DocumentETag string `json:"document_etag,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondPollError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Find and update assignment
	assignment, err := h.assignmentRepo.FindByJobAndAgent(ctx, jobID, agentID)
	if err != nil {
		respondPollError(w, apperrors.ErrNotFound)
		return
	}

	if err := h.assignmentRepo.UpdateStatus(ctx, assignment.ID, "completed"); err != nil {
		respondPollError(w, apperrors.Wrap(err, "failed to complete job", http.StatusInternalServerError))
		return
	}

	respondPollJSON(w, http.StatusOK, map[string]interface{}{
		"job_id":        jobID,
		"status":        "completed",
		"pages_printed": req.PagesPrinted,
		"completed_at":  time.Now().Format(time.RFC3339),
	})
}

// FailJob marks a job as failed by an agent.
// POST /agents/{agent_id}/jobs/{job_id}/fail
func (h *AgentPollHandler) FailJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract agent and job IDs from path
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 5 {
		respondPollError(w, apperrors.New("invalid path", http.StatusBadRequest))
		return
	}
	agentID := parts[1]
	jobID := parts[3]

	var req struct {
		ErrorCode string `json:"error_code"`
		Message   string `json:"message"`
		Retry     bool   `json:"retry"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondPollError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Find the assignment
	assignment, err := h.assignmentRepo.FindByJobAndAgent(ctx, jobID, agentID)
	if err != nil {
		respondPollError(w, apperrors.ErrNotFound)
		return
	}

	// Update assignment status to failed
	if err := h.assignmentRepo.UpdateStatus(ctx, assignment.ID, "failed"); err != nil {
		respondPollError(w, apperrors.Wrap(err, "failed to update job status", http.StatusInternalServerError))
		return
	}

	response := map[string]interface{}{
		"job_id":      jobID,
		"status":      "failed",
		"error_code":  req.ErrorCode,
		"message":     req.Message,
		"updated_at":  time.Now().Format(time.RFC3339),
	}

	if req.Retry {
		// Increment retry count
		// In production, this would requeue the job
		response["retry_eligible"] = true
		response["retry_count"] = assignment.RetryCount + 1
	}

	respondPollJSON(w, http.StatusOK, response)
}

// GetJobDetails retrieves details of a specific job for an agent.
// GET /agents/{agent_id}/jobs/{job_id}
func (h *AgentPollHandler) GetJobDetails(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract job ID from path
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		respondPollError(w, apperrors.New("invalid path", http.StatusBadRequest))
		return
	}
	jobID := parts[3]

	// Get job details
	job, printerInfo, err := h.assignmentRepo.GetJobWithPrinter(ctx, jobID)
	if err != nil {
		respondPollError(w, apperrors.ErrNotFound)
		return
	}

	// Parse options
	var options map[string]string
	if job.Options != "" {
		json.Unmarshal([]byte(job.Options), &options)
	}

	// Build document URL
	documentURL := fmt.Sprintf("%s/documents/%s", h.documentBaseURL, job.DocumentID)

	response := map[string]interface{}{
		"job_id":      job.ID,
		"document_id": job.DocumentID,
		"document_url": documentURL,
		"printer_id":  job.PrinterID,
		"printer_name": printerInfo["name"],
		"user_email":  job.UserEmail,
		"title":       job.Title,
		"copies":      job.Copies,
		"color_mode":  job.ColorMode,
		"duplex":      job.Duplex,
		"media_type":  job.MediaType,
		"quality":     job.Quality,
		"options":     options,
		"status":      job.Status,
		"priority":    job.Priority,
		"created_at":  job.CreatedAt.Format(time.RFC3339),
	}

	respondPollJSON(w, http.StatusOK, response)
}

// Helper functions

func respondPollJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondPollError(w http.ResponseWriter, err error) {
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
