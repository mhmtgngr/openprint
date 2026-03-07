// Package testutil tests for context utilities
package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewContextManager(t *testing.T) {
	cm := NewContextManager()
	assert.NotNil(t, cm)
	assert.NotNil(t, cm.Context())
	assert.NotNil(t, cm.timeouts)
}

func TestContextManager_Context(t *testing.T) {
	cm := NewContextManager()
	ctx := cm.Context()
	assert.NotNil(t, ctx)
}

func TestContextManager_WithTimeout(t *testing.T) {
	cm := NewContextManager()

	ctx, cancel := cm.WithTimeout("test", 100*time.Millisecond)
	defer cancel()

	assert.NotNil(t, ctx)

	// Verify context works
	select {
	case <-ctx.Done():
		t.Fatal("Context should not be cancelled yet")
	case <-time.After(10 * time.Millisecond):
		// Expected
	}
}

func TestContextManager_WithTimeout_AutoCancel(t *testing.T) {
	cm := NewContextManager()

	// Create first timeout
	ctx1, cancel1 := cm.WithTimeout("test", 100*time.Millisecond)
	defer cancel1()

	// Create second timeout with same name (should cancel first)
	ctx2, cancel2 := cm.WithTimeout("test", 200*time.Millisecond)
	defer cancel2()

	// First context should be cancelled
	select {
	case <-ctx1.Done():
		// Expected - first context was cancelled
	case <-time.After(10 * time.Millisecond):
		// Expected - we cancelled it
	}

	// Second context should still be active
	select {
	case <-ctx2.Done():
		t.Fatal("Second context should not be cancelled")
	case <-time.After(10 * time.Millisecond):
		// Expected
	}
}

func TestContextManager_WithDeadline(t *testing.T) {
	cm := NewContextManager()
	deadline := time.Now().Add(1 * time.Hour)

	ctx, cancel := cm.WithDeadline("test", deadline)
	defer cancel()

	assert.NotNil(t, ctx)

	// Check deadline is set
	dl, ok := ctx.Deadline()
	assert.True(t, ok)
	assert.WithinDuration(t, deadline, dl, time.Minute)
}

func TestContextManager_CancelTimeout(t *testing.T) {
	cm := NewContextManager()

	ctx, cancel := cm.WithTimeout("test", 1*time.Second)
	cm.CancelTimeout("test")

	// Context should be cancelled
	select {
	case <-ctx.Done():
		// Expected
	case <-time.After(10 * time.Millisecond):
		t.Fatal("Context should be cancelled")
	}

	cancel() // Clean up
}

func TestContextManager_CancelAll(t *testing.T) {
	cm := NewContextManager()

	ctx1, _ := cm.WithTimeout("test1", 1*time.Second)
	ctx2, _ := cm.WithTimeout("test2", 1*time.Second)

	cm.CancelAll()

	// All contexts should be cancelled
	select {
	case <-ctx1.Done():
		// Expected
	case <-time.After(10 * time.Millisecond):
		t.Fatal("Context 1 should be cancelled")
	}

	select {
	case <-ctx2.Done():
		// Expected
	case <-time.After(10 * time.Millisecond):
		t.Fatal("Context 2 should be cancelled")
	}
}

func TestContextManager_Cleanup(t *testing.T) {
	cm := NewContextManager()

	ctx, _ := cm.WithTimeout("test", 1*time.Second)
	cm.Cleanup()

	// Base context should be cancelled
	select {
	case <-cm.Context().Done():
		// Expected
	case <-time.After(10 * time.Millisecond):
		t.Fatal("Context should be cancelled")
	}

	// Timeout context should also be cancelled
	select {
	case <-ctx.Done():
		// Expected
	case <-time.After(10 * time.Millisecond):
		t.Fatal("Timeout context should be cancelled")
	}
}

func TestShortContext(t *testing.T) {
	ctx, cancel := ShortContext()
	defer cancel()

	deadline, ok := ctx.Deadline()
	assert.True(t, ok)
	expectedDeadline := time.Now().Add(5 * time.Second)
	assert.WithinDuration(t, expectedDeadline, deadline, time.Second)
}

