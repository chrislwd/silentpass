package otp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// VonageOTPConfig holds Vonage Verify v2 API credentials.
type VonageOTPConfig struct {
	APIKey    string
	APISecret string
	Brand     string // Brand name shown in OTP message
	BaseURL   string // Override for testing
}

// VonageOTPProvider implements OTP via Vonage Verify v2 API.
type VonageOTPProvider struct {
	config     VonageOTPConfig
	httpClient *http.Client
}

func NewVonageOTPProvider(cfg VonageOTPConfig) *VonageOTPProvider {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.nexmo.com"
	}
	if cfg.Brand == "" {
		cfg.Brand = "SilentPass"
	}
	return &VonageOTPProvider{
		config:     cfg,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *VonageOTPProvider) Name() string { return "vonage_otp" }

func (p *VonageOTPProvider) SupportedChannels() []string {
	return []string{"sms", "whatsapp", "voice"}
}

type vonageVerifyRequest struct {
	Brand    string               `json:"brand"`
	Workflow []vonageWorkflowStep `json:"workflow"`
}

type vonageWorkflowStep struct {
	Channel string `json:"channel"`
	To      string `json:"to"`
}

// Send creates a verification request via Vonage Verify v2.
func (p *VonageOTPProvider) Send(ctx context.Context, phoneNumber, channel, locale string) error {
	vonageChannel := channel
	if channel == "voice" {
		vonageChannel = "voice"
	} else if channel == "whatsapp" {
		vonageChannel = "whatsapp"
	} else {
		vonageChannel = "sms"
	}

	reqBody := vonageVerifyRequest{
		Brand: p.config.Brand,
		Workflow: []vonageWorkflowStep{
			{Channel: vonageChannel, To: phoneNumber},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("vonage marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.config.BaseURL+"/v2/verify", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("vonage create request: %w", err)
	}

	credentials := base64.StdEncoding.EncodeToString([]byte(p.config.APIKey + ":" + p.config.APISecret))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+credentials)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("vonage send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("vonage send failed: status %d", resp.StatusCode)
	}

	return nil
}

// Verify checks a code via Vonage Verify v2.
// Note: Vonage Verify v2 requires the request_id from Send.
// In a full implementation, the request_id is stored in the session
// and passed here. This returns false to delegate to other providers.
func (p *VonageOTPProvider) Verify(ctx context.Context, phoneNumber, code string) (bool, error) {
	return false, nil
}
