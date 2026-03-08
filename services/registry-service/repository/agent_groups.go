// Package repository provides data access layer for agent groups.
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AgentGroup represents an agent group in the database.
type AgentGroup struct {
	ID             string
	Name           string
	Description    string
	OrganizationID string
	OwnerUserID    string
	Type           string
	Location       string
	Tags           []string
	PolicyID       string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// AgentGroupRepository handles agent group data operations.
type AgentGroupRepository struct {
	db *pgxpool.Pool
}

// NewAgentGroupRepository creates a new agent group repository.
func NewAgentGroupRepository(db *pgxpool.Pool) *AgentGroupRepository {
	return &AgentGroupRepository{db: db}
}

// Create inserts a new agent group.
func (r *AgentGroupRepository) Create(ctx context.Context, group *AgentGroup) error {
	now := time.Now()
	group.ID = uuid.New().String()
	group.CreatedAt = now
	group.UpdatedAt = now

	query := `
		INSERT INTO agent_groups (id, name, description, organization_id, owner_user_id, type, location, tags, policy_id, created_at, updated_at)
		VALUES ($1, $2, $3, NULLIF($4, '')::uuid, NULLIF($5, '')::uuid, $6, $7, $8, NULLIF($9, '')::uuid, $10, $11)
		RETURNING id
	`

	err := r.db.QueryRow(ctx, query,
		group.ID,
		group.Name,
		group.Description,
		group.OrganizationID,
		group.OwnerUserID,
		group.Type,
		group.Location,
		group.Tags,
		group.PolicyID,
		group.CreatedAt,
		group.UpdatedAt,
	).Scan(&group.ID)

	if err != nil {
		return fmt.Errorf("create agent group: %w", err)
	}

	return nil
}

// FindByID retrieves an agent group by ID.
func (r *AgentGroupRepository) FindByID(ctx context.Context, id string) (*AgentGroup, error) {
	query := `
		SELECT id, name, description, organization_id, owner_user_id, type, location, tags, policy_id, created_at, updated_at
		FROM agent_groups
		WHERE id = $1
	`

	group, err := r.scanAgentGroup(r.db.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("agent group not found")
		}
		return nil, fmt.Errorf("find agent group by id: %w", err)
	}

	return group, nil
}

// FindByOrganization retrieves all groups for an organization.
func (r *AgentGroupRepository) FindByOrganization(ctx context.Context, orgID string) ([]*AgentGroup, error) {
	query := `
		SELECT id, name, description, organization_id, owner_user_id, type, location, tags, policy_id, created_at, updated_at
		FROM agent_groups
		WHERE organization_id = $1
		ORDER BY name ASC
	`

	rows, err := r.db.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("find agent groups by organization: %w", err)
	}
	defer rows.Close()

	var groups []*AgentGroup
	for rows.Next() {
		group, err := r.scanAgentGroup(rows)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}

	return groups, rows.Err()
}

// FindByOwner retrieves all groups owned by a specific user.
func (r *AgentGroupRepository) FindByOwner(ctx context.Context, ownerUserID string) ([]*AgentGroup, error) {
	query := `
		SELECT id, name, description, organization_id, owner_user_id, type, location, tags, policy_id, created_at, updated_at
		FROM agent_groups
		WHERE owner_user_id = $1
		ORDER BY name ASC
	`

	rows, err := r.db.Query(ctx, query, ownerUserID)
	if err != nil {
		return nil, fmt.Errorf("find agent groups by owner: %w", err)
	}
	defer rows.Close()

	var groups []*AgentGroup
	for rows.Next() {
		group, err := r.scanAgentGroup(rows)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}

	return groups, rows.Err()
}

// List retrieves all groups with pagination.
func (r *AgentGroupRepository) List(ctx context.Context, limit, offset int) ([]*AgentGroup, int, error) {
	// Get total count
	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM agent_groups").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count agent groups: %w", err)
	}

	// Get groups
	query := `
		SELECT id, name, description, organization_id, owner_user_id, type, location, tags, policy_id, created_at, updated_at
		FROM agent_groups
		ORDER BY name ASC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list agent groups: %w", err)
	}
	defer rows.Close()

	var groups []*AgentGroup
	for rows.Next() {
		group, err := r.scanAgentGroup(rows)
		if err != nil {
			return nil, 0, err
		}
		groups = append(groups, group)
	}

	return groups, total, rows.Err()
}

// Update updates an agent group.
func (r *AgentGroupRepository) Update(ctx context.Context, group *AgentGroup) error {
	group.UpdatedAt = time.Now()

	query := `
		UPDATE agent_groups
		SET name = $2, description = $3, organization_id = NULLIF($4, '')::uuid,
		    owner_user_id = NULLIF($5, '')::uuid, type = $6, location = $7, tags = $8,
		    policy_id = NULLIF($9, '')::uuid, updated_at = $10
		WHERE id = $1
	`

	cmdTag, err := r.db.Exec(ctx, query,
		group.ID,
		group.Name,
		group.Description,
		group.OrganizationID,
		group.OwnerUserID,
		group.Type,
		group.Location,
		group.Tags,
		group.PolicyID,
		group.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("update agent group: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("agent group not found")
	}

	return nil
}

// Delete removes an agent group.
func (r *AgentGroupRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM agent_groups WHERE id = $1`

	cmdTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete agent group: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("agent group not found")
	}

	return nil
}

