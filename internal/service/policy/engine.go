package policy

import (
	"context"
	"sort"
	"time"

	"github.com/silentpass/silentpass/internal/model"
	"github.com/silentpass/silentpass/internal/service/risk"
)

// PolicyResolver loads policies for a tenant.
type PolicyResolver interface {
	List(ctx context.Context, tenantID string) ([]*model.Policy, error)
}

// Engine evaluates risk signals against configurable policy rules.
// Evaluation order:
//  1. Find matching policies by use_case + country
//  2. Sort by priority (descending)
//  3. Evaluate built-in signals (SIM swap, verification result)
//  4. Evaluate custom rules in priority order
//  5. Most restrictive verdict wins
type Engine struct {
	resolver PolicyResolver
}

func NewEngine(resolver PolicyResolver) *Engine {
	return &Engine{resolver: resolver}
}

func (e *Engine) Evaluate(ctx context.Context, tenantID string, input *risk.PolicyInput) (*model.VerdictResponse, error) {
	// Step 1: Compute base risk score from signals
	result := &evaluationResult{
		riskScore: input.RiskScore,
		verdict:   model.VerdictAllow,
		riskLevel: model.RiskLow,
	}

	// Step 2: Built-in signal evaluation
	e.evaluateBuiltinSignals(input, result)

	// Step 3: Load and evaluate policy rules if resolver available
	if e.resolver != nil {
		policies, err := e.resolver.List(ctx, tenantID)
		if err == nil {
			matching := filterPolicies(policies, input.UseCase, input.CountryCode)
			e.evaluatePolicyRules(matching, input, result)
		}
	}

	// Step 4: Determine final risk level from score
	result.riskLevel = riskLevelFromScore(result.riskScore)

	// Step 5: Build response
	action := actionFromVerdict(result.verdict)

	return &model.VerdictResponse{
		Verdict:        result.verdict,
		RiskLevel:      result.riskLevel,
		Reasons:        result.reasons,
		ActionRequired: action,
	}, nil
}

type evaluationResult struct {
	riskScore float64
	verdict   model.Verdict
	riskLevel model.RiskLevel
	reasons   []string
}

// escalate only increases severity, never decreases.
func (r *evaluationResult) escalate(v model.Verdict, reason string) {
	if verdictSeverity(v) > verdictSeverity(r.verdict) {
		r.verdict = v
	}
	if reason != "" {
		r.reasons = append(r.reasons, reason)
	}
}

func (r *evaluationResult) addRisk(score float64, reason string) {
	r.riskScore += score
	if reason != "" {
		r.reasons = append(r.reasons, reason)
	}
}

// evaluateBuiltinSignals checks hardcoded risk signals.
func (e *Engine) evaluateBuiltinSignals(input *risk.PolicyInput, result *evaluationResult) {
	// SIM Swap
	if input.SIMSwapResult != nil && input.SIMSwapResult.SIMSwapDetected {
		result.addRisk(50, "sim_swap_detected")

		switch input.SIMSwapResult.RiskLevel {
		case model.RiskHigh:
			result.escalate(model.VerdictBlock, "sim_swap_high_risk")
		case model.RiskMedium:
			result.escalate(model.VerdictChallenge, "sim_swap_medium_risk")
		default:
			result.escalate(model.VerdictChallenge, "sim_swap_low_risk")
		}
	}

	// Verification failed
	if input.VerificationResult == "failed" {
		result.addRisk(20, "verification_failed")
		result.escalate(model.VerdictChallenge, "")
	}

	// Low confidence
	if input.ConfidenceScore > 0 && input.ConfidenceScore < 0.7 {
		result.addRisk(15, "low_confidence_score")
		result.escalate(model.VerdictChallenge, "")
	}

	// Device changed
	if input.DeviceChanged {
		result.addRisk(10, "device_changed")
	}
}

