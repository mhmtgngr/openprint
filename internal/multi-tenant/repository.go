// Package multitenant provides multi-tenancy support for OpenPrint services.
// This file contains repository helpers for tenant-scoped queries.
package multitenant

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Querier is an interface that covers the database query methods we use.
type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

// TenantQueryBuilder builds SQL queries with automatic tenant filtering.
type TenantQueryBuilder struct {
	tenantColumn string
	tenantID     string
}

// NewTenantQueryBuilder creates a new query builder for tenant-scoped queries.
func NewTenantQueryBuilder(ctx context.Context, tableAlias string) (*TenantQueryBuilder, error) {
	tenantID, err := GetTenantID(ctx)
	if err != nil {
		return nil, err
	}

	column := "tenant_id"
	if tableAlias != "" {
		column = fmt.Sprintf("%s.tenant_id", tableAlias)
	}

	return &TenantQueryBuilder{
		tenantColumn: column,
		tenantID:     tenantID,
	}, nil
}

// MustNewTenantQueryBuilder creates a new query builder, panicking if no tenant context.
func MustNewTenantQueryBuilder(ctx context.Context, tableAlias string) *TenantQueryBuilder {
	b, err := NewTenantQueryBuilder(ctx, tableAlias)
	if err != nil {
		panic("tenant context required for query builder")
	}
	return b
}

// WhereClause returns the WHERE clause for tenant filtering.
func (b *TenantQueryBuilder) WhereClause() string {
	return fmt.Sprintf("%s = @tenant_id", b.tenantColumn)
}

// WhereClauseWith adds tenant filtering to an existing WHERE clause.
func (b *TenantQueryBuilder) WhereClauseWith(existingClause string) string {
	if existingClause == "" {
		return b.WhereClause()
	}
	return fmt.Sprintf("(%s) AND %s", existingClause, b.WhereClause())
}

// Args returns the arguments map for tenant filtering.
func (b *TenantQueryBuilder) Args() map[string]interface{} {
	return map[string]interface{}{
		"tenant_id": b.tenantID,
	}
}

// MergeArgs merges tenant args with additional query arguments.
func (b *TenantQueryBuilder) MergeArgs(args map[string]interface{}) map[string]interface{} {
	if args == nil {
		return b.Args()
	}
	args["tenant_id"] = b.tenantID
	return args
}

// ApplyToQuery applies tenant filtering to a SQL query string.
// It uses named parameters (@tenant_id) for PostgreSQL.
func (b *TenantQueryBuilder) ApplyToQuery(query string) string {
	// This is a simple implementation - for production, use a proper SQL parser
	// to properly inject the WHERE clause

	// Convert to lowercase for checking
	lowerQuery := strings.ToLower(query)

	// Check if WHERE clause exists
	if !strings.Contains(lowerQuery, " where ") {
		// Simple case: no WHERE clause, add one before ORDER BY, GROUP BY, LIMIT, etc.
		query = b.injectWhereBefore(query, []string{" order by ", " group by ", " limit ", " offset ", " for "})
	} else {
		// WHERE clause exists, add AND condition
		whereIndex := strings.Index(lowerQuery, " where ")
		beforeWhere := query[:whereIndex+7] // " where " is 7 chars
		afterWhere := query[whereIndex+7:]

		// Find the end of WHERE clause (before ORDER BY, GROUP BY, LIMIT, etc.)
		whereEnd := len(afterWhere)
		for _, keyword := range []string{" order by ", " group by ", " limit ", " offset ", " for "} {
			if idx := strings.Index(strings.ToLower(afterWhere), keyword); idx != -1 && idx < whereEnd {
				whereEnd = idx
			}
		}

		whereClause := afterWhere[:whereEnd]
		restOfQuery := afterWhere[whereEnd:]

		query = beforeWhere + "(" + whereClause + ") AND " + b.WhereClause() + restOfQuery
	}

	return query
}

