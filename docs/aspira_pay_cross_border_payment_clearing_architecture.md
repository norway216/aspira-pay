# Aspira Pay 跨境支付及清算系统完整架构设计文档

**项目名称**：Aspira Pay  
**作者**：Aspira Studio  
**文档版本**：V3.0 Mature C++/Go/Web Edition  
**核心技术栈**：C++20 + Go + TypeScript Web Frontend  
**系统定位**：面向 B2B / B2B2C 场景的跨境支付、交易处理、清算结算、对账审计与联盟链证明系统  
**设计目标**：高性能、高稳定性、可审计、可回滚、可追溯、可监管、可扩展  

---

# 1. 系统总体定位

Aspira Pay 是一套成熟的跨境支付及清算系统，面向商户、支付机构、银行合作方、做市商、审计方和监管观察节点。

系统采用：

```text
C++ 高性能交易计算核心
+ Go 分布式支付服务
+ Web 管理后台
+ PostgreSQL 核心账务数据库
+ Kafka / NATS 事件总线
+ Redis 缓存与限流
+ MinIO / S3 加密对象存储
+ 联盟链审计与交易协作网络
+ Kubernetes 高可用部署
```

系统不以“完全去中心化支付”为目标，而是采用更加现实、合规、可落地的混合金融架构：

```text
真实资金划转：银行 / PSP / SWIFT / 本地支付网络 / 卡组织等中心化合规通道
交易协作证明：联盟链记录订单、报价、状态、回执哈希、审计、对账证明
敏感数据：链下加密存储
```

---

# 2. 核心设计原则

## 2.1 支付执行合规中心化

跨境支付涉及：

- 法币账户
- 银行清算
- 支付牌照
- 外汇监管
- KYC / KYB
- AML / CFT
- 税务报表
- 退款、拒付、冻结
- 司法协助

因此，真实资金划转必须走合规支付通道。

## 2.2 交易协作链上证明

以下内容适合使用联盟链增强可信度：

- 订单登记
- 报价承诺
- 汇率锁定
- 状态流转
- 支付回执哈希
- 审计事件哈希
- 对账报告哈希
- 争议处理状态
- 退款证明
- 批量 Merkle Root

## 2.3 敏感数据不上链

链上只保存：

```text
哈希
承诺
状态
索引
证明
签名
Merkle Root
```

链下保存：

```text
用户真实身份
银行账号
交易明细
KYC 文件
支付回执原文
对账报告原文
审计报告原文
```

## 2.4 交易系统优先保证正确性

对于支付系统来说，优先级应该是：

```text
正确性 > 一致性 > 可追溯性 > 稳定性 > 性能 > 开发速度
```

不能为了追求 TPS 牺牲账务准确性。

---

# 3. 总体架构

```text
┌──────────────────────────────────────────────────────────────────────┐
│                         Aspira Pay Web Frontend                      │
│ Merchant Portal / Admin Console / Risk Console / Chain Explorer      │
│ TypeScript + React / Vue + ECharts + WebSocket                       │
└──────────────────────────────────────┬───────────────────────────────┘
                                       │ HTTPS / WebSocket
                                       ▼
┌──────────────────────────────────────────────────────────────────────┐
│                         Go API Gateway                               │
│ TLS / mTLS / JWT / OAuth2 / HMAC / Ed25519 / RateLimit / WAF         │
└──────────────────────────────────────┬───────────────────────────────┘
                                       │ gRPC / HTTP
          ┌────────────────────────────┼────────────────────────────┐
          ▼                            ▼                            ▼
┌───────────────────┐        ┌───────────────────┐        ┌───────────────────┐
│ Go Payment API    │        │ Go Orchestrator   │        │ Go Merchant API   │
│ 下单/退款/状态查询 │        │ Saga / Workflow   │        │ 商户/密钥/Webhook │
└─────────┬─────────┘        └─────────┬─────────┘        └─────────┬─────────┘
          │                            │                            │
          └────────────────────────────┼────────────────────────────┘
                                       ▼
┌──────────────────────────────────────────────────────────────────────┐
│                    C++ Trading & Settlement Core                     │
│ FX Engine / Fee Engine / Route Engine / Risk Precheck / Ledger Calc  │
│ Lock-free Queue / Memory Snapshot / Batch Reconciliation / Merkle    │
└──────────────────────────────────────┬───────────────────────────────┘
                                       │
           ┌───────────────────────────┼───────────────────────────┐
           ▼                           ▼                           ▼
┌───────────────────┐       ┌───────────────────┐       ┌───────────────────┐
│ Go Risk Service   │       │ Go Compliance Svc │       │ Go Payment Exec   │
│ 风控规则/评分      │       │ KYC/KYB/AML       │       │ Bank/PSP/SWIFT    │
└─────────┬─────────┘       └─────────┬─────────┘       └─────────┬─────────┘
          │                           │                           │
          └───────────────────────────┼───────────────────────────┘
                                      ▼
┌──────────────────────────────────────────────────────────────────────┐
│                         Event Bus                                    │
│ Kafka / NATS: order.created, payment.executed, audit.created...      │
└──────────────────────────────────────┬───────────────────────────────┘
                                       │
          ┌────────────────────────────┼────────────────────────────┐
          ▼                            ▼                            ▼
┌───────────────────┐        ┌───────────────────┐        ┌───────────────────┐
│ PostgreSQL Cluster│        │ Redis Cluster     │        │ MinIO / S3        │
│ 订单/账务/状态     │        │ 限流/缓存/幂等     │        │ 回执/KYC/对账文件  │
└───────────────────┘        └───────────────────┘        └───────────────────┘
                                       │
                                       ▼
┌──────────────────────────────────────────────────────────────────────┐
│                    Aspira Consortium Chain                           │
│ OrderRegistry / QuoteCommitment / PaymentState / AuditLedger         │
│ SettlementProof / DisputeResolution / MerkleAnchor                   │
└──────────────────────────────────────┬───────────────────────────────┘
                                       │
                                       ▼
┌──────────────────────────────────────────────────────────────────────┐
│                Observability / Audit / Report / SIEM                 │
│ Prometheus / Grafana / OpenTelemetry / Jaeger / ELK / Loki           │
└──────────────────────────────────────────────────────────────────────┘
```

