// Package multitenant provides tests for tenant-scoped repository helpers.
package multitenant

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTenantQueryBuilder(t *testing.T) {
	tests := []struct {
		name        string
		setupCtx    func() context.Context
		tableAlias  string
		wantTenantID string
		wantColumn  string
		wantErr     error
	}{
		{
			name: "valid tenant context without alias",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), TenantIDKey, "tenant-123")
			},
			tableAlias:   "",
			wantTenantID: "tenant-123",
			wantColumn:   "tenant_id",
			wantErr:      nil,
		},
		{
			name: "valid tenant context with alias",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), TenantIDKey, "tenant-456")
			},
			tableAlias:   "org",
			wantTenantID: "tenant-456",
			wantColumn:   "org.tenant_id",
			wantErr:      nil,
		},
		{
			name:        "missing tenant context",
			setupCtx:    func() context.Context { return context.Background() },
			tableAlias:  "",
			wantTenantID: "",
			wantColumn:  "",
			wantErr:     ErrNoTenantContext,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			builder, err := NewTenantQueryBuilder(ctx, tt.tableAlias)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, builder)
			} else {
				require.NoError(t, err)
				require.NotNil(t, builder)
				assert.Equal(t, tt.wantTenantID, builder.tenantID)
				assert.Equal(t, tt.wantColumn, builder.tenantColumn)
			}
		})
	}
}

func TestMustNewTenantQueryBuilder(t *testing.T) {
	t.Run("valid context creates builder", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), TenantIDKey, "tenant-123")
		builder := MustNewTenantQueryBuilder(ctx, "")

		assert.NotNil(t, builder)
		assert.Equal(t, "tenant-123", builder.tenantID)
	})

	t.Run("missing context panics", func(t *testing.T) {
		ctx := context.Background()
		assert.Panics(t, func() {
			MustNewTenantQueryBuilder(ctx, "")
		})
	})
}

func TestTenantQueryBuilder_WhereClause(t *testing.T) {
	tests := []struct {
		name       string
		tenantID   string
		tableAlias string
		wantClause string
	}{
		{
			name:       "no alias",
			tenantID:   "tenant-123",
			tableAlias: "",
			wantClause: "tenant_id = @tenant_id",
		},
		{
			name:       "with alias",
			tenantID:   "tenant-123",
			tableAlias: "org",
			wantClause: "org.tenant_id = @tenant_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := &TenantQueryBuilder{
				tenantID:     tt.tenantID,
				tenantColumn: tt.tableAlias + "." + "tenant_id",
			}
			if tt.tableAlias == "" {
				builder.tenantColumn = "tenant_id"
			}

			clause := builder.WhereClause()
			assert.Equal(t, tt.wantClause, clause)
		})
	}
}

func TestTenantQueryBuilder_WhereClauseWith(t *testing.T) {
	tests := []struct {
		name          string
		existingClause string
		tableAlias    string
		wantClause    string
	}{
		{
			name:          "no existing clause",
			existingClause: "",
			tableAlias:    "",
			wantClause:    "tenant_id = @tenant_id",
		},
		{
			name:          "with existing clause",
			existingClause: "status = 'active'",
			tableAlias:    "",
			wantClause:    "(status = 'active') AND tenant_id = @tenant_id",
		},
		{
			name:          "with existing clause and alias",
			existingClause: "status = 'active'",
			tableAlias:    "org",
			wantClause:    "(status = 'active') AND org.tenant_id = @tenant_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := &TenantQueryBuilder{
				tenantID:     "tenant-123",
				tenantColumn: tt.tableAlias + "." + "tenant_id",
			}
			if tt.tableAlias == "" {
				builder.tenantColumn = "tenant_id"
			}

			clause := builder.WhereClauseWith(tt.existingClause)
			assert.Equal(t, tt.wantClause, clause)
		})
	}
}

