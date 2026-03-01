// Package handler provides HTTP handlers for the auth service endpoints.
package handler

import (
	"fmt"
	"net/http"
	"strings"

	apperrors "github.com/openprint/openprint/internal/shared/errors"
)

// Logout handles user logout by revoking all refresh tokens for the user.
// This implements token revocation by removing the tokens from the session store,
// effectively preventing any further use of those tokens for refreshing access tokens.
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

	// Revoke all sessions (refresh tokens) for this user
	// This implements the token blacklist mechanism - deleted tokens cannot be used
	// to refresh access tokens even if the JWT itself hasn't expired yet.
	if err := h.sessionRepo.RevokeUserTokens(ctx, claims.UserID); err != nil {
		// Log but don't fail logout - the access token will expire naturally
		fmt.Printf("Failed to revoke user tokens: %v", err)
	}

	w.WriteHeader(http.StatusNoContent)
}
