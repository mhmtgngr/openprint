// Package main is the entry point for the OpenPrint Compliance Service.
// This service handles FedRAMP, HIPAA, GDPR, and SOC2 compliance tracking and reporting.
package main

import (
	"context"
	"encoding/json"
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
)

// ServerConfig holds service configuration.
type ServerConfig struct {
	ServerAddr     string
	DatabaseURL    string
	JWTSecret      string
	JaegerEndpoint string
	ServiceName    string
}

func main() {
	cfg := loadServerConfig()

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

	// Initialize compliance service
	compSvc := New(Config{
		DB: db,
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

	// Health check
	mux.HandleFunc("/health", healthHandler)

	// Compliance endpoints
	mux.HandleFunc("/api/v1/controls/", controlsHandler(compSvc))
	mux.HandleFunc("/api/v1/controls", listControlsHandler(compSvc))
	mux.HandleFunc("/api/v1/controls/", updateControlStatusHandler(compSvc))
	mux.HandleFunc("/api/v1/audit", auditLogHandler(compSvc))
	mux.HandleFunc("/api/v1/audit/export", exportAuditLogsHandler(compSvc))
	mux.HandleFunc("/api/v1/reports/generate", generateReportHandler(compSvc))
	mux.HandleFunc("/api/v1/reports/summary", summaryHandler(compSvc))
	mux.HandleFunc("/api/v1/breaches", breachesHandler(compSvc))
	mux.HandleFunc("/api/v1/reviews/pending", pendingReviewsHandler(compSvc))

	// Build middleware chain
	middlewareChain := middleware.Chain(
		middleware.LoggingMiddleware(log.New(os.Stdout, "[COMPLIANCE] ", log.LstdFlags)),
		middleware.RecoveryMiddleware(log.New(os.Stdout, "[COMPLIANCE] ", log.LstdFlags)),
		middleware.AuthMiddleware(middleware.JWTConfig{
			SecretKey:  cfg.JWTSecret,
			JWTManager: jwtManager,
			SkipPaths:  []string{"/health"},
		}),
		telemetry.HTTPMiddleware(cfg.ServiceName),
		middleware.SecurityHeadersMiddleware(),
		middleware.CORSMiddleware(
			[]string{"*"},
			[]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			[]string{"Content-Type", "Authorization"},
		),
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

func loadServerConfig() *ServerConfig {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	return &ServerConfig{
		ServerAddr:     getEnv("SERVER_ADDR", ":8006"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://openprint:openprint@localhost:5432/openprint"),
		JWTSecret:      jwtSecret,
		JaegerEndpoint: getEnv("JAEGER_ENDPOINT", ""),
		ServiceName:    getEnv("SERVICE_NAME", "compliance-service"),
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
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "compliance-service",
	})
}

func controlsHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = r.Context()
		controlID := extractIDFromPath(r.URL.Path, "/api/v1/controls/")

		// In a real implementation, you'd need to expose the DB or refactor

		respondJSON(w, http.StatusOK, map[string]string{
			"control_id": controlID,
			"message":    "Use the repository to get control details",
		})
	}
}

func listControlsHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse query parameters
		framework := r.URL.Query().Get("framework")
		status := r.URL.Query().Get("status")

		// TODO: Implement with actual service call
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"framework": framework,
			"status":    status,
			"controls":  []interface{}{},
		})
	}
}

func updateControlStatusHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// TODO: Implement status update
		respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
	}
}

func auditLogHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			// Create audit event
			respondJSON(w, http.StatusCreated, map[string]string{"id": "audit-event-id"})
		} else if r.Method == http.MethodGet {
			// Query audit events
			respondJSON(w, http.StatusOK, map[string]interface{}{"events": []interface{}{}})
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func exportAuditLogsHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		format := r.URL.Query().Get("format")
		if format == "" {
			format = "json"
		}

		// TODO: Implement export
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}
}

func generateReportHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Framework    string `json:"framework"`
			PeriodStart  string `json:"period_start"`
			PeriodEnd    string `json:"period_end"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		// TODO: Generate report
		respondJSON(w, http.StatusOK, map[string]string{
			"report_id": "generated-report-id",
		})
	}
}

func summaryHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// TODO: Get summary
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"fedramp": "compliant",
			"hipaa":   "compliant",
			"gdpr":    "pending",
			"soc2":    "compliant",
		})
	}
}

func breachesHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			// Record breach
			respondJSON(w, http.StatusCreated, map[string]string{"id": "breach-id"})
		} else if r.Method == http.MethodGet {
			// List breaches
			respondJSON(w, http.StatusOK, map[string]interface{}{"breaches": []interface{}{}})
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func pendingReviewsHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// TODO: Get pending reviews
		respondJSON(w, http.StatusOK, map[string]interface{}{"controls": []interface{}{}})
	}
}

func extractIDFromPath(path, prefix string) string {
	if len(path) > len(prefix) {
		return path[len(prefix):]
	}
	return ""
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
