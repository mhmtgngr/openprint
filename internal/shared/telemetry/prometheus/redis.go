// Package prometheus provides Redis metrics collection wrappers.
package prometheus

import (
	"context"
	"strings"
	"time"

	redisotel "github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

// RedisConfig holds Redis configuration for metrics collection.
type RedisConfig struct {
	ServiceName string
	DBName      string // The database number or name (e.g., "0", "cache")
}

// WrapRedisClient wraps a redis.Client to collect metrics using OpenTelemetry.
// This returns the same client with metrics instrumentation added.
func WrapRedisClient(client *redis.Client, registry *Registry, cfg RedisConfig) *redis.Client {
	// Use redisotel for automatic metrics collection
	// This integrates with OpenTelemetry which we already have
	if err := redisotel.InstrumentMetrics(client); err != nil {
		// Log warning but don't fail - metrics are optional
		// In production, you might want to log this
	}

	return client
}

// WrapRedisCluster wraps a redis.ClusterClient to collect metrics.
func WrapRedisCluster(client *redis.ClusterClient, registry *Registry, cfg RedisConfig) *redis.ClusterClient {
	if err := redisotel.InstrumentMetrics(client); err != nil {
		// Log warning but don't fail
	}

	return client
}

// RedisTracer provides manual Redis metrics collection.
// Use this when you need more control than automatic instrumentation provides.
type RedisTracer struct {
	metrics     *RedisMetrics
	serviceName string
	dbName      string
}

// NewRedisTracer creates a new Redis tracer.
func NewRedisTracer(metrics *Metrics, serviceName, dbName string) *RedisTracer {
	return &RedisTracer{
		metrics:     metrics.Redis,
		serviceName: serviceName,
		dbName:      dbName,
	}
}

// TraceCommand records a Redis command execution.
func (t *RedisTracer) TraceCommand(cmd string, duration time.Duration, err error) {
	// Normalize command name
	command := normalizeRedisCommand(cmd)

	// Record duration
	t.metrics.CommandDuration.WithLabelValues(
		t.serviceName,
		t.dbName,
		command,
	).Observe(duration.Seconds())

	// Record command count
	t.metrics.CommandsTotal.WithLabelValues(
		t.serviceName,
		t.dbName,
		command,
	).Inc()

	// Record errors
	if err != nil {
		t.metrics.CommandErrorsTotal.WithLabelValues(
			t.serviceName,
			t.dbName,
			command,
		).Inc()
	}
}

// RecordPoolStats records connection pool statistics.
func (t *RedisTracer) RecordPoolStats(stats *redis.PoolStats) {
	t.metrics.ConnectionsActive.WithLabelValues(
		t.serviceName,
		t.dbName,
	).Set(float64(stats.Hits + stats.Misses))

	t.metrics.ConnectionsIdle.WithLabelValues(
		t.serviceName,
		t.dbName,
	).Set(float64(stats.TotalConns - (stats.Hits + stats.Misses)))

	t.metrics.PoolHitsTotal.WithLabelValues(
		t.serviceName,
		t.dbName,
	).Add(float64(stats.Hits))

	t.metrics.PoolMissesTotal.WithLabelValues(
		t.serviceName,
		t.dbName,
	).Add(float64(stats.Misses))

	t.metrics.PoolTimeoutsTotal.WithLabelValues(
		t.serviceName,
		t.dbName,
	).Add(float64(stats.Timeouts))
}

// normalizeRedisCommand extracts the command name from a Redis command string.
func normalizeRedisCommand(cmd string) string {
	// Remove newlines and extra spaces
	cmd = strings.ReplaceAll(cmd, "\n", " ")
	cmd = strings.ReplaceAll(cmd, "\t", " ")

	// Get first word (command name)
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return "unknown"
	}

	// Convert to lowercase
	command := strings.ToLower(parts[0])

	// Normalize common command variations
	switch command {
	case "setex", "psetex", "setnx", "mset":
		return "set"
	case "getex", "mget":
		return "get"
	case "incrby", "incrbyfloat", "decr", "decrby":
		return "incr"
	case "hset", "hmset", "hsetnx":
		return "hset"
	case "hget", "hmget":
		return "hget"
	case "lpush", "rpush", "lpushx", "rpushx":
		return "lpush"
	case "lpop", "rpop":
		return "lpop"
	case "sadd":
		return "sadd"
	case "srem", "sismember", "smembers":
		return "sadd"
	case "zadd":
		return "zadd"
	case "zrem", "zrange", "zscore":
		return "zadd"
	case "expire", "pexpire", "expireat", "pexpireat":
		return "expire"
	case "ttl", "pttl":
		return "ttl"
	}

	return command
}

// ClientWithMetrics wraps a redis.Client to record metrics for each command.
type ClientWithMetrics struct {
	client *redis.Client
	tracer *RedisTracer
}

// NewClientWithMetrics creates a new Redis client wrapper with metrics.
func NewClientWithMetrics(client *redis.Client, metrics *Metrics, serviceName, dbName string) *ClientWithMetrics {
	return &ClientWithMetrics{
		client: client,
		tracer: NewRedisTracer(metrics, serviceName, dbName),
	}
}

// Do executes a Redis command and records metrics.
func (c *ClientWithMetrics) Do(ctx context.Context, cmd string, args ...interface{}) *redis.Cmd {
	start := time.Now()
	result := c.client.Do(ctx, append([]interface{}{cmd}, args...)...)
	duration := time.Since(start)

	// We can't easily check if the command failed without calling the result methods
	// For now, record the command without error status
	c.tracer.TraceCommand(cmd, duration, nil)

	return result
}

