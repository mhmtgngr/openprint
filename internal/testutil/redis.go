// Package testutil provides Redis testcontainer setup for testing.
package testutil

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	// DefaultRedisPort is the default Redis port.
	DefaultRedisPort = "6379"
)

// TestRedis holds resources for a Redis test container.
type TestRedis struct {
	Container testcontainers.Container
	Client    *redis.Client
	Host      string
	Port      string
}

// SetupRedisContainer creates and starts a Redis container for testing.
// It returns a TestRedis struct containing the container, client, and connection details.
//
// Usage in tests:
//
//	func TestMain(m *testing.M) {
//	    testRedis, err := testutil.SetupRedisContainer(context.Background())
//	    if err != nil {
//	        log.Fatalf("Failed to setup test Redis: %v", err)
//	    }
//	    defer testutil.CleanupRedis(testRedis)
//	    os.Exit(m.Run())
//	}
func SetupRedisContainer(ctx context.Context) (*TestRedis, error) {
	// Create Redis container request
	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{DefaultRedisPort + "/tcp"},
		WaitingFor: wait.ForLog("Ready to accept connections").
			WithOccurrence(1).
			WithStartupTimeout(30 * time.Second),
		Cmd: []string{"redis-server", "--appendonly", "yes"}, // Enable AOF for persistence testing
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("create redis container: %w", err)
	}

	// Get the mapped port
	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("get container host: %w", err)
	}

	port, err := container.MappedPort(ctx, DefaultRedisPort)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("get container port: %w", err)
	}

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr:            host + ":" + port.Port(),
		Password:        "",
		DB:              0, // Default DB
		MaxRetries:      3,
		MaxRetryBackoff: 500 * time.Millisecond,
		DialTimeout:     10 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolSize:        10,
		MinIdleConns:    2,
	})

	// Test connection with retry
	var lastErr error
	for i := 0; i < 10; i++ {
		if err := client.Ping(ctx).Err(); err == nil {
			break
		} else {
			lastErr = err
			select {
			case <-ctx.Done():
				client.Close()
				container.Terminate(ctx)
				return nil, ctx.Err()
			case <-time.After(time.Duration(i+1) * 200 * time.Millisecond):
			}
		}
	}

	if lastErr != nil {
		client.Close()
		container.Terminate(ctx)
		return nil, fmt.Errorf("redis ping failed: %w", lastErr)
	}

	return &TestRedis{
		Container: container,
		Client:    client,
		Host:      host,
		Port:      port.Port(),
	}, nil
}

// SetupRedisContainerWithDB creates a Redis container with a specific database number.
// This is useful for tests that need isolated databases.
func SetupRedisContainerWithDB(ctx context.Context, db int) (*TestRedis, error) {
	testRedis, err := SetupRedisContainer(ctx)
	if err != nil {
		return nil, err
	}

	// Update client to use the specified database
	testRedis.Client.Close()
	testRedis.Client = redis.NewClient(&redis.Options{
		Addr:            testRedis.Host + ":" + testRedis.Port,
		Password:        "",
		DB:              db,
		MaxRetries:      3,
		MaxRetryBackoff: 500 * time.Millisecond,
		DialTimeout:     10 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolSize:        10,
		MinIdleConns:    2,
	})

	// Test connection
	if err := testRedis.Client.Ping(ctx).Err(); err != nil {
		CleanupRedis(testRedis)
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return testRedis, nil
}

// CleanupRedis terminates the Redis container and closes the client.
// It should be called in a defer statement after SetupRedisContainer.
func CleanupRedis(tr *TestRedis) {
	if tr == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if tr.Client != nil {
		if err := tr.Client.Close(); err != nil {
			log.Printf("Warning: failed to close redis client: %v", err)
		}
	}

	if tr.Container != nil {
		if err := tr.Container.Terminate(ctx); err != nil {
			log.Printf("Warning: failed to terminate redis container: %v", err)
		}
	}
}

// FlushAll clears all data from the Redis test instance.
// This is useful for cleaning up between tests without re-creating the container.
func (tr *TestRedis) FlushAll(ctx context.Context) error {
	if tr == nil || tr.Client == nil {
		return nil
	}
	return tr.Client.FlushAll(ctx).Err()
}

// FlushDB clears the current database from the Redis test instance.
func (tr *TestRedis) FlushDB(ctx context.Context) error {
	if tr == nil || tr.Client == nil {
		return nil
	}
	return tr.Client.FlushDB(ctx).Err()
}

// GetConnectionURL returns a Redis connection URL string.
// This is useful for tests that need the connection string directly.
func (tr *TestRedis) GetConnectionURL() string {
	if tr == nil {
		return ""
	}
	return "redis://" + tr.Host + ":" + tr.Port
}

// GetConnectionAddr returns the Redis address in host:port format.
func (tr *TestRedis) GetConnectionAddr() string {
	if tr == nil {
		return ""
	}
	return tr.Host + ":" + tr.Port
}

// SelectDB switches the Redis client to a different database.
func (tr *TestRedis) SelectDB(ctx context.Context, db int) error {
	if tr == nil || tr.Client == nil {
		return fmt.Errorf("redis client is nil")
	}

	oldClient := tr.Client
	tr.Client = redis.NewClient(&redis.Options{
		Addr:            tr.Host + ":" + tr.Port,
		Password:        "",
		DB:              db,
		MaxRetries:      oldClient.Options().MaxRetries,
		DialTimeout:     oldClient.Options().DialTimeout,
		ReadTimeout:     oldClient.Options().ReadTimeout,
		WriteTimeout:    oldClient.Options().WriteTimeout,
		PoolSize:        oldClient.Options().PoolSize,
		MinIdleConns:    oldClient.Options().MinIdleConns,
		MaxRetryBackoff: oldClient.Options().MaxRetryBackoff,
	})

	// Close old client
	oldClient.Close()

	// Test new connection
	if err := tr.Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed on db %d: %w", db, err)
	}

	return nil
}

