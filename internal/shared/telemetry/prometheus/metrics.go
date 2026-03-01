// Package prometheus provides standard metric types for HTTP, database, and business operations.
package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
)

// HTTPMetrics holds HTTP-related metrics.
type HTTPMetrics struct {
	// RequestsTotal is a counter for the total number of HTTP requests.
	RequestsTotal *prometheus.CounterVec

	// RequestDuration is a histogram for HTTP request latencies.
	RequestDuration *prometheus.HistogramVec

	// ResponseSize is a histogram for HTTP response sizes.
	ResponseSize *prometheus.HistogramVec

	// RequestsInFlight is a gauge for the number of requests currently being processed.
	RequestsInFlight *prometheus.GaugeVec

	// ErrorsTotal is a counter for HTTP errors (4xx and 5xx status codes).
	ErrorsTotal *prometheus.CounterVec
}

// NewHTTPMetrics creates a new HTTPMetrics instance with properly configured metrics.
// The metrics use the "openprint_http" namespace.
func NewHTTPMetrics(registry *Registry) *HTTPMetrics {
	labels := []string{
		LabelServiceName,
		LabelMethod,
		LabelPath,
		LabelStatus,
	}

	httpMetrics := &HTTPMetrics{
		RequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "openprint",
				Subsystem: "http",
				Name:      "requests_total",
				Help:      "Total number of HTTP requests processed.",
			},
			labels,
		),

		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "openprint",
				Subsystem: "http",
				Name:      "request_duration_seconds",
				Help:      "HTTP request latencies in seconds.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{LabelServiceName, LabelMethod, LabelPath},
		),

		ResponseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "openprint",
				Subsystem: "http",
				Name:      "response_size_bytes",
				Help:      "HTTP response sizes in bytes.",
				Buckets:   []float64{100, 1000, 10000, 100000, 1000000, 10000000},
			},
			[]string{LabelServiceName, LabelMethod, LabelPath},
		),

		RequestsInFlight: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "openprint",
				Subsystem: "http",
				Name:      "requests_in_flight",
				Help:      "Current number of HTTP requests being processed.",
			},
			[]string{LabelServiceName, LabelMethod, LabelPath},
		),

		ErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "openprint",
				Subsystem: "http",
				Name:      "errors_total",
				Help:      "Total number of HTTP errors (4xx and 5xx status codes).",
			},
			append(labels, LabelErrorCode),
		),
	}

	// Register all metrics
	registry.MustRegister(httpMetrics.RequestsTotal, "http_requests_total")
	registry.MustRegister(httpMetrics.RequestDuration, "http_request_duration")
	registry.MustRegister(httpMetrics.ResponseSize, "http_response_size")
	registry.MustRegister(httpMetrics.RequestsInFlight, "http_requests_in_flight")
	registry.MustRegister(httpMetrics.ErrorsTotal, "http_errors_total")

	return httpMetrics
}

// DBMetrics holds database-related metrics.
type DBMetrics struct {
	// ConnectionsActive is a gauge for the number of active database connections.
	ConnectionsActive *prometheus.GaugeVec

	// ConnectionsIdle is a gauge for the number of idle database connections.
	ConnectionsIdle *prometheus.GaugeVec

	// QueryDuration is a histogram for database query latencies.
	QueryDuration *prometheus.HistogramVec

	// QueriesTotal is a counter for the total number of database queries.
	QueriesTotal *prometheus.CounterVec

	// QueryErrorsTotal is a counter for database query errors.
	QueryErrorsTotal *prometheus.CounterVec

	// RowsAffected is a histogram for the number of rows affected by queries.
	RowsAffected *prometheus.HistogramVec

	// TransactionsTotal is a counter for database transactions.
	TransactionsTotal *prometheus.CounterVec

	// TransactionDuration is a histogram for transaction latencies.
	TransactionDuration *prometheus.HistogramVec
}

