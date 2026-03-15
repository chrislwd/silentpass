package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/silentpass/silentpass/internal/model"
)

// PGSessionRepo implements session storage with PostgreSQL.
type PGSessionRepo struct {
	pool *pgxpool.Pool
}

func NewPGSessionRepo(pool *pgxpool.Pool) *PGSessionRepo {
	return &PGSessionRepo{pool: pool}
}

func (r *PGSessionRepo) CreateSession(ctx context.Context, session *model.Session) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO sessions (id, tenant_id, phone_hash, country_code, verification_type, use_case, status, device_ip, user_agent, callback_url, created_at, expires_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		session.ID, session.TenantID, session.PhoneHash, session.CountryCode,
		session.VerificationType, session.UseCase, session.Status,
		session.DeviceIP, session.UserAgent, session.CallbackURL,
		session.CreatedAt, session.ExpiresAt, session.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

func (r *PGSessionRepo) GetSession(ctx context.Context, sessionID string) (*model.Session, error) {
	var s model.Session
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, phone_hash, country_code, verification_type, use_case, status,
		       COALESCE(device_ip, ''), COALESCE(user_agent, ''), COALESCE(callback_url, ''),
		       created_at, expires_at, updated_at
		FROM sessions WHERE id = $1`, sessionID,
	).Scan(
		&s.ID, &s.TenantID, &s.PhoneHash, &s.CountryCode,
		&s.VerificationType, &s.UseCase, &s.Status,
		&s.DeviceIP, &s.UserAgent, &s.CallbackURL,
		&s.CreatedAt, &s.ExpiresAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	return &s, nil
}

func (r *PGSessionRepo) UpdateSessionStatus(ctx context.Context, sessionID string, status model.SessionStatus) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE sessions SET status = $1, updated_at = $2 WHERE id = $3`,
		status, time.Now(), sessionID,
	)
	if err != nil {
		return fmt.Errorf("update session status: %w", err)
	}
	return nil
}

func (r *PGSessionRepo) ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]*model.Session, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, phone_hash, country_code, verification_type, use_case, status,
		       COALESCE(device_ip, ''), COALESCE(user_agent, ''), COALESCE(callback_url, ''),
		       created_at, expires_at, updated_at
		FROM sessions WHERE tenant_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		tenantID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*model.Session
	for rows.Next() {
		var s model.Session
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.PhoneHash, &s.CountryCode,
			&s.VerificationType, &s.UseCase, &s.Status,
			&s.DeviceIP, &s.UserAgent, &s.CallbackURL,
			&s.CreatedAt, &s.ExpiresAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		sessions = append(sessions, &s)
	}
	return sessions, nil
}
