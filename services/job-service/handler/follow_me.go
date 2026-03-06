// Package handler provides HTTP handlers for the job service.
package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
)

// FollowMeHandler provides Follow-Me (Find-Me / Roaming Print) HTTP handlers.
type FollowMeHandler struct {
	db *sql.DB
}

// NewFollowMeHandler creates a new FollowMeHandler instance.
func NewFollowMeHandler(db *sql.DB) *FollowMeHandler {
	return &FollowMeHandler{db: db}
}

// --- Request / Response types ---

// CreatePoolRequest represents a request to create a follow-me pool.
type CreatePoolRequest struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	OrganizationID string `json:"organization_id"`
	Location       string `json:"location"`
}

// UpdatePoolRequest represents a request to update a follow-me pool.
type UpdatePoolRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Location    string `json:"location"`
	IsActive    *bool  `json:"is_active"`
}

// AddPrinterRequest represents a request to add a printer to a pool.
type AddPrinterRequest struct {
	PrinterID string `json:"printer_id"`
	Priority  int    `json:"priority"`
}

// SubmitFollowMeJobRequest represents a request to submit a follow-me job.
type SubmitFollowMeJobRequest struct {
	JobID        string `json:"job_id"`
	PoolID       string `json:"pool_id"`
	UserID       string `json:"user_id"`
	UserEmail    string `json:"user_email"`
	DocumentName string `json:"document_name"`
	PageCount    int    `json:"page_count"`
	Copies       int    `json:"copies"`
	Color        bool   `json:"color"`
	Duplex       bool   `json:"duplex"`
}

// ReleaseAtPrinterRequest represents a request to release a job at a printer.
type ReleaseAtPrinterRequest struct {
	PrinterID string `json:"printer_id"`
}

// --- Pool CRUD handlers ---

// CreatePool handles POST requests to create a new follow-me printer pool.
func (h *FollowMeHandler) CreatePool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreatePoolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	if req.Name == "" {
		respondError(w, apperrors.New("name is required", http.StatusBadRequest))
		return
	}
	if req.OrganizationID == "" {
		respondError(w, apperrors.New("organization_id is required", http.StatusBadRequest))
		return
	}

	id := uuid.New().String()
	now := time.Now()

	_, err := h.db.ExecContext(r.Context(),
		`INSERT INTO follow_me_pools (id, name, description, organization_id, location, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, true, $6, $7)`,
		id, req.Name, req.Description, req.OrganizationID, req.Location, now, now,
	)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to create pool", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":              id,
		"name":            req.Name,
		"description":     req.Description,
		"organization_id": req.OrganizationID,
		"location":        req.Location,
		"is_active":       true,
		"created_at":      now.Format(time.RFC3339),
		"updated_at":      now.Format(time.RFC3339),
	})
}

// ListPools handles GET requests to list pools for an organization.
func (h *FollowMeHandler) ListPools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgID := r.URL.Query().Get("organization_id")
	if orgID == "" {
		respondError(w, apperrors.New("organization_id is required", http.StatusBadRequest))
		return
	}

	rows, err := h.db.QueryContext(r.Context(),
		`SELECT id, name, description, organization_id, location, is_active, created_at, updated_at
		 FROM follow_me_pools WHERE organization_id = $1 ORDER BY name`, orgID,
	)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to list pools", http.StatusInternalServerError))
		return
	}
	defer rows.Close()

	pools := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id, name, orgID string
		var description, location sql.NullString
		var isActive bool
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&id, &name, &description, &orgID, &location, &isActive, &createdAt, &updatedAt); err != nil {
			respondError(w, apperrors.Wrap(err, "failed to scan pool", http.StatusInternalServerError))
			return
		}

		pools = append(pools, map[string]interface{}{
			"id":              id,
			"name":            name,
			"description":     description.String,
			"organization_id": orgID,
			"location":        location.String,
			"is_active":       isActive,
			"created_at":      createdAt.Format(time.RFC3339),
			"updated_at":      updatedAt.Format(time.RFC3339),
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"pools": pools,
		"total": len(pools),
	})
}

