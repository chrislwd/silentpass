package verification

import (
	"context"
	"testing"

	"github.com/silentpass/silentpass/internal/model"
)

type mockSessionRepo struct {
	sessions map[string]*model.Session
}

func newMockSessionRepo() *mockSessionRepo {
	return &mockSessionRepo{sessions: make(map[string]*model.Session)}
}

func (m *mockSessionRepo) CreateSession(ctx context.Context, s *model.Session) error {
	m.sessions[s.ID] = s
	return nil
}

func (m *mockSessionRepo) GetSession(ctx context.Context, id string) (*model.Session, error) {
	s, ok := m.sessions[id]
	if !ok {
		return nil, &notFoundError{}
	}
	return s, nil
}

func (m *mockSessionRepo) UpdateSessionStatus(ctx context.Context, id string, status model.SessionStatus) error {
	if s, ok := m.sessions[id]; ok {
		s.Status = status
	}
	return nil
}

type notFoundError struct{}

func (e *notFoundError) Error() string { return "not found" }

type mockTelco struct {
	supported bool
	result    model.VerificationResult
}

func (m *mockTelco) SilentVerify(ctx context.Context, phoneHash, countryCode string) (*model.SilentVerifyResponse, error) {
	return &model.SilentVerifyResponse{
		Status:          m.result,
		ConfidenceScore: 0.98,
		TelcoSignal:     "match",
		Token:           "test-token",
	}, nil
}

func (m *mockTelco) IsSupported(countryCode string) bool {
	return m.supported
}

type mockOTP struct{}

func (m *mockOTP) Send(ctx context.Context, phone, channel, locale string) error { return nil }
func (m *mockOTP) Verify(ctx context.Context, phone, code string) (bool, error) {
	return code == "123456", nil
}

func TestCreateSession(t *testing.T) {
	svc := NewService(newMockSessionRepo(), &mockTelco{supported: true}, &mockOTP{})

	resp, err := svc.CreateSession(context.Background(), "tenant-1", &model.CreateSessionRequest{
		AppID:            "app1",
		PhoneNumber:      "+6281234567890",
		CountryCode:      "ID",
		VerificationType: model.VerificationSilentOrOTP,
		UseCase:          model.UseCaseSignup,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.SessionID == "" {
		t.Fatal("session ID should not be empty")
	}
	if resp.RecommendedAction != "silent_verify" {
		t.Fatalf("expected silent_verify, got %s", resp.RecommendedAction)
	}
}

func TestCreateSession_UnsupportedCountry(t *testing.T) {
	svc := NewService(newMockSessionRepo(), &mockTelco{supported: false}, &mockOTP{})

	resp, err := svc.CreateSession(context.Background(), "tenant-1", &model.CreateSessionRequest{
		AppID:            "app1",
		PhoneNumber:      "+1234567890",
		CountryCode:      "US",
		VerificationType: model.VerificationSilentOrOTP,
		UseCase:          model.UseCaseSignup,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.RecommendedAction != "otp" {
		t.Fatalf("expected otp for unsupported country, got %s", resp.RecommendedAction)
	}
}

func TestSilentVerify_Success(t *testing.T) {
	repo := newMockSessionRepo()
	svc := NewService(repo, &mockTelco{supported: true, result: model.ResultVerified}, &mockOTP{})

	resp, err := svc.CreateSession(context.Background(), "tenant-1", &model.CreateSessionRequest{
		AppID:            "app1",
		PhoneNumber:      "+6281234567890",
		CountryCode:      "ID",
		VerificationType: model.VerificationSilent,
		UseCase:          model.UseCaseLogin,
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	verifyResp, err := svc.SilentVerify(context.Background(), "tenant-1", &model.SilentVerifyRequest{
		SessionID: resp.SessionID,
	})
	if err != nil {
		t.Fatalf("silent verify: %v", err)
	}
	if verifyResp.Status != model.ResultVerified {
		t.Fatalf("expected verified, got %s", verifyResp.Status)
	}
	if verifyResp.Token == "" {
		t.Fatal("token should not be empty on verified")
	}
}

func TestSilentVerify_Fallback(t *testing.T) {
	repo := newMockSessionRepo()
	svc := NewService(repo, &mockTelco{supported: false}, &mockOTP{})

	resp, err := svc.CreateSession(context.Background(), "tenant-1", &model.CreateSessionRequest{
		AppID:            "app1",
		PhoneNumber:      "+6281234567890",
		CountryCode:      "XX",
		VerificationType: model.VerificationSilentOrOTP,
		UseCase:          model.UseCaseSignup,
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	verifyResp, err := svc.SilentVerify(context.Background(), "tenant-1", &model.SilentVerifyRequest{
		SessionID: resp.SessionID,
	})
	if err != nil {
		t.Fatalf("silent verify: %v", err)
	}
	if verifyResp.Status != model.ResultFallbackRequired {
		t.Fatalf("expected fallback_required, got %s", verifyResp.Status)
	}
}

func TestOTPFlow(t *testing.T) {
	repo := newMockSessionRepo()
	svc := NewService(repo, &mockTelco{supported: false}, &mockOTP{})

	resp, err := svc.CreateSession(context.Background(), "tenant-1", &model.CreateSessionRequest{
		AppID:            "app1",
		PhoneNumber:      "+6281234567890",
		CountryCode:      "ID",
		VerificationType: model.VerificationOTPOnly,
		UseCase:          model.UseCaseSignup,
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	// Send OTP
	sendResp, err := svc.SendOTP(context.Background(), "tenant-1", &model.OTPSendRequest{
		SessionID: resp.SessionID,
		Channel:   "sms",
	})
	if err != nil {
		t.Fatalf("send OTP: %v", err)
	}
	if sendResp.DeliveryStatus != "sent" {
		t.Fatalf("expected sent, got %s", sendResp.DeliveryStatus)
	}

	// Check OTP - wrong code
	checkResp, err := svc.CheckOTP(context.Background(), "tenant-1", &model.OTPCheckRequest{
		SessionID: resp.SessionID,
		Code:      "000000",
	})
	if err != nil {
		t.Fatalf("check OTP: %v", err)
	}
	if checkResp.Status != model.ResultFailed {
		t.Fatalf("expected failed for wrong code, got %s", checkResp.Status)
	}

	// Check OTP - correct code
	checkResp, err = svc.CheckOTP(context.Background(), "tenant-1", &model.OTPCheckRequest{
		SessionID: resp.SessionID,
		Code:      "123456",
	})
	if err != nil {
		t.Fatalf("check OTP: %v", err)
	}
	if checkResp.Status != model.ResultVerified {
		t.Fatalf("expected verified, got %s", checkResp.Status)
	}
}
