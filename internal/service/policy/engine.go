package policy

import (
	"context"

	"github.com/silentpass/silentpass/internal/model"
	"github.com/silentpass/silentpass/internal/service/risk"
)

type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) Evaluate(ctx context.Context, tenantID string, input *risk.PolicyInput) (*model.VerdictResponse, error) {
	var reasons []string
	riskLevel := model.RiskLow
	verdict := model.VerdictAllow

	// SIM Swap signal
	if input.SIMSwapResult != nil && input.SIMSwapResult.SIMSwapDetected {
		reasons = append(reasons, "sim_swap_detected")
		riskLevel = model.RiskHigh

		switch input.SIMSwapResult.RiskLevel {
		case model.RiskHigh:
			verdict = model.VerdictBlock
		case model.RiskMedium:
			verdict = model.VerdictChallenge
		}
	}

	// Verification result signal
	if input.VerificationResult == "failed" {
		reasons = append(reasons, "verification_failed")
		if riskLevel != model.RiskHigh {
			riskLevel = model.RiskMedium
		}
		if verdict == model.VerdictAllow {
			verdict = model.VerdictChallenge
		}
	}

	var action string
	switch verdict {
	case model.VerdictChallenge:
		action = "require_otp"
	case model.VerdictBlock:
		action = "deny_operation"
	case model.VerdictReview:
		action = "manual_review"
	default:
		action = "proceed"
	}

	return &model.VerdictResponse{
		Verdict:        verdict,
		RiskLevel:      riskLevel,
		Reasons:        reasons,
		ActionRequired: action,
	}, nil
}
