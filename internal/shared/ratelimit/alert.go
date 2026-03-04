package ratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// AlertManager handles violation alerts and notifications.
type AlertManager struct {
	redis         *RedisClient
	alertChannels []AlertChannel
	mu            sync.RWMutex

	// Alert aggregation settings
	aggregateWindow time.Duration
	aggregateMax    int
}

// AlertChannel defines an interface for sending alerts.
type AlertChannel interface {
	Send(ctx context.Context, alert *Alert) error
}

// Alert represents a rate limit violation alert.
type Alert struct {
	ID             string                 `json:"id"`
	Type           string                 `json:"type"`     // "violation", "circuit_open", "quota_exceeded"
	Severity       string                 `json:"severity"` // "low", "medium", "high", "critical"
	PolicyID       string                 `json:"policy_id"`
	PolicyName     string                 `json:"policy_name"`
	Identifier     string                 `json:"identifier"`
	IdentifierType string                 `json:"identifier_type"`
	Path           string                 `json:"path"`
	Method         string                 `json:"method"`
	Current        int64                  `json:"current"`
	Limit          int64                  `json:"limit"`
	Message        string                 `json:"message"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
}

// NewAlertManager creates a new alert manager.
func NewAlertManager(redis *RedisClient) *AlertManager {
	am := &AlertManager{
		redis:           redis,
		alertChannels:   make([]AlertChannel, 0),
		aggregateWindow: time.Minute,
		aggregateMax:    100,
	}

	return am
}

// SendViolationAlert sends an alert for a rate limit violation.
func (am *AlertManager) SendViolationAlert(ctx context.Context, req *Request, policy *Policy, current, limit int64) error {
	if am == nil {
		return nil
	}

	alert := &Alert{
		ID:             generateID(),
		Type:           "violation",
		Severity:       policy.Severity,
		PolicyID:       policy.ID,
		PolicyName:     policy.Name,
		Identifier:     req.Identifier,
		IdentifierType: req.Type,
		Path:           req.Path,
		Method:         req.Method,
		Current:        current,
		Limit:          limit,
		Message:        fmt.Sprintf("Rate limit exceeded: %d/%d requests", current, limit),
		Metadata: map[string]interface{}{
			"user_agent": req.Identifier,
			"timestamp":  req.Timestamp.Format(time.RFC3339),
		},
		CreatedAt: time.Now(),
	}

	return am.sendAlert(ctx, alert)
}

// sendAlert sends an alert through all registered channels.
func (am *AlertManager) sendAlert(ctx context.Context, alert *Alert) error {
	am.mu.RLock()
	channels := make([]AlertChannel, len(am.alertChannels))
	copy(channels, am.alertChannels)
	am.mu.RUnlock()

	// Store alert in Redis for history
	if am.redis != nil {
		key := fmt.Sprintf("ratelimit:alert:%s", alert.ID)
		data, _ := json.Marshal(alert)
		_ = am.redis.SetJSON(ctx, key, data, 24*time.Hour)
	}

	// Send to all channels
	var lastErr error
	for _, ch := range channels {
		if err := ch.Send(ctx, alert); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// AddChannel adds an alert channel.
func (am *AlertManager) AddChannel(channel AlertChannel) {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.alertChannels = append(am.alertChannels, channel)
}

// RemoveChannel removes an alert channel.
func (am *AlertManager) RemoveChannel(channel AlertChannel) {
	am.mu.Lock()
	defer am.mu.Unlock()

	for i, ch := range am.alertChannels {
		if ch == channel {
			am.alertChannels = append(am.alertChannels[:i], am.alertChannels[i+1:]...)
			break
		}
	}
}

// SendCircuitBreakerAlert sends an alert when a circuit breaker opens.
func (am *AlertManager) SendCircuitBreakerAlert(ctx context.Context, path string, failureCount int) {
	alert := &Alert{
		ID:       generateID(),
		Type:     "circuit_open",
		Severity: "high",
		Path:     path,
		Message:  fmt.Sprintf("Circuit breaker opened for %s after %d failures", path, failureCount),
		Metadata: map[string]interface{}{
			"failure_count": failureCount,
		},
		CreatedAt: time.Now(),
	}

	_ = am.sendAlert(ctx, alert)
}

// SendQuotaExceededAlert sends an alert when a quota is exceeded.
func (am *AlertManager) SendQuotaExceededAlert(ctx context.Context, entityID, entityType, quotaType string, used, limit int) {
	alert := &Alert{
		ID:             generateID(),
		Type:           "quota_exceeded",
		Severity:       "medium",
		Identifier:     entityID,
		IdentifierType: entityType,
		Message:        fmt.Sprintf("Quota exceeded for %s: %d/%d %s", entityID, used, limit, quotaType),
		Metadata: map[string]interface{}{
			"quota_type": quotaType,
			"used":       used,
			"limit":      limit,
		},
		CreatedAt: time.Now(),
	}

	_ = am.sendAlert(ctx, alert)
}

// GetAlertHistory retrieves recent alerts.
func (am *AlertManager) GetAlertHistory(ctx context.Context, limit int) ([]*Alert, error) {
	if am.redis == nil {
		return []*Alert{}, nil
	}

	pattern := "ratelimit:alert:*"
	keys, err := am.redis.ListKeys(ctx, pattern)
	if err != nil {
		return nil, err
	}

	alerts := make([]*Alert, 0, limit)

	for _, key := range keys {
		var alert Alert
		if err := am.redis.GetJSON(ctx, key, &alert); err != nil {
			continue
		}
		alerts = append(alerts, &alert)

		if len(alerts) >= limit {
			break
		}
	}

	return alerts, nil
}

// WebhookChannel sends alerts via webhook.
type WebhookChannel struct {
	URL     string
	Headers map[string]string
}

// NewWebhookChannel creates a new webhook alert channel.
func NewWebhookChannel(url string) *WebhookChannel {
	return &WebhookChannel{
		URL:     url,
		Headers: make(map[string]string),
	}
}

// Send sends an alert via webhook.
func (wc *WebhookChannel) Send(ctx context.Context, alert *Alert) error {
	// In a real implementation, this would make an HTTP POST request
	// For now, just log the alert
	return nil
}

// LogChannel sends alerts to the log.
type LogChannel struct {
	Logger interface {
		Printf(format string, args ...interface{})
	}
}

// NewLogChannel creates a new log alert channel.
func NewLogChannel(logger interface {
	Printf(format string, args ...interface{})
}) *LogChannel {
	return &LogChannel{Logger: logger}
}

// Send logs an alert.
func (lc *LogChannel) Send(ctx context.Context, alert *Alert) error {
	if lc.Logger == nil {
		return nil
	}

	lc.Logger.Printf("[ALERT] %s: %s (severity: %s)", alert.Type, alert.Message, alert.Severity)
	return nil
}

// AggregatingChannel aggregates alerts before sending.
type AggregatingChannel struct {
	upstream      AlertChannel
	window        time.Duration
	maxAlerts     int
	mu            sync.Mutex
	pendingAlerts []*Alert
	lastFlush     time.Time
}

// NewAggregatingChannel creates an aggregating alert channel.
func NewAggregatingChannel(upstream AlertChannel, window time.Duration, maxAlerts int) *AggregatingChannel {
	ac := &AggregatingChannel{
		upstream:      upstream,
		window:        window,
		maxAlerts:     maxAlerts,
		pendingAlerts: make([]*Alert, 0),
		lastFlush:     time.Now(),
	}

	// Start flush goroutine
	go ac.flushLoop()

	return ac
}

// Send adds an alert to the aggregation buffer.
func (ac *AggregatingChannel) Send(ctx context.Context, alert *Alert) error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	ac.pendingAlerts = append(ac.pendingAlerts, alert)

	// Flush if we've reached max alerts
	if len(ac.pendingAlerts) >= ac.maxAlerts {
		return ac.flush(ctx)
	}

	return nil
}

// flushLoop periodically flushes pending alerts.
func (ac *AggregatingChannel) flushLoop() {
	ticker := time.NewTicker(ac.window)
	defer ticker.Stop()

	for range ticker.C {
		ac.mu.Lock()
		_ = ac.flush(context.Background())
		ac.mu.Unlock()
	}
}

// flush sends pending alerts to the upstream channel.
func (ac *AggregatingChannel) flush(ctx context.Context) error {
	if len(ac.pendingAlerts) == 0 {
		ac.lastFlush = time.Now()
		return nil
	}

	// Create aggregated alert
	aggregated := &Alert{
		ID:       generateID(),
		Type:     "aggregated",
		Severity: "medium",
		Message:  fmt.Sprintf("%d aggregated alerts", len(ac.pendingAlerts)),
		Metadata: map[string]interface{}{
			"count":  len(ac.pendingAlerts),
			"alerts": ac.pendingAlerts,
		},
		CreatedAt: time.Now(),
	}

	// Clear pending
	ac.pendingAlerts = make([]*Alert, 0)
	ac.lastFlush = time.Now()

	return ac.upstream.Send(ctx, aggregated)
}

// FilteredChannel filters alerts based on criteria.
type FilteredChannel struct {
	upstream  AlertChannel
	severity  string // Minimum severity to send
	policyIDs map[string]bool
}

// NewFilteredChannel creates a filtered alert channel.
func NewFilteredChannel(upstream AlertChannel, minSeverity string) *FilteredChannel {
	return &FilteredChannel{
		upstream:  upstream,
		severity:  minSeverity,
		policyIDs: make(map[string]bool),
	}
}

// Send sends an alert if it passes the filter.
func (fc *FilteredChannel) Send(ctx context.Context, alert *Alert) error {
	// Check severity
	if !fc.severityMatches(alert.Severity) {
		return nil
	}

	// Check policy ID filter
	if len(fc.policyIDs) > 0 && !fc.policyIDs[alert.PolicyID] {
		return nil
	}

	return fc.upstream.Send(ctx, alert)
}

// severityMatches checks if the alert severity matches the filter.
func (fc *FilteredChannel) severityMatches(severity string) bool {
	severityOrder := map[string]int{
		"low":      1,
		"medium":   2,
		"high":     3,
		"critical": 4,
	}

	filterLevel := severityOrder[fc.severity]
	alertLevel := severityOrder[severity]

	return alertLevel >= filterLevel
}

// AddPolicyID adds a policy ID to the filter whitelist.
func (fc *FilteredChannel) AddPolicyID(policyID string) {
	fc.policyIDs[policyID] = true
}

// RemovePolicyID removes a policy ID from the filter.
func (fc *FilteredChannel) RemovePolicyID(policyID string) {
	delete(fc.policyIDs, policyID)
}

// AlertStats represents statistics about alerts.
type AlertStats struct {
	TotalAlerts  int64            `json:"total_alerts"`
	ByType       map[string]int64 `json:"by_type"`
	BySeverity   map[string]int64 `json:"by_severity"`
	ByPolicy     map[string]int64 `json:"by_policy"`
	RecentAlerts []*Alert         `json:"recent_alerts"`
}

// GetStats returns alert statistics.
func (am *AlertManager) GetStats(ctx context.Context) (*AlertStats, error) {
	alerts, err := am.GetAlertHistory(ctx, 100)
	if err != nil {
		return nil, err
	}

	stats := &AlertStats{
		TotalAlerts:  int64(len(alerts)),
		ByType:       make(map[string]int64),
		BySeverity:   make(map[string]int64),
		ByPolicy:     make(map[string]int64),
		RecentAlerts: alerts,
	}

	for _, alert := range alerts {
		stats.ByType[alert.Type]++
		stats.BySeverity[alert.Severity]++
		stats.ByPolicy[alert.PolicyID]++
	}

	return stats, nil
}

// ClearHistory clears alert history from Redis.
func (am *AlertManager) ClearHistory(ctx context.Context) error {
	if am.redis == nil {
		return nil
	}

	pattern := "ratelimit:alert:*"
	keys, err := am.redis.ListKeys(ctx, pattern)
	if err != nil {
		return err
	}

	for _, key := range keys {
		_ = am.redis.DeleteKey(ctx, key)
	}

	return nil
}
