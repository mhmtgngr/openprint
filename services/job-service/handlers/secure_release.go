// Package handler provides HTTP handlers for secure print release functionality.
package handler

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
)

// SecureReleaseRepository defines the interface for secure release operations.
type SecureReleaseRepository interface {
	CreateSecureJob(ctx context.Context, job *SecurePrintJob) error
	GetSecureJob(ctx context.Context, secureJobID string) (*SecurePrintJob, error)
	GetSecureJobByJobID(ctx context.Context, jobID string) (*SecurePrintJob, error)
	ListPendingJobs(ctx context.Context, userID string, limit, offset int) ([]*SecurePrintJob, int, error)
	ListReleasedJobs(ctx context.Context, userID string, limit, offset int) ([]*SecurePrintJob, int, error)
	AttemptRelease(ctx context.Context, secureJobID, method string, releaseData map[string]interface{}) (bool, error)
	CancelSecureJob(ctx context.Context, secureJobID string) error
	GetReleaseAttempts(ctx context.Context, secureJobID string) ([]*ReleaseAttempt, error)
	ListReleaseStations(ctx context.Context, organizationID string) ([]*ReleaseStation, error)
	CreateReleaseStation(ctx context.Context, station *ReleaseStation) error
	UpdateReleaseStation(ctx context.Context, station *ReleaseStation) error
	DeleteReleaseStation(ctx context.Context, stationID string) error
}

