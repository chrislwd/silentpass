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

// CAMARAConfig holds CAMARA / Open Gateway API credentials.
// This adapter works with any GSMA Open Gateway aggregator or direct
// MNO integration that exposes CAMARA-standard APIs.
type CAMARAConfig struct {
	BaseURL      string   // Aggregator or MNO gateway base URL
	TokenURL     string   // OAuth2 token endpoint
	ClientID     string
	ClientSecret string
	Countries    []string
	ProviderName string   // e.g. "telefonica", "orange", "dt", "singtel"
}

// CAMARAAdapter implements CAMARA-standard Number Verification and SIM Swap APIs.
// CAMARA (an open-source project within the Linux Foundation) defines standardized
// telco APIs adopted by GSMA Open Gateway. This adapter works with any provider
// exposing the standard CAMARA API surface.
//
// Supported CAMARA APIs:
//   - number-verification v0.3: POST /number-verification/v0/verify
//   - sim-swap v0.4: POST /sim-swap/v0/check
//   - sim-swap v0.4: POST /sim-swap/v0/retrieve-date
type CAMARAAdapter struct {
	config     CAMARAConfig
	oauth      *OAuth2Client
	httpClient *http.Client
}

func NewCAMARAAdapter(cfg CAMARAConfig) *CAMARAAdapter {
	return &CAMARAAdapter{
		config: cfg,
		oauth: NewOAuth2Client(
			cfg.TokenURL, cfg.ClientID, cfg.ClientSecret,
			[]string{"number-verification:verify", "sim-swap:check"},
		),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (a *CAMARAAdapter) Name() string {
	if a.config.ProviderName != "" {
		return "camara_" + a.config.ProviderName
	}
	return "camara"
}

func (a *CAMARAAdapter) SupportedCountries() []string {
	return a.config.Countries
}

func (a *CAMARAAdapter) SupportedCapabilities() []string {
	return []string{"silent_verify", "sim_swap"}
}

// CAMARA Number Verification types (CAMARA v0.3 standard)
type camaraNumberVerifyRequest struct {
	PhoneNumber string `json:"phoneNumber"` // E.164 format
}

type camaraNumberVerifyResponse struct {
	DevicePhoneNumberVerified bool `json:"devicePhoneNumberVerified"`
}

// SilentVerify performs CAMARA-standard Number Verification.
//
// The CAMARA Number Verification API verifies whether the device originating
// the API call is using a specific phone number, by leveraging the network
// authentication already performed by the mobile operator. The verification
// is performed server-side by the MNO without any user interaction.
//
// Prerequisites:
//   - The device must be connected via mobile data (not Wi-Fi)
//   - The phone number must belong to the network operator
//   - Frontend SDK must have initiated the OIDC/CIBA flow to obtain
//     the necessary authorization context
func (a *CAMARAAdapter) SilentVerify(ctx context.Context, phoneNumber, countryCode string) (*model.SilentVerifyResponse, error) {
	token, err := a.oauth.GetToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("camara auth: %w", err)
	}

	reqBody := camaraNumberVerifyRequest{PhoneNumber: phoneNumber}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
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
			TelcoSignal: fmt.Sprintf("camara_error: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusServiceUnavailable {
		// Operator may not support this number or capability
		return &model.SilentVerifyResponse{
			Status:      model.ResultFallbackRequired,
			TelcoSignal: fmt.Sprintf("camara_unsupported_%d", resp.StatusCode),
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return &model.SilentVerifyResponse{
			Status:      model.ResultFallbackRequired,
			TelcoSignal: fmt.Sprintf("camara_http_%d", resp.StatusCode),
		}, nil
	}

	var result camaraNumberVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return &model.SilentVerifyResponse{
			Status:      model.ResultFallbackRequired,
			TelcoSignal: "camara_decode_error",
		}, nil
	}

	if result.DevicePhoneNumberVerified {
		return &model.SilentVerifyResponse{
			Status:          model.ResultVerified,
			ConfidenceScore: 1.0,
			TelcoSignal:     fmt.Sprintf("camara_%s_verified_%dms", a.config.ProviderName, latency.Milliseconds()),
			Token:           "cmr_" + uuid.New().String(),
		}, nil
	}

	return &model.SilentVerifyResponse{
		Status:      model.ResultFallbackRequired,
		TelcoSignal: fmt.Sprintf("camara_%s_not_verified", a.config.ProviderName),
	}, nil
}

// CAMARA SIM Swap types (CAMARA v0.4 standard)
type camaraSIMSwapCheckRequest struct {
	PhoneNumber string `json:"phoneNumber"`
	MaxAge      int    `json:"maxAge,omitempty"` // hours, default 240
}

type camaraSIMSwapCheckResponse struct {
	Swapped bool `json:"swapped"`
}

type camaraSIMSwapDateRequest struct {
	PhoneNumber string `json:"phoneNumber"`
}

type camaraSIMSwapDateResponse struct {
	LatestSimChange string `json:"latestSimChange"` // RFC3339
}

// CheckSIMSwap queries the CAMARA-standard SIM Swap API.
//
// The CAMARA SIM Swap API provides two operations:
//   - /check: Returns whether a SIM swap has occurred within maxAge hours
//   - /retrieve-date: Returns the actual date of the latest SIM change
//
// This method calls /check first, and if swapped, calls /retrieve-date
// to get the exact timestamp.
func (a *CAMARAAdapter) CheckSIMSwap(ctx context.Context, phoneNumber, countryCode string) (*model.SIMSwapResponse, error) {
	token, err := a.oauth.GetToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("camara auth: %w", err)
	}

	// Step 1: Check if swapped
	checkBody, _ := json.Marshal(camaraSIMSwapCheckRequest{
		PhoneNumber: phoneNumber,
		MaxAge:      72,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		a.config.BaseURL+"/sim-swap/v0/check", bytes.NewReader(checkBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("camara sim swap check: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("camara sim swap check: status %d", resp.StatusCode)
	}

	var checkResult camaraSIMSwapCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&checkResult); err != nil {
		return nil, fmt.Errorf("decode check response: %w", err)
	}

	result := &model.SIMSwapResponse{
		SIMSwapDetected: checkResult.Swapped,
		RiskLevel:       model.RiskLow,
		Recommendation:  model.VerdictAllow,
	}

	if checkResult.Swapped {
		result.RiskLevel = model.RiskHigh
		result.Recommendation = model.VerdictBlock

		// Step 2: Get exact date
		dateBody, _ := json.Marshal(camaraSIMSwapDateRequest{PhoneNumber: phoneNumber})
		dateReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
			a.config.BaseURL+"/sim-swap/v0/retrieve-date", bytes.NewReader(dateBody))
		if err == nil {
			dateReq.Header.Set("Content-Type", "application/json")
			dateReq.Header.Set("Authorization", "Bearer "+token)

			dateResp, err := a.httpClient.Do(dateReq)
			if err == nil && dateResp.StatusCode == http.StatusOK {
				var dateResult camaraSIMSwapDateResponse
				if json.NewDecoder(dateResp.Body).Decode(&dateResult) == nil {
					result.LastChangeTime = dateResult.LatestSimChange
				}
				dateResp.Body.Close()
			}
		}
	}

	return result, nil
}
