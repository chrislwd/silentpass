package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type mockResolver struct {
	tenants map[string]struct{ id, secret string }
}

func (m *mockResolver) ResolveByAPIKey(apiKey string) (string, string, error) {
	t, ok := m.tenants[apiKey]
	if !ok {
		return "", "", fmt.Errorf("not found")
	}
	return t.id, t.secret, nil
}

func newTestRouter() (*gin.Engine, *mockResolver) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	resolver := &mockResolver{
		tenants: map[string]struct{ id, secret string }{
			"sk_valid": {"tenant-1", "secret-123"},
		},
	}
	r.Use(APIKeyAuth(resolver))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"tenant_id": c.GetString("tenant_id")})
	})
	return r, resolver
}

func TestAPIKeyAuth_ValidKey(t *testing.T) {
	r, _ := newTestRouter()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "sk_valid")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAPIKeyAuth_BearerToken(t *testing.T) {
	r, _ := newTestRouter()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer sk_valid")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200 with Bearer, got %d", w.Code)
	}
}

func TestAPIKeyAuth_MissingKey(t *testing.T) {
	r, _ := newTestRouter()
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAPIKeyAuth_InvalidKey(t *testing.T) {
	r, _ := newTestRouter()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "sk_invalid")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAPIKeyAuth_HMACSignature(t *testing.T) {
	r, _ := newTestRouter()

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	payload := "GET/test" + timestamp
	mac := hmac.New(sha256.New, []byte("secret-123"))
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "sk_valid")
	req.Header.Set("X-Timestamp", timestamp)
	req.Header.Set("X-Signature", sig)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200 with valid HMAC, got %d", w.Code)
	}
}

func TestAPIKeyAuth_BadHMACSignature(t *testing.T) {
	r, _ := newTestRouter()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "sk_valid")
	req.Header.Set("X-Timestamp", "12345")
	req.Header.Set("X-Signature", "badsignature")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("expected 401 for bad HMAC, got %d", w.Code)
	}
}
