// Package handler provides HTTP handlers for the auth service endpoints.
// This file contains quota management handlers.
package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/internal/shared/middleware"
	"github.com/openprint/openprint/internal/multi-tenant"
	"github.com/openprint/openprint/services/auth-service/repository"
)

const (
	defaultMaxPrinters     = 100
	defaultMaxStorageGB    = 100
	defaultMaxJobsPerMonth = 10000
	defaultMaxUsers        = 50
	defaultAlertThreshold  = 80
)

// QuotaConfigRequest represents a quota configuration request.
type QuotaConfigRequest struct {
	MaxPrinters     int32 `json:"max_printers,omitempty"`
	MaxStorageGB    int32 `json:"max_storage_gb,omitempty"`
	MaxJobsPerMonth int32 `json:"max_jobs_per_month,omitempty"`
	MaxUsers        int32 `json:"max_users,omitempty"`
	AlertThreshold  int32 `json:"alert_threshold,omitempty"`
}

// QuotaConfigResponse represents a quota configuration response.
type QuotaConfigResponse struct {
	TenantID        string  `json:"tenant_id"`
	MaxPrinters     int32   `json:"max_printers"`
	MaxStorageGB    int32   `json:"max_storage_gb"`
	MaxJobsPerMonth int32   `json:"max_jobs_per_month"`
	MaxUsers        int32   `json:"max_users"`
	AlertThreshold  int32   `json:"alert_threshold"`
}

