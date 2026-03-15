package pricing

import (
	"context"
	"testing"
)

func TestPaygPricing(t *testing.T) {
	e := NewEngine()
	result := e.CalculatePrice(context.Background(), "tenant-1", "silent_verify", "ID", 0)

	if result.UnitPrice != 30000 { // $0.03
		t.Fatalf("expected 30000, got %d", result.UnitPrice)
	}
	if result.UpstreamCost != 15000 { // $0.015
		t.Fatalf("expected upstream 15000, got %d", result.UpstreamCost)
	}
	if result.Margin != 15000 {
		t.Fatalf("expected margin 15000, got %d", result.Margin)
	}
}

func TestGrowthPlan_VolumeTiers(t *testing.T) {
	e := NewEngine()
	e.SetTenantPlan("tenant-g", "plan_growth", 0)

	// Low volume
	r1 := e.CalculatePrice(context.Background(), "tenant-g", "silent_verify", "ID", 100)
	if r1.UnitPrice != 25000 {
		t.Fatalf("low volume: expected 25000, got %d", r1.UnitPrice)
	}

	// Mid volume
	r2 := e.CalculatePrice(context.Background(), "tenant-g", "silent_verify", "ID", 50000)
	if r2.UnitPrice != 20000 {
		t.Fatalf("mid volume: expected 20000, got %d", r2.UnitPrice)
	}

	// High volume
	r3 := e.CalculatePrice(context.Background(), "tenant-g", "silent_verify", "ID", 200000)
	if r3.UnitPrice != 15000 {
		t.Fatalf("high volume: expected 15000, got %d", r3.UnitPrice)
	}
}

func TestEnterprisePlan(t *testing.T) {
	e := NewEngine()
	e.SetTenantPlan("tenant-e", "plan_enterprise", 0)

	result := e.CalculatePrice(context.Background(), "tenant-e", "silent_verify", "ID", 0)
	if result.UnitPrice != 12000 {
		t.Fatalf("expected 12000, got %d", result.UnitPrice)
	}
}

func TestCustomDiscount(t *testing.T) {
	e := NewEngine()
	e.SetTenantPlan("tenant-d", "plan_payg", 20) // 20% discount

	result := e.CalculatePrice(context.Background(), "tenant-d", "silent_verify", "ID", 0)
	expected := int64(30000 * 80 / 100) // 24000
	if result.UnitPrice != expected {
		t.Fatalf("expected %d with 20%% discount, got %d", expected, result.UnitPrice)
	}
	if result.DiscountPct != 20 {
		t.Fatalf("expected discount 20, got %d", result.DiscountPct)
	}
}

func TestCustomPriceOverride(t *testing.T) {
	e := NewEngine()
	e.SetCustomPrice("tenant-c", "silent_verify", "ID", 10000)

	result := e.CalculatePrice(context.Background(), "tenant-c", "silent_verify", "ID", 0)
	if result.UnitPrice != 10000 {
		t.Fatalf("expected custom price 10000, got %d", result.UnitPrice)
	}
	if result.TierApplied != "custom" {
		t.Fatalf("expected tier 'custom', got '%s'", result.TierApplied)
	}
}

func TestSMSOTPPricing(t *testing.T) {
	e := NewEngine()
	result := e.CalculatePrice(context.Background(), "", "sms_otp", "ID", 0)
	if result.UnitPrice != 45000 {
		t.Fatalf("sms otp: expected 45000, got %d", result.UnitPrice)
	}
}

func TestSIMSwapPricing(t *testing.T) {
	e := NewEngine()
	result := e.CalculatePrice(context.Background(), "", "sim_swap", "ID", 0)
	if result.UnitPrice != 10000 {
		t.Fatalf("sim swap: expected 10000, got %d", result.UnitPrice)
	}
}

func TestListPlans(t *testing.T) {
	e := NewEngine()
	plans := e.ListPlans()
	if len(plans) != 3 {
		t.Fatalf("expected 3 plans, got %d", len(plans))
	}
}

func TestFormatPrice(t *testing.T) {
	tests := []struct {
		microcents int64
		want       string
	}{
		{30000, "$0.0300"},
		{1000000, "$1.0000"},
		{15000, "$0.0150"},
	}
	for _, tt := range tests {
		got := FormatPrice(tt.microcents)
		if got != tt.want {
			t.Errorf("FormatPrice(%d) = %s, want %s", tt.microcents, got, tt.want)
		}
	}
}

func TestFallbackToDefaultPlan(t *testing.T) {
	e := NewEngine()
	// Unknown tenant should get PAYG pricing
	result := e.CalculatePrice(context.Background(), "unknown-tenant", "silent_verify", "ID", 0)
	if result.UnitPrice != 30000 {
		t.Fatalf("expected PAYG price, got %d", result.UnitPrice)
	}
}