// NewDBMetrics creates a new DBMetrics instance with properly configured metrics.
func NewDBMetrics(registry *Registry) *DBMetrics {
	dbMetrics := &DBMetrics{
		ConnectionsActive: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "openprint",
				Subsystem: "db",
				Name:      "connections_active",
				Help:      "Number of active database connections.",
			},
			[]string{LabelServiceName, LabelDBName, LabelDBSystem},
		),

		ConnectionsIdle: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "openprint",
				Subsystem: "db",
				Name:      "connections_idle",
				Help:      "Number of idle database connections.",
			},
			[]string{LabelServiceName, LabelDBName, LabelDBSystem},
		),

		QueryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "openprint",
				Subsystem: "db",
				Name:      "query_duration_seconds",
				Help:      "Database query latencies in seconds.",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{LabelServiceName, LabelDBName, LabelOperation},
		),

		QueriesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "openprint",
				Subsystem: "db",
				Name:      "queries_total",
				Help:      "Total number of database queries executed.",
			},
			[]string{LabelServiceName, LabelDBName, LabelOperation},
		),

		QueryErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "openprint",
				Subsystem: "db",
				Name:      "query_errors_total",
				Help:      "Total number of database query errors.",
			},
			[]string{LabelServiceName, LabelDBName, LabelOperation},
		),

		RowsAffected: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "openprint",
				Subsystem: "db",
				Name:      "rows_affected",
				Help:      "Number of rows affected by database queries.",
				Buckets:   []float64{1, 10, 100, 1000, 10000},
			},
			[]string{LabelServiceName, LabelDBName, LabelOperation},
		),

		TransactionsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "openprint",
				Subsystem: "db",
				Name:      "transactions_total",
				Help:      "Total number of database transactions.",
			},
			[]string{LabelServiceName, LabelDBName, LabelSuccess},
		),

		TransactionDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "openprint",
				Subsystem: "db",
				Name:      "transaction_duration_seconds",
				Help:      "Database transaction latencies in seconds.",
				Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{LabelServiceName, LabelDBName},
		),
	}

	// Register all metrics
	registry.MustRegister(dbMetrics.ConnectionsActive, "db_connections_active")
	registry.MustRegister(dbMetrics.ConnectionsIdle, "db_connections_idle")
	registry.MustRegister(dbMetrics.QueryDuration, "db_query_duration")
	registry.MustRegister(dbMetrics.QueriesTotal, "db_queries_total")
	registry.MustRegister(dbMetrics.QueryErrorsTotal, "db_query_errors_total")
	registry.MustRegister(dbMetrics.RowsAffected, "db_rows_affected")
	registry.MustRegister(dbMetrics.TransactionsTotal, "db_transactions_total")
	registry.MustRegister(dbMetrics.TransactionDuration, "db_transaction_duration")

	return dbMetrics
}

// RedisMetrics holds Redis-related metrics.
type RedisMetrics struct {
	// CommandsTotal is a counter for the total number of Redis commands.
	CommandsTotal *prometheus.CounterVec

	// CommandDuration is a histogram for Redis command latencies.
	CommandDuration *prometheus.HistogramVec

	// CommandErrorsTotal is a counter for Redis command errors.
	CommandErrorsTotal *prometheus.CounterVec

	// ConnectionsActive is a gauge for the number of active Redis connections.
	ConnectionsActive *prometheus.GaugeVec

	// ConnectionsIdle is a gauge for the number of idle Redis connections.
	ConnectionsIdle *prometheus.GaugeVec

	// PoolHitsTotal is a counter for connection pool hits.
	PoolHitsTotal *prometheus.CounterVec

	// PoolMissesTotal is a counter for connection pool misses.
	PoolMissesTotal *prometheus.CounterVec

	// PoolTimeoutsTotal is a counter for connection pool timeouts.
	PoolTimeoutsTotal *prometheus.CounterVec
}

