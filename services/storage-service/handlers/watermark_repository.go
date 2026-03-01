// Package handler provides watermark repository implementation.
package handler

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// watermarkRepository implements the WatermarkRepository interface.
type watermarkRepository struct {
	db *pgxpool.Pool
}

// NewWatermarkRepository creates a new watermark repository.
func NewWatermarkRepository(db *pgxpool.Pool) WatermarkRepository {
	return &watermarkRepository{db: db}
}

// CreateTemplate creates a new watermark template.
func (r *watermarkRepository) CreateTemplate(ctx context.Context, template *WatermarkTemplate) error {
	// Ensure table exists
	r.initTable(ctx)

	query := `
		INSERT INTO watermark_templates (
			id, organization_id, name, type, content, position,
			opacity, rotation, font_size, font_color, image_data,
			is_default, apply_to_all, created_by, created_at, updated_at
		) VALUES (
			$1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)
	`

	_, err := r.db.Exec(ctx, query,
		template.ID, template.OrganizationID, template.Name, template.Type,
		template.Content, template.Position, template.Opacity, template.Rotation,
		template.FontSize, template.FontColor, template.ImageData,
		template.IsDefault, template.ApplyToAll, template.CreatedBy,
		template.CreatedAt, template.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("create watermark template: %w", err)
	}

	return nil
}

// GetTemplate retrieves a watermark template by ID.
func (r *watermarkRepository) GetTemplate(ctx context.Context, templateID string) (*WatermarkTemplate, error) {
	query := `
		SELECT id, organization_id, name, type, content, position,
		       opacity, rotation, font_size, font_color, image_data,
		       is_default, apply_to_all, created_by, created_at, updated_at
		FROM watermark_templates
		WHERE id = $1::uuid
	`

	var template WatermarkTemplate
	err := r.db.QueryRow(ctx, query, templateID).Scan(
		&template.ID, &template.OrganizationID, &template.Name, &template.Type,
		&template.Content, &template.Position, &template.Opacity, &template.Rotation,
		&template.FontSize, &template.FontColor, &template.ImageData,
		&template.IsDefault, &template.ApplyToAll, &template.CreatedBy,
		&template.CreatedAt, &template.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get watermark template: %w", err)
	}

	return &template, nil
}

// ListTemplates retrieves all watermark templates for an organization.
func (r *watermarkRepository) ListTemplates(ctx context.Context, organizationID string) ([]*WatermarkTemplate, error) {
	query := `
		SELECT id, organization_id, name, type, content, position,
		       opacity, rotation, font_size, font_color, image_data,
		       is_default, apply_to_all, created_by, created_at, updated_at
		FROM watermark_templates
		WHERE organization_id = $1::uuid
		ORDER BY is_default DESC, name
	`

	rows, err := r.db.Query(ctx, query, organizationID)
	if err != nil {
		return nil, fmt.Errorf("list watermark templates: %w", err)
	}
	defer rows.Close()

	var templates []*WatermarkTemplate
	for rows.Next() {
		var template WatermarkTemplate
		if err := rows.Scan(
			&template.ID, &template.OrganizationID, &template.Name, &template.Type,
			&template.Content, &template.Position, &template.Opacity, &template.Rotation,
			&template.FontSize, &template.FontColor, &template.ImageData,
			&template.IsDefault, &template.ApplyToAll, &template.CreatedBy,
			&template.CreatedAt, &template.UpdatedAt,
		); err != nil {
			return nil, err
		}
		templates = append(templates, &template)
	}

	return templates, nil
}

// UpdateTemplate updates an existing watermark template.
func (r *watermarkRepository) UpdateTemplate(ctx context.Context, template *WatermarkTemplate) error {
	query := `
		UPDATE watermark_templates
		SET name = $2, type = $3, content = $4, position = $5,
		    opacity = $6, rotation = $7, font_size = $8, font_color = $9,
		    image_data = COALESCE($10, image_data),
		    is_default = $11, apply_to_all = $12, updated_at = $13
		WHERE id = $1::uuid
	`

	_, err := r.db.Exec(ctx, query,
		template.ID, template.Name, template.Type, template.Content,
		template.Position, template.Opacity, template.Rotation,
		template.FontSize, template.FontColor, template.ImageData,
		template.IsDefault, template.ApplyToAll, template.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("update watermark template: %w", err)
	}

	return nil
}

// DeleteTemplate deletes a watermark template.
func (r *watermarkRepository) DeleteTemplate(ctx context.Context, templateID string) error {
	query := `DELETE FROM watermark_templates WHERE id = $1::uuid`

	cmdTag, err := r.db.Exec(ctx, query, templateID)
	if err != nil {
		return fmt.Errorf("delete watermark template: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("watermark template not found")
	}

	return nil
}

// initTable creates the watermark_templates table if it doesn't exist.
func (r *watermarkRepository) initTable(ctx context.Context) {
	query := `
		CREATE TABLE IF NOT EXISTS watermark_templates (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			type VARCHAR(50) NOT NULL CHECK (type IN ('text', 'image', 'overlay')),
			content TEXT,
			position VARCHAR(50) DEFAULT 'center' CHECK (position IN ('top-left', 'top-center', 'top-right', 'center', 'bottom-left', 'bottom-center', 'bottom-right')),
			opacity DECIMAL(3, 2) DEFAULT 0.3 CHECK (opacity >= 0 AND opacity <= 1),
			rotation INTEGER DEFAULT 0 CHECK (rotation >= 0 AND rotation <= 360),
			font_size INTEGER DEFAULT 48,
			font_color VARCHAR(7) DEFAULT '#CCCCCC',
			image_data BYTEA,
			is_default BOOLEAN DEFAULT false,
			apply_to_all BOOLEAN DEFAULT false,
			created_by VARCHAR(255),
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_watermark_templates_org ON watermark_templates(organization_id);
		CREATE TRIGGER update_watermark_templates_updated_at BEFORE UPDATE ON watermark_templates
		    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
	`

	r.db.Exec(ctx, query)
}