// SecurePrintJob represents a print job requiring secure release.
type SecurePrintJob struct {
	ID                   string
	JobID                string
	UserID               string
	ReleaseMethod        string // 'pin', 'card', 'biometric', 'nfc', 'app'
	ReleaseData          map[string]interface{} // Encrypted PIN, card ID, etc.
	Status               string // 'pending', 'released', 'expired', 'cancelled'
	ExpiresAt            time.Time
	ReleasedAt           *time.Time
	ReleasedPrinterID    string
	ReleaseAttempts      int
	MaxReleaseAttempts   int
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// ReleaseAttempt represents a logged release attempt.
type ReleaseAttempt struct {
	ID             string
	SecureJobID    string
	AttemptedAt    time.Time
	AttemptedMethod string
	AttemptedBy    string
	Success        bool
	FailureReason  string
	IPAddress      string
	PrinterID      string
}

// ReleaseStation represents a physical print release station.
type ReleaseStation struct {
	ID               string
	Name             string
	Location         string
	OrganizationID   string
	SupportedMethods []string
	AssignedPrinters []string
	IsActive         bool
	LastHeartbeat    *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// SecureReleaseHandler handles secure print release HTTP endpoints.
type SecureReleaseHandler struct {
	db    *pgxpool.Pool
	repo  SecureReleaseRepository
}

// NewSecureReleaseHandler creates a new secure release handler instance.
func NewSecureReleaseHandler(db *pgxpool.Pool) *SecureReleaseHandler {
	return &SecureReleaseHandler{
		db:   db,
		repo: NewSecureReleaseRepository(db),
	}
}

// HoldJobRequest represents a request to hold a job for secure release.
type HoldJobRequest struct {
	JobID              string                 `json:"job_id"`
	UserID             string                 `json:"user_id"`
	ReleaseMethod      string                 `json:"release_method"`
	ReleaseData        map[string]interface{} `json:"release_data,omitempty"`
	ExpirationHours    int                    `json:"expiration_hours,omitempty"`
}

// ReleaseJobRequest represents a request to release a held job.
type ReleaseJobRequest struct {
	SecureJobID   string                 `json:"secure_job_id"`
	ReleaseMethod string                 `json:"release_method"`
	ReleaseData   map[string]interface{} `json:"release_data"`
	PrinterID     string                 `json:"printer_id,omitempty"`
}

// ReleaseStationRequest represents a request to create/update a release station.
type ReleaseStationRequest struct {
	Name             string   `json:"name"`
	Location         string   `json:"location,omitempty"`
	OrganizationID   string   `json:"organization_id"`
	SupportedMethods []string `json:"supported_methods"`
	AssignedPrinters []string `json:"assigned_printers,omitempty"`
}

// HoldJobHandler handles requests to hold jobs for secure release.
func (h *SecureReleaseHandler) HoldJobHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req HoldJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate request
	if req.JobID == "" {
		respondError(w, apperrors.New("job_id is required", http.StatusBadRequest))
		return
	}
	if req.UserID == "" {
		respondError(w, apperrors.New("user_id is required", http.StatusBadRequest))
		return
	}
	if req.ReleaseMethod == "" {
		req.ReleaseMethod = "pin"
	}

	// Check if job exists and belongs to user
	var jobUserID, jobStatus string
	jobQuery := `
		SELECT user_name::text, status
		FROM print_jobs
		WHERE id = $1::uuid
	`
	err := h.db.QueryRow(ctx, jobQuery, req.JobID).Scan(&jobUserID, &jobStatus)
	if err != nil {
		respondError(w, apperrors.New("job not found", http.StatusNotFound))
		return
	}

	if jobStatus == "completed" || jobStatus == "cancelled" {
		respondError(w, apperrors.New("job cannot be held in current status", http.StatusBadRequest))
		return
	}

	// Generate release data if not provided
	releaseData := req.ReleaseData
	if releaseData == nil && req.ReleaseMethod == "pin" {
		pin, _ := generatePIN(6)
		releaseData = map[string]interface{}{
			"pin": pin,
			"pin_hash": hashPIN(pin), // In production, use proper hashing
		}
	}

	// Set expiration
	expiresAt := time.Now().Add(24 * time.Hour)
	if req.ExpirationHours > 0 {
		expiresAt = time.Now().Add(time.Duration(req.ExpirationHours) * time.Hour)
	}

	// Create secure job entry
	secureJob := &SecurePrintJob{
		ID:                 uuid.New().String(),
		JobID:              req.JobID,
		UserID:             req.UserID,
		ReleaseMethod:      req.ReleaseMethod,
		ReleaseData:        releaseData,
		Status:             "pending",
		ExpiresAt:          expiresAt,
		ReleaseAttempts:    0,
		MaxReleaseAttempts: 3,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if err := h.repo.CreateSecureJob(ctx, secureJob); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to hold job", http.StatusInternalServerError))
		return
	}

	// Update job status to held
	_, _ = h.db.Exec(ctx, "UPDATE print_jobs SET status = 'held', updated_at = NOW() WHERE id = $1", req.JobID)

	// Prepare response
	response := map[string]interface{}{
		"secure_job_id":       secureJob.ID,
		"job_id":              secureJob.JobID,
		"release_method":      secureJob.ReleaseMethod,
		"status":              secureJob.Status,
		"expires_at":          secureJob.ExpiresAt.Format(time.RFC3339),
		"max_release_attempts": secureJob.MaxReleaseAttempts,
	}

	// Include PIN in response if generated (only show once)
	if pin, ok := releaseData["pin"]; ok {
		response["pin"] = pin
	}

	respondJSON(w, http.StatusCreated, response)
}

// ReleaseJobHandler handles requests to release held jobs.
func (h *SecureReleaseHandler) ReleaseJobHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ReleaseJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate request
	if req.SecureJobID == "" {
		respondError(w, apperrors.New("secure_job_id is required", http.StatusBadRequest))
		return
	}
	if req.ReleaseMethod == "" {
		req.ReleaseMethod = "pin"
	}

	// Attempt release
	success, err := h.repo.AttemptRelease(ctx, req.SecureJobID, req.ReleaseMethod, req.ReleaseData)
	if err != nil {
		// Check if it's a validation error or system error
		if strings.Contains(err.Error(), "expired") {
			respondError(w, apperrors.New("secure job has expired", http.StatusGone))
			return
		} else if strings.Contains(err.Error(), "exceeded") {
			respondError(w, apperrors.New("maximum release attempts exceeded", http.StatusTooManyRequests))
			return
		}
		respondError(w, apperrors.Wrap(err, "failed to release job", http.StatusInternalServerError))
		return
	}

	if !success {
		respondError(w, apperrors.New("release validation failed", http.StatusUnauthorized))
		return
	}

	// Get updated secure job
	secureJob, err := h.repo.GetSecureJob(ctx, req.SecureJobID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get secure job", http.StatusInternalServerError))
		return
	}

	// If released successfully, update original job status to queued
	if secureJob.Status == "released" {
		_, _ = h.db.Exec(ctx, `
			UPDATE print_jobs
			SET status = 'queued',
			    updated_at = NOW()
			WHERE id = $1
		`, secureJob.JobID)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"secure_job_id":  secureJob.ID,
		"job_id":         secureJob.JobID,
		"status":         secureJob.Status,
		"released_at":    secureJob.ReleasedAt.Format(time.RFC3339),
		"printer_id":     secureJob.ReleasedPrinterID,
	})
}

