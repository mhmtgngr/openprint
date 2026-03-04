// Package handler provides HTTP handlers for the analytics service.
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/openprint/openprint/services/analytics-service/processor"
)

// Handler handles analytics HTTP requests.
type Handler struct {
	aggregator *processor.Aggregator
}

// New creates a new analytics handler.
func New(aggregator *processor.Aggregator) *Handler {
	return &Handler{
		aggregator: aggregator,
	}
}

// JobsAnalyticsHandler handles GET /api/v1/analytics/jobs
func (h *Handler) JobsAnalyticsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
			days = parsed
		}
	}

	// Get job statistics
	stats, err := h.aggregator.AggregateJobsByDate(ctx, days)
	if err != nil {
		http.Error(w, "Failed to get job statistics", http.StatusInternalServerError)
		return
	}

	// Get status breakdown
	statusBreakdown, err := h.aggregator.GetJobsByStatus(ctx)
	if err != nil {
		http.Error(w, "Failed to get status breakdown", http.StatusInternalServerError)
		return
	}
	stats.StatusBreakdown = statusBreakdown

	// Get daily job counts for trends
	dailyStats, err := h.aggregator.GetDailyJobCounts(ctx, days)
	if err != nil {
		http.Error(w, "Failed to get daily trends", http.StatusInternalServerError)
		return
	}
	stats.DailyTrends = dailyStats

	// Calculate trends
	stats.Trends = h.calculateTrends(dailyStats)

	respondJSON(w, http.StatusOK, stats)
}

// PrintersAnalyticsHandler handles GET /api/v1/analytics/printers
func (h *Handler) PrintersAnalyticsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
			days = parsed
		}
	}

	// Get printer usage statistics
	stats, err := h.aggregator.AggregateByPrinter(ctx, days)
	if err != nil {
		http.Error(w, "Failed to get printer statistics", http.StatusInternalServerError)
		return
	}

	// Get printer status distribution
	statusDist, err := h.aggregator.GetPrinterStatusDistribution(ctx)
	if err != nil {
		http.Error(w, "Failed to get printer status distribution", http.StatusInternalServerError)
		return
	}
	stats.StatusDistribution = statusDist

	respondJSON(w, http.StatusOK, stats)
}

// UsersAnalyticsHandler handles GET /api/v1/analytics/users
func (h *Handler) UsersAnalyticsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
			days = parsed
		}
	}

	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	// Get user activity statistics
	stats, err := h.aggregator.AggregateByUser(ctx, days, limit)
	if err != nil {
		http.Error(w, "Failed to get user statistics", http.StatusInternalServerError)
		return
	}

	// Get active users count
	activeUsers, err := h.aggregator.GetActiveUsersCount(ctx, days)
	if err != nil {
		http.Error(w, "Failed to get active users count", http.StatusInternalServerError)
		return
	}
	stats.ActiveUsersCount = activeUsers

	respondJSON(w, http.StatusOK, stats)
}

// calculateTrends calculates trend percentages from daily stats
func (h *Handler) calculateTrends(dailyStats []processor.DailyJobCount) processor.Trends {
	if len(dailyStats) < 2 {
		return processor.Trends{
			JobsChangePercent:  0,
			PagesChangePercent: 0,
		}
	}

	// Compare last 7 days with previous 7 days
	now := time.Now()
	sevenDaysAgo := now.AddDate(0, 0, -7)
	fourteenDaysAgo := now.AddDate(0, 0, -14)

	var recentJobs, previousJobs int
	var recentPages, previousPages int

	for _, stat := range dailyStats {
		if stat.Date.After(sevenDaysAgo) {
			recentJobs += stat.Count
			recentPages += stat.Pages
		} else if stat.Date.After(fourteenDaysAgo) {
			previousJobs += stat.Count
			previousPages += stat.Pages
		}
	}

	jobsChange := calculatePercentChange(previousJobs, recentJobs)
	pagesChange := calculatePercentChange(previousPages, recentPages)

	return processor.Trends{
		JobsChangePercent:  jobsChange,
		PagesChangePercent: pagesChange,
	}
}

// calculatePercentChange calculates the percentage change from old to new
func calculatePercentChange(old, new int) float64 {
	if old == 0 {
		if new > 0 {
			return 100.0
		}
		return 0.0
	}
	return float64(new-old) / float64(old) * 100.0
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
