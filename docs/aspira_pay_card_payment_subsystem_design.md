# Aspira Pay 银行卡支付子系统补充设计文档

**项目名称**：Aspira Pay Card Payment Subsystem  
**作者**：Aspira Studio  
**文档版本**：V1.0 Card Payment & Multi-Currency Clearing Edition  
**适配主系统**：Aspira Pay 跨境支付及清算系统 V3.0  
**主要实现语言**：C++20 + Go + TypeScript Web Frontend  
**核心目标**：在 Aspira Pay 跨境支付与清算系统中新增银行卡支付、虚拟卡/实体卡管理、多币种账户、实时汇率兑换、手续费扣除、卡交易清算、对账和风控能力。

---

# 1. 重要说明

银行卡支付系统涉及真实支付网络、发卡机构、收单机构、清算组织和监管许可。生产环境中不能随意生成可用银行卡号，也不能自行接入 Visa、Mastercard、银联等网络。

生产环境必须满足：

```text
发卡资质 / BIN Sponsor / 合作发卡银行
收单机构 / PSP / Acquirer 接入
PCI DSS 合规
KYC / KYB / AML / CFT
卡组织规则
本地金融监管要求
数据隐私与密钥管理要求
```

本文中的“银行卡号码生成”分为两类：

```text
1. 测试卡号生成：用于沙箱、Demo、测试环境，使用测试 BIN + Luhn 校验。
2. 生产卡号生成：必须由持牌发卡银行、发卡处理商或卡组织授权 BIN/IIN 后生成，不允许私自生成真实可支付卡号。
```

---

# 2. 子系统总体定位

银行卡支付子系统是 Aspira Pay 的一个核心支付通道模块，主要支持：

```text
虚拟卡 / 实体卡管理
多币种账户绑定
银行卡支付授权
实时汇率换算
手续费计算
余额扣减 / 冻结
卡交易清算
退款 / 撤销 / 冲正
卡交易对账
卡交易风控
卡交易审计
卡交易链上证明
```

在系统中的定位：

```text
Aspira Pay 主支付系统
    ├── 银行转账通道
    ├── PSP 通道
    ├── SWIFT 通道
    ├── 本地支付网络通道
    ├── 稳定币通道
    └── 银行卡支付子系统  ← 新增
```

---

# 3. Wise-like 设计原则

参考 Wise 类跨境支付产品，银行卡支付子系统采用以下原则：

```text
多币种余额
中间市场汇率参考
费用透明展示
支付前显示总费用
没有隐藏汇率加价
优先使用同币种余额
余额不足时自动选择最优币种兑换
手续费单独列示
卡消费与跨境转账账务分离
```

Wise 官方资料中强调其价格透明、无订阅或月费，并会在支付前展示费用；Wise 卡在没有对应币种余额时会按透明换汇费进行转换，具体费用会因交易类型、币种、地区等因素变化。因此 Aspira Pay 采用动态费率配置，而不是硬编码固定费率。

---

# 4. 总体架构

```text
┌─────────────────────────────────────────────────────────────────────┐
│                       Aspira Pay Web Frontend                       │
│ Merchant Portal / Admin Console / Card Console / Risk Console       │
└──────────────────────────────────────┬──────────────────────────────┘
                                       │
                                       ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         Go API Gateway                              │
│ TLS / mTLS / JWT / OAuth2 / Signature / Rate Limit / Idempotency    │
└──────────────────────────────────────┬──────────────────────────────┘
                                       │
        ┌──────────────────────────────┼──────────────────────────────┐
        ▼                              ▼                              ▼
┌───────────────────┐        ┌───────────────────┐        ┌───────────────────┐
│ Go Card API       │        │ Go Payment API    │        │ Go Merchant API   │
│ 卡申请/卡管理      │        │ 支付/退款/状态查询  │        │ 商户/密钥/Webhook │
└─────────┬─────────┘        └─────────┬─────────┘        └─────────┬─────────┘
          │                            │                            │
          └────────────────────────────┼────────────────────────────┘
                                       ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     Go Card Orchestrator                            │
│ Card Issuing / Authorization / Settlement / Refund / Chargeback     │
└──────────────────────────────────────┬──────────────────────────────┘
                                       │
      ┌────────────────────────────────┼────────────────────────────────┐
      ▼                                ▼                                ▼
┌───────────────────┐        ┌────────────────────┐        ┌───────────────────┐
│ C++ FX Engine     │        │ C++ Fee Engine     │        │ C++ Ledger Engine │
│ 实时汇率/兑换路径  │        │ Wise-like 手续费    │        │ 复式记账/冻结扣款  │
└─────────┬─────────┘        └─────────┬──────────┘        └─────────┬─────────┘
          │                            │                             │
          └────────────────────────────┼─────────────────────────────┘
                                       ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       Card Processing Layer                         │
│ Issuer Processor / Acquirer / Card Network / 3DS / Tokenization     │
└──────────────────────────────────────┬──────────────────────────────┘
                                       │
        ┌──────────────────────────────┼──────────────────────────────┐
        ▼                              ▼                              ▼
┌───────────────────┐        ┌────────────────────┐       ┌───────────────────┐
│ PostgreSQL        │        │ Redis              │       │ MinIO / S3        │
│ 卡/账户/交易/账务   │        │ 幂等/限流/临时授权   │       │ 账单/回执/文件      │
└───────────────────┘        └────────────────────┘       └───────────────────┘
                                       │
                                       ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Aspira Consortium Chain                          │
│ CardTxProof / SettlementProof / AuditLedger / MerkleAnchor          │
└─────────────────────────────────────────────────────────────────────┘
```

