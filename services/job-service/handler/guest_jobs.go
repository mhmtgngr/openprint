// Package handler provides HTTP handlers for the job service.
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/internal/shared/telemetry/prometheus"
	"github.com/openprint/openprint/services/job-service/repository"
)

// GuestJobHandler handles guest print job submissions that integrate with the job queue.
type GuestJobHandler struct {
	db          *pgxpool.Pool
	jobRepo     Repository
	historyRepo HistoryRepository
	processor   Processor
	metrics     *prometheus.Metrics
	serviceName string
}

// GuestJobHandlerConfig holds guest job handler dependencies.
type GuestJobHandlerConfig struct {
	DB          *pgxpool.Pool
	JobRepo     Repository
	HistoryRepo HistoryRepository
	Processor   Processor
	Metrics     *prometheus.Metrics
	ServiceName string
}

// NewGuestJobHandler creates a new guest job handler instance.
func NewGuestJobHandler(cfg GuestJobHandlerConfig) *GuestJobHandler {
	serviceName := cfg.ServiceName
	if serviceName == "" {
		serviceName = "job-service"
	}
	return &GuestJobHandler{
		db:          cfg.DB,
		jobRepo:     cfg.JobRepo,
		historyRepo: cfg.HistoryRepo,
		processor:   cfg.Processor,
		metrics:     cfg.Metrics,
		serviceName: serviceName,
	}
}

// GuestJobSubmitRequest represents a guest job submission request that creates
// a full print job in the job queue.
type GuestJobSubmitRequest struct {
	Token        string `json:"token"`
	DocumentID   string `json:"document_id"`
	DocumentName string `json:"document_name"`
	PrinterID    string `json:"printer_id"`
	PageCount    int    `json:"page_count"`
	ColorMode    string `json:"color_mode"`
	Duplex       bool   `json:"duplex"`
	Copies       int    `json:"copies"`
}

// guestTokenInfo holds the token data retrieved from the guest_print_tokens table.
type guestTokenInfo struct {
	ID             string
	Email          string
	Name           string
	OrganizationID string
	PrinterIDs     []string
	MaxPages       int
	MaxJobs        int
	PagesUsed      int
	JobsUsed       int
	ColorAllowed   bool
	DuplexRequired bool
}

// GuestJobsHandler handles guest job submission and listing.
// POST creates a new guest print job in the main job queue.
// GET lists guest jobs by token.
func (gh *GuestJobHandler) GuestJobsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		gh.submitGuestJob(w, r)
	case http.MethodGet:
		gh.listGuestJobs(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// GuestJobStatusHandler handles GET requests to check the status of a specific guest job.
func (gh *GuestJobHandler) GuestJobStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Extract job ID from path: /guest/jobs/{id}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		respondError(w, apperrors.New("invalid job path", http.StatusBadRequest))
		return
	}
	jobID := parts[len(parts)-1]

	// Look up the guest job record
	var tokenID, documentName, status string
	var pageCount int
	var printerID *string
	var submittedAt time.Time
	var completedAt *time.Time
	var errorMessage *string

	err := gh.db.QueryRow(ctx,
		`SELECT gj.id, gj.token_id, gj.document_name, gj.page_count, gj.printer_id,
		        gj.status, gj.submitted_at, gj.completed_at, gj.error_message
		 FROM guest_print_jobs gj
		 WHERE gj.id = $1`,
		jobID,
	).Scan(&jobID, &tokenID, &documentName, &pageCount, &printerID,
		&status, &submittedAt, &completedAt, &errorMessage)
	if err != nil {
		if err == pgx.ErrNoRows {
			respondError(w, apperrors.ErrNotFound)
			return
		}
		respondError(w, apperrors.Wrap(err, "failed to get guest job", http.StatusInternalServerError))
		return
	}

	resp := map[string]interface{}{
		"job_id":        jobID,
		"token_id":      tokenID,
		"document_name": documentName,
		"page_count":    pageCount,
		"status":        status,
		"submitted_at":  submittedAt.Format(time.RFC3339),
	}
	if printerID != nil {
		resp["printer_id"] = *printerID
	}
	if completedAt != nil {
		resp["completed_at"] = completedAt.Format(time.RFC3339)
	}
	if errorMessage != nil {
		resp["error_message"] = *errorMessage
	}

	respondJSON(w, http.StatusOK, resp)
}

