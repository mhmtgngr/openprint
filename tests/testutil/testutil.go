// Package testutil provides shared testing utilities for OpenPrint Cloud.
// It includes helpers for database setup, HTTP testing, and test fixtures.
package testutil

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// TestDB wraps a database connection for testing.
type TestDB struct {
	Conn     *pgx.Conn
	Pool     *pgxpool.Pool
	Dsn      string
	Cleanup  func()
}

// TestHTTPClient wraps an HTTP client for testing.
type TestHTTPClient struct {
	Client       *http.Client
	BaseURL      string
	AuthToken    string
	Headers      map[string]string
	Interceptors []InterceptorFunc
}

// InterceptorFunc is a function that can intercept and modify HTTP requests.
type InterceptorFunc func(*http.Request) (*http.Request, error)

// TestConfig holds test configuration.
type TestConfig struct {
	DatabaseURL string
	BaseURL     string
	Timeout     time.Duration
}

// GetTestConfig returns test configuration from environment or defaults.
func GetTestConfig() *TestConfig {
	return &TestConfig{
		DatabaseURL: getEnv("TEST_DATABASE_URL", "postgres://openprint:openprint@localhost:15432/openprint_test?sslmode=disable"),
		BaseURL:     getEnv("TEST_BASE_URL", "http://localhost:8000"),
		Timeout:     30 * time.Second,
	}
}

// getEnv retrieves an environment variable or returns the default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// SetupTestDB creates a test database connection.
// It returns a TestDB struct that includes a cleanup function.
func SetupTestDB(t *testing.T) *TestDB {
	t.Helper()

	config := GetTestConfig()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, config.DatabaseURL)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		conn.Close(ctx)
	}

	return &TestDB{
		Conn:    conn,
		Dsn:     config.DatabaseURL,
		Cleanup: cleanup,
	}
}

// SetupTestDBPool creates a test database connection pool.
func SetupTestDBPool(t *testing.T) *TestDB {
	t.Helper()

	config := GetTestConfig()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(config.DatabaseURL)
	if err != nil {
		t.Fatalf("failed to parse database config: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("failed to create connection pool: %v", err)
	}

	cleanup := func() {
		pool.Close()
	}

	return &TestDB{
		Pool:    pool,
		Dsn:     config.DatabaseURL,
		Cleanup: cleanup,
	}
}

// TeardownTestDB closes the test database connection.
func TeardownTestDB(t *testing.T, db *TestDB) {
	t.Helper()
	if db.Cleanup != nil {
		db.Cleanup()
	}
}

// InTransaction runs a function within a database transaction.
// The transaction is rolled back after the function completes.
func InTransaction(t *testing.T, db *TestDB, fn func(pgx.Tx) error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := db.Conn.Begin(ctx)
	require.NoError(t, err, "failed to begin transaction")

	defer func() {
		err := tx.Rollback(ctx)
		require.NoError(t, err, "failed to rollback transaction")
	}()

	err = fn(tx)
	require.NoError(t, err, "transaction function failed")
}

// TruncateTable truncates a table for test cleanup.
func TruncateTable(t *testing.T, ctx context.Context, conn *pgx.Conn, table string) {
	t.Helper()

	_, err := conn.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
	require.NoError(t, err, "failed to truncate table %s", table)
}

// NewTestHTTPClient creates a new HTTP client for testing.
func NewTestHTTPClient() *TestHTTPClient {
	return &TestHTTPClient{
		Client:  &http.Client{Timeout: 30 * time.Second},
		Headers: make(map[string]string),
	}
}

// NewTestHTTPServer creates a test HTTP server with the given handler.
func NewTestHTTPServer(handler http.Handler) (*httptest.Server, *TestHTTPClient) {
	server := httptest.NewServer(handler)

	client := &TestHTTPClient{
		Client:  server.Client(),
		BaseURL: server.URL,
		Headers: make(map[string]string),
	}

	return server, client
}

// Do executes an HTTP request with the client's configuration.
func (c *TestHTTPClient) Do(req *http.Request) (*http.Response, error) {
	// Apply default headers
	for k, v := range c.Headers {
		req.Header.Set(k, v)
	}

	// Apply auth token if set
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	// Apply interceptors
	for _, interceptor := range c.Interceptors {
		var err error
		req, err = interceptor(req)
		if err != nil {
			return nil, err
		}
	}

	return c.Client.Do(req)
}

// Get executes a GET request.
func (c *TestHTTPClient) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Post executes a POST request with a JSON body.
func (c *TestHTTPClient) Post(url string, body []byte) (*http.Response, error) {
	var bodyReader io.ReadCloser
	if body != nil {
		bodyReader = io.NopCloser(strings.NewReader(string(body)))
	}
	req, err := http.NewRequest("POST", url, bodyReader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.Do(req)
}

// SetAuthToken sets the authentication token for requests.
func (c *TestHTTPClient) SetAuthToken(token string) {
	c.AuthToken = token
}

// SetHeader sets a default header for all requests.
func (c *TestHTTPClient) SetHeader(key, value string) {
	c.Headers[key] = value
}

// AddInterceptor adds a request interceptor.
func (c *TestHTTPClient) AddInterceptor(interceptor InterceptorFunc) {
	c.Interceptors = append(c.Interceptors, interceptor)
}

// testReadCloser is a simple io.ReadCloser for testing.
type testReadCloser struct {
	Reader []byte
	pos    int
}

func (r *testReadCloser) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.Reader) {
		return 0, fmt.Errorf("EOF")
	}
	n = copy(p, r.Reader[r.pos:])
	r.pos += n
	return n, nil
}

