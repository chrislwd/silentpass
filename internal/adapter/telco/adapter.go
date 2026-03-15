package telco

import (
	"context"
	"fmt"

	"github.com/silentpass/silentpass/internal/model"
)

// Adapter is the interface for telco/channel partner integrations.
// Each upstream provider implements this interface.
type Adapter interface {
	Name() string
	SilentVerify(ctx context.Context, phoneHash, countryCode string) (*model.SilentVerifyResponse, error)
	CheckSIMSwap(ctx context.Context, phoneNumber, countryCode string) (*model.SIMSwapResponse, error)
	SupportedCountries() []string
	SupportedCapabilities() []string
}

// Router manages multiple telco adapters and routes requests
// to the appropriate upstream based on country and capability.
type Router struct {
	adapters       []Adapter
	countryMapping map[string]Adapter // countryCode -> preferred adapter
}

func NewRouter() *Router {
	return &Router{
		adapters:       make([]Adapter, 0),
		countryMapping: make(map[string]Adapter),
	}
}

func (r *Router) Register(adapter Adapter) {
	r.adapters = append(r.adapters, adapter)
	for _, country := range adapter.SupportedCountries() {
		r.countryMapping[country] = adapter
	}
}

func (r *Router) IsSupported(countryCode string) bool {
	_, ok := r.countryMapping[countryCode]
	return ok
}

func (r *Router) SilentVerify(ctx context.Context, phoneHash, countryCode string) (*model.SilentVerifyResponse, error) {
	adapter, ok := r.countryMapping[countryCode]
	if !ok {
		return nil, fmt.Errorf("no adapter for country: %s", countryCode)
	}
	return adapter.SilentVerify(ctx, phoneHash, countryCode)
}

func (r *Router) CheckSIMSwap(ctx context.Context, phoneNumber, countryCode string) (*model.SIMSwapResponse, error) {
	adapter, ok := r.countryMapping[countryCode]
	if !ok {
		return nil, fmt.Errorf("no adapter for country: %s", countryCode)
	}
	return adapter.CheckSIMSwap(ctx, phoneNumber, countryCode)
}
