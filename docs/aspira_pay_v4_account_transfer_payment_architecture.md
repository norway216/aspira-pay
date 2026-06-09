# Aspira Pay V4 架构设计文档

**版本**：V4.0  
**作者**：Aspira Studio  
**定位**：在 Aspira Pay V3 跨境支付、银行卡支付、多币种账户、交易清算、风控审计体系基础上，新增账户之间的主动转账、收款、QR 码收款、支付链接、历史收款账户记录能力。  
**当前实现重点**：优先实现主动转账与通过有效支付链接稍后转账；QR 码收款与扫码支付作为接口和架构预留。

---

## 1. V4 设计目标

Aspira Pay V3 已具备完整的支付系统基础，包括：

- 用户注册、登录、KYC / AML；
- 多币种钱包账户；
- 银行卡绑定、虚拟卡/实体卡申请与卡支付；
- 跨境支付、实时汇率、手续费扣除；
- 交易冻结、扣款、入账、退款、冲正；
- 风控、审计、对账、联盟链审计凭证；
- C++20 高性能交易核心 + Go 分布式服务 + TypeScript Web / Android 客户端。

V4 在此基础上新增面向个人和商户的“账户之间交易能力”：

1. 主动转账：用户从自己的 Aspira Pay 账户向另一个 Aspira Pay 账户转账。
2. 收款：收款方生成收款请求，付款方完成付款。
3. 支付链接：收款方生成一个带有效期的支付链接，付款方可稍后在有效期内支付。
4. QR 码收款：收款方生成 QR 码，付款方扫码后进入付款确认页。
5. 历史转账账户：记录用户历史转账对象，方便下次快速选择。
6. 交易审计：所有转账、收款请求、链接生成、链接访问、支付确认、状态变化均进入审计链路。

> V4 第一阶段只实现：主动转账、支付链接稍后转账、历史转账账户记录。  
> QR 码收款、扫码支付、商户收款作为 V4.1 / V4.2 预留能力。

---

## 2. V4 功能范围

### 2.1 本期必须实现的功能

| 功能 | 说明 | 优先级 |
|---|---|---|
| 账户间主动转账 | 用户主动输入对方账户、手机号、邮箱、Aspira ID 或历史收款人进行转账 | P0 |
| 转账确认页 | 显示收款人、金额、币种、手续费、汇率、预计到账金额 | P0 |
| 转账交易处理 | 账户余额校验、冻结、扣款、入账、失败回滚 | P0 |
| 历史转账账户 | 成功转账后记录收款人账户，支持下次快速选择 | P0 |
| 支付链接生成 | 收款方生成指定金额、币种、备注、有效期的支付链接 | P0 |
| 支付链接支付 | 付款方打开链接，在有效期内完成付款 | P0 |
| 支付链接状态管理 | pending / paid / expired / cancelled / failed | P0 |
| 链接有效期校验 | 超过有效期后不可支付 | P0 |
| 风控与限额 | 对转账金额、频率、账户状态、KYC 等级做限制 | P0 |
| 审计与交易流水 | 每一步交易状态变化可追踪、可对账、可审计 | P0 |

### 2.2 本期暂不完整实现但预留接口的功能

| 功能 | 说明 | 预留方式 |
|---|---|---|
| QR 码收款 | 收款方展示 QR 码，付款方扫码进入付款页 | Payment Link Token 可直接编码成 QR |
| 扫码支付 | Android App 使用摄像头扫描 QR | 预留 Scan API 与 Deeplink |
| 商户静态收款码 | 商户生成长期固定收款码 | 预留 merchant_payment_profile |
| 一次性动态付款码 | 类似支付宝付款码，付款方展示动态码 | V4 暂不实现 |
| NFC / Apple Pay / Google Pay | 近场支付 | V5 之后考虑 |

---

## 3. 整体架构

```text
┌───────────────────────────────────────────────────────────────┐
│                        Android / Web Client                    │
│  Home / Pay / Wallet / Activity / Profile                      │
│  Transfer / Request Money / Payment Link / QR Receive          │
└───────────────────────┬───────────────────────────────────────┘
                        │ HTTPS + TLS + JWT / OAuth2
┌───────────────────────▼───────────────────────────────────────┐
│                         API Gateway                            │
│  Auth / Rate Limit / Signature / Device Check / Idempotency     │
└───────────────────────┬───────────────────────────────────────┘
                        │
┌───────────────────────▼───────────────────────────────────────┐
│                    Aspira Pay Go Service Layer                  │
│                                                               │
│  User Service        Account Service       Transfer Service     │
│  Payment Link Svc    QR Receive Service    Risk Service         │
│  Notification Svc    Audit Service         Activity Service     │
└───────────────────────┬───────────────────────────────────────┘
                        │ gRPC / NATS / Kafka
┌───────────────────────▼───────────────────────────────────────┐
│              C++20 High Performance Transaction Engine          │
│                                                               │
│  Ledger Engine       Balance Engine       Fee Engine            │
│  FX Engine           Settlement Engine    Reversal Engine       │
└───────────────────────┬───────────────────────────────────────┘
                        │
┌───────────────────────▼───────────────────────────────────────┐
│                         Data Layer                             │
│                                                               │
│  PostgreSQL / Redis / Kafka / MinIO / ClickHouse / Audit Chain  │
└───────────────────────────────────────────────────────────────┘
```

