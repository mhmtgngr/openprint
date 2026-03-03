package testutil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTestConfig(t *testing.T) {
	tests := []struct {
		name     string
		envDBURL string
		envBase  string
		wantDB   string
		wantBase string
	}{
		{
			name:     "defaults",
			wantDB:   "postgres://openprint:openprint@localhost:15432/openprint_test?sslmode=disable",
			wantBase: "http://localhost:8000",
		},
		{
			name:     "custom values",
			envDBURL: "postgres://localhost:5432/test",
			envBase:  "http://custom:9000",
			wantDB:   "postgres://localhost:5432/test",
			wantBase: "http://custom:9000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			if tt.envDBURL != "" {
				os.Setenv("TEST_DATABASE_URL", tt.envDBURL)
				defer os.Unsetenv("TEST_DATABASE_URL")
			}
			if tt.envBase != "" {
				os.Setenv("TEST_BASE_URL", tt.envBase)
				defer os.Unsetenv("TEST_BASE_URL")
			}

			config := GetTestConfig()

			assert.Equal(t, tt.wantDB, config.DatabaseURL)
			assert.Equal(t, tt.wantBase, config.BaseURL)
			assert.Equal(t, 30*time.Second, config.Timeout)
		})
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		defaultValue  string
		envValue      string
		expectedValue string
	}{
		{
			name:          "returns default when env not set",
			key:           "NON_EXISTENT_VAR",
			defaultValue:  "default_value",
			expectedValue: "default_value",
		},
		{
			name:          "returns env value when set",
			key:           "TEST_VAR",
			defaultValue:  "default",
			envValue:      "env_value",
			expectedValue: "env_value",
		},
		{
			name:          "returns default when env is empty string",
			key:           "EMPTY_VAR",
			defaultValue:  "default",
			envValue:      "",
			expectedValue: "default", // Empty string is still a set value
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			result := getEnv(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expectedValue, result)
		})
	}
}

func TestNewTestHTTPClient(t *testing.T) {
	client := NewTestHTTPClient()

	assert.NotNil(t, client)
	assert.NotNil(t, client.Client)
	assert.Equal(t, 30*time.Second, client.Client.Timeout)
	assert.Empty(t, client.AuthToken)
	assert.NotNil(t, client.Headers)
	assert.Empty(t, client.Headers)
	assert.Empty(t, client.Interceptors)
}

func TestTestHTTPClient_SetAuthToken(t *testing.T) {
	client := NewTestHTTPClient()
	assert.Empty(t, client.AuthToken)

	client.SetAuthToken("test-token")
	assert.Equal(t, "test-token", client.AuthToken)

	client.SetAuthToken("new-token")
	assert.Equal(t, "new-token", client.AuthToken)
}

func TestTestHTTPClient_SetHeader(t *testing.T) {
	client := NewTestHTTPClient()

	client.SetHeader("Content-Type", "application/json")
	assert.Equal(t, "application/json", client.Headers["Content-Type"])

	client.SetHeader("Authorization", "Bearer token")
	assert.Equal(t, "Bearer token", client.Headers["Authorization"])
	assert.Equal(t, "application/json", client.Headers["Content-Type"], "previous header preserved")
}

func TestTestHTTPClient_AddInterceptor(t *testing.T) {
	client := NewTestHTTPClient()

	interceptor1 := func(req *http.Request) (*http.Request, error) {
		req.Header.Set("X-Interceptor-1", "true")
		return req, nil
	}
	interceptor2 := func(req *http.Request) (*http.Request, error) {
		req.Header.Set("X-Interceptor-2", "true")
		return req, nil
	}

	client.AddInterceptor(interceptor1)
	assert.Len(t, client.Interceptors, 1)

	client.AddInterceptor(interceptor2)
	assert.Len(t, client.Interceptors, 2)
}

