// Package main provides tests for the policy engine.
package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/openprint/openprint/internal/testutil"
)

func TestEngineEvaluateRules(t *testing.T) {
	e := &Engine{}

	tests := []struct {
		name     string
		rules    []Rule
		ctx      *EvaluationContext
		expected bool
	}{
		{
			name: "Single matching rule - equals",
			rules: []Rule{
				{ID: "r1", Field: "document.page_count", Operator: OpEquals, Value: 10},
			},
			ctx: &EvaluationContext{
				PageCount: 10,
			},
			expected: true,
		},
		{
			name: "Single non-matching rule - equals",
			rules: []Rule{
				{ID: "r1", Field: "document.page_count", Operator: OpEquals, Value: 10},
			},
			ctx: &EvaluationContext{
				PageCount: 5,
			},
			expected: false,
		},
		{
			name: "Greater than - match",
			rules: []Rule{
				{ID: "r1", Field: "document.page_count", Operator: OpGreaterThan, Value: 5},
			},
			ctx: &EvaluationContext{
				PageCount: 10,
			},
			expected: true,
		},
		{
			name: "Less than - match",
			rules: []Rule{
				{ID: "r1", Field: "document.page_count", Operator: OpLessThan, Value: 100},
			},
			ctx: &EvaluationContext{
				PageCount: 10,
			},
			expected: true,
		},
		{
			name: "Contains - match",
			rules: []Rule{
				{ID: "r1", Field: "document.name", Operator: OpContains, Value: "confidential"},
			},
			ctx: &EvaluationContext{
				DocumentName: "confidential_report.pdf",
			},
			expected: true,
		},
		{
			name: "Not contains - match",
			rules: []Rule{
				{ID: "r1", Field: "document.name", Operator: OpNotContains, Value: "draft"},
			},
			ctx: &EvaluationContext{
				DocumentName: "final_report.pdf",
			},
			expected: true,
		},
		{
			name: "In - match",
			rules: []Rule{
				{ID: "r1", Field: "document.type", Operator: OpIn, Value: []string{"pdf", "docx"}},
			},
			ctx: &EvaluationContext{
				DocumentType: "pdf",
			},
			expected: true,
		},
		{
			name: "NotIn - match",
			rules: []Rule{
				{ID: "r1", Field: "document.type", Operator: OpNotIn, Value: []string{"pdf", "jpg"}},
			},
			ctx: &EvaluationContext{
				DocumentType: "docx",
			},
			expected: true,
		},
		{
			name: "Between - match",
			rules: []Rule{
				{ID: "r1", Field: "document.page_count", Operator: OpBetween, Value: []int{1, 10}},
			},
			ctx: &EvaluationContext{
				PageCount: 5,
			},
			expected: true,
		},
		{
			name: "Between - not match",
			rules: []Rule{
				{ID: "r1", Field: "document.page_count", Operator: OpBetween, Value: []int{1, 10}},
			},
			ctx: &EvaluationContext{
				PageCount: 15,
			},
			expected: false,
		},
		{
			name: "Always - match",
			rules: []Rule{
				{ID: "r1", Field: "document.page_count", Operator: OpAlways},
			},
			ctx:      &EvaluationContext{},
			expected: true,
		},
		{
			name: "Never - no match",
			rules: []Rule{
				{ID: "r1", Field: "document.page_count", Operator: OpNever},
			},
			ctx:      &EvaluationContext{},
			expected: false,
		},
		{
			name: "Multiple rules - all match",
			rules: []Rule{
				{ID: "r1", Field: "document.page_count", Operator: OpGreaterThan, Value: 5},
				{ID: "r2", Field: "document.page_count", Operator: OpLessThan, Value: 100},
			},
			ctx: &EvaluationContext{
				PageCount: 10,
			},
			expected: true,
		},
		{
			name: "Multiple rules - one fails",
			rules: []Rule{
				{ID: "r1", Field: "document.page_count", Operator: OpGreaterThan, Value: 5},
				{ID: "r2", Field: "document.page_count", Operator: OpLessThan, Value: 8},
			},
			ctx: &EvaluationContext{
				PageCount: 10,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, _ := e.evaluateRules(tt.rules, tt.ctx)
			if matched != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, matched)
			}
		})
	}
}

