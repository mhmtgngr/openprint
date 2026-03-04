// Package processor provides data aggregation functions for analytics.
package processor

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Aggregator handles data aggregation for analytics.
type Aggregator struct {
	db *pgxpool.Pool
}

// NewAggregator creates a new aggregator.
func NewAggregator(db *pgxpool.Pool) *Aggregator {
	return &Aggregator{db: db}
}

// JobsStats holds aggregated job statistics.
type JobsStats struct {
	TotalJobs          int             `json:"total_jobs"`
	CompletedJobs      int             `json:"completed_jobs"`
	FailedJobs         int             `json:"failed_jobs"`
	PendingJobs        int             `json:"pending_jobs"`
	TotalPages         int             `json:"total_pages"`
	AveragePagesPerJob float64         `json:"average_pages_per_job"`
	StatusBreakdown    map[string]int  `json:"status_breakdown"`
	DailyTrends        []DailyJobCount `json:"daily_trends"`
	Trends             Trends          `json:"trends"`
}

// DailyJobCount represents job count for a single day.
type DailyJobCount struct {
	Date  time.Time `json:"date"`
	Count int       `json:"count"`
	Pages int       `json:"pages"`
}

// Trends shows percentage changes over time.
type Trends struct {
	JobsChangePercent  float64 `json:"jobs_change_percent"`
	PagesChangePercent float64 `json:"pages_change_percent"`
}

// PrinterStats holds aggregated printer statistics.
type PrinterStats struct {
	TotalPrinters      int            `json:"total_printers"`
	OnlinePrinters     int            `json:"online_printers"`
	OfflinePrinters    int            `json:"offline_printers"`
	TopPrinters        []PrinterUsage `json:"top_printers"`
	StatusDistribution map[string]int `json:"status_distribution"`
	TotalJobsProcessed int            `json:"total_jobs_processed"`
}

// PrinterUsage represents usage statistics for a single printer.
type PrinterUsage struct {
	PrinterID   string `json:"printer_id"`
	PrinterName string `json:"printer_name"`
	JobCount    int    `json:"job_count"`
	PageCount   int    `json:"page_count"`
}

// UserStats holds aggregated user statistics.
type UserStats struct {
	TotalUsers       int            `json:"total_users"`
	ActiveUsersCount int            `json:"active_users_count"`
	TopUsers         []UserActivity `json:"top_users"`
	TotalJobsByUsers int            `json:"total_jobs_by_users"`
}

// UserActivity represents activity statistics for a single user.
type UserActivity struct {
	UserEmail string `json:"user_email"`
	JobCount  int    `json:"job_count"`
	PageCount int    `json:"page_count"`
}

// AggregateJobsByDate aggregates job statistics by date.
func (a *Aggregator) AggregateJobsByDate(ctx context.Context, days int) (*JobsStats, error) {
	cutoffDate := time.Now().AddDate(0, 0, -days)

	query := `
		SELECT
			COUNT(*) as total_jobs,
			COUNT(*) FILTER (WHERE status = 'completed') as completed_jobs,
			COUNT(*) FILTER (WHERE status = 'failed') as failed_jobs,
			COUNT(*) FILTER (WHERE status = 'queued' OR status = 'processing' OR status = 'pending_agent') as pending_jobs,
			COALESCE(SUM(pages), 0) as total_pages
		FROM print_jobs
		WHERE created_at >= $1
	`

	var stats JobsStats
	err := a.db.QueryRow(ctx, query, cutoffDate).Scan(
		&stats.TotalJobs,
		&stats.CompletedJobs,
		&stats.FailedJobs,
		&stats.PendingJobs,
		&stats.TotalPages,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate jobs: %w", err)
	}

	if stats.TotalJobs > 0 {
		stats.AveragePagesPerJob = float64(stats.TotalPages) / float64(stats.TotalJobs)
	}

	return &stats, nil
}

// GetJobsByStatus retrieves job counts grouped by status.
func (a *Aggregator) GetJobsByStatus(ctx context.Context) (map[string]int, error) {
	query := `
		SELECT status, COUNT(*) as count
		FROM print_jobs
		GROUP BY status
	`

	rows, err := a.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs by status: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan status row: %w", err)
		}
		result[status] = count
	}

	return result, nil
}

