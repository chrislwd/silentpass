package handler

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/silentpass/silentpass/internal/model"
)

// PolicyHandler manages policies with in-memory store.
// Replace with PGPolicyRepo for production.
type PolicyHandler struct {
	mu       sync.RWMutex
	policies map[string]*model.Policy
}

func NewPolicyHandler() *PolicyHandler {
	h := &PolicyHandler{
		policies: make(map[string]*model.Policy),
	}
	// Seed default policies
	h.seed()
	return h
}

func (h *PolicyHandler) seed() {
	defaults := []struct {
		name, useCase, strategy, simSwap string
		countries                        []string
		priority                         int
	}{
		{"Signup - Silent First", "signup", "silent_or_otp", "challenge", []string{"ID", "TH", "PH", "MY"}, 10},
		{"Login - Low Friction", "login", "silent", "challenge", []string{"ID", "TH", "PH", "MY", "SG"}, 10},
		{"Transaction - Strict", "transaction", "silent_or_otp", "block", []string{"ID", "TH"}, 10},
		{"Phone Change - Max Security", "phone_change", "otp_only", "block", []string{"*"}, 10},
	}
	for _, d := range defaults {
		p := &model.Policy{
			ID:            uuid.New().String(),
			Name:          d.name,
			UseCase:       model.UseCase(d.useCase),
			Strategy:      model.VerificationType(d.strategy),
			SIMSwapAction: model.Verdict(d.simSwap),
			Countries:     d.countries,
			Priority:      d.priority,
			Active:        true,
			Config:        map[string]interface{}{},
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		h.policies[p.ID] = p
	}
}

// List handles GET /v1/policies
func (h *PolicyHandler) List(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var list []*model.Policy
	for _, p := range h.policies {
		list = append(list, p)
	}
	c.JSON(http.StatusOK, gin.H{"policies": list})
}

// Create handles POST /v1/policies
func (h *PolicyHandler) Create(c *gin.Context) {
	var req model.CreatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	now := time.Now()
	p := &model.Policy{
		ID:            uuid.New().String(),
		TenantID:      tenantID,
		Name:          req.Name,
		UseCase:       model.UseCase(req.UseCase),
		Strategy:      model.VerificationType(req.Strategy),
		SIMSwapAction: model.Verdict(req.SIMSwapAction),
		Countries:     req.Countries,
		Priority:      req.Priority,
		Active:        true,
		Config:        map[string]interface{}{},
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	h.mu.Lock()
	h.policies[p.ID] = p
	h.mu.Unlock()

	c.JSON(http.StatusCreated, p)
}

// Update handles PUT /v1/policies/:id
func (h *PolicyHandler) Update(c *gin.Context) {
	id := c.Param("id")

	h.mu.Lock()
	defer h.mu.Unlock()

	p, ok := h.policies[id]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}

	var req model.UpdatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.Strategy != nil {
		p.Strategy = model.VerificationType(*req.Strategy)
	}
	if req.SIMSwapAction != nil {
		p.SIMSwapAction = model.Verdict(*req.SIMSwapAction)
	}
	if req.Countries != nil {
		p.Countries = req.Countries
	}
	if req.Priority != nil {
		p.Priority = *req.Priority
	}
	if req.Active != nil {
		p.Active = *req.Active
	}
	p.UpdatedAt = time.Now()

	c.JSON(http.StatusOK, p)
}

// Delete handles DELETE /v1/policies/:id
func (h *PolicyHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.policies[id]; !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}

	delete(h.policies, id)
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}
