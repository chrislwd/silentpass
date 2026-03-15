package otp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TwilioConfig holds Twilio Verify API credentials.
type TwilioConfig struct {
	AccountSID string // Twilio Account SID
	AuthToken  string // Twilio Auth Token
	ServiceSID string // Twilio Verify Service SID
	BaseURL    string // Override for testing, defaults to api.twilio.com
}

// TwilioProvider implements OTP via Twilio Verify API.
// Supports SMS, WhatsApp, and Voice channels.
type TwilioProvider struct {
	config     TwilioConfig
	httpClient *http.Client
}

func NewTwilioProvider(cfg TwilioConfig) *TwilioProvider {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://verify.twilio.com"
	}
	return &TwilioProvider{
		config:     cfg,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *TwilioProvider) Name() string { return "twilio" }

func (p *TwilioProvider) SupportedChannels() []string {
	return []string{"sms", "whatsapp", "voice"}
}

// twilioVerificationResponse is the Twilio Verify API response.
type twilioVerificationResponse struct {
	SID     string `json:"sid"`
	Status  string `json:"status"` // pending, approved, canceled
	Channel string `json:"channel"`
	Valid   bool   `json:"valid"`
}

// Send creates a verification via Twilio Verify API.
func (p *TwilioProvider) Send(ctx context.Context, phoneNumber, channel, locale string) error {
	twilioChannel := channel
	if channel == "voice" {
		twilioChannel = "call"
	}

	data := url.Values{
		"To":      {phoneNumber},
		"Channel": {twilioChannel},
	}
	if locale != "" {
		data.Set("Locale", locale)
	}

	endpoint := fmt.Sprintf("%s/v2/Services/%s/Verifications", p.config.BaseURL, p.config.ServiceSID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("twilio create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(p.config.AccountSID, p.config.AuthToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("twilio send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("twilio send failed: status %d, %v", resp.StatusCode, errResp["message"])
	}

	return nil
}

// Verify checks a verification code via Twilio Verify API.
func (p *TwilioProvider) Verify(ctx context.Context, phoneNumber, code string) (bool, error) {
	data := url.Values{
		"To":   {phoneNumber},
		"Code": {code},
	}

	endpoint := fmt.Sprintf("%s/v2/Services/%s/VerificationCheck", p.config.BaseURL, p.config.ServiceSID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return false, fmt.Errorf("twilio create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(p.config.AccountSID, p.config.AuthToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("twilio verify: %w", err)
	}
	defer resp.Body.Close()

	var result twilioVerificationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("twilio decode: %w", err)
	}

	return result.Status == "approved" || result.Valid, nil
}
