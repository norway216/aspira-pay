# Aspira Pay V5 支付系统架构设计文档

**版本**：V5.0  
**作者**：Aspira Studio  
**基础版本**：Aspira Pay V4 Account Transfer & Payment Link Architecture  
**定位**：在 V4 主动转账、支付链接、历史收款账户、QR 收款预留能力基础上，升级为面向个人用户、小微商户和跨境支付场景的完整支付产品系统。  
**V5 核心目标**：产品体验更完整、Web 前台可用、运营后台可控、交易引擎更稳、服务端更易扩展、风控审计更成熟。

---

## 1. V5 总体升级方向

V4 已经完成了账户间主动转账、支付链接、历史转账联系人、QR 收款预留、交易引擎、账务分录、风控限额、审计事件等基础设计。

V5 不再只是增加单个功能，而是从“可用支付系统”升级为“可运营、可增长、可审计、可扩展的金融科技支付平台”。

### 1.1 V5 相比 V4 的关键变化

| 方向 | V4 | V5 优化 |
|---|---|---|
| 产品定位 | 内部账户转账 + 支付链接 | 个人钱包 + 小微商户收款 + Web 支付前台 + 运营后台 |
| Web 页面 | 主要是管理后台 | 增加用户 Web Portal、支付链接落地页、商户 Dashboard、运营后台 |
| UI 设计 | 功能结构为主 | 建立完整视觉系统：色彩、排版、组件、响应式布局 |
| 交易引擎 | 转账、扣款、入账、手续费、汇率 | 增加交易编排器、资金预留、异步补偿、Outbox、Saga、对账状态机 |
| 服务端 | Go 服务 + C++ 交易核心 | 增加 BFF、配置中心、规则引擎、通知中心、报表中心、可观测性 |
| 风控 | 基础 KYC、限额、频率 | 增加实时评分、设备指纹、行为风控、链接风控、人工审核台 |
| 商户能力 | 预留 | 增加 Merchant、Checkout、Payment Intent、Webhook、API Key |
| 运营能力 | 基础后台 | 增加工单、退款、冲正、冻结、对账、审计报表、权限分级 |
| 开发方式 | 服务拆分 | 增加领域边界、API 契约、事件驱动、灰度发布、故障恢复 |

---

## 2. V5 产品模块总览

```text
Aspira Pay V5
├── 用户端产品
│   ├── Web Portal
│   ├── Android App
│   ├── Wallet
│   ├── Transfer
│   ├── Request Money
│   ├── Payment Link
│   ├── QR Receive
│   └── Activity
│
├── 商户端产品
│   ├── Merchant Dashboard
│   ├── Checkout Page
│   ├── Payment Intent
│   ├── Payment Link for Merchant
│   ├── API Key / Webhook
│   └── Settlement Report
│
├── 运营后台
│   ├── User Management
│   ├── Account Monitor
│   ├── Transfer Orders
│   ├── Payment Links
│   ├── Risk Review
│   ├── Ledger Viewer
│   ├── Refund / Reversal
│   ├── Audit Events
│   └── Reconciliation Center
│
├── 服务端
│   ├── API Gateway
│   ├── BFF Layer
│   ├── Auth Service
│   ├── User Service
│   ├── Account Service
│   ├── Transfer Service
│   ├── Payment Link Service
│   ├── Merchant Service
│   ├── Checkout Service
│   ├── Risk Service
│   ├── Ledger Service
│   ├── Notification Service
│   ├── Activity Service
│   ├── Audit Service
│   └── Admin Service
│
└── 高性能核心
    ├── C++ Transaction Engine
    ├── Ledger Engine
    ├── Balance Engine
    ├── Fee Engine
    ├── FX Engine
    ├── Settlement Engine
    ├── Reversal Engine
    └── Reconciliation Engine
```

---

## 3. V5 Web 页面体系设计

V5 的 Web 不应该只有后台页面，而应该分成四类：

1. 用户 Web Portal；
2. 支付链接 / Checkout 公共支付页；
3. 商户 Dashboard；
4. 内部运营后台 Admin Console。

---

## 4. 用户 Web Portal 页面布局

### 4.1 信息架构

```text
User Web Portal
├── Dashboard
├── Wallet
│   ├── Balance
│   ├── Currency Accounts
│   ├── Add Money
│   └── Withdraw
│
├── Transfer
│   ├── New Transfer
│   ├── Recent Recipients
│   ├── Transfer Quote
│   └── Transfer Result
│
├── Request Money
│   ├── Create Payment Link
│   ├── My Payment Links
│   ├── QR Receive
│   └── Link Detail
│
├── Activity
│   ├── Transactions
│   ├── Filters
│   ├── Transaction Detail
│   └── Export
│
├── Cards
│   ├── Virtual Card
│   ├── Physical Card
│   ├── Card Transactions
│   └── Card Controls
│
└── Settings
    ├── Profile
    ├── Security
    ├── Devices
    ├── KYC
    ├── Notifications
    └── API Access
```

### 4.2 用户首页 Dashboard 布局

