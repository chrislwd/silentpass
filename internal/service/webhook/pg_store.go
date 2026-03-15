package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PGStore persists webhook subscriptions and delivery logs in PostgreSQL.
type PGStore struct {
	pool *pgxpool.Pool
}

func NewPGStore(pool *pgxpool.Pool) *PGStore {
	return &PGStore{pool: pool}
}

func (s *PGStore) GetSubscriptions(ctx context.Context, tenantID string, eventType string) ([]*Subscription, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, tenant_id, url, COALESCE(secret, ''), events, active
		FROM webhook_subscriptions
		WHERE tenant_id = $1 AND active = true
		  AND ($2 = ANY(events) OR '*' = ANY(events))`,
		tenantID, eventType)
	if err != nil {
		return nil, fmt.Errorf("query subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []*Subscription
	for rows.Next() {
		var sub Subscription
		if err := rows.Scan(&sub.ID, &sub.TenantID, &sub.URL, &sub.Secret, &sub.Events, &sub.Active); err != nil {
			return nil, err
		}
		subs = append(subs, &sub)
	}
	return subs, nil
}

// LogDelivery records a webhook delivery attempt.
func (s *PGStore) LogDelivery(ctx context.Context, d *DeliveryLog) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO webhook_deliveries (id, subscription_id, event_id, event_type, status, attempts, last_status_code, last_error, payload, created_at, delivered_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		d.ID, d.SubscriptionID, d.EventID, d.EventType, d.Status,
		d.Attempts, d.LastStatusCode, d.LastError, d.Payload,
		d.CreatedAt, d.DeliveredAt)
	if err != nil {
		return fmt.Errorf("log delivery: %w", err)
	}
	return nil
}

// UpdateDelivery updates a delivery log after retry.
func (s *PGStore) UpdateDelivery(ctx context.Context, id string, status string, attempts int, statusCode int, errMsg string) error {
	var deliveredAt *time.Time
	if status == "delivered" {
		now := time.Now()
		deliveredAt = &now
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE webhook_deliveries SET status=$1, attempts=$2, last_status_code=$3, last_error=$4, delivered_at=$5
		WHERE id=$6`,
		status, attempts, statusCode, errMsg, deliveredAt, id)
	return err
}

// ListDeliveries returns recent deliveries for a subscription.
func (s *PGStore) ListDeliveries(ctx context.Context, subscriptionID string, limit int) ([]*DeliveryLog, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, subscription_id, event_id, event_type, status, attempts, last_status_code, COALESCE(last_error, ''), payload, created_at, delivered_at
		FROM webhook_deliveries
		WHERE subscription_id = $1
		ORDER BY created_at DESC LIMIT $2`,
		subscriptionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deliveries []*DeliveryLog
	for rows.Next() {
		var d DeliveryLog
		var payload []byte
		if err := rows.Scan(&d.ID, &d.SubscriptionID, &d.EventID, &d.EventType,
			&d.Status, &d.Attempts, &d.LastStatusCode, &d.LastError,
			&payload, &d.CreatedAt, &d.DeliveredAt); err != nil {
			return nil, err
		}
		json.Unmarshal(payload, &d.Payload)
		deliveries = append(deliveries, &d)
	}
	return deliveries, nil
}

// DeliveryLog represents a webhook delivery attempt record.
type DeliveryLog struct {
	ID              string      `json:"id"`
	SubscriptionID  string      `json:"subscription_id"`
	EventID         string      `json:"event_id"`
	EventType       string      `json:"event_type"`
	Status          string      `json:"status"` // pending, delivered, failed
	Attempts        int         `json:"attempts"`
	LastStatusCode  int         `json:"last_status_code"`
	LastError       string      `json:"last_error"`
	Payload         interface{} `json:"payload"`
	CreatedAt       time.Time   `json:"created_at"`
	DeliveredAt     *time.Time  `json:"delivered_at,omitempty"`
}
