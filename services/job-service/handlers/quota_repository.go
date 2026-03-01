// Package handler provides quota repository implementation.
package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// quotaRepository implements the QuotaRepository interface.
type quotaRepository struct {
	db *pgxpool.Pool
}

// NewQuotaRepository creates a new quota repository.
func NewQuotaRepository(db *pgxpool.Pool) QuotaRepository {
	return &quotaRepository{db: db}
}

// CheckQuota checks if an entity has sufficient quota for a given operation.
func (r *quotaRepository) CheckQuota(ctx context.Context, entityID, entityType, quotaType string, increment int) (bool, int64, error) {
	// Call the database function to check and deduct quota
	var allowed bool
	var remaining int64

	query := `
		SELECT check_quota(
			$1::uuid,
			$2::varchar,
			$3::varchar,
			$4::integer
		) as allowed
	`

	err := r.db.QueryRow(ctx, query, entityID, entityType, quotaType, increment).Scan(&allowed)
	if err != nil {
		return false, 0, fmt.Errorf("check quota: %w", err)
	}

	// Get the current remaining amount
	remainingQuery := `
		SELECT "limit" - used as remaining
		FROM print_quotas
		WHERE entity_id = $1::uuid
		  AND entity_type = $2
		  AND quota_type = $3
	`

	err = r.db.QueryRow(ctx, remainingQuery, entityID, entityType, quotaType).Scan(&remaining)
	if err != nil {
		if err == pgx.ErrNoRows {
			// No quota set means unlimited
			return true, -1, nil
		}
		return false, 0, fmt.Errorf("get remaining: %w", err)
	}

	return allowed, remaining, nil
}

// GetQuota retrieves a specific quota configuration.
func (r *quotaRepository) GetQuota(ctx context.Context, entityID, entityType, quotaType, period string) (*PrintQuota, error) {
	query := `
		SELECT id, entity_id, entity_type, quota_type, period,
		       "limit", used, reset_date, created_at, updated_at
		FROM print_quotas
		WHERE entity_id = $1::uuid
		  AND entity_type = $2
		  AND quota_type = COALESCE($3, quota_type)
		  AND period = COALESCE($4, period)
		LIMIT 1
	`

	var quota PrintQuota
	err := r.db.QueryRow(ctx, query, entityID, entityType, nullIfEmpty(quotaType), nullIfEmpty(period)).Scan(
		&quota.ID, &quota.EntityID, &quota.EntityType, &quota.QuotaType,
		&quota.Period, &quota.Limit, &quota.Used, &quota.ResetDate,
		&quota.CreatedAt, &quota.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get quota: %w", err)
	}

	return &quota, nil
}

// ListQuotas retrieves all quotas for an entity.
func (r *quotaRepository) ListQuotas(ctx context.Context, entityID, entityType string) ([]*PrintQuota, error) {
	query := `
		SELECT id, entity_id, entity_type, quota_type, period,
		       "limit", used, reset_date, created_at, updated_at
		FROM print_quotas
		WHERE entity_id = $1::uuid
		  AND entity_type = $2
		ORDER BY period DESC, quota_type
	`

	rows, err := r.db.Query(ctx, query, entityID, entityType)
	if err != nil {
		return nil, fmt.Errorf("list quotas: %w", err)
	}
	defer rows.Close()

	var quotas []*PrintQuota
	for rows.Next() {
		var quota PrintQuota
		if err := rows.Scan(
			&quota.ID, &quota.EntityID, &quota.EntityType, &quota.QuotaType,
			&quota.Period, &quota.Limit, &quota.Used, &quota.ResetDate,
			&quota.CreatedAt, &quota.UpdatedAt,
		); err != nil {
			return nil, err
		}
		quotas = append(quotas, &quota)
	}

	return quotas, nil
}