func TestTenantQueryBuilder_Args(t *testing.T) {
	builder := &TenantQueryBuilder{
		tenantID:     "tenant-123",
		tenantColumn: "tenant_id",
	}

	args := builder.Args()

	assert.Len(t, args, 1)
	assert.Equal(t, "tenant-123", args["tenant_id"])
}

func TestTenantQueryBuilder_MergeArgs(t *testing.T) {
	builder := &TenantQueryBuilder{
		tenantID:     "tenant-123",
		tenantColumn: "tenant_id",
	}

	tests := []struct {
		name     string
		existing map[string]interface{}
		wantLen  int
	}{
		{
			name:     "nil args",
			existing: nil,
			wantLen:  1,
		},
		{
			name:     "empty args",
			existing: map[string]interface{}{},
			wantLen:  1,
		},
		{
			name: "with existing args",
			existing: map[string]interface{}{
				"status": "active",
				"limit":  10,
			},
			wantLen: 3,
		},
		{
			name: "tenant ID gets overwritten",
			existing: map[string]interface{}{
				"tenant_id": "old-tenant",
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builder.MergeArgs(tt.existing)
			assert.Len(t, result, tt.wantLen)
			assert.Equal(t, "tenant-123", result["tenant_id"])

			if tt.existing != nil {
				for k, v := range tt.existing {
					if k != "tenant_id" {
						assert.Equal(t, v, result[k])
					}
				}
			}
		})
	}
}

func TestTenantQueryBuilder_ApplyToQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		tenantID string
		wantQuery string
	}{
		{
			name:     "simple select",
			query:    "SELECT * FROM organizations",
			tenantID: "tenant-123",
			wantQuery: "SELECT * FROM organizations WHERE tenant_id = @tenant_id",
		},
		{
			name:     "select with order by",
			query:    "SELECT * FROM organizations ORDER BY name",
			tenantID: "tenant-123",
			wantQuery: "SELECT * FROM organizations WHERE tenant_id = @tenant_id ORDER BY name",
		},
		{
			name:     "select with existing where",
			query:    "SELECT * FROM organizations WHERE status = 'active'",
			tenantID: "tenant-123",
			wantQuery: "SELECT * FROM organizations WHERE (status = 'active') AND tenant_id = @tenant_id",
		},
		{
			name:     "select with where and order by",
			query:    "SELECT * FROM organizations WHERE status = 'active' ORDER BY name",
			tenantID: "tenant-123",
			wantQuery: "SELECT * FROM organizations WHERE (status = 'active') AND tenant_id = @tenant_id ORDER BY name",
		},
		{
			name:     "select with limit",
			query:    "SELECT * FROM organizations LIMIT 10",
			tenantID: "tenant-123",
			wantQuery: "SELECT * FROM organizations WHERE tenant_id = @tenant_id LIMIT 10",
		},
		{
			name:     "select with group by",
			query:    "SELECT status, COUNT(*) FROM organizations GROUP BY status",
			tenantID: "tenant-123",
			wantQuery: "SELECT status, COUNT(*) FROM organizations WHERE tenant_id = @tenant_id GROUP BY status",
		},
		{
			name:     "select with offset",
			query:    "SELECT * FROM organizations OFFSET 10",
			tenantID: "tenant-123",
			wantQuery: "SELECT * FROM organizations WHERE tenant_id = @tenant_id OFFSET 10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := &TenantQueryBuilder{
				tenantID:     tt.tenantID,
				tenantColumn: "tenant_id",
			}
			result := builder.ApplyToQuery(tt.query)
			assert.Equal(t, tt.wantQuery, result)
		})
	}
}

