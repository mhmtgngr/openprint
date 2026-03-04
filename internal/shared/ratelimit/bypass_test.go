package ratelimit

import (
	"context"
	"testing"
	"time"
)

// TestBypassManagerShouldBypass tests bypass checking functionality.
func TestBypassManagerShouldBypass(t *testing.T) {
	t.Skip("Requires Redis instance - use testcontainers for integration testing")

	ctx := context.Background()
	redisClient, err := NewRedisClient(&RedisConfig{
		Addr: "localhost:6379",
	})
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer redisClient.Close()

	bm := NewBypassManager(redisClient)

	// Add a trusted client
	client := &TrustedClient{
		ID:          "test-client-1",
		APIKey:      "test-api-key-123",
		Name:        "Test Client",
		IsActive:    true,
		IPWhitelist: []string{"192.168.1.100"},
	}

	err = bm.AddTrustedClient(ctx, client)
	if err != nil {
		t.Fatalf("AddTrustedClient failed: %v", err)
	}

	// Test API key bypass
	req := &Request{
		APIKey:     "test-api-key-123",
		Identifier: "test-user",
		Type:       "user",
	}

	if !bm.ShouldBypass(ctx, req) {
		t.Error("Request with trusted API key should bypass")
	}

	// Test IP bypass
	req = &Request{
		IP:         "192.168.1.100",
		Identifier: "test-user",
		Type:       "ip",
	}

	if !bm.ShouldBypass(ctx, req) {
		t.Error("Request from trusted IP should bypass")
	}

	// Test non-bypassed request
	req = &Request{
		Identifier: "untrusted-user",
		Type:       "user",
	}

	if bm.ShouldBypass(ctx, req) {
		t.Error("Untrusted request should not bypass")
	}

	// Test admin priority bypass
	req = &Request{
		Identifier: "admin-user",
		Type:       "user",
		Priority:   1000,
	}

	if !bm.ShouldBypass(ctx, req) {
		t.Error("High priority request (admin) should bypass")
	}
}

// TestBypassManagerIPInRange tests CIDR range checking.
func TestBypassManagerIPInRange(t *testing.T) {
	bm := &BypassManager{}

	tests := []struct {
		name     string
		ip       string
		cidr     string
		expected bool
	}{
		{"Exact match", "192.168.1.100", "192.168.1.100", true},
		{"CIDR range", "192.168.1.100", "192.168.1.0/24", true},
		{"CIDR range not in", "192.168.2.100", "192.168.1.0/24", false},
		{"Wildcard match", "192.168.1.100", "192.168.1.*", true},
		{"Wildcard not in", "192.168.2.100", "192.168.1.*", false},
		{"IPv4 in range", "10.0.0.5", "10.0.0.0/8", true},
		{"Invalid IP", "not-an-ip", "192.168.1.0/24", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bm.ipInRange(tt.ip, tt.cidr)
			if result != tt.expected {
				t.Errorf("ipInRange(%q, %q) = %v, want %v", tt.ip, tt.cidr, result, tt.expected)
			}
		})
	}
}

// TestBypassManagerAddRemoveTrustedClient tests adding and removing clients.
func TestBypassManagerAddRemoveTrustedClient(t *testing.T) {
	t.Skip("Requires Redis instance - use testcontainers for integration testing")

	ctx := context.Background()
	redisClient, err := NewRedisClient(&RedisConfig{
		Addr: "localhost:6379",
	})
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer redisClient.Close()

	bm := NewBypassManager(redisClient)

	client := &TrustedClient{
		ID:          "test-add-remove",
		APIKey:      "test-key-add-remove",
		Name:        "Test Add Remove",
		IsActive:    true,
		IPWhitelist: []string{"10.0.0.1"},
	}

	// Add client
	err = bm.AddTrustedClient(ctx, client)
	if err != nil {
		t.Fatalf("AddTrustedClient failed: %v", err)
	}

	// Verify it was added
	req := &Request{
		APIKey:     "test-key-add-remove",
		Identifier: "test-user",
		Type:       "user",
	}

	if !bm.ShouldBypass(ctx, req) {
		t.Error("Added client should bypass")
	}

	// Remove client
	err = bm.RemoveTrustedClient(ctx, "test-add-remove")
	if err != nil {
		t.Fatalf("RemoveTrustedClient failed: %v", err)
	}

	// Give time for cache refresh
	time.Sleep(100 * time.Millisecond)

	// Verify it was removed
	if bm.ShouldBypass(ctx, req) {
		t.Error("Removed client should not bypass")
	}
}

