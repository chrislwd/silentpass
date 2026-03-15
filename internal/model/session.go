package model

import "time"

type UseCase string

const (
	UseCaseSignup      UseCase = "signup"
	UseCaseLogin       UseCase = "login"
	UseCaseTransaction UseCase = "transaction"
	UseCasePhoneChange UseCase = "phone_change"
)

type VerificationType string

const (
	VerificationSilent      VerificationType = "silent"
	VerificationSilentOrOTP VerificationType = "silent_or_otp"
	VerificationOTPOnly     VerificationType = "otp_only"
)

type SessionStatus string

const (
	SessionPending  SessionStatus = "pending"
	SessionVerified SessionStatus = "verified"
	SessionFailed   SessionStatus = "failed"
	SessionExpired  SessionStatus = "expired"
)

type Session struct {
	ID               string           `json:"session_id" db:"id"`
	TenantID         string           `json:"tenant_id" db:"tenant_id"`
	PhoneHash        string           `json:"-" db:"phone_hash"`
	CountryCode      string           `json:"country_code" db:"country_code"`
	VerificationType VerificationType `json:"verification_type" db:"verification_type"`
	UseCase          UseCase          `json:"use_case" db:"use_case"`
	Status           SessionStatus    `json:"status" db:"status"`
	DeviceIP         string           `json:"-" db:"device_ip"`
	UserAgent        string           `json:"-" db:"user_agent"`
	CallbackURL      string           `json:"-" db:"callback_url"`
	CreatedAt        time.Time        `json:"created_at" db:"created_at"`
	ExpiresAt        time.Time        `json:"expires_at" db:"expires_at"`
	UpdatedAt        time.Time        `json:"updated_at" db:"updated_at"`
}

type CreateSessionRequest struct {
	AppID            string           `json:"app_id" binding:"required"`
	PhoneNumber      string           `json:"phone_number" binding:"required"`
	CountryCode      string           `json:"country_code" binding:"required"`
	VerificationType VerificationType `json:"verification_type" binding:"required"`
	UseCase          UseCase          `json:"use_case" binding:"required"`
	DeviceContext    *DeviceContext   `json:"device_context"`
	CallbackURL      string           `json:"callback_url"`
}

type DeviceContext struct {
	IPAddress string `json:"ip_address"`
	UserAgent string `json:"user_agent"`
}

type CreateSessionResponse struct {
	SessionID         string `json:"session_id"`
	RecommendedAction string `json:"recommended_action"`
	ExpiresAt         string `json:"expires_at"`
}
