package telco

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/silentpass/silentpass/internal/model"
)

// VonageConfig holds Vonage API credentials.
type VonageConfig struct {
	BaseURL       string   // e.g. "https://api-eu.vonage.com"
	ApplicationID string   // Vonage Application ID
	PrivateKey    []byte   // PEM-encoded private key for JWT signing
	Countries     []string // ISO country codes
}

// VonageAdapter implements the Adapter interface for Vonage Network APIs.
// Vonage exposes CAMARA-compliant Number Verification and SIM Swap APIs
// through their Network API platform.
type VonageAdapter struct {
	config     VonageConfig
	httpClient *http.Client
}

func NewVonageAdapter(cfg VonageConfig) *VonageAdapter {
	return &VonageAdapter{
		config:     cfg,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (a *VonageAdapter) Name() string {
	return "vonage"
}

func (a *VonageAdapter) SupportedCountries() []string {
	return a.config.Countries
}

func (a *VonageAdapter) SupportedCapabilities() []string {
	return []string{"silent_verify", "sim_swap"}
}

// generateJWT creates a short-lived JWT for Vonage API authentication.
func (a *VonageAdapter) generateJWT() (string, error) {
	key, err := jwt.ParseRSAPrivateKeyFromPEM(a.config.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("parse private key: %w", err)
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iat":            now.Unix(),
		"exp":            now.Add(15 * time.Minute).Unix(),
		"jti":            uuid.New().String(),
		"application_id": a.config.ApplicationID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(key)
}

// Vonage Number Verification types
type vonageNumberVerifyRequest struct {
	PhoneNumber string `json:"phoneNumber"`
}

type vonageNumberVerifyResponse struct {
	DevicePhoneNumberVerified bool   `json:"devicePhoneNumberVerified"`
	ResponseCode              string `json:"response_code,omitempty"`
}

// SilentVerify performs Number Verification via Vonage's CAMARA-compatible API.
func (a *VonageAdapter) SilentVerify(ctx context.Context, phoneNumber, countryCode string) (*model.SilentVerifyResponse, error) {
	token, err := a.generateJWT()
	if err != nil {
		return nil, fmt.Errorf("vonage jwt: %w", err)
	}

	reqBody := vonageNumberVerifyRequest{PhoneNumber: phoneNumber}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		a.config.BaseURL+"/camara/number-verification/v031/verify", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	start := time.Now()
	resp, err := a.httpClient.Do(req)
	latency := time.Since(start)
	if err != nil {
		return &model.SilentVerifyResponse{
			Status:      model.ResultFallbackRequired,
			TelcoSignal: fmt.Sprintf("vonage_error: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &model.SilentVerifyResponse{
			Status:      model.ResultFallbackRequired,
			TelcoSignal: fmt.Sprintf("vonage_http_%d", resp.StatusCode),
		}, nil
	}

	var result vonageNumberVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return &model.SilentVerifyResponse{
			Status:      model.ResultFallbackRequired,
			TelcoSignal: "vonage_decode_error",
		}, nil
	}

	if result.DevicePhoneNumberVerified {
		return &model.SilentVerifyResponse{
			Status:          model.ResultVerified,
			ConfidenceScore: 1.0,
			TelcoSignal:     fmt.Sprintf("vonage_verified_%dms", latency.Milliseconds()),
			Token:           "vng_" + uuid.New().String(),
		}, nil
	}

	return &model.SilentVerifyResponse{
		Status:      model.ResultFallbackRequired,
		TelcoSignal: "vonage_not_verified",
	}, nil
}

// Vonage SIM Swap types
type vonageSIMSwapRequest struct {
	PhoneNumber string `json:"phoneNumber"`
	MaxAge      int    `json:"maxAge,omitempty"`
}

type vonageSIMSwapResponse struct {
	Swapped bool `json:"swapped"`
}

// CheckSIMSwap queries Vonage's SIM Swap API.
func (a *VonageAdapter) CheckSIMSwap(ctx context.Context, phoneNumber, countryCode string) (*model.SIMSwapResponse, error) {
	token, err := a.generateJWT()
	if err != nil {
		return nil, fmt.Errorf("vonage jwt: %w", err)
	}

	reqBody := vonageSIMSwapRequest{
		PhoneNumber: phoneNumber,
		MaxAge:      72,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		a.config.BaseURL+"/camara/sim-swap/v040/check", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vonage sim swap request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vonage sim swap: status %d", resp.StatusCode)
	}

	var result vonageSIMSwapResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	riskLevel := model.RiskLow
	recommendation := model.VerdictAllow
	if result.Swapped {
		riskLevel = model.RiskHigh
		recommendation = model.VerdictBlock
	}

	return &model.SIMSwapResponse{
		SIMSwapDetected: result.Swapped,
		RiskLevel:       riskLevel,
		Recommendation:  recommendation,
	}, nil
}