---

# 4. 技术栈选型

## 4.1 后端主要语言

| 语言 | 使用位置 | 说明 |
|---|---|---|
| C++20 | 交易计算、汇率、手续费、路由、批量对账、Merkle 计算 | 低延迟、高吞吐、可控内存 |
| Go | API 网关、支付服务、编排服务、风控合规、链上交互、运维服务 | 并发友好、适合微服务和云原生 |
| TypeScript | Web 前端、管理后台、商户平台 | 类型安全、适合复杂后台系统 |
| Python | 可选，用于离线风控、数据分析、模型训练 | 不作为核心支付链路 |

## 4.2 基础组件

| 类型 | 推荐技术 |
|---|---|
| API 协议 | REST + gRPC |
| 服务通信 | gRPC / HTTP/2 |
| 数据库 | PostgreSQL |
| 缓存 | Redis |
| 消息队列 | Kafka / NATS |
| 对象存储 | MinIO / S3 |
| 日志 | ELK / OpenSearch / Loki |
| 监控 | Prometheus + Grafana |
| 链路追踪 | OpenTelemetry + Jaeger |
| 容器 | Docker |
| 编排 | Kubernetes |
| CI/CD | GitHub Actions / GitLab CI / ArgoCD |
| 密钥 | Vault / KMS / HSM |
| 区块链 | Hyperledger Fabric / Besu / Tendermint / CometBFT |

---

# 5. 系统模块设计

---

# 5.1 Go API Gateway

## 5.1.1 作用

API Gateway 是系统统一入口，负责所有外部请求的安全接入。

## 5.1.2 核心功能

```text
HTTPS / TLS 1.3
mTLS 商户双向认证
JWT / OAuth2
HMAC-SHA256 请求签名
Ed25519 高安全签名
IP 白名单
WAF
限流
熔断
请求幂等
参数校验
请求路由
审计日志
```

## 5.1.3 典型接口

```http
POST /api/v1/quote
POST /api/v1/order/create
POST /api/v1/order/confirm
POST /api/v1/payment/execute
GET  /api/v1/payment/status/{payment_id}
POST /api/v1/refund
GET  /api/v1/reconciliation/report
GET  /api/v1/audit/proof/{order_id}
POST /api/v1/webhook/test
```

## 5.1.4 请求签名格式

```json
{
  "merchant_id": "MCH_10001",
  "request_id": "REQ_20260608_000001",
  "timestamp": 1780900000,
  "nonce": "random_string",
  "body_hash": "sha256(body)",
  "signature": "HMAC-SHA256 or Ed25519"
}
```

## 5.1.5 幂等键

```text
idempotency_key = merchant_id + request_id
```

数据库唯一索引：

```sql
CREATE UNIQUE INDEX uk_idempotency
ON idempotency_record(merchant_id, request_id);
```

---

# 5.2 Go Payment Service

## 5.2.1 作用

Payment Service 负责支付订单的创建、确认、状态查询、退款申请和回调处理。

## 5.2.2 主要职责

```text
创建支付订单
确认支付订单
查询支付状态
处理退款请求
处理支付通道回调
维护订单状态
写入 Outbox 事件
调用 C++ 交易计算核心
调用 Go Orchestrator 编排支付流程
```

## 5.2.3 支付订单状态

```text
CREATED
QUOTE_LOCKED
COMPLIANCE_PRECHECKED
PAYMENT_PENDING
PAYMENT_EXECUTING
PAYMENT_CONFIRMED
SETTLEMENT_PROOFED
RECONCILED
CLOSED
```

异常状态：

```text
RISK_REJECTED
PAYMENT_FAILED
REFUND_PENDING
REFUNDED
DISPUTED
FROZEN
MANUAL_REVIEW
CANCELLED
```

## 5.2.4 状态转移约束

| 当前状态 | 允许下一状态 |
|---|---|
| CREATED | QUOTE_LOCKED / CANCELLED |
| QUOTE_LOCKED | COMPLIANCE_PRECHECKED / RISK_REJECTED |
| COMPLIANCE_PRECHECKED | PAYMENT_PENDING / MANUAL_REVIEW |
| PAYMENT_PENDING | PAYMENT_EXECUTING / CANCELLED |
| PAYMENT_EXECUTING | PAYMENT_CONFIRMED / PAYMENT_FAILED |
| PAYMENT_CONFIRMED | SETTLEMENT_PROOFED |
| SETTLEMENT_PROOFED | RECONCILED / DISPUTED |
| RECONCILED | CLOSED |
| REFUND_PENDING | REFUNDED / DISPUTED |

