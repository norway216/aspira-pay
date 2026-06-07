#!/bin/bash
# Aspira Pay V2 — Benchmark Client Launcher
# High-concurrency simulated trading for stress testing

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BACKEND_DIR="$SCRIPT_DIR/backend-go"

cd "$BACKEND_DIR"

# Default values
TARGET="${TARGET:-http://localhost:8080}"
WORKERS="${WORKERS:-10}"
DURATION="${DURATION:-60}"
RATE="${RATE:-0}"
AMOUNT="${AMOUNT:-100}"
MODE="${MODE:-trade}"
PAIRS="${PAIRS:-USD/JPY,USD/EUR,USD/CNY,EUR/JPY,GBP/USD}"

echo "╔══════════════════════════════════════════════════════════╗"
echo "║   Aspira Pay V2 — Benchmark Client Launcher             ║"
echo "╚══════════════════════════════════════════════════════════╝"
echo ""
echo "  Target:   $TARGET"
echo "  Workers:  $WORKERS"
echo "  Duration: ${DURATION}s"
echo "  Rate:     $RATE TPS (0=unlimited)"
echo "  Amount:   $AMOUNT avg (major unit)"
echo "  Pairs:    $PAIRS"
echo "  Mode:     $MODE"
echo ""

# Parse extra args
EXTRA_ARGS=()
while [[ $# -gt 0 ]]; do
    case "$1" in
        --target) TARGET="$2"; shift 2 ;;
        --workers) WORKERS="$2"; shift 2 ;;
        --duration) DURATION="$2"; shift 2 ;;
        --rate) RATE="$2"; shift 2 ;;
        --amount) AMOUNT="$2"; shift 2 ;;
        --mode) MODE="$2"; shift 2 ;;
        --pairs) PAIRS="$2"; shift 2 ;;
        -h|--help)
            echo "Usage: $0 [options]"
            echo ""
            echo "Environment variables (or use flags):"
            echo "  TARGET    API base URL"
            echo "  WORKERS   Number of concurrent workers"
            echo "  DURATION  Test duration in seconds"
            echo "  RATE      Target TPS (0=unlimited)"
            echo "  AMOUNT    Average tx amount in major units"
            echo "  MODE      trade | full-lifecycle | burst"
            echo "  PAIRS     Comma-separated currency pairs"
            echo ""
            echo "Examples:"
            echo "  $0                                                  # Default stress test"
            echo "  $0 --workers 50 --duration 120 --rate 1000          # 1000 TPS target"
            echo "  $0 --mode burst --workers 20 --duration 30          # Burst mode"
            echo "  $0 --mode full-lifecycle --workers 5 --duration 120  # Full lifecycle"
            echo "  TARGET=http://prod:8080 WORKERS=100 DURATION=300 $0  # Via env vars"
            exit 0
            ;;
        *) EXTRA_ARGS+=("$1"); shift ;;
    esac
done

# Build and run
echo "Building benchmark client..."
go build -o /tmp/aspira-bench-client ./cmd/bench-client/main.go 2>&1 || {
    echo "Build failed — running with 'go run' instead"
    go run ./cmd/bench-client/main.go \
        -target="$TARGET" \
        -workers="$WORKERS" \
        -duration="$DURATION" \
        -rate="$RATE" \
        -amount="$AMOUNT" \
        -mode="$MODE" \
        -pairs="$PAIRS" \
        "${EXTRA_ARGS[@]}"
    exit $?
}

echo "Starting benchmark..."
exec /tmp/aspira-bench-client \
    -target="$TARGET" \
    -workers="$WORKERS" \
    -duration="$DURATION" \
    -rate="$RATE" \
    -amount="$AMOUNT" \
    -mode="$MODE" \
    -pairs="$PAIRS" \
    "${EXTRA_ARGS[@]}"
