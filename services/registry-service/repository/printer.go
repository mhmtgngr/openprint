// Package repository provides printer data access for the registry service.
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Printer represents a registered printer.
type Printer struct {
	ID             string
	Name           string
	AgentID        string
	OrganizationID string
	Status         string // "online", "offline", "busy", "error"
	Capabilities   string // JSON string of printer capabilities
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// PrinterRepository handles printer data operations.
type PrinterRepository struct {
	db *pgxpool.Pool
}

// NewPrinterRepository creates a new printer repository.
func NewPrinterRepository(db *pgxpool.Pool) *PrinterRepository {
	return &PrinterRepository{db: db}
}

// Create inserts a new printer.
func (r *PrinterRepository) Create(ctx context.Context, printer *Printer) error {
	now := time.Now()
	printer.CreatedAt = now
	printer.UpdatedAt = now

	// Handle empty capabilities by setting it to a valid empty JSON object
	capabilities := printer.Capabilities
	if capabilities == "" {
		capabilities = "{}"
	}

	query := `
		INSERT INTO printers (id, name, agent_id, organization_id, status, capabilities, created_at, updated_at)
		VALUES ($1, $2, $3, NULLIF($4, '')::uuid, $5, $6, $7, $8)
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

	if err != nil {
		return fmt.Errorf("create printer: %w", err)
	}

	return nil
}

// FindByID retrieves a printer by ID.
func (r *PrinterRepository) FindByID(ctx context.Context, id string) (*Printer, error) {
	query := `
		SELECT id, name, agent_id, organization_id, status, capabilities, created_at, updated_at
		FROM printers
		WHERE id = $1
	`

	printer, err := r.scanPrinter(r.db.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("printer not found")
		}
		return nil, fmt.Errorf("find printer by id: %w", err)
	}

	return printer, nil
}

// FindByAgent retrieves all printers for an agent.
func (r *PrinterRepository) FindByAgent(ctx context.Context, agentID string) ([]*Printer, error) {
	query := `
		SELECT id, name, agent_id, organization_id, status, capabilities, created_at, updated_at
		FROM printers
		WHERE agent_id = $1
		ORDER BY name ASC
	`

	rows, err := r.db.Query(ctx, query, agentID)
	if err != nil {
		return nil, fmt.Errorf("find printers by agent: %w", err)
	}
	defer rows.Close()

	var printers []*Printer
	for rows.Next() {
		printer, err := r.scanPrinter(rows)
		if err != nil {
			return nil, err
		}
		printers = append(printers, printer)
	}

	return printers, rows.Err()
}

// FindByOrganization retrieves all printers for an organization with pagination.
func (r *PrinterRepository) FindByOrganization(ctx context.Context, orgID string, limit, offset int) ([]*Printer, int, error) {
	// Get total count
	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM printers WHERE organization_id = $1", orgID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count printers: %w", err)
	}

	// Get printers
	query := `
		SELECT id, name, agent_id, organization_id, status, capabilities, created_at, updated_at
		FROM printers
		WHERE organization_id = $1
		ORDER BY name ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, orgID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("find printers by organization: %w", err)
	}
	defer rows.Close()

	var printers []*Printer
	for rows.Next() {
		printer, err := r.scanPrinter(rows)
		if err != nil {
			return nil, 0, err
		}
		printers = append(printers, printer)
	}

	return printers, total, rows.Err()
}

// FindByStatus retrieves all printers with a given status.
func (r *PrinterRepository) FindByStatus(ctx context.Context, status string) ([]*Printer, error) {
	query := `
		SELECT id, name, agent_id, organization_id, status, capabilities, created_at, updated_at
		FROM printers
		WHERE status = $1
		ORDER BY name ASC
	`

	rows, err := r.db.Query(ctx, query, status)
	if err != nil {
		return nil, fmt.Errorf("find printers by status: %w", err)
	}
	defer rows.Close()

	var printers []*Printer
	for rows.Next() {
		printer, err := r.scanPrinter(rows)
		if err != nil {
			return nil, err
		}
		printers = append(printers, printer)
	}

	return printers, rows.Err()
}

