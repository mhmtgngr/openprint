// Package repository provides tests for agent data access layer.
package repository

import (
	"context"
	"testing"
	"time"
)

func TestNewAgentRepository(t *testing.T) {
	repo := NewAgentRepository(nil)

	if repo == nil {
		t.Fatal("NewAgentRepository() returned nil")
	}
}

func TestAgent_Struct(t *testing.T) {
	now := time.Now()
	orgID := "org-123"

	agent := &Agent{
		ID:             "agent-123",
		Name:           "Test Agent",
		Version:        "1.0.0",
		OS:             "linux",
		Architecture:   "amd64",
		Hostname:       "test-host",
		OrganizationID: orgID,
		Status:         "online",
		LastHeartbeat:  now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if agent.ID != "agent-123" {
		t.Error("Agent ID not set correctly")
	}
	if agent.Name != "Test Agent" {
		t.Error("Agent Name not set correctly")
	}
	if agent.Status != "online" {
		t.Error("Agent Status should be online")
	}
	if agent.OrganizationID != orgID {
		t.Error("Agent OrganizationID not set correctly")
	}
}

func TestAgentRepository_CRUD(t *testing.T) {
	repo := NewAgentRepository(nil)
	ctx := context.Background()

	agent := &Agent{
		ID:             "agent-123",
		Name:           "Test Agent",
		Version:        "1.0.0",
		OS:             "linux",
		Architecture:   "amd64",
		Hostname:       "test-host",
		OrganizationID: "org-123",
		Status:         "online",
	}

	t.Run("create agent", func(t *testing.T) {
		// Skip this test as it requires a database
		t.Skip("Requires database connection")
		err := repo.Create(ctx, agent)
		if err == nil {
			t.Log("Create() succeeded (unexpected without DB)")
		}
	})

	t.Run("find by ID", func(t *testing.T) {
		t.Skip("Requires database connection")
		_, err := repo.FindByID(ctx, "agent-123")
		if err == nil {
			t.Log("FindByID() succeeded (unexpected without DB)")
		}
	})

	t.Run("find by hostname", func(t *testing.T) {
		t.Skip("Requires database connection")
		_, err := repo.FindByHostname(ctx, "test-host")
		if err == nil {
			t.Log("FindByHostname() succeeded (unexpected without DB)")
		}
	})

	t.Run("update agent", func(t *testing.T) {
		t.Skip("Requires database connection")
		err := repo.Update(ctx, agent)
		if err == nil {
			t.Log("Update() succeeded (unexpected without DB)")
		}
	})

	t.Run("update heartbeat", func(t *testing.T) {
		t.Skip("Requires database connection")
		err := repo.UpdateHeartbeat(ctx, "agent-123", time.Now())
		if err == nil {
			t.Log("UpdateHeartbeat() succeeded (unexpected without DB)")
		}
	})

	t.Run("set status", func(t *testing.T) {
		t.Skip("Requires database connection")
		err := repo.SetStatus(ctx, "agent-123", "offline")
		if err == nil {
			t.Log("SetStatus() succeeded (unexpected without DB)")
		}
	})

	t.Run("delete agent", func(t *testing.T) {
		t.Skip("Requires database connection")
		err := repo.Delete(ctx, "agent-123")
		if err == nil {
			t.Log("Delete() succeeded (unexpected without DB)")
		}
	})
}

func TestAgentRepository_QueryMethods(t *testing.T) {
	t.Skip("Requires database connection")
	repo := NewAgentRepository(nil)
	ctx := context.Background()

	t.Run("find by status", func(t *testing.T) {
		_, err := repo.FindByStatus(ctx, "online")
		if err == nil {
			t.Log("FindByStatus() succeeded (unexpected without DB)")
		}
	})

	t.Run("find by organization", func(t *testing.T) {
		_, err := repo.FindByOrganization(ctx, "org-123")
		if err == nil {
			t.Log("FindByOrganization() succeeded (unexpected without DB)")
		}
	})

	t.Run("list agents", func(t *testing.T) {
		_, _, err := repo.List(ctx, 10, 0)
		if err == nil {
			t.Log("List() succeeded (unexpected without DB)")
		}
	})

	t.Run("count by status", func(t *testing.T) {
		_, err := repo.CountByStatus(ctx, "online")
		if err == nil {
			t.Log("CountByStatus() succeeded (unexpected without DB)")
		}
	})
}

func TestAgentRepository_StatusManagement(t *testing.T) {
	t.Skip("Requires database connection")
	repo := NewAgentRepository(nil)
	ctx := context.Background()

	t.Run("mark offline before", func(t *testing.T) {
		threshold := time.Now()
		affected, err := repo.MarkOfflineBefore(ctx, threshold)
		if err == nil {
			t.Logf("MarkOfflineBefore() affected %d agents (unexpected without DB)", affected)
		}
	})

	t.Run("get stale agents", func(t *testing.T) {
		since := time.Now().Add(-1 * time.Hour)
		_, err := repo.GetStaleAgents(ctx, since)
		if err == nil {
			t.Log("GetStaleAgents() succeeded (unexpected without DB)")
		}
	})
}

func TestAgentRepository_RegisterOrFindByHostname(t *testing.T) {
	t.Skip("Requires database connection")
	repo := NewAgentRepository(nil)
	ctx := context.Background()

	t.Run("register or find", func(t *testing.T) {
		agent, err := repo.RegisterOrFindByHostname(ctx, "Test Agent", "1.0.0", "linux", "amd64", "new-host")
		if err == nil {
			t.Log("RegisterOrFindByHostname() succeeded (unexpected without DB)")
		}
		_ = agent
	})
}

func TestAgent_StatusValues(t *testing.T) {
	validStatuses := []string{"online", "offline"}

	for _, status := range validStatuses {
		agent := &Agent{Status: status}
		if agent.Status != status {
			t.Errorf("Agent status not set correctly to %s", status)
		}
	}
}

func TestAgent_TimeFields(t *testing.T) {
	now := time.Now()
	agent := &Agent{
		LastHeartbeat: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if agent.LastHeartbeat.IsZero() {
		t.Error("LastHeartbeat should not be zero")
	}
	if agent.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
	if agent.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero")
	}
}

func TestAgent_OrganizationScoped(t *testing.T) {
	tests := []struct {
		name           string
		organizationID string
		hasOrg         bool
	}{
		{
			name:           "with organization",
			organizationID: "org-123",
			hasOrg:         true,
		},
		{
			name:           "without organization",
			organizationID: "",
			hasOrg:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &Agent{
				ID:             "agent-123",
				Name:           "Test Agent",
				OrganizationID: tt.organizationID,
			}

			if tt.hasOrg && agent.OrganizationID == "" {
				t.Error("Agent should have organization ID")
			}
			if !tt.hasOrg && agent.OrganizationID != "" {
				t.Error("Agent should not have organization ID")
			}
		})
	}
}
