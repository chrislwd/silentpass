package verification

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/silentpass/silentpass/internal/model"
	"github.com/silentpass/silentpass/internal/pkg/auth"
)

type SessionRepository interface {
	CreateSession(ctx context.Context, session *model.Session) error
	GetSession(ctx context.Context, sessionID string) (*model.Session, error)
	UpdateSessionStatus(ctx context.Context, sessionID string, status model.SessionStatus) error
}

type TelcoAdapter interface {
	SilentVerify(ctx context.Context, phoneHash, countryCode string) (*model.SilentVerifyResponse, error)
	IsSupported(countryCode string) bool
}

type OTPAdapter interface {
	Send(ctx context.Context, phoneNumber, channel, locale string) error
	Verify(ctx context.Context, phoneNumber, code string) (bool, error)
}

type Service struct {
	sessions SessionRepository
	telco    TelcoAdapter
	otp      OTPAdapter
	tokens   *auth.TokenService
}

func NewService(sessions SessionRepository, telco TelcoAdapter, otp OTPAdapter) *Service {
	return &Service{
		sessions: sessions,
		telco:    telco,
		otp:      otp,
	}
}

func (s *Service) SetTokenService(ts *auth.TokenService) {
	s.tokens = ts
}

func (s *Service) generateToken(session *model.Session, method string) string {
	if s.tokens != nil {
		token, err := s.tokens.Generate(session.ID, session.TenantID, session.PhoneHash, string(session.UseCase), method)
		if err == nil {
			return token
		}
	}
	// Fallback to UUID token
	return "sv_" + uuid.New().String()
}

func (s *Service) CreateSession(ctx context.Context, tenantID string, req *model.CreateSessionRequest) (*model.CreateSessionResponse, error) {
	now := time.Now()
	sessionID := uuid.New().String()

	session := &model.Session{
		ID:               sessionID,
		TenantID:         tenantID,
		PhoneHash:        hashPhone(req.PhoneNumber),
		CountryCode:      req.CountryCode,
		VerificationType: req.VerificationType,
		UseCase:          req.UseCase,
		Status:           model.SessionPending,
		CreatedAt:        now,
		ExpiresAt:        now.Add(10 * time.Minute),
		UpdatedAt:        now,
	}

	if req.DeviceContext != nil {
		session.DeviceIP = req.DeviceContext.IPAddress
		session.UserAgent = req.DeviceContext.UserAgent
	}
	if req.CallbackURL != "" {
		session.CallbackURL = req.CallbackURL
	}

	if err := s.sessions.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	action := s.recommendAction(req.VerificationType, req.CountryCode)

	return &model.CreateSessionResponse{
		SessionID:         sessionID,
		RecommendedAction: action,
		ExpiresAt:         session.ExpiresAt.Format(time.RFC3339),
	}, nil
}

func (s *Service) SilentVerify(ctx context.Context, tenantID string, req *model.SilentVerifyRequest) (*model.SilentVerifyResponse, error) {
	session, err := s.sessions.GetSession(ctx, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if session.TenantID != tenantID {
		return nil, fmt.Errorf("session not found")
	}

	if time.Now().After(session.ExpiresAt) {
		_ = s.sessions.UpdateSessionStatus(ctx, session.ID, model.SessionExpired)
		return &model.SilentVerifyResponse{Status: model.ResultFailed}, nil
	}

	if !s.telco.IsSupported(session.CountryCode) {
		return &model.SilentVerifyResponse{Status: model.ResultFallbackRequired}, nil
	}

	resp, err := s.telco.SilentVerify(ctx, session.PhoneHash, session.CountryCode)
	if err != nil {
		return &model.SilentVerifyResponse{Status: model.ResultFallbackRequired}, nil
	}

	if resp.Status == model.ResultVerified {
		_ = s.sessions.UpdateSessionStatus(ctx, session.ID, model.SessionVerified)
		resp.Token = s.generateToken(session, "silent")
	}

	return resp, nil
}

func (s *Service) SendOTP(ctx context.Context, tenantID string, req *model.OTPSendRequest) (*model.OTPSendResponse, error) {
	session, err := s.sessions.GetSession(ctx, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	if session.TenantID != tenantID {
		return nil, fmt.Errorf("session not found")
	}

	if err := s.otp.Send(ctx, session.PhoneHash, req.Channel, req.Locale); err != nil {
		return nil, fmt.Errorf("send otp: %w", err)
	}

	return &model.OTPSendResponse{
		DeliveryStatus:     "sent",
		ResendAfterSeconds: 60,
	}, nil
}

func (s *Service) CheckOTP(ctx context.Context, tenantID string, req *model.OTPCheckRequest) (*model.OTPCheckResponse, error) {
	session, err := s.sessions.GetSession(ctx, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	if session.TenantID != tenantID {
		return nil, fmt.Errorf("session not found")
	}

	ok, err := s.otp.Verify(ctx, session.PhoneHash, req.Code)
	if err != nil {
		return nil, fmt.Errorf("verify otp: %w", err)
	}

	if ok {
		_ = s.sessions.UpdateSessionStatus(ctx, session.ID, model.SessionVerified)
		return &model.OTPCheckResponse{
			Status:       model.ResultVerified,
			Token:        s.generateToken(session, "otp"),
			AttemptsLeft: 0,
		}, nil
	}

	return &model.OTPCheckResponse{
		Status:       model.ResultFailed,
		AttemptsLeft: 2,
	}, nil
}

func (s *Service) recommendAction(vType model.VerificationType, countryCode string) string {
	switch vType {
	case model.VerificationSilent:
		if s.telco.IsSupported(countryCode) {
			return "silent_verify"
		}
		return "otp"
	case model.VerificationSilentOrOTP:
		if s.telco.IsSupported(countryCode) {
			return "silent_verify"
		}
		return "otp"
	default:
		return "otp"
	}
}

func hashPhone(phone string) string {
	h := sha256.Sum256([]byte(phone))
	return fmt.Sprintf("%x", h)
}