func (r *testReadCloser) Close() error {
	return nil
}

// StringReader creates an io.ReadCloser from a string.
type StringReader struct {
	*strings.Reader
}

func (s *StringReader) Close() error {
	return nil
}

// WaitForCondition waits for a condition to be true or timeout.
func WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration, checkInterval time.Duration) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("condition not met within %v", timeout)
		case <-ticker.C:
			if condition() {
				return
			}
		}
	}
}

// SkipIfShort skips the test if -short flag is set.
func SkipIfShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
}

// Must panics if err is not nil, otherwise returns v.
// This is useful for test setup where failure should immediately stop the test.
func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// Contains checks if a slice contains a value.
func Contains[T comparable](slice []T, value T) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// ContainsAny checks if a slice contains any of the specified values.
func ContainsAny[T comparable](slice []T, values ...T) bool {
	for _, v := range values {
		if Contains(slice, v) {
			return true
		}
	}
	return false
}

// Unique returns a new slice with duplicate values removed.
func Unique[T comparable](slice []T) []T {
	seen := make(map[T]struct{})
	result := make([]T, 0, len(slice))

	for _, item := range slice {
		if _, exists := seen[item]; !exists {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

// WaitForValue waits until a function returns a non-error value or times out.
func WaitForValue[T any](t *testing.T, fn func() (T, error), timeout time.Duration) T {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	var lastErr error
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("waitForValue timed out after %v: %v", timeout, lastErr)
			var zero T
			return zero
		case <-ticker.C:
			result, err := fn()
			if err == nil {
				return result
			}
			lastErr = err
		}
	}
}

// TemporaryDir creates a temporary directory for testing and returns a cleanup function.
func TemporaryDir(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	return dir
}

// TemporaryFile creates a temporary file with the given content for testing.
func TemporaryFile(t *testing.T, content string) string {
	t.Helper()

	dir := TemporaryDir(t)
	file, err := os.CreateTemp(dir, "test-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer file.Close()

	if _, err := file.WriteString(content); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	return file.Name()
}

// EqualStringSlices checks if two string slices are equal regardless of order.
func EqualStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aMap := make(map[string]struct{})
	for _, s := range a {
		aMap[s] = struct{}{}
	}
	for _, s := range b {
		if _, ok := aMap[s]; !ok {
			return false
		}
	}
	return true
}

// ParseBool parses a string to bool, returning false on error.
func ParseBool(s string) bool {
	b, err := parseBoolStrict(s)
	if err != nil {
		return false
	}
	return b
}

// parseBoolStrict parses a string to bool strictly.
func parseBoolStrict(s string) (bool, error) {
	switch s {
	case "true", "1", "yes", "on":
		return true, nil
	case "false", "0", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid bool value: %s", s)
	}
}

// SafeString returns a safe string representation of a value.
func SafeString(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// JoinNonEmpty joins non-empty strings with a separator.
func JoinNonEmpty(sep string, parts ...string) string {
	var nonEmpty []string
	for _, p := range parts {
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	return strings.Join(nonEmpty, sep)
}
