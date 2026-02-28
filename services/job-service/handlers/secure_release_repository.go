// Package handler provides secure release repository implementation.
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/types"
)

// secureReleaseRepository implements the SecureReleaseRepository interface.
type secureReleaseRepository struct {
	db *pgxpool.Pool
}

// NewSecureReleaseRepository creates a new secure release repository.
func NewSecureReleaseRepository(db *pgxpool.Pool) SecureReleaseRepository {
	return &secureReleaseRepository{db: db}
}

// CreateSecureJob creates a new secure print job entry.
func (r *secureReleaseRepository) CreateSecureJob(ctx context.Context, job *SecurePrintJob) error {
	// Ensure tables exist
	r.initTables(ctx)

	// Serialize release data
	var releaseDataJSON []byte
	if job.ReleaseData != nil {
		releaseDataJSON, _ = json.Marshal(job.ReleaseData)
	}

	query := `
		INSERT INTO secure_print_jobs (
			id, job_id, user_id, release_method, release_data,
			status, expires_at, released_at, released_printer_id,
			release_attempts, max_release_attempts, created_at, updated_at
		) VALUES (
			$1::uuid, $2::uuid, $3::uuid, $4, $5::jsonb,
			$6, $7, $8, $9::uuid, $10, $11, $12, $13
		)
	`

	_, err := r.db.Exec(ctx, query,
		job.ID, job.JobID, job.UserID, job.ReleaseMethod, releaseDataJSON,
		job.Status, job.ExpiresAt, job.ReleasedAt, nullIfEmpty(job.ReleasedPrinterID),
		job.ReleaseAttempts, job.MaxReleaseAttempts, job.CreatedAt, job.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("create secure job: %w", err)
	}

	return nil
}

// GetSecureJob retrieves a secure job by ID.
func (r *secureReleaseRepository) GetSecureJob(ctx context.Context, secureJobID string) (*SecurePrintJob, error) {
	query := `
		SELECT id, job_id, user_id, release_method, release_data,
		       status, expires_at, released_at, released_printer_id,
		       release_attempts, max_release_attempts, created_at, updated_at
		FROM secure_print_jobs
		WHERE id = $1::uuid
	`

	var job SecurePrintJob
	var releaseDataJSON []byte
	var releasedAt, releasedPrinterID *string

	err := r.db.QueryRow(ctx, query, secureJobID).Scan(
		&job.ID, &job.JobID, &job.UserID, &job.ReleaseMethod, &releaseDataJSON,
		&job.Status, &job.ExpiresAt, &releasedAt, &releasedPrinterID,
		&job.ReleaseAttempts, &job.MaxReleaseAttempts, &job.CreatedAt, &job.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("secure job not found")
		}
		return nil, fmt.Errorf("get secure job: %w", err)
	}

	// Parse release data
	if releaseDataJSON != nil {
		json.Unmarshal(releaseDataJSON, &job.ReleaseData)
	}

	// Parse optional fields
	if releasedAt != nil {
		t, _ := time.Parse(time.RFC3339Nano, *releasedAt)
		job.ReleasedAt = &t
	}
	if releasedPrinterID != nil {
		job.ReleasedPrinterID = *releasedPrinterID
	}

	return &job, nil
}

// GetSecureJobByJobID retrieves a secure job by print job ID.
func (r *secureReleaseRepository) GetSecureJobByJobID(ctx context.Context, jobID string) (*SecurePrintJob, error) {
	query := `
		SELECT id, job_id, user_id, release_method, release_data,
		       status, expires_at, released_at, released_printer_id,
		       release_attempts, max_release_attempts, created_at, updated_at
		FROM secure_print_jobs
		WHERE job_id = $1::uuid
		LIMIT 1
	`

	var job SecurePrintJob
	var releaseDataJSON []byte
	var releasedAt, releasedPrinterID *string

	err := r.db.QueryRow(ctx, query, jobID).Scan(
		&job.ID, &job.JobID, &job.UserID, &job.ReleaseMethod, &releaseDataJSON,
		&job.Status, &job.ExpiresAt, &releasedAt, &releasedPrinterID,
		&job.ReleaseAttempts, &job.MaxReleaseAttempts, &job.CreatedAt, &job.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get secure job by job id: %w", err)
	}

	// Parse release data
	if releaseDataJSON != nil {
		json.Unmarshal(releaseDataJSON, &job.ReleaseData)
	}

	return &job, nil
}