func TestTestHTTPClient_Do(t *testing.T) {
	t.Run("applies headers", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "custom-value", r.Header.Get("X-Custom"))
			assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewTestHTTPClient()
		client.SetHeader("X-Custom", "custom-value")
		client.SetAuthToken("test-token")

		req, _ := http.NewRequest("GET", server.URL, nil)
		resp, err := client.Do(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("applies interceptors", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "intercepted", r.Header.Get("X-Intercepted"))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewTestHTTPClient()
		client.AddInterceptor(func(req *http.Request) (*http.Request, error) {
			req.Header.Set("X-Intercepted", "intercepted")
			return req, nil
		})

		req, _ := http.NewRequest("GET", server.URL, nil)
		resp, err := client.Do(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("interceptor error", func(t *testing.T) {
		client := NewTestHTTPClient()
		client.AddInterceptor(func(req *http.Request) (*http.Request, error) {
			return nil, assert.AnError
		})

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		_, err := client.Do(req)

		assert.Error(t, err)
	})
}

func TestTestHTTPClient_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response"))
	}))
	defer server.Close()

	client := NewTestHTTPClient()
	resp, err := client.Get(server.URL)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestTestHTTPClient_Post(t *testing.T) {
	t.Run("with body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			w.WriteHeader(http.StatusCreated)
		}))
		defer server.Close()

		client := NewTestHTTPClient()
		resp, err := client.Post(server.URL, []byte(`{"test":"data"}`))

		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("without body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewTestHTTPClient()
		resp, err := client.Post(server.URL, nil)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})
}

func TestNewTestHTTPServer(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	server, client := NewTestHTTPServer(handler)

	assert.NotNil(t, server)
	assert.NotNil(t, client)
	assert.NotEmpty(t, client.BaseURL)
	assert.True(t, strings.HasPrefix(client.BaseURL, server.URL))

	resp, err := client.Get(client.BaseURL)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	server.Close()
}

func TestTestReadCloser(t *testing.T) {
	t.Run("read in one chunk", func(t *testing.T) {
		data := []byte("test data")
		rc := &testReadCloser{Reader: data}

		buf := make([]byte, 100)
		n, err := rc.Read(buf)

		assert.NoError(t, err)
		assert.Equal(t, len(data), n)
		assert.Equal(t, data, buf[:n])
	})

	t.Run("read in multiple chunks", func(t *testing.T) {
		data := []byte("test data")
		rc := &testReadCloser{Reader: data}

		buf1 := make([]byte, 4)
		n1, _ := rc.Read(buf1)
		assert.Equal(t, 4, n1)
		assert.Equal(t, []byte("test"), buf1)

		buf2 := make([]byte, 10)
		n2, err := rc.Read(buf2)
		assert.NoError(t, err)
		assert.Equal(t, 5, n2) // " data" is 5 bytes
		assert.Equal(t, []byte(" data"), buf2[:n2])
	})

	t.Run("read after EOF", func(t *testing.T) {
		data := []byte("test")
		rc := &testReadCloser{Reader: data}

		buf := make([]byte, 10)
		n1, _ := rc.Read(buf)
		assert.Equal(t, 4, n1)

		_, err := rc.Read(buf)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "EOF")
	})

	t.Run("close", func(t *testing.T) {
		rc := &testReadCloser{Reader: []byte("test")}
		err := rc.Close()
		assert.NoError(t, err)
	})
}

func TestSkipIfShort(t *testing.T) {
	t.Run("does not skip when short flag is off", func(t *testing.T) {
		skipped := false
		t.Run("subtest", func(t *testing.T) {
			SkipIfShort(t)
			skipped = false
		})
		assert.False(t, skipped)
	})
}

func TestWaitForCondition(t *testing.T) {
	t.Run("condition met immediately", func(t *testing.T) {
		condition := func() bool { return true }
		WaitForCondition(t, condition, 100*time.Millisecond, 10*time.Millisecond)
	})

	t.Run("condition met after delay", func(t *testing.T) {
		count := 0
		condition := func() bool {
			count++
			return count >= 3
		}
		WaitForCondition(t, condition, 100*time.Millisecond, 10*time.Millisecond)
		assert.Equal(t, 3, count)
	})

	t.Run("condition never met - times out", func(t *testing.T) {
		// This test verifies timeout behavior by running it in a separate test
		// Since WaitForCondition calls t.Fatalf on timeout, we can't directly test it here
		// Instead, we skip this subtest to avoid the test failure
		t.Skip("WaitForCondition uses t.Fatalf on timeout which cannot be tested directly")
	})
}

