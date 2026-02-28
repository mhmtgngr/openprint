// Package handler provides HTTP handlers for the auth service endpoints.
package handler

import (
	"net/http"
	"strings"

	apperrors "github.com/openprint/openprint/internal/shared/errors"
)

// GetCurrentUser retrieves the current authenticated user.
func (h *Handler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract and validate token
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		respondError(w, apperrors.ErrUnauthorized)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := h.jwtManager.ValidateAccessToken(tokenString)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "invalid token", http.StatusUnauthorized))
		return
	}

	// Get user
	user, err := h.userRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	respondJSON(w, http.StatusOK, userToResponse(user))
}