// ListPendingJobs lists pending secure jobs for a user.
func (r *secureReleaseRepository) ListPendingJobs(ctx context.Context, userID string, limit, offset int) ([]*SecurePrintJob, int, error) {
	// Get total count
	var total int
	countQuery := `
		SELECT COUNT(*)
		FROM secure_print_jobs
		WHERE user_id = $1::uuid
		  AND status = 'pending'
		  AND expires_at > NOW()
	`
	_ = r.db.QueryRow(ctx, countQuery, userID).Scan(&total)

	// Get jobs
	query := `
		SELECT id, job_id, user_id, release_method, release_data,
		       status, expires_at, released_at, released_printer_id,
		       release_attempts, max_release_attempts, created_at, updated_at
		FROM secure_print_jobs
		WHERE user_id = $1::uuid
		  AND status = 'pending'
		  AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list pending jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*SecurePrintJob
	for rows.Next() {
		var job SecurePrintJob
		var releaseDataJSON []byte
		var releasedAt, releasedPrinterID *string

		if err := rows.Scan(
			&job.ID, &job.JobID, &job.UserID, &job.ReleaseMethod, &releaseDataJSON,
			&job.Status, &job.ExpiresAt, &releasedAt, &releasedPrinterID,
			&job.ReleaseAttempts, &job.MaxReleaseAttempts, &job.CreatedAt, &job.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}

		if releaseDataJSON != nil {
			json.Unmarshal(releaseDataJSON, &job.ReleaseData)
		}

		jobs = append(jobs, &job)
	}

	return jobs, total, nil
}

// ListReleasedJobs lists released jobs for a user.
func (r *secureReleaseRepository) ListReleasedJobs(ctx context.Context, userID string, limit, offset int) ([]*SecurePrintJob, int, error) {
	// Get total count
	var total int
	countQuery := `
		SELECT COUNT(*)
		FROM secure_print_jobs
		WHERE user_id = $1::uuid
		  AND status IN ('released', 'expired', 'cancelled')
	`
	_ = r.db.QueryRow(ctx, countQuery, userID).Scan(&total)

	// Get jobs
	query := `
		SELECT id, job_id, user_id, release_method, release_data,
		       status, expires_at, released_at, released_printer_id,
		       release_attempts, max_release_attempts, created_at, updated_at
		FROM secure_print_jobs
		WHERE user_id = $1::uuid
		  AND status IN ('released', 'expired', 'cancelled')
		ORDER BY released_at DESC NULLS LAST, created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list released jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*SecurePrintJob
	for rows.Next() {
		var job SecurePrintJob
		var releaseDataJSON []byte
		var releasedAt, releasedPrinterID *string

		if err := rows.Scan(
			&job.ID, &job.JobID, &job.UserID, &job.ReleaseMethod, &releaseDataJSON,
			&job.Status, &job.ExpiresAt, &releasedAt, &releasedPrinterID,
			&job.ReleaseAttempts, &job.MaxReleaseAttempts, &job.CreatedAt, &job.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}

		if releaseDataJSON != nil {
			json.Unmarshal(releaseDataJSON, &job.ReleaseData)
		}
		if releasedAt != nil {
			t, _ := time.Parse(time.RFC3339Nano, *releasedAt)
			job.ReleasedAt = &t
		}
		if releasedPrinterID != nil {
			job.ReleasedPrinterID = *releasedPrinterID
		}

		jobs = append(jobs, &job)
	}

	return jobs, total, nil
}

