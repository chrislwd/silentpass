package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/silentpass/silentpass/internal/model"
)

// PGTenantRepo implements tenant storage with PostgreSQL.
type PGTenantRepo struct {
	pool *pgxpool.Pool
}

func NewPGTenantRepo(pool *pgxpool.Pool) *PGTenantRepo {
	return &PGTenantRepo{pool: pool}
}

func (r *PGTenantRepo) ResolveByAPIKey(apiKey string) (tenantID string, apiSecret string, err error) {
	err = r.pool.QueryRow(context.Background(), `
		SELECT id, api_secret FROM tenants WHERE api_key = $1 AND status = 'active'`,
		apiKey,
	).Scan(&tenantID, &apiSecret)
	if err != nil {
		return "", "", fmt.Errorf("resolve tenant: %w", err)
	}
	return tenantID, apiSecret, nil
}

func (r *PGTenantRepo) GetByID(ctx context.Context, id string) (*model.Tenant, error) {
	var t model.Tenant
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, api_key, api_secret, status, plan, created_at, updated_at
		FROM tenants WHERE id = $1`,
		id,
	).Scan(&t.ID, &t.Name, &t.APIKey, &t.APISecret, &t.Status, &t.Plan, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get tenant: %w", err)
	}
	return &t, nil
}

func (r *PGTenantRepo) Create(ctx context.Context, t *model.Tenant) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO tenants (id, name, api_key, api_secret, status, plan, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		t.ID, t.Name, t.APIKey, t.APISecret, t.Status, t.Plan, t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create tenant: %w", err)
	}
	return nil
}

func (r *PGTenantRepo) List(ctx context.Context) ([]*model.Tenant, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, api_key, api_secret, status, plan, created_at, updated_at
		FROM tenants ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*model.Tenant
	for rows.Next() {
		var t model.Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.APIKey, &t.APISecret, &t.Status, &t.Plan, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan tenant: %w", err)
		}
		tenants = append(tenants, &t)
	}
	return tenants, nil
}
