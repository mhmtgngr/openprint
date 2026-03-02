// Package handler provides HTTP handlers for the auth service endpoints.
// This file contains organization user management handlers.
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
	"github.com/openprint/openprint/services/auth-service/repository"
)

const (
	defaultMemberListLimit = 50
	maxMemberListLimit     = 100
)

// AddOrganizationUserRequest represents a request to add a user to an organization.
type AddOrganizationUserRequest struct {
	UserID    string                       `json:"user_id"`
	Email     string                       `json:"email,omitempty"`
	Role      repository.OrganizationUserRole `json:"role"`
	Settings  map[string]interface{}       `json:"settings,omitempty"`
}

// UpdateOrganizationUserRoleRequest represents a request to update a user's role.
type UpdateOrganizationUserRoleRequest struct {
	Role repository.OrganizationUserRole `json:"role"`
}

// OrganizationUserResponse represents an organization user response.
type OrganizationUserResponse struct {
	ID         string                       `json:"id"`
	UserID     string                       `json:"user_id"`
	Email      string                       `json:"email,omitempty"`
	Name       string                       `json:"name,omitempty"`
	Role       string                       `json:"role"`
	Settings   map[string]interface{}       `json:"settings,omitempty"`
	JoinedAt   time.Time                    `json:"joined_at"`
	InvitedBy  string                       `json:"invited_by,omitempty"`
}

// OrganizationUserListResponse represents a paginated list of organization members.
type OrganizationUserListResponse struct {
	Members  []*OrganizationUserResponse `json:"members"`
	Total    int                         `json:"total"`
	Limit    int                         `json:"limit"`
	Offset   int                         `json:"offset"`
}

// ListOrganizationUsers handles listing users in an organization.
func (h *Handler) ListOrganizationUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract organization ID from URL
	orgID := extractOrgID(r.URL.Path)
	if orgID == "" {
		respondError(w, apperrors.New("organization ID required", http.StatusBadRequest))
		return
	}

	// Check authorization - must be org member
	if !canAccessOrganization(ctx, r, orgID) {
		respondError(w, apperrors.New("forbidden: cannot access this organization", http.StatusForbidden))
		return
	}

	// Parse pagination parameters
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 {
		limit = defaultMemberListLimit
	}
	if limit > maxMemberListLimit {
		limit = maxMemberListLimit
	}

	orgUserRepo := h.orgUserRepo()
	users, total, err := orgUserRepo.ListByOrganization(ctx, orgID, limit, offset)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to list organization users", http.StatusInternalServerError))
		return
	}

	response := &OrganizationUserListResponse{
		Members: make([]*OrganizationUserResponse, len(users)),
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	}

	for i, user := range users {
		response.Members[i] = organizationUserToResponse(user)
	}

	respondJSON(w, http.StatusOK, response)
}

// GetOrganizationUser handles retrieving a specific organization user.
func (h *Handler) GetOrganizationUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract organization ID and user ID from URL
	orgID := extractOrgID(r.URL.Path)
	if orgID == "" {
		respondError(w, apperrors.New("organization ID required", http.StatusBadRequest))
		return
	}

	// Extract user ID from URL path
	// Expected format: /organizations/{org_id}/users/{user_id}
	pathParts := splitPath(r.URL.Path)
	userID := ""
	for i, part := range pathParts {
		if part == "users" && i+1 < len(pathParts) {
			userID = pathParts[i+1]
			break
		}
	}

	if userID == "" {
		respondError(w, apperrors.New("user ID required", http.StatusBadRequest))
		return
	}

	// Check authorization
	if !canAccessOrganization(ctx, r, orgID) {
		respondError(w, apperrors.New("forbidden: cannot access this organization", http.StatusForbidden))
		return
	}

	orgUserRepo := h.orgUserRepo()
	orgUser, err := orgUserRepo.GetByOrganizationAndUser(ctx, orgID, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			respondError(w, apperrors.ErrNotFound)
			return
		}
		respondError(w, apperrors.Wrap(err, "failed to get organization user", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, organizationUserToResponse(orgUser))
}

