// Package prometheus provides database metrics collection wrappers.
package prometheus

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DBStatsCollector collects statistics from database connections.
// It wraps sql.DB or pgxpool.Pool to record metrics periodically.
type DBStatsCollector struct {
	metrics     *DBMetrics
	serviceName string
	dbName      string
	dbSystem    string
}

// NewDBStatsCollector creates a new database stats collector.
func NewDBStatsCollector(metrics *Metrics, serviceName, dbName, dbSystem string) *DBStatsCollector {
	return &DBStatsCollector{
		metrics:     metrics.DB,
		serviceName: serviceName,
		dbName:      dbName,
		dbSystem:    dbSystem,
	}
}

// RecordStats records the current database statistics.
// This should be called periodically (e.g., every 15 seconds).
func (c *DBStatsCollector) RecordStats(dbStats sql.DBStats) {
	c.metrics.ConnectionsActive.WithLabelValues(
		c.serviceName,
		c.dbName,
		c.dbSystem,
	).Set(float64(dbStats.OpenConnections))

	c.metrics.ConnectionsIdle.WithLabelValues(
		c.serviceName,
		c.dbName,
		c.dbSystem,
	).Set(float64(dbStats.Idle))

	// InUse = OpenConnections - Idle
	inUse := dbStats.OpenConnections - dbStats.Idle
	c.metrics.ConnectionsActive.WithLabelValues(
		c.serviceName,
		c.dbName,
		c.dbSystem,
	).Set(float64(inUse))
}

// RecordStatsFromPool records statistics from a pgxpool.Pool.
func (c *DBStatsCollector) RecordStatsFromPool(pool *pgxpool.Pool) {
	stat := pool.Stat()

	c.metrics.ConnectionsActive.WithLabelValues(
		c.serviceName,
		c.dbName,
		c.dbSystem,
	).Set(float64(stat.TotalConns() - stat.IdleConns()))

	c.metrics.ConnectionsIdle.WithLabelValues(
		c.serviceName,
		c.dbName,
		c.dbSystem,
	).Set(float64(stat.IdleConns()))
}

// DBTracer wraps database operations to record query metrics.
type DBTracer struct {
	metrics     *DBMetrics
	serviceName string
	dbName      string
	dbSystem    string
}

// NewDBTracer creates a new database tracer.
func NewDBTracer(metrics *Metrics, serviceName, dbName, dbSystem string) *DBTracer {
	return &DBTracer{
		metrics:     metrics.DB,
		serviceName: serviceName,
		dbName:      dbName,
		dbSystem:    dbSystem,
	}
}

// TraceQuery records a query execution.
func (t *DBTracer) TraceQuery(operation string, duration time.Duration, err error, rowsAffected int64) {
	// Record query duration
	t.metrics.QueryDuration.WithLabelValues(
		t.serviceName,
		t.dbName,
		operation,
	).Observe(duration.Seconds())

	// Record query count
	t.metrics.QueriesTotal.WithLabelValues(
		t.serviceName,
		t.dbName,
		operation,
	).Inc()

	// Record errors
	if err != nil {
		t.metrics.QueryErrorsTotal.WithLabelValues(
			t.serviceName,
			t.dbName,
			operation,
		).Inc()
	}

	// Record rows affected
	if rowsAffected > 0 {
		t.metrics.RowsAffected.WithLabelValues(
			t.serviceName,
			t.dbName,
			operation,
		).Observe(float64(rowsAffected))
	}
}

// TraceTransaction records a transaction execution.
func (t *DBTracer) TraceTransaction(duration time.Duration, success bool) {
	t.metrics.TransactionDuration.WithLabelValues(
		t.serviceName,
		t.dbName,
	).Observe(duration.Seconds())

	successLabel := SuccessFalse
	if success {
		successLabel = SuccessTrue
	}

	t.metrics.TransactionsTotal.WithLabelValues(
		t.serviceName,
		t.dbName,
		successLabel,
	).Inc()
}

// DBConfig holds database configuration for metrics collection.
type DBConfig struct {
	ServiceName string
	DBName      string
	DBSystem    string
}

// WrapDB wraps a sql.DB to collect metrics.
// The returned wrapper updates metrics every time stats are collected.
func WrapDB(db *sql.DB, registry *Registry, cfg DBConfig) *sql.DB {
	metrics := NewMetrics(registry)
	collector := NewDBStatsCollector(metrics, cfg.ServiceName, cfg.DBName, cfg.DBSystem)

	// Start background goroutine to collect stats
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			stats := db.Stats()
			collector.RecordStats(stats)
		}
	}()

	return db
}

// WrapPgxPool wraps a pgxpool.Pool to collect metrics.
func WrapPgxPool(pool *pgxpool.Pool, registry *Registry, cfg DBConfig) *pgxpool.Pool {
	metrics := NewMetrics(registry)
	collector := NewDBStatsCollector(metrics, cfg.ServiceName, cfg.DBName, cfg.DBSystem)

	// Start background goroutine to collect stats
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			collector.RecordStatsFromPool(pool)
		}
	}()

	return pool
}

