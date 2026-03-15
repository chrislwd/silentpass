package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/silentpass/silentpass/internal/model"
	"github.com/silentpass/silentpass/internal/pkg/auth"
)

// JWTAuth authenticates requests using JWT tokens (for console/dashboard users).
func JWTAuth(tokenSvc *auth.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}

		token := strings.TrimPrefix(header, "Bearer ")
		claims, err := tokenSvc.Validate(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		c.Set("user_id", claims.Subject)
		c.Set("tenant_id", claims.TenantID)
		c.Next()
	}
}

// RequireRole checks if the current user has the required role permission.
func RequireRole(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleStr, exists := c.Get("user_role")
		if !exists {
			// If no role set, allow (API key auth doesn't set roles)
			c.Next()
			return
		}

		role := roleStr.(model.Role)
		if !role.HasPermission(permission) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":      "insufficient permissions",
				"required":   permission,
			})
			return
		}

		c.Next()
	}
}
