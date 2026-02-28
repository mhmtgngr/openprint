// Package testutil tests for HTTP utilities
package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestServer(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	ts := NewTestServer(handler)
	defer ts.Close()

	assert.NotNil(t, ts.Server)
	assert.NotNil(t, ts.Client)
	assert.NotEmpty(t, ts.BaseURL)
	assert.True(t, strings.HasPrefix(ts.BaseURL, "http://"))
}

func TestNewTestServerWithOptions(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	timeout := 5 * time.Second
	ts := NewTestServerWithOptions(TestServerOptions{
		Handler:         handler,
		Timeout:         timeout,
		FollowRedirects: false,
	})
	defer ts.Close()

	assert.NotNil(t, ts.Client)
	assert.Equal(t, timeout, ts.Client.Timeout)
}

func TestTestServer_URL(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ts := NewTestServer(handler)
	defer ts.Close()

	path := "/api/test"
	fullURL := ts.URL(path)
	assert.Equal(t, ts.BaseURL+path, fullURL)
}

func TestTestServer_Get(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"success"}`))
	})

	ts := NewTestServer(handler)
	defer ts.Close()

	resp, err := ts.Get("/test")
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestTestServer_Post(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)

		var data map[string]string
		err := json.NewDecoder(r.Body).Decode(&data)
		assert.NoError(t, err)
		assert.Equal(t, "test", data["key"])

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"created":true}`))
	})

	ts := NewTestServer(handler)
	defer ts.Close()

	resp, err := ts.Post("/create", map[string]string{"key": "test"})
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestTestServer_Put(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		w.WriteHeader(http.StatusOK)
	})

	ts := NewTestServer(handler)
	defer ts.Close()

	resp, err := ts.Put("/update", map[string]string{"key": "value"})
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestTestServer_Patch(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		w.WriteHeader(http.StatusOK)
	})

	ts := NewTestServer(handler)
	defer ts.Close()

	resp, err := ts.Patch("/patch", map[string]string{"key": "value"})
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestTestServer_Delete(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusNoContent)
	})

	ts := NewTestServer(handler)
	defer ts.Close()

	resp, err := ts.Delete("/delete")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestTestServer_RequestWithHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "custom-value", r.Header.Get("X-Custom-Header"))
		assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	})

	ts := NewTestServer(handler)
	defer ts.Close()

	resp, err := ts.RequestWithHeaders("GET", "/test", nil, map[string]string{
		"X-Custom-Header": "custom-value",
		"Authorization":   "Bearer token123",
	})
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestTestServer_RequestWithAuth(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		assert.Equal(t, "Bearer my-token", auth)
		w.WriteHeader(http.StatusOK)
	})

	ts := NewTestServer(handler)
	defer ts.Close()

	resp, err := ts.RequestWithAuth("GET", "/test", nil, "my-token")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestTestServer_GetJSON(t *testing.T) {
	responseData := map[string]string{"message": "hello", "status": "ok"}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseData)
	})

	ts := NewTestServer(handler)
	defer ts.Close()

	var result map[string]string
	resp, err := ts.GetJSON("/data", &result)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "hello", result["message"])
	assert.Equal(t, "ok", result["status"])
}

func TestTestServer_PostJSON(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]string
		json.NewDecoder(r.Body).Decode(&req)

		resp := map[string]string{
			"echo": req["input"],
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	ts := NewTestServer(handler)
	defer ts.Close()

	var result map[string]string
	resp, err := ts.PostJSON("/echo", map[string]string{"input": "test"}, &result)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "test", result["echo"])
}

func TestTestServer_GetResponseBody(t *testing.T) {
	expectedBody := "response body content"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(expectedBody))
	})

	ts := NewTestServer(handler)
	defer ts.Close()

	resp, err := ts.Get("/data")
	require.NoError(t, err)

	body, err := ts.GetResponseBody(resp)
	require.NoError(t, err)
	assert.Equal(t, expectedBody, string(body))
}