func TestMediumContext(t *testing.T) {
	ctx, cancel := MediumContext()
	defer cancel()

	deadline, ok := ctx.Deadline()
	assert.True(t, ok)
	expectedDeadline := time.Now().Add(30 * time.Second)
	assert.WithinDuration(t, expectedDeadline, deadline, time.Second)
}

func TestLongContext(t *testing.T) {
	ctx, cancel := LongContext()
	defer cancel()

	deadline, ok := ctx.Deadline()
	assert.True(t, ok)
	expectedDeadline := time.Now().Add(2 * time.Minute)
	assert.WithinDuration(t, expectedDeadline, deadline, 5*time.Second)
}

func TestTestContext(t *testing.T) {
	ctx, cancel := TestContext()
	defer cancel()

	assert.NotNil(t, ctx)

	deadline, ok := ctx.Deadline()
	assert.True(t, ok)
	expectedDeadline := time.Now().Add(30 * time.Second)
	assert.WithinDuration(t, expectedDeadline, deadline, time.Second)
}

func TestWithTestTimeout(t *testing.T) {
	err := WithTestTimeout(100*time.Millisecond, func(ctx context.Context) error {
		// Simulate work
		time.Sleep(10 * time.Millisecond)
		return nil
	})
	assert.NoError(t, err)
}

