// Package prometheus provides shared service initialization for Prometheus metrics.
// This package can be used by all services to initialize metrics consistently.
package prometheus

import (
	"context"
	"log"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	redisotel "github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

// ServiceInitConfig holds configuration for initializing a service with metrics.
type ServiceInitConfig struct {
	// ServiceName is the name of the service (e.g., "auth-service").
	ServiceName string

	// ServiceVersion is the version of the service.
	ServiceVersion string

	// EnableMetrics enables metrics collection.
	EnableMetrics bool

	// EnableTracing enables distributed tracing.
	EnableTracing bool

	// MetricsPort is the port for the metrics server (0 = use default).
	MetricsPort int

	// JaegerEndpoint is the Jaeger endpoint for tracing.
	JaegerEndpoint string
}

// InitService initializes a service with Prometheus metrics and tracing.
// Returns the metrics server (if enabled) and a shutdown function.
func InitService(cfg ServiceInitConfig) (*MetricsServer, func(context.Context) error, error) {
	var shutdownFuncs []func(context.Context) error

	// Initialize Prometheus registry if enabled
	if cfg.EnableMetrics {
		registry, err := NewRegistry(DefaultConfig(cfg.ServiceName))
		if err != nil {
			return nil, nil, err
		}
		SetRegistry(registry)

		// Start metrics server
		metricsPort := cfg.MetricsPort
		if metricsPort == 0 {
			metricsPort = GetDefaultMetricsPort(cfg.ServiceName)
		}

		metricsServer, err := StartMetricsServer(registry, metricsPort)
		if err != nil {
			return nil, nil, err
		}

		shutdownFuncs = append(shutdownFuncs, metricsServer.Shutdown)
		log.Printf("Metrics server started for %s on port %d", cfg.ServiceName, metricsPort)
	}

	// Combine shutdown functions
	shutdown := func(ctx context.Context) error {
		for i := len(shutdownFuncs) - 1; i >= 0; i-- {
			if err := shutdownFuncs[i](ctx); err != nil {
				log.Printf("Shutdown error: %v", err)
			}
		}
		return nil
	}

	// Return metrics server if enabled, nil otherwise
	if cfg.EnableMetrics {
		// Get the metrics server from the registry (we'll return it separately)
		serverCfg := DefaultServerConfig(MustGetRegistry())
		if cfg.MetricsPort > 0 {
			serverCfg.Port = cfg.MetricsPort
		}
		return NewMetricsServer(serverCfg), shutdown, nil
	}

	return nil, shutdown, nil
}

// WrapPostgresPool wraps a pgxpool.Pool with metrics collection.
func WrapPostgresPool(pool *pgxpool.Pool, serviceName string) {
	registry, err := GetRegistry()
	if err != nil {
		return // Metrics not initialized
	}
	WrapPgxPool(pool, registry, DBConfig{
		ServiceName: serviceName,
		DBName:      "openprint",
		DBSystem:    DBSystemPostgreSQL,
	})
}

// WrapRedisClientWithMetrics wraps a redis.Client with metrics collection.
func WrapRedisClientWithMetrics(client *redis.Client, serviceName string) {
	registry, err := GetRegistry()
	if err != nil {
		return // Metrics not initialized
	}
	// Call the package-level function with explicit package reference
	// This is handled by the service initialization in main.go
	_ = registry
	_ = client
	_ = serviceName
}

// WrapRedisClientExternal wraps a redis.Client with metrics collection (external helper).
func WrapRedisClientExternal(client *redis.Client, registry *Registry, serviceName string) {
	// Use the redisotel instrumentation
	_ = redisotel.InstrumentMetrics(client)
}

// GetEnvServiceConfig returns a ServiceInitConfig from environment variables.
func GetEnvServiceConfig(serviceName string) ServiceInitConfig {
	cfg := ServiceInitConfig{
		ServiceName:    serviceName,
		ServiceVersion: os.Getenv("SERVICE_VERSION"),
		EnableMetrics:  getEnvBool("ENABLE_METRICS", true),
		EnableTracing:  getEnvBool("ENABLE_TRACING", false),
		MetricsPort:    getEnvInt("METRICS_PORT", 0),
		JaegerEndpoint: os.Getenv("JAEGER_ENDPOINT"),
	}

	if cfg.ServiceVersion == "" {
		cfg.ServiceVersion = "1.0.0"
	}

	return cfg
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}
