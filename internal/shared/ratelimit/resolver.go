package ratelimit

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// PolicyResolver handles hierarchical policy resolution.
// Resolution priority: Global > Organization > User/API Key > Endpoint
type PolicyResolver struct {
	defaultPolicy *Policy
	cache         map[string]*Policy // Simple in-memory cache
}

// NewPolicyResolver creates a new policy resolver.
func NewPolicyResolver(defaultPolicy *Policy) *PolicyResolver {
	if defaultPolicy == nil {
		defaultPolicy = DefaultGlobalPolicy()
	}
	return &PolicyResolver{
		defaultPolicy: defaultPolicy,
		cache:         make(map[string]*Policy),
	}
}

// Resolve finds the applicable policy for a request using hierarchical resolution.
// Resolution order (highest to lowest priority):
// 1. Endpoint-specific policy (path + method)
// 2. API Key policy
// 3. User policy
// 4. Organization policy
// 5. IP-based policy
// 6. Global default policy
func (r *PolicyResolver) Resolve(ctx context.Context, repo *Repository, req *Request) (*Policy, error) {
	// Build cache key
	cacheKey := r.buildCacheKey(req)

	// Check cache first
	if cached, ok := r.cache[cacheKey]; ok {
		return cached, nil
	}

	// Fetch all applicable policies
	policies, err := repo.GetActivePolicies(ctx)
	if err != nil {
		// Return default policy on error
		return r.defaultPolicy, nil
	}

	// Filter and score policies based on specificity
	candidates := r.scorePolicies(policies, req)

	if len(candidates) == 0 {
		return r.defaultPolicy, nil
	}

	// Sort by score (descending) and then by priority (descending)
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].policy.Priority > candidates[j].policy.Priority
	})

	// Return the highest scoring policy
	best := candidates[0].policy

	// Cache the result
	r.cache[cacheKey] = best

	return best, nil
}

type policyScore struct {
	policy *Policy
	score  int
}

// scorePolicies scores policies based on how specifically they match the request.
func (r *PolicyResolver) scorePolicies(policies []*Policy, req *Request) []policyScore {
	var candidates []policyScore

	for _, policy := range policies {
		if !policy.IsActive {
			continue
		}

		score := r.calculateScore(policy, req)
		if score > 0 {
			candidates = append(candidates, policyScore{
				policy: policy,
				score:  score,
			})
		}
	}

	return candidates
}

// calculateScore calculates a specificity score for a policy.
// Higher score = more specific policy.
func (r *PolicyResolver) calculateScore(policy *Policy, req *Request) int {
	score := 0

	// Check if policy matches basic criteria
	if !policy.Matches(policy.Scope, req.Identifier, req.Method, req.Path) {
		return 0
	}

	// Base score by scope priority (more specific = higher)
	switch policy.Scope {
	case "endpoint":
		score += 1000
	case "api_key":
		score += 800
	case "user":
		score += 600
	case "organization":
		score += 400
	case "ip":
		score += 200
	case "global":
		score += 100
	}

	// Bonus points for specificity

	// Exact identifier match
	if policy.Identifier != "*" && policy.Identifier == req.Identifier {
		score += 50
	}

	// Exact method match
	if len(policy.Methods) > 0 {
		methodMatched := false
		for _, m := range policy.Methods {
			if m == req.Method {
				methodMatched = true
				break
			}
		}
		if methodMatched {
			score += 30
		}
	}

	// Path pattern specificity
	if policy.PathPattern != "" && policy.PathPattern != "*" {
		if policy.PathPattern == req.Path {
			// Exact match
			score += 100
		} else if strings.Contains(policy.PathPattern, "*") {
			// Wildcard match - less specific
			score += 50
		} else {
			// Prefix match
			if strings.HasPrefix(req.Path, policy.PathPattern) {
				score += 40
			}
		}
	}

	// Role-specific bonus (if policy has role requirements)
	if policy.Identifier != "*" && strings.HasPrefix(policy.Identifier, "role:") {
		role := strings.TrimPrefix(policy.Identifier, "role:")
		if role == req.Role {
			score += 25
		}
	}

	return score
}

// buildCacheKey builds a cache key for policy lookups.
func (r *PolicyResolver) buildCacheKey(req *Request) string {
	parts := []string{
		req.Type,
		req.Identifier,
		req.Method,
		req.Path,
		req.Role,
		req.OrgID,
	}
	return strings.Join(parts, ":")
}

// ClearCache clears the policy cache.
func (r *PolicyResolver) ClearCache() {
	r.cache = make(map[string]*Policy)
}

// InvalidateCache invalidates cache entries for a specific scope.
func (r *PolicyResolver) InvalidateCache(scope, identifier string) {
	prefix := fmt.Sprintf("%s:%s:", scope, identifier)
	for key := range r.cache {
		if strings.HasPrefix(key, prefix) {
			delete(r.cache, key)
		}
	}
}

// AddPolicy adds a policy to the resolver and invalidates relevant cache.
func (r *PolicyResolver) AddPolicy(policy *Policy) {
	r.InvalidateCache(policy.Scope, policy.Identifier)
}

// UpdatePolicy updates a policy and invalidates relevant cache.
func (r *PolicyResolver) UpdatePolicy(policy *Policy) {
	r.InvalidateCache(policy.Scope, policy.Identifier)
}

