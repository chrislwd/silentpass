package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/silentpass/silentpass/internal/adapter/telco"
	"github.com/silentpass/silentpass/internal/middleware"
	"github.com/silentpass/silentpass/internal/repository"
	"github.com/silentpass/silentpass/internal/service/policy"
	"github.com/silentpass/silentpass/internal/service/risk"
)

func setupRiskRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	tenantRepo := repository.NewTenantRepo()
	telcoRouter := telco.NewRouter()
	telcoRouter.Register(telco.NewSandboxAdapter())
	policyEngine := policy.NewEngine(nil)
	riskSvc := risk.NewService(telcoRouter, policyEngine)
	h := NewRiskHandler(riskSvc)

	v1 := r.Group("/v1")
	v1.Use(middleware.APIKeyAuth(tenantRepo))

	rg := v1.Group("/risk")
	rg.POST("/sim-swap", h.SIMSwap)
	rg.POST("/verdict", h.Verdict)

	return r
}

func TestSIMSwap_Integration(t *testing.T) {
	r := setupRiskRouter()

	w := doJSON(r, "POST", "/v1/risk/sim-swap", map[string]string{
		"phone_number": "+6281234567890",
		"country_code": "ID",
	}, "sk_test_sandbox_key_001")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["risk_level"] == nil {
		t.Fatal("risk_level missing")
	}
	if resp["recommendation"] == nil {
		t.Fatal("recommendation missing")
	}
}

func TestVerdict_VerifiedAllow(t *testing.T) {
	r := setupRiskRouter()

	w := doJSON(r, "POST", "/v1/risk/verdict", map[string]interface{}{
		"session_id":          "test-session",
		"verification_result": "verified",
	}, "sk_test_sandbox_key_001")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["verdict"] != "allow" {
		t.Fatalf("expected allow for verified result, got %v", resp["verdict"])
	}
}

func TestVerdict_FailedChallenge(t *testing.T) {
	r := setupRiskRouter()

	w := doJSON(r, "POST", "/v1/risk/verdict", map[string]interface{}{
		"session_id":          "test-session",
		"verification_result": "failed",
	}, "sk_test_sandbox_key_001")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["verdict"] != "challenge" {
		t.Fatalf("expected challenge for failed result, got %v", resp["verdict"])
	}
}

func TestVerdict_SIMSwapBlock(t *testing.T) {
	r := setupRiskRouter()

	w := doJSON(r, "POST", "/v1/risk/verdict", map[string]interface{}{
		"session_id":          "test-session",
		"verification_result": "verified",
		"sim_swap_result": map[string]interface{}{
			"sim_swap_detected": true,
			"risk_level":        "high",
			"recommendation":    "block",
		},
	}, "sk_test_sandbox_key_001")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["verdict"] != "block" {
		t.Fatalf("expected block for high-risk SIM swap, got %v", resp["verdict"])
	}
}

func TestSIMSwap_Unauthorized(t *testing.T) {
	r := setupRiskRouter()

	w := doJSON(r, "POST", "/v1/risk/sim-swap", map[string]string{
		"phone_number": "+6281234567890",
		"country_code": "ID",
	}, "")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestSIMSwap_BadRequest(t *testing.T) {
	r := setupRiskRouter()

	w := &httptest.ResponseRecorder{}
	w = doJSON(r, "POST", "/v1/risk/sim-swap", map[string]string{}, "sk_test_sandbox_key_001")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
