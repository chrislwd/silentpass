package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/silentpass/silentpass/internal/model"
)

// PGVerificationRepo stores verification attempts in PostgreSQL.
type PGVerificationRepo struct {
	pool *pgxpool.Pool
}

func NewPGVerificationRepo(pool *pgxpool.Pool) *PGVerificationRepo {
	return &PGVerificationRepo{pool: pool}
}

func (r *PGVerificationRepo) Create(ctx context.Context, a *model.VerificationAttempt) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO verification_attempts (id, session_id, method, upstream_provider, upstream_operator, result, latency_ms, error_code, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		a.ID, a.SessionID, a.Method, a.UpstreamProvider, a.UpstreamOperator,
		a.Result, a.LatencyMs, a.ErrorCode, a.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert verification attempt: %w", err)
	}
	return nil
}

func (r *PGVerificationRepo) ListBySession(ctx context.Context, sessionID string) ([]*model.VerificationAttempt, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, session_id, method, COALESCE(upstream_provider, ''), COALESCE(upstream_operator, ''),
		       result, COALESCE(latency_ms, 0), COALESCE(error_code, ''), created_at
		FROM verification_attempts WHERE session_id = $1
		ORDER BY created_at ASC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("list attempts: %w", err)
	}
	defer rows.Close()

	var attempts []*model.VerificationAttempt
	for rows.Next() {
		var a model.VerificationAttempt
		if err := rows.Scan(
			&a.ID, &a.SessionID, &a.Method, &a.UpstreamProvider, &a.UpstreamOperator,
			&a.Result, &a.LatencyMs, &a.ErrorCode, &a.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan attempt: %w", err)
		}
		attempts = append(attempts, &a)
	}
	return attempts, nil
}
