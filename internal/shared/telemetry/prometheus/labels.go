// Package prometheus provides standardized label names and values for metrics.
package prometheus

import (
	"net/http"
	"strings"
)

// LabelNames contains the standard label names used across all metrics.
// Using consistent label names enables querying and aggregation in Prometheus.
type LabelNames struct{}

// Standard label names for HTTP metrics.
const (
	LabelServiceName    = "service_name"
	LabelServiceVersion = "service_version"
	LabelNamespace      = "namespace"
	LabelMethod         = "method"          // HTTP method
	LabelPath           = "path"            // HTTP path template
	LabelStatus         = "status"          // HTTP status code
	LabelErrorCode      = "error_code"      // Application error code
	LabelPeer           = "peer"            // Remote service/peer name
	LabelDBName         = "db_name"         // Database name
	LabelDBSystem       = "db_system"       // Database type (postgresql, etc.)
	LabelOperation      = "operation"       // Database operation name
	LabelRedisDB        = "redis_db"        // Redis database number
	LabelRedisCommand   = "redis_command"   // Redis command name
	LabelSuccess        = "success"         // Boolean for operation success
	LabelAuthMethod     = "auth_method"     // Authentication method (password, oidc, saml)
	LabelRole           = "role"            // User role
	LabelOrgID          = "org_id"          // Organization ID
	LabelPrinterID      = "printer_id"      // Printer ID
	LabelJobID          = "job_id"          // Job ID
	LabelJobStatus      = "job_status"      // Job status
	LabelDocumentType   = "document_type"   // Document MIME type
	LabelStorageBackend = "storage_backend" // Storage backend (s3, local)
	LabelLeveled        = "le"              // Histogram bucket label
	LabelQuantile       = "quantile"        // Summary quantile label
)

// LabelValues contains standard label values.
type LabelValues struct{}

// Service names.
const (
	ServiceAuthService         = "auth-service"
	ServiceRegistryService     = "registry-service"
	ServiceJobService          = "job-service"
	ServiceStorageService      = "storage-service"
	ServiceNotificationService = "notification-service"
)

// Database system values.
const (
	DBSystemPostgreSQL = "postgresql"
	DBSystemMySQL      = "mysql"
	DBSystemSQLite     = "sqlite"
)

// Storage backend values.
const (
	StorageBackendS3    = "s3"
	StorageBackendLocal = "local"
)

// Job status values.
const (
	JobStatusPending    = "pending"
	JobStatusQueued     = "queued"
	JobStatusProcessing = "processing"
	JobStatusCompleted  = "completed"
	JobStatusFailed     = "failed"
	JobStatusCancelled  = "cancelled"
)

// Authentication method values.
const (
	AuthMethodPassword = "password"
	AuthMethodOIDC     = "oidc"
	AuthMethodSAML     = "saml"
	AuthMethodAPIKey   = "api_key"
)

// Success values.
const (
	SuccessTrue  = "true"
	SuccessFalse = "false"
)

// HTTPStatusLabel returns the HTTP status code label value.
func HTTPStatusLabel(code int) string {
	return strings.ReplaceAll(http.StatusText(code), " ", "_")
}

// HTTPStatusClass returns the HTTP status class (2xx, 3xx, 4xx, 5xx).
func HTTPStatusClass(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}

// SanitizeLabelValue sanitizes a string for use as a Prometheus label value.
// Prometheus label values must match: [a-zA-Z_][a-zA-Z0-9_]*
func SanitizeLabelValue(s string) string {
	// Replace invalid characters with underscores
	var result []rune
	for i, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == ':' {
			result = append(result, r)
		} else {
			result = append(result, '_')
		}

		// Limit label value length
		if i >= 200 {
			break
		}
	}

	return string(result)
}

// SanitizeLabelName sanitizes a string for use as a Prometheus label name.
// Prometheus label names must match: [a-zA-Z_][a-zA-Z0-9_]*
func SanitizeLabelName(s string) string {
	// Replace invalid characters with underscores
	var result []rune
	for i, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			result = append(result, r)
		} else {
			result = append(result, '_')
		}

		// Limit label name length
		if i >= 200 {
			break
		}
	}

	// Ensure name doesn't start with a number
	if len(result) > 0 && result[0] >= '0' && result[0] <= '9' {
		result = append([]rune{'_'}, result...)
	}

	return string(result)
}

// Labels is a helper type for building label sets.
type Labels map[string]string

// NewLabels creates a new Labels map.
func NewLabels() Labels {
	return make(Labels)
}

// With adds a key-value pair to the labels.
func (l Labels) With(key, value string) Labels {
	l[key] = value
	return l
}

// WithService adds the service name label.
func (l Labels) WithService(service string) Labels {
	l[LabelServiceName] = service
	return l
}

// WithMethod adds the HTTP method label.
func (l Labels) WithMethod(method string) Labels {
	l[LabelMethod] = method
	return l
}

// WithPath adds the path label.
func (l Labels) WithPath(path string) Labels {
	l[LabelPath] = path
	return l
}

// WithStatus adds the HTTP status code label.
func (l Labels) WithStatus(status int) Labels {
	l[LabelStatus] = HTTPStatusLabel(status)
	return l
}

// WithSuccess adds the success label.
func (l Labels) WithSuccess(success bool) Labels {
	if success {
		l[LabelSuccess] = SuccessTrue
	} else {
		l[LabelSuccess] = SuccessFalse
	}
	return l
}

// WithAuthMethod adds the auth method label.
func (l Labels) WithAuthMethod(method string) Labels {
	l[LabelAuthMethod] = method
	return l
}

// WithJobStatus adds the job status label.
func (l Labels) WithJobStatus(status string) Labels {
	l[LabelJobStatus] = status
	return l
}

// WithDBOperation adds the database operation label.
func (l Labels) WithDBOperation(operation string) Labels {
	l[LabelOperation] = operation
	return l
}

// WithRedisCommand adds the Redis command label.
func (l Labels) WithRedisCommand(command string) Labels {
	l[LabelRedisCommand] = command
	return l
}

// ToPrometheusLabels converts Labels to prometheus.Labels.
func (l Labels) ToPrometheusLabels() map[string]string {
	return map[string]string(l)
}

// Merge combines multiple label maps into one.
// Later values override earlier values for duplicate keys.
func Merge(labels ...Labels) Labels {
	result := NewLabels()
	for _, l := range labels {
		for k, v := range l {
			result[k] = v
		}
	}
	return result
}

// CommonLabels returns labels that should be present on all metrics.
func CommonLabels(serviceName, serviceVersion string) Labels {
	return NewLabels().
		WithService(serviceName).
		With("service_version", serviceVersion).
		With("namespace", "openprint")
}
