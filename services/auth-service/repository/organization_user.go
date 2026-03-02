// Package repository provides data access layer for auth service.
// This file contains organization user repository for managing tenant members.
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
)

// OrganizationUserRole represents the role of a user within an organization.
type OrganizationUserRole string

const (
	// OrgRoleOwner is the organization owner with full access.
	OrgRoleOwner OrganizationUserRole = "owner"
	// OrgRoleAdmin is the organization administrator.
	OrgRoleAdmin OrganizationUserRole = "admin"
	// OrgRoleMember is a regular organization member.
	OrgRoleMember OrganizationUserRole = "member"
	// OrgRoleViewer is a read-only organization member.
	OrgRoleViewer OrganizationUserRole = "viewer"
	// OrgRoleBilling manages billing and invoices.
	OrgRoleBilling OrganizationUserRole = "billing"
)

// OrganizationUser represents a user's membership in an organization.
type OrganizationUser struct {
	ID             string                 `json:"id" db:"id"`
	OrganizationID string                 `json:"organization_id" db:"organization_id"`
	UserID         string                 `json:"user_id" db:"user_id"`
	Role           OrganizationUserRole   `json:"role" db:"role"`
	Settings       map[string]interface{} `json:"settings,omitempty" db:"settings"`
	JoinedAt       time.Time              `json:"joined_at" db:"joined_at"`
	InvitedBy      string                 `json:"invited_by,omitempty" db:"invited_by"`
	DeletedAt      *time.Time             `json:"deleted_at,omitempty" db:"deleted_at"`
}

// OrganizationUserRepository provides data access for organization members.
type OrganizationUserRepository struct {
	db *pgxpool.Pool
}

// NewOrganizationUserRepository creates a new organization user repository.
func NewOrganizationUserRepository(db *pgxpool.Pool) *OrganizationUserRepository {
	return &OrganizationUserRepository{db: db}
}

// Add adds a user to an organization.
func (r *OrganizationUserRepository) Add(ctx context.Context, orgUser *OrganizationUser) error {
	orgUser.ID = uuid.New().String()
	orgUser.JoinedAt = time.Now().UTC()

	query := `
		INSERT INTO organization_users (id, organization_id, user_id, role, settings, joined_at, invited_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (organization_id, user_id) DO UPDATE SET
			role = EXCLUDED.role,
			settings = EXCLUDED.settings,
			deleted_at = NULL
	`

	_, err := r.db.Exec(ctx, query,
		orgUser.ID, orgUser.OrganizationID, orgUser.UserID, orgUser.Role,
		orgUser.Settings, orgUser.JoinedAt, orgUser.InvitedBy,
	)

	if err != nil {
		return apperrors.Wrap(err, "failed to add user to organization", 500)
	}

	return nil
}

// Get retrieves an organization user by ID.
func (r *OrganizationUserRepository) Get(ctx context.Context, id string) (*OrganizationUser, error) {
	query := `
		SELECT id, organization_id, user_id, role, settings, joined_at, invited_by, deleted_at
		FROM organization_users
		WHERE id = $1 AND deleted_at IS NULL
	`

	orgUser := &OrganizationUser{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&orgUser.ID, &orgUser.OrganizationID, &orgUser.UserID, &orgUser.Role,
		&orgUser.Settings, &orgUser.JoinedAt, &orgUser.InvitedBy, &orgUser.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get organization user", 500)
	}

	return orgUser, nil
}

// GetByOrganizationAndUser retrieves a user's membership in an organization.
func (r *OrganizationUserRepository) GetByOrganizationAndUser(ctx context.Context, orgID, userID string) (*OrganizationUser, error) {
	query := `
		SELECT id, organization_id, user_id, role, settings, joined_at, invited_by, deleted_at
		FROM organization_users
		WHERE organization_id = $1 AND user_id = $2 AND deleted_at IS NULL
	`

	orgUser := &OrganizationUser{}
	err := r.db.QueryRow(ctx, query, orgID, userID).Scan(
		&orgUser.ID, &orgUser.OrganizationID, &orgUser.UserID, &orgUser.Role,
		&orgUser.Settings, &orgUser.JoinedAt, &orgUser.InvitedBy, &orgUser.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.Wrap(err, "failed to get organization user", 500)
	}

	return orgUser, nil
}