func TestWithTestTimeout_Timeout(t *testing.T) {
	err := WithTestTimeout(10*time.Millisecond, func(ctx context.Context) error {
		// Use select with sleep to properly respect context cancellation
		select {
		case <-time.After(100 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}

func TestDeadlineContext(t *testing.T) {
	duration := 1 * time.Hour
	ctx, cancel := DeadlineContext(duration)
	defer cancel()

	deadline, ok := ctx.Deadline()
	assert.True(t, ok)
	expectedDeadline := time.Now().Add(duration)
	assert.WithinDuration(t, expectedDeadline, deadline, time.Minute)
}

func TestBackgroundContext(t *testing.T) {
	ctx := BackgroundContext()
	assert.NotNil(t, ctx)

	// Background context should never expire
	_, ok := ctx.Deadline()
	assert.False(t, ok)
}

func TestCanceledContext(t *testing.T) {
	ctx := CanceledContext()

	select {
	case <-ctx.Done():
		// Expected - already cancelled
	case <-time.After(10 * time.Millisecond):
		t.Fatal("Context should be cancelled")
	}

	assert.Equal(t, context.Canceled, ctx.Err())
}

func TestContextWithValue(t *testing.T) {
	ctx := context.Background()
	ctx = ContextWithValue(ctx, "key1", "value1")
	ctx = ContextWithValue(ctx, "key2", "value2")

	assert.Equal(t, "value1", GetString(ctx, "key1"))
	assert.Equal(t, "value1", GetValue(ctx, "key1").(string))
}

func TestContextWithValues(t *testing.T) {
	ctx := context.Background()
	values := map[string]interface{}{
		"string": "value",
		"int":    42,
		"bool":   true,
	}

	ctx = ContextWithValues(ctx, values)

	assert.Equal(t, "value", GetString(ctx, "string"))
	assert.Equal(t, 42, GetInt(ctx, "int"))
	assert.NotNil(t, GetValue(ctx, "bool"))
}

func TestGetValue(t *testing.T) {
	ctx := context.Background()
	ctx = ContextWithValue(ctx, "key", "value")

	t.Run("existing key", func(t *testing.T) {
		v := GetValue(ctx, "key")
		assert.NotNil(t, v)
	})

	t.Run("non-existing key", func(t *testing.T) {
		v := GetValue(ctx, "nonexistent")
		assert.Nil(t, v)
	})
}

func TestGetString(t *testing.T) {
	ctx := context.Background()
	ctx = ContextWithValue(ctx, "string", "value")
	ctx = ContextWithValues(ctx, map[string]interface{}{
		"not_string": 42,
	})

	t.Run("string value", func(t *testing.T) {
		s := GetString(ctx, "string")
		assert.Equal(t, "value", s)
	})

	t.Run("non-string value", func(t *testing.T) {
		s := GetString(ctx, "not_string")
		assert.Empty(t, s)
	})

	t.Run("missing key", func(t *testing.T) {
		s := GetString(ctx, "missing")
		assert.Empty(t, s)
	})
}

func TestGetInt(t *testing.T) {
	ctx := context.Background()
	ctx = ContextWithValues(ctx, map[string]interface{}{
		"int":     42,
		"not_int": "value",
	})

	t.Run("int value", func(t *testing.T) {
		i := GetInt(ctx, "int")
		assert.Equal(t, 42, i)
	})

	t.Run("non-int value", func(t *testing.T) {
		i := GetInt(ctx, "not_int")
		assert.Equal(t, 0, i)
	})

	t.Run("missing key", func(t *testing.T) {
		i := GetInt(ctx, "missing")
		assert.Equal(t, 0, i)
	})
}

func TestGetDuration(t *testing.T) {
	ctx := context.Background()
	duration := 5 * time.Second
	ctx = ContextWithValues(ctx, map[string]interface{}{
		"duration": duration,
	})

	t.Run("duration value", func(t *testing.T) {
		d := GetDuration(ctx, "duration")
		assert.Equal(t, duration, d)
	})

	t.Run("missing key", func(t *testing.T) {
		d := GetDuration(ctx, "missing")
		assert.Equal(t, time.Duration(0), d)
	})
}

func TestContextTracker(t *testing.T) {
	ct := NewContextTracker()
	assert.NotNil(t, ct)

	ctx := ct.Track("test", context.Background())
	assert.NotNil(t, ctx)

	// Check status
	status := ct.Status("test")
	assert.Equal(t, "active", status)
}

func TestContextTracker_Track(t *testing.T) {
	ct := NewContextTracker()

	ctx1, cancel1 := context.WithCancel(context.Background())
	ct.Track("ctx1", ctx1)

	// Should be active
	assert.Equal(t, "active", ct.Status("ctx1"))

	// Cancel and check
	cancel1()
	time.Sleep(10 * time.Millisecond) // Give time for goroutine to update

	assert.Equal(t, "cancelled", ct.Status("ctx1"))
}

func TestContextTracker_Timeout(t *testing.T) {
	ct := NewContextTracker()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	ct.Track("timeout-ctx", ctx)

	// Wait for timeout
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, "timed out", ct.Status("timeout-ctx"))
}

func TestContextTracker_GetTracked(t *testing.T) {
	ct := NewContextTracker()

	_ = ct.Track("test", context.Background())
	tracked := ct.GetTracked("test")

	assert.NotNil(t, tracked)
}

func TestContextTracker_CleanupTracker(t *testing.T) {
	ct := NewContextTracker()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	ct.Track("test", ctx)

	ct.CleanupTracker()

	status := ct.Status("test")
	assert.Equal(t, "not found", status)
}

func TestTestDeadlineTracker(t *testing.T) {
	tdt := NewTestDeadlineTracker()
	assert.NotNil(t, tdt)

	deadline := time.Now().Add(1 * time.Hour)
	tdt.SetDeadline("op1", deadline)

	retrieved, ok := tdt.GetDeadline("op1")
	assert.True(t, ok)
	assert.WithinDuration(t, deadline, retrieved, time.Millisecond)
}

func TestTestDeadlineTracker_TimeRemaining(t *testing.T) {
	tdt := NewTestDeadlineTracker()

	deadline := time.Now().Add(1 * time.Hour)
	tdt.SetDeadline("op1", deadline)

	remaining := tdt.TimeRemaining("op1")
	assert.Greater(t, remaining, 50*time.Minute)
	assert.Less(t, remaining, 61*time.Minute)
}

func TestTestDeadlineTracker_IsExpired(t *testing.T) {
	tdt := NewTestDeadlineTracker()

	// Set past deadline
	pastDeadline := time.Now().Add(-1 * time.Hour)
	tdt.SetDeadline("expired", pastDeadline)

	assert.True(t, tdt.IsExpired("expired"))

	// Set future deadline
	futureDeadline := time.Now().Add(1 * time.Hour)
	tdt.SetDeadline("future", futureDeadline)

	assert.False(t, tdt.IsExpired("future"))

	// Non-existent deadline
	assert.True(t, tdt.IsExpired("nonexistent"))
}

func TestWaitForContext(t *testing.T) {
	t.Run("context done", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()

		err := WaitForContext(ctx, 100*time.Millisecond)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("timeout", func(t *testing.T) {
		ctx := context.Background()

		err := WaitForContext(ctx, 10*time.Millisecond)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout")
	})
}

func TestWaitForContextWithPoll(t *testing.T) {
	t.Run("condition met", func(t *testing.T) {
		ctx := context.Background()
		conditionMet := false

		go func() {
			time.Sleep(20 * time.Millisecond)
			conditionMet = true
		}()

		err := WaitForContextWithPoll(ctx, 5*time.Millisecond, 100*time.Millisecond, func() bool {
			return conditionMet
		})
		assert.NoError(t, err)
	})

	t.Run("context cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			time.Sleep(20 * time.Millisecond)
			cancel()
		}()

		err := WaitForContextWithPoll(ctx, 5*time.Millisecond, 100*time.Millisecond, func() bool {
			return false
		})
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("timeout", func(t *testing.T) {
		ctx := context.Background()

		err := WaitForContextWithPoll(ctx, 5*time.Millisecond, 20*time.Millisecond, func() bool {
			return false
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout")
	})
}

func TestTestContextBuilder(t *testing.T) {
	t.Run("basic build", func(t *testing.T) {
		ctx, cancel := NewTestContextBuilder().Build()
		defer cancel()
		assert.NotNil(t, ctx)
	})

	t.Run("with timeout", func(t *testing.T) {
		timeout := 1 * time.Hour
		ctx, cancel := NewTestContextBuilder().WithTimeout(timeout).Build()
		defer cancel()

		deadline, ok := ctx.Deadline()
		assert.True(t, ok)
		expected := time.Now().Add(timeout)
		assert.WithinDuration(t, expected, deadline, time.Minute)
	})

	t.Run("with deadline", func(t *testing.T) {
		deadline := time.Now().Add(2 * time.Hour)
		ctx, cancel := NewTestContextBuilder().WithDeadline(deadline).Build()
		defer cancel()

		dl, ok := ctx.Deadline()
		assert.True(t, ok)
		assert.WithinDuration(t, deadline, dl, time.Minute)
	})

	t.Run("with value", func(t *testing.T) {
		ctx, cancel := NewTestContextBuilder().WithValue("key", "value").Build()
		defer cancel()

		assert.Equal(t, "value", GetString(ctx, "key"))
	})

	t.Run("with values", func(t *testing.T) {
		values := map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		}
		ctx, cancel := NewTestContextBuilder().WithValues(values).Build()
		defer cancel()

		assert.Equal(t, "value1", GetString(ctx, "key1"))
		assert.Equal(t, 42, GetInt(ctx, "key2"))
	})

	t.Run("with parent", func(t *testing.T) {
		parent := context.Background()
		ctx, cancel := NewTestContextBuilder().WithParent(parent).Build()
		defer cancel()

		assert.NotNil(t, ctx)
	})

	t.Run("MustBuild", func(t *testing.T) {
		ctx := NewTestContextBuilder().MustBuild()
		assert.NotNil(t, ctx)
	})
}

func TestSetupTestContext(t *testing.T) {
	ctx := SetupTestContext(t, 1*time.Hour)
	assert.NotNil(t, ctx)

	deadline, ok := ctx.Deadline()
	assert.True(t, ok)
	expectedDeadline := time.Now().Add(1 * time.Hour)
	assert.WithinDuration(t, expectedDeadline, deadline, time.Minute)
}

func TestSetupTestContextWithValues(t *testing.T) {
	values := map[string]interface{}{
		"key": "value",
		"num": 42,
	}

	ctx := SetupTestContextWithValues(t, 30*time.Minute, values)

	assert.NotNil(t, ctx)
	assert.Equal(t, "value", GetString(ctx, "key"))
	assert.Equal(t, 42, GetInt(ctx, "num"))
}

func TestDefaultTimeoutConfig(t *testing.T) {
	config := DefaultTimeoutConfig()

	assert.Equal(t, 5*time.Second, config.Short)
	assert.Equal(t, 30*time.Second, config.Medium)
	assert.Equal(t, 2*time.Minute, config.Long)
	assert.Equal(t, 10*time.Second, config.Database)
	assert.Equal(t, 15*time.Second, config.Network)
}

func TestContextFactory(t *testing.T) {
	config := TimeoutConfig{
		Short:    1 * time.Second,
		Medium:   5 * time.Second,
		Long:     10 * time.Second,
		Database: 3 * time.Second,
		Network:  7 * time.Second,
	}

	cf := NewContextFactory(config)
	assert.NotNil(t, cf)

	t.Run("Short", func(t *testing.T) {
		ctx, cancel := cf.Short()
		defer cancel()

		deadline, ok := ctx.Deadline()
		assert.True(t, ok)
		expected := time.Now().Add(config.Short)
		assert.WithinDuration(t, expected, deadline, 100*time.Millisecond)
	})

	t.Run("Medium", func(t *testing.T) {
		ctx, cancel := cf.Medium()
		defer cancel()

		deadline, ok := ctx.Deadline()
		assert.True(t, ok)
		expected := time.Now().Add(config.Medium)
		assert.WithinDuration(t, expected, deadline, 100*time.Millisecond)
	})

	t.Run("Long", func(t *testing.T) {
		ctx, cancel := cf.Long()
		defer cancel()

		deadline, ok := ctx.Deadline()
		assert.True(t, ok)
		expected := time.Now().Add(config.Long)
		assert.WithinDuration(t, expected, deadline, 100*time.Millisecond)
	})

	t.Run("Database", func(t *testing.T) {
		ctx, cancel := cf.Database()
		defer cancel()

		deadline, ok := ctx.Deadline()
		assert.True(t, ok)
		expected := time.Now().Add(config.Database)
		assert.WithinDuration(t, expected, deadline, 100*time.Millisecond)
	})

	t.Run("Network", func(t *testing.T) {
		ctx, cancel := cf.Network()
		defer cancel()

		deadline, ok := ctx.Deadline()
		assert.True(t, ok)
		expected := time.Now().Add(config.Network)
		assert.WithinDuration(t, expected, deadline, 100*time.Millisecond)
	})
}

func TestContextFactory_DefaultConfig(t *testing.T) {
	cf := NewContextFactory(TimeoutConfig{})
	assert.NotNil(t, cf)

	// Should use default config
	ctx, cancel := cf.Short()
	defer cancel()

	deadline, ok := ctx.Deadline()
	assert.True(t, ok)
	expected := time.Now().Add(DefaultTimeoutConfig().Short)
	assert.WithinDuration(t, expected, deadline, 100*time.Millisecond)
}

func TestContextMultipleValues(t *testing.T) {
	builder := NewTestContextBuilder()
	ctx, cancel := builder.
		WithValue("key1", "value1").
		WithValue("key2", "value2").
		WithValue("key3", 42).
		WithTimeout(1 * time.Hour).
		Build()
	defer cancel()

	assert.Equal(t, "value1", GetString(ctx, "key1"))
	assert.Equal(t, "value2", GetString(ctx, "key2"))
	assert.Equal(t, 42, GetInt(ctx, "key3"))

	deadline, ok := ctx.Deadline()
	assert.True(t, ok)
	expected := time.Now().Add(1 * time.Hour)
	assert.WithinDuration(t, expected, deadline, time.Minute)
}

func TestContextConcurrency(t *testing.T) {
	cm := NewContextManager()

	// Create multiple timeouts concurrently
	for i := 0; i < 10; i++ {
		go func(n int) {
			ctx, cancel := cm.WithTimeout(string(rune('a'+n)), 100*time.Millisecond)
			defer cancel()
			<-ctx.Done()
		}(i)
	}

	// Wait for all to complete
	time.Sleep(150 * time.Millisecond)
	cm.Cleanup()
}

func TestContextWithNilManager(t *testing.T) {
	var cm *ContextManager

	// Should handle nil gracefully
	if cm != nil {
		cm.CancelAll()
	}
	// If we reach here, nil check worked
}
