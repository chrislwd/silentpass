package otp

import "context"

// Provider is the interface for OTP delivery providers (SMS, WhatsApp, Voice).
type Provider interface {
	Name() string
	Send(ctx context.Context, phoneNumber, channel, locale string) error
	Verify(ctx context.Context, phoneNumber, code string) (bool, error)
	SupportedChannels() []string
}

// Router selects the appropriate OTP provider based on channel and country.
type Router struct {
	providers map[string]Provider // channel -> provider
}

func NewRouter() *Router {
	return &Router{
		providers: make(map[string]Provider),
	}
}

func (r *Router) Register(channel string, provider Provider) {
	r.providers[channel] = provider
}

func (r *Router) Send(ctx context.Context, phoneNumber, channel, locale string) error {
	p, ok := r.providers[channel]
	if !ok {
		// Fallback to SMS
		p, ok = r.providers["sms"]
		if !ok {
			return ErrNoProvider
		}
	}
	return p.Send(ctx, phoneNumber, channel, locale)
}

func (r *Router) Verify(ctx context.Context, phoneNumber, code string) (bool, error) {
	// OTP verification is provider-agnostic; check against stored code
	for _, p := range r.providers {
		ok, err := p.Verify(ctx, phoneNumber, code)
		if err == nil && ok {
			return true, nil
		}
	}
	return false, nil
}

var ErrNoProvider = &ProviderError{Message: "no OTP provider available"}

type ProviderError struct {
	Message string
}

func (e *ProviderError) Error() string {
	return e.Message
}
