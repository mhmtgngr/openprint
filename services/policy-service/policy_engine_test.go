package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestEngine() *Engine {
	return NewEngine(Config{DB: nil})
}

// --- Engine unit tests ---

func TestEvaluateRule_OpAlways(t *testing.T) {
	engine := newTestEngine()
	rule := &Rule{ID: "r1", Field: "user.id", Operator: OpAlways}
	ctx := &EvaluationContext{UserID: "u1"}

	if !engine.evaluateRule(rule, ctx) {
		t.Error("OpAlways should always return true")
	}
}

func TestEvaluateRule_OpNever(t *testing.T) {
	engine := newTestEngine()
	rule := &Rule{ID: "r1", Field: "user.id", Operator: OpNever}
	ctx := &EvaluationContext{UserID: "u1"}

	if engine.evaluateRule(rule, ctx) {
		t.Error("OpNever should always return false")
	}
}

func TestEvaluateRule_OpEquals(t *testing.T) {
	engine := newTestEngine()
	rule := &Rule{ID: "r1", Field: "user.id", Operator: OpEquals, Value: "user-123"}
	ctx := &EvaluationContext{UserID: "user-123"}

	if !engine.evaluateRule(rule, ctx) {
		t.Error("OpEquals should match when values are equal")
	}

	ctx.UserID = "user-456"
	if engine.evaluateRule(rule, ctx) {
		t.Error("OpEquals should not match when values differ")
	}
}

func TestEvaluateRule_OpNotEquals(t *testing.T) {
	engine := newTestEngine()
	rule := &Rule{ID: "r1", Field: "user.id", Operator: OpNotEquals, Value: "user-123"}

	ctx := &EvaluationContext{UserID: "user-456"}
	if !engine.evaluateRule(rule, ctx) {
		t.Error("OpNotEquals should match when values differ")
	}

	ctx.UserID = "user-123"
	if engine.evaluateRule(rule, ctx) {
		t.Error("OpNotEquals should not match when values are equal")
	}
}

func TestEvaluateRule_OpGreaterThan(t *testing.T) {
	engine := newTestEngine()
	rule := &Rule{ID: "r1", Field: "document.page_count", Operator: OpGreaterThan, Value: float64(10)}

	ctx := &EvaluationContext{PageCount: 20}
	if !engine.evaluateRule(rule, ctx) {
		t.Error("OpGreaterThan should match when field > value")
	}

	ctx.PageCount = 5
	if engine.evaluateRule(rule, ctx) {
		t.Error("OpGreaterThan should not match when field < value")
	}

	ctx.PageCount = 10
	if engine.evaluateRule(rule, ctx) {
		t.Error("OpGreaterThan should not match when field == value")
	}
}

func TestEvaluateRule_OpLessThan(t *testing.T) {
	engine := newTestEngine()
	rule := &Rule{ID: "r1", Field: "document.page_count", Operator: OpLessThan, Value: float64(10)}

	ctx := &EvaluationContext{PageCount: 5}
	if !engine.evaluateRule(rule, ctx) {
		t.Error("OpLessThan should match when field < value")
	}

	ctx.PageCount = 15
	if engine.evaluateRule(rule, ctx) {
		t.Error("OpLessThan should not match when field > value")
	}
}

func TestEvaluateRule_OpContains(t *testing.T) {
	engine := newTestEngine()
	rule := &Rule{ID: "r1", Field: "document.name", Operator: OpContains, Value: "report"}

	ctx := &EvaluationContext{DocumentName: "quarterly_report_2024.pdf"}
	if !engine.evaluateRule(rule, ctx) {
		t.Error("OpContains should match when field contains value")
	}

	ctx.DocumentName = "invoice_2024.pdf"
	if engine.evaluateRule(rule, ctx) {
		t.Error("OpContains should not match when field does not contain value")
	}
}

func TestEvaluateRule_OpNotContains(t *testing.T) {
	engine := newTestEngine()
	rule := &Rule{ID: "r1", Field: "document.name", Operator: OpNotContains, Value: "confidential"}

	ctx := &EvaluationContext{DocumentName: "public_memo.pdf"}
	if !engine.evaluateRule(rule, ctx) {
		t.Error("OpNotContains should match when field does not contain value")
	}

	ctx.DocumentName = "confidential_report.pdf"
	if engine.evaluateRule(rule, ctx) {
		t.Error("OpNotContains should not match when field contains value")
	}
}

