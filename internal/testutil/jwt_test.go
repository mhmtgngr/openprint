// Package testutil tests for JWT utilities
package testutil

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateTestToken(t *testing.T) {
	token, err := GenerateTestToken()
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// Parse and verify the token
	claims, err := ParseTestToken(token)
	require.NoError(t, err)
	assert.Equal(t, "openprint.test", claims.Issuer)
	assert.Equal(t, "access", claims.TokenType)
	assert.Equal(t, "user", claims.Role)
	assert.NotEmpty(t, claims.UserID)
	assert.Equal(t, "test@example.com", claims.Email)
}

func TestGenerateTokenWithOptions(t *testing.T) {
	tests := []struct {
		name    string
		options TokenOptions
		verify  func(*testing.T, string, TokenOptions)
	}{
		{
			name: "default options",
			options: TokenOptions{
				UserID:    "user-123",
				Email:     "custom@example.com",
				Role:      "admin",
				TokenType: "access",
			},
			verify: func(t *testing.T, token string, opts TokenOptions) {
				claims, err := ParseTestToken(token)
				require.NoError(t, err)
				assert.Equal(t, opts.UserID, claims.UserID)
				assert.Equal(t, opts.Email, claims.Email)
				assert.Equal(t, opts.Role, claims.Role)
				assert.Equal(t, opts.TokenType, claims.TokenType)
			},
		},
		{
			name: "with custom secret",
			options: TokenOptions{
				Secret:    "my-custom-secret-key-that-is-long-enough",
				UserID:    "user-456",
				Email:     "secret@example.com",
				TokenType: "refresh",
			},
			verify: func(t *testing.T, token string, opts TokenOptions) {
				// Should parse with custom secret
				claims, err := ParseTestTokenWithSecret(token, opts.Secret)
				require.NoError(t, err)
				assert.Equal(t, opts.UserID, claims.UserID)
			},
		},
		{
			name: "with expiry",
			options: TokenOptions{
				UserID:    "user-789",
				Email:     "expiry@example.com",
				Expiry:    time.Hour,
				TokenType: "access",
			},
			verify: func(t *testing.T, token string, opts TokenOptions) {
				claims, err := ParseTestToken(token)
				require.NoError(t, err)
				assert.True(t, claims.ExpiresAt.Time.After(time.Now()))
				assert.True(t, claims.ExpiresAt.Time.Before(time.Now().Add(2*time.Hour)))
			},
		},
		{
			name: "with scopes",
			options: TokenOptions{
				UserID:    "user-scopes",
				Email:     "scopes@example.com",
				Scopes:    []string{"read", "write", "delete"},
				TokenType: "access",
			},
			verify: func(t *testing.T, token string, opts TokenOptions) {
				claims, err := ParseTestToken(token)
				require.NoError(t, err)
				assert.Equal(t, opts.Scopes, claims.Scopes)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateTokenWithOptions(tt.options)
			require.NoError(t, err)
			require.NotEmpty(t, token)
			if tt.verify != nil {
				tt.verify(t, token, tt.options)
			}
		})
	}
}

func TestGenerateTestAccessToken(t *testing.T) {
	userID := "user-123"
	email := "user@example.com"
	role := "admin"
	orgID := "org-456"
	scopes := []string{"read", "write", "delete"}

	token, err := GenerateTestAccessToken(userID, email, role, orgID, scopes)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := ParseTestToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, role, claims.Role)
	assert.Equal(t, orgID, claims.OrgID)
	assert.Equal(t, scopes, claims.Scopes)
	assert.Equal(t, "access", claims.TokenType)
}

func TestGenerateTestRefreshToken(t *testing.T) {
	userID := "user-123"
	email := "user@example.com"

	token, err := GenerateTestRefreshToken(userID, email)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := ParseTestToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, "refresh", claims.TokenType)
	assert.Nil(t, claims.Scopes)

	// Refresh tokens should have longer expiry
	expectedExpiry := time.Now().Add(DefaultTestRefreshDuration)
	assert.WithinDuration(t, expectedExpiry, claims.ExpiresAt.Time, time.Minute)
}