V4 的关键原则是：客户端只负责发起请求和确认交易，真正的余额变更必须由服务端交易引擎完成。任何客户端传入的金额、手续费、汇率、收款人信息都不能直接信任，服务端必须重新计算和校验。

---

## 4. 核心业务对象

### 4.1 用户 User

用户是 Aspira Pay 的登录主体。

```text
User
- user_id
- aspira_id
- phone
- email
- kyc_level
- status: active / frozen / restricted / closed
- risk_level
- created_at
- updated_at
```

### 4.2 账户 Account

账户是资金归属主体。一个用户可以拥有多个币种账户。

```text
Account
- account_id
- user_id
- account_no
- currency: USD / EUR / GBP / HKD / CNY / JPY / SGD
- account_type: personal / business / system / clearing
- status: active / frozen / closed
- available_balance
- frozen_balance
- ledger_balance
- created_at
- updated_at
```

### 4.3 账户别名 Account Alias

用于转账时搜索收款人。

```text
AccountAlias
- alias_id
- user_id
- account_id
- alias_type: aspira_id / phone / email / account_no
- alias_value_hash
- alias_value_masked
- verified
- created_at
```

### 4.4 历史转账账户 Transfer Contact

用户完成转账后，系统自动记录历史收款人。

```text
TransferContact
- contact_id
- owner_user_id
- target_user_id
- target_account_id
- target_display_name
- target_aspira_id
- target_account_no_masked
- target_currency
- last_transfer_at
- transfer_count
- total_amount
- status: active / hidden / blocked
- created_at
- updated_at
```

说明：

- owner_user_id 是当前用户；
- target_user_id 是历史收款人；
- target_account_no_masked 只保存脱敏账号；
- 不建议在历史记录中保存完整手机号、邮箱、真实账号；
- 再次转账时必须重新查询真实账户状态，不能直接信任历史记录。

---

## 5. 主动转账架构

### 5.1 主动转账业务流程

```text
1. 用户进入 Transfer 页面
2. 选择历史收款人，或输入 Aspira ID / 手机号 / 邮箱 / 账户号
3. 客户端请求服务端解析收款账户
4. 服务端返回脱敏后的收款人信息
5. 用户输入金额、币种、备注
6. 服务端生成转账报价 quote
7. 客户端展示手续费、汇率、预计到账金额
8. 用户确认转账
9. 客户端提交 transfer_confirm 请求
10. API Gateway 校验登录态、设备、幂等键
11. Transfer Service 校验账户、KYC、限额、风控
12. C++ Transaction Engine 执行冻结、扣款、入账
13. Ledger Engine 写入双边分录
14. Transfer Service 更新交易状态
15. Activity Service 生成交易动态
16. Notification Service 通知收款方
17. Transfer Contact Service 更新历史收款人
```

### 5.2 主动转账状态机

```text
created
  │
  ▼
quoted
  │
  ▼
confirmed
  │
  ▼
risk_checking
  │
  ├── rejected
  │
  ▼
processing
  │
  ├── failed
  │
  ├── reversed
  │
  ▼
succeeded
```

### 5.3 主动转账核心 API

#### 5.3.1 解析收款账户

```http
POST /api/v4/transfer/resolve-recipient
Authorization: Bearer <token>
Idempotency-Key: <uuid>
Content-Type: application/json
```

请求：

```json
{
  "recipient_type": "aspira_id",
  "recipient_value": "aspira_10086",
  "currency": "USD"
}
```

响应：

```json
{
  "recipient_user_id": "usr_9x8x",
  "recipient_account_id": "acct_usd_001",
  "display_name": "M*** Y**",
  "aspira_id": "aspira_10086",
  "account_no_masked": "****8899",
  "currency": "USD",
  "status": "active"
}
```

#### 5.3.2 创建转账报价

```http
POST /api/v4/transfer/quote
Authorization: Bearer <token>
Idempotency-Key: <uuid>
Content-Type: application/json
```

请求：

```json
{
  "source_account_id": "acct_usd_sender",
  "target_account_id": "acct_usd_001",
  "source_currency": "USD",
  "target_currency": "USD",
  "amount": "100.00",
  "remark": "Dinner"
}
```

响应：

```json
{
  "quote_id": "q_20260609_001",
  "amount": "100.00",
  "source_currency": "USD",
  "target_currency": "USD",
  "fx_rate": "1.000000",
  "fee": "0.20",
  "total_debit_amount": "100.20",
  "target_receive_amount": "100.00",
  "quote_expire_at": "2026-06-09T23:10:00Z"
}
```

#### 5.3.3 确认转账

