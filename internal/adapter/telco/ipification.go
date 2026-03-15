package telco

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/silentpass/silentpass/internal/model"
)

// IPificationConfig holds ipification API credentials and settings.
type IPificationConfig struct {
	BaseURL      string   // e.g. "https://api.ipification.com"
	TokenURL     string   // e.g. "https://auth.ipification.com/oauth2/token"
	ClientID     string
	ClientSecret string
	Countries    []string // ISO country codes this instance covers
}

// IPificationAdapter implements the Adapter interface for ipification.
// ipification provides Number Verification and Phone Verify (silent auth)
// through direct mobile network integration and as a GSMA Open Gateway channel partner.
type IPificationAdapter struct {
	config     IPificationConfig
	oauth      *OAuth2Client
	httpClient *http.Client
}

func NewIPificationAdapter(cfg IPificationConfig) *IPificationAdapter {
	return &IPificationAdapter{
		config: cfg,
		oauth: NewOAuth2Client(
			cfg.TokenURL, cfg.ClientID, cfg.ClientSecret,
			[]string{"openid", "phone_verify", "number_verification"},
		),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (a *IPificationAdapter) Name() string {
	return "ipification"
}

func (a *IPificationAdapter) SupportedCountries() []string {
	return a.config.Countries
}

func (a *IPificationAdapter) SupportedCapabilities() []string {
	return []string{"silent_verify", "sim_swap"}
}

// ipification Number Verification request/response types
type ipifNumberVerifyRequest struct {
	PhoneNumber string `json:"phoneNumber"`
}

type ipifNumberVerifyResponse struct {
	DevicePhoneNumberVerified bool   `json:"devicePhoneNumberVerified"`
	ResponseCode              int    `json:"response_code,omitempty"`
	Message                   string `json:"message,omitempty"`
}

// SilentVerify performs Number Verification via ipification's CAMARA-compatible API.
// Flow: Server-side call → ipification routes to MNO → MNO checks if device IP
// matches the phone number's network context → returns verified/not verified.
func (a *IPificationAdapter) SilentVerify(ctx context.Context, phoneNumber, countryCode string) (*model.SilentVerifyResponse, error) {
	token, err := a.oauth.GetToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("ipification auth: %w", err)
	}

	reqBody := ipifNumberVerifyRequest{
		PhoneNumber: phoneNumber,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		a.config.BaseURL+"/number-verification/v0/verify", bytes.NewReader(body))
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
			TelcoSignal: fmt.Sprintf("ipification_error: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &model.SilentVerifyResponse{
			Status:      model.ResultFallbackRequired,
			TelcoSignal: fmt.Sprintf("ipification_http_%d", resp.StatusCode),
		}, nil
	}

	var result ipifNumberVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return &model.SilentVerifyResponse{
			Status:      model.ResultFallbackRequired,
			TelcoSignal: "ipification_decode_error",
		}, nil
	}

	if result.DevicePhoneNumberVerified {
		return &model.SilentVerifyResponse{
			Status:          model.ResultVerified,
			ConfidenceScore: 1.0,
			TelcoSignal:     fmt.Sprintf("ipification_verified_%dms", latency.Milliseconds()),
			Token:           "ipif_" + uuid.New().String(),
		}, nil
	}

	return &model.SilentVerifyResponse{
		Status:      model.ResultFallbackRequired,
		TelcoSignal: "ipification_not_verified",
	}, nil
}

// ipification SIM Swap types
type ipifSIMSwapRequest struct {
	PhoneNumber string `json:"phoneNumber"`
	MaxAge      int    `json:"maxAge,omitempty"` // hours
}

type ipifSIMSwapResponse struct {
	Swapped    bool   `json:"swapped"`
	SwapDate   string `json:"swapDate,omitempty"`
	LastSwapAt string `json:"latestSimChange,omitempty"`
}

// CheckSIMSwap queries ipification's SIM Swap API.
func (a *IPificationAdapter) CheckSIMSwap(ctx context.Context, phoneNumber, countryCode string) (*model.SIMSwapResponse, error) {
	token, err := a.oauth.GetToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("ipification auth: %w", err)
	}

	reqBody := ipifSIMSwapRequest{
		PhoneNumber: phoneNumber,
		MaxAge:      72, // check last 72 hours
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		a.config.BaseURL+"/sim-swap/v0/check", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ipification sim swap request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ipification sim swap: status %d", resp.StatusCode)
	}

	var result ipifSIMSwapResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode sim swap response: %w", err)
	}

	riskLevel := model.RiskLow
	recommendation := model.VerdictAllow
	if result.Swapped {
		riskLevel = model.RiskHigh
		recommendation = model.VerdictBlock
	}

	lastChange := result.SwapDate
	if lastChange == "" {
		lastChange = result.LastSwapAt
	}

	return &model.SIMSwapResponse{
		SIMSwapDetected: result.Swapped,
		LastChangeTime:  lastChange,
		RiskLevel:       riskLevel,
		Recommendation:  recommendation,
	}, nil
}
