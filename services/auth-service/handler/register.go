// Package handler provides HTTP handlers for the auth service endpoints.
package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/internal/auth/password"
	"github.com/openprint/openprint/internal/auth/jwt"
	"github.com/openprint/openprint/services/auth-service/repository"
)

// Register handles user registration.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate request
	if req.Email == "" || req.Password == "" {
		respondError(w, apperrors.New("email and password are required", http.StatusBadRequest))
		return
	}

	// Check password strength
	strengthResult := password.DefaultStrengthChecker().Check(req.Password)
	if !strengthResult.Valid {
		respondError(w, apperrors.NewValidationError("password", strings.Join(strengthResult.Errors, ", ")))
		return
	}

	// Check if user exists
	existing, err := h.userRepo.FindByEmail(ctx, req.Email)
	if err == nil && existing != nil {
		respondError(w, apperrors.New("user already exists", http.StatusConflict))
		return
	}

	// Validate organization_id if provided
	if req.OrganizationID != "" {
		hasAccess, err := h.userRepo.ValidateOrganizationAccess(ctx, req.OrganizationID, req.Email)
		if err != nil {
			respondError(w, apperrors.Wrap(err, "failed to validate organization access", http.StatusInternalServerError))
			return
		}
		if !hasAccess {
			respondError(w, apperrors.New("you do not have permission to join this organization", http.StatusForbidden))
			return
		}
	}

	// Hash password
	hashedPassword, err := h.passwordHasher.Generate(req.Password)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to hash password", http.StatusInternalServerError))
		return
	}

	// Create user
	user := &repository.User{
		ID:        uuid.New().String(),
		Email:     req.Email,
		Password:  hashedPassword,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Role:      "user",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if req.OrganizationID != "" {
		user.OrganizationID = &req.OrganizationID
	}

	if err := h.userRepo.Create(ctx, user); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to create user", http.StatusInternalServerError))
		return
	}

	// Generate tokens
	accessToken, refreshToken, err := h.jwtManager.GenerateTokenPair(
		user.ID,
		user.Email,
		user.Role,
		user.GetOrgID(),
		jwt.DefaultScopes(),
	)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to generate tokens", http.StatusInternalServerError))
		return
	}

	// Store refresh token
	if err := h.sessionRepo.Store(ctx, user.ID, refreshToken, 7*24*time.Hour); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to store session", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusCreated, RegisterResponse{
		UserID:       user.ID,
		Email:        user.Email,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}
