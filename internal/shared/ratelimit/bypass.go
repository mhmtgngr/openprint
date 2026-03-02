package ratelimit

import (
	"context"
	"net"
	"strings"
	"sync"
)

// BypassManager handles trusted clients that bypass rate limiting.
type BypassManager struct {
	redis          *RedisClient
	trustedAPIKeys map[string]*TrustedClient
	trustedIPs     map[string]*TrustedClient
	mu             sync.RWMutex
}

// NewBypassManager creates a new bypass manager.
func NewBypassManager(apiKeys []string) *BypassManager {
	bm := &BypassManager{
		trustedAPIKeys: make(map[string]*TrustedClient),
		trustedIPs:     make(map[string]*TrustedClient),
	}

	for _, key := range apiKeys {
		bm.trustedAPIKeys[key] = &TrustedClient{
			APIKey:  key,
			IsActive: true,
		}
	}

	return bm
}

// ShouldBypass checks if a request should bypass rate limiting.
func (b *BypassManager) ShouldBypass(ctx context.Context, req *Request) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Check API key
	if req.APIKey != "" {
		if client, ok := b.trustedAPIKeys[req.APIKey]; ok && client.IsActive {
			return true
		}
	}

	// Check IP address
	if req.IP != "" {
		if client, ok := b.trustedIPs[req.IP]; ok && client.IsActive {
			return true
		}

		// Check IP ranges (CIDR notation)
		for _, client := range b.trustedIPs {
			if !client.IsActive {
				continue
			}
			for _, ipRange := range client.IPWhitelist {
				if b.ipInRange(req.IP, ipRange) {
					return true
				}
			}
		}
	}

	// Check for bypass token in request (from context)
	// This allows internal services to bypass
	if req.Priority >= 1000 {
		return true
	}

	return false
}

// ipInRange checks if an IP is within a CIDR range.
func (b *BypassManager) ipInRange(ip, cidr string) bool {
	// Parse the IP address
	checkIP := net.ParseIP(ip)
	if checkIP == nil {
		return false
	}

	// Parse the CIDR range
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		// If not a valid CIDR, check for exact match or wildcard
		if strings.Contains(cidr, "*") {
			// Simple wildcard matching
			parts := strings.Split(cidr, ".")
			ipParts := strings.Split(ip, ".")
			if len(parts) != len(ipParts) {
				return false
			}
			for i, part := range parts {
				if part != "*" && part != ipParts[i] {
					return false
				}
			}
			return true
		}
		return ip == cidr
	}

	return ipNet.Contains(checkIP)
}

// AddTrustedClient adds a trusted client to the bypass list.
func (b *BypassManager) AddTrustedClient(client *TrustedClient) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if client.APIKey != "" {
		b.trustedAPIKeys[client.APIKey] = client
	}

	for _, ip := range client.IPWhitelist {
		b.trustedIPs[ip] = client
	}
}

// RemoveTrustedClient removes a trusted client from the bypass list.
func (b *BypassManager) RemoveTrustedClient(clientID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Remove from API keys
	for key, client := range b.trustedAPIKeys {
		if client.ID == clientID {
			delete(b.trustedAPIKeys, key)
		}
	}

	// Remove from IPs
	for key, client := range b.trustedIPs {
		if client.ID == clientID {
			delete(b.trustedIPs, key)
		}
	}
}

// GetTrustedClient retrieves a trusted client by API key.
func (b *BypassManager) GetTrustedClient(apiKey string) (*TrustedClient, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	client, ok := b.trustedAPIKeys[apiKey]
	return client, ok
}

// ListTrustedClients returns all trusted clients.
func (b *BypassManager) ListTrustedClients() []*TrustedClient {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Use a map to deduplicate clients by ID
	clientMap := make(map[string]*TrustedClient)

	for _, client := range b.trustedAPIKeys {
		clientMap[client.ID] = client
	}

	for _, client := range b.trustedIPs {
		clientMap[client.ID] = client
	}

	result := make([]*TrustedClient, 0, len(clientMap))
	for _, client := range clientMap {
		result = append(result, client)
	}

	return result
}

