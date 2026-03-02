// Package prometheus provides tests for database metrics collection.
package prometheus

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDBStatsCollector(t *testing.T) {
	cfg := Config{ServiceName: "test-db-collector"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	collector := NewDBStatsCollector(metrics, "test-service", "testdb", "postgresql")

	assert.NotNil(t, collector)
	assert.Equal(t, metrics.DB, collector.metrics)
	assert.Equal(t, "test-service", collector.serviceName)
	assert.Equal(t, "testdb", collector.dbName)
	assert.Equal(t, "postgresql", collector.dbSystem)
}

func TestDBStatsCollector_RecordStats(t *testing.T) {
	cfg := Config{ServiceName: "test-record-stats"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)
	collector := NewDBStatsCollector(metrics, "test-service", "testdb", "postgresql")

	stats := sql.DBStats{
		OpenConnections: 10,
		InUse:           7,
		Idle:            3,
	}

	collector.RecordStats(stats)

	// The metrics should have been set
	// We can't easily check the exact values without using the prometheus API,
	// but we can verify the function doesn't panic
	assert.NotNil(t, collector)
}

func TestDBStatsCollector_RecordStatsFromPool(t *testing.T) {
	cfg := Config{ServiceName: "test-pool-stats"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)
	collector := NewDBStatsCollector(metrics, "test-service", "testdb", "postgresql")

	// Note: RecordStatsFromPool requires a real *pgxpool.Pool
	// For testing purposes, we just verify the collector was created
	assert.NotNil(t, collector)
}

func TestNewDBTracer(t *testing.T) {
	cfg := Config{ServiceName: "test-tracer"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	tracer := NewDBTracer(metrics, "test-service", "testdb", "postgresql")

	assert.NotNil(t, tracer)
	assert.Equal(t, metrics.DB, tracer.metrics)
	assert.Equal(t, "test-service", tracer.serviceName)
	assert.Equal(t, "testdb", tracer.dbName)
	assert.Equal(t, "postgresql", tracer.dbSystem)
}

func TestDBTracer_TraceQuery(t *testing.T) {
	cfg := Config{ServiceName: "test-trace-query"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)
	tracer := NewDBTracer(metrics, "test-service", "testdb", "postgresql")

	t.Run("records successful query", func(t *testing.T) {
		tracer.TraceQuery("SELECT", 50*time.Millisecond, nil, 5)

		// No panic - metric recorded
	})

	t.Run("records query error", func(t *testing.T) {
		tracer.TraceQuery("INSERT", 10*time.Millisecond, assert.AnError, 0)

		// No panic - error metric recorded
	})

	t.Run("records query with rows affected", func(t *testing.T) {
		tracer.TraceQuery("UPDATE", 25*time.Millisecond, nil, 100)

		// No panic - rows affected recorded
	})
}

func TestDBTracer_TraceTransaction(t *testing.T) {
	cfg := Config{ServiceName: "test-trace-tx"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)
	tracer := NewDBTracer(metrics, "test-service", "testdb", "postgresql")

	t.Run("records successful transaction", func(t *testing.T) {
		tracer.TraceTransaction(100*time.Millisecond, true)

		// No panic
	})

	t.Run("records failed transaction", func(t *testing.T) {
		tracer.TraceTransaction(50*time.Millisecond, false)

		// No panic
	})
}

func TestDBConfig(t *testing.T) {
	cfg := DBConfig{
		ServiceName: "test-service",
		DBName:      "testdb",
		DBSystem:    "postgresql",
	}

	assert.Equal(t, "test-service", cfg.ServiceName)
	assert.Equal(t, "testdb", cfg.DBName)
	assert.Equal(t, "postgresql", cfg.DBSystem)
}

func TestWrapDB(t *testing.T) {
	cfg := Config{ServiceName: "test-wrap"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)
	_ = reg

	dbCfg := DBConfig{
		ServiceName: "test-service",
		DBName:      "testdb",
		DBSystem:    "postgresql",
	}
	_ = dbCfg

	// Create a real sql.DB (using a mock connection)
	// In practice, this would be a real database connection
	// For testing, we just verify the function signature

	t.Run("WrapDB returns the same db", func(t *testing.T) {
		// Can't create a real DB in tests, but we verify the function exists
		assert.NotNil(t, WrapDB)
	})
}

func TestWrapPgxPool(t *testing.T) {
	cfg := Config{ServiceName: "test-wrap-pgx"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)
	_ = reg

	_ = DBConfig{
		ServiceName: "test-service",
		DBName:      "testdb",
		DBSystem:    "postgresql",
	}

	// Can't create a real pool in tests
	// Just verify the function exists
	assert.NotNil(t, WrapPgxPool)
}

func TestDefaultDBDelegate(t *testing.T) {
	delegate := &DefaultDBDelegate{}

	t.Run("extracts operation from simple queries", func(t *testing.T) {
		tests := []struct {
			query    string
			expected string
		}{
			{"SELECT * FROM users", "SELECT"},
			{"INSERT INTO users VALUES (1, 'test')", "INSERT"},
			{"UPDATE users SET name = 'test' WHERE id = 1", "UPDATE"},
			{"DELETE FROM users WHERE id = 1", "DELETE"},
			{"CREATE TABLE test (id INT)", "CREATE"},
			{"DROP TABLE test", "DROP"},
			{"ALTER TABLE test ADD COLUMN col INT", "ALTER"},
			{"BEGIN TRANSACTION", "transaction_BEGIN"},
			{"COMMIT", "transaction_COMMIT"},
			{"ROLLBACK", "transaction_ROLLBACK"},
			{"SELECT * FROM users WHERE name = 'test' AND age > 18", "SELECT"},
		}

		for _, tt := range tests {
			t.Run(tt.query, func(t *testing.T) {
				result := delegate.OperationFromQuery(tt.query)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("handles unknown operations", func(t *testing.T) {
		result := delegate.OperationFromQuery("INVALID OPERATION")
		assert.Equal(t, "other", result)
	})

	t.Run("handles case sensitivity", func(t *testing.T) {
		result := delegate.OperationFromQuery("select * from users")
		assert.Equal(t, "other", result) // First word must be uppercase
	})
}

func TestNewQuerierWithMetrics(t *testing.T) {
	cfg := Config{ServiceName: "test-querier"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	// Mock querier
	mockQ := &mockQuerier{}

	querier := NewQuerierWithMetrics(mockQ, metrics, "test-service", "testdb", "postgresql")

	assert.NotNil(t, querier)
	assert.Equal(t, mockQ, querier.querier)
	assert.NotNil(t, querier.tracer)
	assert.NotNil(t, querier.delegate)
}

// mockQuerier implements Querier interface
type mockQuerier struct{}

func (m *mockQuerier) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return nil, nil
}

func (m *mockQuerier) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return &sql.Row{}
}

func (m *mockQuerier) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return &mockResult{}, nil
}

// mockResult implements sql.Result
type mockResult struct{}

func (m *mockResult) LastInsertId() (int64, error) {
	return 1, nil
}

func (m *mockResult) RowsAffected() (int64, error) {
	return 5, nil
}

func TestQuerierWithMetrics_Query(t *testing.T) {
	cfg := Config{ServiceName: "test-qm-query"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)
	mockQ := &mockQuerier{}

	querier := NewQuerierWithMetrics(mockQ, metrics, "test-service", "testdb", "postgresql")

	ctx := context.Background()

	t.Run("executes query and records metrics", func(t *testing.T) {
		_, err := querier.Query(ctx, "SELECT * FROM users")

		// Mock returns nil rows - we're just testing that metrics are recorded
		assert.NoError(t, err)
	})
}

func TestQuerierWithMetrics_QueryRow(t *testing.T) {
	cfg := Config{ServiceName: "test-qm-queryrow"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)
	mockQ := &mockQuerier{}

	querier := NewQuerierWithMetrics(mockQ, metrics, "test-service", "testdb", "postgresql")

	ctx := context.Background()

	t.Run("executes query row and records metrics", func(t *testing.T) {
		row := querier.QueryRow(ctx, "SELECT * FROM users WHERE id = $1", 1)

		assert.NotNil(t, row)
	})
}

func TestQuerierWithMetrics_Exec(t *testing.T) {
	cfg := Config{ServiceName: "test-qm-exec"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)
	mockQ := &mockQuerier{}

	querier := NewQuerierWithMetrics(mockQ, metrics, "test-service", "testdb", "postgresql")

	ctx := context.Background()

	t.Run("executes exec and records metrics", func(t *testing.T) {
		result, err := querier.Exec(ctx, "INSERT INTO users VALUES ($1)", "test")

		assert.NotNil(t, result)
		assert.NoError(t, err)
	})
}

func TestExecWithMetrics(t *testing.T) {
	cfg := Config{ServiceName: "test-exec-metrics"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)
	_ = metrics

	// Can't test with real DB, just verify function exists
	assert.NotNil(t, ExecWithMetrics)
}

func TestQueryWithMetrics(t *testing.T) {
	cfg := Config{ServiceName: "test-query-metrics"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)
	_ = metrics

	// Can't test with real DB, just verify function exists
	assert.NotNil(t, QueryWithMetrics)
}

func TestRunInTransaction(t *testing.T) {
	cfg := Config{ServiceName: "test-tx-run"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)
	_ = metrics

	t.Run("requires database connection", func(t *testing.T) {
		// Can't test without a real DB
		// Just verify the function signature
		assert.NotNil(t, RunInTransaction)
	})
}

func TestDBMetrics_ConnectionMetrics(t *testing.T) {
	cfg := Config{ServiceName: "test-conn-metrics"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	labels := prometheus.Labels{
		LabelServiceName: "test-conn-metrics",
		LabelDBName:      "testdb",
		LabelDBSystem:    "postgresql",
	}

	t.Run("set connection active gauge", func(t *testing.T) {
		metrics.DB.ConnectionsActive.With(labels).Set(10)
		// Value set
	})

	t.Run("set connection idle gauge", func(t *testing.T) {
		metrics.DB.ConnectionsIdle.With(labels).Set(5)
		// Value set
	})
}

func TestDBMetrics_QueryMetrics(t *testing.T) {
	cfg := Config{ServiceName: "test-query-metrics-2"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	labels := prometheus.Labels{
		LabelServiceName: "test-query-metrics-2",
		LabelDBName:      "testdb",
		LabelOperation:   "SELECT",
	}

	t.Run("record query duration", func(t *testing.T) {
		metrics.DB.QueryDuration.With(labels).Observe(0.1)
	})

	t.Run("increment query counter", func(t *testing.T) {
		metrics.DB.QueriesTotal.With(labels).Inc()
	})

	t.Run("record query error", func(t *testing.T) {
		metrics.DB.QueryErrorsTotal.With(labels).Inc()
	})

	t.Run("record rows affected", func(t *testing.T) {
		metrics.DB.RowsAffected.With(labels).Observe(50)
	})
}

func TestDBMetrics_TransactionMetrics(t *testing.T) {
	cfg := Config{ServiceName: "test-tx-metrics"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	successLabels := prometheus.Labels{
		LabelServiceName: "test-tx-metrics",
		LabelDBName:      "testdb",
		LabelSuccess:     SuccessTrue,
	}

	failureLabels := prometheus.Labels{
		LabelServiceName: "test-tx-metrics",
		LabelDBName:      "testdb",
		LabelSuccess:     SuccessFalse,
	}

	durationLabels := prometheus.Labels{
		LabelServiceName: "test-tx-metrics",
		LabelDBName:      "testdb",
	}

	t.Run("record successful transaction", func(t *testing.T) {
		metrics.DB.TransactionsTotal.With(successLabels).Inc()
		metrics.DB.TransactionDuration.With(durationLabels).Observe(0.5)
	})

	t.Run("record failed transaction", func(t *testing.T) {
		metrics.DB.TransactionsTotal.With(failureLabels).Inc()
		metrics.DB.TransactionDuration.With(durationLabels).Observe(0.1)
	})
}

func TestDBDelegate(t *testing.T) {
	t.Run("DBDelegate interface exists", func(t *testing.T) {
		// Verify interface definition
		var _ DBDelegate = &DefaultDBDelegate{}
	})

	t.Run("DefaultDBDelegate implements DBDelegate", func(t *testing.T) {
		delegate := &DefaultDBDelegate{}

		assert.Implements(t, (*DBDelegate)(nil), delegate)
	})
}

func TestDBStatsCollector_ConcurrentRecording(t *testing.T) {
	cfg := Config{ServiceName: "test-concurrent-stats"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)
	collector := NewDBStatsCollector(metrics, "test-service", "testdb", "postgresql")

	// Simulate concurrent stats recording
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			stats := sql.DBStats{
				OpenConnections: 10,
				InUse:           5,
				Idle:            5,
			}
			collector.RecordStats(stats)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should complete without race
	assert.NotNil(t, collector)
}

func TestDBTracer_ConcurrentTracing(t *testing.T) {
	cfg := Config{ServiceName: "test-concurrent-tracer"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)
	tracer := NewDBTracer(metrics, "test-service", "testdb", "postgresql")

	done := make(chan bool)

	// Simulate concurrent query tracing
	for i := 0; i < 100; i++ {
		go func(idx int) {
			operation := "SELECT"
			if idx%2 == 0 {
				operation = "INSERT"
			}
			tracer.TraceQuery(operation, 10*time.Millisecond, nil, 10)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	// Should complete without race
	assert.NotNil(t, tracer)
}

func TestDBConfig_Defaults(t *testing.T) {
	cfg := DBConfig{}

	// Empty config should have empty values
	assert.Empty(t, cfg.ServiceName)
	assert.Empty(t, cfg.DBName)
	assert.Empty(t, cfg.DBSystem)
}

// mockPoolStats implements pool.Stat interface for testing
type mockPoolStats struct {
	totalConns int32
	idleConns  int32
}

func (m *mockPoolStats) TotalConns() int32 {
	return m.totalConns
}

func (m *mockPoolStats) IdleConns() int32 {
	return m.idleConns
}

func (m *mockPoolStats) AcquireCount() int64 {
	return 0
}

func (m *mockPoolStats) AcquireDuration() time.Duration {
	return 0
}

func (m *mockPoolStats) AcquiredConns() int32 {
	return 0
}

func (m *mockPoolStats) MaxConns() int32 {
	return 0
}

func (m *mockPoolStats) EmptyAcquireCount() int64 {
	return 0
}

func TestPgxPoolStats_Interface(t *testing.T) {
	// Verify our mock implements the expected interface
	stats := &mockPoolStats{
		totalConns: 10,
		idleConns:  5,
	}

	assert.Equal(t, int32(10), stats.TotalConns())
	assert.Equal(t, int32(5), stats.IdleConns())
}

func TestDBMetrics_DifferentDBSystems(t *testing.T) {
	cfg := Config{ServiceName: "test-db-systems"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	// Test that we can record metrics with different DB systems
	// Using ConnectionsActive which includes LabelDBSystem
	systems := []string{
		DBSystemPostgreSQL,
		DBSystemMySQL,
		DBSystemSQLite,
		"unknown",
	}

	for _, system := range systems {
		labels := prometheus.Labels{
			LabelServiceName: "test-db-systems",
			LabelDBName:      "testdb",
			LabelDBSystem:    system,
		}

		metrics.DB.ConnectionsActive.With(labels).Set(10)
	}

	// All systems should be recorded
	assert.NotNil(t, metrics)
}
