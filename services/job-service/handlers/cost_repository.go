// Package handler provides cost repository implementation.
package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// costRepository implements the CostRepository interface.
type costRepository struct {
	db *pgxpool.Pool
}

// NewCostRepository creates a new cost repository.
func NewCostRepository(db *pgxpool.Pool) CostRepository {
	return &costRepository{db: db}
}

// GetCostConfig retrieves a specific cost configuration.
func (r *costRepository) GetCostConfig(ctx context.Context, organizationID, printerID, costType string) (*PrintCost, error) {
	query := `
		SELECT id, organization_id, printer_id, cost_type, cost_per_page,
		       currency, effective_from, effective_to, created_at, updated_at
		FROM print_costs
		WHERE (organization_id = $1::uuid OR organization_id IS NULL)
		  AND (printer_id = $2::uuid OR printer_id IS NULL)
		  AND cost_type = COALESCE($3, cost_type)
		  AND effective_from <= NOW()
		  AND (effective_to IS NULL OR effective_to > NOW())
		ORDER BY organization_id DESC, printer_id DESC
		LIMIT 1
	`

	var cost PrintCost
	err := r.db.QueryRow(ctx, query, nullIfEmpty(organizationID), nullIfEmpty(printerID), nullIfEmpty(costType)).Scan(
		&cost.ID, &cost.OrganizationID, &cost.PrinterID, &cost.CostType,
		&cost.CostPerPage, &cost.Currency, &cost.EffectiveFrom,
		&cost.EffectiveTo, &cost.CreatedAt, &cost.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get cost config: %w", err)
	}

	return &cost, nil
}

// ListCostConfigs retrieves all cost configurations for an organization.
func (r *costRepository) ListCostConfigs(ctx context.Context, organizationID string) ([]*PrintCost, error) {
	query := `
		SELECT id, organization_id, printer_id, cost_type, cost_per_page,
		       currency, effective_from, effective_to, created_at, updated_at
		FROM print_costs
		WHERE organization_id = $1::uuid
		  AND (effective_to IS NULL OR effective_to > NOW())
		ORDER BY cost_type, printer_id
	`

	rows, err := r.db.Query(ctx, query, organizationID)
	if err != nil {
		return nil, fmt.Errorf("list cost configs: %w", err)
	}
	defer rows.Close()

	var costs []*PrintCost
	for rows.Next() {
		var cost PrintCost
		if err := rows.Scan(
			&cost.ID, &cost.OrganizationID, &cost.PrinterID, &cost.CostType,
			&cost.CostPerPage, &cost.Currency, &cost.EffectiveFrom,
			&cost.EffectiveTo, &cost.CreatedAt, &cost.UpdatedAt,
		); err != nil {
			return nil, err
		}
		costs = append(costs, &cost)
	}

	return costs, nil
}

// SetCostConfig creates or updates a cost configuration.
func (r *costRepository) SetCostConfig(ctx context.Context, cost *PrintCost) error {
	query := `
		INSERT INTO print_costs (
			id, organization_id, printer_id, cost_type, cost_per_page,
			currency, effective_from, effective_to, created_at, updated_at
		) VALUES (
			$1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8, $9, $10
		)
	`

	_, err := r.db.Exec(ctx, query,
		cost.ID, cost.OrganizationID, nullIfEmpty(cost.PrinterID),
		cost.CostType, cost.CostPerPage, cost.Currency,
		cost.EffectiveFrom, cost.EffectiveTo,
		cost.CreatedAt, cost.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("set cost config: %w", err)
	}

	return nil
}