```text
┌──────────────────────────────────────────────────────────────┐
│ Top Bar: Aspira Pay | Search | Notifications | Profile        │
├───────────────┬──────────────────────────────────────────────┤
│ Sidebar       │ Main Content                                  │
│               │                                               │
│ Dashboard     │ ┌──────────────────────────────────────────┐  │
│ Wallet        │ │ Total Balance Card                       │  │
│ Transfer      │ │ USD / EUR / HKD / CNY / JPY               │  │
│ Request       │ └──────────────────────────────────────────┘  │
│ Activity      │                                               │
│ Cards         │ ┌──────────────┐ ┌──────────────┐ ┌────────┐ │
│ Settings      │ │ Send Money   │ │ Request      │ │ Add    │ │
│               │ │              │ │ Money        │ │ Money   │ │
│               │ └──────────────┘ └──────────────┘ └────────┘ │
│               │                                               │
│               │ ┌──────────────────────────────────────────┐  │
│               │ │ Recent Activity                          │  │
│               │ │ Transfer / Payment Link / Card / FX       │  │
│               │ └──────────────────────────────────────────┘  │
└───────────────┴──────────────────────────────────────────────┘
```

### 4.3 Transfer 页面布局

```text
Transfer Page
├── Step 1: Choose Recipient
│   ├── Search by Aspira ID / Email / Phone / Account No
│   ├── Recent Recipients
│   └── Recipient Verification Card
│
├── Step 2: Enter Amount
│   ├── Source Account
│   ├── Target Currency
│   ├── Amount
│   ├── Remark
│   └── Continue
│
├── Step 3: Quote
│   ├── Exchange Rate
│   ├── Fee
│   ├── Total Debit
│   ├── Receiver Gets
│   ├── Estimated Arrival
│   └── Confirm
│
└── Step 4: Result
    ├── Success / Failed / Under Review
    ├── Transfer ID
    ├── Ledger Status
    ├── Download Receipt
    └── Transfer Again
```

### 4.4 Request Money 页面布局

```text
Request Money
├── Create Payment Link
│   ├── Amount
│   ├── Currency
│   ├── Title
│   ├── Description
│   ├── Expiry
│   ├── Max Pay Count
│   └── Create
│
├── Result Panel
│   ├── Payment URL
│   ├── QR Code
│   ├── Copy Link
│   ├── Share
│   └── Cancel Link
│
└── My Payment Links
    ├── Pending
    ├── Paid
    ├── Expired
    ├── Cancelled
    └── Detail
```

---

## 5. 支付链接公共落地页设计

支付链接落地页是 V5 的关键增长入口。它需要看起来可信、简洁、移动端友好。

### 5.1 页面结构

```text
Payment Link Landing Page
┌────────────────────────────────────┐
│ Aspira Pay Logo                    │
├────────────────────────────────────┤
│ Receiver Card                      │
│ - Receiver Name                    │
│ - Aspira ID                        │
│ - Verified Badge                   │
├────────────────────────────────────┤
│ Amount Card                        │
│ - 100.00 USD                       │
│ - Consulting Fee                   │
│ - Expires in 23:59:12              │
├────────────────────────────────────┤
│ Pay Button                         │
│ "Pay with Aspira Pay"              │
├────────────────────────────────────┤
│ Security Notice                    │
│ Encrypted payment · Login required │
└────────────────────────────────────┘
```

### 5.2 页面状态

| 状态 | 页面展示 |
|---|---|
| pending | 显示收款人、金额、标题、有效期、支付按钮 |
| viewed | 与 pending 类似，后台记录访问事件 |
| quoted | 显示报价和确认按钮 |
| paid | 显示已支付，不允许重复支付 |
| expired | 显示链接已过期，提示联系收款方 |
| cancelled | 显示链接已取消 |
| risk_blocked | 显示暂不可支付，不展示过多风控细节 |
| login_required | 引导登录或打开 App |

### 5.3 移动端优先设计

支付链接通常从聊天工具、邮件、短信、社交媒体中打开，因此 V5 必须优先考虑移动端：

- 页面宽度：`max-width: 420px`；
- 主按钮高度：`48px - 56px`；
- 金额字体：`40px - 48px`；
- 操作按钮放在底部安全区域；
- 过期倒计时清晰可见；
- 支持 App Deeplink；
- 未安装 App 时降级到 Web 登录支付。

---

## 6. 商户 Dashboard 设计

V5 可以开始引入轻量商户能力，形成类似 Stripe / Wise Business 的基础结构。

### 6.1 商户首页

```text
Merchant Dashboard
├── Today Revenue
├── Pending Settlement
├── Successful Payments
├── Failed Payments
├── Payment Links
├── Checkout Payments
├── Recent Customers
└── Risk Alerts
```

### 6.2 商户页面

| 页面 | 功能 |
|---|---|
| Overview | 收款概览、今日收入、待结算金额 |
| Payments | 所有商户收款交易 |
| Payment Links | 创建和管理支付链接 |
| Checkout | 创建 Checkout Session |
| Customers | 付款客户列表 |
| Settlements | 结算批次、手续费、到账记录 |
| Developers | API Key、Webhook、签名密钥 |
| Reports | CSV / PDF / Excel 报表 |
| Settings | 商户资料、结算账户、通知设置 |

### 6.3 商户 Checkout 页面

V5 新增 `Payment Intent` 和 `Checkout Session`，用于后续接入外部网站或 App。

```text
Merchant Website
    │
    ▼
Create Payment Intent
    │
    ▼
Create Checkout Session
    │
    ▼
Redirect to Aspira Checkout
    │
    ▼
User Pays
    │
    ▼
Webhook Notify Merchant
```

---

## 7. 运营后台 Admin Console 设计

运营后台要服务于客服、风控、财务、审计、管理员。

### 7.1 后台页面布局

