// Package handler provides HTTP handlers for print cost calculation and allocation.
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
)

// CostRepository defines the interface for cost repository operations.
type CostRepository interface {
	GetCostConfig(ctx context.Context, organizationID, printerID, costType string) (*PrintCost, error)
	ListCostConfigs(ctx context.Context, organizationID string) ([]*PrintCost, error)
	SetCostConfig(ctx context.Context, cost *PrintCost) error
	DeleteCostConfig(ctx context.Context, costID string) error
	CalculateJobCost(ctx context.Context, jobID string) (*JobCost, error)
	GetJobCost(ctx context.Context, jobID string) (*JobCost, error)
	GetCostByPeriod(ctx context.Context, organizationID, startDate, endDate string) ([]*CostSummary, error)
	GetCostByUser(ctx context.Context, organizationID, userID string, startDate, endDate string) ([]*UserCostSummary, error)
	GetCostByPrinter(ctx context.Context, organizationID, printerID string, startDate, endDate string) ([]*PrinterCostSummary, error)
}

// PrintCost represents a cost configuration for printing.
type PrintCost struct {
	ID            string
	OrganizationID string
	PrinterID     string
	CostType      string // 'monochrome_a4', 'color_a4', 'duplex_a4', etc.
	CostPerPage   float64
	Currency      string
	EffectiveFrom time.Time
	EffectiveTo   *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// JobCost represents the calculated cost for a print job.
type JobCost struct {
	ID          string
	JobID       string
	PageCount   int
	ColorPages  int
	DuplexPages int
	Cost        float64
	Currency    string
	CalculatedAt time.Time
}

// CostSummary represents aggregated cost data.
type CostSummary struct {
	Date        string
	TotalJobs   int
	TotalPages  int
	ColorPages  int
	DuplexPages int
	TotalCost   float64
	Currency    string
}

// UserCostSummary represents cost data aggregated by user.
type UserCostSummary struct {
	UserID      string
	UserEmail   string
	UserName    string
	TotalJobs   int
	TotalPages  int
	ColorPages  int
	DuplexPages int
	TotalCost   float64
	Currency    string
}

// PrinterCostSummary represents cost data aggregated by printer.
type PrinterCostSummary struct {
	PrinterID   string
	PrinterName string
	TotalJobs   int
	TotalPages  int
	ColorPages  int
	DuplexPages int
	TotalCost   float64
	Currency    string
}

// CostHandler handles cost management HTTP endpoints.
type CostHandler struct {
	db     *pgxpool.Pool
	costRepo CostRepository
}

// NewCostHandler creates a new cost handler instance.
func NewCostHandler(db *pgxpool.Pool) *CostHandler {
	return &CostHandler{
		db:       db,
		costRepo: NewCostRepository(db),
	}
}

// CostConfigRequest represents a request to set cost configuration.
type CostConfigRequest struct {
	PrinterID    string   `json:"printer_id"`
	CostType     string   `json:"cost_type"`
	CostPerPage  float64  `json:"cost_per_page"`
	Currency     string   `json:"currency"`
	EffectiveTo  *string  `json:"effective_to,omitempty"`
}

// CalculateCostRequest represents a request to calculate print job cost.
type CalculateCostRequest struct {
	PrinterID    string  `json:"printer_id"`
	PageCount    int     `json:"page_count"`
	ColorPages   int     `json:"color_pages"`
	DuplexPages  int     `json:"duplex_pages"`
}

// CostConfigListHandler handles listing cost configurations.
func (h *CostHandler) CostConfigListHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgID := r.URL.Query().Get("organization_id")
	if orgID == "" {
		respondError(w, apperrors.New("organization_id is required", http.StatusBadRequest))
		return
	}

	costs, err := h.costRepo.ListCostConfigs(ctx, orgID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to list cost configs", http.StatusInternalServerError))
		return
	}

	response := make([]map[string]interface{}, len(costs))
	for i, c := range costs {
		response[i] = costConfigToResponse(c)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"costs": response,
		"count": len(response),
	})
}

