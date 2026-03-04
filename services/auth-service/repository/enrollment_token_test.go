// Package repository provides unit tests for enrollment token operations.
package repository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openprint/openprint/internal/testutil"
)

// testDB holds the test database connection and cleanup function.
type testDB struct {
	pool    *pgxpool.Pool
	cleanup func()
}

// setupEnrollmentTokenTestDBFull creates a test database connection with environment-aware DSN.
// It checks for DATABASE_URL env var, then defaults to localhost:15432 (Docker external port).
func setupEnrollmentTokenTestDBFull(t *testing.T) *testDB {
	t.Helper()

	ctx := context.Background()
	var dbURL string

	// Check for DATABASE_URL environment variable first
	if envURL := os.Getenv("DATABASE_URL"); envURL != "" {
		dbURL = envURL
	} else {
		// Default to Docker external port for PostgreSQL
		dbURL = "postgres://openprint:openprint@localhost:15432/openprint?sslmode=disable"
	}

	// Try to connect to the database
	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		t.Skipf("unable to parse database URL: %v", err)
		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Skipf("database not available for testing: %v", err)
		return nil
	}

	// Verify connection is alive
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("database ping failed: %v", err)
		return nil
	}

	// Ensure uuid-ossp extension is available (required for gen_random_uuid)
	if _, err := pool.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\""); err != nil {
		pool.Close()
		t.Skipf("failed to create uuid-ossp extension: %v", err)
		return nil
	}

	// Create the enrollment_tokens table using the migration system
	if err := testutil.CreateSchema(ctx, pool); err != nil {
		pool.Close()
		t.Skipf("schema setup: %v", err)
		return nil
	}

	// Clean up test data from previous runs to avoid duplicate key errors
	if _, err := pool.Exec(ctx, "DELETE FROM enrollment_tokens WHERE token LIKE 'test-%' OR token LIKE 'find-%' OR token LIKE 'valid-%' OR token LIKE 'expired-%' OR token LIKE 'revoke-%' OR token LIKE 'max-%' OR token LIKE 'org-%' OR token LIKE 'increment-%' OR token LIKE 'list-%' OR token LIKE 'other-%'"); err != nil {
		pool.Close()
		t.Skipf("failed to clean up test data: %v", err)
		return nil
	}

	// Cleanup function to close the pool
	cleanup := func() {
		pool.Close()
	}

	return &testDB{
		pool:    pool,
		cleanup: cleanup,
	}
}

// teardownEnrollmentTokenTestDB closes the test database connection.
func teardownEnrollmentTokenTestDB(t *testing.T, db *testDB) {
	t.Helper()
	if db != nil && db.cleanup != nil {
		db.cleanup()
	}
}

// TestEnrollmentTokenRepository_Create tests creating a new enrollment token.
func TestEnrollmentTokenRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	db := setupEnrollmentTokenTestDBFull(t)
	defer teardownEnrollmentTokenTestDB(t, db)
	require.NotNil(t, db.pool, "database pool should not be nil")

	repo := NewEnrollmentTokenRepository(db.pool)

	token := &EnrollmentToken{
		Token:          "test-token-" + time.Now().Format("20060102150405"),
		OrganizationID: "org-123",
		Name:           "Test Token",
		CreatedBy:      "admin@example.com",
		MaxUses:        10,
	}

	err := repo.Create(ctx, token)
	require.NoError(t, err)
	assert.NotEmpty(t, token.ID)
	assert.NotEmpty(t, token.CreatedAt)
	assert.Equal(t, 0, token.UseCount)
}

// TestEnrollmentTokenRepository_FindByToken tests finding an enrollment token.
func TestEnrollmentTokenRepository_FindByToken(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	db := setupEnrollmentTokenTestDBFull(t)
	defer teardownEnrollmentTokenTestDB(t, db)
	require.NotNil(t, db.pool, "database pool should not be nil")

	repo := NewEnrollmentTokenRepository(db.pool)

	// Create a token first
	token := &EnrollmentToken{
		Token:          "find-test-token",
		OrganizationID: "org-123",
		Name:           "Find Test Token",
		CreatedBy:      "admin@example.com",
		MaxUses:        5,
	}
	err := repo.Create(ctx, token)
	require.NoError(t, err)

	// Find the token
	found, err := repo.FindByToken(ctx, "find-test-token")
	require.NoError(t, err)
	assert.Equal(t, token.Token, found.Token)
	assert.Equal(t, token.OrganizationID, found.OrganizationID)
	assert.Equal(t, token.Name, found.Name)
}

