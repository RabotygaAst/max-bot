#!/usr/bin/env bash
set -euo pipefail

: "${ONEC_BASE_URL:?ONEC_BASE_URL is required, e.g. https://server/base/hs}"
: "${ONEC_TOKEN:?ONEC_TOKEN is required}"
: "${MAX_USER_ID:?MAX_USER_ID is required}"
: "${CHAT_ID:?CHAT_ID is required}"
: "${ACCOUNT_NUMBER:?ACCOUNT_NUMBER is required}"

BASE="${ONEC_BASE_URL%/}"
AUTH=( -H "Authorization: Bearer ${ONEC_TOKEN}" -H "Content-Type: application/json" )
findings=()

call() {
  local method="$1" path="$2" body="${3:-}"
  echo "\n>>> ${method} ${BASE}${path}"
  [[ -n "$body" ]] && echo "$body"
  local tmp status
  tmp=$(mktemp)
  if [[ -n "$body" ]]; then
    status=$(curl -sS -o "$tmp" -w '%{http_code}' -X "$method" "${AUTH[@]}" -d "$body" "${BASE}${path}")
  else
    status=$(curl -sS -o "$tmp" -w '%{http_code}' -X "$method" "${AUTH[@]}" "${BASE}${path}")
  fi
  cat "$tmp"; echo
  if [[ "$status" -lt 200 || "$status" -ge 300 ]]; then findings+=("${method} ${path} returned HTTP ${status}"); fi
  RESPONSE=$(cat "$tmp")
  rm -f "$tmp"
}

call POST /max/v1/users/start "{\"max_user_id\":${MAX_USER_ID},\"chat_id\":${CHAT_ID},\"first_name\":\"Smoke\",\"source\":\"MAX\"}"
call POST /max/v1/consents "{\"max_user_id\":${MAX_USER_ID},\"consent_version\":\"smoke-real-1c\",\"source\":\"MAX\"}"
call POST /max/v1/account-link/start "{\"max_user_id\":${MAX_USER_ID},\"account_number\":\"${ACCOUNT_NUMBER}\",\"source\":\"MAX\"}"
read -r -p "Введите код подтверждения, полученный штатным каналом 1С: " CODE
call POST /max/v1/account-link/confirm "{\"max_user_id\":${MAX_USER_ID},\"account_number\":\"${ACCOUNT_NUMBER}\",\"code\":\"${CODE}\",\"source\":\"MAX\"}"
account_id=$(printf '%s' "$RESPONSE" | jq -r '.data.account_id // empty')
[[ -n "$account_id" ]] || { echo "account_id пустой: 1С не вернула идентификатор ЛС" >&2; exit 1; }

call GET "/max/v1/accounts?max_user_id=${MAX_USER_ID}"
call GET "/max/v1/accounts/${account_id}/balance?max_user_id=${MAX_USER_ID}"
call GET "/max/v1/accounts/${account_id}/meters?max_user_id=${MAX_USER_ID}"
meter_id=$(printf '%s' "$RESPONSE" | jq -r '.data[]? | select(.can_submit == true) | .meter_id' | head -n 1)
if [[ -z "$meter_id" ]]; then
  findings+=("нет can_submit=true точки передачи показаний по ЛС ${account_id}")
else
  reading_value=$(printf '%s' "$RESPONSE" | jq -r --arg meter_id "$meter_id" '(.data[]? | select(.meter_id == $meter_id) | (.last_value // 0)) + 1')
  call POST "/max/v1/accounts/${account_id}/meters/${meter_id}/readings" "{\"period\":\"$(date +%Y-%m)\",\"value\":${reading_value},\"source\":\"MAX\",\"max_user_id\":${MAX_USER_ID},\"message_id\":\"smoke-real-1c\",\"operation_id\":\"smoke-real-1c-$(date +%s)\"}"
  printf '%s' "$RESPONSE" | jq -e '.success == true' >/dev/null || findings+=("readings: success не true")
  printf '%s' "$RESPONSE" | jq -e '.data.posted == false' >/dev/null || findings+=("readings: posted не false")
  printf '%s' "$RESPONSE" | jq -e '(.data.document_number // "") != ""' >/dev/null || findings+=("readings: document_number пустой")
  printf '%s' "$RESPONSE" | jq -e '.data.status == "saved_unposted"' >/dev/null || findings+=("readings: status не saved_unposted")
fi
call GET "/max/v1/accounts/${account_id}/invoice?period=$(date +%Y-%m)&max_user_id=${MAX_USER_ID}"
call POST "/max/v1/accounts/${account_id}/appeals" "{\"max_user_id\":${MAX_USER_ID},\"category\":\"smoke\",\"text\":\"Smoke test\",\"source\":\"MAX\",\"message_id\":\"smoke-real-1c\",\"operation_id\":\"smoke-real-1c-appeal-$(date +%s)\"}"

echo "\n=== Подозрительные ответы / findings ==="
if ((${#findings[@]} == 0)); then echo "Не обнаружены скриптом"; else printf '%s\n' "${findings[@]}"; fi
