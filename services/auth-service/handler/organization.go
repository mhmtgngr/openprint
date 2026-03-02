// Package handler provides HTTP handlers for the auth service endpoints.
// This file contains organization management handlers.
package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/internal/shared/middleware"
	"github.com/openprint/openprint/internal/multi-tenant"
	"github.com/openprint/openprint/services/auth-service/repository"
)

const (
	maxOrgNameLength     = 100
	maxOrgSlugLength     = 50
	maxOrgDescLength     = 500
	maxWebsiteLength     = 255
	maxLogoURLLength     = 500
	defaultOrgListLimit  = 50
	maxOrgListLimit      = 100
)

// CreateOrganizationRequest represents a request to create an organization.
type CreateOrganizationRequest struct {
	Name        string                 `json:"name"`
	Slug        string                 `json:"slug,omitempty"`
	LogoURL     string                 `json:"logo_url,omitempty"`
	Website     string                 `json:"website,omitempty"`
	Description string                 `json:"description,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
}

// UpdateOrganizationRequest represents a request to update an organization.
type UpdateOrganizationRequest struct {
	Name        string                 `json:"name,omitempty"`
	Slug        string                 `json:"slug,omitempty"`
	LogoURL     string                 `json:"logo_url,omitempty"`
	Website     string                 `json:"website,omitempty"`
	Description string                 `json:"description,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
}

