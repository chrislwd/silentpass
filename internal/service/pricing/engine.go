package pricing

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// Plan defines a pricing plan with tiered rates.
type Plan struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Type           string  `json:"type"` // payg, package, enterprise
	BaseFeeMonthly int64   `json:"base_fee_monthly"` // microcents
	MinCommitment  int64   `json:"min_commitment"`   // microcents
	Rules          []Rule  `json:"rules"`
}

// Rule defines per-unit pricing for a product/country with volume tiers.
type Rule struct {
	ProductType string `json:"product_type"` // silent_verify, sms_otp, sim_swap, etc.
	CountryCode string `json:"country_code"` // ISO code or "*" for default
	Tiers       []Tier `json:"tiers"`
}

// Tier defines volume-based pricing.
type Tier struct {
	MinVolume int   `json:"min_volume"`
	MaxVolume *int  `json:"max_volume,omitempty"` // nil = unlimited
	UnitPrice int64 `json:"unit_price"`           // microcents per unit
}

// PriceResult is the computed price for a single operation.
type PriceResult struct {
	UnitPrice     int64  `json:"unit_price"`      // microcents
	UpstreamCost  int64  `json:"upstream_cost"`    // microcents
	Margin        int64  `json:"margin"`           // microcents
	TierApplied   string `json:"tier_applied"`
	DiscountPct   int    `json:"discount_pct"`
}

// Engine computes pricing based on plans, tiers, and tenant-specific discounts.
type Engine struct {
	mu       sync.RWMutex
	plans    map[string]*Plan           // plan_id -> plan
	tenants  map[string]*TenantPricing  // tenant_id -> pricing config
	upstream map[string]int64           // "product:country" -> upstream cost in microcents
}

type TenantPricing struct {
	PlanID          string
	CustomDiscount  int // percentage 0-100
	VolumeOverrides map[string]int64 // "product:country" -> custom price
}

func NewEngine() *Engine {
	e := &Engine{
		plans:    make(map[string]*Plan),
		tenants:  make(map[string]*TenantPricing),
		upstream: make(map[string]int64),
	}
	e.seedDefaults()
	return e
}

func (e *Engine) seedDefaults() {
	// Default upstream costs (microcents)
	e.upstream["silent_verify:ID"] = 15000  // $0.015
	e.upstream["silent_verify:TH"] = 18000
	e.upstream["silent_verify:PH"] = 20000
	e.upstream["silent_verify:MY"] = 16000
	e.upstream["silent_verify:SG"] = 12000
	e.upstream["sms_otp:ID"] = 25000         // $0.025
	e.upstream["sms_otp:TH"] = 30000
	e.upstream["whatsapp_otp:*"] = 35000
	e.upstream["sim_swap:*"] = 5000           // $0.005
	e.upstream["voice_otp:*"] = 45000

	// Pay-as-you-go plan
	payg := &Plan{
		ID: "plan_payg", Name: "Pay As You Go", Type: "payg",
		Rules: []Rule{
			{ProductType: "silent_verify", CountryCode: "*", Tiers: []Tier{
				{MinVolume: 0, UnitPrice: 30000},      // $0.03
			}},
			{ProductType: "sms_otp", CountryCode: "*", Tiers: []Tier{
				{MinVolume: 0, UnitPrice: 45000},       // $0.045
			}},
			{ProductType: "whatsapp_otp", CountryCode: "*", Tiers: []Tier{
				{MinVolume: 0, UnitPrice: 60000},
			}},
			{ProductType: "sim_swap", CountryCode: "*", Tiers: []Tier{
				{MinVolume: 0, UnitPrice: 10000},       // $0.01
			}},
		},
	}

	// Growth plan with volume tiers
	tiers10k := intPtr(10000)
	tiers100k := intPtr(100000)
	growth := &Plan{
		ID: "plan_growth", Name: "Growth", Type: "package",
		BaseFeeMonthly: 9900000, // $99/month
		Rules: []Rule{
			{ProductType: "silent_verify", CountryCode: "*", Tiers: []Tier{
				{MinVolume: 0, MaxVolume: tiers10k, UnitPrice: 25000},
				{MinVolume: 10001, MaxVolume: tiers100k, UnitPrice: 20000},
				{MinVolume: 100001, UnitPrice: 15000},
			}},
			{ProductType: "sms_otp", CountryCode: "*", Tiers: []Tier{
				{MinVolume: 0, MaxVolume: tiers10k, UnitPrice: 40000},
				{MinVolume: 10001, UnitPrice: 30000},
			}},
			{ProductType: "sim_swap", CountryCode: "*", Tiers: []Tier{
				{MinVolume: 0, UnitPrice: 8000},
			}},
		},
	}

	// Enterprise
	enterprise := &Plan{
		ID: "plan_enterprise", Name: "Enterprise", Type: "enterprise",
		BaseFeeMonthly: 99900000, // $999/month
		MinCommitment:  50000000, // $500 min usage
		Rules: []Rule{
			{ProductType: "silent_verify", CountryCode: "*", Tiers: []Tier{
				{MinVolume: 0, UnitPrice: 12000},
			}},
			{ProductType: "sms_otp", CountryCode: "*", Tiers: []Tier{
				{MinVolume: 0, UnitPrice: 25000},
			}},
			{ProductType: "sim_swap", CountryCode: "*", Tiers: []Tier{
				{MinVolume: 0, UnitPrice: 5000},
			}},
		},
	}

	e.plans["plan_payg"] = payg
	e.plans["plan_growth"] = growth
	e.plans["plan_enterprise"] = enterprise
}