---

# 5. 新增核心模块

## 5.1 Go Card API Service

负责对外提供银行卡相关 API。

功能：

```text
创建虚拟卡
申请实体卡
查询卡信息
冻结 / 解冻卡
挂失卡
注销卡
设置卡限额
查询卡交易
查询卡账单
管理卡支付开关
管理卡币种偏好
```

典型接口：

```http
POST /api/v1/cards/virtual
POST /api/v1/cards/physical
GET  /api/v1/cards/{card_id}
POST /api/v1/cards/{card_id}/freeze
POST /api/v1/cards/{card_id}/unfreeze
POST /api/v1/cards/{card_id}/limits
GET  /api/v1/cards/{card_id}/transactions
GET  /api/v1/cards/{card_id}/statement
```

---

## 5.2 Go Card Orchestrator

负责卡交易业务编排。

核心职责：

```text
卡申请流程编排
卡状态机管理
卡授权交易编排
币种余额选择
实时汇率兑换编排
手续费扣除
交易冻结
交易确认
退款 / 撤销 / 冲正
卡清算
卡对账
卡交易证明写链
```

---

## 5.3 Card Issuing Service

负责卡发行管理。

功能：

```text
虚拟卡发行
实体卡发行申请
卡号分配
有效期生成
CVV / CVC 生成或托管
PIN 设置
卡状态维护
卡生命周期管理
卡网络 Token 管理
Apple Pay / Google Pay Token 预留接口
```

生产环境中，PAN、CVV、PIN 等敏感信息应尽量由发卡处理商或 HSM 托管，Aspira Pay 只保存 token、last4、card_id、card_fingerprint。

---

## 5.4 Card Authorization Service

负责卡支付授权。

交易授权流程：

```text
收到卡支付授权请求
  ↓
校验卡状态
  ↓
校验商户类型 MCC
  ↓
校验国家 / 地区
  ↓
校验限额
  ↓
风控预检查
  ↓
确定支付币种
  ↓
检查同币种余额
  ↓
余额不足则选择最优兑换路径
  ↓
计算实时汇率
  ↓
计算手续费
  ↓
冻结资金
  ↓
返回授权成功 / 失败
```

---

## 5.5 C++ FX Engine

负责实时汇率计算。

功能：

```text
获取实时汇率
缓存汇率快照
计算中间市场参考汇率
计算兑换路径
处理多币种兑换
计算滑点
汇率有效期控制
汇率风控熔断
```

汇率来源：

```text
银行 FX Provider
做市商报价
交易所报价
第三方汇率服务
内部流动性池
```

汇率快照：

```json
{
  "base_currency": "USD",
  "quote_currency": "EUR",
  "mid_rate": "0.920000",
  "bid_rate": "0.919800",
  "ask_rate": "0.920200",
  "source": "FX_PROVIDER_A",
  "valid_until": "2026-06-08T12:05:00Z"
}
```

---

## 5.6 C++ Fee Engine

负责手续费计算。

采用 Wise-like 透明费率模型：

```text
总费用 = 固定费用 + 百分比费用 + 通道成本 + 风险附加费 - 阶梯优惠
```

费用组成：

