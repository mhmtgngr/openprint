// Package handler provides HTTP handlers for the auth service endpoints.
package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/internal/shared/telemetry/prometheus"
)

// GuestHandler manages guest printing tokens and submissions.
type GuestHandler struct {
	db          *pgxpool.Pool
	metrics     *prometheus.Metrics
	serviceName string
}

// GuestHandlerConfig holds guest handler dependencies.
type GuestHandlerConfig struct {
	DB          *pgxpool.Pool
	Metrics     *prometheus.Metrics
	ServiceName string
}

// NewGuestHandler creates a new guest handler instance.
func NewGuestHandler(cfg GuestHandlerConfig) *GuestHandler {
	serviceName := cfg.ServiceName
	if serviceName == "" {
		serviceName = "auth-service"
	}
	return &GuestHandler{
		db:          cfg.DB,
		metrics:     cfg.Metrics,
		serviceName: serviceName,
	}
}

// GuestToken represents a guest print token in the database.
type GuestToken struct {
	ID             string     `json:"id"`
	Token          string     `json:"token"`
	Email          string     `json:"email,omitempty"`
	Name           string     `json:"name,omitempty"`
	OrganizationID string     `json:"organization_id"`
	CreatedBy      string     `json:"created_by"`
	PrinterIDs     []string   `json:"printer_ids"`
	MaxPages       int        `json:"max_pages"`
	MaxJobs        int        `json:"max_jobs"`
	PagesUsed      int        `json:"pages_used"`
	JobsUsed       int        `json:"jobs_used"`
	ColorAllowed   bool       `json:"color_allowed"`
	DuplexRequired bool       `json:"duplex_required"`
	ExpiresAt      time.Time  `json:"expires_at"`
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
	LastUsedAt     *time.Time `json:"last_used_at,omitempty"`
}

// CreateGuestTokenRequest represents a request to create a guest print token.
type CreateGuestTokenRequest struct {
	Email          string   `json:"email"`
	Name           string   `json:"name"`
	OrganizationID string   `json:"organization_id"`
	CreatedBy      string   `json:"created_by"`
	PrinterIDs     []string `json:"printer_ids"`
	MaxPages       int      `json:"max_pages"`
	MaxJobs        int      `json:"max_jobs"`
	ColorAllowed   bool     `json:"color_allowed"`
	DuplexRequired bool     `json:"duplex_required"`
	ExpiresInHours int      `json:"expires_in_hours"`
}

// ValidateGuestTokenRequest represents a request to validate a guest token.
type ValidateGuestTokenRequest struct {
	Token string `json:"token"`
}

// GuestSubmitJobRequest represents a guest print job submission.
type GuestSubmitJobRequest struct {
	Token        string `json:"token"`
	DocumentName string `json:"document_name"`
	PageCount    int    `json:"page_count"`
	PrinterID    string `json:"printer_id"`
}

// CreateGuestToken handles POST requests to create a new guest print token.
// This endpoint is restricted to admin users who can issue tokens to visitors.
func (gh *GuestHandler) CreateGuestToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	var req CreateGuestTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate required fields
	if req.OrganizationID == "" {
		respondError(w, apperrors.New("organization_id is required", http.StatusBadRequest))
		return
	}
	if req.CreatedBy == "" {
		respondError(w, apperrors.New("created_by is required", http.StatusBadRequest))
		return
	}

	// Set defaults
	if req.MaxPages <= 0 {
		req.MaxPages = 10
	}
	if req.MaxJobs <= 0 {
		req.MaxJobs = 5
	}
	if req.ExpiresInHours <= 0 {
		req.ExpiresInHours = 24
	}

	// Generate a secure random token
	token, err := generateSecureToken(32)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to generate token", http.StatusInternalServerError))
		return
	}

	expiresAt := time.Now().Add(time.Duration(req.ExpiresInHours) * time.Hour)

	// Insert into database
	var id string
	err = gh.db.QueryRow(ctx,
		`INSERT INTO guest_print_tokens
			(token, email, name, organization_id, created_by, printer_ids, max_pages, max_jobs,
			 color_allowed, duplex_required, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING id`,
		token, req.Email, req.Name, req.OrganizationID, req.CreatedBy,
		req.PrinterIDs, req.MaxPages, req.MaxJobs,
		req.ColorAllowed, req.DuplexRequired, expiresAt,
	).Scan(&id)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to create guest token", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":              id,
		"token":           token,
		"email":           req.Email,
		"name":            req.Name,
		"organization_id": req.OrganizationID,
		"max_pages":       req.MaxPages,
		"max_jobs":        req.MaxJobs,
		"color_allowed":   req.ColorAllowed,
		"duplex_required": req.DuplexRequired,
		"expires_at":      expiresAt.Format(time.RFC3339),
	})
}