// TestBypassManagerGetTrustedClient tests retrieving a trusted client.
func TestBypassManagerGetTrustedClient(t *testing.T) {
	t.Skip("Requires Redis instance - use testcontainers for integration testing")

	ctx := context.Background()
	redisClient, err := NewRedisClient(&RedisConfig{
		Addr: "localhost:6379",
	})
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer redisClient.Close()

	bm := NewBypassManager(redisClient)

	client := &TrustedClient{
		ID:       "test-get-client",
		APIKey:   "test-key-get",
		Name:     "Test Get Client",
		IsActive: true,
	}

	err = bm.AddTrustedClient(ctx, client)
	if err != nil {
		t.Fatalf("AddTrustedClient failed: %v", err)
	}

	// Get by API key
	retrieved, ok := bm.GetTrustedClient(ctx, "test-key-get")
	if !ok {
		t.Error("Client should be found")
	}
	if retrieved.ID != "test-get-client" {
		t.Errorf("Got client ID %s, want 'test-get-client'", retrieved.ID)
	}

	// Get non-existent client
	_, ok = bm.GetTrustedClient(ctx, "non-existent-key")
	if ok {
		t.Error("Non-existent client should not be found")
	}
}

// TestBypassManagerListTrustedClients tests listing all trusted clients.
func TestBypassManagerListTrustedClients(t *testing.T) {
	t.Skip("Requires Redis instance - use testcontainers for integration testing")

	ctx := context.Background()
	redisClient, err := NewRedisClient(&RedisConfig{
		Addr: "localhost:6379",
	})
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer redisClient.Close()

	bm := NewBypassManager(redisClient)

	// Add multiple clients
	clients := []*TrustedClient{
		{ID: "client-1", APIKey: "key-1", Name: "Client 1", IsActive: true},
		{ID: "client-2", APIKey: "key-2", Name: "Client 2", IsActive: true},
		{ID: "client-3", APIKey: "key-3", Name: "Client 3", IsActive: false},
	}

	for _, client := range clients {
		err = bm.AddTrustedClient(ctx, client)
		if err != nil {
			t.Fatalf("AddTrustedClient failed: %v", err)
		}
	}

	// List clients
	list, err := bm.ListTrustedClients(ctx)
	if err != nil {
		t.Fatalf("ListTrustedClients failed: %v", err)
	}

	if len(list) < 3 {
		t.Errorf("Expected at least 3 clients, got %d", len(list))
	}
}

// TestBypassManagerIsTrustedIP tests IP trust checking.
func TestBypassManagerIsTrustedIP(t *testing.T) {
	t.Skip("Requires Redis instance - use testcontainers for integration testing")

	ctx := context.Background()
	redisClient, err := NewRedisClient(&RedisConfig{
		Addr: "localhost:6379",
	})
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer redisClient.Close()

	bm := NewBypassManager(redisClient)

	client := &TrustedClient{
		ID:          "test-ip-trust",
		IPWhitelist: []string{"10.0.0.0/24"},
		IsActive:    true,
	}

	err = bm.AddTrustedClient(ctx, client)
	if err != nil {
		t.Fatalf("AddTrustedClient failed: %v", err)
	}

	// Check IP in range
	if !bm.IsTrustedIP(ctx, "10.0.0.50") {
		t.Error("IP in trusted range should be trusted")
	}

	// Check IP not in range
	if bm.IsTrustedIP(ctx, "192.168.1.1") {
		t.Error("IP not in trusted range should not be trusted")
	}
}