// ListByOrganization retrieves all users in an organization.
func (r *OrganizationUserRepository) ListByOrganization(ctx context.Context, orgID string, limit, offset int) ([]*OrganizationUser, int, error) {
	// Get total count
	countQuery := `
		SELECT COUNT(*)
		FROM organization_users
		WHERE organization_id = $1 AND deleted_at IS NULL
	`
	var total int
	err := r.db.QueryRow(ctx, countQuery, orgID).Scan(&total)
	if err != nil {
		return nil, 0, apperrors.Wrap(err, "failed to count organization users", 500)
	}

	// Get paginated results
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	query := `
		SELECT ou.id, ou.organization_id, ou.user_id, ou.role, ou.settings, ou.joined_at, ou.invited_by, ou.deleted_at,
		       u.email, u.name
		FROM organization_users ou
		INNER JOIN users u ON u.id = ou.user_id
		WHERE ou.organization_id = $1 AND ou.deleted_at IS NULL
		ORDER BY ou.joined_at ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, orgID, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Wrap(err, "failed to list organization users", 500)
	}
	defer rows.Close()

	orgUsers := []*OrganizationUser{}
	for rows.Next() {
		orgUser := &OrganizationUser{}
		err := rows.Scan(
			&orgUser.ID, &orgUser.OrganizationID, &orgUser.UserID, &orgUser.Role,
			&orgUser.Settings, &orgUser.JoinedAt, &orgUser.InvitedBy, &orgUser.DeletedAt,
		)
		if err != nil {
			return nil, 0, apperrors.Wrap(err, "failed to scan organization user", 500)
		}
		orgUsers = append(orgUsers, orgUser)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, apperrors.Wrap(err, "error iterating organization users", 500)
	}

	return orgUsers, total, nil
}

// ListByUser retrieves all organizations for a user.
func (r *OrganizationUserRepository) ListByUser(ctx context.Context, userID string) ([]*OrganizationUser, error) {
	query := `
		SELECT ou.id, ou.organization_id, ou.user_id, ou.role, ou.settings, ou.joined_at, ou.invited_by, ou.deleted_at
		FROM organization_users ou
		WHERE ou.user_id = $1 AND ou.deleted_at IS NULL
		ORDER BY ou.joined_at ASC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to list user organizations", 500)
	}
	defer rows.Close()

	orgUsers := []*OrganizationUser{}
	for rows.Next() {
		orgUser := &OrganizationUser{}
		err := rows.Scan(
			&orgUser.ID, &orgUser.OrganizationID, &orgUser.UserID, &orgUser.Role,
			&orgUser.Settings, &orgUser.JoinedAt, &orgUser.InvitedBy, &orgUser.DeletedAt,
		)
		if err != nil {
			return nil, apperrors.Wrap(err, "failed to scan organization user", 500)
		}
		orgUsers = append(orgUsers, orgUser)
	}

	return orgUsers, nil
}

// UpdateRole updates a user's role in an organization.
func (r *OrganizationUserRepository) UpdateRole(ctx context.Context, orgID, userID string, role OrganizationUserRole) error {
	query := `
		UPDATE organization_users
		SET role = $3
		WHERE organization_id = $1 AND user_id = $2 AND deleted_at IS NULL
	`

	result, err := r.db.Exec(ctx, query, orgID, userID, role)
	if err != nil {
		return apperrors.Wrap(err, "failed to update user role", 500)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return apperrors.ErrNotFound
	}

	return nil
}

// Remove removes a user from an organization (soft delete).
func (r *OrganizationUserRepository) Remove(ctx context.Context, orgID, userID string) error {
	now := time.Now().UTC()
	query := `
		UPDATE organization_users
		SET deleted_at = $3
		WHERE organization_id = $1 AND user_id = $2 AND deleted_at IS NULL
	`

	result, err := r.db.Exec(ctx, query, orgID, userID, now)
	if err != nil {
		return apperrors.Wrap(err, "failed to remove user from organization", 500)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return apperrors.ErrNotFound
	}

	return nil
}

// IsMember checks if a user is a member of an organization.
func (r *OrganizationUserRepository) IsMember(ctx context.Context, orgID, userID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM organization_users
			WHERE organization_id = $1 AND user_id = $2 AND deleted_at IS NULL
		)
	`
	var exists bool
	err := r.db.QueryRow(ctx, query, orgID, userID).Scan(&exists)
	return exists, err
}

// GetMemberCount returns the number of members in an organization.
func (r *OrganizationUserRepository) GetMemberCount(ctx context.Context, orgID string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM organization_users
		WHERE organization_id = $1 AND deleted_at IS NULL
	`
	var count int
	err := r.db.QueryRow(ctx, query, orgID).Scan(&count)
	return count, err
}

