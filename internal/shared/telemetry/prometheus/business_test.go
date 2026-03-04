// Package prometheus provides tests for business metrics.
package prometheus

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordJobCreated(t *testing.T) {
	cfg := Config{ServiceName: "test-record-job"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	RecordJobCreated(metrics, "test-service", "org-123")

	// Metric should be recorded
	labels := prometheus.Labels{
		LabelServiceName: "test-service",
		LabelOrgID:       "org-123",
	}

	// Check via direct metric access
	var metric dto.Metric
	err = metrics.Business.JobsCreatedTotal.With(labels).Write(&metric)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), metric.Counter.GetValue())
}

func TestRecordJobCompleted(t *testing.T) {
	cfg := Config{ServiceName: "test-job-complete"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	duration := 30 * time.Second

	RecordJobCompleted(metrics, "test-service", "org-456", duration)

	// Both completed count and duration should be recorded
	labels := prometheus.Labels{
		LabelServiceName: "test-service",
		LabelOrgID:       "org-456",
	}

	var metric dto.Metric
	err = metrics.Business.JobsCompletedTotal.With(labels).Write(&metric)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), metric.Counter.GetValue())
}

func TestRecordJobCompleted_ZeroDuration(t *testing.T) {
	cfg := Config{ServiceName: "test-job-zero-dur"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	RecordJobCompleted(metrics, "test-service", "org-789", 0)

	// Should only increment counter, not record duration
	labels := prometheus.Labels{
		LabelServiceName: "test-service",
		LabelOrgID:       "org-789",
	}

	var metric dto.Metric
	err = metrics.Business.JobsCompletedTotal.With(labels).Write(&metric)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), metric.Counter.GetValue())
}

func TestRecordJobFailed(t *testing.T) {
	cfg := Config{ServiceName: "test-job-fail"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	RecordJobFailed(metrics, "test-service", "org-999", "OUT_OF_PAPER")

	labels := prometheus.Labels{
		LabelServiceName: "test-service",
		LabelOrgID:       "org-999",
		LabelErrorCode:   "OUT_OF_PAPER",
	}

	var metric dto.Metric
	err = metrics.Business.JobsFailedTotal.With(labels).Write(&metric)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), metric.Counter.GetValue())
}

func TestRecordAuthSuccess(t *testing.T) {
	cfg := Config{ServiceName: "test-auth-success"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	RecordAuthSuccess(metrics, "test-service", AuthMethodPassword, "admin")

	labels := prometheus.Labels{
		LabelServiceName: "test-service",
		LabelAuthMethod:  AuthMethodPassword,
		LabelRole:        "admin",
	}

	var metric dto.Metric
	err = metrics.Business.AuthSuccessTotal.With(labels).Write(&metric)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), metric.Counter.GetValue())
}

func TestRecordAuthFailure(t *testing.T) {
	cfg := Config{ServiceName: "test-auth-fail"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	RecordAuthFailure(metrics, "test-service", AuthMethodOIDC)

	labels := prometheus.Labels{
		LabelServiceName: "test-service",
		LabelAuthMethod:  AuthMethodOIDC,
	}

	var metric dto.Metric
	err = metrics.Business.AuthFailureTotal.With(labels).Write(&metric)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), metric.Counter.GetValue())
}

func TestRecordPrinterRegistered(t *testing.T) {
	cfg := Config{ServiceName: "test-printer-reg"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	RecordPrinterRegistered(metrics, "test-service", "org-123")

	labels := prometheus.Labels{
		LabelServiceName: "test-service",
		LabelOrgID:       "org-123",
	}

	var metric dto.Metric
	err = metrics.Business.PrintersRegisteredTotal.With(labels).Write(&metric)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), metric.Counter.GetValue())
}