// CostConfigSetHandler handles setting/updating cost configurations.
func (h *CostHandler) CostConfigSetHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgID := r.URL.Query().Get("organization_id")
	if orgID == "" {
		respondError(w, apperrors.New("organization_id is required", http.StatusBadRequest))
		return
	}

	var req CostConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate request
	if req.CostType == "" {
		respondError(w, apperrors.New("cost_type is required", http.StatusBadRequest))
		return
	}
	if req.CostPerPage < 0 {
		respondError(w, apperrors.New("cost_per_page cannot be negative", http.StatusBadRequest))
		return
	}
	if req.Currency == "" {
		req.Currency = "USD"
	}

	// Parse effective_to if provided
	var effectiveTo *time.Time
	if req.EffectiveTo != nil && *req.EffectiveTo != "" {
		t, err := time.Parse(time.RFC3339, *req.EffectiveTo)
		if err != nil {
			respondError(w, apperrors.Wrap(err, "invalid effective_to format", http.StatusBadRequest))
			return
		}
		effectiveTo = &t
	}

	cost := &PrintCost{
		ID:            uuid.New().String(),
		OrganizationID: orgID,
		PrinterID:     req.PrinterID,
		CostType:      req.CostType,
		CostPerPage:   req.CostPerPage,
		Currency:      req.Currency,
		EffectiveFrom: time.Now(),
		EffectiveTo:   effectiveTo,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := h.costRepo.SetCostConfig(ctx, cost); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to set cost config", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusCreated, costConfigToResponse(cost))
}

// CostConfigDeleteHandler handles deleting a cost configuration.
func (h *CostHandler) CostConfigDeleteHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract cost ID from path
	parts := parsePath(r.URL.Path)
	if len(parts) < 2 {
		respondError(w, apperrors.New("invalid cost config path", http.StatusBadRequest))
		return
	}
	costID := parts[1]

	if err := h.costRepo.DeleteCostConfig(ctx, costID); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to delete cost config", http.StatusInternalServerError))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CalculateCostHandler handles cost calculation requests.
func (h *CostHandler) CalculateCostHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgID := r.URL.Query().Get("organization_id")
	if orgID == "" {
		respondError(w, apperrors.New("organization_id is required", http.StatusBadRequest))
		return
	}

	var req CalculateCostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate request
	if req.PrinterID == "" {
		respondError(w, apperrors.New("printer_id is required", http.StatusBadRequest))
		return
	}
	if req.PageCount <= 0 {
		respondError(w, apperrors.New("page_count must be positive", http.StatusBadRequest))
		return
	}

	// Calculate cost using database function
	var calculatedCost float64
	query := `
		SELECT calculate_print_job_cost(
			$1::uuid,
			$2::uuid,
			$3::uuid,
			$4::integer,
			$5::integer,
			$6::integer
		)
	`

	// Generate a temporary job ID for calculation (won't be stored)
	tempJobID := uuid.New().String()
	err := h.db.QueryRow(ctx, query, tempJobID, orgID, req.PrinterID, req.PageCount, req.ColorPages, req.DuplexPages).Scan(&calculatedCost)
	if err != nil {
		// If function doesn't exist, calculate manually
		calculatedCost, err = h.calculateCostManually(ctx, orgID, req.PrinterID, req.PageCount, req.ColorPages, req.DuplexPages)
		if err != nil {
			respondError(w, apperrors.Wrap(err, "failed to calculate cost", http.StatusInternalServerError))
			return
		}
	}

	// Clean up temporary cost entry
	h.db.Exec(ctx, "DELETE FROM print_job_costs WHERE job_id = $1", tempJobID)

	// Get currency
	currency := "USD"
	costConfig, _ := h.costRepo.GetCostConfig(ctx, orgID, req.PrinterID, "monochrome_a4")
	if costConfig != nil {
		currency = costConfig.Currency
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"cost":         calculatedCost,
		"currency":     currency,
		"page_count":   req.PageCount,
		"color_pages":  req.ColorPages,
		"duplex_pages": req.DuplexPages,
		"breakdown": map[string]interface{}{
			"monochrome_pages": req.PageCount - req.ColorPages,
			"color_pages":      req.ColorPages,
			"duplex_savings":   float64(req.DuplexPages) * 0.1, // 10% savings
		},
	})
}

// JobCostHandler handles retrieving cost for a specific job.
func (h *CostHandler) JobCostHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract job ID from path
	parts := parsePath(r.URL.Path)
	if len(parts) < 3 {
		respondError(w, apperrors.New("invalid job cost path", http.StatusBadRequest))
		return
	}
	jobID := parts[2]

	cost, err := h.costRepo.GetJobCost(ctx, jobID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get job cost", http.StatusInternalServerError))
		return
	}

	if cost == nil {
		respondError(w, apperrors.New("job cost not found", http.StatusNotFound))
		return
	}

	respondJSON(w, http.StatusOK, jobCostToResponse(cost))
}