// NewRedisMetrics creates a new RedisMetrics instance with properly configured metrics.
func NewRedisMetrics(registry *Registry) *RedisMetrics {
	redisMetrics := &RedisMetrics{
		CommandsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "openprint",
				Subsystem: "redis",
				Name:      "commands_total",
				Help:      "Total number of Redis commands executed.",
			},
			[]string{LabelServiceName, LabelRedisDB, LabelRedisCommand},
		),

		CommandDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "openprint",
				Subsystem: "redis",
				Name:      "command_duration_seconds",
				Help:      "Redis command latencies in seconds.",
				Buckets:   []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25},
			},
			[]string{LabelServiceName, LabelRedisDB, LabelRedisCommand},
		),

		CommandErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "openprint",
				Subsystem: "redis",
				Name:      "command_errors_total",
				Help:      "Total number of Redis command errors.",
			},
			[]string{LabelServiceName, LabelRedisDB, LabelRedisCommand},
		),

		ConnectionsActive: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "openprint",
				Subsystem: "redis",
				Name:      "connections_active",
				Help:      "Number of active Redis connections.",
			},
			[]string{LabelServiceName, LabelRedisDB},
		),

		ConnectionsIdle: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "openprint",
				Subsystem: "redis",
				Name:      "connections_idle",
				Help:      "Number of idle Redis connections.",
			},
			[]string{LabelServiceName, LabelRedisDB},
		),

		PoolHitsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "openprint",
				Subsystem: "redis",
				Name:      "pool_hits_total",
				Help:      "Total number of connection pool hits.",
			},
			[]string{LabelServiceName, LabelRedisDB},
		),

		PoolMissesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "openprint",
				Subsystem: "redis",
				Name:      "pool_misses_total",
				Help:      "Total number of connection pool misses.",
			},
			[]string{LabelServiceName, LabelRedisDB},
		),

		PoolTimeoutsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "openprint",
				Subsystem: "redis",
				Name:      "pool_timeouts_total",
				Help:      "Total number of connection pool timeouts.",
			},
			[]string{LabelServiceName, LabelRedisDB},
		),
	}

	// Register all metrics
	registry.MustRegister(redisMetrics.CommandsTotal, "redis_commands_total")
	registry.MustRegister(redisMetrics.CommandDuration, "redis_command_duration")
	registry.MustRegister(redisMetrics.CommandErrorsTotal, "redis_command_errors_total")
	registry.MustRegister(redisMetrics.ConnectionsActive, "redis_connections_active")
	registry.MustRegister(redisMetrics.ConnectionsIdle, "redis_connections_idle")
	registry.MustRegister(redisMetrics.PoolHitsTotal, "redis_pool_hits_total")
	registry.MustRegister(redisMetrics.PoolMissesTotal, "redis_pool_misses_total")
	registry.MustRegister(redisMetrics.PoolTimeoutsTotal, "redis_pool_timeouts_total")

	return redisMetrics
}

// BusinessMetrics holds business-specific metrics.
type BusinessMetrics struct {
	// AuthSuccessTotal is a counter for successful authentications.
	AuthSuccessTotal *prometheus.CounterVec

	// AuthFailureTotal is a counter for failed authentications.
	AuthFailureTotal *prometheus.CounterVec

	// JobsCreatedTotal is a counter for print jobs created.
	JobsCreatedTotal *prometheus.CounterVec

	// JobsCompletedTotal is a counter for print jobs completed.
	JobsCompletedTotal *prometheus.CounterVec

	// JobsFailedTotal is a counter for print jobs failed.
	JobsFailedTotal *prometheus.CounterVec

	// JobProcessingDuration is a histogram for job processing times.
	JobProcessingDuration *prometheus.HistogramVec

	// PrintersRegisteredTotal is a counter for printers registered.
	PrintersRegisteredTotal *prometheus.CounterVec

	// PrinterHeartbeatsTotal is a counter for printer heartbeats received.
	PrinterHeartbeatsTotal *prometheus.CounterVec

	// DocumentsStoredTotal is a counter for documents stored.
	DocumentsStoredTotal *prometheus.CounterVec

	// DocumentsRetrievedTotal is a counter for documents retrieved.
	DocumentsRetrievedTotal *prometheus.CounterVec

	// DocumentStorageSize is a gauge for total document storage size.
	DocumentStorageSize *prometheus.GaugeVec

	// WebSocketConnectionsActive is a gauge for active WebSocket connections.
	WebSocketConnectionsActive *prometheus.GaugeVec

	// WebSocketMessagesTotal is a counter for WebSocket messages sent.
	WebSocketMessagesTotal *prometheus.CounterVec
}

