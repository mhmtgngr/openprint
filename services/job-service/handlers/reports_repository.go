// Package handler provides reports repository implementation.
package handler

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// reportsRepository implements the ReportsRepository interface.
type reportsRepository struct {
	db *pgxpool.Pool
}

// NewReportsRepository creates a new reports repository.
func NewReportsRepository(db *pgxpool.Pool) ReportsRepository {
	return &reportsRepository{db: db}
}

// GetUsageSummary retrieves usage summary for an organization.
func (r *reportsRepository) GetUsageSummary(ctx context.Context, organizationID, startDate, endDate string) (*UsageSummary, error) {
	query := `
		SELECT
			COUNT(*) as total_jobs,
			COALESCE(SUM(j.copies), 0) as total_pages,
			0 as color_pages,
			0 as duplex_pages,
			SUM(CASE WHEN j.status = 'completed' THEN 1 ELSE 0 END) as completed_jobs,
			SUM(CASE WHEN j.status = 'failed' THEN 1 ELSE 0 END) as failed_jobs,
			SUM(CASE WHEN j.status = 'cancelled' THEN 1 ELSE 0 END) as cancelled_jobs,
			COALESCE(AVG(
				CASE WHEN j.completed_at IS NOT NULL AND j.started_at IS NOT NULL
				THEN EXTRACT(EPOCH FROM (j.completed_at - j.started_at))
				ELSE NULL END
			), 0)::integer as avg_job_time,
			COALESCE(SUM(c.cost), 0) as total_cost,
			COALESCE(c.currency, 'USD') as currency
		FROM print_jobs j
		LEFT JOIN users u ON u.email = j.user_email
		LEFT JOIN print_job_costs c ON c.job_id = j.id
		WHERE u.organization_id = $1::uuid
		  AND DATE(j.created_at) >= $2::date
		  AND DATE(j.created_at) <= $3::date
	`

	var summary UsageSummary
	err := r.db.QueryRow(ctx, query, organizationID, startDate, endDate).Scan(
		&summary.TotalJobs, &summary.TotalPages, &summary.ColorPages, &summary.DuplexPages,
		&summary.CompletedJobs, &summary.FailedJobs, &summary.CancelledJobs,
		&summary.AverageJobTime, &summary.TotalCost, &summary.Currency,
	)

	if err != nil {
		return nil, fmt.Errorf("get usage summary: %w", err)
	}

	// Calculate environmental metrics
	// CO2: approximately 0.01 kg per page
	summary.CO2Emission = float64(summary.TotalPages) * 0.01
	// Trees saved: ~8000 pages per tree, duplex saves 50%
	summary.TreesSaved = float64(summary.DuplexPages) / (8000 * 2)

	return &summary, nil
}

// GetTopUsers retrieves top users by print usage.
func (r *reportsRepository) GetTopUsers(ctx context.Context, organizationID string, startDate, endDate string, limit int) ([]*UserUsageStats, error) {
	query := `
		SELECT
			u.id as user_id,
			u.email as user_email,
			CONCAT(u.first_name, ' ', u.last_name) as user_name,
			COUNT(j.id) as total_jobs,
			COALESCE(SUM(j.copies), 0) as total_pages,
			0 as color_pages,
			0 as duplex_pages,
			COALESCE(SUM(c.cost), 0) as total_cost
		FROM users u
		LEFT JOIN print_jobs j ON j.user_email = u.email
		LEFT JOIN print_job_costs c ON c.job_id = j.id
		WHERE u.organization_id = $1::uuid
		  AND (j.created_at IS NULL OR DATE(j.created_at) >= $2::date)
		  AND (j.created_at IS NULL OR DATE(j.created_at) <= $3::date)
		GROUP BY u.id, u.email, u.first_name, u.last_name
		HAVING COUNT(j.id) > 0
		ORDER BY total_pages DESC
		LIMIT $4
	`

	rows, err := r.db.Query(ctx, query, organizationID, startDate, endDate, limit)
	if err != nil {
		return nil, fmt.Errorf("get top users: %w", err)
	}
	defer rows.Close()

	var users []*UserUsageStats
	rank := 1
	for rows.Next() {
		var user UserUsageStats
		if err := rows.Scan(
			&user.UserID, &user.UserEmail, &user.UserName,
			&user.TotalJobs, &user.TotalPages, &user.ColorPages,
			&user.DuplexPages, &user.TotalCost,
		); err != nil {
			return nil, err
		}
		user.Rank = rank
		rank++
		users = append(users, &user)
	}

	return users, nil
}

