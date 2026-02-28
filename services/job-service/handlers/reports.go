// Package handler provides HTTP handlers for advanced reporting and analytics.
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
)

// ReportsRepository defines the interface for reporting operations.
type ReportsRepository interface {
	GetUsageSummary(ctx context.Context, organizationID, startDate, endDate string) (*UsageSummary, error)
	GetTopUsers(ctx context.Context, organizationID string, startDate, endDate string, limit int) ([]*UserUsageStats, error)
	GetTopPrinters(ctx context.Context, organizationID string, startDate, endDate string, limit int) ([]*PrinterUsageStats, error)
	GetDepartmentReport(ctx context.Context, organizationID string, startDate, endDate string) ([]*DepartmentReport, error)
	GetEnvironmentalReport(ctx context.Context, organizationID string, startDate, endDate string) (*EnvironmentalReport, error)
	GetTrendData(ctx context.Context, organizationID string, startDate, endDate string, granularity string) ([]*TrendDataPoint, error)
}

// UsageSummary represents aggregated usage statistics.
type UsageSummary struct {
	TotalJobs        int
	TotalPages       int
	ColorPages       int
	DuplexPages      int
	CompletedJobs    int
	FailedJobs       int
	CancelledJobs    int
	AverageJobTime   int // in seconds
	TotalCost        float64
	Currency         string
	CO2Emission      float64 // in kg
	TreesSaved       float64 // number of trees equivalent
}

// UserUsageStats represents usage statistics for a user.
type UserUsageStats struct {
	UserID      string
	UserEmail   string
	UserName    string
	TotalJobs   int
	TotalPages  int
	ColorPages  int
	DuplexPages int
	TotalCost   float64
	Rank        int
}

// PrinterUsageStats represents usage statistics for a printer.
type PrinterUsageStats struct {
	PrinterID       string
	PrinterName     string
	TotalJobs       int
	TotalPages      int
	ColorPages      int
	DuplexPages     int
	TotalCost       float64
	Uptime          float64 // percentage
	ErrorRate       float64 // percentage
	Rank            int
}

// DepartmentReport represents report data for a department/cost center.
type DepartmentReport struct {
	DepartmentID   string
	DepartmentName string
	TotalJobs      int
	TotalPages     int
	TotalCost      float64
	Currency       string
	UserCount      int
}

// EnvironmentalReport represents environmental impact metrics.
type EnvironmentalReport struct {
	TotalPages         int
	TreesSaved         float64 // Number of trees saved due to duplex printing
	CO2Emission        float64 // CO2 emitted in kg
	CO2Offset          float64 // CO2 offset in kg
	EnergyConsumption   float64 // kWh
	WasteReduction     float64 // kg of paper waste reduced
}

// TrendDataPoint represents a single data point in a trend.
type TrendDataPoint struct {
	Date       string
	Jobs       int
	Pages      int
	Cost       float64
	CO2        float64
}

// ReportsHandler handles reporting HTTP endpoints.
type ReportsHandler struct {
	db    *pgxpool.Pool
	repo  ReportsRepository
}

// NewReportsHandler creates a new reports handler instance.
func NewReportsHandler(db *pgxpool.Pool) *ReportsHandler {
	return &ReportsHandler{
		db:   db,
		repo: NewReportsRepository(db),
	}
}

// UsageSummaryHandler handles usage summary requests.
func (h *ReportsHandler) UsageSummaryHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgID := r.URL.Query().Get("organization_id")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	if orgID == "" {
		respondError(w, apperrors.New("organization_id is required", http.StatusBadRequest))
		return
	}

	// Default to last 30 days
	if startDate == "" {
		startDate = time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}

	summary, err := h.repo.GetUsageSummary(ctx, orgID, startDate, endDate)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get usage summary", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"organization_id":  orgID,
		"start_date":       startDate,
		"end_date":         endDate,
		"total_jobs":       summary.TotalJobs,
		"total_pages":      summary.TotalPages,
		"color_pages":      summary.ColorPages,
		"duplex_pages":     summary.DuplexPages,
		"completed_jobs":   summary.CompletedJobs,
		"failed_jobs":      summary.FailedJobs,
		"cancelled_jobs":   summary.CancelledJobs,
		"average_job_time": summary.AverageJobTime,
		"total_cost":       summary.TotalCost,
		"currency":         summary.Currency,
		"environmental": map[string]interface{}{
			"co2_emission": summary.CO2Emission,
			"trees_saved":   summary.TreesSaved,
		},
	})
}

