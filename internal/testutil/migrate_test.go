//go:build integration

package testutil

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunMigrations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Verify migrations table exists
	var tableExists bool
	err = db.Pool.QueryRow(ctx,
		"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = '_test_migrations')",
	).Scan(&tableExists)
	require.NoError(t, err)
	assert.True(t, tableExists, "migrations tracking table should exist")

	// Verify some tables were created by migrations
	var tableCount int
	err = db.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name LIKE '%'",
	).Scan(&tableCount)
	require.NoError(t, err)
	assert.Greater(t, tableCount, 1, "migrations should have created tables")
}

func TestRunMigrations_Idempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// First run
	db1, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db1)

	// Get migration count after first run
	var count1 int
	err = db1.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM _test_migrations").Scan(&count1)
	require.NoError(t, err)

	// Create another database instance (simulates re-running tests)
	db2, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	Cleanup(db2)

	// Get migration count after second run
	var count2 int
	err = db2.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM _test_migrations").Scan(&count2)
	require.NoError(t, err)

	// Migration counts should be the same
	assert.Equal(t, count1, count2)
}

func TestCreateMigrationsTable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Verify table structure
	var columnName, dataType string
	err = db.Pool.QueryRow(ctx,
		"SELECT column_name, data_type FROM information_schema.columns WHERE table_name = '_test_migrations' AND column_name = 'name'",
	).Scan(&columnName, &dataType)
	require.NoError(t, err)
	assert.Equal(t, "name", columnName)
	assert.Equal(t, "character varying", dataType)
}

func TestGetAppliedMigrations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Record a test migration
	_, err = db.Pool.Exec(ctx, "INSERT INTO _test_migrations (name) VALUES ($1)", "test_migration.up.sql")
	require.NoError(t, err)

	// Get applied migrations
	applied, err := getAppliedMigrations(ctx, db.Pool)
	require.NoError(t, err)
	assert.True(t, applied["test_migration.up.sql"])
	assert.False(t, applied["non_existent.sql"])
}

func TestExecuteMigration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Create a test table using executeMigration
	testSQL := `
		CREATE TABLE IF NOT EXISTS test_table (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100)
		)
	`
	err = executeMigration(ctx, db.Pool, testSQL)
	require.NoError(t, err)

	// Verify table exists
	var exists bool
	err = db.Pool.QueryRow(ctx,
		"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'test_table')",
	).Scan(&exists)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestExecuteMigration_TransactionRollback(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Try to execute invalid SQL (should fail and rollback)
	invalidSQL := `
		CREATE TABLE test_table (id SERIAL PRIMARY KEY);
		INSERT INTO non_existent_table VALUES (1);
	`
	err = executeMigration(ctx, db.Pool, invalidSQL)
	assert.Error(t, err)

	// Verify test_table was not created (transaction rolled back)
	var exists bool
	err = db.Pool.QueryRow(ctx,
		"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'test_table')",
	).Scan(&exists)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestRecordMigration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Record a migration
	err = recordMigration(ctx, db.Pool, "001_test.up.sql")
	require.NoError(t, err)

	// Verify it was recorded
	var name string
	err = db.Pool.QueryRow(ctx, "SELECT name FROM _test_migrations WHERE name = $1", "001_test.up.sql").Scan(&name)
	require.NoError(t, err)
	assert.Equal(t, "001_test.up.sql", name)
}

func TestRecordMigration_Duplicate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Record a migration twice
	err = recordMigration(ctx, db.Pool, "001_test.up.sql")
	require.NoError(t, err)

	err = recordMigration(ctx, db.Pool, "001_test.up.sql")
	assert.Error(t, err) // Should fail due to unique constraint
}

func TestResetMigrations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Verify migrations exist
	var count int
	err = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM _test_migrations").Scan(&count)
	require.NoError(t, err)
	assert.Greater(t, count, 0)

	// Reset migrations
	err = ResetMigrations(ctx, db.Pool)
	require.NoError(t, err)

	// Verify table is gone
	var exists bool
	err = db.Pool.QueryRow(ctx,
		"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = '_test_migrations')",
	).Scan(&exists)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestExecuteMigrationFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Find the migrations directory
	projectRoot, err := findProjectRoot()
	require.NoError(t, err)
	migrationsDir := filepath.Join(projectRoot, "migrations")

	// List migration files
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.up.sql"))
	require.NoError(t, err)

	if len(files) == 0 {
		t.Skip("no migration files found")
	}

	// Execute first migration file
	firstFile := filepath.Base(files[0])
	err = ExecuteMigrationFile(ctx, db.Pool, migrationsDir, firstFile)
	// May fail if already applied, that's okay
	if err != nil {
		// Verify it's because of duplicate record, not SQL error
		var exists int
		_ = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM _test_migrations WHERE name = $1", firstFile).Scan(&exists)
		if exists == 0 {
			t.Errorf("execute migration file failed: %v", err)
		}
	}
}

