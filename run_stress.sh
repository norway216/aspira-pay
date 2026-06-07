#!/bin/bash
# Aspira Pay V2 — Progressive Stress Test Launcher
# Starts with 100 accounts, scales to 1000 based on success rate

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BACKEND_DIR="$SCRIPT_DIR/backend-go"
cd "$BACKEND_DIR"

echo "╔══════════════════════════════════════════════════════════╗"
echo "║   Aspira Pay V2 — Progressive Stress Test               ║"
echo "║   100 accounts → 1000 accounts (adaptive scaling)       ║"
echo "╚══════════════════════════════════════════════════════════╝"
echo ""

go run ./cmd/stress-test/main.go \
    -target="${TARGET:-http://localhost:8080}" \
    -initial="${INITIAL:-100}" \
    -max="${MAX:-1000}" \
    -batch="${BATCH:-50}" \
    -scale-rate="${SCALE_RATE:-0.90}" \
    -scale-interval="${SCALE_INTERVAL:-30s}" \
    -duration="${DURATION:-0}" \
    -rate="${RATE:-1}" \
    "$@"