func (gh *GuestJobHandler) submitGuestJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req GuestJobSubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate required fields
	if req.Token == "" {
		respondError(w, apperrors.New("token is required", http.StatusBadRequest))
		return
	}
	if req.DocumentID == "" {
		respondError(w, apperrors.New("document_id is required", http.StatusBadRequest))
		return
	}
	if req.PrinterID == "" {
		respondError(w, apperrors.New("printer_id is required", http.StatusBadRequest))
		return
	}
	if req.PageCount <= 0 {
		respondError(w, apperrors.New("page_count must be greater than 0", http.StatusBadRequest))
		return
	}

	// Validate the guest token
	token, err := gh.lookupGuestToken(ctx, req.Token)
	if err != nil {
		respondError(w, apperrors.New("invalid or expired guest token", http.StatusUnauthorized))
		return
	}

	// Check job quota
	if token.JobsUsed >= token.MaxJobs {
		respondError(w, apperrors.New("job quota exceeded for this guest token", http.StatusForbidden))
		return
	}

	// Check page quota
	if token.PagesUsed+req.PageCount > token.MaxPages {
		respondError(w, apperrors.New("page quota would be exceeded for this guest token", http.StatusForbidden))
		return
	}

	// Validate printer is in the allowed list (if restricted)
	if len(token.PrinterIDs) > 0 {
		allowed := false
		for _, pid := range token.PrinterIDs {
			if pid == req.PrinterID {
				allowed = true
				break
			}
		}
		if !allowed {
			respondError(w, apperrors.New("printer not allowed for this guest token", http.StatusForbidden))
			return
		}
	}

	// Enforce color policy
	if req.ColorMode == "" {
		req.ColorMode = "monochrome"
	}
	if req.ColorMode == "color" && !token.ColorAllowed {
		respondError(w, apperrors.New("color printing not allowed for this guest token", http.StatusForbidden))
		return
	}

	// Enforce duplex policy
	if token.DuplexRequired {
		req.Duplex = true
	}

	// Set defaults
	if req.Copies <= 0 {
		req.Copies = 1
	}
	if req.DocumentName == "" {
		req.DocumentName = "Guest Print Job"
	}

	// Begin transaction
	tx, err := gh.db.Begin(ctx)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to begin transaction", http.StatusInternalServerError))
		return
	}
	defer tx.Rollback(ctx)

	// Create the guest_print_jobs tracking record
	var guestJobID string
	err = tx.QueryRow(ctx,
		`INSERT INTO guest_print_jobs (token_id, document_name, page_count, printer_id, status)
		 VALUES ($1, $2, $3, $4, 'queued')
		 RETURNING id`,
		token.ID, req.DocumentName, req.PageCount, req.PrinterID,
	).Scan(&guestJobID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to create guest job record", http.StatusInternalServerError))
		return
	}

	// Update token usage counters
	_, err = tx.Exec(ctx,
		`UPDATE guest_print_tokens
		 SET pages_used = pages_used + $1, jobs_used = jobs_used + 1, last_used_at = NOW()
		 WHERE id = $2`,
		req.PageCount, token.ID,
	)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to update token usage", http.StatusInternalServerError))
		return
	}

	if err := tx.Commit(ctx); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to commit transaction", http.StatusInternalServerError))
		return
	}

	// Create the main print job in the job queue
	guestUserName := token.Name
	if guestUserName == "" {
		guestUserName = "Guest"
	}
	guestUserEmail := token.Email
	if guestUserEmail == "" {
		guestUserEmail = fmt.Sprintf("guest+%s@openprint.local", token.ID[:8])
	}

	job := &repository.PrintJob{
		ID:         uuid.New().String(),
		DocumentID: req.DocumentID,
		PrinterID:  req.PrinterID,
		UserName:   guestUserName,
		UserEmail:  guestUserEmail,
		Title:      req.DocumentName,
		Copies:     req.Copies,
		ColorMode:  req.ColorMode,
		Duplex:     req.Duplex,
		MediaType:  "a4",
		Quality:    "normal",
		Pages:      req.PageCount,
		Status:     "queued",
		Priority:   3, // Guest jobs get lower priority than authenticated user jobs
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := gh.jobRepo.Create(ctx, job); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to create print job", http.StatusInternalServerError))
		return
	}

	// Record job creation metric
	if gh.metrics != nil {
		gh.metrics.Business.JobsCreatedTotal.WithLabelValues(
			gh.serviceName,
			token.OrganizationID,
		).Inc()
	}

	// Enqueue for processing
	if err := gh.processor.Enqueue(ctx, job); err != nil {
		fmt.Printf("Failed to enqueue guest job: %v", err)
	}

	// Create history entry
	history := &repository.JobHistory{
		JobID:     job.ID,
		Status:    job.Status,
		Message:   fmt.Sprintf("Guest print job created via token %s", token.ID[:8]),
		CreatedAt: time.Now(),
	}
	gh.historyRepo.Create(ctx, history)

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"job_id":          job.ID,
		"guest_job_id":    guestJobID,
		"token_id":        token.ID,
		"document_id":     req.DocumentID,
		"document_name":   req.DocumentName,
		"printer_id":      req.PrinterID,
		"page_count":      req.PageCount,
		"color_mode":      req.ColorMode,
		"duplex":          req.Duplex,
		"copies":          req.Copies,
		"status":          "queued",
		"pages_remaining": token.MaxPages - token.PagesUsed - req.PageCount,
		"jobs_remaining":  token.MaxJobs - token.JobsUsed - 1,
	})
}

