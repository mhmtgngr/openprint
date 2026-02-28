// Package testutil provides HTTP test server helpers for testing.
package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestServer wraps httptest.Server with additional utilities.
type TestServer struct {
	Server     *httptest.Server
	Client     *http.Client
	BaseURL    string
	mu         sync.Mutex
	responses  map[string]*MockResponse
	requests   []*RequestRecord
	handler    http.Handler
	middleware []func(http.Handler) http.Handler
}

// MockResponse defines a mock HTTP response.
type MockResponse struct {
	StatusCode int
	Body       []byte
	Headers    map[string]string
	Delay      time.Duration
	Error      error
}

// RequestRecord records details of an incoming request.
type RequestRecord struct {
	Method      string
	URL         string
	Headers     http.Header
	Body        []byte
	Timestamp   time.Time
	QueryParams map[string][]string
}

// TestServerOptions holds configuration for creating a test server.
type TestServerOptions struct {
	Handler         http.Handler
	Middleware      []func(http.Handler) http.Handler
	Timeout         time.Duration
	FollowRedirects bool
}

// NewTestServer creates a new test server with the given handler.
func NewTestServer(handler http.Handler) *TestServer {
	return NewTestServerWithOptions(TestServerOptions{
		Handler: handler,
	})
}

// NewTestServerWithOptions creates a new test server with custom options.
func NewTestServerWithOptions(opts TestServerOptions) *TestServer {
	if opts.Handler == nil {
		opts.Handler = http.DefaultServeMux
	}

	// Apply middleware in reverse order (last applied is first to execute)
	h := opts.Handler
	for i := len(opts.Middleware) - 1; i >= 0; i-- {
		h = opts.Middleware[i](h)
	}

	server := httptest.NewServer(h)

	client := &http.Client{
		Timeout: opts.Timeout,
	}
	if !opts.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	return &TestServer{
		Server:    server,
		Client:    client,
		BaseURL:   server.URL,
		responses: make(map[string]*MockResponse),
		requests:  make([]*RequestRecord, 0),
		handler:   h,
	}
}

// NewTestServerWithHandler creates a new test server with a custom handler.
func NewTestServerWithHandler(handler http.HandlerFunc) *TestServer {
	return NewTestServer(handler)
}

// Close closes the test server.
func (ts *TestServer) Close() {
	if ts.Server != nil {
		ts.Server.Close()
	}
}

// URL returns the full URL for a path.
func (ts *TestServer) URL(path string) string {
	return ts.BaseURL + path
}

// Get performs a GET request to the test server.
func (ts *TestServer) Get(path string) (*http.Response, error) {
	return ts.Request("GET", path, nil)
}

// Post performs a POST request to the test server.
func (ts *TestServer) Post(path string, body interface{}) (*http.Response, error) {
	return ts.Request("POST", path, body)
}

// Put performs a PUT request to the test server.
func (ts *TestServer) Put(path string, body interface{}) (*http.Response, error) {
	return ts.Request("PUT", path, body)
}

// Patch performs a PATCH request to the test server.
func (ts *TestServer) Patch(path string, body interface{}) (*http.Response, error) {
	return ts.Request("PATCH", path, body)
}

// Delete performs a DELETE request to the test server.
func (ts *TestServer) Delete(path string) (*http.Response, error) {
	return ts.Request("DELETE", path, nil)
}

// Request performs a generic HTTP request to the test server.
func (ts *TestServer) Request(method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		switch v := body.(type) {
		case []byte:
			bodyReader = bytes.NewReader(v)
		case string:
			bodyReader = strings.NewReader(v)
		default:
			jsonData, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("marshal body: %w", err)
			}
			bodyReader = bytes.NewReader(jsonData)
		}
	}

	req, err := http.NewRequest(method, ts.URL(path), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return ts.Client.Do(req)
}

// RequestWithContext performs an HTTP request with context.
func (ts *TestServer) RequestWithContext(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		switch v := body.(type) {
		case []byte:
			bodyReader = bytes.NewReader(v)
		case string:
			bodyReader = strings.NewReader(v)
		default:
			jsonData, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("marshal body: %w", err)
			}
			bodyReader = bytes.NewReader(jsonData)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, ts.URL(path), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return ts.Client.Do(req)
}

// RequestWithHeaders performs an HTTP request with custom headers.
func (ts *TestServer) RequestWithHeaders(method, path string, body interface{}, headers map[string]string) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		switch v := body.(type) {
		case []byte:
			bodyReader = bytes.NewReader(v)
		case string:
			bodyReader = strings.NewReader(v)
		default:
			jsonData, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("marshal body: %w", err)
			}
			bodyReader = bytes.NewReader(jsonData)
		}
	}

	req, err := http.NewRequest(method, ts.URL(path), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return ts.Client.Do(req)
}

