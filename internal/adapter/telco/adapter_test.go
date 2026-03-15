package telco

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIPificationAdapter_SilentVerify_Success(t *testing.T) {
	// Mock ipification API
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer tokenServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(401)
			return
		}
		var req ipifNumberVerifyRequest
		json.NewDecoder(r.Body).Decode(&req)
		json.NewEncoder(w).Encode(ipifNumberVerifyResponse{
			DevicePhoneNumberVerified: true,
		})
	}))
	defer apiServer.Close()

	adapter := NewIPificationAdapter(IPificationConfig{
		BaseURL:      apiServer.URL,
		TokenURL:     tokenServer.URL,
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		Countries:    []string{"ID"},
	})

	resp, err := adapter.SilentVerify(context.Background(), "+6281234567890", "ID")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "verified" {
		t.Fatalf("expected verified, got %s", resp.Status)
	}
	if resp.ConfidenceScore != 1.0 {
		t.Fatalf("expected confidence 1.0, got %f", resp.ConfidenceScore)
	}
}

func TestIPificationAdapter_SilentVerify_NotVerified(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "t", "expires_in": 3600})
	}))
	defer tokenServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ipifNumberVerifyResponse{DevicePhoneNumberVerified: false})
	}))
	defer apiServer.Close()

	adapter := NewIPificationAdapter(IPificationConfig{
		BaseURL: apiServer.URL, TokenURL: tokenServer.URL,
		ClientID: "id", ClientSecret: "secret", Countries: []string{"ID"},
	})

	resp, err := adapter.SilentVerify(context.Background(), "+6281234567890", "ID")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "fallback_required" {
		t.Fatalf("expected fallback_required, got %s", resp.Status)
	}
}

func TestIPificationAdapter_SIMSwap_Detected(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "t", "expires_in": 3600})
	}))
	defer tokenServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ipifSIMSwapResponse{Swapped: true, SwapDate: "2026-03-15T10:00:00Z"})
	}))
	defer apiServer.Close()

	adapter := NewIPificationAdapter(IPificationConfig{
		BaseURL: apiServer.URL, TokenURL: tokenServer.URL,
		ClientID: "id", ClientSecret: "secret", Countries: []string{"ID"},
	})

	resp, err := adapter.CheckSIMSwap(context.Background(), "+6281234567890", "ID")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.SIMSwapDetected {
		t.Fatal("expected SIM swap detected")
	}
	if resp.RiskLevel != "high" {
		t.Fatalf("expected high risk, got %s", resp.RiskLevel)
	}
}

func TestCAMARAAdapter_SilentVerify(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "camara-token", "expires_in": 3600})
	}))
	defer tokenServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/number-verification/v0/verify" {
			json.NewEncoder(w).Encode(camaraNumberVerifyResponse{DevicePhoneNumberVerified: true})
		}
	}))
	defer apiServer.Close()

	adapter := NewCAMARAAdapter(CAMARAConfig{
		BaseURL: apiServer.URL, TokenURL: tokenServer.URL,
		ClientID: "id", ClientSecret: "secret",
		Countries: []string{"ES"}, ProviderName: "telefonica",
	})

	if adapter.Name() != "camara_telefonica" {
		t.Fatalf("expected camara_telefonica, got %s", adapter.Name())
	}

	resp, err := adapter.SilentVerify(context.Background(), "+34612345678", "ES")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "verified" {
		t.Fatalf("expected verified, got %s", resp.Status)
	}
}

func TestCAMARAAdapter_SIMSwap(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "t", "expires_in": 3600})
	}))
	defer tokenServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sim-swap/v0/check":
			json.NewEncoder(w).Encode(camaraSIMSwapCheckResponse{Swapped: true})
		case "/sim-swap/v0/retrieve-date":
			json.NewEncoder(w).Encode(camaraSIMSwapDateResponse{LatestSimChange: "2026-03-15T08:00:00Z"})
		}
	}))
	defer apiServer.Close()

	adapter := NewCAMARAAdapter(CAMARAConfig{
		BaseURL: apiServer.URL, TokenURL: tokenServer.URL,
		ClientID: "id", ClientSecret: "secret",
		Countries: []string{"ES"}, ProviderName: "test",
	})

	resp, err := adapter.CheckSIMSwap(context.Background(), "+34612345678", "ES")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.SIMSwapDetected {
		t.Fatal("expected SIM swap detected")
	}
	if resp.LastChangeTime != "2026-03-15T08:00:00Z" {
		t.Fatalf("expected date, got %s", resp.LastChangeTime)
	}
}

func TestRouter_MultiAdapter(t *testing.T) {
	router := NewRouter()
	router.Register(NewSandboxAdapter())

	if !router.IsSupported("ID") {
		t.Fatal("ID should be supported")
	}
	if router.IsSupported("ZZ") {
		t.Fatal("ZZ should not be supported")
	}

	resp, err := router.SilentVerify(context.Background(), "+6281234567890", "ID")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "verified" && resp.Status != "fallback_required" {
		t.Fatalf("unexpected status: %s", resp.Status)
	}
}

func TestBuildAdapters_Config(t *testing.T) {
	cfg := &ProvidersConfig{
		Providers: []ProviderEntry{
			{Type: "sandbox", Name: "Sandbox", Countries: []string{"ID"}, Enabled: true},
			{Type: "sandbox", Name: "Disabled", Countries: []string{"US"}, Enabled: false},
			{Type: "ipification", Name: "IPif", BaseURL: "https://api.test.com", TokenURL: "https://auth.test.com/token",
				ClientID: "id", ClientSecret: "secret", Countries: []string{"TH"}, Enabled: true},
		},
	}

	router := NewRouter()
	err := BuildAdapters(cfg, router)
	if err != nil {
		t.Fatalf("build adapters: %v", err)
	}

	if !router.IsSupported("ID") {
		t.Fatal("ID should be supported (sandbox)")
	}
	if !router.IsSupported("TH") {
		t.Fatal("TH should be supported (ipification)")
	}
	if router.IsSupported("US") {
		t.Fatal("US should not be supported (disabled)")
	}
}
