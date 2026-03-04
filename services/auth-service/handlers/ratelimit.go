// Package handlers provides HTTP handlers for rate limit management.
package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/internal/shared/ratelimit"
)

// RateLimitHandler handles rate limit policy management endpoints.
type RateLimitHandler struct {
	db        *pgxpool.Pool
	limiter   *ratelimit.Limiter
	redisAddr string
}

// NewRateLimitHandler creates a new rate limit handler.
func NewRateLimitHandler(db *pgxpool.Pool, redisAddr string) *RateLimitHandler {
	return &RateLimitHandler{
		db:        db,
		redisAddr: redisAddr,
	}
}

// Initialize initializes the rate limiter.
func (h *RateLimitHandler) Initialize() error {
	if h.redisAddr == "" {
		return nil
	}

	cfg := &ratelimit.Config{
		RedisAddr:     h.redisAddr,
		RedisPassword: "",
		RedisDB:       0,
		DefaultPolicy: ratelimit.DefaultGlobalPolicy(),
		EnableMetrics: true,
		EnableAlerts:  true,
	}

	limiter, err := ratelimit.NewLimiter(cfg)
	if err != nil {
		return fmt.Errorf("failed to create rate limiter: %w", err)
	}

	// Set database connection
	repo := limiter.GetRepository()
	repo.SetDB(h.db)

	h.limiter = limiter

	return nil
}

// GetLimiter returns the configured rate limiter.
func (h *RateLimitHandler) GetLimiter() *ratelimit.Limiter {
	return h.limiter
}

// GetRepository returns the rate limit repository.
func (h *RateLimitHandler) GetRepository() *ratelimit.Repository {
	if h.limiter != nil {
		return h.limiter.GetRepository()
	}
	return nil
}

// CreatePolicyRequest represents a request to create a rate limit policy.
type CreatePolicyRequest struct {
	Name                    string   `json:"name"`
	Description             string   `json:"description"`
	Priority                int      `json:"priority"`
	Scope                   string   `json:"scope"`
	Identifier              string   `json:"identifier"`
	Methods                 []string `json:"methods"`
	PathPattern             string   `json:"path_pattern"`
	Limit                   int64    `json:"limit"`
	Window                  int      `json:"window"` // seconds
	BurstLimit              int64    `json:"burst_limit"`
	BurstDuration           int      `json:"burst_duration"` // seconds
	EnableQueue             bool     `json:"enable_queue"`
	MaxQueueSize            int      `json:"max_queue_size"`
	CircuitBreakerThreshold int      `json:"circuit_breaker_threshold"`
	CircuitBreakerTimeout   int      `json:"circuit_breaker_timeout"` // seconds
	Severity                string   `json:"severity"`
	Action                  string   `json:"action"`
	ThrottleRate            float64  `json:"throttle_rate"`
	IsActive                bool     `json:"is_active"`
}

// CreatePolicy handles policy creation requests.
func (h *RateLimitHandler) CreatePolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate request
	if req.Name == "" {
		respondError(w, apperrors.New("name is required", http.StatusBadRequest))
		return
	}
	if req.Limit <= 0 {
		respondError(w, apperrors.New("limit must be positive", http.StatusBadRequest))
		return
	}
	if req.Window <= 0 {
		respondError(w, apperrors.New("window must be positive", http.StatusBadRequest))
		return
	}

	// Create policy
	policy := &ratelimit.Policy{
		ID:                      uuid.New().String(),
		Name:                    req.Name,
		Description:             req.Description,
		Priority:                req.Priority,
		Scope:                   req.Scope,
		Identifier:              req.Identifier,
		Methods:                 req.Methods,
		PathPattern:             req.PathPattern,
		Limit:                   req.Limit,
		Window:                  time.Duration(req.Window) * time.Second,
		BurstLimit:              req.BurstLimit,
		BurstDuration:           time.Duration(req.BurstDuration) * time.Second,
		EnableQueue:             req.EnableQueue,
		MaxQueueSize:            req.MaxQueueSize,
		CircuitBreakerThreshold: req.CircuitBreakerThreshold,
		CircuitBreakerTimeout:   time.Duration(req.CircuitBreakerTimeout) * time.Second,
		Severity:                req.Severity,
		Action:                  req.Action,
		ThrottleRate:            req.ThrottleRate,
		IsActive:                req.IsActive,
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	}

	// Validate policy
	if err := ratelimit.ValidatePolicy(policy); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid policy", http.StatusBadRequest))
		return
	}

	repo := h.GetRepository()
	if repo == nil {
		respondError(w, apperrors.New("rate limiter not initialized", http.StatusServiceUnavailable))
		return
	}

	if err := repo.CreatePolicy(ctx, policy); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to create policy", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusCreated, policyToResponse(policy))
}

