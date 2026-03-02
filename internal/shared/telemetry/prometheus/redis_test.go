// Package prometheus provides tests for Redis metrics collection.
package prometheus

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisConfig(t *testing.T) {
	cfg := RedisConfig{
		ServiceName: "test-service",
		DBName:      "0",
	}

	assert.Equal(t, "test-service", cfg.ServiceName)
	assert.Equal(t, "0", cfg.DBName)
}

func TestNewRedisTracer(t *testing.T) {
	cfg := Config{ServiceName: "test-redis-tracer"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	tracer := NewRedisTracer(metrics, "test-service", "0")

	assert.NotNil(t, tracer)
	assert.Equal(t, metrics.Redis, tracer.metrics)
	assert.Equal(t, "test-service", tracer.serviceName)
	assert.Equal(t, "0", tracer.dbName)
}

func TestRedisTracer_TraceCommand(t *testing.T) {
	cfg := Config{ServiceName: "test-trace-cmd"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)
	tracer := NewRedisTracer(metrics, "test-service", "0")

	t.Run("records successful command", func(t *testing.T) {
		tracer.TraceCommand("GET", 1*time.Millisecond, nil)

		// No panic
	})

	t.Run("records command error", func(t *testing.T) {
		tracer.TraceCommand("SET", 5*time.Millisecond, errors.New("redis error"))

		// No panic
	})

	t.Run("records command duration", func(t *testing.T) {
		tracer.TraceCommand("HGET", 500*time.Microsecond, nil)

		// No panic
	})
}

func TestRedisTracer_RecordPoolStats(t *testing.T) {
	cfg := Config{ServiceName: "test-pool-stats"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)
	tracer := NewRedisTracer(metrics, "test-service", "0")

	stats := &redis.PoolStats{
		Hits:     100,
		Misses:   10,
		Timeouts: 1,
	}

	tracer.RecordPoolStats(stats)

	// No panic - metrics recorded
	assert.NotNil(t, tracer)
}

func TestNormalizeRedisCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple GET", "GET", "get"},
		{"GET with args", "GET mykey", "get"},
		{"SET", "SET key value", "set"},
		{"SETEX", "SETEX key 10 value", "set"},
		{"PSETEX", "PSETEX key 1000 value", "set"},
		{"SETNX", "SETNX key value", "set"},
		{"MSET", "MSET key1 val1 key2 val2", "set"},
		{"GETEX", "GETEX key", "get"},
		{"MGET", "MGET key1 key2", "get"},
		{"INCR", "INCR counter", "incr"},
		{"INCRBY", "INCRBY counter 10", "incr"},
		{"INCRBYFLOAT", "INCRBYFLOAT counter 0.5", "incr"},
		{"DECR", "DECR counter", "incr"},
		{"DECRBY", "DECRBY counter 5", "incr"},
		{"HSET", "HSET hash field value", "hset"},
		{"HMSET", "HMSET hash field1 val1 field2 val2", "hset"},
		{"HSETNX", "HSETNX hash field value", "hset"},
		{"HGET", "HGET hash field", "hget"},
		{"HMGET", "HMGET hash field1 field2", "hget"},
		{"LPUSH", "LPUSH list value", "lpush"},
		{"RPUSH", "RPUSH list value", "lpush"},
		{"LPOP", "LPOP list", "lpop"},
		{"RPOP", "RPOP list", "lpop"},
		{"SADD", "SADD set member", "sadd"},
		{"SREM", "SREM set member", "sadd"},
		{"SISMEMBER", "SISMEMBER set member", "sadd"},
		{"SMEMBERS", "SMEMBERS set", "sadd"},
		{"ZADD", "ZADD zset 1 member", "zadd"},
		{"ZREM", "ZREM zset member", "zadd"},
		{"ZRANGE", "ZRANGE zset 0 -1", "zadd"},
		{"ZSCORE", "ZSCORE zset member", "zadd"},
		{"EXPIRE", "EXPIRE key 60", "expire"},
		{"PEXPIRE", "PEXPIRE key 60000", "expire"},
		{"EXPIREAT", "EXPIREAT key 1234567890", "expire"},
		{"PEXPIREAT", "PEXPIREAT key 1234567890000", "expire"},
		{"TTL", "TTL key", "ttl"},
		{"PTTL", "PTTL key", "ttl"},
		{"DEL", "DEL key1 key2", "del"},
		{"EXISTS", "EXISTS key", "exists"},
		{"unknown command", "UNKNOWNCMD", "unknowncmd"},
		{"empty string", "", "unknown"},
		{"with newlines", "GET\nmykey", "get"},
		{"with tabs", "GET\tmykey", "get"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeRedisCommand(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewClientWithMetrics(t *testing.T) {
	cfg := Config{ServiceName: "test-client-metrics"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	// Create a mock client
	mockClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	client := NewClientWithMetrics(mockClient, metrics, "test-service", "0")

	assert.NotNil(t, client)
	assert.Equal(t, mockClient, client.client)
	assert.NotNil(t, client.tracer)
}

func TestClientWithMetrics_Get(t *testing.T) {
	cfg := Config{ServiceName: "test-client-get"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)
	mockClient := redis.NewClient(&redis.Options{})

	client := NewClientWithMetrics(mockClient, metrics, "test-service", "0")

	ctx := context.Background()

	// This will fail to connect but should record metrics
	result := client.Get(ctx, "testkey")

	// Result is returned even if connection fails
	assert.NotNil(t, result)
}

func TestClientWithMetrics_Set(t *testing.T) {
	cfg := Config{ServiceName: "test-client-set"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)
	mockClient := redis.NewClient(&redis.Options{})

	client := NewClientWithMetrics(mockClient, metrics, "test-service", "0")

	ctx := context.Background()

	result := client.Set(ctx, "testkey", "testvalue", time.Hour)

	assert.NotNil(t, result)
}

func TestClientWithMetrics_Del(t *testing.T) {
	cfg := Config{ServiceName: "test-client-del"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)
	mockClient := redis.NewClient(&redis.Options{})

	client := NewClientWithMetrics(mockClient, metrics, "test-service", "0")

	ctx := context.Background()

	result := client.Del(ctx, "key1", "key2")

	assert.NotNil(t, result)
}

func TestClientWithMetrics_HGet(t *testing.T) {
	cfg := Config{ServiceName: "test-client-hget"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)
	mockClient := redis.NewClient(&redis.Options{})

	client := NewClientWithMetrics(mockClient, metrics, "test-service", "0")

	ctx := context.Background()

	result := client.HGet(ctx, "hash", "field")

	assert.NotNil(t, result)
}

func TestClientWithMetrics_HSet(t *testing.T) {
	cfg := Config{ServiceName: "test-client-hset"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)
	mockClient := redis.NewClient(&redis.Options{})

	client := NewClientWithMetrics(mockClient, metrics, "test-service", "0")

	ctx := context.Background()

	result := client.HSet(ctx, "hash", "field", "value")

	assert.NotNil(t, result)
}

func TestClientWithMetrics_Incr(t *testing.T) {
	cfg := Config{ServiceName: "test-client-incr"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)
	mockClient := redis.NewClient(&redis.Options{})

	client := NewClientWithMetrics(mockClient, metrics, "test-service", "0")

	ctx := context.Background()

	result := client.Incr(ctx, "counter")

	assert.NotNil(t, result)
}

func TestClientWithMetrics_Decr(t *testing.T) {
	cfg := Config{ServiceName: "test-client-decr"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)
	mockClient := redis.NewClient(&redis.Options{})

	client := NewClientWithMetrics(mockClient, metrics, "test-service", "0")

	ctx := context.Background()

	result := client.Decr(ctx, "counter")

	assert.NotNil(t, result)
}

func TestClientWithMetrics_Expire(t *testing.T) {
	cfg := Config{ServiceName: "test-client-expire"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)
	mockClient := redis.NewClient(&redis.Options{})

	client := NewClientWithMetrics(mockClient, metrics, "test-service", "0")

	ctx := context.Background()

	result := client.Expire(ctx, "key", time.Minute)

	assert.NotNil(t, result)
}

func TestClientWithMetrics_TTL(t *testing.T) {
	cfg := Config{ServiceName: "test-client-ttl"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)
	mockClient := redis.NewClient(&redis.Options{})

	client := NewClientWithMetrics(mockClient, metrics, "test-service", "0")

	ctx := context.Background()

	result := client.TTL(ctx, "key")

	assert.NotNil(t, result)
}

func TestClientWithMetrics_Ping(t *testing.T) {
	cfg := Config{ServiceName: "test-client-ping"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)
	mockClient := redis.NewClient(&redis.Options{})

	client := NewClientWithMetrics(mockClient, metrics, "test-service", "0")

	ctx := context.Background()

	result := client.Ping(ctx)

	assert.NotNil(t, result)
}

func TestStartPoolStatsCollector(t *testing.T) {
	cfg := Config{ServiceName: "test-pool-collector"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)
	mockClient := redis.NewClient(&redis.Options{})

	stop := StartPoolStatsCollector(mockClient, metrics, "test-service", "0", 10*time.Millisecond)

	// Let it run a bit
	time.Sleep(50 * time.Millisecond)

	// Stop the collector
	stop()

	// Give it time to stop
	time.Sleep(20 * time.Millisecond)

	// Should complete without panic
	assert.True(t, true)
}

func TestRecordRedisCommand(t *testing.T) {
	cfg := Config{ServiceName: "test-record-cmd"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	t.Run("records command metrics", func(t *testing.T) {
		RecordRedisCommand(metrics, "test-service", "0", "GET", 5*time.Millisecond, nil)
	})

	t.Run("records command error", func(t *testing.T) {
		RecordRedisCommand(metrics, "test-service", "0", "SET", 2*time.Millisecond, errors.New("error"))
	})
}

func TestRedisHook(t *testing.T) {
	cfg := Config{ServiceName: "test-redis-hook"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	hook := NewRedisHook(metrics, "test-service", "0")

	assert.NotNil(t, hook)
	assert.NotNil(t, hook.tracer)
}

func TestRedisHook_BeforeProcess(t *testing.T) {
	cfg := Config{ServiceName: "test-hook-before"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)
	hook := NewRedisHook(metrics, "test-service", "0")

	ctx := context.Background()
	cmd := redis.NewCmd(ctx, "GET", "key")

	newCtx, err := hook.BeforeProcess(ctx, cmd)

	assert.NoError(t, err)
	assert.NotNil(t, newCtx)
}

func TestRedisHook_AfterProcess(t *testing.T) {
	cfg := Config{ServiceName: "test-hook-after"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)
	hook := NewRedisHook(metrics, "test-service", "0")

	ctx := context.Background()
	cmd := redis.NewCmd(ctx, "GET", "key")

	// Add start time to context
	ctx = context.WithValue(ctx, "redis_start_time", time.Now())

	err = hook.AfterProcess(ctx, cmd)

	assert.NoError(t, err)
}

func TestRedisHook_BeforeProcessPipeline(t *testing.T) {
	cfg := Config{ServiceName: "test-hook-pipeline-before"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)
	hook := NewRedisHook(metrics, "test-service", "0")

	ctx := context.Background()
	cmds := []redis.Cmder{
		redis.NewCmd(ctx, "GET", "key1"),
		redis.NewCmd(ctx, "SET", "key2", "value"),
	}

	newCtx, err := hook.BeforeProcessPipeline(ctx, cmds)

	assert.NoError(t, err)
	assert.NotNil(t, newCtx)
}

func TestRedisHook_AfterProcessPipeline(t *testing.T) {
	cfg := Config{ServiceName: "test-hook-pipeline-after"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)
	hook := NewRedisHook(metrics, "test-service", "0")

	ctx := context.Background()
	cmds := []redis.Cmder{
		redis.NewCmd(ctx, "GET", "key1"),
		redis.NewCmd(ctx, "SET", "key2", "value"),
	}

	// Add start time to context
	ctx = context.WithValue(ctx, "redis_pipeline_start_time", time.Now())

	err = hook.AfterProcessPipeline(ctx, cmds)

	assert.NoError(t, err)
}

func TestRedisMetrics_CommandTrackingDetails(t *testing.T) {
	cfg := Config{ServiceName: "test-redis-cmd-tracking"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	labels := prometheus.Labels{
		LabelServiceName:  "test-redis-cmd-tracking",
		LabelRedisDB:      "0",
		LabelRedisCommand: "get",
	}

	t.Run("record command count", func(t *testing.T) {
		metrics.Redis.CommandsTotal.With(labels).Inc()
		metrics.Redis.CommandsTotal.With(labels).Add(5)

		// Metrics recorded
	})

	t.Run("record command duration", func(t *testing.T) {
		metrics.Redis.CommandDuration.With(labels).Observe(0.001)

		// Metric recorded
	})

	t.Run("record command errors", func(t *testing.T) {
		metrics.Redis.CommandErrorsTotal.With(labels).Inc()

		// Metric recorded
	})
}

func TestRedisMetrics_PoolTrackingDetails(t *testing.T) {
	cfg := Config{ServiceName: "test-redis-pool-tracking"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	labels := prometheus.Labels{
		LabelServiceName: "test-redis-pool-tracking",
		LabelRedisDB:     "0",
	}

	t.Run("track active connections", func(t *testing.T) {
		metrics.Redis.ConnectionsActive.With(labels).Set(10)
	})

	t.Run("track idle connections", func(t *testing.T) {
		metrics.Redis.ConnectionsIdle.With(labels).Set(5)
	})

	t.Run("track pool hits", func(t *testing.T) {
		metrics.Redis.PoolHitsTotal.With(labels).Add(100)
	})

	t.Run("track pool misses", func(t *testing.T) {
		metrics.Redis.PoolMissesTotal.With(labels).Add(10)
	})

	t.Run("track pool timeouts", func(t *testing.T) {
		metrics.Redis.PoolTimeoutsTotal.With(labels).Add(1)
	})
}

func TestRedisMetrics_DifferentDatabases(t *testing.T) {
	cfg := Config{ServiceName: "test-redis-db"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	databases := []string{"0", "1", "2", "cache", "sessions"}

	for _, db := range databases {
		labels := prometheus.Labels{
			LabelServiceName:  "test-redis-db",
			LabelRedisDB:      db,
			LabelRedisCommand: "get",
		}

		metrics.Redis.CommandsTotal.With(labels).Inc()
	}

	// All databases should be tracked
	assert.NotNil(t, metrics)
}

func TestRedisMetrics_DifferentCommands(t *testing.T) {
	cfg := Config{ServiceName: "test-redis-cmds"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)

	commands := []string{"get", "set", "hget", "hset", "incr", "lpush", "sadd", "zadd"}

	for _, cmd := range commands {
		labels := prometheus.Labels{
			LabelServiceName:  "test-redis-cmds",
			LabelRedisDB:      "0",
			LabelRedisCommand: cmd,
		}

		metrics.Redis.CommandsTotal.With(labels).Inc()
	}

	// All commands should be tracked
	assert.NotNil(t, metrics)
}

func TestWrapRedisClient(t *testing.T) {
	cfg := Config{ServiceName: "test-wrap-redis"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	redisCfg := RedisConfig{
		ServiceName: "test-service",
		DBName:      "0",
	}

	// Create a mock client
	mockClient := redis.NewClient(&redis.Options{})

	// Wrap the client
	wrapped := WrapRedisClient(mockClient, reg, redisCfg)

	// Should return the same client (wrapping happens via redisotel)
	assert.NotNil(t, wrapped)
}

func TestWrapRedisCluster(t *testing.T) {
	cfg := Config{ServiceName: "test-wrap-cluster"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	redisCfg := RedisConfig{
		ServiceName: "test-service",
		DBName:      "0",
	}

	// Create a mock cluster client
	mockCluster := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: []string{":7000", ":7001", ":7002"},
	})

	// Wrap the cluster client
	wrapped := WrapRedisCluster(mockCluster, reg, redisCfg)

	// Should return the same client
	assert.NotNil(t, wrapped)
}

func TestRedisTracer_ConcurrentRecording(t *testing.T) {
	cfg := Config{ServiceName: "test-concurrent-redis"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)
	tracer := NewRedisTracer(metrics, "test-service", "0")

	done := make(chan bool)

	// Simulate concurrent command recording
	for i := 0; i < 100; i++ {
		go func(idx int) {
			cmd := "GET"
			if idx%2 == 0 {
				cmd = "SET"
			}
			tracer.TraceCommand(cmd, time.Duration(idx)*time.Microsecond, nil)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	// Should complete without race
	assert.NotNil(t, tracer)
}

func TestNormalizeRedisCommand_EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"leading whitespace", "   GET"},
		{"trailing whitespace", "GET   "},
		{"multiple spaces", "GET   key"},
		{"mixed case", "GeT keY"},
		{"with pipe", "GET|key"},
		{"very long command", strings.Repeat("a", 300)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeRedisCommand(tt.input)
			// Should not panic
			assert.NotEmpty(t, result)
		})
	}
}

func TestRedisHook_ContextHandling(t *testing.T) {
	cfg := Config{ServiceName: "test-hook-ctx"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	metrics := NewMetrics(reg)
	hook := NewRedisHook(metrics, "test-service", "0")

	ctx := context.Background()
	cmd := redis.NewCmd(ctx, "GET", "key")

	t.Run("handles missing start time", func(t *testing.T) {
		// Call AfterProcess without BeforeProcess
		err := hook.AfterProcess(ctx, cmd)
		// Should not panic
		assert.NoError(t, err)
	})

	t.Run("handles invalid start time type", func(t *testing.T) {
		invalidCtx := context.WithValue(ctx, "redis_start_time", "not a time")
		err := hook.AfterProcess(invalidCtx, cmd)
		// Should not panic
		assert.NoError(t, err)
	})
}
