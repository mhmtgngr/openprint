// Package main is the entry point for the OpenPrint Policy Service.
// This service handles print policy engine with rule evaluation and enforcement.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
)

// PolicyType represents the type of print policy.
type PolicyType string

const (
	PolicyTypeQuota       PolicyType = "quota"
	PolicyTypeAccess      PolicyType = "access"
	PolicyTypeContent     PolicyType = "content"
	PolicyTypeRouting     PolicyType = "routing"
	PolicyTypeWatermark   PolicyType = "watermark"
	PolicyTypeRetention   PolicyType = "retention"
	PolicyTypeCostCenter  PolicyType = "cost_center"
)

// PolicyStatus represents the status of a policy.
type PolicyStatus string

const (
	PolicyStatusActive    PolicyStatus = "active"
	PolicyStatusInactive  PolicyStatus = "inactive"
	PolicyStatusDraft     PolicyStatus = "draft"
	PolicyStatusArchived  PolicyStatus = "archived"
)

// PolicyAction represents the action to take when a policy is triggered.
type PolicyAction string

const (
	ActionAllow       PolicyAction = "allow"
	ActionDeny        PolicyAction = "deny"
	ActionRequireAuth PolicyAction = "require_auth"
	ActionRouteTo     PolicyAction = "route_to"
	ActionWatermark   PolicyAction = "watermark"
	ActionLog         PolicyAction = "log"
	ActionNotify      PolicyAction = "notify"
)

// Operator represents comparison operators for rule conditions.
type Operator string

const (
	OpEquals         Operator = "equals"
	OpNotEquals      Operator = "not_equals"
	OpGreaterThan    Operator = "greater_than"
	OpLessThan       Operator = "less_than"
	OpContains       Operator = "contains"
	OpNotContains    Operator = "not_contains"
	OpMatches        Operator = "matches"
	OpIn             Operator = "in"
	OpNotIn          Operator = "not_in"
	OpBetween        Operator = "between"
	OpAlways         Operator = "always"
	OpNever          Operator = "never"
)

