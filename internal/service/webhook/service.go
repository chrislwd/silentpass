package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Event struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	Timestamp string      `json:"timestamp"`
	TenantID  string      `json:"tenant_id"`
	Data      interface{} `json:"data"`
}

type Subscription struct {
	ID        string   `json:"id"`
	TenantID  string   `json:"tenant_id"`
	URL       string   `json:"url"`
	Secret    string   `json:"-"`
	Events    []string `json:"events"`
	Active    bool     `json:"active"`
	CreatedAt string   `json:"created_at"`
}

type Store interface {
	GetSubscriptions(ctx context.Context, tenantID string, eventType string) ([]*Subscription, error)
}

type Service struct {
	store  Store
	client *http.Client
}

func NewService(store Store) *Service {
	return &Service{
		store: store,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Emit sends an event to all matching webhook subscriptions asynchronously.
func (s *Service) Emit(ctx context.Context, event *Event) {
	subs, err := s.store.GetSubscriptions(ctx, event.TenantID, event.Type)
	if err != nil {
		log.Printf("webhook: failed to get subscriptions: %v", err)
		return
	}

	for _, sub := range subs {
		go s.deliver(sub, event)
	}
}

func (s *Service) deliver(sub *Subscription, event *Event) {
	body, err := json.Marshal(event)
	if err != nil {
		log.Printf("webhook: marshal error: %v", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, sub.URL, bytes.NewReader(body))
	if err != nil {
		log.Printf("webhook: create request error: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-ID", event.ID)
	req.Header.Set("X-Webhook-Event", event.Type)
	req.Header.Set("X-Webhook-Timestamp", event.Timestamp)

	if sub.Secret != "" {
		sig := sign(body, sub.Secret)
		req.Header.Set("X-Webhook-Signature", sig)
	}

	// Retry up to 3 times with exponential backoff
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*attempt) * time.Second)
		}

		resp, err := s.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return
		}
		lastErr = fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	log.Printf("webhook: delivery failed after 3 attempts to %s: %v", sub.URL, lastErr)
}

func sign(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// Event types
const (
	EventVerificationCompleted = "verification.completed"
	EventVerificationFailed    = "verification.failed"
	EventOTPSent               = "otp.sent"
	EventOTPVerified           = "otp.verified"
	EventSIMSwapDetected       = "sim_swap.detected"
	EventRiskVerdictBlock      = "risk.verdict.block"
	EventRiskVerdictChallenge  = "risk.verdict.challenge"
)
