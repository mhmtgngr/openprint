// Package jwt provides JWT token generation and validation for OpenPrint authentication.
package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	// MinSecretKeyLength is the minimum required length for JWT secret keys (32 bytes = 256 bits).
	MinSecretKeyLength = 32
	// MaxRefreshDuration is the maximum allowed refresh token duration (48 hours for security).
	MaxRefreshDuration = 48 * time.Hour
)

var (
	// ErrInvalidToken is returned when a token is invalid.
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken is returned when a token has expired.
	ErrExpiredToken = errors.New("token expired")
	// ErrInvalidSigningMethod is returned when the signing method is invalid.
	ErrInvalidSigningMethod = errors.New("invalid signing method")
	// ErrSecretKeyTooShort is returned when the secret key is too short.
	ErrSecretKeyTooShort = errors.New("secret key must be at least 32 characters")
)

// Claims represents the JWT claims structure.
type Claims struct {
	UserID    string   `json:"user_id"`
	Email     string   `json:"email"`
	OrgID     string   `json:"org_id,omitempty"`
	Role      string   `json:"role"`
	TokenType string   `json:"token_type"` // "access" or "refresh"
	Scopes    []string `json:"scopes,omitempty"`
	jwt.RegisteredClaims
}

// TokenType represents the type of JWT token.
type TokenType string

const (
	// AccessTokenType is for short-lived access tokens.
	AccessTokenType TokenType = "access"
	// RefreshTokenType is for long-lived refresh tokens.
	RefreshTokenType TokenType = "refresh"
)

// Config holds JWT configuration.
type Config struct {
	SecretKey        string
	AccessDuration   time.Duration
	RefreshDuration  time.Duration
	Issuer           string
	AllowedAudiences []string // Allowed audiences for JWT validation
	// RequireAudienceValidation forces audience validation even if no allowed audiences are set.
	// SECURITY: This defaults to true for security and cannot be disabled in production.
	// Audience validation prevents tokens issued for one service from being used by another.
	RequireAudienceValidation bool
	// DisableRequireAudienceValidation is an escape hatch ONLY for testing/dev.
	// MUST NOT be set in production deployments.
	DisableRequireAudienceValidation bool
}

// DefaultConfig returns a JWT configuration with sensible defaults.
func DefaultConfig(secretKey string) (*Config, error) {
	if len(secretKey) < MinSecretKeyLength {
		return nil, fmt.Errorf("%w: got %d characters, want at least %d", ErrSecretKeyTooShort, len(secretKey), MinSecretKeyLength)
	}
	return &Config{
		SecretKey:                 secretKey,
		AccessDuration:            15 * time.Minute,
		RefreshDuration:           MaxRefreshDuration, // 48 hours - reduced from 7 days for security
		Issuer:                    "openprint.cloud",
		AllowedAudiences:          []string{"openprint.cloud", "api.openprint.cloud"},
		RequireAudienceValidation: true, // SECURITY: Always validate audience by default
	}, nil
}

// Manager handles JWT token generation and validation.
type Manager struct {
	config *Config
}

// NewManager creates a new JWT manager.
func NewManager(config *Config) (*Manager, error) {
	if config == nil {
		return nil, errors.New("jwt config cannot be nil")
	}
	if len(config.SecretKey) < MinSecretKeyLength {
		return nil, fmt.Errorf("%w: got %d characters, want at least %d", ErrSecretKeyTooShort, len(config.SecretKey), MinSecretKeyLength)
	}

	// SECURITY: Enforce audience validation by default unless explicitly disabled for testing
	// This prevents token reuse across different services/applications
	if !config.DisableRequireAudienceValidation {
		config.RequireAudienceValidation = true
	}

	return &Manager{config: config}, nil
}

// GenerateTokenPair generates both access and refresh tokens for a user.
func (m *Manager) GenerateTokenPair(userID, email, role string, orgID string, scopes []string) (accessToken, refreshToken string, err error) {
	accessToken, err = m.GenerateToken(userID, email, role, orgID, scopes, AccessTokenType)
	if err != nil {
		return "", "", fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err = m.GenerateToken(userID, email, role, orgID, nil, RefreshTokenType)
	if err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// GenerateToken generates a JWT token with proper audience and issuer claims.
func (m *Manager) GenerateToken(userID, email, role string, orgID string, scopes []string, tokenType TokenType) (string, error) {
	now := time.Now()
	var duration time.Duration
	if tokenType == AccessTokenType {
		duration = m.config.AccessDuration
	} else {
		duration = m.config.RefreshDuration
	}

	// Set audience - use the first allowed audience or default to issuer
	audience := m.config.AllowedAudiences
	if len(audience) == 0 {
		if m.config.Issuer != "" {
			audience = []string{m.config.Issuer}
		}
	}

	claims := Claims{
		UserID:    userID,
		Email:     email,
		OrgID:     orgID,
		Role:      role,
		TokenType: string(tokenType),
		Scopes:    scopes,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.config.Issuer,
			Subject:   userID,
			Audience:  audience,
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.config.SecretKey))
}

// ValidateToken validates a JWT token and returns the claims.
// It performs comprehensive validation including signature, expiration, issuer, and audience.
func (m *Manager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Explicitly verify the signing method to prevent algorithm confusion attacks
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidSigningMethod
		}
		return []byte(m.config.SecretKey), nil
	}, jwt.WithValidMethods([]string{"HS256"}))

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Validate issuer claim
	if err := m.validateIssuer(claims); err != nil {
		return nil, err
	}

	// Validate audience claim
	if err := m.validateAudience(claims); err != nil {
		return nil, err
	}

	return claims, nil
}