func TestExecuteMigrationFile_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	projectRoot, err := findProjectRoot()
	require.NoError(t, err)
	migrationsDir := filepath.Join(projectRoot, "migrations")

	// Try to execute non-existent file
	err = ExecuteMigrationFile(ctx, db.Pool, migrationsDir, "non_existent.up.sql")
	assert.Error(t, err)
}

func TestExecuteSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Execute custom SQL
	err = ExecuteSQL(ctx, db.Pool, "CREATE TABLE custom_table (id SERIAL PRIMARY KEY)")
	require.NoError(t, err)

	// Verify table exists
	var exists bool
	err = db.Pool.QueryRow(ctx,
		"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'custom_table')",
	).Scan(&exists)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestExecuteSQL_Error(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Execute invalid SQL
	err = ExecuteSQL(ctx, db.Pool, "INVALID SQL STATEMENT")
	assert.Error(t, err)
}

func TestCreateSchema(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Verify schema was created (via migrations in SetupPostgresContainer)
	// This is mostly a smoke test since SetupPostgresContainer already runs migrations
	var tableCount int
	err = db.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'",
	).Scan(&tableCount)
	require.NoError(t, err)
	assert.Greater(t, tableCount, 0)
}

func TestMigrationOrder(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Query migrations in order
	rows, err := db.Pool.Query(ctx, "SELECT name FROM _test_migrations ORDER BY id")
	require.NoError(t, err)
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		err = rows.Scan(&name)
		require.NoError(t, err)
		names = append(names, name)
	}

	require.NoError(t, rows.Err())

	// Verify names are in order (numerical prefix)
	if len(names) > 1 {
		for i := 1; i < len(names); i++ {
			// Names should be sorted alphabetically, which corresponds to numerical order
			assert.Less(t, names[i-1], names[i])
		}
	}
}

func TestMigrationFilesExist(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Find the migrations directory
	projectRoot, err := findProjectRoot()
	require.NoError(t, err)
	migrationsDir := filepath.Join(projectRoot, "migrations")

	// Check if migrations directory exists
	info, err := os.Stat(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("migrations directory not found")
		}
		t.Fatalf("error checking migrations directory: %v", err)
	}

	if !info.IsDir() {
		t.Fatalf("migrations path is not a directory")
	}

	// List migration files
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.up.sql"))
	require.NoError(t, err)
	t.Logf("Found %d migration files", len(files))

	// Verify at least one migration file exists
	assert.Greater(t, len(files), 0, "expected at least one migration file")

	// Verify file names follow the pattern
	for _, file := range files {
		base := filepath.Base(file)
		assert.Regexp(t, `^\d+.*\.up\.sql$`, base, "migration file should follow NNN_description.up.sql pattern")
		t.Logf("Migration: %s", base)
	}
}

func TestMigrationExecution_ContextTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Create a context with very short timeout
	shortCtx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Try to execute migration with short context
	err = executeMigration(shortCtx, db.Pool, "SELECT 1")
	assert.Error(t, err)
}

func TestMigrationWithComplexSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Complex SQL with multiple statements and functions
	complexSQL := `
		CREATE TABLE test_users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);

		CREATE OR REPLACE FUNCTION test_get_user_count() RETURNS INTEGER AS $$
			BEGIN
				RETURN (SELECT COUNT(*) FROM test_users);
			END;
		$$ LANGUAGE plpgsql;

		INSERT INTO test_users (email) VALUES ('test@example.com');
	`

	err = executeMigration(ctx, db.Pool, complexSQL)
	require.NoError(t, err)

	// Verify function works
	var count int
	err = db.Pool.QueryRow(ctx, "SELECT test_get_user_count()").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Cleanup
	_, _ = db.Pool.Exec(ctx, "DROP TABLE test_users CASCADE")
	_, _ = db.Pool.Exec(ctx, "DROP FUNCTION test_get_user_count()")
}

func TestRunMigrations_EmptyDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Create a temporary empty directory
	tmpDir := t.TempDir()

	// Run migrations on empty directory (should succeed with no migrations)
	err = RunMigrations(ctx, db.Pool, tmpDir)
	require.NoError(t, err)

	// Verify tracking table was created but empty
	var count int
	err = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM _test_migrations").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestMigrationsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgresContainer(ctx)
	require.NoError(t, err)
	defer Cleanup(db)

	// Verify critical tables exist after migrations
	criticalTables := []string{
		"organizations",
		"users",
		"agents",
		"printers",
		"documents",
		"print_jobs",
	}

	for _, table := range criticalTables {
		var exists bool
		err = db.Pool.QueryRow(ctx,
			"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = $1)",
			table).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "table %s should exist", table)
	}
}
