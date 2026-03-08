// Package agent provides types for agent group management.
package agent

import "time"

// AgentGroup represents a logical grouping of agents.
// Groups can be used to organize agents by location, department, floor, etc.
type AgentGroup struct {
	// ID is the unique identifier for this group
	ID string `json:"id"`
	// Name is the human-readable name for this group
	Name string `json:"name"`
	// Description is an optional description of the group
	Description string `json:"description,omitempty"`
	// OrganizationID is the organization this group belongs to
	OrganizationID string `json:"organization_id,omitempty"`
	// OwnerUserID is the user who manages this group (optional)
	OwnerUserID string `json:"owner_user_id,omitempty"`
	// Type indicates what kind of group this is (location, department, custom, etc.)
	Type GroupType `json:"type"`
	// Location is the physical location this group represents (optional)
	Location string `json:"location,omitempty"`
	// Tags are user-defined labels for this group
	Tags []string `json:"tags,omitempty"`
	// PolicyID is an optional policy applied to all agents in this group
	PolicyID string `json:"policy_id,omitempty"`
	// Config is group-level configuration that applies to all agents
	Config map[string]interface{} `json:"config,omitempty"`
	// CreatedAt is when this group was created
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is when this group was last updated
	UpdatedAt time.Time `json:"updated_at"`
}

// GroupType represents the type/kind of agent group.
type GroupType string

const (
	// GroupTypeLocation represents a location-based group (floor, building, etc.)
	GroupTypeLocation GroupType = "location"
	// GroupTypeDepartment represents a department-based group
	GroupTypeDepartment GroupType = "department"
	// GroupTypeCustom represents a user-defined custom group
	GroupTypeCustom GroupType = "custom"
	// GroupTypeAuto represents an automatically generated group
	GroupTypeAuto GroupType = "auto"
)

// AgentGroupMembership represents an agent's membership in a group.
type AgentGroupMembership struct {
	// ID is the unique identifier for this membership
	ID string `json:"id"`
	// GroupID is the group this membership belongs to
	GroupID string `json:"group_id"`
	// AgentID is the agent in this group
	AgentID string `json:"agent_id"`
	// AddedAt is when this agent was added to the group
	AddedAt time.Time `json:"added_at"`
	// AddedBy is the user who added this agent to the group
	AddedBy string `json:"added_by,omitempty"`
}

// GroupPolicy represents a policy that can be applied to an agent group.
type GroupPolicy struct {
	// ID is the unique identifier for this policy
	ID string `json:"id"`
	// Name is the policy name
	Name string `json:"name"`
	// Description is an optional description
	Description string `json:"description,omitempty"`
	// OrganizationID is the organization this policy belongs to
	OrganizationID string `json:"organization_id,omitempty"`
	// Config is the policy configuration
	Config PolicyConfig `json:"config"`
	// CreatedAt is when this policy was created
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is when this policy was last updated
	UpdatedAt time.Time `json:"updated_at"`
}

// PolicyConfig contains configuration settings for a group policy.
type PolicyConfig struct {
	// HeartbeatInterval is how often agents should send heartbeats (seconds)
	HeartbeatInterval int `json:"heartbeat_interval,omitempty"`
	// JobPollInterval is how often agents should poll for jobs (seconds)
	JobPollInterval int `json:"job_poll_interval,omitempty"`
	// MaxConcurrentJobs is the maximum number of concurrent jobs per agent
	MaxConcurrentJobs int `json:"max_concurrent_jobs,omitempty"`
	// AllowedDocumentTypes is a list of allowed document types
	AllowedDocumentTypes []string `json:"allowed_document_types,omitempty"`
	// RequireSSL indicates if agents must use SSL/TLS
	RequireSSL bool `json:"require_ssl,omitempty"`
	// AutoUpdateEnabled indicates if agents should auto-update
	AutoUpdateEnabled bool `json:"auto_update_enabled,omitempty"`
	// MaintenanceWindow is when agents can be restarted for updates
	MaintenanceWindow *MaintenanceWindow `json:"maintenance_window,omitempty"`
	// CustomConfig is for any additional custom settings
	CustomConfig map[string]interface{} `json:"custom_config,omitempty"`
}

// MaintenanceWindow represents a time window for maintenance operations.
type MaintenanceWindow struct {
	// Start is the start time in HH:MM format
	Start string `json:"start"`
	// End is the end time in HH:MM format
	End string `json:"end"`
	// DaysOfWeek are the days this window applies to (0=Sunday, 6=Saturday)
	DaysOfWeek []int `json:"days_of_week"`
	// Timezone is the timezone for the window (e.g., "America/New_York")
	Timezone string `json:"timezone"`
}

// GroupStatus represents the current status of an agent group.
type GroupStatus struct {
	// GroupID is the group this status is for
	GroupID string `json:"group_id"`
	// TotalAgents is the total number of agents in the group
	TotalAgents int `json:"total_agents"`
	// OnlineAgents is the number of agents currently online
	OnlineAgents int `json:"online_agents"`
	// OfflineAgents is the number of agents currently offline
	OfflineAgents int `json:"offline_agents"`
	// ErrorAgents is the number of agents in error state
	ErrorAgents int `json:"error_agents"`
	// TotalPrinters is the total number of printers across all agents
	TotalPrinters int `json:"total_printers"`
	// OnlinePrinters is the number of printers currently online
	OnlinePrinters int `json:"online_printers"`
	// ActiveJobs is the number of currently active print jobs
	ActiveJobs int `json:"active_jobs"`
	// LastUpdated is when this status was last calculated
	LastUpdated time.Time `json:"last_updated"`
}

