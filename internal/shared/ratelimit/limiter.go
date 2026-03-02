// Package ratelimit provides a comprehensive Redis-based rate limiting system
// with sliding window algorithm, hierarchical policy resolution, and advanced features.
package ratelimit

import (
	"context"
	"fmt"
	"time"
)

// Limiter is the main rate limiter using sliding window algorithm.
type Limiter struct {
	redis      *RedisClient
	resolver   *PolicyResolver
	metrics    *Metrics
	alert      *AlertManager
	circuit    *CircuitBreaker
	bypass     *BypassManager
	queue      *RequestQueue
	repository *Repository
}

// Config holds rate limiter configuration.
type Config struct {
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	// Default policies applied when no specific policy is found
	DefaultPolicy *Policy
	// Enable metrics collection
	EnableMetrics bool
	// Enable violation alerts
	EnableAlerts bool
	// Enable circuit breaker
	EnableCircuitBreaker bool
	// Enable request queuing
	EnableQueue bool
	// Trusted clients bypass
	TrustedClients []string
}

// NewLimiter creates a new rate limiter instance.
func NewLimiter(cfg *Config) (*Limiter, error) {
	// Initialize Redis client
	redisCfg := &RedisConfig{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	}
	redisClient, err := NewRedisClient(redisCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create redis client: %w", err)
	}

	limiter := &Limiter{
		redis: redisClient,
	}

	// Initialize policy resolver
	limiter.resolver = NewPolicyResolver(cfg.DefaultPolicy)

	// Initialize metrics if enabled
	if cfg.EnableMetrics {
		limiter.metrics = NewMetrics()
	}

	// Initialize alert manager if enabled
	if cfg.EnableAlerts {
		limiter.alert = NewAlertManager(redisClient)
	}

	// Initialize circuit breaker if enabled
	if cfg.EnableCircuitBreaker {
		limiter.circuit = NewCircuitBreaker(redisClient)
	}

	// Initialize bypass manager
	limiter.bypass = NewBypassManager(cfg.TrustedClients)

	// Initialize request queue if enabled
	if cfg.EnableQueue {
		limiter.queue = NewRequestQueue(redisClient)
	}

	// Initialize repository
	limiter.repository = NewRepository(redisClient)

	return limiter, nil
}

// CheckResult represents the result of a rate limit check.
type CheckResult struct {
	Allowed         bool          `json:"allowed"`
	Limit           int64         `json:"limit"`
	Remaining       int64         `json:"remaining"`
	ResetAt         time.Time     `json:"reset_at"`
	RetryAfter      time.Duration `json:"retry_after,omitempty"`
	Policy          *Policy       `json:"policy,omitempty"`
	ViolationID     string        `json:"violation_id,omitempty"`
	IsBypassed      bool          `json:"is_bypassed,omitempty"`
	IsQueued        bool          `json:"is_queued,omitempty"`
	QueuePosition   int           `json:"queue_position,omitempty"`
	EstimatedWait   time.Duration `json:"estimated_wait,omitempty"`
}

// Request represents a rate limit check request.
type Request struct {
	// Identifier for the entity being rate limited (IP, user ID, API key, etc.)
	Identifier string `json:"identifier"`
	// Type of identifier: "ip", "user", "api_key", "organization"
	Type string `json:"type"`
	// The endpoint/method being accessed
	Method string `json:"method"`
	// The full request path
	Path string `json:"path"`
	// User's role (for policy resolution)
	Role string `json:"role,omitempty"`
	// User's organization ID
	OrgID string `json:"org_id,omitempty"`
	// API key if used
	APIKey string `json:"api_key,omitempty"`
	// Client IP address
	IP string `json:"ip,omitempty"`
	// Request timestamp
	Timestamp time.Time `json:"timestamp"`
	// Whether this is a burst request
	IsBurst bool `json:"is_burst,omitempty"`
	// Priority for queuing (0-100, higher = more important)
	Priority int `json:"priority,omitempty"`
}

// Check checks if a request should be allowed based on rate limits.
func (l *Limiter) Check(ctx context.Context, req *Request) (*CheckResult, error) {
	result := &CheckResult{
		Allowed: true,
	}

	// Check for bypass first
	if l.bypass != nil && l.bypass.ShouldBypass(ctx, req) {
		result.IsBypassed = true
		return result, nil
	}

	// Resolve the applicable policy
	policy, err := l.resolver.Resolve(ctx, l.repository, req)
	if err != nil {
		// Log error but allow request if policy resolution fails
		result.Allowed = true
		return result, nil
	}

	result.Policy = policy

	// Check circuit breaker
	if l.circuit != nil {
		tripped, until, err := l.circuit.Check(ctx, req.Path)
		if err == nil && tripped {
			result.Allowed = false
			result.RetryAfter = time.Until(until)
			return result, nil
		}
	}

	// Perform the actual rate limit check using sliding window
	now := req.Timestamp
	if now.IsZero() {
		now = time.Now()
	}

	windowStart := now.Add(-policy.Window)

	// Get the key for this request
	key := l.getRedisKey(req, policy)

	// Add current request to the window
	current, limit, resetAt, err := l.slidingWindowCheck(ctx, key, now, windowStart, policy)
	if err != nil {
		// Log error but allow request if Redis fails (fail open)
		result.Allowed = true
		return result, nil
	}

	result.Limit = limit
	result.Remaining = limit - current
	result.ResetAt = resetAt

	// Check if limit exceeded
	if current > limit {
		// Check if request queuing is enabled and enabled for this policy
		if l.queue != nil && policy.EnableQueue && req.Priority > 0 {
			queued, position, wait := l.queue.Enqueue(ctx, req, policy)
			if queued {
				result.Allowed = false
				result.IsQueued = true
				result.QueuePosition = position
				result.EstimatedWait = wait
				return result, nil
			}
		}

		result.Allowed = false
		result.RetryAfter = time.Until(resetAt)

		// Record violation
		if l.repository != nil {
			violationID, _ := l.recordViolation(ctx, req, policy, current, limit)
			result.ViolationID = violationID
		}

		// Check if circuit breaker should be triggered
		if l.circuit != nil && policy.CircuitBreakerThreshold > 0 {
			_ = l.circuit.RecordFailure(ctx, req.Path, policy.CircuitBreakerThreshold)
		}

		// Send alert
		if l.alert != nil {
			_ = l.alert.SendViolationAlert(ctx, req, policy, current, limit)
		}

		// Record metrics
		if l.metrics != nil {
			l.metrics.RecordDenied(req.Type, req.Path, policy.Name)
		}

		return result, nil
	}

	// Record metrics for allowed request
	if l.metrics != nil {
		l.metrics.RecordAllowed(req.Type, req.Path, policy.Name)
	}

	// Record success for circuit breaker
	if l.circuit != nil {
		_ = l.circuit.RecordSuccess(ctx, req.Path)
	}

	return result, nil
}

