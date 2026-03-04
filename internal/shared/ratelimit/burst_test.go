package ratelimit

import (
	"context"
	"testing"
	"time"
)

// TestBurstManagerCheckBurst tests the burst checking functionality.
func TestBurstManagerCheckBurst(t *testing.T) {
	// Create a mock redis client for testing
	// In a real test, you would use a testcontainers Redis instance
	t.Skip("Requires Redis instance - use testcontainers for integration testing")

	ctx := context.Background()
	redisClient, err := NewRedisClient(&RedisConfig{
		Addr: "localhost:6379",
	})
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer redisClient.Close()

	bm := NewBurstManager(redisClient)

	policy := &Policy{
		Limit:         100,
		BurstLimit:    200,
		BurstDuration: time.Minute,
		Window:        time.Minute,
	}

	// First request should be allowed
	allowed, remaining, burstRemaining, err := bm.CheckBurst(ctx, "test-key-1", policy)
	if err != nil {
		t.Fatalf("CheckBurst failed: %v", err)
	}
	if !allowed {
		t.Error("First request should be allowed")
	}
	if remaining < 0 {
		t.Errorf("Invalid remaining tokens: %d", remaining)
	}
	if burstRemaining <= 0 {
		t.Error("Should be in burst period")
	}
}

// TestBurstManagerGetBurstStatus tests getting burst status.
func TestBurstManagerGetBurstStatus(t *testing.T) {
	t.Skip("Requires Redis instance - use testcontainers for integration testing")

	ctx := context.Background()
	redisClient, err := NewRedisClient(&RedisConfig{
		Addr: "localhost:6379",
	})
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer redisClient.Close()

	bm := NewBurstManager(redisClient)

	policy := &Policy{
		Limit:         100,
		BurstLimit:    200,
		BurstDuration: time.Minute,
		Window:        time.Minute,
	}

	status := bm.GetBurstStatus(ctx, "test-key-status", policy)
	if status == nil {
		t.Fatal("GetBurstStatus should never return nil")
	}
	if status.Key != "test-key-status" {
		t.Errorf("Expected key 'test-key-status', got '%s'", status.Key)
	}
}

// TestBurstManagerResetBurst tests resetting burst state.
func TestBurstManagerResetBurst(t *testing.T) {
	t.Skip("Requires Redis instance - use testcontainers for integration testing")

	ctx := context.Background()
	redisClient, err := NewRedisClient(&RedisConfig{
		Addr: "localhost:6379",
	})
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer redisClient.Close()

	bm := NewBurstManager(redisClient)

	policy := &Policy{
		Limit:         100,
		BurstLimit:    200,
		BurstDuration: time.Minute,
		Window:        time.Minute,
	}

	err = bm.ResetBurst(ctx, "test-key-reset", policy)
	if err != nil {
		t.Fatalf("ResetBurst failed: %v", err)
	}

	// Verify burst is active
	status := bm.GetBurstStatus(ctx, "test-key-reset", policy)
	if !status.InBurstPeriod {
		t.Error("Should be in burst period after reset")
	}
}

// TestBurstManagerConsumeBurst tests consuming multiple burst tokens.
func TestBurstManagerConsumeBurst(t *testing.T) {
	t.Skip("Requires Redis instance - use testcontainers for integration testing")

	ctx := context.Background()
	redisClient, err := NewRedisClient(&RedisConfig{
		Addr: "localhost:6379",
	})
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer redisClient.Close()

	bm := NewBurstManager(redisClient)

	policy := &Policy{
		Limit:         100,
		BurstLimit:    200,
		BurstDuration: time.Minute,
		Window:        time.Minute,
	}

	// Reset first to initialize
	_ = bm.ResetBurst(ctx, "test-key-consume", policy)

	// Consume 10 tokens
	allowed, remaining, err := bm.ConsumeBurst(ctx, "test-key-consume", policy, 10)
	if err != nil {
		t.Fatalf("ConsumeBurst failed: %v", err)
	}
	if !allowed {
		t.Error("Should have enough tokens for 10")
	}
	if remaining < 0 {
		t.Errorf("Invalid remaining: %d", remaining)
	}

	// Try to consume more than available
	allowed, _, _ = bm.ConsumeBurst(ctx, "test-key-consume", policy, 1000)
	if allowed {
		t.Error("Should not have enough tokens for 1000")
	}
}

