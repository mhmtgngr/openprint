// Package testutil provides JWT token generation helpers for testing.
package testutil

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	// DefaultTestSecret is the default JWT secret key for testing.
	// DO NOT use this in production.
	DefaultTestSecret = "test-secret-key-min-32-chars-for-testing"
	// DefaultTestIssuer is the default JWT issuer for testing.
	DefaultTestIssuer = "openprint.test"
	// DefaultTestAccessDuration is the default access token duration for testing.
	DefaultTestAccessDuration = 15 * time.Minute
	// DefaultTestRefreshDuration is the default refresh token duration for testing.
	DefaultTestRefreshDuration = 7 * 24 * time.Hour
)

// TestClaims represents JWT claims for testing.
type TestClaims struct {
	UserID    string   `json:"user_id"`
	Email     string   `json:"email"`
	OrgID     string   `json:"org_id,omitempty"`
	Role      string   `json:"role"`
	TokenType string   `json:"token_type"`
	Scopes    []string `json:"scopes,omitempty"`
	jwt.RegisteredClaims
}

// TokenOptions holds options for generating test tokens.
type TokenOptions struct {
	Secret     string
	UserID     string
	Email      string
	OrgID      string
	Role       string
	Scopes     []string
	TokenType  string
	Expiry     time.Duration
	NotBefore  time.Time
	Issuer     string
}

// DefaultTokenOptions returns default token options for testing.
func DefaultTokenOptions() TokenOptions {
	return TokenOptions{
		Secret:    DefaultTestSecret,
		UserID:    uuid.New().String(),
		Email:     "test@example.com",
		OrgID:     uuid.New().String(),
		Role:      "user",
		Scopes:    []string{"print:read", "print:write"},
		TokenType: "access",
		Expiry:    DefaultTestAccessDuration,
		Issuer:    DefaultTestIssuer,
	}
}

// GenerateTestToken generates a JWT token for testing with default options.
func GenerateTestToken() (string, error) {
	return GenerateTokenWithOptions(DefaultTokenOptions())
}

