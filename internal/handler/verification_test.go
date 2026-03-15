package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/silentpass/silentpass/internal/adapter/otp"
	"github.com/silentpass/silentpass/internal/adapter/telco"
	"github.com/silentpass/silentpass/internal/middleware"
	"github.com/silentpass/silentpass/internal/repository"
	"github.com/silentpass/silentpass/internal/service/verification"
)

func setupRouter() (*gin.Engine, *repository.TenantRepo) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	sessionRepo := repository.NewSessionRepo()
	tenantRepo := repository.NewTenantRepo()

	telcoRouter := telco.NewRouter()
	telcoRouter.Register(telco.NewSandboxAdapter())

	otpRouter := otp.NewRouter()
	otpProvider := otp.NewSandboxProvider()
	otpRouter.Register("sms", otpProvider)

	svc := verification.NewService(sessionRepo, telcoRouter, otpRouter)
	h := NewVerificationHandler(svc)

	v1 := r.Group("/v1")
	v1.Use(middleware.APIKeyAuth(tenantRepo))

	vg := v1.Group("/verification")
	vg.POST("/session", h.CreateSession)
	vg.POST("/silent", h.SilentVerify)
	vg.POST("/otp/send", h.SendOTP)
	vg.POST("/otp/check", h.CheckOTP)

	return r, tenantRepo
}

func doJSON(r *gin.Engine, method, path string, body interface{}, apiKey string) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCreateSession_Integration(t *testing.T) {
	r, _ := setupRouter()

	w := doJSON(r, "POST", "/v1/verification/session", map[string]string{
		"app_id":            "test",
		"phone_number":      "+6281234567890",
		"country_code":      "ID",
		"verification_type": "silent_or_otp",
		"use_case":          "signup",
	}, "sk_test_sandbox_key_001")

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["session_id"] == nil || resp["session_id"] == "" {
		t.Fatal("session_id missing")
	}
	if resp["recommended_action"] != "silent_verify" {
		t.Fatalf("expected silent_verify, got %v", resp["recommended_action"])
	}
}

func TestCreateSession_Unauthorized(t *testing.T) {
	r, _ := setupRouter()

	w := doJSON(r, "POST", "/v1/verification/session", map[string]string{
		"app_id":            "test",
		"phone_number":      "+6281234567890",
		"country_code":      "ID",
		"verification_type": "silent",
		"use_case":          "signup",
	}, "")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestCreateSession_BadAPIKey(t *testing.T) {
	r, _ := setupRouter()

	w := doJSON(r, "POST", "/v1/verification/session", map[string]string{
		"app_id":            "test",
		"phone_number":      "+6281234567890",
		"country_code":      "ID",
		"verification_type": "silent",
		"use_case":          "signup",
	}, "sk_invalid_key")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestCreateSession_BadRequest(t *testing.T) {
	r, _ := setupRouter()

	// Missing required fields
	w := doJSON(r, "POST", "/v1/verification/session", map[string]string{
		"app_id": "test",
	}, "sk_test_sandbox_key_001")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFullVerificationFlow_Integration(t *testing.T) {
	r, _ := setupRouter()

	// Step 1: Create session
	w := doJSON(r, "POST", "/v1/verification/session", map[string]string{
		"app_id":            "test",
		"phone_number":      "+6281234567890",
		"country_code":      "ID",
		"verification_type": "silent_or_otp",
		"use_case":          "signup",
	}, "sk_test_sandbox_key_001")

	if w.Code != http.StatusCreated {
		t.Fatalf("create session: expected 201, got %d", w.Code)
	}

	var sessionResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &sessionResp)
	sessionID := sessionResp["session_id"].(string)

	// Step 2: Silent verify
	w = doJSON(r, "POST", "/v1/verification/silent", map[string]string{
		"session_id": sessionID,
	}, "sk_test_sandbox_key_001")

	if w.Code != http.StatusOK {
		t.Fatalf("silent verify: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var silentResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &silentResp)
	status := silentResp["status"].(string)

	// Sandbox has ~85% success rate, so either verified or fallback is OK
	if status != "verified" && status != "fallback_required" {
		t.Fatalf("unexpected silent status: %s", status)
	}

	// Step 3: If fallback, send OTP
	if status == "fallback_required" {
		w = doJSON(r, "POST", "/v1/verification/otp/send", map[string]string{
			"session_id": sessionID,
			"channel":    "sms",
		}, "sk_test_sandbox_key_001")

		if w.Code != http.StatusOK {
			t.Fatalf("otp send: expected 200, got %d", w.Code)
		}

		// Step 4: Check OTP with sandbox universal code
		w = doJSON(r, "POST", "/v1/verification/otp/check", map[string]string{
			"session_id": sessionID,
			"code":       "000000",
		}, "sk_test_sandbox_key_001")

		if w.Code != http.StatusOK {
			t.Fatalf("otp check: expected 200, got %d", w.Code)
		}

		var otpResp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &otpResp)
		if otpResp["status"] != "verified" {
			t.Fatalf("expected verified after OTP, got %s", otpResp["status"])
		}
	}
}

func TestSilentVerify_WrongTenant(t *testing.T) {
	r, tenantRepo := setupRouter()
	tenantRepo.Seed("sk_test_other_key", "other_secret", "Other Tenant")

	// Create session with tenant A
	w := doJSON(r, "POST", "/v1/verification/session", map[string]string{
		"app_id":            "test",
		"phone_number":      "+6281234567890",
		"country_code":      "ID",
		"verification_type": "silent",
		"use_case":          "signup",
	}, "sk_test_sandbox_key_001")

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	sessionID := resp["session_id"].(string)

	// Try to verify with tenant B
	w = doJSON(r, "POST", "/v1/verification/silent", map[string]string{
		"session_id": sessionID,
	}, "sk_test_other_key")

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for wrong tenant, got %d", w.Code)
	}
}

// Ensure sandbox adapter introduces realistic latency
func TestSilentVerify_Latency(t *testing.T) {
	r, _ := setupRouter()

	w := doJSON(r, "POST", "/v1/verification/session", map[string]string{
		"app_id":            "test",
		"phone_number":      "+6281234567890",
		"country_code":      "ID",
		"verification_type": "silent",
		"use_case":          "login",
	}, "sk_test_sandbox_key_001")

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	start := time.Now()
	doJSON(r, "POST", "/v1/verification/silent", map[string]string{
		"session_id": resp["session_id"].(string),
	}, "sk_test_sandbox_key_001")
	elapsed := time.Since(start)

	// Sandbox simulates 200-500ms latency
	if elapsed < 150*time.Millisecond {
		t.Fatalf("silent verify too fast (%v), sandbox should simulate latency", elapsed)
	}
}
