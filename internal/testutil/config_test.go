// Package testutil tests for config utilities
package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestConfig(t *testing.T) {
	tc := NewTestConfig()
	assert.NotNil(t, tc)
	assert.NotNil(t, tc.values)
	assert.NotNil(t, tc.envBackup)
	assert.NotNil(t, tc.cleanupFuncs)
}

func TestTestConfig_SetAndGet(t *testing.T) {
	tc := NewTestConfig()

	tc.Set("key1", "value1")
	tc.Set("key2", "value2")

	assert.Equal(t, "value1", tc.Get("key1"))
	assert.Equal(t, "value2", tc.Get("key2"))
}

func TestTestConfig_GetDefault(t *testing.T) {
	tc := NewTestConfig()

	tc.Set("existing", "value")

	assert.Equal(t, "value", tc.GetDefault("existing", "default"))
	assert.Equal(t, "default", tc.GetDefault("missing", "default"))
}

func TestTestConfig_GetAll(t *testing.T) {
	tc := NewTestConfig()

	tc.Set("key1", "value1")
	tc.Set("key2", "value2")

	all := tc.GetAll()
	assert.Len(t, all, 2)
	assert.Equal(t, "value1", all["key1"])
	assert.Equal(t, "value2", all["key2"])
}

func TestTestConfig_SetEnv(t *testing.T) {
	tc := NewTestConfig()
	originalValue := os.Getenv("TEST_UTIL_VAR")

	// Set a new value
	tc.SetEnv("TEST_UTIL_VAR", "test-value")

	// Verify it's set
	assert.Equal(t, "test-value", tc.GetEnv("TEST_UTIL_VAR"))

	// Clean up
	tc.Cleanup()

	// Verify original value is restored
	if originalValue != "" {
		assert.Equal(t, originalValue, os.Getenv("TEST_UTIL_VAR"))
	} else {
		_, exists := os.LookupEnv("TEST_UTIL_VAR")
		assert.False(t, exists)
	}
}

func TestTestEnv_UnsetEnv(t *testing.T) {
	tc := NewTestConfig()

	// Set an env var
	os.Setenv("TEST_UTIL_UNSET", "original")
	tc.SetEnv("TEST_UTIL_UNSET", "new-value")

	// Unset it
	tc.UnsetEnv("TEST_UTIL_UNSET")

	// Verify it's unset
	_, exists := os.LookupEnv("TEST_UTIL_UNSET")
	assert.False(t, exists)

	// Clean up should restore
	tc.Cleanup()
	value, exists := os.LookupEnv("TEST_UTIL_UNSET")
	assert.True(t, exists)
	assert.Equal(t, "original", value)

	os.Unsetenv("TEST_UTIL_UNSET")
}

func TestTestConfig_TempDir(t *testing.T) {
	tc := NewTestConfig()

	dir1, err := tc.TempDir()
	require.NoError(t, err)
	assert.NotEmpty(t, dir1)
	assert.True(t, strings.Contains(dir1, "test-"))

	// Calling again should return same dir
	dir2, err := tc.TempDir()
	require.NoError(t, err)
	assert.Equal(t, dir1, dir2)

	// Verify directory exists
	info, err := os.Stat(dir1)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Cleanup should remove it
	tc.Cleanup()
	_, err = os.Stat(dir1)
	assert.True(t, os.IsNotExist(err))
}

