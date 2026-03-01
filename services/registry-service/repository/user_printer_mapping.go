// Package repository provides data access layer for user-printer mappings.
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserPrinterMapping maps an RDP session user to their local client-side agent and printer.
type UserPrinterMapping struct {
	ID                string
	OrganizationID    string
	UserEmail         string
	UserName          string
	ClientAgentID     string
	TargetPrinterID   string
	TargetPrinterName string
	ServerAgentID     string
	IsActive          bool
	IsDefault         bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// UserPrinterMappingRepository handles user-printer mapping data operations.
type UserPrinterMappingRepository struct {
	db *pgxpool.Pool
}

// NewUserPrinterMappingRepository creates a new repository.
func NewUserPrinterMappingRepository(db *pgxpool.Pool) *UserPrinterMappingRepository {
	return &UserPrinterMappingRepository{db: db}
}

// Create inserts a new user-printer mapping.
func (r *UserPrinterMappingRepository) Create(ctx context.Context, m *UserPrinterMapping) error {
	now := time.Now()
	m.CreatedAt = now
	m.UpdatedAt = now

	if m.ID == "" {
		m.ID = uuid.New().String()
	}

	query := `
		INSERT INTO user_printer_mappings (
			id, organization_id, user_email, user_name, client_agent_id,
			target_printer_id, target_printer_name, server_agent_id,
			is_active, is_default, created_at, updated_at
		)
		VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, NULLIF($6, '')::uuid, $7, NULLIF($8, '')::uuid, $9, $10, $11, $12)
		RETURNING id
	`

	err := r.db.QueryRow(ctx, query,
		m.ID, m.OrganizationID, m.UserEmail, m.UserName, m.ClientAgentID,
		m.TargetPrinterID, m.TargetPrinterName, m.ServerAgentID,
		m.IsActive, m.IsDefault, m.CreatedAt, m.UpdatedAt,
	).Scan(&m.ID)

	if err != nil {
		return fmt.Errorf("create user-printer mapping: %w", err)
	}

	return nil
}

// FindByID retrieves a mapping by ID.
func (r *UserPrinterMappingRepository) FindByID(ctx context.Context, id string) (*UserPrinterMapping, error) {
	query := `
		SELECT id, organization_id, user_email, user_name, client_agent_id,
		       target_printer_id, target_printer_name, server_agent_id,
		       is_active, is_default, created_at, updated_at
		FROM user_printer_mappings
		WHERE id = $1
	`

	m, err := r.scanMapping(r.db.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("mapping not found")
		}
		return nil, fmt.Errorf("find mapping by id: %w", err)
	}
	return m, nil
}

// FindByUserEmail retrieves active mappings for a user email.
func (r *UserPrinterMappingRepository) FindByUserEmail(ctx context.Context, userEmail string) ([]*UserPrinterMapping, error) {
	query := `
		SELECT id, organization_id, user_email, user_name, client_agent_id,
		       target_printer_id, target_printer_name, server_agent_id,
		       is_active, is_default, created_at, updated_at
		FROM user_printer_mappings
		WHERE user_email = $1 AND is_active = true
		ORDER BY is_default DESC, created_at ASC
	`

	rows, err := r.db.Query(ctx, query, userEmail)
	if err != nil {
		return nil, fmt.Errorf("find mappings by user email: %w", err)
	}
	defer rows.Close()

	var mappings []*UserPrinterMapping
	for rows.Next() {
		m, err := r.scanMapping(rows)
		if err != nil {
			return nil, err
		}
		mappings = append(mappings, m)
	}

	return mappings, rows.Err()
}

// FindDefaultByUserEmail retrieves the default mapping for a user.
func (r *UserPrinterMappingRepository) FindDefaultByUserEmail(ctx context.Context, userEmail string) (*UserPrinterMapping, error) {
	query := `
		SELECT id, organization_id, user_email, user_name, client_agent_id,
		       target_printer_id, target_printer_name, server_agent_id,
		       is_active, is_default, created_at, updated_at
		FROM user_printer_mappings
		WHERE user_email = $1 AND is_active = true AND is_default = true
		LIMIT 1
	`

	m, err := r.scanMapping(r.db.QueryRow(ctx, query, userEmail))
	if err != nil {
		if err == pgx.ErrNoRows {
			// Fall back to any active mapping
			return r.findFirstActiveByUserEmail(ctx, userEmail)
		}
		return nil, fmt.Errorf("find default mapping: %w", err)
	}
	return m, nil
}