// QuotaUsageResponse represents quota usage information.
type QuotaUsageResponse struct {
	TenantID        string  `json:"tenant_id"`
	Printers        QuotaStat `json:"printers"`
	Storage         QuotaStat `json:"storage"`
	Jobs            QuotaStat `json:"jobs"`
	Users           QuotaStat `json:"users"`
	Month           string   `json:"month"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// QuotaStat represents a single quota statistic.
type QuotaStat struct {
	Current       int64   `json:"current"`
	Maximum       int64   `json:"maximum"`
	UsagePercent  float64 `json:"usage_percent"`
	Remaining     int64   `json:"remaining"`
	IsNearLimit   bool    `json:"is_near_limit"`
}

// GetQuotaConfig handles retrieving quota configuration for an organization.
func (h *Handler) GetQuotaConfig(w http.ResponseWriter, r *http.Request) {
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

	quotaRepo := h.quotaRepo()
	config, err := quotaRepo.GetConfig(ctx, orgID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get quota config", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, quotaConfigToResponse(config))
}

// UpdateQuotaConfig handles updating quota configuration (platform admin or org admin only).
func (h *Handler) UpdateQuotaConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract organization ID from URL
	orgID := extractOrgID(r.URL.Path)
	if orgID == "" {
		respondError(w, apperrors.New("organization ID required", http.StatusBadRequest))
		return
	}

	// Check authorization - only platform admin or org admin can update quotas
	if !canManageOrganization(ctx, r, orgID) {
		respondError(w, apperrors.New("forbidden: cannot manage quotas for this organization", http.StatusForbidden))
		return
	}

	var req QuotaConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate request
	if err := validateQuotaConfigRequest(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid input", http.StatusBadRequest))
		return
	}

	quotaRepo := h.quotaRepo()
	config, err := quotaRepo.GetConfig(ctx, orgID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get quota config", http.StatusInternalServerError))
		return
	}

	// Update fields if provided
	if req.MaxPrinters > 0 {
		config.MaxPrinters = req.MaxPrinters
	}
	if req.MaxStorageGB > 0 {
		config.MaxStorageGB = req.MaxStorageGB
	}
	if req.MaxJobsPerMonth > 0 {
		config.MaxJobsPerMonth = req.MaxJobsPerMonth
	}
	if req.MaxUsers > 0 {
		config.MaxUsers = req.MaxUsers
	}
	if req.AlertThreshold > 0 {
		config.AlertThreshold = req.AlertThreshold
	}

	if err := quotaRepo.UpdateConfig(ctx, config); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to update quota config", http.StatusInternalServerError))
		return
	}

	// Log the quota change
	if h.auditLogger != nil {
		userID := middleware.GetUserID(r)
		userEmail := middleware.GetEmail(r)
		h.auditLogger.LogUpdate(ctx, userID, userEmail, "quota_config", orgID, orgID, map[string]interface{}{
			"max_printers":      config.MaxPrinters,
			"max_storage_gb":    config.MaxStorageGB,
			"max_jobs_per_month": config.MaxJobsPerMonth,
			"max_users":         config.MaxUsers,
		})
	}

	respondJSON(w, http.StatusOK, quotaConfigToResponse(config))
}

// GetQuotaUsage handles retrieving quota usage for an organization.
func (h *Handler) GetQuotaUsage(w http.ResponseWriter, r *http.Request) {
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

	quotaRepo := h.quotaRepo()
	config, err := quotaRepo.GetConfig(ctx, orgID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get quota config", http.StatusInternalServerError))
		return
	}

	usage, err := quotaRepo.GetUsage(ctx, orgID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get quota usage", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, quotaUsageToResponse(config, usage))
}

// CheckQuotaAvailability checks if an organization has quota available for a resource type.
func (h *Handler) CheckQuotaAvailability(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ResourceType string `json:"resource_type"`
		Count        int    `json:"count"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	if req.Count <= 0 {
		req.Count = 1
	}

	// Get quota enforcer
	enforcer := multi_tenant.NewQuotaEnforcer(h.quotaRepo())

	var result *multi_tenant.QuotaCheckResult
	var err error

	switch multi_tenant.ResourceType(req.ResourceType) {
	case multi_tenant.ResourcePrinters:
		result, err = enforcer.CheckPrinterQuota(ctx, req.Count)
	case multi_tenant.ResourceStorage:
		result, err = enforcer.CheckPrinterQuota(ctx, req.Count)
	case multi_tenant.ResourceJobs:
		result, err = enforcer.CheckJobQuota(ctx, req.Count)
	case multi_tenant.ResourceUsers:
		result, err = enforcer.CheckUserQuota(ctx, req.Count)
	default:
		respondError(w, apperrors.New("invalid resource type", http.StatusBadRequest))
		return
	}

	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to check quota", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"allowed":        result.IsAllowed(),
		"resource_type":  result.ResourceType,
		"current":        result.Current,
		"maximum":        result.Maximum,
		"usage_percent":  result.UsagePercent,
		"remaining":      result.Remaining,
		"status":         string(result.Status),
	})
}

// ListQuotaConfigs lists quota configurations for multiple organizations (platform admin only).
func (h *Handler) ListQuotaConfigs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Only platform admins can list all quota configs
	role := middleware.GetRole(r)
	if role != "admin" && role != "platform_admin" {
		respondError(w, apperrors.New("only platform admins can list all quota configurations", http.StatusForbidden))
		return
	}

	// Parse organization IDs from query
	orgIDs := r.URL.Query()["organization_id"]
	if len(orgIDs) == 0 {
		// Return empty list if no orgs specified
		respondJSON(w, http.StatusOK, []*QuotaConfigResponse{})
		return
	}

	quotaRepo := h.quotaRepo()
	configs, err := quotaRepo.ListQuotaConfigs(ctx, orgIDs)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to list quota configs", http.StatusInternalServerError))
		return
	}

	responses := make([]*QuotaConfigResponse, len(configs))
	for i, config := range configs {
		responses[i] = quotaConfigToResponse(config)
	}

	respondJSON(w, http.StatusOK, responses)
}

