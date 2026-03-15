package model

import "time"

type User struct {
	ID           string    `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Name         string    `json:"name" db:"name"`
	Status       string    `json:"status" db:"status"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type Role string

const (
	RoleOwner   Role = "owner"
	RoleAdmin   Role = "admin"
	RoleDev     Role = "developer"
	RoleAnalyst Role = "analyst"
	RoleBilling Role = "billing_manager"
	RoleSupport Role = "support"
)

type UserTenant struct {
	UserID    string    `json:"user_id" db:"user_id"`
	TenantID  string    `json:"tenant_id" db:"tenant_id"`
	Role      Role      `json:"role" db:"role"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type APIKeyRecord struct {
	ID        string    `json:"id" db:"id"`
	TenantID  string    `json:"tenant_id" db:"tenant_id"`
	Name      string    `json:"name" db:"name"`
	KeyPrefix string    `json:"key_prefix" db:"key_prefix"`
	KeyHash   string    `json:"-" db:"key_hash"`
	Scopes    []string  `json:"scopes" db:"scopes"`
	LastUsed  *time.Time `json:"last_used,omitempty" db:"last_used"`
	ExpiresAt *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	CreatedBy string    `json:"created_by" db:"created_by"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Permission checks
var rolePermissions = map[Role][]string{
	RoleOwner:   {"*"},
	RoleAdmin:   {"api_keys", "policies", "webhooks", "logs", "billing", "users"},
	RoleDev:     {"api_keys:read", "policies:read", "logs", "webhooks"},
	RoleAnalyst: {"logs", "billing:read", "policies:read"},
	RoleBilling: {"billing"},
	RoleSupport: {"logs:read"},
}

func (r Role) HasPermission(permission string) bool {
	perms, ok := rolePermissions[r]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == "*" || p == permission {
			return true
		}
		// Check prefix match: "logs" matches "logs:read"
		if len(p) < len(permission) && permission[:len(p)] == p {
			return true
		}
	}
	return false
}

// Request/Response types

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
	Company  string `json:"company" binding:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token    string  `json:"token"`
	User     *User   `json:"user"`
	TenantID string  `json:"tenant_id"`
	Role     Role    `json:"role"`
}

type CreateAPIKeyRequest struct {
	Name   string   `json:"name" binding:"required"`
	Scopes []string `json:"scopes"`
}

type CreateAPIKeyResponse struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Key       string   `json:"key"` // Only shown once
	KeyPrefix string   `json:"key_prefix"`
	Scopes    []string `json:"scopes"`
}

type InviteUserRequest struct {
	Email string `json:"email" binding:"required,email"`
	Role  string `json:"role" binding:"required"`
}