func TestGenerateAdminToken(t *testing.T) {
	token, err := GenerateAdminToken()
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := ParseTestToken(token)
	require.NoError(t, err)
	assert.Equal(t, "admin", claims.Role)
	assert.Contains(t, claims.Scopes, "admin")
	assert.Contains(t, claims.Scopes, "print:read")
	assert.Contains(t, claims.Scopes, "user:write")
}

func TestGenerateOrgAdminToken(t *testing.T) {
	orgID := "org-123"
	token, err := GenerateOrgAdminToken(orgID)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := ParseTestToken(token)
	require.NoError(t, err)
	assert.Equal(t, "org_admin", claims.Role)
	assert.Equal(t, orgID, claims.OrgID)
	assert.Contains(t, claims.Scopes, "print:read")
	assert.Contains(t, claims.Scopes, "printer:write")
}

func TestGenerateExpiredToken(t *testing.T) {
	token, err := GenerateExpiredToken()
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// Parsing should fail due to expired token
	_, err = ParseTestToken(token)
	assert.Error(t, err)
}

func TestGenerateTokenWithExpiry(t *testing.T) {
	expiry := time.Now().Add(2 * time.Hour)
	token, err := GenerateTokenWithExpiry(expiry)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := ParseTestToken(token)
	require.NoError(t, err)
	assert.WithinDuration(t, expiry, claims.ExpiresAt.Time, time.Minute)
}

func TestGenerateTokenWithCustomClaims(t *testing.T) {
	customClaims := map[string]interface{}{
		"custom_field": "custom_value",
		"number_field": 42,
		"bool_field":   true,
	}

	token, err := GenerateTokenWithCustomClaims(customClaims)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// Parse as map claims to access custom fields
	parsed, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(DefaultTestSecret), nil
	})
	require.NoError(t, err)

	claims := parsed.Claims.(jwt.MapClaims)
	assert.Equal(t, "custom_value", claims["custom_field"])
	assert.Equal(t, float64(42), claims["number_field"])
	assert.Equal(t, true, claims["bool_field"])
}

func TestParseTestToken(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		secret    string
		wantError bool
		verify    func(*testing.T, *TestClaims)
	}{
		{
			name:      "valid token",
			token:     generateValidToken(t, "user-123", "user@example.com"),
			secret:    DefaultTestSecret,
			wantError: false,
			verify: func(t *testing.T, claims *TestClaims) {
				assert.Equal(t, "user-123", claims.UserID)
				assert.Equal(t, "user@example.com", claims.Email)
			},
		},
		{
			name:      "invalid token",
			token:     "invalid.token.string",
			secret:    DefaultTestSecret,
			wantError: true,
		},
		{
			name:      "wrong secret",
			token:     generateValidToken(t, "user-123", "user@example.com"),
			secret:    "wrong-secret",
			wantError: true,
		},
		{
			name:      "empty token",
			token:     "",
			secret:    DefaultTestSecret,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var claims *TestClaims
			var err error

			if tt.secret == DefaultTestSecret || tt.secret == "" {
				claims, err = ParseTestToken(tt.token)
			} else {
				claims, err = ParseTestTokenWithSecret(tt.token, tt.secret)
			}

			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.verify != nil {
					tt.verify(t, claims)
				}
			}
		})
	}
}

