// Package handler provides HTTP handlers for the auth service endpoints.
package handler

import (
	"context"
	"encoding/json"
	stderrors "errors"
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
		authURL, err := manager.AuthURL(ctx, state)
		if err != nil {
			respondError(w, apperrors.Wrap(err, "failed to generate auth URL", http.StatusInternalServerError))
			return
		}
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