func TestEngineGetFieldValue(t *testing.T) {
	e := &Engine{}

	now := time.Now()
	ctx := &EvaluationContext{
		UserID:       "user123",
		UserEmail:    "user@example.com",
		UserGroups:   []string{"admin", "staff"},
		PrinterID:    "printer1",
		DocumentName: "document.pdf",
		DocumentType: "pdf",
		PageCount:    10,
		ColorMode:    "color",
		DuplexMode:   "duplex",
		Cost:         5.50,
		TimeOfDay:    now,
		DayOfWeek:    int(now.Weekday()),
		IPAddress:    "192.168.1.1",
		DeviceID:     "device1",
		Quota: &QuotaInfo{
			Limit:     1000,
			Used:      100,
			Remaining: 900,
		},
		Tags: []string{"urgent", "confidential"},
	}

	tests := []struct {
		field    string
		expected interface{}
	}{
		{"user.id", "user123"},
		{"user.email", "user@example.com"},
		{"user.groups", []string{"admin", "staff"}},
		{"printer.id", "printer1"},
		{"document.name", "document.pdf"},
		{"document.type", "pdf"},
		{"document.page_count", 10},
		{"document.color_mode", "color"},
		{"document.duplex_mode", "duplex"},
		{"document.cost", 5.50},
		{"time.hour", now.Hour()},
		{"time.day_of_week", int(now.Weekday())},
		{"quota.remaining", 900},
		{"quota.used", 100},
		{"quota.limit", 1000},
		{"ip.address", "192.168.1.1"},
		{"device.id", "device1"},
		{"document.tags", []string{"urgent", "confidential"}},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			result := e.getFieldValue(ctx, tt.field)
			if !compareValues(result, tt.expected) {
				t.Errorf("Field %s: expected %v (%T), got %v (%T)", tt.field, tt.expected, tt.expected, result, result)
			}
		})
	}
}

func TestEngineGetFieldValue_UnknownField(t *testing.T) {
	e := &Engine{}
	ctx := &EvaluationContext{}

	result := e.getFieldValue(ctx, "unknown.field")
	if result != nil {
		t.Errorf("Expected nil for unknown field, got %v", result)
	}
}

