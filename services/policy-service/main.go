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
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openprint/openprint/internal/auth/jwt"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
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
		ctx := r.Context()
		repo := NewRepository(engine.db)

		switch r.Method {
		case http.MethodGet:
			// Parse query parameters for filtering
			q := r.URL.Query()
			filter := PolicyFilter{
				Limit:  50,
				Offset: 0,
			}
			if t := q.Get("type"); t != "" {
				filter.Type = PolicyType(t)
			}
			if s := q.Get("status"); s != "" {
				filter.Status = PolicyStatus(s)
			}
			if orgID := q.Get("organization_id"); orgID != "" {
				filter.OrganizationID = orgID
			}
			if l := q.Get("limit"); l != "" {
				if parsed, err := parseIntParam(l); err == nil && parsed > 0 {
					filter.Limit = parsed
				}
			}
			if o := q.Get("offset"); o != "" {
				if parsed, err := parseIntParam(o); err == nil && parsed >= 0 {
					filter.Offset = parsed
				}
			}

			policies, total, err := repo.List(ctx, filter)
			if err != nil {
				http.Error(w, "failed to list policies", http.StatusInternalServerError)
				return
			}

			respondJSON(w, http.StatusOK, map[string]interface{}{
				"policies": policies,
				"total":    total,
			})
		case http.MethodPost:
			// Create policy
			var req Policy
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}

			// Validate required fields
			if req.Name == "" {
				http.Error(w, "name is required", http.StatusBadRequest)
				return
			}
			if req.Type == "" {
				http.Error(w, "type is required", http.StatusBadRequest)
				return
			}

			// Set defaults
			if req.Status == "" {
				req.Status = PolicyStatusDraft
			}
			req.CreatedBy = middleware.GetUserID(r)

			if err := repo.Create(ctx, &req); err != nil {
				http.Error(w, "failed to create policy", http.StatusInternalServerError)
				return
			}

			// Reload policies into engine cache
			_ = engine.LoadPolicies(ctx)

			respondJSON(w, http.StatusCreated, req)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func policyByIDHandler(engine *Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		policyID := extractIDFromPath(r.URL.Path, "/api/v1/policies/")
		if policyID == "" {
			http.Error(w, "missing policy ID", http.StatusBadRequest)
			return
		}
		repo := NewRepository(engine.db)

		switch r.Method {
		case http.MethodGet:
			policy, err := repo.Get(ctx, policyID)
			if err != nil {
				if err == apperrors.ErrNotFound {
					http.Error(w, "policy not found", http.StatusNotFound)
					return
				}
				http.Error(w, "failed to get policy", http.StatusInternalServerError)
				return
			}
			respondJSON(w, http.StatusOK, policy)
		case http.MethodPut:
			var req Policy
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}

			// Fetch existing policy to preserve immutable fields
			existing, err := repo.Get(ctx, policyID)
			if err != nil {
				if err == apperrors.ErrNotFound {
					http.Error(w, "policy not found", http.StatusNotFound)
					return
				}
				http.Error(w, "failed to get policy", http.StatusInternalServerError)
				return
			}

			// Apply updates
			existing.Name = req.Name
			existing.Description = req.Description
			existing.Type = req.Type
			existing.Status = req.Status
			existing.Priority = req.Priority
			existing.Rules = req.Rules
			existing.Actions = req.Actions
			existing.Scope = req.Scope
			existing.ModifiedBy = middleware.GetUserID(r)

			if err := repo.Update(ctx, existing); err != nil {
				http.Error(w, "failed to update policy", http.StatusInternalServerError)
				return
			}

			// Reload policies into engine cache
			_ = engine.LoadPolicies(ctx)

			respondJSON(w, http.StatusOK, existing)
		case http.MethodDelete:
			if err := repo.Delete(ctx, policyID); err != nil {
				if err == apperrors.ErrNotFound {
					http.Error(w, "policy not found", http.StatusNotFound)
					return
				}
				http.Error(w, "failed to delete policy", http.StatusInternalServerError)
				return
			}

			// Reload policies into engine cache
			_ = engine.LoadPolicies(ctx)

			w.WriteHeader(http.StatusNoContent)
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
			TestContext *EvaluationContext  `json:"test_context"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.Policy == nil {
			http.Error(w, "policy is required", http.StatusBadRequest)
			return
		}
		if req.TestContext == nil {
			http.Error(w, "test_context is required", http.StatusBadRequest)
			return
		}

		// Evaluate the policy rules against the test context using the engine
		matched, ruleMatches := engine.evaluateRules(req.Policy.Rules, req.TestContext)

		// Collect triggered actions
		var actions []PolicyActionConfig
		if matched {
			actions = req.Policy.Actions
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"matched":      matched,
			"actions":      actions,
			"rule_matches": ruleMatches,
		})
	}
}

func parseIntParam(s string) (int, error) {
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return v, nil
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
