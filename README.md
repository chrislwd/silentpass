# SilentPass

**English** | [中文](#silentpass-中文)

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
│                      Client Layer                        │
│              iOS SDK / Android SDK / JS SDK              │
├──────────────────────────────────────────────────────────┤
│                    API Gateway Layer                     │
│     Auth (API Key + HMAC + JWT) │ Rate Limit │ CORS     │
├──────────────────────────────────────────────────────────┤
│                  Orchestration Layer                     │
│  Verification │ Risk/Verdict │ Policy Engine │ Webhooks  │
├──────────────────────────────────────────────────────────┤
│                  Supply Adapter Layer                    │
│  ipification │ Vonage │ CAMARA │ Twilio │ WhatsApp API  │
├──────────────────────────────────────────────────────────┤
│                Data & Analytics Layer                    │
│  PostgreSQL │ Redis │ Billing │ Logs │ Prometheus        │
└──────────────────────────────────────────────────────────┘
```

## API Endpoints (26)

### Verification
| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/verification/session` | Create verification session |
| `POST` | `/v1/verification/silent` | Execute silent verification |
| `POST` | `/v1/verification/otp/send` | Send OTP (SMS/WhatsApp/Voice) |
| `POST` | `/v1/verification/otp/check` | Verify OTP code |

### Risk & Fraud
| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/risk/sim-swap` | Check SIM swap status |
| `POST` | `/v1/risk/verdict` | Get unified risk verdict |

### Policies
| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/v1/policies` | List verification policies |
| `POST` | `/v1/policies` | Create policy with custom rules |
| `PUT` | `/v1/policies/:id` | Update policy |
| `DELETE` | `/v1/policies/:id` | Delete policy |

### Account & Auth
| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/auth/register` | Register user + create org |
| `POST` | `/v1/auth/login` | Login with email/password |
| `POST` | `/v1/account/api-keys` | Generate API key |
| `GET` | `/v1/account/api-keys` | List API keys |
| `DELETE` | `/v1/account/api-keys/:id` | Revoke API key |
| `POST` | `/v1/account/users` | Invite user to org |

### Operations
| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/v1/stats/dashboard` | Dashboard metrics |
| `GET` | `/v1/stats/activity` | Recent activity |
| `GET` | `/v1/logs` | Request trace logs |
| `GET` | `/v1/billing/summary` | Billing summary |
| `GET` | `/v1/pricing/plans` | List pricing plans |
| `POST` | `/v1/pricing/calculate` | Calculate unit price |
| `POST` | `/v1/webhooks` | Register webhook |
| `GET` | `/health` | Health check |
| `GET` | `/metrics` | Prometheus metrics |

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

Server starts on `http://localhost:8080` with in-memory storage and sandbox adapters.

### Run Dashboard

```bash
cd web/dashboard
npm install
npm run dev
```

Dashboard at `http://localhost:3000`.

### Run with Docker

```bash
docker-compose up -d
```

### Test the API

```bash
# Register
curl -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"you@example.com","password":"SecureP@ss1","name":"Your Name","company":"YourCo"}'

# Create session
curl -X POST http://localhost:8080/v1/verification/session \
  -H "X-API-Key: sk_test_sandbox_key_001" \
  -H "Content-Type: application/json" \
  -d '{"app_id":"my_app","phone_number":"+6281234567890","country_code":"ID","verification_type":"silent_or_otp","use_case":"signup"}'

# Silent verify
curl -X POST http://localhost:8080/v1/verification/silent \
  -H "X-API-Key: sk_test_sandbox_key_001" \
  -H "Content-Type: application/json" \
  -d '{"session_id": "<session_id>"}'
```

### Run Tests

```bash
make test   # 104 tests
```

## Upstream Integrations

### Telco Adapters (Silent Verify + SIM Swap)
| Provider | Auth | Markets |
|----------|------|---------|
| **ipification** | OAuth2 Client Credentials | SEA (ID, TH, PH, MY, SG) |
| **Vonage** | RSA JWT | Europe (DE, ES, IT, GB) |
| **CAMARA Open Gateway** | OAuth2 | Any GSMA aggregator (Telefonica, Singtel, Orange, DT) |
| **Sandbox** | None | All (development) |

### OTP Providers
| Provider | Channels | Verify Method |
|----------|----------|---------------|
| **Twilio Verify** | SMS, Voice, WhatsApp | Twilio-managed |
| **Vonage Verify v2** | SMS, Voice, WhatsApp | Workflow-based |
| **WhatsApp Business API** | WhatsApp | Self-managed codes |
| **Sandbox** | All | In-memory (test code: `000000`) |

## Mobile SDKs

### iOS (Swift Package)
```swift
let sp = SilentPass(config: SilentPassConfig(apiKey: "sk_...", appID: "my_app"))
let result = try await sp.verify(phoneNumber: "+628...", countryCode: "ID", useCase: .signup)
```

### Android (Kotlin)
```kotlin
val sp = SilentPass(context, SilentPassConfig(apiKey = "sk_...", appID = "my_app"))
when (val result = sp.verify("+628...", "ID", UseCase.SIGNUP)) {
    is VerificationResult.Verified -> { /* use result.token */ }
    is VerificationResult.OTPRequired -> { /* show OTP input */ }
}
```

## Policy Engine

Configurable rules with 10 condition dimensions:

| Condition | Description |
|-----------|-------------|
| `countries` | Match country codes |
| `operators` | Match carrier/operator |
| `use_cases` | Match signup/login/transaction/phone_change |
| `channels` | Match verification channel |
| `sim_swap_detected` | SIM recently swapped |
| `verification_failed` | Silent verify failed |
| `confidence_below` | Confidence score threshold |
| `device_changed` | New device detected |
| `risk_score_above` | Cumulative risk score threshold |
| `hour_range` | UTC time window |

Risk scoring: SIM swap (+50), verification failed (+20), low confidence (+15), device changed (+10). Verdicts escalate: allow < challenge < review < block.

## Pricing

3 built-in plans with volume-based tiers:

| Plan | Monthly Fee | Silent Verify | SMS OTP | SIM Swap |
|------|-------------|---------------|---------|----------|
| Pay-As-You-Go | $0 | $0.030 | $0.045 | $0.010 |
| Growth | $99 | $0.025 - $0.015 | $0.040 - $0.030 | $0.008 |
| Enterprise | $999 | $0.012 | $0.025 | $0.005 |

Per-tenant custom discounts and price overrides supported.

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Backend | Go, Gin |
| Database | PostgreSQL (auto-fallback to in-memory) |
| Cache / Rate Limit | Redis (auto-fallback to in-memory) |
| Frontend | Next.js, React, TypeScript, Tailwind CSS |
| Auth | API Key + HMAC, JWT, bcrypt, RBAC (6 roles) |
| Monitoring | Prometheus metrics |
| API Spec | OpenAPI 3.0 |
| Mobile | iOS (Swift), Android (Kotlin) |
| Containerization | Docker, docker-compose |

## Project Structure

```
silentpass/
├── cmd/server/                  # Application entrypoint
├── api/openapi/                 # OpenAPI 3.0 spec
├── sdk/
│   ├── ios/                     # iOS Swift Package
│   └── android/                 # Android Kotlin library
├── internal/
│   ├── adapter/
│   │   ├── telco/               # ipification, Vonage, CAMARA, Sandbox, SmartRouter
│   │   └── otp/                 # Twilio, Vonage, WhatsApp, Sandbox
│   ├── config/                  # Environment config
│   ├── database/                # PostgreSQL pool
│   ├── handler/                 # HTTP handlers (26 endpoints)
│   ├── metrics/                 # Prometheus metrics
│   ├── middleware/              # Auth, CORS, rate limit, RBAC
│   ├── model/                   # Data models
│   ├── pkg/{auth,crypto,errors} # JWT, bcrypt, API key gen
│   ├── repository/              # Data access (memory + PG)
│   ├── router/                  # Routes & dependency injection
│   └── service/
│       ├── verification/        # Silent + OTP orchestration
│       ├── risk/                # SIM swap + verdict
│       ├── policy/              # Rule-based decision engine
│       ├── pricing/             # Tiered pricing engine
│       └── webhook/             # Event delivery + PG logs
├── migrations/                  # 4 SQL migrations
├── config/                      # Example provider configs
├── web/dashboard/               # Next.js console (7 pages)
├── Dockerfile
├── docker-compose.yaml
└── Makefile
```

## Sandbox Mode

Built-in sandbox adapters for zero-dependency development:

- **Silent verification** — ~85% success rate, 200-500ms simulated latency
- **SIM swap detection** — ~10% positive rate
- **OTP delivery** — Codes logged to stdout, universal test code `000000`

Sandbox countries: ID, TH, PH, MY, SG, VN, BR, MX.

## License

MIT

---

# SilentPass 中文

[English](#silentpass)

**基于运营商 Network APIs 的无感手机号验证与反欺诈平台**

SilentPass 是一个面向开发者的平台，通过接入运营商 Network APIs（CAMARA / Open Gateway），实现无感手机号验证、静默认证、SIM Swap 检测和智能 OTP 降级 —— 统一 API、一次集成。

## 为什么选择 SilentPass

- **提升转化** — 无感验证 3 秒内完成，用户零操作
- **降低成本** — 在支持市场中减少 70-85% 的短信 OTP 开支
- **增强安全** — 网络侧身份信号 + SIM Swap 检测，拦截纯 OTP 无法发现的欺诈
- **全球覆盖** — 跨国家、跨运营商统一 API，自动降级
- **一次集成** — 一套 SDK 和 API 替代多个供应商的分别对接

## 架构

```
┌──────────────────────────────────────────────────────────┐
│                       客户端层                            │
│              iOS SDK / Android SDK / JS SDK              │
├──────────────────────────────────────────────────────────┤
│                     API 网关层                            │
│    认证 (API Key + HMAC + JWT) │ 限流 │ CORS             │
├──────────────────────────────────────────────────────────┤
│                       编排层                              │
│   验证编排 │ 风控决策 │ 策略引擎 │ Webhook                │
├──────────────────────────────────────────────────────────┤
│                     供应适配层                             │
│  ipification │ Vonage │ CAMARA │ Twilio │ WhatsApp API   │
├──────────────────────────────────────────────────────────┤
│                    数据与分析层                            │
│  PostgreSQL │ Redis │ 计费 │ 日志 │ Prometheus            │
└──────────────────────────────────────────────────────────┘
```

## API 端点 (26 个)

### 验证
| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/v1/verification/session` | 创建验证会话 |
| `POST` | `/v1/verification/silent` | 执行无感验证 |
| `POST` | `/v1/verification/otp/send` | 发送 OTP（短信/WhatsApp/语音）|
| `POST` | `/v1/verification/otp/check` | 校验 OTP |

### 风控
| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/v1/risk/sim-swap` | SIM Swap 检查 |
| `POST` | `/v1/risk/verdict` | 统一风控决策 |

### 策略
| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/v1/policies` | 列出验证策略 |
| `POST` | `/v1/policies` | 创建策略（支持自定义规则）|
| `PUT` | `/v1/policies/:id` | 更新策略 |
| `DELETE` | `/v1/policies/:id` | 删除策略 |

### 账户与认证
| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/v1/auth/register` | 注册用户 + 创建组织 |
| `POST` | `/v1/auth/login` | 邮箱密码登录 |
| `POST` | `/v1/account/api-keys` | 生成 API Key |
| `GET` | `/v1/account/api-keys` | 列出 API Key |
| `DELETE` | `/v1/account/api-keys/:id` | 吊销 API Key |
| `POST` | `/v1/account/users` | 邀请用户加入组织 |

### 运营
| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/v1/stats/dashboard` | 仪表盘指标 |
| `GET` | `/v1/stats/activity` | 最近活动 |
| `GET` | `/v1/logs` | 请求链路日志 |
| `GET` | `/v1/billing/summary` | 账单摘要 |
| `GET` | `/v1/pricing/plans` | 定价方案列表 |
| `POST` | `/v1/pricing/calculate` | 计算单价 |
| `POST` | `/v1/webhooks` | 注册 Webhook |
| `GET` | `/health` | 健康检查 |
| `GET` | `/metrics` | Prometheus 指标 |

## 快速开始

### 环境要求

- Go 1.23+
- Node.js 20+（控制台前端）
- PostgreSQL 16 & Redis 7（可选 — 不可用时自动降级为内存存储）

### 启动后端

```bash
cd silentpass
go mod tidy
make dev
```

服务启动在 `http://localhost:8080`，默认使用内存存储和沙箱适配器。

### 启动控制台

```bash
cd web/dashboard
npm install
npm run dev
```

控制台地址 `http://localhost:3000`。

### Docker 启动

```bash
docker-compose up -d
```

### 测试 API

```bash
# 注册
curl -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"you@example.com","password":"SecureP@ss1","name":"你的名字","company":"你的公司"}'

# 创建验证会话
curl -X POST http://localhost:8080/v1/verification/session \
  -H "X-API-Key: sk_test_sandbox_key_001" \
  -H "Content-Type: application/json" \
  -d '{"app_id":"my_app","phone_number":"+6281234567890","country_code":"ID","verification_type":"silent_or_otp","use_case":"signup"}'

# 无感验证
curl -X POST http://localhost:8080/v1/verification/silent \
  -H "X-API-Key: sk_test_sandbox_key_001" \
  -H "Content-Type: application/json" \
  -d '{"session_id": "<session_id>"}'
```

### 运行测试

```bash
make test   # 104 个测试
```

## 上游集成

### 运营商适配器（无感验证 + SIM Swap）
| 供应商 | 认证方式 | 覆盖市场 |
|--------|---------|---------|
| **ipification** | OAuth2 Client Credentials | 东南亚 (ID, TH, PH, MY, SG) |
| **Vonage** | RSA JWT | 欧洲 (DE, ES, IT, GB) |
| **CAMARA Open Gateway** | OAuth2 | 任意 GSMA 聚合商 (Telefonica, Singtel, Orange, DT) |
| **Sandbox** | 无 | 全部（开发环境）|

### OTP 供应商
| 供应商 | 通道 | 验证方式 |
|--------|------|---------|
| **Twilio Verify** | 短信、语音、WhatsApp | Twilio 托管验证 |
| **Vonage Verify v2** | 短信、语音、WhatsApp | Workflow 驱动 |
| **WhatsApp Business API** | WhatsApp | 自管理验证码 |
| **Sandbox** | 全部 | 内存存储（测试码：`000000`）|

## 移动端 SDK

### iOS (Swift Package)
```swift
let sp = SilentPass(config: SilentPassConfig(apiKey: "sk_...", appID: "my_app"))
let result = try await sp.verify(phoneNumber: "+628...", countryCode: "ID", useCase: .signup)
```

### Android (Kotlin)
```kotlin
val sp = SilentPass(context, SilentPassConfig(apiKey = "sk_...", appID = "my_app"))
when (val result = sp.verify("+628...", "ID", UseCase.SIGNUP)) {
    is VerificationResult.Verified -> { /* 使用 result.token */ }
    is VerificationResult.OTPRequired -> { /* 显示 OTP 输入框 */ }
}
```

## 策略引擎

支持 10 种条件维度的可配置规则：

| 条件 | 说明 |
|------|------|
| `countries` | 匹配国家代码 |
| `operators` | 匹配运营商 |
| `use_cases` | 匹配场景：注册/登录/交易/改绑 |
| `channels` | 匹配验证通道 |
| `sim_swap_detected` | SIM 卡近期被换 |
| `verification_failed` | 无感验证失败 |
| `confidence_below` | 置信度低于阈值 |
| `device_changed` | 新设备检测 |
| `risk_score_above` | 累计风险分超过阈值 |
| `hour_range` | UTC 时间窗口 |

风险评分：SIM swap (+50)、验证失败 (+20)、低置信度 (+15)、设备变更 (+10)。裁决逐级升级：allow < challenge < review < block。

## 定价

3 个内置方案，支持阶梯价：

| 方案 | 月费 | 无感验证 | 短信 OTP | SIM Swap |
|------|------|---------|---------|----------|
| 按量付费 | $0 | $0.030 | $0.045 | $0.010 |
| 成长版 | $99 | $0.025 - $0.015 | $0.040 - $0.030 | $0.008 |
| 企业版 | $999 | $0.012 | $0.025 | $0.005 |

支持租户级自定义折扣和价格覆盖。

## 技术栈

| 组件 | 技术 |
|------|------|
| 后端 | Go, Gin |
| 数据库 | PostgreSQL（自动降级到内存）|
| 缓存/限流 | Redis（自动降级到内存）|
| 前端 | Next.js, React, TypeScript, Tailwind CSS |
| 认证 | API Key + HMAC, JWT, bcrypt, RBAC（6 角色）|
| 监控 | Prometheus 指标 |
| API 规范 | OpenAPI 3.0 |
| 移动端 | iOS (Swift), Android (Kotlin) |
| 容器化 | Docker, docker-compose |

## 目标客户

- 金融科技 / 数字钱包 / 券商 / 交易所
- 高价值电商 / 交易平台
- 游戏与社交 App
- 出海互联网平台
- 企业账号安全场景

## 沙箱模式

内置沙箱适配器，零依赖开发：

- **无感验证** — ~85% 成功率，200-500ms 模拟延迟
- **SIM Swap 检测** — ~10% 阳性率
- **OTP 发送** — 验证码输出到控制台，通用测试码 `000000`

沙箱覆盖国家：ID, TH, PH, MY, SG, VN, BR, MX。

## 许可证

MIT
