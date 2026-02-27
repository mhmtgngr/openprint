// Package oidc provides OpenID Connect (OIDC) and OAuth2 integration for OpenPrint.
// This enables authentication with providers like Azure AD, Google Workspace, Okta, etc.
package oidc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

var (
	// ErrInvalidState is returned when the OAuth state parameter doesn't match.
	ErrInvalidState = errors.New("invalid OAuth state")
	// ErrInvalidCode is returned when the authorization code is invalid.
	ErrInvalidCode = errors.New("invalid authorization code")
	// ErrNoEmail is returned when the provider doesn't return an email.
	ErrNoEmail = errors.New("provider did not return email address")
	// ErrUnverifiedEmail is returned when the email is not verified.
	ErrUnverifiedEmail = errors.New("email is not verified")
)

// ProviderType represents the type of OIDC provider.
type ProviderType string

const (
	ProviderAzureAD     ProviderType = "azure_ad"
	ProviderGoogle      ProviderType = "google"
	ProviderOkta        ProviderType = "okta"
	ProviderAuth0       ProviderType = "auth0"
	ProviderGenericOIDC ProviderType = "generic_oidc"
)

// Config holds OIDC provider configuration.
type Config struct {
	// ProviderType is the type of identity provider.
	ProviderType ProviderType
	// IssuerURL is the OIDC issuer URL (e.g., https://accounts.google.com).
	IssuerURL string
	// ClientID is the OAuth2 client ID.
	ClientID string
	// ClientSecret is the OAuth2 client secret.
	ClientSecret string
	// Scopes to request from the provider.
	Scopes []string
	// RedirectURL is the OAuth redirect URL.
	RedirectURL string
	// EndpointURL is the token endpoint for non-OIDC OAuth2 providers.
	EndpointURL string
	// AuthURL is the authorization URL for non-OIDC providers.
	AuthURL string
}

// UserInfo represents user information from an OIDC provider.
type UserInfo struct {
	Subject        string   `json:"sub"`
	Email          string   `json:"email"`
	EmailVerified  bool     `json:"email_verified"`
	Name           string   `json:"name"`
	GivenName      string   `json:"given_name"`
	FamilyName     string   `json:"family_name"`
	Picture        string   `json:"picture"`
	Groups         []string `json:"groups"`
	Provider       ProviderType
	AccessToken    string
	RefreshToken   string
	Expiry         time.Time
}

// Manager handles OIDC authentication operations.
type Manager struct {
	config       *Config
	oauth2Config *oauth2.Config
	stateStore   sync.Map // state -> creation time
	userInfoURL  string
}

// NewManager creates a new OIDC manager.
func NewManager(ctx context.Context, config *Config) (*Manager, error) {
	if config == nil {
		return nil, errors.New("oidc config cannot be nil")
	}

	if config.ClientID == "" {
		return nil, errors.New("client ID is required")
	}

	m := &Manager{config: config}

	// Set default scopes if none provided
	if len(config.Scopes) == 0 {
		config.Scopes = []string{"openid", "profile", "email"}
	}

	// Build OAuth2 config
	endpoint := m.buildEndpoint()

	m.oauth2Config = &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Scopes:       config.Scopes,
		Endpoint:     endpoint,
	}

	// Set userinfo URL based on provider type
	m.userInfoURL = config.EndpointURL + "/userinfo"
	if config.ProviderType == ProviderGoogle {
		m.userInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"
	}

	return m, nil
}

// buildEndpoint constructs the OAuth endpoint from config.
func (m *Manager) buildEndpoint() oauth2.Endpoint {
	endpoint := oauth2.Endpoint{}

	if m.config.EndpointURL != "" && m.config.AuthURL != "" {
		// Custom endpoint
		endpoint.AuthURL = m.config.AuthURL
		endpoint.TokenURL = m.config.EndpointURL + "/token"
		return endpoint
	}

	// Well-known provider endpoints
	switch m.config.ProviderType {
	case ProviderGoogle:
		endpoint.AuthURL = "https://accounts.google.com/o/oauth2/v2/auth"
		endpoint.TokenURL = "https://oauth2.googleapis.com/token"
	case ProviderAzureAD:
		endpoint.AuthURL = "https://login.microsoftonline.com/common/oauth2/v2.0/authorize"
		endpoint.TokenURL = "https://login.microsoftonline.com/common/oauth2/v2.0/token"
	case ProviderOkta, ProviderAuth0:
		// Use issuer URL as base
		if m.config.IssuerURL != "" {
			endpoint.AuthURL = m.config.IssuerURL + "/v1/authorize"
			endpoint.TokenURL = m.config.IssuerURL + "/v1/token"
		}
	default:
		// Generic OIDC - try standard discovery endpoints
		if m.config.IssuerURL != "" {
			endpoint.AuthURL = m.config.IssuerURL + "/authorize"
			endpoint.TokenURL = m.config.IssuerURL + "/token"
		}
	}

	return endpoint
}

// AuthURL generates the OAuth authorization URL.
func (m *Manager) AuthURL(state string) string {
	if state == "" {
		state = generateState()
	}

	// Store state with timestamp for validation
	m.stateStore.Store(state, time.Now())

	return m.oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// Exchange exchanges the authorization code for tokens.
func (m *Manager) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	if code == "" {
		return nil, ErrInvalidCode
	}

	token, err := m.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}

	return token, nil
}