func (r *UserPrinterMappingRepository) findFirstActiveByUserEmail(ctx context.Context, userEmail string) (*UserPrinterMapping, error) {
	query := `
		SELECT id, organization_id, user_email, user_name, client_agent_id,
		       target_printer_id, target_printer_name, server_agent_id,
		       is_active, is_default, created_at, updated_at
		FROM user_printer_mappings
		WHERE user_email = $1 AND is_active = true
		ORDER BY created_at ASC
		LIMIT 1
	`

	m, err := r.scanMapping(r.db.QueryRow(ctx, query, userEmail))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("no active mapping found for user")
		}
		return nil, fmt.Errorf("find first active mapping: %w", err)
	}
	return m, nil
}

// FindByClientAgent retrieves all mappings targeting a specific client agent.
func (r *UserPrinterMappingRepository) FindByClientAgent(ctx context.Context, clientAgentID string) ([]*UserPrinterMapping, error) {
	query := `
		SELECT id, organization_id, user_email, user_name, client_agent_id,
		       target_printer_id, target_printer_name, server_agent_id,
		       is_active, is_default, created_at, updated_at
		FROM user_printer_mappings
		WHERE client_agent_id = $1 AND is_active = true
		ORDER BY user_email, created_at ASC
	`

	rows, err := r.db.Query(ctx, query, clientAgentID)
	if err != nil {
		return nil, fmt.Errorf("find mappings by client agent: %w", err)
	}
	defer rows.Close()

	var mappings []*UserPrinterMapping
	for rows.Next() {
		m, err := r.scanMapping(rows)
		if err != nil {
			return nil, err
		}
		mappings = append(mappings, m)
	}

	return mappings, rows.Err()
}

// FindByOrganization retrieves all mappings for an organization.
func (r *UserPrinterMappingRepository) FindByOrganization(ctx context.Context, orgID string, limit, offset int) ([]*UserPrinterMapping, int, error) {
	countQuery := `SELECT COUNT(*) FROM user_printer_mappings WHERE organization_id = $1`
	var total int
	if err := r.db.QueryRow(ctx, countQuery, orgID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count mappings: %w", err)
	}

	query := `
		SELECT id, organization_id, user_email, user_name, client_agent_id,
		       target_printer_id, target_printer_name, server_agent_id,
		       is_active, is_default, created_at, updated_at
		FROM user_printer_mappings
		WHERE organization_id = $1
		ORDER BY user_email, created_at ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, orgID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("find mappings by org: %w", err)
	}
	defer rows.Close()

	var mappings []*UserPrinterMapping
	for rows.Next() {
		m, err := r.scanMapping(rows)
		if err != nil {
			return nil, 0, err
		}
		mappings = append(mappings, m)
	}

	return mappings, total, rows.Err()
}

// Update updates a user-printer mapping.
func (r *UserPrinterMappingRepository) Update(ctx context.Context, m *UserPrinterMapping) error {
	m.UpdatedAt = time.Now()

	query := `
		UPDATE user_printer_mappings
		SET user_name = $2, client_agent_id = $3, target_printer_id = NULLIF($4, '')::uuid,
		    target_printer_name = $5, server_agent_id = NULLIF($6, '')::uuid,
		    is_active = $7, is_default = $8, updated_at = $9
		WHERE id = $1
	`

	cmdTag, err := r.db.Exec(ctx, query,
		m.ID, m.UserName, m.ClientAgentID, m.TargetPrinterID,
		m.TargetPrinterName, m.ServerAgentID, m.IsActive, m.IsDefault, m.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update mapping: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("mapping not found")
	}

	return nil
}

// Delete removes a mapping.
func (r *UserPrinterMappingRepository) Delete(ctx context.Context, id string) error {
	cmdTag, err := r.db.Exec(ctx, `DELETE FROM user_printer_mappings WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete mapping: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("mapping not found")
	}

	return nil
}

// ResolveUsername looks up a user email from a Windows username across mappings.
func (r *UserPrinterMappingRepository) ResolveUsername(ctx context.Context, username string) (string, error) {
	query := `
		SELECT user_email
		FROM user_printer_mappings
		WHERE user_name = $1 AND is_active = true
		LIMIT 1
	`

	var email string
	err := r.db.QueryRow(ctx, query, username).Scan(&email)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("no mapping found for username")
		}
		return "", fmt.Errorf("resolve username: %w", err)
	}

	return email, nil
}

func (r *UserPrinterMappingRepository) scanMapping(row interface{ Scan(...interface{}) error }) (*UserPrinterMapping, error) {
	var m UserPrinterMapping
	var orgID, targetPrinterID, serverAgentID *string

	err := row.Scan(
		&m.ID, &orgID, &m.UserEmail, &m.UserName, &m.ClientAgentID,
		&targetPrinterID, &m.TargetPrinterName, &serverAgentID,
		&m.IsActive, &m.IsDefault, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if orgID != nil {
		m.OrganizationID = *orgID
	}
	if targetPrinterID != nil {
		m.TargetPrinterID = *targetPrinterID
	}
	if serverAgentID != nil {
		m.ServerAgentID = *serverAgentID
	}

	return &m, nil
}
