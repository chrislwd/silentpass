package crypto

import (
	"strings"
	"testing"
)

func TestHashAndCheckPassword(t *testing.T) {
	password := "mySecureP@ssw0rd"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if hash == password {
		t.Fatal("hash should not equal plaintext")
	}
	if !CheckPassword(password, hash) {
		t.Fatal("correct password should match")
	}
	if CheckPassword("wrongpassword", hash) {
		t.Fatal("wrong password should not match")
	}
}

func TestGenerateAPIKey(t *testing.T) {
	key, hash, err := GenerateAPIKey("sk_test_")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if !strings.HasPrefix(key, "sk_test_") {
		t.Fatalf("key should start with prefix, got %s", key[:10])
	}
	if len(key) < 40 {
		t.Fatalf("key too short: %d", len(key))
	}
	if hash == "" {
		t.Fatal("hash should not be empty")
	}
	if HashAPIKey(key) != hash {
		t.Fatal("hash should match")
	}
}

func TestGenerateAPIKey_Unique(t *testing.T) {
	key1, _, _ := GenerateAPIKey("sk_")
	key2, _, _ := GenerateAPIKey("sk_")
	if key1 == key2 {
		t.Fatal("keys should be unique")
	}
}

func TestGenerateSecret(t *testing.T) {
	s1, err := GenerateSecret(16)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(s1) != 32 { // 16 bytes = 32 hex chars
		t.Fatalf("expected 32 chars, got %d", len(s1))
	}
	s2, _ := GenerateSecret(16)
	if s1 == s2 {
		t.Fatal("secrets should be unique")
	}
}
