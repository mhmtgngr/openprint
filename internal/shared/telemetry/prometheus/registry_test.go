// Package prometheus provides tests for the Prometheus registry.
package prometheus

import (
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistry(t *testing.T) {
	t.Run("creates registry with default config", func(t *testing.T) {
		cfg := Config{
			ServiceName: "test-service",
		}

		reg, err := NewRegistry(cfg)

		require.NoError(t, err)
		assert.NotNil(t, reg)
		assert.Equal(t, "test-service", reg.ServiceName())
		assert.Equal(t, "", reg.ServiceVersion()) // Empty when not set
		assert.NotNil(t, reg.Registry())
	})

	t.Run("creates registry with custom config", func(t *testing.T) {
		cfg := Config{
			ServiceName:    "custom-service",
			ServiceVersion: "2.0.0",
			Namespace:      "custom-namespace",
			Labels: prometheus.Labels{
				"environment": "test",
				"region":      "us-west",
			},
		}

		reg, err := NewRegistry(cfg)

		require.NoError(t, err)
		assert.Equal(t, "custom-service", reg.ServiceName())
		assert.Equal(t, "2.0.0", reg.ServiceVersion())

		labels := reg.Labels()
		assert.Equal(t, "custom-service", labels["service_name"])
		assert.Equal(t, "custom-namespace", labels["namespace"])
		assert.Equal(t, "2.0.0", labels["service_version"])
		assert.Equal(t, "test", labels["environment"])
		assert.Equal(t, "us-west", labels["region"])
	})

	t.Run("uses default service name when empty", func(t *testing.T) {
		cfg := Config{}

		reg, err := NewRegistry(cfg)

		require.NoError(t, err)
		assert.Equal(t, defaultServiceName, reg.ServiceName())
	})

	t.Run("uses default namespace when empty", func(t *testing.T) {
		cfg := Config{
			ServiceName: "test",
			Namespace:   "",
		}

		reg, err := NewRegistry(cfg)

		require.NoError(t, err)
		labels := reg.Labels()
		assert.Equal(t, "openprint", labels["namespace"])
	})

	t.Run("includes default collectors", func(t *testing.T) {
		cfg := Config{
			ServiceName: "test-service",
		}

		reg, err := NewRegistry(cfg)

		require.NoError(t, err)
		// Verify default collectors are registered
		// The registry should have go_collector and process_collector
		assert.NotNil(t, reg.Registry())
	})
}

func TestRegistry_MustRegister(t *testing.T) {
	t.Run("registers collector successfully", func(t *testing.T) {
		cfg := Config{ServiceName: "test"}
		reg, err := NewRegistry(cfg)
		require.NoError(t, err)

		collector := prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "test_counter",
				Help: "A test counter",
			},
			[]string{"label"},
		)

		// Should not panic
		assert.NotPanics(t, func() {
			reg.MustRegister(collector, "test_counter")
		})

		// Registering again should not panic (duplicate detection)
		assert.NotPanics(t, func() {
			reg.MustRegister(collector, "test_counter")
		})
	})

	t.Run("handles duplicate registration", func(t *testing.T) {
		cfg := Config{ServiceName: "test"}
		reg, err := NewRegistry(cfg)
		require.NoError(t, err)

		collector1 := prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "test_counter_unique",
				Help: "A test counter",
			},
		)

		collector2 := prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "test_counter_unique2",
				Help: "Another test counter",
			},
		)

		reg.MustRegister(collector1, "unique_collector")
		// Different collector with same name should be skipped
		reg.MustRegister(collector2, "unique_collector")

		// Verify only first was registered
		assert.True(t, reg.Unregister("unique_collector"))
		assert.False(t, reg.Unregister("unique_collector"))
	})
}

func TestRegistry_Register(t *testing.T) {
	t.Run("registers and returns nil on success", func(t *testing.T) {
		cfg := Config{ServiceName: "test"}
		reg, err := NewRegistry(cfg)
		require.NoError(t, err)

		collector := prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "test_counter_register",
				Help: "A test counter",
			},
		)

		err = reg.Register(collector, "test_counter_register")
		assert.NoError(t, err)
	})

	t.Run("returns nil for duplicate registration", func(t *testing.T) {
		cfg := Config{ServiceName: "test"}
		reg, err := NewRegistry(cfg)
		require.NoError(t, err)

		collector := prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "test_counter_dup",
				Help: "A test counter",
			},
		)

		err = reg.Register(collector, "test_counter_dup")
		assert.NoError(t, err)

		// Register again - should return nil (no error for duplicates)
		err = reg.Register(collector, "test_counter_dup")
		assert.NoError(t, err)
	})
}

