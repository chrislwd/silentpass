package otp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTwilioProvider_Send(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}

		user, pass, ok := r.BasicAuth()
		if !ok || user != "AC_test" || pass != "token_test" {
			t.Fatalf("bad auth: %s/%s", user, pass)
		}

		r.ParseForm()
		if r.FormValue("To") != "+628123" {
			t.Fatalf("expected +628123, got %s", r.FormValue("To"))
		}
		if r.FormValue("Channel") != "sms" {
			t.Fatalf("expected sms channel, got %s", r.FormValue("Channel"))
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"sid": "VE123", "status": "pending"})
	}))
	defer server.Close()

	p := NewTwilioProvider(TwilioConfig{
		AccountSID: "AC_test", AuthToken: "token_test",
		ServiceSID: "VA_test", BaseURL: server.URL,
	})

	err := p.Send(context.Background(), "+628123", "sms", "")
	if err != nil {
		t.Fatalf("send: %v", err)
	}
}

func TestTwilioProvider_Verify(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		code := r.FormValue("Code")
		json.NewEncoder(w).Encode(twilioVerificationResponse{
			Status: "approved", Valid: code == "123456",
		})
	}))
	defer server.Close()

	p := NewTwilioProvider(TwilioConfig{
		AccountSID: "AC", AuthToken: "tok",
		ServiceSID: "VA", BaseURL: server.URL,
	})

	ok, err := p.Verify(context.Background(), "+628123", "123456")
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !ok {
		t.Fatal("expected verified")
	}
}

func TestVonageOTPProvider_Send(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}

		var req vonageVerifyRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Brand != "TestBrand" {
			t.Fatalf("expected TestBrand, got %s", req.Brand)
		}
		if len(req.Workflow) == 0 || req.Workflow[0].Channel != "sms" {
			t.Fatal("expected sms workflow")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"request_id": "req123"})
	}))
	defer server.Close()

	p := NewVonageOTPProvider(VonageOTPConfig{
		APIKey: "key", APISecret: "secret",
		Brand: "TestBrand", BaseURL: server.URL,
	})

	err := p.Send(context.Background(), "+628123", "sms", "")
	if err != nil {
		t.Fatalf("send: %v", err)
	}
}

func TestWhatsAppProvider_SendAndVerify(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatal("bad auth")
		}
		var msg waMessage
		json.NewDecoder(r.Body).Decode(&msg)
		if msg.MessagingProduct != "whatsapp" {
			t.Fatalf("expected whatsapp, got %s", msg.MessagingProduct)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"messages": "sent"})
	}))
	defer server.Close()

	p := NewWhatsAppProvider(WhatsAppConfig{
		BaseURL:       server.URL,
		PhoneNumberID: "12345",
		AccessToken:   "test-token",
	})

	err := p.Send(context.Background(), "+628123", "whatsapp", "en")
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	// Get the stored code
	p.mu.RLock()
	entry, ok := p.codes["+628123"]
	p.mu.RUnlock()
	if !ok {
		t.Fatal("code not stored")
	}

	// Verify with correct code
	verified, err := p.Verify(context.Background(), "+628123", entry.code)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !verified {
		t.Fatal("expected verified")
	}

	// Should be consumed
	verified2, _ := p.Verify(context.Background(), "+628123", entry.code)
	if verified2 {
		t.Fatal("code should be consumed after verification")
	}
}

func TestWhatsAppProvider_VerifyWrongCode(t *testing.T) {
	p := NewWhatsAppProvider(WhatsAppConfig{BaseURL: "http://unused"})
	p.mu.Lock()
	p.codes["+628123"] = codeEntry{code: "123456", expiresAt: addMinutes(10)}
	p.mu.Unlock()

	ok, _ := p.Verify(context.Background(), "+628123", "999999")
	if ok {
		t.Fatal("wrong code should not verify")
	}
}

func TestRouter_FallbackToSMS(t *testing.T) {
	r := NewRouter()
	sandbox := NewSandboxProvider()
	r.Register("sms", sandbox)

	// Request voice channel, should fallback to SMS
	err := r.Send(context.Background(), "+628123456", "voice", "")
	if err != nil {
		t.Fatalf("fallback send: %v", err)
	}
}

func TestRouter_NoProvider(t *testing.T) {
	r := NewRouter()
	err := r.Send(context.Background(), "+628123456", "sms", "")
	if err != ErrNoProvider {
		t.Fatalf("expected ErrNoProvider, got %v", err)
	}
}

func TestBuildOTPProviders_Config(t *testing.T) {
	cfg := &OTPProvidersConfig{
		Providers: []OTPProviderEntry{
			{Type: "sandbox", Name: "Sandbox", Channels: []string{"sms", "voice"}, Enabled: true},
			{Type: "sandbox", Name: "Disabled", Channels: []string{"whatsapp"}, Enabled: false},
		},
	}

	router := NewRouter()
	err := BuildOTPProviders(cfg, router)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	// SMS should work (sandbox registered)
	err = router.Send(context.Background(), "+62812345678", "sms", "")
	if err != nil {
		t.Fatalf("sms send: %v", err)
	}

	// WhatsApp should fallback to SMS since disabled entry was skipped
	err = router.Send(context.Background(), "+62812345678", "whatsapp", "")
	if err != nil {
		t.Fatalf("whatsapp fallback: %v", err)
	}
}

func addMinutes(n int) time.Time {
	return time.Now().Add(time.Duration(n) * time.Minute)
}