func TestRecordPrinterHeartbeat(t *testing.T) {
	cfg := Config{ServiceName: "test-printer-heartbeat"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	RecordPrinterHeartbeat(metrics, "test-service", "org-456")

	// Record multiple heartbeats
	RecordPrinterHeartbeat(metrics, "test-service", "org-456")
	RecordPrinterHeartbeat(metrics, "test-service", "org-456")

	labels := prometheus.Labels{
		LabelServiceName: "test-service",
		LabelOrgID:       "org-456",
	}

	var metric dto.Metric
	err = metrics.Business.PrinterHeartbeatsTotal.With(labels).Write(&metric)
	assert.NoError(t, err)
	assert.Equal(t, float64(3), metric.Counter.GetValue())
}

func TestRecordDocumentStored(t *testing.T) {
	cfg := Config{ServiceName: "test-doc-store"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	RecordDocumentStored(metrics, "test-service", StorageBackendS3, "application/pdf", 1024000)

	labels := prometheus.Labels{
		LabelServiceName:    "test-service",
		LabelStorageBackend: StorageBackendS3,
		LabelDocumentType:   "application/pdf",
	}

	var metric dto.Metric
	err = metrics.Business.DocumentsStoredTotal.With(labels).Write(&metric)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), metric.Counter.GetValue())
}

func TestRecordDocumentStored_ZeroSize(t *testing.T) {
	cfg := Config{ServiceName: "test-doc-zero-size"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	RecordDocumentStored(metrics, "test-service", StorageBackendLocal, "text/plain", 0)

	// Should only increment counter
	labels := prometheus.Labels{
		LabelServiceName:    "test-service",
		LabelStorageBackend: StorageBackendLocal,
		LabelDocumentType:   "text/plain",
	}

	var metric dto.Metric
	err = metrics.Business.DocumentsStoredTotal.With(labels).Write(&metric)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), metric.Counter.GetValue())
}

