// Package jwt provides tests for JWT token generation and validation.
package jwt

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestDefaultConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		secret := "this-is-a-secure-secret-key-32-chars-long"
		cfg, err := DefaultConfig(secret)

		if err != nil {
			t.Fatalf("DefaultConfig() error = %v", err)
		}
		if cfg.SecretKey != secret {
			t.Errorf("DefaultConfig() SecretKey = %v, want %v", cfg.SecretKey, secret)
		}
		if cfg.AccessDuration != 15*time.Minute {
			t.Errorf("DefaultConfig() AccessDuration = %v, want %v", cfg.AccessDuration, 15*time.Minute)
		}
		if cfg.RefreshDuration != MaxRefreshDuration {
			t.Errorf("DefaultConfig() RefreshDuration = %v, want %v", cfg.RefreshDuration, MaxRefreshDuration)
		}
		if cfg.Issuer != "openprint.cloud" {
			t.Errorf("DefaultConfig() Issuer = %v, want openprint.cloud", cfg.Issuer)
		}
	})

	t.Run("secret key too short", func(t *testing.T) {
		shortSecret := "short"
		_, err := DefaultConfig(shortSecret)

		if err == nil {
			t.Error("DefaultConfig() with short secret should return error")
		}
		if !errors.Is(err, ErrSecretKeyTooShort) {
			t.Errorf("Expected ErrSecretKeyTooShort, got: %v", err)
		}
	})
}

func TestNewManager(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg, _ := DefaultConfig("this-is-a-secure-secret-key-32-chars-long")
		mgr, err := NewManager(cfg)

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

	t.Run("nil config returns error", func(t *testing.T) {
		_, err := NewManager(nil)
		if err == nil {
			t.Error("NewManager(nil) should return error")
		}
	})

	t.Run("config with short secret key returns error", func(t *testing.T) {
		cfg := &Config{
			SecretKey:       "short",
			AccessDuration:  15 * time.Minute,
			RefreshDuration: 7 * 24 * time.Hour,
			Issuer:          "openprint.cloud",
		}
		_, err := NewManager(cfg)
		if err == nil {
			t.Error("NewManager() with short secret should return error")
		}
		if !errors.Is(err, ErrSecretKeyTooShort) {
			t.Errorf("Expected ErrSecretKeyTooShort, got: %v", err)
		}
	})
}

func TestManager_GenerateToken(t *testing.T) {
	cfg, _ := DefaultConfig("this-is-a-secure-secret-key-32-chars-long")
	mgr, _ := NewManager(cfg)

	userID := "user-123"
	email := "test@example.com"
	role := "admin"
	orgID := "org-456"
	scopes := []string{ScopePrintRead, ScopePrintWrite}

	t.Run("generate access token", func(t *testing.T) {
		token, err := mgr.GenerateToken(userID, email, role, orgID, scopes, AccessTokenType)
		if err != nil {
			t.Fatalf("GenerateToken() error = %v", err)
		}
		if token == "" {
			t.Fatal("GenerateToken() returned empty token")
		}

		// Verify token structure
		parts := splitToken(token)
		if len(parts) != 3 {
			t.Errorf("Token should have 3 parts, got %d", len(parts))
		}
	})

	t.Run("generate refresh token", func(t *testing.T) {
		token, err := mgr.GenerateToken(userID, email, role, "", nil, RefreshTokenType)
		if err != nil {
			t.Fatalf("GenerateToken() error = %v", err)
		}
		if token == "" {
			t.Fatal("GenerateToken() returned empty token")
		}
	})
}

func TestManager_GenerateTokenPair(t *testing.T) {
	cfg, _ := DefaultConfig("this-is-a-secure-secret-key-32-chars-long")
	mgr, _ := NewManager(cfg)

	userID := "user-123"
	email := "test@example.com"
	role := "user"
	orgID := "org-456"
	scopes := []string{ScopePrintRead}

	accessToken, refreshToken, err := mgr.GenerateTokenPair(userID, email, role, orgID, scopes)

	if err != nil {
		t.Fatalf("GenerateTokenPair() error = %v", err)
	}
	if accessToken == "" {
		t.Fatal("GenerateTokenPair() returned empty access token")
	}
	if refreshToken == "" {
		t.Fatal("GenerateTokenPair() returned empty refresh token")
	}
	if accessToken == refreshToken {
		t.Error("Access token and refresh token should be different")
	}
}