// OrganizationResponse represents an organization response.
type OrganizationResponse struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Slug        string                 `json:"slug"`
	Status      string                 `json:"status"`
	LogoURL     string                 `json:"logo_url,omitempty"`
	Website     string                 `json:"website,omitempty"`
	Description string                 `json:"description,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// OrganizationListResponse represents a paginated list of organizations.
type OrganizationListResponse struct {
	Organizations []*OrganizationResponse `json:"organizations"`
	Total         int                     `json:"total"`
	Limit         int                     `json:"limit"`
	Offset        int                     `json:"offset"`
}

// CreateOrganization handles organization creation (platform admin only).
func (h *Handler) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Only platform admins can create organizations
	role := middleware.GetRole(r)
	if role != "admin" && role != "platform_admin" {
		respondError(w, apperrors.New("only platform admins can create organizations", http.StatusForbidden))
		return
	}

	var req CreateOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate request
	if err := validateCreateOrganizationRequest(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid input", http.StatusBadRequest))
		return
	}

	// Generate slug if not provided
	if req.Slug == "" {
		req.Slug = generateSlug(req.Name)
	} else {
		req.Slug = strings.ToLower(strings.TrimSpace(req.Slug))
		req.Slug = strings.ReplaceAll(req.Slug, " ", "-")
	}

	// Check slug availability
	orgRepo := h.orgRepo()
	exists, err := orgRepo.SlugExists(ctx, req.Slug, "")
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to check slug availability", http.StatusInternalServerError))
		return
	}
	if exists {
		respondError(w, apperrors.New("slug already in use", http.StatusConflict))
		return
	}

	// Create organization
	org := &repository.Organization{
		Name:        req.Name,
		Slug:        req.Slug,
		Status:      repository.OrgStatusActive,
		LogoURL:     req.LogoURL,
		Website:     req.Website,
		Description: req.Description,
		Settings:    req.Settings,
	}

	if err := orgRepo.Create(ctx, org); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to create organization", http.StatusInternalServerError))
		return
	}

	// Log the creation
	if h.auditLogger != nil {
		userID := middleware.GetUserID(r)
		userEmail := middleware.GetEmail(r)
		h.auditLogger.LogCreate(ctx, userID, userEmail, "organization", org.ID, org.Name, nil)
	}

	respondJSON(w, http.StatusCreated, organizationToResponse(org))
}

// GetOrganization handles retrieving a single organization.
func (h *Handler) GetOrganization(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract organization ID from URL
	orgID := strings.TrimPrefix(r.URL.Path, "/api/v1/organizations/")
	orgID = strings.TrimSuffix(orgID, "/")

	if orgID == "" {
		respondError(w, apperrors.New("organization ID required", http.StatusBadRequest))
		return
	}

	// Check authorization - platform admin or org member can view
	if !canAccessOrganization(ctx, r, orgID) {
		respondError(w, apperrors.New("forbidden: cannot access this organization", http.StatusForbidden))
		return
	}

	orgRepo := h.orgRepo()
	org, err := orgRepo.GetByID(ctx, orgID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			respondError(w, apperrors.ErrNotFound)
			return
		}
		respondError(w, apperrors.Wrap(err, "failed to get organization", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, organizationToResponse(org))
}

// ListOrganizations handles listing organizations with pagination.
func (h *Handler) ListOrganizations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	status := repository.OrganizationStatus(r.URL.Query().Get("status"))

	// Apply defaults
	if limit <= 0 {
		limit = defaultOrgListLimit
	}
	if limit > maxOrgListLimit {
		limit = maxOrgListLimit
	}

	orgRepo := h.orgRepo()

	// Platform admins can list all orgs, regular users can only see their own
	role := middleware.GetRole(r)
	var orgs []*repository.Organization
	var total int
	var err error

	if role == "admin" || role == "platform_admin" {
		orgs, total, err = orgRepo.List(ctx, limit, offset, status)
	} else {
		// Regular users can only see their own organization
		userID := middleware.GetUserID(r)
		org, _, getErr := orgRepo.GetUserOrganization(ctx, userID)
		if getErr != nil {
			// Return empty list if user has no organization
			orgs = []*repository.Organization{}
			total = 0
		} else {
			orgs = []*repository.Organization{org}
			total = 1
		}
		err = nil
	}

	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to list organizations", http.StatusInternalServerError))
		return
	}

	response := &OrganizationListResponse{
		Organizations: make([]*OrganizationResponse, len(orgs)),
		Total:         total,
		Limit:         limit,
		Offset:        offset,
	}

	for i, org := range orgs {
		response.Organizations[i] = organizationToResponse(org)
	}

	respondJSON(w, http.StatusOK, response)
}

// UpdateOrganization handles updating an organization.
func (h *Handler) UpdateOrganization(w http.ResponseWriter, r *http.Request) {
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

	// Check authorization - only platform admin or org admin can update
	if !canManageOrganization(ctx, r, orgID) {
		respondError(w, apperrors.New("forbidden: cannot manage this organization", http.StatusForbidden))
		return
	}

	var req UpdateOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate request
	if err := validateUpdateOrganizationRequest(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid input", http.StatusBadRequest))
		return
	}

	orgRepo := h.orgRepo()
	org, err := orgRepo.GetByID(ctx, orgID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get organization", http.StatusInternalServerError))
		return
	}

	// Update fields if provided
	if req.Name != "" {
		org.Name = req.Name
	}
	if req.Slug != "" {
		// Check new slug availability
		exists, err := orgRepo.SlugExists(ctx, req.Slug, orgID)
		if err != nil {
			respondError(w, apperrors.Wrap(err, "failed to check slug availability", http.StatusInternalServerError))
			return
		}
		if exists {
			respondError(w, apperrors.New("slug already in use", http.StatusConflict))
			return
		}
		org.Slug = req.Slug
	}
	if req.LogoURL != "" {
		org.LogoURL = req.LogoURL
	}
	if req.Website != "" {
		org.Website = req.Website
	}
	if req.Description != "" {
		org.Description = req.Description
	}
	if req.Settings != nil {
		org.Settings = req.Settings
	}

	if err := orgRepo.Update(ctx, org); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to update organization", http.StatusInternalServerError))
		return
	}

	// Log the update
	if h.auditLogger != nil {
		userID := middleware.GetUserID(r)
		userEmail := middleware.GetEmail(r)
		h.auditLogger.LogUpdate(ctx, userID, userEmail, "organization", org.ID, org.Name, nil)
	}

	respondJSON(w, http.StatusOK, organizationToResponse(org))
}

// UpdateOrganizationStatus handles updating an organization's status (platform admin only).
func (h *Handler) UpdateOrganizationStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Only platform admins can change organization status
	role := middleware.GetRole(r)
	if role != "admin" && role != "platform_admin" {
		respondError(w, apperrors.New("only platform admins can change organization status", http.StatusForbidden))
		return
	}

	// Extract organization ID from URL
	orgID := extractOrgID(r.URL.Path)
	if orgID == "" {
		respondError(w, apperrors.New("organization ID required", http.StatusBadRequest))
		return
	}

	var req struct {
		Status repository.OrganizationStatus `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate status
	if req.Status != repository.OrgStatusActive &&
		req.Status != repository.OrgStatusSuspended &&
		req.Status != repository.OrgStatusDeleted {
		respondError(w, apperrors.New("invalid status", http.StatusBadRequest))
		return
	}

	orgRepo := h.orgRepo()
	if err := orgRepo.UpdateStatus(ctx, orgID, req.Status); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to update organization status", http.StatusInternalServerError))
		return
	}

	// Log the status change
	if h.auditLogger != nil {
		userID := middleware.GetUserID(r)
		userEmail := middleware.GetEmail(r)
		h.auditLogger.LogUpdate(ctx, userID, userEmail, "organization_status", orgID, string(req.Status),
			map[string]interface{}{"new_status": req.Status})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":     orgID,
		"status": req.Status,
	})
}

