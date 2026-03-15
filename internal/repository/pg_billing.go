package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/silentpass/silentpass/internal/model"
)

// PGBillingRepo stores billing records in PostgreSQL.
type PGBillingRepo struct {
	pool *pgxpool.Pool
}

func NewPGBillingRepo(pool *pgxpool.Pool) *PGBillingRepo {
	return &PGBillingRepo{pool: pool}
}

func (r *PGBillingRepo) Create(ctx context.Context, b *model.BillingRecord) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO billing_records (id, tenant_id, product_type, country_code, provider, unit_cost, unit_price, margin, billable_status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		b.ID, b.TenantID, b.ProductType, b.CountryCode, b.Provider,
		b.UnitCost, b.UnitPrice, b.Margin, b.BillableStatus, b.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert billing record: %w", err)
	}
	return nil
}

func (r *PGBillingRepo) ListByTenant(ctx context.Context, tenantID string, from, to time.Time) ([]*model.BillingRecord, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, product_type, country_code, COALESCE(provider, ''),
		       unit_cost, unit_price, margin, billable_status, created_at
		FROM billing_records
		WHERE tenant_id = $1 AND created_at >= $2 AND created_at < $3
		ORDER BY created_at DESC`,
		tenantID, from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("list billing records: %w", err)
	}
	defer rows.Close()

	var records []*model.BillingRecord
	for rows.Next() {
		var b model.BillingRecord
		if err := rows.Scan(
			&b.ID, &b.TenantID, &b.ProductType, &b.CountryCode, &b.Provider,
			&b.UnitCost, &b.UnitPrice, &b.Margin, &b.BillableStatus, &b.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan billing record: %w", err)
		}
		records = append(records, &b)
	}
	return records, nil
}

type BillingSummary struct {
	ProductType  string `json:"product_type"`
	CountryCode  string `json:"country_code"`
	TotalCalls   int64  `json:"total_calls"`
	SuccessCalls int64  `json:"success_calls"`
	TotalCost    int64  `json:"total_cost"`
	TotalRevenue int64  `json:"total_revenue"`
	TotalMargin  int64  `json:"total_margin"`
}

func (r *PGBillingRepo) SummaryByTenant(ctx context.Context, tenantID string, from, to time.Time) ([]*BillingSummary, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT product_type, country_code,
		       COUNT(*) as total_calls,
		       COUNT(*) FILTER (WHERE billable_status = 'success') as success_calls,
		       COALESCE(SUM(unit_cost), 0) as total_cost,
		       COALESCE(SUM(unit_price), 0) as total_revenue,
		       COALESCE(SUM(margin), 0) as total_margin
		FROM billing_records
		WHERE tenant_id = $1 AND created_at >= $2 AND created_at < $3
		GROUP BY product_type, country_code
		ORDER BY total_revenue DESC`,
		tenantID, from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("billing summary: %w", err)
	}
	defer rows.Close()

	var summaries []*BillingSummary
	for rows.Next() {
		var s BillingSummary
		if err := rows.Scan(
			&s.ProductType, &s.CountryCode,
			&s.TotalCalls, &s.SuccessCalls,
			&s.TotalCost, &s.TotalRevenue, &s.TotalMargin,
		); err != nil {
			return nil, fmt.Errorf("scan summary: %w", err)
		}
		summaries = append(summaries, &s)
	}
	return summaries, nil
}
