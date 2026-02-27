// Package main is the entry point for the OpenPrint Registry Service.
// This service manages printer and agent registration and heartbeat monitoring.
package main

import (
	"context"
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
	"github.com/openprint/openprint/services/registry-service/handler"
	"github.com/openprint/openprint/services/registry-service/repository"
)

// Config holds service configuration.
type Config struct {
	ServerAddr       string
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

	// Initialize telemetry
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

	// Initialize repositories
	agentRepo := repository.NewAgentRepository(db)
	printerRepo := repository.NewPrinterRepository(db)

	// Create handlers
	h := handler.New(handler.Config{
		AgentRepo:        agentRepo,
		PrinterRepo:      printerRepo,
		HeartbeatTimeout: cfg.HeartbeatTimeout,
	})

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

	// Build middleware chain: logging -> recovery -> auth -> telemetry -> security headers -> handler
	// For registry service, we also support API key authentication for agents
	middlewareChain := middleware.Chain(
		middleware.LoggingMiddleware(log.New(os.Stdout, "[REGISTRY] ", log.LstdFlags)),
		middleware.RecoveryMiddleware(log.New(os.Stdout, "[REGISTRY] ", log.LstdFlags)),
		middleware.AuthMiddleware(middleware.JWTConfig{
			SecretKey:  cfg.JWTSecret,
			JWTManager: jwtManager,
			SkipPaths:  []string{"/health", "/agents/register", "/printers/register"}, // Allow agent/printer registration with API key
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
		log.Printf("%s listening on %s", cfg.ServiceName, cfg.ServerAddr)
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
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://openprint:openprint@localhost:5432/openprint"),
		JWTSecret:        jwtSecret,
		JaegerEndpoint:   getEnv("JAEGER_ENDPOINT", ""),
		ServiceName:      getEnv("SERVICE_NAME", "registry-service"),
		HeartbeatTimeout: 5 * time.Minute,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","service":"registry-service"}`))
}
