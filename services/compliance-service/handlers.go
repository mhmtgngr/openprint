// Package main provides HTTP handlers for the OpenPrint Compliance Service.
package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HandlerDependencies provides dependencies for HTTP handlers.
type HandlerDependencies struct {
	DB        *pgxpool.Pool
	JWTSecret string
}

// healthHandler handles GET /health
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

// controlByIDHandler handles GET/PUT/DELETE /api/v1/controls/{id}
func controlByIDHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		controlID := extractIDFromPath(r.URL.Path, "/api/v1/controls/")

		if controlID == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "control ID required"})
			return
		}

		repo := NewRepository(svc.db)

		switch r.Method {
		case http.MethodGet:
			control, err := repo.GetControl(ctx, controlID)
			if err != nil {
				log.Printf("Error getting control: %v", err)
				http.Error(w, "control not found", http.StatusNotFound)
				return
			}
			respondJSON(w, http.StatusOK, control)

		case http.MethodPut:
			var req struct {
				Status       *string    `json:"status"`
				LastAssessed *time.Time `json:"last_assessed"`
				NextReview   *time.Time `json:"next_review"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}

			if req.Status == nil || req.LastAssessed == nil || req.NextReview == nil {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": "status, last_assessed, and next_review are required"})
				return
			}

			if err := repo.UpdateControlStatus(ctx, controlID, ComplianceStatus(*req.Status), *req.LastAssessed, *req.NextReview); err != nil {
				log.Printf("Error updating control: %v", err)
				http.Error(w, "failed to update control", http.StatusInternalServerError)
				return
			}
			respondJSON(w, http.StatusOK, map[string]string{"id": controlID, "status": "updated"})

		case http.MethodDelete:
			// Not implemented in repository yet
			respondJSON(w, http.StatusNotImplemented, map[string]string{"error": "delete not implemented"})

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// listControlsHandler handles GET/POST /api/v1/controls
func listControlsHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		switch r.Method {
		case http.MethodGet:
			// Parse query parameters
			framework := r.URL.Query().Get("framework")
			status := r.URL.Query().Get("status")

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

			repo := NewRepository(svc.db)
			controls, total, err := repo.ListControls(ctx, ComplianceFramework(framework), ComplianceStatus(status), limit, offset)
			if err != nil {
				log.Printf("Error listing controls: %v", err)
				http.Error(w, "failed to list controls", http.StatusInternalServerError)
				return
			}

			respondJSON(w, http.StatusOK, map[string]interface{}{
				"controls": controls,
				"page":     page,
				"limit":    limit,
				"total":    total,
			})

		case http.MethodPost:
			var req struct {
				Framework       string `json:"framework"`
				Family          string `json:"family"`
				Title           string `json:"title"`
				Description     string `json:"description"`
				Implementation  string `json:"implementation"`
				Status          string `json:"status"`
				NextReview      string `json:"next_review"`
				ResponsibleTeam string `json:"responsible_team"`
				RiskLevel       string `json:"risk_level"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}

			// Validate framework
			validFrameworks := map[string]bool{"fedramp": true, "hipaa": true, "gdpr": true, "soc2": true}
			if !validFrameworks[req.Framework] {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid framework"})
				return
			}

			// Create control using direct database insertion
			controlID := uuid.New().String()
			var nextReviewTime time.Time
			if req.NextReview != "" {
				var err error
				nextReviewTime, err = time.Parse(time.RFC3339, req.NextReview)
				if err != nil {
					respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid next_review format"})
					return
				}
			} else {
				nextReviewTime = time.Now().AddDate(0, 0, 30) // Default 30 days
			}

			// Insert control directly
			const query = `
				INSERT INTO compliance_controls (id, framework, family, title, description, implementation, status, next_review, responsible_team, risk_level)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			`
			_, err := svc.db.Exec(ctx, query, controlID, req.Framework, req.Family, req.Title, req.Description, req.Implementation, req.Status, nextReviewTime, req.ResponsibleTeam, req.RiskLevel)
			if err != nil {
				log.Printf("Error creating control: %v", err)
				http.Error(w, "failed to create control", http.StatusInternalServerError)
				return
			}

			// Fetch the created control
			repo := NewRepository(svc.db)
			control, _ := repo.GetControl(ctx, controlID)
			respondJSON(w, http.StatusCreated, control)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// updateControlStatusHandler handles PUT /api/v1/controls/status/{id}
func updateControlStatusHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		controlID := extractIDFromPath(r.URL.Path, "/api/v1/controls/status/")

		if controlID == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "control ID required"})
			return
		}

		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Status       string     `json:"status"`
			LastAssessed *time.Time `json:"last_assessed"`
			NextReview   *time.Time `json:"next_review"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		// Validate status
		validStatuses := map[string]bool{"compliant": true, "non_compliant": true, "pending": true, "not_applicable": true, "unknown": true}
		if !validStatuses[req.Status] {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid status"})
			return
		}

		lastAssessed := req.LastAssessed
		if lastAssessed == nil {
			now := time.Now()
			lastAssessed = &now
		}

		nextReview := req.NextReview
		if nextReview == nil {
			t := time.Now().AddDate(0, 0, 30)
			nextReview = &t
		}

		repo := NewRepository(svc.db)
		if err := repo.UpdateControlStatus(ctx, controlID, ComplianceStatus(req.Status), *lastAssessed, *nextReview); err != nil {
			log.Printf("Error updating control status: %v", err)
			http.Error(w, "failed to update control status", http.StatusInternalServerError)
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"control_id": controlID,
			"status":     req.Status,
			"updated_at": time.Now().Format(time.RFC3339),
		})
	}
}

// auditLogHandler handles GET/POST /api/v1/audit
func auditLogHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		switch r.Method {
		case http.MethodGet:
			// Parse query parameters
			limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
			if limit < 1 || limit > 100 {
				limit = 50
			}
			offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

			// Build filter
			filter := AuditFilter{
				Limit:  limit,
				Offset: offset,
			}
			if startStr := r.URL.Query().Get("start_time"); startStr != "" {
				if t, err := time.Parse(time.RFC3339, startStr); err == nil {
					filter.StartTime = t
				}
			}
			if endStr := r.URL.Query().Get("end_time"); endStr != "" {
				if t, err := time.Parse(time.RFC3339, endStr); err == nil {
					filter.EndTime = t
				}
			}
			if userID := r.URL.Query().Get("user_id"); userID != "" {
				filter.UserID = userID
			}
			if eventType := r.URL.Query().Get("event_type"); eventType != "" {
				filter.EventType = eventType
			}

			repo := NewRepository(svc.db)
			events, total, err := repo.QueryAuditEvents(ctx, filter)
			if err != nil {
				log.Printf("Error getting audit logs: %v", err)
				http.Error(w, "failed to get audit logs", http.StatusInternalServerError)
				return
			}
			respondJSON(w, http.StatusOK, map[string]interface{}{"events": events, "limit": limit, "offset": offset, "total": total})

		case http.MethodPost:
			var req AuditEvent
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}

			repo := NewRepository(svc.db)
			if err := repo.CreateAuditEvent(ctx, &req); err != nil {
				log.Printf("Error creating audit log: %v", err)
				http.Error(w, "failed to create audit log", http.StatusInternalServerError)
				return
			}
			respondJSON(w, http.StatusCreated, map[string]string{"id": req.ID})

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// exportAuditLogsHandler handles GET /api/v1/audit/export
func exportAuditLogsHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		format := r.URL.Query().Get("format")
		if format == "" {
			format = "json"
		}

		// Build filter
		filter := AuditFilter{
			Limit: 10000, // Max for export
		}
		if startStr := r.URL.Query().Get("start_time"); startStr != "" {
			if t, err := time.Parse(time.RFC3339, startStr); err == nil {
				filter.StartTime = t
			}
		}
		if endStr := r.URL.Query().Get("end_time"); endStr != "" {
			if t, err := time.Parse(time.RFC3339, endStr); err == nil {
				filter.EndTime = t
			}
		}

		// Export based on format
		switch format {
		case "csv":
			// Get events
			repo := NewRepository(svc.db)
			events, _, err := repo.QueryAuditEvents(ctx, filter)
			if err != nil {
				log.Printf("Error exporting audit logs: %v", err)
				http.Error(w, "failed to export audit logs", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "text/csv")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=audit_logs_%s.csv", time.Now().Format("20060102_150405")))
			writer := csv.NewWriter(w)
			defer writer.Flush()
			writer.Write([]string{"Timestamp", "EventType", "Category", "UserID", "UserName", "ResourceID", "ResourceType", "Action", "Outcome", "IPAddress"})
			for _, event := range events {
				writer.Write([]string{
					event.Timestamp.Format(time.RFC3339),
					event.EventType,
					event.Category,
					event.UserID,
					event.UserName,
					event.ResourceID,
					event.ResourceType,
					event.Action,
					event.Outcome,
					event.IPAddress,
				})
			}

		default: // json
			data, err := svc.ExportAuditLogs(ctx, filter, "json")
			if err != nil {
				log.Printf("Error exporting audit logs: %v", err)
				http.Error(w, "failed to export audit logs", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=audit_logs_%s.json", time.Now().Format("20060102_150405")))
			w.Write(data)
		}
	}
}

// generateReportHandler handles POST /api/v1/reports/generate
func generateReportHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

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

		// Validate framework
		validFrameworks := map[string]bool{"fedramp": true, "hipaa": true, "gdpr": true, "soc2": true, "all": true}
		if !validFrameworks[req.Framework] {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid framework"})
			return
		}

		// Parse dates
		periodStart, err := time.Parse(time.RFC3339, req.PeriodStart)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid period_start format"})
			return
		}
		periodEnd, err := time.Parse(time.RFC3339, req.PeriodEnd)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid period_end format"})
			return
		}

		generatedBy := req.GeneratedBy
		if generatedBy == "" {
			generatedBy = "system"
		}

		report, err := svc.GenerateReport(ctx, ComplianceFramework(req.Framework), periodStart, periodEnd, generatedBy)
		if err != nil {
			log.Printf("Error generating report: %v", err)
			http.Error(w, "failed to generate report", http.StatusInternalServerError)
			return
		}

		// Save report to database
		findingsJSON, _ := json.Marshal(report.Findings)
		const reportQuery = `
			INSERT INTO compliance_reports
			(id, framework, period_start, period_end, overall_status, compliant_count,
			 non_compliant_count, pending_count, total_controls, high_risk_count, findings, generated_by, generated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		`
		_, err = svc.db.Exec(ctx, reportQuery,
			report.ID, report.Framework, report.PeriodStart, report.PeriodEnd, report.OverallStatus,
			report.CompliantCount, report.NonCompliant, report.PendingCount,
			report.TotalControls, report.HighRiskCount, findingsJSON, report.GeneratedBy, report.GeneratedAt,
		)
		if err != nil {
			log.Printf("Warning: failed to save report: %v", err)
		}

		respondJSON(w, http.StatusOK, report)
	}
}

// summaryHandler handles GET /api/v1/reports/summary
func summaryHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		summary, err := svc.GetComplianceSummary(ctx)
		if err != nil {
			log.Printf("Error getting summary: %v", err)
			http.Error(w, "failed to get summary", http.StatusInternalServerError)
			return
		}

		// Convert map to slice for JSON response
		type FrameworkSummary struct {
			Framework string `json:"framework"`
			Status    string `json:"status"`
		}
		var result []FrameworkSummary
		for fw, status := range summary {
			result = append(result, FrameworkSummary{
				Framework: string(fw),
				Status:    string(status),
			})
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"frameworks": result,
		})
	}
}

// breachesHandler handles GET/POST /api/v1/breaches
func breachesHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		switch r.Method {
		case http.MethodGet:
			// Get all breaches - need to implement this in the repository
			// For now, return empty list
			respondJSON(w, http.StatusOK, map[string]interface{}{"breaches": []*DataBreach{}, "count": 0})

		case http.MethodPost:
			var req DataBreach
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}

			// Set defaults
			req.DiscoveredAt = time.Now()
			req.ContainmentStatus = "identifying"

			repo := NewRepository(svc.db)
			if err := repo.RecordDataBreach(ctx, &req); err != nil {
				log.Printf("Error creating breach: %v", err)
				http.Error(w, "failed to create breach", http.StatusInternalServerError)
				return
			}
			respondJSON(w, http.StatusCreated, map[string]string{"id": req.ID})

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// pendingReviewsHandler handles GET /api/v1/reviews/pending
func pendingReviewsHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse days parameter
		daysStr := r.URL.Query().Get("days")
		days := 30 // default
		if daysStr != "" {
			if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
				days = d
			}
		}

		repo := NewRepository(svc.db)
		controls, err := repo.GetPendingReviews(ctx, time.Duration(days)*24*time.Hour)
		if err != nil {
			log.Printf("Error getting pending reviews: %v", err)
			http.Error(w, "failed to get pending reviews", http.StatusInternalServerError)
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"controls":   controls,
			"count":      len(controls),
			"days_ahead": days,
		})
	}
}

// Helper functions

// extractIDFromPath extracts an ID from the URL path after a given prefix.
func extractIDFromPath(path, prefix string) string {
	if len(path) > len(prefix) {
		return path[len(prefix):]
	}
	return ""
}

// respondJSON writes a JSON response with the given status code.
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
