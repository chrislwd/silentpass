package model

import "time"

type Tenant struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	APIKey    string    `json:"-" db:"api_key"`
	APISecret string    `json:"-" db:"api_secret"`
	Status    string    `json:"status" db:"status"`
	Plan      string    `json:"plan" db:"plan"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type BillingRecord struct {
	ID             string    `json:"id" db:"id"`
	TenantID       string    `json:"tenant_id" db:"tenant_id"`
	ProductType    string    `json:"product_type" db:"product_type"`
	CountryCode    string    `json:"country_code" db:"country_code"`
	Provider       string    `json:"provider" db:"provider"`
	UnitCost       int64     `json:"unit_cost" db:"unit_cost"`       // in microcents
	UnitPrice      int64     `json:"unit_price" db:"unit_price"`     // in microcents
	Margin         int64     `json:"margin" db:"margin"`             // in microcents
	BillableStatus string    `json:"billable_status" db:"billable_status"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}