// evaluatePolicyRules evaluates custom rules from matching policies.
func (e *Engine) evaluatePolicyRules(policies []*model.Policy, input *risk.PolicyInput, result *evaluationResult) {
	for _, p := range policies {
		// Apply policy-level SIM swap action
		if input.SIMSwapResult != nil && input.SIMSwapResult.SIMSwapDetected {
			result.escalate(p.SIMSwapAction, "policy_sim_swap_action:"+p.Name)
		}

		// Evaluate custom rules
		rules := p.Rules
		sort.Slice(rules, func(i, j int) bool {
			return rules[i].Priority > rules[j].Priority
		})

		for _, rule := range rules {
			if !rule.Enabled {
				continue
			}
			if matchesCondition(&rule.Condition, input, result.riskScore) {
				result.addRisk(rule.Action.RiskAdjustment, "rule:"+rule.Name)
				result.escalate(rule.Action.Verdict, rule.Action.Reason)
			}
		}
	}
}

// matchesCondition checks if all non-empty condition fields match.
func matchesCondition(cond *RuleCondition, input *risk.PolicyInput, currentRiskScore float64) bool {
	c := (*model.RuleCondition)(cond)

	if len(c.Countries) > 0 && !containsStr(c.Countries, input.CountryCode) {
		return false
	}
	if len(c.Operators) > 0 && !containsStr(c.Operators, input.Operator) {
		return false
	}
	if len(c.UseCases) > 0 && !containsStr(c.UseCases, input.UseCase) {
		return false
	}
	if len(c.Channels) > 0 && !containsStr(c.Channels, input.VerificationMethod) {
		return false
	}

	if c.SIMSwapDetected != nil {
		swapped := input.SIMSwapResult != nil && input.SIMSwapResult.SIMSwapDetected
		if swapped != *c.SIMSwapDetected {
			return false
		}
	}

	if c.VerificationFailed != nil {
		failed := input.VerificationResult == "failed"
		if failed != *c.VerificationFailed {
			return false
		}
	}

	if c.ConfidenceBelow != nil {
		if input.ConfidenceScore >= *c.ConfidenceBelow {
			return false
		}
	}

	if c.DeviceChanged != nil {
		if input.DeviceChanged != *c.DeviceChanged {
			return false
		}
	}

	if c.RiskScoreAbove != nil {
		if currentRiskScore <= *c.RiskScoreAbove {
			return false
		}
	}

	if c.HourRange != nil {
		hour := time.Now().UTC().Hour()
		start, end := c.HourRange[0], c.HourRange[1]
		if start <= end {
			if hour < start || hour >= end {
				return false
			}
		} else {
			// Wraps midnight, e.g. [22, 6]
			if hour < start && hour >= end {
				return false
			}
		}
	}

	return true
}

// RuleCondition is an alias to avoid import cycle.
type RuleCondition = model.RuleCondition

// filterPolicies returns active policies matching use case and country.
func filterPolicies(policies []*model.Policy, useCase, countryCode string) []*model.Policy {
	var matched []*model.Policy
	for _, p := range policies {
		if !p.Active {
			continue
		}
		if useCase != "" && string(p.UseCase) != useCase {
			continue
		}
		if countryCode != "" && len(p.Countries) > 0 {
			if !containsStr(p.Countries, countryCode) && !containsStr(p.Countries, "*") {
				continue
			}
		}
		matched = append(matched, p)
	}
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].Priority > matched[j].Priority
	})
	return matched
}

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func verdictSeverity(v model.Verdict) int {
	switch v {
	case model.VerdictAllow:
		return 0
	case model.VerdictChallenge:
		return 1
	case model.VerdictReview:
		return 2
	case model.VerdictBlock:
		return 3
	}
	return 0
}

func riskLevelFromScore(score float64) model.RiskLevel {
	switch {
	case score >= 50:
		return model.RiskHigh
	case score >= 20:
		return model.RiskMedium
	default:
		return model.RiskLow
	}
}

func actionFromVerdict(v model.Verdict) string {
	switch v {
	case model.VerdictChallenge:
		return "require_otp"
	case model.VerdictBlock:
		return "deny_operation"
	case model.VerdictReview:
		return "manual_review"
	default:
		return "proceed"
	}
}