```text
fx_conversion_fee      换汇手续费
card_authorization_fee 卡授权费用
network_fee            卡组织网络成本
cross_border_fee       跨境成本
atm_fee                ATM 取现费用
refund_fee             退款处理费用，可选
chargeback_fee         拒付处理费用
risk_fee               高风险交易附加费
```

推荐初始费率模型：

```text
同币种卡消费：0% 平台换汇费，可能有通道成本
跨币种卡消费：0.3% ~ 1.5% 动态换汇费
高风险币种：额外 0.2% ~ 1.0%
小额交易：可收固定费用，例如 0.1 ~ 0.5 USD 等值
大额交易：阶梯折扣
ATM 取现：免费额度后收固定费 + 百分比费
```

注意：实际费率必须根据通道成本、国家、币种、监管、商业策略动态配置。

---

# 6. 银行卡号码生成设计

## 6.1 PAN 结构

银行卡号通常称为 PAN，即 Primary Account Number。

基本结构：

```text
IIN / BIN + Account Identifier + Check Digit
```

示例：

```text
前 6~8 位：IIN / BIN，表示发卡机构识别号
中间位：账户标识
最后 1 位：Luhn 校验位
```

常见长度：

```text
Visa：通常 16 位
Mastercard：通常 16 位
部分卡网络：可能 13~19 位
```

## 6.2 测试环境卡号生成

测试环境可以使用测试 BIN + Luhn 算法生成不可用于真实支付的测试卡号。

流程：

```text
输入 test_bin
生成随机 account_identifier
拼接 test_bin + account_identifier
计算 Luhn check digit
生成测试 PAN
保存 card_token 和 last4
```

伪代码：

```text
function generate_test_pan(test_bin, total_length):
    body_length = total_length - 1
    body = test_bin + random_digits(body_length - len(test_bin))
    check_digit = luhn_check_digit(body)
    return body + check_digit
```

Luhn 校验位计算：

```text
从右向左每隔一位乘以 2
如果结果大于 9，则减 9
所有数字求和
校验位使总和 % 10 == 0
```

## 6.3 生产环境卡号生成

生产环境不建议 Aspira Pay 自行生成真实可支付卡号。

推荐方式：

```text
与持牌发卡银行合作
使用 BIN Sponsor
接入发卡处理商
由 HSM / 发卡处理商生成 PAN、CVV、PIN
Aspira Pay 只保存 token、last4、card_id
```

生产环境数据保存：

```text
card_id
card_token
pan_last4
card_network
card_type
issuer_id
expiry_month
expiry_year
status
user_id / merchant_id
```

不建议保存：

```text
完整 PAN
CVV / CVC
PIN 明文
磁道数据
EMV 密钥
```

---

# 7. 多币种账户管理

## 7.1 账户模型

每个用户 / 商户可以拥有多个币种余额。

```text
Customer Wallet
├── USD Balance
├── EUR Balance
├── GBP Balance
├── HKD Balance
├── JPY Balance
├── SGD Balance
└── CNY Balance
```

每个币种包含：

```text
available_balance
frozen_balance
pending_balance
settled_balance
```

## 7.2 账户类型

```text
CARD_SPENDING_ACCOUNT
CARD_FROZEN_ACCOUNT
FX_CONVERSION_ACCOUNT
PLATFORM_FEE_ACCOUNT
CARD_NETWORK_CLEARING_ACCOUNT
MERCHANT_SETTLEMENT_ACCOUNT
REFUND_ACCOUNT
CHARGEBACK_RESERVE_ACCOUNT
```

## 7.3 数据表设计

```sql
CREATE TABLE card_account (
    id BIGSERIAL PRIMARY KEY,
    account_no VARCHAR(64) UNIQUE NOT NULL,
    owner_type VARCHAR(32) NOT NULL,
    owner_id VARCHAR(64) NOT NULL,
    currency VARCHAR(16) NOT NULL,
    account_type VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE card_account_balance (
    account_no VARCHAR(64) PRIMARY KEY,
    currency VARCHAR(16) NOT NULL,
    available_balance NUMERIC(30, 8) NOT NULL DEFAULT 0,
    frozen_balance NUMERIC(30, 8) NOT NULL DEFAULT 0,
    pending_balance NUMERIC(30, 8) NOT NULL DEFAULT 0,
    version BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);
```

---

# 8. 卡支付币种选择与内部兑换

## 8.1 支付币种判断

卡交易请求包含：

