// Package repository provides session management using Redis.
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Session represents a user session.
type Session struct {
	UserID    string
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// SessionRepository handles session storage in Redis.
type SessionRepository struct {
	client *redis.Client
	prefix string
}

// NewSessionRepository creates a new session repository.
func NewSessionRepository(client *redis.Client) *SessionRepository {
	return &SessionRepository{
		client: client,
		prefix: "session:",
	}
}

// Store stores a session (refresh token) in Redis.
func (r *SessionRepository) Store(ctx context.Context, userID, token string, ttl time.Duration) error {
	key := r.key(token)

	// Store user ID and creation time
	pipe := r.client.Pipeline()
	pipe.HSet(ctx, key,
		"user_id", userID,
		"created_at", time.Now().Unix(),
	)
	pipe.Expire(ctx, key, ttl)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("store session: %w", err)
	}

	// Also add to user's session list for tracking
	userSessionsKey := r.userKey(userID)
	r.client.SAdd(ctx, userSessionsKey, token)
	r.client.Expire(ctx, userSessionsKey, ttl)

	return nil
}

// Get retrieves a session by token.
func (r *SessionRepository) Get(ctx context.Context, token string) (*Session, error) {
	key := r.key(token)

	data, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if len(data) == 0 {
		return nil, ErrSessionNotFound
	}

	createdAt := time.Now()
	if createdTS, ok := data["created_at"]; ok {
		if unixSec, err := time.ParseDuration(createdTS + "s"); err == nil {
			createdAt = time.Unix(int64(unixSec.Seconds()), 0)
		}
	}

	ttl, _ := r.client.TTL(ctx, key).Result()

	return &Session{
		UserID:    data["user_id"],
		Token:     token,
		CreatedAt: createdAt,
		ExpiresAt: time.Now().Add(ttl),
	}, nil
}

// GetUserID retrieves the user ID for a session token.
func (r *SessionRepository) GetUserID(ctx context.Context, token string) (string, error) {
	key := r.key(token)

	userID, err := r.client.HGet(ctx, key, "user_id").Result()
	if err != nil {
		if err == redis.Nil {
			return "", ErrSessionNotFound
		}
		return "", fmt.Errorf("get user id: %w", err)
	}

	return userID, nil
}

// Delete removes a session by token.
func (r *SessionRepository) Delete(ctx context.Context, token string) error {
	key := r.key(token)

	// Get user ID before deleting
	userID, err := r.client.HGet(ctx, key, "user_id").Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("get user id before delete: %w", err)
	}

	// Delete session
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	// Remove from user's session list
	if userID != "" {
		userSessionsKey := r.userKey(userID)
		r.client.SRem(ctx, userSessionsKey, token)
	}

	return nil
}

// DeleteByUserID removes all sessions for a user.
func (r *SessionRepository) DeleteByUserID(ctx context.Context, userID string) error {
	userSessionsKey := r.userKey(userID)

	// Get all session tokens for the user
	tokens, err := r.client.SMembers(ctx, userSessionsKey).Result()
	if err != nil {
		return fmt.Errorf("get user sessions: %w", err)
	}

	// Delete each session
	for _, token := range tokens {
		key := r.key(token)
		r.client.Del(ctx, key)
	}

	// Clear the user's session set
	r.client.Del(ctx, userSessionsKey)

	return nil
}

// Exists checks if a session exists.
func (r *SessionRepository) Exists(ctx context.Context, token string) (bool, error) {
	key := r.key(token)
	exists, err := r.client.Exists(ctx, key).Result()
	return exists > 0, err
}

// Refresh extends a session's TTL.
func (r *SessionRepository) Refresh(ctx context.Context, token string, ttl time.Duration) error {
	key := r.key(token)
	return r.client.Expire(ctx, key, ttl).Err()
}