// GenerateTokenWithOptions generates a JWT token with custom options.
func GenerateTokenWithOptions(opts TokenOptions) (string, error) {
	if opts.Secret == "" {
		opts.Secret = DefaultTestSecret
	}
	if opts.UserID == "" {
		opts.UserID = uuid.New().String()
	}
	if opts.Email == "" {
		opts.Email = "test@example.com"
	}
	if opts.Role == "" {
		opts.Role = "user"
	}
	if opts.TokenType == "" {
		opts.TokenType = "access"
	}
	if opts.Expiry == 0 {
		opts.Expiry = DefaultTestAccessDuration
	}
	if opts.Issuer == "" {
		opts.Issuer = DefaultTestIssuer
	}

	now := time.Now()
	if opts.NotBefore.IsZero() {
		opts.NotBefore = now
	}

	claims := TestClaims{
		UserID:    opts.UserID,
		Email:     opts.Email,
		OrgID:     opts.OrgID,
		Role:      opts.Role,
		TokenType: opts.TokenType,
		Scopes:    opts.Scopes,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    opts.Issuer,
			Subject:   opts.UserID,
			ExpiresAt: jwt.NewNumericDate(now.Add(opts.Expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(opts.NotBefore),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(opts.Secret))
}

// GenerateTestAccessToken generates a test access token.
func GenerateTestAccessToken(userID, email, role, orgID string, scopes []string) (string, error) {
	opts := DefaultTokenOptions()
	opts.UserID = userID
	opts.Email = email
	opts.Role = role
	opts.OrgID = orgID
	opts.Scopes = scopes
	opts.TokenType = "access"
	opts.Expiry = DefaultTestAccessDuration
	return GenerateTokenWithOptions(opts)
}

// GenerateTestRefreshToken generates a test refresh token.
func GenerateTestRefreshToken(userID, email string) (string, error) {
	opts := DefaultTokenOptions()
	opts.UserID = userID
	opts.Email = email
	opts.TokenType = "refresh"
	opts.Expiry = DefaultTestRefreshDuration
	opts.Scopes = nil // Refresh tokens don't need scopes
	return GenerateTokenWithOptions(opts)
}

// GenerateAdminToken generates a test admin token.
func GenerateAdminToken() (string, error) {
	opts := DefaultTokenOptions()
	opts.Role = "admin"
	opts.Scopes = []string{"admin", "print:read", "print:write", "print:delete", "user:read", "user:write"}
	return GenerateTokenWithOptions(opts)
}

// GenerateOrgAdminToken generates a test organization admin token.
func GenerateOrgAdminToken(orgID string) (string, error) {
	opts := DefaultTokenOptions()
	opts.Role = "org_admin"
	opts.OrgID = orgID
	opts.Scopes = []string{"print:read", "print:write", "printer:read", "printer:write", "user:read"}
	return GenerateTokenWithOptions(opts)
}

// GenerateExpiredToken generates an expired JWT token for testing expiration logic.
func GenerateExpiredToken() (string, error) {
	opts := DefaultTokenOptions()
	opts.Expiry = -1 * time.Hour // Expired 1 hour ago
	return GenerateTokenWithOptions(opts)
}

// GenerateTokenWithExpiry generates a token that expires at the specified time.
func GenerateTokenWithExpiry(expiry time.Time) (string, error) {
	opts := DefaultTokenOptions()
	opts.Expiry = time.Until(expiry)
	return GenerateTokenWithOptions(opts)
}

// GenerateTokenWithCustomClaims generates a token with custom claims.
func GenerateTokenWithCustomClaims(claims map[string]interface{}) (string, error) {
	opts := DefaultTokenOptions()

	// Build registered claims
	now := time.Now()
	registeredClaims := jwt.RegisteredClaims{
		Issuer:    opts.Issuer,
		Subject:   getStringValue(claims, "sub", opts.UserID),
		ExpiresAt: jwt.NewNumericDate(now.Add(opts.Expiry)),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		ID:        uuid.New().String(),
	}

	// Merge custom claims
	tokenClaims := jwt.MapClaims{}
	for k, v := range claims {
		tokenClaims[k] = v
	}

	// Add registered claims
	tokenClaims["iss"] = registeredClaims.Issuer
	tokenClaims["sub"] = registeredClaims.Subject
	tokenClaims["exp"] = registeredClaims.ExpiresAt
	tokenClaims["iat"] = registeredClaims.IssuedAt
	tokenClaims["nbf"] = registeredClaims.NotBefore
	tokenClaims["jti"] = registeredClaims.ID

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims)
	return token.SignedString([]byte(opts.Secret))
}

// ParseTestToken parses a test JWT token and returns the claims.
func ParseTestToken(tokenString string) (*TestClaims, error) {
	return ParseTestTokenWithSecret(tokenString, DefaultTestSecret)
}

// ParseTestTokenWithSecret parses a test JWT token with a custom secret.
func ParseTestTokenWithSecret(tokenString, secret string) (*TestClaims, error) {
	if secret == "" {
		secret = DefaultTestSecret
	}

	token, err := jwt.ParseWithClaims(tokenString, &TestClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*TestClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// GenerateUserTokens generates both access and refresh tokens for a test user.
func GenerateUserTokens(userID, email string, orgID string, role string, scopes []string) (accessToken, refreshToken string, err error) {
	// Generate access token
	accessOpts := DefaultTokenOptions()
	accessOpts.UserID = userID
	accessOpts.Email = email
	accessOpts.OrgID = orgID
	accessOpts.Role = role
	accessOpts.Scopes = scopes
	accessOpts.TokenType = "access"
	accessOpts.Expiry = DefaultTestAccessDuration

	accessToken, err = GenerateTokenWithOptions(accessOpts)
	if err != nil {
		return "", "", fmt.Errorf("generate access token: %w", err)
	}

	// Generate refresh token
	refreshOpts := accessOpts
	refreshOpts.TokenType = "refresh"
	refreshOpts.Expiry = DefaultTestRefreshDuration
	refreshOpts.Scopes = nil

	refreshToken, err = GenerateTokenWithOptions(refreshOpts)
	if err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// GetTestUserInfo returns a test user info for token generation.
func GetTestUserInfo() (userID, email, orgID string) {
	return uuid.New().String(), "test@example.com", uuid.New().String()
}

// GetAdminUserInfo returns test admin user info for token generation.
func GetAdminUserInfo() (userID, email string) {
	return uuid.New().String(), "admin@example.com"
}

// TokenBuilder provides a fluent interface for building test tokens.
type TokenBuilder struct {
	opts TokenOptions
}

// NewTokenBuilder creates a new token builder.
func NewTokenBuilder() *TokenBuilder {
	return &TokenBuilder{
		opts: DefaultTokenOptions(),
	}
}

// WithSecret sets the token secret.
func (tb *TokenBuilder) WithSecret(secret string) *TokenBuilder {
	tb.opts.Secret = secret
	return tb
}

// WithUserID sets the user ID.
func (tb *TokenBuilder) WithUserID(userID string) *TokenBuilder {
	tb.opts.UserID = userID
	return tb
}

// WithEmail sets the email.
func (tb *TokenBuilder) WithEmail(email string) *TokenBuilder {
	tb.opts.Email = email
	return tb
}

// WithOrgID sets the organization ID.
func (tb *TokenBuilder) WithOrgID(orgID string) *TokenBuilder {
	tb.opts.OrgID = orgID
	return tb
}

// WithRole sets the role.
func (tb *TokenBuilder) WithRole(role string) *TokenBuilder {
	tb.opts.Role = role
	return tb
}

// WithScopes sets the scopes.
func (tb *TokenBuilder) WithScopes(scopes []string) *TokenBuilder {
	tb.opts.Scopes = scopes
	return tb
}

// WithTokenType sets the token type.
func (tb *TokenBuilder) WithTokenType(tokenType string) *TokenBuilder {
	tb.opts.TokenType = tokenType
	return tb
}

// WithExpiry sets the token expiry duration.
func (tb *TokenBuilder) WithExpiry(expiry time.Duration) *TokenBuilder {
	tb.opts.Expiry = expiry
	return tb
}

// WithIssuer sets the issuer.
func (tb *TokenBuilder) WithIssuer(issuer string) *TokenBuilder {
	tb.opts.Issuer = issuer
	return tb
}

// Build generates the token.
func (tb *TokenBuilder) Build() (string, error) {
	return GenerateTokenWithOptions(tb.opts)
}

// MustBuild generates the token and panics on error.
// Useful for test setup where you want to fail fast.
func (tb *TokenBuilder) MustBuild() string {
	token, err := tb.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to build token: %v", err))
	}
	return token
}

// AsAdmin configures the token as an admin token.
func (tb *TokenBuilder) AsAdmin() *TokenBuilder {
	return tb.WithRole("admin").WithScopes([]string{
		"admin", "print:read", "print:write", "print:delete",
		"user:read", "user:write", "printer:read", "printer:write",
	})
}

// AsOrgAdmin configures the token as an org admin token.
func (tb *TokenBuilder) AsOrgAdmin() *TokenBuilder {
	return tb.WithRole("org_admin").WithScopes([]string{
		"print:read", "print:write", "printer:read", "printer:write", "user:read",
	})
}

// AsUser configures the token as a regular user token.
func (tb *TokenBuilder) AsUser() *TokenBuilder {
	return tb.WithRole("user").WithScopes([]string{"print:read", "print:write"})
}

// Expired configures the token to be expired.
func (tb *TokenBuilder) Expired() *TokenBuilder {
	tb.opts.Expiry = -1 * time.Hour
	return tb
}

// ExpiringIn configures the token to expire in the specified duration.
func (tb *TokenBuilder) ExpiringIn(duration time.Duration) *TokenBuilder {
	tb.opts.Expiry = duration
	return tb
}

// getStringValue safely gets a string value from a map.
func getStringValue(m map[string]interface{}, key, defaultValue string) string {
	if v, ok := m[key]; ok {
		if str, ok := v.(string); ok {
			return str
		}
	}
	return defaultValue
}

// Common test scopes for quick access.
var (
	// ScopeReadOnly represents read-only access scopes.
	ScopeReadOnly = []string{"print:read", "printer:read"}
	// ScopeReadWrite represents read-write access scopes.
	ScopeReadWrite = []string{"print:read", "print:write", "printer:read", "printer:write"}
	// ScopeAdmin represents full admin access scopes.
	ScopeAdmin = []string{
		"admin", "print:read", "print:write", "print:delete",
		"user:read", "user:write", "printer:read", "printer:write",
	}
)