```text
transaction_currency
transaction_amount
merchant_country
merchant_category_code
card_id
user_id
```

系统需要判断：

```text
用户是否有 transaction_currency 余额
余额是否足够
是否允许该币种消费
是否需要自动兑换
最优兑换币种是什么
手续费是多少
```

## 8.2 余额使用优先级

推荐规则：

```text
1. 优先使用交易币种余额
2. 如果交易币种余额不足，使用用户设置的默认扣款币种兑换
3. 如果默认币种余额不足，按最优兑换路径选择其他币种
4. 如果多币种合计仍不足，拒绝交易
5. 如果交易触发高风险规则，进入人工审核或拒绝
```

## 8.3 自动兑换流程

```text
卡交易请求：100 EUR
  ↓
检查 EUR 余额
  ↓
EUR 不足
  ↓
检查 USD 余额
  ↓
获取 USD/EUR 实时汇率
  ↓
计算需要扣减的 USD
  ↓
计算换汇手续费
  ↓
冻结 USD 本金 + 手续费
  ↓
授权成功
```

## 8.4 换汇计算公式

假设：

```text
目标支付金额：target_amount
目标币种：target_currency
源币种：source_currency
实时汇率：fx_rate = 1 source_currency 可兑换多少 target_currency
换汇手续费率：fx_fee_rate
固定费用：fixed_fee
```

则：

```text
source_amount_before_fee = target_amount / fx_rate
fx_fee = source_amount_before_fee * fx_fee_rate + fixed_fee
total_source_amount = source_amount_before_fee + fx_fee
```

示例：

```text
支付金额：100 EUR
扣款币种：USD
实时汇率：1 USD = 0.92 EUR
换汇手续费率：0.45%
固定费用：0.20 USD

source_amount_before_fee = 100 / 0.92 = 108.6957 USD
fx_fee = 108.6957 * 0.0045 + 0.20 = 0.6891 USD
total_source_amount = 109.3848 USD
```

---

# 9. 手续费系统设计

## 9.1 Wise-like 透明手续费模型

支付前必须展示：

```text
交易金额
支付币种
扣款币种
实时汇率
换汇手续费
固定手续费
卡网络成本
总扣款金额
预计清算时间
```

## 9.2 手续费配置表

```sql
CREATE TABLE fee_rule (
    id BIGSERIAL PRIMARY KEY,
    rule_id VARCHAR(64) UNIQUE NOT NULL,
    scenario VARCHAR(64) NOT NULL,
    source_currency VARCHAR(16),
    target_currency VARCHAR(16),
    country VARCHAR(16),
    card_network VARCHAR(32),
    percentage_fee NUMERIC(12, 8) NOT NULL,
    fixed_fee NUMERIC(30, 8) NOT NULL,
    min_fee NUMERIC(30, 8),
    max_fee NUMERIC(30, 8),
    risk_level VARCHAR(32),
    effective_from TIMESTAMP NOT NULL,
    effective_to TIMESTAMP,
    status VARCHAR(32) NOT NULL
);
```

## 9.3 手续费类型

```text
CARD_SAME_CURRENCY_SPEND
CARD_CROSS_CURRENCY_SPEND
CARD_ATM_WITHDRAWAL
CARD_REFUND
CARD_CHARGEBACK
CARD_REISSUE
CARD_PHYSICAL_DELIVERY
CARD_INACTIVE
```

## 9.4 手续费入账

示例：

```text
用户 USD 账户              -109.3848 USD
FX Conversion Account      +108.6957 USD
Platform Fee Account       +0.6891 USD
Card Network Clearing      -100 EUR 等值
```

---

# 10. 卡交易授权流程

## 10.1 授权状态

```text
AUTH_RECEIVED
AUTH_VALIDATING
AUTH_RISK_CHECKING
AUTH_FX_CALCULATING
AUTH_FUND_CHECKING
AUTH_APPROVED
AUTH_DECLINED
AUTH_REVERSED
AUTH_EXPIRED
```

## 10.2 授权流程

```text
1. 收到卡网络 / 发卡处理商授权请求
2. 根据 card_token 查询卡信息
3. 校验卡状态
4. 校验用户 / 商户状态
5. 校验限额
6. 校验 MCC 和地区限制
7. 调用 Risk Service
8. 判断是否需要 3DS
9. 判断支付币种和扣款币种
10. 调用 C++ FX Engine 计算汇率
11. 调用 C++ Fee Engine 计算手续费
12. 调用 C++ Ledger Engine 冻结资金
13. 返回 APPROVED / DECLINED
14. 写入 card_authorization 表
15. 写入 outbox_event
16. 审计事件哈希写入链上
```

