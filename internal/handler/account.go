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
	"github.com/silentpass/silentpass/internal/pkg/auth"
	"github.com/silentpass/silentpass/internal/pkg/crypto"
)

// UserStore abstracts user persistence.
type UserStore interface {
	CreateUser(ctx context.Context, user *model.User) error
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id string) (*model.User, error)
}

// TenantStore abstracts tenant management.
type TenantStore interface {
	CreateTenant(ctx context.Context, tenant *model.Tenant) error
	GetTenantByID(ctx context.Context, id string) (*model.Tenant, error)
	AddUserToTenant(ctx context.Context, userID, tenantID string, role model.Role) error
	GetUserTenants(ctx context.Context, userID string) ([]model.UserTenant, error)
	GetUserRole(ctx context.Context, userID, tenantID string) (model.Role, error)
}

// APIKeyStore abstracts API key management.
type APIKeyStore interface {
	CreateAPIKey(ctx context.Context, key *model.APIKeyRecord) error
	ListByTenant(ctx context.Context, tenantID string) ([]*model.APIKeyRecord, error)
	DeleteAPIKey(ctx context.Context, id, tenantID string) error
}

type AccountHandler struct {
	users   UserStore
	tenants TenantStore
	apiKeys APIKeyStore
	tokens  *auth.TokenService
}

func NewAccountHandler(users UserStore, tenants TenantStore, apiKeys APIKeyStore, tokens *auth.TokenService) *AccountHandler {
	return &AccountHandler{users: users, tenants: tenants, apiKeys: apiKeys, tokens: tokens}
}

// Register handles POST /v1/auth/register
func (h *AccountHandler) Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if email exists
	if existing, _ := h.users.GetByEmail(c.Request.Context(), req.Email); existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}

	// Hash password
	hash, err := crypto.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "registration failed"})
		return
	}

	now := time.Now()
	user := &model.User{
		ID: uuid.New().String(), Email: req.Email,
		PasswordHash: hash, Name: req.Name,
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}

	if err := h.users.CreateUser(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "registration failed"})
		return
	}

	// Create tenant for this user
	apiKey, apiKeyHash, _ := crypto.GenerateAPIKey("sk_live_")
	apiSecret, _ := crypto.GenerateSecret(32)
	tenant := &model.Tenant{
		ID: uuid.New().String(), Name: req.Company,
		APIKey: apiKey, APISecret: apiSecret,
		Status: "active", Plan: "free",
		CreatedAt: now, UpdatedAt: now,
	}

	if err := h.tenants.CreateTenant(c.Request.Context(), tenant); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create organization"})
		return
	}

	// Link user as owner
	_ = h.tenants.AddUserToTenant(c.Request.Context(), user.ID, tenant.ID, model.RoleOwner)

	// Store the API key record
	_ = h.apiKeys.CreateAPIKey(c.Request.Context(), &model.APIKeyRecord{
		ID: uuid.New().String(), TenantID: tenant.ID,
		Name: "Default Key", KeyPrefix: apiKey[:12],
		KeyHash: apiKeyHash, Scopes: []string{"*"},
		CreatedBy: user.ID, CreatedAt: now,
	})

	// Generate session token
	token, _ := h.tokens.Generate(tenant.ID, tenant.ID, "", "session", "register")

	c.JSON(http.StatusCreated, model.AuthResponse{
		Token: token, User: user,
		TenantID: tenant.ID, Role: model.RoleOwner,
	})
}

// Login handles POST /v1/auth/login
func (h *AccountHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.users.GetByEmail(c.Request.Context(), req.Email)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if !crypto.CheckPassword(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Get user's first tenant
	memberships, _ := h.tenants.GetUserTenants(c.Request.Context(), user.ID)
	if len(memberships) == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "no organization found"})
		return
	}

	m := memberships[0]
	token, _ := h.tokens.Generate(m.TenantID, m.TenantID, "", "session", "login")

	c.JSON(http.StatusOK, model.AuthResponse{
		Token: token, User: user,
		TenantID: m.TenantID, Role: m.Role,
	})
}