// DeleteOrganization handles deleting an organization (platform admin only).
func (h *Handler) DeleteOrganization(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Only platform admins can delete organizations
	role := middleware.GetRole(r)
	if role != "admin" && role != "platform_admin" {
		respondError(w, apperrors.New("only platform admins can delete organizations", http.StatusForbidden))
		return
	}

	// Extract organization ID from URL
	orgID := extractOrgID(r.URL.Path)
	if orgID == "" {
		respondError(w, apperrors.New("organization ID required", http.StatusBadRequest))
		return
	}

	orgRepo := h.orgRepo()
	org, err := orgRepo.GetByID(ctx, orgID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get organization", http.StatusInternalServerError))
		return
	}

	if err := orgRepo.Delete(ctx, orgID); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to delete organization", http.StatusInternalServerError))
		return
	}

	// Log the deletion
	if h.auditLogger != nil {
		userID := middleware.GetUserID(r)
		userEmail := middleware.GetEmail(r)
		h.auditLogger.LogDelete(ctx, userID, userEmail, "organization", org.ID, org.Name, nil)
	}

	respondJSON(w, http.StatusNoContent, nil)
}

// Helper functions

// orgRepo returns the organization repository.
// In production, this would be initialized in the Handler struct.
func (h *Handler) orgRepo() *repository.OrganizationRepository {
	// This is a placeholder - in actual implementation,
	// the Handler would have orgRepo as a field initialized in New()
	// For now, we'll use a nil check approach
	return nil
}

// canAccessOrganization checks if the current user can access the organization.
func canAccessOrganization(ctx context.Context, r *http.Request, orgID string) bool {
	role := middleware.GetRole(r)
	if role == "admin" || role == "platform_admin" {
		return true
	}

	// Check if user is a member of the organization
	userOrgID := middleware.GetOrgID(r)
	return userOrgID == orgID
}

// canManageOrganization checks if the current user can manage the organization.
func canManageOrganization(ctx context.Context, r *http.Request, orgID string) bool {
	role := middleware.GetRole(r)
	if role == "admin" || role == "platform_admin" {
		return true
	}

	// Check if user is org admin
	userOrgID := middleware.GetOrgID(r)
	if userOrgID != orgID {
		return false
	}

	return role == "org_admin"
}

// organizationToResponse converts an organization to its response format.
func organizationToResponse(org *repository.Organization) *OrganizationResponse {
	return &OrganizationResponse{
		ID:          org.ID,
		Name:        org.Name,
		Slug:        org.Slug,
		Status:      string(org.Status),
		LogoURL:     org.LogoURL,
		Website:     org.Website,
		Description: org.Description,
		Settings:    org.Settings,
		CreatedAt:   org.CreatedAt,
		UpdatedAt:   org.UpdatedAt,
	}
}

// extractOrgID extracts the organization ID from the URL path.
func extractOrgID(path string) string {
	// Handle both /organizations/{id} and /organizations/{id}/*
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) >= 2 && parts[0] == "organizations" || parts[0] == "api" {
		// Find the organizations segment
		for i, part := range parts {
			if part == "organizations" && i+1 < len(parts) {
				return parts[i+1]
			}
		}
	}
	return ""
}

// validateCreateOrganizationRequest validates the create organization request.
func validateCreateOrganizationRequest(req *CreateOrganizationRequest) error {
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(req.Name) > maxOrgNameLength {
		return fmt.Errorf("name exceeds maximum length")
	}
	if req.Slug != "" {
		if len(req.Slug) > maxOrgSlugLength {
			return fmt.Errorf("slug exceeds maximum length")
		}
		if !isValidSlug(req.Slug) {
			return fmt.Errorf("slug contains invalid characters")
		}
	}
	if len(req.Description) > maxOrgDescLength {
		return fmt.Errorf("description exceeds maximum length")
	}
	if len(req.Website) > maxWebsiteLength {
		return fmt.Errorf("website exceeds maximum length")
	}
	if len(req.LogoURL) > maxLogoURLLength {
		return fmt.Errorf("logo URL exceeds maximum length")
	}
	return nil
}

// validateUpdateOrganizationRequest validates the update organization request.
func validateUpdateOrganizationRequest(req *UpdateOrganizationRequest) error {
	if req.Name != "" && len(req.Name) > maxOrgNameLength {
		return fmt.Errorf("name exceeds maximum length")
	}
	if req.Slug != "" {
		if len(req.Slug) > maxOrgSlugLength {
			return fmt.Errorf("slug exceeds maximum length")
		}
		if !isValidSlug(req.Slug) {
			return fmt.Errorf("slug contains invalid characters")
		}
	}
	if len(req.Description) > maxOrgDescLength {
		return fmt.Errorf("description exceeds maximum length")
	}
	if len(req.Website) > maxWebsiteLength {
		return fmt.Errorf("website exceeds maximum length")
	}
	if len(req.LogoURL) > maxLogoURLLength {
		return fmt.Errorf("logo URL exceeds maximum length")
	}
	return nil
}

// isValidSlug checks if a slug contains only valid characters.
func isValidSlug(slug string) bool {
	if len(slug) == 0 {
		return false
	}
	for _, c := range slug {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

// generateSlug generates a URL-friendly slug from a name.
func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")

	// Remove non-alphanumeric characters except hyphens
	result := ""
	for _, c := range slug {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			result += string(c)
		}
	}

	// Ensure uniqueness by adding UUID if needed
	if result == "" {
		result = "org-" + uuid.New().String()[:8]
	}

	return result
}
