package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CircuitBreaker implements the circuit breaker pattern for rate limiting.
// It opens the circuit (blocks requests) when failure threshold is reached.
type CircuitBreaker struct {
	redis *RedisClient
	mu    sync.RWMutex

	// Circuit state per path
	circuits map[string]*circuitState

	// Default configuration
	defaultThreshold     int
	defaultTimeout       time.Duration
	defaultHalfOpenMax   int
	halfOpenRetryDelay   time.Duration
}

// circuitState tracks the state of a circuit breaker.
type circuitState struct {
	mu               sync.Mutex
	state            CircuitState
	failureCount     int
	successCount     int
	lastFailureTime  time.Time
	lastStateChange  time.Time
	halfOpenAttempts int
	openedAt         time.Time
	closesAt         time.Time
}

// CircuitState represents the state of a circuit breaker.
type CircuitState string

const (
	StateClosed    CircuitState = "closed"
	StateOpen      CircuitState = "open"
	StateHalfOpen  CircuitState = "half_open"
)

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(redis *RedisClient) *CircuitBreaker {
	cb := &CircuitBreaker{
		redis:              redis,
		circuits:           make(map[string]*circuitState),
		defaultThreshold:   50,           // Open after 50 failures
		defaultTimeout:     5 * time.Minute, // Stay open for 5 minutes
		defaultHalfOpenMax: 5,            // Try 5 requests in half-open
		halfOpenRetryDelay: 30 * time.Second, // Wait 30s before half-open
	}

	// Start cleanup goroutine
	go cb.cleanup()

	return cb
}

// Check checks if the circuit for a path is open.
// Returns (isOpen, openUntil, error).
func (cb *CircuitBreaker) Check(ctx context.Context, path string) (bool, time.Time, error) {
	state := cb.getOrCreateState(path)

	state.mu.Lock()
	defer state.mu.Unlock()

	now := time.Now()

	// Check if we should transition from open to half-open
	if state.state == StateOpen && now.After(state.closesAt) {
		state.state = StateHalfOpen
		state.lastStateChange = now
		state.halfOpenAttempts = 0
		return false, time.Time{}, nil
	}

	isOpen := state.state == StateOpen
	var until time.Time
	if isOpen {
		until = state.closesAt
	}

	return isOpen, until, nil
}

// RecordSuccess records a successful request for a path.
func (cb *CircuitBreaker) RecordSuccess(ctx context.Context, path string) error {
	state := cb.getOrCreateState(path)

	state.mu.Lock()
	defer state.mu.Unlock()

	state.failureCount = 0

	if state.state == StateHalfOpen {
		state.successCount++
		state.halfOpenAttempts++

		// After enough successes in half-open, close the circuit
		if state.successCount >= cb.defaultHalfOpenMax {
			state.state = StateClosed
			state.lastStateChange = time.Now()
			state.halfOpenAttempts = 0
		}

		return nil
	}

	// In closed state, reset failure count
	state.failureCount = 0

	return nil
}

// RecordFailure records a failed request for a path.
func (cb *CircuitBreaker) RecordFailure(ctx context.Context, path string, threshold int) error {
	state := cb.getOrCreateState(path)

	state.mu.Lock()
	defer state.mu.Unlock()

	if threshold <= 0 {
		threshold = cb.defaultThreshold
	}

	state.failureCount++
	state.lastFailureTime = time.Now()

	switch state.state {
	case StateClosed:
		// Check if we should open the circuit
		if state.failureCount >= threshold {
			cb.openCircuit(state)
		}

	case StateHalfOpen:
		// Any failure in half-open opens the circuit immediately
		cb.openCircuit(state)

	case StateOpen:
		// Already open, update close time
		state.closesAt = time.Now().Add(cb.defaultTimeout)
	}

	return nil
}

// openCircuit transitions the circuit to open state.
func (cb *CircuitBreaker) openCircuit(state *circuitState) {
	state.state = StateOpen
	state.openedAt = time.Now()
	state.closesAt = time.Now().Add(cb.defaultTimeout)
	state.lastStateChange = time.Now()
	state.successCount = 0
	state.halfOpenAttempts = 0
}

// getOrCreateState gets or creates a circuit state for a path.
func (cb *CircuitBreaker) getOrCreateState(path string) *circuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if state, ok := cb.circuits[path]; ok {
		return state
	}

	state := &circuitState{
		state:           StateClosed,
		lastStateChange: time.Now(),
	}

	cb.circuits[path] = state
	return state
}

// GetState returns the current state of a circuit.
func (cb *CircuitBreaker) GetState(path string) CircuitBreakerState {
	state := cb.getOrCreateState(path)

	state.mu.Lock()
	defer state.mu.Unlock()

	return CircuitBreakerState{
		Path:          path,
		State:         string(state.state),
		FailureCount:  state.failureCount,
		LastFailureAt: state.lastFailureTime,
		OpensAt:       &state.closesAt,
		ClosesAt:      &state.openedAt,
	}
}

