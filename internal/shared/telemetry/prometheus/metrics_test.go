// Package prometheus provides tests for metric definitions.
package prometheus

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHTTPMetrics(t *testing.T) {
	t.Run("creates all HTTP metrics", func(t *testing.T) {
		cfg := Config{ServiceName: "test-http"}
		reg, err := NewRegistry(cfg)
		require.NoError(t, err)

		httpMetrics := NewHTTPMetrics(reg)

		assert.NotNil(t, httpMetrics)
		assert.NotNil(t, httpMetrics.RequestsTotal)
		assert.NotNil(t, httpMetrics.RequestDuration)
		assert.NotNil(t, httpMetrics.ResponseSize)
		assert.NotNil(t, httpMetrics.RequestsInFlight)
		assert.NotNil(t, httpMetrics.ErrorsTotal)
	})

	t.Run("registers metrics with registry", func(t *testing.T) {
		cfg := Config{ServiceName: "test-register"}
		reg, err := NewRegistry(cfg)
		require.NoError(t, err)

		httpMetrics := NewHTTPMetrics(reg)

		// Try to get metric values - if registered, this should work
		labels := prometheus.Labels{
			LabelServiceName: "test-register",
			LabelMethod:      "GET",
			LabelPath:        "/api/test",
			LabelStatus:      "200",
		}

		// Increment and check
		httpMetrics.RequestsTotal.With(labels).Inc()
		httpMetrics.RequestDuration.With(prometheus.Labels{
			LabelServiceName: "test-register",
			LabelMethod:      "GET",
			LabelPath:        "/api/test",
		}).Observe(0.1)
	})
}

