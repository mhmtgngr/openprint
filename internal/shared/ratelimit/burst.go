package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// BurstManager handles burst vs sustained rate limiting.
// Burst allows temporary higher request rates before settling to sustained limits.
type BurstManager struct {
	redis *RedisClient
	mu    sync.RWMutex

	// In-memory burst token buckets for fast access
	burstBuckets map[string]*burstBucket
}

// burstBucket tracks burst tokens for an identifier.
type burstBucket struct {
	tokens        int64
	capacity      int64
	lastRefill    time.Time
	refillRate    int64         // tokens per second
	sustainedRate int64         // sustained rate after burst
	burstDuration time.Duration // how long burst lasts
	windowStart   time.Time     // when burst window started
	mu            sync.Mutex
}

// NewBurstManager creates a new burst manager.
func NewBurstManager(redis *RedisClient) *BurstManager {
	bm := &BurstManager{
		redis:        redis,
		burstBuckets: make(map[string]*burstBucket),
	}

	// Start cleanup goroutine
	go bm.cleanup()

	return bm
}

// CheckBurst checks if a burst request should be allowed.
// Returns (allowed, remainingTokens, burstRemainingTime).
func (bm *BurstManager) CheckBurst(ctx context.Context, key string, policy *Policy) (bool, int64, time.Duration, error) {
	// Get or create burst bucket
	bucket := bm.getBucket(key, policy)

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	now := time.Now()

	// Refill tokens based on time elapsed
	bucket.refill(now)

	// Check if in burst period
	if bucket.inBurstPeriod(now) {
		// Check burst tokens
		if bucket.tokens > 0 {
			bucket.tokens--
			return true, bucket.tokens, time.Until(bucket.windowStart.Add(bucket.burstDuration)), nil
		}
		// Burst exhausted, check sustained rate
		if bucket.sustainedRate > 0 && bucket.tokens >= 0 {
			return false, bucket.tokens, time.Until(bucket.windowStart.Add(bucket.burstDuration)), nil
		}
		return false, bucket.tokens, time.Until(bucket.windowStart.Add(bucket.burstDuration)), nil
	}

	// Not in burst period, use sustained rate
	if bucket.tokens > 0 {
		bucket.tokens--
		return true, bucket.tokens, 0, nil
	}

	return false, bucket.tokens, 0, nil
}

// getBucket gets or creates a burst bucket for a key.
func (bm *BurstManager) getBucket(key string, policy *Policy) *burstBucket {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if bucket, ok := bm.burstBuckets[key]; ok {
		return bucket
	}

	bucket := &burstBucket{
		capacity:      policy.Limit,
		tokens:        policy.BurstLimit,
		lastRefill:    time.Now(),
		refillRate:    int64(float64(policy.Limit) / policy.Window.Seconds()),
		sustainedRate: policy.Limit,
		burstDuration: policy.BurstDuration,
		windowStart:   time.Now(),
	}

	bm.burstBuckets[key] = bucket
	return bucket
}

// refill refills tokens based on elapsed time.
func (b *burstBucket) refill(now time.Time) {
	elapsed := now.Sub(b.lastRefill)
	if elapsed < time.Second {
		return
	}

	// Calculate tokens to add
	tokensToAdd := int64(elapsed.Seconds()) * b.refillRate

	b.tokens += tokensToAdd
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}

	b.lastRefill = now
}

// inBurstPeriod checks if currently in the burst period.
func (b *burstBucket) inBurstPeriod(now time.Time) bool {
	if b.burstDuration == 0 {
		return false
	}
	return now.Before(b.windowStart.Add(b.burstDuration))
}

// ResetBurst resets the burst window for a key.
func (bm *BurstManager) ResetBurst(key string, policy *Policy) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	bucket := &burstBucket{
		capacity:      policy.Limit,
		tokens:        policy.BurstLimit,
		lastRefill:    time.Now(),
		refillRate:    int64(float64(policy.Limit) / policy.Window.Seconds()),
		sustainedRate: policy.Limit,
		burstDuration: policy.BurstDuration,
		windowStart:   time.Now(),
	}

	bm.burstBuckets[key] = bucket
}

// cleanup removes expired burst buckets.
func (bm *BurstManager) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		bm.mu.Lock()
		for key, bucket := range bm.burstBuckets {
			bucket.mu.Lock()
			// Remove if inactive for 1 hour
			if time.Since(bucket.lastRefill) > time.Hour {
				delete(bm.burstBuckets, key)
			}
			bucket.mu.Unlock()
		}
		bm.mu.Unlock()
	}
}

// GetBurstStatus returns the current burst status for a key.
func (bm *BurstManager) GetBurstStatus(key string, policy *Policy) *BurstStatus {
	bucket := bm.getBucket(key, policy)

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	now := time.Now()
	bucket.refill(now)

	status := &BurstStatus{
		Key:               key,
		BurstTokens:       bucket.tokens,
		BurstCapacity:     bucket.capacity,
		SustainedRate:     bucket.sustainedRate,
		InBurstPeriod:     bucket.inBurstPeriod(now),
		LastRefill:        bucket.lastRefill,
	}

	if bucket.burstDuration > 0 {
		burstEnd := bucket.windowStart.Add(bucket.burstDuration)
		status.BurstRemaining = time.Until(burstEnd)
		if status.BurstRemaining < 0 {
			status.BurstRemaining = 0
		}
	}

	return status
}