func TestGenerateUserTokens(t *testing.T) {
	userID := "user-123"
	email := "user@example.com"
	orgID := "org-456"
	role := "user"
	scopes := []string{"read", "write"}

	accessToken, refreshToken, err := GenerateUserTokens(userID, email, orgID, role, scopes)
	require.NoError(t, err)
	require.NotEmpty(t, accessToken)
	require.NotEmpty(t, refreshToken)

	// Verify access token
	accessClaims, err := ParseTestToken(accessToken)
	require.NoError(t, err)
	assert.Equal(t, userID, accessClaims.UserID)
	assert.Equal(t, email, accessClaims.Email)
	assert.Equal(t, role, accessClaims.Role)
	assert.Equal(t, "access", accessClaims.TokenType)
	assert.Equal(t, scopes, accessClaims.Scopes)

	// Verify refresh token
	refreshClaims, err := ParseTestToken(refreshToken)
	require.NoError(t, err)
	assert.Equal(t, userID, refreshClaims.UserID)
	assert.Equal(t, email, refreshClaims.Email)
	assert.Equal(t, "refresh", refreshClaims.TokenType)
	assert.Nil(t, refreshClaims.Scopes)

	// Verify different token IDs
	assert.NotEqual(t, accessClaims.ID, refreshClaims.ID)
}

func TestGetTestUserInfo(t *testing.T) {
	userID, email, orgID := GetTestUserInfo()
	assert.NotEmpty(t, userID)
	assert.NotEmpty(t, email)
	assert.NotEmpty(t, orgID)
	assert.Equal(t, "test@example.com", email)
}

func TestGetAdminUserInfo(t *testing.T) {
	userID, email := GetAdminUserInfo()
	assert.NotEmpty(t, userID)
	assert.NotEmpty(t, email)
	assert.Equal(t, "admin@example.com", email)
}

func TestTokenBuilder(t *testing.T) {
	t.Run("basic builder", func(t *testing.T) {
		token := NewTokenBuilder().
			WithUserID("user-123").
			WithEmail("builder@example.com").
			WithRole("admin").
			MustBuild()

		claims, err := ParseTestToken(token)
		require.NoError(t, err)
		assert.Equal(t, "user-123", claims.UserID)
		assert.Equal(t, "builder@example.com", claims.Email)
		assert.Equal(t, "admin", claims.Role)
	})

	t.Run("AsAdmin shortcut", func(t *testing.T) {
		token := NewTokenBuilder().AsAdmin().MustBuild()
		claims, err := ParseTestToken(token)
		require.NoError(t, err)
		assert.Equal(t, "admin", claims.Role)
		assert.Contains(t, claims.Scopes, "admin")
	})

	t.Run("AsOrgAdmin shortcut", func(t *testing.T) {
		token := NewTokenBuilder().AsOrgAdmin().MustBuild()
		claims, err := ParseTestToken(token)
		require.NoError(t, err)
		assert.Equal(t, "org_admin", claims.Role)
		assert.NotEmpty(t, claims.OrgID) // Just verify it has an org ID
		assert.Contains(t, claims.Scopes, "print:read")
		assert.Contains(t, claims.Scopes, "printer:write")
	})

	t.Run("AsUser shortcut", func(t *testing.T) {
		token := NewTokenBuilder().AsUser().MustBuild()
		claims, err := ParseTestToken(token)
		require.NoError(t, err)
		assert.Equal(t, "user", claims.Role)
		assert.Contains(t, claims.Scopes, "print:read")
	})

	t.Run("Expired shortcut", func(t *testing.T) {
		token := NewTokenBuilder().Expired().MustBuild()
		_, err := ParseTestToken(token)
		assert.Error(t, err) // Token should be expired
	})

	t.Run("ExpiringIn", func(t *testing.T) {
		token := NewTokenBuilder().ExpiringIn(time.Hour).MustBuild()
		claims, err := ParseTestToken(token)
		require.NoError(t, err)
		expectedExpiry := time.Now().Add(time.Hour)
		assert.WithinDuration(t, expectedExpiry, claims.ExpiresAt.Time, time.Minute)
	})

	t.Run("chaining", func(t *testing.T) {
		userID := "chain-123"
		email := "chain@example.com"
		token := NewTokenBuilder().
			WithUserID(userID).
			WithEmail(email).
			WithRole("user").
			WithScopes([]string{"read", "write"}).
			WithExpiry(30 * time.Minute).
			MustBuild()

		claims, err := ParseTestToken(token)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, "user", claims.Role)
		assert.Contains(t, claims.Scopes, "read")
		assert.Contains(t, claims.Scopes, "write")
	})

	t.Run("Build with error handling", func(t *testing.T) {
		token, err := NewTokenBuilder().
			WithUserID("user-123").
			Build()
		require.NoError(t, err)
		require.NotEmpty(t, token)
	})
}

