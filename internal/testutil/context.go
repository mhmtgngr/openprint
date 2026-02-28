// Package testutil provides test context management utilities for testing.
package testutil

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// ContextManager manages test contexts with timeouts and cancellation.
type ContextManager struct {
	mu            sync.Mutex
	ctx           context.Context
	cancel        context.CancelFunc
	timeouts      map[string]context.CancelFunc
	defaultCancel context.CancelFunc
}

// NewContextManager creates a new context manager.
func NewContextManager() *ContextManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &ContextManager{
		ctx:           ctx,
		cancel:        cancel,
		timeouts:      make(map[string]context.CancelFunc),
		defaultCancel: cancel,
	}
}

// Context returns the base context.
func (cm *ContextManager) Context() context.Context {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.ctx
}

// WithTimeout creates a context with a timeout.
// The context is automatically cancelled when the timeout expires.
func (cm *ContextManager) WithTimeout(name string, timeout time.Duration) (context.Context, context.CancelFunc) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	ctx, cancel := context.WithTimeout(cm.ctx, timeout)

	// Store the cancel function so we can clean it up later
	if oldCancel, exists := cm.timeouts[name]; exists {
		oldCancel()
	}
	cm.timeouts[name] = cancel

	return ctx, cancel
}

// WithDeadline creates a context with a deadline.
func (cm *ContextManager) WithDeadline(name string, deadline time.Time) (context.Context, context.CancelFunc) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	ctx, cancel := context.WithDeadline(cm.ctx, deadline)

	if oldCancel, exists := cm.timeouts[name]; exists {
		oldCancel()
	}
	cm.timeouts[name] = cancel

	return ctx, cancel
}

// CancelTimeout cancels a named timeout context.
func (cm *ContextManager) CancelTimeout(name string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cancel, exists := cm.timeouts[name]; exists {
		cancel()
		delete(cm.timeouts, name)
	}
}

// CancelAll cancels all contexts.
func (cm *ContextManager) CancelAll() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Cancel all named timeouts
	for _, cancel := range cm.timeouts {
		cancel()
	}
	cm.timeouts = make(map[string]context.CancelFunc)

	// Cancel base context
	if cm.cancel != nil {
		cm.cancel()
	}
}

// Cleanup cleans up all resources.
func (cm *ContextManager) Cleanup() {
	cm.CancelAll()
}

// ShortContext returns a context with a short timeout (5 seconds).
func ShortContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

// MediumContext returns a context with a medium timeout (30 seconds).
func MediumContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

// LongContext returns a context with a long timeout (2 minutes).
func LongContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 2*time.Minute)
}

// TestContext returns a context appropriate for testing (30 seconds).
func TestContext() (context.Context, context.CancelFunc) {
	return MediumContext()
}

// WithTestTimeout wraps a function with a test timeout.
func WithTestTimeout(timeout time.Duration, fn func(ctx context.Context) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return fn(ctx)
}

// DeadlineContext creates a context with a deadline at the specified duration from now.
func DeadlineContext(duration time.Duration) (context.Context, context.CancelFunc) {
	return context.WithDeadline(context.Background(), time.Now().Add(duration))
}

// BackgroundContext is a convenience function for context.Background().
func BackgroundContext() context.Context {
	return context.Background()
}

// CanceledContext returns a context that is already cancelled.
func CanceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

// ContextWithValues creates a context with multiple key-value pairs.
type contextKey string

// ContextWithValue creates a context with a single value.
func ContextWithValue(ctx context.Context, key, value string) context.Context {
	return context.WithValue(ctx, contextKey(key), value)
}

// ContextWithValues creates a context with multiple values.
func ContextWithValues(ctx context.Context, values map[string]interface{}) context.Context {
	for k, v := range values {
		ctx = context.WithValue(ctx, contextKey(k), v)
	}
	return ctx
}

// GetValue retrieves a value from the context.
func GetValue(ctx context.Context, key string) interface{} {
	return ctx.Value(contextKey(key))
}

// GetString retrieves a string value from the context.
func GetString(ctx context.Context, key string) string {
	if v := GetValue(ctx, key); v != nil {
		if str, ok := v.(string); ok {
			return str
		}
	}
	return ""
}

// GetInt retrieves an int value from the context.
func GetInt(ctx context.Context, key string) int {
	if v := GetValue(ctx, key); v != nil {
		if i, ok := v.(int); ok {
			return i
		}
	}
	return 0
}

