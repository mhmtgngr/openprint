// Package handler provides HTTP handlers for the auth service endpoints.
package handler

import (
	"fmt"
	"net/http"
	"strings"

	apperrors "github.com/openprint/openprint/internal/shared/errors"
)

// Logout handles user logout.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract token from authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		respondError(w, apperrors.ErrUnauthorized)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		respondError(w, apperrors.ErrUnauthorized)
		return
	}

	// Validate and extract user info from token
	claims, err := h.jwtManager.ValidateAccessToken(tokenString)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "invalid token", http.StatusUnauthorized))
		return
	}

	// Delete session (refresh token) from Redis
	if err := h.sessionRepo.DeleteByUserID(ctx, claims.UserID); err != nil {
		// Log but don't fail logout
		fmt.Printf("Failed to delete session: %v", err)
	}

	// Add token to blacklist (optional implementation)
	// In production, you'd want to cache this to prevent immediate reuse

	w.WriteHeader(http.StatusNoContent)
}
