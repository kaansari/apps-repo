#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

status_line() {
  local name="$1"
  local port="$2"
  local url="$3"
  local pid
  pid="$(pid_for_port "$port")"

  if [[ -n "$pid" ]]; then
    printf '%-14s running  port=%-5s pid=%-8s %s\n' "$name" "$port" "$pid" "$url"
  else
    printf '%-14s stopped  port=%-5s %s\n' "$name" "$port" "$url"
  fi
}

status_line "Postgres" "$CEERAT_DB_PORT" "$CEERAT_DB_HOST:$CEERAT_DB_PORT"
status_line "User service" "$CEERAT_SERVICE_PORT" "grpc://localhost:$CEERAT_SERVICE_PORT"
status_line "Web UI" "$CEERAT_WEB_UI_PORT" "http://localhost:$CEERAT_WEB_UI_PORT"