// List retrieves all printers with pagination.
func (r *PrinterRepository) List(ctx context.Context, limit, offset int) ([]*Printer, int, error) {
	// Get total count
	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM printers").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count printers: %w", err)
	}

	// Get printers
	query := `
		SELECT id, name, agent_id, organization_id, status, capabilities, created_at, updated_at
		FROM printers
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list printers: %w", err)
	}
	defer rows.Close()

	var printers []*Printer
	for rows.Next() {
		printer, err := r.scanPrinter(rows)
		if err != nil {
			return nil, 0, err
		}
		printers = append(printers, printer)
	}

	return printers, total, rows.Err()
}

// Update updates a printer.
func (r *PrinterRepository) Update(ctx context.Context, printer *Printer) error {
	printer.UpdatedAt = time.Now()

	query := `
		UPDATE printers
		SET name = $2, agent_id = $3, organization_id = NULLIF($4, '')::uuid, status = $5, capabilities = $6, updated_at = $7
		WHERE id = $1
	`

	cmdTag, err := r.db.Exec(ctx, query,
		printer.ID,
		printer.Name,
		printer.AgentID,
		printer.OrganizationID,
		printer.Status,
		printer.Capabilities,
		printer.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("update printer: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("printer not found")
	}

	return nil
}

// SetStatus updates a printer's status.
func (r *PrinterRepository) SetStatus(ctx context.Context, id, status string) error {
	query := `
		UPDATE printers
		SET status = $2, updated_at = $3
		WHERE id = $1
	`

	cmdTag, err := r.db.Exec(ctx, query, id, status, time.Now())
	if err != nil {
		return fmt.Errorf("set printer status: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("printer not found")
	}

	return nil
}

// SetStatusByAgent sets status for all printers of an agent.
func (r *PrinterRepository) SetStatusByAgent(ctx context.Context, agentID, status string) (int64, error) {
	query := `
		UPDATE printers
		SET status = $2, updated_at = $3
		WHERE agent_id = $1
	`

	cmdTag, err := r.db.Exec(ctx, query, agentID, status, time.Now())
	if err != nil {
		return 0, fmt.Errorf("set printer status by agent: %w", err)
	}

	return cmdTag.RowsAffected(), nil
}

// Delete removes a printer.
func (r *PrinterRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM printers WHERE id = $1`

	cmdTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete printer: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("printer not found")
	}

	return nil
}

// DeleteByAgent removes all printers for an agent.
func (r *PrinterRepository) DeleteByAgent(ctx context.Context, agentID string) (int64, error) {
	query := `DELETE FROM printers WHERE agent_id = $1`

	cmdTag, err := r.db.Exec(ctx, query, agentID)
	if err != nil {
		return 0, fmt.Errorf("delete printers by agent: %w", err)
	}

	return cmdTag.RowsAffected(), nil
}

// FindAvailable finds all printers that are online.
func (r *PrinterRepository) FindAvailable(ctx context.Context) ([]*Printer, error) {
	return r.FindByStatus(ctx, "online")
}

// UpdateCapabilities updates a printer's capabilities.
func (r *PrinterRepository) UpdateCapabilities(ctx context.Context, id, capabilities string) error {
	query := `
		UPDATE printers
		SET capabilities = $2, updated_at = $3
		WHERE id = $1
	`

	cmdTag, err := r.db.Exec(ctx, query, id, capabilities, time.Now())
	if err != nil {
		return fmt.Errorf("update capabilities: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("printer not found")
	}

	return nil
}

// CountByStatus returns the count of printers by status.
func (r *PrinterRepository) CountByStatus(ctx context.Context, status string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM printers WHERE status = $1", status).Scan(&count)
	return count, err
}

// CountByAgent returns the number of printers for an agent.
func (r *PrinterRepository) CountByAgent(ctx context.Context, agentID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM printers WHERE agent_id = $1", agentID).Scan(&count)
	return count, err
}

// ExistsByID checks if a printer with the given ID exists.
func (r *PrinterRepository) ExistsByID(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM printers WHERE id = $1)", id).Scan(&exists)
	return exists, err
}

// GetPrintersByAgents retrieves all printers for multiple agent IDs.
func (r *PrinterRepository) GetPrintersByAgents(ctx context.Context, agentIDs []string) ([]*Printer, error) {
	if len(agentIDs) == 0 {
		return []*Printer{}, nil
	}

	query := `
		SELECT id, name, agent_id, organization_id, status, capabilities, created_at, updated_at
		FROM printers
		WHERE agent_id = ANY($1)
		ORDER BY name ASC
	`

	rows, err := r.db.Query(ctx, query, agentIDs)
	if err != nil {
		return nil, fmt.Errorf("get printers by agents: %w", err)
	}
	defer rows.Close()

	var printers []*Printer
	for rows.Next() {
		printer, err := r.scanPrinter(rows)
		if err != nil {
			return nil, err
		}
		printers = append(printers, printer)
	}

	return printers, rows.Err()
}

// scanPrinter scans a printer from a database row.
func (r *PrinterRepository) scanPrinter(row interface{ Scan(...interface{}) error }) (*Printer, error) {
	var printer Printer
	// Use a pointer for organization_id to handle NULL values
	var orgID *string
	err := row.Scan(
		&printer.ID,
		&printer.Name,
		&printer.AgentID,
		&orgID,
		&printer.Status,
		&printer.Capabilities,
		&printer.CreatedAt,
		&printer.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	// Convert pointer to string, defaulting to empty string if NULL
	if orgID != nil {
		printer.OrganizationID = *orgID
	}
	return &printer, nil
}
