package model

import "time"

type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

type Verdict string

const (
	VerdictAllow     Verdict = "allow"
	VerdictChallenge Verdict = "challenge"
	VerdictBlock     Verdict = "block"
	VerdictReview    Verdict = "review"
)

type RiskCheck struct {
	ID               string    `json:"risk_check_id" db:"id"`
	SessionID        string    `json:"session_id" db:"session_id"`
	RiskType         string    `json:"risk_type" db:"risk_type"`
	RawSignal        string    `json:"-" db:"raw_signal"`
	NormalizedSignal string    `json:"normalized_signal" db:"normalized_signal"`
	Verdict          Verdict   `json:"verdict" db:"verdict"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

type SIMSwapRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
	CountryCode string `json:"country_code" binding:"required"`
}

type SIMSwapResponse struct {
	SIMSwapDetected bool      `json:"sim_swap_detected"`
	LastChangeTime  string    `json:"last_change_time,omitempty"`
	RiskLevel       RiskLevel `json:"risk_level"`
	Recommendation  Verdict   `json:"recommendation"`
}

type VerdictRequest struct {
	SessionID        string `json:"session_id" binding:"required"`
	VerificationResult string `json:"verification_result,omitempty"`
	SIMSwapResult    *SIMSwapResponse `json:"sim_swap_result,omitempty"`
	DeviceStatus     map[string]interface{} `json:"device_status,omitempty"`
	PolicyID         string `json:"policy_id,omitempty"`
}

type VerdictResponse struct {
	Verdict    Verdict   `json:"verdict"`
	RiskLevel  RiskLevel `json:"risk_level"`
	Reasons    []string  `json:"reasons,omitempty"`
	ActionRequired string `json:"action_required,omitempty"`
}