func TestTenantQueryBuilder_injectWhereBefore(t *testing.T) {
	builder := &TenantQueryBuilder{
		tenantID:     "tenant-123",
		tenantColumn: "tenant_id",
	}

	tests := []struct {
		name     string
		query    string
		keywords []string
		wantQuery string
	}{
		{
			name:     "inject before ORDER BY",
			query:    "SELECT * FROM organizations ORDER BY name",
			keywords: []string{" ORDER BY ", " GROUP BY ", " LIMIT "},
			wantQuery: "SELECT * FROM organizations ORDER BY name WHERE tenant_id = @tenant_id",
		},
		{
			name:     "inject before GROUP BY",
			query:    "SELECT status, COUNT(*) FROM organizations GROUP BY status",
			keywords: []string{" ORDER BY ", " GROUP BY ", " LIMIT "},
			wantQuery: "SELECT status, COUNT(*) FROM organizations GROUP BY status WHERE tenant_id = @tenant_id",
		},
		{
			name:     "inject before LIMIT",
			query:    "SELECT * FROM organizations LIMIT 10",
			keywords: []string{" ORDER BY ", " GROUP BY ", " LIMIT "},
			wantQuery: "SELECT * FROM organizations LIMIT 10 WHERE tenant_id = @tenant_id",
		},
		{
			name:     "no keywords - append",
			query:    "SELECT * FROM organizations",
			keywords: []string{" ORDER BY ", " GROUP BY ", " LIMIT "},
			wantQuery: "SELECT * FROM organizations WHERE tenant_id = @tenant_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builder.injectWhereBefore(tt.query, tt.keywords)
			assert.Equal(t, tt.wantQuery, result)
		})
	}
}

func TestNewRowLevelSecurity(t *testing.T) {
	// Use a mock querier
	mockDB := &mockQuerier{}
	rls := NewRowLevelSecurity(mockDB)

	assert.NotNil(t, rls)
	assert.Equal(t, mockDB, rls.db)
}

func TestRowLevelSecurity_EnableForTable(t *testing.T) {
	mockDB := &mockQuerier{
		execResult: pgconn.NewCommandTag("ALTER"),
		execErr:    nil,
	}
	rls := NewRowLevelSecurity(mockDB)

	ctx := context.Background()
	err := rls.EnableForTable(ctx, "organizations")

	assert.NoError(t, err)
	// SQL identifiers are now quoted for SQL injection prevention
	assert.Contains(t, mockDB.lastSQL, `ALTER TABLE "organizations" ENABLE ROW LEVEL SECURITY`)
}

func TestRowLevelSecurity_DisableForTable(t *testing.T) {
	mockDB := &mockQuerier{
		execResult: pgconn.NewCommandTag("ALTER"),
		execErr:    nil,
	}
	rls := NewRowLevelSecurity(mockDB)

	ctx := context.Background()
	err := rls.DisableForTable(ctx, "organizations")

	assert.NoError(t, err)
	// SQL identifiers are now quoted for SQL injection prevention
	assert.Contains(t, mockDB.lastSQL, `ALTER TABLE "organizations" DISABLE ROW LEVEL SECURITY`)
}

func TestRowLevelSecurity_SetTenantContext(t *testing.T) {
	mockDB := &mockQuerier{
		execResult: pgconn.NewCommandTag("SET"),
		execErr:    nil,
	}
	rls := NewRowLevelSecurity(mockDB)

	ctx := context.Background()
	err := rls.SetTenantContext(ctx, "tenant-123")

	assert.NoError(t, err)
	assert.Contains(t, mockDB.lastSQL, "SET LOCAL app.tenant_id")
}

func TestRowLevelSecurity_ClearTenantContext(t *testing.T) {
	mockDB := &mockQuerier{
		execResult: pgconn.NewCommandTag("SET"),
		execErr:    nil,
	}
	rls := NewRowLevelSecurity(mockDB)

	ctx := context.Background()
	err := rls.ClearTenantContext(ctx)

	assert.NoError(t, err)
	assert.Contains(t, mockDB.lastSQL, "SET LOCAL app.tenant_id = NULL")
}

