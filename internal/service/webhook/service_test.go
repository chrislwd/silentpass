package webhook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestEmit_DeliversToMatchingSubscriptions(t *testing.T) {
	var received int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&received, 1)

		// Verify headers
		if r.Header.Get("X-Webhook-Event") == "" {
			t.Error("missing X-Webhook-Event header")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("wrong content type")
		}

		var event Event
		json.NewDecoder(r.Body).Decode(&event)
		if event.Type != EventVerificationCompleted {
			t.Errorf("expected verification.completed, got %s", event.Type)
		}

		w.WriteHeader(200)
	}))
	defer ts.Close()

	store := NewMemoryStore()
	store.Add(&Subscription{
		ID:       "sub-1",
		TenantID: "tenant-1",
		URL:      ts.URL,
		Events:   []string{EventVerificationCompleted},
		Active:   true,
	})

	svc := NewService(store)
	svc.Emit(context.Background(), &Event{
		ID:        "evt-1",
		Type:      EventVerificationCompleted,
		Timestamp: time.Now().Format(time.RFC3339),
		TenantID:  "tenant-1",
		Data:      map[string]string{"session_id": "sess-123"},
	})

	// Wait for async delivery
	time.Sleep(100 * time.Millisecond)
	if atomic.LoadInt64(&received) != 1 {
		t.Fatalf("expected 1 delivery, got %d", received)
	}
}

func TestEmit_SkipsInactiveSubscription(t *testing.T) {
	var received int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&received, 1)
		w.WriteHeader(200)
	}))
	defer ts.Close()

	store := NewMemoryStore()
	store.Add(&Subscription{
		ID:       "sub-1",
		TenantID: "tenant-1",
		URL:      ts.URL,
		Events:   []string{"*"},
		Active:   false,
	})

	svc := NewService(store)
	svc.Emit(context.Background(), &Event{
		ID:       "evt-1",
		Type:     EventVerificationCompleted,
		TenantID: "tenant-1",
	})

	time.Sleep(100 * time.Millisecond)
	if atomic.LoadInt64(&received) != 0 {
		t.Fatalf("expected 0 deliveries for inactive sub, got %d", received)
	}
}

func TestEmit_SkipsWrongTenant(t *testing.T) {
	var received int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&received, 1)
		w.WriteHeader(200)
	}))
	defer ts.Close()

	store := NewMemoryStore()
	store.Add(&Subscription{
		ID:       "sub-1",
		TenantID: "tenant-1",
		URL:      ts.URL,
		Events:   []string{"*"},
		Active:   true,
	})

	svc := NewService(store)
	svc.Emit(context.Background(), &Event{
		ID:       "evt-1",
		Type:     EventVerificationCompleted,
		TenantID: "tenant-2", // different tenant
	})

	time.Sleep(100 * time.Millisecond)
	if atomic.LoadInt64(&received) != 0 {
		t.Fatalf("expected 0 deliveries for wrong tenant, got %d", received)
	}
}

func TestEmit_WildcardEvent(t *testing.T) {
	var received int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&received, 1)
		w.WriteHeader(200)
	}))
	defer ts.Close()

	store := NewMemoryStore()
	store.Add(&Subscription{
		ID:       "sub-1",
		TenantID: "tenant-1",
		URL:      ts.URL,
		Events:   []string{"*"},
		Active:   true,
	})

	svc := NewService(store)
	svc.Emit(context.Background(), &Event{
		ID:       "evt-1",
		Type:     EventSIMSwapDetected,
		TenantID: "tenant-1",
	})

	time.Sleep(100 * time.Millisecond)
	if atomic.LoadInt64(&received) != 1 {
		t.Fatalf("expected 1 delivery for wildcard, got %d", received)
	}
}

func TestEmit_HMACSignature(t *testing.T) {
	var gotSig string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSig = r.Header.Get("X-Webhook-Signature")
		w.WriteHeader(200)
	}))
	defer ts.Close()

	store := NewMemoryStore()
	store.Add(&Subscription{
		ID:       "sub-1",
		TenantID: "tenant-1",
		URL:      ts.URL,
		Secret:   "webhook-secret",
		Events:   []string{"*"},
		Active:   true,
	})

	svc := NewService(store)
	svc.Emit(context.Background(), &Event{
		ID:       "evt-1",
		Type:     EventVerificationCompleted,
		TenantID: "tenant-1",
	})

	time.Sleep(100 * time.Millisecond)
	if gotSig == "" {
		t.Fatal("expected X-Webhook-Signature header")
	}
}