// AddAgentToGroup adds an agent to a group.
func (r *AgentGroupRepository) AddAgentToGroup(ctx context.Context, groupID, agentID, addedBy string) error {
	now := time.Now()
	membershipID := uuid.New().String()

	query := `
		INSERT INTO agent_group_memberships (id, group_id, agent_id, added_at, added_by)
		VALUES ($1, $2, $3, $4, NULLIF($5, '')::uuid)
		ON CONFLICT (group_id, agent_id) DO NOTHING
	`

	_, err := r.db.Exec(ctx, query, membershipID, groupID, agentID, now, addedBy)
	if err != nil {
		return fmt.Errorf("add agent to group: %w", err)
	}

	return nil
}

// RemoveAgentFromGroup removes an agent from a group.
func (r *AgentGroupRepository) RemoveAgentFromGroup(ctx context.Context, groupID, agentID string) error {
	query := `DELETE FROM agent_group_memberships WHERE group_id = $1 AND agent_id = $2`

	cmdTag, err := r.db.Exec(ctx, query, groupID, agentID)
	if err != nil {
		return fmt.Errorf("remove agent from group: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("agent not in group")
	}

	return nil
}

// GetAgentsInGroup retrieves all agents in a group.
func (r *AgentGroupRepository) GetAgentsInGroup(ctx context.Context, groupID string) ([]string, error) {
	query := `
		SELECT agent_id
		FROM agent_group_memberships
		WHERE group_id = $1
		ORDER BY added_at DESC
	`

	rows, err := r.db.Query(ctx, query, groupID)
	if err != nil {
		return nil, fmt.Errorf("get agents in group: %w", err)
	}
	defer rows.Close()

	var agentIDs []string
	for rows.Next() {
		var agentID string
		if err := rows.Scan(&agentID); err != nil {
			return nil, err
		}
		agentIDs = append(agentIDs, agentID)
	}

	return agentIDs, rows.Err()
}

// RemoveAllAgentsFromGroup removes all agents from a group.
func (r *AgentGroupRepository) RemoveAllAgentsFromGroup(ctx context.Context, groupID string) error {
	query := `DELETE FROM agent_group_memberships WHERE group_id = $1`

	_, err := r.db.Exec(ctx, query, groupID)
	if err != nil {
		return fmt.Errorf("remove all agents from group: %w", err)
	}

	return nil
}

// GetGroupsForAgent retrieves all groups that an agent belongs to.
func (r *AgentGroupRepository) GetGroupsForAgent(ctx context.Context, agentID string) ([]*AgentGroup, error) {
	query := `
		SELECT g.id, g.name, g.description, g.organization_id, g.owner_user_id, g.type, g.location, g.tags, g.policy_id, g.created_at, g.updated_at
		FROM agent_groups g
		INNER JOIN agent_group_memberships m ON g.id = m.group_id
		WHERE m.agent_id = $1
		ORDER BY g.name ASC
	`

	rows, err := r.db.Query(ctx, query, agentID)
	if err != nil {
		return nil, fmt.Errorf("get groups for agent: %w", err)
	}
	defer rows.Close()

	var groups []*AgentGroup
	for rows.Next() {
		group, err := r.scanAgentGroup(rows)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}

	return groups, rows.Err()
}

// SetAgentsForGroup replaces all agents in a group with the provided list.
func (r *AgentGroupRepository) SetAgentsForGroup(ctx context.Context, groupID string, agentIDs []string, addedBy string) error {
	// Start a transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Remove existing memberships
	if _, err := tx.Exec(ctx, "DELETE FROM agent_group_memberships WHERE group_id = $1", groupID); err != nil {
		return fmt.Errorf("remove existing memberships: %w", err)
	}

	// Add new memberships
	if len(agentIDs) > 0 {
		now := time.Now()
		for _, agentID := range agentIDs {
			membershipID := uuid.New().String()
			query := `
				INSERT INTO agent_group_memberships (id, group_id, agent_id, added_at, added_by)
				VALUES ($1, $2, $3, $4, NULLIF($5, '')::uuid)
			`
			if _, err := tx.Exec(ctx, query, membershipID, groupID, agentID, now, addedBy); err != nil {
				return fmt.Errorf("add agent %s to group: %w", agentID, err)
			}
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// scanAgentGroup scans an agent group from a database row.
func (r *AgentGroupRepository) scanAgentGroup(row interface{ Scan(...interface{}) error }) (*AgentGroup, error) {
	var group AgentGroup
	var orgID, ownerUserID, policyID *string
	var tags []string

	err := row.Scan(
		&group.ID,
		&group.Name,
		&group.Description,
		&orgID,
		&ownerUserID,
		&group.Type,
		&group.Location,
		&tags,
		&policyID,
		&group.CreatedAt,
		&group.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Convert pointers to strings
	if orgID != nil {
		group.OrganizationID = *orgID
	}
	if ownerUserID != nil {
		group.OwnerUserID = *ownerUserID
	}
	if policyID != nil {
		group.PolicyID = *policyID
	}
	group.Tags = tags

	return &group, nil
}