// slidingWindowCheck implements the sliding window algorithm using Redis sorted sets.
func (l *Limiter) slidingWindowCheck(ctx context.Context, key string, now, windowStart time.Time, policy *Policy) (current int64, limit int64, resetAt time.Time, err error) {
	// Use a Lua script for atomic operations
	script := `
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local window_start = tonumber(ARGV[2])
		local window = tonumber(ARGV[3])
		local limit = tonumber(ARGV[4])

		-- Remove entries outside the window
		redis.call('ZREMRANGEBYSCORE', key, 0, window_start)

		-- Count current entries
		local current = redis.call('ZCARD', key)

		-- Add current request
		redis.call('ZADD', key, now, now)

		-- Set expiration to window duration + buffer
		redis.call('EXPIRE', key, window + 1)

		-- Calculate window end for reset time
		local window_end = now + window

		return {current, limit, window_end}
	`

	limit = policy.Limit
	if policy.BurstLimit > 0 && policy.BurstDuration > 0 {
		// Check if within burst window
		burstEnd := now.Add(policy.BurstDuration)
		if now.Before(burstEnd) {
			limit = policy.BurstLimit
		}
	}

	result, err := l.redis.client.Eval(ctx, script, []string{key},
		now.UnixMilli(),
		windowStart.UnixMilli(),
		policy.Window.Milliseconds(),
		limit,
	).Result()

	if err != nil {
		return 0, limit, time.Time{}, err
	}

	values := result.([]interface{})
	current = int64(values[0].(int64))
	resetAt = time.UnixMilli(values[2].(int64))

	return current, limit, resetAt, nil
}

// getRedisKey generates the Redis key for storing rate limit data.
func (l *Limiter) getRedisKey(req *Request, policy *Policy) string {
	return fmt.Sprintf("ratelimit:%s:%s:%s", policy.Scope, req.Type, req.Identifier)
}

// RecordViolation records a rate limit violation in the database.
func (l *Limiter) recordViolation(ctx context.Context, req *Request, policy *Policy, current, limit int64) (string, error) {
	violation := &Violation{
		ID:           generateID(),
		PolicyID:     policy.ID,
		PolicyName:   policy.Name,
		Identifier:   req.Identifier,
		IdentifierType: req.Type,
		Path:         req.Path,
		Method:       req.Method,
		Current:      current,
		Limit:        limit,
		Severity:     policy.Severity,
		OccurredAt:   time.Now(),
	}

	if err := l.repository.CreateViolation(ctx, violation); err != nil {
		return "", err
	}

	return violation.ID, nil
}

// Reset resets the rate limit counter for a given identifier.
func (l *Limiter) Reset(ctx context.Context, req *Request) error {
	policy, err := l.resolver.Resolve(ctx, l.repository, req)
	if err != nil {
		return err
	}

	key := l.getRedisKey(req, policy)
	return l.redis.client.Del(ctx, key).Err()
}

// GetUsage returns current usage statistics for a request.
func (l *Limiter) GetUsage(ctx context.Context, req *Request) (current int64, limit int64, windowStart time.Time, err error) {
	policy, err := l.resolver.Resolve(ctx, l.repository, req)
	if err != nil {
		return 0, 0, time.Time{}, err
	}

	key := l.getRedisKey(req, policy)
	now := time.Now()
	windowStart = now.Add(-policy.Window)

	// Count entries in the current window
	script := `
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local window_start = tonumber(ARGV[2])

		-- Count entries in the window
		local current = redis.call('ZCOUNT', key, window_start, now)

		return current
	`

	result, err := l.redis.client.Eval(ctx, script, []string{key},
		now.UnixMilli(),
		windowStart.UnixMilli(),
	).Result()

	if err != nil {
		return 0, policy.Limit, windowStart, err
	}

	current = result.(int64)
	limit = policy.Limit

	return current, limit, windowStart, nil
}

// Close closes the rate limiter and releases resources.
func (l *Limiter) Close() error {
	if l.redis != nil {
		return l.redis.Close()
	}
	return nil
}

// GetCircuitBreaker returns the circuit breaker instance.
func (l *Limiter) GetCircuitBreaker() *CircuitBreaker {
	return l.circuit
}

// GetRepository returns the repository instance.
func (l *Limiter) GetRepository() *Repository {
	return l.repository
}

// generateID generates a unique ID for violations.
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
