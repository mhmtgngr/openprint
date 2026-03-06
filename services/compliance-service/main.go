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

	// Compliance endpoints - register specific routes before generic ones
	mux.HandleFunc("/api/v1/controls/status/", updateControlStatusHandler(compSvc))
	mux.HandleFunc("/api/v1/controls/", controlsHandler(compSvc))
	mux.HandleFunc("/api/v1/controls", listControlsHandler(compSvc))
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
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx := r.Context()
		controlID := extractIDFromPath(r.URL.Path, "/api/v1/controls/")
		if controlID == "" {
			http.Error(w, "control ID required", http.StatusBadRequest)
			return
		}

		repo := NewRepository(svc.db)
		control, err := repo.GetControl(ctx, controlID)
		if err != nil {
			if apperrors.IsNotFound(err) {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": "control not found"})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to retrieve control"})
			return
		}

		respondJSON(w, http.StatusOK, control)
	}
}

func listControlsHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx := r.Context()

		// Parse query parameters
		framework := ComplianceFramework(r.URL.Query().Get("framework"))
		status := ComplianceStatus(r.URL.Query().Get("status"))
		limit := 50
		offset := 0

		if l := r.URL.Query().Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
				limit = parsed
			}
		}
		if o := r.URL.Query().Get("offset"); o != "" {
			if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
				offset = parsed
			}
		}

		repo := NewRepository(svc.db)
		controls, total, err := repo.ListControls(ctx, framework, status, limit, offset)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list controls"})
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"controls": controls,
			"total":    total,
			"limit":    limit,
			"offset":   offset,
		})
	}
}

func updateControlStatusHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx := r.Context()
		controlID := extractIDFromPath(r.URL.Path, "/api/v1/controls/status/")
		if controlID == "" {
			http.Error(w, "control ID required", http.StatusBadRequest)
			return
		}

		var req struct {
			Status     string `json:"status"`
			NextReview string `json:"next_review"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		status := ComplianceStatus(req.Status)
		if status != StatusCompliant && status != StatusNonCompliant && status != StatusPending && status != StatusNotApplicable {
			http.Error(w, "invalid status value", http.StatusBadRequest)
			return
		}

		now := time.Now()
		nextReview := now.AddDate(0, 3, 0) // default: 3 months from now
		if req.NextReview != "" {
			parsed, err := time.Parse(time.RFC3339, req.NextReview)
			if err != nil {
				http.Error(w, "invalid next_review date format, use RFC3339", http.StatusBadRequest)
				return
			}
			nextReview = parsed
		}

		repo := NewRepository(svc.db)
		if err := repo.UpdateControlStatus(ctx, controlID, status, now, nextReview); err != nil {
			if apperrors.IsNotFound(err) {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": "control not found"})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update control status"})
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"control_id": controlID,
			"status":     string(status),
			"message":    "status updated successfully",
		})
	}
}

func auditLogHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		repo := NewRepository(svc.db)

		if r.Method == http.MethodPost {
			var event AuditEvent
			if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			if err := repo.CreateAuditEvent(ctx, &event); err != nil {
				respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create audit event"})
				return
			}
			respondJSON(w, http.StatusCreated, map[string]string{"id": event.ID})
		} else if r.Method == http.MethodGet {
			filter := AuditFilter{
				Limit:  50,
				Offset: 0,
			}
			if l := r.URL.Query().Get("limit"); l != "" {
				if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
					filter.Limit = parsed
				}
			}
			if o := r.URL.Query().Get("offset"); o != "" {
				if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
					filter.Offset = parsed
				}
			}
			if uid := r.URL.Query().Get("user_id"); uid != "" {
				filter.UserID = uid
			}
			if et := r.URL.Query().Get("event_type"); et != "" {
				filter.EventType = et
			}
			if cat := r.URL.Query().Get("category"); cat != "" {
				filter.Category = cat
			}
			if st := r.URL.Query().Get("start_time"); st != "" {
				if parsed, err := time.Parse(time.RFC3339, st); err == nil {
					filter.StartTime = parsed
				}
			}
			if et := r.URL.Query().Get("end_time"); et != "" {
				if parsed, err := time.Parse(time.RFC3339, et); err == nil {
					filter.EndTime = parsed
				}
			}

			events, total, err := repo.QueryAuditEvents(ctx, filter)
			if err != nil {
				respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query audit events"})
				return
			}
			respondJSON(w, http.StatusOK, map[string]interface{}{
				"events": events,
				"total":  total,
			})
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

		ctx := r.Context()

		format := r.URL.Query().Get("format")
		if format == "" {
			format = "json"
		}
		if format != "json" && format != "csv" {
			http.Error(w, "unsupported format, use json or csv", http.StatusBadRequest)
			return
		}

		filter := AuditFilter{
			Limit:  10000,
			Offset: 0,
		}
		if st := r.URL.Query().Get("start_time"); st != "" {
			if parsed, err := time.Parse(time.RFC3339, st); err == nil {
				filter.StartTime = parsed
			}
		}
		if et := r.URL.Query().Get("end_time"); et != "" {
			if parsed, err := time.Parse(time.RFC3339, et); err == nil {
				filter.EndTime = parsed
			}
		}

		data, err := svc.ExportAuditLogs(ctx, filter, format)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to export audit logs"})
			return
		}

		switch format {
		case "csv":
			w.Header().Set("Content-Type", "text/csv")
			w.Header().Set("Content-Disposition", "attachment; filename=audit_log.csv")
		default:
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Disposition", "attachment; filename=audit_log.json")
		}
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}
}

func generateReportHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx := r.Context()

		var req struct {
			Framework   string `json:"framework"`
			PeriodStart string `json:"period_start"`
			PeriodEnd   string `json:"period_end"`
			GeneratedBy string `json:"generated_by"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		framework := ComplianceFramework(req.Framework)
		if framework != FrameworkFedRAMP && framework != FrameworkHIPAA && framework != FrameworkGDPR && framework != FrameworkSOC2 {
			http.Error(w, "invalid framework, must be one of: fedramp, hipaa, gdpr, soc2", http.StatusBadRequest)
			return
		}

		periodStart, err := time.Parse(time.RFC3339, req.PeriodStart)
		if err != nil {
			http.Error(w, "invalid period_start, use RFC3339 format", http.StatusBadRequest)
			return
		}

		periodEnd, err := time.Parse(time.RFC3339, req.PeriodEnd)
		if err != nil {
			http.Error(w, "invalid period_end, use RFC3339 format", http.StatusBadRequest)
			return
		}

		generatedBy := req.GeneratedBy
		if generatedBy == "" {
			generatedBy = "system"
		}

		report, err := svc.GenerateReport(ctx, framework, periodStart, periodEnd, generatedBy)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate report"})
			return
		}

		respondJSON(w, http.StatusOK, report)
	}
}

func summaryHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx := r.Context()

		summary, err := svc.GetComplianceSummary(ctx)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get compliance summary"})
			return
		}

		respondJSON(w, http.StatusOK, summary)
	}
}

func breachesHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		repo := NewRepository(svc.db)

		if r.Method == http.MethodPost {
			var breach DataBreach
			if err := json.NewDecoder(r.Body).Decode(&breach); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			if err := repo.RecordDataBreach(ctx, &breach); err != nil {
				respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to record data breach"})
				return
			}
			respondJSON(w, http.StatusCreated, map[string]string{"id": breach.ID})
		} else if r.Method == http.MethodGet {
			// List breaches via a simple query
			rows, err := svc.db.Query(ctx, `
				SELECT id, discovered_at, reported_at, severity, affected_records,
				       data_types, description, containment_status, notification_sent,
				       resolved_at, lessons_learned
				FROM data_breaches ORDER BY reported_at DESC LIMIT 100
			`)
			if err != nil {
				respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list breaches"})
				return
			}
			defer rows.Close()

			var breaches []DataBreach
			for rows.Next() {
				var b DataBreach
				var dataTypesJSON []byte
				if err := rows.Scan(
					&b.ID, &b.DiscoveredAt, &b.ReportedAt, &b.Severity,
					&b.AffectedRecords, &dataTypesJSON, &b.Description,
					&b.ContainmentStatus, &b.NotificationSent, &b.ResolvedAt,
					&b.LessonsLearned,
				); err != nil {
					respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to scan breach"})
					return
				}
				if len(dataTypesJSON) > 0 {
					json.Unmarshal(dataTypesJSON, &b.DataTypes)
				}
				breaches = append(breaches, b)
			}
			if breaches == nil {
				breaches = []DataBreach{}
			}
			respondJSON(w, http.StatusOK, map[string]interface{}{"breaches": breaches})
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

		ctx := r.Context()

		// Default: reviews due within 30 days
		withinDays := 30
		if d := r.URL.Query().Get("within_days"); d != "" {
			if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
				withinDays = parsed
			}
		}

		repo := NewRepository(svc.db)
		controls, err := repo.GetPendingReviews(ctx, time.Duration(withinDays)*24*time.Hour)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get pending reviews"})
			return
		}

		if controls == nil {
			controls = []*Control{}
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"controls":    controls,
			"within_days": withinDays,
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