// GroupAssignmentRequest represents a request to assign agents to a group.
type GroupAssignmentRequest struct {
	// GroupID is the group to assign agents to
	GroupID string `json:"group_id"`
	// AgentIDs is the list of agents to assign
	AgentIDs []string `json:"agent_ids"`
	// Replace indicates if this should replace existing assignments
	Replace bool `json:"replace,omitempty"`
}

// GroupAssignmentResponse represents the response to a group assignment.
type GroupAssignmentResponse struct {
	// Assigned is the count of newly assigned agents
	Assigned int `json:"assigned"`
	// Removed is the count of removed agents (if Replace=true)
	Removed int `json:"removed,omitempty"`
	// FailedAgents is a list of agent IDs that failed to assign
	FailedAgents []string `json:"failed_agents,omitempty"`
}

// AgentMetrics represents performance metrics for an agent.
type AgentMetrics struct {
	// AgentID is the agent this metrics are for
	AgentID string `json:"agent_id"`
	// Timestamp is when these metrics were collected
	Timestamp time.Time `json:"timestamp"`
	// CPUUsage is the current CPU usage percentage (0-100)
	CPUUsage float64 `json:"cpu_usage,omitempty"`
	// MemoryUsage is the current memory usage in bytes
	MemoryUsage int64 `json:"memory_usage,omitempty"`
	// MemoryTotal is the total available memory in bytes
	MemoryTotal int64 `json:"memory_total,omitempty"`
	// DiskUsage is the current disk usage in bytes
	DiskUsage int64 `json:"disk_usage,omitempty"`
	// DiskTotal is the total disk space in bytes
	DiskTotal int64 `json:"disk_total,omitempty"`
	// JobSuccessRate is the percentage of successful jobs (0-100)
	JobSuccessRate float64 `json:"job_success_rate,omitempty"`
	// JobSuccessRateWindow is the time window for the success rate calculation
	JobSuccessRateWindow string `json:"job_success_rate_window,omitempty"`
	// AverageJobDuration is the average job completion time in milliseconds
	AverageJobDuration int64 `json:"average_job_duration_ms,omitempty"`
	// TotalJobs is the total number of jobs processed
	TotalJobs int64 `json:"total_jobs,omitempty"`
	// FailedJobs is the number of failed jobs
	FailedJobs int64 `json:"failed_jobs,omitempty"`
	// Uptime is the agent uptime in seconds
	Uptime int64 `json:"uptime_seconds,omitempty"`
	// ActiveConnections is the number of active network connections
	ActiveConnections int `json:"active_connections,omitempty"`
	// QueueDepth is the current job queue depth
	QueueDepth int `json:"queue_depth,omitempty"`
}

// AgentAlert represents an alert condition for an agent.
type AgentAlert struct {
	// ID is the unique identifier for this alert
	ID string `json:"id"`
	// AgentID is the agent this alert is for
	AgentID string `json:"agent_id,omitempty"`
	// GroupID is the group this alert is for (all agents in group)
	GroupID string `json:"group_id,omitempty"`
	// Type is the alert type
	Type AlertType `json:"type"`
	// Severity is the alert severity
	Severity AlertSeverity `json:"severity"`
	// Message is the alert message
	Message string `json:"message"`
	// Resolved indicates if this alert has been resolved
	Resolved bool `json:"resolved"`
	// ResolvedAt is when this alert was resolved
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
	// CreatedAt is when this alert was created
	CreatedAt time.Time `json:"created_at"`
	// Metadata contains additional alert-specific data
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// AlertType represents the type of alert.
type AlertType string

const (
	// AlertTypeAgentOffline indicates an agent has gone offline
	AlertTypeAgentOffline AlertType = "agent_offline"
	// AlertTypeAgentError indicates an agent is in error state
	AlertTypeAgentError AlertType = "agent_error"
	// AlertTypeHighFailureRate indicates high job failure rate
	AlertTypeHighFailureRate AlertType = "high_failure_rate"
	// AlertTypeHighMemoryUsage indicates high memory usage
	AlertTypeHighMemoryUsage AlertType = "high_memory_usage"
	// AlertTypeHighCPUUsage indicates high CPU usage
	AlertTypeHighCPUUsage AlertType = "high_cpu_usage"
	// AlertTypeLowDiskSpace indicates low disk space
	AlertTypeLowDiskSpace AlertType = "low_disk_space"
	// AlertTypeQueueBacklog indicates job queue backlog
	AlertTypeQueueBacklog AlertType = "queue_backlog"
	// AlertTypeCertificateExpiring indicates agent certificate is expiring
	AlertTypeCertificateExpiring AlertType = "certificate_expiring"
	// AlertTypeAgentOffline indicates an agent needs update
	AlertTypeAgentUpdateAvailable AlertType = "agent_update_available"
)

// AlertSeverity represents the severity level of an alert.
type AlertSeverity string

const (
	// SeverityInfo is for informational alerts
	SeverityInfo AlertSeverity = "info"
	// SeverityWarning is for warnings that need attention
	SeverityWarning AlertSeverity = "warning"
	// SeverityCritical is for critical issues requiring immediate action
	SeverityCritical AlertSeverity = "critical"
)
