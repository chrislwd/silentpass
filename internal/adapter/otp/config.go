package otp

import (
	"encoding/json"
	"fmt"
	"os"
)

// OTPProvidersConfig defines all OTP provider configurations.
type OTPProvidersConfig struct {
	Providers []OTPProviderEntry `json:"providers"`
}

type OTPProviderEntry struct {
	Type          string   `json:"type"`           // "sandbox", "twilio", "vonage", "whatsapp"
	Name          string   `json:"name"`
	Channels      []string `json:"channels"`       // which channels this provider handles
	Enabled       bool     `json:"enabled"`
	// Twilio
	AccountSID    string   `json:"account_sid"`
	AuthToken     string   `json:"auth_token"`
	ServiceSID    string   `json:"service_sid"`
	// Vonage
	APIKey        string   `json:"api_key"`
	APISecret     string   `json:"api_secret"`
	Brand         string   `json:"brand"`
	// WhatsApp
	PhoneNumberID string   `json:"phone_number_id"`
	AccessToken   string   `json:"access_token"`
	TemplateName  string   `json:"template_name"`
	// Common
	BaseURL       string   `json:"base_url"`
}

// LoadOTPProvidersConfig reads OTP provider configuration from a JSON file.
func LoadOTPProvidersConfig(path string) (*OTPProvidersConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read OTP config: %w", err)
	}
	var cfg OTPProvidersConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse OTP config: %w", err)
	}
	return &cfg, nil
}

// BuildOTPProviders creates provider instances and registers them with the router.
func BuildOTPProviders(cfg *OTPProvidersConfig, router *Router) error {
	for _, entry := range cfg.Providers {
		if !entry.Enabled {
			continue
		}
		provider, err := buildOTPProvider(entry)
		if err != nil {
			return fmt.Errorf("build OTP provider %q: %w", entry.Name, err)
		}
		for _, ch := range entry.Channels {
			router.Register(ch, provider)
		}
	}
	return nil
}

func buildOTPProvider(entry OTPProviderEntry) (Provider, error) {
	switch entry.Type {
	case "sandbox":
		return NewSandboxProvider(), nil

	case "twilio":
		return NewTwilioProvider(TwilioConfig{
			AccountSID: entry.AccountSID,
			AuthToken:  entry.AuthToken,
			ServiceSID: entry.ServiceSID,
			BaseURL:    entry.BaseURL,
		}), nil

	case "vonage":
		return NewVonageOTPProvider(VonageOTPConfig{
			APIKey:    entry.APIKey,
			APISecret: entry.APISecret,
			Brand:     entry.Brand,
			BaseURL:   entry.BaseURL,
		}), nil

	case "whatsapp":
		return NewWhatsAppProvider(WhatsAppConfig{
			BaseURL:         entry.BaseURL,
			PhoneNumberID:   entry.PhoneNumberID,
			AccessToken:     entry.AccessToken,
			TemplateNameOTP: entry.TemplateName,
		}), nil

	default:
		return nil, fmt.Errorf("unknown OTP provider type: %s", entry.Type)
	}
}
