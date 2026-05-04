#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

message="${*:-}"
if [[ -z "$message" ]]; then
  echo "Usage: $0 \"your message for the Ceerat agent\"" >&2
  exit 1
fi

if [[ -z "${CEERAT_TOKEN:-}" || "${CEERAT_TOKEN:-}" == "your-ceerat-jwt" ]]; then
  echo "CEERAT_TOKEN is not set." >&2
  echo "Set it in .env after logging in or registering through the web/API." >&2
  exit 1
fi

curl -sS -X POST "$CEERAT_AGENT_BASE_URL/agent/chat" \
  -H "Authorization: Bearer $CEERAT_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"message\":$(printf '%s' "$message" | python3 -c 'import json,sys; print(json.dumps(sys.stdin.read()))')}" \
  | python3 -m json.tool