// GetDuration retrieves a duration value from the context.
func GetDuration(ctx context.Context, key string) time.Duration {
	if v := GetValue(ctx, key); v != nil {
		if d, ok := v.(time.Duration); ok {
			return d
		}
	}
	return 0
}

// ContextTracker tracks context creation and cancellation for debugging.
type ContextTracker struct {
	mu        sync.Mutex
	contexts  map[string]time.Time
	cancelled map[string]time.Time
	timedout  map[string]time.Time
}

// NewContextTracker creates a new context tracker.
func NewContextTracker() *ContextTracker {
	return &ContextTracker{
		contexts:  make(map[string]time.Time),
		cancelled: make(map[string]time.Time),
		timedout:  make(map[string]time.Time),
	}
}

// Track tracks a context with the given name.
func (ct *ContextTracker) Track(name string, ctx context.Context) context.Context {
	ct.mu.Lock()
	ct.contexts[name] = time.Now()
	ct.mu.Unlock()

	// Track cancellation
	go func() {
		<-ctx.Done()
		ct.mu.Lock()
		defer ct.mu.Unlock()

		if ctx.Err() == context.DeadlineExceeded {
			ct.timedout[name] = time.Now()
		} else if ctx.Err() == context.Canceled {
			ct.cancelled[name] = time.Now()
		}
	}()

	return ctx
}

// GetTracked returns a tracked context by name.
func (ct *ContextTracker) GetTracked(name string) context.Context {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	if _, exists := ct.contexts[name]; !exists {
		return nil
	}

	// Return a new context that will be tracked
	ctx, _ := context.WithCancel(context.Background())
	ct.Track(name, ctx)
	return ctx
}

// Status returns the status of a tracked context.
func (ct *ContextTracker) Status(name string) string {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	if _, exists := ct.contexts[name]; !exists {
		return "not found"
	}

	if _, exists := ct.timedout[name]; exists {
		return "timed out"
	}

	if _, exists := ct.cancelled[name]; exists {
		return "cancelled"
	}

	return "active"
}

// CleanupTracker cleans up the context tracker.
func (ct *ContextTracker) CleanupTracker() {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.contexts = make(map[string]time.Time)
	ct.cancelled = make(map[string]time.Time)
	ct.timedout = make(map[string]time.Time)
}

// TestDeadlineTracker helps track deadlines in tests.
type TestDeadlineTracker struct {
	mu        sync.Mutex
	deadlines map[string]time.Time
}

// NewTestDeadlineTracker creates a new test deadline tracker.
func NewTestDeadlineTracker() *TestDeadlineTracker {
	return &TestDeadlineTracker{
		deadlines: make(map[string]time.Time),
	}
}

// SetDeadline sets a deadline for a named operation.
func (tdt *TestDeadlineTracker) SetDeadline(name string, deadline time.Time) {
	tdt.mu.Lock()
	defer tdt.mu.Unlock()
	tdt.deadlines[name] = deadline
}

// GetDeadline returns the deadline for a named operation.
func (tdt *TestDeadlineTracker) GetDeadline(name string) (time.Time, bool) {
	tdt.mu.Lock()
	defer tdt.mu.Unlock()
	d, ok := tdt.deadlines[name]
	return d, ok
}

// TimeRemaining returns the time remaining until the deadline.
func (tdt *TestDeadlineTracker) TimeRemaining(name string) time.Duration {
	tdt.mu.Lock()
	defer tdt.mu.Unlock()

	d, ok := tdt.deadlines[name]
	if !ok {
		return -1
	}

	return time.Until(d)
}

// IsExpired checks if a deadline has expired.
func (tdt *TestDeadlineTracker) IsExpired(name string) bool {
	remaining := tdt.TimeRemaining(name)
	return remaining <= 0
}

// WaitForContext waits for a context to be done or a timeout.
func WaitForContext(ctx context.Context, timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return fmt.Errorf("timeout waiting for context")
	}
}

// WaitForContextWithPoll waits for a context to be done with polling.
// This is useful when you need to check conditions while waiting.
func WaitForContextWithPoll(ctx context.Context, pollInterval, timeout time.Duration, condition func() bool) error {
	timer := time.NewTimer(timeout)
	poller := time.NewTicker(pollInterval)
	defer timer.Stop()
	defer poller.Stop()

	for {
		if condition() {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return fmt.Errorf("timeout waiting for condition")
		case <-poller.C:
			// Check condition again
		}
	}
}