---

# 5.3 Go Orchestrator 编排服务

## 5.3.1 作用

Orchestrator 是跨境支付流程的大脑，负责 Saga 流程编排。

## 5.3.2 Saga 主流程

```text
CreateOrder
  ↓
CalculateQuote
  ↓
LockQuoteOnChain
  ↓
CompliancePrecheck
  ↓
RiskPrecheck
  ↓
CreatePaymentInstruction
  ↓
ExecutePayment
  ↓
ReceivePaymentCallback
  ↓
WriteSettlementProof
  ↓
Reconcile
  ↓
CloseOrder
```

## 5.3.3 补偿流程

| 失败位置 | 补偿策略 |
|---|---|
| 报价失败 | 订单取消 |
| 报价上链失败 | Outbox 重试 |
| 合规拒绝 | RISK_REJECTED |
| 风控高风险 | MANUAL_REVIEW / FROZEN |
| 支付执行失败 | PAYMENT_FAILED |
| 支付成功但写链失败 | 持续重试写链 |
| 数据库失败 | 从 Outbox / 链上事件恢复 |
| 对账失败 | DISPUTED |
| 退款失败 | REFUND_PENDING + 告警 |

---

# 5.4 C++ Trading & Settlement Core

## 5.4.1 作用

C++ 核心模块负责高性能、低延迟、计算密集型任务。

## 5.4.2 子模块

```text
C++ Trading & Settlement Core
├── Request Decoder
├── Lock-Free Ring Buffer
├── FX Rate Engine
├── Fee Calculation Engine
├── Liquidity Route Engine
├── Risk Precheck Engine
├── Limit Check Engine
├── Ledger Calculation Engine
├── Batch Reconciliation Engine
├── Merkle Tree Builder
├── State Validation Engine
├── Result Publisher
└── Performance Metrics Exporter
```

## 5.4.3 模块职责

| 模块 | 职责 |
|---|---|
| FX Rate Engine | 汇率快照、汇率计算、滑点控制 |
| Fee Engine | 手续费规则计算 |
| Route Engine | 根据币种、国家、通道成本选择支付路径 |
| Risk Precheck | 交易金额、频率、黑名单、通道风险预检查 |
| Ledger Calculation | 生成复式记账分录 |
| Batch Reconciliation | 批量对账计算 |
| Merkle Builder | 批量事件 Merkle Root 计算 |
| State Validator | 校验状态转移是否合法 |

## 5.4.4 性能设计

```text
使用 C++20
使用 Protobuf 定义消息
使用 gRPC 与 Go 服务通信
内部采用 lock-free ring buffer
热点配置使用内存快照
汇率表使用 copy-on-write
读多写少数据使用 RCU 思想
批量任务异步处理
避免动态频繁分配内存
使用对象池和内存池
使用 perf / FlameGraph 分析性能
```

## 5.4.5 C++ 与 Go 通信

推荐：

```text
Go Service  ── gRPC/Protobuf ──> C++ Engine
C++ Engine ── gRPC/Protobuf ──> Go Service
```

也可以在极低延迟场景使用：

```text
Shared Memory + Ring Buffer
Unix Domain Socket
ZeroMQ
NATS
```

初期推荐 gRPC，工程复杂度较低。

---

# 5.5 账务 Ledger Service

## 5.5.1 账务原则

支付系统必须采用类似复式记账的账务模型。

核心原则：

```text
每笔账务都有凭证
每个凭证包含多条分录
借贷必须平衡
账户余额不可随意直接修改
所有变更必须可追溯
所有修正通过反向分录完成
```

## 5.5.2 账户类型

```text
USER_AVAILABLE_ACCOUNT
USER_FROZEN_ACCOUNT
MERCHANT_SETTLEMENT_ACCOUNT
PLATFORM_FEE_ACCOUNT
CHANNEL_CLEARING_ACCOUNT
FX_GAIN_LOSS_ACCOUNT
REFUND_ACCOUNT
RESERVE_ACCOUNT
```

## 5.5.3 示例：用户付款 100，手续费 2

```text
用户可用账户            -100
商户待结算账户           +98
平台手续费收入账户        +2
```

## 5.5.4 数据表设计

```sql
CREATE TABLE account (
    id BIGSERIAL PRIMARY KEY,
    account_no VARCHAR(64) UNIQUE NOT NULL,
    owner_type VARCHAR(32) NOT NULL,
    owner_id VARCHAR(64) NOT NULL,
    account_type VARCHAR(64) NOT NULL,
    currency VARCHAR(16) NOT NULL,
    status VARCHAR(32) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE ledger_voucher (
    id BIGSERIAL PRIMARY KEY,
    voucher_no VARCHAR(64) UNIQUE NOT NULL,
    business_type VARCHAR(64) NOT NULL,
    business_id VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE ledger_entry (
    id BIGSERIAL PRIMARY KEY,
    voucher_no VARCHAR(64) NOT NULL,
    account_no VARCHAR(64) NOT NULL,
    direction VARCHAR(16) NOT NULL,
    amount NUMERIC(30, 8) NOT NULL,
    currency VARCHAR(16) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE account_balance (
    account_no VARCHAR(64) PRIMARY KEY,
    available_balance NUMERIC(30, 8) NOT NULL,
    frozen_balance NUMERIC(30, 8) NOT NULL,
    currency VARCHAR(16) NOT NULL,
    version BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);
```

