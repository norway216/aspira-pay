#!/bin/bash
set -e

# ============================================================
#  Aspira Pay V2 — 一键部署启动脚本
#  Cross-Border Payment Clearing & Transaction System
# ============================================================

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BACKEND_DIR="$SCRIPT_DIR/backend-go"
ENGINE_DIR="$SCRIPT_DIR/engine-cpp"
WEB_ADMIN_DIR="$SCRIPT_DIR/web-admin"
MIGRATIONS_DIR="$SCRIPT_DIR/migrations"
DEPLOY_DIR="$SCRIPT_DIR/deploy"
LOG_DIR="$SCRIPT_DIR/logs"
PID_DIR="$SCRIPT_DIR/.pids"

# PID/Log files
API_PID_FILE="$PID_DIR/api.pid"
ENGINE_PID_FILE="$PID_DIR/engine.pid"
WEB_PID_FILE="$PID_DIR/web.pid"
API_LOG="$LOG_DIR/api.log"
ENGINE_LOG="$LOG_DIR/engine.log"
WEB_LOG="$LOG_DIR/web.log"

# Ports
API_PORT="${API_PORT:-8080}"
ENGINE_PORT="${ENGINE_PORT:-9090}"
WEB_PORT="${WEB_PORT:-3000}"
DB_PORT="${DB_PORT:-5432}"
REDIS_PORT="${REDIS_PORT:-6379}"
NATS_PORT="${NATS_PORT:-4222}"

# Database config
DB_HOST="${DB_HOST:-localhost}"
DB_USER="${DB_USER:-aspirapay}"
DB_PASSWORD="${DB_PASSWORD:-aspirapay_secret}"
DB_NAME="${DB_NAME:-aspirapay}"