// ListGuestTokens handles GET requests to list active guest tokens for an organization.
func (gh *GuestHandler) ListGuestTokens(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	orgID := r.URL.Query().Get("organization_id")
	if orgID == "" {
		respondError(w, apperrors.New("organization_id query parameter is required", http.StatusBadRequest))
		return
	}

	rows, err := gh.db.Query(ctx,
		`SELECT id, token, email, name, organization_id, created_by, printer_ids,
		        max_pages, max_jobs, pages_used, jobs_used, color_allowed, duplex_required,
		        expires_at, is_active, created_at, last_used_at
		 FROM guest_print_tokens
		 WHERE organization_id = $1 AND is_active = true AND expires_at > NOW()
		 ORDER BY created_at DESC`,
		orgID,
	)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to list guest tokens", http.StatusInternalServerError))
		return
	}
	defer rows.Close()

	tokens := make([]map[string]interface{}, 0)
	for rows.Next() {
		var t GuestToken
		if err := rows.Scan(
			&t.ID, &t.Token, &t.Email, &t.Name, &t.OrganizationID, &t.CreatedBy,
			&t.PrinterIDs, &t.MaxPages, &t.MaxJobs, &t.PagesUsed, &t.JobsUsed,
			&t.ColorAllowed, &t.DuplexRequired, &t.ExpiresAt, &t.IsActive,
			&t.CreatedAt, &t.LastUsedAt,
		); err != nil {
			respondError(w, apperrors.Wrap(err, "failed to scan token", http.StatusInternalServerError))
			return
		}
		tokens = append(tokens, guestTokenToResponse(&t))
	}
	if err := rows.Err(); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to iterate tokens", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"tokens": tokens,
		"count":  len(tokens),
	})
}

// ValidateGuestToken handles POST requests to validate a guest print token.
// This is used by the guest print portal to check if a token is valid before
// allowing a guest to submit a print job.
func (gh *GuestHandler) ValidateGuestToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	var req ValidateGuestTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	if req.Token == "" {
		respondError(w, apperrors.New("token is required", http.StatusBadRequest))
		return
	}

	token, err := gh.findActiveToken(ctx, req.Token)
	if err != nil {
		respondError(w, apperrors.New("invalid or expired token", http.StatusUnauthorized))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"valid":          true,
		"name":           token.Name,
		"email":          token.Email,
		"printer_ids":    token.PrinterIDs,
		"max_pages":      token.MaxPages,
		"max_jobs":       token.MaxJobs,
		"pages_used":     token.PagesUsed,
		"jobs_used":      token.JobsUsed,
		"pages_remaining": token.MaxPages - token.PagesUsed,
		"jobs_remaining":  token.MaxJobs - token.JobsUsed,
		"color_allowed":  token.ColorAllowed,
		"duplex_required": token.DuplexRequired,
		"expires_at":     token.ExpiresAt.Format(time.RFC3339),
	})
}

// RevokeGuestToken handles DELETE requests to revoke a guest print token.
func (gh *GuestHandler) RevokeGuestToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Extract token ID from path: /guest/tokens/{id}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		respondError(w, apperrors.New("invalid token path", http.StatusBadRequest))
		return
	}
	tokenID := parts[len(parts)-1]

	result, err := gh.db.Exec(ctx,
		`UPDATE guest_print_tokens SET is_active = false WHERE id = $1 AND is_active = true`,
		tokenID,
	)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to revoke token", http.StatusInternalServerError))
		return
	}

	if result.RowsAffected() == 0 {
		respondError(w, apperrors.New("token not found or already revoked", http.StatusNotFound))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"revoked": true,
		"id":      tokenID,
	})
}

