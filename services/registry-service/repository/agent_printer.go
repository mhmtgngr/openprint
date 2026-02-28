// Package repository provides data access layer for agent-discovered printers.
package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openprint/openprint/internal/agent"
)

// DiscoveredPrinter represents a printer discovered by an agent.
type DiscoveredPrinter struct {
	ID             string
	AgentID        string
	Name           string
	DisplayName    string
	Driver         string
	DriverVersion  string
	Port           string
	ConnectionType agent.PrinterConnectionType
	Status         agent.PrinterStatus
	IsDefault      bool
	IsShared       bool
	ShareName      string
	Location       string
	Capabilities   string // JSON string of agent.PrinterCapabilities
	LastSeen       time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// AgentPrinterRepository handles discovered printer data operations.
type AgentPrinterRepository struct {
	db *pgxpool.Pool
}

// NewAgentPrinterRepository creates a new agent printer repository.
func NewAgentPrinterRepository(db *pgxpool.Pool) *AgentPrinterRepository {
	return &AgentPrinterRepository{db: db}
}

// RegisterPrinter registers or updates a discovered printer.
func (r *AgentPrinterRepository) RegisterPrinter(ctx context.Context, printer *DiscoveredPrinter) error {
	now := time.Now()

	// Serialize capabilities
	var capabilitiesJSON string
	if printer.Capabilities != "" {
		capabilitiesJSON = printer.Capabilities
	}

	query := `
		INSERT INTO discovered_printers (
			id, agent_id, name, display_name, driver, driver_version, port,
			connection_type, status, is_default, is_shared, share_name, location,
			capabilities, last_seen, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		)
		ON CONFLICT (agent_id, name)
		DO UPDATE SET
			display_name = EXCLUDED.display_name,
			driver = EXCLUDED.driver,
			driver_version = EXCLUDED.driver_version,
			port = EXCLUDED.port,
			connection_type = EXCLUDED.connection_type,
			status = EXCLUDED.status,
			is_default = EXCLUDED.is_default,
			is_shared = EXCLUDED.is_shared,
			share_name = EXCLUDED.share_name,
			location = EXCLUDED.location,
			capabilities = EXCLUDED.capabilities,
			last_seen = EXCLUDED.last_seen,
			updated_at = EXCLUDED.updated_at
		RETURNING id
	`

	if printer.ID == "" {
		printer.ID = uuid.New().String()
	}
	printer.LastSeen = now
	printer.UpdatedAt = now
	if printer.CreatedAt.IsZero() {
		printer.CreatedAt = now
	}

	err := r.db.QueryRow(ctx, query,
		printer.ID,
		printer.AgentID,
		printer.Name,
		printer.DisplayName,
		printer.Driver,
		printer.DriverVersion,
		printer.Port,
		printer.ConnectionType,
		printer.Status,
		printer.IsDefault,
		printer.IsShared,
		printer.ShareName,
		printer.Location,
		capabilitiesJSON,
		printer.LastSeen,
		printer.CreatedAt,
		printer.UpdatedAt,
	).Scan(&printer.ID)

	if err != nil {
		return fmt.Errorf("register printer: %w", err)
	}

	return nil
}

// RegisterPrinters registers multiple printers in a single transaction.
func (r *AgentPrinterRepository) RegisterPrinters(ctx context.Context, printers []*DiscoveredPrinter) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now()

	for _, printer := range printers {
		if printer.ID == "" {
			printer.ID = uuid.New().String()
		}
		printer.LastSeen = now
		printer.UpdatedAt = now
		if printer.CreatedAt.IsZero() {
			printer.CreatedAt = now
		}

		query := `
			INSERT INTO discovered_printers (
				id, agent_id, name, display_name, driver, driver_version, port,
				connection_type, status, is_default, is_shared, share_name, location,
				capabilities, last_seen, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
			)
			ON CONFLICT (agent_id, name)
			DO UPDATE SET
				display_name = EXCLUDED.display_name,
				driver = EXCLUDED.driver,
				driver_version = EXCLUDED.driver_version,
				port = EXCLUDED.port,
				connection_type = EXCLUDED.connection_type,
				status = EXCLUDED.status,
				is_default = EXCLUDED.is_default,
				is_shared = EXCLUDED.is_shared,
				share_name = EXCLUDED.share_name,
				location = EXCLUDED.location,
				capabilities = EXCLUDED.capabilities,
				last_seen = EXCLUDED.last_seen,
				updated_at = EXCLUDED.updated_at
		`

		_, err := tx.Exec(ctx, query,
			printer.ID,
			printer.AgentID,
			printer.Name,
			printer.DisplayName,
			printer.Driver,
			printer.DriverVersion,
			printer.Port,
			printer.ConnectionType,
			printer.Status,
			printer.IsDefault,
			printer.IsShared,
			printer.ShareName,
			printer.Location,
			printer.Capabilities,
			printer.LastSeen,
			printer.CreatedAt,
			printer.UpdatedAt,
		)

		if err != nil {
			return fmt.Errorf("register printer %s: %w", printer.Name, err)
		}
	}

	return tx.Commit(ctx)
}