func TestTestConfig_MustTempDir(t *testing.T) {
	tc := NewTestConfig()

	dir := tc.MustTempDir()
	assert.NotEmpty(t, dir)

	// Verify it exists
	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestTestConfig_TempFile(t *testing.T) {
	tc := NewTestConfig()

	file, err := tc.TempFile("test-*.txt")
	require.NoError(t, err)
	require.NotNil(t, file)

	assert.True(t, strings.HasPrefix(filepath.Base(file.Name()), "test-"))
	assert.True(t, strings.HasSuffix(filepath.Base(file.Name()), ".txt"))

	file.Close()

	// Verify file exists
	_, err = os.Stat(file.Name())
	assert.NoError(t, err)

	// Clean up
	tc.Cleanup()
	_, err = os.Stat(file.Name())
	assert.True(t, os.IsNotExist(err))
}

func TestTestConfig_MustTempFile(t *testing.T) {
	tc := NewTestConfig()

	file := tc.MustTempFile("data-*.bin")
	assert.NotNil(t, file)
	assert.True(t, strings.HasPrefix(filepath.Base(file.Name()), "data-"))

	file.Close()
}

func TestTestConfig_AddCleanup(t *testing.T) {
	tc := NewTestConfig()

	cleanupCalled := false
	tc.AddCleanup(func() {
		cleanupCalled = true
	})

	tc.Cleanup()
	assert.True(t, cleanupCalled)
}

func TestTestConfig_CleanupOrder(t *testing.T) {
	tc := NewTestConfig()

	var order []string
	tc.AddCleanup(func() { order = append(order, "first") })
	tc.AddCleanup(func() { order = append(order, "second") })
	tc.AddCleanup(func() { order = append(order, "third") })

	tc.Cleanup()

	// Should run in reverse order
	assert.Len(t, order, 3)
	assert.Equal(t, "third", order[0])
	assert.Equal(t, "second", order[1])
	assert.Equal(t, "first", order[2])
}

func TestTestConfig_SetParent(t *testing.T) {
	parent := NewTestConfig()
	parent.Set("key1", "parent-value")

	child := NewTestConfig()
	child.SetParent(parent)

	// Child should get parent's value
	assert.Equal(t, "parent-value", child.Get("key1"))

	// Child can override
	child.Set("key1", "child-value")
	assert.Equal(t, "child-value", child.Get("key1"))

	// Parent should not be affected
	assert.Equal(t, "parent-value", parent.Get("key1"))
}

func TestConfigBuilder(t *testing.T) {
	t.Run("basic building", func(t *testing.T) {
		config := NewConfigBuilder().
			With("key1", "value1").
			With("key2", "value2").
			Build()

		assert.Equal(t, "value1", config.Get("key1"))
		assert.Equal(t, "value2", config.Get("key2"))
	})

	t.Run("with env", func(t *testing.T) {
		config := NewConfigBuilder().
			WithEnv("TEST_CONFIG_BUILDER", "test-value").
			Build()

		assert.Equal(t, "test-value", config.GetEnv("TEST_CONFIG_BUILDER"))
		config.Cleanup()
	})

	t.Run("with database config", func(t *testing.T) {
		config := NewConfigBuilder().
			WithDatabaseConfig("localhost", "5432", "testdb", "user", "pass").
			Build()

		assert.Equal(t, "localhost", config.Get("db.host"))
		assert.Equal(t, "5432", config.Get("db.port"))

		envURL := os.Getenv("DATABASE_URL")
		assert.Contains(t, envURL, "postgres://user:pass@localhost:5432/testdb")

		config.Cleanup()
		os.Unsetenv("DATABASE_URL")
	})

	t.Run("with redis config", func(t *testing.T) {
		config := NewConfigBuilder().
			WithRedisConfig("localhost", "6379", "").
			Build()

		assert.Equal(t, "localhost", config.Get("redis.host"))
		assert.Equal(t, "6379", config.Get("redis.port"))

		envURL := os.Getenv("REDIS_URL")
		assert.Equal(t, "redis://localhost:6379/0", envURL)

		config.Cleanup()
		os.Unsetenv("REDIS_URL")
	})

	t.Run("with S3 config", func(t *testing.T) {
		config := NewConfigBuilder().
			WithS3Config("localhost:9000", "test-bucket", "access", "secret", "us-east-1").
			Build()

		assert.Equal(t, "localhost:9000", config.Get("s3.endpoint"))
		assert.Equal(t, "test-bucket", config.Get("s3.bucket"))

		config.Cleanup()
		os.Unsetenv("S3_ENDPOINT")
		os.Unsetenv("S3_BUCKET")
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		os.Unsetenv("AWS_REGION")
	})

	t.Run("with JWT config", func(t *testing.T) {
		config := NewConfigBuilder().
			WithJWTConfig("my-secret-key").
			Build()

		assert.Equal(t, "my-secret-key", config.Get("jwt.secret"))
		assert.Equal(t, "my-secret-key", os.Getenv("JWT_SECRET"))

		config.Cleanup()
		os.Unsetenv("JWT_SECRET")
	})

	t.Run("with server config", func(t *testing.T) {
		config := NewConfigBuilder().
			WithServerConfig("0.0.0.0", "8080").
			Build()

		assert.Equal(t, "0.0.0.0", config.Get("server.host"))
		assert.Equal(t, "8080", config.Get("server.port"))

		config.Cleanup()
		os.Unsetenv("SERVER_HOST")
		os.Unsetenv("SERVER_PORT")
	})

	t.Run("chaining", func(t *testing.T) {
		config := NewConfigBuilder().
			With("key1", "value1").
			WithEnv("ENV_VAR", "env-value").
			WithDatabaseConfig("host", "port", "db", "user", "pass").
			WithJWTConfig("secret").
			Build()

		assert.Equal(t, "value1", config.Get("key1"))
		assert.Equal(t, "secret", config.Get("jwt.secret"))

		config.Cleanup()
		os.Unsetenv("ENV_VAR")
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("JWT_SECRET")
	})
}

func TestSetupTestConfig(t *testing.T) {
	config := SetupTestConfig(t)

	// Should be able to set values
	config.Set("test-key", "test-value")
	assert.Equal(t, "test-value", config.Get("test-key"))

	// Cleanup should be registered with t
}

func TestSetupTestConfigWithDefaults(t *testing.T) {
	config := SetupTestConfigWithDefaults(t)

	assert.Equal(t, "true", config.GetEnv("TEST_MODE"))
	assert.Equal(t, "debug", config.GetEnv("LOG_LEVEL"))
	assert.NotEmpty(t, config.Get("test.run_id"))
}

func TestConfigLoader_LoadFromEnv(t *testing.T) {
	// Set some env vars with a prefix
	os.Setenv("TESTAPP_HOST", "localhost")
	os.Setenv("TESTAPP_PORT", "8080")
	os.Setenv("TESTAPP_DEBUG", "true")
	defer func() {
		os.Unsetenv("TESTAPP_HOST")
		os.Unsetenv("TESTAPP_PORT")
		os.Unsetenv("TESTAPP_DEBUG")
	}()

	config := NewTestConfig()
	loader := NewConfigLoader(config)
	loader.LoadFromEnv("TESTAPP")

	assert.Equal(t, "localhost", config.Get("HOST"))
	assert.Equal(t, "8080", config.Get("PORT"))
	assert.Equal(t, "true", config.Get("DEBUG"))
}

func TestConfigLoader_LoadFromMap(t *testing.T) {
	values := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	config := NewTestConfig()
	loader := NewConfigLoader(config)
	loader.LoadFromMap(values)

	assert.Equal(t, "value1", config.Get("key1"))
	assert.Equal(t, "value2", config.Get("key2"))
}

func TestDatabaseConfig_ConnectionString(t *testing.T) {
	tests := []struct {
		name     string
		config   DatabaseConfig
		expected string
	}{
		{
			name: "basic connection",
			config: DatabaseConfig{
				Host:     "localhost",
				Port:     "5432",
				Database: "testdb",
				User:     "user",
				Password: "pass",
				SSLMode:  "disable",
			},
			expected: "postgres://user:pass@localhost:5432/testdb?sslmode=disable",
		},
		{
			name: "with SSL",
			config: DatabaseConfig{
				Host:     "db.example.com",
				Port:     "5433",
				Database: "production",
				User:     "admin",
				Password: "secret123",
				SSLMode:  "require",
			},
			expected: "postgres://admin:secret123@db.example.com:5433/production?sslmode=require",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connStr := tt.config.ConnectionString()
			assert.Equal(t, tt.expected, connStr)
		})
	}
}

