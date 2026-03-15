package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RedisRateLimit uses Redis for distributed rate limiting with sliding window.
func RedisRateLimit(client *redis.Client, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetString("tenant_id")
		if key == "" {
			key = c.ClientIP()
		}

		redisKey := fmt.Sprintf("rl:%s:%d", key, time.Now().Unix()/int64(window.Seconds()))
		ctx := context.Background()

		pipe := client.Pipeline()
		incr := pipe.Incr(ctx, redisKey)
		pipe.Expire(ctx, redisKey, window)
		_, err := pipe.Exec(ctx)
		if err != nil {
			// Redis error: fall through (fail open)
			c.Next()
			return
		}

		count := incr.Val()
		remaining := int64(limit) - count
		if remaining < 0 {
			remaining = 0
		}

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(window).Unix()))

		if count > int64(limit) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":            "rate limit exceeded",
				"retry_after_secs": int(window.Seconds()),
			})
			return
		}

		c.Next()
	}
}
