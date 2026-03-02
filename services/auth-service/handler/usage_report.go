// Package handler provides HTTP handlers for the auth service endpoints.
// This file contains usage report handlers for resource usage trends.
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/internal/shared/middleware"
	"github.com/openprint/openprint/services/auth-service/repository"
)

// UsageReportRequest represents a request for usage reports.
type UsageReportRequest struct {
	StartDate string `json:"start_date,omitempty"`
	EndDate   string `json:"end_date,omitempty"`
	Months    int    `json:"months,omitempty"`
}

// UsageTrend represents a single data point in usage trends.
type UsageTrend struct {
	Month          string  `json:"month"`
	PrintersCount  int32   `json:"printers_count"`
	StorageUsedGB  int64   `json:"storage_used_gb"`
	JobsCount      int32   `json:"jobs_count"`
	UsersCount     int32   `json:"users_count"`
}

// UsageReportResponse represents a comprehensive usage report.
type UsageReportResponse struct {
	TenantID       string          `json:"tenant_id"`
	TenantName     string          `json:"tenant_name,omitempty"`
	Period         PeriodInfo      `json:"period"`
	Current        CurrentUsage    `json:"current"`
	Trends         []UsageTrend    `json:"trends,omitempty"`
	ResourceQuotas ResourceQuotas  `json:"resource_quotas"`
	Alerts         []UsageAlert    `json:"alerts,omitempty"`
}

// PeriodInfo represents the reporting period.
type PeriodInfo struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Months    int    `json:"months"`
}

// CurrentUsage represents current resource usage.
type CurrentUsage struct {
	Printers  UsageMetric `json:"printers"`
	Storage   UsageMetric `json:"storage"`
	Jobs      UsageMetric `json:"jobs"`
	Users     UsageMetric `json:"users"`
}

// UsageMetric represents a single resource usage metric.
type UsageMetric struct {
	Current      int64   `json:"current"`
	Maximum      int64   `json:"maximum"`
	UsagePercent float64 `json:"usage_percent"`
	Remaining    int64   `json:"remaining"`
	IsNearLimit  bool    `json:"is_near_limit"`
	Trend        string  `json:"trend,omitempty"` // "up", "down", "stable"
}

// ResourceQuotas represents quota configurations.
type ResourceQuotas struct {
	MaxPrinters     int32 `json:"max_printers"`
	MaxStorageGB    int32 `json:"max_storage_gb"`
	MaxJobsPerMonth int32 `json:"max_jobs_per_month"`
	MaxUsers        int32 `json:"max_users"`
	AlertThreshold  int32 `json:"alert_threshold"`
}

// UsageAlert represents a quota alert.
type UsageAlert struct {
	ResourceType string  `json:"resource_type"`
	Severity     string  `json:"severity"` // "warning", "critical"
	Message      string  `json:"message"`
	UsagePercent float64 `json:"usage_percent"`
}

// GetUsageReport handles retrieving a usage report for an organization.
func (h *Handler) GetUsageReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract organization ID from URL or use user's organization
	orgID := extractOrgID(r.URL.Path)
	if orgID == "" {
		orgID = middleware.GetOrgID(r)
	}

	// Check authorization
	if !canAccessOrganization(ctx, r, orgID) {
		respondError(w, apperrors.New("forbidden: cannot access this organization", http.StatusForbidden))
		return
	}

	// Parse request parameters
	var req UsageReportRequest
	if r.Method == http.MethodPost {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
			return
		}
	} else {
		// Parse from query parameters for GET requests
		req.StartDate = r.URL.Query().Get("start_date")
		req.EndDate = r.URL.Query().Get("end_date")
		req.Months, _ = strconv.Atoi(r.URL.Query().Get("months"))
	}

	// Set defaults
	if req.Months == 0 {
		req.Months = 6 // Default to 6 months of trends
	}

	// Get repositories
	quotaRepo := h.quotaRepo()
	orgRepo := h.orgRepo()

	// Get organization info
	org, err := orgRepo.GetByID(ctx, orgID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get organization", http.StatusInternalServerError))
		return
	}

	// Get quota config
	config, err := quotaRepo.GetConfig(ctx, orgID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get quota config", http.StatusInternalServerError))
		return
	}

	// Get current usage
	usage, err := quotaRepo.GetUsage(ctx, orgID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get usage", http.StatusInternalServerError))
		return
	}

	// Build response
	response := &UsageReportResponse{
		TenantID:   orgID,
		TenantName: org.Name,
		Period: PeriodInfo{
			StartDate: time.Now().AddDate(0, -req.Months, 0).Format("2006-01-02"),
			EndDate:   time.Now().Format("2006-01-02"),
			Months:    req.Months,
		},
		Current: buildCurrentUsage(config, usage),
		ResourceQuotas: ResourceQuotas{
			MaxPrinters:     config.MaxPrinters,
			MaxStorageGB:    config.MaxStorageGB,
			MaxJobsPerMonth: config.MaxJobsPerMonth,
			MaxUsers:        config.MaxUsers,
			AlertThreshold:  config.AlertThreshold,
		},
	}

	// Get trends if requested
	if req.Months > 0 {
		response.Trends, err = h.getUsageTrends(ctx, quotaRepo, orgID, req.Months)
		if err != nil {
			// Don't fail on trends error, just log it
			response.Trends = []UsageTrend{}
		}
	}

	// Generate alerts
	response.Alerts = generateUsageAlerts(config, usage)

	respondJSON(w, http.StatusOK, response)
}

