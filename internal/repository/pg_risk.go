package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/silentpass/silentpass/internal/model"
)

// PGRiskRepo stores risk checks in PostgreSQL.
type PGRiskRepo struct {
	pool *pgxpool.Pool
}

func NewPGRiskRepo(pool *pgxpool.Pool) *PGRiskRepo {
	return &PGRiskRepo{pool: pool}
}

func (r *PGRiskRepo) Create(ctx context.Context, rc *model.RiskCheck) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO risk_checks (id, session_id, risk_type, raw_signal, normalized_signal, verdict, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		rc.ID, rc.SessionID, rc.RiskType, rc.RawSignal, rc.NormalizedSignal, rc.Verdict, rc.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert risk check: %w", err)
	}
	return nil
}

func (r *PGRiskRepo) ListBySession(ctx context.Context, sessionID string) ([]*model.RiskCheck, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, COALESCE(session_id::text, ''), risk_type, COALESCE(raw_signal::text, ''),
		       COALESCE(normalized_signal, ''), verdict, created_at
		FROM risk_checks WHERE session_id = $1
		ORDER BY created_at ASC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("list risk checks: %w", err)
	}
	defer rows.Close()

	var checks []*model.RiskCheck
	for rows.Next() {
		var rc model.RiskCheck
		if err := rows.Scan(
			&rc.ID, &rc.SessionID, &rc.RiskType, &rc.RawSignal,
			&rc.NormalizedSignal, &rc.Verdict, &rc.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan risk check: %w", err)
		}
		checks = append(checks, &rc)
	}
	return checks, nil
}
