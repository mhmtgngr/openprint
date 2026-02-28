// Package handler provides HTTP handlers for the auth service endpoints.
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/openprint/openprint/internal/auth/jwt"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
)

// Login handles user login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Find user
	user, err := h.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		respondError(w, apperrors.New("invalid credentials", http.StatusUnauthorized))
		return
	}

	// Verify password
	valid, err := h.passwordHasher.Verify(req.Password, user.Password)
	if err != nil || !valid {
		respondError(w, apperrors.New("invalid credentials", http.StatusUnauthorized))
		return
	}

	// Check if user is active
	if !user.IsActive {
		respondError(w, apperrors.New("account is disabled", http.StatusForbidden))
		return
	}

	// Update last login
	user.LastLoginAt = &[]time.Time{time.Now()}[0]
	if err := h.userRepo.Update(ctx, user); err != nil {
		// Log but don't fail login
		fmt.Printf("Failed to update last login: %v", err)
	}

	// Generate tokens
	scopes := jwt.DefaultScopes()
	if user.Role == "admin" {
		scopes = jwt.AdminScopes()
	}

	accessToken, refreshToken, err := h.jwtManager.GenerateTokenPair(
		user.ID,
		user.Email,
		user.Role,
		user.GetOrgID(),
		scopes,
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

	name := fmt.Sprintf("%s %s", user.FirstName, user.LastName)
	if user.FirstName == "" && user.LastName == "" {
		name = user.Email
	}

	respondJSON(w, http.StatusOK, LoginResponse{
		UserID:       user.ID,
		Email:        user.Email,
		Name:         name,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(15 * time.Minute / time.Second),
	})
}