// CostReportHandler handles cost report requests by period.
func (h *CostHandler) CostReportHandler(w http.ResponseWriter, r *http.Request) {
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

	// Default to last 30 days if not specified
	if startDate == "" {
		startDate = time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}

	summaries, err := h.costRepo.GetCostByPeriod(ctx, orgID, startDate, endDate)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get cost report", http.StatusInternalServerError))
		return
	}

	// Calculate totals
	var totalJobs, totalPages, totalColorPages, totalDuplexPages int
	var totalCost float64
	currency := "USD"

	for _, s := range summaries {
		totalJobs += s.TotalJobs
		totalPages += s.TotalPages
		totalColorPages += s.ColorPages
		totalDuplexPages += s.DuplexPages
		totalCost += s.TotalCost
		if s.Currency != "" {
			currency = s.Currency
		}
	}

	response := make([]map[string]interface{}, len(summaries))
	for i, s := range summaries {
		response[i] = map[string]interface{}{
			"date":         s.Date,
			"total_jobs":   s.TotalJobs,
			"total_pages":  s.TotalPages,
			"color_pages":  s.ColorPages,
			"duplex_pages": s.DuplexPages,
			"total_cost":   s.TotalCost,
			"currency":     s.Currency,
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"organization_id":   orgID,
		"start_date":        startDate,
		"end_date":          endDate,
		"summaries":         response,
		"totals": map[string]interface{}{
			"total_jobs":      totalJobs,
			"total_pages":     totalPages,
			"color_pages":     totalColorPages,
			"duplex_pages":    totalDuplexPages,
			"total_cost":      totalCost,
			"currency":        currency,
		},
	})
}

// CostByUserHandler handles cost breakdown by user.
func (h *CostHandler) CostByUserHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgID := r.URL.Query().Get("organization_id")
	userID := r.URL.Query().Get("user_id")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	if orgID == "" {
		respondError(w, apperrors.New("organization_id is required", http.StatusBadRequest))
		return
	}

	// Default to last 30 days if not specified
	if startDate == "" {
		startDate = time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}

	summaries, err := h.costRepo.GetCostByUser(ctx, orgID, userID, startDate, endDate)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get user costs", http.StatusInternalServerError))
		return
	}

	response := make([]map[string]interface{}, len(summaries))
	for i, s := range summaries {
		response[i] = map[string]interface{}{
			"user_id":       s.UserID,
			"user_email":    s.UserEmail,
			"user_name":     s.UserName,
			"total_jobs":    s.TotalJobs,
			"total_pages":   s.TotalPages,
			"color_pages":   s.ColorPages,
			"duplex_pages":  s.DuplexPages,
			"total_cost":    s.TotalCost,
			"currency":      s.Currency,
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"organization_id": orgID,
		"start_date":      startDate,
		"end_date":        endDate,
		"summaries":       response,
		"count":           len(response),
	})
}

// CostByPrinterHandler handles cost breakdown by printer.
func (h *CostHandler) CostByPrinterHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgID := r.URL.Query().Get("organization_id")
	printerID := r.URL.Query().Get("printer_id")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	if orgID == "" {
		respondError(w, apperrors.New("organization_id is required", http.StatusBadRequest))
		return
	}

	// Default to last 30 days if not specified
	if startDate == "" {
		startDate = time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}

	summaries, err := h.costRepo.GetCostByPrinter(ctx, orgID, printerID, startDate, endDate)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get printer costs", http.StatusInternalServerError))
		return
	}

	response := make([]map[string]interface{}, len(summaries))
	for i, s := range summaries {
		response[i] = map[string]interface{}{
			"printer_id":    s.PrinterID,
			"printer_name":  s.PrinterName,
			"total_jobs":    s.TotalJobs,
			"total_pages":   s.TotalPages,
			"color_pages":   s.ColorPages,
			"duplex_pages":  s.DuplexPages,
			"total_cost":    s.TotalCost,
			"currency":      s.Currency,
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"organization_id": orgID,
		"start_date":      startDate,
		"end_date":        endDate,
		"summaries":       response,
		"count":           len(response),
	})
}

