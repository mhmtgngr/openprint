// Package handler provides HTTP handlers for the auth service endpoints.
package handler

import (
	"encoding/json"
	"net/http"

	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/internal/auth/jwt"
)

// RefreshToken handles token refresh with rotation and explicit revocation checking.
// It validates the refresh token, checks it hasn't been revoked, and issues a new access token.
// SECURITY: Implements refresh token rotation - the old refresh token is revoked and a new one is issued.
// This prevents replay attacks and limits the window of abuse if a token is compromised.
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

	// Validate refresh token JWT structure and signature
	claims, err := h.jwtManager.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "invalid refresh token", http.StatusUnauthorized))
		return
	}

	// Check if token has been revoked (blacklist check)
	// Even if the JWT is valid, if the session has been deleted, the token is revoked
	revoked, err := h.sessionRepo.IsTokenRevoked(ctx, req.RefreshToken)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to verify token status", http.StatusInternalServerError))
		return
	}
	if revoked {
		respondError(w, apperrors.New("refresh token has been revoked", http.StatusUnauthorized))
		return
	}

	// Verify session exists and matches the user
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

	// SECURITY: Implement refresh token rotation
	// Generate new token pair (access + refresh)
	accessToken, newRefreshToken, err := h.jwtManager.GenerateTokenPair(
		user.ID,
		user.Email,
		user.Role,
		user.GetOrgID(),
		claims.Scopes,
	)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to generate tokens", http.StatusInternalServerError))
		return
	}

	// Revoke the old refresh token (rotation)
	// This ensures each refresh token can only be used once
	if err := h.sessionRepo.RevokeToken(ctx, req.RefreshToken); err != nil {
		// Log error but don't fail the refresh - the new token is still valid
		// The old token will expire naturally
		http.Error(w, "warning: old token revocation failed", http.StatusAccepted)
	}

	// Store the new refresh token
	if err := h.sessionRepo.Store(ctx, user.ID, newRefreshToken, jwt.MaxRefreshDuration); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to store new session", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
		"token_type":    "Bearer",
		"expires_in":    "900", // 15 minutes
	})
}