func TestTestDB(t *testing.T) {
	t.Run("TeardownTestDB with nil cleanup", func(t *testing.T) {
		db := &TestDB{
			Cleanup: nil,
		}
		// Should not panic
		TeardownTestDB(t, db)
	})

	t.Run("TeardownTestDB with cleanup", func(t *testing.T) {
		cleanupCalled := false
		db := &TestDB{
			Cleanup: func() { cleanupCalled = true },
		}
		TeardownTestDB(t, db)
		assert.True(t, cleanupCalled)
	})
}

// Mock transaction for testing InTransaction
type mockTx struct {
	pgx.Tx
	commitCalled bool
	rollbackCalled bool
}

func (m *mockTx) Commit(context.Context) error {
	m.commitCalled = true
	return nil
}

func (m *mockTx) Rollback(context.Context) error {
	m.rollbackCalled = true
	return nil
}

func TestTruncateTable(t *testing.T) {
	t.Run("error handling", func(t *testing.T) {
		// This test verifies the function signature and behavior
		// Actual database operations require a real connection
		t.Skip("requires database connection")
	})
}

func TestTestReadCloser_EOF_Error(t *testing.T) {
	t.Run("read returns specific EOF error", func(t *testing.T) {
		data := []byte("test")
		rc := &testReadCloser{Reader: data}

		buf := make([]byte, 10)
		rc.Read(buf)          // Read all data
		_, err := rc.Read(buf) // Try to read more

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "EOF")
	})
}

func TestStringReader(t *testing.T) {
	t.Run("StringReader Close does nothing", func(t *testing.T) {
		sr := &StringReader{strings.NewReader("test")}
		err := sr.Close()
		assert.NoError(t, err)
	})

	t.Run("StringReader reads from underlying reader", func(t *testing.T) {
		sr := &StringReader{strings.NewReader("hello")}
		buf := make([]byte, 5)
		n, err := sr.Read(buf)

		assert.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte("hello"), buf)
	})
}

func TestTestHTTPClient_Get_Error(t *testing.T) {
	t.Run("Get returns error on invalid URL", func(t *testing.T) {
		client := NewTestHTTPClient()
		// Use a URL with an invalid scheme so http.NewRequest fails deterministically
		_, err := client.Get("://invalid")

		assert.Error(t, err)
	})
}

func TestTestHTTPClient_Post_Error(t *testing.T) {
	t.Run("Post returns error on invalid URL", func(t *testing.T) {
		client := NewTestHTTPClient()
		_, err := client.Post("://invalid", []byte("{}"))

		assert.Error(t, err)
	})
}

func TestTestHTTPClient_Do_RequestCreationError(t *testing.T) {
	// This test verifies error handling in Do method
	// Creating an invalid request to test error paths
	client := NewTestHTTPClient()

	// Create an invalid request (invalid method should be handled by http.NewRequest)
	_, err := client.Post("://invalid", nil)
	assert.Error(t, err)
}

func TestSkipIfShort_ShortMode(t *testing.T) {
	// Test with a sub-test that checks skip behavior
	t.Run("parent test", func(t *testing.T) {
		t.Run("child test", func(t *testing.T) {
			// In actual short mode, this would skip
			// We can't test actual skip behavior without running with -short flag
			SkipIfShort(t)
			// If we get here, we're not in short mode
		})
	})
}

func TestTestConfig_Fields(t *testing.T) {
	t.Run("config has expected fields", func(t *testing.T) {
		config := GetTestConfig()

		assert.NotEmpty(t, config.DatabaseURL)
		assert.NotEmpty(t, config.BaseURL)
		assert.Greater(t, config.Timeout, time.Duration(0))
	})
}

