package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type TenantResolver interface {
	ResolveByAPIKey(apiKey string) (tenantID string, apiSecret string, err error)
}

func APIKeyAuth(resolver TenantResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			auth := c.GetHeader("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				apiKey = strings.TrimPrefix(auth, "Bearer ")
			}
		}

		if apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing API key"})
			return
		}

		tenantID, apiSecret, err := resolver.ResolveByAPIKey(apiKey)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
			return
		}

		// Verify HMAC signature if present
		signature := c.GetHeader("X-Signature")
		if signature != "" {
			if !verifyHMAC(c, apiSecret, signature) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
				return
			}
		}

		c.Set("tenant_id", tenantID)
		c.Next()
	}
}

func verifyHMAC(c *gin.Context, secret, signature string) bool {
	timestamp := c.GetHeader("X-Timestamp")
	payload := c.Request.Method + c.Request.URL.Path + timestamp

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(signature))
}
