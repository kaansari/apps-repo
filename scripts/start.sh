#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

ensure_postgres() {
  if [[ ! -x "$PG_CTL" || ! -x "$INITDB" || ! -x "$PSQL" ]]; then
    echo "PostgreSQL 14 tools were not found under /usr/local/opt/postgresql@14/bin." >&2
    echo "Install PostgreSQL first or set PG_CTL, INITDB, and PSQL." >&2
    exit 1
  fi

  if [[ ! -d "$CEERAT_PGDATA" ]]; then
    echo "Initializing Postgres data directory: $CEERAT_PGDATA"
    env LANG=C LC_ALL=C "$INITDB" -D "$CEERAT_PGDATA" -U "$CEERAT_DB_USER" -A trust -E UTF8 --locale=C
  fi

  if is_port_listening "$CEERAT_DB_PORT"; then
    echo "Postgres already listening on $CEERAT_DB_HOST:$CEERAT_DB_PORT"
    return
  fi

  echo "Starting Postgres on $CEERAT_DB_HOST:$CEERAT_DB_PORT"
  env LANG=C LC_ALL=C "$PG_CTL" \
    -D "$CEERAT_PGDATA" \
    -l "$POSTGRES_LOG" \
    -o "-p $CEERAT_DB_PORT" \
    start

  PGPASSWORD="$CEERAT_DB_PASSWORD" "$PSQL" \
    -h "$CEERAT_DB_HOST" \
    -p "$CEERAT_DB_PORT" \
    -U "$CEERAT_DB_USER" \
    -d "$CEERAT_DB_NAME" \
    -c "ALTER USER $CEERAT_DB_USER PASSWORD '$CEERAT_DB_PASSWORD';" >/dev/null
}

start_user_service() {
  if is_port_listening "$CEERAT_SERVICE_PORT"; then
    echo "User service already listening on localhost:$CEERAT_SERVICE_PORT"
    return
  fi

	echo "Starting user service on localhost:$CEERAT_SERVICE_PORT"
	cd "$ROOT_DIR"
	nohup env \
		PORT="$CEERAT_SERVICE_PORT" \
		DB_HOST="$CEERAT_DB_HOST" \
		DB_PORT="$CEERAT_DB_PORT" \
		DB_USER="$CEERAT_DB_USER" \
		DB_PASSWORD="$CEERAT_DB_PASSWORD" \
		DB_NAME="$CEERAT_DB_NAME" \
		JWT_SECRET="$CEERAT_JWT_SECRET" \
		CEERAT_ENV="$CEERAT_ENV" \
		"$BIN_DIR/ceerat-user-service" >>"$SERVICE_LOG" 2>&1 &
	echo $! >"$SERVICE_PID"
	sleep 1
}

start_agent_service() {
  if is_port_listening "$CEERAT_AGENT_PORT"; then
    echo "Agent service already listening on http://localhost:$CEERAT_AGENT_PORT"
    return
  fi

	echo "Starting agent service on http://localhost:$CEERAT_AGENT_PORT"
	cd "$ROOT_DIR"
	nohup env \
		PORT="$CEERAT_AGENT_PORT" \
		CEERAT_USER_SERVICE_ADDR="localhost:$CEERAT_SERVICE_PORT" \
		OPENAI_API_KEY="${OPENAI_API_KEY:-}" \
		OPENAI_MODEL="${OPENAI_MODEL:-gpt-4.1-mini}" \
		"$BIN_DIR/ceerat-agent-service" >>"$AGENT_LOG" 2>&1 &
	echo $! >"$AGENT_PID"
	sleep 1
}

start_web_ui() {
  if is_port_listening "$CEERAT_WEB_UI_PORT"; then
    echo "Web UI already listening on http://localhost:$CEERAT_WEB_UI_PORT"
    return
  fi

	echo "Starting web UI on http://localhost:$CEERAT_WEB_UI_PORT"
	cd "$ROOT_DIR"
	nohup env \
		CEERAT_WEB_UI_PORT="$CEERAT_WEB_UI_PORT" \
		CEERAT_API_BASE_URL="localhost:$CEERAT_SERVICE_PORT" \
		CEERAT_AGENT_BASE_URL="$CEERAT_AGENT_BASE_URL" \
		CEERAT_ENV="$CEERAT_ENV" \
		"$BIN_DIR/ceerat-web-ui" >>"$WEB_LOG" 2>&1 &
	echo $! >"$WEB_PID"
	sleep 1
}

ensure_dirs

echo "Building Ceerat platform"
make -C "$ROOT_DIR" build

ensure_postgres
start_user_service
start_agent_service
start_web_ui

"$ROOT_DIR/scripts/status.sh"
print_log_paths