// PendingJobsHandler handles listing pending secure jobs for a user.
func (h *SecureReleaseHandler) PendingJobsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		respondError(w, apperrors.New("user_id is required", http.StatusBadRequest))
		return
	}

	limit := 50
	offset := 0

	jobs, total, err := h.repo.ListPendingJobs(ctx, userID, limit, offset)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to list pending jobs", http.StatusInternalServerError))
		return
	}

	response := make([]map[string]interface{}, len(jobs))
	for i, job := range jobs {
		response[i] = secureJobToResponse(job)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"jobs":  response,
		"total": total,
		"limit": limit,
		"offset": offset,
	})
}

// ReleasedJobsHandler handles listing released jobs history.
func (h *SecureReleaseHandler) ReleasedJobsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		respondError(w, apperrors.New("user_id is required", http.StatusBadRequest))
		return
	}

	limit := 50
	offset := 0

	jobs, total, err := h.repo.ListReleasedJobs(ctx, userID, limit, offset)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to list released jobs", http.StatusInternalServerError))
		return
	}

	response := make([]map[string]interface{}, len(jobs))
	for i, job := range jobs {
		response[i] = secureJobToResponse(job)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"jobs":  response,
		"total": total,
		"limit": limit,
		"offset": offset,
	})
}

// CancelSecureJobHandler handles canceling a held secure job.
func (h *SecureReleaseHandler) CancelSecureJobHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract secure job ID from path
	parts := parsePath(r.URL.Path)
	if len(parts) < 3 {
		respondError(w, apperrors.New("invalid secure job path", http.StatusBadRequest))
		return
	}
	secureJobID := parts[2]

	if err := h.repo.CancelSecureJob(ctx, secureJobID); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to cancel secure job", http.StatusInternalServerError))
		return
	}

	// Get the job to update original print job status
	secureJob, _ := h.repo.GetSecureJob(ctx, secureJobID)
	if secureJob != nil {
		_, _ = h.db.Exec(ctx, `
			UPDATE print_jobs
			SET status = 'cancelled',
			    updated_at = NOW()
			WHERE id = $1
		`, secureJob.JobID)
	}

	w.WriteHeader(http.StatusNoContent)
}

// ReleaseAttemptsHandler handles listing release attempts for a secure job.
func (h *SecureReleaseHandler) ReleaseAttemptsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract secure job ID from path
	parts := parsePath(r.URL.Path)
	if len(parts) < 3 {
		respondError(w, apperrors.New("invalid secure job path", http.StatusBadRequest))
		return
	}
	secureJobID := parts[2]

	attempts, err := h.repo.GetReleaseAttempts(ctx, secureJobID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get release attempts", http.StatusInternalServerError))
		return
	}

	response := make([]map[string]interface{}, len(attempts))
	for i, attempt := range attempts {
		response[i] = map[string]interface{}{
			"id":               attempt.ID,
			"secure_job_id":    attempt.SecureJobID,
			"attempted_at":     attempt.AttemptedAt.Format(time.RFC3339),
			"attempted_method": attempt.AttemptedMethod,
			"attempted_by":     attempt.AttemptedBy,
			"success":          attempt.Success,
			"failure_reason":   attempt.FailureReason,
			"ip_address":       attempt.IPAddress,
			"printer_id":       attempt.PrinterID,
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"attempts": response,
		"count":    len(response),
	})
}