func TestEvaluateRule_OpIn(t *testing.T) {
	engine := newTestEngine()
	rule := &Rule{
		ID:       "r1",
		Field:    "document.type",
		Operator: OpIn,
		Value:    []interface{}{"pdf", "docx", "xlsx"},
	}

	ctx := &EvaluationContext{DocumentType: "pdf"}
	if !engine.evaluateRule(rule, ctx) {
		t.Error("OpIn should match when value is in list")
	}

	ctx.DocumentType = "png"
	if engine.evaluateRule(rule, ctx) {
		t.Error("OpIn should not match when value is not in list")
	}
}

func TestEvaluateRule_OpNotIn(t *testing.T) {
	engine := newTestEngine()
	rule := &Rule{
		ID:       "r1",
		Field:    "document.type",
		Operator: OpNotIn,
		Value:    []interface{}{"exe", "bat", "sh"},
	}

	ctx := &EvaluationContext{DocumentType: "pdf"}
	if !engine.evaluateRule(rule, ctx) {
		t.Error("OpNotIn should match when value is not in list")
	}

	ctx.DocumentType = "exe"
	if engine.evaluateRule(rule, ctx) {
		t.Error("OpNotIn should not match when value is in list")
	}
}

func TestEvaluateRule_OpBetween(t *testing.T) {
	engine := newTestEngine()
	rule := &Rule{
		ID:       "r1",
		Field:    "document.page_count",
		Operator: OpBetween,
		Value:    []interface{}{float64(5), float64(50)},
	}

	ctx := &EvaluationContext{PageCount: 25}
	if !engine.evaluateRule(rule, ctx) {
		t.Error("OpBetween should match when value is within range")
	}

	ctx.PageCount = 3
	if engine.evaluateRule(rule, ctx) {
		t.Error("OpBetween should not match when value is below range")
	}

	ctx.PageCount = 100
	if engine.evaluateRule(rule, ctx) {
		t.Error("OpBetween should not match when value is above range")
	}

	// Boundary values
	ctx.PageCount = 5
	if !engine.evaluateRule(rule, ctx) {
		t.Error("OpBetween should match when value equals lower bound")
	}
	ctx.PageCount = 50
	if !engine.evaluateRule(rule, ctx) {
		t.Error("OpBetween should match when value equals upper bound")
	}
}

// --- evaluateRules tests ---

func TestEvaluateRules_EmptyRules(t *testing.T) {
	engine := newTestEngine()
	ctx := &EvaluationContext{UserID: "u1"}

	matched, ruleMatches := engine.evaluateRules([]Rule{}, ctx)
	if !matched {
		t.Error("empty rules should match (allow-by-default)")
	}
	if len(ruleMatches) != 0 {
		t.Error("empty rules should produce empty rule matches map")
	}
}

func TestEvaluateRules_AllAND_AllMatch(t *testing.T) {
	engine := newTestEngine()
	rules := []Rule{
		{ID: "r1", Field: "document.type", Operator: OpEquals, Value: "pdf", LogicalOp: "AND"},
		{ID: "r2", Field: "document.page_count", Operator: OpLessThan, Value: float64(100), LogicalOp: "AND"},
	}
	ctx := &EvaluationContext{DocumentType: "pdf", PageCount: 50}

	matched, ruleMatches := engine.evaluateRules(rules, ctx)
	if !matched {
		t.Error("all AND rules matching should return true")
	}
	if !ruleMatches["r1"] || !ruleMatches["r2"] {
		t.Error("individual rule matches should be true")
	}
}

func TestEvaluateRules_AllAND_OneFails(t *testing.T) {
	engine := newTestEngine()
	rules := []Rule{
		{ID: "r1", Field: "document.type", Operator: OpEquals, Value: "pdf", LogicalOp: "AND"},
		{ID: "r2", Field: "document.page_count", Operator: OpLessThan, Value: float64(100), LogicalOp: "AND"},
	}
	ctx := &EvaluationContext{DocumentType: "docx", PageCount: 50}

	matched, ruleMatches := engine.evaluateRules(rules, ctx)
	if matched {
		t.Error("AND rules with one failing should return false")
	}
	if ruleMatches["r1"] {
		t.Error("first rule should not match")
	}
}

