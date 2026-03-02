// Package repository provides printer data access with tenant support.
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a record is not found.
var ErrNotFound = fmt.Errorf("record not found")

// PrinterRepositoryWithTenant extends PrinterRepository with tenant-aware methods.
type PrinterRepositoryWithTenant struct {
	db *pgxpool.Pool
}

// NewPrinterRepositoryWithTenant creates a new tenant-aware printer repository.
func NewPrinterRepositoryWithTenant(db *pgxpool.Pool) *PrinterRepositoryWithTenant {
	return &PrinterRepositoryWithTenant{db: db}
}

// ListByTenant retrieves printers for a specific tenant with pagination.
func (r *PrinterRepositoryWithTenant) ListByTenant(ctx context.Context, tenantID string, limit, offset int, status string) ([]*Printer, int, error) {
	whereClause := "WHERE organization_id = $1"
	args := []interface{}{tenantID}
	argCount := 2

	if status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
		argCount++
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM printers " + whereClause
	var total int
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count printers by tenant: %w", err)
	}

	// Get paginated results
	query := `
		SELECT id, name, agent_id, organization_id, status, capabilities, created_at, updated_at
		FROM printers
		` + whereClause + `
		ORDER BY name ASC
		LIMIT $` + fmt.Sprint(argCount) + ` OFFSET $` + fmt.Sprint(argCount+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list printers by tenant: %w", err)
	}
	defer rows.Close()

	printers := []*Printer{}
	for rows.Next() {
		printer, err := r.scanPrinter(rows)
		if err != nil {
			return nil, 0, err
		}
		printers = append(printers, printer)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate printers: %w", err)
	}

	return printers, total, nil
}

// CreateWithTenant creates a printer with explicit tenant context.
// This method sets the tenant context in the database session for RLS.
func (r *PrinterRepositoryWithTenant) CreateWithTenant(ctx context.Context, printer *Printer, tenantID string) error {
	// Set tenant context for RLS
	if _, err := r.db.Exec(ctx, "SET LOCAL app.tenant_id = $1", tenantID); err != nil {
		return fmt.Errorf("set tenant context: %w", err)
	}

	now := time.Now()
	printer.CreatedAt = now
	printer.UpdatedAt = now
	printer.OrganizationID = tenantID

	// Handle empty capabilities
	capabilities := printer.Capabilities
	if capabilities == "" {
		capabilities = "{}"
	}

	query := `
		INSERT INTO printers (id, name, agent_id, organization_id, status, capabilities, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	err := r.db.QueryRow(ctx, query,
		printer.ID,
		printer.Name,
		printer.AgentID,
		printer.OrganizationID,
		printer.Status,
		capabilities,
		printer.CreatedAt,
		printer.UpdatedAt,
	).Scan(&printer.ID)

	// Clear tenant context
	_, _ = r.db.Exec(ctx, "SET LOCAL app.tenant_id = NULL")

	if err != nil {
		return fmt.Errorf("create printer with tenant: %w", err)
	}

	return nil
}

// FindByTenant retrieves a printer by ID within a tenant context.
func (r *PrinterRepositoryWithTenant) FindByTenant(ctx context.Context, printerID, tenantID string) (*Printer, error) {
	// Set tenant context for RLS
	if _, err := r.db.Exec(ctx, "SET LOCAL app.tenant_id = $1", tenantID); err != nil {
		return nil, fmt.Errorf("set tenant context: %w", err)
	}

	query := `
		SELECT id, name, agent_id, organization_id, status, capabilities, created_at, updated_at
		FROM printers
		WHERE id = $1
	`

	printer, err := r.scanPrinter(r.db.QueryRow(ctx, query, printerID))

	// Clear tenant context
	_, _ = r.db.Exec(ctx, "SET LOCAL app.tenant_id = NULL")

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find printer by tenant: %w", err)
	}

	// Verify the printer belongs to the tenant
	if printer.OrganizationID != tenantID {
		return nil, ErrNotFound
	}

	return printer, nil
}

// UpdateWithTenant updates a printer within a tenant context.
func (r *PrinterRepositoryWithTenant) UpdateWithTenant(ctx context.Context, printer *Printer, tenantID string) error {
	// Set tenant context for RLS
	if _, err := r.db.Exec(ctx, "SET LOCAL app.tenant_id = $1", tenantID); err != nil {
		return fmt.Errorf("set tenant context: %w", err)
	}

	printer.UpdatedAt = time.Now()

	query := `
		UPDATE printers
		SET name = $2, status = $3, capabilities = $4, updated_at = $5
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query,
		printer.ID,
		printer.Name,
		printer.Status,
		printer.Capabilities,
		printer.UpdatedAt,
	)

	// Clear tenant context
	_, _ = r.db.Exec(ctx, "SET LOCAL app.tenant_id = NULL")

	if err != nil {
		return fmt.Errorf("update printer with tenant: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// DeleteWithTenant deletes a printer within a tenant context.
func (r *PrinterRepositoryWithTenant) DeleteWithTenant(ctx context.Context, printerID, tenantID string) error {
	// Set tenant context for RLS
	if _, err := r.db.Exec(ctx, "SET LOCAL app.tenant_id = $1", tenantID); err != nil {
		return fmt.Errorf("set tenant context: %w", err)
	}

	query := `DELETE FROM printers WHERE id = $1`

	result, err := r.db.Exec(ctx, query, printerID)

	// Clear tenant context
	_, _ = r.db.Exec(ctx, "SET LOCAL app.tenant_id = NULL")

	if err != nil {
		return fmt.Errorf("delete printer with tenant: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// CountByTenant returns the number of printers for a tenant.
func (r *PrinterRepositoryWithTenant) CountByTenant(ctx context.Context, tenantID string) (int, error) {
	query := `SELECT COUNT(*) FROM printers WHERE organization_id = $1`

	var count int
	err := r.db.QueryRow(ctx, query, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count printers by tenant: %w", err)
	}

	return count, nil
}

// scanPrinter scans a row into a Printer struct.
func (r *PrinterRepositoryWithTenant) scanPrinter(row pgx.Row) (*Printer, error) {
	var p Printer
	err := row.Scan(
		&p.ID,
		&p.Name,
		&p.AgentID,
		&p.OrganizationID,
		&p.Status,
		&p.Capabilities,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}