func TestEngineAppliesToScope(t *testing.T) {
	evalCtx := &EvaluationContext{
		UserID:       "user123",
		UserGroups:   []string{"staff"},
		PrinterID:    "printer1",
		DocumentType: "pdf",
	}

	tests := []struct {
		name     string
		policy   *Policy
		expected bool
	}{
		{
			name: "No scope restrictions",
			policy: &Policy{
				Scope: PolicyScope{},
			},
			expected: true,
		},
		{
			name: "User ID match",
			policy: &Policy{
				Scope: PolicyScope{
					UserIDs: []string{"user123", "user456"},
				},
			},
			expected: true,
		},
		{
			name: "User ID no match",
			policy: &Policy{
				Scope: PolicyScope{
					UserIDs: []string{"user456"},
				},
			},
			expected: false,
		},
		{
			name: "Group ID match",
			policy: &Policy{
				Scope: PolicyScope{
					GroupIDs: []string{"admin", "staff"},
				},
			},
			expected: true,
		},
		{
			name: "Group ID no match",
			policy: &Policy{
				Scope: PolicyScope{
					GroupIDs: []string{"admin", "manager"},
				},
			},
			expected: false,
		},
		{
			name: "Printer ID match",
			policy: &Policy{
				Scope: PolicyScope{
					PrinterIDs: []string{"printer1", "printer2"},
				},
			},
			expected: true,
		},
		{
			name: "Printer ID no match",
			policy: &Policy{
				Scope: PolicyScope{
					PrinterIDs: []string{"printer2"},
				},
			},
			expected: false,
		},
		{
			name: "Document type match",
			policy: &Policy{
				Scope: PolicyScope{
					DocumentTypes: []string{"pdf", "docx"},
				},
			},
			expected: true,
		},
		{
			name: "Document type no match",
			policy: &Policy{
				Scope: PolicyScope{
					DocumentTypes: []string{"jpg"},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Engine{}
			result := e.appliesToScope(tt.policy, evalCtx)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestRepositoryCreate(t *testing.T) {
	if testDB == nil || testDB.Pool == nil {
		t.Skip("Test database not available")
	}

	// Create a test organization and user first for foreign key constraint
	orgID, err := testutil.CreateTestOrganization(ctx, testDB.Pool)
	if err != nil {
		t.Fatalf("Failed to create test organization: %v", err)
	}

	userID, err := testutil.CreateTestUser(ctx, testDB.Pool, orgID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	repo := NewRepository(testDB.Pool)

	policy := &Policy{
		Name:        "Test Policy",
		Description: "Test description",
		Type:        PolicyTypeQuota,
		Status:      PolicyStatusDraft,
		Priority:    50,
		Rules: []Rule{
			{ID: "rule1", Field: "document.page_count", Operator: OpLessThan, Value: 100},
		},
		Actions: []PolicyActionConfig{
			{Type: ActionDeny, Order: 1},
		},
		Scope: PolicyScope{
			UserIDs: []string{"user1"},
		},
		CreatedBy: userID,
	}

	err = repo.Create(ctx, policy)
	if err != nil {
		t.Fatalf("Failed to create policy: %v", err)
	}

	if policy.ID == "" {
		t.Error("Expected policy ID to be set")
	}

	if policy.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}

	if policy.Version != 1 {
		t.Errorf("Expected version 1, got %d", policy.Version)
	}
}

func TestRepositoryGet(t *testing.T) {
	if testDB == nil || testDB.Pool == nil {
		t.Skip("Test database not available")
	}

	// Create a test organization and user first for foreign key constraint
	orgID, err := testutil.CreateTestOrganization(ctx, testDB.Pool)
	if err != nil {
		t.Fatalf("Failed to create test organization: %v", err)
	}

	userID, err := testutil.CreateTestUser(ctx, testDB.Pool, orgID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	repo := NewRepository(testDB.Pool)

	// Create a policy first
	policy := &Policy{
		Name:      "Get Test Policy",
		Type:      PolicyTypeQuota,
		Status:    PolicyStatusActive,
		Rules:     []Rule{{ID: "r1", Field: "document.page_count", Operator: OpLessThan, Value: 100}},
		Actions:   []PolicyActionConfig{{Type: ActionDeny}},
		Scope:     PolicyScope{},
		CreatedBy: userID,
	}

	if err := repo.Create(ctx, policy); err != nil {
		t.Fatalf("Failed to create policy: %v", err)
	}

	// Get the policy
	fetched, err := repo.Get(ctx, policy.ID)
	if err != nil {
		t.Fatalf("Failed to get policy: %v", err)
	}

	if fetched.Name != policy.Name {
		t.Errorf("Expected name %s, got %s", policy.Name, fetched.Name)
	}

	if fetched.Type != policy.Type {
		t.Errorf("Expected type %s, got %s", policy.Type, fetched.Type)
	}

	if len(fetched.Rules) != len(policy.Rules) {
		t.Errorf("Expected %d rules, got %d", len(policy.Rules), len(fetched.Rules))
	}
}

func TestRepositoryList(t *testing.T) {
	if testDB == nil || testDB.Pool == nil {
		t.Skip("Test database not available")
	}

	// Create a test organization and user first for foreign key constraint
	orgID, err := testutil.CreateTestOrganization(ctx, testDB.Pool)
	if err != nil {
		t.Fatalf("Failed to create test organization: %v", err)
	}

	userID, err := testutil.CreateTestUser(ctx, testDB.Pool, orgID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	repo := NewRepository(testDB.Pool)

	// Create multiple policies
	for i := 0; i < 3; i++ {
		policy := &Policy{
			CreatedBy: userID,
			Name:      fmt.Sprintf("Policy %d", i),
			Type:      PolicyTypeQuota,
			Status:    PolicyStatusActive,
			Rules:     []Rule{{ID: "r1", Field: "document.page_count", Operator: OpLessThan, Value: 100}},
			Actions:   []PolicyActionConfig{{Type: ActionDeny}},
			Scope:     PolicyScope{},
		}
		if err := repo.Create(ctx, policy); err != nil {
			t.Fatalf("Failed to create policy: %v", err)
		}
	}

	// List policies
	policies, total, err := repo.List(ctx, PolicyFilter{
		Status: PolicyStatusActive,
		Limit:  10,
		Offset: 0,
	})

	if err != nil {
		t.Fatalf("Failed to list policies: %v", err)
	}

	if total < 3 {
		t.Errorf("Expected at least 3 policies, got %d", total)
	}

	if len(policies) < 3 {
		t.Errorf("Expected at least 3 policies returned, got %d", len(policies))
	}
}

func TestRepositoryUpdate(t *testing.T) {
	if testDB == nil || testDB.Pool == nil {
		t.Skip("Test database not available")
	}

	// Create a test organization and user first for foreign key constraint
	orgID, err := testutil.CreateTestOrganization(ctx, testDB.Pool)
	if err != nil {
		t.Fatalf("Failed to create test organization: %v", err)
	}

	userID, err := testutil.CreateTestUser(ctx, testDB.Pool, orgID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create another user for modified_by
	modifiedByID, err := testutil.CreateTestUser(ctx, testDB.Pool, orgID)
	if err != nil {
		t.Fatalf("Failed to create test user for modified_by: %v", err)
	}

	repo := NewRepository(testDB.Pool)

	// Create a policy
	policy := &Policy{
		CreatedBy: userID,
		Name:      "Original Name",
		Type:      PolicyTypeQuota,
		Status:    PolicyStatusDraft,
		Rules:     []Rule{{ID: "r1", Field: "document.page_count", Operator: OpLessThan, Value: 100}},
		Actions:   []PolicyActionConfig{{Type: ActionDeny}},
		Scope:     PolicyScope{},
	}

	if err := repo.Create(ctx, policy); err != nil {
		t.Fatalf("Failed to create policy: %v", err)
	}

	originalVersion := policy.Version

	// Update the policy
	policy.Name = "Updated Name"
	policy.Status = PolicyStatusActive
	policy.ModifiedBy = modifiedByID

	if err := repo.Update(ctx, policy); err != nil {
		t.Fatalf("Failed to update policy: %v", err)
	}

	if policy.Version != originalVersion+1 {
		t.Errorf("Expected version %d, got %d", originalVersion+1, policy.Version)
	}

	// Fetch and verify
	fetched, _ := repo.Get(ctx, policy.ID)
	if fetched.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got %s", fetched.Name)
	}

	if fetched.Status != PolicyStatusActive {
		t.Errorf("Expected status active, got %s", fetched.Status)
	}
}

func TestRepositoryDelete(t *testing.T) {
	if testDB == nil || testDB.Pool == nil {
		t.Skip("Test database not available")
	}

	// Create a test organization and user first for foreign key constraint
	orgID, err := testutil.CreateTestOrganization(ctx, testDB.Pool)
	if err != nil {
		t.Fatalf("Failed to create test organization: %v", err)
	}

	userID, err := testutil.CreateTestUser(ctx, testDB.Pool, orgID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	repo := NewRepository(testDB.Pool)

	// Create a policy
	policy := &Policy{
		CreatedBy: userID,
		Name:      "To Delete",
		Type:      PolicyTypeQuota,
		Status:    PolicyStatusDraft,
		Rules:     []Rule{{ID: "r1", Field: "document.page_count", Operator: OpLessThan, Value: 100}},
		Actions:   []PolicyActionConfig{{Type: ActionDeny}},
		Scope:     PolicyScope{},
	}

	if err := repo.Create(ctx, policy); err != nil {
		t.Fatalf("Failed to create policy: %v", err)
	}

	// Delete the policy
	if err := repo.Delete(ctx, policy.ID); err != nil {
		t.Fatalf("Failed to delete policy: %v", err)
	}

	// Try to get it - should fail
	_, err = repo.Get(ctx, policy.ID)
	if err == nil {
		t.Error("Expected error when getting deleted policy")
	}
}

func TestRepositoryGetNotFound(t *testing.T) {
	if testDB == nil || testDB.Pool == nil {
		t.Skip("Test database not available")
	}

	repo := NewRepository(testDB.Pool)

	_, err := repo.Get(ctx, uuid.New().String())
	if err == nil {
		t.Error("Expected error for non-existent policy")
	}
}

func compareValues(a, b interface{}) bool {
	switch av := a.(type) {
	case int:
		bv, ok := b.(int)
		return ok && av == bv
	case int64:
		bv, ok := b.(int64)
		return ok && av == bv
	case float64:
		bv, ok := b.(float64)
		return ok && av == bv
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case []string:
		bv, ok := b.([]string)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if av[i] != bv[i] {
				return false
			}
		}
		return true
	default:
		return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
	}
}
