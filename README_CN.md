# SilentPass

[English](./README.md)

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
│                      客户端层                             │
│              iOS SDK / Android SDK / JS SDK              │
├──────────────────────────────────────────────────────────┤
│                    API 网关层                              │
│       认证 (API Key + HMAC) │ 限流 │ CORS                │
├──────────────────────────────────────────────────────────┤
│                     编排层                                │
│  验证编排 │ 风控决策 │ 策略引擎 │ Webhook                  │
├──────────────────────────────────────────────────────────┤
│                   供应适配层                               │
│    运营商适配器 │ OTP 供应商 │ Channel Partner              │
├──────────────────────────────────────────────────────────┤
│                  数据与分析层                              │
│     PostgreSQL │ Redis │ 计费 │ 日志 │ 指标               │
└──────────────────────────────────────────────────────────┘
```

## API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/v1/verification/session` | 创建验证会话 |
| `POST` | `/v1/verification/silent` | 执行无感验证 |
| `POST` | `/v1/verification/otp/send` | 发送 OTP（短信/WhatsApp/语音） |
| `POST` | `/v1/verification/otp/check` | 校验 OTP |
| `POST` | `/v1/risk/sim-swap` | SIM Swap 检查 |
| `POST` | `/v1/risk/verdict` | 统一风控决策 |
| `GET/POST` | `/v1/policies` | 管理验证策略 |
| `PUT/DELETE` | `/v1/policies/:id` | 更新/删除策略 |
| `POST` | `/v1/webhooks` | 注册 Webhook |
| `GET` | `/v1/stats/dashboard` | 仪表盘指标 |
| `GET` | `/v1/stats/activity` | 最近活动 |
| `GET` | `/v1/logs` | 请求链路日志 |
| `GET` | `/v1/billing/summary` | 账单摘要 |
| `GET` | `/health` | 健康检查 |

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

一键启动 API + PostgreSQL + Redis。

### 测试 API

```bash
# 创建验证会话
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

# 无感验证
curl -X POST http://localhost:8080/v1/verification/silent \
  -H "X-API-Key: sk_test_sandbox_key_001" \
  -H "Content-Type: application/json" \
  -d '{"session_id": "<session_id>"}'

# SIM Swap 检查
curl -X POST http://localhost:8080/v1/risk/sim-swap \
  -H "X-API-Key: sk_test_sandbox_key_001" \
  -H "Content-Type: application/json" \
  -d '{"phone_number": "+6281234567890", "country_code": "ID"}'
```

### 运行测试

```bash
make test
```

39 个测试覆盖 handler、service、middleware、JWT、webhook。

## 技术栈

| 组件 | 技术 |
|------|------|
| 后端 | Go, Gin |
| 数据库 | PostgreSQL |
| 缓存/限流 | Redis |
| 前端 | Next.js, React, TypeScript, Tailwind CSS |
| 认证 | API Key + HMAC 签名, JWT Token |
| API 规范 | OpenAPI 3.0 |
| 容器化 | Docker, docker-compose |

## 项目结构

```
silentpass/
├── cmd/server/              # 应用入口
├── api/openapi/             # OpenAPI 3.0 规范
├── internal/
│   ├── adapter/telco/       # 运营商/Channel Partner 适配器
│   ├── adapter/otp/         # OTP 供应商适配器
│   ├── config/              # 环境配置
│   ├── database/            # PostgreSQL 连接池
│   ├── handler/             # HTTP 处理器
│   ├── middleware/          # 认证、CORS、限流
│   ├── model/               # 数据模型
│   ├── pkg/auth/            # JWT Token 服务
│   ├── pkg/errors/          # 错误类型
│   ├── repository/          # 数据访问（内存 + PostgreSQL）
│   ├── router/              # 路由定义与依赖注入
│   └── service/             # 业务逻辑
│       ├── verification/    # 无感验证 + OTP 编排
│       ├── risk/            # SIM Swap + 风控决策
│       ├── policy/          # 决策引擎
│       └── webhook/         # 事件投递
├── migrations/              # SQL 迁移
├── web/dashboard/           # Next.js 控制台
├── Dockerfile
├── docker-compose.yaml
└── Makefile
```

## 沙箱模式

平台内置沙箱适配器，模拟真实场景：

- **无感验证** — ~85% 成功率，200-500ms 延迟
- **SIM Swap 检测** — ~10% 阳性率
- **OTP 发送** — 验证码打印到控制台，通用测试码 `000000`

开发和测试无需任何外部服务。

## 覆盖国家

沙箱支持：印尼 (ID)、泰国 (TH)、菲律宾 (PH)、马来西亚 (MY)、新加坡 (SG)、越南 (VN)、巴西 (BR)、墨西哥 (MX)。

## 目标客户

- 金融科技 / 数字钱包 / 券商 / 交易所
- 高价值电商 / 交易平台
- 游戏与社交 App
- 出海互联网平台
- 企业账号安全场景

## 许可证

MIT