// UpdateTrustedClient updates a trusted client.
func (b *BypassManager) UpdateTrustedClient(client *TrustedClient) {
	b.RemoveTrustedClient(client.ID)
	b.AddTrustedClient(client)
}

// IsTrustedIP checks if an IP address is trusted.
func (b *BypassManager) IsTrustedIP(ip string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if _, ok := b.trustedIPs[ip]; ok {
		return true
	}

	for _, client := range b.trustedIPs {
		if !client.IsActive {
			continue
		}
		for _, ipRange := range client.IPWhitelist {
			if b.ipInRange(ip, ipRange) {
				return true
			}
		}
	}

	return false
}

// IsTrustedAPIKey checks if an API key is trusted.
func (b *BypassManager) IsTrustedAPIKey(apiKey string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	client, ok := b.trustedAPIKeys[apiKey]
	return ok && client.IsActive
}

// SetClientActive sets the active status of a trusted client.
func (b *BypassManager) SetClientActive(clientID string, isActive bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, client := range b.trustedAPIKeys {
		if client.ID == clientID {
			client.IsActive = isActive
		}
	}

	for _, client := range b.trustedIPs {
		if client.ID == clientID {
			client.IsActive = isActive
		}
	}
}

// ReloadFromRepository reloads trusted clients from the repository.
func (b *BypassManager) ReloadFromRepository(ctx context.Context, repo *Repository) error {
	clients, err := ListTrustedClients(ctx, repo)
	if err != nil {
		return err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.trustedAPIKeys = make(map[string]*TrustedClient)
	b.trustedIPs = make(map[string]*TrustedClient)

	for _, client := range clients {
		if client.APIKey != "" {
			b.trustedAPIKeys[client.APIKey] = client
		}
		for _, ip := range client.IPWhitelist {
			b.trustedIPs[ip] = client
		}
	}

	return nil
}

// GetBypassStats returns statistics about bypass usage.
func (b *BypassManager) GetBypassStats() *BypassStats {
	b.mu.RLock()
	defer b.mu.RUnlock()

	activeAPIKeys := 0
	activeIPs := 0

	for _, client := range b.trustedAPIKeys {
		if client.IsActive {
			activeAPIKeys++
		}
	}

	for _, client := range b.trustedIPs {
		if client.IsActive {
			activeIPs++
		}
	}

	return &BypassStats{
		TotalTrustedAPIKeys: len(b.trustedAPIKeys),
		ActiveTrustedAPIKeys: activeAPIKeys,
		TotalTrustedIPs:     len(b.trustedIPs),
		ActiveTrustedIPs:     activeIPs,
	}
}

// BypassStats represents bypass manager statistics.
type BypassStats struct {
	TotalTrustedAPIKeys int `json:"total_trusted_api_keys"`
	ActiveTrustedAPIKeys int `json:"active_trusted_api_keys"`
	TotalTrustedIPs     int `json:"total_trusted_ips"`
	ActiveTrustedIPs     int `json:"active_trusted_ips"`
}

// CheckBypassToken checks if a bypass token is valid.
func (b *BypassManager) CheckBypassToken(token string) bool {
	// In production, validate against a secure token store
	// For now, return false as tokens should be explicitly configured
	return false
}

// AddBypassToken adds a bypass token for temporary bypass.
func (b *BypassManager) AddBypassToken(token string, duration int) {
	// Store token with expiration in Redis
	// Implementation would use Redis with TTL
}

// AddServiceAccount adds a service account that bypasses rate limiting.
func (b *BypassManager) AddServiceAccount(serviceID string) {
	client := &TrustedClient{
		ID:       serviceID,
		Name:     "Service Account: " + serviceID,
		APIKey:   serviceID,
		IsActive: true,
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.trustedAPIKeys[serviceID] = client
}

// RemoveServiceAccount removes a service account.
func (b *BypassManager) RemoveServiceAccount(serviceID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.trustedAPIKeys, serviceID)
}

// ListServiceAccounts lists all service accounts.
func (b *BypassManager) ListServiceAccounts() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var accounts []string
	for key, client := range b.trustedAPIKeys {
		if client.IsActive && strings.HasPrefix(client.Name, "Service Account:") {
			accounts = append(accounts, key)
		}
	}

	return accounts
}