```http
POST /api/v4/transfer/confirm
Authorization: Bearer <token>
Idempotency-Key: <uuid>
Content-Type: application/json
```

请求：

```json
{
  "quote_id": "q_20260609_001",
  "pay_password_token": "pwd_token_xxx",
  "device_id": "dev_abc",
  "client_confirmed_at": "2026-06-09T23:05:10Z"
}
```

响应：

```json
{
  "transfer_id": "trf_20260609_0001",
  "status": "succeeded",
  "source_account_id": "acct_usd_sender",
  "target_account_id": "acct_usd_001",
  "amount": "100.00",
  "fee": "0.20",
  "currency": "USD",
  "created_at": "2026-06-09T23:05:10Z",
  "completed_at": "2026-06-09T23:05:11Z"
}
```

---

## 6. 支付链接架构

### 6.1 支付链接的定位

支付链接是“收款请求”的一种轻量形式。收款方创建链接后，可以通过聊天、邮件、短信、网站按钮等方式发送给付款方。付款方打开链接后，在有效期内完成支付。

支付链接可以用于：

- 朋友之间收款；
- 服务费收款；
- 订单尾款；
- 跨境小额付款；
- 商户简单收款；
- 稍后付款场景。

V4 第一阶段支付链接只支持 Aspira Pay 内部账户付款，即付款方需要登录 Aspira Pay 后完成支付。

### 6.2 支付链接核心字段

```text
PaymentLink
- payment_link_id
- link_token_hash
- link_token_prefix
- creator_user_id
- receiver_account_id
- amount
- currency
- title
- description
- expire_at
- max_pay_count
- paid_count
- status: pending / paid / expired / cancelled / failed
- allow_partial_payment: false
- require_login: true
- risk_level
- created_at
- updated_at
```

安全原则：

- 数据库不保存明文 token，只保存 token hash；
- URL 中的 token 必须高强度随机生成；
- token 只用于定位链接，支付仍然必须登录、风控、确认；
- 支付链接不能直接携带可篡改的金额；
- 金额、币种、收款账户必须以服务端数据库为准。

### 6.3 支付链接 URL 格式

```text
https://pay.aspira.com/l/pay_<random_token>
```

或者 App Deeplink：

```text
aspirapay://payment-link/pay_<random_token>
```

QR 码预留形式：

```text
QR 内容 = https://pay.aspira.com/l/pay_<random_token>
```

这样 QR 收款可以直接复用支付链接体系，不需要另建一套交易模型。

### 6.4 支付链接创建流程

```text
1. 收款方进入 Request Money / Create Payment Link 页面
2. 输入收款金额、币种、标题、备注、有效期
3. 客户端提交 create payment link 请求
4. 服务端校验收款账户状态、KYC、限额、风控
5. Payment Link Service 生成随机 token
6. token 明文只返回给客户端一次
7. 数据库保存 token hash、金额、币种、收款账户、有效期
8. 返回可分享链接和 QR 码内容
9. 用户复制链接或分享给付款方
```

### 6.5 支付链接支付流程

```text
1. 付款方打开支付链接
2. Web / App 通过 token 查询链接信息
3. 服务端校验 token 是否存在、未过期、未取消、未支付
4. 付款方登录 Aspira Pay
5. 服务端返回收款方脱敏信息、金额、币种、备注
6. 付款方选择付款账户
7. 服务端生成 payment link quote
8. 付款方确认支付
9. Transfer Service 调用交易引擎完成账户间转账
10. Payment Link Service 将链接状态更新为 paid
11. Activity Service 记录付款方和收款方交易动态
12. Notification Service 通知双方
13. Transfer Contact Service 记录历史转账账户
```

### 6.6 支付链接状态机

```text
pending
  │
  ├── cancelled
  │
  ├── expired
  │
  ▼
viewed
  │
  ▼
quoted
  │
  ▼
processing
  │
  ├── failed
  │
  ▼
paid
```

说明：

- pending：链接已创建，等待支付；
- viewed：有人打开过链接；
- quoted：付款方已生成付款报价；
- processing：正在扣款和入账；
- paid：支付完成；
- expired：超过有效期；
- cancelled：收款方主动取消；
- failed：支付失败。

### 6.7 支付链接核心 API

#### 6.7.1 创建支付链接

```http
POST /api/v4/payment-links
Authorization: Bearer <token>
Idempotency-Key: <uuid>
Content-Type: application/json
```

请求：

```json
{
  "receiver_account_id": "acct_usd_receiver",
  "amount": "100.00",
  "currency": "USD",
  "title": "Consulting Fee",
  "description": "Payment for project consultation",
  "expire_minutes": 1440
}
```

响应：

```json
{
  "payment_link_id": "plink_20260609_001",
  "payment_url": "https://pay.aspira.com/l/pay_xxxxx",
  "deeplink": "aspirapay://payment-link/pay_xxxxx",
  "qr_payload": "https://pay.aspira.com/l/pay_xxxxx",
  "amount": "100.00",
  "currency": "USD",
  "status": "pending",
  "expire_at": "2026-06-10T23:05:00Z"
}
```