func TestRegistry_Unregister(t *testing.T) {
	t.Run("unregisters existing collector", func(t *testing.T) {
		cfg := Config{ServiceName: "test"}
		reg, err := NewRegistry(cfg)
		require.NoError(t, err)

		collector := prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "test_counter_unregister",
				Help: "A test counter",
			},
		)

		reg.MustRegister(collector, "unregister_test")

		unregistered := reg.Unregister("unregister_test")
		assert.True(t, unregistered)

		// Second unregister should return false
		unregistered = reg.Unregister("unregister_test")
		assert.False(t, unregistered)
	})

	t.Run("returns false for non-existent collector", func(t *testing.T) {
		cfg := Config{ServiceName: "test"}
		reg, err := NewRegistry(cfg)
		require.NoError(t, err)

		unregistered := reg.Unregister("non_existent")
		assert.False(t, unregistered)
	})
}

func TestRegistry_ServiceName(t *testing.T) {
	cfg := Config{ServiceName: "my-service"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	assert.Equal(t, "my-service", reg.ServiceName())
}

func TestRegistry_ServiceVersion(t *testing.T) {
	t.Run("returns custom version", func(t *testing.T) {
		cfg := Config{
			ServiceName:    "test",
			ServiceVersion: "3.2.1",
		}
		reg, err := NewRegistry(cfg)
		require.NoError(t, err)

		assert.Equal(t, "3.2.1", reg.ServiceVersion())
	})

	t.Run("returns empty when not set", func(t *testing.T) {
		cfg := Config{
			ServiceName: "test",
		}
		reg, err := NewRegistry(cfg)
		require.NoError(t, err)

		// ServiceVersion returns the value directly, empty if not set
		assert.Equal(t, "", reg.ServiceVersion())
	})
}

func TestRegistry_Labels(t *testing.T) {
	cfg := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.5.0",
		Namespace:      "test-ns",
		Labels: prometheus.Labels{
			"custom": "value",
		},
	}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	labels := reg.Labels()

	assert.Equal(t, "test-service", labels["service_name"])
	assert.Equal(t, "test-ns", labels["namespace"])
	assert.Equal(t, "1.5.0", labels["service_version"])
	assert.Equal(t, "value", labels["custom"])
}

func TestRegistry_MergeLabels(t *testing.T) {
	cfg := Config{
		ServiceName: "base-service",
		Labels: prometheus.Labels{
			"base_label": "base_value",
		},
	}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	additional := prometheus.Labels{
		"new_label":    "new_value",
		"base_label":   "overridden", // Registry labels take precedence
		"another_base": "another_value",
	}

	merged := reg.MergeLabels(additional)

	assert.Equal(t, "base_value", merged["base_label"]) // Registry wins
	assert.Equal(t, "new_value", merged["new_label"])
	assert.Equal(t, "another_value", merged["another_base"])
	assert.Equal(t, "base-service", merged["service_name"])
}

