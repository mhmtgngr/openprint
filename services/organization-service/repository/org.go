// Package repository provides data access layer for the organization service.
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openprint/openprint/internal/shared/errors"
)

// Organization represents an organization/tenant.
type Organization struct {
	ID        string
	Name      string
	Slug      string
	Plan      string
	Settings  map[string]interface{}
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Permission represents an organization user permission.
type Permission struct {
	ID             string
	OrganizationID string
	UserID         string
	PermissionType string
	GrantedAt      time.Time
	GrantedBy      *string
}

// OrganizationRepository handles organization data operations.
type OrganizationRepository struct {
	db *pgxpool.Pool
}

// NewOrganizationRepository creates a new organization repository.
func NewOrganizationRepository(db *pgxpool.Pool) *OrganizationRepository {
	return &OrganizationRepository{db: db}
}

// Create inserts a new organization.
func (r *OrganizationRepository) Create(ctx context.Context, org *Organization) error {
	now := time.Now()
	org.CreatedAt = now
	org.UpdatedAt = now
	org.ID = uuid.New().String()

	query := `
		INSERT INTO organizations (id, name, slug, plan, settings, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`

	err := r.db.QueryRow(ctx, query,
		org.ID,
		org.Name,
		org.Slug,
		org.Plan,
		org.Settings,
		org.CreatedAt,
		org.UpdatedAt,
	).Scan(&org.ID, &org.CreatedAt)

	if err != nil {
		return fmt.Errorf("create organization: %w", err)
	}

	return nil
}

// FindByID retrieves an organization by ID.
func (r *OrganizationRepository) FindByID(ctx context.Context, id string) (*Organization, error) {
	query := `
		SELECT id, name, slug, plan, settings, created_at, updated_at
		FROM organizations
		WHERE id = $1
	`

	org, err := r.scanOrganization(r.db.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.ErrNotFound
		}
		return nil, fmt.Errorf("find organization by id: %w", err)
	}

	return org, nil
}

// FindBySlug retrieves an organization by slug.
func (r *OrganizationRepository) FindBySlug(ctx context.Context, slug string) (*Organization, error) {
	query := `
		SELECT id, name, slug, plan, settings, created_at, updated_at
		FROM organizations
		WHERE slug = $1
	`

	org, err := r.scanOrganization(r.db.QueryRow(ctx, query, slug))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.ErrNotFound
		}
		return nil, fmt.Errorf("find organization by slug: %w", err)
	}

	return org, nil
}

// List retrieves all organizations.
func (r *OrganizationRepository) List(ctx context.Context) ([]*Organization, error) {
	query := `
		SELECT id, name, slug, plan, settings, created_at, updated_at
		FROM organizations
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}
	defer rows.Close()

	var orgs []*Organization
	for rows.Next() {
		org, err := r.scanOrganization(rows)
		if err != nil {
			return nil, err
		}
		orgs = append(orgs, org)
	}

	return orgs, rows.Err()
}

// Update updates an organization.
func (r *OrganizationRepository) Update(ctx context.Context, org *Organization) error {
	org.UpdatedAt = time.Now()

	query := `
		UPDATE organizations
		SET name = $2, slug = $3, plan = $4, settings = $5, updated_at = $6
		WHERE id = $1
	`

	cmdTag, err := r.db.Exec(ctx, query,
		org.ID,
		org.Name,
		org.Slug,
		org.Plan,
		org.Settings,
		org.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("update organization: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.ErrNotFound
	}

	return nil
}

// Delete deletes an organization.
func (r *OrganizationRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM organizations WHERE id = $1`

	cmdTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete organization: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.ErrNotFound
	}

	return nil
}

