package handler

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/silentpass/silentpass/internal/model"
)

// PolicyHandler manages policies via a PolicyStore.
type PolicyHandler struct {
	store PolicyStore
}

func NewPolicyHandler(store PolicyStore) *PolicyHandler {
	return &PolicyHandler{store: store}
}

// List handles GET /v1/policies
func (h *PolicyHandler) List(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	policies, err := h.store.List(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list policies"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"policies": policies})
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

	if err := h.store.Create(c.Request.Context(), p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create policy"})
		return
	}

	c.JSON(http.StatusCreated, p)
}

// Update handles PUT /v1/policies/:id
func (h *PolicyHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req model.UpdatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.store.Update(c.Request.Context(), id, &req); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}

	p, _ := h.store.GetByID(c.Request.Context(), id)
	c.JSON(http.StatusOK, p)
}

// Delete handles DELETE /v1/policies/:id
func (h *PolicyHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.store.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

// --- In-Memory PolicyStore ---

type MemoryPolicyStore struct {
	mu       sync.RWMutex
	policies map[string]*model.Policy
}

func NewMemoryPolicyStore() *MemoryPolicyStore {
	s := &MemoryPolicyStore{policies: make(map[string]*model.Policy)}
	s.seed()
	return s
}

func (s *MemoryPolicyStore) seed() {
	defaults := []struct {
		name, useCase, strategy, simSwap string
		countries                        []string
	}{
		{"Signup - Silent First", "signup", "silent_or_otp", "challenge", []string{"ID", "TH", "PH", "MY"}},
		{"Login - Low Friction", "login", "silent", "challenge", []string{"ID", "TH", "PH", "MY", "SG"}},
		{"Transaction - Strict", "transaction", "silent_or_otp", "block", []string{"ID", "TH"}},
		{"Phone Change - Max Security", "phone_change", "otp_only", "block", []string{"*"}},
	}
	for _, d := range defaults {
		p := &model.Policy{
			ID: uuid.New().String(), Name: d.name,
			UseCase: model.UseCase(d.useCase), Strategy: model.VerificationType(d.strategy),
			SIMSwapAction: model.Verdict(d.simSwap), Countries: d.countries,
			Priority: 10, Active: true, Config: map[string]interface{}{},
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		s.policies[p.ID] = p
	}
}

func (s *MemoryPolicyStore) List(_ context.Context, _ string) ([]*model.Policy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var list []*model.Policy
	for _, p := range s.policies {
		list = append(list, p)
	}
	return list, nil
}

func (s *MemoryPolicyStore) GetByID(_ context.Context, id string) (*model.Policy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.policies[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return p, nil
}

func (s *MemoryPolicyStore) Create(_ context.Context, p *model.Policy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policies[p.ID] = p
	return nil
}

func (s *MemoryPolicyStore) Update(_ context.Context, id string, req *model.UpdatePolicyRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.policies[id]
	if !ok {
		return fmt.Errorf("not found")
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
	return nil
}

func (s *MemoryPolicyStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.policies[id]; !ok {
		return fmt.Errorf("not found")
	}
	delete(s.policies, id)
	return nil
}