func TestHTTPMetrics_IncRequest(t *testing.T) {
	cfg := Config{ServiceName: "test-inc"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	httpMetrics := NewHTTPMetrics(reg)

	labels := prometheus.Labels{
		LabelServiceName: "test-inc",
		LabelMethod:      "POST",
		LabelPath:        "/users",
		LabelStatus:      "201",
	}

	// Get initial value
	var metric dto.Metric
	err = httpMetrics.RequestsTotal.With(labels).Write(&metric)
	require.NoError(t, err)
	initial := metric.Counter.GetValue()

	// Increment
	httpMetrics.RequestsTotal.With(labels).Inc()

	// Get new value
	err = httpMetrics.RequestsTotal.With(labels).Write(&metric)
	require.NoError(t, err)
	newValue := metric.Counter.GetValue()

	assert.Equal(t, initial+1, newValue)
}

func TestHTTPMetrics_ObserveDuration(t *testing.T) {
	cfg := Config{ServiceName: "test-duration"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	httpMetrics := NewHTTPMetrics(reg)

	labels := prometheus.Labels{
		LabelServiceName: "test-duration",
		LabelMethod:      "GET",
		LabelPath:        "/api/data",
	}

	// Observe some durations
	httpMetrics.RequestDuration.With(labels).Observe(0.1)
	httpMetrics.RequestDuration.With(labels).Observe(0.2)
	httpMetrics.RequestDuration.With(labels).Observe(0.3)

	// Just verify no panic - metrics are recorded
}

func TestHTTPMetrics_ObserveResponseSize(t *testing.T) {
	cfg := Config{ServiceName: "test-size"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	httpMetrics := NewHTTPMetrics(reg)

	labels := prometheus.Labels{
		LabelServiceName: "test-size",
		LabelMethod:      "GET",
		LabelPath:        "/download",
	}

	sizes := []float64{100, 1000, 10000, 100000}
	for _, size := range sizes {
		httpMetrics.ResponseSize.With(labels).Observe(size)
	}

	// Just verify no panic - metrics are recorded
}

func TestHTTPMetrics_IncErrors(t *testing.T) {
	cfg := Config{ServiceName: "test-errors"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	httpMetrics := NewHTTPMetrics(reg)

	labels := prometheus.Labels{
		LabelServiceName: "test-errors",
		LabelMethod:      "GET",
		LabelPath:        "/notfound",
		LabelStatus:      "404",
		LabelErrorCode:   "NOT_FOUND",
	}

	httpMetrics.ErrorsTotal.With(labels).Inc()
	httpMetrics.ErrorsTotal.With(labels).Inc()

	var metric dto.Metric
	err = httpMetrics.ErrorsTotal.With(labels).Write(&metric)
	require.NoError(t, err)

	assert.Equal(t, float64(2), metric.Counter.GetValue())
}

func TestNewDBMetrics(t *testing.T) {
	t.Run("creates all DB metrics", func(t *testing.T) {
		cfg := Config{ServiceName: "test-db"}
		reg, err := NewRegistry(cfg)
		require.NoError(t, err)

		dbMetrics := NewDBMetrics(reg)

		assert.NotNil(t, dbMetrics.ConnectionsActive)
		assert.NotNil(t, dbMetrics.ConnectionsIdle)
		assert.NotNil(t, dbMetrics.QueryDuration)
		assert.NotNil(t, dbMetrics.QueriesTotal)
		assert.NotNil(t, dbMetrics.QueryErrorsTotal)
		assert.NotNil(t, dbMetrics.RowsAffected)
		assert.NotNil(t, dbMetrics.TransactionsTotal)
		assert.NotNil(t, dbMetrics.TransactionDuration)
	})
}

func TestDBMetrics_SetConnections(t *testing.T) {
	cfg := Config{ServiceName: "test-conn"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	dbMetrics := NewDBMetrics(reg)

	labels := prometheus.Labels{
		LabelServiceName: "test-conn",
		LabelDBName:      "openprint",
		LabelDBSystem:    DBSystemPostgreSQL,
	}

	// Set connection counts
	dbMetrics.ConnectionsActive.With(labels).Set(10)
	dbMetrics.ConnectionsIdle.With(labels).Set(5)

	var metric dto.Metric

	err = dbMetrics.ConnectionsActive.With(labels).Write(&metric)
	require.NoError(t, err)
	assert.Equal(t, float64(10), metric.Gauge.GetValue())

	err = dbMetrics.ConnectionsIdle.With(labels).Write(&metric)
	require.NoError(t, err)
	assert.Equal(t, float64(5), metric.Gauge.GetValue())
}

func TestDBMetrics_QueryTracking(t *testing.T) {
	cfg := Config{ServiceName: "test-query"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	dbMetrics := NewDBMetrics(reg)

	labels := prometheus.Labels{
		LabelServiceName: "test-query",
		LabelDBName:      "openprint",
		LabelOperation:   "SELECT",
	}

	// Record query
	dbMetrics.QueriesTotal.With(labels).Inc()
	dbMetrics.QueryDuration.With(labels).Observe(0.05)

	// Just verify no panic - metrics are recorded
}

func TestDBMetrics_TransactionTracking(t *testing.T) {
	cfg := Config{ServiceName: "test-tx"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	dbMetrics := NewDBMetrics(reg)

	labels := prometheus.Labels{
		LabelServiceName: "test-tx",
		LabelDBName:      "openprint",
		LabelSuccess:     SuccessTrue,
	}

	dbMetrics.TransactionsTotal.With(labels).Inc()
	dbMetrics.TransactionDuration.With(prometheus.Labels{
		LabelServiceName: "test-tx",
		LabelDBName:      "openprint",
	}).Observe(0.15)

	// Just verify no panic - metrics are recorded
}

func TestNewRedisMetrics(t *testing.T) {
	t.Run("creates all Redis metrics", func(t *testing.T) {
		cfg := Config{ServiceName: "test-redis"}
		reg, err := NewRegistry(cfg)
		require.NoError(t, err)

		redisMetrics := NewRedisMetrics(reg)

		assert.NotNil(t, redisMetrics.CommandsTotal)
		assert.NotNil(t, redisMetrics.CommandDuration)
		assert.NotNil(t, redisMetrics.CommandErrorsTotal)
		assert.NotNil(t, redisMetrics.ConnectionsActive)
		assert.NotNil(t, redisMetrics.ConnectionsIdle)
		assert.NotNil(t, redisMetrics.PoolHitsTotal)
		assert.NotNil(t, redisMetrics.PoolMissesTotal)
		assert.NotNil(t, redisMetrics.PoolTimeoutsTotal)
	})
}

func TestRedisMetrics_CommandTracking(t *testing.T) {
	cfg := Config{ServiceName: "test-redis-cmd"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	redisMetrics := NewRedisMetrics(reg)

	labels := prometheus.Labels{
		LabelServiceName:  "test-redis-cmd",
		LabelRedisDB:      "0",
		LabelRedisCommand: "GET",
	}

	redisMetrics.CommandsTotal.With(labels).Inc()
	redisMetrics.CommandDuration.With(labels).Observe(0.001)

	var metric dto.Metric

	err = redisMetrics.CommandsTotal.With(labels).Write(&metric)
	require.NoError(t, err)
	assert.Equal(t, float64(1), metric.Counter.GetValue())
}

func TestRedisMetrics_PoolTracking(t *testing.T) {
	cfg := Config{ServiceName: "test-redis-pool"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	redisMetrics := NewRedisMetrics(reg)

	labels := prometheus.Labels{
		LabelServiceName: "test-redis-pool",
		LabelRedisDB:     "0",
	}

	redisMetrics.PoolHitsTotal.With(labels).Add(10)
	redisMetrics.PoolMissesTotal.With(labels).Add(2)
	redisMetrics.PoolTimeoutsTotal.With(labels).Add(1)

	var metric dto.Metric

	err = redisMetrics.PoolHitsTotal.With(labels).Write(&metric)
	require.NoError(t, err)
	assert.Equal(t, float64(10), metric.Counter.GetValue())

	err = redisMetrics.PoolMissesTotal.With(labels).Write(&metric)
	require.NoError(t, err)
	assert.Equal(t, float64(2), metric.Counter.GetValue())
}

func TestNewBusinessMetrics(t *testing.T) {
	t.Run("creates all business metrics", func(t *testing.T) {
		cfg := Config{ServiceName: "test-business"}
		reg, err := NewRegistry(cfg)
		require.NoError(t, err)

		businessMetrics := NewBusinessMetrics(reg)

		assert.NotNil(t, businessMetrics.AuthSuccessTotal)
		assert.NotNil(t, businessMetrics.AuthFailureTotal)
		assert.NotNil(t, businessMetrics.JobsCreatedTotal)
		assert.NotNil(t, businessMetrics.JobsCompletedTotal)
		assert.NotNil(t, businessMetrics.JobsFailedTotal)
		assert.NotNil(t, businessMetrics.JobProcessingDuration)
		assert.NotNil(t, businessMetrics.PrintersRegisteredTotal)
		assert.NotNil(t, businessMetrics.PrinterHeartbeatsTotal)
		assert.NotNil(t, businessMetrics.DocumentsStoredTotal)
		assert.NotNil(t, businessMetrics.DocumentsRetrievedTotal)
		assert.NotNil(t, businessMetrics.DocumentStorageSize)
		assert.NotNil(t, businessMetrics.WebSocketConnectionsActive)
		assert.NotNil(t, businessMetrics.WebSocketMessagesTotal)
	})
}

func TestBusinessMetrics_AuthTracking(t *testing.T) {
	cfg := Config{ServiceName: "test-auth"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	businessMetrics := NewBusinessMetrics(reg)

	successLabels := prometheus.Labels{
		LabelServiceName: "test-auth",
		LabelAuthMethod:  AuthMethodPassword,
		LabelRole:        "admin",
	}

	failureLabels := prometheus.Labels{
		LabelServiceName: "test-auth",
		LabelAuthMethod:  AuthMethodPassword,
	}

	businessMetrics.AuthSuccessTotal.With(successLabels).Inc()
	businessMetrics.AuthFailureTotal.With(failureLabels).Inc()

	var metric dto.Metric

	err = businessMetrics.AuthSuccessTotal.With(successLabels).Write(&metric)
	require.NoError(t, err)
	assert.Equal(t, float64(1), metric.Counter.GetValue())

	err = businessMetrics.AuthFailureTotal.With(failureLabels).Write(&metric)
	require.NoError(t, err)
	assert.Equal(t, float64(1), metric.Counter.GetValue())
}

func TestBusinessMetrics_JobTracking(t *testing.T) {
	cfg := Config{ServiceName: "test-jobs"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	businessMetrics := NewBusinessMetrics(reg)

	labels := prometheus.Labels{
		LabelServiceName: "test-jobs",
		LabelOrgID:       "org-123",
	}

	businessMetrics.JobsCreatedTotal.With(labels).Inc()
	businessMetrics.JobsCompletedTotal.With(labels).Inc()
	businessMetrics.JobProcessingDuration.With(labels).Observe(30.5)

	// Just verify no panic - metrics are recorded
}

func TestBusinessMetrics_JobFailureTracking(t *testing.T) {
	cfg := Config{ServiceName: "test-job-fail"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	businessMetrics := NewBusinessMetrics(reg)

	labels := prometheus.Labels{
		LabelServiceName: "test-job-fail",
		LabelOrgID:       "org-456",
		LabelErrorCode:   "OUT_OF_PAPER",
	}

	businessMetrics.JobsFailedTotal.With(labels).Inc()
	businessMetrics.JobsFailedTotal.With(labels).Inc()

	// Just verify no panic - metrics are recorded
}

func TestBusinessMetrics_PrinterTracking(t *testing.T) {
	cfg := Config{ServiceName: "test-printers"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	businessMetrics := NewBusinessMetrics(reg)

	labels := prometheus.Labels{
		LabelServiceName: "test-printers",
		LabelOrgID:       "org-789",
	}

	businessMetrics.PrintersRegisteredTotal.With(labels).Inc()
	businessMetrics.PrinterHeartbeatsTotal.With(labels).Inc()
	businessMetrics.PrinterHeartbeatsTotal.With(labels).Inc()

	var metric dto.Metric

	err = businessMetrics.PrintersRegisteredTotal.With(labels).Write(&metric)
	require.NoError(t, err)
	assert.Equal(t, float64(1), metric.Counter.GetValue())

	err = businessMetrics.PrinterHeartbeatsTotal.With(labels).Write(&metric)
	require.NoError(t, err)
	assert.Equal(t, float64(2), metric.Counter.GetValue())
}

func TestBusinessMetrics_DocumentTracking(t *testing.T) {
	cfg := Config{ServiceName: "test-docs"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	businessMetrics := NewBusinessMetrics(reg)

	storeLabels := prometheus.Labels{
		LabelServiceName:    "test-docs",
		LabelStorageBackend: StorageBackendS3,
		LabelDocumentType:   "application/pdf",
	}

	backendLabels := prometheus.Labels{
		LabelServiceName:    "test-docs",
		LabelStorageBackend: StorageBackendS3,
	}

	businessMetrics.DocumentsStoredTotal.With(storeLabels).Inc()
	businessMetrics.DocumentStorageSize.With(backendLabels).Add(1024000)
	businessMetrics.DocumentsRetrievedTotal.With(backendLabels).Inc()

	var metric dto.Metric

	err = businessMetrics.DocumentsStoredTotal.With(storeLabels).Write(&metric)
	require.NoError(t, err)
	assert.Equal(t, float64(1), metric.Counter.GetValue())

	err = businessMetrics.DocumentStorageSize.With(backendLabels).Write(&metric)
	require.NoError(t, err)
	assert.Equal(t, float64(1024000), metric.Gauge.GetValue())
}

func TestBusinessMetrics_WebSocketTracking(t *testing.T) {
	cfg := Config{ServiceName: "test-ws"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	businessMetrics := NewBusinessMetrics(reg)

	labels := prometheus.Labels{
		LabelServiceName: "test-ws",
	}

	// Simulate connections
	businessMetrics.WebSocketConnectionsActive.With(labels).Inc()
	businessMetrics.WebSocketConnectionsActive.With(labels).Inc()
	businessMetrics.WebSocketConnectionsActive.With(labels).Inc()
	businessMetrics.WebSocketMessagesTotal.With(labels).Inc()

	// Disconnect one
	businessMetrics.WebSocketConnectionsActive.With(labels).Dec()

	var metric dto.Metric

	err = businessMetrics.WebSocketConnectionsActive.With(labels).Write(&metric)
	require.NoError(t, err)
	assert.Equal(t, float64(2), metric.Gauge.GetValue())

	err = businessMetrics.WebSocketMessagesTotal.With(labels).Write(&metric)
	require.NoError(t, err)
	assert.Equal(t, float64(1), metric.Counter.GetValue())
}

func TestNewMetrics(t *testing.T) {
	t.Run("creates aggregate metrics", func(t *testing.T) {
		cfg := Config{ServiceName: "test-aggregate"}
		reg, err := NewRegistry(cfg)
		require.NoError(t, err)

		metrics := NewMetrics(reg)

		assert.NotNil(t, metrics.HTTP)
		assert.NotNil(t, metrics.DB)
		assert.NotNil(t, metrics.Redis)
		assert.NotNil(t, metrics.Business)
		assert.NotNil(t, metrics.Registry())
		assert.Same(t, reg, metrics.Registry())
	})
}

func TestMetrics_Registry(t *testing.T) {
	cfg := Config{ServiceName: "test-reg-access"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	assert.Same(t, reg, metrics.Registry())
}

func TestMetrics_LabelConsistency(t *testing.T) {
	cfg := Config{ServiceName: "test-labels"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	serviceName := "test-labels"

	// Test that all metrics accept the same service label
	httpLabels := prometheus.Labels{
		LabelServiceName: serviceName,
		LabelMethod:      "GET",
		LabelPath:        "/test",
		LabelStatus:      "200",
	}
	metrics.HTTP.RequestsTotal.With(httpLabels).Inc()

	dbLabels := prometheus.Labels{
		LabelServiceName: serviceName,
		LabelDBName:      "testdb",
		LabelOperation:   "SELECT",
	}
	metrics.DB.QueriesTotal.With(dbLabels).Inc()

	redisLabels := prometheus.Labels{
		LabelServiceName:  serviceName,
		LabelRedisDB:      "0",
		LabelRedisCommand: "GET",
	}
	metrics.Redis.CommandsTotal.With(redisLabels).Inc()

	businessLabels := prometheus.Labels{
		LabelServiceName: serviceName,
		LabelOrgID:       "org-1",
	}
	metrics.Business.JobsCreatedTotal.With(businessLabels).Inc()
}
