package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/silentpass/silentpass/internal/pkg/auth"
)

func setupAccountRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	tokenSvc := auth.NewTokenService("test-secret-32chars-long!!!!!", 5*time.Minute)
	h := NewAccountHandler(
		NewMemoryUserStore(),
		NewMemoryTenantStore(),
		NewMemoryAPIKeyStore(),
		tokenSvc,
	)

	r.POST("/v1/auth/register", h.Register)
	r.POST("/v1/auth/login", h.Login)
	return r
}

func TestRegister(t *testing.T) {
	r := setupAccountRouter()

	body, _ := json.Marshal(map[string]string{
		"email": "test@example.com", "password": "secureP@ss1",
		"name": "Test User", "company": "TestCo",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["token"] == nil || resp["token"] == "" {
		t.Fatal("token missing")
	}
	if resp["tenant_id"] == nil || resp["tenant_id"] == "" {
		t.Fatal("tenant_id missing")
	}
	if resp["role"] != "owner" {
		t.Fatalf("expected owner role, got %v", resp["role"])
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	r := setupAccountRouter()

	body, _ := json.Marshal(map[string]string{
		"email": "dup@example.com", "password": "secureP@ss1",
		"name": "User", "company": "Co",
	})

	req := httptest.NewRequest("POST", "/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("first register failed: %d", w.Code)
	}

	// Duplicate
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/v1/auth/register", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w2.Code)
	}
}

func TestRegister_BadRequest(t *testing.T) {
	r := setupAccountRouter()

	body, _ := json.Marshal(map[string]string{"email": "bad"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestLoginFlow(t *testing.T) {
	r := setupAccountRouter()

	// Register
	regBody, _ := json.Marshal(map[string]string{
		"email": "login@example.com", "password": "myP@ssword1",
		"name": "Login User", "company": "LoginCo",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/v1/auth/register", bytes.NewReader(regBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("register: %d", w.Code)
	}

	// Login
	loginBody, _ := json.Marshal(map[string]string{
		"email": "login@example.com", "password": "myP@ssword1",
	})
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/v1/auth/login", bytes.NewReader(loginBody))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &resp)
	if resp["token"] == nil || resp["token"] == "" {
		t.Fatal("token missing on login")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	r := setupAccountRouter()

	// Register
	regBody, _ := json.Marshal(map[string]string{
		"email": "wrong@example.com", "password": "correct123",
		"name": "User", "company": "Co",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/v1/auth/register", bytes.NewReader(regBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// Login with wrong password
	loginBody, _ := json.Marshal(map[string]string{
		"email": "wrong@example.com", "password": "incorrect",
	})
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/v1/auth/login", bytes.NewReader(loginBody))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w2.Code)
	}
}

func TestLogin_NonexistentUser(t *testing.T) {
	r := setupAccountRouter()

	body, _ := json.Marshal(map[string]string{
		"email": "noone@example.com", "password": "whatever",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
