package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/silentpass/silentpass/internal/model"
	"github.com/silentpass/silentpass/internal/repository"
)

// PolicyStore abstracts policy storage (in-memory or PostgreSQL).
type PolicyStore interface {
	List(ctx context.Context, tenantID string) ([]*model.Policy, error)
	GetByID(ctx context.Context, id string) (*model.Policy, error)
	Create(ctx context.Context, p *model.Policy) error
	Update(ctx context.Context, id string, req *model.UpdatePolicyRequest) error
	Delete(ctx context.Context, id string) error
}

// LogsStore abstracts verification log queries.
type LogsStore interface {
	Query(ctx context.Context, tenantID string, search string, sessionID string, limit int) ([]LogEntry, int, error)
	Append(ctx context.Context, entry LogEntry) error
}

// BillingStore abstracts billing queries.
type BillingStore interface {
	Summary(ctx context.Context, tenantID string, from, to time.Time) ([]BillingEntry, float64, error)
}

// StatsStore abstracts dashboard statistics queries.
type StatsStore interface {
	Dashboard(ctx context.Context, tenantID string) (*DashboardStats, error)
	RecentActivity(ctx context.Context, tenantID string, limit int) ([]ActivityEntry, error)
}

type DashboardStats struct {
	TotalVerifications int64         `json:"total_verifications"`
	SilentSuccessRate  float64       `json:"silent_success_rate"`
	FallbackRate       float64       `json:"fallback_rate"`
	OTPCostSaved       float64       `json:"otp_cost_saved"`
	HighRiskBlocked    int64         `json:"high_risk_blocked"`
	AvgLatencyMs       int           `json:"avg_latency_ms"`
	Countries          []CountryStat `json:"countries"`
}

type CountryStat struct {
	Code       string  `json:"code"`
	Requests   int64   `json:"requests"`
	SilentRate float64 `json:"silent_rate"`
}

type ActivityEntry struct {
	Time      string `json:"time"`
	Event     string `json:"event"`
	Country   string `json:"country"`
	Status    string `json:"status"`
	LatencyMs int    `json:"latency_ms"`
}

// --- PostgreSQL Implementations ---

// PGLogsStore queries verification_attempts from PostgreSQL.
type PGLogsStore struct {
	pool repository.Pool
}

func NewPGLogsStore(pool repository.Pool) *PGLogsStore {
	return &PGLogsStore{pool: pool}
}

