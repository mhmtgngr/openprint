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

// policiesHandler handles GET/POST /api/v1/policies
func policiesHandler(engine *Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		switch r.Method {
		case http.MethodGet:
			// Parse query parameters
			policyType := r.URL.Query().Get("type")
			status := r.URL.Query().Get("status")
			orgID := r.URL.Query().Get("organization_id")

			// Parse pagination
			page, _ := strconv.Atoi(r.URL.Query().Get("page"))
			if page < 1 {
				page = 1
			}
			limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
			if limit < 1 || limit > 100 {
				limit = 20
			}
			offset := (page - 1) * limit

			repo := NewRepository(engine.db)
			policies, total, err := repo.List(ctx, PolicyFilter{
				Type:           PolicyType(policyType),
				Status:         PolicyStatus(status),
				OrganizationID: orgID,
				Limit:          limit,
				Offset:         offset,
			})
			if err != nil {
				log.Printf("Error listing policies: %v", err)
				http.Error(w, "failed to list policies", http.StatusInternalServerError)
				return
			}

			// Ensure policies is never nil for JSON encoding
			if policies == nil {
				policies = []*Policy{}
			}

			respondJSON(w, http.StatusOK, map[string]interface{}{
				"policies": policies,
				"page":     page,
				"limit":    limit,
				"total":    total,
			})

		case http.MethodPost:
			var req Policy
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}

			// Validate policy type
			validTypes := map[PolicyType]bool{
				PolicyTypeQuota:      true,
				PolicyTypeAccess:     true,
				PolicyTypeContent:    true,
				PolicyTypeRouting:    true,
				PolicyTypeWatermark:  true,
				PolicyTypeRetention:  true,
				PolicyTypeCostCenter: true,
			}
			if req.Type == "" || !validTypes[req.Type] {
				respondJSON(w, http.StatusBadRequest, map[string]string{
					"error":       "invalid policy type",
					"valid_types": "quota, access, content, routing, watermark, retention, cost_center",
				})
				return
			}

			// Validate status
			validStatuses := map[PolicyStatus]bool{
				PolicyStatusActive:   true,
				PolicyStatusInactive: true,
				PolicyStatusDraft:    true,
			}
			if req.Status == "" {
				req.Status = PolicyStatusDraft
			} else if !validStatuses[req.Status] {
				respondJSON(w, http.StatusBadRequest, map[string]string{
					"error":          "invalid status",
					"valid_statuses": "active, inactive, draft",
				})
				return
			}

			// Set default priority if not provided
			if req.Priority == 0 {
				req.Priority = 50
			}

			// Set default scope if not provided
			if req.Scope.UserIDs == nil {
				req.Scope.UserIDs = []string{}
			}
			if req.Scope.GroupIDs == nil {
				req.Scope.GroupIDs = []string{}
			}
			if req.Scope.PrinterIDs == nil {
				req.Scope.PrinterIDs = []string{}
			}
			if req.Scope.DocumentTypes == nil {
				req.Scope.DocumentTypes = []string{}
			}

			// Validate rules
			for i, rule := range req.Rules {
				if rule.ID == "" {
					respondJSON(w, http.StatusBadRequest, map[string]string{
						"error": fmt.Sprintf("rule %d: missing ID", i),
					})
					return
				}
				if rule.Field == "" {
					respondJSON(w, http.StatusBadRequest, map[string]string{
						"error": fmt.Sprintf("rule %d: missing field", i),
					})
					return
				}
				if rule.Operator == "" {
					respondJSON(w, http.StatusBadRequest, map[string]string{
						"error": fmt.Sprintf("rule %d: missing operator", i),
					})
					return
				}
			}

			// Validate actions
			for i, action := range req.Actions {
				validActions := map[PolicyAction]bool{
					ActionAllow:       true,
					ActionDeny:        true,
					ActionRequireAuth: true,
					ActionRouteTo:     true,
					ActionWatermark:   true,
					ActionLog:         true,
					ActionNotify:      true,
				}
				if action.Type == "" || !validActions[action.Type] {
					respondJSON(w, http.StatusBadRequest, map[string]string{
						"error": fmt.Sprintf("action %d: invalid type", i),
					})
					return
				}
			}

			repo := NewRepository(engine.db)
			if err := repo.Create(ctx, &req); err != nil {
				log.Printf("Error creating policy: %v", err)
				http.Error(w, "failed to create policy", http.StatusInternalServerError)
				return
			}

			// Reload policies into memory
			_ = engine.LoadPolicies(ctx)

			respondJSON(w, http.StatusCreated, map[string]string{
				"id":      req.ID,
				"message": "Policy created",
			})

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// policyByIDHandler handles GET/PUT/DELETE /api/v1/policies/{id}
func policyByIDHandler(engine *Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		policyID := extractIDFromPath(r.URL.Path, "/api/v1/policies/")

		if policyID == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "policy ID required"})
			return
		}

		switch r.Method {
		case http.MethodGet:
			repo := NewRepository(engine.db)
			policy, err := repo.Get(ctx, policyID)
			if err != nil {
				log.Printf("Error getting policy: %v", err)
				http.Error(w, "policy not found", http.StatusNotFound)
				return
			}
			respondJSON(w, http.StatusOK, policy)

		case http.MethodPut:
			var req Policy
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}

			req.ID = policyID

			// Validate at least one rule or action
			if len(req.Rules) == 0 && len(req.Actions) == 0 {
				respondJSON(w, http.StatusBadRequest, map[string]string{
					"error": "policy must have at least one rule or action",
				})
				return
			}

			repo := NewRepository(engine.db)
			if err := repo.Update(ctx, &req); err != nil {
				log.Printf("Error updating policy: %v", err)
				http.Error(w, "failed to update policy", http.StatusInternalServerError)
				return
			}

			// Reload policies into memory
			_ = engine.LoadPolicies(ctx)

			respondJSON(w, http.StatusOK, map[string]string{
				"id":      policyID,
				"message": "Policy updated",
			})

		case http.MethodDelete:
			repo := NewRepository(engine.db)
			if err := repo.Delete(ctx, policyID); err != nil {
				log.Printf("Error deleting policy: %v", err)
				http.Error(w, "failed to delete policy", http.StatusInternalServerError)
				return
			}

			// Reload policies into memory
			_ = engine.LoadPolicies(ctx)

			respondJSON(w, http.StatusOK, map[string]string{
				"id":      policyID,
				"message": "Policy deleted",
			})

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// evaluateHandler handles POST /api/v1/evaluate
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

		// Set current time if not provided
		if evalCtx.TimeOfDay.IsZero() {
			evalCtx.TimeOfDay = time.Now()
			evalCtx.DayOfWeek = int(evalCtx.TimeOfDay.Weekday())
		}

		// Evaluate policies
		results, err := engine.Evaluate(r.Context(), &evalCtx)
		if err != nil {
			log.Printf("Error evaluating policies: %v", err)
			http.Error(w, "evaluation failed", http.StatusInternalServerError)
			return
		}

		// Determine final action based on all results
		finalAction := ActionAllow
		finalMessage := "No policies matched - allowed by default"
		var matchedPolicies []*EvaluationResult

		for _, result := range results {
			if result.Matched {
				matchedPolicies = append(matchedPolicies, result)
				if result.Action == ActionDeny {
					finalAction = ActionDeny
					finalMessage = "Print denied by policy"
					break // Deny takes precedence
				} else if result.Action != ActionAllow && finalAction == ActionAllow {
					finalAction = result.Action
					finalMessage = result.Message
				}
			}
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"action":           finalAction,
			"message":          finalMessage,
			"matched_policies": matchedPolicies,
			"total_evaluated":  len(results),
		})
	}
}

