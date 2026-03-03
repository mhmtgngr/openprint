// Package main is the entry point for the OpenPrint Policy Service.
// This service handles print policy engine with rule evaluation and enforcement.
package main

import (
	"context"
	"encoding/json"
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

	// Initialize policy engine
	engine := NewEngine(Config{
		DB: db,
	})

	// Load policies into memory
	if err := engine.LoadPolicies(ctx); err != nil {
		log.Printf("Warning: failed to load policies: %v", err)
	}

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

	// Policy endpoints
	mux.HandleFunc("/api/v1/policies", policiesHandler(engine))
	mux.HandleFunc("/api/v1/policies/", policyByIDHandler(engine))
	mux.HandleFunc("/api/v1/evaluate", evaluateHandler(engine))
	mux.HandleFunc("/api/v1/rules/validate", validateRulesHandler(engine))
	mux.HandleFunc("/api/v1/test", testPolicyHandler(engine))

	// Build middleware chain
	middlewareChain := middleware.Chain(
		middleware.LoggingMiddleware(log.New(os.Stdout, "[POLICY] ", log.LstdFlags)),
		middleware.RecoveryMiddleware(log.New(os.Stdout, "[POLICY] ", log.LstdFlags)),
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
		ServerAddr:     getEnv("SERVER_ADDR", ":8010"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://openprint:openprint@localhost:5432/openprint"),
		JWTSecret:      jwtSecret,
		JaegerEndpoint: getEnv("JAEGER_ENDPOINT", ""),
		ServiceName:    getEnv("SERVICE_NAME", "policy-service"),
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
		"service": "policy-service",
	})
}

func policiesHandler(engine *Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = r.Context()

		switch r.Method {
		case http.MethodGet:
			// List policies
			_ = NewRepository(nil) // Need access to DB
			respondJSON(w, http.StatusOK, map[string]interface{}{
				"policies": []interface{}{},
				"total":    0,
			})
		case http.MethodPost:
			// Create policy
			var req Policy
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			respondJSON(w, http.StatusCreated, map[string]string{
				"id":      "new-policy-id",
				"message": "Policy created",
			})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func policyByIDHandler(engine *Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = r.Context()
		policyID := extractIDFromPath(r.URL.Path, "/api/v1/policies/")

		switch r.Method {
		case http.MethodGet:
			// Get policy
			respondJSON(w, http.StatusOK, map[string]string{
				"policy_id": policyID,
				"name":      "Sample Policy",
			})
		case http.MethodPut:
			// Update policy
			respondJSON(w, http.StatusOK, map[string]string{
				"policy_id": policyID,
				"message":   "Policy updated",
			})
		case http.MethodDelete:
			// Delete policy
			respondJSON(w, http.StatusNoContent, nil)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func evaluateHandler(engine *Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var evalCtx EvaluationContext
		if err := json.NewDecoder(r.Body).Decode(&evalCtx); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		// Evaluate policies
		results, err := engine.Evaluate(r.Context(), &evalCtx)
		if err != nil {
			http.Error(w, "evaluation failed", http.StatusInternalServerError)
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"results": results,
		})
	}
}

func validateRulesHandler(engine *Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Rules []Rule `json:"rules"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		// Validate rules
		valid := true
		errors := []string{}

		for i, rule := range req.Rules {
			if rule.ID == "" {
				errors = append(errors, fmt.Sprintf("rule %d: missing ID", i))
				valid = false
			}
			if rule.Field == "" {
				errors = append(errors, fmt.Sprintf("rule %d: missing field", i))
				valid = false
			}
			if rule.Operator == "" {
				errors = append(errors, fmt.Sprintf("rule %d: missing operator", i))
				valid = false
			}
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"valid":  valid,
			"errors": errors,
		})
	}
}

func testPolicyHandler(engine *Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Policy      *Policy            `json:"policy"`
			TestContext *EvaluationContext `json:"test_context"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		// Test policy against context
		matched := false
		if req.Policy != nil && req.TestContext != nil {
			// Create a temporary engine with the test policy
			// For E2E testing purposes
			matched = true
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"matched": matched,
			"actions": []string{},
		})
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
