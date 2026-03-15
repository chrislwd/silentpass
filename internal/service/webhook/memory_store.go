package webhook

import (
	"context"
	"sync"
)

// MemoryStore provides in-memory webhook subscription storage.
type MemoryStore struct {
	mu            sync.RWMutex
	subscriptions []*Subscription
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func (s *MemoryStore) Add(sub *Subscription) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subscriptions = append(s.subscriptions, sub)
}

func (s *MemoryStore) GetSubscriptions(ctx context.Context, tenantID string, eventType string) ([]*Subscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Subscription
	for _, sub := range s.subscriptions {
		if sub.TenantID != tenantID || !sub.Active {
			continue
		}
		for _, evt := range sub.Events {
			if evt == eventType || evt == "*" {
				result = append(result, sub)
				break
			}
		}
	}
	return result, nil
}
