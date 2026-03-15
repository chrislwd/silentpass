package policy

import (
	"context"
	"testing"

	"github.com/silentpass/silentpass/internal/model"
	"github.com/silentpass/silentpass/internal/service/risk"
)

func TestEngine_Allow(t *testing.T) {
	e := NewEngine()
	resp, err := e.Evaluate(context.Background(), "tenant-1", &risk.PolicyInput{
		VerificationResult: "verified",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Verdict != model.VerdictAllow {
		t.Fatalf("expected allow, got %s", resp.Verdict)
	}
}

func TestEngine_SIMSwapBlock(t *testing.T) {
	e := NewEngine()
	resp, err := e.Evaluate(context.Background(), "tenant-1", &risk.PolicyInput{
		VerificationResult: "verified",
		SIMSwapResult: &model.SIMSwapResponse{
			SIMSwapDetected: true,
			RiskLevel:       model.RiskHigh,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Verdict != model.VerdictBlock {
		t.Fatalf("expected block on high risk SIM swap, got %s", resp.Verdict)
	}
}

func TestEngine_SIMSwapChallenge(t *testing.T) {
	e := NewEngine()
	resp, err := e.Evaluate(context.Background(), "tenant-1", &risk.PolicyInput{
		VerificationResult: "verified",
		SIMSwapResult: &model.SIMSwapResponse{
			SIMSwapDetected: true,
			RiskLevel:       model.RiskMedium,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Verdict != model.VerdictChallenge {
		t.Fatalf("expected challenge on medium risk SIM swap, got %s", resp.Verdict)
	}
}

func TestEngine_VerificationFailed(t *testing.T) {
	e := NewEngine()
	resp, err := e.Evaluate(context.Background(), "tenant-1", &risk.PolicyInput{
		VerificationResult: "failed",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Verdict != model.VerdictChallenge {
		t.Fatalf("expected challenge on verification failure, got %s", resp.Verdict)
	}
	if resp.RiskLevel != model.RiskMedium {
		t.Fatalf("expected medium risk, got %s", resp.RiskLevel)
	}
}