## 5.5.5 原子扣减

```sql
UPDATE account_balance
SET available_balance = available_balance - :amount,
    frozen_balance = frozen_balance + :amount,
    version = version + 1
WHERE account_no = :account_no
  AND available_balance >= :amount;
```

检查影响行数：

```text
影响 1 行：冻结成功
影响 0 行：余额不足或账户异常
```

---

# 5.6 清算 Settlement Service

## 5.6.1 清算和结算的区别

```text
清算 Clearing：计算各方应该收付多少钱
结算 Settlement：真正完成资金划转
```

## 5.6.2 清算类型

```text
T+0 实时清算
T+1 批量清算
T+2 跨境结算
按商户清算
按通道清算
按币种清算
按国家/地区清算
手续费清算
退款冲正清算
```

## 5.6.3 清算流程

```text
交易完成
  ↓
生成账务分录
  ↓
进入待清算池
  ↓
按商户 / 币种 / 通道聚合
  ↓
生成清算批次
  ↓
生成结算指令
  ↓
调用银行 / PSP 执行结算
  ↓
保存结算回执
  ↓
更新清算状态
  ↓
回执哈希写入链上
```

## 5.6.4 清算批次状态

```text
BATCH_CREATED
BATCH_CALCULATED
BATCH_APPROVED
SETTLEMENT_INSTRUCTED
SETTLEMENT_PROCESSING
SETTLEMENT_CONFIRMED
RECONCILED
BATCH_CLOSED
BATCH_FAILED
```

---

# 5.7 Reconciliation 对账服务

## 5.7.1 三账对账

Aspira Pay 采用三账对账：

```text
内部订单账
+ 支付通道账
+ 链上状态账
= 最终对账结果
```

## 5.7.2 对账数据源

```text
内部订单表
账务流水表
银行 / PSP 回执
银行流水文件
卡组织清算文件
链上 PaymentState
链上 SettlementProof
通道回调日志
```

## 5.7.3 对账维度

```text
order_id
payment_id
merchant_id
amount
currency
fee
fx_rate
channel
status
receipt_hash
settlement_date
created_at
confirmed_at
```

## 5.7.4 异常类型

| 异常 | 处理 |
|---|---|
| 内部成功，通道失败 | 冲正 / 退款 / 人工审核 |
| 通道成功，内部失败 | 事件回放恢复 |
| 通道成功，链上未确认 | 重试写链 |
| 链上成功，数据库缺失 | 根据链上事件恢复 |
| 金额不一致 | 冻结进入人工审核 |
| 币种不一致 | 阻断清算 |
| 重复回调 | 幂等忽略 |
| 超时无回执 | 主动轮询 + 告警 |

---

# 5.8 Go Payment Executor 支付执行层

## 5.8.1 作用

Payment Executor 负责连接真实支付通道。

## 5.8.2 支付通道

```text
Bank Connector
PSP Connector
SWIFT Connector
Card Network Connector
Local Payment Rail Connector
Stablecoin Issuer Connector
FX Provider Connector
```

## 5.8.3 Connector 设计

每个通道实现统一接口：

```go
type PaymentConnector interface {
    CreatePayment(ctx context.Context, req PaymentRequest) (PaymentResponse, error)
    QueryPayment(ctx context.Context, paymentID string) (PaymentStatus, error)
    Refund(ctx context.Context, req RefundRequest) (RefundResponse, error)
    QueryRefund(ctx context.Context, refundID string) (RefundStatus, error)
}
```

## 5.8.4 通道可靠性

```text
超时控制
重试策略
熔断
限流
通道降级
主动轮询
回调验签
回调幂等
通道健康检查
```

---

# 5.9 Risk Service 风控服务

## 5.9.1 风控类型

```text
交易金额风险
交易频率风险
设备风险
IP / 地理位置风险
商户风险
国家 / 地区风险
通道风险
汇率波动风险
黑名单风险
异常行为风险
```

## 5.9.2 风控规则示例

```text
单笔金额超过限额 → MANUAL_REVIEW
新商户大额交易 → 延迟结算
高风险地区 IP → 拒绝
短时间多笔相同金额 → 风险提升
设备指纹异常 → 二次验证
命中制裁名单 → 拒绝并上报
汇率波动超过阈值 → 暂停报价
```

## 5.9.3 技术实现

```text
Go 实时规则引擎
C++ 高性能预检查
Redis 保存短期频率计数
Kafka 接收实时交易事件
PostgreSQL 保存风控结果
ClickHouse 做离线分析
Python 可选用于模型训练
```

---

# 5.10 Compliance Service 合规服务

## 5.10.1 合规能力