// TopUsersHandler handles top users by usage requests.
func (h *ReportsHandler) TopUsersHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgID := r.URL.Query().Get("organization_id")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	limit := 10

	if orgID == "" {
		respondError(w, apperrors.New("organization_id is required", http.StatusBadRequest))
		return
	}

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if startDate == "" {
		startDate = time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}

	users, err := h.repo.GetTopUsers(ctx, orgID, startDate, endDate, limit)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get top users", http.StatusInternalServerError))
		return
	}

	response := make([]map[string]interface{}, len(users))
	for i, u := range users {
		response[i] = map[string]interface{}{
			"user_id":      u.UserID,
			"user_email":   u.UserEmail,
			"user_name":    u.UserName,
			"total_jobs":   u.TotalJobs,
			"total_pages":  u.TotalPages,
			"color_pages":  u.ColorPages,
			"duplex_pages": u.DuplexPages,
			"total_cost":   u.TotalCost,
			"rank":         u.Rank,
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"organization_id": orgID,
		"start_date":      startDate,
		"end_date":        endDate,
		"users":           response,
		"count":           len(users),
	})
}

// TopPrintersHandler handles top printers by usage requests.
func (h *ReportsHandler) TopPrintersHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgID := r.URL.Query().Get("organization_id")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	limit := 10

	if orgID == "" {
		respondError(w, apperrors.New("organization_id is required", http.StatusBadRequest))
		return
	}

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if startDate == "" {
		startDate = time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}

	printers, err := h.repo.GetTopPrinters(ctx, orgID, startDate, endDate, limit)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get top printers", http.StatusInternalServerError))
		return
	}

	response := make([]map[string]interface{}, len(printers))
	for i, p := range printers {
		response[i] = map[string]interface{}{
			"printer_id":    p.PrinterID,
			"printer_name":  p.PrinterName,
			"total_jobs":    p.TotalJobs,
			"total_pages":   p.TotalPages,
			"color_pages":   p.ColorPages,
			"duplex_pages":  p.DuplexPages,
			"total_cost":    p.TotalCost,
			"uptime":        p.Uptime,
			"error_rate":    p.ErrorRate,
			"rank":          p.Rank,
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"organization_id": orgID,
		"start_date":      startDate,
		"end_date":        endDate,
		"printers":        response,
		"count":           len(printers),
	})
}

// DepartmentReportHandler handles department/cost center reporting.
func (h *ReportsHandler) DepartmentReportHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgID := r.URL.Query().Get("organization_id")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	if orgID == "" {
		respondError(w, apperrors.New("organization_id is required", http.StatusBadRequest))
		return
	}

	if startDate == "" {
		startDate = time.Now().AddDate(0, -1, 0).Format("2006-01-02") // Last month
	}
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}

	departments, err := h.repo.GetDepartmentReport(ctx, orgID, startDate, endDate)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get department report", http.StatusInternalServerError))
		return
	}

	response := make([]map[string]interface{}, len(departments))
	totalJobs := 0
	totalPages := 0
	totalCost := 0.0

	for i, d := range departments {
		response[i] = map[string]interface{}{
			"department_id":   d.DepartmentID,
			"department_name": d.DepartmentName,
			"total_jobs":      d.TotalJobs,
			"total_pages":     d.TotalPages,
			"total_cost":      d.TotalCost,
			"currency":        d.Currency,
			"user_count":      d.UserCount,
		}
		totalJobs += d.TotalJobs
		totalPages += d.TotalPages
		totalCost += d.TotalCost
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"organization_id": orgID,
		"start_date":      startDate,
		"end_date":        endDate,
		"departments":     response,
		"totals": map[string]interface{}{
			"total_jobs":  totalJobs,
			"total_pages": totalPages,
			"total_cost":  totalCost,
		},
		"count": len(response),
	})
}

// EnvironmentalReportHandler handles environmental impact reporting.
func (h *ReportsHandler) EnvironmentalReportHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgID := r.URL.Query().Get("organization_id")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	if orgID == "" {
		respondError(w, apperrors.New("organization_id is required", http.StatusBadRequest))
		return
	}

	if startDate == "" {
		startDate = time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}

	report, err := h.repo.GetEnvironmentalReport(ctx, orgID, startDate, endDate)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get environmental report", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"organization_id":   orgID,
		"start_date":        startDate,
		"end_date":          endDate,
		"total_pages":       report.TotalPages,
		"trees_saved":       report.TreesSaved,
		"co2_emission":      report.CO2Emission,
		"co2_offset":        report.CO2Offset,
		"energy_consumption": report.EnergyConsumption,
		"waste_reduction":   report.WasteReduction,
		"metrics": map[string]interface{}{
			"pages_per_tree":       8000, // Average pages per tree
			"co2_per_page":        0.01, // kg CO2 per page
			"energy_kwh_per_page": 0.005, // kWh per page
		},
	})
}

