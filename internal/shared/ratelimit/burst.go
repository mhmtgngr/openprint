package ratelimit

import (
	"context"
	"fmt"
	"time"
)

// BurstManager handles burst vs sustained rate limiting.
// Burst allows temporary higher request rates before settling to sustained limits.
// State is stored in Redis for distributed compatibility.
type BurstManager struct {
	redis *RedisClient
}

// NewBurstManager creates a new burst manager with Redis-backed state.
func NewBurstManager(redis *RedisClient) *BurstManager {
	return &BurstManager{
		redis: redis,
	}
}

// burstState represents the burst state stored in Redis.
type burstState struct {
	Tokens        int64
	LastRefill    int64
	WindowStart   int64
	BurstDuration int64 // seconds
	Capacity      int64
}

// CheckBurst checks if a burst request should be allowed.
// Returns (allowed, remainingTokens, burstRemainingTime, error).
func (bm *BurstManager) CheckBurst(ctx context.Context, key string, policy *Policy) (bool, int64, time.Duration, error) {
	redisKey := fmt.Sprintf("burst:%s", key)

	// Lua script for atomic burst check
	script := `
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local capacity = tonumber(ARGV[2])
		local burst_limit = tonumber(ARGV[3])
		local refill_rate = tonumber(ARGV[4])
		local burst_duration = tonumber(ARGV[5])
		local sustained_rate = tonumber(ARGV[6])

		-- Get current state or initialize
		local tokens = tonumber(redis.call("HGET", key, "tokens"))
		local last_refill = tonumber(redis.call("HGET", key, "last_refill"))
		local window_start = tonumber(redis.call("HGET", key, "window_start"))

		if not tokens then
			-- First request, initialize with burst limit
			tokens = burst_limit
			last_refill = now
			window_start = now
			redis.call("HMSET", key, "tokens", tokens, "last_refill", last_refill, "window_start", window_start, "capacity", capacity, "burst_duration", burst_duration)
			redis.call("EXPIRE", key, 3600)
			return {1, tokens - 1, burst_duration, now}
		end

		-- Calculate elapsed time and refill
		local elapsed = now - last_refill
		if elapsed >= 1 then
			local tokens_to_add = elapsed * refill_rate
			tokens = tokens + tokens_to_add
			if tokens > capacity then
				tokens = capacity
			end
			last_refill = now
		end

		-- Check if in burst period
		local in_burst = (now - window_start) < burst_duration

		if in_burst and tokens > 0 then
			-- Consume from burst
			tokens = tokens - 1
			redis.call("HMSET", key, "tokens", tokens, "last_refill", last_refill)
			redis.call("EXPIRE", key, 3600)
			local remaining = math.floor(burst_duration - (now - window_start))
			return {1, tokens, remaining, now}
		end

		if tokens > 0 then
			-- Use sustained rate
			tokens = tokens - 1
			redis.call("HMSET", key, "tokens", tokens, "last_refill", last_refill)
			redis.call("EXPIRE", key, 3600)
			return {1, tokens, 0, now}
		end

		-- Rate limited
		local retry_after = burst_duration
		if in_burst then
			retry_after = math.floor(burst_duration - (now - window_start))
		else
			retry_after = 60
		end
		return {0, 0, retry_after, now}
	`

	now := time.Now()
	refillRate := int64(float64(policy.Limit) / policy.Window.Seconds())
	burstDurationSec := int64(policy.BurstDuration.Seconds())
	sustainedRate := policy.Limit

	result, err := bm.redis.ExecuteLua(ctx, script, []string{redisKey},
		now.Unix(),
		policy.Limit,
		policy.BurstLimit,
		refillRate,
		burstDurationSec,
		sustainedRate,
	)

	if err != nil {
		return false, 0, 0, fmt.Errorf("redis error: %w", err)
	}

	values := result.([]interface{})
	allowed := values[0].(int64) == 1
	remaining := values[1].(int64)
	burstRemaining := time.Duration(values[2].(int64)) * time.Second

	return allowed, remaining, burstRemaining, nil
}

