package ratelimit

import (
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Metrics handles OpenTelemetry metrics for rate limiting.
type Metrics struct {
	mu sync.RWMutex

	// OpenTelemetry instruments
	requestsAllowed metric.Int64Counter
	requestsDenied  metric.Int64Counter
	violations      metric.Int64Counter
	currentUsage    metric.Int64Gauge
	queueSize       metric.Int64Gauge
	circuitState    metric.Int64Gauge

	// Common attributes
	attrPolicy   attribute.Key
	attrType     attribute.Key
	attrPath     attribute.Key
	attrSeverity attribute.Key
	attrState    attribute.Key
	attrBypassed attribute.Key
}

// NewMetrics creates a new metrics instance.
func NewMetrics() *Metrics {
	meter := otel.Meter("github.com/openprint/openprint/ratelimit")

	m := &Metrics{
		attrPolicy:   attribute.Key("policy"),
		attrType:     attribute.Key("type"),
		attrPath:     attribute.Key("path"),
		attrSeverity: attribute.Key("severity"),
		attrState:    attribute.Key("state"),
		attrBypassed: attribute.Key("bypassed"),
	}

	// Initialize instruments
	m.requestsAllowed, _ = meter.Int64Counter(
		"ratelimit_requests_allowed",
		metric.WithDescription("Number of requests allowed by rate limiter"),
		metric.WithUnit("1"),
	)

	m.requestsDenied, _ = meter.Int64Counter(
		"ratelimit_requests_denied",
		metric.WithDescription("Number of requests denied by rate limiter"),
		metric.WithUnit("1"),
	)

	m.violations, _ = meter.Int64Counter(
		"ratelimit_violations",
		metric.WithDescription("Number of rate limit violations"),
		metric.WithUnit("1"),
	)

	m.currentUsage, _ = meter.Int64Gauge(
		"ratelimit_current_usage",
		metric.WithDescription("Current rate limit usage"),
		metric.WithUnit("1"),
	)

	m.queueSize, _ = meter.Int64Gauge(
		"ratelimit_queue_size",
		metric.WithDescription("Current size of rate limit queue"),
		metric.WithUnit("1"),
	)

	m.circuitState, _ = meter.Int64Gauge(
		"ratelimit_circuit_state",
		metric.WithDescription("Circuit breaker state (0=closed, 1=half-open, 2=open)"),
		metric.WithUnit("1"),
	)

	return m
}

// RecordAllowed records an allowed request.
func (m *Metrics) RecordAllowed(requestType, path, policy string) {
	if m == nil || m.requestsAllowed == nil {
		return
	}

	opts := metric.WithAttributes(
		m.attrType.String(requestType),
		m.attrPath.String(path),
		m.attrPolicy.String(policy),
	)

	m.requestsAllowed.Add(nil, 1, opts)
}

// RecordDenied records a denied request.
func (m *Metrics) RecordDenied(requestType, path, policy string) {
	if m == nil || m.requestsDenied == nil {
		return
	}

	opts := metric.WithAttributes(
		m.attrType.String(requestType),
		m.attrPath.String(path),
		m.attrPolicy.String(policy),
	)

	m.requestsDenied.Add(nil, 1, opts)
}

// RecordViolation records a rate limit violation.
func (m *Metrics) RecordViolation(requestType, path, policy, severity string) {
	if m == nil || m.violations == nil {
		return
	}

	opts := metric.WithAttributes(
		m.attrType.String(requestType),
		m.attrPath.String(path),
		m.attrPolicy.String(policy),
		m.attrSeverity.String(severity),
	)

	m.violations.Add(nil, 1, opts)
}

// RecordUsage records the current usage for a policy.
func (m *Metrics) RecordUsage(requestType, path, policy string, current, limit int64) {
	if m == nil || m.currentUsage == nil {
		return
	}

	opts := metric.WithAttributes(
		m.attrType.String(requestType),
		m.attrPath.String(path),
		m.attrPolicy.String(policy),
	)

	m.currentUsage.Record(nil, current, opts)
}

// RecordQueueSize records the current queue size.
func (m *Metrics) RecordQueueSize(path string, size int) {
	if m == nil || m.queueSize == nil {
		return
	}

	opts := metric.WithAttributes(
		m.attrPath.String(path),
	)

	m.queueSize.Record(nil, int64(size), opts)
}

// RecordCircuitState records the circuit breaker state.
func (m *Metrics) RecordCircuitState(path string, state string) {
	if m == nil || m.circuitState == nil {
		return
	}

	// Convert state to numeric value
	stateValue := int64(0)
	switch state {
	case "half_open":
		stateValue = 1
	case "open":
		stateValue = 2
	}

	opts := metric.WithAttributes(
		m.attrPath.String(path),
		m.attrState.String(state),
	)

	m.circuitState.Record(nil, stateValue, opts)
}

// RecordBypass records a bypassed request.
func (m *Metrics) RecordBypass(requestType, path string) {
	if m == nil || m.requestsAllowed == nil {
		return
	}

	opts := metric.WithAttributes(
		m.attrType.String(requestType),
		m.attrPath.String(path),
		m.attrBypassed.Bool(true),
	)

	m.requestsAllowed.Add(nil, 1, opts)
}

// RecordThrottle records a throttled request.
func (m *Metrics) RecordThrottle(requestType, path string, throttleRate float64) {
	// Throttled requests are counted as allowed but with special attributes
	if m == nil || m.requestsAllowed == nil {
		return
	}

	opts := metric.WithAttributes(
		m.attrType.String(requestType),
		m.attrPath.String(path),
	)

	m.requestsAllowed.Add(nil, 1, opts)
}

// MetricsSnapshot represents a snapshot of metrics at a point in time.
type MetricsSnapshot struct {
	Timestamp       time.Time `json:"timestamp"`
	TotalAllowed    int64     `json:"total_allowed"`
	TotalDenied     int64     `json:"total_denied"`
	TotalViolations int64     `json:"total_violations"`
	AvgUsage        float64   `json:"avg_usage"`
	PeakUsage       int64     `json:"peak_usage"`
	DenyRate        float64   `json:"deny_rate"`
	ViolationRate   float64   `json:"violation_rate"`
}

// InMemoryMetrics provides an in-memory metrics collector for testing.
type InMemoryMetrics struct {
	mu             sync.RWMutex
	allowedCount   map[string]int64
	deniedCount    map[string]int64
	violationCount map[string]int64
	usageHistory   map[string][]int64
	snapshots      []MetricsSnapshot
	maxHistorySize int
}

// NewInMemoryMetrics creates an in-memory metrics collector.
func NewInMemoryMetrics() *InMemoryMetrics {
	return &InMemoryMetrics{
		allowedCount:   make(map[string]int64),
		deniedCount:    make(map[string]int64),
		violationCount: make(map[string]int64),
		usageHistory:   make(map[string][]int64),
		snapshots:      make([]MetricsSnapshot, 0),
		maxHistorySize: 1000,
	}
}

// RecordAllowed records an allowed request.
func (m *InMemoryMetrics) RecordAllowed(requestType, path, policy string) {
	key := policy + ":" + path
	m.mu.Lock()
	defer m.mu.Unlock()
	m.allowedCount[key]++
}

// RecordDenied records a denied request.
func (m *InMemoryMetrics) RecordDenied(requestType, path, policy string) {
	key := policy + ":" + path
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deniedCount[key]++
}

// RecordViolation records a rate limit violation.
func (m *InMemoryMetrics) RecordViolation(requestType, path, policy, severity string) {
	key := policy + ":" + path
	m.mu.Lock()
	defer m.mu.Unlock()
	m.violationCount[key]++
}

// RecordUsage records the current usage.
func (m *InMemoryMetrics) RecordUsage(requestType, path, policy string, current, limit int64) {
	key := policy + ":" + path
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.usageHistory[key] == nil {
		m.usageHistory[key] = make([]int64, 0, m.maxHistorySize)
	}

	m.usageHistory[key] = append(m.usageHistory[key], current)
	if len(m.usageHistory[key]) > m.maxHistorySize {
		m.usageHistory[key] = m.usageHistory[key][1:]
	}
}

// GetSnapshot returns a metrics snapshot.
func (m *InMemoryMetrics) GetSnapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalAllowed := int64(0)
	totalDenied := int64(0)
	totalViolations := int64(0)

	for _, count := range m.allowedCount {
		totalAllowed += count
	}
	for _, count := range m.deniedCount {
		totalDenied += count
	}
	for _, count := range m.violationCount {
		totalViolations += count
	}

	totalRequests := totalAllowed + totalDenied
	denyRate := 0.0
	if totalRequests > 0 {
		denyRate = float64(totalDenied) / float64(totalRequests)
	}

	violationRate := 0.0
	if totalRequests > 0 {
		violationRate = float64(totalViolations) / float64(totalRequests)
	}

	peakUsage := int64(0)
	sumUsage := int64(0)
	usageCount := 0

	for _, history := range m.usageHistory {
		for _, usage := range history {
			if usage > peakUsage {
				peakUsage = usage
			}
			sumUsage += usage
			usageCount++
		}
	}

	avgUsage := 0.0
	if usageCount > 0 {
		avgUsage = float64(sumUsage) / float64(usageCount)
	}

	return MetricsSnapshot{
		Timestamp:       time.Now(),
		TotalAllowed:    totalAllowed,
		TotalDenied:     totalDenied,
		TotalViolations: totalViolations,
		AvgUsage:        avgUsage,
		PeakUsage:       peakUsage,
		DenyRate:        denyRate,
		ViolationRate:   violationRate,
	}
}