// GetPool handles GET requests to get pool details with member printers.
func (h *FollowMeHandler) GetPool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	poolID := extractPoolID(r.URL.Path)
	if poolID == "" {
		respondError(w, apperrors.New("invalid pool path", http.StatusBadRequest))
		return
	}

	// Get pool details
	var id, name, orgID string
	var description, location sql.NullString
	var isActive bool
	var createdAt, updatedAt time.Time

	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, name, description, organization_id, location, is_active, created_at, updated_at
		 FROM follow_me_pools WHERE id = $1`, poolID,
	).Scan(&id, &name, &description, &orgID, &location, &isActive, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		respondError(w, apperrors.ErrNotFound)
		return
	}
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get pool", http.StatusInternalServerError))
		return
	}

	// Get member printers
	printerRows, err := h.db.QueryContext(r.Context(),
		`SELECT printer_id, priority FROM follow_me_pool_printers WHERE pool_id = $1 ORDER BY priority DESC`, poolID,
	)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get pool printers", http.StatusInternalServerError))
		return
	}
	defer printerRows.Close()

	printers := make([]map[string]interface{}, 0)
	for printerRows.Next() {
		var printerID string
		var priority int
		if err := printerRows.Scan(&printerID, &priority); err != nil {
			respondError(w, apperrors.Wrap(err, "failed to scan printer", http.StatusInternalServerError))
			return
		}
		printers = append(printers, map[string]interface{}{
			"printer_id": printerID,
			"priority":   priority,
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":              id,
		"name":            name,
		"description":     description.String,
		"organization_id": orgID,
		"location":        location.String,
		"is_active":       isActive,
		"printers":        printers,
		"created_at":      createdAt.Format(time.RFC3339),
		"updated_at":      updatedAt.Format(time.RFC3339),
	})
}

// UpdatePool handles PUT requests to update pool settings.
func (h *FollowMeHandler) UpdatePool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	poolID := extractPoolID(r.URL.Path)
	if poolID == "" {
		respondError(w, apperrors.New("invalid pool path", http.StatusBadRequest))
		return
	}

	var req UpdatePoolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Verify pool exists
	var exists bool
	err := h.db.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM follow_me_pools WHERE id = $1)`, poolID).Scan(&exists)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to check pool", http.StatusInternalServerError))
		return
	}
	if !exists {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	now := time.Now()
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	_, err = h.db.ExecContext(r.Context(),
		`UPDATE follow_me_pools SET name = $1, description = $2, location = $3, is_active = $4, updated_at = $5 WHERE id = $6`,
		req.Name, req.Description, req.Location, isActive, now, poolID,
	)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to update pool", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":         poolID,
		"name":       req.Name,
		"description": req.Description,
		"location":   req.Location,
		"is_active":  isActive,
		"updated_at": now.Format(time.RFC3339),
	})
}

// DeletePool handles DELETE requests to remove a pool.
func (h *FollowMeHandler) DeletePool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	poolID := extractPoolID(r.URL.Path)
	if poolID == "" {
		respondError(w, apperrors.New("invalid pool path", http.StatusBadRequest))
		return
	}

	result, err := h.db.ExecContext(r.Context(), `DELETE FROM follow_me_pools WHERE id = $1`, poolID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to delete pool", http.StatusInternalServerError))
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "pool deleted",
	})
}

// --- Pool printer membership ---