// BurstStatus represents the current burst status.
type BurstStatus struct {
	Key               string        `json:"key"`
	BurstTokens       int64         `json:"burst_tokens"`
	BurstCapacity     int64         `json:"burst_capacity"`
	SustainedRate     int64         `json:"sustained_rate"`
	InBurstPeriod     bool          `json:"in_burst_period"`
	BurstRemaining    time.Duration `json:"burst_remaining"`
	LastRefill        time.Time     `json:"last_refill"`
}

// CheckSustained checks if the sustained rate allows a request.
func (bm *BurstManager) CheckSustained(ctx context.Context, key string, policy *Policy) (bool, int64, time.Time, error) {
	bucket := bm.getBucket(key, policy)

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	now := time.Now()
	bucket.refill(now)

	if bucket.tokens > 0 {
		bucket.tokens--
		resetAt := now.Add(policy.Window)
		return true, bucket.tokens, resetAt, nil
	}

	resetAt := now.Add(policy.Window)
	return false, bucket.tokens, resetAt, nil
}

// ConsumeBurst consumes from burst tokens first, then sustained.
func (bm *BurstManager) ConsumeBurst(ctx context.Context, key string, policy *Policy, tokens int64) (bool, int64, error) {
	bucket := bm.getBucket(key, policy)

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	now := time.Now()
	bucket.refill(now)

	// First try to consume from burst
	if bucket.inBurstPeriod(now) && bucket.tokens >= tokens {
		bucket.tokens -= tokens
		return true, bucket.tokens, nil
	}

	// Burst exhausted or not in period, check sustained
	if bucket.tokens >= tokens {
		bucket.tokens -= tokens
		return true, bucket.tokens, nil
	}

	return false, bucket.tokens, fmt.Errorf("insufficient tokens")
}

// ReplenishBurst adds tokens back to the burst bucket.
func (bm *BurstManager) ReplenishBurst(ctx context.Context, key string, tokens int64) error {
	bm.mu.RLock()
	bucket, ok := bm.burstBuckets[key]
	bm.mu.RUnlock()

	if !ok {
		return fmt.Errorf("bucket not found for key: %s", key)
	}

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	bucket.tokens += tokens
	if bucket.tokens > bucket.capacity {
		bucket.tokens = bucket.capacity
	}

	return nil
}

// GetStats returns statistics about burst usage.
func (bm *BurstManager) GetStats() *BurstStats {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	totalBuckets := len(bm.burstBuckets)
	inBurst := 0
	totalTokens := int64(0)

	for _, bucket := range bm.burstBuckets {
		bucket.mu.Lock()
		if bucket.inBurstPeriod(time.Now()) {
			inBurst++
		}
		totalTokens += bucket.tokens
		bucket.mu.Unlock()
	}

	return &BurstStats{
		TotalBuckets:   totalBuckets,
		BucketsInBurst: inBurst,
		TotalTokens:    totalTokens,
	}
}

// BurstStats represents burst manager statistics.
type BurstStats struct {
	TotalBuckets   int   `json:"total_buckets"`
	BucketsInBurst int   `json:"buckets_in_burst"`
	TotalTokens    int64 `json:"total_tokens"`
}

// SetBurstCapacity dynamically adjusts the burst capacity for a key.
func (bm *BurstManager) SetBurstCapacity(key string, capacity int64) {
	bm.mu.RLock()
	bucket, ok := bm.burstBuckets[key]
	bm.mu.RUnlock()

	if !ok {
		return
	}

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	bucket.capacity = capacity
	if bucket.tokens > capacity {
		bucket.tokens = capacity
	}
}

// SetSustainedRate dynamically adjusts the sustained rate for a key.
func (bm *BurstManager) SetSustainedRate(key string, rate int64) {
	bm.mu.RLock()
	bucket, ok := bm.burstBuckets[key]
	bm.mu.RUnlock()

	if !ok {
		return
	}

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	bucket.sustainedRate = rate
	bucket.refillRate = int64(float64(rate) / 60) // Assume 1-minute window for refill
}

// EnableBurst enables burst mode for a key.
func (bm *BurstManager) EnableBurst(key string, policy *Policy) {
	bm.ResetBurst(key, policy)
}

// DisableBurst disables burst mode and uses sustained rate.
func (bm *BurstManager) DisableBurst(key string, policy *Policy) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	bucket := &burstBucket{
		capacity:      policy.Limit,
		tokens:        policy.Limit,
		lastRefill:    time.Now(),
		refillRate:    int64(float64(policy.Limit) / policy.Window.Seconds()),
		sustainedRate: policy.Limit,
		burstDuration: 0,
		windowStart:   time.Now(),
	}

	bm.burstBuckets[key] = bucket
}

// EstimateTokensAvailable estimates available tokens at a future time.
func (bm *BurstManager) EstimateTokensAvailable(key string, at time.Time) int64 {
	bm.mu.RLock()
	bucket, ok := bm.burstBuckets[key]
	bm.mu.RUnlock()

	if !ok {
		return 0
	}

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	elapsed := at.Sub(bucket.lastRefill)
	tokensToAdd := int64(elapsed.Seconds()) * bucket.refillRate
	projectedTokens := bucket.tokens + tokensToAdd

	if projectedTokens > bucket.capacity {
		return bucket.capacity
	}

	return projectedTokens
}
