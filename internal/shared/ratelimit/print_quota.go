// Package ratelimit provides print-specific quota management.
package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// PrintQuotaManager manages print-specific quotas extending the base quota system.
type PrintQuotaManager struct {
	redis *RedisClient
	mu    sync.RWMutex

	// In-memory quota cache
	quotaCache map[string]*PrintQuotaState
}

// PrintQuotaState tracks the state of a print quota.
type PrintQuotaState struct {
	mu             sync.RWMutex
	EntityID       string
	EntityType     string // "user" or "organization"
	QuotaType      string // "pages", "jobs", "color_pages", "duplex_pages"
	Period         string // "daily", "weekly", "monthly", "quarterly", "yearly"
	Limit          int
	Used           int
	Remaining      int
	PeriodStart    time.Time
	PeriodEnd      time.Time
	LastReset      time.Time
	History        []QuotaEvent
	OverageAllowed bool
	OverageLimit   int
}

// QuotaEvent represents a quota usage event.
type QuotaEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"` // "granted", "used", "reset", "adjusted"
	Amount    int       `json:"amount"`
	Previous  int       `json:"previous"`
	Remaining int       `json:"remaining"`
	Reason    string    `json:"reason,omitempty"`
	JobID     string    `json:"job_id,omitempty"`
}

// PrintJobRequest represents a print job for quota checking.
type PrintJobRequest struct {
	JobID         string
	UserID        string
	OrgID         string
	PageCount     int
	ColorPages    int
	DuplexPages   int
	Priority      int
	IsOverridable bool
}