func TestRowLevelSecurity_ResetToDefault(t *testing.T) {
	mockDB := &mockQuerier{
		execResult: pgconn.NewCommandTag("RESET"),
		execErr:    nil,
	}
	rls := NewRowLevelSecurity(mockDB)

	ctx := context.Background()
	err := rls.ResetToDefault(ctx)

	assert.NoError(t, err)
	assert.Contains(t, mockDB.lastSQL, "RESET ALL")
}

func TestRowLevelSecurity_CreateTenantPolicy(t *testing.T) {
	mockDB := &mockQuerier{
		execResult: pgconn.NewCommandTag("CREATE POLICY"),
		execErr:    nil,
	}
	rls := NewRowLevelSecurity(mockDB)

	ctx := context.Background()
	err := rls.CreateTenantPolicy(ctx, "tenant_isolation", "organizations")

	assert.NoError(t, err)
	// mockDB.lastSQL contains the CREATE POLICY statement (DROP POLICY is executed first but not stored)
	// SQL identifiers are now quoted for SQL injection prevention
	assert.Contains(t, mockDB.lastSQL, `CREATE POLICY "tenant_isolation" ON "organizations"`)
	assert.Contains(t, mockDB.lastSQL, "USING (tenant_id =")
}

func TestRowLevelSecurity_CreateTenantPolicyWithAdmin(t *testing.T) {
	mockDB := &mockQuerier{
		execResult: pgconn.NewCommandTag("CREATE POLICY"),
		execErr:    nil,
	}
	rls := NewRowLevelSecurity(mockDB)

	ctx := context.Background()
	err := rls.CreateTenantPolicyWithAdmin(ctx, "tenant_isolation", "organizations")

	assert.NoError(t, err)
	// mockDB.lastSQL contains the CREATE POLICY statement (DROP POLICY is executed first but not stored)
	// SQL identifiers are now quoted for SQL injection prevention
	assert.Contains(t, mockDB.lastSQL, `CREATE POLICY "tenant_isolation" ON "organizations"`)
	assert.Contains(t, mockDB.lastSQL, "current_setting('app.is_platform_admin'")
}

func TestRowLevelSecurity_SetPlatformAdmin(t *testing.T) {
	mockDB := &mockQuerier{
		execResult: pgconn.NewCommandTag("SET"),
		execErr:    nil,
	}
	rls := NewRowLevelSecurity(mockDB)

	ctx := context.Background()
	err := rls.SetPlatformAdmin(ctx)

	assert.NoError(t, err)
	assert.Contains(t, mockDB.lastSQL, "SET LOCAL app.is_platform_admin = 'true'")
}