// CreateAPIKey handles POST /v1/account/api-keys
func (h *AccountHandler) CreateAPIKey(c *gin.Context) {
	var req model.CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	key, keyHash, err := crypto.GenerateAPIKey("sk_live_")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "key generation failed"})
		return
	}

	scopes := req.Scopes
	if len(scopes) == 0 {
		scopes = []string{"*"}
	}

	record := &model.APIKeyRecord{
		ID: uuid.New().String(), TenantID: tenantID,
		Name: req.Name, KeyPrefix: key[:12], KeyHash: keyHash,
		Scopes: scopes, CreatedAt: time.Now(),
	}

	if err := h.apiKeys.CreateAPIKey(c.Request.Context(), record); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create key"})
		return
	}

	c.JSON(http.StatusCreated, model.CreateAPIKeyResponse{
		ID: record.ID, Name: record.Name,
		Key: key, KeyPrefix: record.KeyPrefix,
		Scopes: scopes,
	})
}

// ListAPIKeys handles GET /v1/account/api-keys
func (h *AccountHandler) ListAPIKeys(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	keys, err := h.apiKeys.ListByTenant(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list keys"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"api_keys": keys})
}

// DeleteAPIKey handles DELETE /v1/account/api-keys/:id
func (h *AccountHandler) DeleteAPIKey(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")
	if err := h.apiKeys.DeleteAPIKey(c.Request.Context(), id, tenantID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "key not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

// InviteUser handles POST /v1/account/users
func (h *AccountHandler) InviteUser(c *gin.Context) {
	var req model.InviteUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	user, _ := h.users.GetByEmail(c.Request.Context(), req.Email)
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found, they must register first"})
		return
	}

	role := model.Role(req.Role)
	if err := h.tenants.AddUserToTenant(c.Request.Context(), user.ID, tenantID, role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user_id": user.ID, "role": role})
}

// --- In-Memory Stores ---

type MemoryUserStore struct {
	mu      sync.RWMutex
	byEmail map[string]*model.User
	byID    map[string]*model.User
}

func NewMemoryUserStore() *MemoryUserStore {
	return &MemoryUserStore{
		byEmail: make(map[string]*model.User),
		byID:    make(map[string]*model.User),
	}
}

func (s *MemoryUserStore) CreateUser(_ context.Context, u *model.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byEmail[u.Email] = u
	s.byID[u.ID] = u
	return nil
}

func (s *MemoryUserStore) GetByEmail(_ context.Context, email string) (*model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.byEmail[email]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return u, nil
}

func (s *MemoryUserStore) GetByID(_ context.Context, id string) (*model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.byID[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return u, nil
}

type MemoryTenantStore struct {
	mu          sync.RWMutex
	tenants     map[string]*model.Tenant
	memberships []model.UserTenant
}

func NewMemoryTenantStore() *MemoryTenantStore {
	return &MemoryTenantStore{tenants: make(map[string]*model.Tenant)}
}

func (s *MemoryTenantStore) CreateTenant(_ context.Context, t *model.Tenant) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tenants[t.ID] = t
	return nil
}

func (s *MemoryTenantStore) GetTenantByID(_ context.Context, id string) (*model.Tenant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tenants[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return t, nil
}

func (s *MemoryTenantStore) AddUserToTenant(_ context.Context, userID, tenantID string, role model.Role) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.memberships = append(s.memberships, model.UserTenant{
		UserID: userID, TenantID: tenantID, Role: role, CreatedAt: time.Now(),
	})
	return nil
}

func (s *MemoryTenantStore) GetUserTenants(_ context.Context, userID string) ([]model.UserTenant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []model.UserTenant
	for _, m := range s.memberships {
		if m.UserID == userID {
			result = append(result, m)
		}
	}
	return result, nil
}

func (s *MemoryTenantStore) GetUserRole(_ context.Context, userID, tenantID string) (model.Role, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, m := range s.memberships {
		if m.UserID == userID && m.TenantID == tenantID {
			return m.Role, nil
		}
	}
	return "", fmt.Errorf("not found")
}

type MemoryAPIKeyStore struct {
	mu   sync.RWMutex
	keys []*model.APIKeyRecord
}

func NewMemoryAPIKeyStore() *MemoryAPIKeyStore {
	return &MemoryAPIKeyStore{}
}

func (s *MemoryAPIKeyStore) CreateAPIKey(_ context.Context, key *model.APIKeyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys = append(s.keys, key)
	return nil
}

func (s *MemoryAPIKeyStore) ListByTenant(_ context.Context, tenantID string) ([]*model.APIKeyRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*model.APIKeyRecord
	for _, k := range s.keys {
		if k.TenantID == tenantID {
			result = append(result, k)
		}
	}
	return result, nil
}

func (s *MemoryAPIKeyStore) DeleteAPIKey(_ context.Context, id, tenantID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, k := range s.keys {
		if k.ID == id && k.TenantID == tenantID {
			s.keys = append(s.keys[:i], s.keys[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("not found")
}
