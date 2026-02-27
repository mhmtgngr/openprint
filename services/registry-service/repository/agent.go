// Package repository provides data access layer for the registry service.
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Agent represents a print server agent.
type Agent struct {
	ID             string
	Name           string
	Version        string
	OS             string
	Architecture   string
	Hostname       string
	OrganizationID string
	Status         string // "online", "offline"
	LastHeartbeat  time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// AgentRepository handles agent data operations.
type AgentRepository struct {
	db *pgxpool.Pool
}

// NewAgentRepository creates a new agent repository.
func NewAgentRepository(db *pgxpool.Pool) *AgentRepository {
	return &AgentRepository{db: db}
}

// Create inserts a new agent.
func (r *AgentRepository) Create(ctx context.Context, agent *Agent) error {
	now := time.Now()
	agent.CreatedAt = now
	agent.UpdatedAt = now
	agent.LastHeartbeat = now

	query := `
		INSERT INTO agents (id, name, version, os, architecture, hostname, organization_id, status, last_heartbeat, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, '')::uuid, $8, $9, $10, $11)
		RETURNING id
	`

	err := r.db.QueryRow(ctx, query,
		agent.ID,
		agent.Name,
		agent.Version,
		agent.OS,
		agent.Architecture,
		agent.Hostname,
		agent.OrganizationID,
		agent.Status,
		agent.LastHeartbeat,
		agent.CreatedAt,
		agent.UpdatedAt,
	).Scan(&agent.ID)

	if err != nil {
		return fmt.Errorf("create agent: %w", err)
	}

	return nil
}

// FindByID retrieves an agent by ID.
func (r *AgentRepository) FindByID(ctx context.Context, id string) (*Agent, error) {
	query := `
		SELECT id, name, version, os, architecture, hostname, organization_id, status, last_heartbeat, created_at, updated_at
		FROM agents
		WHERE id = $1
	`

	agent, err := r.scanAgent(r.db.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("agent not found")
		}
		return nil, fmt.Errorf("find agent by id: %w", err)
	}

	return agent, nil
}

// FindByHostname retrieves an agent by hostname.
func (r *AgentRepository) FindByHostname(ctx context.Context, hostname string) (*Agent, error) {
	query := `
		SELECT id, name, version, os, architecture, hostname, organization_id, status, last_heartbeat, created_at, updated_at
		FROM agents
		WHERE hostname = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	agent, err := r.scanAgent(r.db.QueryRow(ctx, query, hostname))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("agent not found")
		}
		return nil, fmt.Errorf("find agent by hostname: %w", err)
	}

	return agent, nil
}

// FindByStatus retrieves all agents with a given status.
func (r *AgentRepository) FindByStatus(ctx context.Context, status string) ([]*Agent, error) {
	query := `
		SELECT id, name, version, os, architecture, hostname, organization_id, status, last_heartbeat, created_at, updated_at
		FROM agents
		WHERE status = $1
		ORDER BY last_heartbeat DESC
	`

	rows, err := r.db.Query(ctx, query, status)
	if err != nil {
		return nil, fmt.Errorf("find agents by status: %w", err)
	}
	defer rows.Close()

	var agents []*Agent
	for rows.Next() {
		agent, err := r.scanAgent(rows)
		if err != nil {
			return nil, err
		}
		agents = append(agents, agent)
	}

	return agents, rows.Err()
}

// FindByOrganization retrieves all agents for an organization.
func (r *AgentRepository) FindByOrganization(ctx context.Context, orgID string) ([]*Agent, error) {
	query := `
		SELECT id, name, version, os, architecture, hostname, organization_id, status, last_heartbeat, created_at, updated_at
		FROM agents
		WHERE organization_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("find agents by organization: %w", err)
	}
	defer rows.Close()

	var agents []*Agent
	for rows.Next() {
		agent, err := r.scanAgent(rows)
		if err != nil {
			return nil, err
		}
		agents = append(agents, agent)
	}

	return agents, rows.Err()
}