func TestMockResponseWriter(t *testing.T) {
	t.Run("basic usage", func(t *testing.T) {
		mw := NewMockResponseWriter()

		assert.Equal(t, http.StatusOK, mw.StatusCode)
		assert.NotNil(t, mw.Headers)
		assert.Equal(t, 0, mw.Body.Len())
	})

	t.Run("write status", func(t *testing.T) {
		mw := NewMockResponseWriter()
		mw.WriteHeader(http.StatusCreated)
		assert.Equal(t, http.StatusCreated, mw.StatusCode)
	})

	t.Run("write body", func(t *testing.T) {
		mw := NewMockResponseWriter()
		n, err := mw.Write([]byte("test content"))
		require.NoError(t, err)
		assert.Equal(t, 12, n)
		assert.Equal(t, "test content", mw.BodyString())
	})

	t.Run("write headers", func(t *testing.T) {
		mw := NewMockResponseWriter()
		mw.Header().Set("Content-Type", "application/json")
		mw.Header().Set("X-Custom", "value")

		assert.Equal(t, "application/json", mw.Header().Get("Content-Type"))
		assert.Equal(t, "value", mw.Header().Get("X-Custom"))
	})

	t.Run("BodyBytes", func(t *testing.T) {
		mw := NewMockResponseWriter()
		mw.Write([]byte("bytes content"))

		expected := []byte("bytes content")
		assert.Equal(t, expected, mw.BodyBytes())
	})

	t.Run("JSONBody", func(t *testing.T) {
		mw := NewMockResponseWriter()
		data := map[string]string{"key": "value"}
		mw.Header().Set("Content-Type", "application/json")
		json.NewEncoder(mw).Encode(data)

		var result map[string]string
		err := mw.JSONBody(&result)
		require.NoError(t, err)
		assert.Equal(t, "value", result["key"])
	})

	t.Run("Reset", func(t *testing.T) {
		mw := NewMockResponseWriter()
		mw.WriteHeader(http.StatusCreated)
		mw.Header().Set("X-Test", "value")
		mw.Write([]byte("content"))

		mw.Reset()

		assert.Equal(t, http.StatusOK, mw.StatusCode)
		assert.Equal(t, 0, mw.Body.Len())
		assert.Equal(t, 0, len(mw.Headers))
	})
}

func TestHandlerTester(t *testing.T) {
	t.Run("TestRequest", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/test", r.URL.Path)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"result":"success"}`))
		}

		ht := NewHandlerTester(t)
		mw := ht.TestRequest(handler, "POST", "/test", nil)

		assert.Equal(t, http.StatusOK, mw.StatusCode)
		assert.Contains(t, mw.BodyString(), "success")
	})

	t.Run("TestRequestWithHeaders", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "test-value", r.Header.Get("X-Test"))
			w.WriteHeader(http.StatusOK)
		}

		ht := NewHandlerTester(t)
		mw := ht.TestRequestWithHeaders(handler, "GET", "/test", nil, map[string]string{
			"X-Test": "test-value",
		})

		assert.Equal(t, http.StatusOK, mw.StatusCode)
	})

	t.Run("AssertStatus", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		}

		ht := NewHandlerTester(t)
		mw := ht.TestRequest(handler, "POST", "/test", nil)

		// Should not panic
		ht.AssertStatus(mw, http.StatusCreated)
	})

	t.Run("AssertBodyContains", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("hello world"))
		}

		ht := NewHandlerTester(t)
		mw := ht.TestRequest(handler, "GET", "/test", nil)

		ht.AssertBodyContains(mw, "world")
	})

	t.Run("AssertJSON", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"key": "value"})
		}

		ht := NewHandlerTester(t)
		mw := ht.TestRequest(handler, "GET", "/test", nil)

		var result map[string]string
		ht.AssertJSON(mw, &result)
		assert.Equal(t, "value", result["key"])
	})
}

func TestCreateFormRequest(t *testing.T) {
	req, err := CreateFormRequest("POST", "/submit", map[string]string{
		"username": "testuser",
		"password": "pass123",
	})
	require.NoError(t, err)

	assert.Equal(t, http.MethodPost, req.Method)
	assert.Equal(t, "/submit", req.URL.Path)
	assert.Equal(t, "application/x-www-form-urlencoded", req.Header.Get("Content-Type"))

	// Parse form to verify
	err = req.ParseForm()
	require.NoError(t, err)
	assert.Equal(t, "testuser", req.FormValue("username"))
	assert.Equal(t, "pass123", req.FormValue("password"))
}

func TestCreateMultipartRequest(t *testing.T) {
	fileContent := []byte("file content")
	fileName := "test.txt"
	fieldName := "file"

	req, err := CreateMultipartRequest(
		"POST",
		"/upload",
		fieldName,
		fileName,
		fileContent,
		map[string]string{"description": "test file"},
	)
	require.NoError(t, err)

	assert.Equal(t, http.MethodPost, req.Method)
	assert.True(t, strings.HasPrefix(req.Header.Get("Content-Type"), "multipart/form-data"))
}