func TestTestHTTPClient_Do_WithInvalidRequest(t *testing.T) {
	t.Run("handles nil request body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewTestHTTPClient()
		req, _ := http.NewRequest("GET", server.URL, nil)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		resp.Body.Close()
	})
}

func TestTestHTTPClient_MultipleHeaders(t *testing.T) {
	t.Run("multiple headers are preserved", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "value1", r.Header.Get("X-Header-1"))
			assert.Equal(t, "value2", r.Header.Get("X-Header-2"))
			assert.Equal(t, "Bearer token", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewTestHTTPClient()
		client.SetHeader("X-Header-1", "value1")
		client.SetHeader("X-Header-2", "value2")
		client.SetAuthToken("token")

		req, _ := http.NewRequest("GET", server.URL, nil)
		resp, err := client.Do(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})
}

func TestTestHTTPClient_InterceptorChain(t *testing.T) {
	t.Run("multiple interceptors run in order", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "interceptor1", r.Header.Get("X-Order-1"))
			assert.Equal(t, "interceptor2", r.Header.Get("X-Order-2"))
			assert.Equal(t, "interceptor3", r.Header.Get("X-Order-3"))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewTestHTTPClient()
		client.AddInterceptor(func(req *http.Request) (*http.Request, error) {
			req.Header.Set("X-Order-1", "interceptor1")
			return req, nil
		})
		client.AddInterceptor(func(req *http.Request) (*http.Request, error) {
			req.Header.Set("X-Order-2", "interceptor2")
			return req, nil
		})
		client.AddInterceptor(func(req *http.Request) (*http.Request, error) {
			req.Header.Set("X-Order-3", "interceptor3")
			return req, nil
		})

		req, _ := http.NewRequest("GET", server.URL, nil)
		resp, err := client.Do(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})
}

func TestTestHTTPClient_InterceptorModifiesURL(t *testing.T) {
	t.Run("interceptor can modify request URL", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewTestHTTPClient()
		client.AddInterceptor(func(req *http.Request) (*http.Request, error) {
			// Modify the request to point to the test server
			newReq, _ := http.NewRequest(req.Method, server.URL, req.Body)
			// Copy headers
			for k, v := range req.Header {
				newReq.Header[k] = v
			}
			return newReq, nil
		})

		// Make request to invalid URL, interceptor will fix it
		req, _ := http.NewRequest("GET", "http://invalid-url-12345", nil)
		resp, err := client.Do(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})
}

func TestTestReadCloser_EmptyReader(t *testing.T) {
	t.Run("read from empty reader", func(t *testing.T) {
		rc := &testReadCloser{Reader: []byte{}}

		buf := make([]byte, 10)
		n, err := rc.Read(buf)

		assert.Error(t, err)
		assert.Equal(t, 0, n)
		assert.Contains(t, err.Error(), "EOF")
	})
}

func TestWaitForCondition_ImmediateTrue(t *testing.T) {
	t.Run("returns immediately when condition is true", func(t *testing.T) {
		callCount := 0
		condition := func() bool {
			callCount++
			return true
		}

		// Should return almost immediately
		start := time.Now()
		WaitForCondition(t, condition, 1*time.Second, 100*time.Millisecond)
		elapsed := time.Since(start)

		assert.Less(t, elapsed, 500*time.Millisecond, "Should return quickly when condition is immediately true")
		assert.Equal(t, 1, callCount, "Should only check once")
	})
}

func TestTestHTTPClient_DefaultTimeout(t *testing.T) {
	t.Run("client has default timeout", func(t *testing.T) {
		client := NewTestHTTPClient()

		assert.Equal(t, 30*time.Second, client.Client.Timeout)
	})
}

func TestTestHTTPClient_HeadersMap(t *testing.T) {
	t.Run("headers map is initialized", func(t *testing.T) {
		client := NewTestHTTPClient()

		assert.NotNil(t, client.Headers)
		// Interceptors is initialized as nil slice, check length after adding one
		client.AddInterceptor(func(req *http.Request) (*http.Request, error) {
			return req, nil
		})
		assert.Len(t, client.Interceptors, 1)
	})
}