// TestBurstManagerReplenishBurst tests replenishing burst tokens.
func TestBurstManagerReplenishBurst(t *testing.T) {
	t.Skip("Requires Redis instance - use testcontainers for integration testing")

	ctx := context.Background()
	redisClient, err := NewRedisClient(&RedisConfig{
		Addr: "localhost:6379",
	})
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer redisClient.Close()

	bm := NewBurstManager(redisClient)

	policy := &Policy{
		Limit:         100,
		BurstLimit:    200,
		BurstDuration: time.Minute,
		Window:        time.Minute,
	}

	// Reset first to initialize
	_ = bm.ResetBurst(ctx, "test-key-replenish", policy)

	// Consume tokens
	_, remaining, _ := bm.ConsumeBurst(ctx, "test-key-replenish", policy, 50)

	// Replenish
	err = bm.ReplenishBurst(ctx, "test-key-replenish", 25)
	if err != nil {
		t.Fatalf("ReplenishBurst failed: %v", err)
	}

	// Check status
	status := bm.GetBurstStatus(ctx, "test-key-replenish", policy)
	if status.BurstTokens < remaining {
		t.Errorf("Tokens not replenished: got %d, want at least %d", status.BurstTokens, remaining)
	}
}

// TestBurstManagerSetBurstCapacity tests setting burst capacity.
func TestBurstManagerSetBurstCapacity(t *testing.T) {
	t.Skip("Requires Redis instance - use testcontainers for integration testing")

	ctx := context.Background()
	redisClient, err := NewRedisClient(&RedisConfig{
		Addr: "localhost:6379",
	})
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer redisClient.Close()

	bm := NewBurstManager(redisClient)

	err = bm.SetBurstCapacity(ctx, "test-key-capacity", 500)
	if err != nil {
		t.Fatalf("SetBurstCapacity failed: %v", err)
	}
}

// TestBurstManagerEnableDisableBurst tests enabling/disabling burst mode.
func TestBurstManagerEnableDisableBurst(t *testing.T) {
	t.Skip("Requires Redis instance - use testcontainers for integration testing")

	ctx := context.Background()
	redisClient, err := NewRedisClient(&RedisConfig{
		Addr: "localhost:6379",
	})
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer redisClient.Close()

	bm := NewBurstManager(redisClient)

	policy := &Policy{
		Limit:         100,
		BurstLimit:    200,
		BurstDuration: time.Minute,
		Window:        time.Minute,
	}

	// Enable burst
	err = bm.EnableBurst(ctx, "test-key-enable", policy)
	if err != nil {
		t.Fatalf("EnableBurst failed: %v", err)
	}

	status := bm.GetBurstStatus(ctx, "test-key-enable", policy)
	if !status.InBurstPeriod {
		t.Error("Should be in burst period after enable")
	}

	// Disable burst
	err = bm.DisableBurst(ctx, "test-key-enable", policy)
	if err != nil {
		t.Fatalf("DisableBurst failed: %v", err)
	}

	status = bm.GetBurstStatus(ctx, "test-key-enable", policy)
	if status.InBurstPeriod {
		t.Error("Should not be in burst period after disable")
	}
}

// TestBurstManagerGetStats tests getting burst statistics.
func TestBurstManagerGetStats(t *testing.T) {
	t.Skip("Requires Redis instance - use testcontainers for integration testing")

	ctx := context.Background()
	redisClient, err := NewRedisClient(&RedisConfig{
		Addr: "localhost:6379",
	})
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer redisClient.Close()

	bm := NewBurstManager(redisClient)

	stats, err := bm.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats == nil {
		t.Fatal("GetStats should never return nil")
	}
	if stats.TotalBuckets < 0 {
		t.Errorf("Invalid TotalBuckets: %d", stats.TotalBuckets)
	}
	if stats.BucketsInBurst < 0 {
		t.Errorf("Invalid BucketsInBurst: %d", stats.BucketsInBurst)
	}
	if stats.TotalTokens < 0 {
		t.Errorf("Invalid TotalTokens: %d", stats.TotalTokens)
	}
}