// injectWhereBefore injects a WHERE clause before certain SQL keywords.
func (b *TenantQueryBuilder) injectWhereBefore(query string, keywords []string) string {
	lowerQuery := strings.ToLower(query)
	lowestIndex := len(query)

	for _, keyword := range keywords {
		if idx := strings.Index(lowerQuery, keyword); idx != -1 && idx < lowestIndex {
			lowestIndex = idx
		}
	}

	if lowestIndex < len(query) {
		// No extra space needed before query[lowestIndex:] since keywords have leading space
		return query[:lowestIndex] + " WHERE "+b.WhereClause()+query[lowestIndex:]
	}

	return query + " WHERE " + b.WhereClause()
}

// RowLevelSecurity enables or disables RLS for the current session.
// Note: This requires SUPERUSER or specific permissions.
type RowLevelSecurity struct {
	db Querier
}

// NewRowLevelSecurity creates a new RLS helper.
func NewRowLevelSecurity(db Querier) *RowLevelSecurity {
	return &RowLevelSecurity{db: db}
}

// EnableForTable enables RLS for a specific table.
func (rls *RowLevelSecurity) EnableForTable(ctx context.Context, tableName string) error {
	_, err := rls.db.Exec(ctx, fmt.Sprintf("ALTER TABLE %s ENABLE ROW LEVEL SECURITY", tableName))
	return err
}

// DisableForTable disables RLS for a specific table.
func (rls *RowLevelSecurity) DisableForTable(ctx context.Context, tableName string) error {
	_, err := rls.db.Exec(ctx, fmt.Sprintf("ALTER TABLE %s DISABLE ROW LEVEL SECURITY", tableName))
	return err
}

// SetTenantContext sets the tenant context for RLS policies.
// This should be called at the beginning of each request.
func (rls *RowLevelSecurity) SetTenantContext(ctx context.Context, tenantID string) error {
	// Set the tenant_id for the current session
	// This requires the app.tenant_id variable to be defined
	_, err := rls.db.Exec(ctx, "SET LOCAL app.tenant_id = $1", tenantID)
	return err
}

// ClearTenantContext clears the tenant context.
func (rls *RowLevelSecurity) ClearTenantContext(ctx context.Context) error {
	_, err := rls.db.Exec(ctx, "SET LOCAL app.tenant_id = NULL")
	return err
}

// ResetToDefault resets RLS to default behavior.
func (rls *RowLevelSecurity) ResetToDefault(ctx context.Context) error {
	_, err := rls.db.Exec(ctx, "RESET ALL")
	return err
}

// CreateTenantPolicy creates a tenant isolation policy for a table.
func (rls *RowLevelSecurity) CreateTenantPolicy(ctx context.Context, policyName, tableName string) error {
	// Drop existing policy if it exists
	_, _ = rls.db.Exec(ctx, fmt.Sprintf("DROP POLICY IF EXISTS %s ON %s", policyName, tableName))

	// Create policy that filters by the session's tenant_id
	query := fmt.Sprintf(`
		CREATE POLICY %s ON %s
		USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
	`, policyName, tableName)

	_, err := rls.db.Exec(ctx, query)
	return err
}

// CreateTenantPolicyWithAdmin creates a tenant isolation policy with admin bypass.
func (rls *RowLevelSecurity) CreateTenantPolicyWithAdmin(ctx context.Context, policyName, tableName string) error {
	// Drop existing policy if it exists
	_, _ = rls.db.Exec(ctx, fmt.Sprintf("DROP POLICY IF EXISTS %s ON %s", policyName, tableName))

	// Create policy that filters by tenant_id but allows platform admins
	// Platform admins have NULL tenant_id in the users table
	query := fmt.Sprintf(`
		CREATE POLICY %s ON %s
		USING (
			tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
			OR current_setting('app.is_platform_admin', true) = 'true'
		)
	`, policyName, tableName)

	_, err := rls.db.Exec(ctx, query)
	return err
}