func TestDefaultTokenOptions(t *testing.T) {
	opts := DefaultTokenOptions()
	assert.NotEmpty(t, opts.Secret)
	assert.NotEmpty(t, opts.UserID)
	assert.NotEmpty(t, opts.Email)
	assert.NotEmpty(t, opts.OrgID)
	assert.NotEmpty(t, opts.Role)
	assert.NotEmpty(t, opts.Scopes)
	assert.NotEmpty(t, opts.TokenType)
	assert.Greater(t, opts.Expiry, time.Duration(0))
	assert.NotEmpty(t, opts.Issuer)
}

func TestCommonTestScopes(t *testing.T) {
	assert.Contains(t, ScopeReadOnly, "print:read")
	assert.Contains(t, ScopeReadWrite, "print:read")
	assert.Contains(t, ScopeReadWrite, "print:write")
	assert.Contains(t, ScopeAdmin, "admin")
	assert.Contains(t, ScopeAdmin, "user:write")
}

func TestTokenWithNotBefore(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	opts := DefaultTokenOptions()
	opts.NotBefore = past

	token, err := GenerateTokenWithOptions(opts)
	require.NoError(t, err)

	claims, err := ParseTestToken(token)
	require.NoError(t, err)
	assert.WithinDuration(t, past, claims.NotBefore.Time, time.Minute)
}

func TestTokenWithCustomIssuer(t *testing.T) {
	customIssuer := "custom.issuer.test"
	opts := DefaultTokenOptions()
	opts.Issuer = customIssuer

	token, err := GenerateTokenWithOptions(opts)
	require.NoError(t, err)

	claims, err := ParseTestToken(token)
	require.NoError(t, err)
	assert.Equal(t, customIssuer, claims.Issuer)
}

// Helper function to generate a valid token for tests
func generateValidToken(t *testing.T, userID, email string) string {
	token, err := GenerateTestAccessToken(userID, email, "user", "org-123", []string{"read"})
	require.NoError(t, err)
	return token
}

func TestTokenSigningMethod(t *testing.T) {
	token, err := GenerateTestToken()
	require.NoError(t, err)

	// Verify the token uses HMAC signing
	parsed, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		assert.True(t, ok, "Token should use HMAC signing method")
		return []byte(DefaultTestSecret), nil
	})
	require.NoError(t, err)
	assert.True(t, parsed.Valid)
}

func TestTokenIDUniqueness(t *testing.T) {
	// Generate multiple tokens and verify they have unique IDs
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		token, err := GenerateTestToken()
		require.NoError(t, err)

		claims, err := ParseTestToken(token)
		require.NoError(t, err)

		// Check for uniqueness
		_, exists := ids[claims.ID]
		assert.False(t, exists, "Token ID should be unique")
		ids[claims.ID] = true
	}
}

func TestTokenTypes(t *testing.T) {
	tests := []struct {
		name      string
		generate  func() (string, error)
		tokenType string
	}{
		{"access token", func() (string, error) { return GenerateTestToken() }, "access"},
		{"refresh token", func() (string, error) { return GenerateTestRefreshToken("user-123", "user@example.com") }, "refresh"},
		{"admin token", func() (string, error) { return GenerateAdminToken() }, "access"},
		{"org admin token", func() (string, error) { return GenerateOrgAdminToken("org-123") }, "access"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := tt.generate()
			require.NoError(t, err)

			claims, err := ParseTestToken(token)
			require.NoError(t, err)
			assert.Equal(t, tt.tokenType, claims.TokenType)
		})
	}
}
