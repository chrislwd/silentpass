package handler

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type LogEntry struct {
	ID               string `json:"id"`
	SessionID        string `json:"session_id"`
	Timestamp        string `json:"timestamp"`
	Method           string `json:"method"`
	CountryCode      string `json:"country_code"`
	UpstreamProvider string `json:"upstream_provider"`
	Result           string `json:"result"`
	LatencyMs        int    `json:"latency_ms"`
	ErrorCode        string `json:"error_code"`
}

type LogsHandler struct {
	store LogsStore
}

func NewLogsHandler(store LogsStore) *LogsHandler {
	return &LogsHandler{store: store}
}

// List handles GET /v1/logs
func (h *LogsHandler) List(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	search := c.Query("q")
	sessionFilter := c.Query("session_id")

	logs, total, err := h.store.Query(c.Request.Context(), tenantID, search, sessionFilter, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query logs"})
		return
	}
	if logs == nil {
		logs = []LogEntry{}
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"total": total,
	})
}

// --- In-Memory LogsStore ---

type MemoryLogsStore struct {
	mu      sync.RWMutex
	entries []LogEntry
}

func NewMemoryLogsStore() *MemoryLogsStore {
	s := &MemoryLogsStore{}
	now := time.Now()
	seed := []struct {
		session, method, country, provider, result, errCode string
		latency                                             int
		ago                                                 time.Duration
	}{
		{"a1b2c3d4", "silent", "ID", "sandbox", "verified", "", 820, 2 * time.Minute},
		{"e5f6g7h8", "silent", "TH", "sandbox", "fallback_required", "TIMEOUT", 2340, 3 * time.Minute},
		{"e5f6g7h8", "sms", "TH", "sandbox_otp", "sent", "", 410, 3 * time.Minute},
		{"e5f6g7h8", "sms", "TH", "sandbox_otp", "verified", "", 50, 3 * time.Minute},
		{"i9j0k1l2", "sim_swap", "PH", "sandbox", "clean", "", 290, 5 * time.Minute},
		{"m3n4o5p6", "silent", "MY", "sandbox", "verified", "", 670, 6 * time.Minute},
		{"q7r8s9t0", "verdict", "ID", "-", "block", "SIM_SWAP_HIGH", 45, 8 * time.Minute},
		{"u1v2w3x4", "silent", "SG", "sandbox", "verified", "", 610, 10 * time.Minute},
		{"y5z6a7b8", "silent", "ID", "sandbox", "verified", "", 750, 12 * time.Minute},
		{"c9d0e1f2", "silent", "VN", "sandbox", "fallback_required", "NO_COVERAGE", 100, 15 * time.Minute},
	}
	for _, e := range seed {
		s.entries = append(s.entries, LogEntry{
			ID: uuid.New().String(), SessionID: e.session,
			Timestamp: now.Add(-e.ago).Format("2006-01-02 15:04:05"),
			Method: e.method, CountryCode: e.country, UpstreamProvider: e.provider,
			Result: e.result, LatencyMs: e.latency, ErrorCode: e.errCode,
		})
	}
	return s
}

func (s *MemoryLogsStore) Query(_ context.Context, _ string, search string, sessionID string, limit int) ([]LogEntry, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []LogEntry
	for _, e := range s.entries {
		if sessionID != "" && e.SessionID != sessionID {
			continue
		}
		if search != "" && !matchesSearch(e, search) {
			continue
		}
		filtered = append(filtered, e)
	}
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered, len(filtered), nil
}

func (s *MemoryLogsStore) Append(_ context.Context, entry LogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().Format("2006-01-02 15:04:05")
	}
	s.entries = append([]LogEntry{entry}, s.entries...)
	if len(s.entries) > 1000 {
		s.entries = s.entries[:1000]
	}
	return nil
}

func matchesSearch(e LogEntry, q string) bool {
	return contains(e.SessionID, q) || contains(e.Method, q) ||
		contains(e.CountryCode, q) || contains(e.Result, q) ||
		contains(e.ErrorCode, q)
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
