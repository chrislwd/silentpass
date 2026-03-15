-- SilentPass initial schema

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Tenants
CREATE TABLE tenants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    api_key     VARCHAR(64) NOT NULL UNIQUE,
    api_secret  VARCHAR(128) NOT NULL,
    status      VARCHAR(20) NOT NULL DEFAULT 'active',
    plan        VARCHAR(50) NOT NULL DEFAULT 'free',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_tenants_api_key ON tenants(api_key);

-- Verification sessions
CREATE TABLE sessions (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants(id),
    phone_hash        VARCHAR(64) NOT NULL,
    country_code      VARCHAR(5) NOT NULL,
    verification_type VARCHAR(20) NOT NULL,
    use_case          VARCHAR(20) NOT NULL,
    status            VARCHAR(20) NOT NULL DEFAULT 'pending',
    device_ip         VARCHAR(45),
    user_agent        TEXT,
    callback_url      TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at        TIMESTAMPTZ NOT NULL,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_sessions_tenant ON sessions(tenant_id);
CREATE INDEX idx_sessions_status ON sessions(status);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);

-- Verification attempts
CREATE TABLE verification_attempts (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id        UUID NOT NULL REFERENCES sessions(id),
    method            VARCHAR(20) NOT NULL,
    upstream_provider VARCHAR(100),
    upstream_operator VARCHAR(100),
    result            VARCHAR(30) NOT NULL,
    latency_ms        INTEGER,
    error_code        VARCHAR(50),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_attempts_session ON verification_attempts(session_id);

-- Risk checks
CREATE TABLE risk_checks (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id        UUID REFERENCES sessions(id),
    risk_type         VARCHAR(30) NOT NULL,
    raw_signal        JSONB,
    normalized_signal VARCHAR(50),
    verdict           VARCHAR(20) NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_risk_checks_session ON risk_checks(session_id);

-- Billing records
CREATE TABLE billing_records (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    product_type    VARCHAR(50) NOT NULL,
    country_code    VARCHAR(5) NOT NULL,
    provider        VARCHAR(100),
    unit_cost       BIGINT NOT NULL DEFAULT 0,
    unit_price      BIGINT NOT NULL DEFAULT 0,
    margin          BIGINT NOT NULL DEFAULT 0,
    billable_status VARCHAR(20) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_billing_tenant ON billing_records(tenant_id);
CREATE INDEX idx_billing_created ON billing_records(created_at);