// RemovePolicy removes a policy and invalidates relevant cache.
func (r *PolicyResolver) RemovePolicy(policyID string) {
	// Find and remove from cache, then invalidate
	for key, policy := range r.cache {
		if policy.ID == policyID {
			delete(r.cache, key)
			r.InvalidateCache(policy.Scope, policy.Identifier)
			break
		}
	}
}

// PolicyChain represents a chain of policies for debugging.
type PolicyChain struct {
	Global       *Policy `json:"global,omitempty"`
	Organization *Policy `json:"organization,omitempty"`
	User         *Policy `json:"user,omitempty"`
	APIKey       *Policy `json:"api_key,omitempty"`
	Endpoint     *Policy `json:"endpoint,omitempty"`
	IP           *Policy `json:"ip,omitempty"`
	Applied      *Policy `json:"applied"`
}

// ResolveWithChain resolves the policy and returns the full chain for debugging.
func (r *PolicyResolver) ResolveWithChain(ctx context.Context, repo *Repository, req *Request) (*PolicyChain, error) {
	policies, err := repo.GetActivePolicies(ctx)
	if err != nil {
		return &PolicyChain{Applied: r.defaultPolicy}, nil
	}

	chain := &PolicyChain{
		Applied: r.defaultPolicy,
	}

	// Find matching policies at each scope level
	for _, policy := range policies {
		if !policy.IsActive {
			continue
		}

		if !policy.Matches(policy.Scope, req.Identifier, req.Method, req.Path) {
			continue
		}

		switch policy.Scope {
		case "global":
			if chain.Global == nil || policy.Priority > chain.Global.Priority {
				chain.Global = policy
			}
		case "organization":
			if chain.Organization == nil || policy.Priority > chain.Organization.Priority {
				chain.Organization = policy
			}
		case "user":
			if chain.User == nil || policy.Priority > chain.User.Priority {
				chain.User = policy
			}
		case "api_key":
			if chain.APIKey == nil || policy.Priority > chain.APIKey.Priority {
				chain.APIKey = policy
			}
		case "endpoint":
			if chain.Endpoint == nil || policy.Priority > chain.Endpoint.Priority {
				chain.Endpoint = policy
			}
		case "ip":
			if chain.IP == nil || policy.Priority > chain.IP.Priority {
				chain.IP = policy
			}
		}
	}

	// Determine which policy to apply based on hierarchy
	// Priority: endpoint > api_key > user > organization > ip > global
	if chain.Endpoint != nil {
		chain.Applied = chain.Endpoint
	} else if chain.APIKey != nil {
		chain.Applied = chain.APIKey
	} else if chain.User != nil {
		chain.Applied = chain.User
	} else if chain.Organization != nil {
		chain.Applied = chain.Organization
	} else if chain.IP != nil {
		chain.Applied = chain.IP
	} else if chain.Global != nil {
		chain.Applied = chain.Global
	}

	return chain, nil
}

// EffectiveLimits calculates the effective rate limit by considering all applicable policies.
// When multiple policies apply, the most restrictive limit is used.
func (r *PolicyResolver) EffectiveLimits(ctx context.Context, repo *Repository, req *Request) (*Policy, error) {
	chain, err := r.ResolveWithChain(ctx, repo, req)
	if err != nil {
		return r.defaultPolicy, nil
	}

	// Start with the applied policy
	effective := chain.Applied

	// If no specific policy, use default
	if effective == nil {
		return r.defaultPolicy, nil
	}

	// Check if any other policy in the chain is more restrictive
	policies := []*Policy{chain.Global, chain.Organization, chain.User, chain.APIKey, chain.Endpoint, chain.IP}
	for _, p := range policies {
		if p == nil || p == effective {
			continue
		}

		// If another policy has a lower limit, it's more restrictive
		if p.Limit < effective.Limit {
			effective.Limit = p.Limit
		}
		// If another policy has a shorter window, it's more restrictive
		if p.Window < effective.Window {
			effective.Window = p.Window
		}
	}

	return effective, nil
}

// ValidatePolicy checks if a policy configuration is valid.
func ValidatePolicy(policy *Policy) error {
	if policy.Name == "" {
		return fmt.Errorf("policy name is required")
	}

	validScopes := map[string]bool{
		"global":       true,
		"endpoint":     true,
		"user":         true,
		"api_key":      true,
		"organization": true,
		"ip":           true,
	}

	if !validScopes[policy.Scope] {
		return fmt.Errorf("invalid scope: %s", policy.Scope)
	}

	if policy.Limit <= 0 {
		return fmt.Errorf("limit must be positive")
	}

	if policy.Window <= 0 {
		return fmt.Errorf("window must be positive")
	}

	if policy.BurstLimit > 0 && policy.BurstDuration <= 0 {
		return fmt.Errorf("burst_duration is required when burst_limit is set")
	}

	if policy.BurstLimit > policy.Limit && policy.BurstDuration == 0 {
		// Burst limit higher than sustained limit requires duration
		return fmt.Errorf("burst_duration required when burst_limit exceeds limit")
	}

	validActions := map[string]bool{
		"reject":     true,
		"throttle":   true,
		"queue":      true,
		"alert_only": true,
	}

	if !validActions[policy.Action] {
		return fmt.Errorf("invalid action: %s", policy.Action)
	}

	validSeverities := map[string]bool{
		"low":      true,
		"medium":   true,
		"high":     true,
		"critical": true,
	}

	if !validSeverities[policy.Severity] {
		return fmt.Errorf("invalid severity: %s", policy.Severity)
	}

	return nil
}