// BudgetAllocationHandler handles budget allocation and tracking.
func (h *CostHandler) BudgetAllocationHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		h.getBudgetAllocations(w, r, ctx)
	case http.MethodPost:
		h.setBudgetAllocation(w, r, ctx)
	case http.MethodPut:
		h.updateBudgetAllocation(w, r, ctx)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *CostHandler) getBudgetAllocations(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	orgID := r.URL.Query().Get("organization_id")
	if orgID == "" {
		respondError(w, apperrors.New("organization_id is required", http.StatusBadRequest))
		return
	}

	// Query budget allocations
	query := `
		SELECT id, organization_id, cost_center_id, cost_center_name,
		       budget_amount, spent_amount, currency, period_start, period_end
		FROM budget_allocations
		WHERE organization_id = $1::uuid
		ORDER BY period_start DESC
	`

	rows, err := h.db.Query(ctx, query, orgID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get budget allocations", http.StatusInternalServerError))
		return
	}
	defer rows.Close()

	allocations := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id, orgID, costCenterID, costCenterName, currency string
		var budgetAmount, spentAmount float64
		var periodStart, periodEnd time.Time

		if err := rows.Scan(&id, &orgID, &costCenterID, &costCenterName, &budgetAmount, &spentAmount, &currency, &periodStart, &periodEnd); err != nil {
			continue
		}

		remaining := budgetAmount - spentAmount
		percentageUsed := 0.0
		if budgetAmount > 0 {
			percentageUsed = (spentAmount / budgetAmount) * 100
		}

		allocations = append(allocations, map[string]interface{}{
			"id":              id,
			"cost_center_id":  costCenterID,
			"cost_center_name": costCenterName,
			"budget_amount":   budgetAmount,
			"spent_amount":    spentAmount,
			"remaining":       remaining,
			"percentage_used": percentageUsed,
			"currency":        currency,
			"period_start":    periodStart.Format(time.RFC3339),
			"period_end":      periodEnd.Format(time.RFC3339),
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"allocations": allocations,
		"count":       len(allocations),
	})
}

func (h *CostHandler) setBudgetAllocation(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	var req struct {
		OrganizationID string  `json:"organization_id"`
		CostCenterID   string  `json:"cost_center_id"`
		CostCenterName string  `json:"cost_center_name"`
		BudgetAmount   float64 `json:"budget_amount"`
		Currency       string  `json:"currency"`
		PeriodStart    string  `json:"period_start"`
		PeriodEnd      string  `json:"period_end"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate request
	if req.OrganizationID == "" {
		respondError(w, apperrors.New("organization_id is required", http.StatusBadRequest))
		return
	}
	if req.CostCenterID == "" {
		respondError(w, apperrors.New("cost_center_id is required", http.StatusBadRequest))
		return
	}
	if req.BudgetAmount <= 0 {
		respondError(w, apperrors.New("budget_amount must be positive", http.StatusBadRequest))
		return
	}
	if req.Currency == "" {
		req.Currency = "USD"
	}

	// Create budget allocation table if not exists
	initQuery := `
		CREATE TABLE IF NOT EXISTS budget_allocations (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			cost_center_id VARCHAR(100) NOT NULL,
			cost_center_name VARCHAR(255),
			budget_amount DECIMAL(12, 2) NOT NULL,
			spent_amount DECIMAL(12, 2) DEFAULT 0,
			currency VARCHAR(3) DEFAULT 'USD',
			period_start TIMESTAMPTZ NOT NULL,
			period_end TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE(organization_id, cost_center_id, period_start)
		);
	`
	h.db.Exec(ctx, initQuery)

	// Parse dates
	periodStart, err := time.Parse(time.RFC3339, req.PeriodStart)
	if err != nil {
		periodStart = time.Now().Truncate(24 * time.Hour)
	}
	periodEnd, err := time.Parse(time.RFC3339, req.PeriodEnd)
	if err != nil {
		periodEnd = periodStart.AddDate(0, 1, 0)
	}

	// Insert budget allocation
	id := uuid.New().String()
	query := `
		INSERT INTO budget_allocations (
			id, organization_id, cost_center_id, cost_center_name,
			budget_amount, spent_amount, currency, period_start, period_end
		) VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (organization_id, cost_center_id, period_start)
		DO UPDATE SET
			budget_amount = EXCLUDED.budget_amount,
			cost_center_name = EXCLUDED.cost_center_name,
			period_end = EXCLUDED.period_end,
			updated_at = NOW()
		RETURNING id
	`

	err = h.db.QueryRow(ctx, query, id, req.OrganizationID, req.CostCenterID, req.CostCenterName,
		req.BudgetAmount, 0, req.Currency, periodStart, periodEnd).Scan(&id)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to set budget allocation", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":              id,
		"cost_center_id":  req.CostCenterID,
		"cost_center_name": req.CostCenterName,
		"budget_amount":   req.BudgetAmount,
		"currency":        req.Currency,
		"period_start":    periodStart.Format(time.RFC3339),
		"period_end":      periodEnd.Format(time.RFC3339),
	})
}

func (h *CostHandler) updateBudgetAllocation(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	// Extract allocation ID from path
	parts := parsePath(r.URL.Path)
	if len(parts) < 2 {
		respondError(w, apperrors.New("invalid budget allocation path", http.StatusBadRequest))
		return
	}
	allocationID := parts[1]

	var req struct {
		BudgetAmount float64 `json:"budget_amount"`
		AddSpent     float64 `json:"add_spent"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	query := `
		UPDATE budget_allocations
		SET budget_amount = COALESCE($2, budget_amount),
		    spent_amount = spent_amount + COALESCE($3, 0),
		    updated_at = NOW()
		WHERE id = $1::uuid
		RETURNING id, budget_amount, spent_amount, currency
	`

	var id string
	var budgetAmount, spentAmount float64
	var currency string
	err := h.db.QueryRow(ctx, query, allocationID,
		nullIfZero(req.BudgetAmount), nullIfZero(req.AddSpent)).
		Scan(&id, &budgetAmount, &spentAmount, &currency)

	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to update budget allocation", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":           id,
		"budget_amount": budgetAmount,
		"spent_amount":  spentAmount,
		"remaining":    budgetAmount - spentAmount,
		"currency":     currency,
	})
}

