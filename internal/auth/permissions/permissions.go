// Package permissions provides granular permission constants and validation for OpenPrint.
//
// Permissions follow the resource:action format:
// - Resource: The entity being accessed (e.g., "printers", "jobs", "agents")
// - Action: The operation being performed (e.g., "create", "read", "update", "delete")
//
// This enables fine-grained access control beyond basic role-based permissions.
package permissions

import (
	"strings"
)

// Permission represents a granular permission in the resource:action format.
type Permission string

// Resource constants
const (
	ResourcePrinters      = "printers"
	ResourceAgents        = "agents"
	ResourceJobs          = "jobs"
	ResourceDocuments     = "documents"
	ResourceUsers         = "users"
	ResourceOrganizations = "organizations"
	ResourcePolicies      = "policies"
	ResourceQuotas        = "quotas"
	ResourceReports       = "reports"
	ResourceAuditLogs     = "audit_logs"
	ResourceSettings      = "settings"
	ResourceCertificates  = "certificates"
)

// Action constants
const (
	ActionCreate = "create"
	ActionRead   = "read"
	ActionUpdate = "update"
	ActionDelete = "delete"
	ActionList   = "list"
	ActionManage = "manage" // Full control including delegation
	ActionApprove = "approve"
	ActionExport  = "export"
)

// Printer permissions
const (
	PrintersCreate      Permission = "printers:create"
	PrintersRead        Permission = "printers:read"
	PrintersUpdate      Permission = "printers:update"
	PrintersDelete      Permission = "printers:delete"
	PrintersList        Permission = "printers:list"
	PrintersManage      Permission = "printers:manage"
)

// Agent permissions
const (
	AgentsCreate        Permission = "agents:create"
	AgentsRead          Permission = "agents:read"
	AgentsUpdate        Permission = "agents:update"
	AgentsDelete        Permission = "agents:delete"
	AgentsList          Permission = "agents:list"
	AgentsManage        Permission = "agents:manage"
	AgentsRestart       Permission = "agents:restart"
	AgentsViewMetrics   Permission = "agents:view_metrics"
)

// Job permissions
const (
	JobsCreate          Permission = "jobs:create"
	JobsRead            Permission = "jobs:read"
	JobsReadOwn         Permission = "jobs:read_own"
	JobsUpdate          Permission = "jobs:update"
	JobsDelete          Permission = "jobs:delete"
	JobsCancel          Permission = "jobs:cancel"
	JobsApprove         Permission = "jobs:approve"
	JobsList            Permission = "jobs:list"
	JobsListAll         Permission = "jobs:list_all"
)

// Document permissions
const (
	DocumentsCreate     Permission = "documents:create"
	DocumentsRead       Permission = "documents:read"
	DocumentsReadOwn    Permission = "documents:read_own"
	DocumentsDelete     Permission = "documents:delete"
	DocumentsList       Permission = "documents:list"
)

// User permissions
const (
	UsersCreate         Permission = "users:create"
	UsersRead           Permission = "users:read"
	UsersUpdate         Permission = "users:update"
	UsersDelete         Permission = "users:delete"
	UsersList           Permission = "users:list"
	UsersManageRoles    Permission = "users:manage_roles"
)

// Organization permissions
const (
	OrgsCreate          Permission = "organizations:create"
	OrgsRead            Permission = "organizations:read"
	OrgsUpdate          Permission = "organizations:update"
	OrgsDelete          Permission = "organizations:delete"
	OrgsList            Permission = "organizations:list"
	OrgsManage          Permission = "organizations:manage"
	OrgsManageBilling   Permission = "organizations:manage_billing"
	OrgsManageSettings  Permission = "organizations:manage_settings"
)

// Policy permissions
const (
	PoliciesCreate      Permission = "policies:create"
	PoliciesRead        Permission = "policies:read"
	PoliciesUpdate      Permission = "policies:update"
	PoliciesDelete      Permission = "policies:delete"
	PoliciesList        Permission = "policies:list"
	PoliciesAssign      Permission = "policies:assign"
)

// Quota permissions
const (
	QuotasRead          Permission = "quotas:read"
	QuotasUpdate        Permission = "quotas:update"
	QuotasList          Permission = "quotas:list"
	QuotasManage        Permission = "quotas:manage"
)

// Report permissions
const (
	ReportsRead         Permission = "reports:read"
	ReportsExport       Permission = "reports:export"
	ReportsViewCosts    Permission = "reports:view_costs"
	ReportsViewUsage    Permission = "reports:view_usage"
)