func TestTestDatabaseConfig(t *testing.T) {
	config := TestDatabaseConfig("localhost", "5432")

	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, "5432", config.Port)
	assert.Equal(t, DefaultTestDatabase, config.Database)
	assert.Equal(t, DefaultTestUser, config.User)
	assert.Equal(t, DefaultTestPassword, config.Password)
	assert.Equal(t, "disable", config.SSLMode)
}

func TestRedisConfig_ConnectionString(t *testing.T) {
	tests := []struct {
		name     string
		config   RedisConfig
		expected string
	}{
		{
			name: "no password",
			config: RedisConfig{
				Host: "localhost",
				Port: "6379",
				DB:   0,
			},
			expected: "redis://localhost:6379/0",
		},
		{
			name: "with password",
			config: RedisConfig{
				Host:     "redis.example.com",
				Port:     "6380",
				Password: "secret",
				DB:       2,
			},
			expected: "redis://:secret@redis.example.com:6380/2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connStr := tt.config.ConnectionString()
			assert.Equal(t, tt.expected, connStr)
		})
	}
}

func TestRedisConfig_Addr(t *testing.T) {
	config := RedisConfig{
		Host: "localhost",
		Port: "6379",
	}

	assert.Equal(t, "localhost:6379", config.Addr())
}