func TestRegistry_Registry(t *testing.T) {
	cfg := Config{ServiceName: "test"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	assert.NotNil(t, reg.Registry())
	assert.IsType(t, &prometheus.Registry{}, reg.Registry())
}

func TestGlobalRegistry(t *testing.T) {
	// Reset global state before and after test
	ResetGlobalRegistry()
	defer ResetGlobalRegistry()

	t.Run("GetRegistry returns error when not initialized", func(t *testing.T) {
		_, err := GetRegistry()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no global registry")
	})

	t.Run("SetRegistry and GetRegistry", func(t *testing.T) {
		cfg := Config{ServiceName: "global-test"}
		reg, err := NewRegistry(cfg)
		require.NoError(t, err)

		SetRegistry(reg)

		retrieved, err := GetRegistry()
		assert.NoError(t, err)
		assert.Same(t, reg, retrieved)
	})

	t.Run("MustGetRegistry panics when not initialized", func(t *testing.T) {
		ResetGlobalRegistry()

		assert.Panics(t, func() {
			MustGetRegistry()
		})
	})

	t.Run("MustGetRegistry returns registry when initialized", func(t *testing.T) {
		ResetGlobalRegistry()

		cfg := Config{ServiceName: "must-get-test"}
		reg, err := NewRegistry(cfg)
		require.NoError(t, err)

		SetRegistry(reg)

		retrieved := MustGetRegistry()
		assert.Same(t, reg, retrieved)
	})

	t.Run("ResetGlobalRegistry clears the global registry", func(t *testing.T) {
		cfg := Config{ServiceName: "reset-test"}
		reg, err := NewRegistry(cfg)
		require.NoError(t, err)

		SetRegistry(reg)
		ResetGlobalRegistry()

		_, err = GetRegistry()
		assert.Error(t, err)
	})
}

func TestGlobalRegistry_ConcurrentAccess(t *testing.T) {
	ResetGlobalRegistry()
	defer ResetGlobalRegistry()

	cfg := Config{ServiceName: "concurrent-test"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	// Concurrent writes and reads
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			SetRegistry(reg)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			GetRegistry()
		}()
	}

	wg.Wait()

	// Should have a registry without race
	retrieved, err := GetRegistry()
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
}

func TestDefaultConfig(t *testing.T) {
	t.Run("creates default config", func(t *testing.T) {
		cfg := DefaultConfig("my-service")

		assert.Equal(t, "my-service", cfg.ServiceName)
		assert.Equal(t, "1.0.0", cfg.ServiceVersion)
		assert.Equal(t, "openprint", cfg.Namespace)
		assert.NotNil(t, cfg.Labels)
		assert.Empty(t, cfg.Labels)
	})
}

func TestRegistry_ConcurrentRegistration(t *testing.T) {
	cfg := Config{ServiceName: "concurrent-reg-test"}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	var wg sync.WaitGroup

	// Register multiple collectors concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			collector := prometheus.NewCounter(
				prometheus.CounterOpts{
					Name: "concurrent_counter_test",
					Help: "A concurrent test counter",
				},
			)

			name := "concurrent_test"
			reg.MustRegister(collector, name)
		}(i)
	}

	wg.Wait()

	// Should complete without deadlock or race
	assert.NotNil(t, reg.Registry())
}

func TestRegistry_LabelsThreadSafety(t *testing.T) {
	cfg := Config{
		ServiceName: "labels-test",
		Labels: prometheus.Labels{
			"base": "value",
		},
	}
	reg, err := NewRegistry(cfg)
	require.NoError(t, err)

	var wg sync.WaitGroup

	// Concurrent reads of labels
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			labels := reg.Labels()
			_ = labels
		}()
	}

	// Concurrent merges
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			extra := prometheus.Labels{
				"extra":  string(rune(idx)),
				"extra2": string(rune(idx + 1)),
			}
			_ = reg.MergeLabels(extra)
		}(i)
	}

	wg.Wait()

	// Verify base labels are intact
	labels := reg.Labels()
	assert.Equal(t, "value", labels["base"])
}

func TestRegistry_MultipleServiceIsolation(t *testing.T) {
	// Create multiple registries for different services
	cfg1 := Config{ServiceName: "service-1"}
	reg1, err := NewRegistry(cfg1)
	require.NoError(t, err)

	cfg2 := Config{ServiceName: "service-2"}
	reg2, err := NewRegistry(cfg2)
	require.NoError(t, err)

	// Register collectors with same name in different registries
	collector1 := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "shared_name_counter",
			Help: "Counter in registry 1",
		},
	)

	collector2 := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "shared_name_counter",
			Help: "Counter in registry 2",
		},
	)

	reg1.MustRegister(collector1, "shared_counter")
	reg2.MustRegister(collector2, "shared_counter")

	// Registries should be independent
	assert.NotSame(t, reg1.Registry(), reg2.Registry())
	assert.Equal(t, "service-1", reg1.ServiceName())
	assert.Equal(t, "service-2", reg2.ServiceName())
}
