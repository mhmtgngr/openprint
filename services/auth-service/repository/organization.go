// Package repository provides data access layer for auth service.
// This file contains organization repository with tenant data access.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
)

// OrganizationStatus represents the status of an organization.
type OrganizationStatus string

const (
	// OrgStatusActive is for active organizations.
	OrgStatusActive OrganizationStatus = "active"
	// OrgStatusSuspended is for suspended organizations.
	OrgStatusSuspended OrganizationStatus = "suspended"
	// OrgStatusDeleted is for deleted organizations (soft delete).
	OrgStatusDeleted OrganizationStatus = "deleted"
	// OrgStatusTrial is for trial organizations.
	OrgStatusTrial OrganizationStatus = "trial"
)

// Organization represents a tenant organization.
type Organization struct {
	ID          string            `json:"id" db:"id"`
	Name        string            `json:"name" db:"name"`
	Slug        string            `json:"slug" db:"slug"`
	Status      OrganizationStatus `json:"status" db:"status"`
	LogoURL     string            `json:"logo_url,omitempty" db:"logo_url"`
	Website     string            `json:"website,omitempty" db:"website"`
	Description string            `json:"description,omitempty" db:"description"`
	Settings    map[string]interface{} `json:"settings,omitempty" db:"settings"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" db:"updated_at"`
	DeletedAt   *time.Time        `json:"deleted_at,omitempty" db:"deleted_at"`
}

// OrganizationRepository provides data access for organizations.
type OrganizationRepository struct {
	db *pgxpool.Pool
}

// NewOrganizationRepository creates a new organization repository.
func NewOrganizationRepository(db *pgxpool.Pool) *OrganizationRepository {
	return &OrganizationRepository{db: db}
}

// Create creates a new organization.
func (r *OrganizationRepository) Create(ctx context.Context, org *Organization) error {
	org.ID = uuid.New().String()
	org.CreatedAt = time.Now().UTC()
	org.UpdatedAt = org.CreatedAt

	query := `
		INSERT INTO organizations (id, name, slug, status, logo_url, website, description, settings, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at
	`

	_, err := r.db.Exec(ctx, query,
		org.ID, org.Name, org.Slug, org.Status, org.LogoURL, org.Website, org.Description, org.Settings,
		org.CreatedAt, org.UpdatedAt,
	)

	if err != nil {
		return apperrors.Wrap(err, "failed to create organization", 500)
	}

	return nil
}

// GetByID retrieves an organization by ID.
func (r *OrganizationRepository) GetByID(ctx context.Context, id string) (*Organization, error) {
	query := `
		SELECT id, name, slug, status, logo_url, website, description, settings, created_at, updated_at, deleted_at
		FROM organizations
		WHERE id = $1 AND deleted_at IS NULL
	`

	org := &Organization{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&org.ID, &org.Name, &org.Slug, &org.Status, &org.LogoURL, &org.Website,
		&org.Description, &org.Settings, &org.CreatedAt, &org.UpdatedAt, &org.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get organization", 500)
	}

	return org, nil
}

// GetBySlug retrieves an organization by slug.
func (r *OrganizationRepository) GetBySlug(ctx context.Context, slug string) (*Organization, error) {
	query := `
		SELECT id, name, slug, status, logo_url, website, description, settings, created_at, updated_at, deleted_at
		FROM organizations
		WHERE slug = $1 AND deleted_at IS NULL
	`

	org := &Organization{}
	err := r.db.QueryRow(ctx, query, slug).Scan(
		&org.ID, &org.Name, &org.Slug, &org.Status, &org.LogoURL, &org.Website,
		&org.Description, &org.Settings, &org.CreatedAt, &org.UpdatedAt, &org.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get organization", 500)
	}

	return org, nil
}

// List retrieves a list of organizations with pagination.
func (r *OrganizationRepository) List(ctx context.Context, limit, offset int, status OrganizationStatus) ([]*Organization, int, error) {
	whereClause := "WHERE deleted_at IS NULL"
	args := []interface{}{}
	argCount := 1

	if status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
		argCount++
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM organizations " + whereClause
	var total int
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, apperrors.Wrap(err, "failed to count organizations", 500)
	}

	// Get paginated results
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	query := `
		SELECT id, name, slug, status, logo_url, website, description, settings, created_at, updated_at, deleted_at
		FROM organizations
		` + whereClause + `
		ORDER BY created_at DESC
		LIMIT $` + fmt.Sprint(argCount) + ` OFFSET $` + fmt.Sprint(argCount+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, apperrors.Wrap(err, "failed to list organizations", 500)
	}
	defer rows.Close()

	orgs := []*Organization{}
	for rows.Next() {
		org := &Organization{}
		err := rows.Scan(
			&org.ID, &org.Name, &org.Slug, &org.Status, &org.LogoURL, &org.Website,
			&org.Description, &org.Settings, &org.CreatedAt, &org.UpdatedAt, &org.DeletedAt,
		)
		if err != nil {
			return nil, 0, apperrors.Wrap(err, "failed to scan organization", 500)
		}
		orgs = append(orgs, org)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, apperrors.Wrap(err, "error iterating organizations", 500)
	}

	return orgs, total, nil
}