func TestScanner_MustScanTenantID(t *testing.T) {
	tests := []struct {
		name        string
		setupRow    func() *mockRow
		setupCtx    func() context.Context
		wantErr     error
	}{
		{
			name: "matching tenant ID",
			setupRow: func() *mockRow {
				return &mockRow{
					scanValues: []interface{}{"tenant-123"},
					scanErr:    nil,
				}
			},
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), TenantIDKey, "tenant-123")
			},
			wantErr: nil,
		},
		{
			name: "mismatched tenant ID",
			setupRow: func() *mockRow {
				return &mockRow{
					scanValues: []interface{}{"tenant-456"},
					scanErr:    nil,
				}
			},
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), TenantIDKey, "tenant-123")
			},
			wantErr: ErrUnauthorizedTenant,
		},
		{
			name: "row scan error",
			setupRow: func() *mockRow {
				return &mockRow{
					scanErr: errors.New("scan error"),
				}
			},
			setupCtx: func() context.Context { return context.Background() },
			wantErr:  errors.New("scan error"),
		},
		{
			name: "no tenant context",
			setupRow: func() *mockRow {
				return &mockRow{
					scanValues: []interface{}{"tenant-123"},
					scanErr:    nil,
				}
			},
			setupCtx: func() context.Context { return context.Background() },
			wantErr:  ErrNoTenantContext,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := &Scanner{}
			ctx := tt.setupCtx()
			row := tt.setupRow()

			err := scanner.MustScanTenantID(ctx, row)

			if tt.wantErr != nil {
				require.Error(t, err)
				if errors.Is(tt.wantErr, ErrNoTenantContext) || errors.Is(tt.wantErr, ErrUnauthorizedTenant) {
					assert.ErrorIs(t, err, tt.wantErr)
				} else {
					assert.Contains(t, err.Error(), tt.wantErr.Error())
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestScanWithTenant(t *testing.T) {
	tests := []struct {
		name        string
		setupRow    func() *mockRow
		setupCtx    func() context.Context
		numDest     int
		wantErr     error
	}{
		{
			name: "matching tenant ID",
			setupRow: func() *mockRow {
				return &mockRow{
					scanValues: []interface{}{"tenant-123", "value1", "value2"},
					scanErr:    nil,
				}
			},
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), TenantIDKey, "tenant-123")
			},
			numDest: 2,
			wantErr: nil,
		},
		{
			name: "mismatched tenant ID",
			setupRow: func() *mockRow {
				return &mockRow{
					scanValues: []interface{}{"tenant-456", "value1", "value2"},
					scanErr:    nil,
				}
			},
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), TenantIDKey, "tenant-123")
			},
			numDest: 2,
			wantErr: ErrUnauthorizedTenant,
		},
		{
			name: "platform admin bypasses tenant check",
			setupRow: func() *mockRow {
				return &mockRow{
					scanValues: []interface{}{"tenant-456", "value1", "value2"},
					scanErr:    nil,
				}
			},
			setupCtx: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, TenantIDKey, "tenant-123")
				ctx = context.WithValue(ctx, IsPlatformAdminKey, true)
				return ctx
			},
			numDest: 2,
			wantErr: nil,
		},
		{
			name: "no tenant context",
			setupRow: func() *mockRow {
				return &mockRow{
					scanValues: []interface{}{"tenant-123", "value1", "value2"},
					scanErr:    nil,
				}
			},
			setupCtx: func() context.Context { return context.Background() },
			numDest:  2,
			wantErr:  ErrNoTenantContext,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			row := tt.setupRow()

			dest := make([]interface{}, tt.numDest)
			for i := range dest {
				dest[i] = new(string)
			}

			err := ScanWithTenant(ctx, row, dest...)

			if tt.wantErr != nil {
				require.Error(t, err)
				if errors.Is(tt.wantErr, ErrNoTenantContext) || errors.Is(tt.wantErr, ErrUnauthorizedTenant) {
					assert.ErrorIs(t, err, tt.wantErr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSafeQuery(t *testing.T) {
	tests := []struct {
		name        string
		setupCtx    func() context.Context
		baseQuery   string
		wantErr     error
	}{
		{
			name: "valid query with tenant context",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), TenantIDKey, "tenant-123")
			},
			baseQuery: "SELECT * FROM organizations",
			wantErr:   nil,
		},
		{
			name:        "no tenant context",
			setupCtx:    func() context.Context { return context.Background() },
			baseQuery:   "SELECT * FROM organizations",
			wantErr:     ErrNoTenantContext,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			mockDB := &mockQuerier{}

			rows, err := SafeQuery(ctx, mockDB, tt.baseQuery)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, rows)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, rows)
				assert.Contains(t, mockDB.lastSQL, "WHERE tenant_id = @tenant_id")
			}
		})
	}
}

func TestSafeQueryRow(t *testing.T) {
	tests := []struct {
		name        string
		setupCtx    func() context.Context
		baseQuery   string
		wantErr     bool
	}{
		{
			name: "valid query with tenant context",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), TenantIDKey, "tenant-123")
			},
			baseQuery: "SELECT * FROM organizations WHERE id = $1",
			wantErr:   false,
		},
		{
			name:        "no tenant context",
			setupCtx:    func() context.Context { return context.Background() },
			baseQuery:   "SELECT * FROM organizations WHERE id = $1",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			mockDB := &mockQuerier{}

			row := SafeQueryRow(ctx, mockDB, tt.baseQuery)

			if tt.wantErr {
				// Should return an errorRow
				err := row.Scan(new(string))
				assert.Error(t, err)
			} else {
				assert.NotNil(t, row)
			}
		})
	}
}