// Reset resets all metrics.
func (m *InMemoryMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.allowedCount = make(map[string]int64)
	m.deniedCount = make(map[string]int64)
	m.violationCount = make(map[string]int64)
	m.usageHistory = make(map[string][]int64)
	m.snapshots = make([]MetricsSnapshot, 0)
}

// GetPolicyStats returns statistics for a specific policy.
func (m *InMemoryMetrics) GetPolicyStats(policy string) (allowed, denied, violations int64) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	prefix := policy + ":"

	for key, count := range m.allowedCount {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			allowed += count
		}
	}

	for key, count := range m.deniedCount {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			denied += count
		}
	}

	for key, count := range m.violationCount {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			violations += count
		}
	}

	return
}

// TakeSnapshot records a snapshot of current metrics.
func (m *InMemoryMetrics) TakeSnapshot() {
	m.mu.Lock()
	defer m.mu.Unlock()

	snapshot := m.GetSnapshot()
	m.snapshots = append(m.snapshots, snapshot)

	if len(m.snapshots) > m.maxHistorySize {
		m.snapshots = m.snapshots[1:]
	}
}

// GetSnapshots returns all recorded snapshots.
func (m *InMemoryMetrics) GetSnapshots() []MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshots := make([]MetricsSnapshot, len(m.snapshots))
	copy(snapshots, m.snapshots)

	return snapshots
}
