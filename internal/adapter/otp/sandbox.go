package otp

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// SandboxProvider simulates OTP send/verify for development.
// Codes are stored in memory and printed to stdout.
type SandboxProvider struct {
	mu    sync.RWMutex
	codes map[string]string // phoneHash -> code
}

func NewSandboxProvider() *SandboxProvider {
	return &SandboxProvider{
		codes: make(map[string]string),
	}
}

func (p *SandboxProvider) Name() string {
	return "sandbox_otp"
}

func (p *SandboxProvider) SupportedChannels() []string {
	return []string{"sms", "whatsapp", "voice"}
}

func (p *SandboxProvider) Send(ctx context.Context, phoneNumber, channel, locale string) error {
	time.Sleep(time.Duration(50+rand.Intn(100)) * time.Millisecond)

	code := fmt.Sprintf("%06d", rand.Intn(1000000))

	p.mu.Lock()
	p.codes[phoneNumber] = code
	p.mu.Unlock()

	// In sandbox mode, log the code for testing
	fmt.Printf("[SANDBOX OTP] phone=%s channel=%s code=%s\n", phoneNumber[:8]+"...", channel, code)
	return nil
}

func (p *SandboxProvider) Verify(ctx context.Context, phoneNumber, code string) (bool, error) {
	p.mu.RLock()
	stored, ok := p.codes[phoneNumber]
	p.mu.RUnlock()

	if !ok {
		return false, nil
	}

	// Sandbox: also accept "000000" as universal test code
	if code == stored || code == "000000" {
		p.mu.Lock()
		delete(p.codes, phoneNumber)
		p.mu.Unlock()
		return true, nil
	}

	return false, nil
}
