package main

import (
	"context"
	"time"
)

// Default timeout durations for database operations
var (
	defaultQueryTimeout = 30 * time.Second
	defaultDBTimeout    = 60 * time.Second
)

// WithQueryTimeout creates a context with the default query timeout
func WithQueryTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, defaultQueryTimeout)
}

// WithQueryTimeout creates a context with a custom query timeout
func WithQueryTimeoutCustom(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, timeout)
}

// WithDBTimeout creates a context with the default database timeout
func WithDBTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, defaultDBTimeout)
}

// WithDBTimeoutCustom creates a context with a custom database timeout
func WithDBTimeoutCustom(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, timeout)
}

// GetDefaultQueryTimeout returns the default query timeout
func GetDefaultQueryTimeout() time.Duration {
	return defaultQueryTimeout
}

// GetDefaultDBTimeout returns the default database timeout
func GetDefaultDBTimeout() time.Duration {
	return defaultDBTimeout
}

// SetDefaultQueryTimeout sets the default query timeout (use with caution)
func SetDefaultQueryTimeout(timeout time.Duration) {
	defaultQueryTimeout = timeout
}

// SetDefaultDBTimeout sets the default database timeout (use with caution)
func SetDefaultDBTimeout(timeout time.Duration) {
	defaultDBTimeout = timeout
}