// validateIssuer validates the issuer claim of the token.
func (m *Manager) validateIssuer(claims *Claims) error {
	// If no issuer configured, skip validation
	if m.config.Issuer == "" {
		return nil
	}

	// Check if issuer matches
	if claims.Issuer != m.config.Issuer {
		return fmt.Errorf("invalid issuer: expected %q, got %q", m.config.Issuer, claims.Issuer)
	}

	return nil
}

// validateAudience validates the audience claim of the token.
func (m *Manager) validateAudience(claims *Claims) error {
	// Skip validation if not required and no allowed audiences configured
	if !m.config.RequireAudienceValidation && len(m.config.AllowedAudiences) == 0 {
		return nil
	}

	// Check if token has any audience
	if len(claims.Audience) == 0 {
		return errors.New("token missing audience claim")
	}

	// If no specific audiences configured, just check that audience is present
	if len(m.config.AllowedAudiences) == 0 {
		return nil
	}

	// Check if token's audience matches any of the allowed audiences
	for _, allowedAud := range m.config.AllowedAudiences {
		for _, tokenAud := range claims.Audience {
			if tokenAud == allowedAud {
				return nil
			}
		}
	}

	return fmt.Errorf("invalid audience: token audiences %v, allowed audiences %v",
		claims.Audience, m.config.AllowedAudiences)
}

// ValidateAccessToken validates an access token specifically.
func (m *Manager) ValidateAccessToken(tokenString string) (*Claims, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != string(AccessTokenType) {
		return nil, errors.New("invalid token type: expected access token")
	}

	return claims, nil
}

// ValidateRefreshToken validates a refresh token specifically.
func (m *Manager) ValidateRefreshToken(tokenString string) (*Claims, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != string(RefreshTokenType) {
		return nil, errors.New("invalid token type: expected refresh token")
	}

	return claims, nil
}

// RefreshAccessToken generates a new access token from a valid refresh token.
func (m *Manager) RefreshAccessToken(refreshToken string) (string, error) {
	claims, err := m.ValidateRefreshToken(refreshToken)
	if err != nil {
		return "", fmt.Errorf("invalid refresh token: %w", err)
	}

	return m.GenerateToken(claims.UserID, claims.Email, claims.Role, claims.OrgID, claims.Scopes, AccessTokenType)
}

// GetTokenID returns the unique ID (jti) from a token without full validation.
func GetTokenID(tokenString string) (string, error) {
	parser := jwt.NewParser()
	claims := &Claims{}

	token, _, err := parser.ParseUnverified(tokenString, claims)
	if err != nil {
		return "", err
	}

	if token == nil {
		return "", ErrInvalidToken
	}

	return claims.ID, nil
}

// ExtractUserInfo extracts basic user info from a token string.
// This does minimal validation - use ValidateToken for full validation.
func ExtractUserInfo(tokenString string) (userID, email string, err error) {
	parser := jwt.NewParser()
	claims := &Claims{}

	_, _, err = parser.ParseUnverified(tokenString, claims)
	if err != nil {
		return "", "", err
	}

	return claims.UserID, claims.Email, nil
}

// HasScope checks if the claims contain a specific scope.
func (c *Claims) HasScope(scope string) bool {
	for _, s := range c.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// HasAnyScope checks if the claims contain any of the specified scopes.
func (c *Claims) HasAnyScope(scopes ...string) bool {
	for _, scope := range scopes {
		if c.HasScope(scope) {
			return true
		}
	}
	return false
}

// IsAdmin checks if the user has admin role.
func (c *Claims) IsAdmin() bool {
	return c.Role == "admin"
}

// IsOrgAdmin checks if the user has organization admin role.
func (c *Claims) IsOrgAdmin() bool {
	return c.Role == "org_admin"
}

// IsValidForOrg checks if the token is valid for the given organization.
func (c *Claims) IsValidForOrg(orgID string) bool {
	if c.OrgID == "" {
		return false
	}
	return c.OrgID == orgID
}

// Scope constants for authorization.
const (
	ScopePrintRead    = "print:read"
	ScopePrintWrite   = "print:write"
	ScopePrintDelete  = "print:delete"
	ScopePrinterRead  = "printer:read"
	ScopePrinterWrite = "printer:write"
	ScopeUserRead     = "user:read"
	ScopeUserWrite    = "user:write"
	ScopeAdmin        = "admin"
)

// AllScopes returns all available scopes.
func AllScopes() []string {
	return []string{
		ScopePrintRead,
		ScopePrintWrite,
		ScopePrintDelete,
		ScopePrinterRead,
		ScopePrinterWrite,
		ScopeUserRead,
		ScopeUserWrite,
		ScopeAdmin,
	}
}

// DefaultScopes returns default scopes for new users.
func DefaultScopes() []string {
	return []string{
		ScopePrintRead,
		ScopePrintWrite,
		ScopePrinterRead,
	}
}

// AdminScopes returns scopes for admin users.
func AdminScopes() []string {
	return AllScopes()
}