func (gh *GuestJobHandler) listGuestJobs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		respondError(w, apperrors.New("token query parameter is required", http.StatusBadRequest))
		return
	}

	// Validate the token is active
	token, err := gh.lookupGuestToken(ctx, tokenStr)
	if err != nil {
		respondError(w, apperrors.New("invalid or expired guest token", http.StatusUnauthorized))
		return
	}

	rows, err := gh.db.Query(ctx,
		`SELECT id, token_id, document_name, page_count, printer_id, status,
		        submitted_at, completed_at, error_message
		 FROM guest_print_jobs
		 WHERE token_id = $1
		 ORDER BY submitted_at DESC`,
		token.ID,
	)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to list guest jobs", http.StatusInternalServerError))
		return
	}
	defer rows.Close()

	jobs := make([]map[string]interface{}, 0)
	for rows.Next() {
		var (
			id, tokenID, documentName, status string
			pageCount                         int
			printerID                         *string
			submittedAt                       time.Time
			completedAt                       *time.Time
			errorMessage                      *string
		)
		if err := rows.Scan(&id, &tokenID, &documentName, &pageCount, &printerID,
			&status, &submittedAt, &completedAt, &errorMessage); err != nil {
			respondError(w, apperrors.Wrap(err, "failed to scan guest job", http.StatusInternalServerError))
			return
		}

		entry := map[string]interface{}{
			"job_id":        id,
			"token_id":      tokenID,
			"document_name": documentName,
			"page_count":    pageCount,
			"status":        status,
			"submitted_at":  submittedAt.Format(time.RFC3339),
		}
		if printerID != nil {
			entry["printer_id"] = *printerID
		}
		if completedAt != nil {
			entry["completed_at"] = completedAt.Format(time.RFC3339)
		}
		if errorMessage != nil {
			entry["error_message"] = *errorMessage
		}
		jobs = append(jobs, entry)
	}
	if err := rows.Err(); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to iterate guest jobs", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"jobs":  jobs,
		"count": len(jobs),
	})
}

// lookupGuestToken retrieves and validates a guest token from the database.
func (gh *GuestJobHandler) lookupGuestToken(ctx context.Context, tokenStr string) (*guestTokenInfo, error) {
	var t guestTokenInfo
	err := gh.db.QueryRow(ctx,
		`SELECT id, email, name, organization_id, printer_ids,
		        max_pages, max_jobs, pages_used, jobs_used, color_allowed, duplex_required
		 FROM guest_print_tokens
		 WHERE token = $1 AND is_active = true AND expires_at > NOW()`,
		tokenStr,
	).Scan(
		&t.ID, &t.Email, &t.Name, &t.OrganizationID, &t.PrinterIDs,
		&t.MaxPages, &t.MaxJobs, &t.PagesUsed, &t.JobsUsed,
		&t.ColorAllowed, &t.DuplexRequired,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
