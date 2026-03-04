// Package handler provides HTTP handlers for print quota management.
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
)

// QuotaRepository defines the interface for quota repository operations.
type QuotaRepository interface {
	CheckQuota(ctx context.Context, entityID, entityType, quotaType string, increment int) (bool, int64, error)
	GetQuota(ctx context.Context, entityID, entityType, quotaType, period string) (*PrintQuota, error)
	ListQuotas(ctx context.Context, entityID, entityType string) ([]*PrintQuota, error)
	SetQuota(ctx context.Context, quota *PrintQuota) error
	ResetQuota(ctx context.Context, quotaID string) error
	DeleteQuota(ctx context.Context, quotaID string) error
	GetQuotaHistory(ctx context.Context, entityID, entityType string, limit, offset int) ([]*QuotaHistoryEntry, int, error)
}

// PrintQuota represents a print quota configuration.
type PrintQuota struct {
	ID         string
	EntityID   string // user_id or organization_id
	EntityType string // 'user' or 'organization'
	QuotaType  string // 'pages', 'jobs', 'color_pages', 'duplex_pages'
	Period     string // 'daily', 'weekly', 'monthly', 'quarterly', 'yearly'
	Limit      int    // 0 means unlimited
	Used       int
	ResetDate  *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// QuotaHistoryEntry represents a historical quota usage record.
type QuotaHistoryEntry struct {
	ID          string
	QuotaID     string
	EntityID    string
	EntityType  string
	QuotaType   string
	Action      string // 'granted', 'used', 'reset', 'adjusted'
	Amount      int
	Previous    int
	Remaining   int
	Description string
	CreatedAt   time.Time
}

// QuotaHandler handles quota management HTTP endpoints.
type QuotaHandler struct {
	db        *pgxpool.Pool
	quotaRepo QuotaRepository
}

// NewQuotaHandler creates a new quota handler instance.
func NewQuotaHandler(db *pgxpool.Pool) *QuotaHandler {
	return &QuotaHandler{
		db:        db,
		quotaRepo: NewQuotaRepository(db),
	}
}

// CheckQuotaRequest represents a request to check quota availability.
type CheckQuotaRequest struct {
	EntityID   string `json:"entity_id"`
	EntityType string `json:"entity_type"` // 'user' or 'organization'
	QuotaType  string `json:"quota_type"`  // 'pages', 'jobs', 'color_pages', 'duplex_pages'
	Amount     int    `json:"amount"`      // Amount to check (default 1)
}

// CheckQuotaResponse represents the response from a quota check.
type CheckQuotaResponse struct {
	Allowed   bool   `json:"allowed"`
	Remaining int    `json:"remaining"`
	Limit     int    `json:"limit"`
	Used      int    `json:"used"`
	ResetDate string `json:"reset_date,omitempty"`
	Message   string `json:"message,omitempty"`
}

// SetQuotaRequest represents a request to set a quota.
type SetQuotaRequest struct {
	EntityID   string `json:"entity_id"`
	EntityType string `json:"entity_type"`
	QuotaType  string `json:"quota_type"`
	Period     string `json:"period"`
	Limit      int    `json:"limit"`
}

// QuotaCheckHandler handles quota availability checks.
func (h *QuotaHandler) QuotaCheckHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CheckQuotaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate request
	if req.EntityID == "" {
		respondError(w, apperrors.New("entity_id is required", http.StatusBadRequest))
		return
	}
	if req.EntityType != "user" && req.EntityType != "organization" {
		respondError(w, apperrors.New("entity_type must be 'user' or 'organization'", http.StatusBadRequest))
		return
	}
	if req.QuotaType == "" {
		req.QuotaType = "pages"
	}
	if req.Amount <= 0 {
		req.Amount = 1
	}

	// Check quota using database function
	allowed, remaining, err := h.quotaRepo.CheckQuota(ctx, req.EntityID, req.EntityType, req.QuotaType, req.Amount)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to check quota", http.StatusInternalServerError))
		return
	}

	// Get current quota details for response
	quota, err := h.quotaRepo.GetQuota(ctx, req.EntityID, req.EntityType, req.QuotaType, "")
	if err != nil {
		// Log but don't fail - return basic response
		fmt.Printf("Failed to get quota details: %v", err)
	}

	resp := CheckQuotaResponse{
		Allowed:   allowed,
		Remaining: int(remaining),
	}

	if quota != nil {
		resp.Limit = quota.Limit
		resp.Used = quota.Used
		if quota.ResetDate != nil {
			resp.ResetDate = quota.ResetDate.Format(time.RFC3339)
		}
	}

	if !allowed {
		resp.Message = "Quota limit exceeded"
	} else if resp.Limit == 0 {
		resp.Message = "Unlimited quota"
	}

	respondJSON(w, http.StatusOK, resp)
}

