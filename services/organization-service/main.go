// Package main is the entry point for the OpenPrint Organization Service.
// This service handles organization management, permissions, and member management.
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
	_ "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/internal/shared/middleware"
	"github.com/openprint/openprint/internal/shared/telemetry"
	"github.com/openprint/openprint/services/organization-service/handler"
	"github.com/openprint/openprint/services/organization-service/repository"
)

// Config holds service configuration.
type Config struct {
	ServerAddr     string
	DatabaseURL    string
	JaegerEndpoint string
	ServiceName    string
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
	orgRepo := repository.NewOrganizationRepository(db)

	// Create handlers
	h := handler.New(handler.Config{
		OrgRepo: orgRepo,
	})

	// Setup HTTP server with middleware
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/api/v1/organizations", h.ListOrganizations)
	mux.HandleFunc("/api/v1/organizations/", h.OrganizationHandler)

	// Apply security middleware chain
	generalRateLimiter := middleware.RateLimitMiddleware(60, 5*time.Minute)
	securityHeaders := middleware.SecurityHeadersMiddleware()
	corsMiddleware := middleware.CORSMiddleware(
		[]string{"*"}, // Configure appropriately for production
		[]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		[]string{"Content-Type", "Authorization"},
	)

	// Wrap the mux with middleware
	wrappedMux := corsMiddleware(securityHeaders(generalRateLimiter(mux)))

	// Apply telemetry middleware
	wrappedMux = telemetry.HTTPMiddleware(cfg.ServiceName)(wrappedMux)

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
	return &Config{
		ServerAddr:     getEnv("SERVER_ADDR", ":8009"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://openprint:openprint@localhost:5432/openprint"),
		JaegerEndpoint: getEnv("JAEGER_ENDPOINT", ""),
		ServiceName:    getEnv("SERVICE_NAME", "organization-service"),
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
	fmt.Fprintf(w, `{"status":"healthy","service":"%s"}`, "organization-service")
}