// ListPolicies handles policy listing requests.
func (h *RateLimitHandler) ListPolicies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	filter := &ratelimit.PolicyFilter{}

	if scope := r.URL.Query().Get("scope"); scope != "" {
		filter.Scope = scope
	}
	if isActive := r.URL.Query().Get("is_active"); isActive != "" {
		if b, err := strconv.ParseBool(isActive); err == nil {
			filter.IsActive = &b
		}
	}
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 {
			filter.Limit = l
		}
	}

	repo := h.GetRepository()
	if repo == nil {
		respondError(w, apperrors.New("rate limiter not initialized", http.StatusServiceUnavailable))
		return
	}

	policies, err := repo.ListPolicies(ctx, filter)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to list policies", http.StatusInternalServerError))
		return
	}

	response := make([]map[string]interface{}, len(policies))
	for i, p := range policies {
		response[i] = policyToResponse(p)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"policies": response,
		"count":    len(response),
	})
}

// GetPolicy handles retrieving a single policy.
func (h *RateLimitHandler) GetPolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract policy ID from path
	parts := parsePath(r.URL.Path)
	if len(parts) < 4 {
		respondError(w, apperrors.New("invalid policy path", http.StatusBadRequest))
		return
	}
	policyID := parts[3]

	repo := h.GetRepository()
	if repo == nil {
		respondError(w, apperrors.New("rate limiter not initialized", http.StatusServiceUnavailable))
		return
	}

	policy, err := repo.GetPolicy(ctx, policyID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respondError(w, apperrors.ErrNotFound)
			return
		}
		respondError(w, apperrors.Wrap(err, "failed to get policy", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, policyToResponse(policy))
}

// UpdatePolicyRequest represents a request to update a rate limit policy.
type UpdatePolicyRequest struct {
	Name                    string   `json:"name"`
	Description             string   `json:"description"`
	Priority                int      `json:"priority"`
	Scope                   string   `json:"scope"`
	Identifier              string   `json:"identifier"`
	Methods                 []string `json:"methods"`
	PathPattern             string   `json:"path_pattern"`
	Limit                   int64    `json:"limit"`
	Window                  int      `json:"window"`
	BurstLimit              int64    `json:"burst_limit"`
	BurstDuration           int      `json:"burst_duration"`
	EnableQueue             bool     `json:"enable_queue"`
	MaxQueueSize            int      `json:"max_queue_size"`
	CircuitBreakerThreshold int      `json:"circuit_breaker_threshold"`
	CircuitBreakerTimeout   int      `json:"circuit_breaker_timeout"`
	Severity                string   `json:"severity"`
	Action                  string   `json:"action"`
	ThrottleRate            float64  `json:"throttle_rate"`
	IsActive                *bool    `json:"is_active"`
}

// UpdatePolicy handles policy update requests.
func (h *RateLimitHandler) UpdatePolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract policy ID from path
	parts := parsePath(r.URL.Path)
	if len(parts) < 4 {
		respondError(w, apperrors.New("invalid policy path", http.StatusBadRequest))
		return
	}
	policyID := parts[3]

	repo := h.GetRepository()
	if repo == nil {
		respondError(w, apperrors.New("rate limiter not initialized", http.StatusServiceUnavailable))
		return
	}

	// Get existing policy
	policy, err := repo.GetPolicy(ctx, policyID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respondError(w, apperrors.ErrNotFound)
			return
		}
		respondError(w, apperrors.Wrap(err, "failed to get policy", http.StatusInternalServerError))
		return
	}

	// Parse request body
	var req UpdatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Update fields
	if req.Name != "" {
		policy.Name = req.Name
	}
	if req.Description != "" {
		policy.Description = req.Description
	}
	if req.Priority > 0 {
		policy.Priority = req.Priority
	}
	if req.Scope != "" {
		policy.Scope = req.Scope
	}
	if req.Identifier != "" {
		policy.Identifier = req.Identifier
	}
	if req.Methods != nil {
		policy.Methods = req.Methods
	}
	if req.PathPattern != "" {
		policy.PathPattern = req.PathPattern
	}
	if req.Limit > 0 {
		policy.Limit = req.Limit
	}
	if req.Window > 0 {
		policy.Window = time.Duration(req.Window) * time.Second
	}
	if req.BurstLimit >= 0 {
		policy.BurstLimit = req.BurstLimit
	}
	if req.BurstDuration > 0 {
		policy.BurstDuration = time.Duration(req.BurstDuration) * time.Second
	}
	policy.EnableQueue = req.EnableQueue
	if req.MaxQueueSize > 0 {
		policy.MaxQueueSize = req.MaxQueueSize
	}
	if req.CircuitBreakerThreshold >= 0 {
		policy.CircuitBreakerThreshold = req.CircuitBreakerThreshold
	}
	if req.CircuitBreakerTimeout > 0 {
		policy.CircuitBreakerTimeout = time.Duration(req.CircuitBreakerTimeout) * time.Second
	}
	if req.Severity != "" {
		policy.Severity = req.Severity
	}
	if req.Action != "" {
		policy.Action = req.Action
	}
	if req.ThrottleRate >= 0 {
		policy.ThrottleRate = req.ThrottleRate
	}
	if req.IsActive != nil {
		policy.IsActive = *req.IsActive
	}
	policy.UpdatedAt = time.Now()

	// Validate policy
	if err := ratelimit.ValidatePolicy(policy); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid policy", http.StatusBadRequest))
		return
	}

	if err := repo.UpdatePolicy(ctx, policy); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to update policy", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, policyToResponse(policy))
}