#### 6.7.2 查询支付链接

```http
GET /api/v4/payment-links/public/{token}
```

响应：

```json
{
  "payment_link_id": "plink_20260609_001",
  "status": "pending",
  "receiver_display_name": "Aspira Studio",
  "receiver_aspira_id": "aspira_studio",
  "amount": "100.00",
  "currency": "USD",
  "title": "Consulting Fee",
  "description": "Payment for project consultation",
  "expire_at": "2026-06-10T23:05:00Z",
  "require_login": true
}
```

#### 6.7.3 通过支付链接生成付款报价

```http
POST /api/v4/payment-links/{payment_link_id}/quote
Authorization: Bearer <token>
Idempotency-Key: <uuid>
Content-Type: application/json
```

请求：

```json
{
  "source_account_id": "acct_usd_payer"
}
```

响应：

```json
{
  "quote_id": "q_plink_20260609_001",
  "payment_link_id": "plink_20260609_001",
  "amount": "100.00",
  "currency": "USD",
  "fee": "0.20",
  "total_debit_amount": "100.20",
  "target_receive_amount": "100.00",
  "quote_expire_at": "2026-06-09T23:15:00Z"
}
```

#### 6.7.4 确认支付链接付款

```http
POST /api/v4/payment-links/{payment_link_id}/pay
Authorization: Bearer <token>
Idempotency-Key: <uuid>
Content-Type: application/json
```

请求：

```json
{
  "quote_id": "q_plink_20260609_001",
  "pay_password_token": "pwd_token_xxx",
  "device_id": "dev_abc"
}
```

响应：

```json
{
  "payment_link_id": "plink_20260609_001",
  "transfer_id": "trf_20260609_0002",
  "status": "paid",
  "amount": "100.00",
  "currency": "USD",
  "paid_at": "2026-06-09T23:08:00Z"
}
```

#### 6.7.5 取消支付链接

```http
POST /api/v4/payment-links/{payment_link_id}/cancel
Authorization: Bearer <token>
Content-Type: application/json
```

请求：

```json
{
  "reason": "No longer needed"
}
```

响应：

```json
{
  "payment_link_id": "plink_20260609_001",
  "status": "cancelled",
  "cancelled_at": "2026-06-09T23:20:00Z"
}
```

---

## 7. QR 码收款预留设计

虽然 V4 当前只需要主动转账和有效链接稍后转账，但 QR 码收款可以直接建立在支付链接之上。

### 7.1 QR 收款设计方式

```text
收款方创建 Payment Link
        │
        ▼
服务端返回 qr_payload
        │
        ▼
客户端使用 qr_payload 生成 QR 码
        │
        ▼
付款方扫码
        │
        ▼
打开支付链接付款页
```

### 7.2 QR 码内容

```text
https://pay.aspira.com/l/pay_xxxxx
```

或：

```json
{
  "type": "aspira_payment_link",
  "version": "v4",
  "url": "https://pay.aspira.com/l/pay_xxxxx"
}
```

推荐第一阶段使用 URL 形式，兼容性更好。后续如果需要离线校验、商户码、动态码，可以升级为 JSON 格式。

### 7.3 Android 客户端预留页面

| 页面 | 功能 |
|---|---|
| ReceiveMoneyPage | 输入金额和备注，生成支付链接 |
| PaymentLinkResultPage | 展示链接、复制、分享、QR 码 |
| ScanPage | 扫码入口，V4.1 实现 |
| PaymentLinkPayPage | 打开链接后的付款确认页 |
| PaymentLinkDetailPage | 查看链接状态、取消链接、查看付款记录 |

---

## 8. 交易引擎设计

### 8.1 交易处理原则

账户之间转账必须满足以下原则：

1. 幂等性：同一个 Idempotency-Key 不能重复扣款。
2. 原子性：扣款和入账必须同时成功或同时失败。
3. 可回滚：处理中失败时必须支持冲正或撤销。
4. 可审计：每一个状态变化都必须记录。
5. 不信任客户端：余额、手续费、汇率、收款账户都以服务端为准。
6. 双边记账：任何资金变化必须有借贷双方分录。
7. 余额隔离：available_balance、frozen_balance、ledger_balance 必须分离。

### 8.2 账务分录示例

用户 A 向用户 B 转账 100 USD，手续费 0.20 USD。

```text
借：User A USD Account                100.20
贷：User B USD Account                100.00
贷：Aspira Fee Revenue Account          0.20
```

如果存在跨币种，例如 A 使用 USD 支付，B 收 EUR：

```text
借：User A USD Account                100.20 USD
贷：Aspira FX Clearing USD Account     100.00 USD
借：Aspira FX Clearing EUR Account      91.50 EUR
贷：User B EUR Account                  91.50 EUR
贷：Aspira Fee Revenue USD Account       0.20 USD
```

### 8.3 交易引擎模块

