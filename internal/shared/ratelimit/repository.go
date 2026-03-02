package ratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for rate limiting.
type Repository struct {
	db    *pgxpool.Pool
	redis *RedisClient
}

// NewRepository creates a new repository.
func NewRepository(redis *RedisClient) *Repository {
	return &Repository{
		redis: redis,
	}
}

// SetDB sets the database connection pool.
func (r *Repository) SetDB(db *pgxpool.Pool) {
	r.db = db
}

// Policy CRUD operations

// CreatePolicy creates a new rate limit policy.
func (r *Repository) CreatePolicy(ctx context.Context, policy *Policy) error {
	if r.db == nil {
		return fmt.Errorf("database not configured")
	}

	query := `
		INSERT INTO rate_limit_policies (
			id, name, description, priority, scope, identifier,
			methods, path_pattern, limit, window, burst_limit, burst_duration,
			enable_queue, max_queue_size, circuit_breaker_threshold,
			circuit_breaker_timeout, severity, action, throttle_rate, is_active
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17, $18, $19, $20
		)
	`

	methodsJSON, _ := json.Marshal(policy.Methods)

	_, err := r.db.Exec(ctx, query,
		policy.ID, policy.Name, policy.Description, policy.Priority,
		policy.Scope, policy.Identifier, methodsJSON, policy.PathPattern,
		policy.Limit, int64(policy.Window.Seconds()), policy.BurstLimit,
		int64(policy.BurstDuration.Seconds()), policy.EnableQueue,
		policy.MaxQueueSize, policy.CircuitBreakerThreshold,
		int64(policy.CircuitBreakerTimeout.Seconds()), policy.Severity,
		policy.Action, policy.ThrottleRate, policy.IsActive,
	)

	if err != nil {
		return fmt.Errorf("failed to create policy: %w", err)
	}

	// Invalidate policy cache
	return nil
}

// GetPolicy retrieves a policy by ID.
func (r *Repository) GetPolicy(ctx context.Context, id string) (*Policy, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not configured")
	}

	query := `
		SELECT id, name, description, priority, scope, identifier,
		       methods, path_pattern, limit, window, burst_limit, burst_duration,
		       enable_queue, max_queue_size, circuit_breaker_threshold,
		       circuit_breaker_timeout, severity, action, throttle_rate,
		       is_active, created_at, updated_at
		FROM rate_limit_policies
		WHERE id = $1
	`

	var policy Policy
	var methodsJSON []byte
	var windowSec, burstDurationSec, cbTimeoutSec int64

	err := r.db.QueryRow(ctx, query, id).Scan(
		&policy.ID, &policy.Name, &policy.Description, &policy.Priority,
		&policy.Scope, &policy.Identifier, &methodsJSON, &policy.PathPattern,
		&policy.Limit, &windowSec, &policy.BurstLimit, &burstDurationSec,
		&policy.EnableQueue, &policy.MaxQueueSize, &policy.CircuitBreakerThreshold,
		&cbTimeoutSec, &policy.Severity, &policy.Action, &policy.ThrottleRate,
		&policy.IsActive, &policy.CreatedAt, &policy.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("policy not found")
	}
	if err != nil {
		return nil, err
	}

	policy.Window = time.Duration(windowSec) * time.Second
	if burstDurationSec > 0 {
		policy.BurstDuration = time.Duration(burstDurationSec) * time.Second
	}
	if cbTimeoutSec > 0 {
		policy.CircuitBreakerTimeout = time.Duration(cbTimeoutSec) * time.Second
	}

	_ = json.Unmarshal(methodsJSON, &policy.Methods)

	return &policy, nil
}

// ListPolicies lists all policies with optional filtering.
func (r *Repository) ListPolicies(ctx context.Context, filter *PolicyFilter) ([]*Policy, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not configured")
	}

	query := `
		SELECT id, name, description, priority, scope, identifier,
		       methods, path_pattern, limit, window, burst_limit, burst_duration,
		       enable_queue, max_queue_size, circuit_breaker_threshold,
		       circuit_breaker_timeout, severity, action, throttle_rate,
		       is_active, created_at, updated_at
		FROM rate_limit_policies
		WHERE 1=1
	`

	args := []interface{}{}
	argIdx := 1

	if filter != nil {
		if filter.Scope != "" {
			query += fmt.Sprintf(" AND scope = $%d", argIdx)
			args = append(args, filter.Scope)
			argIdx++
		}
		if filter.IsActive != nil {
			query += fmt.Sprintf(" AND is_active = $%d", argIdx)
			args = append(args, *filter.IsActive)
			argIdx++
		}
	}

	query += " ORDER BY priority DESC, created_at DESC"

	if filter != nil && filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, filter.Limit)
		argIdx++
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []*Policy
	for rows.Next() {
		var policy Policy
		var methodsJSON []byte
		var windowSec, burstDurationSec, cbTimeoutSec int64

		err := rows.Scan(
			&policy.ID, &policy.Name, &policy.Description, &policy.Priority,
			&policy.Scope, &policy.Identifier, &methodsJSON, &policy.PathPattern,
			&policy.Limit, &windowSec, &policy.BurstLimit, &burstDurationSec,
			&policy.EnableQueue, &policy.MaxQueueSize, &policy.CircuitBreakerThreshold,
			&cbTimeoutSec, &policy.Severity, &policy.Action, &policy.ThrottleRate,
			&policy.IsActive, &policy.CreatedAt, &policy.UpdatedAt,
		)

		if err != nil {
			continue
		}

		policy.Window = time.Duration(windowSec) * time.Second
		if burstDurationSec > 0 {
			policy.BurstDuration = time.Duration(burstDurationSec) * time.Second
		}
		if cbTimeoutSec > 0 {
			policy.CircuitBreakerTimeout = time.Duration(cbTimeoutSec) * time.Second
		}

		_ = json.Unmarshal(methodsJSON, &policy.Methods)

		policies = append(policies, &policy)
	}

	return policies, nil
}

