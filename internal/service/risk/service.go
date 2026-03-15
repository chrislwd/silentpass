package risk

import (
	"context"
	"fmt"

	"github.com/silentpass/silentpass/internal/model"
)

type SIMSwapAdapter interface {
	CheckSIMSwap(ctx context.Context, phoneNumber, countryCode string) (*model.SIMSwapResponse, error)
}

type PolicyEngine interface {
	Evaluate(ctx context.Context, tenantID string, input *PolicyInput) (*model.VerdictResponse, error)
}

type PolicyInput struct {
	VerificationResult string
	SIMSwapResult      *model.SIMSwapResponse
	DeviceStatus       map[string]interface{}
	PolicyID           string
}

type Service struct {
	simSwap SIMSwapAdapter
	policy  PolicyEngine
}

func NewService(simSwap SIMSwapAdapter, policy PolicyEngine) *Service {
	return &Service{
		simSwap: simSwap,
		policy:  policy,
	}
}

func (s *Service) CheckSIMSwap(ctx context.Context, tenantID string, req *model.SIMSwapRequest) (*model.SIMSwapResponse, error) {
	resp, err := s.simSwap.CheckSIMSwap(ctx, req.PhoneNumber, req.CountryCode)
	if err != nil {
		return nil, fmt.Errorf("sim swap check: %w", err)
	}
	return resp, nil
}

func (s *Service) EvaluateVerdict(ctx context.Context, tenantID string, req *model.VerdictRequest) (*model.VerdictResponse, error) {
	input := &PolicyInput{
		VerificationResult: req.VerificationResult,
		SIMSwapResult:      req.SIMSwapResult,
		DeviceStatus:       req.DeviceStatus,
		PolicyID:           req.PolicyID,
	}

	resp, err := s.policy.Evaluate(ctx, tenantID, input)
	if err != nil {
		return nil, fmt.Errorf("evaluate verdict: %w", err)
	}
	return resp, nil
}