// NewBusinessMetrics creates a new BusinessMetrics instance with properly configured metrics.
func NewBusinessMetrics(registry *Registry) *BusinessMetrics {
	businessMetrics := &BusinessMetrics{
		AuthSuccessTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "openprint",
				Subsystem: "auth",
				Name:      "success_total",
				Help:      "Total number of successful authentications.",
			},
			[]string{LabelServiceName, LabelAuthMethod, LabelRole},
		),

		AuthFailureTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "openprint",
				Subsystem: "auth",
				Name:      "failures_total",
				Help:      "Total number of failed authentications.",
			},
			[]string{LabelServiceName, LabelAuthMethod},
		),

		JobsCreatedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "openprint",
				Subsystem: "job",
				Name:      "created_total",
				Help:      "Total number of print jobs created.",
			},
			[]string{LabelServiceName, LabelOrgID},
		),

		JobsCompletedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "openprint",
				Subsystem: "job",
				Name:      "completed_total",
				Help:      "Total number of print jobs completed successfully.",
			},
			[]string{LabelServiceName, LabelOrgID},
		),

		JobsFailedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "openprint",
				Subsystem: "job",
				Name:      "failed_total",
				Help:      "Total number of print jobs that failed.",
			},
			[]string{LabelServiceName, LabelOrgID, LabelErrorCode},
		),

		JobProcessingDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "openprint",
				Subsystem: "job",
				Name:      "processing_duration_seconds",
				Help:      "Time taken to process print jobs in seconds.",
				Buckets:   []float64{1, 5, 10, 30, 60, 120, 300, 600, 1800},
			},
			[]string{LabelServiceName, LabelOrgID},
		),

		PrintersRegisteredTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "openprint",
				Subsystem: "printer",
				Name:      "registered_total",
				Help:      "Total number of printers registered.",
			},
			[]string{LabelServiceName, LabelOrgID},
		),

		PrinterHeartbeatsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "openprint",
				Subsystem: "printer",
				Name:      "heartbeats_total",
				Help:      "Total number of printer heartbeats received.",
			},
			[]string{LabelServiceName, LabelOrgID},
		),

		DocumentsStoredTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "openprint",
				Subsystem: "document",
				Name:      "stored_total",
				Help:      "Total number of documents stored.",
			},
			[]string{LabelServiceName, LabelStorageBackend, LabelDocumentType},
		),

		DocumentsRetrievedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "openprint",
				Subsystem: "document",
				Name:      "retrieved_total",
				Help:      "Total number of documents retrieved.",
			},
			[]string{LabelServiceName, LabelStorageBackend},
		),

		DocumentStorageSize: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "openprint",
				Subsystem: "document",
				Name:      "storage_size_bytes",
				Help:      "Total size of stored documents in bytes.",
			},
			[]string{LabelServiceName, LabelStorageBackend},
		),

		WebSocketConnectionsActive: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "openprint",
				Subsystem: "websocket",
				Name:      "connections_active",
				Help:      "Current number of active WebSocket connections.",
			},
			[]string{LabelServiceName},
		),

		WebSocketMessagesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "openprint",
				Subsystem: "websocket",
				Name:      "messages_total",
				Help:      "Total number of WebSocket messages sent.",
			},
			[]string{LabelServiceName},
		),
	}

	// Register all metrics
	registry.MustRegister(businessMetrics.AuthSuccessTotal, "auth_success_total")
	registry.MustRegister(businessMetrics.AuthFailureTotal, "auth_failure_total")
	registry.MustRegister(businessMetrics.JobsCreatedTotal, "jobs_created_total")
	registry.MustRegister(businessMetrics.JobsCompletedTotal, "jobs_completed_total")
	registry.MustRegister(businessMetrics.JobsFailedTotal, "jobs_failed_total")
	registry.MustRegister(businessMetrics.JobProcessingDuration, "job_processing_duration")
	registry.MustRegister(businessMetrics.PrintersRegisteredTotal, "printers_registered_total")
	registry.MustRegister(businessMetrics.PrinterHeartbeatsTotal, "printer_heartbeats_total")
	registry.MustRegister(businessMetrics.DocumentsStoredTotal, "documents_stored_total")
	registry.MustRegister(businessMetrics.DocumentsRetrievedTotal, "documents_retrieved_total")
	registry.MustRegister(businessMetrics.DocumentStorageSize, "document_storage_size")
	registry.MustRegister(businessMetrics.WebSocketConnectionsActive, "websocket_connections_active")
	registry.MustRegister(businessMetrics.WebSocketMessagesTotal, "websocket_messages_total")

	return businessMetrics
}

// Metrics aggregates all metric types for a service.
type Metrics struct {
	HTTP     *HTTPMetrics
	DB       *DBMetrics
	Redis    *RedisMetrics
	Business *BusinessMetrics

	registry *Registry
}

// NewMetrics creates all metric types for a service.
func NewMetrics(registry *Registry) *Metrics {
	return &Metrics{
		HTTP:     NewHTTPMetrics(registry),
		DB:       NewDBMetrics(registry),
		Redis:    NewRedisMetrics(registry),
		Business: NewBusinessMetrics(registry),
		registry: registry,
	}
}

// Registry returns the associated registry.
func (m *Metrics) Registry() *Registry {
	return m.registry
}