// Update updates an organization.
func (r *OrganizationRepository) Update(ctx context.Context, org *Organization) error {
	org.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE organizations
		SET name = $2, slug = $3, status = $4, logo_url = $5, website = $6, description = $7, settings = $8, updated_at = $9
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.Exec(ctx, query,
		org.ID, org.Name, org.Slug, org.Status, org.LogoURL, org.Website, org.Description, org.Settings, org.UpdatedAt,
	)

	if err != nil {
		return apperrors.Wrap(err, "failed to update organization", 500)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return apperrors.ErrNotFound
	}

	return nil
}

// UpdateStatus updates the status of an organization.
func (r *OrganizationRepository) UpdateStatus(ctx context.Context, id string, status OrganizationStatus) error {
	query := `
		UPDATE organizations
		SET status = $2, updated_at = $3
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.Exec(ctx, query, id, status, time.Now().UTC())
	if err != nil {
		return apperrors.Wrap(err, "failed to update organization status", 500)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return apperrors.ErrNotFound
	}

	return nil
}

// Delete soft deletes an organization.
func (r *OrganizationRepository) Delete(ctx context.Context, id string) error {
	now := time.Now().UTC()
	query := `
		UPDATE organizations
		SET deleted_at = $2, status = 'deleted', updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.Exec(ctx, query, id, now)
	if err != nil {
		return apperrors.Wrap(err, "failed to delete organization", 500)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return apperrors.ErrNotFound
	}

	return nil
}

// Exists checks if an organization exists.
func (r *OrganizationRepository) Exists(ctx context.Context, id string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM organizations WHERE id = $1 AND deleted_at IS NULL)`
	var exists bool
	err := r.db.QueryRow(ctx, query, id).Scan(&exists)
	return exists, err
}

// SlugExists checks if a slug is already in use.
func (r *OrganizationRepository) SlugExists(ctx context.Context, slug string, excludeID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM organizations WHERE slug = $1 AND id != COALESCE($2, '') AND deleted_at IS NULL)`
	var exists bool
	err := r.db.QueryRow(ctx, query, slug, excludeID).Scan(&exists)
	return exists, err
}

// GetUserOrganization returns the organization for a user.
func (r *OrganizationRepository) GetUserOrganization(ctx context.Context, userID string) (*Organization, string, error) {
	query := `
		SELECT o.id, o.name, o.slug, o.status, o.logo_url, o.website, o.description, o.settings, o.created_at, o.updated_at, o.deleted_at,
		       ou.role
		FROM organizations o
		INNER JOIN organization_users ou ON ou.organization_id = o.id
		WHERE ou.user_id = $1 AND o.deleted_at IS NULL AND ou.deleted_at IS NULL
		LIMIT 1
	`

	org := &Organization{}
	var role string
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&org.ID, &org.Name, &org.Slug, &org.Status, &org.LogoURL, &org.Website,
		&org.Description, &org.Settings, &org.CreatedAt, &org.UpdatedAt, &org.DeletedAt,
		&role,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", apperrors.ErrNotFound
		}
		return nil, "", apperrors.Wrap(err, "failed to get user organization", 500)
	}

	return org, role, nil
}

// SetTenantContext sets the tenant context for the current database session.
// This is used for Row Level Security (RLS) policies.
func (r *OrganizationRepository) SetTenantContext(ctx context.Context, tenantID string) error {
	_, err := r.db.Exec(ctx, "SET LOCAL app.tenant_id = $1", tenantID)
	return err
}

// ClearTenantContext clears the tenant context for the current database session.
func (r *OrganizationRepository) ClearTenantContext(ctx context.Context) error {
	_, err := r.db.Exec(ctx, "SET LOCAL app.tenant_id = NULL")
	return err
}

// EnableRowLevelSecurity enables RLS for the organizations table.
func (r *OrganizationRepository) EnableRowLevelSecurity(ctx context.Context) error {
	_, err := r.db.Exec(ctx, "ALTER TABLE organizations ENABLE ROW LEVEL SECURITY")
	return err
}

// CreateTenantPolicy creates a tenant isolation policy for organizations.
func (r *OrganizationRepository) CreateTenantPolicy(ctx context.Context) error {
	query := `
		CREATE POLICY IF NOT EXISTS tenant_isolation_policy ON organizations
		FOR ALL
		USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
		WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
	`
	_, err := r.db.Exec(ctx, query)
	return err
}