// GetActivePolicies retrieves all active policies.
func (r *Repository) GetActivePolicies(ctx context.Context) ([]*Policy, error) {
	return r.ListPolicies(ctx, &PolicyFilter{IsActive: boolPtr(true)})
}

// UpdatePolicy updates an existing policy.
func (r *Repository) UpdatePolicy(ctx context.Context, policy *Policy) error {
	if r.db == nil {
		return fmt.Errorf("database not configured")
	}

	query := `
		UPDATE rate_limit_policies
		SET name = $2, description = $3, priority = $4, scope = $5,
		    identifier = $6, methods = $7, path_pattern = $8, limit = $9,
		    window = $10, burst_limit = $11, burst_duration = $12,
		    enable_queue = $13, max_queue_size = $14,
		    circuit_breaker_threshold = $15, circuit_breaker_timeout = $16,
		    severity = $17, action = $18, throttle_rate = $19,
		    is_active = $20, updated_at = NOW()
		WHERE id = $1
	`

	methodsJSON, _ := json.Marshal(policy.Methods)

	_, err := r.db.Exec(ctx, query,
		policy.ID, policy.Name, policy.Description, policy.Priority,
		policy.Scope, policy.Identifier, methodsJSON, policy.PathPattern,
		policy.Limit, int64(policy.Window.Seconds()), policy.BurstLimit,
		int64(policy.BurstDuration.Seconds()), policy.EnableQueue,
		policy.MaxQueueSize, policy.CircuitBreakerThreshold,
		int64(policy.CircuitBreakerTimeout.Seconds()), policy.Severity,
		policy.Action, policy.ThrottleRate, policy.IsActive,
	)

	return err
}

// DeletePolicy deletes a policy.
func (r *Repository) DeletePolicy(ctx context.Context, id string) error {
	if r.db == nil {
		return fmt.Errorf("database not configured")
	}

	query := `DELETE FROM rate_limit_policies WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// Violation CRUD operations

// CreateViolation creates a violation record.
func (r *Repository) CreateViolation(ctx context.Context, violation *Violation) error {
	if r.db == nil {
		return fmt.Errorf("database not configured")
	}

	query := `
		INSERT INTO rate_limit_violations (
			id, policy_id, policy_name, identifier, identifier_type,
			path, method, current, limit, severity, occurred_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.Exec(ctx, query,
		violation.ID, violation.PolicyID, violation.PolicyName,
		violation.Identifier, violation.IdentifierType, violation.Path,
		violation.Method, violation.Current, violation.Limit,
		violation.Severity, violation.OccurredAt,
	)

	return err
}

// GetViolation retrieves a violation by ID.
func (r *Repository) GetViolation(ctx context.Context, id string) (*Violation, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not configured")
	}

	query := `
		SELECT id, policy_id, policy_name, identifier, identifier_type,
		       path, method, current, limit, severity, occurred_at
		FROM rate_limit_violations
		WHERE id = $1
	`

	var violation Violation
	err := r.db.QueryRow(ctx, query, id).Scan(
		&violation.ID, &violation.PolicyID, &violation.PolicyName,
		&violation.Identifier, &violation.IdentifierType, &violation.Path,
		&violation.Method, &violation.Current, &violation.Limit,
		&violation.Severity, &violation.OccurredAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("violation not found")
	}
	if err != nil {
		return nil, err
	}

	return &violation, nil
}