// AttemptRelease attempts to release a secure job.
func (r *secureReleaseRepository) AttemptRelease(ctx context.Context, secureJobID, method string, releaseData map[string]interface{}) (bool, error) {
	// Get the secure job
	job, err := r.GetSecureJob(ctx, secureJobID)
	if err != nil {
		return false, err
	}

	// Check if already processed
	if job.Status != "pending" {
		return false, fmt.Errorf("job already %s", job.Status)
	}

	// Check if expired
	if job.ExpiresAt.Before(time.Now()) {
		// Mark as expired
		_, _ = r.db.Exec(ctx, `
			UPDATE secure_print_jobs
			SET status = 'expired', updated_at = NOW()
			WHERE id = $1::uuid
		`, secureJobID)
		return false, fmt.Errorf("secure job has expired")
	}

	// Check release attempts
	if job.ReleaseAttempts >= job.MaxReleaseAttempts {
		// Mark as cancelled
		_, _ = r.db.Exec(ctx, `
			UPDATE secure_print_jobs
			SET status = 'cancelled', updated_at = NOW()
			WHERE id = $1::uuid
		`, secureJobID)
		return false, fmt.Errorf("maximum release attempts exceeded")
	}

	// Validate release method
	if job.ReleaseMethod != method {
		return false, fmt.Errorf("invalid release method")
	}

	// Validate release data (simplified for demonstration)
	valid := false
	if method == "pin" {
		// In production, do proper PIN validation with hashed comparison
		storedPin, _ := job.ReleaseData["pin"].(string)
		providedPin, _ := releaseData["pin"].(string)
		valid = storedPin == providedPin
	} else if method == "card" {
		// Validate card ID
		storedCard, _ := job.ReleaseData["card_id"].(string)
		providedCard, _ := releaseData["card_id"].(string)
		valid = storedCard == providedCard
	} else {
		// For other methods, assume validation passes
		valid = true
	}

	// Increment attempts
	attempts := job.ReleaseAttempts + 1

	// Get user identity for logging
	userIdentity := "unknown"
	if u, ok := releaseData["user_id"]; ok {
		userIdentity = fmt.Sprintf("%v", u)
	}

	if !valid {
		// Update attempts and log failure
		_, _ = r.db.Exec(ctx, `
			UPDATE secure_print_jobs
			SET release_attempts = $2, updated_at = NOW()
			WHERE id = $1::uuid
		`, secureJobID, attempts)

		// Log failed attempt
		r.logReleaseAttempt(ctx, secureJobID, method, userIdentity, false, "Invalid credentials", "")
		return false, nil
	}

	// Success - update job as released
	now := time.Now()
	printerID := ""
	if p, ok := releaseData["printer_id"]; ok {
		printerID = fmt.Sprintf("%v", p)
	}

	query := `
		UPDATE secure_print_jobs
		SET status = 'released',
		    released_at = $2,
		    released_printer_id = $3::uuid,
		    release_attempts = $4,
		    updated_at = $2
		WHERE id = $1::uuid
	`

	_, err = r.db.Exec(ctx, query, secureJobID, now, nullIfEmpty(printerID), attempts)
	if err != nil {
		return false, fmt.Errorf("update secure job: %w", err)
	}

	// Log successful attempt
	ipAddress := ""
	if ip, ok := releaseData["ip_address"]; ok {
		ipAddress = fmt.Sprintf("%v", ip)
	}
	r.logReleaseAttempt(ctx, secureJobID, method, userIdentity, true, "", ipAddress)

	return true, nil
}