// ListUserSessions returns all session tokens for a user.
func (r *SessionRepository) ListUserSessions(ctx context.Context, userID string) ([]string, error) {
	userSessionsKey := r.userKey(userID)
	tokens, err := r.client.SMembers(ctx, userSessionsKey).Result()
	if err != nil {
		return nil, fmt.Errorf("list user sessions: %w", err)
	}

	// Filter out expired tokens
	var validTokens []string
	for _, token := range tokens {
		if exists, _ := r.Exists(ctx, token); exists {
			validTokens = append(validTokens, token)
		}
	}

	// Update the set with only valid tokens
	if len(validTokens) != len(tokens) {
		r.client.Del(ctx, userSessionsKey)
		if len(validTokens) > 0 {
			r.client.SAdd(ctx, userSessionsKey, validTokens)
		}
	}

	return validTokens, nil
}

// RevokeAll revokes all sessions (for logout all devices).
func (r *SessionRepository) RevokeAll(ctx context.Context) error {
	// Get all session keys
	iter := r.client.Scan(ctx, 0, r.prefix+"*", 0).Iterator()
	var keys []string

	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("scan sessions: %w", err)
	}

	if len(keys) > 0 {
		return r.client.Del(ctx, keys...).Err()
	}

	return nil
}

// RevokeToken revokes a specific refresh token by removing it from the session store.
// This is the primary method for token revocation - when a token is deleted from
// the session store, it cannot be used to refresh access tokens even if the JWT
// itself hasn't expired yet.
func (r *SessionRepository) RevokeToken(ctx context.Context, token string) error {
	return r.Delete(ctx, token)
}

// IsTokenRevoked checks if a token has been revoked (deleted from session store).
// This provides a blacklist mechanism for tokens that are still within their
// validity period but have been explicitly revoked.
func (r *SessionRepository) IsTokenRevoked(ctx context.Context, token string) (bool, error) {
	exists, err := r.Exists(ctx, token)
	if err != nil {
		return false, fmt.Errorf("check token revocation status: %w", err)
	}
	// Token is revoked if it does NOT exist in the session store
	return !exists, nil
}

// RevokeUserTokens revokes all tokens for a specific user.
func (r *SessionRepository) RevokeUserTokens(ctx context.Context, userID string) error {
	return r.DeleteByUserID(ctx, userID)
}

// CountActiveSessions returns the number of active sessions for a user.
func (r *SessionRepository) CountActiveSessions(ctx context.Context, userID string) (int, error) {
	tokens, err := r.ListUserSessions(ctx, userID)
	if err != nil {
		return 0, err
	}
	return len(tokens), nil
}

// CleanupExpired removes expired sessions from user session sets.
func (r *SessionRepository) CleanupExpired(ctx context.Context) error {
	// Get all user session keys
	iter := r.client.Scan(ctx, 0, "user_sessions:*", 0).Iterator()

	for iter.Next(ctx) {
		userSessionsKey := iter.Val()

		// Get all tokens
		tokens, err := r.client.SMembers(ctx, userSessionsKey).Result()
		if err != nil {
			continue
		}

		// Check each token
		var validTokens []string
		for _, token := range tokens {
			if exists, _ := r.Exists(ctx, token); exists {
				validTokens = append(validTokens, token)
			}
		}

		// Update set with only valid tokens
		if len(validTokens) == 0 {
			r.client.Del(ctx, userSessionsKey)
		} else if len(validTokens) != len(tokens) {
			r.client.Del(ctx, userSessionsKey)
			// Convert []string to []interface{} for SAdd
			members := make([]interface{}, len(validTokens))
			for i, token := range validTokens {
				members[i] = token
			}
			r.client.SAdd(ctx, userSessionsKey, members...)
		}
	}

	return iter.Err()
}

// key returns the Redis key for a session token.
func (r *SessionRepository) key(token string) string {
	return r.prefix + token
}

// userKey returns the Redis key for a user's session set.
func (r *SessionRepository) userKey(userID string) string {
	return "user_sessions:" + userID
}

// ErrSessionNotFound is returned when a session is not found.
var ErrSessionNotFound error = fmt.Errorf("session not found")