// Querier is an interface for database query operations.
type Querier interface {
	Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row
	Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// QuerierWithMetrics wraps a Querier to collect query metrics.
type QuerierWithMetrics struct {
	querier  Querier
	tracer   *DBTracer
	delegate DBDelegate
}

// DBDelegate extracts operation names from queries.
type DBDelegate interface {
	OperationFromQuery(query string) string
}

// DefaultDBDelegate is a simple delegate that infers operation from SQL.
type DefaultDBDelegate struct{}

// OperationFromQuery extracts the operation name from a SQL query.
func (d *DefaultDBDelegate) OperationFromQuery(query string) string {
	// Simple heuristic: first word is the operation
	var op string
	for _, c := range query {
		if c == ' ' || c == '\n' || c == '\t' {
			break
		}
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			op += string(c)
		}
	}

	// Normalize operation name
	switch op {
	case "SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "DROP", "ALTER":
		return op
	case "BEGIN", "COMMIT", "ROLLBACK":
		return "transaction_" + op
	default:
		return "other"
	}
}

// NewQuerierWithMetrics creates a new querier wrapper with metrics.
func NewQuerierWithMetrics(querier Querier, metrics *Metrics, serviceName, dbName, dbSystem string) *QuerierWithMetrics {
	return &QuerierWithMetrics{
		querier:  querier,
		tracer:   NewDBTracer(metrics, serviceName, dbName, dbSystem),
		delegate: &DefaultDBDelegate{},
	}
}

// Query executes a query and records metrics.
func (q *QuerierWithMetrics) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	operation := q.delegate.OperationFromQuery(query)

	rows, err := q.querier.Query(ctx, query, args...)
	duration := time.Since(start)

	q.tracer.TraceQuery(operation, duration, err, 0)

	return rows, err
}

// QueryRow executes a query that returns a single row and records metrics.
func (q *QuerierWithMetrics) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	start := time.Now()
	operation := q.delegate.OperationFromQuery(query)

	// Note: QueryRow doesn't return an error directly
	// We'll record the duration, but won't know about errors until Scan
	row := q.querier.QueryRow(ctx, query, args...)
	duration := time.Since(start)

	q.tracer.TraceQuery(operation, duration, nil, 0)

	return row
}

// Exec executes a query without returning rows and records metrics.
func (q *QuerierWithMetrics) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	operation := q.delegate.OperationFromQuery(query)

	result, err := q.querier.Exec(ctx, query, args...)
	duration := time.Since(start)

	var rowsAffected int64
	if result != nil && err == nil {
		rowsAffected, _ = result.RowsAffected()
	}

	q.tracer.TraceQuery(operation, duration, err, rowsAffected)

	return result, err
}

// ExecWithMetrics is a helper function to execute a database operation with metrics.
func ExecWithMetrics(ctx context.Context, db *sql.DB, metrics *Metrics, serviceName, dbName, dbSystem, query string, args ...interface{}) (sql.Result, error) {
	tracer := NewDBTracer(metrics, serviceName, dbName, dbSystem)
	delegate := &DefaultDBDelegate{}
	operation := delegate.OperationFromQuery(query)

	start := time.Now()
	result, err := db.ExecContext(ctx, query, args...)
	duration := time.Since(start)

	var rowsAffected int64
	if result != nil && err == nil {
		rowsAffected, _ = result.RowsAffected()
	}

	tracer.TraceQuery(operation, duration, err, rowsAffected)

	return result, err
}

// QueryWithMetrics is a helper function to query with metrics.
func QueryWithMetrics(ctx context.Context, db *sql.DB, metrics *Metrics, serviceName, dbName, dbSystem, query string, args ...interface{}) (*sql.Rows, error) {
	tracer := NewDBTracer(metrics, serviceName, dbName, dbSystem)
	delegate := &DefaultDBDelegate{}
	operation := delegate.OperationFromQuery(query)

	start := time.Now()
	rows, err := db.QueryContext(ctx, query, args...)
	duration := time.Since(start)

	tracer.TraceQuery(operation, duration, err, 0)

	return rows, err
}

// RunInTransaction executes a function within a transaction and records metrics.
func RunInTransaction(ctx context.Context, db *sql.DB, metrics *Metrics, serviceName, dbName, dbSystem string, fn func(*sql.Tx) error) error {
	tracer := NewDBTracer(metrics, serviceName, dbName, dbSystem)

	start := time.Now()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		tracer.TraceTransaction(time.Since(start), false)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			tracer.TraceTransaction(time.Since(start), false)
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction failed: %v, rollback failed: %w", err, rbErr)
		}
		tracer.TraceTransaction(time.Since(start), false)
		return err
	}

	if err := tx.Commit(); err != nil {
		tracer.TraceTransaction(time.Since(start), false)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	tracer.TraceTransaction(time.Since(start), true)
	return nil
}