// SetQuota creates or updates a quota configuration.
func (r *quotaRepository) SetQuota(ctx context.Context, quota *PrintQuota) error {
	query := `
		INSERT INTO print_quotas (
			id, entity_id, entity_type, quota_type, period,
			"limit", used, reset_date, created_at, updated_at
		) VALUES (
			$1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10
		)
		ON CONFLICT (entity_id, entity_type, quota_type, period)
		DO UPDATE SET
			"limit" = EXCLUDED."limit",
			reset_date = EXCLUDED.reset_date,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.db.Exec(ctx, query,
		quota.ID, quota.EntityID, quota.EntityType, quota.QuotaType,
		quota.Period, quota.Limit, quota.Used, quota.ResetDate,
		quota.CreatedAt, quota.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("set quota: %w", err)
	}

	// Log the quota change
	r.logQuotaHistory(ctx, quota, "adjusted", quota.Limit, fmt.Sprintf("Quota set to %d", quota.Limit))

	return nil
}

// ResetQuota resets the usage counter for a quota.
func (r *quotaRepository) ResetQuota(ctx context.Context, quotaID string) error {
	query := `
		UPDATE print_quotas
		SET used = 0,
		    updated_at = $1
		WHERE id = $2::uuid
		RETURNING entity_id, entity_type, quota_type, "limit"
	`

	var entityID, entityType, quotaType string
	var limit int
	err := r.db.QueryRow(ctx, query, time.Now(), quotaID).Scan(&entityID, &entityType, &quotaType, &limit)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("quota not found")
		}
		return fmt.Errorf("reset quota: %w", err)
	}

	// Log the reset
	quota := &PrintQuota{
		ID:         quotaID,
		EntityID:   entityID,
		EntityType: entityType,
		QuotaType:  quotaType,
		Limit:      limit,
	}
	r.logQuotaHistory(ctx, quota, "reset", 0, "Quota reset to 0")

	return nil
}

// DeleteQuota removes a quota configuration.
func (r *quotaRepository) DeleteQuota(ctx context.Context, quotaID string) error {
	query := `DELETE FROM print_quotas WHERE id = $1::uuid`

	cmdTag, err := r.db.Exec(ctx, query, quotaID)
	if err != nil {
		return fmt.Errorf("delete quota: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("quota not found")
	}

	return nil
}

// GetQuotaHistory retrieves historical quota usage records.
func (r *quotaRepository) GetQuotaHistory(ctx context.Context, entityID, entityType string, limit, offset int) ([]*QuotaHistoryEntry, int, error) {
	// Build query with filters
	baseQuery := `
		SELECT id, quota_id, entity_id, entity_type, quota_type,
		       action, amount, previous, remaining, description, created_at
		FROM quota_history
		WHERE ($1::uuid = ''::uuid OR entity_id = $1::uuid)
		  AND ($2 = '' OR entity_type = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	countQuery := `
		SELECT COUNT(*)
		FROM quota_history
		WHERE ($1::uuid = ''::uuid OR entity_id = $1::uuid)
		  AND ($2 = '' OR entity_type = $2)
	`

	// Get total count
	var total int
	if err := r.db.QueryRow(ctx, countQuery, nullIfEmpty(entityID), entityType).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count history: %w", err)
	}

	// Get history entries
	rows, err := r.db.Query(ctx, baseQuery, nullIfEmpty(entityID), entityType, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query history: %w", err)
	}
	defer rows.Close()

	var history []*QuotaHistoryEntry
	for rows.Next() {
		var entry QuotaHistoryEntry
		if err := rows.Scan(
			&entry.ID, &entry.QuotaID, &entry.EntityID, &entry.EntityType,
			&entry.QuotaType, &entry.Action, &entry.Amount, &entry.Previous,
			&entry.Remaining, &entry.Description, &entry.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		history = append(history, &entry)
	}

	return history, total, nil
}

// logQuotaHistory creates a history entry for quota changes.
func (r *quotaRepository) logQuotaHistory(ctx context.Context, quota *PrintQuota, action string, amount int, description string) error {
	// Create history table if it doesn't exist
	initQuery := `
		CREATE TABLE IF NOT EXISTS quota_history (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			quota_id UUID NOT NULL REFERENCES print_quotas(id) ON DELETE CASCADE,
			entity_id UUID NOT NULL,
			entity_type VARCHAR(20) NOT NULL,
			quota_type VARCHAR(50) NOT NULL,
			action VARCHAR(50) NOT NULL,
			amount INTEGER NOT NULL,
			previous INTEGER NOT NULL,
			remaining INTEGER NOT NULL,
			description TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_quota_history_entity ON quota_history(entity_id, entity_type);
		CREATE INDEX IF NOT EXISTS idx_quota_history_quota ON quota_history(quota_id);
	`
	r.db.Exec(ctx, initQuery)

	query := `
		INSERT INTO quota_history (
			quota_id, entity_id, entity_type, quota_type,
			action, amount, previous, remaining, description
		) VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.Exec(ctx, query,
		quota.ID, quota.EntityID, quota.EntityType, quota.QuotaType,
		action, amount, quota.Used, quota.Limit-quota.Used, description,
	)

	return err
}