// ListUsageReports handles listing usage reports for multiple organizations (platform admin only).
func (h *Handler) ListUsageReports(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Only platform admins can list all usage reports
	role := middleware.GetRole(r)
	if role != "admin" && role != "platform_admin" {
		respondError(w, apperrors.New("only platform admins can list all usage reports", http.StatusForbidden))
		return
	}

	// Parse query parameters
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	status := repository.OrganizationStatus(r.URL.Query().Get("status"))

	if limit <= 0 {
		limit = defaultOrgListLimit
	}
	if limit > maxOrgListLimit {
		limit = maxOrgListLimit
	}

	// Get organizations
	orgRepo := h.orgRepo()
	orgs, total, err := orgRepo.List(ctx, limit, offset, status)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to list organizations", http.StatusInternalServerError))
		return
	}

	// Build usage reports for each organization
	quotaRepo := h.quotaRepo()
	reports := make([]*UsageReportResponse, 0, len(orgs))

	for _, org := range orgs {
		config, _ := quotaRepo.GetConfig(ctx, org.ID)
		usage, _ := quotaRepo.GetUsage(ctx, org.ID)

		report := &UsageReportResponse{
			TenantID:   org.ID,
			TenantName: org.Name,
			Current:    buildCurrentUsage(config, usage),
			ResourceQuotas: ResourceQuotas{
				MaxPrinters:     config.MaxPrinters,
				MaxStorageGB:    config.MaxStorageGB,
				MaxJobsPerMonth: config.MaxJobsPerMonth,
				MaxUsers:        config.MaxUsers,
				AlertThreshold:  config.AlertThreshold,
			},
			Alerts: generateUsageAlerts(config, usage),
		}

		reports = append(reports, report)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"reports": reports,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// GetResourceUsage handles retrieving usage for a specific resource type.
func (h *Handler) GetResourceUsage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract organization ID from URL or use user's organization
	orgID := extractOrgID(r.URL.Path)
	if orgID == "" {
		orgID = middleware.GetOrgID(r)
	}

	// Check authorization
	if !canAccessOrganization(ctx, r, orgID) {
		respondError(w, apperrors.New("forbidden: cannot access this organization", http.StatusForbidden))
		return
	}

	// Get resource type from URL
	// Expected format: /organizations/{org_id}/usage/{resource_type}
	pathParts := splitPath(r.URL.Path)
	resourceType := ""
	for i, part := range pathParts {
		if part == "usage" && i+1 < len(pathParts) {
			resourceType = pathParts[i+1]
			break
		}
	}

	quotaRepo := h.quotaRepo()
	config, err := quotaRepo.GetConfig(ctx, orgID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get quota config", http.StatusInternalServerError))
		return
	}

	usage, err := quotaRepo.GetUsage(ctx, orgID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get usage", http.StatusInternalServerError))
		return
	}

	var metric UsageMetric
	switch resourceType {
	case "printers":
		metric = buildUsageMetric(int64(usage.PrintersCount), int64(config.MaxPrinters), config.AlertThreshold)
	case "storage":
		metric = buildUsageMetric(usage.StorageUsedGB, int64(config.MaxStorageGB)*1024*1024*1024, config.AlertThreshold)
	case "jobs":
		metric = buildUsageMetric(int64(usage.JobsThisMonth), int64(config.MaxJobsPerMonth), config.AlertThreshold)
	case "users":
		metric = buildUsageMetric(int64(usage.UsersCount), int64(config.MaxUsers), config.AlertThreshold)
	default:
		respondError(w, apperrors.New("invalid resource type", http.StatusBadRequest))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"resource_type": resourceType,
		"usage":         metric,
	})
}