// DeleteCostConfig removes a cost configuration.
func (r *costRepository) DeleteCostConfig(ctx context.Context, costID string) error {
	query := `DELETE FROM print_costs WHERE id = $1::uuid`

	cmdTag, err := r.db.Exec(ctx, query, costID)
	if err != nil {
		return fmt.Errorf("delete cost config: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("cost config not found")
	}

	return nil
}

// CalculateJobCost calculates and stores the cost for a print job.
func (r *costRepository) CalculateJobCost(ctx context.Context, jobID string) (*JobCost, error) {
	// First get job details to calculate cost
	var printerID, organizationID string
	var pageCount int
	query := `
		SELECT j.printer_id, u.organization_id, COALESCE(j.pages, 0)
		FROM print_jobs j
		LEFT JOIN users u ON u.email = j.user_email
		WHERE j.id = $1::uuid
	`

	err := r.db.QueryRow(ctx, query, jobID).Scan(&printerID, &organizationID, &pageCount)
	if err != nil {
		return nil, fmt.Errorf("get job details: %w", err)
	}

	// Get costs
	monoCost, _ := r.GetCostConfig(ctx, organizationID, printerID, "monochrome_a4")
	_, _ = r.GetCostConfig(ctx, organizationID, printerID, "color_a4")

	// For simplicity, assume all pages are monochrome
	costPerPage := 0.0
	if monoCost != nil {
		costPerPage = monoCost.CostPerPage
	}

	totalCost := float64(pageCount) * costPerPage

	// Store the calculated cost
	cost := &JobCost{
		ID:       uuid.New().String(),
		JobID:    jobID,
		PageCount: pageCount,
		Cost:     totalCost,
		Currency: "USD",
		CalculatedAt: time.Now(),
	}

	storeQuery := `
		INSERT INTO print_job_costs (id, job_id, page_count, cost, currency, calculated_at)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6)
		ON CONFLICT (job_id) DO UPDATE
		SET page_count = EXCLUDED.page_count,
		    cost = EXCLUDED.cost,
		    calculated_at = EXCLUDED.calculated_at
	`

	_, err = r.db.Exec(ctx, storeQuery, cost.ID, cost.JobID, cost.PageCount, cost.Cost, cost.Currency, cost.CalculatedAt)
	if err != nil {
		return nil, fmt.Errorf("store job cost: %w", err)
	}

	return cost, nil
}

// GetJobCost retrieves the cost for a specific job.
func (r *costRepository) GetJobCost(ctx context.Context, jobID string) (*JobCost, error) {
	query := `
		SELECT id, job_id, page_count, color_pages, duplex_pages, cost, currency, calculated_at
		FROM print_job_costs
		WHERE job_id = $1::uuid
	`

	var cost JobCost
	err := r.db.QueryRow(ctx, query, jobID).Scan(
		&cost.ID, &cost.JobID, &cost.PageCount, &cost.ColorPages,
		&cost.DuplexPages, &cost.Cost, &cost.Currency, &cost.CalculatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get job cost: %w", err)
	}

	return &cost, nil
}

// GetCostByPeriod retrieves cost data aggregated by date.
func (r *costRepository) GetCostByPeriod(ctx context.Context, organizationID, startDate, endDate string) ([]*CostSummary, error) {
	query := `
		SELECT DATE(j.created_at)::text as date,
		       COUNT(*) as total_jobs,
		       COALESCE(SUM(j.copies), 0) as total_pages,
		       0 as color_pages,
		       0 as duplex_pages,
		       COALESCE(SUM(c.cost), 0) as total_cost,
		       COALESCE(c.currency, 'USD') as currency
		FROM print_jobs j
		LEFT JOIN print_job_costs c ON c.job_id = j.id
		LEFT JOIN users u ON u.email = j.user_email
		WHERE u.organization_id = $1::uuid
		  AND DATE(j.created_at) >= $2::date
		  AND DATE(j.created_at) <= $3::date
		GROUP BY DATE(j.created_at), c.currency
		ORDER BY date
	`

	rows, err := r.db.Query(ctx, query, organizationID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("get cost by period: %w", err)
	}
	defer rows.Close()

	var summaries []*CostSummary
	for rows.Next() {
		var summary CostSummary
		if err := rows.Scan(&summary.Date, &summary.TotalJobs, &summary.TotalPages,
			&summary.ColorPages, &summary.DuplexPages, &summary.TotalCost, &summary.Currency); err != nil {
			return nil, err
		}
		summaries = append(summaries, &summary)
	}

	return summaries, nil
}

// GetCostByUser retrieves cost data aggregated by user.
func (r *costRepository) GetCostByUser(ctx context.Context, organizationID, userID, startDate, endDate string) ([]*UserCostSummary, error) {
	query := `
		SELECT
		    u.id as user_id,
		    u.email as user_email,
		    CONCAT(u.first_name, ' ', u.last_name) as user_name,
		    COUNT(j.id) as total_jobs,
		    COALESCE(SUM(j.copies), 0) as total_pages,
		    0 as color_pages,
		    0 as duplex_pages,
		    COALESCE(SUM(c.cost), 0) as total_cost,
		    COALESCE(c.currency, 'USD') as currency
		FROM users u
		LEFT JOIN print_jobs j ON j.user_email = u.email
		LEFT JOIN print_job_costs c ON c.job_id = j.id
		WHERE u.organization_id = $1::uuid
		  AND ($2::uuid = ''::uuid OR u.id = $2::uuid)
		  AND (j.created_at IS NULL OR DATE(j.created_at) >= $3::date)
		  AND (j.created_at IS NULL OR DATE(j.created_at) <= $4::date)
		GROUP BY u.id, u.email, u.first_name, u.last_name, c.currency
		ORDER BY total_cost DESC
	`

	rows, err := r.db.Query(ctx, query, organizationID, nullIfEmpty(userID), startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("get cost by user: %w", err)
	}
	defer rows.Close()

	var summaries []*UserCostSummary
	for rows.Next() {
		var summary UserCostSummary
		if err := rows.Scan(&summary.UserID, &summary.UserEmail, &summary.UserName,
			&summary.TotalJobs, &summary.TotalPages, &summary.ColorPages,
			&summary.DuplexPages, &summary.TotalCost, &summary.Currency); err != nil {
			return nil, err
		}
		summaries = append(summaries, &summary)
	}

	return summaries, nil
}

// GetCostByPrinter retrieves cost data aggregated by printer.
func (r *costRepository) GetCostByPrinter(ctx context.Context, organizationID, printerID, startDate, endDate string) ([]*PrinterCostSummary, error) {
	query := `
		SELECT
		    p.id as printer_id,
		    p.name as printer_name,
		    COUNT(j.id) as total_jobs,
		    COALESCE(SUM(j.copies), 0) as total_pages,
		    0 as color_pages,
		    0 as duplex_pages,
		    COALESCE(SUM(c.cost), 0) as total_cost,
		    COALESCE(c.currency, 'USD') as currency
		FROM printers p
		LEFT JOIN print_jobs j ON j.printer_id = p.id
		LEFT JOIN print_job_costs c ON c.job_id = j.id
		WHERE p.organization_id = $1::uuid
		  AND ($2::uuid = ''::uuid OR p.id = $2::uuid)
		  AND (j.created_at IS NULL OR DATE(j.created_at) >= $3::date)
		  AND (j.created_at IS NULL OR DATE(j.created_at) <= $4::date)
		GROUP BY p.id, p.name, c.currency
		ORDER BY total_cost DESC
	`

	rows, err := r.db.Query(ctx, query, organizationID, nullIfEmpty(printerID), startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("get cost by printer: %w", err)
	}
	defer rows.Close()

	var summaries []*PrinterCostSummary
	for rows.Next() {
		var summary PrinterCostSummary
		if err := rows.Scan(&summary.PrinterID, &summary.PrinterName,
			&summary.TotalJobs, &summary.TotalPages, &summary.ColorPages,
			&summary.DuplexPages, &summary.TotalCost, &summary.Currency); err != nil {
			return nil, err
		}
		summaries = append(summaries, &summary)
	}

	return summaries, nil
}

