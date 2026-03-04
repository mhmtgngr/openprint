// Package handlers provides HTTP handlers for trusted client management.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/internal/shared/ratelimit"
)

// TrustedClientsHandler handles trusted client management endpoints.
type TrustedClientsHandler struct {
	rateLimitHandler *RateLimitHandler
}

// NewTrustedClientsHandler creates a new trusted clients handler.
func NewTrustedClientsHandler(rlHandler *RateLimitHandler) *TrustedClientsHandler {
	return &TrustedClientsHandler{
		rateLimitHandler: rlHandler,
	}
}

// CreateTrustedClientRequest represents a request to create a trusted client.
type CreateTrustedClientRequest struct {
	Name        string   `json:"name"`
	APIKey      string   `json:"api_key,omitempty"`
	IPWhitelist []string `json:"ip_whitelist,omitempty"`
	Description string   `json:"description,omitempty"`
}

// CreateTrustedClient handles trusted client creation requests.
func (h *TrustedClientsHandler) CreateTrustedClient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateTrustedClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate request
	if req.Name == "" {
		respondError(w, apperrors.New("name is required", http.StatusBadRequest))
		return
	}

	// Generate API key if not provided
	if req.APIKey == "" {
		req.APIKey = generateAPIKey()
	}

	// Create trusted client
	client := &ratelimit.TrustedClient{
		ID:          uuid.New().String(),
		Name:        req.Name,
		APIKey:      req.APIKey,
		IPWhitelist: req.IPWhitelist,
		Description: req.Description,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	repo := h.rateLimitHandler.GetRepository()
	if repo == nil {
		respondError(w, apperrors.New("rate limiter not initialized", http.StatusServiceUnavailable))
		return
	}

	if err := repo.CreateTrustedClient(ctx, client); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to create trusted client", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusCreated, trustedClientToResponse(client))
}

// ListTrustedClients handles listing trusted clients.
func (h *TrustedClientsHandler) ListTrustedClients(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	repo := h.rateLimitHandler.GetRepository()
	if repo == nil {
		respondError(w, apperrors.New("rate limiter not initialized", http.StatusServiceUnavailable))
		return
	}

	clients, err := repo.ListTrustedClients(ctx)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to list trusted clients", http.StatusInternalServerError))
		return
	}

	response := make([]map[string]interface{}, len(clients))
	for i, c := range clients {
		response[i] = trustedClientToResponse(c)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"clients": response,
		"count":   len(response),
	})
}

// GetTrustedClient handles retrieving a single trusted client.
func (h *TrustedClientsHandler) GetTrustedClient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract client ID from path
	parts := parsePath(r.URL.Path)
	if len(parts) < 4 {
		respondError(w, apperrors.New("invalid trusted client path", http.StatusBadRequest))
		return
	}
	clientID := parts[3]

	repo := h.rateLimitHandler.GetRepository()
	if repo == nil {
		respondError(w, apperrors.New("rate limiter not initialized", http.StatusServiceUnavailable))
		return
	}

	client, err := repo.GetTrustedClient(ctx, clientID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respondError(w, apperrors.ErrNotFound)
			return
		}
		respondError(w, apperrors.Wrap(err, "failed to get trusted client", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, trustedClientToResponse(client))
}

// UpdateTrustedClientRequest represents a request to update a trusted client.
type UpdateTrustedClientRequest struct {
	Name        string   `json:"name"`
	IPWhitelist []string `json:"ip_whitelist"`
	Description string   `json:"description"`
	IsActive     *bool    `json:"is_active"`
}

// UpdateTrustedClient handles trusted client update requests.
func (h *TrustedClientsHandler) UpdateTrustedClient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract client ID from path
	parts := parsePath(r.URL.Path)
	if len(parts) < 4 {
		respondError(w, apperrors.New("invalid trusted client path", http.StatusBadRequest))
		return
	}
	clientID := parts[3]

	repo := h.rateLimitHandler.GetRepository()
	if repo == nil {
		respondError(w, apperrors.New("rate limper not initialized", http.StatusServiceUnavailable))
		return
	}

	// Get existing client
	client, err := repo.GetTrustedClient(ctx, clientID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respondError(w, apperrors.ErrNotFound)
			return
		}
		respondError(w, apperrors.Wrap(err, "failed to get trusted client", http.StatusInternalServerError))
		return
	}

	// Parse request body
	var req UpdateTrustedClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Update fields
	if req.Name != "" {
		client.Name = req.Name
	}
	if req.IPWhitelist != nil {
		client.IPWhitelist = req.IPWhitelist
	}
	if req.Description != "" {
		client.Description = req.Description
	}
	if req.IsActive != nil {
		client.IsActive = *req.IsActive
	}
	client.UpdatedAt = time.Now()

	if err := repo.UpdateTrustedClient(ctx, client); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to update trusted client", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, trustedClientToResponse(client))
}

// DeleteTrustedClient handles trusted client deletion requests.
func (h *TrustedClientsHandler) DeleteTrustedClient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract client ID from path
	parts := parsePath(r.URL.Path)
	if len(parts) < 4 {
		respondError(w, apperrors.New("invalid trusted client path", http.StatusBadRequest))
		return
	}
	clientID := parts[3]

	repo := h.rateLimitHandler.GetRepository()
	if repo == nil {
		respondError(w, apperrors.New("rate limiter not initialized", http.StatusServiceUnavailable))
		return
	}

	if err := repo.DeleteTrustedClient(ctx, clientID); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to delete trusted client", http.StatusInternalServerError))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RegenerateAPIKey regenerates the API key for a trusted client.
func (h *TrustedClientsHandler) RegenerateAPIKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract client ID from path
	parts := parsePath(r.URL.Path)
	if len(parts) < 5 {
		respondError(w, apperrors.New("invalid trusted client path", http.StatusBadRequest))
		return
	}
	clientID := parts[3]

	repo := h.rateLimitHandler.GetRepository()
	if repo == nil {
		respondError(w, apperrors.New("rate limiter not initialized", http.StatusServiceUnavailable))
		return
	}

	// Get existing client
	client, err := repo.GetTrustedClient(ctx, clientID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respondError(w, apperrors.ErrNotFound)
			return
		}
		respondError(w, apperrors.Wrap(err, "failed to get trusted client", http.StatusInternalServerError))
		return
	}

	// Generate new API key
	client.APIKey = generateAPIKey()
	client.UpdatedAt = time.Now()

	if err := repo.UpdateTrustedClient(ctx, client); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to update trusted client", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"api_key": client.APIKey,
		"message": "API key regenerated. Save it now as you won't be able to see it again.",
	})
}

// Helper functions

func trustedClientToResponse(c *ratelimit.TrustedClient) map[string]interface{} {
	return map[string]interface{}{
		"id":          c.ID,
		"name":        c.Name,
		"api_key":     maskAPIKey(c.APIKey),
		"ip_whitelist": c.IPWhitelist,
		"description": c.Description,
		"is_active":   c.IsActive,
		"created_at":  c.CreatedAt.Format(time.RFC3339),
		"updated_at":  c.UpdatedAt.Format(time.RFC3339),
		"last_used_at": func() string {
			if c.LastUsedAt != nil {
				return c.LastUsedAt.Format(time.RFC3339)
			}
			return ""
		}(),
	}
}

func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "********"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

func generateAPIKey() string {
	return uuid.New().String()
}
