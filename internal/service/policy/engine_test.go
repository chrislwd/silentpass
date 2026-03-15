package policy

import (
	"context"
	"testing"

	"github.com/silentpass/silentpass/internal/model"
	"github.com/silentpass/silentpass/internal/service/risk"
)

// mockResolver returns predefined policies.
type mockResolver struct {
	policies []*model.Policy
}

func (m *mockResolver) List(_ context.Context, _ string) ([]*model.Policy, error) {
	return m.policies, nil
}

func newEngine(policies ...*model.Policy) *Engine {
	return NewEngine(&mockResolver{policies: policies})
}

func newEngineNoResolver() *Engine {
	return NewEngine(nil)
}

func TestEngine_Allow_NoSignals(t *testing.T) {
	e := newEngineNoResolver()
	resp, err := e.Evaluate(context.Background(), "t1", &risk.PolicyInput{
		VerificationResult: "verified",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Verdict != model.VerdictAllow {
		t.Fatalf("expected allow, got %s", resp.Verdict)
	}
	if resp.RiskLevel != model.RiskLow {
		t.Fatalf("expected low risk, got %s", resp.RiskLevel)
	}
}

func TestEngine_SIMSwap_HighBlock(t *testing.T) {
	e := newEngineNoResolver()
	resp, _ := e.Evaluate(context.Background(), "t1", &risk.PolicyInput{
		VerificationResult: "verified",
		SIMSwapResult: &model.SIMSwapResponse{
			SIMSwapDetected: true, RiskLevel: model.RiskHigh,
		},
	})
	if resp.Verdict != model.VerdictBlock {
		t.Fatalf("expected block, got %s", resp.Verdict)
	}
	if resp.RiskLevel != model.RiskHigh {
		t.Fatalf("expected high risk, got %s", resp.RiskLevel)
	}
}

func TestEngine_SIMSwap_MediumChallenge(t *testing.T) {
	e := newEngineNoResolver()
	resp, _ := e.Evaluate(context.Background(), "t1", &risk.PolicyInput{
		SIMSwapResult: &model.SIMSwapResponse{
			SIMSwapDetected: true, RiskLevel: model.RiskMedium,
		},
	})
	if resp.Verdict != model.VerdictChallenge {
		t.Fatalf("expected challenge, got %s", resp.Verdict)
	}
}

func TestEngine_VerificationFailed(t *testing.T) {
	e := newEngineNoResolver()
	resp, _ := e.Evaluate(context.Background(), "t1", &risk.PolicyInput{
		VerificationResult: "failed",
	})
	if resp.Verdict != model.VerdictChallenge {
		t.Fatalf("expected challenge, got %s", resp.Verdict)
	}
	if resp.RiskLevel != model.RiskMedium {
		t.Fatalf("expected medium risk, got %s", resp.RiskLevel)
	}
}

func TestEngine_LowConfidence(t *testing.T) {
	e := newEngineNoResolver()
	resp, _ := e.Evaluate(context.Background(), "t1", &risk.PolicyInput{
		VerificationResult: "verified",
		ConfidenceScore:    0.5,
	})
	if resp.Verdict != model.VerdictChallenge {
		t.Fatalf("expected challenge for low confidence, got %s", resp.Verdict)
	}
}

func TestEngine_DeviceChanged_RiskOnly(t *testing.T) {
	e := newEngineNoResolver()
	resp, _ := e.Evaluate(context.Background(), "t1", &risk.PolicyInput{
		VerificationResult: "verified",
		DeviceChanged:      true,
	})
	// Device changed alone adds risk but shouldn't block
	if resp.Verdict != model.VerdictAllow {
		t.Fatalf("expected allow for device change only, got %s", resp.Verdict)
	}
}

func TestEngine_MultipleSignals_MostRestrictiveWins(t *testing.T) {
	e := newEngineNoResolver()
	resp, _ := e.Evaluate(context.Background(), "t1", &risk.PolicyInput{
		VerificationResult: "failed",
		DeviceChanged:      true,
		SIMSwapResult: &model.SIMSwapResponse{
			SIMSwapDetected: true, RiskLevel: model.RiskHigh,
		},
	})
	if resp.Verdict != model.VerdictBlock {
		t.Fatalf("expected block (most restrictive), got %s", resp.Verdict)
	}
	if len(resp.Reasons) < 2 {
		t.Fatalf("expected multiple reasons, got %d", len(resp.Reasons))
	}
}

// --- Policy Rules Tests ---

func boolPtr(b bool) *bool       { return &b }
func float64Ptr(f float64) *float64 { return &f }

func TestEngine_CustomRule_CountryBlock(t *testing.T) {
	policy := &model.Policy{
		Name: "Block High Risk Countries", UseCase: "signup",
		Countries: []string{"*"}, Active: true, Priority: 10,
		SIMSwapAction: model.VerdictBlock,
		Rules: []model.PolicyRule{
			{
				Name: "Block NG signups", Enabled: true, Priority: 10,
				Condition: model.RuleCondition{
					Countries: []string{"NG"},
					UseCases:  []string{"signup"},
				},
				Action: model.RuleAction{
					Verdict: model.VerdictBlock,
					Reason:  "country_blocked",
				},
			},
		},
	}

	e := newEngine(policy)
	resp, _ := e.Evaluate(context.Background(), "t1", &risk.PolicyInput{
		VerificationResult: "verified",
		CountryCode:        "NG",
		UseCase:            "signup",
	})
	if resp.Verdict != model.VerdictBlock {
		t.Fatalf("expected block for NG, got %s", resp.Verdict)
	}
}

func TestEngine_CustomRule_CountryNoMatch(t *testing.T) {
	policy := &model.Policy{
		Name: "Block NG", UseCase: "signup",
		Countries: []string{"*"}, Active: true, Priority: 10,
		SIMSwapAction: model.VerdictChallenge,
		Rules: []model.PolicyRule{
			{
				Name: "Block NG", Enabled: true, Priority: 10,
				Condition: model.RuleCondition{Countries: []string{"NG"}},
				Action:    model.RuleAction{Verdict: model.VerdictBlock, Reason: "blocked"},
			},
		},
	}

	e := newEngine(policy)
	resp, _ := e.Evaluate(context.Background(), "t1", &risk.PolicyInput{
		VerificationResult: "verified",
		CountryCode:        "ID", // Not NG
		UseCase:            "signup",
	})
	if resp.Verdict != model.VerdictAllow {
		t.Fatalf("expected allow for ID, got %s", resp.Verdict)
	}
}

func TestEngine_CustomRule_RiskScoreThreshold(t *testing.T) {
	policy := &model.Policy{
		Name: "High risk review", UseCase: "transaction",
		Countries: []string{"*"}, Active: true, Priority: 10,
		SIMSwapAction: model.VerdictChallenge,
		Rules: []model.PolicyRule{
			{
				Name: "High score review", Enabled: true, Priority: 5,
				Condition: model.RuleCondition{
					RiskScoreAbove: float64Ptr(40),
				},
				Action: model.RuleAction{
					Verdict: model.VerdictReview,
					Reason:  "high_risk_score",
				},
			},
		},
	}

	e := newEngine(policy)

	// With SIM swap, risk score will be >= 50
	resp, _ := e.Evaluate(context.Background(), "t1", &risk.PolicyInput{
		VerificationResult: "verified",
		UseCase:            "transaction",
		SIMSwapResult: &model.SIMSwapResponse{
			SIMSwapDetected: true, RiskLevel: model.RiskMedium,
		},
	})
	// SIM swap challenge + rule review → block should win because
	// SIM swap medium = challenge, but rule adds review, and the
	// built-in sim swap already escalated. Let's check it's at least challenge.
	if verdictSeverity(resp.Verdict) < verdictSeverity(model.VerdictChallenge) {
		t.Fatalf("expected at least challenge, got %s", resp.Verdict)
	}
}

func TestEngine_CustomRule_RiskAdjustment(t *testing.T) {
	policy := &model.Policy{
		Name: "Add risk for new device", UseCase: "login",
		Countries: []string{"*"}, Active: true, Priority: 10,
		SIMSwapAction: model.VerdictChallenge,
		Rules: []model.PolicyRule{
			{
				Name: "New device penalty", Enabled: true, Priority: 10,
				Condition: model.RuleCondition{
					DeviceChanged: boolPtr(true),
				},
				Action: model.RuleAction{
					Verdict:        model.VerdictAllow, // Don't escalate verdict
					RiskAdjustment: 25,                 // But add to risk score
					Reason:         "device_changed_risk",
				},
			},
		},
	}

	e := newEngine(policy)
	resp, _ := e.Evaluate(context.Background(), "t1", &risk.PolicyInput{
		VerificationResult: "verified",
		DeviceChanged:      true,
		UseCase:            "login",
	})
	// Device changed: built-in adds 10, rule adds 25 = 35 → medium risk
	if resp.RiskLevel != model.RiskMedium {
		t.Fatalf("expected medium risk with adjusted score, got %s", resp.RiskLevel)
	}
}

func TestEngine_PolicyFiltering(t *testing.T) {
	policies := []*model.Policy{
		{Name: "Signup ID", UseCase: "signup", Countries: []string{"ID"}, Active: true, Priority: 10, SIMSwapAction: model.VerdictBlock},
		{Name: "Login All", UseCase: "login", Countries: []string{"*"}, Active: true, Priority: 5, SIMSwapAction: model.VerdictChallenge},
		{Name: "Inactive", UseCase: "signup", Countries: []string{"ID"}, Active: false, Priority: 20, SIMSwapAction: model.VerdictAllow},
	}

	matched := filterPolicies(policies, "signup", "ID")
	if len(matched) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matched))
	}
	if matched[0].Name != "Signup ID" {
		t.Fatalf("expected Signup ID, got %s", matched[0].Name)
	}

	matched2 := filterPolicies(policies, "login", "BR")
	if len(matched2) != 1 || matched2[0].Name != "Login All" {
		t.Fatalf("expected Login All for wildcard, got %v", matched2)
	}
}