// Policy represents a print policy with rules and actions.
type Policy struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Type           PolicyType             `json:"type"`
	Status         PolicyStatus           `json:"status"`
	Priority       int                    `json:"priority"`
	Rules          []Rule                 `json:"rules"`
	Actions        []PolicyActionConfig   `json:"actions"`
	Scope          PolicyScope            `json:"scope"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	CreatedBy      string                 `json:"created_by"`
	ModifiedBy     string                 `json:"modified_by"`
	Version        int                    `json:"version"`
	EvaluatedCount int                    `json:"evaluated_count"`
	TriggeredCount int                    `json:"triggered_count"`
}

// Rule represents a single rule condition.
type Rule struct {
	ID         string          `json:"id"`
	Field      string          `json:"field"`
	Operator   Operator        `json:"operator"`
	Value      interface{}     `json:"value"`
	LogicalOp  string          `json:"logical_op,omitempty"` // AND, OR
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// PolicyActionConfig represents an action configuration.
type PolicyActionConfig struct {
	Type        PolicyAction          `json:"type"`
	Parameters  map[string]interface{} `json:"parameters"`
	Order       int                   `json:"order"`
}

// PolicyScope defines the scope where the policy applies.
type PolicyScope struct {
	OrganizationID string   `json:"organization_id,omitempty"`
	UserIDs        []string `json:"user_ids,omitempty"`
	GroupIDs       []string `json:"group_ids,omitempty"`
	PrinterIDs     []string `json:"printer_ids,omitempty"`
	DocumentTypes  []string `json:"document_types,omitempty"`
}

// EvaluationContext provides data for policy evaluation.
type EvaluationContext struct {
	UserID        string                 `json:"user_id"`
	UserName      string                 `json:"user_name"`
	UserEmail     string                 `json:"user_email"`
	UserGroups    []string               `json:"user_groups"`
	PrinterID     string                 `json:"printer_id"`
	DocumentName  string                 `json:"document_name"`
	DocumentType  string                 `json:"document_type"`
	PageCount     int                    `json:"page_count"`
	ColorMode     string                 `json:"color_mode"`
	DuplexMode    string                 `json:"duplex_mode"`
	Cost          float64                `json:"cost"`
	TimeOfDay     time.Time              `json:"time_of_day"`
	DayOfWeek     int                    `json:"day_of_week"`
	Metadata      map[string]interface{} `json:"metadata"`
	IPAddress     string                 `json:"ip_address"`
	DeviceID      string                 `json:"device_id"`
	Quota         *QuotaInfo             `json:"quota,omitempty"`
	Tags          []string               `json:"tags"`
}

// QuotaInfo represents quota usage information.
type QuotaInfo struct {
	Limit      int       `json:"limit"`
	Used       int       `json:"used"`
	Remaining  int       `json:"remaining"`
	Period     string    `json:"period"`
	ResetsAt   time.Time `json:"resets_at"`
}

// EvaluationResult represents the result of policy evaluation.
type EvaluationResult struct {
	PolicyID       string                 `json:"policy_id"`
	PolicyName     string                 `json:"policy_name"`
	Matched        bool                   `json:"matched"`
	Action         PolicyAction           `json:"action"`
	Parameters     map[string]interface{} `json:"parameters"`
	Message        string                 `json:"message"`
	RuleMatches    map[string]bool        `json:"rule_matches"`
	EvaluatedAt    time.Time              `json:"evaluated_at"`
	ProcessingTime time.Duration          `json:"processing_time"`
}

// Engine evaluates print policies against context.
type Engine struct {
	db        *pgxpool.Pool
	policies  map[string]*Policy
	mu        sync.RWMutex
	evalHooks []EvaluationHook
}

// EvaluationHook is a function called during policy evaluation.
type EvaluationHook func(ctx context.Context, policy *Policy, evalCtx *EvaluationContext, result *EvaluationResult)

// Config holds engine configuration.
type Config struct {
	DB *pgxpool.Pool
}

// NewEngine creates a new policy engine.
func NewEngine(cfg Config) *Engine {
	return &Engine{
		db:       cfg.DB,
		policies: make(map[string]*Policy),
	}
}

// Repository handles policy data operations.
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new policy repository.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Create creates a new policy.
func (r *Repository) Create(ctx context.Context, policy *Policy) error {
	policy.ID = uuid.New().String()
	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()
	policy.Version = 1

	rulesJSON, _ := json.Marshal(policy.Rules)
	actionsJSON, _ := json.Marshal(policy.Actions)
	scopeJSON, _ := json.Marshal(policy.Scope)

	query := `
		INSERT INTO print_policies
		(id, name, description, type, status, priority, rules, actions, scope,
		 created_at, updated_at, created_by, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := r.db.Exec(ctx, query,
		policy.ID, policy.Name, policy.Description, policy.Type, policy.Status,
		policy.Priority, rulesJSON, actionsJSON, scopeJSON,
		policy.CreatedAt, policy.UpdatedAt, policy.CreatedBy, policy.Version,
	)

	if err != nil {
		return fmt.Errorf("create policy: %w", err)
	}

	return nil
}