// TestEnrollmentTokenRepository_Validate tests token validation.
func TestEnrollmentTokenRepository_Validate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	db := setupEnrollmentTokenTestDBFull(t)
	defer teardownEnrollmentTokenTestDB(t, db)
	require.NotNil(t, db.pool, "database pool should not be nil")

	repo := NewEnrollmentTokenRepository(db.pool)

	t.Run("valid token", func(t *testing.T) {
		token := &EnrollmentToken{
			Token:          "valid-token",
			OrganizationID: "org-123",
			Name:           "Valid Token",
			CreatedBy:      "admin@example.com",
			MaxUses:        10,
		}
		err := repo.Create(ctx, token)
		require.NoError(t, err)

		valid, err := repo.Validate(ctx, "valid-token", "org-123")
		require.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("non-existent token", func(t *testing.T) {
		valid, err := repo.Validate(ctx, "non-existent", "org-123")
		require.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("expired token", func(t *testing.T) {
		past := time.Now().Add(-1 * time.Hour)
		token := &EnrollmentToken{
			Token:          "expired-token",
			OrganizationID: "org-123",
			Name:           "Expired Token",
			CreatedBy:      "admin@example.com",
			ExpiresAt:      &past,
		}
		err := repo.Create(ctx, token)
		require.NoError(t, err)

		valid, err := repo.Validate(ctx, "expired-token", "org-123")
		require.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("revoked token", func(t *testing.T) {
		token := &EnrollmentToken{
			Token:          "revoke-test-token",
			OrganizationID: "org-123",
			Name:           "Revoke Test Token",
			CreatedBy:      "admin@example.com",
			MaxUses:        10,
		}
		err := repo.Create(ctx, token)
		require.NoError(t, err)

		// Revoke the token
		err = repo.Revoke(ctx, token.ID, "admin@example.com")
		require.NoError(t, err)

		valid, err := repo.Validate(ctx, "revoke-test-token", "org-123")
		require.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("max uses exceeded", func(t *testing.T) {
		token := &EnrollmentToken{
			Token:          "max-uses-token",
			OrganizationID: "org-123",
			Name:           "Max Uses Token",
			CreatedBy:      "admin@example.com",
			MaxUses:        2,
		}
		err := repo.Create(ctx, token)
		require.NoError(t, err)

		// Use the token twice
		err = repo.IncrementUseCount(ctx, token.ID)
		require.NoError(t, err)
		err = repo.IncrementUseCount(ctx, token.ID)
		require.NoError(t, err)

		// Now it should be invalid
		valid, err := repo.Validate(ctx, "max-uses-token", "org-123")
		require.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("wrong organization", func(t *testing.T) {
		token := &EnrollmentToken{
			Token:          "org-specific-token",
			OrganizationID: "org-123",
			Name:           "Org Specific Token",
			CreatedBy:      "admin@example.com",
			MaxUses:        10,
		}
		err := repo.Create(ctx, token)
		require.NoError(t, err)

		// Try to validate with different org
		valid, err := repo.Validate(ctx, "org-specific-token", "org-456")
		require.NoError(t, err)
		assert.False(t, valid)
	})
}

// TestEnrollmentTokenRepository_IncrementUseCount tests incrementing use count.
func TestEnrollmentTokenRepository_IncrementUseCount(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	db := setupEnrollmentTokenTestDBFull(t)
	defer teardownEnrollmentTokenTestDB(t, db)
	require.NotNil(t, db.pool, "database pool should not be nil")

	repo := NewEnrollmentTokenRepository(db.pool)

	token := &EnrollmentToken{
		Token:          "increment-test-token",
		OrganizationID: "org-123",
		Name:           "Increment Test Token",
		CreatedBy:      "admin@example.com",
		MaxUses:        10,
	}
	err := repo.Create(ctx, token)
	require.NoError(t, err)

	// Increment twice
	err = repo.IncrementUseCount(ctx, token.ID)
	require.NoError(t, err)

	err = repo.IncrementUseCount(ctx, token.ID)
	require.NoError(t, err)

	// Verify count
	updated, err := repo.FindByToken(ctx, "increment-test-token")
	require.NoError(t, err)
	assert.Equal(t, 2, updated.UseCount)
}

// TestEnrollmentTokenRepository_Revoke tests revoking a token.
func TestEnrollmentTokenRepository_Revoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	db := setupEnrollmentTokenTestDBFull(t)
	defer teardownEnrollmentTokenTestDB(t, db)
	require.NotNil(t, db.pool, "database pool should not be nil")

	repo := NewEnrollmentTokenRepository(db.pool)

	token := &EnrollmentToken{
		Token:          "revoke-token",
		OrganizationID: "org-123",
		Name:           "Revoke Token",
		CreatedBy:      "admin@example.com",
		MaxUses:        10,
	}
	err := repo.Create(ctx, token)
	require.NoError(t, err)

	// Revoke the token
	err = repo.Revoke(ctx, token.ID, "admin@example.com")
	require.NoError(t, err)

	// Verify it's revoked
	updated, err := repo.FindByToken(ctx, "revoke-token")
	require.NoError(t, err)
	assert.NotNil(t, updated.RevokedAt)
	assert.Equal(t, "admin@example.com", updated.RevokedBy)
}

// TestEnrollmentTokenRepository_List tests listing tokens.
func TestEnrollmentTokenRepository_List(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	db := setupEnrollmentTokenTestDBFull(t)
	defer teardownEnrollmentTokenTestDB(t, db)
	require.NotNil(t, db.pool, "database pool should not be nil")

	repo := NewEnrollmentTokenRepository(db.pool)

	// Create tokens for two orgs
	for i := 0; i < 3; i++ {
		token := &EnrollmentToken{
			Token:          fmt.Sprintf("list-test-token-%c", 'a'+i),
			OrganizationID: "org-123",
			Name:           "List Test Token",
			CreatedBy:      "admin@example.com",
			MaxUses:        10,
		}
		err := repo.Create(ctx, token)
		require.NoError(t, err)
	}

	// Create token for different org
	token := &EnrollmentToken{
		Token:          "other-org-token",
		OrganizationID: "org-456",
		Name:           "Other Org Token",
		CreatedBy:      "admin@example.com",
		MaxUses:        10,
	}
	err := repo.Create(ctx, token)
	require.NoError(t, err)

	// List for specific org
	tokens, err := repo.List(ctx, "org-123", false, 100)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(tokens), 3)
}

// setupEnrollmentTokenTestDB creates a test database connection.
// Deprecated: Use setupEnrollmentTokenTestDBFull instead.
func setupEnrollmentTokenTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	db := setupEnrollmentTokenTestDBFull(t)
	if db == nil {
		return nil
	}
	return db.pool
}