```text
KYC 个人认证
KYB 企业认证
AML 反洗钱
CFT 反恐融资
Sanctions 制裁名单筛查
PEP 政治公众人物筛查
Travel Rule
交易限额控制
可疑交易报告
监管报表
资金冻结接口
司法协助接口
```

## 5.10.2 合规结果上链

链上不保存合规明文，只保存：

```json
{
  "order_id": "ORD_xxx",
  "compliance_result_hash": "0xabc...",
  "risk_level": "LOW",
  "review_required": false,
  "timestamp": 1780900000
}
```

---

# 5.11 Blockchain Service 区块链协作层

## 5.11.1 Aspira Consortium Chain

Aspira Consortium Chain 是 Aspira Pay 的联盟链交易协作与审计账本。

参与节点：

```text
Aspira Core Node
Bank Partner Node
PSP Partner Node
FX Provider Node
Liquidity Provider Node
Auditor Node
Regulator Observer Node
Disaster Recovery Node
```

## 5.11.2 链上合约

```text
DIDRegistry
MerchantRegistry
QuoteCommitment
OrderRegistry
PaymentStateMachine
SettlementProofRegistry
AuditLedger
DisputeResolution
RefundRegistry
MerkleAnchor
```

## 5.11.3 链上记录内容

| 数据 | 是否上链 | 说明 |
|---|---:|---|
| order_id | 是 | 订单唯一标识 |
| merchant_hash | 是 | 商户哈希 |
| customer_hash | 是 | 用户哈希 |
| amount_commitment | 是 | 金额承诺 |
| quote_id | 是 | 报价 ID |
| quote_commitment | 是 | 报价承诺 |
| payment_state | 是 | 支付状态 |
| receipt_hash | 是 | 支付回执哈希 |
| audit_event_hash | 是 | 审计事件哈希 |
| reconciliation_hash | 是 | 对账报告哈希 |
| KYC 明文 | 否 | 链下加密存储 |
| 银行账号 | 否 | 链下加密存储 |
| 交易明细全文 | 否 | 链下加密存储 |

## 5.11.4 联盟链用途

```text
防止订单状态被篡改
防止报价反悔
增强多方对账可信度
提高审计可信度
支持监管观察节点
支持争议处理
支持跨机构协作
```

---

# 5.12 Audit Service 审计服务

## 5.12.1 审计内容

```text
登录审计
权限变更审计
商户操作审计
管理员操作审计
支付状态变更审计
账务分录审计
清算批次审计
风控决策审计
合规审核审计
链上写入审计
密钥操作审计
```

## 5.12.2 审计哈希链

每个审计事件包含上一个事件哈希：

```json
{
  "audit_id": "AUD_001",
  "order_id": "ORD_001",
  "event_type": "PAYMENT_CONFIRMED",
  "payload_hash": "0xabc",
  "prev_event_hash": "0xdef",
  "operator_hash": "0x123",
  "timestamp": 1780900000
}
```

批量审计事件生成 Merkle Root 后写入链上。

---

# 5.13 Web Frontend 前端系统

## 5.13.1 技术栈

```text
TypeScript
React / Vue
Vite / Next.js
Tailwind CSS
Ant Design / Arco Design
ECharts
WebSocket
OpenAPI Client
```

## 5.13.2 商户后台 Merchant Portal

```text
Overview Dashboard
交易列表
交易详情
创建支付
退款管理
结算账户
对账单下载
API Key 管理
Webhook 设置
开发者文档
审计证明查询
```

## 5.13.3 管理后台 Admin Console

```text
实时交易监控
支付通道监控
清算批次管理
对账中心
退款中心
争议中心
风控案件
合规审核
商户管理
用户管理
权限管理
审计日志
区块链浏览器
节点管理
密钥管理
系统告警
```

## 5.13.4 风控后台 Risk Console

```text
风险交易列表
人工审核队列
规则配置
黑名单管理
设备指纹查询
IP 风险分析
商户风险画像
风控命中详情
模型评分解释
```

---

# 6. 数据库设计

## 6.1 核心表

```text
merchant
merchant_api_key
customer
payment_order
payment_instruction
payment_channel
payment_receipt
refund_order
settlement_batch
settlement_detail
reconciliation_record
account
account_balance
ledger_voucher
ledger_entry
risk_case
compliance_case
audit_event
outbox_event
idempotency_record
webhook_event
```

## 6.2 payment_order 表

```sql
CREATE TABLE payment_order (
    id BIGSERIAL PRIMARY KEY,
    order_id VARCHAR(64) UNIQUE NOT NULL,
    merchant_id VARCHAR(64) NOT NULL,
    customer_id_hash VARCHAR(128),
    source_currency VARCHAR(16) NOT NULL,
    target_currency VARCHAR(16) NOT NULL,
    source_amount NUMERIC(30, 8) NOT NULL,
    target_amount NUMERIC(30, 8),
    fee_amount NUMERIC(30, 8),
    fx_rate NUMERIC(30, 12),
    status VARCHAR(64) NOT NULL,
    quote_id VARCHAR(64),
    payment_id VARCHAR(64),
    channel_id VARCHAR(64),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);
```

## 6.3 outbox_event 表