// GetUserInfo retrieves user information using the access token.
func (m *Manager) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := m.oauth2Config.Client(ctx, token)

	resp, err := client.Get(m.userInfoURL)
	if err != nil {
		return nil, fmt.Errorf("get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user info request failed: %d", resp.StatusCode)
	}

	var info UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode user info: %w", err)
	}

	if info.Email == "" {
		return nil, ErrNoEmail
	}

	info.Provider = m.config.ProviderType
	info.AccessToken = token.AccessToken
	info.RefreshToken = token.RefreshToken
	info.Expiry = token.Expiry

	return &info, nil
}

// RefreshToken refreshes an expired access token using the refresh token.
func (m *Manager) RefreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	token := &oauth2.Token{
		RefreshToken: refreshToken,
		Expiry:       time.Now().Add(-time.Hour), // Force refresh
	}

	return m.oauth2Config.TokenSource(ctx, token).Token()
}

// ValidateState validates the OAuth state parameter.
func (m *Manager) ValidateState(state string) error {
	if state == "" {
		return ErrInvalidState
	}

	value, ok := m.stateStore.LoadAndDelete(state)
	if !ok {
		return ErrInvalidState
	}

	// Check if state is too old (10 minutes)
	creationTime, ok := value.(time.Time)
	if !ok || time.Since(creationTime) > 10*time.Minute {
		return ErrInvalidState
	}

	return nil
}

// Handler returns an HTTP handler for the OAuth callback.
func (m *Manager) Handler(callback func(w http.ResponseWriter, r *http.Request, info *UserInfo, err error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Validate state
		state := r.URL.Query().Get("state")
		if err := m.ValidateState(state); err != nil {
			callback(w, r, nil, fmt.Errorf("validate state: %w", err))
			return
		}

		// Exchange code for token
		code := r.URL.Query().Get("code")
		if code == "" {
			callback(w, r, nil, ErrInvalidCode)
			return
		}

		token, err := m.Exchange(ctx, code)
		if err != nil {
			callback(w, r, nil, fmt.Errorf("exchange token: %w", err))
			return
		}

		// Get user info
		info, err := m.GetUserInfo(ctx, token)
		if err != nil {
			callback(w, r, nil, fmt.Errorf("get user info: %w", err))
			return
		}

		callback(w, r, info, nil)
	})
}

// generateState generates a secure random state parameter.
func generateState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// Registry manages multiple OIDC providers.
type Registry struct {
	mu        sync.RWMutex
	providers map[ProviderType]*Manager
}

// NewRegistry creates a new provider registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[ProviderType]*Manager),
	}
}

// Register adds a provider to the registry.
func (r *Registry) Register(ctx context.Context, providerType ProviderType, config *Config) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	config.ProviderType = providerType
	manager, err := NewManager(ctx, config)
	if err != nil {
		return fmt.Errorf("create provider %s: %w", providerType, err)
	}

	r.providers[providerType] = manager
	return nil
}

// Get retrieves a provider by type.
func (r *Registry) Get(providerType ProviderType) (*Manager, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	m, ok := r.providers[providerType]
	return m, ok
}

// ProviderConfigForType returns a standard config for common providers.
func ProviderConfigForType(providerType ProviderType, clientID, clientSecret, redirectURL string) *Config {
	switch providerType {
	case ProviderGoogle:
		return &Config{
			ProviderType: ProviderGoogle,
			IssuerURL:    "https://accounts.google.com",
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"openid", "profile", "email"},
		}
	case ProviderAzureAD:
		// Note: Azure AD tenant-specific URL should be used in production
		return &Config{
			ProviderType: ProviderAzureAD,
			IssuerURL:    "https://login.microsoftonline.com/common/v2.0",
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"openid", "profile", "email", "offline_access"},
		}
	case ProviderOkta:
		return &Config{
			ProviderType: ProviderOkta,
			IssuerURL:    "", // Must be provided by user
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"openid", "profile", "email"},
		}
	default:
		return &Config{
			ProviderType: providerType,
			IssuerURL:    "", // Must be provided
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
		}
	}
}

// TokenResponse represents the token exchange response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	IDToken      string `json:"id_token"`
}

// DiscoveryDocument represents the OpenID Connect discovery document.
type DiscoveryDocument struct {
	Issuer                  string `json:"issuer"`
	AuthorizationEndpoint  string `json:"authorization_endpoint"`
	TokenEndpoint          string `json:"token_endpoint"`
	UserInfoEndpoint       string `json:"userinfo_endpoint"`
	JWKSURI                string `json:"jwks_uri"`
	ResponseTypesSupported []string `json:"response_types_supported"`
	SubjectTypesSupported  []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
}

// FetchDiscovery fetches the discovery document from the issuer.
func FetchDiscovery(ctx context.Context, issuerURL string) (*DiscoveryDocument, error) {
	wellKnown := issuerURL
	if issuerURL[len(issuerURL)-1] != '/' {
		wellKnown += "/"
	}
	wellKnown += ".well-known/openid-configuration"

	req, err := http.NewRequestWithContext(ctx, "GET", wellKnown, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery request failed: %d", resp.StatusCode)
	}

	var doc DiscoveryDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, err
	}

	return &doc, nil
}

// ParseCallbackURL extracts the code and state from a callback URL.
func ParseCallbackURL(rawURL string) (code, state string, err error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", err
	}

	code = u.Query().Get("code")
	state = u.Query().Get("state")

	// Check for error response
	if errMsg := u.Query().Get("error"); errMsg != "" {
		return "", "", fmt.Errorf("oauth error: %s", errMsg)
	}

	return code, state, nil
}