// TestContextBuilder helps build test contexts.
type TestContextBuilder struct {
	parent    context.Context
	values    map[string]interface{}
	timeout   time.Duration
	deadline  time.Time
	hasCancel bool
}

// NewTestContextBuilder creates a new test context builder.
func NewTestContextBuilder() *TestContextBuilder {
	return &TestContextBuilder{
		parent:  context.Background(),
		values:  make(map[string]interface{}),
		timeout: 30 * time.Second,
	}
}

// WithParent sets the parent context.
func (tcb *TestContextBuilder) WithParent(ctx context.Context) *TestContextBuilder {
	tcb.parent = ctx
	return tcb
}

// WithTimeout sets the timeout.
func (tcb *TestContextBuilder) WithTimeout(timeout time.Duration) *TestContextBuilder {
	tcb.timeout = timeout
	return tcb
}

// WithDeadline sets the deadline.
func (tcb *TestContextBuilder) WithDeadline(deadline time.Time) *TestContextBuilder {
	tcb.deadline = deadline
	return tcb
}

// WithValue adds a value to the context.
func (tcb *TestContextBuilder) WithValue(key string, value interface{}) *TestContextBuilder {
	tcb.values[key] = value
	return tcb
}

// WithValues adds multiple values to the context.
func (tcb *TestContextBuilder) WithValues(values map[string]interface{}) *TestContextBuilder {
	for k, v := range values {
		tcb.values[k] = v
	}
	return tcb
}

// Build creates the context with the specified configuration.
func (tcb *TestContextBuilder) Build() (context.Context, context.CancelFunc) {
	ctx := tcb.parent
	var cancel context.CancelFunc

	if !tcb.deadline.IsZero() {
		ctx, cancel = context.WithDeadline(ctx, tcb.deadline)
	} else if tcb.timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, tcb.timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}

	// Add values
	for k, v := range tcb.values {
		ctx = context.WithValue(ctx, contextKey(k), v)
	}

	return ctx, cancel
}

// MustBuild creates the context and panics on error.
func (tcb *TestContextBuilder) MustBuild() context.Context {
	ctx, _ := tcb.Build()
	return ctx
}

// SetupTestContext is a helper function to set up a test context with cleanup.
// It registers the cleanup with testing.T and returns the context.
func SetupTestContext(t *testing.T, timeout time.Duration) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	t.Cleanup(func() {
		cancel()
	})
	return ctx
}

// SetupTestContextWithValues sets up a test context with values and cleanup.
func SetupTestContextWithValues(t *testing.T, timeout time.Duration, values map[string]interface{}) context.Context {
	builder := NewTestContextBuilder().WithTimeout(timeout).WithValues(values)
	ctx, cancel := builder.Build()
	t.Cleanup(func() {
		cancel()
	})
	return ctx
}

// TimeoutConfig holds timeout configurations for different test scenarios.
type TimeoutConfig struct {
	Short    time.Duration
	Medium   time.Duration
	Long     time.Duration
	Database time.Duration
	Network  time.Duration
}

// DefaultTimeoutConfig returns default timeout configurations.
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Short:    5 * time.Second,
		Medium:   30 * time.Second,
		Long:     2 * time.Minute,
		Database: 10 * time.Second,
		Network:  15 * time.Second,
	}
}

// ContextFactory creates contexts with different timeouts.
type ContextFactory struct {
	config TimeoutConfig
}

// NewContextFactory creates a new context factory.
func NewContextFactory(config TimeoutConfig) *ContextFactory {
	if config.Short == 0 {
		config = DefaultTimeoutConfig()
	}
	return &ContextFactory{config: config}
}

// Short creates a short-lived context.
func (cf *ContextFactory) Short() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), cf.config.Short)
}

// Medium creates a medium-lived context.
func (cf *ContextFactory) Medium() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), cf.config.Medium)
}

// Long creates a long-lived context.
func (cf *ContextFactory) Long() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), cf.config.Long)
}

// Database creates a context for database operations.
func (cf *ContextFactory) Database() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), cf.config.Database)
}

// Network creates a context for network operations.
func (cf *ContextFactory) Network() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), cf.config.Network)
}