// AddPrinterToPool handles POST requests to add a printer to a pool.
func (h *FollowMeHandler) AddPrinterToPool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	poolID := extractPoolID(r.URL.Path)
	if poolID == "" {
		respondError(w, apperrors.New("invalid pool path", http.StatusBadRequest))
		return
	}

	var req AddPrinterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	if req.PrinterID == "" {
		respondError(w, apperrors.New("printer_id is required", http.StatusBadRequest))
		return
	}

	// Verify pool exists
	var exists bool
	err := h.db.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM follow_me_pools WHERE id = $1)`, poolID).Scan(&exists)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to check pool", http.StatusInternalServerError))
		return
	}
	if !exists {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	_, err = h.db.ExecContext(r.Context(),
		`INSERT INTO follow_me_pool_printers (pool_id, printer_id, priority) VALUES ($1, $2, $3)
		 ON CONFLICT (pool_id, printer_id) DO UPDATE SET priority = EXCLUDED.priority`,
		poolID, req.PrinterID, req.Priority,
	)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to add printer to pool", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"pool_id":    poolID,
		"printer_id": req.PrinterID,
		"priority":   req.Priority,
	})
}

// RemovePrinterFromPool handles DELETE requests to remove a printer from a pool.
func (h *FollowMeHandler) RemovePrinterFromPool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	poolID, printerID := extractPoolAndPrinterID(r.URL.Path)
	if poolID == "" || printerID == "" {
		respondError(w, apperrors.New("invalid path, pool_id and printer_id are required", http.StatusBadRequest))
		return
	}

	result, err := h.db.ExecContext(r.Context(),
		`DELETE FROM follow_me_pool_printers WHERE pool_id = $1 AND printer_id = $2`,
		poolID, printerID,
	)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to remove printer from pool", http.StatusInternalServerError))
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "printer removed from pool",
	})
}

// --- Follow-me job handlers ---

// SubmitFollowMeJob handles POST requests to submit a job to a pool (no specific printer).
func (h *FollowMeHandler) SubmitFollowMeJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SubmitFollowMeJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	if req.PoolID == "" {
		respondError(w, apperrors.New("pool_id is required", http.StatusBadRequest))
		return
	}
	if req.UserID == "" {
		respondError(w, apperrors.New("user_id is required", http.StatusBadRequest))
		return
	}
	if req.UserEmail == "" {
		respondError(w, apperrors.New("user_email is required", http.StatusBadRequest))
		return
	}
	if req.DocumentName == "" {
		respondError(w, apperrors.New("document_name is required", http.StatusBadRequest))
		return
	}

	// Verify pool exists and is active
	var isActive bool
	err := h.db.QueryRowContext(r.Context(),
		`SELECT is_active FROM follow_me_pools WHERE id = $1`, req.PoolID,
	).Scan(&isActive)
	if err == sql.ErrNoRows {
		respondError(w, apperrors.New("pool not found", http.StatusNotFound))
		return
	}
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to check pool", http.StatusInternalServerError))
		return
	}
	if !isActive {
		respondError(w, apperrors.New("pool is not active", http.StatusBadRequest))
		return
	}

	// Set defaults
	if req.Copies == 0 {
		req.Copies = 1
	}
	if req.JobID == "" {
		req.JobID = uuid.New().String()
	}

	id := uuid.New().String()
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)

	_, err = h.db.ExecContext(r.Context(),
		`INSERT INTO follow_me_jobs (id, job_id, pool_id, user_id, user_email, document_name, page_count, copies, color, duplex, status, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'waiting', $11, $12)`,
		id, req.JobID, req.PoolID, req.UserID, req.UserEmail, req.DocumentName,
		req.PageCount, req.Copies, req.Color, req.Duplex, expiresAt, now,
	)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to submit follow-me job", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":            id,
		"job_id":        req.JobID,
		"pool_id":       req.PoolID,
		"user_id":       req.UserID,
		"user_email":    req.UserEmail,
		"document_name": req.DocumentName,
		"page_count":    req.PageCount,
		"copies":        req.Copies,
		"color":         req.Color,
		"duplex":        req.Duplex,
		"status":        "waiting",
		"expires_at":    expiresAt.Format(time.RFC3339),
		"created_at":    now.Format(time.RFC3339),
	})
}

// ListPendingJobs handles GET requests to list a user's pending follow-me jobs.
func (h *FollowMeHandler) ListPendingJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		respondError(w, apperrors.New("user_id is required", http.StatusBadRequest))
		return
	}

	rows, err := h.db.QueryContext(r.Context(),
		`SELECT fj.id, fj.job_id, fj.pool_id, fp.name AS pool_name, fj.user_id, fj.user_email,
		        fj.document_name, fj.page_count, fj.copies, fj.color, fj.duplex,
		        fj.status, fj.expires_at, fj.created_at
		 FROM follow_me_jobs fj
		 JOIN follow_me_pools fp ON fp.id = fj.pool_id
		 WHERE fj.user_id = $1 AND fj.status = 'waiting' AND fj.expires_at > NOW()
		 ORDER BY fj.created_at DESC`, userID,
	)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to list pending jobs", http.StatusInternalServerError))
		return
	}
	defer rows.Close()

	jobs := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id, jobID, poolID, poolName, uid, email, docName, status string
		var pageCount, copies int
		var color, duplex bool
		var expiresAt, createdAt time.Time

		if err := rows.Scan(&id, &jobID, &poolID, &poolName, &uid, &email,
			&docName, &pageCount, &copies, &color, &duplex,
			&status, &expiresAt, &createdAt); err != nil {
			respondError(w, apperrors.Wrap(err, "failed to scan job", http.StatusInternalServerError))
			return
		}

		jobs = append(jobs, map[string]interface{}{
			"id":            id,
			"job_id":        jobID,
			"pool_id":       poolID,
			"pool_name":     poolName,
			"user_id":       uid,
			"user_email":    email,
			"document_name": docName,
			"page_count":    pageCount,
			"copies":        copies,
			"color":         color,
			"duplex":        duplex,
			"status":        status,
			"expires_at":    expiresAt.Format(time.RFC3339),
			"created_at":    createdAt.Format(time.RFC3339),
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"jobs":  jobs,
		"total": len(jobs),
	})
}

// ReleaseAtPrinter handles POST requests to release a follow-me job at a specific printer.
func (h *FollowMeHandler) ReleaseAtPrinter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract follow-me job ID from path: /follow-me/jobs/{id}/release
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	var fmJobID string
	for i, p := range parts {
		if p == "jobs" && i+1 < len(parts) {
			fmJobID = parts[i+1]
			break
		}
	}
	if fmJobID == "" {
		respondError(w, apperrors.New("invalid job path", http.StatusBadRequest))
		return
	}

	var req ReleaseAtPrinterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	if req.PrinterID == "" {
		respondError(w, apperrors.New("printer_id is required", http.StatusBadRequest))
		return
	}

	// Get the follow-me job
	var poolID, status string
	var expiresAt time.Time
	err := h.db.QueryRowContext(r.Context(),
		`SELECT pool_id, status, expires_at FROM follow_me_jobs WHERE id = $1`, fmJobID,
	).Scan(&poolID, &status, &expiresAt)
	if err == sql.ErrNoRows {
		respondError(w, apperrors.ErrNotFound)
		return
	}
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get follow-me job", http.StatusInternalServerError))
		return
	}

	if status != "waiting" {
		respondError(w, apperrors.New("job is not in waiting status", http.StatusBadRequest))
		return
	}
	if time.Now().After(expiresAt) {
		respondError(w, apperrors.New("job has expired", http.StatusBadRequest))
		return
	}

	// Verify the printer belongs to the pool
	var printerInPool bool
	err = h.db.QueryRowContext(r.Context(),
		`SELECT EXISTS(SELECT 1 FROM follow_me_pool_printers WHERE pool_id = $1 AND printer_id = $2)`,
		poolID, req.PrinterID,
	).Scan(&printerInPool)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to verify printer membership", http.StatusInternalServerError))
		return
	}
	if !printerInPool {
		respondError(w, apperrors.New("printer is not a member of this pool", http.StatusBadRequest))
		return
	}

	// Release the job
	now := time.Now()
	_, err = h.db.ExecContext(r.Context(),
		`UPDATE follow_me_jobs SET status = 'released', released_at_printer = $1, released_at = $2 WHERE id = $3`,
		req.PrinterID, now, fmJobID,
	)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to release job", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":                  fmJobID,
		"status":              "released",
		"released_at_printer": req.PrinterID,
		"released_at":         now.Format(time.RFC3339),
	})
}

// CancelFollowMeJob handles DELETE requests to cancel a pending follow-me job.
func (h *FollowMeHandler) CancelFollowMeJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract follow-me job ID from path: /follow-me/jobs/{id}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	var fmJobID string
	for i, p := range parts {
		if p == "jobs" && i+1 < len(parts) {
			fmJobID = parts[i+1]
			break
		}
	}
	if fmJobID == "" {
		respondError(w, apperrors.New("invalid job path", http.StatusBadRequest))
		return
	}

	// Verify the job exists and is in waiting status
	var status string
	err := h.db.QueryRowContext(r.Context(),
		`SELECT status FROM follow_me_jobs WHERE id = $1`, fmJobID,
	).Scan(&status)
	if err == sql.ErrNoRows {
		respondError(w, apperrors.ErrNotFound)
		return
	}
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get follow-me job", http.StatusInternalServerError))
		return
	}

	if status != "waiting" {
		respondError(w, apperrors.New("can only cancel jobs in waiting status", http.StatusBadRequest))
		return
	}

	_, err = h.db.ExecContext(r.Context(),
		`UPDATE follow_me_jobs SET status = 'cancelled' WHERE id = $1`, fmJobID,
	)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to cancel follow-me job", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":      fmJobID,
		"status":  "cancelled",
		"message": "follow-me job cancelled",
	})
}

// --- Path helpers ---

// extractPoolID extracts the pool ID from URL paths like /follow-me/pools/{id}
func extractPoolID(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, p := range parts {
		if p == "pools" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// extractPoolAndPrinterID extracts pool and printer IDs from paths like /follow-me/pools/{pool_id}/printers/{printer_id}
func extractPoolAndPrinterID(path string) (string, string) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	var poolID, printerID string
	for i, p := range parts {
		if p == "pools" && i+1 < len(parts) {
			poolID = parts[i+1]
		}
		if p == "printers" && i+1 < len(parts) {
			printerID = parts[i+1]
		}
	}
	return poolID, printerID
}