```text
┌──────────────────────────────────────────────────────────────┐
│ Top Bar: Environment | Search | Alert | Operator Profile      │
├───────────────┬──────────────────────────────────────────────┤
│ Sidebar       │ Main Workspace                                │
│               │                                               │
│ Overview      │ Data Table + Filter + Detail Drawer            │
│ Users         │                                               │
│ Accounts      │                                               │
│ Transfers     │                                               │
│ Payment Links │                                               │
│ Merchants     │                                               │
│ Risk Review   │                                               │
│ Ledger        │                                               │
│ Reconcile     │                                               │
│ Audit         │                                               │
│ Settings      │                                               │
└───────────────┴──────────────────────────────────────────────┘
```

### 7.2 后台核心页面

| 页面 | 功能 | 权限 |
|---|---|---|
| Overview | 交易量、失败率、风险订单、系统状态 | Support / Admin |
| Users | 用户查询、KYC 状态、账户状态 | Support / Risk |
| Accounts | 余额、冻结、币种账户 | Finance / Admin |
| Transfers | 转账订单、状态流转、交易详情 | Support / Finance |
| Payment Links | 链接状态、创建人、访问事件 | Support / Risk |
| Merchants | 商户资料、结算配置、API 状态 | Admin / Finance |
| Risk Review | 高风险交易审核、放行、拒绝 | Risk Officer |
| Ledger Viewer | 借贷分录、余额快照、账务链路 | Finance / Auditor |
| Reconciliation | 对账任务、差异处理、重跑 | Finance |
| Audit Events | 操作日志、交易审计、导出 | Auditor |
| System Settings | 限额、费率、汇率源、规则配置 | Admin |

### 7.3 后台表格规范

后台表格应该支持：

- 高级筛选；
- 状态标签；
- 金额按币种对齐；
- 时间按用户时区展示；
- 行点击打开右侧详情抽屉；
- 敏感信息默认脱敏；
- 高危操作二次确认；
- 所有人工操作进入审计日志。

---

## 8. V5 视觉设计系统

Aspira Pay 的视觉风格建议走“年轻、干净、科技感、可信赖”，不要做成传统银行那种沉重的蓝色金融后台，也不要过度炫光。

### 8.1 品牌关键词

```text
Minimal / Trustworthy / Global / Clean / Financial Technology / Lightweight
```

中文理解：

- 极简；
- 可信；
- 国际化；
- 数字产品感；
- 年轻但不幼稚；
- 科技感但不过度赛博朋克。

### 8.2 色彩系统

#### 主色

| 角色 | 色值 | 用途 |
|---|---|---|
| Primary Navy | `#0B1220` | 顶部栏、主文字、品牌深色背景 |
| Aspira Blue | `#2563EB` | 主按钮、链接、焦点状态 |
| Soft Cyan | `#38BDF8` | 渐变辅助、成功路径提示 |
| Violet Accent | `#7C3AED` | 高级功能、卡片渐变 |
| Success Green | `#16A34A` | 成功状态 |
| Warning Amber | `#F59E0B` | 风控提醒、待处理 |
| Danger Red | `#DC2626` | 失败、拒绝、冻结 |
| Background | `#F8FAFC` | Web 页面背景 |
| Card White | `#FFFFFF` | 卡片背景 |
| Border | `#E5E7EB` | 分割线、输入框边框 |
| Text Primary | `#111827` | 主文字 |
| Text Secondary | `#6B7280` | 辅助文字 |

#### 渐变建议

```css
background: linear-gradient(135deg, #0B1220 0%, #1E3A8A 45%, #7C3AED 100%);
```

适合用于：

- 首页 Hero；
- 虚拟卡卡面；
- 支付成功页；
- Dashboard 顶部资产卡。

#### 不建议使用

- 大面积纯黑；
- 大面积高饱和紫色；
- 过多玻璃拟态；
- 复杂金融地球光束背景；
- 传统银行式深蓝金色组合。

### 8.3 字体与排版

| 场景 | 字体建议 |
|---|---|
| 英文 Web | Inter / SF Pro / system-ui |
| 中文 Web | Noto Sans SC / PingFang SC / Microsoft YaHei |
| 数字金额 | Inter / Roboto Mono / tabular-nums |
| 后台表格 | Inter + tabular-nums |

CSS 建议：

```css
body {
  font-family: Inter, -apple-system, BlinkMacSystemFont, "Segoe UI",
               "Noto Sans SC", "PingFang SC", sans-serif;
  background: #F8FAFC;
  color: #111827;
}

.amount {
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.03em;
}
```

### 8.4 字号层级

| 层级 | 大小 | 用途 |
|---|---:|---|
| Display | 48px / 56px | 首页大标题 |
| H1 | 32px / 40px | 页面标题 |
| H2 | 24px / 32px | 卡片标题 |
| H3 | 20px / 28px | 分区标题 |
| Body | 16px / 24px | 正文 |
| Small | 14px / 20px | 辅助文字 |
| Caption | 12px / 16px | 标签、说明 |

### 8.5 圆角与阴影

| 元素 | 圆角 |
|---|---:|
| 主卡片 | 20px |
| 普通卡片 | 16px |
| 按钮 | 12px |
| 输入框 | 12px |
| 标签 | 999px |

阴影建议：

```css
box-shadow: 0 12px 32px rgba(15, 23, 42, 0.08);
```

不要使用过重阴影，金融产品应保持克制。

---

## 9. Web 组件规范

### 9.1 基础组件

```text
Button
Input
Select
AmountInput
CurrencySelector
AccountSelector
RecipientCard
QuoteCard
StatusBadge
RiskBadge
TransactionTimeline
LedgerEntryTable
QRCodeCard
CopyLinkBox
ConfirmDialog
DetailDrawer
Toast
EmptyState
```

### 9.2 金额输入组件

金额输入是支付系统最关键组件之一。

要求：

