// Package main is the entry point for the OpenPrint Notification Service.
// This service handles real-time WebSocket notifications for print job status updates.
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openprint/openprint/internal/shared/telemetry"
	"github.com/openprint/openprint/internal/shared/telemetry/prometheus"
	"github.com/openprint/openprint/services/notification-service/websocket"
)

// Config holds service configuration.
type Config struct {
	ServerAddr     string
	MetricsPort    int
	DatabaseURL    string
	RedisURL       string
	JaegerEndpoint string
	ServiceName    string
	PingInterval   time.Duration
	PongTimeout    time.Duration
}

func main() {
	cfg := loadConfig()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Prometheus metrics registry
	registry, err := prometheus.NewRegistry(prometheus.DefaultConfig(cfg.ServiceName))
	if err != nil {
		log.Fatalf("Failed to create Prometheus registry: %v", err)
	}
	prometheus.SetRegistry(registry)
	metrics := prometheus.NewMetrics(registry)

	// Start metrics server on dedicated port
	metricsPort := cfg.MetricsPort
	if metricsPort == 0 {
		metricsPort = prometheus.GetDefaultMetricsPort(cfg.ServiceName)
	}
	metricsServer, err := prometheus.StartMetricsServer(registry, metricsPort)
	if err != nil {
		log.Fatalf("Failed to start metrics server: %v", err)
	}
	defer func() {
		if err := metricsServer.Shutdown(ctx); err != nil {
			log.Printf("Metrics server shutdown error: %v", err)
		}
	}()

	// Initialize telemetry (tracing)
	shutdown, err := telemetry.InitTracer(cfg.ServiceName, "1.0.0", cfg.JaegerEndpoint)
	if err != nil {
		log.Printf("Warning: failed to initialize tracer: %v", err)
	}
	if shutdown != nil {
		defer shutdown(ctx)
	}

	// Connect to PostgreSQL
	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Wrap database with metrics collector
	prometheus.WrapPgxPool(db, registry, prometheus.DBConfig{
		ServiceName: cfg.ServiceName,
		DBName:      "openprint",
		DBSystem:    prometheus.DBSystemPostgreSQL,
	})

	// Initialize WebSocket hub with metrics
	hub := websocket.NewHub(websocket.Config{
		PingInterval: cfg.PingInterval,
		PongTimeout:  cfg.PongTimeout,
		Metrics:      metrics,
		ServiceName:  cfg.ServiceName,
	})

	// Start hub
	go hub.Run(ctx)

	// Create handlers
	h := websocket.NewHandler(websocket.HandlerConfig{
		Hub:     hub,
		DB:      db,
		Metrics: metrics,
	})

	// Setup HTTP server with middleware
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/ws", h.ServeWS)
	mux.HandleFunc("/broadcast", h.BroadcastHandler)
	mux.HandleFunc("/connections", h.ConnectionsHandler)

	// Apply telemetry and metrics middleware
	wrappedMux := telemetry.HTTPMiddleware(cfg.ServiceName)(mux)

	server := &http.Server{
		Addr:         cfg.ServerAddr,
		Handler:      wrappedMux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}

	// Start server in goroutine
	go func() {
		log.Printf("%s listening on %s (metrics on :%d)", cfg.ServiceName, cfg.ServerAddr, metricsPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	log.Println("Shutting down server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	hub.Shutdown()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

func loadConfig() *Config {
	return &Config{
		ServerAddr:     getEnv("SERVER_ADDR", ":8005"),
		MetricsPort:    getEnvInt("METRICS_PORT", 0), // 0 = use default
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://openprint:openprint@localhost:5432/openprint"),
		RedisURL:       getEnv("REDIS_URL", "redis://localhost:6379"),
		JaegerEndpoint: getEnv("JAEGER_ENDPOINT", ""),
		ServiceName:    getEnv("SERVICE_NAME", "notification-service"),
		PingInterval:   30 * time.Second,
		PongTimeout:    60 * time.Second,
	}
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intVal int
		if _, err := fmt.Sscanf(value, "%d", &intVal); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodGet {
		w.Write([]byte(`{"status":"healthy","service":"notification-service"}`))
	}
}