// GetDailyJobCounts retrieves daily job counts for the specified period.
func (a *Aggregator) GetDailyJobCounts(ctx context.Context, days int) ([]DailyJobCount, error) {
	cutoffDate := time.Now().AddDate(0, 0, -days)

	query := `
		SELECT
			DATE(created_at) as date,
			COUNT(*) as count,
			COALESCE(SUM(pages), 0) as pages
		FROM print_jobs
		WHERE created_at >= $1
		GROUP BY DATE(created_at)
		ORDER BY date ASC
	`

	rows, err := a.db.Query(ctx, query, cutoffDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily job counts: %w", err)
	}
	defer rows.Close()

	var result []DailyJobCount
	for rows.Next() {
		var stat DailyJobCount
		if err := rows.Scan(&stat.Date, &stat.Count, &stat.Pages); err != nil {
			return nil, fmt.Errorf("failed to scan daily count: %w", err)
		}
		result = append(result, stat)
	}

	return result, nil
}

// AggregateByPrinter aggregates printer usage statistics.
func (a *Aggregator) AggregateByPrinter(ctx context.Context, days int) (*PrinterStats, error) {
	cutoffDate := time.Now().AddDate(0, 0, -days)

	// Get printer counts
	countQuery := `
		SELECT
			COUNT(*) as total_printers,
			COUNT(*) FILTER (WHERE status = 'online') as online_printers,
			COUNT(*) FILTER (WHERE status = 'offline') as offline_printers
		FROM printers
	`

	var stats PrinterStats
	err := a.db.QueryRow(ctx, countQuery).Scan(
		&stats.TotalPrinters,
		&stats.OnlinePrinters,
		&stats.OfflinePrinters,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get printer counts: %w", err)
	}

	// Get top printers by job count
	topPrintersQuery := `
		SELECT
			p.id as printer_id,
			p.name as printer_name,
			COUNT(pj.id) as job_count,
			COALESCE(SUM(pj.pages), 0) as page_count
		FROM printers p
		LEFT JOIN print_jobs pj ON p.id = pj.printer_id AND pj.created_at >= $1
		GROUP BY p.id, p.name
		ORDER BY job_count DESC
		LIMIT 10
	`

	rows, err := a.db.Query(ctx, topPrintersQuery, cutoffDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get top printers: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var pu PrinterUsage
		if err := rows.Scan(&pu.PrinterID, &pu.PrinterName, &pu.JobCount, &pu.PageCount); err != nil {
			return nil, fmt.Errorf("failed to scan printer usage: %w", err)
		}
		stats.TopPrinters = append(stats.TopPrinters, pu)
		stats.TotalJobsProcessed += pu.JobCount
	}

	return &stats, nil
}

// GetPrinterStatusDistribution retrieves printer counts grouped by status.
func (a *Aggregator) GetPrinterStatusDistribution(ctx context.Context) (map[string]int, error) {
	query := `
		SELECT status, COUNT(*) as count
		FROM printers
		GROUP BY status
	`

	rows, err := a.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get printer status distribution: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan status row: %w", err)
		}
		result[status] = count
	}

	return result, nil
}

// AggregateByUser aggregates user activity statistics.
func (a *Aggregator) AggregateByUser(ctx context.Context, days int, limit int) (*UserStats, error) {
	cutoffDate := time.Now().AddDate(0, 0, -days)

	// Get total users count
	var totalUsers int
	err := a.db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE is_active = true").Scan(&totalUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to get total users: %w", err)
	}

	// Get top users by job count
	topUsersQuery := `
		SELECT
			pj.user_email,
			COUNT(*) as job_count,
			COALESCE(SUM(pj.pages), 0) as page_count
		FROM print_jobs pj
		WHERE pj.created_at >= $1
		GROUP BY pj.user_email
		ORDER BY job_count DESC
		LIMIT $2
	`

	rows, err := a.db.Query(ctx, topUsersQuery, cutoffDate, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top users: %w", err)
	}
	defer rows.Close()

	stats := &UserStats{
		TotalUsers: totalUsers,
	}

	for rows.Next() {
		var ua UserActivity
		if err := rows.Scan(&ua.UserEmail, &ua.JobCount, &ua.PageCount); err != nil {
			return nil, fmt.Errorf("failed to scan user activity: %w", err)
		}
		stats.TopUsers = append(stats.TopUsers, ua)
		stats.TotalJobsByUsers += ua.JobCount
	}

	return stats, nil
}

// GetActiveUsersCount retrieves the count of users who submitted jobs in the specified period.
func (a *Aggregator) GetActiveUsersCount(ctx context.Context, days int) (int, error) {
	cutoffDate := time.Now().AddDate(0, 0, -days)

	query := `
		SELECT COUNT(DISTINCT user_email)
		FROM print_jobs
		WHERE created_at >= $1
	`

	var count int
	err := a.db.QueryRow(ctx, query, cutoffDate).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get active users count: %w", err)
	}

	return count, nil
}