// GetBurstStatus returns the current burst status for a key.
func (bm *BurstManager) GetBurstStatus(ctx context.Context, key string, policy *Policy) *BurstStatus {
	redisKey := fmt.Sprintf("burst:%s", key)

	// Get state from Redis
	hash, err := bm.redis.GetAllHash(ctx, redisKey)
	if err != nil {
		return &BurstStatus{
			Key: key,
		}
	}

	now := time.Now()
	status := &BurstStatus{
		Key:           key,
		BurstCapacity: policy.Limit,
		SustainedRate: policy.Limit,
	}

	if tokens, ok := hash["tokens"]; ok {
		_, _ = fmt.Sscanf(tokens, "%d", &status.BurstTokens)
	}
	if lastRefill, ok := hash["last_refill"]; ok {
		var lastRefillUnix int64
		_, _ = fmt.Sscanf(lastRefill, "%d", &lastRefillUnix)
		status.LastRefill = time.Unix(lastRefillUnix, 0)
	}
	if windowStart, ok := hash["window_start"]; ok {
		var windowStartUnix int64
		_, _ = fmt.Sscanf(windowStart, "%d", &windowStartUnix)
		windowStartTime := time.Unix(windowStartUnix, 0)
		if burstDuration, ok := hash["burst_duration"]; ok {
			var burstDurSec int64
			_, _ = fmt.Sscanf(burstDuration, "%d", &burstDurSec)
			burstEnd := windowStartTime.Add(time.Duration(burstDurSec) * time.Second)
			status.InBurstPeriod = now.Before(burstEnd)
			status.BurstRemaining = time.Until(burstEnd)
			if status.BurstRemaining < 0 {
				status.BurstRemaining = 0
			}
		}
	}

	return status
}

// BurstStatus represents the current burst status.
type BurstStatus struct {
	Key            string        `json:"key"`
	BurstTokens    int64         `json:"burst_tokens"`
	BurstCapacity  int64         `json:"burst_capacity"`
	SustainedRate  int64         `json:"sustained_rate"`
	InBurstPeriod  bool          `json:"in_burst_period"`
	BurstRemaining time.Duration `json:"burst_remaining"`
	LastRefill     time.Time     `json:"last_refill"`
}

// CheckSustained checks if the sustained rate allows a request.
func (bm *BurstManager) CheckSustained(ctx context.Context, key string, policy *Policy) (bool, int64, time.Time, error) {
	redisKey := fmt.Sprintf("burst:%s", key)

	script := `
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local limit = tonumber(ARGV[2])
		local window = tonumber(ARGV[3])

		local tokens = tonumber(redis.call("HGET", key, "tokens"))
		local last_refill = tonumber(redis.call("HGET", key, "last_refill"))

		if not tokens then
			tokens = limit
			last_refill = now
			redis.call("HMSET", key, "tokens", tokens, "last_refill", last_refill)
			redis.call("EXPIRE", key, 3600)
			return {1, tokens - 1, now + window}
		end

		-- Refill based on elapsed time
		local elapsed = now - last_refill
		if elapsed >= window then
			tokens = limit
			last_refill = now
		end

		if tokens > 0 then
			tokens = tokens - 1
			redis.call("HMSET", key, "tokens", tokens, "last_refill", last_refill)
			redis.call("EXPIRE", key, 3600)
			return {1, tokens, now + window}
		end

		return {0, tokens, now + window}
	`

	now := time.Now()
	result, err := bm.redis.ExecuteLua(ctx, script, []string{redisKey},
		now.Unix(),
		policy.Limit,
		policy.Window.Seconds(),
	)

	if err != nil {
		return false, 0, time.Time{}, fmt.Errorf("redis error: %w", err)
	}

	values := result.([]interface{})
	allowed := values[0].(int64) == 1
	remaining := values[1].(int64)
	resetAt := time.Unix(values[2].(int64), 0)

	return allowed, remaining, resetAt, nil
}