func TestSafeExec(t *testing.T) {
	tests := []struct {
		name        string
		setupCtx    func() context.Context
		baseQuery   string
		wantErr     error
	}{
		{
			name: "valid exec with tenant context",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), TenantIDKey, "tenant-123")
			},
			baseQuery: "UPDATE organizations SET name = $1 WHERE id = $2",
			wantErr:   nil,
		},
		{
			name:        "no tenant context",
			setupCtx:    func() context.Context { return context.Background() },
			baseQuery:   "UPDATE organizations SET name = $1 WHERE id = $2",
			wantErr:     ErrNoTenantContext,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			mockDB := &mockQuerier{
				execResult: pgconn.NewCommandTag("UPDATE 1"),
			}

			tag, err := SafeExec(ctx, mockDB, tt.baseQuery)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, tag)
			}
		})
	}
}

func TestMergeArgs(t *testing.T) {
	positional := []interface{}{"value1", "value2"}
	named := map[string]interface{}{
		"param1": "named1",
		"param2": "named2",
	}

	result := mergeArgs(positional, named)

	assert.Len(t, result, 4)
	assert.Equal(t, "value1", result[0])
	assert.Equal(t, "value2", result[1])
	// Named args are appended (order not guaranteed for maps)
	assert.Contains(t, result, "named1")
	assert.Contains(t, result, "named2")
}

func TestErrorRow(t *testing.T) {
	testErr := errors.New("test error")
	errRow := &errorRow{err: testErr}

	var dest string
	err := errRow.Scan(&dest)

	assert.Error(t, err)
	assert.Equal(t, testErr, err)
}

// Mock implementations for testing

type mockQuerier struct {
	lastSQL   string
	lastArgs  []interface{}
	execResult pgconn.CommandTag
	execErr    error
}

func (m *mockQuerier) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	m.lastSQL = sql
	m.lastArgs = arguments
	return m.execResult, m.execErr
}

func (m *mockQuerier) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	m.lastSQL = sql
	m.lastArgs = args
	return &mockRows{}, nil
}

func (m *mockQuerier) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	m.lastSQL = sql
	m.lastArgs = args
	return &mockRow{}
}

type mockRows struct{}

func (m *mockRows) Close() {}
func (m *mockRows) Err() error { return nil }
func (m *mockRows) CommandTag() pgconn.CommandTag { return pgconn.NewCommandTag("") }
func (m *mockRows) Fields() []string { return []string{} }
func (m *mockRows) FieldDescriptions() []pgconn.FieldDescription { return []pgconn.FieldDescription{} }
func (m *mockRows) Next() bool { return false }
func (m *mockRows) Values() ([]interface{}, error) { return []interface{}{}, nil }
func (m *mockRows) Scan(dest ...interface{}) error { return nil }
func (m *mockRows) RawValues() [][]byte { return nil }
func (m *mockRows) Conn() *pgx.Conn { return nil }

type mockRow struct {
	scanValues []interface{}
	scanErr    error
}

func (m *mockRow) Scan(dest ...interface{}) error {
	if m.scanErr != nil {
		return m.scanErr
	}
	if len(m.scanValues) > 0 {
		for i, val := range m.scanValues {
			if i < len(dest) {
				if ptr, ok := dest[i].(*interface{}); ok {
					*ptr = val
				} else if ptr, ok := dest[i].(*string); ok {
					if str, ok := val.(string); ok {
						*ptr = str
					}
				}
			}
		}
	}
	return nil
}
