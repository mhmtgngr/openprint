// Package main is the entry point for the OpenPrint Registry Service.
// This service manages printer and agent registration and heartbeat monitoring.
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
	"github.com/openprint/openprint/internal/auth/jwt"
	"github.com/openprint/openprint/internal/shared/middleware"
	"github.com/openprint/openprint/internal/shared/telemetry"
	"github.com/openprint/openprint/internal/shared/telemetry/prometheus"
	"github.com/openprint/openprint/services/registry-service/handler"
	"github.com/openprint/openprint/services/registry-service/repository"
)

// Config holds service configuration.
type Config struct {
	ServerAddr       string
	MetricsPort      int
	DatabaseURL      string
	JWTSecret        string
	JaegerEndpoint   string
	ServiceName      string
	HeartbeatTimeout time.Duration
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

	// Wrap database with metrics collector
	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	prometheus.WrapPgxPool(db, registry, prometheus.DBConfig{
		ServiceName: cfg.ServiceName,
		DBName:      "openprint",
		DBSystem:    prometheus.DBSystemPostgreSQL,
	})

	// Initialize telemetry (tracing)
	shutdown, err := telemetry.InitTracer(cfg.ServiceName, "1.0.0", cfg.JaegerEndpoint)
	if err != nil {
		log.Printf("Warning: failed to initialize tracer: %v", err)
	}
	if shutdown != nil {
		defer shutdown(ctx)
	}

	// Initialize repositories
	agentRepo := repository.NewAgentRepository(db)
	printerRepo := repository.NewPrinterRepository(db)
	mappingRepo := repository.NewUserPrinterMappingRepository(db)

	// Create handlers with metrics
	h := handler.New(handler.Config{
		AgentRepo:        agentRepo,
		PrinterRepo:      printerRepo,
		HeartbeatTimeout: cfg.HeartbeatTimeout,
		Metrics:          metrics,
		ServiceName:      cfg.ServiceName,
	})

	// Create user-printer mapping handler
	mappingHandler := handler.NewUserPrinterMappingHandler(mappingRepo)

	// Start heartbeat monitor in background
	go h.HeartbeatMonitor(ctx)

	// Create JWT manager for authentication
	jwtCfg, err := jwt.DefaultConfig(cfg.JWTSecret)
	if err != nil {
		log.Fatalf("Failed to create JWT config: %v", err)
	}
	jwtManager, err := jwt.NewManager(jwtCfg)
	if err != nil {
		log.Fatalf("Failed to create JWT manager: %v", err)
	}

	// Setup HTTP server with middleware
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/agents/register", h.RegisterAgent)
	mux.HandleFunc("/agents/", h.AgentHandler)
	mux.HandleFunc("/agents", h.ListAgents)
	mux.HandleFunc("/printers/register", h.RegisterPrinter)
	mux.HandleFunc("/printers/", h.PrinterHandler)
	mux.HandleFunc("/printers", h.ListPrinters)
	mux.HandleFunc("/user-printer-mappings/", mappingHandler.MappingHandler)
	mux.HandleFunc("/user-printer-mappings", mappingHandler.MappingsHandler)

	// Build middleware chain: logging -> recovery -> auth -> metrics -> telemetry -> security headers -> handler
	// For registry service, we also support API key authentication for agents
	middlewareChain := middleware.Chain(
		middleware.LoggingMiddleware(log.New(os.Stdout, "[REGISTRY] ", log.LstdFlags)),
		middleware.RecoveryMiddleware(log.New(os.Stdout, "[REGISTRY] ", log.LstdFlags)),
		middleware.AuthMiddleware(middleware.JWTConfig{
			SecretKey:  cfg.JWTSecret,
			JWTManager: jwtManager,
			SkipPaths:  []string{"/health", "/agents", "/printers", "/user-printer-mappings"}, // Allow agent and printer endpoints
		}),
		middleware.MetricsMiddleware(middleware.MetricsMiddlewareConfig{
			Registry:           registry,
			ServiceName:        cfg.ServiceName,
			SkipPaths:          []string{"/health"},
			ExcludeStaticFiles: true,
		}),
		telemetry.HTTPMiddleware(cfg.ServiceName),
		middleware.SecurityHeadersMiddleware(),
		middleware.CORSMiddleware([]string{"*"}, []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}, []string{"Content-Type", "Authorization", "X-API-Key"}),
	)

	wrappedMux := middlewareChain(mux)

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

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

func loadConfig() *Config {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	return &Config{
		ServerAddr:       getEnv("SERVER_ADDR", ":8002"),
		MetricsPort:      getEnvInt("METRICS_PORT", 0), // 0 = use default
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://openprint:openprint@localhost:5432/openprint"),
		JWTSecret:        jwtSecret,
		JaegerEndpoint:   getEnv("JAEGER_ENDPOINT", ""),
		ServiceName:      getEnv("SERVICE_NAME", "registry-service"),
		HeartbeatTimeout: 5 * time.Minute,
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
		w.Write([]byte(`{"status":"healthy","service":"registry-service"}`))
	}
}