// RequestWithAuth performs an HTTP request with Bearer token authentication.
func (ts *TestServer) RequestWithAuth(method, path string, body interface{}, token string) (*http.Response, error) {
	return ts.RequestWithHeaders(method, path, body, map[string]string{
		"Authorization": "Bearer " + token,
	})
}

// GetJSON performs a GET request and unmarshals the JSON response.
func (ts *TestServer) GetJSON(path string, v interface{}) (*http.Response, error) {
	resp, err := ts.Get(path)
	if err != nil {
		return resp, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return resp, fmt.Errorf("decode response: %w", err)
	}

	return resp, nil
}

// PostJSON performs a POST request with JSON body and unmarshals the JSON response.
func (ts *TestServer) PostJSON(path string, reqBody, respBody interface{}) (*http.Response, error) {
	resp, err := ts.Post(path, reqBody)
	if err != nil {
		return resp, err
	}
	defer resp.Body.Close()

	if respBody != nil {
		if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
			return resp, fmt.Errorf("decode response: %w", err)
		}
	}

	return resp, nil
}

// GetResponseBody returns the response body as bytes.
func (ts *TestServer) GetResponseBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// RecordRequests creates a middleware that records all requests.
func (ts *TestServer) RecordRequests() {
	ts.handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts.mu.Lock()

		// Read body
		body, _ := io.ReadAll(r.Body)
		if len(body) > 0 {
			// Restore body for further processing
			r.Body = io.NopCloser(bytes.NewReader(body))
		}

		record := &RequestRecord{
			Method:    r.Method,
			URL:       r.URL.String(),
			Headers:   r.Header.Clone(),
			Body:      body,
			Timestamp: time.Now(),
			QueryParams: map[string][]string{},
		}

		for k, v := range r.URL.Query() {
			record.QueryParams[k] = v
		}

		ts.requests = append(ts.requests, record)
		ts.mu.Unlock()

		ts.handler.ServeHTTP(w, r)
	})
}

// GetRequests returns all recorded requests.
func (ts *TestServer) GetRequests() []*RequestRecord {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	result := make([]*RequestRecord, len(ts.requests))
	copy(result, ts.requests)
	return result
}

// GetRequestsByMethod returns requests filtered by method.
func (ts *TestServer) GetRequestsByMethod(method string) []*RequestRecord {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	var result []*RequestRecord
	for _, req := range ts.requests {
		if req.Method == method {
			result = append(result, req)
		}
	}
	return result
}

// GetRequestsByPath returns requests filtered by path prefix.
func (ts *TestServer) GetRequestsByPath(prefix string) []*RequestRecord {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	var result []*RequestRecord
	for _, req := range ts.requests {
		if strings.HasPrefix(req.URL, prefix) {
			result = append(result, req)
		}
	}
	return result
}

// ClearRequests clears all recorded requests.
func (ts *TestServer) ClearRequests() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.requests = make([]*RequestRecord, 0)
}

// RequestCount returns the number of recorded requests.
func (ts *TestServer) RequestCount() int {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return len(ts.requests)
}

// MockResponseWriter creates a mock response writer for testing handlers.
type MockResponseWriter struct {
	StatusCode int
	Headers    http.Header
	Body       bytes.Buffer
	written    bool
}

// NewMockResponseWriter creates a new mock response writer.
func NewMockResponseWriter() *MockResponseWriter {
	return &MockResponseWriter{
		StatusCode: http.StatusOK,
		Headers:    make(http.Header),
	}
}

// Header implements http.ResponseWriter.
func (mw *MockResponseWriter) Header() http.Header {
	return mw.Headers
}

// Write implements http.ResponseWriter.
func (mw *MockResponseWriter) Write(b []byte) (int, error) {
	if !mw.written {
		mw.written = true
	}
	return mw.Body.Write(b)
}

// WriteHeader implements http.ResponseWriter.
func (mw *MockResponseWriter) WriteHeader(statusCode int) {
	if !mw.written {
		mw.StatusCode = statusCode
		mw.written = true
	}
}

// BodyString returns the response body as a string.
func (mw *MockResponseWriter) BodyString() string {
	return mw.Body.String()
}

// BodyBytes returns the response body as bytes.
func (mw *MockResponseWriter) BodyBytes() []byte {
	return mw.Body.Bytes()
}

// Reset clears the response writer state.
func (mw *MockResponseWriter) Reset() {
	mw.StatusCode = http.StatusOK
	mw.Headers = make(http.Header)
	mw.Body.Reset()
	mw.written = false
}

// JSONBody unmarshals the response body as JSON.
func (mw *MockResponseWriter) JSONBody(v interface{}) error {
	return json.Unmarshal(mw.BodyBytes(), v)
}