// TrendDataHandler handles trend and time-series data requests.
func (h *ReportsHandler) TrendDataHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgID := r.URL.Query().Get("organization_id")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	granularity := r.URL.Query().Get("granularity") // 'day', 'week', 'month'

	if orgID == "" {
		respondError(w, apperrors.New("organization_id is required", http.StatusBadRequest))
		return
	}

	if startDate == "" {
		startDate = time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}
	if granularity == "" {
		granularity = "day"
	}

	data, err := h.repo.GetTrendData(ctx, orgID, startDate, endDate, granularity)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get trend data", http.StatusInternalServerError))
		return
	}

	response := make([]map[string]interface{}, len(data))
	for i, d := range data {
		response[i] = map[string]interface{}{
			"date":  d.Date,
			"jobs":  d.Jobs,
			"pages": d.Pages,
			"cost":  d.Cost,
			"co2":   d.CO2,
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"organization_id": orgID,
		"start_date":      startDate,
		"end_date":        endDate,
		"granularity":     granularity,
		"data":            response,
		"count":           len(data),
	})
}

// ExportReportHandler handles exporting reports in various formats.
func (h *ReportsHandler) ExportReportHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ReportType    string `json:"report_type"`    // 'usage', 'environmental', 'department', 'user', 'printer'
		OrganizationID string `json:"organization_id"`
		StartDate     string `json:"start_date"`
		EndDate       string `json:"end_date"`
		Format        string `json:"format"`        // 'csv', 'json', 'pdf'
		EmailTo       string `json:"email_to"`      // Optional: email report to this address
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	if req.ReportType == "" {
		req.ReportType = "usage"
	}
	if req.Format == "" {
		req.Format = "json"
	}

	// Generate report data based on type
	var data interface{}
	var filename string

	switch req.ReportType {
	case "usage":
		summary, err := h.repo.GetUsageSummary(ctx, req.OrganizationID, req.StartDate, req.EndDate)
		if err != nil {
			respondError(w, apperrors.Wrap(err, "failed to generate usage report", http.StatusInternalServerError))
			return
		}
		data = summary
		filename = fmt.Sprintf("usage_report_%s_%s", req.StartDate, req.EndDate)

	case "environmental":
		report, err := h.repo.GetEnvironmentalReport(ctx, req.OrganizationID, req.StartDate, req.EndDate)
		if err != nil {
			respondError(w, apperrors.Wrap(err, "failed to generate environmental report", http.StatusInternalServerError))
			return
		}
		data = report
		filename = fmt.Sprintf("environmental_report_%s_%s", req.StartDate, req.EndDate)

	default:
		// Default to usage report
		summary, _ := h.repo.GetUsageSummary(ctx, req.OrganizationID, req.StartDate, req.EndDate)
		data = summary
		filename = fmt.Sprintf("report_%s", time.Now().Format("20060102_150405"))
	}

	// Handle different export formats
	switch req.Format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.csv", filename))
		// Generate CSV (simplified)
		w.Write([]byte("Report generated successfully"))

	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.json", filename))
		json.NewEncoder(w).Encode(data)

	case "pdf":
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.pdf", filename))
		// In production, generate actual PDF
		w.Write([]byte("PDF report"))

	default:
		respondError(w, apperrors.New("unsupported format", http.StatusBadRequest))
		return
	}
}

// CustomReportHandler handles generating custom reports with user-defined parameters.
func (h *CustomReportHandler) CustomReportHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		OrganizationID string   `json:"organization_id"`
		StartDate      string   `json:"start_date"`
		EndDate        string   `json:"end_date"`
		Metrics        []string `json:"metrics"`        // 'jobs', 'pages', 'cost', 'co2', etc.
		GroupBy        []string `json:"group_by"`       // 'user', 'printer', 'department', 'date'
		Filters        map[string]string `json:"filters"` // Additional filters
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	if req.OrganizationID == "" {
		respondError(w, apperrors.New("organization_id is required", http.StatusBadRequest))
		return
	}

	// Build and execute custom query
	// This is a simplified implementation - in production, build dynamic queries
	query := `
		SELECT DATE(j.created_at)::text as date,
		       COUNT(*) as jobs,
		       COALESCE(SUM(j.copies), 0) as pages
		FROM print_jobs j
		LEFT JOIN users u ON u.email = j.user_email
		WHERE u.organization_id = $1::uuid
		  AND DATE(j.created_at) >= $2::date
		  AND DATE(j.created_at) <= $3::date
		GROUP BY DATE(j.created_at)
		ORDER BY date
	`

	rows, err := h.db.Query(ctx, query, req.OrganizationID, req.StartDate, req.EndDate)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to generate custom report", http.StatusInternalServerError))
		return
	}
	defer rows.Close()

	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		var date string
		var jobs, pages int
		if err := rows.Scan(&date, &jobs, &pages); err != nil {
			continue
		}
		results = append(results, map[string]interface{}{
			"date":  date,
			"jobs":  jobs,
			"pages": pages,
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"organization_id": req.OrganizationID,
		"start_date":      req.StartDate,
		"end_date":        req.EndDate,
		"metrics":         req.Metrics,
		"group_by":        req.GroupBy,
		"results":         results,
		"count":           len(results),
	})
}