# ─── 颜色 ───────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# ─── 横幅 ───────────────────────────────────────────────────
banner() {
    echo -e "${CYAN}"
    echo "╔══════════════════════════════════════════════════════════╗"
    echo "║        Aspira Pay V2 — 跨境支付清算交易系统               ║"
    echo "║        Cross-Border Payment Clearing System              ║"
    echo "║        Version: 2.0.0-sandbox                            ║"
    echo "╚══════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

usage() {
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "  deploy      一键部署并启动所有服务 (默认)"
    echo "  start       启动所有服务"
    echo "  stop        停止所有服务"
    echo "  restart     重启所有服务"
    echo "  status      查看所有服务状态"
    echo "  logs        查看实时日志"
    echo "  build       仅构建，不启动"
    echo "  clean       清理构建产物和数据库"
    echo "  dev         开发模式 (全新数据库 + 详细日志)"
    echo "  db-init     仅初始化数据库"
    echo "  db-reset    重置数据库"
    echo "  check       检查所有依赖和环境"
    echo "  help        显示帮助"
    echo ""
    echo "Options:"
    echo "  --docker    使用 Docker Compose 部署 (推荐)"
    echo "  --local     使用本地构建部署"
    echo "  --skip-engine  跳过 C++ 引擎构建"
    echo "  --skip-web     跳过 Web Admin 构建"
    echo "  --port N    指定 API 端口 (默认: 8080)"
    echo ""
    echo "Examples:"
    echo "  $0                    # 本地一键部署启动"
    echo "  $0 deploy --docker    # Docker Compose 部署"
    echo "  $0 dev                # 开发模式（重建数据库）"
    echo "  $0 stop               # 停止所有服务"
    echo "  $0 logs               # 查看实时日志"
    echo "  $0 status             # 查看服务状态"
}

# ─── 日志函数 ────────────────────────────────────────────────
log_info()  { echo -e "${GREEN}[INFO]${NC}  $(date '+%H:%M:%S') $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $(date '+%H:%M:%S') $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $(date '+%H:%M:%S') $1"; }
log_step()  { echo -e "${BLUE}[STEP]${NC}  $(date '+%H:%M:%S') ${BOLD}$1${NC}"; }
log_ok()    { echo -e "${GREEN}  ✓${NC} $1"; }
log_fail()  { echo -e "${RED}  ✗${NC} $1"; }

# ─── 环境检查 ────────────────────────────────────────────────
check_deps() {
    log_step "Checking dependencies..."

    local all_ok=true

    # Go
    if command -v go &> /dev/null; then
        local go_ver=$(go version | grep -oP 'go\K[0-9]+\.[0-9]+' | head -1)
        log_ok "Go $go_ver ($(which go))"
    else
        log_fail "Go not found — install Go 1.22+"
        all_ok=false
    fi

    # CMake (for C++ engine)
    if command -v cmake &> /dev/null; then
        local cmake_ver=$(cmake --version | head -1 | grep -oP '[0-9]+\.[0-9]+')
        log_ok "CMake $cmake_ver ($(which cmake))"
    else
        log_warn "CMake not found — C++ engine will be skipped"
    fi

    # C++ compiler
    if command -v g++ &> /dev/null; then
        local gcc_ver=$(g++ --version | head -1)
        log_ok "g++ found: $gcc_ver"
    elif command -v clang++ &> /dev/null; then
        log_ok "clang++ found: $(clang++ --version | head -1)"
    else
        log_warn "C++ compiler not found — engine will be skipped"
    fi

    # PostgreSQL client
    if command -v psql &> /dev/null; then
        log_ok "psql ($(which psql))"
    else
        log_warn "psql not found — DB init will use Docker"
    fi

    # Docker
    if command -v docker &> /dev/null; then
        log_ok "Docker ($(which docker))"
    else
        log_warn "Docker not found — some features unavailable"
    fi

    # Node.js (for web admin)
    if command -v node &> /dev/null; then
        local node_ver=$(node --version)
        log_ok "Node.js $node_ver ($(which node))"
    else
        log_warn "Node.js not found — Web Admin will be skipped"
    fi

    if [ "$all_ok" = false ]; then
        log_error "Some required dependencies are missing. Please install them first."
        echo ""
        echo "  Ubuntu/Debian:"
        echo "    sudo apt install golang-go cmake g++ libssl-dev postgresql-client nodejs npm"
        echo ""
        echo "  macOS:"
        echo "    brew install go cmake postgresql node"
        echo ""
        exit 1
    fi

    log_info "All critical dependencies satisfied"
}

# ─── 目录初始化 ──────────────────────────────────────────────
init_dirs() {
    mkdir -p "$LOG_DIR" "$PID_DIR" "$BACKEND_DIR/data"
}

# ─── PostgreSQL 辅助函数 ──────────────────────────────────────
# 如果宿主机有 psql，直接用；没有则通过 docker exec 进入容器执行

CONTAINER_NAME="aspira-pay-postgres"

_pg_isready() {
    if command -v pg_isready &> /dev/null; then
        pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" &> /dev/null
        return $?
    elif docker exec "$CONTAINER_NAME" pg_isready -U "$DB_USER" -d "$DB_NAME" &> /dev/null; then
        return 0
    else
        return 1
    fi
}

_pg_exec() {
    local sql="$1"
    if command -v psql &> /dev/null; then
        export PGPASSWORD="$DB_PASSWORD"
        psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "$sql" 2>&1
    else
        docker exec -e PGPASSWORD="$DB_PASSWORD" "$CONTAINER_NAME" \
            psql -U "$DB_USER" -d "$DB_NAME" -c "$sql" 2>&1
    fi
}

_pg_exec_file() {
    local file="$1"
    if command -v psql &> /dev/null; then
        export PGPASSWORD="$DB_PASSWORD"
        psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$file" 2>&1
    else
        # Copy SQL into container, execute, remove
        docker cp "$file" "$CONTAINER_NAME":/tmp/migration.sql
        docker exec -e PGPASSWORD="$DB_PASSWORD" "$CONTAINER_NAME" \
            psql -U "$DB_USER" -d "$DB_NAME" -f /tmp/migration.sql 2>&1
        docker exec "$CONTAINER_NAME" rm -f /tmp/migration.sql 2>/dev/null
    fi
}

_wait_for_postgres() {
    local max_wait=30
    local waited=0
    while [ $waited -lt $max_wait ]; do
        if _pg_isready; then
            return 0
        fi
        printf "."
        sleep 1
        waited=$((waited + 1))
    done
    echo ""
    return 1
}

# ─── 基础设施检查 ────────────────────────────────────────────
check_infra() {
    log_step "Checking infrastructure services..."

    # PostgreSQL — 检查是否已有容器在运行
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "^${CONTAINER_NAME}$"; then
        log_info "PostgreSQL container exists, checking readiness..."
        if _wait_for_postgres; then
            log_ok "PostgreSQL ready ($DB_HOST:$DB_PORT/$DB_NAME)"
            return 0
        fi
    fi

    # 检查宿主机端口是否已有进程
    if _pg_isready; then
        log_ok "PostgreSQL ready ($DB_HOST:$DB_PORT/$DB_NAME)"
        return 0
    fi

    # 尝试启动容器
    log_warn "PostgreSQL not reachable at $DB_HOST:$DB_PORT"
    log_info "Starting PostgreSQL via Docker..."

    if ! command -v docker &> /dev/null; then
        log_error "Docker not found — cannot start PostgreSQL"
        return 1
    fi

    # 先尝试启动已有容器
    if docker ps -a --format '{{.Names}}' 2>/dev/null | grep -q "^${CONTAINER_NAME}$"; then
        log_info "Starting existing container $CONTAINER_NAME..."
        docker start "$CONTAINER_NAME" 2>/dev/null
    else
        log_info "Creating new PostgreSQL container..."
        docker run -d --name "$CONTAINER_NAME" \
            -e POSTGRES_DB="$DB_NAME" \
            -e POSTGRES_USER="$DB_USER" \
            -e POSTGRES_PASSWORD="$DB_PASSWORD" \
            -p "$DB_PORT:5432" \
            -v aspira-pay-pgdata:/var/lib/postgresql/data \
            postgres:16-alpine 2>/dev/null
    fi

    if [ $? -ne 0 ]; then
        log_error "Failed to start PostgreSQL container"
        return 1
    fi

    log_info "Waiting for PostgreSQL to be ready..."
    if _wait_for_postgres; then
        log_ok "PostgreSQL ready ($DB_HOST:$DB_PORT/$DB_NAME)"
        return 0
    else
        log_error "PostgreSQL failed to start within 30s"
        echo "Check logs: docker logs $CONTAINER_NAME"
        return 1
    fi
}

# ─── 数据库初始化 ────────────────────────────────────────────
db_init() {
    log_step "Initializing database..."

    if ! _wait_for_postgres; then
        log_error "PostgreSQL not available — cannot initialize database"
        return 1
    fi

    local failed=0
    for migration in "$MIGRATIONS_DIR"/*.sql; do
        local name=$(basename "$migration")
        local output
        output=$(_pg_exec_file "$migration" 2>&1)
        if [ $? -eq 0 ]; then
            log_ok "Migration: $name"
        else
            # 检查是否是"already exists"类错误（可忽略）
            if echo "$output" | grep -qiE "(already exists|duplicate|already a member)"; then
                log_ok "Migration: $name (already applied)"
            else
                log_warn "Migration $name: $output"
                failed=$((failed + 1))
            fi
        fi
    done

    if [ $failed -gt 0 ]; then
        log_warn "Database initialization completed with $failed warnings"
    else
        log_info "Database initialization complete"
    fi
}

db_reset() {
    log_step "Resetting database..."

    if ! _wait_for_postgres; then
        log_error "PostgreSQL not available"
        return 1
    fi

    local sql="
        SELECT pg_terminate_backend(pg_stat_activity.pid)
        FROM pg_stat_activity
        WHERE pg_stat_activity.datname = '$DB_NAME' AND pid <> pg_backend_pid();
        DROP DATABASE IF EXISTS $DB_NAME;
        CREATE DATABASE $DB_NAME OWNER \"$DB_USER\";
    "
    _pg_exec "$sql" 2>/dev/null

    log_ok "Database $DB_NAME recreated"

    db_init
}

# ─── 构建 C++ 引擎 ───────────────────────────────────────────
build_engine() {
    log_step "Building C++ Trading & Clearing Engine..."

    cd "$ENGINE_DIR"

    if [ ! -f CMakeLists.txt ]; then
        log_error "CMakeLists.txt not found in $ENGINE_DIR"
        return 1
    fi

    mkdir -p build
    cd build

    if cmake .. -DCMAKE_BUILD_TYPE=Release 2>&1 | tail -3; then
        log_ok "CMake configure"
    else
        log_fail "CMake configure failed"
        return 1
    fi

    if make -j$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4) 2>&1 | tail -5; then
        if [ -f aspira_engine ]; then
            local size=$(du -h aspira_engine | cut -f1)
            log_ok "Engine built successfully (binary: $size)"
            return 0
        fi
    fi

    log_error "Engine build failed"
    return 1
}

# ─── 构建 Go API ─────────────────────────────────────────────
build_api() {
    log_step "Building Go API Server..."

    cd "$BACKEND_DIR"

    export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
    export GOFLAGS="${GOFLAGS:--mod=mod}"

    # Download dependencies (offline-friendly)
    log_info "Resolving dependencies..."
    go mod tidy 2>&1 | tail -3 || true
    go mod download 2>&1 | tail -3 || true

    # Build
    log_info "Compiling..."
    if go build -ldflags="-s -w" -o aspira-api ./cmd/server/main.go 2>&1; then
        if [ -f aspira-api ]; then
            local size=$(du -h aspira-api | cut -f1)
            log_ok "API built successfully (binary: $size)"
            return 0
        fi
    fi

    log_error "API build failed"
    return 1
}

# ─── 构建 Web Admin ──────────────────────────────────────────
build_web() {
    log_step "Building Web Admin Dashboard..."

    cd "$WEB_ADMIN_DIR"

    if [ ! -f package.json ]; then
        log_warn "No package.json — skipping web build"
        return 0
    fi

    if ! command -v npm &> /dev/null; then
        log_warn "npm not found — skipping web build"
        return 0
    fi

    log_info "Installing dependencies..."
    npm install --silent 2>&1 | tail -3

    log_info "Building..."
    if npm run build 2>&1 | tail -5; then
        log_ok "Web Admin built successfully (dist/)"
        return 0
    fi

    log_warn "Web Admin build had issues — use 'npm run dev' for development"
    return 0
}

# ─── 构建所有 ────────────────────────────────────────────────
build_all() {
    local skip_engine=false
    local skip_web=false

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --skip-engine) skip_engine=true; shift ;;
            --skip-web) skip_web=true; shift ;;
            *) shift ;;
        esac
    done

    log_info "Starting build process..."
    echo ""

    init_dirs

    if [ "$skip_engine" = false ]; then
        build_engine || log_warn "Engine build failed — API will use local fallback"
    else
        log_info "Skipping C++ engine build (--skip-engine)"
    fi

    echo ""
    build_api || { log_error "API build failed"; exit 1; }

    echo ""
    if [ "$skip_web" = false ]; then
        build_web
    else
        log_info "Skipping Web Admin build (--skip-web)"
    fi

    echo ""
    log_info "═══════════════════════════════════════════════"
    log_info "  Build complete!"
    log_info "═══════════════════════════════════════════════"
}

# ─── 启动服务 ────────────────────────────────────────────────
start_api() {
    if [ -f "$API_PID_FILE" ] && kill -0 $(cat "$API_PID_FILE") 2>/dev/null; then
        log_warn "API server is already running (PID: $(cat $API_PID_FILE))"
        return 0
    fi

    if [ ! -f "$BACKEND_DIR/aspira-api" ]; then
        log_error "API binary not found. Run '$0 build' first."
        return 1
    fi

    log_info "Starting API server on port $API_PORT..."
    cd "$BACKEND_DIR"

    nohup ./aspira-api configs/config.yaml > "$API_LOG" 2>&1 &
    local pid=$!
    echo $pid > "$API_PID_FILE"

    # Wait for ready
    log_info "Waiting for API to be ready..."
    for i in $(seq 1 30); do
        if curl -s http://localhost:$API_PORT/health > /dev/null 2>&1; then
            log_ok "API server ready (PID: $pid, Port: $API_PORT)"
            return 0
        fi
        printf "."
        sleep 1
    done

    echo ""
    log_error "API failed to start. Check logs: $API_LOG"
    tail -20 "$API_LOG"
    return 1
}

start_engine() {
    if [ -f "$ENGINE_PID_FILE" ] && kill -0 $(cat "$ENGINE_PID_FILE") 2>/dev/null; then
        log_warn "C++ Engine is already running (PID: $(cat $ENGINE_PID_FILE))"
        return 0
    fi

    local engine_bin="$ENGINE_DIR/build/aspira_engine"
    if [ ! -f "$engine_bin" ]; then
        log_warn "Engine binary not found — API will use local fallback"
        return 0
    fi

    log_info "Starting C++ Engine on port $ENGINE_PORT..."
    cd "$ENGINE_DIR"

    nohup "$engine_bin" config/engine.json > "$ENGINE_LOG" 2>&1 &
    local pid=$!
    echo $pid > "$ENGINE_PID_FILE"

    sleep 1
    if kill -0 $pid 2>/dev/null; then
        log_ok "C++ Engine started (PID: $pid)"
    else
        log_warn "C++ Engine may have failed — check $ENGINE_LOG"
    fi
}

start_web() {
    if [ -f "$WEB_PID_FILE" ] && kill -0 $(cat "$WEB_PID_FILE") 2>/dev/null; then
        log_warn "Web Admin is already running (PID: $(cat $WEB_PID_FILE))"
        return 0
    fi

    # Always use Vite dev server in Sandbox — it has the API proxy configured.
    # The production build (dist/) is for Docker/nginx deployment only.
    if [ -f "$WEB_ADMIN_DIR/package.json" ] && command -v npm &> /dev/null; then
        log_info "Starting Web Admin (Vite dev + proxy) on port $WEB_PORT..."
        cd "$WEB_ADMIN_DIR"
        nohup npm run dev -- --port $WEB_PORT --host > "$WEB_LOG" 2>&1 &
        local pid=$!
        echo $pid > "$WEB_PID_FILE"
        sleep 2
        log_ok "Web Admin started (PID: $pid, http://localhost:$WEB_PORT)"
        return 0
    fi

    log_warn "Web Admin not available — install Node.js and run: npm install && npm run dev"
    return 0
}

start_all() {
    log_step "Starting all services..."
    echo ""

    init_dirs
    check_infra || return 1
    echo ""
    db_init
    echo ""

    start_engine
    echo ""
    start_api || return 1
    echo ""
    start_web

    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║  Aspira Pay V2 is ready!                                ║${NC}"
    echo -e "${GREEN}╠══════════════════════════════════════════════════════════╣${NC}"
    echo -e "${GREEN}║${NC}  API Server:  ${CYAN}http://localhost:$API_PORT${NC}                       ${GREEN}║${NC}"
    echo -e "${GREEN}║${NC}  Health:      ${CYAN}http://localhost:$API_PORT/health${NC}                  ${GREEN}║${NC}"
    echo -e "${GREEN}║${NC}  API Base:    ${CYAN}http://localhost:$API_PORT/api/v2${NC}                 ${GREEN}║${NC}"
    echo -e "${GREEN}║${NC}  Metrics:     ${CYAN}http://localhost:$API_PORT/metrics${NC}                 ${GREEN}║${NC}"

    if [ -f "$WEB_PID_FILE" ] && kill -0 $(cat "$WEB_PID_FILE") 2>/dev/null; then
        echo -e "${GREEN}║${NC}  Web Admin:   ${CYAN}http://localhost:$WEB_PORT${NC}                        ${GREEN}║${NC}"
    fi
    if [ -f "$ENGINE_PID_FILE" ] && kill -0 $(cat "$ENGINE_PID_FILE") 2>/dev/null; then
        echo -e "${GREEN}║${NC}  C++ Engine:  ${CYAN}localhost:$ENGINE_PORT${NC}                              ${GREEN}║${NC}"
    fi
    echo -e "${GREEN}╚══════════════════════════════════════════════════════════╝${NC}"
}

# ─── 停止服务 ────────────────────────────────────────────────
stop_service() {
    local pid_file="$1"
    local name="$2"
    local port="$3"

    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if kill -0 $pid 2>/dev/null; then
            log_info "Stopping $name (PID: $pid)..."
            kill $pid 2>/dev/null
            sleep 1
            if kill -0 $pid 2>/dev/null; then
                kill -9 $pid 2>/dev/null
                log_warn "$name force-stopped"
            else
                log_ok "$name stopped"
            fi
        else
            log_info "$name was not running (stale PID)"
        fi
        rm -f "$pid_file"
    fi

    # Also try by port
    if [ -n "$port" ]; then
        local port_pid=$(lsof -ti:$port 2>/dev/null)
        if [ -n "$port_pid" ]; then
            kill $port_pid 2>/dev/null
            sleep 1
        fi
    fi
}

stop_all() {
    log_step "Stopping all services..."
    echo ""

    stop_service "$API_PID_FILE" "API Server" "$API_PORT"
    stop_service "$ENGINE_PID_FILE" "C++ Engine" "$ENGINE_PORT"
    stop_service "$WEB_PID_FILE" "Web Admin" "$WEB_PORT"

    log_info "All services stopped"
}

# ─── 服务状态 ────────────────────────────────────────────────
check_process() {
    if [ -f "$1" ]; then
        local pid=$(cat "$1")
        if kill -0 $pid 2>/dev/null; then
            echo -e "  Status:    ${GREEN}Running${NC} (PID: $pid)"
        else
            echo -e "  Status:    ${RED}Stopped${NC} (stale PID: $pid)"
        fi
    else
        echo -e "  Status:    ${YELLOW}Not running${NC}"
    fi
}

status() {
    echo ""
    echo -e "${CYAN}══════════════════════════════════════════════════════════${NC}"
    echo -e "${CYAN}  Aspira Pay V2 — Service Status${NC}"
    echo -e "${CYAN}══════════════════════════════════════════════════════════${NC}"
    echo ""

    echo -e "${BOLD}API Server${NC} (port $API_PORT):"
    check_process "$API_PID_FILE"
    echo ""

    echo -e "${BOLD}C++ Engine${NC} (port $ENGINE_PORT):"
    check_process "$ENGINE_PID_FILE"
    echo ""

    echo -e "${BOLD}Web Admin${NC} (port $WEB_PORT):"
    check_process "$WEB_PID_FILE"
    echo ""

    # Infrastructure
    echo -e "${BOLD}PostgreSQL${NC} ($DB_HOST:$DB_PORT):"
    if _pg_isready; then
        echo -e "  Status:    ${GREEN}Reachable${NC}"
    else
        echo -e "  Status:    ${RED}Not reachable${NC}"
    fi
    echo ""

    echo -e "${BOLD}Redis${NC} ($REDIS_PORT):"
    if command -v redis-cli &> /dev/null && redis-cli -p "$REDIS_PORT" ping &> /dev/null; then
        echo -e "  Status:    ${GREEN}Reachable${NC}"
    elif docker exec "$CONTAINER_NAME" redis-cli -p "$REDIS_PORT" ping &> /dev/null 2>/dev/null; then
        echo -e "  Status:    ${GREEN}Reachable (Docker)${NC}"
    else
        echo -e "  Status:    ${YELLOW}Not reachable (optional)${NC}"
    fi
    echo ""

    echo -e "${BOLD}NATS${NC} ($NATS_PORT):"
    if curl -s http://localhost:8222/healthz &> /dev/null; then
        echo -e "  Status:    ${GREEN}Reachable${NC}"
    else
        echo -e "  Status:    ${YELLOW}Not reachable (optional)${NC}"
    fi

    echo ""
    echo -e "${CYAN}══════════════════════════════════════════════════════════${NC}"
}

# ─── 日志 ────────────────────────────────────────────────────
logs() {
    local service="${1:-api}"

    case "$service" in
        api)
            if [ -f "$API_LOG" ]; then
                tail -f "$API_LOG"
            else
                log_error "No API log at $API_LOG"
            fi
            ;;
        engine)
            if [ -f "$ENGINE_LOG" ]; then
                tail -f "$ENGINE_LOG"
            else
                log_error "No engine log at $ENGINE_LOG"
            fi
            ;;
        web)
            if [ -f "$WEB_LOG" ]; then
                tail -f "$WEB_LOG"
            else
                log_error "No web log at $WEB_LOG"
            fi
            ;;
        all)
            tail -f "$API_LOG" "$ENGINE_LOG" "$WEB_LOG" 2>/dev/null
            ;;
        *)
            log_error "Unknown service: $service. Use: api, engine, web, all"
            ;;
    esac
}

# ─── 清理 ────────────────────────────────────────────────────
clean() {
    log_step "Cleaning build artifacts..."

    rm -f "$BACKEND_DIR/aspira-api"
    rm -f "$BACKEND_DIR/go.sum"
    rm -rf "$ENGINE_DIR/build"
    rm -rf "$WEB_ADMIN_DIR/dist"
    rm -rf "$WEB_ADMIN_DIR/node_modules"
    rm -rf "$LOG_DIR"
    rm -rf "$PID_DIR"

    log_info "Clean complete"
}

# ─── Docker Compose 模式 ─────────────────────────────────────
deploy_docker() {
    log_step "Deploying with Docker Compose..."

    cd "$DEPLOY_DIR"

    if ! command -v docker &> /dev/null; then
        log_error "Docker is required for --docker mode"
        exit 1
    fi

    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        log_error "docker-compose is required"
        exit 1
    fi

    log_info "Building and starting all services..."
    docker compose up -d --build

    echo ""
    log_info "Waiting for services to be ready..."
    sleep 5

    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║  Aspira Pay V2 — Docker Deployment Ready!               ║${NC}"
    echo -e "${GREEN}╠══════════════════════════════════════════════════════════╣${NC}"
    echo -e "${GREEN}║${NC}  API Server:  ${CYAN}http://localhost:$API_PORT${NC}                       ${GREEN}║${NC}"
    echo -e "${GREEN}║${NC}  Web Admin:   ${CYAN}http://localhost:3000${NC}                           ${GREEN}║${NC}"
    echo -e "${GREEN}║${NC}  Prometheus:  ${CYAN}http://localhost:9091${NC}                          ${GREEN}║${NC}"
    echo -e "${GREEN}║${NC}  Grafana:     ${CYAN}http://localhost:3001${NC}  (admin/admin)           ${GREEN}║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════════╝${NC}"
}

# ─── 开发模式 ────────────────────────────────────────────────
dev() {
    log_step "Starting in DEVELOPMENT mode..."
    echo ""

    stop_all 2>/dev/null
    sleep 1

    check_deps
    echo ""

    check_infra || return 1
    echo ""

    db_reset
    echo ""

    build_all "$@"
    echo ""

    start_all
}

# ─── 主入口 ──────────────────────────────────────────────────
main() {
    banner

    local cmd="${1:-deploy}"
    shift 2>/dev/null || true

    # Parse global options
    local use_docker=false
    local args=()

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --docker) use_docker=true; shift ;;
            --local) use_docker=false; shift ;;
            --skip-engine) args+=("--skip-engine"); shift ;;
            --skip-web) args+=("--skip-web"); shift ;;
            --port) API_PORT="$2"; shift 2 ;;
            *) args+=("$1"); shift ;;
        esac
    done

    case "$cmd" in
        deploy)
            if [ "$use_docker" = true ]; then
                deploy_docker
            else
                check_deps
                echo ""
                check_infra || exit 1
                echo ""
                db_init
                echo ""
                build_all "${args[@]}"
                echo ""
                start_all
            fi
            ;;

        start)
            start_all
            ;;

        stop)
            stop_all
            ;;

        restart)
            stop_all
            sleep 2
            start_all
            ;;

        status)
            status
            ;;

        logs)
            logs "${args[0]:-api}"
            ;;

        build)
            check_deps
            build_all "${args[@]}"
            ;;

        clean)
            stop_all 2>/dev/null
            clean
            ;;

        dev)
            dev "${args[@]}"
            ;;

        db-init)
            check_infra
            db_init
            ;;

        db-reset)
            check_infra
            db_reset
            ;;

        check)
            check_deps
            ;;

        help|--help|-h)
            usage
            ;;

        *)
            log_error "Unknown command: $cmd"
            usage
            exit 1
            ;;
    esac
}

# ─── Trap ────────────────────────────────────────────────────
trap 'echo ""; log_info "Interrupted"; exit 1' INT TERM

main "$@"
