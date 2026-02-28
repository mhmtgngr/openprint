// Package testutil provides migration running utilities for tests.
package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RunMigrations executes all migration files from the migrations directory.
// It reads .up.sql files in ascending order and executes them against the database.
//
// Migration files should follow the naming pattern:
//   XXX_description.up.sql - for applying the migration
//   XXX_description.down.sql - for rolling back the migration (not used in tests)
//
// The function skips migrations that have already been applied by tracking
// applied migrations in a test-specific table.
func RunMigrations(ctx context.Context, db *pgxpool.Pool, migrationsDir string) error {
	// Create migrations tracking table if it doesn't exist
	if err := createMigrationsTable(ctx, db); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	// Read all migration files
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.up.sql"))
	if err != nil {
		return fmt.Errorf("glob migration files: %w", err)
	}

	// Sort files to ensure migrations run in order
	sort.Strings(files)

	// Get applied migrations
	applied, err := getAppliedMigrations(ctx, db)
	if err != nil {
		return fmt.Errorf("get applied migrations: %w", err)
	}

	// Run each migration
	for _, file := range files {
		baseName := filepath.Base(file)

		// Skip if already applied
		if applied[baseName] {
			continue
		}

		// Read migration file
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read migration file %s: %w", baseName, err)
		}

		// Execute migration
		if err := executeMigration(ctx, db, string(content)); err != nil {
			return fmt.Errorf("execute migration %s: %w", baseName, err)
		}

		// Record migration
		if err := recordMigration(ctx, db, baseName); err != nil {
			return fmt.Errorf("record migration %s: %w", baseName, err)
		}
	}

	return nil
}

// createMigrationsTable creates a table to track which migrations have been applied.
func createMigrationsTable(ctx context.Context, db *pgxpool.Pool) error {
	query := `
		CREATE TABLE IF NOT EXISTS _test_migrations (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL UNIQUE,
			applied_at TIMESTAMPTZ DEFAULT NOW()
		)
	`

	_, err := db.Exec(ctx, query)
	return err
}

// getAppliedMigrations returns a set of migration names that have already been applied.
func getAppliedMigrations(ctx context.Context, db *pgxpool.Pool) (map[string]bool, error) {
	query := `SELECT name FROM _test_migrations`
	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		applied[name] = true
	}

	return applied, rows.Err()
}

// executeMigration executes a single migration script.
// The migration is executed within a transaction for atomicity.
func executeMigration(ctx context.Context, db *pgxpool.Pool, content string) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Split content by semicolons and execute each statement
	// This is a simple approach - for complex migrations with plpgsql, we execute as-is
	if _, err := tx.Exec(ctx, content); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// recordMigration records that a migration has been applied.
func recordMigration(ctx context.Context, db *pgxpool.Pool, name string) error {
	query := `INSERT INTO _test_migrations (name) VALUES ($1)`
	_, err := db.Exec(ctx, query, name)
	return err
}

// ResetMigrations drops the migrations tracking table, allowing migrations to be re-run.
// This is useful for tests that need a fresh database state.
func ResetMigrations(ctx context.Context, db *pgxpool.Pool) error {
	query := `DROP TABLE IF EXISTS _test_migrations`
	_, err := db.Exec(ctx, query)
	return err
}

// ExecuteMigrationFile executes a specific migration file by name.
// This is useful for tests that need to set up specific schema elements.
func ExecuteMigrationFile(ctx context.Context, db *pgxpool.Pool, migrationsDir, filename string) error {
	if !strings.HasSuffix(filename, ".up.sql") {
		filename += ".up.sql"
	}

	fullPath := filepath.Join(migrationsDir, filename)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("read migration file %s: %w", filename, err)
	}

	return executeMigration(ctx, db, string(content))
}

// ExecuteSQL directly executes SQL against the database.
// This is useful for simple test setup that doesn't require a full migration file.
func ExecuteSQL(ctx context.Context, db *pgxpool.Pool, sql string) error {
	_, err := db.Exec(ctx, sql)
	return err
}

// CreateSchema creates the database schema by running all migrations.
// This is an alias for RunMigrations with a default migrations directory.
func CreateSchema(ctx context.Context, db *pgxpool.Pool) error {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return fmt.Errorf("find project root: %w", err)
	}

	migrationsDir := filepath.Join(projectRoot, "migrations")
	return RunMigrations(ctx, db, migrationsDir)
}
