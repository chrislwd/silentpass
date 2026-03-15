package telco

import (
	"encoding/json"
	"fmt"
	"os"
)

// ProvidersConfig defines all upstream telco providers.
type ProvidersConfig struct {
	Providers []ProviderEntry `json:"providers"`
}

type ProviderEntry struct {
	Type         string   `json:"type"`          // "ipification", "vonage", "camara", "sandbox"
	Name         string   `json:"name"`          // display name
	BaseURL      string   `json:"base_url"`
	TokenURL     string   `json:"token_url"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	AppID        string   `json:"app_id"`         // Vonage application ID
	PrivateKey   string   `json:"private_key"`    // path to PEM file or inline
	ProviderName string   `json:"provider_name"`  // CAMARA provider identifier
	Countries    []string `json:"countries"`
	Enabled      bool     `json:"enabled"`
}

// LoadProvidersConfig reads provider configuration from a JSON file.
func LoadProvidersConfig(path string) (*ProvidersConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read providers config: %w", err)
	}

	var cfg ProvidersConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse providers config: %w", err)
	}

	return &cfg, nil
}

// BuildAdapters creates adapter instances from config and registers them with the router.
func BuildAdapters(cfg *ProvidersConfig, router *Router) error {
	for _, entry := range cfg.Providers {
		if !entry.Enabled {
			continue
		}

		adapter, err := buildAdapter(entry)
		if err != nil {
			return fmt.Errorf("build adapter %q: %w", entry.Name, err)
		}

		router.Register(adapter)
	}
	return nil
}

func buildAdapter(entry ProviderEntry) (Adapter, error) {
	switch entry.Type {
	case "sandbox":
		return NewSandboxAdapter(), nil

	case "ipification":
		return NewIPificationAdapter(IPificationConfig{
			BaseURL:      entry.BaseURL,
			TokenURL:     entry.TokenURL,
			ClientID:     entry.ClientID,
			ClientSecret: entry.ClientSecret,
			Countries:    entry.Countries,
		}), nil

	case "vonage":
		privateKey := []byte(entry.PrivateKey)
		if _, err := os.Stat(entry.PrivateKey); err == nil {
			data, err := os.ReadFile(entry.PrivateKey)
			if err != nil {
				return nil, fmt.Errorf("read vonage private key: %w", err)
			}
			privateKey = data
		}
		return NewVonageAdapter(VonageConfig{
			BaseURL:       entry.BaseURL,
			ApplicationID: entry.AppID,
			PrivateKey:    privateKey,
			Countries:     entry.Countries,
		}), nil

	case "camara":
		return NewCAMARAAdapter(CAMARAConfig{
			BaseURL:      entry.BaseURL,
			TokenURL:     entry.TokenURL,
			ClientID:     entry.ClientID,
			ClientSecret: entry.ClientSecret,
			Countries:    entry.Countries,
			ProviderName: entry.ProviderName,
		}), nil

	default:
		return nil, fmt.Errorf("unknown adapter type: %s", entry.Type)
	}
}
