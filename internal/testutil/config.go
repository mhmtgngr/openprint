// Package testutil provides test configuration management with environment variable isolation.
package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestConfig holds configuration for a test environment.
type TestConfig struct {
	mu              sync.RWMutex
	values          map[string]string
	envBackup       map[string]string
	tempDir         string
	cleanupFuncs    []func()
	configPath      string
	parent          *TestConfig
}

// NewTestConfig creates a new test configuration.
func NewTestConfig() *TestConfig {
	return &TestConfig{
		values:       make(map[string]string),
		envBackup:    make(map[string]string),
		cleanupFuncs: make([]func(), 0),
	}
}

// Set sets a configuration value.
func (tc *TestConfig) Set(key, value string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.values[key] = value
}

// Get gets a configuration value.
func (tc *TestConfig) Get(key string) string {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	if val, ok := tc.values[key]; ok {
		return val
	}

	// Check parent config
	if tc.parent != nil {
		return tc.parent.Get(key)
	}

	return ""
}

// GetDefault gets a configuration value with a default.
func (tc *TestConfig) GetDefault(key, defaultValue string) string {
	if val := tc.Get(key); val != "" {
		return val
	}
	return defaultValue
}

// GetAll returns all configuration values.
func (tc *TestConfig) GetAll() map[string]string {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	result := make(map[string]string)
	for k, v := range tc.values {
		result[k] = v
	}

	// Include parent values
	if tc.parent != nil {
		for k, v := range tc.parent.GetAll() {
			if _, exists := result[k]; !exists {
				result[k] = v
			}
		}
	}

	return result
}

// SetEnv sets an environment variable for the test.
// The original value is restored on cleanup.
func (tc *TestConfig) SetEnv(key, value string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Backup original value if not already backed up
	if _, exists := tc.envBackup[key]; !exists {
		if originalValue, exists := os.LookupEnv(key); exists {
			tc.envBackup[key] = originalValue
		}
	}

	// Set the new value
	if err := os.Setenv(key, value); err != nil {
		panic(fmt.Sprintf("failed to set env var %s: %v", key, err))
	}
}

// GetEnv gets an environment variable value.
func (tc *TestConfig) GetEnv(key string) string {
	return os.Getenv(key)
}

// UnsetEnv removes an environment variable for the test.
// The original value is restored on cleanup.
func (tc *TestConfig) UnsetEnv(key string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Backup original value if not already backed up
	if _, exists := tc.envBackup[key]; !exists {
		if originalValue, exists := os.LookupEnv(key); exists {
			tc.envBackup[key] = originalValue
		}
	}

	// Unset the variable
	os.Unsetenv(key)
}

// TempDir creates a temporary directory for the test.
// The directory is removed on cleanup.
func (tc *TestConfig) TempDir() (string, error) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.tempDir != "" {
		return tc.tempDir, nil
	}

	tempDir, err := os.MkdirTemp("", "test-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}

	tc.tempDir = tempDir
	tc.AddCleanup(func() {
		os.RemoveAll(tempDir)
	})

	return tempDir, nil
}

// MustTempDir creates a temporary directory and panics on error.
func (tc *TestConfig) MustTempDir() string {
	dir, err := tc.TempDir()
	if err != nil {
		panic(err)
	}
	return dir
}

// TempFile creates a temporary file for the test.
func (tc *TestConfig) TempFile(pattern string) (*os.File, error) {
	tempDir, err := tc.TempDir()
	if err != nil {
		return nil, err
	}

	return os.CreateTemp(tempDir, pattern)
}

// MustTempFile creates a temporary file and panics on error.
func (tc *TestConfig) MustTempFile(pattern string) *os.File {
	f, err := tc.TempFile(pattern)
	if err != nil {
		panic(err)
	}
	return f
}

// AddCleanup adds a cleanup function to be called on Cleanup.
func (tc *TestConfig) AddCleanup(fn func()) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.cleanupFuncs = append(tc.cleanupFuncs, fn)
}

// Cleanup runs all cleanup functions and restores environment variables.
func (tc *TestConfig) Cleanup() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Run cleanup functions in reverse order
	for i := len(tc.cleanupFuncs) - 1; i >= 0; i-- {
		fn := tc.cleanupFuncs[i]
		if fn != nil {
			fn()
		}
	}
	tc.cleanupFuncs = make([]func(), 0)

	// Restore environment variables
	for key, value := range tc.envBackup {
		if err := os.Setenv(key, value); err != nil {
			// Log but don't panic during cleanup
			fmt.Printf("Warning: failed to restore env var %s: %v\n", key, err)
		}
	}
	tc.envBackup = make(map[string]string)

	// Clear temp dir reference (already removed by cleanup func)
	tc.tempDir = ""
}

// SetParent sets a parent configuration that provides default values.
func (tc *TestConfig) SetParent(parent *TestConfig) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.parent = parent
}

// ConfigBuilder provides a fluent interface for building test configuration.
type ConfigBuilder struct {
	config *TestConfig
}