// ConsumeBurst consumes from burst tokens first, then sustained.
func (bm *BurstManager) ConsumeBurst(ctx context.Context, key string, policy *Policy, tokens int64) (bool, int64, error) {
	redisKey := fmt.Sprintf("burst:%s", key)

	script := `
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local tokens_to_consume = tonumber(ARGV[2])
		local burst_limit = tonumber(ARGV[3])
		local capacity = tonumber(ARGV[4])
		local refill_rate = tonumber(ARGV[5])
		local burst_duration = tonumber(ARGV[6])

		local tokens = tonumber(redis.call("HGET", key, "tokens"))
		local last_refill = tonumber(redis.call("HGET", key, "last_refill"))
		local window_start = tonumber(redis.call("HGET", key, "window_start"))

		if not tokens then
			tokens = burst_limit
			last_refill = now
			window_start = now
		end

		-- Refill
		local elapsed = now - last_refill
		if elapsed >= 1 then
			tokens = tokens + (elapsed * refill_rate)
			if tokens > capacity then
				tokens = capacity
			end
			last_refill = now
		end

		-- Check if we can consume
		if tokens >= tokens_to_consume then
			tokens = tokens - tokens_to_consume
			redis.call("HMSET", key, "tokens", tokens, "last_refill", last_refill, "window_start", window_start)
			redis.call("EXPIRE", key, 3600)
			return {1, tokens}
		end

		return {0, tokens}
	`

	refillRate := int64(float64(policy.Limit) / policy.Window.Seconds())
	burstDurationSec := int64(policy.BurstDuration.Seconds())

	result, err := bm.redis.ExecuteLua(ctx, script, []string{redisKey},
		time.Now().Unix(),
		tokens,
		policy.BurstLimit,
		policy.Limit,
		refillRate,
		burstDurationSec,
	)

	if err != nil {
		return false, 0, fmt.Errorf("redis error: %w", err)
	}

	values := result.([]interface{})
	allowed := values[0].(int64) == 1
	remaining := values[1].(int64)

	return allowed, remaining, nil
}

// ReplenishBurst adds tokens back to the burst bucket.
func (bm *BurstManager) ReplenishBurst(ctx context.Context, key string, tokens int64) error {
	redisKey := fmt.Sprintf("burst:%s", key)

	script := `
		local key = KEYS[1]
		local tokens_to_add = tonumber(ARGV[1])
		local capacity = tonumber(ARGV[2])

		local tokens = tonumber(redis.call("HGET", key, "tokens"))
		local capacity_stored = tonumber(redis.call("HGET", key, "capacity"))

		if not tokens then
			return {-1, "bucket not found"}
		end

		if capacity_stored and capacity_stored > 0 then
			capacity = capacity_stored
		end

		tokens = tokens + tokens_to_add
		if tokens > capacity then
			tokens = capacity
		end

		redis.call("HSET", key, "tokens", tokens)
		return {1, tokens}
	`

	_, err := bm.redis.ExecuteLua(ctx, script, []string{redisKey}, tokens, 0)
	return err
}

// ResetBurst resets the burst window for a key.
func (bm *BurstManager) ResetBurst(ctx context.Context, key string, policy *Policy) error {
	redisKey := fmt.Sprintf("burst:%s", key)

	now := time.Now()

	hashData := map[string]string{
		"tokens":         fmt.Sprintf("%d", policy.BurstLimit),
		"capacity":       fmt.Sprintf("%d", policy.Limit),
		"last_refill":    fmt.Sprintf("%d", now.Unix()),
		"window_start":   fmt.Sprintf("%d", now.Unix()),
		"burst_duration": fmt.Sprintf("%d", int64(policy.BurstDuration.Seconds())),
	}

	// Use SetJSON to store the burst state
	return bm.redis.SetJSON(ctx, redisKey, hashData, time.Hour)
}

// SetBurstCapacity dynamically adjusts the burst capacity for a key.
func (bm *BurstManager) SetBurstCapacity(ctx context.Context, key string, capacity int64) error {
	redisKey := fmt.Sprintf("burst:%s", key)
	return bm.redis.SetHash(ctx, redisKey, "capacity", fmt.Sprintf("%d", capacity))
}

