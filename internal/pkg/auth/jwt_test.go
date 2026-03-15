package auth

import (
	"testing"
	"time"
)

func TestGenerateAndValidate(t *testing.T) {
	svc := NewTokenService("test-secret-key-32-chars-long!!", 5*time.Minute)

	token, err := svc.Generate("sess-123", "tenant-456", "hash-789", "signup", "silent")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if token == "" {
		t.Fatal("token is empty")
	}

	claims, err := svc.Validate(token)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if claims.SessionID != "sess-123" {
		t.Fatalf("session_id: got %s", claims.SessionID)
	}
	if claims.TenantID != "tenant-456" {
		t.Fatalf("tenant_id: got %s", claims.TenantID)
	}
	if claims.UseCase != "signup" {
		t.Fatalf("use_case: got %s", claims.UseCase)
	}
	if claims.Method != "silent" {
		t.Fatalf("method: got %s", claims.Method)
	}
}

func TestExpiredToken(t *testing.T) {
	svc := NewTokenService("test-secret", 1*time.Millisecond)

	token, err := svc.Generate("sess", "tenant", "hash", "login", "sms")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	_, err = svc.Validate(token)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestInvalidSecret(t *testing.T) {
	svc1 := NewTokenService("secret-1", 5*time.Minute)
	svc2 := NewTokenService("secret-2", 5*time.Minute)

	token, _ := svc1.Generate("sess", "tenant", "hash", "login", "silent")

	_, err := svc2.Validate(token)
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestMalformedToken(t *testing.T) {
	svc := NewTokenService("secret", 5*time.Minute)

	_, err := svc.Validate("not-a-valid-jwt")
	if err == nil {
		t.Fatal("expected error for malformed token")
	}
}
