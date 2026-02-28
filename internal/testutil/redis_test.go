//go:build integration

package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupRedisContainer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	tr, err := SetupRedisContainer(ctx)
	require.NoError(t, err)
	require.NotNil(t, tr)
	defer CleanupRedis(tr)

	// Verify container is running
	assert.NotNil(t, tr.Container)

	// Verify client is created
	assert.NotNil(t, tr.Client)

	// Verify host and port
	assert.NotEmpty(t, tr.Host)
	assert.NotEmpty(t, tr.Port)

	// Test connection
	err = tr.Client.Ping(ctx).Err()
	require.NoError(t, err)
}

func TestSetupRedisContainer_BasicOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	tr, err := SetupRedisContainer(ctx)
	require.NoError(t, err)
	defer CleanupRedis(tr)

	// Test SET and GET
	err = tr.Client.Set(ctx, "test-key", "test-value", time.Hour).Err()
	require.NoError(t, err)

	val, err := tr.Client.Get(ctx, "test-key").Result()
	require.NoError(t, err)
	assert.Equal(t, "test-value", val)
}

func TestSetupRedisContainerWithDB(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Create container with DB 5
	tr, err := SetupRedisContainerWithDB(ctx, 5)
	require.NoError(t, err)
	defer CleanupRedis(tr)

	// Test connection
	err = tr.Client.Ping(ctx).Err()
	require.NoError(t, err)

	// Verify we can store and retrieve data
	err = tr.Client.Set(ctx, "key", "value", 0).Err()
	require.NoError(t, err)

	val, err := tr.Client.Get(ctx, "key").Result()
	require.NoError(t, err)
	assert.Equal(t, "value", val)
}

func TestCleanupRedis_NilTestRedis(t *testing.T) {
	// Should not panic with nil
	CleanupRedis(nil)
}

func TestCleanupRedis_NilClient(t *testing.T) {
	tr := &TestRedis{Client: nil}
	CleanupRedis(tr)
}

func TestFlushAll(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	tr, err := SetupRedisContainer(ctx)
	require.NoError(t, err)
	defer CleanupRedis(tr)

	// Add some data
	for i := 0; i < 10; i++ {
		err = tr.Client.Set(ctx, "key"+string(rune(i)), "value"+string(rune(i)), 0).Err()
		require.NoError(t, err)
	}

	// Verify data exists
	count, err := tr.Client.DBSize(ctx).Result()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(10))

	// Flush all
	err = tr.FlushAll(ctx)
	require.NoError(t, err)

	// Verify all data is gone
	count, err = tr.Client.DBSize(ctx).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestFlushAll_NilTestRedis(t *testing.T) {
	var tr *TestRedis
	err := tr.FlushAll(context.Background())
	assert.NoError(t, err)
}

func TestFlushDB(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	tr, err := SetupRedisContainer(ctx)
	require.NoError(t, err)
	defer CleanupRedis(tr)

	// Add some data
	for i := 0; i < 5; i++ {
		err = tr.Client.Set(ctx, "key"+string(rune(i)), "value"+string(rune(i)), 0).Err()
		require.NoError(t, err)
	}

	// Verify data exists
	count, err := tr.Client.DBSize(ctx).Result()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(5))

	// Flush current DB
	err = tr.FlushDB(ctx)
	require.NoError(t, err)

	// Verify data is gone
	count, err = tr.Client.DBSize(ctx).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestGetConnectionURL(t *testing.T) {
	t.Run("nil TestRedis returns empty string", func(t *testing.T) {
		var tr *TestRedis
		url := tr.GetConnectionURL()
		assert.Empty(t, url)
	})

	t.Run("valid TestRedis returns URL", func(t *testing.T) {
		tr := &TestRedis{Host: "localhost", Port: "6379"}
		url := tr.GetConnectionURL()
		assert.Equal(t, "redis://localhost:6379", url)
	})
}

func TestGetConnectionAddr(t *testing.T) {
	t.Run("nil TestRedis returns empty string", func(t *testing.T) {
		var tr *TestRedis
		addr := tr.GetConnectionAddr()
		assert.Empty(t, addr)
	})

	t.Run("valid TestRedis returns addr", func(t *testing.T) {
		tr := &TestRedis{Host: "localhost", Port: "6379"}
		addr := tr.GetConnectionAddr()
		assert.Equal(t, "localhost:6379", addr)
	})
}