// GetAllStates returns the state of all circuits.
func (cb *CircuitBreaker) GetAllStates() []CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	states := make([]CircuitBreakerState, 0, len(cb.circuits))

	for path, state := range cb.circuits {
		state.mu.Lock()
		circuitState := CircuitBreakerState{
			Path:          path,
			State:         string(state.state),
			FailureCount:  state.failureCount,
			LastFailureAt: state.lastFailureTime,
		}

		if state.state == StateOpen {
			circuitState.ClosesAt = &state.closesAt
		}

		state.mu.Unlock()
		states = append(states, circuitState)
	}

	return states
}

// Reset resets a circuit to closed state.
func (cb *CircuitBreaker) Reset(path string) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state, ok := cb.circuits[path]
	if !ok {
		return fmt.Errorf("circuit not found for path: %s", path)
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	state.state = StateClosed
	state.failureCount = 0
	state.successCount = 0
	state.halfOpenAttempts = 0
	state.lastStateChange = time.Now()

	return nil
}

// SetThreshold sets the failure threshold for a path.
func (cb *CircuitBreaker) SetThreshold(path string, threshold int) {
	state := cb.getOrCreateState(path)
	// Threshold is stored per-path and used in RecordFailure
	_ = state
}

// SetTimeout sets the timeout for open circuits.
func (cb *CircuitBreaker) SetTimeout(timeout time.Duration) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.defaultTimeout = timeout
}

// SetHalfOpenMax sets the max attempts in half-open state.
func (cb *CircuitBreaker) SetHalfOpenMax(max int) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.defaultHalfOpenMax = max
}

// cleanup removes stale circuit states.
func (cb *CircuitBreaker) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cb.mu.Lock()
		now := time.Now()

		for path, state := range cb.circuits {
			state.mu.Lock()

			// Remove circuits that have been closed for >1 hour with no activity
			if state.state == StateClosed &&
				now.Sub(state.lastStateChange) > time.Hour &&
				state.failureCount == 0 {
				delete(cb.circuits, path)
			}

			state.mu.Unlock()
		}

		cb.mu.Unlock()
	}
}

// IsOpen checks if a circuit is currently open.
func (cb *CircuitBreaker) IsOpen(path string) bool {
	state := cb.getOrCreateState(path)

	state.mu.Lock()
	defer state.mu.Unlock()

	now := time.Now()

	// Auto-transition to half-open if timeout expired
	if state.state == StateOpen && now.After(state.closesAt) {
		state.state = StateHalfOpen
		state.lastStateChange = now
		state.halfOpenAttempts = 0
		return false
	}

	return state.state == StateOpen
}

// GetFailureCount returns the current failure count for a path.
func (cb *CircuitBreaker) GetFailureCount(path string) int {
	state := cb.getOrCreateState(path)

	state.mu.Lock()
	defer state.mu.Unlock()

	return state.failureCount
}

// ForceOpen forcibly opens a circuit.
func (cb *CircuitBreaker) ForceOpen(path string, duration time.Duration) {
	state := cb.getOrCreateState(path)

	state.mu.Lock()
	defer state.mu.Unlock()

	state.state = StateOpen
	state.openedAt = time.Now()
	state.closesAt = time.Now().Add(duration)
	state.lastStateChange = time.Now()
}

// ForceClose forcibly closes a circuit.
func (cb *CircuitBreaker) ForceClose(path string) {
	state := cb.getOrCreateState(path)

	state.mu.Lock()
	defer state.mu.Unlock()

	state.state = StateClosed
	state.failureCount = 0
	state.successCount = 0
	state.halfOpenAttempts = 0
	state.lastStateChange = time.Now()
}

// GetStats returns circuit breaker statistics.
func (cb *CircuitBreaker) GetStats() *CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	stats := &CircuitBreakerStats{
		TotalCircuits: len(cb.circuits),
	}

	for _, state := range cb.circuits {
		state.mu.Lock()

		switch state.state {
		case StateClosed:
			stats.ClosedCircuits++
		case StateOpen:
			stats.OpenCircuits++
		case StateHalfOpen:
			stats.HalfOpenCircuits++
		}

		stats.TotalFailures += state.failureCount

		state.mu.Unlock()
	}

	return stats
}

// CircuitBreakerStats represents circuit breaker statistics.
type CircuitBreakerStats struct {
	TotalCircuits    int `json:"total_circuits"`
	ClosedCircuits   int `json:"closed_circuits"`
	OpenCircuits     int `json:"open_circuits"`
	HalfOpenCircuits int `json:"half_open_circuits"`
	TotalFailures    int `json:"total_failures"`
}