// QuotaCheckResult represents the result of a quota check.
type QuotaCheckResult struct {
	Allowed     bool      `json:"allowed"`
	QuotaType   string    `json:"quota_type"`
	Used        int       `json:"used"`
	Limit       int       `json:"limit"`
	Remaining   int       `json:"remaining"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	OverageUsed int       `json:"overage_used,omitempty"`
	Reason      string    `json:"reason,omitempty"`
	RetryAfter  time.Time `json:"retry_after,omitempty"`
}

// NewPrintQuotaManager creates a new print quota manager.
func NewPrintQuotaManager(redis *RedisClient) *PrintQuotaManager {
	pqm := &PrintQuotaManager{
		redis:      redis,
		quotaCache: make(map[string]*PrintQuotaState),
	}

	// Start periodic cleanup
	go pqm.cleanup()

	return pqm
}

// CheckQuota checks if a print job is allowed based on quotas.
func (pqm *PrintQuotaManager) CheckQuota(ctx context.Context, req *PrintJobRequest) (*QuotaCheckResult, error) {
	// Check user-level quota first
	userResult, err := pqm.checkEntityQuota(ctx, req.UserID, "user", req)
	if err != nil {
		return nil, err
	}

	if !userResult.Allowed && !req.IsOverridable {
		return userResult, nil
	}

	// Check organization-level quota
	orgResult, err := pqm.checkEntityQuota(ctx, req.OrgID, "organization", req)
	if err != nil {
		return nil, err
	}

	// Return the most restrictive result
	if orgResult.Remaining < userResult.Remaining {
		return orgResult, nil
	}

	return userResult, nil
}

// checkEntityQuota checks quota for a specific entity (user or org).
func (pqm *PrintQuotaManager) checkEntityQuota(ctx context.Context, entityID, entityType string, req *PrintJobRequest) (*QuotaCheckResult, error) {
	// Get quota state
	state, err := pqm.getQuotaState(ctx, entityID, entityType, "pages", "monthly")
	if err != nil {
		return nil, err
	}

	// Check if period needs reset
	pqm.checkAndResetPeriod(state)

	// Check if job would exceed quota
	remaining := state.Remaining
	overageUsed := 0

	if req.PageCount > remaining {
		if state.OverageAllowed && (req.PageCount-remaining) <= state.OverageLimit {
			overageUsed = req.PageCount - remaining
			remaining = 0
		} else {
			return &QuotaCheckResult{
				Allowed:     false,
				QuotaType:   state.QuotaType,
				Used:        state.Used,
				Limit:       state.Limit,
				Remaining:   state.Remaining,
				PeriodStart: state.PeriodStart,
				PeriodEnd:   state.PeriodEnd,
				Reason:      "Quota exceeded",
				RetryAfter:  state.PeriodEnd,
			}, nil
		}
	}

	// Job is allowed
	return &QuotaCheckResult{
		Allowed:     true,
		QuotaType:   state.QuotaType,
		Used:        state.Used,
		Limit:       state.Limit,
		Remaining:   remaining,
		PeriodStart: state.PeriodStart,
		PeriodEnd:   state.PeriodEnd,
		OverageUsed: overageUsed,
	}, nil
}

// UseQuota records quota usage for a completed print job.
func (pqm *PrintQuotaManager) UseQuota(ctx context.Context, req *PrintJobRequest, actualPages int) error {
	// Update user quota
	if err := pqm.updateQuota(ctx, req.UserID, "user", req.JobID, actualPages); err != nil {
		return err
	}

	// Update organization quota
	if err := pqm.updateQuota(ctx, req.OrgID, "organization", req.JobID, actualPages); err != nil {
		return err
	}

	return nil
}

// updateQuota updates quota for a specific entity.
func (pqm *PrintQuotaManager) updateQuota(ctx context.Context, entityID, entityType, jobID string, pages int) error {
	state, err := pqm.getQuotaState(ctx, entityID, entityType, "pages", "monthly")
	if err != nil {
		return err
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	pqm.checkAndResetPeriod(state)

	// Record event
	event := QuotaEvent{
		Timestamp: time.Now(),
		Action:    "used",
		Amount:    pages,
		Previous:  state.Used,
		Remaining: state.Remaining - pages,
		Reason:    "Print job completed",
		JobID:     jobID,
	}

	state.Used += pages
	state.Remaining -= pages

	if state.Remaining < 0 {
		state.Remaining = 0
	}

	// Add to history
	state.History = append(state.History, event)
	if len(state.History) > 1000 {
		state.History = state.History[len(state.History)-1000:]
	}

	// Update cache
	cacheKey := pqm.getCacheKey(entityID, entityType, "pages", "monthly")
	pqm.mu.Lock()
	pqm.quotaCache[cacheKey] = state
	pqm.mu.Unlock()

	return nil
}

// getQuotaState retrieves or creates quota state for an entity.
func (pqm *PrintQuotaManager) getQuotaState(ctx context.Context, entityID, entityType, quotaType, period string) (*PrintQuotaState, error) {
	cacheKey := pqm.getCacheKey(entityID, entityType, quotaType, period)

	// Check cache first
	pqm.mu.RLock()
	if state, ok := pqm.quotaCache[cacheKey]; ok {
		pqm.mu.RUnlock()
		return state, nil
	}
	pqm.mu.RUnlock()

	// Create new state
	now := time.Now()
	periodStart, periodEnd := pqm.calculatePeriod(period, now)

	state := &PrintQuotaState{
		EntityID:    entityID,
		EntityType:  entityType,
		QuotaType:   quotaType,
		Period:      period,
		Limit:       1000, // Default limit
		Used:        0,
		Remaining:   1000,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		LastReset:   now,
		History:     make([]QuotaEvent, 0),
	}

	// Cache the state
	pqm.mu.Lock()
	pqm.quotaCache[cacheKey] = state
	pqm.mu.Unlock()

	return state, nil
}

// getCacheKey generates a cache key for quota state.
func (pqm *PrintQuotaManager) getCacheKey(entityID, entityType, quotaType, period string) string {
	return fmt.Sprintf("quota:%s:%s:%s:%s", entityType, entityID, quotaType, period)
}

// calculatePeriod calculates the start and end of a quota period.
func (pqm *PrintQuotaManager) calculatePeriod(period string, now time.Time) (time.Time, time.Time) {
	year, month, _ := now.Date()

	switch period {
	case "daily":
		start := time.Date(year, month, now.Day(), 0, 0, 0, 0, now.Location())
		end := start.Add(24 * time.Hour)
		return start, end

	case "weekly":
		// Start of week (Monday)
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := time.Date(year, month, now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		end := start.Add(7 * 24 * time.Hour)
		return start, end

	case "monthly":
		start := time.Date(year, month, 1, 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 1, 0)
		return start, end

	case "quarterly":
		quarter := (int(month) - 1) / 3
		start := time.Date(year, time.Month(quarter*3+1), 1, 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 3, 0)
		return start, end

	case "yearly":
		start := time.Date(year, 1, 1, 0, 0, 0, 0, now.Location())
		end := start.AddDate(1, 0, 0)
		return start, end

	default:
		// Default to monthly
		start := time.Date(year, month, 1, 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 1, 0)
		return start, end
	}
}

// checkAndResetPeriod checks if the quota period has expired and resets if needed.
func (pqm *PrintQuotaManager) checkAndResetPeriod(state *PrintQuotaState) {
	now := time.Now()

	if now.After(state.PeriodEnd) {
		// Calculate new period
		periodStart, periodEnd := pqm.calculatePeriod(state.Period, now)

		// Reset state
		state.Used = 0
		state.Remaining = state.Limit
		state.PeriodStart = periodStart
		state.PeriodEnd = periodEnd
		state.LastReset = now

		// Add reset event
		event := QuotaEvent{
			Timestamp: now,
			Action:    "reset",
			Amount:    0,
			Previous:  state.Used,
			Remaining: state.Remaining,
			Reason:    "Period reset",
		}
		state.History = append(state.History, event)
	}
}

// SetQuota sets a custom quota limit for an entity.
func (pqm *PrintQuotaManager) SetQuota(ctx context.Context, entityID, entityType, quotaType, period string, limit int) error {
	state, err := pqm.getQuotaState(ctx, entityID, entityType, quotaType, period)
	if err != nil {
		return err
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	oldLimit := state.Limit
	state.Limit = limit
	state.Remaining = limit - state.Used

	// Add adjustment event
	event := QuotaEvent{
		Timestamp: time.Now(),
		Action:    "adjusted",
		Amount:    limit - oldLimit,
		Previous:  oldLimit,
		Remaining: state.Remaining,
		Reason:    "Quota limit adjusted",
	}
	state.History = append(state.History, event)

	return nil
}

// GetQuotaStatus returns the current quota status for an entity.
func (pqm *PrintQuotaManager) GetQuotaStatus(ctx context.Context, entityID, entityType string) (*QuotaStatus, error) {
	state, err := pqm.getQuotaState(ctx, entityID, entityType, "pages", "monthly")
	if err != nil {
		return nil, err
	}

	return &QuotaStatus{
		EntityID:    entityID,
		EntityType:  entityType,
		Limit:       state.Limit,
		Used:        state.Used,
		Remaining:   state.Remaining,
		PeriodStart: state.PeriodStart,
		PeriodEnd:   state.PeriodEnd,
	}, nil
}

// QuotaStatus represents the current status of a quota.
type QuotaStatus struct {
	EntityID    string    `json:"entity_id"`
	EntityType  string    `json:"entity_type"`
	Limit       int       `json:"limit"`
	Used        int       `json:"used"`
	Remaining   int       `json:"remaining"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
}