```sql
CREATE TABLE outbox_event (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(64) UNIQUE NOT NULL,
    aggregate_type VARCHAR(64) NOT NULL,
    aggregate_id VARCHAR(64) NOT NULL,
    event_type VARCHAR(128) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    retry_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);
```

## 6.4 idempotency_record 表

```sql
CREATE TABLE idempotency_record (
    id BIGSERIAL PRIMARY KEY,
    merchant_id VARCHAR(64) NOT NULL,
    request_id VARCHAR(128) NOT NULL,
    request_hash VARCHAR(128) NOT NULL,
    response_payload JSONB,
    status VARCHAR(32) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    UNIQUE (merchant_id, request_id)
);
```

---

# 7. 消息队列设计

## 7.1 Kafka / NATS Topic

```text
quote.requested
quote.generated
quote.committed.onchain
order.created
order.confirmed
order.state.updated
risk.prechecked
compliance.prechecked
payment.instruction.created
payment.executing
payment.executed
payment.callback.received
payment.failed
settlement.batch.created
settlement.executed
settlement.proof.created
reconciliation.started
reconciliation.completed
refund.requested
refund.executed
refund.completed
audit.event.created
blockchain.write.requested
blockchain.write.completed
webhook.dispatch.requested
webhook.dispatch.completed
```

## 7.2 事件原则

```text
每个事件有 event_id
每个事件绑定 order_id 或 payment_id
消费者必须幂等
消息失败进入死信队列
重要业务使用 Outbox Pattern
事件可重放
事件可用于恢复读模型
```

---

# 8. 完整支付流程

## 8.1 报价流程

```text
1. 商户请求报价
2. API Gateway 验签
3. Go Payment Service 创建报价请求
4. C++ FX Engine 计算候选汇率
5. C++ Fee Engine 计算手续费
6. C++ Route Engine 选择候选通道
7. 返回报价给商户
8. QuoteCommitment 写入联盟链
9. 报价进入可确认状态
```

## 8.2 下单流程

```text
1. 商户确认报价
2. 系统创建 payment_order
3. 写入幂等记录
4. 写入 outbox_event
5. OrderRegistry 创建链上订单
6. PaymentStateMachine 更新为 QUOTE_LOCKED
7. 风控和合规预审
8. 通过后进入 PAYMENT_PENDING
```

## 8.3 支付执行流程

```text
1. Orchestrator 读取 PAYMENT_PENDING 订单
2. 创建支付指令
3. Payment Executor 选择支付通道
4. 调用银行 / PSP / SWIFT / 本地支付网络
5. 状态更新为 PAYMENT_EXECUTING
6. 等待通道回调或主动轮询
7. 成功后状态更新为 PAYMENT_CONFIRMED
8. 回执加密存储到 MinIO / S3
9. 回执哈希写入 SettlementProofRegistry
```

## 8.4 清算流程

```text
1. 支付成功订单进入待清算池
2. 按商户、币种、通道、日期聚合
3. C++ Settlement Engine 计算清算结果
4. 生成 settlement_batch
5. 财务或系统自动审批
6. Payment Executor 发起结算指令
7. 保存结算回执
8. 更新清算状态
9. 结算证明哈希写入链上
```

## 8.5 对账流程

```text
1. 导入银行 / PSP / 卡组织对账文件
2. 读取内部订单和账务流水
3. 读取链上状态和回执哈希
4. 三账匹配
5. 生成 reconciliation_record
6. 异常订单进入 DISPUTED
7. 对账报告加密存储
8. 报告哈希写入 AuditLedger
```

## 8.6 退款流程

```text
1. 商户发起退款
2. 幂等检查
3. 风控检查
4. 创建 refund_order
5. RefundRegistry 写入链上
6. Payment Executor 调用通道退款
7. 保存退款回执
8. 账务生成反向分录
9. 退款回执哈希写入链上
10. 状态更新为 REFUNDED
```

---

# 9. 安全架构

## 9.1 通信安全

```text
外部 API 使用 TLS 1.3
高等级商户使用 mTLS
内部服务使用 mTLS
服务间通信使用短期证书
Webhook 必须验签
所有请求带 timestamp + nonce 防重放
```

## 9.2 数据加密

```text
敏感字段 AES-256-GCM 加密
对象文件先加密再上传
数据库保存密文和 key_id
密钥由 KMS / HSM 管理
日志必须脱敏
链上不存敏感明文
```

## 9.3 密钥管理

```text
Vault / KMS / HSM
Merchant API Key
Webhook Signing Key
JWT Signing Key
Service Signing Key
Blockchain Node Key
Data Encryption Key
Receipt Encryption Key
```

## 9.4 权限管理

```text
RBAC
ABAC
四眼原则
敏感操作二次确认
密钥操作审批
合约升级多签
财务清算审批
管理员操作审计
```

---

# 10. 高可用与灾备

## 10.1 部署原则

```text
所有核心服务多副本
API Gateway 水平扩展
C++ Engine 按交易类型或币种分片
Kafka 至少 3 节点
PostgreSQL 主从 + 自动故障切换
Redis Cluster
MinIO 多副本
联盟链至少 4 个 BFT 节点
跨可用区部署
核心数据异地备份
```

## 10.2 故障恢复