// QuotaListHandler handles listing quotas for an entity.
func (h *QuotaHandler) QuotaListHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	entityID := r.URL.Query().Get("entity_id")
	entityType := r.URL.Query().Get("entity_type")

	if entityID == "" || entityType == "" {
		respondError(w, apperrors.New("entity_id and entity_type are required", http.StatusBadRequest))
		return
	}

	if entityType != "user" && entityType != "organization" {
		respondError(w, apperrors.New("entity_type must be 'user' or 'organization'", http.StatusBadRequest))
		return
	}

	quotas, err := h.quotaRepo.ListQuotas(ctx, entityID, entityType)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to list quotas", http.StatusInternalServerError))
		return
	}

	response := make([]map[string]interface{}, len(quotas))
	for i, q := range quotas {
		response[i] = quotaToResponse(q)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"quotas": response,
		"count":  len(response),
	})
}

// QuotaSetHandler handles setting/updating quotas.
func (h *QuotaHandler) QuotaSetHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SetQuotaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate request
	if req.EntityID == "" {
		respondError(w, apperrors.New("entity_id is required", http.StatusBadRequest))
		return
	}
	if req.EntityType != "user" && req.EntityType != "organization" {
		respondError(w, apperrors.New("entity_type must be 'user' or 'organization'", http.StatusBadRequest))
		return
	}
	if req.QuotaType == "" {
		respondError(w, apperrors.New("quota_type is required", http.StatusBadRequest))
		return
	}
	if req.Period == "" {
		req.Period = "monthly"
	}
	if req.Limit < 0 {
		respondError(w, apperrors.New("limit cannot be negative", http.StatusBadRequest))
		return
	}

	// Check if quota already exists
	existing, _ := h.quotaRepo.GetQuota(ctx, req.EntityID, req.EntityType, req.QuotaType, req.Period)

	var quota *PrintQuota
	if existing != nil {
		// Update existing quota
		existing.Limit = req.Limit
		existing.UpdatedAt = time.Now()
		quota = existing
	} else {
		// Create new quota
		quota = &PrintQuota{
			ID:         uuid.New().String(),
			EntityID:   req.EntityID,
			EntityType: req.EntityType,
			QuotaType:  req.QuotaType,
			Period:     req.Period,
			Limit:      req.Limit,
			Used:       0,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
	}

	if err := h.quotaRepo.SetQuota(ctx, quota); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to set quota", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, quotaToResponse(quota))
}

// QuotaResetHandler handles resetting quota usage.
func (h *QuotaHandler) QuotaResetHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract quota ID from path
	// Path format: /quotas/{id}/reset
	parts := parsePath(r.URL.Path)
	if len(parts) < 2 {
		respondError(w, apperrors.New("invalid quota path", http.StatusBadRequest))
		return
	}
	quotaID := parts[1]

	if err := h.quotaRepo.ResetQuota(ctx, quotaID); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to reset quota", http.StatusInternalServerError))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// QuotaDeleteHandler handles deleting a quota.
func (h *QuotaHandler) QuotaDeleteHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract quota ID from path
	parts := parsePath(r.URL.Path)
	if len(parts) < 2 {
		respondError(w, apperrors.New("invalid quota path", http.StatusBadRequest))
		return
	}
	quotaID := parts[1]

	if err := h.quotaRepo.DeleteQuota(ctx, quotaID); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to delete quota", http.StatusInternalServerError))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// QuotaHistoryHandler handles quota usage history requests.
