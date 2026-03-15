package model

import "time"

type Policy struct {
	ID            string                 `json:"id" db:"id"`
	TenantID      string                 `json:"tenant_id" db:"tenant_id"`
	Name          string                 `json:"name" db:"name"`
	UseCase       UseCase                `json:"use_case" db:"use_case"`
	Strategy      VerificationType       `json:"strategy" db:"strategy"`
	SIMSwapAction Verdict                `json:"sim_swap_action" db:"sim_swap_action"`
	Countries     []string               `json:"countries" db:"countries"`
	Priority      int                    `json:"priority" db:"priority"`
	Active        bool                   `json:"active" db:"active"`
	Config        map[string]interface{} `json:"config" db:"config"`
	Rules         []PolicyRule           `json:"rules,omitempty"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at" db:"updated_at"`
}

// PolicyRule defines a single condition→action rule within a policy.
type PolicyRule struct {
	Name      string        `json:"name"`
	Condition RuleCondition `json:"condition"`
	Action    RuleAction    `json:"action"`
	Priority  int           `json:"priority"`
	Enabled   bool          `json:"enabled"`
}

// RuleCondition specifies when a rule fires.
type RuleCondition struct {
	// Match criteria (all non-empty fields must match)
	Countries       []string `json:"countries,omitempty"`        // match country code
	Operators       []string `json:"operators,omitempty"`        // match operator/carrier
	UseCases        []string `json:"use_cases,omitempty"`        // match use case
	Channels        []string `json:"channels,omitempty"`         // match verification channel

	// Signal thresholds
	SIMSwapDetected     *bool    `json:"sim_swap_detected,omitempty"`
	SIMSwapMaxAgeHours  *int     `json:"sim_swap_max_age_hours,omitempty"`
	VerificationFailed  *bool    `json:"verification_failed,omitempty"`
	ConfidenceBelow     *float64 `json:"confidence_below,omitempty"`
	DeviceChanged       *bool    `json:"device_changed,omitempty"`
	RiskScoreAbove      *float64 `json:"risk_score_above,omitempty"`

	// Time-based
	HourRange *[2]int `json:"hour_range,omitempty"` // [startHour, endHour] UTC
}

// RuleAction defines what happens when a rule fires.
type RuleAction struct {
	Verdict        Verdict  `json:"verdict"`                     // allow, challenge, block, review
	RiskAdjustment float64  `json:"risk_adjustment,omitempty"`   // add to risk score
	ForceChannel   string   `json:"force_channel,omitempty"`     // force specific OTP channel
	Reason         string   `json:"reason"`                      // human-readable reason
	Notify         bool     `json:"notify,omitempty"`            // emit webhook event
}

type CreatePolicyRequest struct {
	Name          string       `json:"name" binding:"required"`
	UseCase       string       `json:"use_case" binding:"required"`
	Strategy      string       `json:"strategy" binding:"required"`
	SIMSwapAction string       `json:"sim_swap_action"`
	Countries     []string     `json:"countries"`
	Priority      int          `json:"priority"`
	Rules         []PolicyRule `json:"rules,omitempty"`
}

type UpdatePolicyRequest struct {
	Name          *string      `json:"name"`
	Strategy      *string      `json:"strategy"`
	SIMSwapAction *string      `json:"sim_swap_action"`
	Countries     []string     `json:"countries"`
	Priority      *int         `json:"priority"`
	Active        *bool        `json:"active"`
	Rules         []PolicyRule `json:"rules,omitempty"`
}