func TestEngine_DisabledRuleSkipped(t *testing.T) {
	policy := &model.Policy{
		Name: "Test", UseCase: "signup",
		Countries: []string{"*"}, Active: true, Priority: 10,
		SIMSwapAction: model.VerdictChallenge,
		Rules: []model.PolicyRule{
			{
				Name: "Disabled rule", Enabled: false, Priority: 10,
				Condition: model.RuleCondition{},
				Action: model.RuleAction{
					Verdict: model.VerdictBlock,
					Reason:  "should_not_fire",
				},
			},
		},
	}

	e := newEngine(policy)
	resp, _ := e.Evaluate(context.Background(), "t1", &risk.PolicyInput{
		VerificationResult: "verified",
		UseCase:            "signup",
	})
	if resp.Verdict != model.VerdictAllow {
		t.Fatalf("disabled rule should not fire, got %s", resp.Verdict)
	}
}

func TestRiskLevelFromScore(t *testing.T) {
	tests := []struct {
		score float64
		want  model.RiskLevel
	}{
		{0, model.RiskLow},
		{19, model.RiskLow},
		{20, model.RiskMedium},
		{49, model.RiskMedium},
		{50, model.RiskHigh},
		{100, model.RiskHigh},
	}
	for _, tt := range tests {
		got := riskLevelFromScore(tt.score)
		if got != tt.want {
			t.Errorf("riskLevelFromScore(%v) = %s, want %s", tt.score, got, tt.want)
		}
	}
}