// SetSustainedRate dynamically adjusts the sustained rate for a key.
func (bm *BurstManager) SetSustainedRate(ctx context.Context, key string, rate int64) error {
	redisKey := fmt.Sprintf("burst:%s", key)
	// Update refill rate based on new sustained rate
	refillRate := float64(rate) / 60.0 // Assume 1-minute window
	return bm.redis.SetHash(ctx, redisKey, "refill_rate", fmt.Sprintf("%f", refillRate))
}

// EnableBurst enables burst mode for a key.
func (bm *BurstManager) EnableBurst(ctx context.Context, key string, policy *Policy) error {
	return bm.ResetBurst(ctx, key, policy)
}

// DisableBurst disables burst mode and uses sustained rate.
func (bm *BurstManager) DisableBurst(ctx context.Context, key string, policy *Policy) error {
	redisKey := fmt.Sprintf("burst:%s", key)

	now := time.Now()
	hashData := map[string]string{
		"tokens":         fmt.Sprintf("%d", policy.Limit),
		"capacity":       fmt.Sprintf("%d", policy.Limit),
		"last_refill":    fmt.Sprintf("%d", now.Unix()),
		"window_start":   fmt.Sprintf("%d", now.Unix()),
		"burst_duration": "0",
	}

	return bm.redis.SetJSON(ctx, redisKey, hashData, time.Hour)
}

// EstimateTokensAvailable estimates available tokens at a future time.
func (bm *BurstManager) EstimateTokensAvailable(ctx context.Context, key string, at time.Time) (int64, error) {
	redisKey := fmt.Sprintf("burst:%s", key)

	hash, err := bm.redis.GetAllHash(ctx, redisKey)
	if err != nil {
		return 0, fmt.Errorf("redis error: %w", err)
	}

	var tokens, lastRefillUnix, refillRate int64
	fmt.Sscanf(hash["tokens"], "%d", &tokens)
	fmt.Sscanf(hash["last_refill"], "%d", &lastRefillUnix)
	fmt.Sscanf(hash["refill_rate"], "%d", &refillRate)

	elapsed := at.Unix() - lastRefillUnix
	tokensToAdd := elapsed * refillRate
	projectedTokens := tokens + tokensToAdd

	capacity := int64(0)
	if capStr, ok := hash["capacity"]; ok {
		fmt.Sscanf(capStr, "%d", &capacity)
	}
	if capacity > 0 && projectedTokens > capacity {
		return capacity, nil
	}

	return projectedTokens, nil
}

// GetStats returns statistics about burst usage.
func (bm *BurstManager) GetStats(ctx context.Context) (*BurstStats, error) {
	pattern := "burst:*"
	keys, err := bm.redis.ListKeys(ctx, pattern)
	if err != nil {
		return nil, fmt.Errorf("redis error: %w", err)
	}

	stats := &BurstStats{
		TotalBuckets: len(keys),
	}

	now := time.Now()
	for _, key := range keys {
		hash, err := bm.redis.GetAllHash(ctx, key)
		if err != nil {
			continue
		}

		var windowStart, burstDuration int64
		fmt.Sscanf(hash["window_start"], "%d", &windowStart)
		fmt.Sscanf(hash["burst_duration"], "%d", &burstDuration)

		if burstDuration > 0 {
			windowStartTime := time.Unix(windowStart, 0)
			burstEnd := windowStartTime.Add(time.Duration(burstDuration) * time.Second)
			if now.Before(burstEnd) {
				stats.BucketsInBurst++
			}
		}

		var tokens int64
		fmt.Sscanf(hash["tokens"], "%d", &tokens)
		stats.TotalTokens += tokens
	}

	return stats, nil
}

// BurstStats represents burst manager statistics.
type BurstStats struct {
	TotalBuckets   int   `json:"total_buckets"`
	BucketsInBurst int   `json:"buckets_in_burst"`
	TotalTokens    int64 `json:"total_tokens"`
}
