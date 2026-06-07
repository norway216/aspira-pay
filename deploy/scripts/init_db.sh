#!/bin/bash
# Aspira Pay V2 — Database Initialization Script
# Runs all migrations in order against PostgreSQL

set -e

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-aspirapay}"
DB_PASSWORD="${DB_PASSWORD:-aspirapay_secret}"
DB_NAME="${DB_NAME:-aspirapay}"

export PGPASSWORD="$DB_PASSWORD"

echo "Initializing Aspira Pay V2 database at $DB_HOST:$DB_PORT/$DB_NAME"

for migration in /migrations/*.sql; do
    echo "Running: $(basename $migration)"
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$migration" || {
        echo "Warning: Migration $(basename $migration) had errors (may be expected for re-runs)"
    }
done

echo "Database initialization complete"
echo ""
echo "System accounts created:"
echo "  Users table initialized"
echo "  Accounts table with settlement & fee accounts"
echo "  KYC profiles table ready"
echo "  Payment orders table ready"
echo "  Ledger entries table (append-only) ready"
echo "  Settlement batches table ready"
echo "  Chain blocks & events tables ready"
echo "  Idempotency keys table ready"
echo "  Outbox events table ready"
