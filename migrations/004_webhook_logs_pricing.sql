-- Webhook delivery logs
CREATE TABLE webhook_deliveries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL REFERENCES webhook_subscriptions(id),
    event_id        VARCHAR(64) NOT NULL,
    event_type      VARCHAR(50) NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    attempts        INTEGER NOT NULL DEFAULT 0,
    last_status_code INTEGER,
    last_error      TEXT,
    payload         JSONB NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    delivered_at    TIMESTAMPTZ
);

CREATE INDEX idx_webhook_deliveries_sub ON webhook_deliveries(subscription_id);
CREATE INDEX idx_webhook_deliveries_status ON webhook_deliveries(status);

-- Upstream provider metrics (for smart routing)
CREATE TABLE upstream_metrics (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_name   VARCHAR(100) NOT NULL,
    country_code    VARCHAR(5) NOT NULL,
    capability      VARCHAR(30) NOT NULL,
    window_start    TIMESTAMPTZ NOT NULL,
    total_requests  INTEGER NOT NULL DEFAULT 0,
    success_count   INTEGER NOT NULL DEFAULT 0,
    failure_count   INTEGER NOT NULL DEFAULT 0,
    avg_latency_ms  INTEGER NOT NULL DEFAULT 0,
    p95_latency_ms  INTEGER NOT NULL DEFAULT 0,
    UNIQUE (provider_name, country_code, capability, window_start)
);

CREATE INDEX idx_upstream_metrics_lookup ON upstream_metrics(provider_name, country_code, capability);

-- Customer pricing tiers
CREATE TABLE pricing_plans (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(100) NOT NULL,
    plan_type       VARCHAR(20) NOT NULL DEFAULT 'payg',
    base_fee        BIGINT NOT NULL DEFAULT 0,
    min_commitment  BIGINT NOT NULL DEFAULT 0,
    active          BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE pricing_rules (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id         UUID NOT NULL REFERENCES pricing_plans(id),
    product_type    VARCHAR(50) NOT NULL,
    country_code    VARCHAR(5) NOT NULL DEFAULT '*',
    tier_min        INTEGER NOT NULL DEFAULT 0,
    tier_max        INTEGER,
    unit_price      BIGINT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_pricing_rules_plan ON pricing_rules(plan_id);

CREATE TABLE tenant_pricing (
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    plan_id         UUID NOT NULL REFERENCES pricing_plans(id),
    custom_discount INTEGER NOT NULL DEFAULT 0,
    effective_from  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, plan_id)
);