// SetPlatformAdmin sets the platform admin flag for the current session.
func (rls *RowLevelSecurity) SetPlatformAdmin(ctx context.Context) error {
	_, err := rls.db.Exec(ctx, "SET LOCAL app.is_platform_admin = 'true'")
	return err
}

// Scanner helps with scanning tenant-aware rows.
type Scanner struct{}

// MustScanTenantID scans a tenant ID from a row, ensuring it matches the context.
func (s *Scanner) MustScanTenantID(ctx context.Context, row pgx.Row) error {
	var tenantID string
	if err := row.Scan(&tenantID); err != nil {
		return err
	}

	expectedTenantID, err := GetTenantID(ctx)
	if err != nil {
		return err
	}

	if tenantID != expectedTenantID {
		return ErrUnauthorizedTenant
	}

	return nil
}

// ScanWithTenant scans a row and validates tenant ownership.
func ScanWithTenant(ctx context.Context, row pgx.Row, dest ...interface{}) error {
	// First, scan tenant_id to verify ownership
	var tenantID string
	columns := make([]interface{}, 0, len(dest)+1)
	columns = append(columns, &tenantID)
	columns = append(columns, dest...)

	if err := row.Scan(columns...); err != nil {
		return err
	}

	expectedTenantID, err := GetTenantID(ctx)
	if err != nil {
		return err
	}

	// Platform admins can bypass tenant check
	if IsPlatformAdmin(ctx) {
		return nil
	}

	if tenantID != expectedTenantID {
		return ErrUnauthorizedTenant
	}

	return nil
}

// SafeQuery wraps a query with automatic tenant filtering.
// It's a helper for simple queries that need tenant scoping.
func SafeQuery(ctx context.Context, db Querier, baseQuery string, args ...interface{}) (pgx.Rows, error) {
	b, err := NewTenantQueryBuilder(ctx, "")
	if err != nil {
		return nil, err
	}

	query := b.ApplyToQuery(baseQuery)
	mergedArgs := b.MergeArgs(nil)

	// Convert args to use named parameters if needed
	// For simplicity, this implementation assumes the baseQuery uses @tenant_id placeholder
	return db.Query(ctx, query, mergeArgs(args, mergedArgs))
}

// SafeQueryRow wraps a query row with automatic tenant filtering.
func SafeQueryRow(ctx context.Context, db Querier, baseQuery string, args ...interface{}) pgx.Row {
	b, err := NewTenantQueryBuilder(ctx, "")
	if err != nil {
		// Return a row that will error on scan
		return &errorRow{err: err}
	}

	query := b.ApplyToQuery(baseQuery)
	mergedArgs := b.MergeArgs(nil)

	return db.QueryRow(ctx, query, mergeArgs(args, mergedArgs))
}

// SafeExec wraps an exec with automatic tenant filtering.
func SafeExec(ctx context.Context, db Querier, baseQuery string, args ...interface{}) (pgconn.CommandTag, error) {
	b, err := NewTenantQueryBuilder(ctx, "")
	if err != nil {
		return pgconn.CommandTag{}, err
	}

	query := b.ApplyToQuery(baseQuery)
	mergedArgs := b.MergeArgs(nil)

	return db.Exec(ctx, query, mergeArgs(args, mergedArgs))
}

// mergeArgs combines positional args with named args map.
func mergeArgs(positional []interface{}, named map[string]interface{}) []interface{} {
	// This is a simplified implementation
	// In production, you'd want a proper parameter replacement system
	result := make([]interface{}, 0, len(positional)+len(named))
	result = append(result, positional...)
	for _, v := range named {
		result = append(result, v)
	}
	return result
}

// errorRow is a pgx.Row that always returns an error.
type errorRow struct {
	err error
}

func (e *errorRow) Scan(dest ...interface{}) error {
	return e.err
}
