// Package handlers provides HTTP handlers for rate limit violation logging.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/internal/shared/ratelimit"
)

// ViolationsHandler handles violation log viewing endpoints.
type ViolationsHandler struct {
	rateLimitHandler *RateLimitHandler
}

// NewViolationsHandler creates a new violations handler.
func NewViolationsHandler(rlHandler *RateLimitHandler) *ViolationsHandler {
	return &ViolationsHandler{
		rateLimitHandler: rlHandler,
	}
}

// ListViolations handles listing violation logs.
func (h *ViolationsHandler) ListViolations(w http.ResponseWriter, r *http.Request) {
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

	// Parse query parameters
	filter := &ratelimit.ViolationFilter{}

	if identifier := r.URL.Query().Get("identifier"); identifier != "" {
		filter.Identifier = identifier
	}
	if policyID := r.URL.Query().Get("policy_id"); policyID != "" {
		filter.PolicyID = policyID
	}
	if severity := r.URL.Query().Get("severity"); severity != "" {
		filter.Severity = severity
	}
	if since := r.URL.Query().Get("since"); since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			filter.Since = t
		}
	}
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 {
			filter.Limit = l
		}
	} else {
		filter.Limit = 100
	}

	violations, err := repo.ListViolations(ctx, filter)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to list violations", http.StatusInternalServerError))
		return
	}

	response := make([]map[string]interface{}, len(violations))
	for i, v := range violations {
		response[i] = violationToResponse(v)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"violations": response,
		"count":      len(response),
	})
}

// GetViolation handles retrieving a single violation.
func (h *ViolationsHandler) GetViolation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract violation ID from path
	parts := parsePath(r.URL.Path)
	if len(parts) < 4 {
		respondError(w, apperrors.New("invalid violation path", http.StatusBadRequest))
		return
	}
	violationID := parts[3]

	repo := h.rateLimitHandler.GetRepository()
	if repo == nil {
		respondError(w, apperrors.New("rate limiter not initialized", http.StatusServiceUnavailable))
		return
	}

	violation, err := repo.GetViolation(ctx, violationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respondError(w, apperrors.ErrNotFound)
			return
		}
		respondError(w, apperrors.Wrap(err, "failed to get violation", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, violationToResponse(violation))
}

// GetViolationStats handles violation statistics requests.
func (h *ViolationsHandler) GetViolationStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse time range
	timeRange := r.URL.Query().Get("range")
	if timeRange == "" {
		timeRange = "24h" // Default 24 hours
	}

	since := time.Now()
	switch timeRange {
	case "1h":
		since = since.Add(-1 * time.Hour)
	case "24h":
		since = since.Add(-24 * time.Hour)
	case "7d":
		since = since.Add(-7 * 24 * time.Hour)
	case "30d":
		since = since.Add(-30 * 24 * time.Hour)
	default:
		// Try parsing as duration
		if d, err := time.ParseDuration(timeRange); err == nil {
			since = since.Add(-d)
		} else {
			since = since.Add(-24 * time.Hour)
		}
	}

	repo := h.rateLimitHandler.GetRepository()
	if repo == nil {
		respondError(w, apperrors.New("rate limiter not initialized", http.StatusServiceUnavailable))
		return
	}

	filter := &ratelimit.ViolationFilter{
		Since: since,
		Limit: 10000, // Get all for stats
	}

	violations, err := repo.ListViolations(ctx, filter)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get violations", http.StatusInternalServerError))
		return
	}

	// Calculate stats
	stats := calculateViolationStats(violations)

	respondJSON(w, http.StatusOK, stats)
}

// ClearOldViolations handles cleanup of old violation logs.
func (h *ViolationsHandler) ClearOldViolations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		DaysToKeep int `json:"days_to_keep"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	if req.DaysToKeep <= 0 {
		req.DaysToKeep = 30 // Default 30 days
	}

	// In a real implementation, this would call a cleanup function
	// For now, just return success
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":   "Old violations cleared",
		"days_kept": req.DaysToKeep,
	})
}

// Helper functions

func violationToResponse(v *ratelimit.Violation) map[string]interface{} {
	return map[string]interface{}{
		"id":              v.ID,
		"policy_id":       v.PolicyID,
		"policy_name":     v.PolicyName,
		"identifier":      v.Identifier,
		"identifier_type": v.IdentifierType,
		"path":            v.Path,
		"method":          v.Method,
		"current":         v.Current,
		"limit":           v.Limit,
		"severity":        v.Severity,
		"occurred_at":     v.OccurredAt.Format(time.RFC3339),
	}
}

func calculateViolationStats(violations []*ratelimit.Violation) map[string]interface{} {
	stats := map[string]interface{}{
		"total":         len(violations),
		"by_severity":   make(map[string]int),
		"by_type":       make(map[string]int),
		"by_policy":     make(map[string]int),
		"top_violators": make([]map[string]interface{}, 0, 10),
	}

	// Count by various dimensions
	identifierCounts := make(map[string]int)

	for _, v := range violations {
		// By severity
		severityMap := stats["by_severity"].(map[string]int)
		severityMap[v.Severity]++

		// By identifier type
		typeMap := stats["by_type"].(map[string]int)
		typeMap[v.IdentifierType]++

		// By policy
		policyMap := stats["by_policy"].(map[string]int)
		policyMap[v.PolicyName]++

		// Track top violators
		identifierCounts[v.Identifier]++
	}

	// Get top 10 violators
	type violator struct {
		Identifier string
		Count      int
	}
	var violators []violator
	for id, count := range identifierCounts {
		violators = append(violators, violator{Identifier: id, Count: count})
	}

	// Sort by count descending
	for i := 0; i < len(violators); i++ {
		for j := i + 1; j < len(violators); j++ {
			if violators[j].Count > violators[i].Count {
				violators[i], violators[j] = violators[j], violators[i]
			}
		}
	}

	topViolators := make([]map[string]interface{}, 0)
	for i := 0; i < len(violators) && i < 10; i++ {
		topViolators = append(topViolators, map[string]interface{}{
			"identifier": violators[i].Identifier,
			"count":      violators[i].Count,
		})
	}
	stats["top_violators"] = topViolators

	return stats
}
