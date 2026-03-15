package model

import "time"

type VerificationMethod string

const (
	MethodSilent VerificationMethod = "silent"
	MethodSMS    VerificationMethod = "sms"
	MethodWhatsApp VerificationMethod = "whatsapp"
	MethodVoice  VerificationMethod = "voice"
)

type VerificationResult string

const (
	ResultVerified        VerificationResult = "verified"
	ResultFallbackRequired VerificationResult = "fallback_required"
	ResultFailed          VerificationResult = "failed"
)

type VerificationAttempt struct {
	ID               string             `json:"attempt_id" db:"id"`
	SessionID        string             `json:"session_id" db:"session_id"`
	Method           VerificationMethod `json:"method" db:"method"`
	UpstreamProvider string             `json:"upstream_provider" db:"upstream_provider"`
	UpstreamOperator string             `json:"upstream_operator" db:"upstream_operator"`
	Result           VerificationResult `json:"result" db:"result"`
	LatencyMs        int64              `json:"latency_ms" db:"latency_ms"`
	ErrorCode        string             `json:"error_code,omitempty" db:"error_code"`
	CreatedAt        time.Time          `json:"created_at" db:"created_at"`
}

type SilentVerifyRequest struct {
	SessionID string `json:"session_id" binding:"required"`
}

type SilentVerifyResponse struct {
	Status          VerificationResult `json:"status"`
	ConfidenceScore float64            `json:"confidence_score,omitempty"`
	TelcoSignal     string             `json:"telco_signal,omitempty"`
	Token           string             `json:"token,omitempty"`
}

type OTPSendRequest struct {
	SessionID  string `json:"session_id" binding:"required"`
	Channel    string `json:"channel" binding:"required"` // sms, whatsapp, voice
	TemplateID string `json:"template_id,omitempty"`
	Locale     string `json:"locale,omitempty"`
}

type OTPSendResponse struct {
	DeliveryStatus     string `json:"delivery_status"`
	ResendAfterSeconds int    `json:"resend_after_seconds"`
}

type OTPCheckRequest struct {
	SessionID string `json:"session_id" binding:"required"`
	Code      string `json:"code" binding:"required"`
}

type OTPCheckResponse struct {
	Status       VerificationResult `json:"status"`
	Token        string             `json:"token,omitempty"`
	AttemptsLeft int                `json:"attempts_left"`
}