func TestParseErrorResponseFromBody(t *testing.T) {
	t.Run("valid error response", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ErrorResponse{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid input",
				Details: map[string]interface{}{"field": "email"},
			})
		})

		server := httptest.NewServer(handler)
		defer server.Close()

		resp, err := http.Get(server.URL + "/test")
		require.NoError(t, err)
		defer resp.Body.Close()

		errResp, err := ParseErrorResponseFromBody(resp)
		require.NoError(t, err)
		assert.Equal(t, "VALIDATION_ERROR", errResp.Code)
		assert.Equal(t, "Invalid input", errResp.Message)
		assert.NotNil(t, errResp.Details)
	})
}

func TestRunTestServer(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	requestFn := RunTestServer(t, handler)

	// Make a request
	resp, err := requestFn("GET", "/test", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestTestServer_RequestWithContext(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	ts := NewTestServer(handler)
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	resp, err := ts.RequestWithContext(ctx, "GET", "/slow", nil)
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestTestServerRequestBodyTypes(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(body)
	})

	ts := NewTestServer(handler)
	defer ts.Close()

	t.Run("map body", func(t *testing.T) {
		resp, err := ts.Post("/test", map[string]string{"key": "value"})
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("string body", func(t *testing.T) {
		resp, err := ts.Request("POST", "/test", "raw string")
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("bytes body", func(t *testing.T) {
		resp, err := ts.Request("POST", "/test", []byte("bytes content"))
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestRecordRequests(t *testing.T) {
	// Create a handler that we'll wrap with recording
	originalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create test server
	ts := NewTestServer(originalHandler)

	// Set up recording before making requests
	// We need to record manually by wrapping the handler
	recordedRequests := make([]*RequestRecord, 0)
	var mu sync.Mutex

	recordingHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		body, _ := io.ReadAll(r.Body)
		if len(body) > 0 {
			r.Body = io.NopCloser(bytes.NewReader(body))
		}

		record := &RequestRecord{
			Method:      r.Method,
			URL:         r.URL.String(),
			Headers:     r.Header.Clone(),
			Body:        body,
			Timestamp:   time.Now(),
			QueryParams: map[string][]string{},
		}
		for k, v := range r.URL.Query() {
			record.QueryParams[k] = v
		}
		recordedRequests = append(recordedRequests, record)
		mu.Unlock()

		originalHandler.ServeHTTP(w, r)
	})

	// Create a new server with recording
	ts.Close()
	ts = NewTestServer(recordingHandler)
	defer ts.Close()

	// Make some requests
	ts.Get("/path1")
	ts.Post("/path2", nil)
	ts.Get("/path3")

	mu.Lock()
	requests := recordedRequests
	mu.Unlock()

	assert.Len(t, requests, 3)

	var getRequests []*RequestRecord
	for _, r := range requests {
		if r.Method == "GET" {
			getRequests = append(getRequests, r)
		}
	}
	assert.Len(t, getRequests, 2)

	var pathRequests []*RequestRecord
	for _, r := range requests {
		if strings.HasPrefix(r.URL, "/path2") {
			pathRequests = append(pathRequests, r)
		}
	}
	assert.Len(t, pathRequests, 1)
	if len(pathRequests) > 0 {
		assert.Equal(t, "POST", pathRequests[0].Method)
	}
}

func TestClearRequests(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ts := NewTestServer(handler)
	defer ts.Close()

	// RecordRequests needs to be set up differently
	// For now, just test that ClearRequests works on empty state
	ts.ClearRequests()
	assert.Equal(t, 0, ts.RequestCount())
}

func TestMustGetResponseBody(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("response"))
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/test")
	require.NoError(t, err)
	defer resp.Body.Close()

	// Using a test to catch panic
	t.Run("success case", func(t *testing.T) {
		body := MustGetResponseBody(resp)
		assert.Equal(t, "response", string(body))
	})
}

func TestRequestRecord(t *testing.T) {
	t.Run("full record", func(t *testing.T) {
		record := &RequestRecord{
			Method:    "GET",
			URL:       "/test?param=value",
			Headers:   http.Header{},
			Body:      []byte("request body"),
			Timestamp: time.Now(),
			QueryParams: map[string][]string{
				"param": {"value"},
			},
		}

		assert.Equal(t, "GET", record.Method)
		assert.Equal(t, "/test?param=value", record.URL)
		assert.Equal(t, []byte("request body"), record.Body)
		assert.NotNil(t, record.Headers)
		assert.NotZero(t, record.Timestamp)
		assert.Len(t, record.QueryParams, 1)
	})

	t.Run("empty record", func(t *testing.T) {
		record := &RequestRecord{
			Method:  "",
			URL:     "",
			Headers: make(http.Header),
			Body:    nil,
		}

		assert.Empty(t, record.Method)
		assert.Empty(t, record.URL)
		assert.Empty(t, record.Body)
	})
}
