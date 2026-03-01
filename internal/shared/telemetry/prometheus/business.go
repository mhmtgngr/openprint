// Package prometheus provides helper functions for recording business-specific metrics.
package prometheus

import (
	"time"
)

// RecordJobCreated records a new print job creation.
func RecordJobCreated(metrics *Metrics, serviceName, orgID string) {
	metrics.Business.JobsCreatedTotal.WithLabelValues(
		serviceName,
		orgID,
	).Inc()
}

// RecordJobCompleted records a successful print job completion.
func RecordJobCompleted(metrics *Metrics, serviceName, orgID string, duration time.Duration) {
	metrics.Business.JobsCompletedTotal.WithLabelValues(
		serviceName,
		orgID,
	).Inc()

	if duration > 0 {
		metrics.Business.JobProcessingDuration.WithLabelValues(
			serviceName,
			orgID,
		).Observe(duration.Seconds())
	}
}

// RecordJobFailed records a failed print job.
func RecordJobFailed(metrics *Metrics, serviceName, orgID, errorCode string) {
	metrics.Business.JobsFailedTotal.WithLabelValues(
		serviceName,
		orgID,
		errorCode,
	).Inc()
}

// RecordAuthSuccess records a successful authentication.
func RecordAuthSuccess(metrics *Metrics, serviceName, authMethod, role string) {
	metrics.Business.AuthSuccessTotal.WithLabelValues(
		serviceName,
		authMethod,
		role,
	).Inc()
}

// RecordAuthFailure records a failed authentication.
func RecordAuthFailure(metrics *Metrics, serviceName, authMethod string) {
	metrics.Business.AuthFailureTotal.WithLabelValues(
		serviceName,
		authMethod,
	).Inc()
}

// RecordPrinterRegistered records a new printer registration.
func RecordPrinterRegistered(metrics *Metrics, serviceName, orgID string) {
	metrics.Business.PrintersRegisteredTotal.WithLabelValues(
		serviceName,
		orgID,
	).Inc()
}

// RecordPrinterHeartbeat records a printer heartbeat.
func RecordPrinterHeartbeat(metrics *Metrics, serviceName, orgID string) {
	metrics.Business.PrinterHeartbeatsTotal.WithLabelValues(
		serviceName,
		orgID,
	).Inc()
}

// RecordDocumentStored records a document storage operation.
func RecordDocumentStored(metrics *Metrics, serviceName, backend, docType string, size int64) {
	metrics.Business.DocumentsStoredTotal.WithLabelValues(
		serviceName,
		backend,
		docType,
	).Inc()

	if size > 0 {
		metrics.Business.DocumentStorageSize.WithLabelValues(
			serviceName,
			backend,
		).Add(float64(size))
	}
}

// RecordDocumentRetrieved records a document retrieval operation.
func RecordDocumentRetrieved(metrics *Metrics, serviceName, backend string) {
	metrics.Business.DocumentsRetrievedTotal.WithLabelValues(
		serviceName,
		backend,
	).Inc()
}

// RecordDocumentDeleted records a document deletion and updates storage size.
func RecordDocumentDeleted(metrics *Metrics, serviceName, backend string, size int64) {
	if size > 0 {
		metrics.Business.DocumentStorageSize.WithLabelValues(
			serviceName,
			backend,
		).Sub(float64(size))
	}
}

// RecordWebSocketConnected records a new WebSocket connection.
func RecordWebSocketConnected(metrics *Metrics, serviceName string) {
	metrics.Business.WebSocketConnectionsActive.WithLabelValues(
		serviceName,
	).Inc()
}

// RecordWebSocketDisconnected records a WebSocket disconnection.
func RecordWebSocketDisconnected(metrics *Metrics, serviceName string) {
	metrics.Business.WebSocketConnectionsActive.WithLabelValues(
		serviceName,
	).Dec()
}

// RecordWebSocketMessage records a WebSocket message sent.
func RecordWebSocketMessage(metrics *Metrics, serviceName string) {
	metrics.Business.WebSocketMessagesTotal.WithLabelValues(
		serviceName,
	).Inc()
}

// RecordWebSocketBroadcast records a WebSocket broadcast to multiple clients.
func RecordWebSocketBroadcast(metrics *Metrics, serviceName string, clientCount int) {
	metrics.Business.WebSocketMessagesTotal.WithLabelValues(
		serviceName,
	).Add(float64(clientCount))
}

// JobMetricsRecorder provides a convenient interface for recording job-related metrics.
type JobMetricsRecorder struct {
	metrics     *Metrics
	serviceName string
	orgID       string
	startTime   time.Time
}