func TestEvaluateRules_OR_OneMatches(t *testing.T) {
	engine := newTestEngine()
	rules := []Rule{
		{ID: "r1", Field: "document.type", Operator: OpEquals, Value: "pdf", LogicalOp: "OR"},
		{ID: "r2", Field: "document.type", Operator: OpEquals, Value: "docx", LogicalOp: "OR"},
	}
	ctx := &EvaluationContext{DocumentType: "docx"}

	matched, _ := engine.evaluateRules(rules, ctx)
	if !matched {
		t.Error("OR rules with one matching should return true")
	}
}

func TestEvaluateRules_DefaultAND(t *testing.T) {
	engine := newTestEngine()
	// No LogicalOp set defaults to AND behavior
	rules := []Rule{
		{ID: "r1", Field: "document.type", Operator: OpEquals, Value: "pdf"},
		{ID: "r2", Field: "document.page_count", Operator: OpLessThan, Value: float64(100)},
	}

	ctx := &EvaluationContext{DocumentType: "pdf", PageCount: 50}
	matched, _ := engine.evaluateRules(rules, ctx)
	if !matched {
		t.Error("default AND: all matching should return true")
	}

	ctx.DocumentType = "docx"
	matched, _ = engine.evaluateRules(rules, ctx)
	if matched {
		t.Error("default AND: one failing should return false")
	}
}

// --- getFieldValue tests ---

func TestGetFieldValue(t *testing.T) {
	engine := newTestEngine()
	now := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC) // Saturday, 14:00
	ctx := &EvaluationContext{
		UserID:       "user-1",
		UserEmail:    "user@example.com",
		UserGroups:   []string{"admin", "devops"},
		PrinterID:    "printer-1",
		DocumentName: "test.pdf",
		DocumentType: "pdf",
		PageCount:    42,
		ColorMode:    "color",
		DuplexMode:   "duplex",
		Cost:         1.50,
		TimeOfDay:    now,
		DayOfWeek:    6,
		IPAddress:    "192.168.1.100",
		DeviceID:     "device-1",
		Tags:         []string{"urgent", "internal"},
		Quota: &QuotaInfo{
			Limit:     100,
			Used:      60,
			Remaining: 40,
		},
	}

	tests := []struct {
		field    string
		expected interface{}
	}{
		{"user.id", "user-1"},
		{"user.email", "user@example.com"},
		{"printer.id", "printer-1"},
		{"document.name", "test.pdf"},
		{"document.type", "pdf"},
		{"document.page_count", 42},
		{"document.color_mode", "color"},
		{"document.duplex_mode", "duplex"},
		{"document.cost", 1.50},
		{"time.hour", 14},
		{"time.day_of_week", 6},
		{"quota.remaining", 40},
		{"quota.used", 60},
		{"quota.limit", 100},
		{"ip.address", "192.168.1.100"},
		{"device.id", "device-1"},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			got := engine.getFieldValue(ctx, tt.field)
			if got != tt.expected {
				t.Errorf("getFieldValue(%q) = %v, want %v", tt.field, got, tt.expected)
			}
		})
	}
}

func TestGetFieldValue_QuotaNil(t *testing.T) {
	engine := newTestEngine()
	ctx := &EvaluationContext{Quota: nil}

	for _, field := range []string{"quota.remaining", "quota.used", "quota.limit"} {
		got := engine.getFieldValue(ctx, field)
		if got != 0 {
			t.Errorf("getFieldValue(%q) with nil quota = %v, want 0", field, got)
		}
	}
}

func TestGetFieldValue_Unknown(t *testing.T) {
	engine := newTestEngine()
	ctx := &EvaluationContext{}

	got := engine.getFieldValue(ctx, "nonexistent.field")
	if got != nil {
		t.Errorf("getFieldValue for unknown field should return nil, got %v", got)
	}
}

