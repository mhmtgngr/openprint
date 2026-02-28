// Package oidc provides tests for OpenID Connect (OIDC) and OAuth2 integration.
package oidc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

func TestNewManager(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := &Config{
			ProviderType: ProviderGoogle,
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "https://example.com/callback",
		}

		mgr, err := NewManager(context.Background(), cfg)
		if err != nil {
			t.Fatalf("NewManager() error = %v", err)
		}
		if mgr == nil {
			t.Fatal("NewManager() returned nil")
		}
		if mgr.config != cfg {
			t.Error("NewManager() config not set correctly")
		}
	})

	t.Run("nil config", func(t *testing.T) {
		_, err := NewManager(context.Background(), nil)
		if err == nil {
			t.Error("NewManager(nil) should return error")
		}
	})

	t.Run("missing client ID", func(t *testing.T) {
		cfg := &Config{
			ProviderType: ProviderGoogle,
			RedirectURL:  "https://example.com/callback",
		}

		_, err := NewManager(context.Background(), cfg)
		if err == nil {
			t.Error("NewManager() without ClientID should return error")
		}
	})

	t.Run("default scopes", func(t *testing.T) {
		cfg := &Config{
			ProviderType: ProviderGoogle,
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "https://example.com/callback",
		}

		mgr, err := NewManager(context.Background(), cfg)
		if err != nil {
			t.Fatalf("NewManager() error = %v", err)
		}

		if len(mgr.config.Scopes) == 0 {
			t.Error("NewManager() should set default scopes")
		}
	})
}

