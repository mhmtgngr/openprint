// Package handler provides tests for organization service HTTP handlers.
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/services/organization-service/repository"
)

// mockOrgRepo is a mock repository for testing organization handlers.
type mockOrgRepo struct {
	orgs        map[string]*repository.Organization
	permissions map[string][]*repository.Permission
}

func newMockOrgRepo() *mockOrgRepo {
	return &mockOrgRepo{
		orgs:        make(map[string]*repository.Organization),
		permissions: make(map[string][]*repository.Permission),
	}
}

func (m *mockOrgRepo) Create(ctx context.Context, org *repository.Organization) error {
	if org.ID == "" {
		org.ID = uuid.New().String()
	}
	now := time.Now()
	org.CreatedAt = now
	org.UpdatedAt = now
	m.orgs[org.ID] = org
	return nil
}

func (m *mockOrgRepo) FindByID(ctx context.Context, id string) (*repository.Organization, error) {
	org, ok := m.orgs[id]
	if !ok {
		return nil, apperrors.ErrNotFound
	}
	return org, nil
}

func (m *mockOrgRepo) FindBySlug(ctx context.Context, slug string) (*repository.Organization, error) {
	for _, org := range m.orgs {
		if org.Slug == slug {
			return org, nil
		}
	}
	return nil, apperrors.ErrNotFound
}

func (m *mockOrgRepo) List(ctx context.Context) ([]*repository.Organization, error) {
	var orgs []*repository.Organization
	for _, org := range m.orgs {
		orgs = append(orgs, org)
	}
	return orgs, nil
}

func (m *mockOrgRepo) Update(ctx context.Context, org *repository.Organization) error {
	if _, ok := m.orgs[org.ID]; !ok {
		return apperrors.ErrNotFound
	}
	org.UpdatedAt = time.Now()
	m.orgs[org.ID] = org
	return nil
}

func (m *mockOrgRepo) Delete(ctx context.Context, id string) error {
	if _, ok := m.orgs[id]; !ok {
		return apperrors.ErrNotFound
	}
	delete(m.orgs, id)
	return nil
}

func (m *mockOrgRepo) AddMember(ctx context.Context, orgID, userID string) error {
	return nil
}

func (m *mockOrgRepo) RemoveMember(ctx context.Context, orgID, userID string) error {
	return nil
}

func (m *mockOrgRepo) ListMembers(ctx context.Context, orgID string) ([]string, error) {
	return nil, nil
}

func (m *mockOrgRepo) AddPermission(ctx context.Context, perm *repository.Permission) error {
	perm.GrantedAt = time.Now()
	if perm.ID == "" {
		perm.ID = uuid.New().String()
	}
	m.permissions[perm.OrganizationID] = append(m.permissions[perm.OrganizationID], perm)
	return nil
}

func (m *mockOrgRepo) RemovePermission(ctx context.Context, orgID, userID string) error {
	perms, ok := m.permissions[orgID]
	if !ok {
		return apperrors.ErrNotFound
	}
	for i, p := range perms {
		if p.UserID == userID {
			m.permissions[orgID] = append(perms[:i], perms[i+1:]...)
			return nil
		}
	}
	return apperrors.ErrNotFound
}

func (m *mockOrgRepo) ListPermissions(ctx context.Context, orgID string) ([]*repository.Permission, error) {
	return m.permissions[orgID], nil
}

func (m *mockOrgRepo) GetUserPermission(ctx context.Context, orgID, userID string) (string, error) {
	for _, p := range m.permissions[orgID] {
		if p.UserID == userID {
			return p.PermissionType, nil
		}
	}
	return "member", nil
}

// newTestHandler creates a handler with a mock repository for testing.
func newTestHandler() (*Handler, *mockOrgRepo) {
	mock := newMockOrgRepo()
	// We need to use the real repository type, so we build the handler differently.
	// Since the handler uses *repository.OrganizationRepository directly, we test
	// via HTTP round-trips using the exported handler methods.
	return nil, mock
}

// seedOrg creates a test organization in the mock repo.
func seedOrg(mock *mockOrgRepo, name, slug, plan string) *repository.Organization {
	org := &repository.Organization{
		ID:        uuid.New().String(),
		Name:      name,
		Slug:      slug,
		Plan:      plan,
		Settings:  map[string]interface{}{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mock.orgs[org.ID] = org
	return org
}

func TestGenerateUUID(t *testing.T) {
	id := generateUUID()

	// Verify it's a valid UUID
	parsed, err := uuid.Parse(id)
	if err != nil {
		t.Fatalf("generateUUID() returned invalid UUID %q: %v", id, err)
	}

	// Verify it's not the old placeholder
	if id == "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx" {
		t.Fatal("generateUUID() still returns placeholder string")
	}

	// Verify UUID version 4
	if parsed.Version() != 4 {
		t.Errorf("expected UUID version 4, got %d", parsed.Version())
	}
}

func TestGenerateUUID_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateUUID()
		if seen[id] {
			t.Fatalf("generateUUID() produced duplicate UUID: %s", id)
		}
		seen[id] = true
	}
}

