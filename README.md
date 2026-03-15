# SilentPass

[中文文档](./README_CN.md)

**Mobile Identity Verification & Fraud Prevention Platform**

SilentPass is a developer-facing platform that leverages carrier Network APIs (CAMARA / Open Gateway) to provide frictionless phone number verification, silent authentication, SIM swap detection, and intelligent OTP fallback — all through a unified API.

## Why SilentPass

- **Higher conversion** — Silent verification completes in <3s with zero user input
- **Lower cost** — Reduces SMS OTP spend by 70-85% in supported markets
- **Stronger security** — Network-level identity signals + SIM swap detection catch fraud that OTP alone cannot
- **Global coverage** — Unified API across countries and operators with automatic fallback
- **One integration** — Single SDK and API replaces multiple vendor integrations

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│                     Client Layer                         │
│              iOS SDK / Android SDK / JS SDK              │
├──────────────────────────────────────────────────────────┤
│                   API Gateway Layer                      │
│         Auth (API Key + HMAC) │ Rate Limit │ CORS       │
├──────────────────────────────────────────────────────────┤
│                 Orchestration Layer                      │
│  Verification │ Risk/Verdict │ Policy Engine │ Webhooks  │
├──────────────────────────────────────────────────────────┤
│                 Supply Adapter Layer                     │
│    Telco Adapters │ OTP Providers │ Channel Partners     │
├──────────────────────────────────────────────────────────┤
│               Data & Analytics Layer                     │
│     PostgreSQL │ Redis │ Billing │ Logs │ Metrics        │
└──────────────────────────────────────────────────────────┘
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/verification/session` | Create verification session |
| `POST` | `/v1/verification/silent` | Execute silent verification |
| `POST` | `/v1/verification/otp/send` | Send OTP (SMS/WhatsApp/Voice) |
| `POST` | `/v1/verification/otp/check` | Verify OTP code |
| `POST` | `/v1/risk/sim-swap` | Check SIM swap status |
| `POST` | `/v1/risk/verdict` | Get unified risk verdict |
| `GET/POST` | `/v1/policies` | Manage verification policies |
| `PUT/DELETE` | `/v1/policies/:id` | Update/delete policy |
| `POST` | `/v1/webhooks` | Register webhook subscription |
| `GET` | `/v1/stats/dashboard` | Dashboard metrics |
| `GET` | `/v1/stats/activity` | Recent activity feed |
| `GET` | `/v1/logs` | Request trace logs |
| `GET` | `/v1/billing/summary` | Billing summary |
| `GET` | `/health` | Health check |

## Quick Start

### Prerequisites

- Go 1.23+
- Node.js 20+ (for dashboard)
- PostgreSQL 16 & Redis 7 (optional — auto-fallback to in-memory)

### Run Backend

```bash
cd silentpass
go mod tidy
make dev
```

The server starts on `http://localhost:8080` with in-memory storage and sandbox adapters.

### Run Dashboard

```bash
cd web/dashboard
npm install
npm run dev
```

Dashboard available at `http://localhost:3000`.

### Run with Docker

```bash
docker-compose up -d
```

Starts API + PostgreSQL + Redis.

### Test the API

```bash
# Create session
curl -X POST http://localhost:8080/v1/verification/session \
  -H "X-API-Key: sk_test_sandbox_key_001" \
  -H "Content-Type: application/json" \
  -d '{
    "app_id": "my_app",
    "phone_number": "+6281234567890",
    "country_code": "ID",
    "verification_type": "silent_or_otp",
    "use_case": "signup"
  }'

# Silent verify
curl -X POST http://localhost:8080/v1/verification/silent \
  -H "X-API-Key: sk_test_sandbox_key_001" \
  -H "Content-Type: application/json" \
  -d '{"session_id": "<session_id>"}'

# SIM swap check
curl -X POST http://localhost:8080/v1/risk/sim-swap \
  -H "X-API-Key: sk_test_sandbox_key_001" \
  -H "Content-Type: application/json" \
  -d '{"phone_number": "+6281234567890", "country_code": "ID"}'
```

### Run Tests

```bash
make test
```

39 tests covering handlers, services, middleware, JWT, and webhooks.

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Backend | Go, Gin |
| Database | PostgreSQL |
| Cache/Rate Limit | Redis |
| Frontend | Next.js, React, TypeScript, Tailwind CSS |
| Auth | API Key + HMAC signature, JWT tokens |
| API Spec | OpenAPI 3.0 |
| Containerization | Docker, docker-compose |

## Project Structure

```
silentpass/
├── cmd/server/              # Application entrypoint
├── api/openapi/             # OpenAPI 3.0 specification
├── internal/
│   ├── adapter/telco/       # Carrier/channel partner adapters
│   ├── adapter/otp/         # OTP provider adapters
│   ├── config/              # Environment configuration
│   ├── database/            # PostgreSQL connection pool
│   ├── handler/             # HTTP handlers
│   ├── middleware/          # Auth, CORS, rate limiting
│   ├── model/               # Data models
│   ├── pkg/auth/            # JWT token service
│   ├── pkg/errors/          # Error types
│   ├── repository/          # Data access (in-memory + PostgreSQL)
│   ├── router/              # Route definitions & DI wiring
│   └── service/             # Business logic
│       ├── verification/    # Silent verify + OTP orchestration
│       ├── risk/            # SIM swap + verdict
│       ├── policy/          # Decision engine
│       └── webhook/         # Event delivery
├── migrations/              # SQL migrations
├── web/dashboard/           # Next.js console
├── Dockerfile
├── docker-compose.yaml
└── Makefile
```

## Sandbox Mode

The platform ships with sandbox adapters that simulate:

- **Silent verification** — ~85% success rate, 200-500ms latency
- **SIM swap detection** — ~10% positive rate
- **OTP delivery** — Codes printed to stdout, universal test code `000000`

No external services needed for development and testing.

## Coverage

Sandbox supports: Indonesia (ID), Thailand (TH), Philippines (PH), Malaysia (MY), Singapore (SG), Vietnam (VN), Brazil (BR), Mexico (MX).

## License

MIT