- 支持币种切换；
- 支持千分位显示；
- 内部使用 decimal，不使用 float；
- 输入时校验精度；
- 显示可用余额；
- 快捷金额按钮；
- 输入金额超过余额时即时提示；
- 手续费和汇率由服务端 quote 返回，不在前端自行计算。

### 9.3 状态标签

| 状态 | 显示 |
|---|---|
| succeeded | 绿色 `Succeeded` |
| processing | 蓝色 `Processing` |
| pending | 灰色 `Pending` |
| under_review | 黄色 `Under Review` |
| failed | 红色 `Failed` |
| reversed | 紫色 `Reversed` |
| expired | 灰色 `Expired` |
| cancelled | 灰色 `Cancelled` |

---

## 10. V5 整体技术架构

```text
┌──────────────────────────────────────────────────────────────┐
│                         Clients                              │
│ Web Portal / Android App / Merchant Dashboard / Admin Console │
└───────────────────────────┬──────────────────────────────────┘
                            │ HTTPS / TLS / OAuth2 / JWT
┌───────────────────────────▼──────────────────────────────────┐
│                      Edge & Gateway                           │
│ API Gateway / WAF / Rate Limit / Signature / Idempotency       │
└───────────────────────────┬──────────────────────────────────┘
                            │
┌───────────────────────────▼──────────────────────────────────┐
│                         BFF Layer                             │
│ web-bff / mobile-bff / merchant-bff / admin-bff                │
└───────────────────────────┬──────────────────────────────────┘
                            │ gRPC / REST
┌───────────────────────────▼──────────────────────────────────┐
│                    Go Business Service Layer                   │
│ User / Account / Transfer / Payment Link / Merchant / Risk     │
│ Notification / Activity / Audit / Admin / Report / Webhook     │
└───────────────────────────┬──────────────────────────────────┘
                            │ gRPC / NATS / Kafka
┌───────────────────────────▼──────────────────────────────────┐
│              C++ High Performance Transaction Core             │
│ Transaction Orchestrator / Ledger / Balance / Fee / FX         │
│ Settlement / Reversal / Reconciliation                         │
└───────────────────────────┬──────────────────────────────────┘
                            │
┌───────────────────────────▼──────────────────────────────────┐
│                         Data Layer                            │
│ PostgreSQL / Redis / Kafka / ClickHouse / MinIO / Vault        │
└──────────────────────────────────────────────────────────────┘
```

---

## 11. BFF 层设计

V5 建议增加 BFF，即 Backend for Frontend。

### 11.1 为什么需要 BFF

不同端需要的数据不同：

- Web Portal 需要聚合余额、最近交易、通知；
- Android App 需要更轻量、更快响应；
- Merchant Dashboard 需要商户视角；
- Admin Console 需要高权限、多筛选、多审计。

如果所有前端直接调用微服务，会导致：

- 前端组合复杂；
- 权限容易混乱；
- API 变化影响大；
- 页面加载慢；
- 安全边界不清晰。

### 11.2 BFF 服务

| BFF | 面向对象 | 职责 |
|---|---|---|
| web-bff | 普通用户 Web | Dashboard、Wallet、Transfer 聚合 |
| mobile-bff | Android App | 首页、支付、扫码、推送聚合 |
| merchant-bff | 商户 | 收款、结算、Webhook、报表 |
| admin-bff | 内部运营 | 后台查询、详情抽屉、审核操作 |

---

## 12. 交易引擎 V5 优化

V4 的交易引擎已经具备 Ledger、Balance、Fee、FX、Settlement、Reversal。V5 需要进一步增强“交易一致性”和“故障恢复能力”。

### 12.1 新增 Transaction Orchestrator

交易编排器负责统一管理交易生命周期。

```text
Transaction Orchestrator
├── Validate Request
├── Load Quote
├── Risk Pre-check
├── Reserve Funds
├── Execute Ledger Entries
├── Commit Balance
├── Publish Events
├── Update Order Status
├── Trigger Notification
└── Compensation / Recovery
```

### 12.2 统一交易状态机

V5 建议将 Transfer、Payment Link Pay、Checkout Payment 都统一抽象为 Payment Transaction。

```text
created
  │
  ▼
quoted
  │
  ▼
authorized
  │
  ├── risk_rejected
  │
  ├── expired
  │
  ▼
funds_reserved
  │
  ├── reserve_failed
  │
  ▼
ledger_committing
  │
  ├── commit_failed
  │
  ▼
succeeded
  │
  ├── refunded
  │
  ├── partially_refunded
  │
  └── reversed
```

### 12.3 资金预留机制

V4 已经区分 available_balance、frozen_balance、ledger_balance。V5 应明确“资金预留”模型：

```text
available_balance 减少
frozen_balance 增加
ledger_balance 不变

等交易成功：
frozen_balance 减少
ledger_balance 减少

等交易失败：
frozen_balance 减少
available_balance 增加
```

优势：

- 避免重复支付；
- 支持异步风控；
- 支持延迟到账；
- 支持交易中断恢复。

### 12.4 双层幂等设计

#### API 层幂等

```text
Idempotency-Key + user_id + endpoint
```

防止用户重复点击、网络重试导致重复请求。

#### Ledger 层幂等

```text
transaction_id + ledger_operation_type
```

防止服务重试导致重复记账。

### 12.5 Outbox Pattern

交易成功后需要发送 Kafka 事件，但不能出现“数据库成功，事件丢失”。

V5 建议使用 Outbox 表：