// HandlerTester provides utilities for testing HTTP handlers.
type HandlerTester struct {
	t *testing.T
}

// NewHandlerTester creates a new handler tester.
func NewHandlerTester(t *testing.T) *HandlerTester {
	return &HandlerTester{t: t}
}

// TestRequest tests a handler with a request.
func (ht *HandlerTester) TestRequest(handler http.HandlerFunc, method, path string, body interface{}) *MockResponseWriter {
	mw := NewMockResponseWriter()

	var bodyReader io.Reader
	if body != nil {
		switch v := body.(type) {
		case []byte:
			bodyReader = bytes.NewReader(v)
		case string:
			bodyReader = strings.NewReader(v)
		default:
			jsonData, err := json.Marshal(body)
			if err != nil {
				ht.t.Fatalf("failed to marshal body: %v", err)
			}
			bodyReader = bytes.NewReader(jsonData)
		}
	}

	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")

	handler(mw, req)

	return mw
}

// TestRequestWithHeaders tests a handler with custom headers.
func (ht *HandlerTester) TestRequestWithHeaders(handler http.HandlerFunc, method, path string, body interface{}, headers map[string]string) *MockResponseWriter {
	mw := NewMockResponseWriter()

	var bodyReader io.Reader
	if body != nil {
		switch v := body.(type) {
		case []byte:
			bodyReader = bytes.NewReader(v)
		case string:
			bodyReader = strings.NewReader(v)
		default:
			jsonData, err := json.Marshal(body)
			if err != nil {
				ht.t.Fatalf("failed to marshal body: %v", err)
			}
			bodyReader = bytes.NewReader(jsonData)
		}
	}

	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	handler(mw, req)

	return mw
}

// AssertStatus asserts the response status code.
func (ht *HandlerTester) AssertStatus(mw *MockResponseWriter, expectedStatus int) {
	if mw.StatusCode != expectedStatus {
		ht.t.Errorf("expected status %d, got %d", expectedStatus, mw.StatusCode)
	}
}

// AssertBodyContains asserts the response body contains a substring.
func (ht *HandlerTester) AssertBodyContains(mw *MockResponseWriter, substring string) {
	if !strings.Contains(mw.BodyString(), substring) {
		ht.t.Errorf("expected body to contain %q, got %q", substring, mw.BodyString())
	}
}

// AssertJSON asserts the response body is valid JSON and optionally unmarshals it.
func (ht *HandlerTester) AssertJSON(mw *MockResponseWriter, v interface{}) {
	if mw.Header().Get("Content-Type") != "application/json" {
		ht.t.Errorf("expected content-type application/json, got %q", mw.Header().Get("Content-Type"))
	}

	if err := json.Unmarshal(mw.BodyBytes(), v); err != nil {
		ht.t.Errorf("failed to unmarshal JSON: %v\nbody: %s", err, mw.BodyString())
	}
}

// CreateFormRequest creates a form-encoded request.
func CreateFormRequest(method, requestURL string, data map[string]string) (*http.Request, error) {
	form := url.Values{}
	for k, v := range data {
		form.Set(k, v)
	}

	req, err := http.NewRequest(method, requestURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req, nil
}

// CreateMultipartRequest creates a multipart form request with file upload.
func CreateMultipartRequest(method, requestURL string, fieldName, fileName string, fileContent []byte, additionalFields map[string]string) (*http.Request, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file
	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(fileContent); err != nil {
		return nil, fmt.Errorf("write file content: %w", err)
	}

	// Add additional fields
	for k, v := range additionalFields {
		if err := writer.WriteField(k, v); err != nil {
			return nil, fmt.Errorf("write field %s: %w", k, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close writer: %w", err)
	}

	req, err := http.NewRequest(method, requestURL, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, nil
}

// ParseErrorResponse parses an error response from the server.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ParseErrorResponseFromBody parses an error response from a response body.
func ParseErrorResponseFromBody(resp *http.Response) (*ErrorResponse, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		return nil, err
	}

	return &errResp, nil
}

// MustGetResponseBody is a helper that gets the response body or panics.
// Useful in test setup where you want to fail fast.
func MustGetResponseBody(resp *http.Response) []byte {
	if resp == nil {
		panic("response is nil")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Sprintf("failed to read response body: %v", err))
	}
	return body
}

// RunTestServer runs a test server with a handler and returns a function to perform requests.
// This is a convenience function for quick test setup.
func RunTestServer(t *testing.T, handler http.Handler) func(method, path string, body interface{}) (*http.Response, error) {
	ts := NewTestServer(handler)
	t.Cleanup(ts.Close)

	return func(method, path string, body interface{}) (*http.Response, error) {
		return ts.Request(method, path, body)
	}
}
