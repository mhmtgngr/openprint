// Package repository provides data access layer for the auth service.
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

// User represents a user account.
type User struct {
	ID             string
	Email          string
	Password       string // Hashed password
	FirstName      string
	LastName       string
	Role           string
	OrganizationID *string
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
	LastLoginAt    *time.Time
}

// GetOrgID returns the organization ID or empty string if nil.
func (u *User) GetOrgID() string {
	if u.OrganizationID == nil {
		return ""
	}
	return *u.OrganizationID
}

// UserRepository handles user data operations.
type UserRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository creates a new user repository.
func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// Create inserts a new user.
func (r *UserRepository) Create(ctx context.Context, user *User) error {
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	query := `
		INSERT INTO users (id, email, password, first_name, last_name, role, organization_id, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at
	`

	err := r.db.QueryRow(ctx, query,
		user.ID,
		user.Email,
		user.Password,
		user.FirstName,
		user.LastName,
		user.Role,
		user.OrganizationID,
		user.IsActive,
		user.CreatedAt,
		user.UpdatedAt,
	).Scan(&user.ID, &user.CreatedAt)

	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	return nil
}

// FindByID retrieves a user by ID.
func (r *UserRepository) FindByID(ctx context.Context, id string) (*User, error) {
	query := `
		SELECT id, email, password, first_name, last_name, role, organization_id, is_active, created_at, updated_at, last_login_at
		FROM users
		WHERE id = $1
	`

	user, err := r.scanUser(r.db.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.ErrNotFound
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	return user, nil
}

// FindByEmail retrieves a user by email.
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, password, first_name, last_name, role, organization_id, is_active, created_at, updated_at, last_login_at
		FROM users
		WHERE email = $1
	`

	user, err := r.scanUser(r.db.QueryRow(ctx, query, email))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.ErrNotFound
		}
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	return user, nil
}

// FindByOrganization retrieves all users for an organization.
func (r *UserRepository) FindByOrganization(ctx context.Context, orgID string, limit, offset int) ([]*User, error) {
	query := `
		SELECT id, email, password, first_name, last_name, role, organization_id, is_active, created_at, updated_at, last_login_at
		FROM users
		WHERE organization_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, orgID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("find users by organization: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user, err := r.scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, rows.Err()
}

// Update updates a user.
func (r *UserRepository) Update(ctx context.Context, user *User) error {
	user.UpdatedAt = time.Now()

	query := `
		UPDATE users
		SET email = $2, password = $3, first_name = $4, last_name = $5, role = $6,
		    organization_id = $7, is_active = $8, updated_at = $9, last_login_at = $10
		WHERE id = $1
	`

	cmdTag, err := r.db.Exec(ctx, query,
		user.ID,
		user.Email,
		user.Password,
		user.FirstName,
		user.LastName,
		user.Role,
		user.OrganizationID,
		user.IsActive,
		user.UpdatedAt,
		user.LastLoginAt,
	)

	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.ErrNotFound
	}

	return nil
}

// UpdatePassword updates a user's password.
func (r *UserRepository) UpdatePassword(ctx context.Context, userID, hashedPassword string) error {
	query := `
		UPDATE users
		SET password = $2, updated_at = $3
		WHERE id = $1
	`

	cmdTag, err := r.db.Exec(ctx, query, userID, hashedPassword, time.Now())
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.ErrNotFound
	}

	return nil
}

// Delete soft deletes a user.
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	query := `UPDATE users SET is_active = false, updated_at = $2 WHERE id = $1`

	cmdTag, err := r.db.Exec(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.ErrNotFound
	}

	return nil
}

// List retrieves users with pagination.
func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]*User, int, error) {
	// Get total count
	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE is_active = true").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	// Get users
	query := `
		SELECT id, email, password, first_name, last_name, role, organization_id, is_active, created_at, updated_at, last_login_at
		FROM users
		WHERE is_active = true
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user, err := r.scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, user)
	}

	return users, total, rows.Err()
}

// Activate activates a user account.
func (r *UserRepository) Activate(ctx context.Context, id string) error {
	query := `UPDATE users SET is_active = true, updated_at = $2 WHERE id = $1`

	cmdTag, err := r.db.Exec(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("activate user: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.ErrNotFound
	}

	return nil
}

// Deactivate deactivates a user account.
func (r *UserRepository) Deactivate(ctx context.Context, id string) error {
	query := `UPDATE users SET is_active = false, updated_at = $2 WHERE id = $1`

	cmdTag, err := r.db.Exec(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("deactivate user: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.ErrNotFound
	}

	return nil
}

// SetRole updates a user's role.
func (r *UserRepository) SetRole(ctx context.Context, id, role string) error {
	query := `UPDATE users SET role = $2, updated_at = $3 WHERE id = $1`

	cmdTag, err := r.db.Exec(ctx, query, id, role, time.Now())
	if err != nil {
		return fmt.Errorf("set user role: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.ErrNotFound
	}

	return nil
}

// ExistsByEmail checks if a user with the given email exists.
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", email).Scan(&exists)
	return exists, err
}

// ExistsByID checks if a user with the given ID exists.
func (r *UserRepository) ExistsByID(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", id).Scan(&exists)
	return exists, err
}

// FindOrCreateOrganizationUser finds a user by email within an organization, or creates one.
func (r *UserRepository) FindOrCreateOrganizationUser(ctx context.Context, orgID, email, firstName, lastName string) (*User, error) {
	// Try to find first
	user, err := r.FindByEmail(ctx, email)
	if err == nil {
		return user, nil
	}

	// Create new user
	newUser := &User{
		ID:             uuid.New().String(),
		Email:          email,
		FirstName:      firstName,
		LastName:       lastName,
		Role:           "user",
		OrganizationID: &orgID,
		IsActive:       true,
		Password:       "", // No password - SSO only
	}

	if err := r.Create(ctx, newUser); err != nil {
		return nil, err
	}

	return newUser, nil
}

// scanUser scans a user from a database row.
func (r *UserRepository) scanUser(row interface{ Scan(...interface{}) error }) (*User, error) {
	var user User
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.FirstName,
		&user.LastName,
		&user.Role,
		&user.OrganizationID,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
	)
	return &user, err
}
