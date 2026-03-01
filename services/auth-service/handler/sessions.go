// Package handler provides HTTP handlers for session management with httpOnly cookies.
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/openprint/openprint/internal/auth/jwt"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
)

const (
	// SessionCookieName is the name of the session cookie
	SessionCookieName = "openprint_session"
	// RefreshCookieName is the name of the refresh token cookie
	RefreshCookieName = "openprint_refresh"
	// DefaultSessionDuration is the default session duration
	DefaultSessionDuration = 24 * time.Hour
)

// LoginWithCookie handles user login and sets httpOnly cookies.
func (h *Handler) LoginWithCookie(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit request body size to prevent DoS
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req CookieLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate input
	loginReq := LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	}
	if err := validateLoginRequest(&loginReq); err != nil {
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
	sessionDuration := DefaultSessionDuration
	if req.RememberMe {
		sessionDuration = 30 * 24 * time.Hour // 30 days
	}

	if err := h.sessionRepo.Store(ctx, user.ID, refreshToken, sessionDuration); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to store session", http.StatusInternalServerError))
		return
	}

	// Set httpOnly cookies
	setSessionCookies(w, accessToken, refreshToken, sessionDuration, req.RememberMe)

	// Return user info (tokens are in cookies)
	name := fmt.Sprintf("%s %s", user.FirstName, user.LastName)
	if user.FirstName == "" && user.LastName == "" {
		name = user.Email
	}

	respondJSON(w, http.StatusOK, CookieLoginResponse{
		UserID:    user.ID,
		Email:     user.Email,
		Name:      name,
		Role:      user.Role,
		ExpiresIn: int64(sessionDuration / time.Second),
	})
}

// RefreshTokenHandler handles refreshing tokens using cookies.
func (h *Handler) RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get refresh token from cookie
	refreshCookie, err := r.Cookie(RefreshCookieName)
	if err != nil {
		respondError(w, apperrors.New("refresh token not found", http.StatusUnauthorized))
		return
	}

	refreshToken := refreshCookie.Value

	// Validate refresh token
	claims, err := h.jwtManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		// Clear invalid cookies
		clearSessionCookies(w)
		respondError(w, apperrors.New("invalid refresh token", http.StatusUnauthorized))
		return
	}

	// Check if session exists by getting the user ID
	userID, err := h.sessionRepo.GetUserID(ctx, refreshToken)
	if err != nil || userID == "" {
		clearSessionCookies(w)
		respondError(w, apperrors.New("session not found", http.StatusUnauthorized))
		return
	}

	// Generate new tokens
	scopes := jwt.DefaultScopes()
	if claims.Role == "admin" {
		scopes = jwt.AdminScopes()
	}

	accessToken, newRefreshToken, err := h.jwtManager.GenerateTokenPair(
		claims.UserID,
		claims.Email,
		claims.Role,
		claims.OrgID,
		scopes,
	)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to generate tokens", http.StatusInternalServerError))
		return
	}

	// Delete old session and store new one
	_ = h.sessionRepo.Delete(ctx, refreshToken)
	sessionDuration := DefaultSessionDuration

	if err := h.sessionRepo.Store(ctx, userID, newRefreshToken, sessionDuration); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to update session", http.StatusInternalServerError))
		return
	}

	// Set new cookies
	setSessionCookies(w, accessToken, newRefreshToken, sessionDuration, false)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"access_token": accessToken,
		"expires_in":   int64(15 * time.Minute / time.Second),
	})
}

// LogoutHandler handles user logout and clears cookies.
func (h *Handler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get refresh token from cookie
	refreshCookie, err := r.Cookie(RefreshCookieName)
	if err == nil {
		// Delete session from database
		_ = h.sessionRepo.Delete(ctx, refreshCookie.Value)
	}

	// Clear cookies
	clearSessionCookies(w)

	w.WriteHeader(http.StatusNoContent)
}

// SessionHandler handles getting current session info.
func (h *Handler) SessionHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user info from context (set by auth middleware)
	userID := ctx.Value("user_id")
	email := ctx.Value("email")
	role := ctx.Value("role")

	if userID == nil || email == nil {
		respondError(w, apperrors.New("not authenticated", http.StatusUnauthorized))
		return
	}

	// Get full user info
	user, err := h.userRepo.FindByID(ctx, fmt.Sprintf("%v", userID))
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get user info", http.StatusInternalServerError))
		return
	}

	name := fmt.Sprintf("%s %s", user.FirstName, user.LastName)
	if user.FirstName == "" && user.LastName == "" {
		name = user.Email
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":       user.ID,
		"email":         user.Email,
		"name":          name,
		"role":          role,
		"organization_id": user.GetOrgID(),
		"is_active":     user.IsActive,
	})
}