```sql
CREATE TABLE event_outbox (
    event_id          VARCHAR(64) PRIMARY KEY,
    aggregate_type    VARCHAR(64) NOT NULL,
    aggregate_id      VARCHAR(64) NOT NULL,
    event_type        VARCHAR(128) NOT NULL,
    payload_json      JSONB NOT NULL,
    status            VARCHAR(32) NOT NULL DEFAULT 'pending',
    retry_count       INTEGER NOT NULL DEFAULT 0,
    created_at        TIMESTAMP NOT NULL DEFAULT now(),
    published_at      TIMESTAMP
);
```

流程：

```text
业务事务提交
    │
    ├── 更新订单状态
    ├── 写 ledger_entry
    └── 写 event_outbox
            │
            ▼
Outbox Publisher 异步发布 Kafka
```

### 12.6 Saga 补偿机制

适合跨服务交易：

```text
Create Transfer Order
    │
Reserve Funds
    │
Risk Check
    │
Commit Ledger
    │
Notify Receiver
    │
Update Activity
```

如果中间失败：

```text
Notify Failed
    │
Activity Recovery Job
    │
Order State Recovery
    │
Ledger Reconciliation
```

对于资金交易，不能随意“反向补偿”替代原子账务。资金层必须以 Ledger 为准，业务层通过补偿任务修复状态。

### 12.7 交易引擎核心接口

```protobuf
service TransactionEngine {
  rpc QuoteTransfer(QuoteTransferRequest) returns (QuoteTransferResponse);
  rpc AuthorizeTransaction(AuthorizeRequest) returns (AuthorizeResponse);
  rpc ReserveFunds(ReserveFundsRequest) returns (ReserveFundsResponse);
  rpc CommitLedger(CommitLedgerRequest) returns (CommitLedgerResponse);
  rpc ReleaseFunds(ReleaseFundsRequest) returns (ReleaseFundsResponse);
  rpc ReverseTransaction(ReverseRequest) returns (ReverseResponse);
  rpc GetTransactionState(GetTransactionStateRequest) returns (GetTransactionStateResponse);
}
```

---

## 13. Ledger V5 优化

### 13.1 账务原则

1. Ledger 是资金事实来源；
2. 订单状态可以恢复，Ledger 不可随意修改；
3. 所有资金变化必须产生借贷分录；
4. 分录一旦提交，只能通过冲正分录修复；
5. 每个交易批次必须借贷平衡；
6. 每个币种分别平衡；
7. 金额使用 decimal / int64 minor unit，不使用 float。

### 13.2 Ledger Batch

```sql
CREATE TABLE ledger_batch (
    batch_id            VARCHAR(64) PRIMARY KEY,
    transaction_id      VARCHAR(64) NOT NULL,
    transaction_type    VARCHAR(64) NOT NULL,
    status              VARCHAR(32) NOT NULL,
    debit_total         NUMERIC(24, 8) NOT NULL,
    credit_total        NUMERIC(24, 8) NOT NULL,
    currency            VARCHAR(8) NOT NULL,
    created_at          TIMESTAMP NOT NULL DEFAULT now(),
    committed_at        TIMESTAMP
);
```

### 13.3 Ledger Entry 增强

```sql
ALTER TABLE ledger_entry ADD COLUMN batch_id VARCHAR(64);
ALTER TABLE ledger_entry ADD COLUMN account_balance_before NUMERIC(24, 8);
ALTER TABLE ledger_entry ADD COLUMN account_balance_after NUMERIC(24, 8);
ALTER TABLE ledger_entry ADD COLUMN sequence_no BIGINT;
ALTER TABLE ledger_entry ADD COLUMN reversed_entry_id VARCHAR(64);
```

### 13.4 账户余额快照

```sql
CREATE TABLE account_balance_snapshot (
    snapshot_id       VARCHAR(64) PRIMARY KEY,
    account_id        VARCHAR(64) NOT NULL,
    currency          VARCHAR(8) NOT NULL,
    available_balance NUMERIC(24, 8) NOT NULL,
    frozen_balance    NUMERIC(24, 8) NOT NULL,
    ledger_balance    NUMERIC(24, 8) NOT NULL,
    last_ledger_seq   BIGINT NOT NULL,
    created_at        TIMESTAMP NOT NULL DEFAULT now()
);
```

用于：

- 快速查询余额；
- 对账；
- 故障恢复；
- 审计抽样。

---

## 14. 服务端功能优化

### 14.1 服务边界

| 服务 | V5 职责 |
|---|---|
| auth-service | 登录、OAuth2、MFA、设备绑定、Session |
| user-service | 用户资料、KYC 等级、状态 |
| account-service | 多币种账户、余额视图、账户状态 |
| transfer-service | 主动转账业务状态机 |
| payment-link-service | 支付链接、QR 收款、链接访问事件 |
| merchant-service | 商户资料、结算账户、商户状态 |
| checkout-service | Payment Intent、Checkout Session |
| risk-service | 实时风控、规则引擎、评分 |
| ledger-service | 账务查询、分录、余额快照 |
| transaction-engine | C++ 资金交易核心 |
| notification-service | Push、Email、SMS、站内消息 |
| activity-service | 用户交易动态 |
| audit-service | 审计事件、操作日志、Hash Anchor |
| admin-service | 运营后台操作 |
| report-service | 报表、导出、统计 |
| webhook-service | 商户回调 |

### 14.2 配置中心

V5 应将以下配置从代码中抽离：

```text
KYC 限额
单笔限额
每日限额
手续费规则
汇率加点
支付链接有效期上限
高风险地区规则
设备风控规则
黑名单规则
商户结算周期
Webhook 重试次数
```