// --- appliesToScope tests ---

func TestAppliesToScope_EmptyScope(t *testing.T) {
	engine := newTestEngine()
	policy := &Policy{Scope: PolicyScope{}}
	ctx := &EvaluationContext{UserID: "u1"}

	if !engine.appliesToScope(policy, ctx) {
		t.Error("empty scope should apply to all contexts")
	}
}

func TestAppliesToScope_UserIDMatch(t *testing.T) {
	engine := newTestEngine()
	policy := &Policy{Scope: PolicyScope{UserIDs: []string{"u1", "u2"}}}

	ctx := &EvaluationContext{UserID: "u1"}
	if !engine.appliesToScope(policy, ctx) {
		t.Error("scope with matching user ID should apply")
	}

	ctx.UserID = "u3"
	if engine.appliesToScope(policy, ctx) {
		t.Error("scope with non-matching user ID should not apply")
	}
}

func TestAppliesToScope_GroupMatch(t *testing.T) {
	engine := newTestEngine()
	policy := &Policy{Scope: PolicyScope{GroupIDs: []string{"admin"}}}

	ctx := &EvaluationContext{UserGroups: []string{"admin", "users"}}
	if !engine.appliesToScope(policy, ctx) {
		t.Error("scope with matching group should apply")
	}

	ctx.UserGroups = []string{"users"}
	if engine.appliesToScope(policy, ctx) {
		t.Error("scope with non-matching group should not apply")
	}
}

func TestAppliesToScope_PrinterIDMatch(t *testing.T) {
	engine := newTestEngine()
	policy := &Policy{Scope: PolicyScope{PrinterIDs: []string{"p1", "p2"}}}

	ctx := &EvaluationContext{PrinterID: "p1"}
	if !engine.appliesToScope(policy, ctx) {
		t.Error("scope with matching printer should apply")
	}

	ctx.PrinterID = "p3"
	if engine.appliesToScope(policy, ctx) {
		t.Error("scope with non-matching printer should not apply")
	}
}

func TestAppliesToScope_DocumentTypeMatch(t *testing.T) {
	engine := newTestEngine()
	policy := &Policy{Scope: PolicyScope{DocumentTypes: []string{"pdf", "docx"}}}

	ctx := &EvaluationContext{DocumentType: "pdf"}
	if !engine.appliesToScope(policy, ctx) {
		t.Error("scope with matching document type should apply")
	}

	ctx.DocumentType = "png"
	if engine.appliesToScope(policy, ctx) {
		t.Error("scope with non-matching document type should not apply")
	}
}

// --- Comparison helper tests ---

