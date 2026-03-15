package model

import "time"

type Policy struct {
	ID            string            `json:"id" db:"id"`
	TenantID      string            `json:"tenant_id" db:"tenant_id"`
	Name          string            `json:"name" db:"name"`
	UseCase       UseCase           `json:"use_case" db:"use_case"`
	Strategy      VerificationType  `json:"strategy" db:"strategy"`
	SIMSwapAction Verdict           `json:"sim_swap_action" db:"sim_swap_action"`
	Countries     []string          `json:"countries" db:"countries"`
	Priority      int               `json:"priority" db:"priority"`
	Active        bool              `json:"active" db:"active"`
	Config        map[string]interface{} `json:"config" db:"config"`
	CreatedAt     time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at" db:"updated_at"`
}

type CreatePolicyRequest struct {
	Name          string   `json:"name" binding:"required"`
	UseCase       string   `json:"use_case" binding:"required"`
	Strategy      string   `json:"strategy" binding:"required"`
	SIMSwapAction string   `json:"sim_swap_action"`
	Countries     []string `json:"countries"`
	Priority      int      `json:"priority"`
}

type UpdatePolicyRequest struct {
	Name          *string  `json:"name"`
	Strategy      *string  `json:"strategy"`
	SIMSwapAction *string  `json:"sim_swap_action"`
	Countries     []string `json:"countries"`
	Priority      *int     `json:"priority"`
	Active        *bool    `json:"active"`
}