建议数据结构：

```text
config_key
config_value_json
scope: global / region / user / merchant
version
enabled
effective_at
created_by
created_at
```

### 14.3 规则引擎

Risk Service 不应该把所有规则硬编码在代码里。

```text
Rule Engine
├── KYC Rule
├── Amount Rule
├── Frequency Rule
├── Device Rule
├── IP Geo Rule
├── Recipient Rule
├── Payment Link Rule
├── Merchant Rule
└── Manual Review Rule
```

规则结果：

```text
allow
deny
manual_review
step_up_auth
delay_settlement
```

### 14.4 通知中心

通知要统一模板化：

```text
transfer.succeeded.sender
transfer.succeeded.receiver
transfer.failed
payment_link.created
payment_link.paid.receiver
payment_link.expired
risk.review.required
merchant.settlement.completed
```

通知渠道：

- App Push；
- Email；
- SMS；
- WebSocket；
- 站内通知；
- 商户 Webhook。

### 14.5 活动流 Activity Feed

Activity 不应该直接从多个业务表拼接，而应该有独立聚合表。

```sql
CREATE TABLE activity_feed (
    activity_id       VARCHAR(64) PRIMARY KEY,
    user_id           VARCHAR(64) NOT NULL,
    activity_type     VARCHAR(64) NOT NULL,
    ref_type          VARCHAR(64) NOT NULL,
    ref_id            VARCHAR(64) NOT NULL,
    title             VARCHAR(128) NOT NULL,
    subtitle          VARCHAR(256),
    amount            NUMERIC(24, 8),
    currency          VARCHAR(8),
    status            VARCHAR(32),
    created_at        TIMESTAMP NOT NULL DEFAULT now()
);
```

---

## 15. Payment Link V5 优化

### 15.1 增强字段

```text
PaymentLink
- payment_link_id
- link_token_hash
- creator_user_id
- receiver_account_id
- merchant_id nullable
- amount
- currency
- title
- description
- expire_at
- max_pay_count
- paid_count
- allow_partial_payment
- require_login
- require_payer_note
- payer_email_required
- success_redirect_url
- cancel_redirect_url
- status
- risk_level
- created_at
- updated_at
```

### 15.2 支付链接类型

| 类型 | 场景 |
|---|---|
| personal_fixed | 个人固定金额收款 |
| personal_open_amount | 个人可填写金额收款 |
| merchant_fixed | 商户固定金额收款 |
| merchant_checkout | 商户 Checkout 收款 |
| donation | 打赏 / 捐赠 |
| invoice | 账单付款 |

V5 第一阶段建议实现：

- personal_fixed；
- merchant_fixed；
- merchant_checkout 预留。

### 15.3 链接访问风控

```text
同一 IP 高频打开多个链接
同一设备打开多个高额链接
链接被异常传播
链接标题命中敏感词
付款方与收款方关系异常
链接金额超出 KYC 等级
```

### 15.4 链接审计事件

```text
payment_link.created
payment_link.opened
payment_link.login_required
payment_link.quote.created
payment_link.pay.clicked
payment_link.risk.checked
payment_link.paid
payment_link.expired
payment_link.cancelled
payment_link.blocked
```

---

## 16. 商户 Payment Intent 设计

### 16.1 为什么需要 Payment Intent

Payment Link 适合分享链接；Payment Intent 适合商户网站或 App 集成。

```text
PaymentIntent
- payment_intent_id
- merchant_id
- amount
- currency
- status
- capture_method: automatic / manual
- confirmation_method: hosted / api
- customer_id
- order_id
- metadata_json
- created_at
- updated_at
```

### 16.2 Checkout Session

```text
CheckoutSession
- checkout_session_id
- payment_intent_id
- checkout_url
- expire_at
- status
- success_url
- cancel_url
- created_at
```

### 16.3 商户 Webhook

```text
payment_intent.succeeded
payment_intent.failed
payment_intent.cancelled
payment_link.paid
settlement.created
settlement.paid
refund.succeeded
```

Webhook 签名：

```text
Aspira-Signature: t=timestamp,v1=hmac_sha256(secret, timestamp + "." + payload)
```

---

## 17. 风控系统 V5

### 17.1 风控分层

```text
L1: API Gateway
- IP 限速
- WAF
- Token 校验
- 设备异常

L2: Business Risk
- KYC
- 限额
- 频率
- 收款人关系
- 支付链接异常

L3: Transaction Risk
- 金额异常
- 跨币种异常
- 黑名单
- 制裁名单
- 洗钱模式

L4: Manual Review
- 人工审核
- 放行
- 拒绝
- 延迟到账
```

### 17.2 风险评分

```text
risk_score = 
  amount_score +
  frequency_score +
  device_score +
  recipient_score +
  geo_score +
  kyc_score +
  payment_link_score
```

评分结果：

| 分数 | 处理 |
|---:|---|
| 0 - 39 | 自动通过 |
| 40 - 69 | Step-up Auth |
| 70 - 89 | 人工审核 |
| 90 - 100 | 自动拒绝 |

### 17.3 Step-up Auth

当风险中等时，不一定直接拒绝，可以要求增强验证：

- 支付密码；
- 邮箱 OTP；
- 手机 OTP；
- 设备确认；
- 生物识别；
- 延迟到账。

---

## 18. 对账与清算优化

### 18.1 对账层级

```text
订单对账
    Transfer Order / Payment Link / Checkout

账务对账
    Ledger Batch / Ledger Entry / Balance Snapshot

外部通道对账
    Card Network / Bank / FX Provider

商户结算对账
    Merchant Payment / Fee / Settlement
```