// GetQuotaHistory returns quota usage history.
func (pqm *PrintQuotaManager) GetQuotaHistory(ctx context.Context, entityID, entityType string, limit int) ([]QuotaEvent, error) {
	state, err := pqm.getQuotaState(ctx, entityID, entityType, "pages", "monthly")
	if err != nil {
		return nil, err
	}

	state.mu.RLock()
	defer state.mu.RUnlock()

	history := state.History
	if limit > 0 && len(history) > limit {
		history = history[len(history)-limit:]
	}

	return history, nil
}

// ResetQuota resets quota usage for an entity.
func (pqm *PrintQuotaManager) ResetQuota(ctx context.Context, entityID, entityType, quotaType, period string, reason string) error {
	state, err := pqm.getQuotaState(ctx, entityID, entityType, quotaType, period)
	if err != nil {
		return err
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	state.Used = 0
	state.Remaining = state.Limit
	state.LastReset = time.Now()

	// Add reset event
	event := QuotaEvent{
		Timestamp: time.Now(),
		Action:    "reset",
		Amount:    0,
		Previous:  state.Used,
		Remaining: state.Remaining,
		Reason:    reason,
	}
	state.History = append(state.History, event)

	return nil
}

// cleanup removes stale quota states from cache.
func (pqm *PrintQuotaManager) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		pqm.mu.Lock()
		now := time.Now()

		for key, state := range pqm.quotaCache {
			state.mu.Lock()
			// Remove if period ended more than an hour ago
			if now.After(state.PeriodEnd.Add(time.Hour)) {
				delete(pqm.quotaCache, key)
			}
			state.mu.Unlock()
		}

		pqm.mu.Unlock()
	}
}

