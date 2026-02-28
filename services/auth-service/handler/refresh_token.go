// Package handler provides HTTP handlers for the auth service endpoints.
package handler

import (
	"encoding/json"
	"net/http"

	apperrors "github.com/openprint/openprint/internal/shared/errors"
)

// RefreshToken handles token refresh.
func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	if req.RefreshToken == "" {
		respondError(w, apperrors.New("refresh_token is required", http.StatusBadRequest))
		return
	}

	// Validate refresh token
	claims, err := h.jwtManager.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "invalid refresh token", http.StatusUnauthorized))
		return
	}

	// Verify session exists
	userID, err := h.sessionRepo.GetUserID(ctx, req.RefreshToken)
	if err != nil || userID != claims.UserID {
		respondError(w, apperrors.ErrUnauthorized)
		return
	}

	// Get user to ensure they're still active
	user, err := h.userRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	if !user.IsActive {
		respondError(w, apperrors.New("account is disabled", http.StatusForbidden))
		return
	}

	// Generate new access token
	accessToken, err := h.jwtManager.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to refresh token", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   "900", // 15 minutes
	})
}
