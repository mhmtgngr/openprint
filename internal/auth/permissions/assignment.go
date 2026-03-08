// Package permissions provides role-to-permission mapping for OpenPrint.
package permissions

import (
	"fmt"

	"github.com/openprint/openprint/internal/auth/roles"
)

// RolePermissions maps each role to its default set of permissions.
var RolePermissions = map[roles.Role][]Permission{
	roles.RolePlatformAdmin: {
		// Platform admins have full access to everything
		PrintersManage, AgentsManage, JobsCreate, JobsListAll, JobsDelete,
		UsersCreate, UsersManageRoles, OrgsManage, OrgsManageBilling,
		PoliciesCreate, PoliciesAssign, QuotasManage, ReportsViewCosts,
		AuditLogsRead, AuditLogsExport, AuditLogsPurge,
		SettingsManage, CertificatesCreate, CertificatesRevoke, CertificatesRotate,
		AgentsRestart, AgentsViewMetrics, JobsApprove,
	},

	roles.RoleOrgAdmin: {
		// Org admins can manage their organization
		PrintersCreate, PrintersRead, PrintersUpdate, PrintersDelete, PrintersList,
		AgentsCreate, AgentsRead, AgentsUpdate, AgentsDelete, AgentsList, AgentsViewMetrics,
		JobsCreate, JobsRead, JobsList, JobsListAll, JobsCancel, JobsDelete,
		DocumentsCreate, DocumentsRead, DocumentsList,
		UsersCreate, UsersRead, UsersUpdate, UsersDelete, UsersList, UsersManageRoles,
		OrgsRead, OrgsUpdate, OrgsManageSettings,
		PoliciesCreate, PoliciesRead, PoliciesUpdate, PoliciesDelete, PoliciesList, PoliciesAssign,
		QuotasRead, QuotasUpdate, QuotasList,
		ReportsRead, ReportsExport, ReportsViewUsage,
		AuditLogsRead, AuditLogsExport,
		SettingsRead, SettingsUpdate,
	},

	roles.RoleOrgUser: {
		// Standard users can print and view their own resources
		PrintersRead, PrintersList,
		AgentsRead, AgentsList,
		JobsCreate, JobsRead, JobsReadOwn, JobsList, JobsCancel, // Can cancel own jobs
		DocumentsCreate, DocumentsRead, DocumentsReadOwn, DocumentsList,
		ReportsRead, ReportsViewUsage,
		SettingsRead,
	},

	roles.RoleOrgViewer: {
		// Viewers can only read
		PrintersRead, PrintersList,
		AgentsRead, AgentsList,
		JobsRead, JobsReadOwn, JobsList,
		DocumentsRead, DocumentsReadOwn, DocumentsList,
		ReportsRead, ReportsViewUsage,
		SettingsRead,
	},

	// Legacy roles map to new roles
	roles.RoleAdmin: {},  // Maps to platform_admin
	roles.RoleUser:   {},  // Maps to org_user
	roles.RoleViewer: {},  // Maps to org_viewer
}

// GetPermissionsForRole returns the permissions assigned to a given role.
// Handles legacy role normalization automatically.
func GetPermissionsForRole(role roles.Role) []Permission {
	// Normalize legacy roles
	normalized := normalizeRole(role)

	// Fetch permissions for normalized role
	perms, ok := RolePermissions[normalized]
	if !ok {
		return []Permission{}
	}

	// Copy to avoid modifying the original
	result := make([]Permission, len(perms))
	copy(result, perms)

	return result
}

// HasPermission checks if a role has a specific permission.
// Supports permission implication (e.g., "manage" implies "read").
func HasPermission(role roles.Role, perm Permission) bool {
	normalized := normalizeRole(role)
	assignedPerms := RolePermissions[normalized]

	for _, p := range assignedPerms {
		if p == perm || p.Implies(perm) {
			return true
		}
	}

	return false
}

// HasAnyPermission checks if a role has any of the specified permissions.
func HasAnyPermission(role roles.Role, perms []Permission) bool {
	for _, perm := range perms {
		if HasPermission(role, perm) {
			return true
		}
	}
	return false
}

// HasAllPermissions checks if a role has all of the specified permissions.
func HasAllPermissions(role roles.Role, perms []Permission) bool {
	for _, perm := range perms {
		if !HasPermission(role, perm) {
			return false
		}
	}
	return true
}

// FilterAuthorized returns only the permissions that the role is authorized for.
func FilterAuthorized(role roles.Role, perms []Permission) []Permission {
	var result []Permission
	for _, perm := range perms {
		if HasPermission(role, perm) {
			result = append(result, perm)
		}
	}
	return result
}

// normalizeRole converts legacy roles to their modern equivalents.
func normalizeRole(role roles.Role) roles.Role {
	switch role {
	case roles.RoleAdmin:
		return roles.RolePlatformAdmin
	case roles.RoleUser:
		return roles.RoleOrgUser
	case roles.RoleViewer:
		return roles.RoleOrgViewer
	default:
		return role
	}
}

// SetCustomPermissions assigns custom permissions to a role.
// This is useful for organizations that want to customize role permissions.
// Returns the previous permissions for restoration if needed.
func SetCustomPermissions(role roles.Role, perms []Permission) []Permission {
	normalized := normalizeRole(role)

	// Store previous permissions
	previous := RolePermissions[normalized]

	// Set new permissions
	RolePermissions[normalized] = perms

	return previous
}

// AddPermissionToRole adds a single permission to a role's permissions.
func AddPermissionToRole(role roles.Role, perm Permission) {
	normalized := normalizeRole(role)

	current := RolePermissions[normalized]
	for _, p := range current {
		if p == perm {
			return // Already has this permission
		}
	}

	RolePermissions[normalized] = append(current, perm)
}

// RemovePermissionFromRole removes a single permission from a role's permissions.
func RemovePermissionFromRole(role roles.Role, perm Permission) error {
	normalized := normalizeRole(role)

	current := RolePermissions[normalized]
	var updated []Permission
	found := false

	for _, p := range current {
		if p == perm {
			found = true
			continue
		}
		updated = append(updated, p)
	}

	if !found {
		return fmt.Errorf("permission %q not found in role %q", perm, normalized)
	}

	RolePermissions[normalized] = updated
	return nil
}

// RolePermissionSummary provides a summary of permissions for a role.
type RolePermissionSummary struct {
	Role              roles.Role
	PermissionCount   int
	PermissionsByGroup map[string][]Permission
	CanCreate         bool
	CanUpdate         bool
	CanDelete         bool
	CanManage         bool
}

// GetRoleSummary returns a summary of permissions for a given role.
func GetRoleSummary(role roles.Role) RolePermissionSummary {
	normalized := normalizeRole(role)
	perms := RolePermissions[normalized]

	summary := RolePermissionSummary{
		Role:              normalized,
		PermissionCount:   len(perms),
		PermissionsByGroup: make(map[string][]Permission),
		CanCreate:         false,
		CanUpdate:         false,
		CanDelete:         false,
		CanManage:         false,
	}

	for _, perm := range perms {
		// Group by resource
		resource := perm.Resource()
		summary.PermissionsByGroup[resource] = append(summary.PermissionsByGroup[resource], perm)

		// Check capabilities
		switch perm.Action() {
		case ActionCreate:
			summary.CanCreate = true
		case ActionUpdate:
			summary.CanUpdate = true
		case ActionDelete:
			summary.CanDelete = true
		case ActionManage:
			summary.CanManage = true
		}
	}

	return summary
}