### 18.2 对账中心页面

```text
Reconciliation Center
├── Daily Summary
├── Mismatch List
├── Ledger Batch Check
├── Balance Snapshot Check
├── External Channel File
├── Merchant Settlement Check
└── Re-run / Mark Resolved
```

### 18.3 差异处理状态

```text
detected
  │
  ▼
investigating
  │
  ├── ignored
  ├── fixed
  └── escalated
```

---

## 19. 可观测性与运维

### 19.1 指标

| 指标 | 说明 |
|---|---|
| payment_success_rate | 支付成功率 |
| transfer_success_rate | 转账成功率 |
| quote_latency_p95 | 报价接口 P95 延迟 |
| confirm_latency_p95 | 确认支付 P95 延迟 |
| ledger_commit_latency_p95 | 账务提交 P95 延迟 |
| risk_review_queue_size | 风控审核队列 |
| payment_link_open_rate | 支付链接打开率 |
| payment_link_pay_rate | 支付链接转化率 |
| reconciliation_mismatch_count | 对账差异数 |
| webhook_delivery_success_rate | Webhook 成功率 |

### 19.2 日志

每个请求必须有：

```text
request_id
trace_id
user_id
device_id_hash
ip_hash
idempotency_key
transaction_id
service_name
latency_ms
status
error_code
```

### 19.3 链路追踪

推荐：

```text
OpenTelemetry + Prometheus + Grafana + Loki / ELK + Jaeger
```

---

## 20. 数据库优化

### 20.1 分库分表建议

早期不建议过早分库。V5 可以先采用：

```text
PostgreSQL 主库
├── user_db
├── account_db
├── transaction_db
├── ledger_db
├── risk_db
├── merchant_db
└── audit_db
```

交易和账务表按时间和 user_id 做分区：

```text
transfer_order_2026_06
ledger_entry_2026_06
audit_event_2026_06
```

### 20.2 金额字段

强烈建议：

- 数据库存 `NUMERIC(24, 8)`；
- C++ 内部使用定点数 decimal；
- 不允许使用 double / float 处理金额；
- API 金额使用字符串传输。

### 20.3 索引策略

```sql
CREATE INDEX idx_transfer_user_time
ON transfer_order(payer_user_id, created_at DESC);

CREATE INDEX idx_transfer_status_time
ON transfer_order(status, created_at DESC);

CREATE INDEX idx_payment_link_creator_time
ON payment_link(creator_user_id, created_at DESC);

CREATE INDEX idx_payment_link_status_expire
ON payment_link(status, expire_at);

CREATE INDEX idx_ledger_account_seq
ON ledger_entry(account_id, sequence_no DESC);

CREATE INDEX idx_activity_user_time
ON activity_feed(user_id, created_at DESC);
```

---

## 21. API 版本设计

V5 建议从 `/api/v4` 升级到 `/api/v5`，但保留 V4 兼容期。

```text
/api/v5/user
/api/v5/accounts
/api/v5/transfers
/api/v5/payment-links
/api/v5/payment-intents
/api/v5/checkout-sessions
/api/v5/merchant
/api/v5/admin
```

### 21.1 API 统一响应

```json
{
  "request_id": "req_xxx",
  "success": true,
  "data": {},
  "error": null
}
```

错误响应：

```json
{
  "request_id": "req_xxx",
  "success": false,
  "data": null,
  "error": {
    "code": "INSUFFICIENT_BALANCE",
    "message": "Insufficient available balance",
    "detail": {}
  }
}
```

### 21.2 错误码分类

```text
AUTH_*
ACCOUNT_*
TRANSFER_*
PAYMENT_LINK_*
MERCHANT_*
RISK_*
LEDGER_*
FX_*
SYSTEM_*
```

---

## 22. Web 技术栈建议

### 22.1 用户 Web / 商户 Web

```text
Next.js + TypeScript
Tailwind CSS
React Query / TanStack Query
Zustand
Zod
i18n
```

### 22.2 后台 Admin Console

```text
React + TypeScript
Ant Design Pro 或自研组件
React Query
RBAC 权限控制
ECharts / Recharts
```

### 22.3 支付链接页面

```text
Next.js SSR / Edge Rendering
Mobile First
App Deeplink
SEO noindex
Security Headers
```

支付页面建议加入：

```http
Content-Security-Policy
X-Frame-Options
Referrer-Policy
Strict-Transport-Security
```

---

## 23. V5 页面设计示例

### 23.1 Web Portal 首页视觉

```text
背景：#F8FAFC
顶部导航：白色半透明 + 轻阴影
左侧导航：白色卡片式 Sidebar
主资产卡：深蓝到紫色渐变
功能卡片：白底 + 16px 圆角 + 轻阴影
交易列表：白底表格 / 卡片
```

### 23.2 支付成功页

```text
┌────────────────────────────┐
│        Success Icon         │
│     Payment completed       │
│        100.00 USD           │
│ To: Aspira Studio           │
│ Transfer ID: trf_xxx        │
│ [Download Receipt]          │
│ [Back to Dashboard]         │
└────────────────────────────┘
```

### 23.3 运营后台交易详情抽屉

```text
Transfer Detail Drawer
├── Basic Info
│   ├── Transfer ID
│   ├── Status
│   ├── Amount
│   └── Created At
│
├── Parties
│   ├── Payer
│   └── Receiver
│
├── Quote
│   ├── FX Rate
│   ├── Fee
│   └── Total Debit
│
├── Ledger Entries
│   ├── Debit Entry
│   ├── Credit Entry
│   └── Fee Entry
│
├── Risk Result
│   ├── Score
│   ├── Rules Hit
│   └── Review Decision
│
└── Audit Timeline
```

