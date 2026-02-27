// Package handler provides HTTP handlers for the auth service endpoints.
package handler

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/openprint/openprint/internal/auth/jwt"
	"github.com/openprint/openprint/internal/auth/oidc"
	"github.com/openprint/openprint/internal/auth/password"
	"github.com/openprint/openprint/internal/auth/saml"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/services/auth-service/repository"
)

// Config holds handler dependencies.
type Config struct {
	UserRepo       *repository.UserRepository
	SessionRepo    *repository.SessionRepository
	JWTManager     *jwt.Manager
	PasswordHasher *password.Hasher
	OIDCRegistry   *oidc.Registry
	SAMLManager    *saml.Manager
}

// Handler provides auth service HTTP handlers.
type Handler struct {
	userRepo       *repository.UserRepository
	sessionRepo    *repository.SessionRepository
	jwtManager     *jwt.Manager
	passwordHasher *password.Hasher
	oidcRegistry   *oidc.Registry
	samlManager    *saml.Manager
}

// New creates a new handler instance.
func New(cfg Config) *Handler {
	return &Handler{
		userRepo:       cfg.UserRepo,
		sessionRepo:    cfg.SessionRepo,
		jwtManager:     cfg.JWTManager,
		passwordHasher: cfg.PasswordHasher,
		oidcRegistry:   cfg.OIDCRegistry,
		samlManager:    cfg.SAMLManager,
	}
}

// RegisterRequest represents a user registration request.
type RegisterRequest struct {
	Email           string `json:"email"`
	Password        string `json:"password"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	OrganizationID  string `json:"organization_id,omitempty"`
	InviteToken     string `json:"invite_token,omitempty"`
}

// RegisterResponse represents a registration response.
type RegisterResponse struct {
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	AccessToken string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// LoginRequest represents a login request.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents a login response.
type LoginResponse struct {
	UserID       string `json:"user_id"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

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

// OIDCHandler handles OIDC OAuth flows.
func (h *Handler) OIDCHandler(w http.ResponseWriter, r *http.Request) {
	if h.oidcRegistry == nil {
		respondError(w, apperrors.New("OIDC not configured", http.StatusServiceUnavailable))
		return
	}

	// Extract provider type from path
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		respondError(w, apperrors.New("invalid provider", http.StatusBadRequest))
		return
	}

	providerType := oidc.ProviderType(parts[3])
	manager, ok := h.oidcRegistry.Get(providerType)
	if !ok {
		respondError(w, apperrors.New("unknown provider", http.StatusBadRequest))
		return
	}

	ctx := r.Context()

	// Handle redirect to provider
	if r.Method == http.MethodGet && r.URL.Query().Get("code") == "" {
		state := uuid.New().String()
		authURL := manager.AuthURL(state)
		http.Redirect(w, r, authURL, http.StatusFound)
		return
	}

	// Handle callback from provider
	if r.Method == http.MethodGet && r.URL.Query().Get("code") != "" {
		manager.Handler(func(w http.ResponseWriter, r *http.Request, info *oidc.UserInfo, err error) {
			if err != nil {
				respondError(w, apperrors.Wrap(err, "oauth failed", http.StatusBadRequest))
				return
			}

			// Find or create user based on OIDC info
			user, err := h.findOrCreateUserByOIDC(ctx, info)
			if err != nil {
				respondError(w, apperrors.Wrap(err, "failed to process user", http.StatusInternalServerError))
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

			// Store session
			if err := h.sessionRepo.Store(ctx, user.ID, refreshToken, 7*24*time.Hour); err != nil {
				respondError(w, apperrors.Wrap(err, "failed to store session", http.StatusInternalServerError))
				return
			}

			// Return tokens to client
			respondJSON(w, http.StatusOK, map[string]interface{}{
				"user_id":       user.ID,
				"email":         user.Email,
				"access_token":  accessToken,
				"refresh_token": refreshToken,
			})
		}).ServeHTTP(w, r)
		return
	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

// SAMLMetadataHandler serves SAML metadata.
func (h *Handler) SAMLMetadataHandler(w http.ResponseWriter, r *http.Request) {
	if h.samlManager == nil {
		respondError(w, apperrors.New("SAML not configured", http.StatusServiceUnavailable))
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.samlManager.MetadataHandler().ServeHTTP(w, r)
}

// SAMLACSHandler handles SAML Assertion Consumer Service.
func (h *Handler) SAMLACSHandler(w http.ResponseWriter, r *http.Request) {
	if h.samlManager == nil {
		respondError(w, apperrors.New("SAML not configured", http.StatusServiceUnavailable))
		return
	}

	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Handle SAML response
	assertion, err := h.samlManager.HandleResponse(r)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "SAML error", http.StatusBadRequest))
		return
	}

	// Find or create user
	user, err := h.findOrCreateUserBySAML(ctx, assertion)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to process user", http.StatusInternalServerError))
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

	// Store session
	if err := h.sessionRepo.Store(ctx, user.ID, refreshToken, 7*24*time.Hour); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to store session", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":       user.ID,
		"email":         user.Email,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// Helper functions

func (h *Handler) findOrCreateUserByOIDC(ctx context.Context, info *oidc.UserInfo) (*repository.User, error) {
	// Try to find by email first
	user, err := h.userRepo.FindByEmail(ctx, info.Email)
	if err == nil {
		return user, nil
	}

	// Create new user
	user = &repository.User{
		ID:        uuid.New().String(),
		Email:     info.Email,
		FirstName: info.GivenName,
		LastName:  info.FamilyName,
		Role:      "user",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (h *Handler) findOrCreateUserBySAML(ctx context.Context, assertion *saml.Assertion) (*repository.User, error) {
	// Try to find by email first
	user, err := h.userRepo.FindByEmail(ctx, assertion.Email)
	if err == nil {
		return user, nil
	}

	// Create new user
	user = &repository.User{
		ID:        uuid.New().String(),
		Email:     assertion.Email,
		FirstName: assertion.FirstName,
		LastName:  assertion.LastName,
		Role:      "user",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func userToResponse(user *repository.User) map[string]interface{} {
	return map[string]interface{}{
		"user_id":     user.ID,
		"email":       user.Email,
		"first_name":  user.FirstName,
		"last_name":   user.LastName,
		"role":        user.Role,
		"is_active":   user.IsActive,
		"created_at":  user.CreatedAt,
		"last_login":  user.LastLoginAt,
	}
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, err error) {
	var appErr *apperrors.AppError
	if stderrors.As(err, &appErr) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(appErr.StatusCode)
		json.NewEncoder(w).Encode(apperrors.ToJSON(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]string{
		"code":    "INTERNAL_ERROR",
		"message": "An internal error occurred",
	})
}