// NewJobMetricsRecorder creates a new job metrics recorder.
func NewJobMetricsRecorder(metrics *Metrics, serviceName, orgID string) *JobMetricsRecorder {
	RecordJobCreated(metrics, serviceName, orgID)

	return &JobMetricsRecorder{
		metrics:     metrics,
		serviceName: serviceName,
		orgID:       orgID,
		startTime:   time.Now(),
	}
}

// Complete records successful job completion.
func (r *JobMetricsRecorder) Complete() {
	duration := time.Since(r.startTime)
	RecordJobCompleted(r.metrics, r.serviceName, r.orgID, duration)
}

// Fail records job failure with an error code.
func (r *JobMetricsRecorder) Fail(errorCode string) {
	RecordJobFailed(r.metrics, r.serviceName, r.orgID, errorCode)
}

// AuthMetricsRecorder provides a convenient interface for recording authentication metrics.
type AuthMetricsRecorder struct {
	metrics     *Metrics
	serviceName string
	authMethod  string
	success     bool
}

// NewAuthMetricsRecorder creates a new authentication metrics recorder.
func NewAuthMetricsRecorder(metrics *Metrics, serviceName, authMethod string) *AuthMetricsRecorder {
	return &AuthMetricsRecorder{
		metrics:     metrics,
		serviceName: serviceName,
		authMethod:  authMethod,
	}
}

// Success records successful authentication.
func (r *AuthMetricsRecorder) Success(role string) {
	RecordAuthSuccess(r.metrics, r.serviceName, r.authMethod, role)
	r.success = true
}

// Failure records failed authentication.
func (r *AuthMetricsRecorder) Failure() {
	RecordAuthFailure(r.metrics, r.serviceName, r.authMethod)
}

// StorageMetricsRecorder provides a convenient interface for recording storage metrics.
type StorageMetricsRecorder struct {
	metrics     *Metrics
	serviceName string
	backend     string
}

// NewStorageMetricsRecorder creates a new storage metrics recorder.
func NewStorageMetricsRecorder(metrics *Metrics, serviceName, backend string) *StorageMetricsRecorder {
	return &StorageMetricsRecorder{
		metrics:     metrics,
		serviceName: serviceName,
		backend:     backend,
	}
}

// Store records a document storage operation.
func (r *StorageMetricsRecorder) Store(docType string, size int64) {
	RecordDocumentStored(r.metrics, r.serviceName, r.backend, docType, size)
}

// Retrieve records a document retrieval operation.
func (r *StorageMetricsRecorder) Retrieve() {
	RecordDocumentRetrieved(r.metrics, r.serviceName, r.backend)
}

// Delete records a document deletion operation.
func (r *StorageMetricsRecorder) Delete(size int64) {
	RecordDocumentDeleted(r.metrics, r.serviceName, r.backend, size)
}

// PrinterMetricsRecorder provides a convenient interface for recording printer metrics.
type PrinterMetricsRecorder struct {
	metrics     *Metrics
	serviceName string
	orgID       string
}

// NewPrinterMetricsRecorder creates a new printer metrics recorder.
func NewPrinterMetricsRecorder(metrics *Metrics, serviceName, orgID string) *PrinterMetricsRecorder {
	return &PrinterMetricsRecorder{
		metrics:     metrics,
		serviceName: serviceName,
		orgID:       orgID,
	}
}

// Register records a printer registration.
func (r *PrinterMetricsRecorder) Register() {
	RecordPrinterRegistered(r.metrics, r.serviceName, r.orgID)
}

// Heartbeat records a printer heartbeat.
func (r *PrinterMetricsRecorder) Heartbeat() {
	RecordPrinterHeartbeat(r.metrics, r.serviceName, r.orgID)
}

// WebSocketMetricsRecorder provides a convenient interface for recording WebSocket metrics.
type WebSocketMetricsRecorder struct {
	metrics     *Metrics
	serviceName string
}

// NewWebSocketMetricsRecorder creates a new WebSocket metrics recorder.
func NewWebSocketMetricsRecorder(metrics *Metrics, serviceName string) *WebSocketMetricsRecorder {
	return &WebSocketMetricsRecorder{
		metrics:     metrics,
		serviceName: serviceName,
	}
}

// Connected records a new connection.
func (r *WebSocketMetricsRecorder) Connected() {
	RecordWebSocketConnected(r.metrics, r.serviceName)
}

// Disconnected records a disconnection.
func (r *WebSocketMetricsRecorder) Disconnected() {
	RecordWebSocketDisconnected(r.metrics, r.serviceName)
}

// Message records a message sent.
func (r *WebSocketMetricsRecorder) Message() {
	RecordWebSocketMessage(r.metrics, r.serviceName)
}

// Broadcast records a broadcast to multiple clients.
func (r *WebSocketMetricsRecorder) Broadcast(clientCount int) {
	RecordWebSocketBroadcast(r.metrics, r.serviceName, clientCount)
}