// EstimateQuotaCost estimates the quota cost of a print job.
func (pqm *PrintQuotaManager) EstimateQuotaCost(req *PrintJobRequest) map[string]int {
	costs := make(map[string]int)

	costs["pages"] = req.PageCount
	costs["color_pages"] = req.ColorPages
	costs["duplex_pages"] = req.DuplexPages
	costs["jobs"] = 1

	return costs
}

// CheckMultipleQuotaTypes checks quota across multiple quota types.
func (pqm *PrintQuotaManager) CheckMultipleQuotaTypes(ctx context.Context, entityID, entityType string, costs map[string]int) (map[string]*QuotaCheckResult, error) {
	results := make(map[string]*QuotaCheckResult)

	for quotaType, cost := range costs {
		state, err := pqm.getQuotaState(ctx, entityID, entityType, quotaType, "monthly")
		if err != nil {
			return nil, err
		}

		pqm.checkAndResetPeriod(state)

		result := &QuotaCheckResult{
			Allowed:     cost <= state.Remaining,
			QuotaType:   quotaType,
			Used:        state.Used,
			Limit:       state.Limit,
			Remaining:   state.Remaining,
			PeriodStart: state.PeriodStart,
			PeriodEnd:   state.PeriodEnd,
		}

		if !result.Allowed {
			result.Reason = fmt.Sprintf("%s quota exceeded", quotaType)
			result.RetryAfter = state.PeriodEnd
		}

		results[quotaType] = result
	}

	return results, nil
}

// GrantQuota grants additional quota to an entity.
func (pqm *PrintQuotaManager) GrantQuota(ctx context.Context, entityID, entityType, quotaType, period string, amount int, reason string) error {
	state, err := pqm.getQuotaState(ctx, entityID, entityType, quotaType, period)
	if err != nil {
		return err
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	state.Limit += amount
	state.Remaining += amount

	// Add grant event
	event := QuotaEvent{
		Timestamp: time.Now(),
		Action:    "granted",
		Amount:    amount,
		Previous:  state.Used,
		Remaining: state.Remaining,
		Reason:    reason,
	}
	state.History = append(state.History, event)

	return nil
}

// GetAggregatedQuota returns aggregated quota usage across an organization.
func (pqm *PrintQuotaManager) GetAggregatedQuota(ctx context.Context, orgID string) (*AggregatedQuota, error) {
	// Get organization quota
	orgState, err := pqm.getQuotaState(ctx, orgID, "organization", "pages", "monthly")
	if err != nil {
		return nil, err
	}

	return &AggregatedQuota{
		OrganizationID: orgID,
		OrgLimit:       orgState.Limit,
		OrgUsed:        orgState.Used,
		OrgRemaining:   orgState.Remaining,
		PeriodStart:    orgState.PeriodStart,
		PeriodEnd:      orgState.PeriodEnd,
	}, nil
}

// AggregatedQuota represents aggregated quota information.
type AggregatedQuota struct {
	OrganizationID string    `json:"organization_id"`
	OrgLimit       int       `json:"org_limit"`
	OrgUsed        int       `json:"org_used"`
	OrgRemaining   int       `json:"org_remaining"`
	UserLimit      int       `json:"user_limit"`
	UserUsed       int       `json:"user_used"`
	UserRemaining  int       `json:"user_remaining"`
	PeriodStart    time.Time `json:"period_start"`
	PeriodEnd      time.Time `json:"period_end"`
}