| 模块 | 语言 | 职责 |
|---|---|---|
| Transfer Service | Go | 接收转账请求、状态编排、调用风控和交易引擎 |
| Payment Link Service | Go | 支付链接创建、查询、取消、过期处理 |
| Ledger Engine | C++20 | 高性能账务分录、余额更新、原子提交 |
| Balance Engine | C++20 | 可用余额、冻结余额、账面余额计算 |
| Fee Engine | C++20 | 手续费计算，支持 Wise-like 透明费率模型 |
| FX Engine | C++20 / Go | 汇率报价、报价有效期、跨币种换算 |
| Risk Service | Go | 限额、频率、KYC、黑名单、异常交易检测 |
| Audit Service | Go | 审计日志、链上凭证、监管报表 |
| Notification Service | Go | App Push、邮件、短信、站内通知 |

---

## 9. 数据库设计

### 9.1 transfer_order

```sql
CREATE TABLE transfer_order (
    transfer_id           VARCHAR(64) PRIMARY KEY,
    payer_user_id         VARCHAR(64) NOT NULL,
    payer_account_id      VARCHAR(64) NOT NULL,
    receiver_user_id      VARCHAR(64) NOT NULL,
    receiver_account_id   VARCHAR(64) NOT NULL,
    source_currency       VARCHAR(8)  NOT NULL,
    target_currency       VARCHAR(8)  NOT NULL,
    source_amount         NUMERIC(24, 8) NOT NULL,
    target_amount         NUMERIC(24, 8) NOT NULL,
    fee_amount            NUMERIC(24, 8) NOT NULL DEFAULT 0,
    fx_rate               NUMERIC(24, 12),
    quote_id              VARCHAR(64),
    payment_link_id       VARCHAR(64),
    status                VARCHAR(32) NOT NULL,
    remark                TEXT,
    idempotency_key       VARCHAR(128) NOT NULL,
    created_at            TIMESTAMP NOT NULL DEFAULT now(),
    updated_at            TIMESTAMP NOT NULL DEFAULT now(),
    completed_at          TIMESTAMP
);

CREATE UNIQUE INDEX idx_transfer_idempotency
ON transfer_order(payer_user_id, idempotency_key);

CREATE INDEX idx_transfer_payer
ON transfer_order(payer_user_id, created_at DESC);

CREATE INDEX idx_transfer_receiver
ON transfer_order(receiver_user_id, created_at DESC);
```

### 9.2 transfer_contact

```sql
CREATE TABLE transfer_contact (
    contact_id                VARCHAR(64) PRIMARY KEY,
    owner_user_id             VARCHAR(64) NOT NULL,
    target_user_id            VARCHAR(64) NOT NULL,
    target_account_id         VARCHAR(64) NOT NULL,
    target_display_name       VARCHAR(128),
    target_aspira_id          VARCHAR(64),
    target_account_no_masked  VARCHAR(32),
    target_currency           VARCHAR(8),
    last_transfer_at          TIMESTAMP,
    transfer_count            INTEGER NOT NULL DEFAULT 0,
    total_amount              NUMERIC(24, 8) NOT NULL DEFAULT 0,
    status                    VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at                TIMESTAMP NOT NULL DEFAULT now(),
    updated_at                TIMESTAMP NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_transfer_contact_unique
ON transfer_contact(owner_user_id, target_user_id, target_account_id);

CREATE INDEX idx_transfer_contact_owner
ON transfer_contact(owner_user_id, last_transfer_at DESC);
```

### 9.3 payment_link

```sql
CREATE TABLE payment_link (
    payment_link_id        VARCHAR(64) PRIMARY KEY,
    link_token_hash        VARCHAR(128) NOT NULL UNIQUE,
    link_token_prefix      VARCHAR(16) NOT NULL,
    creator_user_id        VARCHAR(64) NOT NULL,
    receiver_account_id    VARCHAR(64) NOT NULL,
    amount                 NUMERIC(24, 8) NOT NULL,
    currency               VARCHAR(8) NOT NULL,
    title                  VARCHAR(128),
    description            TEXT,
    expire_at              TIMESTAMP NOT NULL,
    max_pay_count          INTEGER NOT NULL DEFAULT 1,
    paid_count             INTEGER NOT NULL DEFAULT 0,
    allow_partial_payment  BOOLEAN NOT NULL DEFAULT false,
    require_login          BOOLEAN NOT NULL DEFAULT true,
    status                 VARCHAR(32) NOT NULL DEFAULT 'pending',
    risk_level             VARCHAR(32),
    created_at             TIMESTAMP NOT NULL DEFAULT now(),
    updated_at             TIMESTAMP NOT NULL DEFAULT now(),
    paid_at                TIMESTAMP,
    cancelled_at           TIMESTAMP
);

CREATE INDEX idx_payment_link_creator
ON payment_link(creator_user_id, created_at DESC);

CREATE INDEX idx_payment_link_status_expire
ON payment_link(status, expire_at);
```

### 9.4 payment_link_event