func TestRecordDocumentRetrieved(t *testing.T) {
	cfg := Config{ServiceName: "test-doc-retrieve"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	RecordDocumentRetrieved(metrics, "test-service", StorageBackendS3)

	RecordDocumentRetrieved(metrics, "test-service", StorageBackendS3)

	labels := prometheus.Labels{
		LabelServiceName:    "test-service",
		LabelStorageBackend: StorageBackendS3,
	}

	var metric dto.Metric
	err = metrics.Business.DocumentsRetrievedTotal.With(labels).Write(&metric)
	assert.NoError(t, err)
	assert.Equal(t, float64(2), metric.Counter.GetValue())
}

func TestRecordDocumentDeleted(t *testing.T) {
	cfg := Config{ServiceName: "test-doc-delete"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	// First store a document
	RecordDocumentStored(metrics, "test-service", StorageBackendLocal, "application/pdf", 500000)

	// Then delete it
	RecordDocumentDeleted(metrics, "test-service", StorageBackendLocal, 500000)

	// Storage size should be back to 0 (or very close)
	labels := prometheus.Labels{
		LabelServiceName:    "test-service",
		LabelStorageBackend: StorageBackendLocal,
	}

	var metric dto.Metric
	err = metrics.Business.DocumentStorageSize.With(labels).Write(&metric)
	assert.NoError(t, err)
	assert.Equal(t, float64(0), metric.Gauge.GetValue())
}

func TestRecordWebSocketConnected(t *testing.T) {
	cfg := Config{ServiceName: "test-ws-connect"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	RecordWebSocketConnected(metrics, "test-service")
	RecordWebSocketConnected(metrics, "test-service")
	RecordWebSocketConnected(metrics, "test-service")

	labels := prometheus.Labels{
		LabelServiceName: "test-service",
	}

	var metric dto.Metric
	err = metrics.Business.WebSocketConnectionsActive.With(labels).Write(&metric)
	assert.NoError(t, err)
	assert.Equal(t, float64(3), metric.Gauge.GetValue())
}

func TestRecordWebSocketDisconnected(t *testing.T) {
	cfg := Config{ServiceName: "test-ws-disconnect"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	// Connect 3 clients
	RecordWebSocketConnected(metrics, "test-service")
	RecordWebSocketConnected(metrics, "test-service")
	RecordWebSocketConnected(metrics, "test-service")

	// Disconnect 1
	RecordWebSocketDisconnected(metrics, "test-service")

	labels := prometheus.Labels{
		LabelServiceName: "test-service",
	}

	var metric dto.Metric
	err = metrics.Business.WebSocketConnectionsActive.With(labels).Write(&metric)
	assert.NoError(t, err)
	assert.Equal(t, float64(2), metric.Gauge.GetValue())
}

func TestRecordWebSocketMessage(t *testing.T) {
	cfg := Config{ServiceName: "test-ws-msg"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	RecordWebSocketMessage(metrics, "test-service")
	RecordWebSocketMessage(metrics, "test-service")
	RecordWebSocketMessage(metrics, "test-service")

	labels := prometheus.Labels{
		LabelServiceName: "test-service",
	}

	var metric dto.Metric
	err = metrics.Business.WebSocketMessagesTotal.With(labels).Write(&metric)
	assert.NoError(t, err)
	assert.Equal(t, float64(3), metric.Counter.GetValue())
}

func TestRecordWebSocketBroadcast(t *testing.T) {
	cfg := Config{ServiceName: "test-ws-broadcast"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	// Broadcast to 5 clients
	RecordWebSocketBroadcast(metrics, "test-service", 5)

	// Broadcast to 10 clients
	RecordWebSocketBroadcast(metrics, "test-service", 10)

	labels := prometheus.Labels{
		LabelServiceName: "test-service",
	}

	var metric dto.Metric
	err = metrics.Business.WebSocketMessagesTotal.With(labels).Write(&metric)
	assert.NoError(t, err)
	assert.Equal(t, float64(15), metric.Counter.GetValue())
}

func TestNewJobMetricsRecorder(t *testing.T) {
	cfg := Config{ServiceName: "test-job-recorder"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	recorder := NewJobMetricsRecorder(metrics, "test-service", "org-123")

	assert.NotNil(t, recorder)
	assert.Equal(t, metrics, recorder.metrics)
	assert.Equal(t, "test-service", recorder.serviceName)
	assert.Equal(t, "org-123", recorder.orgID)
	assert.False(t, recorder.startTime.IsZero())

	// Should have recorded job creation
	labels := prometheus.Labels{
		LabelServiceName: "test-service",
		LabelOrgID:       "org-123",
	}

	var metric dto.Metric
	err = metrics.Business.JobsCreatedTotal.With(labels).Write(&metric)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), metric.Counter.GetValue())
}

func TestJobMetricsRecorder_Complete(t *testing.T) {
	cfg := Config{ServiceName: "test-job-complete-rec"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	recorder := NewJobMetricsRecorder(metrics, "test-service", "org-456")

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	recorder.Complete()

	// Check completion was recorded
	labels := prometheus.Labels{
		LabelServiceName: "test-service",
		LabelOrgID:       "org-456",
	}

	var metric dto.Metric
	err = metrics.Business.JobsCompletedTotal.With(labels).Write(&metric)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), metric.Counter.GetValue())
}

func TestJobMetricsRecorder_Fail(t *testing.T) {
	cfg := Config{ServiceName: "test-job-fail-rec"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	recorder := NewJobMetricsRecorder(metrics, "test-service", "org-789")

	recorder.Fail("PAPER_JAM")

	labels := prometheus.Labels{
		LabelServiceName: "test-service",
		LabelOrgID:       "org-789",
		LabelErrorCode:   "PAPER_JAM",
	}

	var metric dto.Metric
	err = metrics.Business.JobsFailedTotal.With(labels).Write(&metric)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), metric.Counter.GetValue())
}

func TestNewAuthMetricsRecorder(t *testing.T) {
	cfg := Config{ServiceName: "test-auth-recorder"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	recorder := NewAuthMetricsRecorder(metrics, "test-service", AuthMethodPassword)

	assert.NotNil(t, recorder)
	assert.Equal(t, metrics, recorder.metrics)
	assert.Equal(t, "test-service", recorder.serviceName)
	assert.Equal(t, AuthMethodPassword, recorder.authMethod)
	assert.False(t, recorder.success)
}

func TestAuthMetricsRecorder_Success(t *testing.T) {
	cfg := Config{ServiceName: "test-auth-success-rec"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	recorder := NewAuthMetricsRecorder(metrics, "test-service", AuthMethodOIDC)

	recorder.Success("admin")

	assert.True(t, recorder.success)

	labels := prometheus.Labels{
		LabelServiceName: "test-service",
		LabelAuthMethod:  AuthMethodOIDC,
		LabelRole:        "admin",
	}

	var metric dto.Metric
	err = metrics.Business.AuthSuccessTotal.With(labels).Write(&metric)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), metric.Counter.GetValue())
}

func TestAuthMetricsRecorder_Failure(t *testing.T) {
	cfg := Config{ServiceName: "test-auth-fail-rec"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	recorder := NewAuthMetricsRecorder(metrics, "test-service", AuthMethodSAML)

	recorder.Failure()

	labels := prometheus.Labels{
		LabelServiceName: "test-service",
		LabelAuthMethod:  AuthMethodSAML,
	}

	var metric dto.Metric
	err = metrics.Business.AuthFailureTotal.With(labels).Write(&metric)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), metric.Counter.GetValue())
}

func TestNewStorageMetricsRecorder(t *testing.T) {
	cfg := Config{ServiceName: "test-storage-recorder"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	recorder := NewStorageMetricsRecorder(metrics, "test-service", StorageBackendS3)

	assert.NotNil(t, recorder)
	assert.Equal(t, metrics, recorder.metrics)
	assert.Equal(t, "test-service", recorder.serviceName)
	assert.Equal(t, StorageBackendS3, recorder.backend)
}

func TestStorageMetricsRecorder_Store(t *testing.T) {
	cfg := Config{ServiceName: "test-storage-store"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	recorder := NewStorageMetricsRecorder(metrics, "test-service", StorageBackendLocal)

	recorder.Store("application/pdf", 2048000)

	storeLabels := prometheus.Labels{
		LabelServiceName:    "test-service",
		LabelStorageBackend: StorageBackendLocal,
		LabelDocumentType:   "application/pdf",
	}

	storageLabels := prometheus.Labels{
		LabelServiceName:    "test-service",
		LabelStorageBackend: StorageBackendLocal,
	}

	var metric dto.Metric
	err = metrics.Business.DocumentsStoredTotal.With(storeLabels).Write(&metric)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), metric.Counter.GetValue())

	err = metrics.Business.DocumentStorageSize.With(storageLabels).Write(&metric)
	assert.NoError(t, err)
	assert.Equal(t, float64(2048000), metric.Gauge.GetValue())
}

func TestStorageMetricsRecorder_Retrieve(t *testing.T) {
	cfg := Config{ServiceName: "test-storage-retrieve"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	recorder := NewStorageMetricsRecorder(metrics, "test-service", StorageBackendS3)

	recorder.Retrieve()
	recorder.Retrieve()

	labels := prometheus.Labels{
		LabelServiceName:    "test-service",
		LabelStorageBackend: StorageBackendS3,
	}

	var metric dto.Metric
	err = metrics.Business.DocumentsRetrievedTotal.With(labels).Write(&metric)
	assert.NoError(t, err)
	assert.Equal(t, float64(2), metric.Counter.GetValue())
}

func TestStorageMetricsRecorder_Delete(t *testing.T) {
	cfg := Config{ServiceName: "test-storage-delete"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	recorder := NewStorageMetricsRecorder(metrics, "test-service", StorageBackendLocal)

	// First store
	recorder.Store("text/plain", 1000000)

	// Then delete
	recorder.Delete(1000000)

	labels := prometheus.Labels{
		LabelServiceName:    "test-service",
		LabelStorageBackend: StorageBackendLocal,
	}

	var metric dto.Metric
	err = metrics.Business.DocumentStorageSize.With(labels).Write(&metric)
	assert.NoError(t, err)
	assert.Equal(t, float64(0), metric.Gauge.GetValue())
}

func TestNewPrinterMetricsRecorder(t *testing.T) {
	cfg := Config{ServiceName: "test-printer-recorder"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	recorder := NewPrinterMetricsRecorder(metrics, "test-service", "org-123")

	assert.NotNil(t, recorder)
	assert.Equal(t, metrics, recorder.metrics)
	assert.Equal(t, "test-service", recorder.serviceName)
	assert.Equal(t, "org-123", recorder.orgID)
}

func TestPrinterMetricsRecorder_Register(t *testing.T) {
	cfg := Config{ServiceName: "test-printer-reg-rec"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	recorder := NewPrinterMetricsRecorder(metrics, "test-service", "org-456")

	recorder.Register()

	labels := prometheus.Labels{
		LabelServiceName: "test-service",
		LabelOrgID:       "org-456",
	}

	var metric dto.Metric
	err = metrics.Business.PrintersRegisteredTotal.With(labels).Write(&metric)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), metric.Counter.GetValue())
}

func TestPrinterMetricsRecorder_Heartbeat(t *testing.T) {
	cfg := Config{ServiceName: "test-printer-heartbeat-rec"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	recorder := NewPrinterMetricsRecorder(metrics, "test-service", "org-789")

	recorder.Heartbeat()
	recorder.Heartbeat()
	recorder.Heartbeat()

	labels := prometheus.Labels{
		LabelServiceName: "test-service",
		LabelOrgID:       "org-789",
	}

	var metric dto.Metric
	err = metrics.Business.PrinterHeartbeatsTotal.With(labels).Write(&metric)
	assert.NoError(t, err)
	assert.Equal(t, float64(3), metric.Counter.GetValue())
}

func TestNewWebSocketMetricsRecorder(t *testing.T) {
	cfg := Config{ServiceName: "test-ws-recorder"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	recorder := NewWebSocketMetricsRecorder(metrics, "test-service")

	assert.NotNil(t, recorder)
	assert.Equal(t, metrics, recorder.metrics)
	assert.Equal(t, "test-service", recorder.serviceName)
}

func TestWebSocketMetricsRecorder_Connected(t *testing.T) {
	cfg := Config{ServiceName: "test-ws-connected-rec"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	recorder := NewWebSocketMetricsRecorder(metrics, "test-service")

	recorder.Connected()
	recorder.Connected()

	labels := prometheus.Labels{
		LabelServiceName: "test-service",
	}

	var metric dto.Metric
	err = metrics.Business.WebSocketConnectionsActive.With(labels).Write(&metric)
	assert.NoError(t, err)
	assert.Equal(t, float64(2), metric.Gauge.GetValue())
}

func TestWebSocketMetricsRecorder_Disconnected(t *testing.T) {
	cfg := Config{ServiceName: "test-ws-disconnected-rec"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	recorder := NewWebSocketMetricsRecorder(metrics, "test-service")

	recorder.Connected()
	recorder.Connected()
	recorder.Connected()
	recorder.Disconnected()

	labels := prometheus.Labels{
		LabelServiceName: "test-service",
	}

	var metric dto.Metric
	err = metrics.Business.WebSocketConnectionsActive.With(labels).Write(&metric)
	assert.NoError(t, err)
	assert.Equal(t, float64(2), metric.Gauge.GetValue())
}

func TestWebSocketMetricsRecorder_Message(t *testing.T) {
	cfg := Config{ServiceName: "test-ws-message-rec"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	recorder := NewWebSocketMetricsRecorder(metrics, "test-service")

	recorder.Message()
	recorder.Message()
	recorder.Message()
	recorder.Message()

	labels := prometheus.Labels{
		LabelServiceName: "test-service",
	}

	var metric dto.Metric
	err = metrics.Business.WebSocketMessagesTotal.With(labels).Write(&metric)
	assert.NoError(t, err)
	assert.Equal(t, float64(4), metric.Counter.GetValue())
}

func TestWebSocketMetricsRecorder_Broadcast(t *testing.T) {
	cfg := Config{ServiceName: "test-ws-broadcast-rec"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	recorder := NewWebSocketMetricsRecorder(metrics, "test-service")

	recorder.Broadcast(100)

	labels := prometheus.Labels{
		LabelServiceName: "test-service",
	}

	var metric dto.Metric
	err = metrics.Business.WebSocketMessagesTotal.With(labels).Write(&metric)
	assert.NoError(t, err)
	assert.Equal(t, float64(100), metric.Counter.GetValue())
}

func TestRecordPrinterMetric(t *testing.T) {
	cfg := Config{ServiceName: "test-record-printer-metric"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	t.Run("records heartbeat", func(t *testing.T) {
		RecordPrinterMetric(metrics, "test-service", "org-123", "printer-1", "heartbeat")
	})

	t.Run("records register", func(t *testing.T) {
		RecordPrinterMetric(metrics, "test-service", "org-456", "printer-2", "register")
	})

	t.Run("handles register_failed", func(t *testing.T) {
		// Currently no specific metric for this
		RecordPrinterMetric(metrics, "test-service", "org-789", "printer-3", "register_failed")
		// Should not panic
	})
}

func TestRecordStorageMetric(t *testing.T) {
	cfg := Config{ServiceName: "test-record-storage-metric"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	t.Run("records store", func(t *testing.T) {
		RecordStorageMetric(metrics, "test-service", StorageBackendS3, "application/pdf", "store", 1000000)
	})

	t.Run("records retrieve", func(t *testing.T) {
		RecordStorageMetric(metrics, "test-service", StorageBackendLocal, "", "retrieve", 0)
	})
}

func TestBusinessMetrics_MultipleOrganizations(t *testing.T) {
	cfg := Config{ServiceName: "test-multi-org"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	orgs := []string{"org-1", "org-2", "org-3", "org-4", "org-5"}

	for _, orgID := range orgs {
		RecordJobCreated(metrics, "test-service", orgID)
		RecordJobCompleted(metrics, "test-service", orgID, 10*time.Second)
	}

	// Each org should have its metrics recorded
	for _, orgID := range orgs {
		labels := prometheus.Labels{
			LabelServiceName: "test-service",
			LabelOrgID:       orgID,
		}

		var metric dto.Metric
		err = metrics.Business.JobsCreatedTotal.With(labels).Write(&metric)
		assert.NoError(t, err)
		assert.Equal(t, float64(1), metric.Counter.GetValue())
	}
}

func TestBusinessMetrics_MultipleAuthMethods(t *testing.T) {
	cfg := Config{ServiceName: "test-multi-auth"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	methods := []struct {
		method string
		role   string
	}{
		{AuthMethodPassword, "user"},
		{AuthMethodOIDC, "admin"},
		{AuthMethodSAML, "user"},
		{AuthMethodAPIKey, "service"},
	}

	for _, m := range methods {
		RecordAuthSuccess(metrics, "test-service", m.method, m.role)
	}

	// Each method should be recorded separately
	for _, m := range methods {
		labels := prometheus.Labels{
			LabelServiceName: "test-service",
			LabelAuthMethod:  m.method,
			LabelRole:        m.role,
		}

		var metric dto.Metric
		err = metrics.Business.AuthSuccessTotal.With(labels).Write(&metric)
		assert.NoError(t, err)
		assert.Equal(t, float64(1), metric.Counter.GetValue())
	}
}

func TestBusinessMetrics_DocumentTypes(t *testing.T) {
	cfg := Config{ServiceName: "test-doc-types"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	docTypes := []struct {
		backend string
		docType string
		size    int64
	}{
		{StorageBackendS3, "application/pdf", 1024000},
		{StorageBackendS3, "image/png", 500000},
		{StorageBackendLocal, "text/plain", 10000},
		{StorageBackendS3, "application/json", 5000},
	}

	for _, d := range docTypes {
		RecordDocumentStored(metrics, "test-service", d.backend, d.docType, d.size)
	}

	// Each document type should be tracked
	for _, d := range docTypes {
		labels := prometheus.Labels{
			LabelServiceName:    "test-service",
			LabelStorageBackend: d.backend,
			LabelDocumentType:   d.docType,
		}

		var metric dto.Metric
		err = metrics.Business.DocumentsStoredTotal.With(labels).Write(&metric)
		assert.NoError(t, err)
		assert.Equal(t, float64(1), metric.Counter.GetValue())
	}
}
