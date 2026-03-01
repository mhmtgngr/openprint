// Package handler provides HTTP handlers for the auth service endpoints.
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/internal/auth/password"
	"github.com/openprint/openprint/internal/auth/jwt"
	"github.com/openprint/openprint/services/auth-service/repository"
)

const (
	maxEmailLength    = 254
	maxPasswordLength = 128
	maxNameLength     = 100
)

// Register handles user registration.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate request
	if err := validateRegisterRequest(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid input", http.StatusBadRequest))
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
	if err := h.sessionRepo.Store(ctx, user.ID, refreshToken, jwt.MaxRefreshDuration); err != nil {
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

// validateRegisterRequest validates the registration request.
func validateRegisterRequest(req *RegisterRequest) error {
	if req.Email == "" {
		return fmt.Errorf("email is required")
	}
	if len(req.Email) > maxEmailLength {
		return fmt.Errorf("email exceeds maximum length")
	}
	if req.Password == "" {
		return fmt.Errorf("password is required")
	}
	if len(req.Password) > maxPasswordLength {
		return fmt.Errorf("password exceeds maximum length")
	}
	if len(req.FirstName) > maxNameLength {
		return fmt.Errorf("first name exceeds maximum length")
	}
	if len(req.LastName) > maxNameLength {
		return fmt.Errorf("last name exceeds maximum length")
	}
	// Check for potential SQL injection in name fields
	if containsControlCharacters(req.FirstName) || containsControlCharacters(req.LastName) {
		return fmt.Errorf("invalid characters in name")
	}
	return nil
}

// containsControlCharacters checks for control characters that could be used in injection attacks.
func containsControlCharacters(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return true
		}
	}
	return false
}