```sql
CREATE TABLE payment_link_event (
    event_id           VARCHAR(64) PRIMARY KEY,
    payment_link_id    VARCHAR(64) NOT NULL,
    event_type         VARCHAR(64) NOT NULL,
    actor_user_id      VARCHAR(64),
    ip_hash            VARCHAR(128),
    user_agent_hash    VARCHAR(128),
    detail_json        JSONB,
    created_at         TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_payment_link_event_link
ON payment_link_event(payment_link_id, created_at DESC);
```

### 9.5 ledger_entry

```sql
CREATE TABLE ledger_entry (
    ledger_entry_id     VARCHAR(64) PRIMARY KEY,
    transaction_id      VARCHAR(64) NOT NULL,
    account_id          VARCHAR(64) NOT NULL,
    direction           VARCHAR(8) NOT NULL,
    currency            VARCHAR(8) NOT NULL,
    amount              NUMERIC(24, 8) NOT NULL,
    balance_after       NUMERIC(24, 8),
    entry_type          VARCHAR(64) NOT NULL,
    created_at          TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_ledger_transaction
ON ledger_entry(transaction_id);

CREATE INDEX idx_ledger_account_time
ON ledger_entry(account_id, created_at DESC);
```

---

## 10. Redis 与 Kafka 设计

### 10.1 Redis 用途

| Key | 用途 | TTL |
|---|---|---|
| transfer:quote:{quote_id} | 转账报价缓存 | 5 分钟 |
| payment_link:token:{token_hash} | 支付链接短缓存 | 5 分钟 |
| idem:{user_id}:{key} | 幂等请求锁 | 24 小时 |
| risk:transfer:freq:{user_id} | 转账频率限制 | 1 分钟 / 1 小时 |
| link:paying:{payment_link_id} | 防止同一链接并发支付 | 30 秒 |

### 10.2 Kafka Topic

| Topic | 说明 |
|---|---|
| transfer.created | 转账订单创建 |
| transfer.succeeded | 转账成功 |
| transfer.failed | 转账失败 |
| payment_link.created | 支付链接创建 |
| payment_link.viewed | 支付链接被打开 |
| payment_link.paid | 支付链接已支付 |
| payment_link.expired | 支付链接过期 |
| account.balance.changed | 账户余额变化 |
| audit.event.appended | 审计事件追加 |
| notification.dispatch | 通知分发 |

---

## 11. 风控与限额设计

### 11.1 转账前风控

转账确认前需要检查：

- 用户是否完成基础 KYC；
- 账户是否被冻结；
- 付款账户余额是否充足；
- 收款账户是否可接收该币种；
- 是否超过单笔限额；
- 是否超过每日累计限额；
- 是否命中黑名单或制裁名单；
- 是否存在异常设备或异常 IP；
- 是否短时间内频繁向新账户转账；
- 是否存在洗钱风险特征。

### 11.2 支付链接风控

支付链接需要额外检查：

- 链接创建频率；
- 同一收款人短时间内创建大量链接；
- 同一付款人打开多个高风险链接；
- 链接传播来源异常；
- 是否存在钓鱼风险；
- 链接标题、备注是否包含敏感词；
- 链接金额是否超过用户 KYC 等级。

### 11.3 KYC 等级与转账限额示例

| KYC 等级 | 单笔转账 | 每日累计 | 支付链接单笔 | 说明 |
|---|---:|---:|---:|---|
| KYC0 | 0 | 0 | 0 | 只能浏览，不可交易 |
| KYC1 | 500 USD | 1,000 USD | 300 USD | 基础身份验证 |
| KYC2 | 5,000 USD | 20,000 USD | 2,000 USD | 完整身份验证 |
| KYC3 | 50,000 USD | 200,000 USD | 20,000 USD | 高级认证 / 商户 |

实际限额需要根据目标市场监管要求调整。

---

## 12. 安全设计

### 12.1 支付链接安全

1. token 必须使用安全随机数生成。
2. token 明文只出现在 URL 中，数据库只保存 hash。
3. token 不应包含用户 ID、金额、账户 ID 等可推测信息。
4. 支付链接必须有有效期。
5. 支付前必须重新查询链接状态。
6. 支付确认必须使用登录态和支付密码 / 二次验证。
7. 单次支付链接必须使用分布式锁防止并发重复付款。
8. 已支付链接不可再次支付。
9. 链接取消后不可恢复支付。
10. 过期链接不可延长，建议重新生成。

### 12.2 转账安全

1. 所有转账请求必须带 Idempotency-Key。
2. 服务端必须重新计算手续费和汇率。
3. 客户端确认金额必须与 quote 一致。
4. quote 必须有短有效期。
5. 付款确认建议使用支付密码、设备绑定或 OTP。
6. 高风险交易进入人工审核或延迟到账。
7. 所有交易写入审计日志。
8. 不允许客户端直接修改交易状态。

### 12.3 数据隐私

