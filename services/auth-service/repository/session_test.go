// Package repository provides tests for session management using Redis.
package repository

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNewSessionRepository(t *testing.T) {
	repo := NewSessionRepository(nil)

	if repo == nil {
		t.Fatal("NewSessionRepository() returned nil")
	}
	if repo.client != nil {
		t.Error("NewSessionRepository() with nil client should have nil client field")
	}
	if repo.prefix != "session:" {
		t.Errorf("NewSessionRepository() prefix = %v, want 'session:'", repo.prefix)
	}
}

func TestSessionRepository_key(t *testing.T) {
	repo := NewSessionRepository(nil)

	tests := []struct {
		name     string
		token    string
		expected string
	}{
		{
			name:     "simple token",
			token:    "abc123",
			expected: "session:abc123",
		},
		{
			name:     "empty token",
			token:    "",
			expected: "session:",
		},
		{
			name:     "uuid token",
			token:    "550e8400-e29b-41d4-a716-446655440000",
			expected: "session:550e8400-e29b-41d4-a716-446655440000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := repo.key(tt.token)
			if result != tt.expected {
				t.Errorf("key() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSessionRepository_userKey(t *testing.T) {
	repo := NewSessionRepository(nil)

	tests := []struct {
		name     string
		userID   string
		expected string
	}{
		{
			name:     "simple user ID",
			userID:   "user-123",
			expected: "user_sessions:user-123",
		},
		{
			name:     "empty user ID",
			userID:   "",
			expected: "user_sessions:",
		},
		{
			name:     "uuid user ID",
			userID:   "550e8400-e29b-41d4-a716-446655440000",
			expected: "user_sessions:550e8400-e29b-41d4-a716-446655440000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := repo.userKey(tt.userID)
			if result != tt.expected {
				t.Errorf("userKey() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSession_Struct(t *testing.T) {
	now := time.Now()
	session := &Session{
		UserID:    "user-123",
		Token:     "token-abc",
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
	}

	if session.UserID != "user-123" {
		t.Error("Session UserID not set correctly")
	}
	if session.Token != "token-abc" {
		t.Error("Session Token not set correctly")
	}
	if session.ExpiresAt.Before(now) {
		t.Error("Session ExpiresAt should be in the future")
	}
	if !session.CreatedAt.Equal(now) {
		t.Error("Session CreatedAt not set correctly")
	}
}

func TestSessionRepository_Store(t *testing.T) {
	// These tests verify method signatures exist
	// In a real test environment, use miniredis or testcontainers Redis

	repo := NewSessionRepository(nil)
	ctx := context.Background()

	t.Run("store session", func(t *testing.T) {
		t.Skip("Requires Redis connection")
		err := repo.Store(ctx, "user-123", "token-abc", 24*time.Hour)
		if err == nil {
			t.Log("Store() succeeded (unexpected without Redis)")
		}
	})

	t.Run("store with zero TTL", func(t *testing.T) {
		t.Skip("Requires Redis connection")
		err := repo.Store(ctx, "user-123", "token-abc", 0)
		if err == nil {
			t.Log("Store() with zero TTL succeeded (unexpected without Redis)")
		}
	})
}

func TestSessionRepository_Get(t *testing.T) {
	t.Skip("Requires Redis connection")
	repo := NewSessionRepository(nil)
	ctx := context.Background()

	_, err := repo.Get(ctx, "token-abc")
	if err == nil {
		t.Log("Get() succeeded (unexpected without Redis)")
	}
}

func TestSessionRepository_GetUserID(t *testing.T) {
	t.Skip("Requires Redis connection")
	repo := NewSessionRepository(nil)
	ctx := context.Background()

	_, err := repo.GetUserID(ctx, "token-abc")
	if err == nil {
		t.Log("GetUserID() succeeded (unexpected without Redis)")
	}
}

func TestSessionRepository_Delete(t *testing.T) {
	t.Skip("Requires Redis connection")
	repo := NewSessionRepository(nil)
	ctx := context.Background()

	err := repo.Delete(ctx, "token-abc")
	if err == nil {
		t.Log("Delete() succeeded (unexpected without Redis)")
	}
}

func TestSessionRepository_DeleteByUserID(t *testing.T) {
	t.Skip("Requires Redis connection")
	repo := NewSessionRepository(nil)
	ctx := context.Background()

	err := repo.DeleteByUserID(ctx, "user-123")
	if err == nil {
		t.Log("DeleteByUserID() succeeded (unexpected without Redis)")
	}
}

func TestSessionRepository_Exists(t *testing.T) {
	t.Skip("Requires Redis connection")
	repo := NewSessionRepository(nil)
	ctx := context.Background()

	_, err := repo.Exists(ctx, "token-abc")
	if err == nil {
		t.Log("Exists() succeeded (unexpected without Redis)")
	}
}

func TestSessionRepository_Refresh(t *testing.T) {
	t.Skip("Requires Redis connection")
	repo := NewSessionRepository(nil)
	ctx := context.Background()

	err := repo.Refresh(ctx, "token-abc", 24*time.Hour)
	if err == nil {
		t.Log("Refresh() succeeded (unexpected without Redis)")
	}
}

func TestSessionRepository_ListUserSessions(t *testing.T) {
	t.Skip("Requires Redis connection")
	repo := NewSessionRepository(nil)
	ctx := context.Background()

	_, err := repo.ListUserSessions(ctx, "user-123")
	if err == nil {
		t.Log("ListUserSessions() succeeded (unexpected without Redis)")
	}
}

func TestSessionRepository_RevokeAll(t *testing.T) {
	t.Skip("Requires Redis connection")
	repo := NewSessionRepository(nil)
	ctx := context.Background()

	err := repo.RevokeAll(ctx)
	if err == nil {
		t.Log("RevokeAll() succeeded (unexpected without Redis)")
	}
}

func TestSessionRepository_CountActiveSessions(t *testing.T) {
	t.Skip("Requires Redis connection")
	repo := NewSessionRepository(nil)
	ctx := context.Background()

	_, err := repo.CountActiveSessions(ctx, "user-123")
	if err == nil {
		t.Log("CountActiveSessions() succeeded (unexpected without Redis)")
	}
}

func TestSessionRepository_CleanupExpired(t *testing.T) {
	t.Skip("Requires Redis connection")
	repo := NewSessionRepository(nil)
	ctx := context.Background()

	err := repo.CleanupExpired(ctx)
	if err == nil {
		t.Log("CleanupExpired() succeeded (unexpected without Redis)")
	}
}

func TestErrSessionNotFound(t *testing.T) {
	if ErrSessionNotFound == nil {
		t.Error("ErrSessionNotFound should not be nil")
	}

	if ErrSessionNotFound.Error() == "" {
		t.Error("ErrSessionNotFound.Error() should not be empty")
	}
}

func TestSession_TTLValues(t *testing.T) {
	tests := []struct {
		name string
		ttl  time.Duration
	}{
		{"1 hour", 1 * time.Hour},
		{"24 hours", 24 * time.Hour},
		{"7 days", 7 * 24 * time.Hour},
		{"30 days", 30 * 24 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ttl < 0 {
				t.Errorf("TTL should be positive, got %v", tt.ttl)
			}
		})
	}
}

func TestSessionRepository_SessionLifecycle(t *testing.T) {
	t.Run("full session lifecycle", func(t *testing.T) {
		t.Skip("Requires Redis connection")
		// This test documents the expected session lifecycle
		// In production, use a real Redis instance

		userID := "user-123"
		token := "session-token-abc"
		ttl := 24 * time.Hour
		ctx := context.Background()
		repo := NewSessionRepository(nil)

		// Store session
		_ = repo.Store(ctx, userID, token, ttl)

		// Get session
		session, _ := repo.Get(ctx, token)
		_ = session

		// Check existence
		exists, _ := repo.Exists(ctx, token)
		_ = exists

		// Refresh TTL
		_ = repo.Refresh(ctx, token, ttl)

		// Get user ID
		uid, _ := repo.GetUserID(ctx, token)
		_ = uid

		// List user sessions
		sessions, _ := repo.ListUserSessions(ctx, userID)
		_ = sessions

		// Count active sessions
		count, _ := repo.CountActiveSessions(ctx, userID)
		_ = count

		// Delete session
		_ = repo.Delete(ctx, token)

		// Verify deleted
		existsAfterDelete, _ := repo.Exists(ctx, token)
		if existsAfterDelete {
			t.Log("Session still exists after delete (unexpected)")
		}
	})
}

func TestSessionRepository_MultipleUserSessions(t *testing.T) {
	t.Run("multiple sessions per user", func(t *testing.T) {
		t.Skip("Requires Redis connection")
		// Test that a user can have multiple sessions
		userID := "user-123"
		tokens := []string{"token-1", "token-2", "token-3"}
		ctx := context.Background()
		repo := NewSessionRepository(nil)

		// Store multiple sessions for same user
		for _, token := range tokens {
			_ = repo.Store(ctx, userID, token, 24*time.Hour)
		}

		// List should return all sessions
		sessions, _ := repo.ListUserSessions(ctx, userID)
		if sessions != nil && len(sessions) > 0 {
			t.Logf("ListUserSessions returned %d sessions", len(sessions))
		}

		// Delete all user sessions
		_ = repo.DeleteByUserID(ctx, userID)

		// All should be deleted
		for _, token := range tokens {
			exists, _ := repo.Exists(ctx, token)
			if exists {
				t.Logf("Token %v still exists after DeleteByUserID", token)
			}
		}
	})
}

func TestSession_KeyFormats(t *testing.T) {
	repo := NewSessionRepository(nil)

	// Test that keys are formatted correctly
	token := "test-token-abc123"
	userID := "user-xyz789"

	sessionKey := repo.key(token)
	userKey := repo.userKey(userID)

	if !strings.HasPrefix(sessionKey, "session:") {
		t.Errorf("Session key should start with 'session:', got %v", sessionKey)
	}
	if !strings.HasPrefix(userKey, "user_sessions:") {
		t.Errorf("User key should start with 'user_sessions:', got %v", userKey)
	}

	if !strings.Contains(sessionKey, token) {
		t.Errorf("Session key should contain token, got %v", sessionKey)
	}
	if !strings.Contains(userKey, userID) {
		t.Errorf("User key should contain user ID, got %v", userKey)
	}
}

func TestSession_Expiration(t *testing.T) {
	t.Run("session expiration calculation", func(t *testing.T) {
		now := time.Now()
		ttl := 24 * time.Hour
		expectedExpiry := now.Add(ttl)

		session := &Session{
			CreatedAt: now,
			ExpiresAt: expectedExpiry,
		}

		timeUntilExpiry := session.ExpiresAt.Sub(session.CreatedAt)
		if timeUntilExpiry != ttl {
			t.Errorf("Time until expiry = %v, want %v", timeUntilExpiry, ttl)
		}
	})

	t.Run("expired session", func(t *testing.T) {
		past := time.Now().Add(-24 * time.Hour)
		session := &Session{
			CreatedAt: past,
			ExpiresAt: past.Add(1 * time.Hour),
		}

		if session.ExpiresAt.After(time.Now()) {
			t.Error("Session should be expired")
		}
	})
}
