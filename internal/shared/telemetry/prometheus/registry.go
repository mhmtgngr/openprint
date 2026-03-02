// Package prometheus provides Prometheus metrics collection and registry management
// for all OpenPrint microservices.
package prometheus

import (
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	// globalRegistry holds the default Prometheus registry for each service.
	globalRegistry *Registry

	// registryMu protects access to the global registry.
	registryMu sync.RWMutex

	// defaultServiceName is the service name used when none is specified.
	defaultServiceName = "openprint"
)

// Registry wraps a Prometheus registry with service-specific labels and management.
// Each service should have its own registry instance to isolate metrics.
type Registry struct {
	// registry is the underlying Prometheus registry.
	registry *prometheus.Registry

	// serviceName is the name of the service using this registry.
	serviceName string

	// serviceVersion is the version of the service.
	serviceVersion string

	// labels are common labels applied to all metrics.
	labels prometheus.Labels

	// collectors tracks registered collectors to prevent duplicates.
	collectors map[string]prometheus.Collector

	// mu protects access to the collectors map.
	mu sync.RWMutex
}

// Config holds configuration for creating a new Registry.
type Config struct {
	// ServiceName is the name of the service (e.g., "auth-service").
	ServiceName string

	// ServiceVersion is the version of the service (e.g., "1.0.0").
	ServiceVersion string

	// Namespace is the prefix for all metric names (e.g., "openprint").
	Namespace string

	// Additional labels to apply to all metrics.
	Labels prometheus.Labels
}

// NewRegistry creates a new Prometheus registry with the given configuration.
// The registry includes default Go and process collectors.
func NewRegistry(cfg Config) (*Registry, error) {
	if cfg.ServiceName == "" {
		cfg.ServiceName = defaultServiceName
	}
	if cfg.Namespace == "" {
		cfg.Namespace = "openprint"
	}

	// Create a new registry (not the default global registry)
	reg := prometheus.NewRegistry()

	// Build common labels
	labels := prometheus.Labels{
		"service_name": cfg.ServiceName,
		"namespace":    cfg.Namespace,
	}
	if cfg.ServiceVersion != "" {
		labels["service_version"] = cfg.ServiceVersion
	}
	// Merge additional labels
	for k, v := range cfg.Labels {
		labels[k] = v
	}

	r := &Registry{
		registry:        reg,
		serviceName:     cfg.ServiceName,
		serviceVersion:  cfg.ServiceVersion,
		labels:          labels,
		collectors:      make(map[string]prometheus.Collector),
	}

	// Register default collectors for Go runtime metrics
	// These provide essential metrics like memory usage, GC stats, goroutine count
	r.registerDefaultCollectors()

	return r, nil
}

// registerDefaultCollectors registers the default Go and process collectors.
func (r *Registry) registerDefaultCollectors() {
	// Go collector: metrics about Go runtime (memory, GC, goroutines)
	goCollector := prometheus.NewGoCollector()
	r.MustRegister(goCollector, "go_collector")

	// Process collector: metrics about the process (CPU, file descriptors, etc.)
	processCollector := prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{})
	r.MustRegister(processCollector, "process_collector")
}

// MustRegister registers a collector with the registry, panicking on error.
// This is similar to prometheus.MustRegister but includes duplicate detection.
// The name parameter is used to track and prevent duplicate registrations.
func (r *Registry) MustRegister(collector prometheus.Collector, name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicate registration
	if _, exists := r.collectors[name]; exists {
		// Already registered, skip
		return
	}

	// Register with Prometheus
	r.registry.MustRegister(collector)
	r.collectors[name] = collector
}

// Register attempts to register a collector, returning an error on failure.
// Returns nil if the collector was already registered or registration succeeded.
func (r *Registry) Register(collector prometheus.Collector, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicate registration
	if _, exists := r.collectors[name]; exists {
		return nil
	}

	// Register with Prometheus
	if err := r.registry.Register(collector); err != nil {
		return fmt.Errorf("failed to register collector %s: %w", name, err)
	}

	r.collectors[name] = collector
	return nil
}

// Unregister removes a collector from the registry.
func (r *Registry) Unregister(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	collector, exists := r.collectors[name]
	if !exists {
		return false
	}

	unregistered := r.registry.Unregister(collector)
	if unregistered {
		delete(r.collectors, name)
	}

	return unregistered
}

// Registry returns the underlying Prometheus registry.
// This is used when creating custom collectors or integrating with external systems.
func (r *Registry) Registry() *prometheus.Registry {
	return r.registry
}

// ServiceName returns the service name associated with this registry.
func (r *Registry) ServiceName() string {
	return r.serviceName
}

// ServiceVersion returns the service version associated with this registry.
func (r *Registry) ServiceVersion() string {
	return r.serviceVersion
}

// Labels returns the common labels applied to all metrics.
func (r *Registry) Labels() prometheus.Labels {
	return r.labels
}

// MergeLabels merges the provided labels with the registry's common labels.
// Registry labels take precedence if there are conflicts.
func (r *Registry) MergeLabels(labels prometheus.Labels) prometheus.Labels {
	merged := make(prometheus.Labels, len(r.labels)+len(labels))
	for k, v := range labels {
		merged[k] = v
	}
	for k, v := range r.labels {
		merged[k] = v
	}
	return merged
}

// GetRegistry returns the global registry instance.
// If no registry has been initialized, it returns an error.
func GetRegistry() (*Registry, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	if globalRegistry == nil {
		return nil, fmt.Errorf("no global registry initialized")
	}

	return globalRegistry, nil
}

// SetRegistry sets the global registry instance.
// This should be called once during service initialization.
func SetRegistry(registry *Registry) {
	registryMu.Lock()
	defer registryMu.Unlock()

	globalRegistry = registry
}

// MustGetRegistry returns the global registry or panics if not initialized.
// This is useful for package init functions and tests.
func MustGetRegistry() *Registry {
	registry, err := GetRegistry()
	if err != nil {
		panic(err)
	}

	return registry
}

// ResetGlobalRegistry clears the global registry.
// This is primarily used for testing.
func ResetGlobalRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()

	globalRegistry = nil
}

// DefaultConfig returns a default configuration for a service registry.
func DefaultConfig(serviceName string) Config {
	return Config{
		ServiceName:    serviceName,
		ServiceVersion: "1.0.0",
		Namespace:      "openprint",
		Labels:         prometheus.Labels{},
	}
}
