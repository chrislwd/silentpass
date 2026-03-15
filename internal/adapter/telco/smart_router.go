package telco

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/silentpass/silentpass/internal/model"
)

// SmartRouter extends Router with intelligent upstream selection
// based on success rate, latency, and circuit breaker state.
type SmartRouter struct {
	adapters       []Adapter
	countryAdapters map[string][]Adapter // countryCode -> adapters (multiple possible)
	stats          map[string]*adapterStats
	mu             sync.RWMutex
}

type adapterStats struct {
	totalRequests int64
	successCount  int64
	failureCount  int64
	totalLatency  int64 // milliseconds
	lastFailure   time.Time
	circuitOpen   bool
	circuitUntil  time.Time
}

func NewSmartRouter() *SmartRouter {
	return &SmartRouter{
		countryAdapters: make(map[string][]Adapter),
		stats:           make(map[string]*adapterStats),
	}
}

func (r *SmartRouter) Register(adapter Adapter) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.adapters = append(r.adapters, adapter)
	for _, country := range adapter.SupportedCountries() {
		r.countryAdapters[country] = append(r.countryAdapters[country], adapter)
	}
	r.stats[adapter.Name()] = &adapterStats{}
}

func (r *SmartRouter) IsSupported(countryCode string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	adapters, ok := r.countryAdapters[countryCode]
	if !ok {
		return false
	}
	// At least one adapter must not be circuit-open
	for _, a := range adapters {
		s := r.stats[a.Name()]
		if !s.circuitOpen || time.Now().After(s.circuitUntil) {
			return true
		}
	}
	return false
}

// selectBest picks the best adapter for a country based on:
// 1. Circuit breaker state (skip open circuits)
// 2. Success rate (higher is better)
// 3. Average latency (lower is better)
func (r *SmartRouter) selectBest(countryCode string) (Adapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapters, ok := r.countryAdapters[countryCode]
	if !ok || len(adapters) == 0 {
		return nil, fmt.Errorf("no adapter for country: %s", countryCode)
	}

	var best Adapter
	bestScore := -1.0

	for _, a := range adapters {
		s := r.stats[a.Name()]

		// Skip circuit-open adapters (unless cooldown passed)
		if s.circuitOpen && time.Now().Before(s.circuitUntil) {
			continue
		}

		score := r.computeScore(s)
		if score > bestScore {
			bestScore = score
			best = a
		}
	}

	if best == nil {
		// All circuits open, try the one with earliest recovery
		var earliest time.Time
		for _, a := range adapters {
			s := r.stats[a.Name()]
			if best == nil || s.circuitUntil.Before(earliest) {
				best = a
				earliest = s.circuitUntil
			}
		}
	}

	return best, nil
}

// computeScore returns a score 0-100 for adapter selection.
// Higher = preferred.
func (r *SmartRouter) computeScore(s *adapterStats) float64 {
	if s.totalRequests == 0 {
		return 50 // Unknown adapter gets middle score
	}

	successRate := float64(s.successCount) / float64(s.totalRequests)
	avgLatency := float64(s.totalLatency) / float64(s.totalRequests)

	// Score = 70% success rate + 30% latency score
	// Latency score: 1000ms = 50, lower = higher score
	latencyScore := math.Max(0, 100-avgLatency/10)

	return successRate*70 + latencyScore*0.3
}

func (r *SmartRouter) recordSuccess(name string, latencyMs int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.stats[name]
	s.totalRequests++
	s.successCount++
	s.totalLatency += latencyMs

	// Reset circuit breaker on success
	if s.circuitOpen {
		s.circuitOpen = false
	}
}

func (r *SmartRouter) recordFailure(name string, latencyMs int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.stats[name]
	s.totalRequests++
	s.failureCount++
	s.totalLatency += latencyMs
	s.lastFailure = time.Now()

	// Open circuit breaker after 5 consecutive failures or >50% failure rate
	if s.totalRequests > 10 {
		failRate := float64(s.failureCount) / float64(s.totalRequests)
		if failRate > 0.5 {
			s.circuitOpen = true
			s.circuitUntil = time.Now().Add(30 * time.Second)
		}
	}
}

func (r *SmartRouter) SilentVerify(ctx context.Context, phoneHash, countryCode string) (*model.SilentVerifyResponse, error) {
	adapter, err := r.selectBest(countryCode)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	resp, err := adapter.SilentVerify(ctx, phoneHash, countryCode)
	latency := time.Since(start).Milliseconds()

	if err != nil || (resp != nil && resp.Status == model.ResultFailed) {
		r.recordFailure(adapter.Name(), latency)
	} else {
		r.recordSuccess(adapter.Name(), latency)
	}

	return resp, err
}

func (r *SmartRouter) CheckSIMSwap(ctx context.Context, phoneNumber, countryCode string) (*model.SIMSwapResponse, error) {
	adapter, err := r.selectBest(countryCode)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	resp, err := adapter.CheckSIMSwap(ctx, phoneNumber, countryCode)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		r.recordFailure(adapter.Name(), latency)
	} else {
		r.recordSuccess(adapter.Name(), latency)
	}

	return resp, err
}

// Stats returns current adapter statistics for monitoring.
func (r *SmartRouter) Stats() map[string]AdapterStat {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]AdapterStat)
	for name, s := range r.stats {
		stat := AdapterStat{
			TotalRequests: s.totalRequests,
			SuccessCount:  s.successCount,
			FailureCount:  s.failureCount,
			CircuitOpen:   s.circuitOpen,
		}
		if s.totalRequests > 0 {
			stat.SuccessRate = float64(s.successCount) / float64(s.totalRequests) * 100
			stat.AvgLatencyMs = float64(s.totalLatency) / float64(s.totalRequests)
		}
		result[name] = stat
	}
	return result
}

type AdapterStat struct {
	TotalRequests int64   `json:"total_requests"`
	SuccessCount  int64   `json:"success_count"`
	FailureCount  int64   `json:"failure_count"`
	SuccessRate   float64 `json:"success_rate"`
	AvgLatencyMs  float64 `json:"avg_latency_ms"`
	CircuitOpen   bool    `json:"circuit_open"`
}