func (s *PGLogsStore) Query(ctx context.Context, tenantID string, search string, sessionID string, limit int) ([]LogEntry, int, error) {
	query := `
		SELECT va.id, va.session_id, va.created_at, va.method,
		       s.country_code, COALESCE(va.upstream_provider, ''),
		       va.result, COALESCE(va.latency_ms, 0), COALESCE(va.error_code, '')
		FROM verification_attempts va
		JOIN sessions s ON s.id = va.session_id
		WHERE s.tenant_id = $1`
	args := []interface{}{tenantID}
	argIdx := 2

	if sessionID != "" {
		query += fmt.Sprintf(" AND va.session_id::text = $%d", argIdx)
		args = append(args, sessionID)
		argIdx++
	}

	if search != "" {
		query += fmt.Sprintf(` AND (va.session_id::text ILIKE $%d OR va.method ILIKE $%d
			OR s.country_code ILIKE $%d OR va.result ILIKE $%d OR va.error_code ILIKE $%d)`,
			argIdx, argIdx, argIdx, argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	query += " ORDER BY va.created_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, limit)
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var entries []LogEntry
	for rows.Next() {
		var e LogEntry
		var ts time.Time
		if err := rows.Scan(&e.ID, &e.SessionID, &ts, &e.Method,
			&e.CountryCode, &e.UpstreamProvider, &e.Result, &e.LatencyMs, &e.ErrorCode); err != nil {
			return nil, 0, err
		}
		e.Timestamp = ts.Format("2006-01-02 15:04:05")
		entries = append(entries, e)
	}

	return entries, len(entries), nil
}

func (s *PGLogsStore) Append(ctx context.Context, entry LogEntry) error {
	return nil // Logs written by verification service
}

// PGBillingStore queries billing_records from PostgreSQL.
type PGBillingStore struct {
	pool repository.Pool
}

func NewPGBillingStore(pool repository.Pool) *PGBillingStore {
	return &PGBillingStore{pool: pool}
}

func (s *PGBillingStore) Summary(ctx context.Context, tenantID string, from, to time.Time) ([]BillingEntry, float64, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT product_type, country_code,
		       COUNT(*) as calls,
		       COUNT(*) FILTER (WHERE billable_status = 'success') as successful,
		       COALESCE(AVG(unit_price), 0) as avg_price,
		       COALESCE(SUM(unit_price), 0) as total
		FROM billing_records
		WHERE tenant_id = $1 AND created_at >= $2 AND created_at < $3
		GROUP BY product_type, country_code
		ORDER BY total DESC`,
		tenantID, from, to)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var entries []BillingEntry
	var totalCost float64
	for rows.Next() {
		var e BillingEntry
		var avgPrice, total int64
		if err := rows.Scan(&e.Product, &e.Country, &e.Calls, &e.Successful, &avgPrice, &total); err != nil {
			return nil, 0, err
		}
		e.UnitPrice = float64(avgPrice) / 1_000_000
		e.Total = float64(total) / 1_000_000
		totalCost += e.Total
		entries = append(entries, e)
	}

	return entries, totalCost, nil
}

// PGStatsStore queries dashboard statistics from PostgreSQL.
type PGStatsStore struct {
	pool repository.Pool
}

func NewPGStatsStore(pool repository.Pool) *PGStatsStore {
	return &PGStatsStore{pool: pool}
}

func (s *PGStatsStore) Dashboard(ctx context.Context, tenantID string) (*DashboardStats, error) {
	stats := &DashboardStats{}

	var silentOK, silentFB int64
	var avgLat float64
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*),
		       COUNT(*) FILTER (WHERE va.method = 'silent' AND va.result = 'verified'),
		       COUNT(*) FILTER (WHERE va.method = 'silent' AND va.result = 'fallback_required'),
		       COALESCE(AVG(va.latency_ms) FILTER (WHERE va.method = 'silent'), 0)
		FROM verification_attempts va
		JOIN sessions s ON s.id = va.session_id
		WHERE s.tenant_id = $1 AND va.created_at >= NOW() - INTERVAL '30 days'`,
		tenantID,
	).Scan(&stats.TotalVerifications, &silentOK, &silentFB, &avgLat)
	if err != nil {
		return stats, nil
	}

	stats.AvgLatencyMs = int(avgLat)
	if stats.TotalVerifications > 0 {
		stats.SilentSuccessRate = round2(float64(silentOK) / float64(stats.TotalVerifications) * 100)
		stats.FallbackRate = round2(float64(silentFB) / float64(stats.TotalVerifications) * 100)
	}
	stats.OTPCostSaved = round2(float64(silentOK) * 0.04)

	_ = s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM risk_checks
		WHERE session_id IN (SELECT id FROM sessions WHERE tenant_id = $1)
		AND verdict = 'block' AND created_at >= NOW() - INTERVAL '30 days'`,
		tenantID,
	).Scan(&stats.HighRiskBlocked)

	rows, err := s.pool.Query(ctx, `
		SELECT s.country_code, COUNT(*) as total,
		       COUNT(*) FILTER (WHERE va.method = 'silent' AND va.result = 'verified') as silent_ok
		FROM verification_attempts va
		JOIN sessions s ON s.id = va.session_id
		WHERE s.tenant_id = $1 AND va.created_at >= NOW() - INTERVAL '30 days'
		GROUP BY s.country_code ORDER BY total DESC LIMIT 10`,
		tenantID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var cs CountryStat
			var silentCount int64
			if rows.Scan(&cs.Code, &cs.Requests, &silentCount) == nil {
				if cs.Requests > 0 {
					cs.SilentRate = round2(float64(silentCount) / float64(cs.Requests) * 100)
				}
				stats.Countries = append(stats.Countries, cs)
			}
		}
	}

	return stats, nil
}

func (s *PGStatsStore) RecentActivity(ctx context.Context, tenantID string, limit int) ([]ActivityEntry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT va.method, s.country_code, va.result, COALESCE(va.latency_ms, 0), va.created_at
		FROM verification_attempts va
		JOIN sessions s ON s.id = va.session_id
		WHERE s.tenant_id = $1
		ORDER BY va.created_at DESC LIMIT $2`,
		tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []ActivityEntry
	now := time.Now()
	for rows.Next() {
		var a ActivityEntry
		var ts time.Time
		if rows.Scan(&a.Event, &a.Country, &a.Status, &a.LatencyMs, &ts) == nil {
			a.Time = formatTimeAgo(now, ts)
			activities = append(activities, a)
		}
	}
	return activities, nil
}

func formatTimeAgo(now, ts time.Time) string {
	d := now.Sub(ts)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