// ResetMonthlyUsage resets monthly job counters (admin or scheduled job only).
func (h *Handler) ResetMonthlyUsage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract organization ID from URL
	orgID := extractOrgID(r.URL.Path)
	if orgID == "" {
		respondError(w, apperrors.New("organization ID required", http.StatusBadRequest))
		return
	}

	// Only platform admins can reset usage
	role := middleware.GetRole(r)
	if role != "admin" && role != "platform_admin" {
		respondError(w, apperrors.New("only platform admins can reset monthly usage", http.StatusForbidden))
		return
	}

	quotaRepo := h.quotaRepo()
	if err := quotaRepo.ResetMonthlyUsage(ctx, orgID); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to reset monthly usage", http.StatusInternalServerError))
		return
	}

	// Log the reset
	if h.auditLogger != nil {
		userID := middleware.GetUserID(r)
		userEmail := middleware.GetEmail(r)
		h.auditLogger.LogUpdate(ctx, userID, userEmail, "quota_usage_reset", orgID, orgID, nil)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "monthly usage reset successfully",
	})
}

// Helper functions

// quotaRepo returns the quota repository.
// In production, this would be initialized in the Handler struct.
func (h *Handler) quotaRepo() *repository.QuotaRepository {
	// Placeholder - in actual implementation, Handler would have quotaRepo as a field
	return nil
}

// quotaConfigToResponse converts a quota config to its response format.
func quotaConfigToResponse(config *repository.QuotaConfig) *QuotaConfigResponse {
	return &QuotaConfigResponse{
		TenantID:        config.TenantID,
		MaxPrinters:     config.MaxPrinters,
		MaxStorageGB:    config.MaxStorageGB,
		MaxJobsPerMonth: config.MaxJobsPerMonth,
		MaxUsers:        config.MaxUsers,
		AlertThreshold:  config.AlertThreshold,
	}
}

// quotaUsageToResponse converts quota config and usage to the usage response format.
func quotaUsageToResponse(config *repository.QuotaConfig, usage *repository.QuotaUsage) *QuotaUsageResponse {
	return &QuotaUsageResponse{
		TenantID:  usage.TenantID,
		Printers:  quotaStat(int64(usage.PrintersCount), int64(config.MaxPrinters), config.AlertThreshold),
		Storage:   quotaStat(usage.StorageUsedGB, int64(config.MaxStorageGB)*1024*1024*1024, config.AlertThreshold),
		Jobs:      quotaStat(int64(usage.JobsThisMonth), int64(config.MaxJobsPerMonth), config.AlertThreshold),
		Users:     quotaStat(int64(usage.UsersCount), int64(config.MaxUsers), config.AlertThreshold),
		Month:     usage.Month.Format("2006-01"),
		UpdatedAt: usage.UpdatedAt,
	}
}

// quotaStat calculates usage statistics for a single quota type.
func quotaStat(current, maximum int64, alertThreshold int32) QuotaStat {
	var usagePercent float64
	if maximum > 0 {
		usagePercent = float64(current) / float64(maximum) * 100
	}

	remaining := maximum - current
	if remaining < 0 {
		remaining = 0
	}

	return QuotaStat{
		Current:      current,
		Maximum:      maximum,
		UsagePercent: usagePercent,
		Remaining:    remaining,
		IsNearLimit:  maximum > 0 && float64(alertThreshold) > 0 && usagePercent >= float64(alertThreshold),
	}
}

// validateQuotaConfigRequest validates the quota configuration request.
func validateQuotaConfigRequest(req *QuotaConfigRequest) error {
	if req.MaxPrinters < 0 {
		return fmt.Errorf("max_printers cannot be negative")
	}
	if req.MaxStorageGB < 0 {
		return fmt.Errorf("max_storage_gb cannot be negative")
	}
	if req.MaxJobsPerMonth < 0 {
		return fmt.Errorf("max_jobs_per_month cannot be negative")
	}
	if req.MaxUsers < 0 {
		return fmt.Errorf("max_users cannot be negative")
	}
	if req.AlertThreshold < 0 || req.AlertThreshold > 100 {
		return fmt.Errorf("alert_threshold must be between 0 and 100")
	}
	return nil
}
