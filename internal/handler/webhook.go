package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/silentpass/silentpass/internal/service/webhook"
)

type WebhookHandler struct {
	store *webhook.MemoryStore
}

func NewWebhookHandler(store *webhook.MemoryStore) *WebhookHandler {
	return &WebhookHandler{store: store}
}

type CreateWebhookRequest struct {
	URL    string   `json:"url" binding:"required"`
	Events []string `json:"events" binding:"required"`
	Secret string   `json:"secret"`
}

// Create handles POST /v1/webhooks
func (h *WebhookHandler) Create(c *gin.Context) {
	var req CreateWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	sub := &webhook.Subscription{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		URL:       req.URL,
		Secret:    req.Secret,
		Events:    req.Events,
		Active:    true,
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	h.store.Add(sub)

	c.JSON(http.StatusCreated, gin.H{
		"id":         sub.ID,
		"url":        sub.URL,
		"events":     sub.Events,
		"active":     sub.Active,
		"created_at": sub.CreatedAt,
	})
}