// FindByID retrieves a printer by ID.
func (r *AgentPrinterRepository) FindByID(ctx context.Context, id string) (*DiscoveredPrinter, error) {
	query := `
		SELECT id, agent_id, name, display_name, driver, driver_version, port,
		       connection_type, status, is_default, is_shared, share_name, location,
		       capabilities, last_seen, created_at, updated_at
		FROM discovered_printers
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
func (r *AgentPrinterRepository) FindByAgent(ctx context.Context, agentID string) ([]*DiscoveredPrinter, error) {
	query := `
		SELECT id, agent_id, name, display_name, driver, driver_version, port,
		       connection_type, status, is_default, is_shared, share_name, location,
		       capabilities, last_seen, created_at, updated_at
		FROM discovered_printers
		WHERE agent_id = $1
		ORDER BY is_default DESC, name ASC
	`

	rows, err := r.db.Query(ctx, query, agentID)
	if err != nil {
		return nil, fmt.Errorf("find printers by agent: %w", err)
	}
	defer rows.Close()

	var printers []*DiscoveredPrinter
	for rows.Next() {
		printer, err := r.scanPrinter(rows)
		if err != nil {
			return nil, err
		}
		printers = append(printers, printer)
	}

	return printers, rows.Err()
}

// FindByAgentAndName retrieves a specific printer by agent and name.
func (r *AgentPrinterRepository) FindByAgentAndName(ctx context.Context, agentID, name string) (*DiscoveredPrinter, error) {
	query := `
		SELECT id, agent_id, name, display_name, driver, driver_version, port,
		       connection_type, status, is_default, is_shared, share_name, location,
		       capabilities, last_seen, created_at, updated_at
		FROM discovered_printers
		WHERE agent_id = $1 AND name = $2
	`

	printer, err := r.scanPrinter(r.db.QueryRow(ctx, query, agentID, name))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("printer not found")
		}
		return nil, fmt.Errorf("find printer by agent and name: %w", err)
	}

	return printer, nil
}

// UpdateStatus updates a printer's status.
func (r *AgentPrinterRepository) UpdateStatus(ctx context.Context, printerID string, status agent.PrinterStatus) error {
	query := `
		UPDATE discovered_printers
		SET status = $2, updated_at = $3
		WHERE id = $1
	`

	cmdTag, err := r.db.Exec(ctx, query, printerID, status, time.Now())
	if err != nil {
		return fmt.Errorf("update printer status: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("printer not found")
	}

	return nil
}

// UpdateCapabilities updates a printer's capabilities.
func (r *AgentPrinterRepository) UpdateCapabilities(ctx context.Context, printerID string, capabilities *agent.PrinterCapabilities) error {
	capabilitiesJSON, err := json.Marshal(capabilities)
	if err != nil {
		return fmt.Errorf("marshal capabilities: %w", err)
	}

	query := `
		UPDATE discovered_printers
		SET capabilities = $2, updated_at = $3
		WHERE id = $1
	`

	cmdTag, err := r.db.Exec(ctx, query, printerID, string(capabilitiesJSON), time.Now())
	if err != nil {
		return fmt.Errorf("update printer capabilities: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("printer not found")
	}

	return nil
}

// Delete removes a printer.
func (r *AgentPrinterRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM discovered_printers WHERE id = $1`

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
func (r *AgentPrinterRepository) DeleteByAgent(ctx context.Context, agentID string) (int64, error) {
	query := `DELETE FROM discovered_printers WHERE agent_id = $1`

	cmdTag, err := r.db.Exec(ctx, query, agentID)
	if err != nil {
		return 0, fmt.Errorf("delete printers by agent: %w", err)
	}

	return cmdTag.RowsAffected(), nil
}

// DeleteExcept removes printers that were not in the provided list (for sync).
func (r *AgentPrinterRepository) DeleteExcept(ctx context.Context, agentID string, printerNames []string) error {
	if len(printerNames) == 0 {
		_, err := r.DeleteByAgent(ctx, agentID)
		return err
	}

	query := `
		DELETE FROM discovered_printers
		WHERE agent_id = $1 AND name NOT = ANY($2)
	`

	_, err := r.db.Exec(ctx, query, agentID, printerNames)
	if err != nil {
		return fmt.Errorf("delete printers except: %w", err)
	}

	return nil
}

// MarkStale marks printers that haven't been seen recently as offline.
func (r *AgentPrinterRepository) MarkStale(ctx context.Context, since time.Time) (int64, error) {
	query := `
		UPDATE discovered_printers
		SET status = 'offline', updated_at = $2
		WHERE last_seen < $1 AND status != 'offline'
	`

	cmdTag, err := r.db.Exec(ctx, query, since, time.Now())
	if err != nil {
		return 0, fmt.Errorf("mark stale printers: %w", err)
	}

	return cmdTag.RowsAffected(), nil
}

// List retrieves all printers with pagination.
func (r *AgentPrinterRepository) List(ctx context.Context, limit, offset int) ([]*DiscoveredPrinter, int, error) {
	// Get total count
	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM discovered_printers").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count printers: %w", err)
	}

	query := `
		SELECT id, agent_id, name, display_name, driver, driver_version, port,
		       connection_type, status, is_default, is_shared, share_name, location,
		       capabilities, last_seen, created_at, updated_at
		FROM discovered_printers
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list printers: %w", err)
	}
	defer rows.Close()

	var printers []*DiscoveredPrinter
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
func (r *AgentPrinterRepository) FindByStatus(ctx context.Context, status agent.PrinterStatus) ([]*DiscoveredPrinter, error) {
	query := `
		SELECT id, agent_id, name, display_name, driver, driver_version, port,
		       connection_type, status, is_default, is_shared, share_name, location,
		       capabilities, last_seen, created_at, updated_at
		FROM discovered_printers
		WHERE status = $1
		ORDER BY last_seen DESC
	`

	rows, err := r.db.Query(ctx, query, status)
	if err != nil {
		return nil, fmt.Errorf("find printers by status: %w", err)
	}
	defer rows.Close()

	var printers []*DiscoveredPrinter
	for rows.Next() {
		printer, err := r.scanPrinter(rows)
		if err != nil {
			return nil, err
		}
		printers = append(printers, printer)
	}

	return printers, rows.Err()
}

// GetDefaultPrinter returns the default printer for an agent.
func (r *AgentPrinterRepository) GetDefaultPrinter(ctx context.Context, agentID string) (*DiscoveredPrinter, error) {
	query := `
		SELECT id, agent_id, name, display_name, driver, driver_version, port,
		       connection_type, status, is_default, is_shared, share_name, location,
		       capabilities, last_seen, created_at, updated_at
		FROM discovered_printers
		WHERE agent_id = $1 AND is_default = true
		LIMIT 1
	`

	printer, err := r.scanPrinter(r.db.QueryRow(ctx, query, agentID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("default printer not found")
		}
		return nil, fmt.Errorf("get default printer: %w", err)
	}

	return printer, nil
}

// scanPrinter scans a printer from a database row.
func (r *AgentPrinterRepository) scanPrinter(row interface{ Scan(...interface{}) error }) (*DiscoveredPrinter, error) {
	var printer DiscoveredPrinter
	err := row.Scan(
		&printer.ID,
		&printer.AgentID,
		&printer.Name,
		&printer.DisplayName,
		&printer.Driver,
		&printer.DriverVersion,
		&printer.Port,
		&printer.ConnectionType,
		&printer.Status,
		&printer.IsDefault,
		&printer.IsShared,
		&printer.ShareName,
		&printer.Location,
		&printer.Capabilities,
		&printer.LastSeen,
		&printer.CreatedAt,
		&printer.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &printer, nil
}

// ToDiscoveredPrinter converts a DiscoveredPrinter to agent.DiscoveredPrinter.
func (p *DiscoveredPrinter) ToDiscoveredPrinter() *agent.DiscoveredPrinter {
	var capabilities *agent.PrinterCapabilities
	if p.Capabilities != "" {
		json.Unmarshal([]byte(p.Capabilities), &capabilities)
	}

	return &agent.DiscoveredPrinter{
		PrinterID:      p.ID,
		AgentID:        p.AgentID,
		Name:           p.Name,
		DisplayName:    p.DisplayName,
		Driver:         p.Driver,
		DriverVersion:  p.DriverVersion,
		Port:           p.Port,
		ConnectionType: p.ConnectionType,
		Status:         p.Status,
		IsDefault:      p.IsDefault,
		IsShared:       p.IsShared,
		ShareName:      p.ShareName,
		Location:       p.Location,
		Capabilities:   capabilities,
		LastSeen:       p.LastSeen,
		CreatedAt:      p.CreatedAt,
	}
}

// FromDiscoveredPrinter converts an agent.DiscoveredPrinter to repository.DiscoveredPrinter.
func FromDiscoveredPrinter(p *agent.DiscoveredPrinter, agentID string) *DiscoveredPrinter {
	var capabilitiesJSON string
	if p.Capabilities != nil {
		data, _ := json.Marshal(p.Capabilities)
		capabilitiesJSON = string(data)
	}

	return &DiscoveredPrinter{
		ID:             p.PrinterID,
		AgentID:        agentID,
		Name:           p.Name,
		DisplayName:    p.DisplayName,
		Driver:         p.Driver,
		DriverVersion:  p.DriverVersion,
		Port:           p.Port,
		ConnectionType: p.ConnectionType,
		Status:         p.Status,
		IsDefault:      p.IsDefault,
		IsShared:       p.IsShared,
		ShareName:      p.ShareName,
		Location:       p.Location,
		Capabilities:   capabilitiesJSON,
		LastSeen:       p.LastSeen,
		CreatedAt:      p.CreatedAt,
	}
}