## 10.3 授权拒绝原因

```text
CARD_NOT_ACTIVE
CARD_FROZEN
INSUFFICIENT_FUNDS
LIMIT_EXCEEDED
MCC_BLOCKED
COUNTRY_BLOCKED
RISK_REJECTED
KYC_REQUIRED
FX_RATE_UNAVAILABLE
FEE_RULE_UNAVAILABLE
DUPLICATE_AUTH
SYSTEM_ERROR
```

---

# 11. 清算与入账流程

## 11.1 授权与清算的区别

```text
授权 Authorization：冻结资金，允许交易发生
清算 Clearing：卡网络或通道提交最终交易金额
结算 Settlement：资金在机构间完成划转
```

## 11.2 清算流程

```text
1. 卡网络 / 发卡处理商发送清算文件
2. Card Settlement Service 导入清算文件
3. 匹配原授权交易
4. 比较授权金额与清算金额
5. 若金额一致，冻结转实际扣款
6. 若金额变小，释放差额
7. 若金额变大，追加扣款或进入异常
8. 生成账务分录
9. 更新 card_transaction 状态
10. 生成 settlement_proof_hash
11. 写入联盟链
```

## 11.3 清算状态

```text
CLEARING_RECEIVED
CLEARING_MATCHED
CLEARING_AMOUNT_ADJUSTED
CLEARING_POSTED
CLEARING_FAILED
CLEARING_DISPUTED
SETTLED
```

---

# 12. 退款、撤销与拒付

## 12.1 授权撤销

适用于授权后商户未完成交易。

```text
AUTH_APPROVED
  ↓
AUTH_REVERSAL_RECEIVED
  ↓
释放冻结资金
  ↓
AUTH_REVERSED
```

## 12.2 退款

```text
交易已清算
  ↓
商户发起退款
  ↓
创建 refund_order
  ↓
原路退回用户账户
  ↓
生成反向账务分录
  ↓
退款证明写链
```

## 12.3 拒付 Chargeback

```text
用户发起争议
  ↓
冻结商户待结算资金
  ↓
进入 dispute_case
  ↓
收集证据
  ↓
卡组织 / 发卡方裁决
  ↓
胜诉：释放冻结
  ↓
败诉：退款 + 罚金入账
```

---

# 13. 风控设计

## 13.1 卡交易风控规则

```text
单笔金额限制
每日金额限制
每日次数限制
MCC 黑名单
国家 / 地区黑名单
高风险商户
异常币种兑换
短时间多笔失败交易
夜间异常交易
跨国家快速切换
设备指纹异常
3DS 失败
CVV 错误次数过多
```

## 13.2 风控动作

```text
ALLOW
STEP_UP_AUTH
REQUIRE_3DS
MANUAL_REVIEW
TEMP_FREEZE_CARD
DECLINE
REPORT_SUSPICIOUS
```

## 13.3 Redis 实时风控计数

```text
card:{card_id}:daily_amount:{date}
card:{card_id}:daily_count:{date}
user:{user_id}:failed_auth_count:{hour}
merchant:{merchant_id}:risk_score
country:{country}:risk_level
```

---

# 14. 安全与合规

## 14.1 PCI DSS

如果系统存储、处理或传输银行卡账户数据，就必须按 PCI DSS 要求设计安全边界。PCI DSS v4.0.1 是当前重要版本之一，PCI SSC 表示该标准用于保护支付账户数据，并提供技术和运营安全要求。

## 14.2 敏感数据处理

不保存或尽量不直接接触：

```text
完整 PAN
CVV / CVC
PIN
磁道数据
EMV 密钥
一次性动态验证码
```

推荐保存：

```text
card_id
card_token
pan_last4
card_fingerprint
expiry_month
expiry_year
card_status
issuer_reference
```

## 14.3 Tokenization

所有业务系统尽量使用：

```text
card_token
network_token
payment_token
```

而不是完整卡号。

## 14.4 HSM / KMS

密钥管理：

```text
PAN 加密密钥
CVV 生成密钥
PIN Block 密钥
Webhook 签名密钥
API 签名密钥
数据加密密钥
链上节点私钥
```

---

# 15. 数据库表设计

## 15.1 card 表

