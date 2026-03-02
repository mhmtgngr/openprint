// Package roles provides tests for role validation and authorization.
package roles

import (
	"testing"
)

func TestIsValid(t *testing.T) {
	tests := []struct {
		role     string
		expected bool
	}{
		{"platform_admin", true},
		{"admin", true},
		{"org_admin", true},
		{"org_user", true},
		{"user", true},
		{"org_viewer", true},
		{"viewer", true},
		{"PLATFORM_ADMIN", false}, // case sensitive - must be lowercase
		{"", false},
		{"hacker", false},
		{"superadmin", false},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			result := IsValid(tt.role)
			if result != tt.expected {
				t.Errorf("IsValid(%q) = %v, want %v", tt.role, result, tt.expected)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		role     string
		expected Role
		wantErr  bool
	}{
		{"platform_admin", RolePlatformAdmin, false},
		{"  platform_admin  ", RolePlatformAdmin, false}, // with whitespace
		{"PLATFORM_ADMIN", RolePlatformAdmin, false},     // case insensitive
		{"admin", RoleAdmin, false},
		{"org_admin", RoleOrgAdmin, false},
		{"org_user", RoleOrgUser, false},
		{"user", RoleUser, false},
		{"org_viewer", RoleOrgViewer, false},
		{"viewer", RoleViewer, false},
		{"", "", true},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			result, err := Parse(tt.role)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse(%q) error = %v, wantErr %v", tt.role, err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("Parse(%q) = %v, want %v", tt.role, result, tt.expected)
			}
		})
	}
}

func TestRole_IsPlatformAdmin(t *testing.T) {
	tests := []struct {
		role     Role
		expected bool
	}{
		{RolePlatformAdmin, true},
		{RoleAdmin, true}, // legacy
		{RoleOrgAdmin, false},
		{RoleOrgUser, false},
		{RoleUser, false},
		{RoleOrgViewer, false},
		{RoleViewer, false},
	}

	for _, tt := range tests {
		t.Run(tt.role.String(), func(t *testing.T) {
			if tt.role.IsPlatformAdmin() != tt.expected {
				t.Errorf("%v.IsPlatformAdmin() = %v, want %v", tt.role, tt.role.IsPlatformAdmin(), tt.expected)
			}
		})
	}
}

func TestRole_IsOrgAdmin(t *testing.T) {
	tests := []struct {
		role     Role
		expected bool
	}{
		{RolePlatformAdmin, true}, // platform admins are also org admins
		{RoleAdmin, true},         // legacy admin
		{RoleOrgAdmin, true},
		{RoleOrgUser, false},
		{RoleUser, false},
		{RoleOrgViewer, false},
		{RoleViewer, false},
	}

	for _, tt := range tests {
		t.Run(tt.role.String(), func(t *testing.T) {
			if tt.role.IsOrgAdmin() != tt.expected {
				t.Errorf("%v.IsOrgAdmin() = %v, want %v", tt.role, tt.role.IsOrgAdmin(), tt.expected)
			}
		})
	}
}

func TestRole_CanWrite(t *testing.T) {
	tests := []struct {
		role     Role
		expected bool
	}{
		{RolePlatformAdmin, true},
		{RoleAdmin, true},
		{RoleOrgAdmin, true},
		{RoleOrgUser, true},
		{RoleUser, true},
		{RoleOrgViewer, false},
		{RoleViewer, false},
	}

	for _, tt := range tests {
		t.Run(tt.role.String(), func(t *testing.T) {
			if tt.role.CanWrite() != tt.expected {
				t.Errorf("%v.CanWrite() = %v, want %v", tt.role, tt.role.CanWrite(), tt.expected)
			}
		})
	}
}

func TestRole_CanDelete(t *testing.T) {
	tests := []struct {
		role     Role
		expected bool
	}{
		{RolePlatformAdmin, true},
		{RoleAdmin, true},
		{RoleOrgAdmin, true},
		{RoleOrgUser, false},
		{RoleUser, false},
		{RoleOrgViewer, false},
		{RoleViewer, false},
	}

	for _, tt := range tests {
		t.Run(tt.role.String(), func(t *testing.T) {
			if tt.role.CanDelete() != tt.expected {
				t.Errorf("%v.CanDelete() = %v, want %v", tt.role, tt.role.CanDelete(), tt.expected)
			}
		})
	}
}

func TestNormalizeRole(t *testing.T) {
	tests := []struct {
		role     string
		expected string
	}{
		{"admin", "platform_admin"},
		{"  admin  ", "platform_admin"},
		{"ADMIN", "platform_admin"},
		{"user", "org_user"},
		{"viewer", "org_viewer"},
		{"org_admin", "org_admin"},
		{"org_user", "org_user"},
		{"org_viewer", "org_viewer"},
		{"platform_admin", "platform_admin"},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			result := NormalizeRole(tt.role)
			if result != tt.expected {
				t.Errorf("NormalizeRole(%q) = %q, want %q", tt.role, result, tt.expected)
			}
		})
	}
}

func TestHasHigherPrivilegeThan(t *testing.T) {
	tests := []struct {
		role     Role
		other    Role
		expected bool
	}{
		{RolePlatformAdmin, RoleOrgAdmin, true},
		{RolePlatformAdmin, RoleOrgUser, true},
		{RoleOrgAdmin, RoleOrgUser, true},
		{RoleOrgUser, RoleOrgViewer, true},
		{RoleOrgViewer, RoleOrgUser, false},
		{RoleOrgAdmin, RolePlatformAdmin, false},
		{RoleOrgUser, RoleOrgUser, false}, // same privilege
	}

	for _, tt := range tests {
		t.Run(tt.role.String()+"_vs_"+tt.other.String(), func(t *testing.T) {
			result := tt.role.HasHigherPrivilegeThan(tt.other)
			if result != tt.expected {
				t.Errorf("%v.HasHigherPrivilegeThan(%v) = %v, want %v",
					tt.role, tt.other, result, tt.expected)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		role    string
		wantErr bool
	}{
		{"platform_admin", false},
		{"admin", false},
		{"org_admin", false},
		{"", true},
		{"invalid", true},
		{"hacker", true},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			err := Validate(tt.role)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate(%q) error = %v, wantErr %v", tt.role, err, tt.wantErr)
			}
		})
	}
}

func TestIsLegacyRole(t *testing.T) {
	tests := []struct {
		role     string
		expected bool
	}{
		{"admin", true},
		{"user", true},
		{"viewer", true},
		{"platform_admin", false},
		{"org_admin", false},
		{"org_user", false},
		{"org_viewer", false},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			result := IsLegacyRole(tt.role)
			if result != tt.expected {
				t.Errorf("IsLegacyRole(%q) = %v, want %v", tt.role, result, tt.expected)
			}
		})
	}
}
