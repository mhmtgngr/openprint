package ratelimit

import (
	"encoding/json"
	"time"
)

// Policy defines a rate limit policy.
type Policy struct {
	// Unique identifier for the policy
	ID string `json:"id" db:"id"`
	// Human-readable name
	Name string `json:"name" db:"name"`
	// Policy description
	Description string `json:"description" db:"description"`
	// Priority for hierarchical resolution (higher = more specific)
	Priority int `json:"priority" db:"priority"`
	// Scope: "global", "endpoint", "user", "api_key", "organization"
	Scope string `json:"scope" db:"scope"`
	// Identifier the policy applies to (user_id, api_key_id, path pattern, etc.)
	Identifier string `json:"identifier" db:"identifier"`
	// HTTP methods this policy applies to (empty = all)
	Methods []string `json:"methods" db:"methods"`
	// Request path pattern (supports wildcards)
	PathPattern string `json:"path_pattern" db:"path_pattern"`
	// Maximum requests allowed
	Limit int64 `json:"limit" db:"limit"`
	// Time window for the limit
	Window time.Duration `json:"window" db:"window"`
	// Burst limit allows temporary higher rate (0 = disabled)
	BurstLimit int64 `json:"burst_limit" db:"burst_limit"`
	// Burst duration
	BurstDuration time.Duration `json:"burst_duration" db:"burst_duration"`
	// Whether to queue requests instead of rejecting
	EnableQueue bool `json:"enable_queue" db:"enable_queue"`
	// Maximum queue size
	MaxQueueSize int `json:"max_queue_size" db:"max_queue_size"`
	// Circuit breaker threshold (0 = disabled)
	CircuitBreakerThreshold int `json:"circuit_breaker_threshold" db:"circuit_breaker_threshold"`
	// Circuit breaker timeout
	CircuitBreakerTimeout time.Duration `json:"circuit_breaker_timeout" db:"circuit_breaker_timeout"`
	// Severity level for violations: "low", "medium", "high", "critical"
	Severity string `json:"severity" db:"severity"`
	// Action on violation: "reject", "throttle", "queue", "alert_only"
	Action string `json:"action" db:"action"`
	// Throttle rate (requests per second) when action is "throttle"
	ThrottleRate float64 `json:"throttle_rate" db:"throttle_rate"`
	// Whether policy is active
	IsActive bool `json:"is_active" db:"is_active"`
	// Created timestamp
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	// Updated timestamp
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Violation represents a rate limit violation record.
type Violation struct {
	ID             string    `json:"id" db:"id"`
	PolicyID       string    `json:"policy_id" db:"policy_id"`
	PolicyName     string    `json:"policy_name" db:"policy_name"`
	Identifier     string    `json:"identifier" db:"identifier"`
	IdentifierType string    `json:"identifier_type" db:"identifier_type"`
	Path           string    `json:"path" db:"path"`
	Method         string    `json:"method" db:"method"`
	Current        int64     `json:"current" db:"current"`
	Limit          int64     `json:"limit" db:"limit"`
	Severity       string    `json:"severity" db:"severity"`
	OccurredAt     time.Time `json:"occurred_at" db:"occurred_at"`
}

// TrustedClient represents a client that bypasses rate limiting.
type TrustedClient struct {
	ID          string     `json:"id" db:"id"`
	Name        string     `json:"name" db:"name"`
	APIKey      string     `json:"api_key" db:"api_key"`
	IPWhitelist []string   `json:"ip_whitelist" db:"ip_whitelist"`
	Description string     `json:"description" db:"description"`
	IsActive    bool       `json:"is_active" db:"is_active"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	LastUsedAt  *time.Time `json:"last_used_at" db:"last_used_at"`
}

// CircuitBreakerState represents the state of a circuit breaker.
type CircuitBreakerState struct {
	Path          string     `json:"path"`
	State         string     `json:"state"` // "closed", "open", "half_open"
	FailureCount  int        `json:"failure_count"`
	LastFailureAt time.Time  `json:"last_failure_at"`
	OpensAt       *time.Time `json:"opens_at,omitempty"`
	ClosesAt      *time.Time `json:"closes_at,omitempty"`
}

// QueueStats represents statistics for the request queue.
type QueueStats struct {
	Path         string        `json:"path"`
	QueueSize    int           `json:"queue_size"`
	MaxQueueSize int           `json:"max_queue_size"`
	AvgWaitTime  time.Duration `json:"avg_wait_time"`
	P95WaitTime  time.Duration `json:"p95_wait_time"`
}

// PolicyStats represents statistics for a policy.
type PolicyStats struct {
	PolicyID      string        `json:"policy_id"`
	PolicyName    string        `json:"policy_name"`
	TotalRequests int64         `json:"total_requests"`
	AllowedReq    int64         `json:"allowed_requests"`
	DeniedReq     int64         `json:"denied_requests"`
	Violations    int64         `json:"violations"`
	AvgUsage      float64       `json:"avg_usage"`
	PeakUsage     int64         `json:"peak_usage"`
	TimeWindow    time.Duration `json:"time_window"`
}

// Common policy presets

// DefaultGlobalPolicy returns the default global rate limit policy.
func DefaultGlobalPolicy() *Policy {
	return &Policy{
		ID:            "global-default",
		Name:          "Global Default",
		Description:   "Default rate limit for all requests",
		Priority:      0,
		Scope:         "global",
		Identifier:    "*",
		Limit:         1000,
		Window:        time.Hour,
		BurstLimit:    100,
		BurstDuration: time.Minute,
		EnableQueue:   false,
		Severity:      "low",
		Action:        "reject",
		IsActive:      true,
	}
}

// DefaultIPPolicy returns a default IP-based rate limit policy.
func DefaultIPPolicy() *Policy {
	return &Policy{
		ID:            "ip-default",
		Name:          "IP Default",
		Description:   "Default rate limit per IP address",
		Priority:      10,
		Scope:         "ip",
		Identifier:    "*",
		Limit:         100,
		Window:        time.Minute,
		BurstLimit:    20,
		BurstDuration: 10 * time.Second,
		EnableQueue:   true,
		MaxQueueSize:  10,
		Severity:      "medium",
		Action:        "reject",
		IsActive:      true,
	}
}

// DefaultUserPolicy returns a default user-based rate limit policy.
func DefaultUserPolicy() *Policy {
	return &Policy{
		ID:            "user-default",
		Name:          "User Default",
		Description:   "Default rate limit per user",
		Priority:      20,
		Scope:         "user",
		Identifier:    "*",
		Limit:         500,
		Window:        time.Hour,
		BurstLimit:    50,
		BurstDuration: time.Minute,
		EnableQueue:   true,
		MaxQueueSize:  20,
		Severity:      "medium",
		Action:        "reject",
		IsActive:      true,
	}
}

// DefaultAPIKeyPolicy returns a default API key-based rate limit policy.
func DefaultAPIKeyPolicy() *Policy {
	return &Policy{
		ID:            "apikey-default",
		Name:          "API Key Default",
		Description:   "Default rate limit per API key",
		Priority:      30,
		Scope:         "api_key",
		Identifier:    "*",
		Limit:         1000,
		Window:        time.Hour,
		BurstLimit:    100,
		BurstDuration: time.Minute,
		EnableQueue:   false,
		Severity:      "low",
		Action:        "reject",
		IsActive:      true,
	}
}

// StrictEndpointPolicy returns a strict policy for sensitive endpoints.
func StrictEndpointPolicy() *Policy {
	return &Policy{
		ID:                      "endpoint-strict",
		Name:                    "Strict Endpoint",
		Description:             "Strict rate limit for sensitive endpoints",
		Priority:                100,
		Scope:                   "endpoint",
		PathPattern:             "/api/auth/*",
		Limit:                   10,
		Window:                  time.Minute,
		BurstLimit:              3,
		BurstDuration:           10 * time.Second,
		CircuitBreakerThreshold: 50,
		CircuitBreakerTimeout:   5 * time.Minute,
		EnableQueue:             false,
		Severity:                "high",
		Action:                  "reject",
		IsActive:                true,
	}
}

// MethodsMatch checks if the policy applies to the given method.
func (p *Policy) MethodsMatch(method string) bool {
	if len(p.Methods) == 0 {
		return true
	}
	for _, m := range p.Methods {
		if m == "*" || m == method {
			return true
		}
	}
	return false
}

// PathMatches checks if the policy applies to the given path.
func (p *Policy) PathMatches(path string) bool {
	if p.PathPattern == "" || p.PathPattern == "*" {
		return true
	}
	// Simple wildcard matching
	// In production, use a proper pattern matching library
	if p.PathPattern == path {
		return true
	}
	// Check for wildcard prefix/suffix
	if len(p.PathPattern) > 0 {
		// Handle patterns like "/api/auth/*"
		if p.PathPattern[len(p.PathPattern)-1] == '*' {
			prefix := p.PathPattern[:len(p.PathPattern)-1]
			if len(path) >= len(prefix) && path[:len(prefix)] == prefix {
				return true
			}
		}
	}
	return false
}

// Matches checks if the policy matches the given criteria.
func (p *Policy) Matches(scope, identifier, method, path string) bool {
	if p.Scope != scope {
		return false
	}
	if p.Identifier != "*" && p.Identifier != identifier {
		return false
	}
	if !p.MethodsMatch(method) {
		return false
	}
	if !p.PathMatches(path) {
		return false
	}
	return true
}

// GetWindowInSeconds returns the window duration in seconds.
func (p *Policy) GetWindowInSeconds() int64 {
	return int64(p.Window.Seconds())
}

// MarshalJSON implements custom JSON marshaling for Policy.
func (p Policy) MarshalJSON() ([]byte, error) {
	type Alias Policy
	return json.Marshal(&struct {
		WindowSec                int64 `json:"window_sec"`
		BurstDurationSec         int64 `json:"burst_duration_sec,omitempty"`
		CircuitBreakerTimeoutSec int64 `json:"circuit_breaker_timeout_sec,omitempty"`
		*Alias
	}{
		WindowSec:                p.GetWindowInSeconds(),
		BurstDurationSec:         int64(p.BurstDuration.Seconds()),
		CircuitBreakerTimeoutSec: int64(p.CircuitBreakerTimeout.Seconds()),
		Alias:                    (*Alias)(&p),
	})
}