| 故障 | 恢复方式 |
|---|---|
| Go 服务宕机 | Kubernetes 自动重启 |
| C++ 引擎宕机 | 服务降级 + 副本接管 |
| PostgreSQL 主库故障 | 自动切换到从库 |
| Kafka 堆积 | 扩容消费者 |
| Kafka 消息丢失 | Outbox 表重放 |
| 链上写入失败 | blockchain.write 重试 |
| 支付回调丢失 | 主动轮询通道 |
| 对象存储故障 | 多副本恢复 |
| 对账异常 | 进入争议处理 |

---

# 11. 监控与可观测性

## 11.1 核心指标

```text
TPS
成功率
错误率
P50 / P95 / P99 延迟
支付通道成功率
支付通道回调延迟
数据库连接数
慢 SQL 数量
Kafka 消费堆积
Redis 命中率
链上写入延迟
链上交易失败率
对账差异数量
退款失败率
风控拒绝率
合规人工审核数量
```

## 11.2 日志字段

每条日志必须包含：

```text
trace_id
span_id
request_id
merchant_id
order_id
payment_id
event_id
service_name
latency_ms
error_code
```

## 11.3 告警规则

```text
支付成功率低于 99%
P99 延迟超过阈值
Kafka 堆积超过阈值
数据库连接池耗尽
支付通道连续失败
链上写入失败率升高
对账差异数量异常
退款失败率异常
风控命中率异常波动
```

---

# 12. Web 前端页面设计

## 12.1 商户端页面

```text
Dashboard
Payments
Payment Detail
Refunds
Settlements
Reconciliation Reports
API Keys
Webhook Settings
Developer Docs
Audit Proof Query
```

## 12.2 运营管理端页面

```text
Global Dashboard
Live Transactions
Merchant Management
Channel Management
Risk Cases
Compliance Review
Settlement Batches
Reconciliation Center
Refund Center
Dispute Center
Ledger Viewer
Audit Logs
Chain Explorer
System Alerts
Key Management
User & Role Management
```

## 12.3 设计风格

```text
深墨蓝 / 深海军蓝为主色
冰蓝 / 科技蓝作为强调色
少量蓝紫渐变
布局克制
偏 B2B 金融基础设施风格
避免过度娱乐化
强调可信、稳定、专业
```

---

# 13. API 设计示例

## 13.1 创建报价

```http
POST /api/v1/quote
Content-Type: application/json
X-Merchant-Id: MCH_10001
X-Request-Id: REQ_001
X-Signature: xxx
```

```json
{
  "source_currency": "USD",
  "target_currency": "EUR",
  "source_amount": "1000.00",
  "target_country": "DE",
  "payment_method": "BANK_TRANSFER"
}
```

## 13.2 返回报价

```json
{
  "quote_id": "QTE_20260608_000001",
  "source_currency": "USD",
  "target_currency": "EUR",
  "source_amount": "1000.00",
  "target_amount": "920.00",
  "fx_rate": "0.920000",
  "fee": "5.00",
  "expire_at": "2026-06-08T12:05:00Z"
}
```

## 13.3 创建支付订单

```http
POST /api/v1/order/create
```

```json
{
  "quote_id": "QTE_20260608_000001",
  "merchant_order_id": "SHOP_ORDER_001",
  "receiver": {
    "name": "Receiver Name",
    "country": "DE",
    "bank_account_token": "tok_xxx"
  }
}
```

## 13.4 查询支付状态

```http
GET /api/v1/payment/status/PAY_20260608_000001
```

```json
{
  "payment_id": "PAY_20260608_000001",
  "order_id": "ORD_20260608_000001",
  "status": "PAYMENT_CONFIRMED",
  "chain_proof": {
    "receipt_hash": "0xabc...",
    "block_height": 102400
  }
}
```

---

# 14. 项目目录结构建议

```text
aspira-pay/
├── backend/
│   ├── go-api-gateway/
│   ├── go-payment-service/
│   ├── go-orchestrator/
│   ├── go-risk-service/
│   ├── go-compliance-service/
│   ├── go-payment-executor/
│   ├── go-blockchain-service/
│   ├── go-audit-service/
│   └── go-webhook-service/
├── cpp-core/
│   ├── trading-engine/
│   ├── fx-engine/
│   ├── fee-engine/
│   ├── route-engine/
│   ├── ledger-engine/
│   ├── reconciliation-engine/
│   ├── merkle-builder/
│   └── common/
├── contracts/
│   ├── OrderRegistry/
│   ├── QuoteCommitment/
│   ├── PaymentStateMachine/
│   ├── SettlementProofRegistry/
│   ├── AuditLedger/
│   └── MerkleAnchor/
├── frontend/
│   ├── merchant-portal/
│   ├── admin-console/
│   ├── risk-console/
│   └── chain-explorer/
├── deployments/
│   ├── docker/
│   ├── kubernetes/
│   ├── helm/
│   └── argocd/
├── docs/
│   ├── architecture/
│   ├── api/
│   ├── database/
│   ├── security/
│   └── operations/
└── scripts/
    ├── database/
    ├── test/
    └── deploy/
```

---

# 15. 开发路线图

## 第一阶段：可运行 MVP

目标：完成基础支付网关和订单状态机。

```text
Go API Gateway
Go Payment Service
PostgreSQL
Redis
基础幂等
基础订单状态机
Web Merchant Portal
模拟支付通道
基础日志和监控
```