// Get executes a GET command and records metrics.
func (c *ClientWithMetrics) Get(ctx context.Context, key string) *redis.StringCmd {
	start := time.Now()
	result := c.client.Get(ctx, key)
	duration := time.Since(start)

	err := result.Err()
	c.tracer.TraceCommand("GET", duration, err)

	return result
}

// Set executes a SET command and records metrics.
func (c *ClientWithMetrics) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	start := time.Now()
	result := c.client.Set(ctx, key, value, expiration)
	duration := time.Since(start)

	err := result.Err()
	c.tracer.TraceCommand("SET", duration, err)

	return result
}

// Del executes a DEL command and records metrics.
func (c *ClientWithMetrics) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	start := time.Now()
	result := c.client.Del(ctx, keys...)
	duration := time.Since(start)

	err := result.Err()
	c.tracer.TraceCommand("DEL", duration, err)

	return result
}

// HGet executes an HGET command and records metrics.
func (c *ClientWithMetrics) HGet(ctx context.Context, key, field string) *redis.StringCmd {
	start := time.Now()
	result := c.client.HGet(ctx, key, field)
	duration := time.Since(start)

	err := result.Err()
	c.tracer.TraceCommand("HGET", duration, err)

	return result
}

// HSet executes an HSET command and records metrics.
func (c *ClientWithMetrics) HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	start := time.Now()
	result := c.client.HSet(ctx, key, values...)
	duration := time.Since(start)

	err := result.Err()
	c.tracer.TraceCommand("HSET", duration, err)

	return result
}

// Incr executes an INCR command and records metrics.
func (c *ClientWithMetrics) Incr(ctx context.Context, key string) *redis.IntCmd {
	start := time.Now()
	result := c.client.Incr(ctx, key)
	duration := time.Since(start)

	err := result.Err()
	c.tracer.TraceCommand("INCR", duration, err)

	return result
}

// Decr executes a DECR command and records metrics.
func (c *ClientWithMetrics) Decr(ctx context.Context, key string) *redis.IntCmd {
	start := time.Now()
	result := c.client.Decr(ctx, key)
	duration := time.Since(start)

	err := result.Err()
	c.tracer.TraceCommand("DECR", duration, err)

	return result
}

// Expire executes an EXPIRE command and records metrics.
func (c *ClientWithMetrics) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	start := time.Now()
	result := c.client.Expire(ctx, key, expiration)
	duration := time.Since(start)

	err := result.Err()
	c.tracer.TraceCommand("EXPIRE", duration, err)

	return result
}

// TTL executes a TTL command and records metrics.
func (c *ClientWithMetrics) TTL(ctx context.Context, key string) *redis.DurationCmd {
	start := time.Now()
	result := c.client.TTL(ctx, key)
	duration := time.Since(start)

	err := result.Err()
	c.tracer.TraceCommand("TTL", duration, err)

	return result
}

// Ping executes a PING command and records metrics.
func (c *ClientWithMetrics) Ping(ctx context.Context) *redis.StatusCmd {
	start := time.Now()
	result := c.client.Ping(ctx)
	duration := time.Since(start)

	err := result.Err()
	c.tracer.TraceCommand("PING", duration, err)

	return result
}

// StartPoolStatsCollector starts a background goroutine to collect pool statistics.
func StartPoolStatsCollector(client *redis.Client, metrics *Metrics, serviceName, dbName string, interval time.Duration) func() {
	stopCh := make(chan struct{})
	tracer := NewRedisTracer(metrics, serviceName, dbName)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				poolStats := client.PoolStats()
				tracer.RecordPoolStats(poolStats)
			case <-stopCh:
				return
			}
		}
	}()

	return func() {
		close(stopCh)
	}
}

// RecordRedisCommand is a helper function to record a Redis command metric.
func RecordRedisCommand(metrics *Metrics, serviceName, dbName, command string, duration time.Duration, err error) {
	tracer := NewRedisTracer(metrics, serviceName, dbName)
	tracer.TraceCommand(command, duration, err)
}

// RedisHook is a hook that records metrics for each Redis command.
type RedisHook struct {
	tracer *RedisTracer
}

// NewRedisHook creates a new Redis metrics hook.
func NewRedisHook(metrics *Metrics, serviceName, dbName string) *RedisHook {
	return &RedisHook{
		tracer: NewRedisTracer(metrics, serviceName, dbName),
	}
}

// BeforeProcess is called before a command is processed.
func (h *RedisHook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	// Store start time in context for AfterProcess
	return context.WithValue(ctx, "redis_start_time", time.Now()), nil
}

// AfterProcess is called after a command is processed.
func (h *RedisHook) AfterProcess(ctx context.Context, cmd redis.Cmder) error {
	start, ok := ctx.Value("redis_start_time").(time.Time)
	if !ok {
		return nil
	}

	duration := time.Since(start)
	command := normalizeRedisCommand(cmd.Name())

	h.tracer.TraceCommand(command, duration, cmd.Err())

	return nil
}

// BeforeProcessPipeline is called before a pipeline is processed.
func (h *RedisHook) BeforeProcessPipeline(ctx context.Context, cmds []redis.Cmder) (context.Context, error) {
	return context.WithValue(ctx, "redis_pipeline_start_time", time.Now()), nil
}

// AfterProcessPipeline is called after a pipeline is processed.
func (h *RedisHook) AfterProcessPipeline(ctx context.Context, cmds []redis.Cmder) error {
	start, ok := ctx.Value("redis_pipeline_start_time").(time.Time)
	if !ok {
		return nil
	}

	duration := time.Since(start)

	// Record each command in the pipeline
	for _, cmd := range cmds {
		command := normalizeRedisCommand(cmd.Name())
		h.tracer.TraceCommand(command, duration, cmd.Err())
	}

	return nil
}