// GetTopPrinters retrieves top printers by usage.
func (r *reportsRepository) GetTopPrinters(ctx context.Context, organizationID string, startDate, endDate string, limit int) ([]*PrinterUsageStats, error) {
	query := `
		SELECT
			p.id as printer_id,
			p.name as printer_name,
			COUNT(j.id) as total_jobs,
			COALESCE(SUM(j.copies), 0) as total_pages,
			0 as color_pages,
			0 as duplex_pages,
			COALESCE(SUM(c.cost), 0) as total_cost,
			COALESCE(
				(SELECT COUNT(*) FROM agent_heartbeats ah WHERE ah.agent_id::text = p.agent_id::text
				 AND ah.created_at >= NOW() - INTERVAL '30 days')::decimal /
				NULLIF((SELECT COUNT(DISTINCT DATE(created_at)) FROM agent_heartbeats ah WHERE ah.agent_id::text = p.agent_id::text
					AND ah.created_at >= NOW() - INTERVAL '30 days') * 24, 0) * 100,
				0
			) as uptime,
			0 as error_rate
		FROM printers p
		LEFT JOIN print_jobs j ON j.printer_id = p.id
		LEFT JOIN print_job_costs c ON c.job_id = j.id
		WHERE p.organization_id = $1::uuid
		  AND (j.created_at IS NULL OR DATE(j.created_at) >= $2::date)
		  AND (j.created_at IS NULL OR DATE(j.created_at) <= $3::date)
		GROUP BY p.id, p.name, p.agent_id
		HAVING COUNT(j.id) > 0
		ORDER BY total_pages DESC
		LIMIT $4
	`

	rows, err := r.db.Query(ctx, query, organizationID, startDate, endDate, limit)
	if err != nil {
		return nil, fmt.Errorf("get top printers: %w", err)
	}
	defer rows.Close()

	var printers []*PrinterUsageStats
	rank := 1
	for rows.Next() {
		var printer PrinterUsageStats
		if err := rows.Scan(
			&printer.PrinterID, &printer.PrinterName, &printer.TotalJobs,
			&printer.TotalPages, &printer.ColorPages, &printer.DuplexPages,
			&printer.TotalCost, &printer.Uptime, &printer.ErrorRate,
		); err != nil {
			return nil, err
		}
		printer.Rank = rank
		rank++
		printers = append(printers, &printer)
	}

	return printers, nil
}

// GetDepartmentReport retrieves report data by department/cost center.
func (r *reportsRepository) GetDepartmentReport(ctx context.Context, organizationID string, startDate, endDate string) ([]*DepartmentReport, error) {
	// Using cost_center from user metadata or similar
	query := `
		SELECT
			COAALESCE(u.cost_center, 'unassigned') as department_id,
			COALESCE(u.cost_center, 'Unassigned') as department_name,
			COUNT(j.id) as total_jobs,
			COALESCE(SUM(j.copies), 0) as total_pages,
			COALESCE(SUM(c.cost), 0) as total_cost,
			COALESCE(c.currency, 'USD') as currency,
			COUNT(DISTINCT u.id) as user_count
		FROM users u
		LEFT JOIN print_jobs j ON j.user_email = u.email
		LEFT JOIN print_job_costs c ON c.job_id = j.id
		WHERE u.organization_id = $1::uuid
		  AND (j.created_at IS NULL OR DATE(j.created_at) >= $2::date)
		  AND (j.created_at IS NULL OR DATE(j.created_at) <= $3::date)
		GROUP BY u.cost_center
		HAVING COUNT(j.id) > 0
		ORDER BY total_pages DESC
	`

	// Add cost_center column to users table if it doesn't exist
	r.db.Exec(ctx, `ALTER TABLE users ADD COLUMN IF NOT EXISTS cost_center VARCHAR(100)`)

	rows, err := r.db.Query(ctx, query, organizationID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("get department report: %w", err)
	}
	defer rows.Close()

	var departments []*DepartmentReport
	for rows.Next() {
		var dept DepartmentReport
		if err := rows.Scan(
			&dept.DepartmentID, &dept.DepartmentName, &dept.TotalJobs,
			&dept.TotalPages, &dept.TotalCost, &dept.Currency, &dept.UserCount,
		); err != nil {
			return nil, err
		}
		departments = append(departments, &dept)
	}

	return departments, nil
}

