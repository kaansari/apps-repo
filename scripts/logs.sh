#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

ensure_dirs

case "${1:-all}" in
  postgres|db)
    touch "$POSTGRES_LOG"
    tail -f "$POSTGRES_LOG"
    ;;
  service|user|api)
    touch "$SERVICE_LOG"
    tail -f "$SERVICE_LOG"
    ;;
  web|app)
    touch "$WEB_LOG"
    tail -f "$WEB_LOG"
    ;;
  agent|ai)
    touch "$AGENT_LOG"
    tail -f "$AGENT_LOG"
    ;;
  all)
    touch "$POSTGRES_LOG" "$SERVICE_LOG" "$WEB_LOG" "$AGENT_LOG"
    tail -f "$POSTGRES_LOG" "$SERVICE_LOG" "$WEB_LOG" "$AGENT_LOG"
    ;;
  paths)
    print_log_paths
    ;;
  *)
    echo "Usage: $0 [all|postgres|service|web|agent|paths]" >&2
    exit 1
    ;;
esac