// Get retrieves a policy by ID.
func (r *Repository) Get(ctx context.Context, id string) (*Policy, error) {
	query := `
		SELECT id, name, description, type, status, priority, rules, actions, scope,
		       created_at, updated_at, created_by, modified_by, version,
		       evaluated_count, triggered_count
		FROM print_policies
		WHERE id = $1 AND status != 'archived'
	`

	var policy Policy
	var rulesJSON, actionsJSON, scopeJSON []byte
	var modifiedBy *string

	err := r.db.QueryRow(ctx, query, id).Scan(
		&policy.ID, &policy.Name, &policy.Description, &policy.Type, &policy.Status,
		&policy.Priority, &rulesJSON, &actionsJSON, &scopeJSON, &policy.CreatedAt,
		&policy.UpdatedAt, &policy.CreatedBy, &modifiedBy, &policy.Version,
		&policy.EvaluatedCount, &policy.TriggeredCount,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("get policy: %w", err)
	}

	if modifiedBy != nil {
		policy.ModifiedBy = *modifiedBy
	}

	json.Unmarshal(rulesJSON, &policy.Rules)
	json.Unmarshal(actionsJSON, &policy.Actions)
	json.Unmarshal(scopeJSON, &policy.Scope)

	return &policy, nil
}

// List retrieves policies with filtering.
func (r *Repository) List(ctx context.Context, filter PolicyFilter) ([]*Policy, int, error) {
	whereClause := "WHERE status != 'archived'"
	args := []interface{}{}
	argIdx := 1

	if filter.Type != "" {
		whereClause += fmt.Sprintf(" AND type = $%d", argIdx)
		args = append(args, filter.Type)
		argIdx++
	}

	if filter.Status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, filter.Status)
		argIdx++
	}

	if filter.OrganizationID != "" {
		whereClause += fmt.Sprintf(" AND scope->>'organization_id' = $%d", argIdx)
		args = append(args, filter.OrganizationID)
		argIdx++
	}

	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) FROM print_policies " + whereClause
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count policies: %w", err)
	}

	// Get policies
	query := `
		SELECT id, name, description, type, status, priority, rules, actions, scope,
		       created_at, updated_at, created_by, modified_by, version,
		       evaluated_count, triggered_count
		FROM print_policies
	` + whereClause + `
		ORDER BY priority DESC, created_at DESC
		LIMIT $` + fmt.Sprintf("%d", argIdx) + ` OFFSET $` + fmt.Sprintf("%d", argIdx+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list policies: %w", err)
	}
	defer rows.Close()

	var policies []*Policy
	for rows.Next() {
		var policy Policy
		var rulesJSON, actionsJSON, scopeJSON []byte
		var modifiedBy *string

		if err := rows.Scan(
			&policy.ID, &policy.Name, &policy.Description, &policy.Type, &policy.Status,
			&policy.Priority, &rulesJSON, &actionsJSON, &scopeJSON, &policy.CreatedAt,
			&policy.UpdatedAt, &policy.CreatedBy, &modifiedBy, &policy.Version,
			&policy.EvaluatedCount, &policy.TriggeredCount,
		); err != nil {
			return nil, 0, err
		}

		if modifiedBy != nil {
			policy.ModifiedBy = *modifiedBy
		}

		json.Unmarshal(rulesJSON, &policy.Rules)
		json.Unmarshal(actionsJSON, &policy.Actions)
		json.Unmarshal(scopeJSON, &policy.Scope)

		policies = append(policies, &policy)
	}

	return policies, total, rows.Err()
}

// PolicyFilter represents filters for listing policies.
type PolicyFilter struct {
	Type           PolicyType
	Status         PolicyStatus
	OrganizationID string
	Limit          int
	Offset         int
}