// CalculatePrice computes the price for a single API call.
func (e *Engine) CalculatePrice(_ context.Context, tenantID, productType, countryCode string, currentVolume int) *PriceResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	tp := e.tenants[tenantID]
	planID := "plan_payg"
	discount := 0
	if tp != nil {
		planID = tp.PlanID
		discount = tp.CustomDiscount

		// Check for volume override
		overrideKey := productType + ":" + countryCode
		if price, ok := tp.VolumeOverrides[overrideKey]; ok {
			upstream := e.getUpstreamCost(productType, countryCode)
			return &PriceResult{
				UnitPrice: price, UpstreamCost: upstream,
				Margin: price - upstream, TierApplied: "custom",
				DiscountPct: discount,
			}
		}
	}

	plan := e.plans[planID]
	if plan == nil {
		plan = e.plans["plan_payg"]
	}

	unitPrice := e.findPrice(plan, productType, countryCode, currentVolume)

	if discount > 0 {
		unitPrice = unitPrice * int64(100-discount) / 100
	}

	upstream := e.getUpstreamCost(productType, countryCode)

	return &PriceResult{
		UnitPrice:    unitPrice,
		UpstreamCost: upstream,
		Margin:       unitPrice - upstream,
		TierApplied:  plan.Name,
		DiscountPct:  discount,
	}
}

func (e *Engine) findPrice(plan *Plan, productType, countryCode string, volume int) int64 {
	// Find matching rule: exact country first, then wildcard
	var matchedRule *Rule
	for i := range plan.Rules {
		r := &plan.Rules[i]
		if r.ProductType != productType {
			continue
		}
		if r.CountryCode == countryCode {
			matchedRule = r
			break
		}
		if r.CountryCode == "*" && matchedRule == nil {
			matchedRule = r
		}
	}

	if matchedRule == nil {
		return 30000 // Default fallback: $0.03
	}

	// Sort tiers by min volume
	tiers := make([]Tier, len(matchedRule.Tiers))
	copy(tiers, matchedRule.Tiers)
	sort.Slice(tiers, func(i, j int) bool {
		return tiers[i].MinVolume < tiers[j].MinVolume
	})

	// Find applicable tier
	for i := len(tiers) - 1; i >= 0; i-- {
		if volume >= tiers[i].MinVolume {
			return tiers[i].UnitPrice
		}
	}

	return tiers[0].UnitPrice
}

func (e *Engine) getUpstreamCost(productType, countryCode string) int64 {
	if cost, ok := e.upstream[productType+":"+countryCode]; ok {
		return cost
	}
	if cost, ok := e.upstream[productType+":*"]; ok {
		return cost
	}
	return 10000 // $0.01 default
}

// SetTenantPlan configures pricing for a tenant.
func (e *Engine) SetTenantPlan(tenantID, planID string, discount int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.tenants[tenantID] = &TenantPricing{
		PlanID: planID, CustomDiscount: discount,
		VolumeOverrides: make(map[string]int64),
	}
}

// SetCustomPrice sets a tenant-specific price override.
func (e *Engine) SetCustomPrice(tenantID, productType, countryCode string, price int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	tp, ok := e.tenants[tenantID]
	if !ok {
		tp = &TenantPricing{PlanID: "plan_payg", VolumeOverrides: make(map[string]int64)}
		e.tenants[tenantID] = tp
	}
	tp.VolumeOverrides[productType+":"+countryCode] = price
}

// ListPlans returns all available plans.
func (e *Engine) ListPlans() []*Plan {
	e.mu.RLock()
	defer e.mu.RUnlock()
	var plans []*Plan
	for _, p := range e.plans {
		plans = append(plans, p)
	}
	return plans
}

func intPtr(i int) *int { return &i }

// FormatPrice converts microcents to dollar string.
func FormatPrice(microcents int64) string {
	dollars := float64(microcents) / 1_000_000
	return fmt.Sprintf("$%.4f", dollars)
}
