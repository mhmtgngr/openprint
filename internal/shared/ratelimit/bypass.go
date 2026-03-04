package ratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	bypassAPIKeysKey = "ratelimit:bypass:apikeys"
	bypassIPsKey     = "ratelimit:bypass:ips"
)

// BypassManager handles trusted clients that bypass rate limiting.
// State is stored in Redis for distributed compatibility with a local cache.
type BypassManager struct {
	redis *RedisClient

	// Local cache for fast lookups (refreshed periodically)
	localCache      *bypassCache
	cacheExpiration time.Duration
	mu              sync.RWMutex
}

// bypassCache holds local cached bypass data.
type bypassCache struct {
	apiKeys     map[string]*TrustedClient
	ips         map[string]*TrustedClient
	lastRefresh time.Time
}

// NewBypassManager creates a new bypass manager with Redis-backed state.
func NewBypassManager(redis *RedisClient) *BypassManager {
	bm := &BypassManager{
		redis:           redis,
		localCache:      &bypassCache{apiKeys: make(map[string]*TrustedClient), ips: make(map[string]*TrustedClient)},
		cacheExpiration: 5 * time.Minute,
	}

	// Load initial data and start refresh goroutine
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = bm.refreshCache(ctx)

	go bm.periodicRefresh()

	return bm
}