func (h *QuotaHandler) QuotaHistoryHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	entityID := r.URL.Query().Get("entity_id")
	entityType := r.URL.Query().Get("entity_type")
	limit := 50
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsedOffset, err := strconv.Atoi(o); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	history, total, err := h.quotaRepo.GetQuotaHistory(ctx, entityID, entityType, limit, offset)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get quota history", http.StatusInternalServerError))
		return
	}

	response := make([]map[string]interface{}, len(history))
	for i, entry := range history {
		response[i] = map[string]interface{}{
			"id":          entry.ID,
			"quota_id":    entry.QuotaID,
			"entity_id":   entry.EntityID,
			"entity_type": entry.EntityType,
			"quota_type":  entry.QuotaType,
			"action":      entry.Action,
			"amount":      entry.Amount,
			"previous":    entry.Previous,
			"remaining":   entry.Remaining,
			"description": entry.Description,
			"created_at":  entry.CreatedAt.Format(time.RFC3339),
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"history": response,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// QuotaSummaryHandler provides a summary of quota usage across an organization.
func (h *QuotaHandler) QuotaSummaryHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgID := r.URL.Query().Get("organization_id")
	if orgID == "" {
		respondError(w, apperrors.New("organization_id is required", http.StatusBadRequest))
		return
	}

	// Get organization quotas
	orgQuotas, err := h.quotaRepo.ListQuotas(ctx, orgID, "organization")
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get organization quotas", http.StatusInternalServerError))
		return
	}

	// Get user quotas for this org
	query := `
		SELECT id, entity_id, entity_type, quota_type, period, "limit", used, reset_date, created_at, updated_at
		FROM print_quotas
		WHERE entity_type = 'user'
		AND entity_id IN (SELECT id FROM users WHERE organization_id = $1)
	`
	rows, err := h.db.Query(ctx, query, orgID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get user quotas", http.StatusInternalServerError))
		return
	}
	defer rows.Close()

	userQuotas := make([]*PrintQuota, 0)
	for rows.Next() {
		var q PrintQuota
		if err := rows.Scan(&q.ID, &q.EntityID, &q.EntityType, &q.QuotaType, &q.Period, &q.Limit, &q.Used, &q.ResetDate, &q.CreatedAt, &q.UpdatedAt); err != nil {
			continue
		}
		userQuotas = append(userQuotas, &q)
	}

	// Aggregate totals by quota type
	summary := make(map[string]map[string]interface{})
	for _, q := range orgQuotas {
		if _, exists := summary[q.QuotaType]; !exists {
			summary[q.QuotaType] = map[string]interface{}{
				"org_limit":   0,
				"org_used":    0,
				"user_limit":  0,
				"user_used":   0,
				"total_limit": 0,
				"total_used":  0,
			}
		}
		data := summary[q.QuotaType]
		data["org_limit"] = data["org_limit"].(int) + q.Limit
		data["org_used"] = data["org_used"].(int) + q.Used
		data["total_limit"] = data["total_limit"].(int) + q.Limit
		data["total_used"] = data["total_used"].(int) + q.Used
	}

	for _, q := range userQuotas {
		if _, exists := summary[q.QuotaType]; !exists {
			summary[q.QuotaType] = map[string]interface{}{
				"org_limit":   0,
				"org_used":    0,
				"user_limit":  0,
				"user_used":   0,
				"total_limit": 0,
				"total_used":  0,
			}
		}
		data := summary[q.QuotaType]
		data["user_limit"] = data["user_limit"].(int) + q.Limit
		data["user_used"] = data["user_used"].(int) + q.Used
		data["total_limit"] = data["total_limit"].(int) + q.Limit
		data["total_used"] = data["total_used"].(int) + q.Used
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"organization_id": orgID,
		"summary":         summary,
	})
}

// Helper functions

func quotaToResponse(q *PrintQuota) map[string]interface{} {
	resp := map[string]interface{}{
		"id":          q.ID,
		"entity_id":   q.EntityID,
		"entity_type": q.EntityType,
		"quota_type":  q.QuotaType,
		"period":      q.Period,
		"limit":       q.Limit,
		"used":        q.Used,
		"remaining":   q.Limit - q.Used,
		"created_at":  q.CreatedAt.Format(time.RFC3339),
		"updated_at":  q.UpdatedAt.Format(time.RFC3339),
	}
	if q.ResetDate != nil {
		resp["reset_date"] = q.ResetDate.Format(time.RFC3339)
	}
	if q.Limit == 0 {
		resp["unlimited"] = true
	}
	return resp
}