| 数据 | 存储方式 |
|---|---|
| 手机号 | 加密或 hash + 脱敏展示 |
| 邮箱 | 加密或 hash + 脱敏展示 |
| 账户号 | 内部保存，外部只展示 masked |
| 支付链接 token | 只保存 hash |
| 设备 ID | hash 保存 |
| IP | hash 或分段保存 |
| KYC 材料 | 加密存储，最小权限访问 |

---

## 13. Android 客户端页面设计

### 13.1 底部主导航

继续沿用 V3 的五大主导航：

```text
Home / Pay / Wallet / Activity / Profile
```

V4 重点改造 Pay 页面和 Activity 页面。

### 13.2 Pay 页面

```text
Pay
├── Transfer
│   ├── Recent Contacts
│   ├── Search by Aspira ID / Phone / Email / Account No
│   ├── Enter Amount
│   ├── Transfer Quote
│   └── Transfer Result
│
├── Request Money
│   ├── Create Payment Link
│   ├── Set Amount / Currency / Expiry
│   ├── Payment Link Result
│   └── QR Code Display
│
├── Scan
│   └── Reserved for V4.1
│
└── Payment Link Detail
    ├── Status
    ├── Copy Link
    ├── Share Link
    ├── Cancel Link
    └── Payment History
```

### 13.3 主动转账页面流程

```text
TransferHomePage
  │
  ├── 选择历史收款人
  │       │
  │       ▼
  │   TransferAmountPage
  │
  └── 搜索收款人
          │
          ▼
      RecipientConfirmPage
          │
          ▼
      TransferAmountPage
          │
          ▼
      TransferQuotePage
          │
          ▼
      PaymentPasswordPage
          │
          ▼
      TransferResultPage
```

### 13.4 支付链接页面流程

```text
RequestMoneyPage
  │
  ▼
CreatePaymentLinkPage
  │
  ▼
PaymentLinkResultPage
  │
  ├── Copy Link
  ├── Share Link
  └── Show QR Code
```

付款方打开链接：

```text
Open Link
  │
  ▼
PaymentLinkLandingPage
  │
  ▼
Login / Auth Check
  │
  ▼
PaymentLinkPayPage
  │
  ▼
PaymentLinkQuotePage
  │
  ▼
PaymentPasswordPage
  │
  ▼
PaymentLinkPaidResultPage
```

---

## 14. Web 管理后台设计

### 14.1 运营后台功能

| 页面 | 功能 |
|---|---|
| Transfer Orders | 查看账户间转账订单 |
| Payment Links | 查看支付链接状态 |
| Risk Review | 高风险转账审核 |
| Account Monitor | 用户账户余额与冻结状态 |
| Ledger Viewer | 查看账务分录 |
| Audit Events | 查看审计事件 |
| User Contacts | 查看用户历史转账联系人，不展示敏感数据 |
| Refund / Reversal | 人工冲正、退款、撤销 |

### 14.2 管理员权限

| 角色 | 权限 |
|---|---|
| Support Agent | 查询交易状态、协助用户，不可改账 |
| Risk Officer | 查看风险订单、审核或拒绝高风险交易 |
| Finance Operator | 查看账务、发起对账、处理清算差异 |
| Admin | 管理配置、限额、权限 |
| Auditor | 只读审计日志和报表 |
| System Service | 服务间调用，不可登录后台 |

---

## 15. 服务拆分设计

### 15.1 Go 服务层

```text
api-gateway
user-service
account-service
transfer-service
payment-link-service
contact-service
risk-service
notification-service
activity-service
audit-service
admin-service
```

Go 适合处理：

- API 编排；
- 分布式服务通信；
- Kafka / Redis / PostgreSQL 集成；
- 业务状态机；
- 权限控制；
- 后台管理 API。

### 15.2 C++20 交易核心

```text
ledger-engine
balance-engine
fee-engine
fx-engine
settlement-engine
reversal-engine
```

C++20 适合处理：

- 高并发账务处理；
- 低延迟余额变更；
- 精确金额计算；
- 交易撮合式队列；
- 批量对账和清算；
- 高性能风控规则预计算。

### 15.3 前端

| 端 | 技术建议 |
|---|---|
| Android 客户端 | Kotlin / Jetpack Compose，或 Flutter |
| Web 管理后台 | TypeScript + React / Vue |
| 支付链接落地页 | Next.js / Nuxt，支持移动端浏览器 |
| 内部运营系统 | TypeScript + React + Ant Design / 自研极简 UI |

---

## 16. 关键异常处理

### 16.1 余额不足

```text
quote 阶段：允许生成报价，但 confirm 阶段必须重新检查余额。
confirm 阶段：余额不足则交易失败，状态为 failed_insufficient_balance。
```

### 16.2 链接过期

```text
打开链接时：返回 expired。
确认支付时：即使之前页面显示 pending，也必须重新检查 expire_at。
```

### 16.3 重复点击支付

```text
通过 Idempotency-Key + payment_link_id 分布式锁防止重复扣款。
```

### 16.4 支付中断

```text
如果扣款成功但状态更新失败，依靠 ledger_entry 和 transaction_id 恢复订单状态。
如果扣款失败但订单进入 processing，后台补偿任务将其改为 failed。
```