// HasRole checks if a user has a specific role or higher in an organization.
func (r *OrganizationUserRepository) HasRole(ctx context.Context, orgID, userID string, minRole OrganizationUserRole) (bool, error) {
	orgUser, err := r.GetByOrganizationAndUser(ctx, orgID, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return false, nil
		}
		return false, err
	}

	return r.compareRoles(orgUser.Role, minRole), nil
}

// compareRoles returns true if userRole is at least as high as minRole.
func (r *OrganizationUserRepository) compareRoles(userRole, minRole OrganizationUserRole) bool {
	roleHierarchy := map[OrganizationUserRole]int{
		OrgRoleViewer: 0,
		OrgRoleBilling: 1,
		OrgRoleMember:  2,
		OrgRoleAdmin:   3,
		OrgRoleOwner:   4,
	}

	userLevel := roleHierarchy[userRole]
	minLevel := roleHierarchy[minRole]

	return userLevel >= minLevel
}

// GetOwners retrieves all owners of an organization.
func (r *OrganizationUserRepository) GetOwners(ctx context.Context, orgID string) ([]*OrganizationUser, error) {
	query := `
		SELECT id, organization_id, user_id, role, settings, joined_at, invited_by, deleted_at
		FROM organization_users
		WHERE organization_id = $1 AND role = 'owner' AND deleted_at IS NULL
	`

	rows, err := r.db.Query(ctx, query, orgID)
	if err != nil {
		return nil, apperrors.Wrap(err, "failed to get organization owners", 500)
	}
	defer rows.Close()

	owners := []*OrganizationUser{}
	for rows.Next() {
		orgUser := &OrganizationUser{}
		err := rows.Scan(
			&orgUser.ID, &orgUser.OrganizationID, &orgUser.UserID, &orgUser.Role,
			&orgUser.Settings, &orgUser.JoinedAt, &orgUser.InvitedBy, &orgUser.DeletedAt,
		)
		if err != nil {
			return nil, apperrors.Wrap(err, "failed to scan organization user", 500)
		}
		owners = append(owners, orgUser)
	}

	return owners, nil
}

// TransferOwnership transfers ownership from one user to another.
// The previous owner becomes an admin.
func (r *OrganizationUserRepository) TransferOwnership(ctx context.Context, orgID, fromUserID, toUserID string) error {
	// Start transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return apperrors.Wrap(err, "failed to start transaction", 500)
	}
	defer tx.Rollback(ctx)

	// Verify from user is owner
	isOwner, err := r.HasRole(ctx, orgID, fromUserID, OrgRoleOwner)
	if err != nil {
		return err
	}
	if !isOwner {
		return apperrors.New("only owner can transfer ownership", 403)
	}

	// Update to user to owner
	_, err = tx.Exec(ctx, `
		UPDATE organization_users
		SET role = 'owner'
		WHERE organization_id = $1 AND user_id = $2 AND deleted_at IS NULL
	`, orgID, toUserID)
	if err != nil {
		return apperrors.Wrap(err, "failed to set new owner", 500)
	}

	// Demote from user to admin
	_, err = tx.Exec(ctx, `
		UPDATE organization_users
		SET role = 'admin'
		WHERE organization_id = $1 AND user_id = $2 AND deleted_at IS NULL
	`, orgID, fromUserID)
	if err != nil {
		return apperrors.Wrap(err, "failed to demote previous owner", 500)
	}

	return tx.Commit(ctx)
}

// UpdateSettings updates a user's settings in an organization.
func (r *OrganizationUserRepository) UpdateSettings(ctx context.Context, orgID, userID string, settings map[string]interface{}) error {
	query := `
		UPDATE organization_users
		SET settings = $3
		WHERE organization_id = $1 AND user_id = $2 AND deleted_at IS NULL
	`

	result, err := r.db.Exec(ctx, query, orgID, userID, settings)
	if err != nil {
		return apperrors.Wrap(err, "failed to update user settings", 500)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return apperrors.ErrNotFound
	}

	return nil
}
