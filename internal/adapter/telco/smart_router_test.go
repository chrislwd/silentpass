package telco

import (
	"context"
	"testing"
)

func TestSmartRouter_SelectBest_SingleAdapter(t *testing.T) {
	r := NewSmartRouter()
	r.Register(NewSandboxAdapter())

	if !r.IsSupported("ID") {
		t.Fatal("ID should be supported")
	}

	resp, err := r.SilentVerify(context.Background(), "+628123", "ID")
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if resp.Status != "verified" && resp.Status != "fallback_required" {
		t.Fatalf("unexpected: %s", resp.Status)
	}
}

func TestSmartRouter_Stats(t *testing.T) {
	r := NewSmartRouter()
	r.Register(NewSandboxAdapter())

	// Make a few calls
	for i := 0; i < 5; i++ {
		r.SilentVerify(context.Background(), "+628123", "ID")
	}

	stats := r.Stats()
	s, ok := stats["sandbox"]
	if !ok {
		t.Fatal("sandbox stats missing")
	}
	if s.TotalRequests != 5 {
		t.Fatalf("expected 5 requests, got %d", s.TotalRequests)
	}
	if s.SuccessRate == 0 && s.FailureCount == 0 {
		t.Fatal("should have some success or failure")
	}
}

func TestSmartRouter_CircuitBreaker(t *testing.T) {
	r := NewSmartRouter()
	r.Register(NewSandboxAdapter())

	// Simulate failures to trigger circuit breaker
	for i := 0; i < 15; i++ {
		r.recordFailure("sandbox", 1000)
	}

	stats := r.Stats()
	if !stats["sandbox"].CircuitOpen {
		t.Fatal("circuit should be open after many failures")
	}
}

func TestSmartRouter_UnsupportedCountry(t *testing.T) {
	r := NewSmartRouter()
	r.Register(NewSandboxAdapter())

	if r.IsSupported("ZZ") {
		t.Fatal("ZZ should not be supported")
	}

	_, err := r.SilentVerify(context.Background(), "+123", "ZZ")
	if err == nil {
		t.Fatal("expected error for unsupported country")
	}
}
