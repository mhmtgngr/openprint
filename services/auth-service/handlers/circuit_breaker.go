// Package handlers provides HTTP handlers for circuit breaker management.
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/internal/shared/ratelimit"
)

// CircuitBreakerHandler handles circuit breaker status and control endpoints.
type CircuitBreakerHandler struct {
	rateLimitHandler *RateLimitHandler
}

// NewCircuitBreakerHandler creates a new circuit breaker handler.
func NewCircuitBreakerHandler(rlHandler *RateLimitHandler) *CircuitBreakerHandler {
	return &CircuitBreakerHandler{
		rateLimitHandler: rlHandler,
	}
}

// GetCircuitState handles getting the state of a circuit breaker.
func (h *CircuitBreakerHandler) GetCircuitState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		path = r.URL.Path
	}

	limiter := h.rateLimitHandler.GetLimiter()
	if limiter == nil {
		respondError(w, apperrors.New("rate limiter not initialized", http.StatusServiceUnavailable))
		return
	}

	cb := limiter.GetCircuitBreaker()
	if cb == nil {
		respondError(w, apperrors.New("circuit breaker not enabled", http.StatusServiceUnavailable))
		return
	}

	state := cb.GetState(path)

	respondJSON(w, http.StatusOK, circuitStateToResponse(state))
}

// ListCircuitStates handles listing all circuit breaker states.
func (h *CircuitBreakerHandler) ListCircuitStates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limiter := h.rateLimitHandler.GetLimiter()
	if limiter == nil {
		respondError(w, apperrors.New("rate limiter not initialized", http.StatusServiceUnavailable))
		return
	}

	cb := limiter.GetCircuitBreaker()
	if cb == nil {
		respondError(w, apperrors.New("circuit breaker not enabled", http.StatusServiceUnavailable))
		return
	}

	states := cb.GetAllStates()

	response := make([]map[string]interface{}, len(states))
	for i, state := range states {
		response[i] = circuitStateToResponse(state)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"circuits": response,
		"count":    len(response),
	})
}

// ResetCircuit handles resetting a circuit breaker to closed state.
func (h *CircuitBreakerHandler) ResetCircuit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	if req.Path == "" {
		respondError(w, apperrors.New("path is required", http.StatusBadRequest))
		return
	}

	limiter := h.rateLimitHandler.GetLimiter()
	if limiter == nil {
		respondError(w, apperrors.New("rate limiter not initialized", http.StatusServiceUnavailable))
		return
	}

	cb := limiter.GetCircuitBreaker()
	if cb == nil {
		respondError(w, apperrors.New("circuit breaker not enabled", http.StatusServiceUnavailable))
		return
	}

	if err := cb.Reset(req.Path); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to reset circuit", http.StatusInternalServerError))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ForceOpenCircuit forcibly opens a circuit breaker.
func (h *CircuitBreakerHandler) ForceOpenCircuit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path     string `json:"path"`
		Duration int    `json:"duration"` // seconds
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	if req.Path == "" {
		respondError(w, apperrors.New("path is required", http.StatusBadRequest))
		return
	}

	duration := time.Duration(req.Duration) * time.Second
	if duration <= 0 {
		duration = 5 * time.Minute // Default 5 minutes
	}

	limiter := h.rateLimitHandler.GetLimiter()
	if limiter == nil {
		respondError(w, apperrors.New("rate limiter not initialized", http.StatusServiceUnavailable))
		return
	}

	cb := limiter.GetCircuitBreaker()
	if cb == nil {
		respondError(w, apperrors.New("circuit breaker not enabled", http.StatusServiceUnavailable))
		return
	}

	cb.ForceOpen(req.Path, duration)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Circuit opened",
		"path":    req.Path,
		"duration": duration.String(),
	})
}

// ForceCloseCircuit forcibly closes a circuit breaker.
func (h *CircuitBreakerHandler) ForceCloseCircuit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	if req.Path == "" {
		respondError(w, apperrors.New("path is required", http.StatusBadRequest))
		return
	}

	limiter := h.rateLimitHandler.GetLimiter()
	if limiter == nil {
		respondError(w, apperrors.New("rate limiter not initialized", http.StatusServiceUnavailable))
		return
	}

	cb := limiter.GetCircuitBreaker()
	if cb == nil {
		respondError(w, apperrors.New("circuit breaker not enabled", http.StatusServiceUnavailable))
		return
	}

	cb.ForceClose(req.Path)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Circuit closed",
		"path":    req.Path,
	})
}

// GetCircuitStats handles circuit breaker statistics.
func (h *CircuitBreakerHandler) GetCircuitStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limiter := h.rateLimitHandler.GetLimiter()
	if limiter == nil {
		respondError(w, apperrors.New("rate limiter not initialized", http.StatusServiceUnavailable))
		return
	}

	cb := limiter.GetCircuitBreaker()
	if cb == nil {
		respondError(w, apperrors.New("circuit breaker not enabled", http.StatusServiceUnavailable))
		return
	}

	stats := cb.GetStats()

	respondJSON(w, http.StatusOK, circuitStatsToResponse(stats))
}

// Helper functions

func circuitStateToResponse(state ratelimit.CircuitBreakerState) map[string]interface{} {
	response := map[string]interface{}{
		"path":          state.Path,
		"state":         state.State,
		"failure_count": state.FailureCount,
	}

	if !state.LastFailureAt.IsZero() {
		response["last_failure_at"] = state.LastFailureAt.Format(time.RFC3339)
	}
	if state.OpensAt != nil {
		response["opens_at"] = state.OpensAt.Format(time.RFC3339)
	}
	if state.ClosesAt != nil {
		response["closes_at"] = state.ClosesAt.Format(time.RFC3339)
	}

	return response
}

func circuitStatsToResponse(stats *ratelimit.CircuitBreakerStats) map[string]interface{} {
	return map[string]interface{}{
		"total_circuits":     stats.TotalCircuits,
		"closed_circuits":    stats.ClosedCircuits,
		"open_circuits":      stats.OpenCircuits,
		"half_open_circuits": stats.HalfOpenCircuits,
		"total_failures":     stats.TotalFailures,
	}
}