// DeletePolicy handles policy deletion requests.
func (h *RateLimitHandler) DeletePolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract policy ID from path
	parts := parsePath(r.URL.Path)
	if len(parts) < 4 {
		respondError(w, apperrors.New("invalid policy path", http.StatusBadRequest))
		return
	}
	policyID := parts[3]

	repo := h.GetRepository()
	if repo == nil {
		respondError(w, apperrors.New("rate limiter not initialized", http.StatusServiceUnavailable))
		return
	}

	if err := repo.DeletePolicy(ctx, policyID); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to delete policy", http.StatusInternalServerError))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CheckRateLimit handles rate limit check requests.
func (h *RateLimitHandler) CheckRateLimit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Identifier string `json:"identifier"`
		Type       string `json:"type"` // "ip", "user", "api_key"
		Method     string `json:"method"`
		Path       string `json:"path"`
		Role       string `json:"role,omitempty"`
		OrgID      string `json:"org_id,omitempty"`
		IsBurst    bool   `json:"is_burst,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	if req.Identifier == "" {
		respondError(w, apperrors.New("identifier is required", http.StatusBadRequest))
		return
	}
	if req.Type == "" {
		req.Type = "ip"
	}

	rlReq := &ratelimit.Request{
		Identifier: req.Identifier,
		Type:       req.Type,
		Method:     req.Method,
		Path:       req.Path,
		Role:       req.Role,
		OrgID:      req.OrgID,
		IsBurst:    req.IsBurst,
		Timestamp:  time.Now(),
	}

	if h.limiter == nil {
		respondError(w, apperrors.New("rate limiter not initialized", http.StatusServiceUnavailable))
		return
	}

	result, err := h.limiter.Check(ctx, rlReq)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to check rate limit", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// ResetRateLimit handles rate limit reset requests.
func (h *RateLimitHandler) ResetRateLimit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Identifier string `json:"identifier"`
		Type       string `json:"type"`
		Method     string `json:"method"`
		Path       string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	rlReq := &ratelimit.Request{
		Identifier: req.Identifier,
		Type:       req.Type,
		Method:     req.Method,
		Path:       req.Path,
		Timestamp:  time.Now(),
	}

	if h.limiter == nil {
		respondError(w, apperrors.New("rate limiter not initialized", http.StatusServiceUnavailable))
		return
	}

	if err := h.limiter.Reset(ctx, rlReq); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to reset rate limit", http.StatusInternalServerError))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetUsage returns current usage statistics.
func (h *RateLimitHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	identifier := r.URL.Query().Get("identifier")
	identifierType := r.URL.Query().Get("type")

	if identifier == "" {
		respondError(w, apperrors.New("identifier is required", http.StatusBadRequest))
		return
	}
	if identifierType == "" {
		identifierType = "ip"
	}

	req := &ratelimit.Request{
		Identifier: identifier,
		Type:       identifierType,
		Timestamp:  time.Now(),
	}

	if h.limiter == nil {
		respondError(w, apperrors.New("rate limiter not initialized", http.StatusServiceUnavailable))
		return
	}

	current, limit, resetAt, err := h.limiter.GetUsage(ctx, req)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get usage", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"current":   current,
		"limit":     limit,
		"remaining": limit - current,
		"reset_at":  resetAt.Format(time.RFC3339),
	})
}

// Helper functions

func policyToResponse(p *ratelimit.Policy) map[string]interface{} {
	return map[string]interface{}{
		"id":                          p.ID,
		"name":                        p.Name,
		"description":                 p.Description,
		"priority":                    p.Priority,
		"scope":                       p.Scope,
		"identifier":                  p.Identifier,
		"methods":                     p.Methods,
		"path_pattern":                p.PathPattern,
		"limit":                       p.Limit,
		"window_sec":                  int64(p.Window.Seconds()),
		"burst_limit":                 p.BurstLimit,
		"burst_duration_sec":          int64(p.BurstDuration.Seconds()),
		"enable_queue":                p.EnableQueue,
		"max_queue_size":              p.MaxQueueSize,
		"circuit_breaker_threshold":   p.CircuitBreakerThreshold,
		"circuit_breaker_timeout_sec": int64(p.CircuitBreakerTimeout.Seconds()),
		"severity":                    p.Severity,
		"action":                      p.Action,
		"throttle_rate":               p.ThrottleRate,
		"is_active":                   p.IsActive,
		"created_at":                  p.CreatedAt.Format(time.RFC3339),
		"updated_at":                  p.UpdatedAt.Format(time.RFC3339),
	}
}

func parsePath(path string) []string {
	return strings.Split(strings.Trim(path, "/"), "/")
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

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