// ListViolations lists violations with filtering.
func (r *Repository) ListViolations(ctx context.Context, filter *ViolationFilter) ([]*Violation, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not configured")
	}

	query := `
		SELECT id, policy_id, policy_name, identifier, identifier_type,
		       path, method, current, limit, severity, occurred_at
		FROM rate_limit_violations
		WHERE 1=1
	`

	args := []interface{}{}
	argIdx := 1

	if filter != nil {
		if filter.Identifier != "" {
			query += fmt.Sprintf(" AND identifier = $%d", argIdx)
			args = append(args, filter.Identifier)
			argIdx++
		}
		if filter.PolicyID != "" {
			query += fmt.Sprintf(" AND policy_id = $%d", argIdx)
			args = append(args, filter.PolicyID)
			argIdx++
		}
		if filter.Severity != "" {
			query += fmt.Sprintf(" AND severity = $%d", argIdx)
			args = append(args, filter.Severity)
			argIdx++
		}
		if !filter.Since.IsZero() {
			query += fmt.Sprintf(" AND occurred_at >= $%d", argIdx)
			args = append(args, filter.Since)
			argIdx++
		}
	}

	query += " ORDER BY occurred_at DESC"

	if filter != nil && filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, filter.Limit)
		argIdx++
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var violations []*Violation
	for rows.Next() {
		var v Violation
		err := rows.Scan(
			&v.ID, &v.PolicyID, &v.PolicyName, &v.Identifier,
			&v.IdentifierType, &v.Path, &v.Method, &v.Current,
			&v.Limit, &v.Severity, &v.OccurredAt,
		)

		if err != nil {
			continue
		}

		violations = append(violations, &v)
	}

	return violations, nil
}

// TrustedClient CRUD operations

// CreateTrustedClient creates a new trusted client.
func (r *Repository) CreateTrustedClient(ctx context.Context, client *TrustedClient) error {
	if r.db == nil {
		return fmt.Errorf("database not configured")
	}

	ipWhitelistJSON, _ := json.Marshal(client.IPWhitelist)

	query := `
		INSERT INTO trusted_clients (
			id, name, api_key, ip_whitelist, description, is_active
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.Exec(ctx, query,
		client.ID, client.Name, client.APIKey, ipWhitelistJSON,
		client.Description, client.IsActive,
	)

	return err
}

// GetTrustedClient retrieves a trusted client by ID.
func (r *Repository) GetTrustedClient(ctx context.Context, id string) (*TrustedClient, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not configured")
	}

	query := `
		SELECT id, name, api_key, ip_whitelist, description,
		       is_active, created_at, updated_at, last_used_at
		FROM trusted_clients
		WHERE id = $1
	`

	var client TrustedClient
	var ipWhitelistJSON []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&client.ID, &client.Name, &client.APIKey, &ipWhitelistJSON,
		&client.Description, &client.IsActive, &client.CreatedAt,
		&client.UpdatedAt, &client.LastUsedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("trusted client not found")
	}
	if err != nil {
		return nil, err
	}

	_ = json.Unmarshal(ipWhitelistJSON, &client.IPWhitelist)

	return &client, nil
}

// ListTrustedClients lists all trusted clients.
func (r *Repository) ListTrustedClients(ctx context.Context) ([]*TrustedClient, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not configured")
	}

	query := `
		SELECT id, name, api_key, ip_whitelist, description,
		       is_active, created_at, updated_at, last_used_at
		FROM trusted_clients
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clients []*TrustedClient
	for rows.Next() {
		var client TrustedClient
		var ipWhitelistJSON []byte

		err := rows.Scan(
			&client.ID, &client.Name, &client.APIKey, &ipWhitelistJSON,
			&client.Description, &client.IsActive, &client.CreatedAt,
			&client.UpdatedAt, &client.LastUsedAt,
		)

		if err != nil {
			continue
		}

		_ = json.Unmarshal(ipWhitelistJSON, &client.IPWhitelist)
		clients = append(clients, &client)
	}

	return clients, nil
}

// UpdateTrustedClient updates a trusted client.
func (r *Repository) UpdateTrustedClient(ctx context.Context, client *TrustedClient) error {
	if r.db == nil {
		return fmt.Errorf("database not configured")
	}

	ipWhitelistJSON, _ := json.Marshal(client.IPWhitelist)

	query := `
		UPDATE trusted_clients
		SET name = $2, api_key = $3, ip_whitelist = $4,
		    description = $5, is_active = $6, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		client.ID, client.Name, client.APIKey, ipWhitelistJSON,
		client.Description, client.IsActive,
	)

	return err
}

// DeleteTrustedClient deletes a trusted client.
func (r *Repository) DeleteTrustedClient(ctx context.Context, id string) error {
	if r.db == nil {
		return fmt.Errorf("database not configured")
	}

	query := `DELETE FROM trusted_clients WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// Helper types

// PolicyFilter filters policy listings.
type PolicyFilter struct {
	Scope    string
	IsActive *bool
	Limit    int
}

// ViolationFilter filters violation listings.
type ViolationFilter struct {
	Identifier string
	PolicyID   string
	Severity   string
	Since      time.Time
	Limit      int
}

// boolPtr returns a pointer to a bool.
func boolPtr(b bool) *bool {
	return &b
}

// ListTrustedClients is a helper function for the bypass manager.
func ListTrustedClients(ctx context.Context, repo *Repository) ([]*TrustedClient, error) {
	return repo.ListTrustedClients(ctx)
}
