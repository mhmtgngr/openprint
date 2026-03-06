// Package handler provides HTTP handlers for the organization permission endpoints.
package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/services/organization-service/repository"
)

// CreatePermissionRequest represents a create permission request.
type CreatePermissionRequest struct {
	UserID         string `json:"user_id"`
	PermissionType string `json:"permission_type"` // admin, member, billing
}

// PermissionResponse represents a permission response.
type PermissionResponse struct {
	ID             string  `json:"id"`
	OrganizationID string  `json:"organization_id"`
	UserID         string  `json:"user_id"`
	PermissionType string  `json:"permission_type"`
	GrantedAt      string  `json:"granted_at"`
	GrantedBy      *string `json:"granted_by,omitempty"`
}

// PermissionsHandler handles permission-specific requests.
// Handles:
// - GET    /api/v1/organizations/:id/permissions - list permissions
// - POST   /api/v1/organizations/:id/permissions - create permission
// - DELETE /api/v1/organizations/:id/permissions/:user_id - revoke permission
func (h *Handler) PermissionsHandler(w http.ResponseWriter, r *http.Request, orgID string) {
	// Extract user_id from path for deletion
	// Path format: /api/v1/organizations/:id/permissions/:user_id
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/organizations/"+orgID+"/permissions")
	path = strings.TrimPrefix(path, "/")
	userID := strings.Split(path, "/")[0]

	switch r.Method {
	case http.MethodGet:
		h.ListPermissions(w, r, orgID)
	case http.MethodPost:
		h.CreatePermission(w, r, orgID)
	case http.MethodDelete:
		if userID == "" {
			respondError(w, apperrors.New("user_id is required for permission revocation", http.StatusBadRequest))
			return
		}
		h.RevokePermission(w, r, orgID, userID)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// ListPermissions handles GET /api/v1/organizations/:id/permissions - list permissions.
func (h *Handler) ListPermissions(w http.ResponseWriter, r *http.Request, orgID string) {
	ctx := r.Context()

	permissions, err := h.orgRepo.ListPermissions(ctx, orgID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to list permissions", http.StatusInternalServerError))
		return
	}

	response := make([]PermissionResponse, len(permissions))
	for i, perm := range permissions {
		response[i] = permissionToResponse(perm)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"permissions": response,
		"count":       len(response),
	})
}

// CreatePermission handles POST /api/v1/organizations/:id/permissions - create permission.
func (h *Handler) CreatePermission(w http.ResponseWriter, r *http.Request, orgID string) {
	ctx := r.Context()

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req CreatePermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate request
	if req.UserID == "" {
		respondError(w, apperrors.NewValidationError("user_id", "user_id is required"))
		return
	}
	if req.PermissionType == "" {
		req.PermissionType = "member" // default
	}

	// Validate permission type
	validTypes := map[string]bool{
		"admin":   true,
		"member":  true,
		"billing": true,
	}
	if !validTypes[req.PermissionType] {
		respondError(w, apperrors.NewValidationError("permission_type", "invalid permission type. Must be admin, member, or billing"))
		return
	}

	// Create permission
	permission := &repository.Permission{
		ID:             generateUUID(),
		OrganizationID: orgID,
		UserID:         req.UserID,
		PermissionType: req.PermissionType,
	}

	if err := h.orgRepo.AddPermission(ctx, permission); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to create permission", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusCreated, permissionToResponse(permission))
}

// RevokePermission handles DELETE /api/v1/organizations/:id/permissions/:user_id - revoke permission.
func (h *Handler) RevokePermission(w http.ResponseWriter, r *http.Request, orgID, userID string) {
	ctx := r.Context()

	if err := h.orgRepo.RemovePermission(ctx, orgID, userID); err != nil {
		if apperrors.IsNotFound(err) {
			respondError(w, apperrors.New("permission not found", http.StatusNotFound))
			return
		}
		respondError(w, apperrors.Wrap(err, "failed to revoke permission", http.StatusInternalServerError))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func permissionToResponse(perm *repository.Permission) PermissionResponse {
	return PermissionResponse{
		ID:             perm.ID,
		OrganizationID: perm.OrganizationID,
		UserID:         perm.UserID,
		PermissionType: perm.PermissionType,
		GrantedAt:      perm.GrantedAt.Format("2006-01-02T15:04:05Z07:00"),
		GrantedBy:      perm.GrantedBy,
	}
}

func generateUUID() string {
	return uuid.New().String()
}
