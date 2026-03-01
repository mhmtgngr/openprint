// Package main is the entry point for the OpenPrint Job Service.
// This service manages print job queuing and routing to agents.
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
	"github.com/redis/go-redis/v9"
	"github.com/openprint/openprint/internal/auth/jwt"
	"github.com/openprint/openprint/internal/shared/middleware"
	"github.com/openprint/openprint/internal/shared/telemetry"
	"github.com/openprint/openprint/internal/shared/telemetry/prometheus"
	"github.com/openprint/openprint/services/job-service/handler"
	"github.com/openprint/openprint/services/job-service/processor"
	"github.com/openprint/openprint/services/job-service/repository"
)

// Config holds service configuration.
type Config struct {
	ServerAddr       string
	MetricsPort      int
	DatabaseURL      string
	RedisURL         string
	JWTSecret        string
	JaegerEndpoint   string
	ServiceName      string
	ProcessorWorkers int
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

	// Connect to Redis for job queue
	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to parse Redis URL: %v", err)
	}
	redisClient := redis.NewClient(redisOpts)

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Wrap Redis with metrics collector
	prometheus.WrapRedisClient(redisClient, registry, prometheus.RedisConfig{
		ServiceName: cfg.ServiceName,
		DBName:      "0",
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
	jobRepo := repository.NewJobRepository(db)
	historyRepo := repository.NewJobHistoryRepository(db)

	// Initialize processor with metrics
	jobProcessor := processor.New(processor.Config{
		JobRepo:       jobRepo,
		HistoryRepo:   historyRepo,
		Redis:         redisClient,
		Workers:       cfg.ProcessorWorkers,
		PollInterval:  1 * time.Second,
		Metrics:       metrics,
		ServiceName:   cfg.ServiceName,
	})

	// Start processor in background
	go jobProcessor.Start(ctx)

	// Create handlers with metrics
	h := handler.New(handler.Config{
		JobRepo:     jobRepo,
		HistoryRepo: historyRepo,
		Processor:   jobProcessor,
		Metrics:     metrics,
		ServiceName: cfg.ServiceName,
	})
	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Connect to Redis for job queue
	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to parse Redis URL: %v", err)
	}
	redisClient := redis.NewClient(redisOpts)

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Initialize repositories
	jobRepo := repository.NewJobRepository(db)
	historyRepo := repository.NewJobHistoryRepository(db)

	// Initialize processor
	jobProcessor := processor.New(processor.Config{
		JobRepo:       jobRepo,
		HistoryRepo:   historyRepo,
		Redis:         redisClient,
		Workers:       cfg.ProcessorWorkers,
		PollInterval:  1 * time.Second,
	})

	// Start processor in background
	go jobProcessor.Start(ctx)

	// Create handlers
	h := handler.New(handler.Config{
		JobRepo:     jobRepo,
		HistoryRepo: historyRepo,
		Processor:   jobProcessor,
	})

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
	mux.HandleFunc("/jobs", h.JobsHandler)
	mux.HandleFunc("/jobs/", h.JobHandler)
	mux.HandleFunc("/jobs/status/", h.JobStatusHandler)
	mux.HandleFunc("/history", h.HistoryHandler)
	mux.HandleFunc("/queue/stats", h.QueueStatsHandler)

	// Build middleware chain: logging -> recovery -> auth -> metrics -> telemetry -> security headers -> handler
	middlewareChain := middleware.Chain(
		middleware.LoggingMiddleware(log.New(os.Stdout, "[JOB] ", log.LstdFlags)),
		middleware.RecoveryMiddleware(log.New(os.Stdout, "[JOB] ", log.LstdFlags)),
		middleware.AuthMiddleware(middleware.JWTConfig{
			SecretKey:  cfg.JWTSecret,
			JWTManager: jwtManager,
			SkipPaths:  []string{"/health"},
		}),
		middleware.MetricsMiddleware(middleware.MetricsMiddlewareConfig{
			Registry:           registry,
			ServiceName:        cfg.ServiceName,
			SkipPaths:          []string{"/health"},
			ExcludeStaticFiles: true,
		}),
		telemetry.HTTPMiddleware(cfg.ServiceName),
		middleware.SecurityHeadersMiddleware(),
		middleware.CORSMiddleware([]string{"*"}, []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}, []string{"Content-Type", "Authorization"}),
	)

	wrappedMux := middlewareChain(mux)

	server := &http.Server{
		Addr:         cfg.ServerAddr,
		Handler:      wrappedMux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
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
		ServerAddr:       getEnv("SERVER_ADDR", ":8003"),
		MetricsPort:      getEnvInt("METRICS_PORT", 0), // 0 = use default
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://openprint:openprint@localhost:5432/openprint"),
		RedisURL:         getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:        jwtSecret,
		JaegerEndpoint:   getEnv("JAEGER_ENDPOINT", ""),
		ServiceName:      getEnv("SERVICE_NAME", "job-service"),
		ProcessorWorkers: getEnvInt("PROCESSOR_WORKERS", 10),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var i int
		if _, err := fmt.Sscanf(value, "%d", &i); err == nil {
			return i
		}
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
		w.Write([]byte(`{"status":"healthy","service":"job-service"}`))
	}
}
