// Package handler provides HTTP handlers for the organization service endpoints.
package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/services/organization-service/repository"
)

const (
	maxRequestBodySize = 1 << 20 // 1MB
	maxOrgNameLength   = 255
	maxSlugLength      = 100
)

// Config holds handler dependencies.
type Config struct {
	OrgRepo *repository.OrganizationRepository
}

// Handler provides organization service HTTP handlers.
type Handler struct {
	orgRepo *repository.OrganizationRepository
}

// New creates a new handler instance.
func New(cfg Config) *Handler {
	return &Handler{
		orgRepo: cfg.OrgRepo,
	}
}

// CreateOrganizationRequest represents a create organization request.
type CreateOrganizationRequest struct {
	Name    string                 `json:"name"`
	Slug    string                 `json:"slug"`
	Plan    string                 `json:"plan,omitempty"`
	Settings map[string]interface{} `json:"settings,omitempty"`
}

// UpdateOrganizationRequest represents an update organization request.
type UpdateOrganizationRequest struct {
	Name     string                 `json:"name,omitempty"`
	Plan     string                 `json:"plan,omitempty"`
	Settings map[string]interface{} `json:"settings,omitempty"`
}

// OrganizationResponse represents an organization response.
type OrganizationResponse struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Slug      string                 `json:"slug"`
	Plan      string                 `json:"plan"`
	Settings  map[string]interface{} `json:"settings"`
	CreatedAt string                 `json:"created_at"`
	UpdatedAt string                 `json:"updated_at"`
}

// ListOrganizations handles GET /api/v1/organizations - list all organizations.
func (h *Handler) ListOrganizations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgs, err := h.orgRepo.List(ctx)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to list organizations", http.StatusInternalServerError))
		return
	}

	response := make([]OrganizationResponse, len(orgs))
	for i, org := range orgs {
		response[i] = orgToResponse(org)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"organizations": response,
		"count":         len(response),
	})
}

// OrganizationHandler routes organization-specific requests.
// Handles:
// - POST   /api/v1/organizations - create organization
// - GET    /api/v1/organizations/:id - get organization
// - PUT    /api/v1/organizations/:id - update organization
// - DELETE /api/v1/organizations/:id - delete organization
func (h *Handler) OrganizationHandler(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/organizations/")
	path = strings.TrimPrefix(path, "/")

	parts := strings.Split(path, "/")
	orgID := parts[0]

	// Handle create with no ID (POST /api/v1/organizations)
	if orgID == "" && r.Method == http.MethodPost {
		h.CreateOrganization(w, r)
		return
	}

	if orgID == "" {
		respondError(w, apperrors.New("organization id is required", http.StatusBadRequest))
		return
	}

	// Validate UUID
	if _, err := uuid.Parse(orgID); err != nil {
		respondError(w, apperrors.New("invalid organization id", http.StatusBadRequest))
		return
	}

	// Handle sub-routes (permissions, members)
	if len(parts) > 1 {
		switch parts[1] {
		case "permissions":
			h.PermissionsHandler(w, r, orgID)
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		h.GetOrganization(w, r, orgID)
	case http.MethodPut:
		h.UpdateOrganization(w, r, orgID)
	case http.MethodDelete:
		h.DeleteOrganization(w, r, orgID)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// CreateOrganization handles POST /api/v1/organizations - create organization.
func (h *Handler) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req CreateOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate request
	if err := validateCreateOrganization(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid input", http.StatusBadRequest))
		return
	}

	// Create organization
	org := &repository.Organization{
		ID:      uuid.New().String(),
		Name:    req.Name,
		Slug:    req.Slug,
		Plan:    req.Plan,
		Settings: req.Settings,
	}

	if org.Plan == "" {
		org.Plan = "free"
	}

	if err := h.orgRepo.Create(ctx, org); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to create organization", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusCreated, orgToResponse(org))
}

// GetOrganization handles GET /api/v1/organizations/:id - get organization.
func (h *Handler) GetOrganization(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	org, err := h.orgRepo.FindByID(ctx, id)
	if err != nil {
		if apperrors.IsNotFound(err) {
			respondError(w, apperrors.New("organization not found", http.StatusNotFound))
			return
		}
		respondError(w, apperrors.Wrap(err, "failed to get organization", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, orgToResponse(org))
}

// UpdateOrganization handles PUT /api/v1/organizations/:id - update organization.
func (h *Handler) UpdateOrganization(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req UpdateOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Get existing organization
	org, err := h.orgRepo.FindByID(ctx, id)
	if err != nil {
		if apperrors.IsNotFound(err) {
			respondError(w, apperrors.New("organization not found", http.StatusNotFound))
			return
		}
		respondError(w, apperrors.Wrap(err, "failed to get organization", http.StatusInternalServerError))
		return
	}

	// Update fields
	if req.Name != "" {
		org.Name = req.Name
	}
	if req.Plan != "" {
		org.Plan = req.Plan
	}
	if req.Settings != nil {
		org.Settings = req.Settings
	}

	if err := h.orgRepo.Update(ctx, org); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to update organization", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, orgToResponse(org))
}

// DeleteOrganization handles DELETE /api/v1/organizations/:id - delete organization.
func (h *Handler) DeleteOrganization(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if err := h.orgRepo.Delete(ctx, id); err != nil {
		if apperrors.IsNotFound(err) {
			respondError(w, apperrors.New("organization not found", http.StatusNotFound))
			return
		}
		respondError(w, apperrors.Wrap(err, "failed to delete organization", http.StatusInternalServerError))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// validateCreateOrganization validates the create organization request.
func validateCreateOrganization(req *CreateOrganizationRequest) error {
	if req.Name == "" {
		return apperrors.NewValidationError("name", "name is required")
	}
	if len(req.Name) > maxOrgNameLength {
		return apperrors.NewValidationError("name", "name exceeds maximum length")
	}
	if req.Slug == "" {
		return apperrors.NewValidationError("slug", "slug is required")
	}
	if len(req.Slug) > maxSlugLength {
		return apperrors.NewValidationError("slug", "slug exceeds maximum length")
	}
	return nil
}

func orgToResponse(org *repository.Organization) OrganizationResponse {
	return OrganizationResponse{
		ID:        org.ID,
		Name:      org.Name,
		Slug:      org.Slug,
		Plan:      org.Plan,
		Settings:  org.Settings,
		CreatedAt: org.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: org.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, err error) {
	var appErr *apperrors.AppError
	if err, ok := err.(*apperrors.AppError); ok {
		appErr = err
	} else {
		// Try to convert using errors.As
		// For now, just wrap as a generic error
		appErr = apperrors.New(err.Error(), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.StatusCode)
	json.NewEncoder(w).Encode(apperrors.ToJSON(appErr))
}
