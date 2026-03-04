// Package roles provides centralized role constants and validation for OpenPrint.
//
// SECURITY: This package defines all valid roles and provides validation functions
// to prevent authorization bypass through role manipulation. All role comparisons
// should use the constants defined here, not string literals.
package roles

import (
	"errors"
	"strings"
)

// Role represents a user role in the OpenPrint system.
type Role string

const (
	// RolePlatformAdmin is the highest privilege role with full system access.
	RolePlatformAdmin Role = "platform_admin"

	// RoleAdmin is the legacy platform admin role (deprecated, use RolePlatformAdmin).
	RoleAdmin Role = "admin"

	// RoleOrgAdmin is the organization administrator role.
	RoleOrgAdmin Role = "org_admin"

	// RoleOrgUser is the standard organization user role.
	RoleOrgUser Role = "org_user"

	// RoleUser is the legacy standard user role (deprecated, use RoleOrgUser).
	RoleUser Role = "user"

	// RoleOrgViewer is the read-only organization viewer role.
	RoleOrgViewer Role = "org_viewer"

	// RoleViewer is the legacy read-only role (deprecated, use RoleOrgViewer).
	RoleViewer Role = "viewer"
)

var (
	// ErrInvalidRole is returned when an invalid role is provided.
	ErrInvalidRole = errors.New("invalid role")

	// ErrEmptyRole is returned when an empty role is provided.
	ErrEmptyRole = errors.New("role cannot be empty")
)

// AllRoles returns all valid role constants.
func AllRoles() []Role {
	return []Role{
		RolePlatformAdmin,
		RoleAdmin,
		RoleOrgAdmin,
		RoleOrgUser,
		RoleUser,
		RoleOrgViewer,
		RoleViewer,
	}
}

// ValidRoles is a set of all valid roles for quick lookup.
var ValidRoles = make(map[Role]bool)

func init() {
	for _, role := range AllRoles() {
		ValidRoles[role] = true
	}
}

// IsValid checks if a role string is valid.
func IsValid(role string) bool {
	return ValidRoles[Role(role)]
}

// Parse parses a role string into a Role type, validating it.
// Returns ErrInvalidRole if the role is not recognized.
func Parse(role string) (Role, error) {
	if role == "" {
		return "", ErrEmptyRole
	}

	normalized := strings.TrimSpace(strings.ToLower(role))
	r := Role(normalized)

	if !ValidRoles[r] {
		return "", ErrInvalidRole
	}

	return r, nil
}

// MustParse parses a role string and panics if invalid.
// Use this only for constants that are known to be valid at compile time.
func MustParse(role string) Role {
	r, err := Parse(role)
	if err != nil {
		panic(err)
	}
	return r
}

// String returns the string representation of the role.
func (r Role) String() string {
	return string(r)
}

// IsPlatformAdmin checks if the role has platform admin privileges.
// This includes both RolePlatformAdmin and the legacy RoleAdmin.
func (r Role) IsPlatformAdmin() bool {
	return r == RolePlatformAdmin || r == RoleAdmin
}

// IsOrgAdmin checks if the role has organization admin privileges.
// This includes platform admins (who can manage any org) and org admins.
func (r Role) IsOrgAdmin() bool {
	return r.IsPlatformAdmin() || r == RoleOrgAdmin
}

// IsAdmin returns true for any admin-level role.
func (r Role) IsAdmin() bool {
	return r.IsPlatformAdmin() || r == RoleOrgAdmin
}

// IsViewer returns true if the role is a viewer role (read-only).
func (r Role) IsViewer() bool {
	return r == RoleOrgViewer || r == RoleViewer
}

// CanRead returns true if the role can read resources.
// All valid roles can read.
func (r Role) CanRead() bool {
	return IsValid(r.String())
}

// CanWrite returns true if the role can write/create resources.
// Viewers cannot write.
func (r Role) CanWrite() bool {
	return !r.IsViewer()
}

// CanDelete returns true if the role can delete resources.
// Only admins can delete.
func (r Role) CanDelete() bool {
	return r.IsAdmin()
}

// RequiresTenant checks if the role requires tenant context.
// Platform admins may operate without tenant context for platform-level operations.
func (r Role) RequiresTenant() bool {
	return !r.IsPlatformAdmin()
}

// NormalizeRole converts legacy role names to their current equivalents.
// - "admin" -> "platform_admin"
// - "user" -> "org_user"
// - "viewer" -> "org_viewer"
func NormalizeRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "admin":
		return RolePlatformAdmin.String()
	case "user":
		return RoleOrgUser.String()
	case "viewer":
		return RoleOrgViewer.String()
	default:
		return strings.ToLower(strings.TrimSpace(role))
	}
}

// HasHigherPrivilegeThan checks if this role has higher privileges than another.
// Returns true if r > other in the privilege hierarchy.
func (r Role) HasHigherPrivilegeThan(other Role) bool {
	privilegeOrder := map[Role]int{
		RoleViewer:        0,
		RoleOrgViewer:     0,
		RoleUser:          1,
		RoleOrgUser:       1,
		RoleOrgAdmin:      2,
		RoleAdmin:         3,
		RolePlatformAdmin: 3,
	}

	rLevel, rOk := privilegeOrder[r]
	otherLevel, otherOk := privilegeOrder[other]

	if !rOk || !otherOk {
		return false
	}

	return rLevel > otherLevel
}

// Validate validates a role and returns an error if invalid.
// This is the preferred way to validate user-provided roles.
func Validate(role string) error {
	if role == "" {
		return ErrEmptyRole
	}

	if !IsValid(role) {
		return ErrInvalidRole
	}

	return nil
}

// IsLegacyRole checks if the role is a legacy (deprecated) role.
func IsLegacyRole(role string) bool {
	switch Role(strings.ToLower(strings.TrimSpace(role))) {
	case RoleAdmin, RoleUser, RoleViewer:
		return true
	default:
		return false
	}
}
