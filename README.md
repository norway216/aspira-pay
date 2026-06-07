# Aspira Pay V2 — Cross-Border Payment Clearing & Settlement System

> Version: V2 Sandbox / Production-Ready Architecture
> Tech Stack: Go + C++20 + PostgreSQL + NATS + Redis + Blockchain

## System Overview

Aspira Pay V2 is a transaction system designed for cross-border payments, clearing, auditing, and on-chain traceability.

Core Architecture:
- **Go** — Payment business orchestration, KYC, risk control, order state machine, settlement services
- **C++20** — High-performance transaction clearing engine (fund freezing, deduction, crediting, WAL)
- **PostgreSQL** — Business ledger, double-entry bookkeeping
- **NATS JetStream** — Event-driven message queue
- **Redis** — Caching, rate limiting
- **Blockchain (Hash Chain / Merkle Tree)** — Tamper-proof transaction audit proofs

## Project Structure

```
aspira-pay/
├── backend-go/          # Go API monolith (all business modules combined)
├── engine-cpp/          # C++ transaction clearing engine
├── web-admin/           # React admin dashboard
├── migrations/          # PostgreSQL database migrations
├── deploy/              # Docker Compose / K8s deployment configs
└── README.md
```

## Quick Start

### Prerequisites

- Go 1.22+
- CMake 3.16+ & C++20 compiler
- Docker & Docker Compose
- PostgreSQL 16+
- Redis 7+

### Minimal Deployment (Docker Compose)

```bash
cd deploy
docker-compose up -d
```

Service Ports:
- API Gateway: `http://localhost:8080`
- Web Admin: `http://localhost:3000`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3001`

### Local Development

```bash
# 1. Start infrastructure services
cd deploy && docker-compose up -d postgres redis nats

# 2. Run database migrations
psql -h localhost -U aspirapay -d aspirapay -f migrations/001_init_users.sql
# ... run all migration files in sequence

# 3. Build the C++ engine
cd engine-cpp && mkdir build && cd build && cmake .. && make -j4

# 4. Start the Go API
cd backend-go && go run cmd/server/main.go -config configs/config.yaml
```

## API Base Path

```
Base URL: http://localhost:8080/api/v2
```

## Key Technical Principles

1. All monetary amounts use int64 (smallest currency unit)
2. All transaction endpoints must be idempotent
3. The ledger is append-only — no deletions allowed
4. Every transaction must have a state machine
5. The C++ engine is responsible only for high-performance execution; Go handles business orchestration
6. Local ledger finality comes first; on-chain confirmation follows with eventual consistency

## Web Admin Screenshots

The Aspira Pay V2 admin dashboard provides a modern, dark-themed interface for managing cross-border payments, users, ledger entries, and blockchain audit trails.

### Dashboard
![Dashboard](docs/screenshots/01-dashboard.png)

The **Dashboard** gives an at-a-glance overview of the system with key metrics — total payments, registered users, settlement batches, and system status. Quick-action shortcuts provide fast navigation to core workflows.

### Transactions
![Transactions](docs/screenshots/02-transactions.png)

The **Transactions** page lists all cross-border payment orders in a sortable table. Users can create new payments via the inline form, selecting source/target currencies and amounts. Each payment is tracked through its full lifecycle with real-time status updates.

### Users
![Users](docs/screenshots/03-users.png)

The **Users** page manages registered participants. New users can be registered directly from the dashboard. The table displays user ID, username, email, account status, and risk level for compliance review.

### Ledger Explorer
![Ledger Explorer](docs/screenshots/04-ledger.png)

The **Ledger Explorer** provides double-entry bookkeeping transparency. Enter a Payment ID to retrieve full debit/credit entries, verify that totals balance, and trace every movement of funds across accounts.

### Blockchain Audit Explorer
![Audit Explorer](docs/screenshots/05-audit.png)

The **Blockchain Audit Explorer** visualises the tamper-proof hash chain. Each block links to its predecessor via cryptographic hashes. Users can audit any payment by looking up its complete on-chain trail, ensuring end-to-end traceability.

## License

Proprietary — Aspira Studio