// GetDBCount returns the number of keys in the current database.
func (tr *TestRedis) GetDBCount(ctx context.Context) (int64, error) {
	if tr == nil || tr.Client == nil {
		return 0, nil
	}
	return tr.Client.DBSize(ctx).Result()
}

// SetupRedisForTest is a convenience function that sets up a Redis container
// and calls testing.Main with cleanup. Use this in TestMain for simpler setup.
//
// Usage:
//
//	func TestMain(m *testing.M) {
//	    os.Exit(testutil.SetupRedisForTest(m))
//	}
func SetupRedisForTest(m *testing.M) int {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	testRedis, err := SetupRedisContainer(ctx)
	if err != nil {
		log.Fatalf("Failed to setup test Redis: %v", err)
	}
	defer CleanupRedis(testRedis)

	return m.Run()
}

// CreateTestCluster creates a Redis cluster for testing advanced scenarios.
// This starts multiple Redis nodes and sets up cluster mode.
func CreateTestCluster(ctx context.Context, numNodes int) ([]*TestRedis, error) {
	if numNodes < 3 {
		return nil, fmt.Errorf("redis cluster requires at least 3 nodes")
	}

	nodes := make([]*TestRedis, 0, numNodes)
	ports := make([]string, 0, numNodes)

	// Start individual Redis nodes
	for i := 0; i < numNodes; i++ {
		req := testcontainers.ContainerRequest{
			Image:        "redis:7-alpine",
			ExposedPorts: []string{DefaultRedisPort + "/tcp"},
			WaitingFor: wait.ForLog("Ready to accept connections").
				WithOccurrence(1).
				WithStartupTimeout(30 * time.Second),
			Cmd: []string{
				"redis-server",
				"--cluster-enabled", "yes",
				"--cluster-config-file", "nodes.conf",
				"--cluster-node-timeout", "5000",
				"--appendonly", "yes",
				"--port", DefaultRedisPort,
			},
		}

		container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		if err != nil {
			// Cleanup any created nodes
			for _, node := range nodes {
				CleanupRedis(node)
			}
			return nil, fmt.Errorf("create redis node %d: %w", i, err)
		}

		host, err := container.Host(ctx)
		if err != nil {
			container.Terminate(ctx)
			for _, node := range nodes {
				CleanupRedis(node)
			}
			return nil, fmt.Errorf("get host for node %d: %w", i, err)
		}

		port, err := container.MappedPort(ctx, DefaultRedisPort)
		if err != nil {
			container.Terminate(ctx)
			for _, node := range nodes {
				CleanupRedis(node)
			}
			return nil, fmt.Errorf("get port for node %d: %w", i, err)
		}

		node := &TestRedis{
			Container: container,
			Host:      host,
			Port:      port.Port(),
		}

		// Create client for this node
		node.Client = redis.NewClient(&redis.Options{
			Addr:        host + ":" + port.Port(),
			DialTimeout: 10 * time.Second,
		})

		nodes = append(nodes, node)
		ports = append(ports, host+":"+port.Port())
	}

	// Note: Full cluster setup would require executing redis-cli --cluster create
	// This is a simplified version that starts nodes in cluster mode

	return nodes, nil
}

// SetupRedisWithModules creates a Redis container with additional modules loaded.
// Useful for testing RedisJSON, RedisSearch, etc.
func SetupRedisWithModules(ctx context.Context, modules []string) (*TestRedis, error) {
	// Redis with modules - using redisstack which includes popular modules
	image := "redis/redis-stack-server:latest"

	req := testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{DefaultRedisPort + "/tcp"},
		WaitingFor: wait.ForLog("Ready to accept connections").
			WithOccurrence(1).
			WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("create redis container with modules: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("get container host: %w", err)
	}

	port, err := container.MappedPort(ctx, DefaultRedisPort)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("get container port: %w", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr:        host + ":" + port.Port(),
		DialTimeout: 10 * time.Second,
	})

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		container.Terminate(ctx)
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &TestRedis{
		Container: container,
		Client:    client,
		Host:      host,
		Port:      port.Port(),
	}, nil
}

// WaitKey waits for a key to appear in Redis with a timeout.
// Useful for testing pub/sub and queue operations.
func (tr *TestRedis) WaitKey(ctx context.Context, key string, timeout time.Duration) error {
	if tr == nil || tr.Client == nil {
		return fmt.Errorf("redis client is nil")
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			exists := tr.Client.Exists(ctx, key).Val()
			if exists > 0 {
				return nil
			}
		}
	}
}

// PubSub returns a PubSub instance for the test Redis.
func (tr *TestRedis) PubSub() *redis.PubSub {
	if tr == nil || tr.Client == nil {
		return nil
	}
	return tr.Client.Subscribe(context.Background())
}

// GetPortAsInt returns the Redis port as an integer.
func (tr *TestRedis) GetPortAsInt() (int, error) {
	if tr == nil {
		return 0, fmt.Errorf("test redis is nil")
	}
	return strconv.Atoi(tr.Port)
}