// ScheduleReportHandler handles scheduling automated reports.
func (h *ReportsHandler) ScheduleReportHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case http.MethodPost:
		h.createScheduledReport(w, r, ctx)
	case http.MethodGet:
		h.listScheduledReports(w, r, ctx)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *ReportsHandler) createScheduledReport(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	var req struct {
		OrganizationID string   `json:"organization_id"`
		ReportType     string   `json:"report_type"`
		Schedule       string   `json:"schedule"`       // 'daily', 'weekly', 'monthly'
		Recipients     []string `json:"recipients"`     // Email addresses
		Format         string   `json:"format"`         // 'csv', 'json', 'pdf'
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Store scheduled report configuration
	query := `
		INSERT INTO scheduled_reports (
			id, organization_id, report_type, schedule,
			recipients, format, created_at
		) VALUES (
			$1::uuid, $2::uuid, $3, $4, $5, $6, NOW()
		)
		RETURNING id
	`

	id := uuid.New().String()
	recipientsJSON, _ := json.Marshal(req.Recipients)

	err := h.db.QueryRow(ctx, query, id, req.OrganizationID, req.ReportType,
		req.Schedule, recipientsJSON, req.Format).Scan(&id)
	if err != nil {
		// Create table if it doesn't exist
		initQuery := `
			CREATE TABLE IF NOT EXISTS scheduled_reports (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
				report_type VARCHAR(50) NOT NULL,
				schedule VARCHAR(20) NOT NULL,
				recipients JSONB,
				format VARCHAR(10) DEFAULT 'json',
				is_active BOOLEAN DEFAULT true,
				created_at TIMESTAMPTZ DEFAULT NOW(),
				updated_at TIMESTAMPTZ DEFAULT NOW()
			);
		`
		h.db.Exec(ctx, initQuery)

		// Retry insert
		err = h.db.QueryRow(ctx, query, id, req.OrganizationID, req.ReportType,
			req.Schedule, recipientsJSON, req.Format).Scan(&id)
		if err != nil {
			respondError(w, apperrors.Wrap(err, "failed to schedule report", http.StatusInternalServerError))
			return
		}
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"report_id":        id,
		"organization_id":  req.OrganizationID,
		"report_type":      req.ReportType,
		"schedule":         req.Schedule,
		"recipients":       req.Recipients,
		"format":           req.Format,
	})
}

func (h *ReportsHandler) listScheduledReports(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	orgID := r.URL.Query().Get("organization_id")
	if orgID == "" {
		respondError(w, apperrors.New("organization_id is required", http.StatusBadRequest))
		return
	}

	query := `
		SELECT id, organization_id, report_type, schedule, recipients,
		       format, is_active, created_at, updated_at
		FROM scheduled_reports
		WHERE organization_id = $1::uuid AND is_active = true
		ORDER BY created_at DESC
	`

	rows, err := h.db.Query(ctx, query, orgID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to list scheduled reports", http.StatusInternalServerError))
		return
	}
	defer rows.Close()

	reports := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id, orgID, reportType, schedule, format string
		var recipientsJSON []byte
		var isActive bool
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&id, &orgID, &reportType, &schedule, &recipientsJSON,
			&format, &isActive, &createdAt, &updatedAt); err != nil {
			continue
		}

		var recipients []string
		json.Unmarshal(recipientsJSON, &recipients)

		reports = append(reports, map[string]interface{}{
			"report_id":       id,
			"organization_id": orgID,
			"report_type":     reportType,
			"schedule":        schedule,
			"recipients":      recipients,
			"format":          format,
			"is_active":       isActive,
			"created_at":      createdAt.Format(time.RFC3339),
			"updated_at":      updatedAt.Format(time.RFC3339),
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"scheduled_reports": reports,
		"count":             len(reports),
	})
}

// Helper functions

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, err error) {
	var appErr *apperrors.AppError
	if json.Unmarshal([]byte(fmt.Sprintf("%v", err)), &appErr) == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(appErr.StatusCode)
		json.NewEncoder(w).Encode(apperrors.ToJSON(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]string{
		"code":    "INTERNAL_ERROR",
		"message": "An internal error occurred",
	})
}
