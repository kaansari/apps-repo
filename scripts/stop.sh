#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

stop_pidfile() {
  local name="$1"
  local pidfile="$2"
  local port="$3"
  local pid=""

  if [[ -f "$pidfile" ]]; then
    pid="$(cat "$pidfile")"
  fi

  if ! is_pid_running "$pid"; then
    pid="$(pid_for_port "$port")"
  fi

  if is_pid_running "$pid"; then
    echo "Stopping $name (pid $pid)"
    kill "$pid" || true
  else
    echo "$name is not running"
  fi

  rm -f "$pidfile"
}

ensure_dirs

stop_pidfile "web UI" "$WEB_PID" "$CEERAT_WEB_UI_PORT"
stop_pidfile "user service" "$SERVICE_PID" "$CEERAT_SERVICE_PORT"

if [[ -x "$PG_CTL" && -d "$CEERAT_PGDATA" ]]; then
  if is_port_listening "$CEERAT_DB_PORT"; then
    echo "Stopping Postgres on port $CEERAT_DB_PORT"
    env LANG=C LC_ALL=C "$PG_CTL" -D "$CEERAT_PGDATA" stop
  else
    echo "Postgres is not listening on port $CEERAT_DB_PORT"
  fi
else
  echo "Postgres data directory or pg_ctl not found"
fi