// GuestSubmitJob handles POST requests for guests to submit a print job using their token.
// This validates the token, checks quotas, creates a guest_print_jobs record,
// and updates the token usage counters.
func (gh *GuestHandler) GuestSubmitJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	var req GuestSubmitJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate required fields
	if req.Token == "" {
		respondError(w, apperrors.New("token is required", http.StatusBadRequest))
		return
	}
	if req.DocumentName == "" {
		respondError(w, apperrors.New("document_name is required", http.StatusBadRequest))
		return
	}
	if req.PageCount <= 0 {
		respondError(w, apperrors.New("page_count must be greater than 0", http.StatusBadRequest))
		return
	}

	// Look up and validate the token
	token, err := gh.findActiveToken(ctx, req.Token)
	if err != nil {
		respondError(w, apperrors.New("invalid or expired token", http.StatusUnauthorized))
		return
	}

	// Check job quota
	if token.JobsUsed >= token.MaxJobs {
		respondError(w, apperrors.New("job quota exceeded for this token", http.StatusForbidden))
		return
	}

	// Check page quota
	if token.PagesUsed+req.PageCount > token.MaxPages {
		respondError(w, apperrors.New("page quota would be exceeded for this token", http.StatusForbidden))
		return
	}

	// If printer_ids are restricted, validate the requested printer
	if req.PrinterID != "" && len(token.PrinterIDs) > 0 {
		allowed := false
		for _, pid := range token.PrinterIDs {
			if pid == req.PrinterID {
				allowed = true
				break
			}
		}
		if !allowed {
			respondError(w, apperrors.New("printer not allowed for this token", http.StatusForbidden))
			return
		}
	}

	// Use a transaction to atomically create the job and update the token counters
	tx, err := gh.db.Begin(ctx)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to begin transaction", http.StatusInternalServerError))
		return
	}
	defer tx.Rollback(ctx)

	// Insert the guest print job
	var jobID string
	err = tx.QueryRow(ctx,
		`INSERT INTO guest_print_jobs (token_id, document_name, page_count, printer_id, status)
		 VALUES ($1, $2, $3, $4, 'pending')
		 RETURNING id`,
		token.ID, req.DocumentName, req.PageCount, nilIfEmpty(req.PrinterID),
	).Scan(&jobID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to create guest print job", http.StatusInternalServerError))
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

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"job_id":        jobID,
		"token_id":      token.ID,
		"document_name": req.DocumentName,
		"page_count":    req.PageCount,
		"printer_id":    req.PrinterID,
		"status":        "pending",
		"pages_remaining": token.MaxPages - token.PagesUsed - req.PageCount,
		"jobs_remaining":  token.MaxJobs - token.JobsUsed - 1,
	})
}

// findActiveToken looks up a guest token by its token string and validates
// that it is active and not expired.
func (gh *GuestHandler) findActiveToken(ctx context.Context, tokenStr string) (*GuestToken, error) {
	var t GuestToken
	err := gh.db.QueryRow(ctx,
		`SELECT id, token, email, name, organization_id, created_by, printer_ids,
		        max_pages, max_jobs, pages_used, jobs_used, color_allowed, duplex_required,
		        expires_at, is_active, created_at, last_used_at
		 FROM guest_print_tokens
		 WHERE token = $1 AND is_active = true AND expires_at > NOW()`,
		tokenStr,
	).Scan(
		&t.ID, &t.Token, &t.Email, &t.Name, &t.OrganizationID, &t.CreatedBy,
		&t.PrinterIDs, &t.MaxPages, &t.MaxJobs, &t.PagesUsed, &t.JobsUsed,
		&t.ColorAllowed, &t.DuplexRequired, &t.ExpiresAt, &t.IsActive,
		&t.CreatedAt, &t.LastUsedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.New("token not found or expired", http.StatusUnauthorized)
		}
		return nil, err
	}
	return &t, nil
}

// generateSecureToken creates a cryptographically secure random hex token.
func generateSecureToken(byteLength int) (string, error) {
	b := make([]byte, byteLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// nilIfEmpty returns nil for an empty string, or a pointer to the string value.
// This is used for optional UUID columns that accept NULL.
func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// guestTokenToResponse converts a GuestToken to a JSON-friendly response map.
func guestTokenToResponse(t *GuestToken) map[string]interface{} {
	resp := map[string]interface{}{
		"id":              t.ID,
		"token":           t.Token,
		"email":           t.Email,
		"name":            t.Name,
		"organization_id": t.OrganizationID,
		"created_by":      t.CreatedBy,
		"printer_ids":     t.PrinterIDs,
		"max_pages":       t.MaxPages,
		"max_jobs":        t.MaxJobs,
		"pages_used":      t.PagesUsed,
		"jobs_used":       t.JobsUsed,
		"color_allowed":   t.ColorAllowed,
		"duplex_required": t.DuplexRequired,
		"expires_at":      t.ExpiresAt.Format(time.RFC3339),
		"is_active":       t.IsActive,
		"created_at":      t.CreatedAt.Format(time.RFC3339),
	}
	if t.LastUsedAt != nil {
		resp["last_used_at"] = t.LastUsedAt.Format(time.RFC3339)
	}
	return resp
}