func TestSelectDB(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	tr, err := SetupRedisContainer(ctx)
	require.NoError(t, err)
	defer CleanupRedis(tr)

	// Set value in DB 0
	err = tr.Client.Set(ctx, "key", "value0", 0).Err()
	require.NoError(t, err)

	// Switch to DB 1
	err = tr.SelectDB(ctx, 1)
	require.NoError(t, err)

	// Value should not exist in DB 1
	_, err = tr.Client.Get(ctx, "key").Result()
	assert.Equal(t, redis.Nil, err)

	// Set different value in DB 1
	err = tr.Client.Set(ctx, "key", "value1", 0).Err()
	require.NoError(t, err)

	val, err := tr.Client.Get(ctx, "key").Result()
	require.NoError(t, err)
	assert.Equal(t, "value1", val)
}

func TestSelectDB_NilTestRedis(t *testing.T) {
	var tr *TestRedis
	err := tr.SelectDB(context.Background(), 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestGetDBCount(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	tr, err := SetupRedisContainer(ctx)
	require.NoError(t, err)
	defer CleanupRedis(tr)

	// Initially empty
	count, err := tr.GetDBCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Add some keys
	for i := 0; i < 5; i++ {
		err = tr.Client.Set(ctx, "key"+string(rune(i)), "value", 0).Err()
		require.NoError(t, err)
	}

	// Check count again
	count, err = tr.GetDBCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(5), count)
}

func TestGetDBCount_NilTestRedis(t *testing.T) {
	var tr *TestRedis
	count, err := tr.GetDBCount(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestSetupRedisForTest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Smoke test for SetupRedisForTest
	done := make(chan int, 1)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		tr, err := SetupRedisContainer(ctx)
		require.NoError(t, err)
		CleanupRedis(tr)
		cancel()
		done <- 1
	}()

	select {
	case <-done:
		// Success
	case <-time.After(3 * time.Minute):
		t.Fatal("timeout waiting for test setup")
	}
}

func TestCreateTestCluster(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Create a 3-node cluster
	nodes, err := CreateTestCluster(ctx, 3)
	require.NoError(t, err)
	require.Len(t, nodes, 3)

	// Cleanup all nodes
	for _, node := range nodes {
		defer CleanupRedis(node)
	}

	// Verify all nodes are running
	for i, node := range nodes {
		assert.NotNil(t, node.Container)
		assert.NotEmpty(t, node.Host)
		assert.NotEmpty(t, node.Port)

		// Test connection to each node
		if node.Client != nil {
			err = node.Client.Ping(ctx).Err()
			assert.NoError(t, err, "node %d should be reachable", i)
		}
	}
}

func TestCreateTestCluster_TooFewNodes(t *testing.T) {
	ctx := context.Background()

	// Cluster requires at least 3 nodes
	nodes, err := CreateTestCluster(ctx, 2)
	assert.Error(t, err)
	assert.Nil(t, nodes)
}

func TestSetupRedisWithModules(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	tr, err := SetupRedisWithModules(ctx, []string{"search", "json"})
	require.NoError(t, err)
	defer CleanupRedis(tr)

	// Verify connection
	err = tr.Client.Ping(ctx).Err()
	require.NoError(t, err)

	// Test basic operations work
	err = tr.Client.Set(ctx, "key", "value", 0).Err()
	require.NoError(t, err)

	val, err := tr.Client.Get(ctx, "key").Result()
	require.NoError(t, err)
	assert.Equal(t, "value", val)
}

func TestWaitKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	tr, err := SetupRedisContainer(ctx)
	require.NoError(t, err)
	defer CleanupRedis(tr)

	// Wait for a key that will be set
	go func() {
		time.Sleep(500 * time.Millisecond)
		tr.Client.Set(ctx, "wait-key", "value", 0)
	}()

	err = tr.WaitKey(ctx, "wait-key", 2*time.Second)
	require.NoError(t, err)

	// Verify key exists
	exists := tr.Client.Exists(ctx, "wait-key").Val()
	assert.Equal(t, int64(1), exists)
}

func TestWaitKey_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	tr, err := SetupRedisContainer(ctx)
	require.NoError(t, err)
	defer CleanupRedis(tr)

	// Wait for a key that will never be set
	err = tr.WaitKey(ctx, "non-existent-key", 100*time.Millisecond)
	assert.Error(t, err)
}

func TestWaitKey_NilTestRedis(t *testing.T) {
	var tr *TestRedis
	err := tr.WaitKey(context.Background(), "key", time.Second)
	assert.Error(t, err)
}

func TestPubSub(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	tr, err := SetupRedisContainer(ctx)
	require.NoError(t, err)
	defer CleanupRedis(tr)

	// Get PubSub instance
	pubsub := tr.PubSub()
	require.NotNil(t, pubsub)

	// Subscribe to a channel
	err = pubsub.Subscribe(ctx, "test-channel")
	require.NoError(t, err)

	// Publish to the channel
	err = tr.Client.Publish(ctx, "test-channel", "test-message").Err()
	require.NoError(t, err)

	// Receive message
	msg, err := pubsub.ReceiveMessage(ctx)
	require.NoError(t, err)
	assert.Equal(t, "test-channel", msg.Channel)
	assert.Equal(t, "test-message", msg.Payload)

	// Close subscription
	pubsub.Close()
}

