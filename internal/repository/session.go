package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/silentpass/silentpass/internal/model"
)

// SessionRepo provides in-memory session storage for development.
// Replace with PostgreSQL implementation for production.
type SessionRepo struct {
	mu       sync.RWMutex
	sessions map[string]*model.Session
}

func NewSessionRepo() *SessionRepo {
	return &SessionRepo{
		sessions: make(map[string]*model.Session),
	}
}

func (r *SessionRepo) CreateSession(ctx context.Context, session *model.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[session.ID] = session
	return nil
}

func (r *SessionRepo) GetSession(ctx context.Context, sessionID string) (*model.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return s, nil
}

func (r *SessionRepo) UpdateSessionStatus(ctx context.Context, sessionID string, status model.SessionStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}
	s.Status = status
	return nil
}