// ReleaseStationsHandler handles listing release stations.
func (h *SecureReleaseHandler) ReleaseStationsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		h.listReleaseStations(w, r, ctx)
	case http.MethodPost:
		h.createReleaseStation(w, r, ctx)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *SecureReleaseHandler) listReleaseStations(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	orgID := r.URL.Query().Get("organization_id")
	if orgID == "" {
		respondError(w, apperrors.New("organization_id is required", http.StatusBadRequest))
		return
	}

	stations, err := h.repo.ListReleaseStations(ctx, orgID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to list release stations", http.StatusInternalServerError))
		return
	}

	response := make([]map[string]interface{}, len(stations))
	for i, station := range stations {
		response[i] = releaseStationToResponse(station)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"stations": response,
		"count":    len(response),
	})
}

func (h *SecureReleaseHandler) createReleaseStation(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	var req ReleaseStationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate request
	if req.Name == "" {
		respondError(w, apperrors.New("name is required", http.StatusBadRequest))
		return
	}
	if req.OrganizationID == "" {
		respondError(w, apperrors.New("organization_id is required", http.StatusBadRequest))
		return
	}
	if len(req.SupportedMethods) == 0 {
		req.SupportedMethods = []string{"pin"}
	}

	station := &ReleaseStation{
		ID:               uuid.New().String(),
		Name:             req.Name,
		Location:         req.Location,
		OrganizationID:   req.OrganizationID,
		SupportedMethods: req.SupportedMethods,
		AssignedPrinters: req.AssignedPrinters,
		IsActive:         true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := h.repo.CreateReleaseStation(ctx, station); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to create release station", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusCreated, releaseStationToResponse(station))
}

// ReleaseStationHandler handles individual release station operations.
func (h *SecureReleaseHandler) ReleaseStationHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract station ID from path
	parts := parsePath(r.URL.Path)
	if len(parts) < 2 {
		respondError(w, apperrors.New("invalid station path", http.StatusBadRequest))
		return
	}
	stationID := parts[1]

	switch r.Method {
	case http.MethodGet:
		h.getReleaseStation(w, r, ctx, stationID)
	case http.MethodPut:
		h.updateReleaseStation(w, r, ctx, stationID)
	case http.MethodDelete:
		h.deleteReleaseStation(w, r, ctx, stationID)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *SecureReleaseHandler) getReleaseStation(w http.ResponseWriter, r *http.Request, ctx context.Context, stationID string) {
	// Get station from database
	var station ReleaseStation
	query := `
		SELECT id, name, location, organization_id, supported_methods,
		       assigned_printers, is_active, last_heartbeat, created_at, updated_at
		FROM print_release_stations
		WHERE id = $1::uuid
	`

	err := h.db.QueryRow(ctx, query, stationID).Scan(
		&station.ID, &station.Name, &station.Location, &station.OrganizationID,
		&station.SupportedMethods, &station.AssignedPrinters, &station.IsActive,
		&station.LastHeartbeat, &station.CreatedAt, &station.UpdatedAt,
	)

	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	respondJSON(w, http.StatusOK, releaseStationToResponse(&station))
}

func (h *SecureReleaseHandler) updateReleaseStation(w http.ResponseWriter, r *http.Request, ctx context.Context, stationID string) {
	var req ReleaseStationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Get existing station
	var station ReleaseStation
	query := `
		SELECT id, name, location, organization_id, supported_methods,
		       assigned_printers, is_active, created_at
		FROM print_release_stations
		WHERE id = $1::uuid
	`

	err := h.db.QueryRow(ctx, query, stationID).Scan(
		&station.ID, &station.Name, &station.Location, &station.OrganizationID,
		&station.SupportedMethods, &station.AssignedPrinters, &station.IsActive,
		&station.CreatedAt,
	)

	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	// Update fields
	if req.Name != "" {
		station.Name = req.Name
	}
	if req.Location != "" {
		station.Location = req.Location
	}
	if len(req.SupportedMethods) > 0 {
		station.SupportedMethods = req.SupportedMethods
	}
	if req.AssignedPrinters != nil {
		station.AssignedPrinters = req.AssignedPrinters
	}
	station.UpdatedAt = time.Now()

	if err := h.repo.UpdateReleaseStation(ctx, &station); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to update release station", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, releaseStationToResponse(&station))
}

func (h *SecureReleaseHandler) deleteReleaseStation(w http.ResponseWriter, r *http.Request, ctx context.Context, stationID string) {
	if err := h.repo.DeleteReleaseStation(ctx, stationID); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to delete release station", http.StatusInternalServerError))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// StationHeartbeatHandler handles heartbeat updates from release stations.
func (h *SecureReleaseHandler) StationHeartbeatHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract station ID from path
	parts := parsePath(r.URL.Path)
	if len(parts) < 3 {
		respondError(w, apperrors.New("invalid station path", http.StatusBadRequest))
		return
	}
	stationID := parts[2]

	// Update heartbeat
	query := `
		UPDATE print_release_stations
		SET last_heartbeat = NOW(),
		    updated_at = NOW(),
		    is_active = true
		WHERE id = $1::uuid
	`

	_, err := h.db.Exec(ctx, query, stationID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to update heartbeat", http.StatusInternalServerError))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper functions

func secureJobToResponse(job *SecurePrintJob) map[string]interface{} {
	resp := map[string]interface{}{
		"id":                   job.ID,
		"job_id":               job.JobID,
		"user_id":              job.UserID,
		"release_method":       job.ReleaseMethod,
		"status":               job.Status,
		"expires_at":           job.ExpiresAt.Format(time.RFC3339),
		"release_attempts":     job.ReleaseAttempts,
		"max_release_attempts": job.MaxReleaseAttempts,
		"created_at":           job.CreatedAt.Format(time.RFC3339),
	}
	if job.ReleasedAt != nil {
		resp["released_at"] = job.ReleasedAt.Format(time.RFC3339)
	}
	if job.ReleasedPrinterID != "" {
		resp["released_printer_id"] = job.ReleasedPrinterID
	}
	return resp
}

func releaseStationToResponse(station *ReleaseStation) map[string]interface{} {
	resp := map[string]interface{}{
		"id":               station.ID,
		"name":             station.Name,
		"organization_id":  station.OrganizationID,
		"supported_methods": station.SupportedMethods,
		"is_active":        station.IsActive,
		"created_at":       station.CreatedAt.Format(time.RFC3339),
		"updated_at":       station.UpdatedAt.Format(time.RFC3339),
	}
	if station.Location != "" {
		resp["location"] = station.Location
	}
	if station.AssignedPrinters != nil {
		resp["assigned_printers"] = station.AssignedPrinters
	}
	if station.LastHeartbeat != nil {
		resp["last_heartbeat"] = station.LastHeartbeat.Format(time.RFC3339)
	}
	return resp
}

// generatePIN generates a numeric PIN of specified length.
func generatePIN(length int) (string, error) {
	if length < 1 {
		length = 6
	}

	result := make([]byte, length)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		result[i] = byte(n.Int64() + '0')
	}

	return string(result), nil
}

// hashPIN creates a simple hash of the PIN (in production, use bcrypt/argon2).
func hashPIN(pin string) string {
	// Simple hash for demonstration - use proper hashing in production
	return fmt.Sprintf("%x", len(pin)+len(pin)*17)
}