func TestTestRedisConfig(t *testing.T) {
	config := TestRedisConfig("localhost", "6379")

	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, "6379", config.Port)
	assert.Equal(t, 0, config.DB)
}

func TestS3Config_EndpointURL(t *testing.T) {
	tests := []struct {
		name     string
		config   S3Config
		expected string
	}{
		{
			name: "HTTP",
			config: S3Config{
				Endpoint: "localhost:9000",
				UseHTTPS: false,
			},
			expected: "http://localhost:9000",
		},
		{
			name: "HTTPS",
			config: S3Config{
				Endpoint: "s3.example.com",
				UseHTTPS: true,
			},
			expected: "https://s3.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := tt.config.EndpointURL()
			assert.Equal(t, tt.expected, url)
		})
	}
}

func TestTestS3Config(t *testing.T) {
	config := TestS3Config("localhost", "9000")

	assert.Equal(t, "localhost:9000", config.Endpoint)
	assert.Equal(t, "test-bucket", config.Bucket)
	assert.Equal(t, DefaultS3AccessKey, config.AccessKey)
	assert.Equal(t, DefaultS3SecretKey, config.SecretKey)
	assert.Equal(t, DefaultS3Region, config.Region)
	assert.False(t, config.UseHTTPS)
}

func TestTestJWTConfig(t *testing.T) {
	config := TestJWTConfig()

	assert.Equal(t, DefaultTestSecret, config.SecretKey)
	assert.Equal(t, "15m", config.AccessDuration)
	assert.Equal(t, "168h", config.RefreshDuration)
	assert.Equal(t, DefaultTestIssuer, config.Issuer)
}

func TestTestServerConfig(t *testing.T) {
	config := TestServerConfig()

	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, "8080", config.Port)
	assert.Equal(t, "30s", config.ReadTimeout)
	assert.Equal(t, "30s", config.WriteTimeout)
	assert.Equal(t, "120s", config.IdleTimeout)
}

func TestIsTestMode(t *testing.T) {
	// Default should be false
	assert.False(t, IsTestMode())

	// Set and check
	os.Setenv("TEST_MODE", "true")
	defer os.Unsetenv("TEST_MODE")

	assert.True(t, IsTestMode())
}

func TestGetTestTimeout(t *testing.T) {
	t.Run("from env", func(t *testing.T) {
		os.Setenv("TEST_TIMEOUT", "30s")
		defer os.Unsetenv("TEST_TIMEOUT")

		timeout := GetTestTimeout("TEST_TIMEOUT", 10)
		assert.Equal(t, 30*time.Second, timeout)
	})

	t.Run("default", func(t *testing.T) {
		timeout := GetTestTimeout("NONEXISTENT_TIMEOUT", 10)
		assert.Equal(t, 10*time.Second, timeout)
	})

	t.Run("invalid env value", func(t *testing.T) {
		os.Setenv("TEST_TIMEOUT", "invalid")
		defer os.Unsetenv("TEST_TIMEOUT")

		timeout := GetTestTimeout("TEST_TIMEOUT", 10)
		assert.Equal(t, 10*time.Second, timeout)
	})
}