// Audit log permissions
const (
	AuditLogsRead       Permission = "audit_logs:read"
	AuditLogsExport     Permission = "audit_logs:export"
	AuditLogsPurge      Permission = "audit_logs:purge"
)

// Settings permissions
const (
	SettingsRead        Permission = "settings:read"
	SettingsUpdate      Permission = "settings:update"
	SettingsManage      Permission = "settings:manage"
)

// Certificate permissions
const (
	CertificatesCreate  Permission = "certificates:create"
	CertificatesRead    Permission = "certificates:read"
	CertificatesRevoke  Permission = "certificates:revoke"
	CertificatesRotate  Permission = "certificates:rotate"
)

// AllPermissions returns a list of all defined permissions.
func AllPermissions() []Permission {
	return []Permission{
		// Printer permissions
		PrintersCreate, PrintersRead, PrintersUpdate, PrintersDelete, PrintersList, PrintersManage,
		// Agent permissions
		AgentsCreate, AgentsRead, AgentsUpdate, AgentsDelete, AgentsList, AgentsManage, AgentsRestart, AgentsViewMetrics,
		// Job permissions
		JobsCreate, JobsRead, JobsReadOwn, JobsUpdate, JobsDelete, JobsCancel, JobsApprove, JobsList, JobsListAll,
		// Document permissions
		DocumentsCreate, DocumentsRead, DocumentsReadOwn, DocumentsDelete, DocumentsList,
		// User permissions
		UsersCreate, UsersRead, UsersUpdate, UsersDelete, UsersList, UsersManageRoles,
		// Organization permissions
		OrgsCreate, OrgsRead, OrgsUpdate, OrgsDelete, OrgsList, OrgsManage, OrgsManageBilling, OrgsManageSettings,
		// Policy permissions
		PoliciesCreate, PoliciesRead, PoliciesUpdate, PoliciesDelete, PoliciesList, PoliciesAssign,
		// Quota permissions
		QuotasRead, QuotasUpdate, QuotasList, QuotasManage,
		// Report permissions
		ReportsRead, ReportsExport, ReportsViewCosts, ReportsViewUsage,
		// Audit log permissions
		AuditLogsRead, AuditLogsExport, AuditLogsPurge,
		// Settings permissions
		SettingsRead, SettingsUpdate, SettingsManage,
		// Certificate permissions
		CertificatesCreate, CertificatesRead, CertificatesRevoke, CertificatesRotate,
	}
}

// Parse parses a permission string into a Permission type.
func Parse(perm string) Permission {
	return Permission(strings.ToLower(strings.TrimSpace(perm)))
}

// String returns the string representation of the permission.
func (p Permission) String() string {
	return string(p)
}

// Resource extracts the resource part from a permission.
func (p Permission) Resource() string {
	parts := strings.Split(string(p), ":")
	if len(parts) >= 1 {
		return parts[0]
	}
	return ""
}