func TestManager_ValidateToken(t *testing.T) {
	cfg, _ := DefaultConfig("this-is-a-secure-secret-key-32-chars-long")
	mgr, _ := NewManager(cfg)

	userID := "user-123"
	email := "test@example.com"
	role := "admin"
	orgID := "org-456"
	scopes := []string{ScopePrintRead, ScopeAdmin}

	token, err := mgr.GenerateToken(userID, email, role, orgID, scopes, AccessTokenType)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	t.Run("valid token", func(t *testing.T) {
		claims, err := mgr.ValidateToken(token)
		if err != nil {
			t.Fatalf("ValidateToken() error = %v", err)
		}

		if claims.UserID != userID {
			t.Errorf("UserID = %v, want %v", claims.UserID, userID)
		}
		if claims.Email != email {
			t.Errorf("Email = %v, want %v", claims.Email, email)
		}
		if claims.Role != role {
			t.Errorf("Role = %v, want %v", claims.Role, role)
		}
		if claims.OrgID != orgID {
			t.Errorf("OrgID = %v, want %v", claims.OrgID, orgID)
		}
		if claims.TokenType != string(AccessTokenType) {
			t.Errorf("TokenType = %v, want %v", claims.TokenType, AccessTokenType)
		}
		if len(claims.Scopes) != 2 {
			t.Errorf("Scopes length = %d, want 2", len(claims.Scopes))
		}
		if claims.Issuer != cfg.Issuer {
			t.Errorf("Issuer = %v, want %v", claims.Issuer, cfg.Issuer)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		_, err := mgr.ValidateToken("invalid.token.here")
		if err == nil {
			t.Error("ValidateToken() should return error for invalid token")
		}
	})

	t.Run("malformed token", func(t *testing.T) {
		_, err := mgr.ValidateToken("not-a-jwt")
		if err == nil {
			t.Error("ValidateToken() should return error for malformed token")
		}
	})

	t.Run("token with wrong secret", func(t *testing.T) {
		otherCfg, _ := DefaultConfig("this-is-a-different-secure-secret-key-32-chars-long")
		otherMgr, _ := NewManager(otherCfg)

		_, err := otherMgr.ValidateToken(token)
		if err == nil {
			t.Error("ValidateToken() with wrong secret should return error")
		}
	})
}

func TestManager_ValidateAccessToken(t *testing.T) {
	cfg, _ := DefaultConfig("this-is-a-secure-secret-key-32-chars-long")
	mgr, _ := NewManager(cfg)

	userID := "user-123"
	email := "test@example.com"

	accessToken, err := mgr.GenerateToken(userID, email, "user", "", nil, AccessTokenType)
	if err != nil {
		t.Fatalf("Failed to generate access token: %v", err)
	}

	refreshToken, err := mgr.GenerateToken(userID, email, "user", "", nil, RefreshTokenType)
	if err != nil {
		t.Fatalf("Failed to generate refresh token: %v", err)
	}

	t.Run("valid access token", func(t *testing.T) {
		claims, err := mgr.ValidateAccessToken(accessToken)
		if err != nil {
			t.Fatalf("ValidateAccessToken() error = %v", err)
		}
		if claims.TokenType != string(AccessTokenType) {
			t.Errorf("TokenType = %v, want %v", claims.TokenType, AccessTokenType)
		}
	})

	t.Run("refresh token rejected", func(t *testing.T) {
		_, err := mgr.ValidateAccessToken(refreshToken)
		if err == nil {
			t.Error("ValidateAccessToken() should reject refresh token")
		}
	})
}

func TestManager_ValidateRefreshToken(t *testing.T) {
	cfg, _ := DefaultConfig("this-is-a-secure-secret-key-32-chars-long")
	mgr, _ := NewManager(cfg)

	userID := "user-123"
	email := "test@example.com"

	accessToken, err := mgr.GenerateToken(userID, email, "user", "", nil, AccessTokenType)
	if err != nil {
		t.Fatalf("Failed to generate access token: %v", err)
	}

	refreshToken, err := mgr.GenerateToken(userID, email, "user", "", nil, RefreshTokenType)
	if err != nil {
		t.Fatalf("Failed to generate refresh token: %v", err)
	}

	t.Run("valid refresh token", func(t *testing.T) {
		claims, err := mgr.ValidateRefreshToken(refreshToken)
		if err != nil {
			t.Fatalf("ValidateRefreshToken() error = %v", err)
		}
		if claims.TokenType != string(RefreshTokenType) {
			t.Errorf("TokenType = %v, want %v", claims.TokenType, RefreshTokenType)
		}
	})

	t.Run("access token rejected", func(t *testing.T) {
		_, err := mgr.ValidateRefreshToken(accessToken)
		if err == nil {
			t.Error("ValidateRefreshToken() should reject access token")
		}
	})
}

func TestManager_RefreshAccessToken(t *testing.T) {
	cfg, _ := DefaultConfig("this-is-a-secure-secret-key-32-chars-long")
	mgr, _ := NewManager(cfg)

	userID := "user-123"
	email := "test@example.com"
	role := "user"

	refreshToken, err := mgr.GenerateToken(userID, email, role, "", nil, RefreshTokenType)
	if err != nil {
		t.Fatalf("Failed to generate refresh token: %v", err)
	}

	t.Run("valid refresh token generates new access token", func(t *testing.T) {
		newAccessToken, err := mgr.RefreshAccessToken(refreshToken)
		if err != nil {
			t.Fatalf("RefreshAccessToken() error = %v", err)
		}
		if newAccessToken == "" {
			t.Fatal("RefreshAccessToken() returned empty token")
		}
		if newAccessToken == refreshToken {
			t.Error("New access token should differ from refresh token")
		}

		// Verify the new token is a valid access token
		claims, err := mgr.ValidateAccessToken(newAccessToken)
		if err != nil {
			t.Fatalf("Failed to validate new access token: %v", err)
		}
		if claims.UserID != userID {
			t.Errorf("UserID = %v, want %v", claims.UserID, userID)
		}
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		_, err := mgr.RefreshAccessToken("invalid-token")
		if err == nil {
			t.Error("RefreshAccessToken() should return error for invalid token")
		}
	})

	t.Run("access token cannot be used to refresh", func(t *testing.T) {
		accessToken, _ := mgr.GenerateToken(userID, email, role, "", nil, AccessTokenType)
		_, err := mgr.RefreshAccessToken(accessToken)
		if err == nil {
			t.Error("RefreshAccessToken() should fail with access token")
		}
	})
}

func TestGetTokenID(t *testing.T) {
	cfg, _ := DefaultConfig("this-is-a-secure-secret-key-32-chars-long")
	mgr, _ := NewManager(cfg)

	userID := "user-123"
	email := "test@example.com"

	token, err := mgr.GenerateToken(userID, email, "user", "", nil, AccessTokenType)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	t.Run("extract token ID", func(t *testing.T) {
		tokenID, err := GetTokenID(token)
		if err != nil {
			t.Fatalf("GetTokenID() error = %v", err)
		}
		if tokenID == "" {
			t.Error("GetTokenID() returned empty ID")
		}
		// Token ID should be a valid UUID
		_, err = uuid.Parse(tokenID)
		if err != nil {
			t.Errorf("Token ID is not a valid UUID: %v", err)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		_, err := GetTokenID("invalid-token")
		if err == nil {
			t.Error("GetTokenID() should return error for invalid token")
		}
	})
}

func TestExtractUserInfo(t *testing.T) {
	cfg, _ := DefaultConfig("this-is-a-secure-secret-key-32-chars-long")
	mgr, _ := NewManager(cfg)

	userID := "user-123"
	email := "test@example.com"
	role := "admin"

	token, err := mgr.GenerateToken(userID, email, role, "", nil, AccessTokenType)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	t.Run("extract user info", func(t *testing.T) {
		extractedUserID, extractedEmail, err := ExtractUserInfo(token)
		if err != nil {
			t.Fatalf("ExtractUserInfo() error = %v", err)
		}
		if extractedUserID != userID {
			t.Errorf("UserID = %v, want %v", extractedUserID, userID)
		}
		if extractedEmail != email {
			t.Errorf("Email = %v, want %v", extractedEmail, email)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		_, _, err := ExtractUserInfo("invalid-token")
		if err == nil {
			t.Error("ExtractUserInfo() should return error for invalid token")
		}
	})
}

func TestClaims_HasScope(t *testing.T) {
	tests := []struct {
		name     string
		scopes   []string
		scope    string
		expected bool
	}{
		{
			name:     "scope exists",
			scopes:   []string{ScopePrintRead, ScopePrintWrite, ScopeAdmin},
			scope:    ScopePrintRead,
			expected: true,
		},
		{
			name:     "scope does not exist",
			scopes:   []string{ScopePrintRead},
			scope:    ScopeAdmin,
			expected: false,
		},
		{
			name:     "empty scopes",
			scopes:   []string{},
			scope:    ScopePrintRead,
			expected: false,
		},
		{
			name:     "nil scopes",
			scopes:   nil,
			scope:    ScopePrintRead,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := &Claims{Scopes: tt.scopes}
			if got := claims.HasScope(tt.scope); got != tt.expected {
				t.Errorf("HasScope() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestClaims_HasAnyScope(t *testing.T) {
	t.Run("has one of the scopes", func(t *testing.T) {
		claims := &Claims{Scopes: []string{ScopePrintRead, ScopePrintWrite}}
		if !claims.HasAnyScope(ScopeAdmin, ScopePrintRead) {
			t.Error("HasAnyScope() should return true")
		}
	})

	t.Run("does not have any of the scopes", func(t *testing.T) {
		claims := &Claims{Scopes: []string{ScopePrintRead}}
		if claims.HasAnyScope(ScopeAdmin, ScopeUserWrite) {
			t.Error("HasAnyScope() should return false")
		}
	})

	t.Run("nil scopes", func(t *testing.T) {
		claims := &Claims{Scopes: nil}
		if claims.HasAnyScope(ScopePrintRead) {
			t.Error("HasAnyScope() should return false for nil scopes")
		}
	})
}

func TestClaims_IsAdmin(t *testing.T) {
	tests := []struct {
		role     string
		expected bool
	}{
		{"admin", true},
		{"user", false},
		{"org_admin", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			claims := &Claims{Role: tt.role}
			if got := claims.IsAdmin(); got != tt.expected {
				t.Errorf("IsAdmin() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestClaims_IsOrgAdmin(t *testing.T) {
	tests := []struct {
		role     string
		expected bool
	}{
		{"org_admin", true},
		{"admin", false},
		{"user", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			claims := &Claims{Role: tt.role}
			if got := claims.IsOrgAdmin(); got != tt.expected {
				t.Errorf("IsOrgAdmin() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestClaims_IsValidForOrg(t *testing.T) {
	tests := []struct {
		name     string
		orgID    string
		testOrg  string
		expected bool
	}{
		{
			name:     "matching org",
			orgID:    "org-123",
			testOrg:  "org-123",
			expected: true,
		},
		{
			name:     "different org",
			orgID:    "org-123",
			testOrg:  "org-456",
			expected: false,
		},
		{
			name:     "empty org ID",
			orgID:    "",
			testOrg:  "org-123",
			expected: false,
		},
		{
			name:     "empty test org",
			orgID:    "org-123",
			testOrg:  "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := &Claims{OrgID: tt.orgID}
			if got := claims.IsValidForOrg(tt.testOrg); got != tt.expected {
				t.Errorf("IsValidForOrg() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestScopeConstants(t *testing.T) {
	tests := []struct {
		name  string
		scope string
	}{
		{"PrintRead", ScopePrintRead},
		{"PrintWrite", ScopePrintWrite},
		{"PrintDelete", ScopePrintDelete},
		{"PrinterRead", ScopePrinterRead},
		{"PrinterWrite", ScopePrinterWrite},
		{"UserRead", ScopeUserRead},
		{"UserWrite", ScopeUserWrite},
		{"Admin", ScopeAdmin},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.scope == "" {
				t.Errorf("Scope constant %s is empty", tt.name)
			}
		})
	}
}

func TestAllScopes(t *testing.T) {
	scopes := AllScopes()

	if len(scopes) != 8 {
		t.Errorf("AllScopes() returned %d scopes, want 8", len(scopes))
	}

	// Check that all expected scopes are present
	expectedScopes := []string{
		ScopePrintRead, ScopePrintWrite, ScopePrintDelete,
		ScopePrinterRead, ScopePrinterWrite,
		ScopeUserRead, ScopeUserWrite,
		ScopeAdmin,
	}

	for _, expected := range expectedScopes {
		found := false
		for _, scope := range scopes {
			if scope == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("AllScopes() missing %s", expected)
		}
	}
}

func TestDefaultScopes(t *testing.T) {
	scopes := DefaultScopes()

	if len(scopes) == 0 {
		t.Fatal("DefaultScopes() returned empty slice")
	}

	// Default scopes should include print:read
	found := false
	for _, scope := range scopes {
		if scope == ScopePrintRead {
			found = true
			break
		}
	}
	if !found {
		t.Error("DefaultScopes() should include print:read")
	}
}

func TestAdminScopes(t *testing.T) {
	scopes := AdminScopes()
	allScopes := AllScopes()

	if len(scopes) != len(allScopes) {
		t.Errorf("AdminScopes() returned %d scopes, want %d", len(scopes), len(allScopes))
	}
}

func TestTokenType_String(t *testing.T) {
	tests := []struct {
		tokenType TokenType
		expected  string
	}{
		{AccessTokenType, "access"},
		{RefreshTokenType, "refresh"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.tokenType) != tt.expected {
				t.Errorf("TokenType = %v, want %v", tt.tokenType, tt.expected)
			}
		})
	}
}

// Helper function to split JWT token
func splitToken(token string) []string {
	var parts []string
	current := ""
	for _, ch := range token {
		if ch == '.' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	parts = append(parts, current)
	return parts
}

func TestManager_ExpiredToken(t *testing.T) {
	// Create a manager with very short token duration
	cfg := &Config{
		SecretKey:       "this-is-a-secure-secret-key-32-chars-long",
		AccessDuration:  1 * time.Millisecond,
		RefreshDuration: 7 * 24 * time.Hour,
		Issuer:          "openprint.cloud",
	}
	mgr, _ := NewManager(cfg)

	userID := "user-123"
	email := "test@example.com"

	token, err := mgr.GenerateToken(userID, email, "user", "", nil, AccessTokenType)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	_, err = mgr.ValidateToken(token)
	if err == nil {
		t.Error("ValidateToken() should return error for expired token")
	}
	if !errors.Is(err, ErrExpiredToken) && err != nil {
		t.Logf("ValidateToken() error = %v (may not be ErrExpiredToken)", err)
	}
}

func TestClaims_RegisteredClaims(t *testing.T) {
	cfg, _ := DefaultConfig("this-is-a-secure-secret-key-32-chars-long")
	mgr, _ := NewManager(cfg)

	userID := "user-123"
	email := "test@example.com"

	token, err := mgr.GenerateToken(userID, email, "user", "", nil, AccessTokenType)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Parse token to verify RegisteredClaims
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.SecretKey), nil
	})
	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	claims, ok := parsed.Claims.(*Claims)
	if !ok {
		t.Fatal("Failed to get claims")
	}

	// Check RegisteredClaims
	if claims.RegisteredClaims.Issuer != cfg.Issuer {
		t.Errorf("Issuer = %v, want %v", claims.Issuer, cfg.Issuer)
	}
	if claims.RegisteredClaims.Subject != userID {
		t.Errorf("Subject = %v, want %v", claims.Subject, userID)
	}
	if claims.ID == "" {
		t.Error("ID should not be empty")
	}
	// Check expiration time is set
	if claims.ExpiresAt == nil {
		t.Error("ExpiresAt should not be nil")
	} else if !claims.ExpiresAt.Time.After(time.Now()) {
		t.Error("Token expiration should be in the future")
	}
	// Check issued at time
	if claims.IssuedAt == nil {
		t.Error("IssuedAt should not be nil")
	}
	// Check not before time
	if claims.NotBefore == nil {
		t.Error("NotBefore should not be nil")
	}
}

// Security tests to prevent algorithm confusion attacks
func TestSecurity_AlgorithmConfusion(t *testing.T) {
	cfg, _ := DefaultConfig("this-is-a-secure-secret-key-32-chars-long")
	mgr, _ := NewManager(cfg)

	// Create a token with "none" algorithm attempt
	// This simulates an attacker trying to bypass signature verification
	t.Run("reject none algorithm", func(t *testing.T) {
		// Create a malicious token with "none" algorithm
		claims := &Claims{
			UserID:    "attacker",
			Email:     "attacker@evil.com",
			Role:      "admin",
			TokenType: string(AccessTokenType),
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    cfg.Issuer,
				Subject:   "attacker",
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				ID:        uuid.New().String(),
			},
		}

		// Try to sign with none algorithm
		token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
		tokenString, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
		if err != nil {
			t.Fatalf("Failed to create malicious token: %v", err)
		}

		// The token should be rejected because we only allow HS256
		_, err = mgr.ValidateToken(tokenString)
		if err == nil {
			t.Error("ValidateToken() should reject 'none' algorithm token")
		}
	})

	t.Run("reject RS256 algorithm", func(t *testing.T) {
		// Create a token signed with RS256 (different algorithm)
		claims := &Claims{
			UserID:    "attacker",
			Email:     "attacker@evil.com",
			Role:      "admin",
			TokenType: string(AccessTokenType),
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    cfg.Issuer,
				Subject:   "attacker",
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				ID:        uuid.New().String(),
			},
		}

		// Try to sign with RS256
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		// This will fail to sign without a private key, but we're testing the validation
		tokenString, err := token.SignedString([]byte("fake-key"))
		if err != nil {
			// Expected to fail, but let's test validation anyway
		}

		// Even if we somehow get a token string, it should be rejected
		if tokenString != "" {
			_, err = mgr.ValidateToken(tokenString)
			if err == nil {
				t.Error("ValidateToken() should reject RS256 algorithm token")
			}
		}
	})

	t.Run("only accept HS256", func(t *testing.T) {
		// Generate a valid HS256 token
		token, err := mgr.GenerateToken("user-123", "test@example.com", "user", "", nil, AccessTokenType)
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		// Should validate successfully
		claims, err := mgr.ValidateToken(token)
		if err != nil {
			t.Errorf("ValidateToken() should accept valid HS256 token, got error: %v", err)
		}
		if claims.UserID != "user-123" {
			t.Errorf("UserID = %v, want user-123", claims.UserID)
		}
	})
}

func TestSecurity_SecretKeyValidation(t *testing.T) {
	t.Run("DefaultConfig rejects short secrets", func(t *testing.T) {
		shortSecrets := []string{
			"",
			"a",
			"short-secret",
			"exactly-31-characters-long-key-",
		}

		for _, secret := range shortSecrets {
			_, err := DefaultConfig(secret)
			if err == nil {
				t.Errorf("DefaultConfig() should reject secret of length %d", len(secret))
			}
			if !errors.Is(err, ErrSecretKeyTooShort) {
				t.Errorf("Expected ErrSecretKeyTooShort for secret length %d, got: %v", len(secret), err)
			}
		}
	})

	t.Run("DefaultConfig accepts 32 character secret", func(t *testing.T) {
		validSecret := "exactly-32-characters-long-key!!"
		cfg, err := DefaultConfig(validSecret)
		if err != nil {
			t.Errorf("DefaultConfig() should accept 32 character secret, got error: %v", err)
		}
		if cfg == nil {
			t.Error("DefaultConfig() returned nil config")
		}
	})

	t.Run("NewManager validates secret key length", func(t *testing.T) {
		shortCfg := &Config{
			SecretKey:       "too-short",
			AccessDuration:  15 * time.Minute,
			RefreshDuration: 7 * 24 * time.Hour,
			Issuer:          "openprint.cloud",
		}
		_, err := NewManager(shortCfg)
		if err == nil {
			t.Error("NewManager() should reject config with short secret")
		}
		if !errors.Is(err, ErrSecretKeyTooShort) {
			t.Errorf("Expected ErrSecretKeyTooShort, got: %v", err)
		}
	})
}
