// Package main is the entry point for the OpenPrint Storage Service.
// This service handles document storage and retrieval for print jobs.
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
	"github.com/openprint/openprint/services/storage-service/handler"
	"github.com/openprint/openprint/services/storage-service/storage"
)

// Config holds service configuration.
type Config struct {
	ServerAddr       string
	MetricsPort      int
	DatabaseURL      string
	S3Endpoint       string
	S3Bucket         string
	S3AccessKey      string
	S3SecretKey      string
	S3Region         string
	EncryptionKey    string
	JWTSecret        string
	JaegerEndpoint   string
	ServiceName      string
	MaxUploadSize    int64
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

	// Initialize storage backend
	var backend storage.Backend
	var storageBackend string
	if cfg.S3Endpoint != "" {
		backend, err = storage.NewS3Backend(storage.S3Config{
			Endpoint:  cfg.S3Endpoint,
			Bucket:    cfg.S3Bucket,
			AccessKey: cfg.S3AccessKey,
			SecretKey: cfg.S3SecretKey,
			Region:    cfg.S3Region,
		})
		storageBackend = "s3"
	} else {
		// Fall back to local filesystem storage
		backend, err = storage.NewLocalStorage("/tmp/openprint/storage")
		storageBackend = "local"
	}

	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// Wrap with encryption if key is provided
	if cfg.EncryptionKey != "" {
		backend, err = storage.NewEncryptedBackend(backend, cfg.EncryptionKey)
		if err != nil {
			log.Fatalf("Failed to initialize encryption: %v", err)
		}
	}

	// Create handlers with metrics
	h := handler.New(handler.Config{
		Backend:       backend,
		DB:            db,
		MaxUploadSize: cfg.MaxUploadSize,
		Metrics:       metrics,
		ServiceName:   cfg.ServiceName,
		StorageBackend: storageBackend,
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
	mux.HandleFunc("/documents", h.DocumentsHandler)
	mux.HandleFunc("/documents/", h.DocumentHandler)
	mux.HandleFunc("/upload", h.UploadHandler)
	mux.HandleFunc("/download/", h.DownloadHandler)

	// Build middleware chain: logging -> recovery -> auth -> metrics -> telemetry -> security headers -> handler
	middlewareChain := middleware.Chain(
		middleware.LoggingMiddleware(log.New(os.Stdout, "[STORAGE] ", log.LstdFlags)),
		middleware.RecoveryMiddleware(log.New(os.Stdout, "[STORAGE] ", log.LstdFlags)),
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
		ReadTimeout:  5 * time.Minute, // Allow large uploads
		WriteTimeout: 5 * time.Minute,
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
		ServerAddr:     getEnv("SERVER_ADDR", ":8004"),
		MetricsPort:    getEnvInt("METRICS_PORT", 0), // 0 = use default
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://openprint:openprint@localhost:5432/openprint"),
		S3Endpoint:     getEnv("S3_ENDPOINT", ""),
		S3Bucket:       getEnv("S3_BUCKET", "openprint-documents"),
		S3AccessKey:    getEnv("S3_ACCESS_KEY", ""),
		S3SecretKey:    getEnv("S3_SECRET_KEY", ""),
		S3Region:       getEnv("S3_REGION", "us-east-1"),
		EncryptionKey:  getEnv("ENCRYPTION_KEY", ""),
		JWTSecret:      jwtSecret,
		JaegerEndpoint: getEnv("JAEGER_ENDPOINT", ""),
		ServiceName:    getEnv("SERVICE_NAME", "storage-service"),
		MaxUploadSize:  int64(getEnvInt("MAX_UPLOAD_MB", 100)) * 1024 * 1024,
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
		w.Write([]byte(`{"status":"healthy","service":"storage-service"}`))
	}
}