---

## 24. 安全增强

### 24.1 Web 安全

- 所有支付页面强制 HTTPS；
- 登录态使用 HttpOnly Secure Cookie；
- CSRF Token；
- CSP；
- XSS 防护；
- 管理后台 IP 白名单；
- 管理员强制 MFA；
- 高危操作二次确认；
- 付款确认必须重新拉取服务端 quote；
- 前端只展示金额，不决定金额。

### 24.2 API 安全

- OAuth2 / JWT；
- API Gateway 统一验签；
- Idempotency-Key；
- Request ID；
- Rate Limit；
- Device Fingerprint；
- Merchant API HMAC 签名；
- Webhook 签名；
- 敏感字段脱敏；
- 审计日志不可删除。

### 24.3 数据安全

- KYC 材料加密；
- token 只保存 hash；
- 手机号、邮箱 hash + 加密；
- IP、设备 ID hash；
- 操作日志长期保存；
- 生产数据库禁止直接人工改账；
- 所有财务修正必须通过 reversal / adjustment 流程。

---

## 25. V5 MVP 开发优先级

### Phase 1：V5 Web Portal + Transfer 优化

- Web Dashboard；
- Wallet 页面；
- Transfer 页面；
- Quote Card；
- Transfer Result；
- Activity Feed；
- V5 BFF 初版。

### Phase 2：Payment Link Web 化

- 支付链接创建页；
- 公共落地页；
- 移动端支付页；
- 链接状态页；
- QR Code 展示；
- 链接访问审计。

### Phase 3：交易引擎增强

- Transaction Orchestrator；
- Funds Reservation；
- Ledger Batch；
- Outbox Pattern；
- Recovery Job；
- 统一错误码。

### Phase 4：运营后台

- Transfer Orders；
- Payment Links；
- Ledger Viewer；
- Risk Review；
- Audit Timeline；
- Reconciliation Center 初版。

### Phase 5：商户基础能力

- Merchant Account；
- Merchant Dashboard；
- Payment Intent；
- Checkout Session；
- Webhook；
- Settlement Report。

---

## 26. 推荐项目目录结构

### 26.1 后端

```text
aspira-pay-server/
├── services/
│   ├── api-gateway/
│   ├── web-bff/
│   ├── mobile-bff/
│   ├── merchant-bff/
│   ├── admin-bff/
│   ├── user-service/
│   ├── account-service/
│   ├── transfer-service/
│   ├── payment-link-service/
│   ├── merchant-service/
│   ├── checkout-service/
│   ├── risk-service/
│   ├── notification-service/
│   ├── audit-service/
│   └── report-service/
│
├── engines/
│   ├── transaction-engine/
│   ├── ledger-engine/
│   ├── balance-engine/
│   ├── fee-engine/
│   ├── fx-engine/
│   └── reconciliation-engine/
│
├── proto/
├── migrations/
├── deploy/
├── docs/
└── scripts/
```

### 26.2 前端

```text
aspira-pay-web/
├── apps/
│   ├── user-portal/
│   ├── merchant-dashboard/
│   ├── admin-console/
│   └── payment-link-page/
│
├── packages/
│   ├── ui/
│   ├── design-tokens/
│   ├── api-client/
│   ├── icons/
│   └── utils/
│
└── docs/
```

---

## 27. V5 架构核心结论

1. V5 应该从“转账系统”升级为“支付产品平台”。
2. Web 页面需要分成用户端、支付链接页、商户端、运营后台四类。
3. 视觉风格建议走极简、年轻、科技感、可信赖路线。
4. 交易引擎必须增加 Transaction Orchestrator、资金预留、Ledger Batch、Outbox 和恢复任务。
5. 服务端建议增加 BFF 层，避免前端直接组合复杂微服务。
6. Payment Link 是 V5 的重要增长入口，必须重点优化移动端体验与安全可信感。
7. 商户能力可以从 Payment Intent、Checkout Session、Webhook 开始逐步建设。
8. 风控要从简单限额升级为实时评分 + Step-up Auth + 人工审核台。
9. Ledger 必须作为资金事实来源，业务订单状态可以恢复，账务分录不可随意修改。
10. V5 的重点不是堆功能，而是建立“产品体验、资金安全、运营可控、系统可恢复”的完整闭环。

---

## 28. 建议的 V5 最小闭环

如果只做第一版 V5，建议优先完成下面这条最小闭环：

```text
用户登录 Web Portal
  │
  ▼
查看多币种余额
  │
  ▼
选择收款人
  │
  ▼
输入金额
  │
  ▼
服务端生成 Quote
  │
  ▼
用户确认支付
  │
  ▼
交易引擎资金预留
  │
  ▼
Ledger Batch 提交
  │
  ▼
订单成功
  │
  ▼
Activity Feed 展示
  │
  ▼
后台可查询交易详情、账务分录、审计事件
```

支付链接闭环：

```text
用户创建 Payment Link
  │
  ▼
生成移动端友好的公共支付页
  │
  ▼
付款方打开链接
  │
  ▼
登录 / 选择账户 / 生成 Quote
  │
  ▼
确认付款
  │
  ▼
交易引擎完成扣款入账
  │
  ▼
链接状态变为 paid
  │
  ▼
通知收款方
  │
  ▼
后台可审计链接访问和付款全过程
```

这两个闭环完成后，Aspira Pay 就从 V4 的账户间转账能力，升级到了 V5 的完整 Web 支付平台雏形。