### 16.5 收款账户被冻结

```text
quote 阶段或 confirm 阶段检查 receiver_account.status。
如果收款账户冻结，拒绝转账。
```

### 16.6 支付链接被取消后仍有人打开

```text
返回 cancelled 页面，不允许继续支付。
```

---

## 17. 定时任务设计

| 任务 | 周期 | 说明 |
|---|---|---|
| ExpirePaymentLinksJob | 每 1 分钟 | 将过期 pending 链接改为 expired |
| TransferRecoveryJob | 每 1 分钟 | 恢复 processing 超时交易 |
| ContactStatsJob | 每 10 分钟 | 聚合历史转账账户统计 |
| RiskReviewTimeoutJob | 每 5 分钟 | 处理超时风控审核 |
| LedgerReconciliationJob | 每日 | 账务对账 |
| AuditChainAnchorJob | 每 5 分钟 / 每小时 | 审计事件上链或生成 hash anchor |

---

## 18. 审计设计

每个关键动作都写入 audit_event：

```text
transfer.recipient.resolved
transfer.quote.created
transfer.confirm.requested
transfer.risk.checked
transfer.ledger.committed
transfer.succeeded
transfer.failed
payment_link.created
payment_link.viewed
payment_link.quote.created
payment_link.pay.requested
payment_link.paid
payment_link.expired
payment_link.cancelled
contact.created
contact.updated
```

审计事件字段：

```text
AuditEvent
- event_id
- event_type
- actor_user_id
- target_id
- target_type
- request_id
- idempotency_key
- ip_hash
- device_id_hash
- before_json
- after_json
- created_at
```

---

## 19. V4 与 V3 的关系

| 模块 | V3 | V4 增强 |
|---|---|---|
| 用户账户 | 多币种账户、KYC、权限 | 增加账户别名、历史转账账户 |
| 银行卡支付 | 虚拟卡/实体卡、卡支付授权 | 可作为转账资金来源的后续扩展 |
| 跨境支付 | 汇率、手续费、清算 | 账户间跨币种转账复用 FX / Fee Engine |
| 交易引擎 | 扣款、冻结、入账、退款 | 新增 P2P Transfer Order 与 Payment Link Order |
| 审计 | 交易审计、链上凭证 | 链接创建、访问、支付、过期全部审计 |
| 客户端 | Home / Pay / Wallet / Activity / Profile | Pay 页面新增 Transfer / Request Money / Payment Link |

---

## 20. 推荐开发里程碑

### Phase 1：主动转账 MVP

- 账户搜索与解析；
- 转账报价；
- 转账确认；
- 余额扣减和入账；
- 交易流水；
- 历史转账账户记录；
- 基础风控与限额。

### Phase 2：支付链接 MVP

- 创建支付链接；
- 查询支付链接；
- 支付链接有效期；
- 登录后付款；
- 链接支付成功后状态更新；
- 链接取消；
- 链接过期任务。

### Phase 3：QR 码收款

- 将支付链接转为 QR；
- Android 展示 QR；
- Android 扫码识别；
- 扫码后打开 PaymentLinkPayPage；
- 商户静态码预留。

### Phase 4：高级风控与运营后台

- 高风险交易审核；
- 异常链接识别；
- 黑名单与灰名单；
- 客服查询交易；
- 财务对账；
- 审计报表。

---

## 21. V4 最小可用版本建议

第一版可以先完成以下闭环：

```text
用户 A 登录
  │
  ▼
搜索用户 B
  │
  ▼
输入金额
  │
  ▼
生成报价
  │
  ▼
确认支付
  │
  ▼
A 扣款 + B 入账
  │
  ▼
生成交易流水
  │
  ▼
记录 B 为历史转账账户
```

支付链接闭环：

```text
用户 B 创建收款链接
  │
  ▼
用户 A 打开链接
  │
  ▼
A 登录并确认付款
  │
  ▼
A 扣款 + B 入账
  │
  ▼
支付链接状态变为 paid
```

这个版本已经可以形成 Aspira Pay 内部账户之间的基础资金流转能力。

---

## 22. 总结

Aspira Pay V4 的核心不是简单增加一个“转账按钮”，而是把账户间交易、支付链接、历史收款人、QR 收款预留统一纳入完整支付系统架构中。

V4 的关键设计结论：

1. 主动转账和支付链接都应该复用同一套 Transfer / Ledger / Fee / FX / Risk 引擎。
2. 支付链接本质上是带有效期的收款请求，不应该直接修改账户余额。
3. QR 码收款可以直接复用 Payment Link URL，避免重复设计交易模型。
4. 历史转账账户只保存脱敏信息，下次转账时必须重新查询真实账户状态。
5. 所有资金变化必须由服务端交易引擎完成，客户端不能直接决定余额变化。
6. 所有交易必须具备幂等、原子性、可回滚、可审计能力。
7. 当前阶段优先实现主动转账和有效链接稍后支付，是最合理的 V4 MVP 路线。