// CancelSecureJob cancels a secure job.
func (r *secureReleaseRepository) CancelSecureJob(ctx context.Context, secureJobID string) error {
	query := `
		UPDATE secure_print_jobs
		SET status = 'cancelled', updated_at = NOW()
		WHERE id = $1::uuid AND status = 'pending'
	`

	cmdTag, err := r.db.Exec(ctx, query, secureJobID)
	if err != nil {
		return fmt.Errorf("cancel secure job: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("secure job not found or not in pending status")
	}

	return nil
}

// GetReleaseAttempts gets all release attempts for a secure job.
func (r *secureReleaseRepository) GetReleaseAttempts(ctx context.Context, secureJobID string) ([]*ReleaseAttempt, error) {
	query := `
		SELECT id, secure_job_id, attempted_at, attempted_method, attempted_by,
		       success, failure_reason, ip_address, printer_id
		FROM secure_release_attempts
		WHERE secure_job_id = $1::uuid
		ORDER BY attempted_at DESC
	`

	rows, err := r.db.Query(ctx, query, secureJobID)
	if err != nil {
		return nil, fmt.Errorf("get release attempts: %w", err)
	}
	defer rows.Close()

	var attempts []*ReleaseAttempt
	for rows.Next() {
		var attempt ReleaseAttempt
		if err := rows.Scan(
			&attempt.ID, &attempt.SecureJobID, &attempt.AttemptedAt,
			&attempt.AttemptedMethod, &attempt.AttemptedBy, &attempt.Success,
			&attempt.FailureReason, &attempt.IPAddress, &attempt.PrinterID,
		); err != nil {
			return nil, err
		}
		attempts = append(attempts, &attempt)
	}

	return attempts, nil
}

// ListReleaseStations lists all release stations for an organization.
func (r *secureReleaseRepository) ListReleaseStations(ctx context.Context, organizationID string) ([]*ReleaseStation, error) {
	query := `
		SELECT id, name, location, organization_id, supported_methods,
		       assigned_printers, is_active, last_heartbeat, created_at, updated_at
		FROM print_release_stations
		WHERE organization_id = $1::uuid
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query, organizationID)
	if err != nil {
		return nil, fmt.Errorf("list release stations: %w", err)
	}
	defer rows.Close()

	var stations []*ReleaseStation
	for rows.Next() {
		var station ReleaseStation
		var assignedPrinters types.JSONB
		var lastHeartbeat *time.Time

		if err := rows.Scan(
			&station.ID, &station.Name, &station.Location, &station.OrganizationID,
			&station.SupportedMethods, &assignedPrinters, &station.IsActive,
			&lastHeartbeat, &station.CreatedAt, &station.UpdatedAt,
		); err != nil {
			return nil, err
		}

		// Parse assigned printers from JSONB
		if len(assignedPrinters.Bytes) > 0 {
			json.Unmarshal(assignedPrinters.Bytes, &station.AssignedPrinters)
		}

		station.LastHeartbeat = lastHeartbeat
		stations = append(stations, &station)
	}

	return stations, nil
}

// CreateReleaseStation creates a new release station.
func (r *secureReleaseRepository) CreateReleaseStation(ctx context.Context, station *ReleaseStation) error {
	// Ensure tables exist
	r.initTables(ctx)

	// Serialize assigned printers
	var assignedPrintersJSON []byte
	if station.AssignedPrinters != nil {
		assignedPrintersJSON, _ = json.Marshal(station.AssignedPrinters)
	}

	query := `
		INSERT INTO print_release_stations (
			id, name, location, organization_id, supported_methods,
			assigned_printers, is_active, created_at, updated_at
		) VALUES (
			$1::uuid, $2, $3, $4::uuid, $5, $6::jsonb, $7, $8, $9
		)
	`

	_, err := r.db.Exec(ctx, query,
		station.ID, station.Name, station.Location, station.OrganizationID,
		station.SupportedMethods, assignedPrintersJSON, station.IsActive,
		station.CreatedAt, station.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("create release station: %w", err)
	}

	return nil
}

// UpdateReleaseStation updates an existing release station.
func (r *secureReleaseRepository) UpdateReleaseStation(ctx context.Context, station *ReleaseStation) error {
	// Serialize assigned printers
	var assignedPrintersJSON []byte
	if station.AssignedPrinters != nil {
		assignedPrintersJSON, _ = json.Marshal(station.AssignedPrinters)
	}

	query := `
		UPDATE print_release_stations
		SET name = $2, location = $3, supported_methods = $4,
		    assigned_printers = $5::jsonb, is_active = $6, updated_at = $7
		WHERE id = $1::uuid
	`

	_, err := r.db.Exec(ctx, query,
		station.ID, station.Name, station.Location, station.SupportedMethods,
		assignedPrintersJSON, station.IsActive, station.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("update release station: %w", err)
	}

	return nil
}

// DeleteReleaseStation deletes a release station.
func (r *secureReleaseRepository) DeleteReleaseStation(ctx context.Context, stationID string) error {
	query := `DELETE FROM print_release_stations WHERE id = $1::uuid`

	cmdTag, err := r.db.Exec(ctx, query, stationID)
	if err != nil {
		return fmt.Errorf("delete release station: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("release station not found")
	}

	return nil
}

// logReleaseAttempt logs a release attempt.
func (r *secureReleaseRepository) logReleaseAttempt(ctx context.Context, secureJobID, method, attemptedBy string, success bool, failureReason, ipAddress string) error {
	query := `
		INSERT INTO secure_release_attempts (
			id, secure_job_id, attempted_at, attempted_method, attempted_by,
			success, failure_reason, ip_address
		) VALUES (
			$1::uuid, $2::uuid, NOW(), $3, $4, $5, $6, $7
		)
	`

	id := uuid.New().String()
	_, err := r.db.Exec(ctx, query, id, secureJobID, method, attemptedBy, success, nullIfEmpty(failureReason), nullIfEmpty(ipAddress))

	return err
}

// initTables creates tables if they don't exist.
func (r *secureReleaseRepository) initTables(ctx context.Context) {
	// Tables are created by migrations, but this ensures they exist for standalone operation
	initQueries := []string{
		`CREATE TABLE IF NOT EXISTS secure_release_stations (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			name VARCHAR(255) NOT NULL,
			location VARCHAR(255),
			organization_id UUID NOT NULL,
			supported_methods VARCHAR(100)[] NOT NULL,
			assigned_printers JSONB,
			is_active BOOLEAN DEFAULT true,
			last_heartbeat TIMESTAMPTZ,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,
	}

	for _, q := range initQueries {
		r.db.Exec(ctx, q)
	}
}

// nullIfEmpty returns nil if string is empty.
func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