// List retrieves all agents with pagination.
func (r *AgentRepository) List(ctx context.Context, limit, offset int) ([]*Agent, int, error) {
	// Get total count
	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM agents").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count agents: %w", err)
	}

	// Get agents
	query := `
		SELECT id, name, version, os, architecture, hostname, organization_id, status, last_heartbeat, created_at, updated_at
		FROM agents
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()

	var agents []*Agent
	for rows.Next() {
		agent, err := r.scanAgent(rows)
		if err != nil {
			return nil, 0, err
		}
		agents = append(agents, agent)
	}

	return agents, total, rows.Err()
}

// Update updates an agent.
func (r *AgentRepository) Update(ctx context.Context, agent *Agent) error {
	agent.UpdatedAt = time.Now()

	query := `
		UPDATE agents
		SET name = $2, version = $3, os = $4, architecture = $5, hostname = $6,
		    organization_id = $7, status = $8, updated_at = $9
		WHERE id = $1
	`

	cmdTag, err := r.db.Exec(ctx, query,
		agent.ID,
		agent.Name,
		agent.Version,
		agent.OS,
		agent.Architecture,
		agent.Hostname,
		agent.OrganizationID,
		agent.Status,
		agent.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("update agent: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("agent not found")
	}

	return nil
}

// UpdateHeartbeat updates the last heartbeat time and status to online.
func (r *AgentRepository) UpdateHeartbeat(ctx context.Context, id string, t time.Time) error {
	query := `
		UPDATE agents
		SET last_heartbeat = $2, status = 'online', updated_at = $3
		WHERE id = $1
	`

	cmdTag, err := r.db.Exec(ctx, query, id, t, t)
	if err != nil {
		return fmt.Errorf("update heartbeat: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("agent not found")
	}

	return nil
}

// SetStatus updates an agent's status.
func (r *AgentRepository) SetStatus(ctx context.Context, id, status string) error {
	query := `
		UPDATE agents
		SET status = $2, updated_at = $3
		WHERE id = $1
	`

	cmdTag, err := r.db.Exec(ctx, query, id, status, time.Now())
	if err != nil {
		return fmt.Errorf("set status: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("agent not found")
	}

	return nil
}

// Delete removes an agent.
func (r *AgentRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM agents WHERE id = $1`

	cmdTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete agent: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("agent not found")
	}

	return nil
}

// MarkOfflineBefore marks agents as offline if their last heartbeat was before the given time.
func (r *AgentRepository) MarkOfflineBefore(ctx context.Context, t time.Time) (int64, error) {
	query := `
		UPDATE agents
		SET status = 'offline', updated_at = $2
		WHERE status = 'online' AND last_heartbeat < $1
	`

	cmdTag, err := r.db.Exec(ctx, query, t, time.Now())
	if err != nil {
		return 0, fmt.Errorf("mark offline: %w", err)
	}

	return cmdTag.RowsAffected(), nil
}

// CountByStatus returns the count of agents by status.
func (r *AgentRepository) CountByStatus(ctx context.Context, status string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM agents WHERE status = $1", status).Scan(&count)
	return count, err
}

// GetStaleAgents returns agents that haven't sent a heartbeat since the given time.
func (r *AgentRepository) GetStaleAgents(ctx context.Context, since time.Time) ([]*Agent, error) {
	query := `
		SELECT id, name, version, os, architecture, hostname, organization_id, status, last_heartbeat, created_at, updated_at
		FROM agents
		WHERE last_heartbeat < $1 AND status = 'online'
		ORDER BY last_heartbeat ASC
	`

	rows, err := r.db.Query(ctx, query, since)
	if err != nil {
		return nil, fmt.Errorf("get stale agents: %w", err)
	}
	defer rows.Close()

	var agents []*Agent
	for rows.Next() {
		agent, err := r.scanAgent(rows)
		if err != nil {
			return nil, err
		}
		agents = append(agents, agent)
	}

	return agents, rows.Err()
}

// RegisterOrFindByHostname creates a new agent or returns an existing one by hostname.
func (r *AgentRepository) RegisterOrFindByHostname(ctx context.Context, name, version, os, arch, hostname string) (*Agent, error) {
	// Try to find by hostname first
	agent, err := r.FindByHostname(ctx, hostname)
	if err == nil {
		// Update existing agent
		agent.Name = name
		agent.Version = version
		agent.OS = os
		agent.Architecture = arch
		agent.Status = "online"
		agent.LastHeartbeat = time.Now()
		r.Update(ctx, agent)
		return agent, nil
	}

	// Create new agent
	newAgent := &Agent{
		ID:           uuid.New().String(),
		Name:         name,
		Version:      version,
		OS:           os,
		Architecture: arch,
		Hostname:     hostname,
		Status:       "online",
	}

	if err := r.Create(ctx, newAgent); err != nil {
		return nil, err
	}

	return newAgent, nil
}

// scanAgent scans an agent from a database row.
func (r *AgentRepository) scanAgent(row interface{ Scan(...interface{}) error }) (*Agent, error) {
	var agent Agent
	err := row.Scan(
		&agent.ID,
		&agent.Name,
		&agent.Version,
		&agent.OS,
		&agent.Architecture,
		&agent.Hostname,
		&agent.OrganizationID,
		&agent.Status,
		&agent.LastHeartbeat,
		&agent.CreatedAt,
		&agent.UpdatedAt,
	)
	return &agent, err
}