// Update updates a policy.
func (r *Repository) Update(ctx context.Context, policy *Policy) error {
	policy.UpdatedAt = time.Now()
	policy.Version++

	rulesJSON, _ := json.Marshal(policy.Rules)
	actionsJSON, _ := json.Marshal(policy.Actions)
	scopeJSON, _ := json.Marshal(policy.Scope)

	query := `
		UPDATE print_policies
		SET name = $2, description = $3, type = $4, status = $5, priority = $6,
		    rules = $7, actions = $8, scope = $9, updated_at = $10,
		    modified_by = $11, version = $12
		WHERE id = $1
	`

	cmdTag, err := r.db.Exec(ctx, query,
		policy.ID, policy.Name, policy.Description, policy.Type, policy.Status,
		policy.Priority, rulesJSON, actionsJSON, scopeJSON, policy.UpdatedAt,
		policy.ModifiedBy, policy.Version,
	)

	if err != nil {
		return fmt.Errorf("update policy: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return apperrors.ErrNotFound
	}

	return nil
}

// Delete soft deletes a policy by archiving it.
func (r *Repository) Delete(ctx context.Context, id string) error {
	query := `UPDATE print_policies SET status = 'archived', updated_at = $2 WHERE id = $1`

	cmdTag, err := r.db.Exec(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("delete policy: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return apperrors.ErrNotFound
	}

	return nil
}

// Evaluate evaluates policies against the given context.
func (e *Engine) Evaluate(ctx context.Context, evalCtx *EvaluationContext) ([]*EvaluationResult, error) {
	startTime := time.Now()

	repo := NewRepository(e.db)
	policies, _, err := repo.List(ctx, PolicyFilter{
		Status: PolicyStatusActive,
		Limit:  1000,
		Offset: 0,
	})
	if err != nil {
		return nil, fmt.Errorf("list policies: %w", err)
	}

	var results []*EvaluationResult

	for _, policy := range policies {
		// Check if policy applies to this context
		if !e.appliesToScope(policy, evalCtx) {
			continue
		}

		// Increment evaluation count
		e.incrementEvaluationCount(ctx, policy.ID)

		// Evaluate rules
		matched, ruleMatches := e.evaluateRules(policy.Rules, evalCtx)

		result := &EvaluationResult{
			PolicyID:    policy.ID,
			PolicyName:  policy.Name,
			Matched:     matched,
			RuleMatches: ruleMatches,
			EvaluatedAt: time.Now(),
		}

		if matched && len(policy.Actions) > 0 {
			// Get the first action
			action := policy.Actions[0]
			result.Action = action.Type
			result.Parameters = action.Parameters
			result.Message = fmt.Sprintf("Policy '%s' matched", policy.Name)

			// Increment triggered count
			e.incrementTriggeredCount(ctx, policy.ID)
		} else if !matched {
			result.Action = ActionAllow
			result.Message = "No policy matched"
		}

		result.ProcessingTime = time.Since(startTime)

		// Call evaluation hooks
		for _, hook := range e.evalHooks {
			hook(ctx, policy, evalCtx, result)
		}

		results = append(results, result)
	}

	return results, nil
}

// appliesToScope checks if the policy applies to the given context.
func (e *Engine) appliesToScope(policy *Policy, evalCtx *EvaluationContext) bool {
	scope := policy.Scope

	// Check organization
	if scope.OrganizationID != "" && !e.userInOrganization(evalCtx.UserID, scope.OrganizationID) {
		return false
	}

	// Check user IDs
	if len(scope.UserIDs) > 0 && !e.contains(scope.UserIDs, evalCtx.UserID) {
		return false
	}

	// Check group membership
	if len(scope.GroupIDs) > 0 && !e.userInGroups(evalCtx.UserGroups, scope.GroupIDs) {
		return false
	}

	// Check printer IDs
	if len(scope.PrinterIDs) > 0 && !e.contains(scope.PrinterIDs, evalCtx.PrinterID) {
		return false
	}

	// Check document types
	if len(scope.DocumentTypes) > 0 && !e.contains(scope.DocumentTypes, evalCtx.DocumentType) {
		return false
	}

	return true
}

// evaluateRules evaluates all rules in a policy.
func (e *Engine) evaluateRules(rules []Rule, evalCtx *EvaluationContext) (bool, map[string]bool) {
	ruleMatches := make(map[string]bool)

	if len(rules) == 0 {
		return true, ruleMatches
	}

	// Evaluate each rule
	for i, rule := range rules {
		ruleMatches[rule.ID] = e.evaluateRule(&rule, evalCtx)

		// Short-circuit on AND
		if rule.LogicalOp == "AND" && !ruleMatches[rule.ID] {
			return false, ruleMatches
		}

		// Short-circuit on OR
		if rule.LogicalOp == "OR" && ruleMatches[rule.ID] {
			// Check remaining rules are OR
			allOR := true
			for j := i + 1; j < len(rules); j++ {
				if rules[j].LogicalOp != "OR" {
					allOR = false
					break
				}
			}
			if allOR {
				return true, ruleMatches
			}
		}
	}

	// Final result: all rules must match (AND behavior by default)
	allMatched := true
	for _, matched := range ruleMatches {
		if !matched {
			allMatched = false
			break
		}
	}

	return allMatched, ruleMatches
}

// evaluateRule evaluates a single rule.
func (e *Engine) evaluateRule(rule *Rule, evalCtx *EvaluationContext) bool {
	fieldValue := e.getFieldValue(evalCtx, rule.Field)
	ruleValue := rule.Value

	switch rule.Operator {
	case OpAlways:
		return true
	case OpNever:
		return false
	case OpEquals:
		return e.compareEquals(fieldValue, ruleValue)
	case OpNotEquals:
		return !e.compareEquals(fieldValue, ruleValue)
	case OpGreaterThan:
		return e.compareGreater(fieldValue, ruleValue)
	case OpLessThan:
		return e.compareLess(fieldValue, ruleValue)
	case OpContains:
		return e.compareContains(fieldValue, ruleValue)
	case OpNotContains:
		return !e.compareContains(fieldValue, ruleValue)
	case OpIn:
		return e.compareIn(fieldValue, ruleValue)
	case OpNotIn:
		return !e.compareIn(fieldValue, ruleValue)
	case OpBetween:
		return e.compareBetween(fieldValue, ruleValue)
	default:
		return false
	}
}

// getFieldValue extracts a field value from the evaluation context.
func (e *Engine) getFieldValue(evalCtx *EvaluationContext, field string) interface{} {
	switch field {
	case "user.id":
		return evalCtx.UserID
	case "user.email":
		return evalCtx.UserEmail
	case "user.groups":
		return evalCtx.UserGroups
	case "printer.id":
		return evalCtx.PrinterID
	case "document.name":
		return evalCtx.DocumentName
	case "document.type":
		return evalCtx.DocumentType
	case "document.page_count":
		return evalCtx.PageCount
	case "document.color_mode":
		return evalCtx.ColorMode
	case "document.duplex_mode":
		return evalCtx.DuplexMode
	case "document.cost":
		return evalCtx.Cost
	case "time.hour":
		return evalCtx.TimeOfDay.Hour()
	case "time.day_of_week":
		return evalCtx.DayOfWeek
	case "quota.remaining":
		if evalCtx.Quota != nil {
			return evalCtx.Quota.Remaining
		}
		return 0
	case "quota.used":
		if evalCtx.Quota != nil {
			return evalCtx.Quota.Used
		}
		return 0
	case "quota.limit":
		if evalCtx.Quota != nil {
			return evalCtx.Quota.Limit
		}
		return 0
	case "ip.address":
		return evalCtx.IPAddress
	case "device.id":
		return evalCtx.DeviceID
	case "document.tags":
		return evalCtx.Tags
	default:
		return nil
	}
}

// Comparison helper functions

func (e *Engine) compareEquals(a, b interface{}) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func (e *Engine) compareGreater(a, b interface{}) bool {
	af, aOk := toFloat64(a)
	bf, bOk := toFloat64(b)
	if aOk && bOk {
		return af > bf
	}
	return false
}

func (e *Engine) compareLess(a, b interface{}) bool {
	af, aOk := toFloat64(a)
	bf, bOk := toFloat64(b)
	if aOk && bOk {
		return af < bf
	}
	return false
}

func (e *Engine) compareContains(a, b interface{}) bool {
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)
	return hasPrefixOrSuffix(aStr, bStr)
}

// compareIn checks if value a is in slice b using type-agnostic JSON comparison.
// This handles strongly-typed slices ([]string, []int, etc.) from JSON unmarshaling.
func (e *Engine) compareIn(a, b interface{}) bool {
	// Convert b to a JSON slice representation
	bSlice, err := toSlice(b)
	if err != nil {
		return false
	}

	// Convert a to JSON for consistent comparison
	aJSON, err := json.Marshal(a)
	if err != nil {
		return false
	}
	aStr := string(aJSON)

	for _, item := range bSlice {
		itemJSON, err := json.Marshal(item)
		if err != nil {
			// Fallback to string comparison
			if fmt.Sprintf("%v", item) == fmt.Sprintf("%v", a) {
				return true
			}
			continue
		}
		if string(itemJSON) == aStr {
			return true
		}
	}
	return false
}

// compareBetween checks if value a is between min and max (inclusive) using type-agnostic JSON comparison.
func (e *Engine) compareBetween(a, b interface{}) bool {
	// Convert b to a slice
	bSlice, err := toSlice(b)
	if err != nil || len(bSlice) != 2 {
		return false
	}

	af, aOk := toFloat64(a)
	min, minOk := toFloat64(bSlice[0])
	max, maxOk := toFloat64(bSlice[1])

	if aOk && minOk && maxOk {
		return af >= min && af <= max
	}
	return false
}

// toSlice converts an interface{} to []interface{} for comparison.
// It handles strongly-typed slices from JSON unmarshaling by converting via JSON.
func toSlice(v interface{}) ([]interface{}, error) {
	// If it's already []interface{}, return it
	if slice, ok := v.([]interface{}); ok {
		return slice, nil
	}

	// Try to convert via JSON marshaling for type-agnostic handling
	jsonData, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	var result []interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case float32:
		return float64(val), true
	case float64:
		return val, true
	default:
		return 0, false
	}
}