func TestGetPortAsInt(t *testing.T) {
	t.Run("valid port", func(t *testing.T) {
		tr := &TestRedis{Port: "6379"}
		port, err := tr.GetPortAsInt()
		require.NoError(t, err)
		assert.Equal(t, 6379, port)
	})

	t.Run("nil TestRedis", func(t *testing.T) {
		var tr *TestRedis
		_, err := tr.GetPortAsInt()
		assert.Error(t, err)
	})

	t.Run("invalid port", func(t *testing.T) {
		tr := &TestRedis{Port: "invalid"}
		_, err := tr.GetPortAsInt()
		assert.Error(t, err)
	})
}

func TestTestRedis_ConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	tr, err := SetupRedisContainer(ctx)
	require.NoError(t, err)
	defer CleanupRedis(tr)

	// Test concurrent operations
	errChan := make(chan error, 20)

	for i := 0; i < 10; i++ {
		go func(idx int) {
			key := "key" + string(rune(idx))
			err := tr.Client.Set(ctx, key, "value", 0).Err()
			errChan <- err
		}(i)
	}

	for i := 0; i < 10; i++ {
		go func(idx int) {
			key := "key" + string(rune(idx))
			_, err := tr.Client.Get(ctx, key).Result()
			errChan <- err
		}(i)
	}

	// Collect results
	for i := 0; i < 20; i++ {
		err := <-errChan
		// Some gets might return key not found, that's okay
		if err != nil && err != redis.Nil {
			t.Errorf("unexpected error: %v", err)
		}
	}
}

func TestTestRedis_Expiry(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	tr, err := SetupRedisContainer(ctx)
	require.NoError(t, err)
	defer CleanupRedis(tr)

	// Set a key with expiry
	err = tr.Client.Set(ctx, "expiring-key", "value", 500*time.Millisecond).Err()
	require.NoError(t, err)

	// Key should exist
	exists := tr.Client.Exists(ctx, "expiring-key").Val()
	assert.Equal(t, int64(1), exists)

	// Wait for expiry
	time.Sleep(600 * time.Millisecond)

	// Key should be gone
	exists = tr.Client.Exists(ctx, "expiring-key").Val()
	assert.Equal(t, int64(0), exists)
}

func TestCleanupRedis_MultipleCalls(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	tr, err := SetupRedisContainer(ctx)
	require.NoError(t, err)

	// Multiple cleanup calls should not panic
	CleanupRedis(tr)
	CleanupRedis(tr)
	CleanupRedis(nil)
}

func TestTestRedis_ListOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	tr, err := SetupRedisContainer(ctx)
	require.NoError(t, err)
	defer CleanupRedis(tr)

	// LPUSH and LRANGE
	err = tr.Client.LPush(ctx, "mylist", "item1", "item2", "item3").Err()
	require.NoError(t, err)

	items, err := tr.Client.LRange(ctx, "mylist", 0, -1).Result()
	require.NoError(t, err)
	assert.Len(t, items, 3)
}

func TestTestRedis_HashOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	tr, err := SetupRedisContainer(ctx)
	require.NoError(t, err)
	defer CleanupRedis(tr)

	// HSET and HGET
	err = tr.Client.HSet(ctx, "myhash", "field1", "value1").Err()
	require.NoError(t, err)

	val, err := tr.Client.HGet(ctx, "myhash", "field1").Result()
	require.NoError(t, err)
	assert.Equal(t, "value1", val)
}

func TestTestRedis_SetOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	tr, err := SetupRedisContainer(ctx)
	require.NoError(t, err)
	defer CleanupRedis(tr)

	// SADD and SMEMBERS
	err = tr.Client.SAdd(ctx, "myset", "member1", "member2", "member3").Err()
	require.NoError(t, err)

	members, err := tr.Client.SMembers(ctx, "myset").Result()
	require.NoError(t, err)
	assert.Len(t, members, 3)
}

func TestTestRedis_SortedSetOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	tr, err := SetupRedisContainer(ctx)
	require.NoError(t, err)
	defer CleanupRedis(tr)

	// ZADD and ZRANGE
	err = tr.Client.ZAdd(ctx, "myzset", redis.Z{Score: 1, Member: "one"},
		redis.Z{Score: 2, Member: "two"},
		redis.Z{Score: 3, Member: "three"}).Err()
	require.NoError(t, err)

	members, err := tr.Client.ZRange(ctx, "myzset", 0, -1).Result()
	require.NoError(t, err)
	assert.Len(t, members, 3)
}