// GetEnvironmentalReport retrieves environmental impact data.
func (r *reportsRepository) GetEnvironmentalReport(ctx context.Context, organizationID string, startDate, endDate string) (*EnvironmentalReport, error) {
	query := `
		SELECT
			COALESCE(SUM(j.copies), 0) as total_pages,
			COALESCE(SUM(CASE WHEN j.duplex = true THEN j.copies ELSE 0 END), 0) as duplex_pages
		FROM print_jobs j
		LEFT JOIN users u ON u.email = j.user_email
		WHERE u.organization_id = $1::uuid
		  AND DATE(j.created_at) >= $2::date
		  AND DATE(j.created_at) <= $3::date
		  AND j.status = 'completed'
	`

	var report EnvironmentalReport
	var totalPages, duplexPages int
	err := r.db.QueryRow(ctx, query, organizationID, startDate, endDate).Scan(&totalPages, &duplexPages)
	if err != nil {
		return nil, fmt.Errorf("get environmental report: %w", err)
	}

	report.TotalPages = totalPages

	// Calculate environmental metrics
	// Trees saved: Each tree = ~8000 pages, duplex saves 50% paper
	report.TreesSaved = float64(duplexPages) / (8000 * 2)

	// CO2 emissions: ~0.01 kg per page
	report.CO2Emission = float64(totalPages) * 0.01

	// CO2 offset from duplex: 50% of the pages would have been additional sheets
	report.CO2Offset = float64(duplexPages) * 0.005

	// Energy consumption: ~0.005 kWh per page
	report.EnergyConsumption = float64(totalPages) * 0.005

	// Waste reduction: duplex reduces paper waste by 50%
	// Average paper weight = 0.005 kg per sheet
	report.WasteReduction = float64(duplexPages) * 0.005 * 0.5

	return &report, nil
}

// GetTrendData retrieves time-series trend data.
func (r *reportsRepository) GetTrendData(ctx context.Context, organizationID string, startDate, endDate string, granularity string) ([]*TrendDataPoint, error) {
	// Determine date truncation based on granularity
	var dateTrunc string
	switch granularity {
	case "hour":
		dateTrunc = "date_trunc('hour', j.created_at)"
	case "week":
		dateTrunc = "date_trunc('week', j.created_at)"
	case "month":
		dateTrunc = "date_trunc('month', j.created_at)"
	default: // day
		dateTrunc = "date_trunc('day', j.created_at)"
		granularity = "day"
	}

	query := fmt.Sprintf(`
		SELECT
			%s::text as date,
			COUNT(*) as jobs,
			COALESCE(SUM(j.copies), 0) as pages,
			COALESCE(SUM(c.cost), 0) as cost,
			COALESCE(SUM(j.copies), 0) * 0.01 as co2
		FROM print_jobs j
		LEFT JOIN users u ON u.email = j.user_email
		LEFT JOIN print_job_costs c ON c.job_id = j.id
		WHERE u.organization_id = $1::uuid
		  AND DATE(j.created_at) >= $2::date
		  AND DATE(j.created_at) <= $3::date
		GROUP BY %s
		ORDER BY date
	`, dateTrunc, dateTrunc)

	rows, err := r.db.Query(ctx, query, organizationID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("get trend data: %w", err)
	}
	defer rows.Close()

	var data []*TrendDataPoint
	for rows.Next() {
		var point TrendDataPoint
		if err := rows.Scan(&point.Date, &point.Jobs, &point.Pages, &point.Cost, &point.CO2); err != nil {
			return nil, err
		}
		data = append(data, &point)
	}

	return data, nil
}