// NewConfigBuilder creates a new config builder.
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		config: NewTestConfig(),
	}
}

// With sets a configuration value.
func (cb *ConfigBuilder) With(key, value string) *ConfigBuilder {
	cb.config.Set(key, value)
	return cb
}

// WithEnv sets an environment variable.
func (cb *ConfigBuilder) WithEnv(key, value string) *ConfigBuilder {
	cb.config.SetEnv(key, value)
	return cb
}

// WithDatabaseConfig sets common database configuration.
func (cb *ConfigBuilder) WithDatabaseConfig(host, port, database, user, password string) *ConfigBuilder {
	cb.config.Set("db.host", host)
	cb.config.Set("db.port", port)
	cb.config.Set("db.database", database)
	cb.config.Set("db.user", user)
	cb.config.Set("db.password", password)
	cb.config.SetEnv("DATABASE_URL", fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, database))
	return cb
}

// WithRedisConfig sets common Redis configuration.
func (cb *ConfigBuilder) WithRedisConfig(host, port, password string) *ConfigBuilder {
	cb.config.Set("redis.host", host)
	cb.config.Set("redis.port", port)
	cb.config.Set("redis.password", password)
	cb.config.SetEnv("REDIS_URL", fmt.Sprintf("redis://:%s@%s:%s", password, host, port))
	return cb
}

// WithS3Config sets common S3 configuration.
func (cb *ConfigBuilder) WithS3Config(endpoint, bucket, accessKey, secretKey, region string) *ConfigBuilder {
	cb.config.Set("s3.endpoint", endpoint)
	cb.config.Set("s3.bucket", bucket)
	cb.config.Set("s3.access_key", accessKey)
	cb.config.Set("s3.secret_key", secretKey)
	cb.config.Set("s3.region", region)
	cb.config.SetEnv("S3_ENDPOINT", endpoint)
	cb.config.SetEnv("S3_BUCKET", bucket)
	cb.config.SetEnv("AWS_ACCESS_KEY_ID", accessKey)
	cb.config.SetEnv("AWS_SECRET_ACCESS_KEY", secretKey)
	cb.config.SetEnv("AWS_REGION", region)
	return cb
}

// WithJWTConfig sets JWT configuration.
func (cb *ConfigBuilder) WithJWTConfig(secret string) *ConfigBuilder {
	cb.config.Set("jwt.secret", secret)
	cb.config.SetEnv("JWT_SECRET", secret)
	return cb
}

// WithServerConfig sets server configuration.
func (cb *ConfigBuilder) WithServerConfig(host, port string) *ConfigBuilder {
	cb.config.Set("server.host", host)
	cb.config.Set("server.port", port)
	cb.config.SetEnv("SERVER_HOST", host)
	cb.config.SetEnv("SERVER_PORT", port)
	return cb
}

// Build returns the built configuration.
func (cb *ConfigBuilder) Build() *TestConfig {
	return cb.config
}

// SetupTestConfig creates a test configuration and registers cleanup with testing.T.
func SetupTestConfig(t *testing.T) *TestConfig {
	config := NewTestConfig()
	t.Cleanup(config.Cleanup)
	return config
}

// SetupTestConfigWithDefaults creates a test configuration with default values.
func SetupTestConfigWithDefaults(t *testing.T) *TestConfig {
	config := NewTestConfig()

	// Set default test values
	config.SetEnv("TEST_MODE", "true")
	config.SetEnv("LOG_LEVEL", "debug")
	config.Set("test.run_id", RandomString(8))

	t.Cleanup(config.Cleanup)
	return config
}

// ConfigLoader loads configuration from different sources.
type ConfigLoader struct {
	config *TestConfig
}

// NewConfigLoader creates a new config loader.
func NewConfigLoader(config *TestConfig) *ConfigLoader {
	if config == nil {
		config = NewTestConfig()
	}
	return &ConfigLoader{config: config}
}

// LoadFromEnv loads configuration from environment variables with a prefix.
func (cl *ConfigLoader) LoadFromEnv(prefix string) {
	for _, env := range os.Environ() {
		key, value, _ := splitEnv(env)
		if hasPrefix(key, prefix) {
			configKey := trimPrefix(key, prefix+"_")
			cl.config.Set(configKey, value)
		}
	}
}

// LoadFromMap loads configuration from a map.
func (cl *ConfigLoader) LoadFromMap(values map[string]string) {
	for k, v := range values {
		cl.config.Set(k, v)
	}
}

// Config returns the loaded configuration.
func (cl *ConfigLoader) Config() *TestConfig {
	return cl.config
}

// DatabaseConfig holds database configuration.
type DatabaseConfig struct {
	Host     string
	Port     string
	Database string
	User     string
	Password string
	SSLMode  string
}

// ConnectionString returns the PostgreSQL connection string.
func (dc *DatabaseConfig) ConnectionString() string {
	if dc.SSLMode == "" {
		dc.SSLMode = "disable"
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		dc.User, dc.Password, dc.Host, dc.Port, dc.Database, dc.SSLMode)
}

