// Package main is the entry point for the OpenPrint API Gateway.
// This service acts as the API gateway routing to all microservices (ports 8001-8005).
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

	"github.com/openprint/openprint/internal/auth/jwt"
	"github.com/openprint/openprint/services/gateway/middleware"
)

const (
	// ServicePort is the port the gateway listens on.
	ServicePort = 8000

	// Backend service ports.
	AuthServicePort         = 8001
	RegistryServicePort     = 8002
	JobServicePort          = 8003
	StorageServicePort      = 8004
	NotificationServicePort = 8005
)

func main() {
	cfg := loadConfig()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jwtCfg, err := jwt.DefaultConfig(cfg.JWTSecret)
	if err != nil {
		log.Fatalf("Failed to create JWT config: %v", err)
	}
	jwtManager, err := jwt.NewManager(jwtCfg)
	if err != nil {
		log.Fatalf("Failed to create JWT manager: %v", err)
	}

	auditLogger := middleware.NewAuditLogger(log.New(os.Stdout, "[GATEWAY] ", log.LstdFlags))

	handler := newGatewayHandler(cfg, jwtManager, auditLogger)

	server := &http.Server{
		Addr:         cfg.ServerAddr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}

	go func() {
		log.Printf("API Gateway listening on %s", cfg.ServerAddr)
		log.Printf("Routing to services:")
		log.Printf("  - auth-service    -> %s", cfg.AuthServiceURL)
		log.Printf("  - registry-service -> %s", cfg.RegistryServiceURL)
		log.Printf("  - job-service      -> %s", cfg.JobServiceURL)
		log.Printf("  - storage-service  -> %s", cfg.StorageServiceURL)
		log.Printf("  - notification-service -> %s", cfg.NotificationServiceURL)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down gateway...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
	log.Println("Gateway stopped")
}

// newGatewayHandler builds the full HTTP handler with routes and middleware.
func newGatewayHandler(cfg *Config, jwtManager *jwt.Manager, auditLogger *middleware.AuditLogger) http.Handler {
	mux := http.NewServeMux()

	registerServiceRoutes(mux, cfg, jwtManager, auditLogger)
	mux.HandleFunc("/health", aggregatedHealthHandler(cfg))

	return middleware.Chain(
		middleware.RateLimitMiddleware(&middleware.RateLimiterConfig{
			RequestsPerMinute: cfg.RequestsPerMinute,
			CleanupInterval:   5 * time.Minute,
		}),
		middleware.AuditMiddleware(auditLogger),
		securityHeadersMiddleware(),
		corsMiddleware(),
	)(mux)
}

// NewHandler creates a new gateway handler with the given config.
// This is useful for testing and embedding.
func NewHandler(cfg *Config, jwtManager *jwt.Manager, auditLogger *middleware.AuditLogger) http.Handler {
	return newGatewayHandler(cfg, jwtManager, auditLogger)
}
