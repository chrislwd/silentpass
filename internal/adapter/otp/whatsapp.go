package otp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// WhatsAppConfig holds WhatsApp Business API credentials.
type WhatsAppConfig struct {
	BaseURL        string // e.g. "https://graph.facebook.com/v19.0"
	PhoneNumberID  string // WhatsApp Business phone number ID
	AccessToken    string // Permanent or long-lived access token
	TemplateNameOTP string // Pre-approved OTP template name
}

// WhatsAppProvider sends OTP via WhatsApp Business API using
// pre-approved authentication message templates.
type WhatsAppProvider struct {
	config     WhatsAppConfig
	httpClient *http.Client
	mu         sync.RWMutex
	codes      map[string]codeEntry // phoneNumber -> {code, expiry}
}

type codeEntry struct {
	code      string
	expiresAt time.Time
}

func NewWhatsAppProvider(cfg WhatsAppConfig) *WhatsAppProvider {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://graph.facebook.com/v19.0"
	}
	if cfg.TemplateNameOTP == "" {
		cfg.TemplateNameOTP = "silentpass_otp"
	}
	return &WhatsAppProvider{
		config:     cfg,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		codes:      make(map[string]codeEntry),
	}
}

func (p *WhatsAppProvider) Name() string { return "whatsapp_business" }

func (p *WhatsAppProvider) SupportedChannels() []string {
	return []string{"whatsapp"}
}

// WhatsApp Cloud API message structure
type waMessage struct {
	MessagingProduct string       `json:"messaging_product"`
	To               string       `json:"to"`
	Type             string       `json:"type"`
	Template         *waTemplate  `json:"template,omitempty"`
}

type waTemplate struct {
	Name       string            `json:"name"`
	Language   *waLanguage       `json:"language"`
	Components []waComponent     `json:"components,omitempty"`
}

type waLanguage struct {
	Code string `json:"code"`
}

type waComponent struct {
	Type       string        `json:"type"`
	Parameters []waParameter `json:"parameters,omitempty"`
}

type waParameter struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// Send generates an OTP code and sends it via WhatsApp Business API
// using a pre-approved authentication template.
func (p *WhatsAppProvider) Send(ctx context.Context, phoneNumber, channel, locale string) error {
	code := fmt.Sprintf("%06d", rand.Intn(1000000))

	// Store code for verification
	p.mu.Lock()
	p.codes[phoneNumber] = codeEntry{
		code:      code,
		expiresAt: time.Now().Add(10 * time.Minute),
	}
	p.mu.Unlock()

	lang := "en"
	if locale != "" {
		lang = locale
	}

	msg := waMessage{
		MessagingProduct: "whatsapp",
		To:               phoneNumber,
		Type:             "template",
		Template: &waTemplate{
			Name:     p.config.TemplateNameOTP,
			Language: &waLanguage{Code: lang},
			Components: []waComponent{
				{
					Type: "body",
					Parameters: []waParameter{
						{Type: "text", Text: code},
					},
				},
				{
					Type: "button",
					Parameters: []waParameter{
						{Type: "text", Text: code},
					},
				},
			},
		},
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("whatsapp marshal: %w", err)
	}

	endpoint := fmt.Sprintf("%s/%s/messages", p.config.BaseURL, p.config.PhoneNumberID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("whatsapp create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.AccessToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("whatsapp send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("whatsapp send failed: status %d, %v", resp.StatusCode, errResp)
	}

	return nil
}

// Verify checks the OTP code against the stored code.
func (p *WhatsAppProvider) Verify(ctx context.Context, phoneNumber, code string) (bool, error) {
	p.mu.RLock()
	entry, ok := p.codes[phoneNumber]
	p.mu.RUnlock()

	if !ok {
		return false, nil
	}

	if time.Now().After(entry.expiresAt) {
		p.mu.Lock()
		delete(p.codes, phoneNumber)
		p.mu.Unlock()
		return false, nil
	}

	if entry.code == code {
		p.mu.Lock()
		delete(p.codes, phoneNumber)
		p.mu.Unlock()
		return true, nil
	}

	return false, nil
}
