// Package handler provides HTTP handlers for Microsoft 365 OIDC authentication.
// This handler includes E2E test mocking support for automated testing.
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"

	"github.com/openprint/openprint/internal/auth/jwt"
	"github.com/openprint/openprint/internal/auth/oidc"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/services/auth-service/repository"
)

// Microsoft365Config holds Microsoft 365 OIDC configuration.
type Microsoft365Config struct {
	TenantID     string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	IsTestMode   bool
	TestUsers    []TestUser
}

// TestUser represents a mock user for E2E testing.
type TestUser struct {
	Email        string   `json:"email"`
	ID           string   `json:"id"`
	DisplayName  string   `json:"display_name"`
	GivenName    string   `json:"given_name"`
	FamilyName   string   `json:"family_name"`
	Groups       []string `json:"groups"`
	RefreshToken string   `json:"refresh_token"`
}

// OIDCHandler provides Microsoft 365 OIDC authentication with test mocking.
type OIDCHandler struct {
	userRepo       *repository.UserRepository
	sessionRepo    *repository.SessionRepository
	jwtManager     *jwt.Manager
	config         *Microsoft365Config
	oidcManager    *oidc.Manager
	mockServer     *httptest.Server
	testModeTokens map[string]string // state -> mock user email
	mu             sync.RWMutex
}

// NewOIDCHandler creates a new Microsoft 365 OIDC handler.
func NewOIDCHandler(
	userRepo *repository.UserRepository,
	sessionRepo *repository.SessionRepository,
	jwtManager *jwt.Manager,
	config *Microsoft365Config,
) *OIDCHandler {
	h := &OIDCHandler{
		userRepo:       userRepo,
		sessionRepo:    sessionRepo,
		jwtManager:     jwtManager,
		config:         config,
		testModeTokens: make(map[string]string),
	}

	// Initialize test mode if enabled
	if config.IsTestMode {
		h.initTestMode()
	} else {
		h.initOIDCManager()
	}

	return h
}

// initTestMode sets up the mock OIDC server for E2E testing.
func (h *OIDCHandler) initTestMode() {
	// Create a mock server that simulates Microsoft 365 OAuth endpoints
	h.mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/oauth2/v2.0/authorize"):
			h.handleMockAuthorize(w, r)
		case strings.HasPrefix(r.URL.Path, "/oauth2/v2.0/token"):
			h.handleMockToken(w, r)
		case strings.HasPrefix(r.URL.Path, "/v1.0/me") || strings.HasPrefix(r.URL.Path, "/openid/userinfo"):
			h.handleMockUserInfo(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
}

// handleMockAuthorize handles the mock authorization endpoint.
func (h *OIDCHandler) handleMockAuthorize(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	email := r.URL.Query().Get("login_hint")

	// Store the state with the email for callback validation
	if email == "" {
		h.mu.RLock()
		if len(h.config.TestUsers) > 0 {
			email = h.config.TestUsers[0].Email // Default test user
		} else {
			email = "test@example.com"
		}
		h.mu.RUnlock()
	}

	h.mu.Lock()
	h.testModeTokens[state] = email
	h.mu.Unlock()

	// Redirect to mock callback URL
	redirectURL := h.config.RedirectURL
	redirectURL += "?code=mock_auth_code_" + state
	redirectURL += "&state=" + state

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// handleMockToken handles the mock token endpoint.
func (h *OIDCHandler) handleMockToken(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token":  "mock_access_token_" + code,
		"refresh_token": "mock_refresh_token_" + code,
		"token_type":    "Bearer",
		"expires_in":    3600,
		"id_token":      "mock_id_token_" + code,
	})
}

// handleMockUserInfo handles the mock user info endpoint.
func (h *OIDCHandler) handleMockUserInfo(w http.ResponseWriter, r *http.Request) {
	// Extract from authorization header
	authHeader := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")

	// Find test user by token (in real mode, this would validate the access token)
	h.mu.RLock()
	var testUser *TestUser
	testUsers := make([]TestUser, len(h.config.TestUsers))
	copy(testUsers, h.config.TestUsers) // Take a snapshot for safe iteration
	h.mu.RUnlock()

	for _, u := range testUsers {
		if strings.Contains(token, u.Email) || len(testUsers) == 1 {
			testUser = &u
			break
		}
	}

	if testUser == nil && len(testUsers) > 0 {
		testUser = &testUsers[0]
	}

	if testUser == nil {
		// Default test user
		testUser = &TestUser{
			Email:       "test@example.com",
			ID:          "test-user-id",
			DisplayName: "Test User",
			GivenName:   "Test",
			FamilyName:  "User",
			Groups:      []string{"Users"},
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sub":                testUser.ID,
		"email":              testUser.Email,
		"email_verified":     true,
		"name":               testUser.DisplayName,
		"given_name":         testUser.GivenName,
		"family_name":        testUser.FamilyName,
		"groups":             testUser.Groups,
		"preferred_username": testUser.Email,
	})
}

// initOIDCManager creates the real OIDC manager for production.
func (h *OIDCHandler) initOIDCManager() {
	_ = &oidc.Config{
		ProviderType: oidc.ProviderAzureAD,
		IssuerURL:    "https://login.microsoftonline.com/" + h.config.TenantID + "/v2.0",
		ClientID:     h.config.ClientID,
		ClientSecret: h.config.ClientSecret,
		RedirectURL:  h.config.RedirectURL,
		Scopes:       []string{"openid", "profile", "email", "offline_access", "User.Read"},
	}

	// In production, the OIDC manager would be created with proper context
	// For now, we'll store the config for later initialization
}