// AddOrganizationUser handles adding a user to an organization (org admin only).
func (h *Handler) AddOrganizationUser(w http.ResponseWriter, r *http.Request) {
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

	// Check authorization - must be org admin
	if !canManageOrganization(ctx, r, orgID) {
		respondError(w, apperrors.New("forbidden: cannot manage users in this organization", http.StatusForbidden))
		return
	}

	var req AddOrganizationUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate request
	if err := validateAddOrganizationUserRequest(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid input", http.StatusBadRequest))
		return
	}

	// If email provided, look up user ID
	if req.UserID == "" && req.Email != "" {
		userRepo := h.userRepo
		if userRepo != nil {
			// This would need a FindByEmail method on the user repository
			// For now, require user_id
		}
	}

	if req.UserID == "" {
		respondError(w, apperrors.New("user_id is required", http.StatusBadRequest))
		return
	}

	// Check if user exists and is not already a member
	orgUserRepo := h.orgUserRepo()
	isMember, err := orgUserRepo.IsMember(ctx, orgID, req.UserID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to check membership", http.StatusInternalServerError))
		return
	}
	if isMember {
		respondError(w, apperrors.New("user is already a member of this organization", http.StatusConflict))
		return
	}

	// Check user quota
	quotaRepo := h.quotaRepo()
	config, _ := quotaRepo.GetConfig(ctx, orgID)
	memberCount, _ := orgUserRepo.GetMemberCount(ctx, orgID)

	if config.MaxUsers > 0 && int32(memberCount) >= config.MaxUsers {
		respondError(w, apperrors.New("organization has reached maximum user limit", http.StatusForbidden))
		return
	}

	// Add user to organization
	orgUser := &repository.OrganizationUser{
		OrganizationID: orgID,
		UserID:         req.UserID,
		Role:           req.Role,
		Settings:       req.Settings,
		InvitedBy:      middleware.GetUserID(r),
	}

	if err := orgUserRepo.Add(ctx, orgUser); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to add user to organization", http.StatusInternalServerError))
		return
	}

	// Log the addition
	if h.auditLogger != nil {
		userID := middleware.GetUserID(r)
		userEmail := middleware.GetEmail(r)
		h.auditLogger.LogCreate(ctx, userID, userEmail, "organization_user", orgUser.ID, req.UserID,
			map[string]interface{}{"role": req.Role, "organization_id": orgID})
	}

	respondJSON(w, http.StatusCreated, organizationUserToResponse(orgUser))
}

// UpdateOrganizationUserRole handles updating a user's role in an organization.
func (h *Handler) UpdateOrganizationUserRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract organization ID and user ID from URL
	orgID := extractOrgID(r.URL.Path)
	if orgID == "" {
		respondError(w, apperrors.New("organization ID required", http.StatusBadRequest))
		return
	}

	pathParts := splitPath(r.URL.Path)
	userID := ""
	for i, part := range pathParts {
		if part == "users" && i+1 < len(pathParts) {
			userID = pathParts[i+1]
			break
		}
	}

	if userID == "" {
		respondError(w, apperrors.New("user ID required", http.StatusBadRequest))
		return
	}

	// Check authorization - must be org owner or admin
	if !canManageOrganization(ctx, r, orgID) {
		respondError(w, apperrors.New("forbidden: cannot manage users in this organization", http.StatusForbidden))
		return
	}

	var req UpdateOrganizationUserRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate role
	if !isValidOrgRole(req.Role) {
		respondError(w, apperrors.New("invalid role", http.StatusBadRequest))
		return
	}

	orgUserRepo := h.orgUserRepo()
	if err := orgUserRepo.UpdateRole(ctx, orgID, userID, req.Role); err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			respondError(w, apperrors.ErrNotFound)
			return
		}
		respondError(w, apperrors.Wrap(err, "failed to update user role", http.StatusInternalServerError))
		return
	}

	// Log the role change
	if h.auditLogger != nil {
		currentUserID := middleware.GetUserID(r)
		currentUserEmail := middleware.GetEmail(r)
		h.auditLogger.LogPermissionChange(ctx, currentUserID, currentUserEmail, userID, string(req.Role), true)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"user_id": userID,
		"role":    req.Role,
	})
}