```sql
CREATE TABLE card (
    id BIGSERIAL PRIMARY KEY,
    card_id VARCHAR(64) UNIQUE NOT NULL,
    owner_type VARCHAR(32) NOT NULL,
    owner_id VARCHAR(64) NOT NULL,
    card_token VARCHAR(128) UNIQUE NOT NULL,
    pan_last4 VARCHAR(4) NOT NULL,
    card_network VARCHAR(32) NOT NULL,
    card_type VARCHAR(32) NOT NULL,
    card_form VARCHAR(32) NOT NULL,
    expiry_month INT NOT NULL,
    expiry_year INT NOT NULL,
    status VARCHAR(32) NOT NULL,
    default_currency VARCHAR(16),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);
```

## 15.2 card_authorization 表

```sql
CREATE TABLE card_authorization (
    id BIGSERIAL PRIMARY KEY,
    auth_id VARCHAR(64) UNIQUE NOT NULL,
    card_id VARCHAR(64) NOT NULL,
    merchant_name VARCHAR(256),
    merchant_country VARCHAR(16),
    merchant_category_code VARCHAR(16),
    transaction_amount NUMERIC(30, 8) NOT NULL,
    transaction_currency VARCHAR(16) NOT NULL,
    debit_amount NUMERIC(30, 8) NOT NULL,
    debit_currency VARCHAR(16) NOT NULL,
    fx_rate NUMERIC(30, 12),
    fee_amount NUMERIC(30, 8),
    status VARCHAR(64) NOT NULL,
    decline_reason VARCHAR(128),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);
```

## 15.3 card_transaction 表

```sql
CREATE TABLE card_transaction (
    id BIGSERIAL PRIMARY KEY,
    tx_id VARCHAR(64) UNIQUE NOT NULL,
    auth_id VARCHAR(64),
    card_id VARCHAR(64) NOT NULL,
    transaction_amount NUMERIC(30, 8) NOT NULL,
    transaction_currency VARCHAR(16) NOT NULL,
    debit_amount NUMERIC(30, 8) NOT NULL,
    debit_currency VARCHAR(16) NOT NULL,
    fx_rate NUMERIC(30, 12),
    fee_amount NUMERIC(30, 8),
    status VARCHAR(64) NOT NULL,
    settlement_date DATE,
    receipt_hash VARCHAR(128),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);
```

## 15.4 fx_quote 表

```sql
CREATE TABLE fx_quote (
    id BIGSERIAL PRIMARY KEY,
    quote_id VARCHAR(64) UNIQUE NOT NULL,
    source_currency VARCHAR(16) NOT NULL,
    target_currency VARCHAR(16) NOT NULL,
    source_amount NUMERIC(30, 8),
    target_amount NUMERIC(30, 8),
    mid_rate NUMERIC(30, 12) NOT NULL,
    applied_rate NUMERIC(30, 12) NOT NULL,
    fee_rate NUMERIC(12, 8) NOT NULL,
    fee_amount NUMERIC(30, 8) NOT NULL,
    valid_until TIMESTAMP NOT NULL,
    status VARCHAR(32) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);
```

---

# 16. API 设计

## 16.1 创建虚拟卡

```http
POST /api/v1/cards/virtual
```

```json
{
  "owner_type": "CUSTOMER",
  "owner_id": "CUS_10001",
  "card_network": "VISA",
  "default_currency": "USD",
  "spending_limits": {
    "daily_amount": "5000.00",
    "monthly_amount": "50000.00"
  }
}
```

响应：

```json
{
  "card_id": "CARD_20260608_000001",
  "card_token": "tok_card_xxx",
  "last4": "1234",
  "card_network": "VISA",
  "status": "ACTIVE"
}
```

## 16.2 查询支付前费用

```http
POST /api/v1/cards/{card_id}/quote-spend
```

```json
{
  "transaction_amount": "100.00",
  "transaction_currency": "EUR",
  "merchant_country": "DE",
  "merchant_category_code": "5812"
}
```

响应：

```json
{
  "transaction_amount": "100.00",
  "transaction_currency": "EUR",
  "debit_amount": "109.3848",
  "debit_currency": "USD",
  "fx_rate": "0.920000",
  "fx_fee": "0.6891",
  "fixed_fee": "0.20",
  "total_fee": "0.6891",
  "valid_until": "2026-06-08T12:05:00Z"
}
```

## 16.3 卡授权

```http
POST /internal/v1/card-authorizations
```

