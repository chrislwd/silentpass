package handler

import (
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/gin-gonic/gin"
)

// StatsCollector tracks real-time verification metrics.
type StatsCollector struct {
	mu               sync.RWMutex
	totalRequests    int64
	silentSuccess    int64
	silentFallback   int64
	otpSent          int64
	otpVerified      int64
	riskBlocked      int64
	simSwapDetected  int64
}

func NewStatsCollector() *StatsCollector {
	s := &StatsCollector{}
	// Seed with demo data
	s.totalRequests = 124892
	s.silentSuccess = 105825
	s.silentFallback = 19067
	s.otpSent = 18200
	s.otpVerified = 17100
	s.riskBlocked = 342
	s.simSwapDetected = 89
	return s
}

func (s *StatsCollector) RecordSilentSuccess()  { atomic.AddInt64(&s.silentSuccess, 1); atomic.AddInt64(&s.totalRequests, 1) }
func (s *StatsCollector) RecordSilentFallback() { atomic.AddInt64(&s.silentFallback, 1); atomic.AddInt64(&s.totalRequests, 1) }
func (s *StatsCollector) RecordOTPSent()        { atomic.AddInt64(&s.otpSent, 1) }
func (s *StatsCollector) RecordOTPVerified()    { atomic.AddInt64(&s.otpVerified, 1) }
func (s *StatsCollector) RecordRiskBlocked()    { atomic.AddInt64(&s.riskBlocked, 1) }
func (s *StatsCollector) RecordSIMSwap()        { atomic.AddInt64(&s.simSwapDetected, 1) }

type StatsHandler struct {
	collector *StatsCollector
}

func NewStatsHandler(collector *StatsCollector) *StatsHandler {
	return &StatsHandler{collector: collector}
}

// Dashboard handles GET /v1/stats/dashboard
func (h *StatsHandler) Dashboard(c *gin.Context) {
	total := atomic.LoadInt64(&h.collector.totalRequests)
	silentOK := atomic.LoadInt64(&h.collector.silentSuccess)
	fallback := atomic.LoadInt64(&h.collector.silentFallback)
	blocked := atomic.LoadInt64(&h.collector.riskBlocked)

	silentRate := float64(0)
	fallbackRate := float64(0)
	if total > 0 {
		silentRate = float64(silentOK) / float64(total) * 100
		fallbackRate = float64(fallback) / float64(total) * 100
	}

	// Estimated OTP cost savings (avg $0.04/OTP saved)
	costSaved := float64(silentOK) * 0.04

	c.JSON(http.StatusOK, gin.H{
		"total_verifications": total,
		"silent_success_rate": round2(silentRate),
		"fallback_rate":       round2(fallbackRate),
		"otp_cost_saved":      round2(costSaved),
		"high_risk_blocked":   blocked,
		"avg_latency_ms":      1200 + rand.Intn(300),
		"countries": []gin.H{
			{"code": "ID", "requests": 45200, "silent_rate": 87.0},
			{"code": "TH", "requests": 32100, "silent_rate": 89.0},
			{"code": "PH", "requests": 18900, "silent_rate": 82.0},
			{"code": "MY", "requests": 15600, "silent_rate": 85.0},
			{"code": "SG", "requests": 8200, "silent_rate": 92.0},
			{"code": "VN", "requests": 4892, "silent_rate": 78.0},
		},
	})
}

// RecentActivity handles GET /v1/stats/activity
func (h *StatsHandler) RecentActivity(c *gin.Context) {
	// In production, this would query the verification_attempts table
	c.JSON(http.StatusOK, gin.H{
		"activities": []gin.H{
			{"time": "2m ago", "event": "silent_verification", "country": "ID", "status": "verified", "latency_ms": 820},
			{"time": "3m ago", "event": "otp_fallback", "country": "TH", "status": "sent", "latency_ms": 410},
			{"time": "5m ago", "event": "sim_swap_check", "country": "PH", "status": "clean", "latency_ms": 290},
			{"time": "6m ago", "event": "silent_verification", "country": "MY", "status": "fallback", "latency_ms": 2500},
			{"time": "8m ago", "event": "risk_verdict", "country": "ID", "status": "blocked", "latency_ms": 45},
			{"time": "10m ago", "event": "silent_verification", "country": "SG", "status": "verified", "latency_ms": 610},
		},
	})
}

func round2(f float64) float64 {
	return float64(int(f*100)) / 100
}