// RemoveOrganizationUser handles removing a user from an organization.
func (h *Handler) RemoveOrganizationUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract organization ID and user ID from URL
	orgID := extractOrgID(r.URL.Path)
	if orgID == "" {
		respondError(w, apperrors.New("organization ID required", http.StatusBadRequest))
		return
	}

	pathParts := splitPath(r.URL.Path)
	userID := ""
	for i, part := range pathParts {
		if part == "users" && i+1 < len(pathParts) {
			userID = pathParts[i+1]
			break
		}
	}

	if userID == "" {
		respondError(w, apperrors.New("user ID required", http.StatusBadRequest))
		return
	}

	// Check authorization - org admin can remove, users can remove themselves
	currentUserID := middleware.GetUserID(r)
	role := middleware.GetRole(r)
	isOwnerOrAdmin := role == "admin" || role == "platform_admin" || role == "org_admin"
	isSelf := currentUserID == userID

	if !isOwnerOrAdmin && !isSelf {
		respondError(w, apperrors.New("forbidden: cannot remove this user", http.StatusForbidden))
		return
	}

	// Check if removing the last owner
	orgUserRepo := h.orgUserRepo()
	owners, err := orgUserRepo.GetOwners(ctx, orgID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to check ownership", http.StatusInternalServerError))
		return
	}

	if len(owners) == 1 && isSelf {
		respondError(w, apperrors.New("cannot remove the last owner from an organization", http.StatusForbidden))
		return
	}

	if err := orgUserRepo.Remove(ctx, orgID, userID); err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			respondError(w, apperrors.ErrNotFound)
			return
		}
		respondError(w, apperrors.Wrap(err, "failed to remove user from organization", http.StatusInternalServerError))
		return
	}

	// Log the removal
	if h.auditLogger != nil {
		currentUserEmail := middleware.GetEmail(r)
		h.auditLogger.LogDelete(ctx, currentUserID, currentUserEmail, "organization_user", userID, userID,
			map[string]interface{}{"organization_id": orgID})
	}

	respondJSON(w, http.StatusNoContent, nil)
}

// TransferOrganizationOwnership handles transferring organization ownership.
func (h *Handler) TransferOrganizationOwnership(w http.ResponseWriter, r *http.Request) {
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

	// Only current owners can transfer ownership
	currentUserID := middleware.GetUserID(r)
	orgUserRepo := h.orgUserRepo()

	isOwner, err := orgUserRepo.HasRole(ctx, orgID, currentUserID, repository.OrgRoleOwner)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to verify ownership", http.StatusInternalServerError))
		return
	}
	if !isOwner {
		respondError(w, apperrors.New("only organization owners can transfer ownership", http.StatusForbidden))
		return
	}

	var req struct {
		ToUserID string `json:"to_user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	if req.ToUserID == "" {
		respondError(w, apperrors.New("to_user_id is required", http.StatusBadRequest))
		return
	}

	if req.ToUserID == currentUserID {
		respondError(w, apperrors.New("cannot transfer ownership to yourself", http.StatusBadRequest))
		return
	}

	// Verify target user is a member
	isMember, err := orgUserRepo.IsMember(ctx, orgID, req.ToUserID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to check membership", http.StatusInternalServerError))
		return
	}
	if !isMember {
		respondError(w, apperrors.New("target user is not a member of this organization", http.StatusBadRequest))
		return
	}

	if err := orgUserRepo.TransferOwnership(ctx, orgID, currentUserID, req.ToUserID); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to transfer ownership", http.StatusInternalServerError))
		return
	}

	// Log the transfer
	if h.auditLogger != nil {
		currentUserEmail := middleware.GetEmail(r)
		h.auditLogger.LogUpdate(ctx, currentUserID, currentUserUserEmail, "organization_ownership", orgID, orgID,
			map[string]interface{}{"from_user_id": currentUserID, "to_user_id": req.ToUserID})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "ownership transferred successfully",
		"from_user_id": currentUserID,
		"to_user_id":   req.ToUserID,
	})
}

// Helper functions

// orgUserRepo returns the organization user repository.
func (h *Handler) orgUserRepo() *repository.OrganizationUserRepository {
	// Placeholder - in actual implementation, Handler would have orgUserRepo as a field
	return nil
}

// organizationUserToResponse converts an organization user to its response format.
func organizationUserToResponse(orgUser *repository.OrganizationUser) *OrganizationUserResponse {
	return &OrganizationUserResponse{
		ID:        orgUser.ID,
		UserID:    orgUser.UserID,
		Role:      string(orgUser.Role),
		Settings:  orgUser.Settings,
		JoinedAt:  orgUser.JoinedAt,
		InvitedBy: orgUser.InvitedBy,
	}
}

// splitPath splits a URL path into components.
func splitPath(path string) []string {
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return []string{}
	}
	return strings.Split(path, "/")
}

// validateAddOrganizationUserRequest validates the add organization user request.
func validateAddOrganizationUserRequest(req *AddOrganizationUserRequest) error {
	if req.UserID == "" && req.Email == "" {
		return fmt.Errorf("either user_id or email must be provided")
	}
	if !isValidOrgRole(req.Role) {
		return fmt.Errorf("invalid role")
	}
	return nil
}

// isValidOrgRole checks if a role is a valid organization role.
func isValidOrgRole(role repository.OrganizationUserRole) bool {
	switch role {
	case repository.OrgRoleOwner, repository.OrgRoleAdmin, repository.OrgRoleMember, repository.OrgRoleViewer, repository.OrgRoleBilling:
		return true
	default:
		return false
	}
}