// TestDatabaseConfig returns default test database configuration.
func TestDatabaseConfig(host, port string) *DatabaseConfig {
	return &DatabaseConfig{
		Host:     host,
		Port:     port,
		Database: DefaultTestDatabase,
		User:     DefaultTestUser,
		Password: DefaultTestPassword,
		SSLMode:  "disable",
	}
}

// RedisConfig holds Redis configuration.
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// ConnectionString returns the Redis connection string.
func (rc *RedisConfig) ConnectionString() string {
	if rc.Password == "" {
		return fmt.Sprintf("redis://%s:%s/%d", rc.Host, rc.Port, rc.DB)
	}
	return fmt.Sprintf("redis://:%s@%s:%s/%d", rc.Password, rc.Host, rc.Port, rc.DB)
}

// Addr returns the Redis address in host:port format.
func (rc *RedisConfig) Addr() string {
	return rc.Host + ":" + rc.Port
}

// TestRedisConfig returns default test Redis configuration.
func TestRedisConfig(host, port string) *RedisConfig {
	return &RedisConfig{
		Host: host,
		Port: port,
		DB:   0,
	}
}

// S3Config holds S3 configuration.
type S3Config struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	Region    string
	UseHTTPS  bool
}

// EndpointURL returns the S3 endpoint URL.
func (sc *S3Config) EndpointURL() string {
	scheme := "http"
	if sc.UseHTTPS {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, sc.Endpoint)
}

// TestS3Config returns default test S3 configuration.
func TestS3Config(host, port string) *S3Config {
	return &S3Config{
		Endpoint:  host + ":" + port,
		Bucket:    "test-bucket",
		AccessKey: DefaultS3AccessKey,
		SecretKey: DefaultS3SecretKey,
		Region:    DefaultS3Region,
		UseHTTPS:  false,
	}
}

// JWTConfig holds JWT configuration.
type JWTConfig struct {
	SecretKey       string
	AccessDuration  string
	RefreshDuration string
	Issuer          string
}

// TestJWTConfig returns default test JWT configuration.
func TestJWTConfig() *JWTConfig {
	return &JWTConfig{
		SecretKey:       DefaultTestSecret,
		AccessDuration:  "15m",
		RefreshDuration: "168h", // 7 days
		Issuer:          DefaultTestIssuer,
	}
}

// ServerConfig holds server configuration.
type ServerConfig struct {
	Host         string
	Port         string
	ReadTimeout  string
	WriteTimeout string
	IdleTimeout  string
}

// TestServerConfig returns default test server configuration.
func TestServerConfig() *ServerConfig {
	return &ServerConfig{
		Host:         "localhost",
		Port:         "8080",
		ReadTimeout:  "30s",
		WriteTimeout: "30s",
		IdleTimeout:  "120s",
	}
}

// IsTestMode returns true if running in test mode.
func IsTestMode() bool {
	return os.Getenv("TEST_MODE") == "true"
}

// GetTestTimeout returns a timeout duration from environment or default.
func GetTestTimeout(key string, defaultTimeout int) time.Duration {
	if val := os.Getenv(key); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			return duration
		}
	}
	return time.Duration(defaultTimeout) * time.Second
}

// FindProjectRoot finds the project root directory by looking for go.mod.
func FindProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		gomod := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(gomod); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("project root not found (go.mod not found)")
		}
		dir = parent
	}
}

// splitEnv splits an environment variable string into key and value.
func splitEnv(s string) (key, value string, ok bool) {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return s[:i], s[i+1:], true
		}
	}
	return "", "", false
}

// hasPrefix checks if a string has a prefix (case-insensitive for env vars).
func hasPrefix(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	return s[:len(prefix)] == prefix
}

// trimPrefix removes a prefix from a string.
func trimPrefix(s, prefix string) string {
	if len(s) < len(prefix) {
		return s
	}
	return s[len(prefix):]
}

// RandomString generates a random string for testing.
func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	seed := intSeed()
	for i := range b {
		b[i] = charset[int(seed)%len(charset)]
	}
	return string(b)
}

// intSeed returns a pseudo-random seed for string generation.
// This uses time for simplicity in tests.
func intSeed() int64 {
	return time.Now().UnixNano()
}

// SetupIsolatedTestEnv creates an isolated test environment with:
// - Temporary directory
// - Isolated environment variables
// - Cleanup registration
func SetupIsolatedTestEnv(t *testing.T) (*TestConfig, string) {
	config := NewTestConfig()

	// Create temp directory
	tempDir := config.MustTempDir()

	// Set test mode
	config.SetEnv("TEST_MODE", "true")
	config.SetEnv("TEST_TEMP_DIR", tempDir)
	config.Set("test.temp_dir", tempDir)

	// Register cleanup
	t.Cleanup(config.Cleanup)

	return config, tempDir
}