// calculateCostManually calculates print cost when database function is unavailable.
func (h *CostHandler) calculateCostManually(ctx context.Context, orgID, printerID string, pageCount, colorPages, duplexPages int) (float64, error) {
	// Get monochrome cost
	var monoCost, colorCost float64
	monoQuery := `
		SELECT cost_per_page
		FROM print_costs
		WHERE (organization_id = $1::uuid OR organization_id IS NULL)
		  AND (printer_id = $2::uuid OR printer_id IS NULL)
		  AND cost_type = 'monochrome_a4'
		  AND effective_from <= NOW()
		  AND (effective_to IS NULL OR effective_to > NOW())
		ORDER BY organization_id DESC, printer_id DESC
		LIMIT 1
	`
	_ = h.db.QueryRow(ctx, monoQuery, orgID, printerID).Scan(&monoCost)

	// Get color cost
	colorQuery := `
		SELECT cost_per_page
		FROM print_costs
		WHERE (organization_id = $1::uuid OR organization_id IS NULL)
		  AND (printer_id = $2::uuid OR printer_id IS NULL)
		  AND cost_type = 'color_a4'
		  AND effective_from <= NOW()
		  AND (effective_to IS NULL OR effective_to > NOW())
		ORDER BY organization_id DESC, printer_id DESC
		LIMIT 1
	`
	_ = h.db.QueryRow(ctx, colorQuery, orgID, printerID).Scan(&colorCost)

	// Calculate cost
	cost := monoCost * float64(pageCount-colorPages)
	cost += colorCost * float64(colorPages)

	// Apply duplex savings
	if duplexPages > 0 {
		cost = cost * (1 - 0.1) // 10% savings
	}

	return cost, nil
}

// Helper functions

func costConfigToResponse(c *PrintCost) map[string]interface{} {
	resp := map[string]interface{}{
		"id":              c.ID,
		"organization_id": c.OrganizationID,
		"cost_type":       c.CostType,
		"cost_per_page":   c.CostPerPage,
		"currency":        c.Currency,
		"effective_from":  c.EffectiveFrom.Format(time.RFC3339),
		"created_at":      c.CreatedAt.Format(time.RFC3339),
		"updated_at":      c.UpdatedAt.Format(time.RFC3339),
	}
	if c.PrinterID != "" {
		resp["printer_id"] = c.PrinterID
	}
	if c.EffectiveTo != nil {
		resp["effective_to"] = c.EffectiveTo.Format(time.RFC3339)
	}
	return resp
}

func jobCostToResponse(c *JobCost) map[string]interface{} {
	return map[string]interface{}{
		"id":            c.ID,
		"job_id":        c.JobID,
		"page_count":    c.PageCount,
		"color_pages":   c.ColorPages,
		"duplex_pages":  c.DuplexPages,
		"cost":          c.Cost,
		"currency":      c.Currency,
		"calculated_at": c.CalculatedAt.Format(time.RFC3339),
	}
}

func nullIfZero(f float64) interface{} {
	if f == 0 {
		return nil
	}
	return f
}