// Helper functions

// getUsageTrends retrieves usage trends for an organization.
func (h *Handler) getUsageTrends(ctx context.Context, quotaRepo *repository.QuotaRepository, orgID string, months int) ([]UsageTrend, error) {
	trends := make([]UsageTrend, 0, months)

	now := time.Now().UTC()
	for i := months - 1; i >= 0; i-- {
		month := now.AddDate(0, -i, 0)
		monthStart := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)

		usage, err := quotaRepo.GetTenantUsageForMonth(ctx, orgID, month)
		if err != nil {
			// Skip months without data
			continue
		}

		trends = append(trends, UsageTrend{
			Month:          month.Format("2006-01"),
			PrintersCount:  usage.PrintersCount,
			StorageUsedGB:  usage.StorageUsedGB / (1024 * 1024 * 1024), // Convert to GB
			JobsCount:      usage.JobsThisMonth,
			UsersCount:     usage.UsersCount,
		})
	}

	return trends, nil
}

// buildCurrentUsage builds the current usage section of the report.
func buildCurrentUsage(config *repository.QuotaConfig, usage *repository.QuotaUsage) CurrentUsage {
	return CurrentUsage{
		Printers: buildUsageMetric(int64(usage.PrintersCount), int64(config.MaxPrinters), config.AlertThreshold),
		Storage:  buildUsageMetric(usage.StorageUsedGB, int64(config.MaxStorageGB)*1024*1024*1024, config.AlertThreshold),
		Jobs:     buildUsageMetric(int64(usage.JobsThisMonth), int64(config.MaxJobsPerMonth), config.AlertThreshold),
		Users:    buildUsageMetric(int64(usage.UsersCount), int64(config.MaxUsers), config.AlertThreshold),
	}
}

// buildUsageMetric builds a usage metric for a single resource.
func buildUsageMetric(current, maximum int64, alertThreshold int32) UsageMetric {
	var usagePercent float64
	if maximum > 0 {
		usagePercent = float64(current) / float64(maximum) * 100
	}

	remaining := maximum - current
	if remaining < 0 {
		remaining = 0
	}

	return UsageMetric{
		Current:      current,
		Maximum:      maximum,
		UsagePercent: usagePercent,
		Remaining:    remaining,
		IsNearLimit:  maximum > 0 && usagePercent >= float64(alertThreshold),
	}
}

// generateUsageAlerts generates alerts based on current usage.
func generateUsageAlerts(config *repository.QuotaConfig, usage *repository.QuotaUsage) []UsageAlert {
	alerts := []UsageAlert{}

	checkAndAddAlert := func(resourceType string, current, maximum int64) {
		if maximum <= 0 {
			return
		}

		usagePercent := float64(current) / float64(maximum) * 100
		threshold := float64(config.AlertThreshold)

		if usagePercent >= 100 {
			alerts = append(alerts, UsageAlert{
				ResourceType: resourceType,
				Severity:     "critical",
				Message:      fmt.Sprintf("%s quota exceeded", resourceType),
				UsagePercent: usagePercent,
			})
		} else if usagePercent >= threshold {
			alerts = append(alerts, UsageAlert{
				ResourceType: resourceType,
				Severity:     "warning",
				Message:      fmt.Sprintf("%s quota at %.1f%% capacity", resourceType, usagePercent),
				UsagePercent: usagePercent,
			})
		}
	}

	checkAndAddAlert("printers", int64(usage.PrintersCount), int64(config.MaxPrinters))
	checkAndAddAlert("storage", usage.StorageUsedGB, int64(config.MaxStorageGB)*1024*1024*1024)
	checkAndAddAlert("jobs", int64(usage.JobsThisMonth), int64(config.MaxJobsPerMonth))
	checkAndAddAlert("users", int64(usage.UsersCount), int64(config.MaxUsers))

	return alerts
}

// Import for fmt.Sprintf in generateUsageAlerts
import "fmt"
