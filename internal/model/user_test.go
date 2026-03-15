package model

import "testing"

func TestRolePermissions(t *testing.T) {
	tests := []struct {
		role       Role
		permission string
		expected   bool
	}{
		{RoleOwner, "anything", true},
		{RoleOwner, "api_keys", true},
		{RoleAdmin, "api_keys", true},
		{RoleAdmin, "billing", true},
		{RoleDev, "logs", true},
		{RoleDev, "api_keys:read", true},
		{RoleDev, "billing", false},
		{RoleAnalyst, "logs", true},
		{RoleAnalyst, "billing:read", true},
		{RoleAnalyst, "api_keys", false},
		{RoleBilling, "billing", true},
		{RoleBilling, "logs", false},
		{RoleSupport, "logs:read", true},
		{RoleSupport, "policies", false},
	}

	for _, tt := range tests {
		got := tt.role.HasPermission(tt.permission)
		if got != tt.expected {
			t.Errorf("Role(%s).HasPermission(%s) = %v, want %v", tt.role, tt.permission, got, tt.expected)
		}
	}
}