// SessionsListHandler handles listing all active sessions for a user.
func (h *Handler) SessionsListHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := ctx.Value("user_id")
	if userID == nil {
		respondError(w, apperrors.New("not authenticated", http.StatusUnauthorized))
		return
	}

	// Get session tokens for the user
	sessionTokens, err := h.sessionRepo.ListUserSessions(ctx, fmt.Sprintf("%v", userID))
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get sessions", http.StatusInternalServerError))
		return
	}

	response := make([]map[string]interface{}, len(sessionTokens))
	currentToken := getCurrentRefreshToken(r)
	for i, token := range sessionTokens {
		_, _ = h.sessionRepo.Get(ctx, token)
		response[i] = map[string]interface{}{
			"id":          token,
			"user_id":     userID,
			"created_at":  time.Now().Format(time.RFC3339),
			"expires_at":  time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			"is_current":  token == currentToken,
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"sessions": response,
		"count":    len(response),
	})
}

// SessionsRevokeHandler handles revoking a specific session.
func (h *Handler) SessionsRevokeHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := ctx.Value("user_id")
	if userID == nil {
		respondError(w, apperrors.New("not authenticated", http.StatusUnauthorized))
		return
	}

	// Extract session ID from path
	pathParts := parsePath(r.URL.Path)
	if len(pathParts) < 3 {
		respondError(w, apperrors.New("invalid session path", http.StatusBadRequest))
		return
	}
	sessionID := pathParts[2]

	// Get the session to verify ownership
	session, err := h.sessionRepo.Get(ctx, sessionID)
	if err != nil {
		respondError(w, apperrors.New("session not found", http.StatusNotFound))
		return
	}

	// Check ownership
	if session.UserID != fmt.Sprintf("%v", userID) {
		respondError(w, apperrors.New("forbidden", http.StatusForbidden))
		return
	}

	// Delete session
	if err := h.sessionRepo.Delete(ctx, sessionID); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to revoke session", http.StatusInternalServerError))
		return
	}

	// If this was the current session, clear cookies
	if session.Token == getCurrentRefreshToken(r) {
		clearSessionCookies(w)
	}

	w.WriteHeader(http.StatusNoContent)
}

// SessionsRevokeAllHandler handles revoking all sessions except current.
func (h *Handler) SessionsRevokeAllHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := ctx.Value("user_id")
	if userID == nil {
		respondError(w, apperrors.New("not authenticated", http.StatusUnauthorized))
		return
	}

	currentToken := getCurrentRefreshToken(r)

	// Get all sessions for the user and revoke except current
	sessionTokens, err := h.sessionRepo.ListUserSessions(ctx, fmt.Sprintf("%v", userID))
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get sessions", http.StatusInternalServerError))
		return
	}

	// Delete all sessions except current
	for _, token := range sessionTokens {
		if token != currentToken {
			_ = h.sessionRepo.Delete(ctx, token)
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "All other sessions revoked successfully",
	})
}

// ValidateSessionHandler handles validating a session token.
func (h *Handler) ValidateSessionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	if req.Token == "" {
		respondError(w, apperrors.New("token is required", http.StatusBadRequest))
		return
	}

	// Validate token
	claims, err := h.jwtManager.ValidateAccessToken(req.Token)
	if err != nil {
		respondError(w, apperrors.New("invalid token", http.StatusUnauthorized))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"valid":    true,
		"user_id":  claims.UserID,
		"email":    claims.Email,
		"role":     claims.Role,
		"org_id":   claims.OrgID,
		"expires":  claims.ExpiresAt.Format(time.RFC3339),
	})
}

// CookieLoginResponse represents the response for cookie-based login.
type CookieLoginResponse struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	ExpiresIn int64  `json:"expires_in"`
}

// CookieLoginRequest is extended for cookie-based login.
type CookieLoginRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	RememberMe bool   `json:"remember_me,omitempty"`
}

// Helper functions

// setSessionCookies sets httpOnly session cookies.
func setSessionCookies(w http.ResponseWriter, accessToken, refreshToken string, duration time.Duration, rememberMe bool) {
	// Set access token cookie (shorter duration)
	accessMaxAge := int(15 * time.Minute / time.Second)
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    accessToken,
		MaxAge:   accessMaxAge,
		Path:     "/",
		Secure:   true,   // HTTPS only
		HttpOnly: true,   // Not accessible via JavaScript
		SameSite: http.SameSiteLaxMode,
	})

	// Set refresh token cookie
	refreshMaxAge := int(duration / time.Second)
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshCookieName,
		Value:    refreshToken,
		MaxAge:   refreshMaxAge,
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// clearSessionCookies clears session cookies.
func clearSessionCookies(w http.ResponseWriter) {
	// Clear access token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// Clear refresh token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshCookieName,
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// getCurrentRefreshToken gets the current refresh token from cookies.
func getCurrentRefreshToken(r *http.Request) string {
	refreshCookie, err := r.Cookie(RefreshCookieName)
	if err != nil {
		return ""
	}
	return refreshCookie.Value
}

// parsePath splits URL path into components.
func parsePath(path string) []string {
	parts := make([]string, 0)
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