func TestManager_AuthURL(t *testing.T) {
	cfg := &Config{
		ProviderType: ProviderGoogle,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "https://example.com/callback",
	}

	mgr, err := NewManager(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	t.Run("generate auth URL", func(t *testing.T) {
		state := "test-state-123"
		authURL, err := mgr.AuthURL(context.Background(), state)

		if err != nil {
			t.Fatalf("AuthURL() error = %v", err)
		}
		if authURL == "" {
			t.Fatal("AuthURL() returned empty URL")
		}

		// Check URL contains expected parameters
		if !strings.Contains(authURL, cfg.ClientID) {
			t.Errorf("AuthURL() should contain ClientID, got %v", authURL)
		}
		if !strings.Contains(authURL, url.QueryEscape(cfg.RedirectURL)) {
			t.Errorf("AuthURL() should contain RedirectURL, got %v", authURL)
		}
		if !strings.Contains(authURL, state) {
			t.Errorf("AuthURL() should contain state, got %v", authURL)
		}
	})

	t.Run("generate auth URL with empty state", func(t *testing.T) {
		authURL, err := mgr.AuthURL(context.Background(), "")

		if err != nil {
			t.Fatalf("AuthURL() error = %v", err)
		}
		if authURL == "" {
			t.Fatal("AuthURL() returned empty URL")
		}

		// State should have been generated and stored
		parsedURL, _ := url.Parse(authURL)
		stateParam := parsedURL.Query().Get("state")
		if stateParam == "" {
			t.Error("AuthURL() should generate state when not provided")
		}
	})
}

func TestManager_ValidateState(t *testing.T) {
	cfg := &Config{
		ProviderType: ProviderGoogle,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "https://example.com/callback",
	}

	mgr, err := NewManager(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	t.Run("valid state", func(t *testing.T) {
		state := "test-state-123"
		// First, generate auth URL to store the state and get signed state
		authURL, err := mgr.AuthURL(context.Background(), state)
		if err != nil {
			t.Fatalf("AuthURL() error = %v", err)
		}

		// Extract the signed state from the URL
		parsedURL, _ := url.Parse(authURL)
		signedState := parsedURL.Query().Get("state")

		err = mgr.ValidateState(context.Background(), signedState)
		if err != nil {
			t.Errorf("ValidateState() error = %v", err)
		}
	})

	t.Run("empty state", func(t *testing.T) {
		err := mgr.ValidateState(context.Background(), "")
		if err != ErrInvalidState {
			t.Errorf("ValidateState() error = %v, want %v", err, ErrInvalidState)
		}
	})

	t.Run("unknown state", func(t *testing.T) {
		err := mgr.ValidateState(context.Background(), "unknown-state")
		if err != ErrInvalidState {
			t.Errorf("ValidateState() error = %v, want %v", err, ErrInvalidState)
		}
	})
}

func TestRegistry(t *testing.T) {
	t.Run("new registry", func(t *testing.T) {
		registry := NewRegistry()

		if registry == nil {
			t.Fatal("NewRegistry() returned nil")
		}
	})

	t.Run("register and get provider", func(t *testing.T) {
		registry := NewRegistry()

		cfg := &Config{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "https://example.com/callback",
		}

		err := registry.Register(context.Background(), ProviderGoogle, cfg)
		if err != nil {
			t.Fatalf("Register() error = %v", err)
		}

		mgr, ok := registry.Get(ProviderGoogle)
		if !ok {
			t.Error("Get() should return true for registered provider")
		}
		if mgr == nil {
			t.Fatal("Get() returned nil manager")
		}
	})

	t.Run("get unregistered provider", func(t *testing.T) {
		registry := NewRegistry()

		_, ok := registry.Get(ProviderGoogle)
		if ok {
			t.Error("Get() should return false for unregistered provider")
		}
	})

	t.Run("register multiple providers", func(t *testing.T) {
		registry := NewRegistry()

		googleCfg := &Config{
			ClientID:    "google-client-id",
			ClientSecret: "google-client-secret",
			RedirectURL: "https://example.com/google/callback",
		}

		azureCfg := &Config{
			ClientID:    "azure-client-id",
			ClientSecret: "azure-client-secret",
			RedirectURL: "https://example.com/azure/callback",
		}

		registry.Register(context.Background(), ProviderGoogle, googleCfg)
		registry.Register(context.Background(), ProviderAzureAD, azureCfg)

		_, ok1 := registry.Get(ProviderGoogle)
		_, ok2 := registry.Get(ProviderAzureAD)

		if !ok1 || !ok2 {
			t.Error("Get() should return true for both registered providers")
		}
	})
}

func TestProviderConfigForType(t *testing.T) {
	t.Run("Google provider", func(t *testing.T) {
		cfg := ProviderConfigForType(ProviderGoogle, "client-id", "client-secret", "https://example.com/callback")

		if cfg.ProviderType != ProviderGoogle {
			t.Errorf("ProviderType = %v, want %v", cfg.ProviderType, ProviderGoogle)
		}
		if cfg.ClientID != "client-id" {
			t.Errorf("ClientID = %v, want client-id", cfg.ClientID)
		}
		if cfg.ClientSecret != "client-secret" {
			t.Errorf("ClientSecret = %v, want client-secret", cfg.ClientSecret)
		}
		if cfg.RedirectURL != "https://example.com/callback" {
			t.Errorf("RedirectURL = %v, want https://example.com/callback", cfg.RedirectURL)
		}
		if cfg.IssuerURL != "https://accounts.google.com" {
			t.Errorf("IssuerURL = %v, want https://accounts.google.com", cfg.IssuerURL)
		}
	})

	t.Run("Azure AD provider", func(t *testing.T) {
		cfg := ProviderConfigForType(ProviderAzureAD, "client-id", "client-secret", "https://example.com/callback")

		if cfg.ProviderType != ProviderAzureAD {
			t.Errorf("ProviderType = %v, want %v", cfg.ProviderType, ProviderAzureAD)
		}
		if cfg.IssuerURL != "https://login.microsoftonline.com/common/v2.0" {
			t.Errorf("IssuerURL = %v, want https://login.microsoftonline.com/common/v2.0", cfg.IssuerURL)
		}
	})

	t.Run("Okta provider", func(t *testing.T) {
		cfg := ProviderConfigForType(ProviderOkta, "client-id", "client-secret", "https://example.com/callback")

		if cfg.ProviderType != ProviderOkta {
			t.Errorf("ProviderType = %v, want %v", cfg.ProviderType, ProviderOkta)
		}
		if cfg.IssuerURL != "" {
			t.Errorf("IssuerURL should be empty for Okta, got %v", cfg.IssuerURL)
		}
	})
}

func TestBuildEndpoint(t *testing.T) {
	t.Run("Google provider", func(t *testing.T) {
		cfg := &Config{
			ProviderType: ProviderGoogle,
		}
		mgr := &Manager{config: cfg}

		endpoint := mgr.buildEndpoint()

		if endpoint.AuthURL != "https://accounts.google.com/o/oauth2/v2/auth" {
			t.Errorf("AuthURL = %v, want https://accounts.google.com/o/oauth2/v2/auth", endpoint.AuthURL)
		}
		if endpoint.TokenURL != "https://oauth2.googleapis.com/token" {
			t.Errorf("TokenURL = %v, want https://oauth2.googleapis.com/token", endpoint.TokenURL)
		}
	})

	t.Run("Azure AD provider", func(t *testing.T) {
		cfg := &Config{
			ProviderType: ProviderAzureAD,
		}
		mgr := &Manager{config: cfg}

		endpoint := mgr.buildEndpoint()

		if endpoint.AuthURL != "https://login.microsoftonline.com/common/oauth2/v2.0/authorize" {
			t.Errorf("AuthURL = %v, want https://login.microsoftonline.com/common/oauth2/v2.0/authorize", endpoint.AuthURL)
		}
		if endpoint.TokenURL != "https://login.microsoftonline.com/common/oauth2/v2.0/token" {
			t.Errorf("TokenURL = %v, want https://login.microsoftonline.com/common/oauth2/v2.0/token", endpoint.TokenURL)
		}
	})

	t.Run("custom endpoint", func(t *testing.T) {
		cfg := &Config{
			ProviderType:  ProviderGenericOIDC,
			EndpointURL:   "https://custom.example.com",
			AuthURL:       "https://custom.example.com/auth",
			IssuerURL:     "https://custom.example.com",
		}
		mgr := &Manager{config: cfg}

		endpoint := mgr.buildEndpoint()

		if endpoint.AuthURL != "https://custom.example.com/auth" {
			t.Errorf("AuthURL = %v, want https://custom.example.com/auth", endpoint.AuthURL)
		}
		if endpoint.TokenURL != "https://custom.example.com/token" {
			t.Errorf("TokenURL = %v, want https://custom.example.com/token", endpoint.TokenURL)
		}
	})
}

func TestParseCallbackURL(t *testing.T) {
	t.Run("valid callback URL", func(t *testing.T) {
		rawURL := "https://example.com/callback?code=auth-code-123&state=state-456"

		code, state, err := ParseCallbackURL(rawURL)
		if err != nil {
			t.Fatalf("ParseCallbackURL() error = %v", err)
		}
		if code != "auth-code-123" {
			t.Errorf("code = %v, want auth-code-123", code)
		}
		if state != "state-456" {
			t.Errorf("state = %v, want state-456", state)
		}
	})

	t.Run("callback URL with error", func(t *testing.T) {
		rawURL := "https://example.com/callback?error=access_denied"

		_, _, err := ParseCallbackURL(rawURL)
		if err == nil {
			t.Error("ParseCallbackURL() should return error for error response")
		}
	})

	t.Run("invalid URL", func(t *testing.T) {
		_, _, err := ParseCallbackURL(":invalid-url")
		if err == nil {
			t.Error("ParseCallbackURL() should return error for invalid URL")
		}
	})
}

func TestHandler(t *testing.T) {
	cfg := &Config{
		ProviderType: ProviderGoogle,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "https://example.com/callback",
	}

	mgr, err := NewManager(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	t.Run("invalid state", func(t *testing.T) {
		callbackCalled := false
		handler := mgr.Handler(func(w http.ResponseWriter, r *http.Request, info *UserInfo, err error) {
			callbackCalled = true
			if err == nil {
				t.Error("Handler callback should receive error for invalid state")
			}
		})

		req := httptest.NewRequest("GET", "/callback?code=auth-code&state=invalid-state", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if !callbackCalled {
			t.Error("Handler callback was not called")
		}
	})

	t.Run("missing code", func(t *testing.T) {
		callbackCalled := false
		handler := mgr.Handler(func(w http.ResponseWriter, r *http.Request, info *UserInfo, err error) {
			callbackCalled = true
			if err == nil {
				t.Error("Handler callback should receive error for missing code")
			}
		})

		// First store valid state and get signed state
		state := "test-state"
		authURL, _ := mgr.AuthURL(context.Background(), state)
		parsedURL, _ := url.Parse(authURL)
		signedState := parsedURL.Query().Get("state")

		req := httptest.NewRequest("GET", "/callback?state="+signedState, nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if !callbackCalled {
			t.Error("Handler callback was not called")
		}
	})
}

func TestUserInfo(t *testing.T) {
	t.Run("create UserInfo", func(t *testing.T) {
		info := &UserInfo{
			Subject:       "user-123",
			Email:         "user@example.com",
			EmailVerified: true,
			Name:          "John Doe",
			GivenName:     "John",
			FamilyName:    "Doe",
			Picture:       "https://example.com/avatar.jpg",
			Groups:        []string{"admins", "users"},
			Provider:      ProviderGoogle,
			AccessToken:   "access-token",
			RefreshToken:  "refresh-token",
		}

		if info.Subject != "user-123" {
			t.Error("UserInfo Subject not set correctly")
		}
		if info.Email != "user@example.com" {
			t.Error("UserInfo Email not set correctly")
		}
		if len(info.Groups) != 2 {
			t.Errorf("UserInfo Groups length = %d, want 2", len(info.Groups))
		}
	})
}

func TestProviderType_String(t *testing.T) {
	providers := []ProviderType{
		ProviderAzureAD,
		ProviderGoogle,
		ProviderOkta,
		ProviderAuth0,
		ProviderGenericOIDC,
	}

	for _, provider := range providers {
		if string(provider) == "" {
			t.Errorf("ProviderType %v string representation is empty", provider)
		}
	}
}

func TestProviderType_Constants(t *testing.T) {
	providers := []struct {
		name  string
		ptype ProviderType
	}{
		{"AzureAD", ProviderAzureAD},
		{"Google", ProviderGoogle},
		{"Okta", ProviderOkta},
		{"Auth0", ProviderAuth0},
		{"GenericOIDC", ProviderGenericOIDC},
	}

	for _, p := range providers {
		t.Run(p.name, func(t *testing.T) {
			if string(p.ptype) == "" {
				t.Errorf("ProviderType %v string representation is empty", p.ptype)
			}
		})
	}
}

func TestDiscoveryDocument(t *testing.T) {
	doc := &DiscoveryDocument{
		Issuer:                  "https://accounts.google.com",
		AuthorizationEndpoint:  "https://accounts.google.com/o/oauth2/v2/auth",
		TokenEndpoint:          "https://oauth2.googleapis.com/token",
		UserInfoEndpoint:       "https://www.googleapis.com/oauth2/v2/userinfo",
		JWKSURI:                 "https://www.googleapis.com/oauth2/v3/certs",
	}

	if doc.Issuer != "https://accounts.google.com" {
		t.Error("DiscoveryDocument Issuer not set correctly")
	}
	if doc.AuthorizationEndpoint != "https://accounts.google.com/o/oauth2/v2/auth" {
		t.Error("DiscoveryDocument AuthorizationEndpoint not set correctly")
	}
}

func TestTokenResponse(t *testing.T) {
	resp := &TokenResponse{
		AccessToken:  "access-token-123",
		TokenType:    "Bearer",
		RefreshToken: "refresh-token-456",
		ExpiresIn:    3600,
		IDToken:      "id-token-789",
	}

	if resp.AccessToken != "access-token-123" {
		t.Error("TokenResponse AccessToken not set correctly")
	}
	if resp.TokenType != "Bearer" {
		t.Errorf("TokenType = %v, want Bearer", resp.TokenType)
	}
	if resp.ExpiresIn != 3600 {
		t.Errorf("ExpiresIn = %d, want 3600", resp.ExpiresIn)
	}
}

func TestManager_RefreshToken(t *testing.T) {
	cfg := &Config{
		ProviderType: ProviderGoogle,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "https://example.com/callback",
	}

	mgr, err := NewManager(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// This test validates the method exists and has correct signature
	// Actual token refresh requires a valid OAuth2 flow
	_ = mgr

	t.Run("refresh token requires real provider", func(t *testing.T) {
		// In a real test with a mock server, we would test the actual refresh
		// For now, we ensure the method exists and handles nil token
		_, err := mgr.RefreshToken(context.Background(), "")
		if err == nil {
			t.Error("RefreshToken() should return error for empty token")
		}
	})
}

func TestManager_GetUserInfo(t *testing.T) {
	cfg := &Config{
		ProviderType: ProviderGoogle,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "https://example.com/callback",
	}

	mgr, err := NewManager(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// Create a mock token that will fail when fetching user info
	// In real scenario, this would require a mock OAuth2 server
	t.Run("get user info requires valid token", func(t *testing.T) {
		// Create a minimal token structure that will fail network call
		token := &oauth2.Token{} // This will cause GetUserInfo to fail

		_, err := mgr.GetUserInfo(context.Background(), token)
		if err == nil {
			t.Error("GetUserInfo() should return error without valid token")
		}
	})
}

func TestExchange(t *testing.T) {
	cfg := &Config{
		ProviderType: ProviderGoogle,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "https://example.com/callback",
	}

	mgr, err := NewManager(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	t.Run("empty code returns error", func(t *testing.T) {
		_, err := mgr.Exchange(context.Background(), "")
		if err == nil {
			t.Error("Exchange() should return error for empty code")
		}
		if err != ErrInvalidCode {
			t.Errorf("Exchange() error = %v, want %v", err, ErrInvalidCode)
		}
	})
}