func hasPrefixOrSuffix(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && (s[len(s)-len(substr):] == substr || middleContains(s, substr))
}

func middleContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (e *Engine) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (e *Engine) userInOrganization(userID, orgID string) bool {
	// Query database for user's organization
	ctx := context.Background()
	var userOrgID string
	err := e.db.QueryRow(ctx, "SELECT organization_id FROM users WHERE id = $1", userID).Scan(&userOrgID)
	return err == nil && userOrgID == orgID
}

func (e *Engine) userInGroups(userGroups, requiredGroups []string) bool {
	for _, required := range requiredGroups {
		for _, userGroup := range userGroups {
			if userGroup == required {
				return true
			}
		}
	}
	return false
}

func (e *Engine) incrementEvaluationCount(ctx context.Context, policyID string) {
	_, _ = e.db.Exec(ctx, "UPDATE print_policies SET evaluated_count = evaluated_count + 1 WHERE id = $1", policyID)
}

func (e *Engine) incrementTriggeredCount(ctx context.Context, policyID string) {
	_, _ = e.db.Exec(ctx, "UPDATE print_policies SET triggered_count = triggered_count + 1 WHERE id = $1", policyID)
}

// AddEvaluationHook adds a hook to be called during evaluation.
func (e *Engine) AddEvaluationHook(hook EvaluationHook) {
	e.evalHooks = append(e.evalHooks, hook)
}

// LoadPolicies loads all active policies into memory.
func (e *Engine) LoadPolicies(ctx context.Context) error {
	repo := NewRepository(e.db)
	policies, _, err := repo.List(ctx, PolicyFilter{
		Status: PolicyStatusActive,
		Limit:  1000,
		Offset: 0,
	})
	if err != nil {
		return err
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.policies = make(map[string]*Policy)
	for _, policy := range policies {
		e.policies[policy.ID] = policy
	}

	return nil
}

// GetPolicy retrieves a policy from memory.
func (e *Engine) GetPolicy(id string) (*Policy, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	policy, ok := e.policies[id]
	return policy, ok
}
