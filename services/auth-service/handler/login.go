// Package handler provides HTTP handlers for the auth service endpoints.
package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/openprint/openprint/internal/auth/jwt"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
)

const (
	// Max request body size to prevent DoS attacks (1MB)
	maxRequestBodySize = 1 << 20 // 1MB
)

// Login handles user login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit request body size to prevent DoS
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate input
	if err := validateLoginRequest(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid input", http.StatusBadRequest))
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
	if err := h.sessionRepo.Store(ctx, user.ID, refreshToken, jwt.MaxRefreshDuration); err != nil {
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

// validateLoginRequest validates the login request input.
func validateLoginRequest(req *LoginRequest) error {
	if req.Email == "" {
		return fmt.Errorf("email is required")
	}
	if len(req.Email) > 254 {
		return fmt.Errorf("email exceeds maximum length")
	}
	if req.Password == "" {
		return fmt.Errorf("password is required")
	}
	// Limit password length to prevent DoS
	if len(req.Password) > 1024 {
		return fmt.Errorf("password exceeds maximum length")
	}
	return nil
}

// SafeErrorResponse returns a safe error response that doesn't leak information.
func SafeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"code":    http.StatusText(statusCode),
		"message": message,
	})
}

// CloseTracker tracks response body closure for security logging.
type CloseTracker struct {
	io.ReadCloser
	closed bool
}

func (ct *CloseTracker) Close() error {
	ct.closed = true
	return ct.ReadCloser.Close()
}