// Action extracts the action part from a permission.
func (p Permission) Action() string {
	parts := strings.Split(string(p), ":")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// IsValid checks if the permission format is valid (resource:action).
func (p Permission) IsValid() bool {
	parts := strings.Split(string(p), ":")
	return len(parts) == 2 && parts[0] != "" && parts[1] != ""
}

// Implies checks if this permission implies another (e.g., "printers:manage" implies "printers:read").
func (p Permission) Implies(other Permission) bool {
	if p == other {
		return true
	}

	// manage implies all actions on the same resource
	if p.Action() == ActionManage && p.Resource() == other.Resource() {
		return true
	}

	// list_all implies read and list
	if p == JobsListAll && other == JobsRead {
		return true
	}
	if p == JobsListAll && other == JobsList {
		return true
	}

	// update implies read
	if p.Action() == ActionUpdate && p.Resource() == other.Resource() && other.Action() == ActionRead {
		return true
	}

	return false
}

// PermissionGroup represents a logical grouping of permissions.
type PermissionGroup struct {
	Name        string
	Description string
	Permissions []Permission
}

// PermissionGroups returns predefined permission groupings for UI display.
func PermissionGroups() []PermissionGroup {
	return []PermissionGroup{
		{
			Name:        "Printer Management",
			Description: "Control access to printer resources",
			Permissions: []Permission{PrintersCreate, PrintersRead, PrintersUpdate, PrintersDelete, PrintersList, PrintersManage},
		},
		{
			Name:        "Agent Management",
			Description: "Control access to print agents",
			Permissions: []Permission{AgentsCreate, AgentsRead, AgentsUpdate, AgentsDelete, AgentsList, AgentsManage, AgentsRestart, AgentsViewMetrics},
		},
		{
			Name:        "Job Management",
			Description: "Control access to print jobs",
			Permissions: []Permission{JobsCreate, JobsRead, JobsReadOwn, JobsUpdate, JobsDelete, JobsCancel, JobsApprove, JobsList, JobsListAll},
		},
		{
			Name:        "User Management",
			Description: "Control access to user accounts",
			Permissions: []Permission{UsersCreate, UsersRead, UsersUpdate, UsersDelete, UsersList, UsersManageRoles},
		},
		{
			Name:        "Organization Management",
			Description: "Control access to organization settings",
			Permissions: []Permission{OrgsCreate, OrgsRead, OrgsUpdate, OrgsDelete, OrgsList, OrgsManage, OrgsManageBilling, OrgsManageSettings},
		},
		{
			Name:        "Reporting & Analytics",
			Description: "Access to reports and usage data",
			Permissions: []Permission{ReportsRead, ReportsExport, ReportsViewCosts, ReportsViewUsage},
		},
		{
			Name:        "System Administration",
			Description: "Administrative functions",
			Permissions: []Permission{AuditLogsRead, AuditLogsExport, SettingsRead, SettingsUpdate, CertificatesCreate, CertificatesRevoke},
		},
	}
}

// PermissionDescription returns a human-readable description for a permission.
func PermissionDescription(p Permission) string {
	descriptions := map[Permission]string{
		// Printer permissions
		PrintersCreate:   "Add new printers to the system",
		PrintersRead:     "View printer details and configuration",
		PrintersUpdate:   "Modify printer settings and configuration",
		PrintersDelete:   "Remove printers from the system",
		PrintersList:     "View list of all printers",
		PrintersManage:   "Full control over printer resources",

		// Agent permissions
		AgentsCreate:     "Register new print agents",
		AgentsRead:       "View agent details and status",
		AgentsUpdate:     "Modify agent configuration",
		AgentsDelete:     "Remove agents from the system",
		AgentsList:       "View list of all agents",
		AgentsManage:     "Full control over agent resources",
		AgentsRestart:    "Restart agent services",
		AgentsViewMetrics: "View agent performance metrics",

		// Job permissions
		JobsCreate:       "Submit new print jobs",
		JobsRead:         "View any print job details",
		JobsReadOwn:      "View only own print job details",
		JobsUpdate:       "Modify print job settings",
		JobsDelete:       "Cancel print jobs",
		JobsCancel:       "Cancel queued or active print jobs",
		JobsApprove:      "Approve print jobs requiring approval",
		JobsList:         "View list of print jobs",
		JobsListAll:      "View all print jobs across the organization",

		// User permissions
		UsersCreate:      "Create new user accounts",
		UsersRead:        "View user account details",
		UsersUpdate:      "Modify user account settings",
		UsersDelete:      "Remove user accounts",
		UsersList:        "View list of all users",
		UsersManageRoles: "Assign and modify user roles",

		// Organization permissions
		OrgsCreate:       "Create new organizations",
		OrgsRead:         "View organization details",
		OrgsUpdate:       "Modify organization settings",
		OrgsDelete:       "Remove organizations",
		OrgsList:         "View list of all organizations",
		OrgsManage:       "Full control over organization",
		OrgsManageBilling: "Manage billing and subscriptions",
		OrgsManageSettings: "Manage organization-wide settings",

		// Report permissions
		ReportsRead:      "View usage and performance reports",
		ReportsExport:    "Export reports in various formats",
		ReportsViewCosts: "View cost and billing information",
		ReportsViewUsage: "View usage statistics and analytics",

		// Audit permissions
		AuditLogsRead:    "View audit log entries",
		AuditLogsExport:  "Export audit log data",
		AuditLogsPurge:   "Purge old audit log entries",

		// Settings permissions
		SettingsRead:     "View system settings",
		SettingsUpdate:   "Modify system configuration",
		SettingsManage:   "Full control over system settings",

		// Certificate permissions
		CertificatesCreate: "Create new agent certificates",
		CertificatesRead:   "View certificate details",
		CertificatesRevoke: "Revoke agent certificates",
		CertificatesRotate: "Rotate agent certificates",
	}

	if desc, ok := descriptions[p]; ok {
		return desc
	}

	// Default description
	return "Allows " + p.Action() + " on " + p.Resource()
}
