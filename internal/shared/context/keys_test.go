package context

import (
	"context"
	"testing"
)

func TestGetUserID(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{
			name:     "returns user ID from context",
			ctx:      context.WithValue(context.Background(), UserIDKey, "user-123"),
			expected: "user-123",
		},
		{
			name:     "returns empty string when not set",
			ctx:      context.Background(),
			expected: "",
		},
		{
			name:     "returns empty string for non-string value",
			ctx:      context.WithValue(context.Background(), UserIDKey, 123),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetUserID(tt.ctx)
			if result != tt.expected {
				t.Errorf("GetUserID() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetEmail(t *testing.T) {
	ctx := context.WithValue(context.Background(), EmailKey, "user@example.com")
	if got := GetEmail(ctx); got != "user@example.com" {
		t.Errorf("GetEmail() = %q, want %q", got, "user@example.com")
	}

	if got := GetEmail(context.Background()); got != "" {
		t.Errorf("GetEmail() on empty context = %q, want empty", got)
	}
}

func TestGetOrgID(t *testing.T) {
	ctx := context.WithValue(context.Background(), OrgIDKey, "org-456")
	if got := GetOrgID(ctx); got != "org-456" {
		t.Errorf("GetOrgID() = %q, want %q", got, "org-456")
	}

	if got := GetOrgID(context.Background()); got != "" {
		t.Errorf("GetOrgID() on empty context = %q, want empty", got)
	}
}

func TestGetRole(t *testing.T) {
	ctx := context.WithValue(context.Background(), RoleKey, "admin")
	if got := GetRole(ctx); got != "admin" {
		t.Errorf("GetRole() = %q, want %q", got, "admin")
	}

	if got := GetRole(context.Background()); got != "" {
		t.Errorf("GetRole() on empty context = %q, want empty", got)
	}
}

func TestGetScopes(t *testing.T) {
	scopes := []string{"read", "write"}
	ctx := context.WithValue(context.Background(), ScopesKey, scopes)
	got := GetScopes(ctx)
	if len(got) != 2 || got[0] != "read" || got[1] != "write" {
		t.Errorf("GetScopes() = %v, want %v", got, scopes)
	}

	if got := GetScopes(context.Background()); got != nil {
		t.Errorf("GetScopes() on empty context = %v, want nil", got)
	}
}

func TestGetRequestID(t *testing.T) {
	ctx := context.WithValue(context.Background(), RequestIDKey, "req-789")
	if got := GetRequestID(ctx); got != "req-789" {
		t.Errorf("GetRequestID() = %q, want %q", got, "req-789")
	}

	if got := GetRequestID(context.Background()); got != "" {
		t.Errorf("GetRequestID() on empty context = %q, want empty", got)
	}
}

func TestWithUserContext(t *testing.T) {
	userCtx := &UserContext{
		UserID: "user-1",
		Email:  "test@example.com",
		OrgID:  "org-1",
		Role:   "admin",
		Scopes: []string{"read", "write"},
		Token:  "token-abc",
	}

	ctx := WithUserContext(context.Background(), userCtx)

	if got := GetUserID(ctx); got != "user-1" {
		t.Errorf("UserID = %q, want %q", got, "user-1")
	}
	if got := GetEmail(ctx); got != "test@example.com" {
		t.Errorf("Email = %q, want %q", got, "test@example.com")
	}
	if got := GetOrgID(ctx); got != "org-1" {
		t.Errorf("OrgID = %q, want %q", got, "org-1")
	}
	if got := GetRole(ctx); got != "admin" {
		t.Errorf("Role = %q, want %q", got, "admin")
	}
	scopes := GetScopes(ctx)
	if len(scopes) != 2 {
		t.Errorf("Scopes length = %d, want 2", len(scopes))
	}
}

func TestWithUserID(t *testing.T) {
	ctx := WithUserID(context.Background(), "user-42")
	if got := GetUserID(ctx); got != "user-42" {
		t.Errorf("WithUserID then GetUserID = %q, want %q", got, "user-42")
	}
}

func TestWithEmail(t *testing.T) {
	ctx := WithEmail(context.Background(), "hello@test.com")
	if got := GetEmail(ctx); got != "hello@test.com" {
		t.Errorf("WithEmail then GetEmail = %q, want %q", got, "hello@test.com")
	}
}

func TestWithOrgID(t *testing.T) {
	ctx := WithOrgID(context.Background(), "org-99")
	if got := GetOrgID(ctx); got != "org-99" {
		t.Errorf("WithOrgID then GetOrgID = %q, want %q", got, "org-99")
	}
}

func TestWithRole(t *testing.T) {
	ctx := WithRole(context.Background(), "viewer")
	if got := GetRole(ctx); got != "viewer" {
		t.Errorf("WithRole then GetRole = %q, want %q", got, "viewer")
	}
}

func TestWithScopes(t *testing.T) {
	ctx := WithScopes(context.Background(), []string{"admin"})
	got := GetScopes(ctx)
	if len(got) != 1 || got[0] != "admin" {
		t.Errorf("WithScopes then GetScopes = %v, want [admin]", got)
	}
}

func TestWithToken(t *testing.T) {
	ctx := WithToken(context.Background(), "my-token")
	val := ctx.Value(TokenKey)
	if val != "my-token" {
		t.Errorf("WithToken value = %v, want %q", val, "my-token")
	}
}

func TestWithRequestID(t *testing.T) {
	ctx := WithRequestID(context.Background(), "req-abc")
	if got := GetRequestID(ctx); got != "req-abc" {
		t.Errorf("WithRequestID then GetRequestID = %q, want %q", got, "req-abc")
	}
}

func TestContextKeyType(t *testing.T) {
	// Verify that our custom type prevents collisions with plain string keys
	ctx := context.WithValue(context.Background(), UserIDKey, "typed-value")
	ctx = context.WithValue(ctx, "user_id", "string-value")

	if got := GetUserID(ctx); got != "typed-value" {
		t.Errorf("GetUserID should use typed key, got %q, want %q", got, "typed-value")
	}
}
