package repository

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/silentpass/silentpass/internal/model"
)

// TenantRepo provides in-memory tenant storage with seed data for development.
type TenantRepo struct {
	mu      sync.RWMutex
	tenants map[string]*model.Tenant // api_key -> tenant
	byID    map[string]*model.Tenant // id -> tenant
}

func NewTenantRepo() *TenantRepo {
	repo := &TenantRepo{
		tenants: make(map[string]*model.Tenant),
		byID:    make(map[string]*model.Tenant),
	}
	// Seed a sandbox tenant
	repo.Seed("sk_test_sandbox_key_001", "sandbox_secret_001", "SilentPass Sandbox")
	return repo
}

func (r *TenantRepo) Seed(apiKey, apiSecret, name string) {
	t := &model.Tenant{
		ID:        uuid.New().String(),
		Name:      name,
		APIKey:    apiKey,
		APISecret: apiSecret,
		Status:    "active",
		Plan:      "sandbox",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	r.tenants[apiKey] = t
	r.byID[t.ID] = t
}

func (r *TenantRepo) ResolveByAPIKey(apiKey string) (tenantID string, apiSecret string, err error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tenants[apiKey]
	if !ok {
		return "", "", fmt.Errorf("tenant not found")
	}
	return t.ID, t.APISecret, nil
}

func (r *TenantRepo) GetByID(id string) (*model.Tenant, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.byID[id]
	if !ok {
		return nil, fmt.Errorf("tenant not found")
	}
	return t, nil
}