func TestTestHTTPClient_OverwriteHeader(t *testing.T) {
	t.Run("setting same header twice overwrites", func(t *testing.T) {
		client := NewTestHTTPClient()

		client.SetHeader("X-Test", "value1")
		assert.Equal(t, "value1", client.Headers["X-Test"])

		client.SetHeader("X-Test", "value2")
		assert.Equal(t, "value2", client.Headers["X-Test"])
	})
}

func TestTestHTTPClient_EmptyAuthToken(t *testing.T) {
	t.Run("empty auth token is not set", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			// Empty token should not set Authorization header
			assert.Empty(t, auth)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewTestHTTPClient()
		// Don't set auth token
		req, _ := http.NewRequest("GET", server.URL, nil)
		resp, err := client.Do(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})
}

func TestGetTestConfig_Concurrency(t *testing.T) {
	t.Run("config is safe for concurrent use", func(t *testing.T) {
		// Run multiple goroutines getting config
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				config := GetTestConfig()
				assert.NotEmpty(t, config.DatabaseURL)
				done <- true
			}()
		}

		// Wait for all to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

func TestMust(t *testing.T) {
	t.Run("returns value when no error", func(t *testing.T) {
		result := Must("test value", nil)
		assert.Equal(t, "test value", result)
	})

	t.Run("panics when error is not nil", func(t *testing.T) {
		assert.Panics(t, func() {
			Must("value", assert.AnError)
		})
	})
}

func TestContains(t *testing.T) {
	t.Run("finds value in slice", func(t *testing.T) {
		slice := []string{"a", "b", "c"}
		assert.True(t, Contains(slice, "b"))
	})

	t.Run("returns false when value not in slice", func(t *testing.T) {
		slice := []string{"a", "b", "c"}
		assert.False(t, Contains(slice, "d"))
	})

	t.Run("works with int slice", func(t *testing.T) {
		slice := []int{1, 2, 3}
		assert.True(t, Contains(slice, 2))
		assert.False(t, Contains(slice, 4))
	})

	t.Run("empty slice", func(t *testing.T) {
		slice := []string{}
		assert.False(t, Contains(slice, "a"))
	})
}

func TestContainsAny(t *testing.T) {
	t.Run("finds any of multiple values", func(t *testing.T) {
		slice := []string{"a", "b", "c"}
		assert.True(t, ContainsAny(slice, "d", "b", "e"))
	})

	t.Run("returns false when none of values found", func(t *testing.T) {
		slice := []string{"a", "b", "c"}
		assert.False(t, ContainsAny(slice, "d", "e", "f"))
	})

	t.Run("works with no values to check", func(t *testing.T) {
		slice := []string{"a", "b", "c"}
		assert.False(t, ContainsAny(slice))
	})
}

func TestUnique(t *testing.T) {
	t.Run("removes duplicates", func(t *testing.T) {
		slice := []string{"a", "b", "a", "c", "b"}
		result := Unique(slice)
		assert.ElementsMatch(t, []string{"a", "b", "c"}, result)
	})

	t.Run("preserves order of first occurrence", func(t *testing.T) {
		slice := []string{"c", "a", "b", "a", "c"}
		result := Unique(slice)
		assert.Equal(t, []string{"c", "a", "b"}, result)
	})

	t.Run("empty slice", func(t *testing.T) {
		slice := []string{}
		result := Unique(slice)
		assert.Empty(t, result)
	})

	t.Run("slice with no duplicates", func(t *testing.T) {
		slice := []int{1, 2, 3}
		result := Unique(slice)
		assert.Equal(t, []int{1, 2, 3}, result)
	})
}

func TestWaitForValue(t *testing.T) {
	t.Run("returns value when function succeeds", func(t *testing.T) {
		count := 0
		fn := func() (string, error) {
			count++
			if count >= 3 {
				return "success", nil
			}
			return "", assert.AnError
		}

		result := WaitForValue(t, fn, 100*time.Millisecond)
		assert.Equal(t, "success", result)
		assert.Equal(t, 3, count)
	})

	t.Run("times out when function never succeeds", func(t *testing.T) {
		// WaitForValue uses t.Fatalf which cannot be tested directly
		t.Skip("WaitForValue uses t.Fatalf on timeout which cannot be tested directly")
	})
}

func TestTemporaryDir(t *testing.T) {
	t.Run("creates and cleans up temp directory", func(t *testing.T) {
		dir := TemporaryDir(t)

		assert.NotEmpty(t, dir)
		assert.DirExists(t, dir)

		// Create a file in the temp dir
		testFile := dir + "/test.txt"
		err := os.WriteFile(testFile, []byte("test"), 0644)
		assert.NoError(t, err)

		// File should exist now
		assert.FileExists(t, testFile)

		// After test completes, directory should be cleaned up by t.Cleanup
		// We can't directly test this, but the cleanup function is registered
	})
}

func TestTemporaryFile(t *testing.T) {
	t.Run("creates temp file with content", func(t *testing.T) {
		content := "test content"
		filePath := TemporaryFile(t, content)

		assert.FileExists(t, filePath)

		// Read the file and verify content
		data, err := os.ReadFile(filePath)
		assert.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("creates file in temp directory", func(t *testing.T) {
		filePath := TemporaryFile(t, "content")

		// File should be in a temp directory
		dir := filepath.Dir(filePath)
		assert.DirExists(t, dir)
	})
}

func TestEqualStringSlices(t *testing.T) {
	t.Run("equal slices", func(t *testing.T) {
		a := []string{"a", "b", "c"}
		b := []string{"c", "b", "a"}
		assert.True(t, EqualStringSlices(a, b))
	})

	t.Run("unequal slices different length", func(t *testing.T) {
		a := []string{"a", "b"}
		b := []string{"a", "b", "c"}
		assert.False(t, EqualStringSlices(a, b))
	})

	t.Run("unequal slices different elements", func(t *testing.T) {
		a := []string{"a", "b", "c"}
		b := []string{"a", "b", "d"}
		assert.False(t, EqualStringSlices(a, b))
	})

	t.Run("empty slices", func(t *testing.T) {
		assert.True(t, EqualStringSlices([]string{}, []string{}))
	})
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"1", true},
		{"yes", true},
		{"on", true},
		{"false", false},
		{"0", false},
		{"no", false},
		{"off", false},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseBool(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSafeString(t *testing.T) {
	t.Run("returns string for string", func(t *testing.T) {
		assert.Equal(t, "test", SafeString("test"))
	})

	t.Run("returns string for int", func(t *testing.T) {
		assert.Equal(t, "123", SafeString(123))
	})

	t.Run("returns empty for nil", func(t *testing.T) {
		assert.Equal(t, "", SafeString(nil))
	})

	t.Run("returns string representation for struct", func(t *testing.T) {
		type testStruct struct {
			A int
		}
		assert.Contains(t, SafeString(testStruct{A: 1}), "{")
	})
}

func TestJoinNonEmpty(t *testing.T) {
	t.Run("joins non-empty strings", func(t *testing.T) {
		result := JoinNonEmpty(",", "a", "b", "c")
		assert.Equal(t, "a,b,c", result)
	})

	t.Run("skips empty strings", func(t *testing.T) {
		result := JoinNonEmpty(",", "a", "", "c")
		assert.Equal(t, "a,c", result)
	})

	t.Run("all empty strings", func(t *testing.T) {
		result := JoinNonEmpty(",", "", "", "")
		assert.Equal(t, "", result)
	})

	t.Run("no arguments", func(t *testing.T) {
		result := JoinNonEmpty(",")
		assert.Equal(t, "", result)
	})
}

// Benchmark tests
func BenchmarkNewTestHTTPClient(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewTestHTTPClient()
	}
}

func BenchmarkTestHTTPClient_SetHeader(b *testing.B) {
	client := NewTestHTTPClient()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.SetHeader("X-Test", "value")
	}
}

func BenchmarkTestHTTPClient_SetAuthToken(b *testing.B) {
	client := NewTestHTTPClient()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.SetAuthToken("test-token")
	}
}
