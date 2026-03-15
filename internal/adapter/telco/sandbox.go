package telco

import (
	"context"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/silentpass/silentpass/internal/model"
)

// SandboxAdapter simulates telco responses for development and testing.
type SandboxAdapter struct{}

func NewSandboxAdapter() *SandboxAdapter {
	return &SandboxAdapter{}
}

func (a *SandboxAdapter) Name() string {
	return "sandbox"
}

func (a *SandboxAdapter) SupportedCountries() []string {
	return []string{"ID", "TH", "PH", "MY", "SG", "VN", "BR", "MX"}
}

func (a *SandboxAdapter) SupportedCapabilities() []string {
	return []string{"silent_verify", "sim_swap"}
}

func (a *SandboxAdapter) SilentVerify(ctx context.Context, phoneHash, countryCode string) (*model.SilentVerifyResponse, error) {
	// Simulate latency
	time.Sleep(time.Duration(200+rand.Intn(300)) * time.Millisecond)

	// Sandbox: success rate ~85%
	if rand.Float64() < 0.85 {
		return &model.SilentVerifyResponse{
			Status:          model.ResultVerified,
			ConfidenceScore: 0.95 + rand.Float64()*0.05,
			TelcoSignal:     "match_confirmed",
			Token:           "sv_" + uuid.New().String(),
		}, nil
	}

	return &model.SilentVerifyResponse{
		Status:      model.ResultFallbackRequired,
		TelcoSignal: "no_match_or_timeout",
	}, nil
}

func (a *SandboxAdapter) CheckSIMSwap(ctx context.Context, phoneNumber, countryCode string) (*model.SIMSwapResponse, error) {
	time.Sleep(time.Duration(100+rand.Intn(200)) * time.Millisecond)

	// Sandbox: 10% SIM swap detected
	if rand.Float64() < 0.10 {
		return &model.SIMSwapResponse{
			SIMSwapDetected: true,
			LastChangeTime:  time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
			RiskLevel:       model.RiskHigh,
			Recommendation:  model.VerdictBlock,
		}, nil
	}

	return &model.SIMSwapResponse{
		SIMSwapDetected: false,
		LastChangeTime:  time.Now().Add(-180 * 24 * time.Hour).Format(time.RFC3339),
		RiskLevel:       model.RiskLow,
		Recommendation:  model.VerdictAllow,
	}, nil
}