// validateRulesHandler handles POST /api/v1/rules/validate
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
		warnings := []string{}

		validFields := map[string]bool{
			"user.id": true, "user.email": true, "user.groups": true,
			"printer.id":    true,
			"document.name": true, "document.type": true, "document.page_count": true,
			"document.color_mode": true, "document.duplex_mode": true, "document.cost": true,
			"time.hour": true, "time.day_of_week": true,
			"quota.remaining": true, "quota.used": true, "quota.limit": true,
			"ip.address": true, "device.id": true, "document.tags": true,
		}

		validOperators := map[Operator]bool{
			OpEquals: true, OpNotEquals: true, OpGreaterThan: true, OpLessThan: true,
			OpContains: true, OpNotContains: true, OpMatches: true,
			OpIn: true, OpNotIn: true, OpBetween: true,
			OpAlways: true, OpNever: true,
		}

		for i, rule := range req.Rules {
			if rule.ID == "" {
				errors = append(errors, fmt.Sprintf("rule %d: missing ID", i))
				valid = false
			}
			if rule.Field == "" {
				errors = append(errors, fmt.Sprintf("rule %d: missing field", i))
				valid = false
			} else if !validFields[rule.Field] {
				warnings = append(warnings, fmt.Sprintf("rule %d: unknown field '%s'", i, rule.Field))
			}
			if rule.Operator == "" {
				errors = append(errors, fmt.Sprintf("rule %d: missing operator", i))
				valid = false
			} else if !validOperators[rule.Operator] {
				errors = append(errors, fmt.Sprintf("rule %d: invalid operator '%s'", i, rule.Operator))
				valid = false
			}
			if rule.Value == nil && rule.Operator != OpAlways && rule.Operator != OpNever {
				errors = append(errors, fmt.Sprintf("rule %d: missing value for operator '%s'", i, rule.Operator))
				valid = false
			}
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"valid":    valid,
			"errors":   errors,
			"warnings": warnings,
		})
	}
}

// testPolicyHandler handles POST /api/v1/test
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

		if req.Policy == nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "policy is required"})
			return
		}

		if req.TestContext == nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "test_context is required"})
			return
		}

		// Set current time if not provided
		if req.TestContext.TimeOfDay.IsZero() {
			req.TestContext.TimeOfDay = time.Now()
			req.TestContext.DayOfWeek = int(req.TestContext.TimeOfDay.Weekday())
		}

		// Test policy against context using engine's rule evaluation
		e := Engine{}
		matched, ruleMatches := e.evaluateRules(req.Policy.Rules, req.TestContext)

		// Determine actions
		actions := []string{}
		parameters := map[string]interface{}{}
		if matched && len(req.Policy.Actions) > 0 {
			for _, action := range req.Policy.Actions {
				actions = append(actions, string(action.Type))
				if action.Parameters != nil {
					parameters = action.Parameters
				}
			}
		} else if !matched {
			actions = append(actions, string(ActionAllow))
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"matched":      matched,
			"actions":      actions,
			"parameters":   parameters,
			"rule_matches": ruleMatches,
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

	// Ensure empty slices are encoded as [] instead of null
	// by using a custom encoder
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}
