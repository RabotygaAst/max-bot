#!/usr/bin/env bash
set -euo pipefail

# Helper for MAX Bot API webhook setup.
# Usage examples:
#   scripts/max-webhook.sh generate-secret
#   MAX_TOKEN=... scripts/max-webhook.sh list
#   MAX_TOKEN=... WEBHOOK_SECRET=... scripts/max-webhook.sh subscribe https://example.com/webhook/max
#   scripts/max-webhook.sh test-local http://localhost:8080

if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi

MAX_BASE_URL="${MAX_BASE_URL:-https://platform-api.max.ru}"
WEBHOOK_SECRET_HEADER="${WEBHOOK_SECRET_HEADER:-X-Max-Bot-Api-Secret}"

usage() {
  cat <<USAGE
Usage: $0 <command> [args]

Commands:
  generate-secret
      Print a 64-character webhook secret allowed by MAX ([a-zA-Z0-9_-]{5,256}).

  list
      GET current MAX webhook subscriptions. Requires MAX_TOKEN.

  subscribe <https-url> [secret] [update-types-csv]
      Register webhook subscription in MAX. Requires MAX_TOKEN.
      Defaults: secret=WEBHOOK_SECRET, update-types=message_created,bot_started.

  delete <https-url>
      Delete webhook subscription from MAX. Requires MAX_TOKEN.

  test-local [base-url] [secret]
      Send a synthetic MAX update to a running bot endpoint.
      Defaults: base-url=http://localhost:8080, secret=WEBHOOK_SECRET.
USAGE
}

require_token() {
  if [[ -z "${MAX_TOKEN:-}" ]]; then
    echo "MAX_TOKEN is required. Export it or put it into .env" >&2
    exit 2
  fi
}

require_secret() {
  if [[ -z "${1:-}" ]]; then
    echo "WEBHOOK_SECRET is required for this command" >&2
    exit 2
  fi
}

json_payload() {
  python3 - "$@" <<'PY'
import json
import sys
url, secret, types_csv = sys.argv[1:4]
update_types = [item.strip() for item in types_csv.split(',') if item.strip()]
print(json.dumps({"url": url, "update_types": update_types, "secret": secret}, ensure_ascii=False))
PY
}

urlencode() {
  python3 - "$1" <<'PY'
from urllib.parse import quote
import sys
print(quote(sys.argv[1], safe=''))
PY
}

cmd="${1:-help}"
case "$cmd" in
  generate-secret)
    openssl rand -hex 32
    ;;
  list)
    require_token
    curl -fsS -H "Authorization: ${MAX_TOKEN}" "${MAX_BASE_URL%/}/subscriptions"
    echo
    ;;
  subscribe)
    require_token
    url="${2:-}"
    if [[ -z "$url" || "$url" != https://* ]]; then
      echo "subscribe requires an https:// webhook URL, for example https://example.com/webhook/max" >&2
      exit 2
    fi
    secret="${3:-${WEBHOOK_SECRET:-}}"
    require_secret "$secret"
    types="${4:-message_created,bot_started}"
    payload="$(json_payload "$url" "$secret" "$types")"
    curl -fsS -X POST "${MAX_BASE_URL%/}/subscriptions" \
      -H "Authorization: ${MAX_TOKEN}" \
      -H "Content-Type: application/json" \
      -d "$payload"
    echo
    ;;
  delete)
    require_token
    url="${2:-}"
    if [[ -z "$url" ]]; then
      echo "delete requires webhook URL" >&2
      exit 2
    fi
    encoded="$(urlencode "$url")"
    curl -fsS -X DELETE "${MAX_BASE_URL%/}/subscriptions?url=${encoded}" \
      -H "Authorization: ${MAX_TOKEN}"
    echo
    ;;
  test-local)
    base="${2:-http://localhost:8080}"
    secret="${3:-${WEBHOOK_SECRET:-}}"
    require_secret "$secret"
    curl -fsS -X POST "${base%/}/webhook/max" \
      -H "Content-Type: application/json" \
      -H "${WEBHOOK_SECRET_HEADER}: ${secret}" \
      -d '{"update_type":"message_created","timestamp":1778068800000,"message":{"sender":{"user_id":123456789,"first_name":"Иван"},"recipient":{"chat_id":987654321},"body":{"mid":"script-webhook-test-001","text":"/start"}}}'
    echo
    ;;
  help|-h|--help)
    usage
    ;;
  *)
    usage >&2
    exit 2
    ;;
esac