// Microsoft365Login initiates Microsoft 365 OAuth login flow.
func (h *OIDCHandler) Microsoft365Login(w http.ResponseWriter, r *http.Request) {
	_ = r.Context()

	// Check for test mode header
	if isE2ETestRequest(r) && h.config.IsTestMode {
		h.handleTestModeLogin(w, r)
		return
	}

	// Production flow would redirect to actual Microsoft 365
	respondError(w, apperrors.New("Microsoft 365 integration not fully configured. Contact administrator.", http.StatusServiceUnavailable))
}

// handleTestModeLogin handles E2E test login with mock responses.
func (h *OIDCHandler) handleTestModeLogin(w http.ResponseWriter, r *http.Request) {
	// Get test user email from query or use default
	testEmail := r.URL.Query().Get("test_email")
	if testEmail == "" {
		h.mu.RLock()
		if len(h.config.TestUsers) > 0 {
			testEmail = h.config.TestUsers[0].Email
		} else {
			testEmail = "test@example.com"
		}
		h.mu.RUnlock()
	}

	// Return mock authorization URL
	authURL := h.mockServer.URL + "/oauth2/v2.0/authorize?client_id=" + h.config.ClientID
	authURL += "&response_type=code"
	authURL += "&redirect_uri=" + h.config.RedirectURL
	authURL += "&scope=openid profile email User.Read"
	authURL += "&state=test_state_" + testEmail
	authURL += "&login_hint=" + testEmail

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"authorization_url": authURL,
		"test_mode":         true,
	})
}

// Microsoft365Callback handles the OAuth callback from Microsoft 365.
func (h *OIDCHandler) Microsoft365Callback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errorCode := r.URL.Query().Get("error")

	if errorCode != "" {
		respondError(w, apperrors.New("OAuth error: "+errorCode, http.StatusBadRequest))
		return
	}

	if code == "" {
		respondError(w, apperrors.New("missing authorization code", http.StatusBadRequest))
		return
	}

	// In test mode, get user info from mock server
	if h.config.IsTestMode && h.mockServer != nil {
		h.handleTestModeCallback(w, r, ctx, state)
		return
	}

	// Production flow would exchange code for tokens
	respondError(w, apperrors.New("Microsoft 365 integration not fully configured", http.StatusServiceUnavailable))
}

// handleTestModeCallback handles the callback in test mode.
func (h *OIDCHandler) handleTestModeCallback(w http.ResponseWriter, r *http.Request, ctx context.Context, state string) {
	// Get the email associated with this state
	h.mu.RLock()
	email := h.testModeTokens[state]
	h.mu.RUnlock()

	// Find or create user
	user, err := h.findOrCreateTestUser(ctx, email)
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
	if err := h.sessionRepo.Store(ctx, user.ID, refreshToken, jwt.MaxRefreshDuration); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to store session", http.StatusInternalServerError))
		return
	}

	// Clean up state
	h.mu.Lock()
	delete(h.testModeTokens, state)
	h.mu.Unlock()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":       user.ID,
		"email":         user.Email,
		"name":          user.FirstName + " " + user.LastName,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"test_mode":     true,
	})
}

// findOrCreateTestUser finds or creates a test user.
func (h *OIDCHandler) findOrCreateTestUser(ctx context.Context, email string) (*repository.User, error) {
	// Try to find existing user
	user, err := h.userRepo.FindByEmail(ctx, email)
	if err == nil {
		return user, nil
	}

	// Find test user details - take a snapshot under lock
	h.mu.RLock()
	var testUser *TestUser
	for _, u := range h.config.TestUsers {
		if u.Email == email {
			// Make a copy to avoid holding the lock
			uCopy := u
			testUser = &uCopy
			break
		}
	}
	h.mu.RUnlock()

	// Create default test user if not found
	if testUser == nil {
		testUser = &TestUser{
			Email:       email,
			ID:          "test-" + email,
			DisplayName: "Test User",
			GivenName:   "Test",
			FamilyName:  "User",
			Groups:      []string{"Users"},
		}
	}

	// Create new user
	user = &repository.User{
		ID:        testUser.ID,
		Email:     testUser.Email,
		FirstName: testUser.GivenName,
		LastName:  testUser.FamilyName,
		Role:      "user",
		IsActive:  true,
	}

	if err := h.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// Microsoft365ConfigHandler returns the Microsoft 365 configuration status.
func (h *OIDCHandler) Microsoft365ConfigHandler(w http.ResponseWriter, r *http.Request) {
	configured := !h.config.IsTestMode && h.config.ClientID != "" && h.config.TenantID != ""

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"configured": configured,
		"test_mode":  h.config.IsTestMode,
		"tenant_id": func() string {
			if h.config.TenantID != "" {
				return h.config.TenantID
			}
			return ""
		}(),
	})
}

// isE2ETestRequest checks if the request is from an E2E test.
func isE2ETestRequest(r *http.Request) bool {
	// Check for test mode header
	if r.Header.Get("X-E2E-Test") == "true" {
		return true
	}

	// Check for test environment variable
	if os.Getenv("E2E_TEST_MODE") == "true" {
		return true
	}

	// Check for test query parameter
	if r.URL.Query().Get("test_mode") == "true" {
		return true
	}

	return false
}

// GetTestUsers returns the configured test users.
func (h *OIDCHandler) GetTestUsers() []TestUser {
	h.mu.RLock()
	defer h.mu.RUnlock()
	// Return a copy to prevent race conditions
	users := make([]TestUser, len(h.config.TestUsers))
	copy(users, h.config.TestUsers)
	return users
}

// SetTestUsers sets the test users for E2E testing.
func (h *OIDCHandler) SetTestUsers(users []TestUser) {
	h.mu.Lock()
	h.config.TestUsers = users
	h.mu.Unlock()
}

// Close cleans up resources.
func (h *OIDCHandler) Close() {
	if h.mockServer != nil {
		h.mockServer.Close()
	}
}