// AddMember adds a user to an organization.
func (r *OrganizationRepository) AddMember(ctx context.Context, orgID, userID string) error {
	query := `
		UPDATE users
		SET organization_id = $2, updated_at = $3
		WHERE id = $1
	`

	cmdTag, err := r.db.Exec(ctx, query, userID, orgID, time.Now())
	if err != nil {
		return fmt.Errorf("add member: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.ErrNotFound
	}

	return nil
}

// RemoveMember removes a user from an organization.
func (r *OrganizationRepository) RemoveMember(ctx context.Context, orgID, userID string) error {
	query := `
		UPDATE users
		SET organization_id = NULL, updated_at = $3
		WHERE id = $1 AND organization_id = $2
	`

	cmdTag, err := r.db.Exec(ctx, query, userID, orgID, time.Now())
	if err != nil {
		return fmt.Errorf("remove member: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.ErrNotFound
	}

	return nil
}

// ListMembers retrieves all members of an organization.
func (r *OrganizationRepository) ListMembers(ctx context.Context, orgID string) ([]string, error) {
	query := `
		SELECT id
		FROM users
		WHERE organization_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	var members []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		members = append(members, id)
	}

	return members, rows.Err()
}

// AddPermission adds a permission for a user in an organization.
func (r *OrganizationRepository) AddPermission(ctx context.Context, perm *Permission) error {
	perm.GrantedAt = time.Now()
	perm.ID = uuid.New().String()

	// First check if organization_permissions table exists, if not create it
	// For now, we'll assume it exists or use the user's role field
	query := `
		INSERT INTO organization_permissions (id, organization_id, user_id, permission_type, granted_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (organization_id, user_id) DO UPDATE
		SET permission_type = $4, granted_at = $5
	`

	_, err := r.db.Exec(ctx, query,
		perm.ID,
		perm.OrganizationID,
		perm.UserID,
		perm.PermissionType,
		perm.GrantedAt,
	)

	if err != nil {
		return fmt.Errorf("add permission: %w", err)
	}

	return nil
}

// RemovePermission removes a permission from a user in an organization.
func (r *OrganizationRepository) RemovePermission(ctx context.Context, orgID, userID string) error {
	query := `
		DELETE FROM organization_permissions
		WHERE organization_id = $1 AND user_id = $2
	`

	cmdTag, err := r.db.Exec(ctx, query, orgID, userID)
	if err != nil {
		return fmt.Errorf("remove permission: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.ErrNotFound
	}

	return nil
}

// ListPermissions retrieves all permissions for an organization.
func (r *OrganizationRepository) ListPermissions(ctx context.Context, orgID string) ([]*Permission, error) {
	query := `
		SELECT id, organization_id, user_id, permission_type, granted_at, granted_by
		FROM organization_permissions
		WHERE organization_id = $1
		ORDER BY granted_at DESC
	`

	rows, err := r.db.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("list permissions: %w", err)
	}
	defer rows.Close()

	var permissions []*Permission
	for rows.Next() {
		perm, err := r.scanPermission(rows)
		if err != nil {
			return nil, err
		}
		permissions = append(permissions, perm)
	}

	return permissions, rows.Err()
}

// GetUserPermission gets the permission type for a user in an organization.
func (r *OrganizationRepository) GetUserPermission(ctx context.Context, orgID, userID string) (string, error) {
	query := `
		SELECT permission_type
		FROM organization_permissions
		WHERE organization_id = $1 AND user_id = $2
	`

	var permType string
	err := r.db.QueryRow(ctx, query, orgID, userID).Scan(&permType)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "member", nil // Default permission
		}
		return "", fmt.Errorf("get user permission: %w", err)
	}

	return permType, nil
}

// scanOrganization scans an organization from a database row.
func (r *OrganizationRepository) scanOrganization(row interface{ Scan(...interface{}) error }) (*Organization, error) {
	var org Organization
	err := row.Scan(
		&org.ID,
		&org.Name,
		&org.Slug,
		&org.Plan,
		&org.Settings,
		&org.CreatedAt,
		&org.UpdatedAt,
	)
	return &org, err
}

// scanPermission scans a permission from a database row.
func (r *OrganizationRepository) scanPermission(row interface{ Scan(...interface{}) error }) (*Permission, error) {
	var perm Permission
	err := row.Scan(
		&perm.ID,
		&perm.OrganizationID,
		&perm.UserID,
		&perm.PermissionType,
		&perm.GrantedAt,
		&perm.GrantedBy,
	)
	return &perm, err
}

