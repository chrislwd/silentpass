package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/silentpass/silentpass/internal/model"
)

type PGPolicyRepo struct {
	pool *pgxpool.Pool
}

func NewPGPolicyRepo(pool *pgxpool.Pool) *PGPolicyRepo {
	return &PGPolicyRepo{pool: pool}
}

func (r *PGPolicyRepo) Create(ctx context.Context, p *model.Policy) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO policies (id, tenant_id, name, use_case, strategy, sim_swap_action, countries, priority, active, config, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		p.ID, p.TenantID, p.Name, p.UseCase, p.Strategy, p.SIMSwapAction,
		p.Countries, p.Priority, p.Active, p.Config, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert policy: %w", err)
	}
	return nil
}

func (r *PGPolicyRepo) GetByID(ctx context.Context, id string) (*model.Policy, error) {
	var p model.Policy
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, name, use_case, strategy, sim_swap_action, countries, priority, active, config, created_at, updated_at
		FROM policies WHERE id = $1`, id,
	).Scan(&p.ID, &p.TenantID, &p.Name, &p.UseCase, &p.Strategy, &p.SIMSwapAction,
		&p.Countries, &p.Priority, &p.Active, &p.Config, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get policy: %w", err)
	}
	return &p, nil
}

func (r *PGPolicyRepo) ListByTenant(ctx context.Context, tenantID string) ([]*model.Policy, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, name, use_case, strategy, sim_swap_action, countries, priority, active, config, created_at, updated_at
		FROM policies WHERE tenant_id = $1
		ORDER BY priority DESC, created_at ASC`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list policies: %w", err)
	}
	defer rows.Close()

	var policies []*model.Policy
	for rows.Next() {
		var p model.Policy
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Name, &p.UseCase, &p.Strategy, &p.SIMSwapAction,
			&p.Countries, &p.Priority, &p.Active, &p.Config, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan policy: %w", err)
		}
		policies = append(policies, &p)
	}
	return policies, nil
}

func (r *PGPolicyRepo) FindForUseCase(ctx context.Context, tenantID string, useCase string, countryCode string) (*model.Policy, error) {
	var p model.Policy
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, name, use_case, strategy, sim_swap_action, countries, priority, active, config, created_at, updated_at
		FROM policies
		WHERE tenant_id = $1 AND use_case = $2 AND active = true
		  AND ($3 = ANY(countries) OR '*' = ANY(countries))
		ORDER BY priority DESC
		LIMIT 1`, tenantID, useCase, countryCode,
	).Scan(&p.ID, &p.TenantID, &p.Name, &p.UseCase, &p.Strategy, &p.SIMSwapAction,
		&p.Countries, &p.Priority, &p.Active, &p.Config, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("find policy: %w", err)
	}
	return &p, nil
}

func (r *PGPolicyRepo) Update(ctx context.Context, id string, req *model.UpdatePolicyRequest) error {
	// Build dynamic update - simplified version
	_, err := r.pool.Exec(ctx, `
		UPDATE policies SET updated_at = $1 WHERE id = $2`,
		time.Now(), id)
	if err != nil {
		return fmt.Errorf("update policy: %w", err)
	}
	return nil
}

func (r *PGPolicyRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM policies WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete policy: %w", err)
	}
	return nil
}
