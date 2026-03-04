// Package repository provides data access layer for enrollment token operations.
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EnrollmentToken represents an enrollment token for agent registration.
type EnrollmentToken struct {
	ID             string
	Token          string
	OrganizationID string
	Name           string
	CreatedBy      string
	MaxUses        int
	UseCount       int
	ExpiresAt      *time.Time
	RevokedAt      *time.Time
	RevokedBy      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// EnrollmentTokenRepository handles enrollment token data operations.
type EnrollmentTokenRepository struct {
	db *pgxpool.Pool
}

// NewEnrollmentTokenRepository creates a new enrollment token repository.
func NewEnrollmentTokenRepository(db *pgxpool.Pool) *EnrollmentTokenRepository {
	return &EnrollmentTokenRepository{db: db}
}

// Create generates a new enrollment token.
func (r *EnrollmentTokenRepository) Create(ctx context.Context, token *EnrollmentToken) error {
	now := time.Now()
	token.ID = uuid.New().String()
	token.CreatedAt = now
	token.UpdatedAt = now
	token.UseCount = 0

	query := `
		INSERT INTO enrollment_tokens (id, token, organization_id, name, created_by,
			max_uses, use_count, expires_at, revoked_at, revoked_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`

	err := r.db.QueryRow(ctx, query,
		token.ID,
		token.Token,
		token.OrganizationID,
		token.Name,
		token.CreatedBy,
		token.MaxUses,
		token.UseCount,
		token.ExpiresAt,
		token.RevokedAt,
		token.RevokedBy,
		token.CreatedAt,
		token.UpdatedAt,
	).Scan(&token.ID)

	if err != nil {
		return fmt.Errorf("create enrollment token: %w", err)
	}

	return nil
}

// FindByToken retrieves an enrollment token by its token string.
func (r *EnrollmentTokenRepository) FindByToken(ctx context.Context, tokenStr string) (*EnrollmentToken, error) {
	query := `
		SELECT id, token, organization_id, name, created_by,
		       max_uses, use_count, expires_at, revoked_at, revoked_by, created_at, updated_at
		FROM enrollment_tokens
		WHERE token = $1
	`

	var t EnrollmentToken
	err := r.db.QueryRow(ctx, query, tokenStr).Scan(
		&t.ID,
		&t.Token,
		&t.OrganizationID,
		&t.Name,
		&t.CreatedBy,
		&t.MaxUses,
		&t.UseCount,
		&t.ExpiresAt,
		&t.RevokedAt,
		&t.RevokedBy,
		&t.CreatedAt,
		&t.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("enrollment token not found")
		}
		return nil, fmt.Errorf("find enrollment token: %w", err)
	}

	return &t, nil
}

// Validate checks if an enrollment token is valid for use.
// A token is valid if:
// - It exists in the database
// - It has not been revoked
// - It has not exceeded its max uses (if max_uses > 0)
// - It has not expired (if expires_at is set)
func (r *EnrollmentTokenRepository) Validate(ctx context.Context, tokenStr, organizationID string) (bool, error) {
	token, err := r.FindByToken(ctx, tokenStr)
	if err != nil {
		return false, nil // Token not found, return false without error
	}

	// Check if token is revoked
	if token.RevokedAt != nil {
		return false, nil
	}

	// Check if token matches the organization (if organization is specified)
	if organizationID != "" && token.OrganizationID != "" && token.OrganizationID != organizationID {
		return false, nil
	}

	// Check if token has expired
	if token.ExpiresAt != nil && token.ExpiresAt.Before(time.Now()) {
		return false, nil
	}

	// Check if token has exceeded max uses
	if token.MaxUses > 0 && token.UseCount >= token.MaxUses {
		return false, nil
	}

	return true, nil
}

// IncrementUseCount increments the use count for a token.
func (r *EnrollmentTokenRepository) IncrementUseCount(ctx context.Context, tokenID string) error {
	query := `
		UPDATE enrollment_tokens
		SET use_count = use_count + 1, updated_at = $2
		WHERE id = $1
	`

	cmdTag, err := r.db.Exec(ctx, query, tokenID, time.Now())
	if err != nil {
		return fmt.Errorf("increment use count: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("enrollment token not found")
	}

	return nil
}

// Revoke revokes an enrollment token.
func (r *EnrollmentTokenRepository) Revoke(ctx context.Context, tokenID, revokedBy string) error {
	query := `
		UPDATE enrollment_tokens
		SET revoked_at = $2, revoked_by = $3, updated_at = $4
		WHERE id = $1
	`

	cmdTag, err := r.db.Exec(ctx, query, tokenID, time.Now(), revokedBy, time.Now())
	if err != nil {
		return fmt.Errorf("revoke enrollment token: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("enrollment token not found")
	}

	return nil
}

// List retrieves enrollment tokens with optional filtering.
func (r *EnrollmentTokenRepository) List(ctx context.Context, organizationID string, includeRevoked bool, limit int) ([]*EnrollmentToken, error) {
	if limit == 0 {
		limit = 100
	}

	query := `
		SELECT id, token, organization_id, name, created_by,
		       max_uses, use_count, expires_at, revoked_at, revoked_by, created_at, updated_at
		FROM enrollment_tokens
		WHERE ($1 = '' OR organization_id = $1)
			AND ($2 OR revoked_at IS NULL)
		ORDER BY created_at DESC
		LIMIT $3
	`

	rows, err := r.db.Query(ctx, query, organizationID, includeRevoked, limit)
	if err != nil {
		return nil, fmt.Errorf("list enrollment tokens: %w", err)
	}
	defer rows.Close()

	var tokens []*EnrollmentToken
	for rows.Next() {
		var t EnrollmentToken
		err := rows.Scan(
			&t.ID,
			&t.Token,
			&t.OrganizationID,
			&t.Name,
			&t.CreatedBy,
			&t.MaxUses,
			&t.UseCount,
			&t.ExpiresAt,
			&t.RevokedAt,
			&t.RevokedBy,
			&t.CreatedAt,
			&t.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, &t)
	}

	return tokens, rows.Err()
}

// Delete removes an enrollment token.
func (r *EnrollmentTokenRepository) Delete(ctx context.Context, tokenID string) error {
	query := `DELETE FROM enrollment_tokens WHERE id = $1`

	cmdTag, err := r.db.Exec(ctx, query, tokenID)
	if err != nil {
		return fmt.Errorf("delete enrollment token: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("enrollment token not found")
	}

	return nil
}