func TestToFloat64(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected float64
		ok       bool
	}{
		{int(42), 42.0, true},
		{int64(42), 42.0, true},
		{float32(3.14), float64(float32(3.14)), true},
		{float64(3.14), 3.14, true},
		{"not a number", 0, false},
		{nil, 0, false},
	}

	for _, tt := range tests {
		got, ok := toFloat64(tt.input)
		if ok != tt.ok {
			t.Errorf("toFloat64(%v): ok = %v, want %v", tt.input, ok, tt.ok)
		}
		if ok && got != tt.expected {
			t.Errorf("toFloat64(%v) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

// --- Handler tests ---

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("health handler returned %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["status"] != "healthy" {
		t.Errorf("status = %q, want %q", resp["status"], "healthy")
	}
	if resp["service"] != "policy-service" {
		t.Errorf("service = %q, want %q", resp["service"], "policy-service")
	}
}

func TestHealthHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	w := httptest.NewRecorder()

	healthHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("health handler returned %d for POST, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestTestPolicyHandler_MatchingPolicy(t *testing.T) {
	engine := newTestEngine()
	handler := testPolicyHandler(engine)

	body := `{
		"policy": {
			"id": "test-1",
			"name": "Block large jobs",
			"rules": [
				{"id": "r1", "field": "document.page_count", "operator": "greater_than", "value": 100}
			],
			"actions": [
				{"type": "deny", "parameters": {"reason": "too many pages"}, "order": 1}
			]
		},
		"test_context": {
			"user_id": "u1",
			"page_count": 200
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("test handler returned %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if matched, ok := resp["matched"].(bool); !ok || !matched {
		t.Error("expected matched=true for page_count 200 > 100")
	}

	actions, ok := resp["actions"].([]interface{})
	if !ok || len(actions) == 0 {
		t.Error("expected actions to be populated when matched")
	}
}

func TestTestPolicyHandler_NonMatchingPolicy(t *testing.T) {
	engine := newTestEngine()
	handler := testPolicyHandler(engine)

	body := `{
		"policy": {
			"id": "test-1",
			"name": "Block large jobs",
			"rules": [
				{"id": "r1", "field": "document.page_count", "operator": "greater_than", "value": 100}
			],
			"actions": [
				{"type": "deny", "parameters": {"reason": "too many pages"}, "order": 1}
			]
		},
		"test_context": {
			"user_id": "u1",
			"page_count": 10
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("test handler returned %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if matched, ok := resp["matched"].(bool); !ok || matched {
		t.Error("expected matched=false for page_count 10 < 100")
	}

	// Actions should be null/nil when not matched
	if resp["actions"] != nil {
		t.Error("expected nil actions when not matched")
	}
}

func TestTestPolicyHandler_MissingPolicy(t *testing.T) {
	engine := newTestEngine()
	handler := testPolicyHandler(engine)

	body := `{"test_context": {"user_id": "u1"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when policy is missing, got %d", w.Code)
	}
}

func TestTestPolicyHandler_MissingContext(t *testing.T) {
	engine := newTestEngine()
	handler := testPolicyHandler(engine)

	body := `{"policy": {"id": "p1", "name": "test"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when test_context is missing, got %d", w.Code)
	}
}

func TestTestPolicyHandler_MethodNotAllowed(t *testing.T) {
	engine := newTestEngine()
	handler := testPolicyHandler(engine)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET, got %d", w.Code)
	}
}

func TestValidateRulesHandler_ValidRules(t *testing.T) {
	engine := newTestEngine()
	handler := validateRulesHandler(engine)

	body := `{
		"rules": [
			{"id": "r1", "field": "document.type", "operator": "equals", "value": "pdf"},
			{"id": "r2", "field": "document.page_count", "operator": "greater_than", "value": 100}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rules/validate", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if valid, ok := resp["valid"].(bool); !ok || !valid {
		t.Error("expected valid=true for valid rules")
	}
}

func TestValidateRulesHandler_InvalidRules(t *testing.T) {
	engine := newTestEngine()
	handler := validateRulesHandler(engine)

	body := `{
		"rules": [
			{"id": "", "field": "", "operator": ""}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rules/validate", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if valid, ok := resp["valid"].(bool); !ok || valid {
		t.Error("expected valid=false for rules with missing fields")
	}

	errors, ok := resp["errors"].([]interface{})
	if !ok || len(errors) != 3 {
		t.Errorf("expected 3 validation errors, got %v", errors)
	}
}

func TestTestPolicyHandler_MultipleRulesAND(t *testing.T) {
	engine := newTestEngine()
	handler := testPolicyHandler(engine)

	body := `{
		"policy": {
			"id": "test-multi",
			"name": "Color duplex policy",
			"rules": [
				{"id": "r1", "field": "document.color_mode", "operator": "equals", "value": "color", "logical_op": "AND"},
				{"id": "r2", "field": "document.page_count", "operator": "greater_than", "value": 50, "logical_op": "AND"}
			],
			"actions": [
				{"type": "require_auth", "parameters": {"message": "Large color jobs require approval"}, "order": 1}
			]
		},
		"test_context": {
			"user_id": "u1",
			"color_mode": "color",
			"page_count": 100
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler(w, req)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if matched, ok := resp["matched"].(bool); !ok || !matched {
		t.Error("expected matched=true when all AND rules match")
	}

	// Now test with non-matching context (bw mode)
	body2 := `{
		"policy": {
			"id": "test-multi",
			"name": "Color duplex policy",
			"rules": [
				{"id": "r1", "field": "document.color_mode", "operator": "equals", "value": "color", "logical_op": "AND"},
				{"id": "r2", "field": "document.page_count", "operator": "greater_than", "value": 50, "logical_op": "AND"}
			],
			"actions": [
				{"type": "require_auth", "parameters": {}, "order": 1}
			]
		},
		"test_context": {
			"user_id": "u1",
			"color_mode": "bw",
			"page_count": 100
		}
	}`

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/test", strings.NewReader(body2))
	w2 := httptest.NewRecorder()
	handler(w2, req2)

	var resp2 map[string]interface{}
	json.NewDecoder(w2.Body).Decode(&resp2)

	if matched, ok := resp2["matched"].(bool); !ok || matched {
		t.Error("expected matched=false when one AND rule fails")
	}
}

func TestPoliciesHandler_MethodNotAllowed(t *testing.T) {
	engine := newTestEngine()
	handler := policiesHandler(engine)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/policies", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for DELETE on policies collection, got %d", w.Code)
	}
}

func TestPoliciesHandler_PostValidation(t *testing.T) {
	engine := newTestEngine()
	handler := policiesHandler(engine)

	// Missing name
	body := `{"type": "quota"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/policies", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing name, got %d", w.Code)
	}

	// Missing type
	body2 := `{"name": "Test Policy"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/policies", strings.NewReader(body2))
	w2 := httptest.NewRecorder()
	handler(w2, req2)

	if w2.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing type, got %d", w2.Code)
	}
}

func TestPoliciesHandler_PostInvalidBody(t *testing.T) {
	engine := newTestEngine()
	handler := policiesHandler(engine)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/policies", strings.NewReader("not json"))
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", w.Code)
	}
}

func TestPolicyByIDHandler_MissingID(t *testing.T) {
	engine := newTestEngine()
	handler := policyByIDHandler(engine)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/policies/", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing ID, got %d", w.Code)
	}
}

func TestPolicyByIDHandler_MethodNotAllowed(t *testing.T) {
	engine := newTestEngine()
	handler := policyByIDHandler(engine)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/policies/some-id", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for PATCH, got %d", w.Code)
	}
}

func TestExtractIDFromPath(t *testing.T) {
	tests := []struct {
		path     string
		prefix   string
		expected string
	}{
		{"/api/v1/policies/abc-123", "/api/v1/policies/", "abc-123"},
		{"/api/v1/policies/", "/api/v1/policies/", ""},
		{"/api/v1/policies", "/api/v1/policies/", ""},
	}

	for _, tt := range tests {
		got := extractIDFromPath(tt.path, tt.prefix)
		if got != tt.expected {
			t.Errorf("extractIDFromPath(%q, %q) = %q, want %q", tt.path, tt.prefix, got, tt.expected)
		}
	}
}

func TestParseIntParam(t *testing.T) {
	tests := []struct {
		input    string
		expected int
		wantErr  bool
	}{
		{"42", 42, false},
		{"0", 0, false},
		{"-1", -1, false},
		{"abc", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		got, err := parseIntParam(tt.input)
		if tt.wantErr && err == nil {
			t.Errorf("parseIntParam(%q) expected error", tt.input)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("parseIntParam(%q) unexpected error: %v", tt.input, err)
		}
		if !tt.wantErr && got != tt.expected {
			t.Errorf("parseIntParam(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

// --- AddEvaluationHook / GetPolicy tests ---

func TestGetPolicy(t *testing.T) {
	engine := newTestEngine()

	// Manually add a policy to the in-memory cache
	engine.mu.Lock()
	engine.policies["p1"] = &Policy{ID: "p1", Name: "Test Policy"}
	engine.mu.Unlock()

	got, ok := engine.GetPolicy("p1")
	if !ok {
		t.Fatal("expected to find policy p1")
	}
	if got.Name != "Test Policy" {
		t.Errorf("policy name = %q, want %q", got.Name, "Test Policy")
	}

	_, ok = engine.GetPolicy("nonexistent")
	if ok {
		t.Error("expected not to find nonexistent policy")
	}
}

func TestNewEngine(t *testing.T) {
	engine := NewEngine(Config{DB: nil})
	if engine == nil {
		t.Fatal("NewEngine returned nil")
	}
	if engine.policies == nil {
		t.Error("policies map should be initialized")
	}
}