## 第二阶段：交易计算核心

目标：引入 C++ 高性能计算。

```text
C++ FX Engine
C++ Fee Engine
C++ Route Engine
gRPC 通信
Kafka / NATS 事件总线
Outbox Pattern
基础对账
```

## 第三阶段：账务和清算

目标：形成金融级账务系统。

```text
Ledger Service
复式记账
账户余额
冻结/解冻
清算批次
退款反向分录
对账中心
```

## 第四阶段：风控合规

目标：具备支付机构级风控能力。

```text
KYC / KYB
AML / Sanctions
风控规则引擎
人工审核
黑名单
交易限额
合规报表
```

## 第五阶段：联盟链审计

目标：增强多机构可信协作。

```text
Aspira Consortium Chain
OrderRegistry
QuoteCommitment
PaymentStateMachine
SettlementProofRegistry
AuditLedger
MerkleAnchor
链上浏览器
```

## 第六阶段：生产级高可用

目标：接近真实生产可用。

```text
Kubernetes
多副本部署
灰度发布
Prometheus / Grafana
OpenTelemetry / Jaeger
ELK / Loki
Vault / KMS
灾备恢复
混沌测试
安全测试
```

---

# 16. 测试方案

## 16.1 功能测试

```text
报价创建
订单创建
支付执行
支付回调
退款
清算批次
对账
链上写入
审计查询
Webhook 通知
```

## 16.2 压力测试

```text
20 并发稳定性测试
100 并发吞吐测试
500 并发容量测试
1000+ 并发扩展测试
P95 / P99 延迟分析
错误率分析
数据库瓶颈分析
Kafka 堆积分析
```

## 16.3 金融一致性测试

```text
重复请求不会重复扣款
支付成功不会丢失账务
退款不会重复执行
借贷分录必须平衡
数据库失败可恢复
Kafka 失败可重放
链上写入失败可补偿
对账异常可追踪
```

## 16.4 安全测试

```text
签名错误测试
重放攻击测试
越权访问测试
SQL 注入测试
XSS 测试
CSRF 测试
密钥泄露模拟
Webhook 伪造测试
权限绕过测试
```

---

# 17. 当前 benchmark 优化建议

如果当前系统出现：

```text
Success Rate: 22.97%
Total Errors: 5610
Avg TPS: 241.5
P95: 462ms
P99: 706ms
```

说明系统已经在低并发下大量失败。

优先优化顺序：

```text
1. 统计错误类型：4xx / 5xx / timeout / db error / connection error
2. 增加 request_id、order_id、trace_id
3. 检查服务端 panic 和错误日志
4. 检查数据库连接池
5. 检查慢 SQL 和锁等待
6. 缩短数据库事务
7. 增加 Kafka / NATS 削峰
8. 引入 Outbox Pattern
9. 接口快速返回 PENDING
10. 后台异步处理支付执行
11. 增加限流和熔断
12. 建立 P95 / P99 监控
```

目标分阶段：

```text
阶段 1：20 并发成功率 > 99%
阶段 2：100 并发成功率 > 99%，TPS > 1000
阶段 3：500 并发成功率 > 99.5%
阶段 4：1000+ 并发，进入分布式扩展
```

---

# 18. 关键工程风险

## 18.1 技术风险

```text
账务一致性设计不足
幂等控制不完整
长事务导致锁冲突
支付回调重复处理
Kafka 消费重复导致状态错误
链上写入失败未补偿
C++ 引擎崩溃影响主链路
数据库索引设计不足
```

## 18.2 合规风险

```text
未完成 KYC / KYB
未做 AML / 制裁名单筛查
敏感数据上链
日志中泄露个人信息
跨境支付通道不合规
未保留审计证据
退款争议流程不完整
```

## 18.3 安全风险

```text
API Key 泄露
Webhook 被伪造
签名算法不安全
密钥明文落盘
管理员越权操作
数据库未加密
对象存储权限过大
合约升级权限过大
```

---

# 19. 最终系统能力总结

成熟版 Aspira Pay 应该具备：

```text
高并发支付 API
高性能 C++ 交易计算核心
Go 微服务编排
支付订单状态机
幂等和防重复扣款
复式记账账务系统
跨境清算批次
三账对账系统
退款和冲正机制
风控与合规审核
支付通道 Connector
联盟链审计证明
链上状态机与回执哈希
加密对象存储
KMS / HSM 密钥管理
Web 商户后台
Web 管理后台
Web 风控后台
全链路追踪
Prometheus / Grafana 监控
Kubernetes 高可用部署
灾备与恢复机制
```

---

# 20. 一句话总结

Aspira Pay 的成熟架构不是一个简单的支付接口，而是：

```text
C++ 高性能交易与清算核心
+ Go 分布式支付网关与编排服务
+ TypeScript Web 管理平台
+ PostgreSQL 账务数据库
+ Kafka / NATS 事件驱动
+ Redis 缓存限流
+ 联盟链审计与交易协作
+ 风控合规与安全加密
+ Kubernetes 高可用基础设施
```

最终目标是构建一套可扩展、可审计、可监管、可对账、可恢复的跨境支付金融基础设施。