// TestBypassManagerSetClientActive tests activating/deactivating clients.
func TestBypassManagerSetClientActive(t *testing.T) {
	t.Skip("Requires Redis instance - use testcontainers for integration testing")

	ctx := context.Background()
	redisClient, err := NewRedisClient(&RedisConfig{
		Addr: "localhost:6379",
	})
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer redisClient.Close()

	bm := NewBypassManager(redisClient)

	client := &TrustedClient{
		ID:       "test-activate",
		APIKey:   "test-key-activate",
		IsActive: true,
	}

	err = bm.AddTrustedClient(ctx, client)
	if err != nil {
		t.Fatalf("AddTrustedClient failed: %v", err)
	}

	// Deactivate
	err = bm.SetClientActive(ctx, "test-activate", false)
	if err != nil {
		t.Fatalf("SetClientActive failed: %v", err)
	}

	// Give time for cache refresh
	time.Sleep(100 * time.Millisecond)

	req := &Request{
		APIKey:     "test-key-activate",
		Identifier: "test-user",
		Type:       "user",
	}

	if bm.ShouldBypass(ctx, req) {
		t.Error("Inactive client should not bypass")
	}

	// Reactivate
	err = bm.SetClientActive(ctx, "test-activate", true)
	if err != nil {
		t.Fatalf("SetClientActive failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	if !bm.ShouldBypass(ctx, req) {
		t.Error("Active client should bypass")
	}
}

// TestBypassManagerGetBypassStats tests getting bypass statistics.
func TestBypassManagerGetBypassStats(t *testing.T) {
	t.Skip("Requires Redis instance - use testcontainers for integration testing")

	ctx := context.Background()
	redisClient, err := NewRedisClient(&RedisConfig{
		Addr: "localhost:6379",
	})
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer redisClient.Close()

	bm := NewBypassManager(redisClient)

	stats, err := bm.GetBypassStats(ctx)
	if err != nil {
		t.Fatalf("GetBypassStats failed: %v", err)
	}

	if stats == nil {
		t.Fatal("GetBypassStats should never return nil")
	}
	if stats.TotalTrustedAPIKeys < 0 {
		t.Errorf("Invalid TotalTrustedAPIKeys: %d", stats.TotalTrustedAPIKeys)
	}
	if stats.ActiveTrustedAPIKeys < 0 {
		t.Errorf("Invalid ActiveTrustedAPIKeys: %d", stats.ActiveTrustedAPIKeys)
	}
	if stats.TotalTrustedIPs < 0 {
		t.Errorf("Invalid TotalTrustedIPs: %d", stats.TotalTrustedIPs)
	}
	if stats.ActiveTrustedIPs < 0 {
		t.Errorf("Invalid ActiveTrustedIPs: %d", stats.ActiveTrustedIPs)
	}
}

// TestBypassManagerServiceAccount tests service account management.
func TestBypassManagerServiceAccount(t *testing.T) {
	t.Skip("Requires Redis instance - use testcontainers for integration testing")

	ctx := context.Background()
	redisClient, err := NewRedisClient(&RedisConfig{
		Addr: "localhost:6379",
	})
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer redisClient.Close()

	bm := NewBypassManager(redisClient)

	// Add service account
	err = bm.AddServiceAccount(ctx, "service-123")
	if err != nil {
		t.Fatalf("AddServiceAccount failed: %v", err)
	}

	// Check it bypasses
	req := &Request{
		APIKey:     "service-123",
		Identifier: "service-123",
		Type:       "api_key",
	}

	if !bm.ShouldBypass(ctx, req) {
		t.Error("Service account should bypass")
	}

	// List service accounts
	accounts, err := bm.ListServiceAccounts(ctx)
	if err != nil {
		t.Fatalf("ListServiceAccounts failed: %v", err)
	}

	if len(accounts) == 0 {
		t.Error("Should have at least one service account")
	}

	// Remove service account
	err = bm.RemoveServiceAccount(ctx, "service-123")
	if err != nil {
		t.Fatalf("RemoveServiceAccount failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	if bm.ShouldBypass(ctx, req) {
		t.Error("Removed service account should not bypass")
	}
}

// TestBypassManagerBypassToken tests temporary bypass tokens.
func TestBypassManagerBypassToken(t *testing.T) {
	t.Skip("Requires Redis instance - use testcontainers for integration testing")

	ctx := context.Background()
	redisClient, err := NewRedisClient(&RedisConfig{
		Addr: "localhost:6379",
	})
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer redisClient.Close()

	bm := NewBypassManager(redisClient)

	// Add token
	token := "temp-bypass-token-123"
	err = bm.AddBypassToken(ctx, token, time.Minute)
	if err != nil {
		t.Fatalf("AddBypassToken failed: %v", err)
	}

	// Check token exists
	if !bm.CheckBypassToken(ctx, token) {
		t.Error("Token should be valid")
	}

	// Check invalid token
	if bm.CheckBypassToken(ctx, "invalid-token") {
		t.Error("Invalid token should not be valid")
	}
}