```json
{
  "card_token": "tok_card_xxx",
  "network_auth_id": "NET_AUTH_001",
  "transaction_amount": "100.00",
  "transaction_currency": "EUR",
  "merchant_name": "Example Store",
  "merchant_country": "DE",
  "merchant_category_code": "5812"
}
```

响应：

```json
{
  "auth_id": "AUTH_20260608_000001",
  "decision": "APPROVED",
  "debit_amount": "109.3848",
  "debit_currency": "USD",
  "fx_rate": "0.920000",
  "fee_amount": "0.6891"
}
```

---

# 17. Web 前端补充

## 17.1 Card Console 页面

```text
卡片总览
虚拟卡列表
实体卡申请
卡详情
卡交易明细
卡限额设置
卡冻结 / 解冻
卡币种偏好
卡费用预估
卡对账单
卡风控事件
拒付 / 争议管理
```

## 17.2 Merchant Portal 补充

```text
卡收款配置
卡支付交易查询
卡组织费用分析
退款管理
拒付管理
卡交易对账文件下载
```

## 17.3 Admin Console 补充

```text
发卡管理
BIN / IIN 管理
卡网络配置
发卡处理商配置
收单处理商配置
卡交易监控
卡清算批次
卡拒付中心
卡费用规则配置
PCI 合规状态
```

---

# 18. 与原 Aspira Pay 系统集成

## 18.1 新增服务目录

```text
aspira-pay/
├── backend/
│   ├── go-card-api/
│   ├── go-card-orchestrator/
│   ├── go-card-authorization/
│   ├── go-card-settlement/
│   ├── go-card-tokenization/
│   └── go-card-webhook/
├── cpp-core/
│   ├── fx-engine/
│   ├── fee-engine/
│   ├── ledger-engine/
│   └── card-risk-precheck/
├── frontend/
│   ├── card-console/
│   └── card-admin/
└── contracts/
    ├── CardTxProof/
    ├── CardSettlementProof/
    └── CardAuditLedger/
```

## 18.2 新增 Kafka Topic

```text
card.created
card.activated
card.frozen
card.unfrozen
card.auth.received
card.auth.approved
card.auth.declined
card.clearing.received
card.clearing.matched
card.settlement.completed
card.refund.requested
card.refund.completed
card.chargeback.opened
card.chargeback.closed
card.audit.created
card.proof.onchain
```

---

# 19. 开发路线

## 第一阶段：沙箱虚拟卡系统

```text
测试 BIN
Luhn 测试卡号生成
虚拟卡创建
卡状态管理
多币种账户
测试授权
基础手续费
基础汇率
基础冻结扣款
```

## 第二阶段：卡支付授权系统

```text
Card Authorization Service
C++ FX Engine
C++ Fee Engine
C++ Ledger Engine
Redis 实时限额
风控预检查
授权撤销
```

## 第三阶段：清算与对账

```text
清算文件导入
授权和清算匹配
差额处理
退款
拒付
对账中心
卡交易账单
```

## 第四阶段：生产发卡集成

```text
接入发卡银行 / BIN Sponsor
接入发卡处理商
Tokenization
HSM / KMS
PCI DSS 环境隔离
3DS
卡组织规则适配
```

## 第五阶段：联盟链证明

```text
CardTxProof
CardSettlementProof
CardAuditLedger
Merkle Root 上链
链上卡交易证明查询
```

---

# 20. 最终总结

加入银行卡支付子系统后，Aspira Pay 的支付能力扩展为：

```text
银行转账
PSP 支付
SWIFT 跨境支付
本地支付网络
稳定币 / 链上证明
银行卡支付
虚拟卡 / 实体卡
多币种消费
实时汇率兑换
透明手续费
卡交易清算
卡交易对账
退款 / 拒付 / 争议
```

完整系统变为：

```text
C++ 高性能交易、汇率、手续费、账务计算核心
+ Go 支付、卡、清算、风控、合规微服务
+ TypeScript Web 前端
+ PostgreSQL 核心账务
+ Kafka / NATS 事件驱动
+ Redis 实时限流与幂等
+ MinIO / S3 加密文件存储
+ Aspira Consortium Chain 审计与证明
+ PCI DSS / KYC / AML / 卡组织合规能力
```

这使 Aspira Pay 更接近 Wise / Revolut 类产品的系统形态，但实现上仍需依赖持牌合作方、发卡银行、收单机构和卡组织规则。