func TestValidateCreateOrganization(t *testing.T) {
	tests := []struct {
		name    string
		req     *CreateOrganizationRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &CreateOrganizationRequest{
				Name: "Test Org",
				Slug: "test-org",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			req: &CreateOrganizationRequest{
				Slug: "test-org",
			},
			wantErr: true,
		},
		{
			name: "missing slug",
			req: &CreateOrganizationRequest{
				Name: "Test Org",
			},
			wantErr: true,
		},
		{
			name: "name too long",
			req: &CreateOrganizationRequest{
				Name: string(make([]byte, maxOrgNameLength+1)),
				Slug: "test-org",
			},
			wantErr: true,
		},
		{
			name: "slug too long",
			req: &CreateOrganizationRequest{
				Name: "Test Org",
				Slug: string(make([]byte, maxSlugLength+1)),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCreateOrganization(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCreateOrganization() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOrgToResponse(t *testing.T) {
	now := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	org := &repository.Organization{
		ID:        "test-id-123",
		Name:      "Test Organization",
		Slug:      "test-org",
		Plan:      "pro",
		Settings:  map[string]interface{}{"theme": "dark"},
		CreatedAt: now,
		UpdatedAt: now,
	}

	resp := orgToResponse(org)

	if resp.ID != org.ID {
		t.Errorf("ID = %q, want %q", resp.ID, org.ID)
	}
	if resp.Name != org.Name {
		t.Errorf("Name = %q, want %q", resp.Name, org.Name)
	}
	if resp.Slug != org.Slug {
		t.Errorf("Slug = %q, want %q", resp.Slug, org.Slug)
	}
	if resp.Plan != org.Plan {
		t.Errorf("Plan = %q, want %q", resp.Plan, org.Plan)
	}
	if resp.CreatedAt != "2025-01-15T10:30:00Z" {
		t.Errorf("CreatedAt = %q, want %q", resp.CreatedAt, "2025-01-15T10:30:00Z")
	}
	if resp.UpdatedAt != "2025-01-15T10:30:00Z" {
		t.Errorf("UpdatedAt = %q, want %q", resp.UpdatedAt, "2025-01-15T10:30:00Z")
	}
}

func TestRespondJSON(t *testing.T) {
	w := httptest.NewRecorder()

	data := map[string]string{"message": "hello"}
	respondJSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}

	var result map[string]string
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result["message"] != "hello" {
		t.Errorf("message = %q, want %q", result["message"], "hello")
	}
}

func TestRespondError(t *testing.T) {
	w := httptest.NewRecorder()

	appErr := apperrors.New("test error", http.StatusBadRequest)
	respondError(w, appErr)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result["message"] != "test error" {
		t.Errorf("message = %q, want %q", result["message"], "test error")
	}
}

func TestRespondError_GenericError(t *testing.T) {
	w := httptest.NewRecorder()

	err := &testError{"something went wrong"}
	respondError(w, err)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestPermissionToResponse(t *testing.T) {
	grantedBy := "admin-user-id"
	now := time.Date(2025, 3, 10, 14, 0, 0, 0, time.UTC)
	perm := &repository.Permission{
		ID:             "perm-id",
		OrganizationID: "org-id",
		UserID:         "user-id",
		PermissionType: "admin",
		GrantedAt:      now,
		GrantedBy:      &grantedBy,
	}

	resp := permissionToResponse(perm)

	if resp.ID != perm.ID {
		t.Errorf("ID = %q, want %q", resp.ID, perm.ID)
	}
	if resp.OrganizationID != perm.OrganizationID {
		t.Errorf("OrganizationID = %q, want %q", resp.OrganizationID, perm.OrganizationID)
	}
	if resp.UserID != perm.UserID {
		t.Errorf("UserID = %q, want %q", resp.UserID, perm.UserID)
	}
	if resp.PermissionType != perm.PermissionType {
		t.Errorf("PermissionType = %q, want %q", resp.PermissionType, perm.PermissionType)
	}
	if resp.GrantedBy == nil || *resp.GrantedBy != grantedBy {
		t.Errorf("GrantedBy = %v, want %q", resp.GrantedBy, grantedBy)
	}
}

func TestListOrganizations_MethodNotAllowed(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", nil)
	w := httptest.NewRecorder()

	h.ListOrganizations(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestOrganizationHandler_InvalidUUID(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/not-a-uuid", nil)
	w := httptest.NewRecorder()

	h.OrganizationHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrganizationHandler_MissingID(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/", nil)
	w := httptest.NewRecorder()

	h.OrganizationHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrganizationHandler_MethodNotAllowed(t *testing.T) {
	h := &Handler{}
	validID := uuid.New().String()

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/organizations/"+validID, nil)
	w := httptest.NewRecorder()

	h.OrganizationHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestCreateOrganization_InvalidJSON(t *testing.T) {
	h := &Handler{}

	body := bytes.NewBufferString("{invalid json")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", body)
	w := httptest.NewRecorder()

	h.CreateOrganization(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateOrganization_MissingName(t *testing.T) {
	h := &Handler{}

	payload := CreateOrganizationRequest{Slug: "test-slug"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	h.CreateOrganization(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateOrganization_MissingSlug(t *testing.T) {
	h := &Handler{}

	payload := CreateOrganizationRequest{Name: "Test Org"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	h.CreateOrganization(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// testError is a simple error type for testing.
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
