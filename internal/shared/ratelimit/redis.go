package ratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient wraps the Redis client with rate limit specific operations.
type RedisClient struct {
	client *redis.Client
	config *RedisConfig
}

// RedisConfig holds Redis connection configuration.
type RedisConfig struct {
	Addr         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	MaxRetries   int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolTimeout  time.Duration
}

// NewRedisClient creates a new Redis client for rate limiting operations.
func NewRedisClient(cfg *RedisConfig) (*RedisClient, error) {
	if cfg == nil {
		cfg = defaultRedisConfig()
	}

	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxRetries:   cfg.MaxRetries,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolTimeout:  cfg.PoolTimeout,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisClient{
		client: client,
		config: cfg,
	}, nil
}

// defaultRedisConfig returns default Redis configuration.
func defaultRedisConfig() *RedisConfig {
	return &RedisConfig{
		Addr:         "localhost:6379",
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
	}
}

// AddToWindow adds a request timestamp to the sliding window.
func (r *RedisClient) AddToWindow(ctx context.Context, key string, timestamp int64, score float64) error {
	return r.client.ZAdd(ctx, key, redis.Z{
		Score:  score,
		Member: timestamp,
	}).Err()
}

// CountWindow counts requests within the time window.
func (r *RedisClient) CountWindow(ctx context.Context, key string, min, max float64) (int64, error) {
	return r.client.ZCount(ctx, key, fmt.Sprintf("%f", min), fmt.Sprintf("%f", max)).Result()
}

// RemoveOldWindow removes entries outside the time window.
func (r *RedisClient) RemoveOldWindow(ctx context.Context, key string, max float64) error {
	return r.client.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%f", max)).Err()
}

// GetWindowStats returns statistics about the current window.
func (r *RedisClient) GetWindowStats(ctx context.Context, key string, min, max float64) (*WindowStats, error) {
	// Get count and total score
	count, err := r.client.ZCount(ctx, key, fmt.Sprintf("%f", min), fmt.Sprintf("%f", max)).Result()
	if err != nil {
		return nil, err
	}

	// Get oldest and newest timestamps
	oldest, err := r.client.ZRange(ctx, key, 0, 0).Result()
	if err != nil {
		return nil, err
	}

	newest, err := r.client.ZRevRange(ctx, key, 0, 0).Result()
	if err != nil {
		return nil, err
	}

	stats := &WindowStats{
		Count: int(count),
	}

	if len(oldest) > 0 {
		stats.OldestTimestamp = oldest[0]
	}
	if len(newest) > 0 {
		stats.NewestTimestamp = newest[0]
	}

	return stats, nil
}

// WindowStats represents statistics about a time window.
type WindowStats struct {
	Count            int    `json:"count"`
	OldestTimestamp  string `json:"oldest_timestamp,omitempty"`
	NewestTimestamp  string `json:"newest_timestamp,omitempty"`
}

// SetKeyExpiration sets an expiration time for a key.
func (r *RedisClient) SetKeyExpiration(ctx context.Context, key string, expiration time.Duration) error {
	return r.client.Expire(ctx, key, expiration).Err()
}

// DeleteKey removes a key from Redis.
func (r *RedisClient) DeleteKey(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// IncrementCounter increments a counter value.
func (r *RedisClient) IncrementCounter(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

// GetCounter retrieves a counter value.
func (r *RedisClient) GetCounter(ctx context.Context, key string) (int64, error) {
	return r.client.Get(ctx, key).Int64()
}

// GetJSON retrieves and parses a JSON value.
func (r *RedisClient) GetJSON(ctx context.Context, key string, dest interface{}) error {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), dest)
}

// SetJSON stores a JSON value.
func (r *RedisClient) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

// GetSet retrieves and updates a set.
func (r *RedisClient) GetSet(ctx context.Context, key string) ([]string, error) {
	return r.client.SMembers(ctx, key).Result()
}

// AddToSet adds a member to a set.
func (r *RedisClient) AddToSet(ctx context.Context, key, member string) error {
	return r.client.SAdd(ctx, key, member).Err()
}

// RemoveFromSet removes a member from a set.
func (r *RedisClient) RemoveFromSet(ctx context.Context, key, member string) error {
	return r.client.SRem(ctx, key, member).Err()
}

// SetMemberExists checks if a member exists in a set.
func (r *RedisClient) SetMemberExists(ctx context.Context, key, member string) (bool, error) {
	return r.client.SIsMember(ctx, key, member).Result()
}

// ListKeys lists keys matching a pattern.
func (r *RedisClient) ListKeys(ctx context.Context, pattern string) ([]string, error) {
	return r.client.Keys(ctx, pattern).Result()
}

// ExecuteLua executes a Lua script.
func (r *RedisClient) ExecuteLua(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	return r.client.Eval(ctx, script, keys, args...).Result()
}

// GetHash retrieves a hash field.
func (r *RedisClient) GetHash(ctx context.Context, key, field string) (string, error) {
	return r.client.HGet(ctx, key, field).Result()
}

// SetHash sets a hash field.
func (r *RedisClient) SetHash(ctx context.Context, key, field, value string) error {
	return r.client.HSet(ctx, key, field, value).Err()
}

// GetAllHash retrieves all hash fields.
func (r *RedisClient) GetAllHash(ctx context.Context, key string) (map[string]string, error) {
	return r.client.HGetAll(ctx, key).Result()
}

// Pipeline returns a Redis pipeline for batched operations.
func (r *RedisClient) Pipeline() redis.Pipeliner {
	return r.client.Pipeline()
}

// Close closes the Redis client connection.
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// Ping checks if the Redis server is responsive.
func (r *RedisClient) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// FlushDB clears the current database (use with caution).
func (r *RedisClient) FlushDB(ctx context.Context) error {
	return r.client.FlushDB(ctx).Err()
}

// GetClient returns the underlying Redis client.
func (r *RedisClient) GetClient() *redis.Client {
	return r.client
}

// ClusterRedisClient wraps Redis Cluster client for distributed rate limiting.
type ClusterRedisClient struct {
	client *redis.ClusterClient
}

// NewClusterRedisClient creates a new Redis Cluster client.
func NewClusterRedisClient(addrs []string, password string) (*ClusterRedisClient, error) {
	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    addrs,
		Password: password,
		PoolSize: 10,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis cluster: %w", err)
	}

	return &ClusterRedisClient{client: client}, nil
}

// Close closes the cluster client.
func (c *ClusterRedisClient) Close() error {
	return c.client.Close()
}