func TestFindProjectRoot(t *testing.T) {
	root, err := FindProjectRoot()
	if err != nil {
		t.Skip("Not in a Go module")
	}

	// Should find go.mod
	gomodPath := filepath.Join(root, "go.mod")
	_, err = os.Stat(gomodPath)
	assert.NoError(t, err)
}

func TestRandomString(t *testing.T) {
	s := RandomString(10)
	assert.Len(t, s, 10)

	// Should only contain lowercase letters and numbers
	for _, c := range s {
		assert.True(t, (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9'))
	}

	// Should be different each time (mostly)
	s2 := RandomString(10)
	// Very unlikely to be the same
	assert.NotEqual(t, s, s2)
}

func TestSetupIsolatedTestEnv(t *testing.T) {
	config, tempDir := SetupIsolatedTestEnv(t)

	assert.NotNil(t, config)
	assert.NotEmpty(t, tempDir)
	assert.True(t, strings.Contains(tempDir, "test-"))
	assert.Equal(t, "true", config.GetEnv("TEST_MODE"))
	assert.Equal(t, tempDir, config.Get("test.temp_dir"))

	// Verify temp dir exists
	info, err := os.Stat(tempDir)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())

	// Temp dir should be cleaned up after test
}

func TestTestConfig_MultipleEnvBackups(t *testing.T) {
	tc := NewTestConfig()

	// Set env var multiple times
	tc.SetEnv("MULTI_BACKUP", "value1")
	assert.Equal(t, "value1", tc.GetEnv("MULTI_BACKUP"))

	tc.SetEnv("MULTI_BACKUP", "value2")
	assert.Equal(t, "value2", tc.GetEnv("MULTI_BACKUP"))

	tc.SetEnv("MULTI_BACKUP", "value3")
	assert.Equal(t, "value3", tc.GetEnv("MULTI_BACKUP"))

	// Clean up should restore original (which was empty)
	tc.Cleanup()
	_, exists := os.LookupEnv("MULTI_BACKUP")
	assert.False(t, exists)
}

func TestTestConfig_NestedCleanup(t *testing.T) {
	parent := NewTestConfig()
	child := NewTestConfig()
	child.SetParent(parent)

	parentCalled := false
	childCalled := false

	parent.AddCleanup(func() { parentCalled = true })
	child.AddCleanup(func() { childCalled = true })

	// Only cleaning up child
	child.Cleanup()

	assert.True(t, childCalled)
	assert.False(t, parentCalled)

	// Now clean up parent
	parent.Cleanup()
	assert.True(t, parentCalled)
}

func TestTestConfig_ConcurrentAccess(t *testing.T) {
	tc := NewTestConfig()

	done := make(chan bool)

	// Concurrent sets
	for i := 0; i < 10; i++ {
		go func(n int) {
			tc.Set("key", string(rune('a'+n)))
			done <- true
		}(i)
	}

	// Wait for all
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have a value
	_ = tc.Get("key")
}

func TestTestConfig_EmptyStringValues(t *testing.T) {
	tc := NewTestConfig()

	tc.Set("empty", "")
	assert.Equal(t, "", tc.Get("empty"))
	assert.Equal(t, "default", tc.GetDefault("empty", "default"))
}

func TestTestConfig_GetAllWithParent(t *testing.T) {
	parent := NewTestConfig()
	parent.Set("parent-key", "parent-value")

	child := NewTestConfig()
	child.SetParent(parent)
	child.Set("child-key", "child-value")

	all := child.GetAll()

	assert.Contains(t, all, "parent-key")
	assert.Contains(t, all, "child-key")
	assert.Equal(t, "parent-value", all["parent-key"])
	assert.Equal(t, "child-value", all["child-key"])
}

func TestSplitEnv(t *testing.T) {
	tests := []struct {
		input    string
		wantKey  string
		wantVal  string
		wantOk   bool
	}{
		{"KEY=value", "KEY", "value", true},
		{"KEY=", "KEY", "", true},
		{"=value", "", "value", true}, // Fixed: "=value" returns value "value", not empty
		{"NOEQUALS", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			key, val, ok := splitEnv(tt.input)
			assert.Equal(t, tt.wantKey, key)
			assert.Equal(t, tt.wantVal, val)
			assert.Equal(t, tt.wantOk, ok)
		})
	}
}