// ShouldBypass checks if a request should bypass rate limiting.
// It checks local cache first for performance, falling back to Redis if needed.
func (b *BypassManager) ShouldBypass(ctx context.Context, req *Request) bool {
	b.mu.RLock()
	cache := b.localCache
	b.mu.RUnlock()

	// Check if cache is stale
	if time.Since(cache.lastRefresh) > b.cacheExpiration {
		b.refreshCache(ctx)
		b.mu.RLock()
		cache = b.localCache
		b.mu.RUnlock()
	}

	// Check API key
	if req.APIKey != "" {
		if client, ok := cache.apiKeys[req.APIKey]; ok && client.IsActive {
			return true
		}
	}

	// Check IP address
	if req.IP != "" {
		if client, ok := cache.ips[req.IP]; ok && client.IsActive {
			return true
		}

		// Check IP ranges (CIDR notation)
		for _, client := range cache.ips {
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
// Writes to Redis and refreshes local cache.
func (b *BypassManager) AddTrustedClient(ctx context.Context, client *TrustedClient) error {
	// Store in Redis as a JSON set entry
	data, err := json.Marshal(client)
	if err != nil {
		return fmt.Errorf("failed to marshal client: %w", err)
	}

	// Add to API keys set
	if client.APIKey != "" {
		if err := b.redis.AddToSet(ctx, bypassAPIKeysKey, string(data)); err != nil {
			return fmt.Errorf("failed to add API key: %w", err)
		}
	}

	// Add IP whitelist entries
	for _, ip := range client.IPWhitelist {
		ipData := map[string]interface{}{
			"client_id": client.ID,
			"ip":        ip,
			"active":    client.IsActive,
		}
		if err := b.redis.SetJSON(ctx, fmt.Sprintf("%s:%s", bypassIPsKey, ip), ipData, 0); err != nil {
			return fmt.Errorf("failed to add IP: %w", err)
		}
	}

	// Refresh cache
	_ = b.refreshCache(ctx)

	return nil
}

// RemoveTrustedClient removes a trusted client from the bypass list.
func (b *BypassManager) RemoveTrustedClient(ctx context.Context, clientID string) error {
	// Get all API keys and find the one with matching client ID
	apiKeys, err := b.redis.GetSet(ctx, bypassAPIKeysKey)
	if err == nil {
		for _, keyData := range apiKeys {
			var client TrustedClient
			if err := json.Unmarshal([]byte(keyData), &client); err == nil {
				if client.ID == clientID {
					_ = b.redis.RemoveFromSet(ctx, bypassAPIKeysKey, keyData)
				}
			}
		}
	}

	// Remove from IPs
	b.mu.RLock()
	cache := b.localCache
	b.mu.RUnlock()

	for _, client := range cache.ips {
		if client.ID == clientID {
			for _, ip := range client.IPWhitelist {
				_ = b.redis.DeleteKey(ctx, fmt.Sprintf("%s:%s", bypassIPsKey, ip))
			}
		}
	}

	// Refresh cache
	_ = b.refreshCache(ctx)

	return nil
}

// GetTrustedClient retrieves a trusted client by API key.
func (b *BypassManager) GetTrustedClient(ctx context.Context, apiKey string) (*TrustedClient, bool) {
	b.mu.RLock()
	cache := b.localCache
	b.mu.RUnlock()

	client, ok := cache.apiKeys[apiKey]
	return client, ok
}

// ListTrustedClients returns all trusted clients.
func (b *BypassManager) ListTrustedClients(ctx context.Context) ([]*TrustedClient, error) {
	b.mu.RLock()
	cache := b.localCache
	b.mu.RUnlock()

	// Use a map to deduplicate clients by ID
	clientMap := make(map[string]*TrustedClient)

	for _, client := range cache.apiKeys {
		clientMap[client.ID] = client
	}

	for _, client := range cache.ips {
		clientMap[client.ID] = client
	}

	result := make([]*TrustedClient, 0, len(clientMap))
	for _, client := range clientMap {
		result = append(result, client)
	}

	return result, nil
}

// UpdateTrustedClient updates a trusted client.
func (b *BypassManager) UpdateTrustedClient(ctx context.Context, client *TrustedClient) error {
	return b.AddTrustedClient(ctx, client)
}

// IsTrustedIP checks if an IP address is trusted.
func (b *BypassManager) IsTrustedIP(ctx context.Context, ip string) bool {
	b.mu.RLock()
	cache := b.localCache
	b.mu.RUnlock()

	if _, ok := cache.ips[ip]; ok {
		return true
	}

	for _, client := range cache.ips {
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
func (b *BypassManager) IsTrustedAPIKey(ctx context.Context, apiKey string) bool {
	b.mu.RLock()
	cache := b.localCache
	b.mu.RUnlock()

	client, ok := cache.apiKeys[apiKey]
	return ok && client.IsActive
}

// SetClientActive sets the active status of a trusted client.
func (b *BypassManager) SetClientActive(ctx context.Context, clientID string, isActive bool) error {
	b.mu.RLock()
	cache := b.localCache
	b.mu.RUnlock()

	updated := false

	// Update in API keys
	for _, client := range cache.apiKeys {
		if client.ID == clientID {
			client.IsActive = isActive
			_ = b.AddTrustedClient(ctx, client)
			updated = true
		}
	}

	// Update in IPs
	for _, client := range cache.ips {
		if client.ID == clientID {
			client.IsActive = isActive
			_ = b.AddTrustedClient(ctx, client)
			updated = true
		}
	}

	if updated {
		_ = b.refreshCache(ctx)
	}

	return nil
}

// ReloadFromRepository reloads trusted clients from the repository.
func (b *BypassManager) ReloadFromRepository(ctx context.Context, repo *Repository) error {
	clients, err := ListTrustedClients(ctx, repo)
	if err != nil {
		return err
	}

	// Clear existing data in Redis
	_ = b.redis.DeleteKey(ctx, bypassAPIKeysKey)

	// Add all clients
	for _, client := range clients {
		data, err := json.Marshal(client)
		if err != nil {
			continue
		}
		if client.APIKey != "" {
			_ = b.redis.AddToSet(ctx, bypassAPIKeysKey, string(data))
		}
		for _, ip := range client.IPWhitelist {
			ipData := map[string]interface{}{
				"client_id": client.ID,
				"ip":        ip,
				"active":    client.IsActive,
			}
			_ = b.redis.SetJSON(ctx, fmt.Sprintf("%s:%s", bypassIPsKey, ip), ipData, 0)
		}
	}

	// Refresh cache
	return b.refreshCache(ctx)
}

// GetBypassStats returns statistics about bypass usage.
func (b *BypassManager) GetBypassStats(ctx context.Context) (*BypassStats, error) {
	b.mu.RLock()
	cache := b.localCache
	b.mu.RUnlock()

	activeAPIKeys := 0
	activeIPs := 0

	for _, client := range cache.apiKeys {
		if client.IsActive {
			activeAPIKeys++
		}
	}

	for _, client := range cache.ips {
		if client.IsActive {
			activeIPs++
		}
	}

	return &BypassStats{
		TotalTrustedAPIKeys: len(cache.apiKeys),
		ActiveTrustedAPIKeys: activeAPIKeys,
		TotalTrustedIPs:     len(cache.ips),
		ActiveTrustedIPs:     activeIPs,
	}, nil
}

// BypassStats represents bypass manager statistics.
type BypassStats struct {
	TotalTrustedAPIKeys int `json:"total_trusted_api_keys"`
	ActiveTrustedAPIKeys int `json:"active_trusted_api_keys"`
	TotalTrustedIPs     int `json:"total_trusted_ips"`
	ActiveTrustedIPs     int `json:"active_trusted_ips"`
}

// CheckBypassToken checks if a bypass token is valid.
// Tokens are stored in Redis with expiration.
func (b *BypassManager) CheckBypassToken(ctx context.Context, token string) bool {
	exists, err := b.redis.SetMemberExists(ctx, "ratelimit:bypass:tokens", token)
	return err == nil && exists
}

// AddBypassToken adds a bypass token for temporary bypass.
func (b *BypassManager) AddBypassToken(ctx context.Context, token string, duration time.Duration) error {
	key := fmt.Sprintf("ratelimit:bypass:token:%s", token)
	if err := b.redis.AddToSet(ctx, "ratelimit:bypass:tokens", token); err != nil {
		return err
	}
	return b.redis.SetKeyExpiration(ctx, key, duration)
}

// AddServiceAccount adds a service account that bypasses rate limiting.
func (b *BypassManager) AddServiceAccount(ctx context.Context, serviceID string) error {
	client := &TrustedClient{
		ID:       serviceID,
		Name:     "Service Account: " + serviceID,
		APIKey:   serviceID,
		IsActive: true,
	}

	return b.AddTrustedClient(ctx, client)
}

// RemoveServiceAccount removes a service account.
func (b *BypassManager) RemoveServiceAccount(ctx context.Context, serviceID string) error {
	return b.RemoveTrustedClient(ctx, serviceID)
}

// ListServiceAccounts lists all service accounts.
func (b *BypassManager) ListServiceAccounts(ctx context.Context) ([]string, error) {
	b.mu.RLock()
	cache := b.localCache
	b.mu.RUnlock()

	var accounts []string
	for key, client := range cache.apiKeys {
		if client.IsActive && strings.HasPrefix(client.Name, "Service Account:") {
			accounts = append(accounts, key)
		}
	}

	return accounts, nil
}

// refreshCache refreshes the local cache from Redis.
func (b *BypassManager) refreshCache(ctx context.Context) error {
	newCache := &bypassCache{
		apiKeys: make(map[string]*TrustedClient),
		ips:     make(map[string]*TrustedClient),
	}

	// Load API keys from Redis set
	apiKeys, err := b.redis.GetSet(ctx, bypassAPIKeysKey)
	if err == nil {
		for _, keyData := range apiKeys {
			var client TrustedClient
			if err := json.Unmarshal([]byte(keyData), &client); err == nil {
				newCache.apiKeys[client.APIKey] = &client
			}
		}
	}

	// Load IPs from Redis keys
	ipKeys, err := b.redis.ListKeys(ctx, bypassIPsKey+":*")
	if err == nil {
		for _, ipKey := range ipKeys {
			var data map[string]interface{}
			if err := b.redis.GetJSON(ctx, ipKey, &data); err == nil {
				clientID, _ := data["client_id"].(string)
				ip, _ := data["ip"].(string)
				active := true
				if a, ok := data["active"].(bool); ok {
					active = a
				}

				// Create a minimal client for IP lookup
				newCache.ips[ip] = &TrustedClient{
					ID:        clientID,
					IPWhitelist: []string{ip},
					IsActive:  active,
				}
			}
		}
	}

	newCache.lastRefresh = time.Now()

	b.mu.Lock()
	b.localCache = newCache
	b.mu.Unlock()

	return nil
}

// periodicRefresh periodically refreshes the local cache.
func (b *BypassManager) periodicRefresh() {
	ticker := time.NewTicker(b.cacheExpiration / 2)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_ = b.refreshCache(ctx)
		cancel()
	}
}
