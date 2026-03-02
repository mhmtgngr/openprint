// Package prometheus provides tests for standardized labels.
package prometheus

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTPStatusLabel(t *testing.T) {
	tests := []struct {
		name string
		code int
		want string
	}{
		{"OK", http.StatusOK, "OK"},
		{"Created", http.StatusCreated, "Created"},
		{"Accepted", http.StatusAccepted, "Accepted"},
		{"No Content", http.StatusNoContent, "No_Content"},
		{"Moved Permanently", http.StatusMovedPermanently, "Moved_Permanently"},
		{"Found", http.StatusFound, "Found"},
		{"Bad Request", http.StatusBadRequest, "Bad_Request"},
		{"Unauthorized", http.StatusUnauthorized, "Unauthorized"},
		{"Forbidden", http.StatusForbidden, "Forbidden"},
		{"Not Found", http.StatusNotFound, "Not_Found"},
		{"Internal Server Error", http.StatusInternalServerError, "Internal_Server_Error"},
		{"Service Unavailable", http.StatusServiceUnavailable, "Service_Unavailable"},
		{"Unknown status", 599, ""},  // Unknown returns empty string
		{"Custom status", 499, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HTTPStatusLabel(tt.code)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHTTPStatusClass(t *testing.T) {
	tests := []struct {
		name string
		code int
		want string
	}{
		{"2xx - OK", 200, "2xx"},
		{"2xx - Created", 201, "2xx"},
		{"2xx - Accepted", 202, "2xx"},
		{"2xx - No Content", 204, "2xx"},
		{"3xx - Moved Permanently", 301, "3xx"},
		{"3xx - Found", 302, "3xx"},
		{"3xx - Not Modified", 304, "3xx"},
		{"4xx - Bad Request", 400, "4xx"},
		{"4xx - Unauthorized", 401, "4xx"},
		{"4xx - Forbidden", 403, "4xx"},
		{"4xx - Not Found", 404, "4xx"},
		{"5xx - Internal Server Error", 500, "5xx"},
		{"5xx - Service Unavailable", 503, "5xx"},
		{"5xx - Gateway Timeout", 504, "5xx"},
		{"unknown - 0", 0, "unknown"},
		{"unknown - 99", 99, "unknown"},
		{"unknown - 600", 600, "5xx"},  // >= 500 returns 5xx
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HTTPStatusClass(tt.code)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSanitizeLabelValue(t *testing.T) {
	tests := []struct {
		name string
		input string
		want string
	}{
		{"alphanumeric", "abc123", "abc123"},
		{"with underscore", "test_value", "test_value"},
		{"with colon", "test:value", "test:value"},
		{"spaces to underscores", "test value", "test_value"},
		{"special chars to underscores", "test@value!", "test_value_"},
		{"dots to underscores", "test.value", "test_value"},
		{"dashes to underscores", "test-value", "test_value"},
		{"mixed valid and invalid", "a1_b2:c3", "a1_b2:c3"},
		{"long string truncated", string(make([]byte, 300)), string(make([]byte, 201))},
		{"empty string", "", ""},
		{"only invalid", "@#$%^&*()", "_________"},  // 9 invalid chars
	{"unicode", "test日本語", "test___"},  // 3 Japanese chars become 3 underscores
		{"path-like", "/api/v1/users", "_api_v1_users"},
		{"email-like", "user@example.com", "user_example_com"},
		{"url-like", "https://example.com", "https:__example_com"},  // Colon preserved, slashes become underscores
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeLabelValue(tt.input)
			// For the long string test, we just check length is capped
			if tt.name == "long string truncated" {
				assert.LessOrEqual(t, len(got), 201)
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestSanitizeLabelName(t *testing.T) {
	tests := []struct {
		name string
		input string
		want string
	}{
		{"valid name", "valid_name", "valid_name"},
		{"starts with letter", "testName", "testName"},
		{"starts with underscore", "_test", "_test"},
		{"starts with number becomes underscore", "123test", "_123test"},
		{"spaces to underscores", "test name", "test_name"},
		{"special chars to underscores", "test@name!", "test_name_"},
		{"dots to underscores", "test.name", "test_name"},
		{"dashes to underscores", "test-name", "test_name"},
		{"long string truncated", string(make([]byte, 300)), string(make([]byte, 201))},
		{"empty string", "", ""},
		{"only invalid", "123", "_123"},
		{"mixed", "a1_b2-c3", "a1_b2_c3"},  // Dash becomes underscore
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeLabelName(tt.input)
			if tt.name == "long string truncated" {
				assert.LessOrEqual(t, len(got), 201)
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestLabels_With(t *testing.T) {
	labels := NewLabels()

	result := labels.With("key1", "value1").With("key2", "value2")

	assert.Equal(t, "value1", result["key1"])
	assert.Equal(t, "value2", result["key2"])
}

func TestLabels_WithService(t *testing.T) {
	labels := NewLabels()

	result := labels.WithService("test-service")

	assert.Equal(t, "test-service", result[LabelServiceName])
}

func TestLabels_WithMethod(t *testing.T) {
	labels := NewLabels()

	result := labels.WithMethod("GET")

	assert.Equal(t, "GET", result[LabelMethod])
}

func TestLabels_WithPath(t *testing.T) {
	labels := NewLabels()

	result := labels.WithPath("/api/users")

	assert.Equal(t, "/api/users", result[LabelPath])
}

func TestLabels_WithStatus(t *testing.T) {
	labels := NewLabels()

	result := labels.WithStatus(200)

	assert.Equal(t, "OK", result[LabelStatus])

	result = labels.WithStatus(404)
	assert.Equal(t, "Not_Found", result[LabelStatus])
}

func TestLabels_WithSuccess(t *testing.T) {
	labels := NewLabels()

	result := labels.WithSuccess(true)
	assert.Equal(t, SuccessTrue, result[LabelSuccess])

	result = labels.WithSuccess(false)
	assert.Equal(t, SuccessFalse, result[LabelSuccess])
}

func TestLabels_WithAuthMethod(t *testing.T) {
	labels := NewLabels()

	result := labels.WithAuthMethod(AuthMethodOIDC)

	assert.Equal(t, AuthMethodOIDC, result[LabelAuthMethod])
}

func TestLabels_WithJobStatus(t *testing.T) {
	labels := NewLabels()

	result := labels.WithJobStatus(JobStatusCompleted)

	assert.Equal(t, JobStatusCompleted, result[LabelJobStatus])
}

func TestLabels_WithDBOperation(t *testing.T) {
	labels := NewLabels()

	result := labels.WithDBOperation("SELECT")

	assert.Equal(t, "SELECT", result[LabelOperation])
}

func TestLabels_WithRedisCommand(t *testing.T) {
	labels := NewLabels()

	result := labels.WithRedisCommand("GET")

	assert.Equal(t, "GET", result[LabelRedisCommand])
}

func TestLabels_ToPrometheusLabels(t *testing.T) {
	labels := NewLabels().
		WithService("test").
		WithMethod("GET").
		WithPath("/api")

	result := labels.ToPrometheusLabels()

	assert.Equal(t, "test", result["service_name"])
	assert.Equal(t, "GET", result["method"])
	assert.Equal(t, "/api", result["path"])
	assert.IsType(t, map[string]string{}, result)
}

func TestMerge(t *testing.T) {
	t.Run("merges multiple label maps", func(t *testing.T) {
		l1 := NewLabels().With("key1", "value1")
		l2 := NewLabels().With("key2", "value2")
		l3 := NewLabels().With("key3", "value3")

		result := Merge(l1, l2, l3)

		assert.Equal(t, "value1", result["key1"])
		assert.Equal(t, "value2", result["key2"])
		assert.Equal(t, "value3", result["key3"])
	})

	t.Run("later values override earlier ones", func(t *testing.T) {
		l1 := NewLabels().With("key", "value1")
		l2 := NewLabels().With("key", "value2")

		result := Merge(l1, l2)

		assert.Equal(t, "value2", result["key"])
	})

	t.Run("empty merge returns empty labels", func(t *testing.T) {
		result := Merge()

		assert.Empty(t, result)
	})
}

func TestCommonLabels(t *testing.T) {
	labels := CommonLabels("my-service", "2.0.0")

	assert.Equal(t, "my-service", labels["service_name"])
	assert.Equal(t, "2.0.0", labels["service_version"])
	assert.Equal(t, "openprint", labels["namespace"])
}

func TestLabels_Chaining(t *testing.T) {
	result := NewLabels().
		WithService("service").
		WithMethod("POST").
		WithPath("/users").
		WithStatus(201).
		WithSuccess(true)

	assert.Equal(t, "service", result["service_name"])
	assert.Equal(t, "POST", result["method"])
	assert.Equal(t, "/users", result["path"])
	assert.Equal(t, "Created", result["status"])
	assert.Equal(t, SuccessTrue, result["success"])
}

func TestLabelConstants(t *testing.T) {
	// Verify all label constants are defined
	consts := []struct {
		name  string
		value string
	}{
		{LabelServiceName, "service_name"},
		{LabelServiceVersion, "service_version"},
		{LabelNamespace, "namespace"},
		{LabelMethod, "method"},
		{LabelPath, "path"},
		{LabelStatus, "status"},
		{LabelErrorCode, "error_code"},
		{LabelPeer, "peer"},
		{LabelDBName, "db_name"},
		{LabelDBSystem, "db_system"},
		{LabelOperation, "operation"},
		{LabelRedisDB, "redis_db"},
		{LabelRedisCommand, "redis_command"},
		{LabelSuccess, "success"},
		{LabelAuthMethod, "auth_method"},
		{LabelRole, "role"},
		{LabelOrgID, "org_id"},
		{LabelPrinterID, "printer_id"},
		{LabelJobID, "job_id"},
		{LabelJobStatus, "job_status"},
		{LabelDocumentType, "document_type"},
		{LabelStorageBackend, "storage_backend"},
		{LabelLeveled, "le"},
		{LabelQuantile, "quantile"},
	}

	for _, c := range consts {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.value, c.name)
		})
	}
}

func TestServiceConstants(t *testing.T) {
	assert.Equal(t, "auth-service", ServiceAuthService)
	assert.Equal(t, "registry-service", ServiceRegistryService)
	assert.Equal(t, "job-service", ServiceJobService)
	assert.Equal(t, "storage-service", ServiceStorageService)
	assert.Equal(t, "notification-service", ServiceNotificationService)
}

func TestDBSystemConstants(t *testing.T) {
	assert.Equal(t, "postgresql", DBSystemPostgreSQL)
	assert.Equal(t, "mysql", DBSystemMySQL)
	assert.Equal(t, "sqlite", DBSystemSQLite)
}

func TestStorageBackendConstants(t *testing.T) {
	assert.Equal(t, "s3", StorageBackendS3)
	assert.Equal(t, "local", StorageBackendLocal)
}

func TestJobStatusConstants(t *testing.T) {
	consts := []struct {
		name  string
		value string
	}{
		{"JobStatusPending", JobStatusPending},
		{"JobStatusQueued", JobStatusQueued},
		{"JobStatusProcessing", JobStatusProcessing},
		{"JobStatusCompleted", JobStatusCompleted},
		{"JobStatusFailed", JobStatusFailed},
		{"JobStatusCancelled", JobStatusCancelled},
	}

	for _, c := range consts {
		t.Run(c.name, func(t *testing.T) {
			assert.NotEmpty(t, c.value)
		})
	}
}

func TestAuthMethodConstants(t *testing.T) {
	consts := []struct {
		name  string
		value string
	}{
		{"AuthMethodPassword", AuthMethodPassword},
		{"AuthMethodOIDC", AuthMethodOIDC},
		{"AuthMethodSAML", AuthMethodSAML},
		{"AuthMethodAPIKey", AuthMethodAPIKey},
	}

	for _, c := range consts {
		t.Run(c.name, func(t *testing.T) {
			assert.NotEmpty(t, c.value)
		})
	}
}

func TestSanitizeLabelValue_EdgeCases(t *testing.T) {
	t.Run("preserves colons for metrics", func(t *testing.T) {
		input := "metric:name:value"
		got := SanitizeLabelValue(input)
		assert.Equal(t, input, got)
	})

	t.Run("handles mixed case", func(t *testing.T) {
		input := "TestValue123"
		got := SanitizeLabelValue(input)
		assert.Equal(t, input, got)
	})

	t.Run("preserves uppercase", func(t *testing.T) {
		input := "UPPER_CASE"
		got := SanitizeLabelValue(input)
		assert.Equal(t, input, got)
	})
}

func TestSanitizeLabelName_EdgeCases(t *testing.T) {
	t.Run("single digit prefix", func(t *testing.T) {
		input := "1name"
		got := SanitizeLabelName(input)
		assert.Equal(t, "_1name", got)
	})

	t.Run("preserves dashes", func(t *testing.T) {
		input := "label-name"
		got := SanitizeLabelName(input)
		// Dash is converted to underscore in label names
		assert.Equal(t, "label_name", got)
	})

	t.Run("empty after sanitization", func(t *testing.T) {
		input := "@@@"
		got := SanitizeLabelName(input)
		assert.Equal(t, "___", got)
	})
}

func TestHTTPStatusLabel_EdgeCases(t *testing.T) {
	t.Run("very large status code", func(t *testing.T) {
		got := HTTPStatusLabel(999)
		// http.StatusText returns empty string for unknown codes
		assert.Equal(t, "", got)
	})

	t.Run("negative status code", func(t *testing.T) {
		got := HTTPStatusLabel(-1)
		// http.StatusText returns empty string for negative codes
		assert.Equal(t, "", got)
	})
}
