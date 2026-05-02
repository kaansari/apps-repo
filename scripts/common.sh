#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_DIR="$ROOT_DIR/bin"
RUN_DIR="$ROOT_DIR/.run"
LOG_DIR="$ROOT_DIR/logs"

PG_CTL="${PG_CTL:-/usr/local/opt/postgresql@14/bin/pg_ctl}"
INITDB="${INITDB:-/usr/local/opt/postgresql@14/bin/initdb}"
PSQL="${PSQL:-/usr/local/opt/postgresql@14/bin/psql}"

CEERAT_PGDATA="${CEERAT_PGDATA:-$HOME/.local/share/ceerat-postgres-14-utf8}"
CEERAT_DB_HOST="${CEERAT_DB_HOST:-localhost}"
CEERAT_DB_PORT="${CEERAT_DB_PORT:-55434}"
CEERAT_DB_USER="${CEERAT_DB_USER:-postgres}"
CEERAT_DB_PASSWORD="${CEERAT_DB_PASSWORD:-postgres}"
CEERAT_DB_NAME="${CEERAT_DB_NAME:-postgres}"
CEERAT_SERVICE_PORT="${CEERAT_SERVICE_PORT:-50051}"
CEERAT_WEB_UI_PORT="${CEERAT_WEB_UI_PORT:-3000}"
CEERAT_JWT_SECRET="${CEERAT_JWT_SECRET:-dev-secret}"
CEERAT_ENV="${CEERAT_ENV:-development}"

POSTGRES_LOG="$LOG_DIR/postgres.log"
SERVICE_LOG="$LOG_DIR/user-service.log"
WEB_LOG="$LOG_DIR/web-ui.log"
SERVICE_PID="$RUN_DIR/user-service.pid"
WEB_PID="$RUN_DIR/web-ui.pid"

ensure_dirs() {
  mkdir -p "$RUN_DIR" "$LOG_DIR" "$(dirname "$CEERAT_PGDATA")"
}

is_pid_running() {
  local pid="${1:-}"
  [[ -n "$pid" ]] && kill -0 "$pid" >/dev/null 2>&1
}

pid_for_port() {
	local port="$1"
	lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null | head -n 1 || true
}

is_port_listening() {
  [[ -n "$(pid_for_port "$1")" ]]
}

print_log_paths() {
  printf 'Logs:\n'
  printf '  Postgres:     %s\n' "$POSTGRES_LOG"
  printf '  User service: %s\n' "$SERVICE_LOG"
  printf '  Web UI:       %s\n' "$WEB_LOG"
}
