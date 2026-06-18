#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

echo "== Local MAX ↔ Go bot ↔ ONEC mock exchange =="
echo "This script does not require published 1C or a real ONEC_BASE_URL."

go test ./internal/service -run 'TestLocalExchangeFullUserScenario|TestGuestButtonsCallbacks|TestAuthorizedButtonsCallbacks|TestOneCRequestsReachedMock|TestReadingRequestContainsOperationID|TestAccountLinkIsReusedAfterStart' -v

go test ./internal/clients/onec -run TestOneCContractMatchesCfBilling -v

if command -v docker >/dev/null 2>&1; then
  echo "Docker is available; no Docker contour is required for the httptest-based exchange."
else
  echo "Docker is not available; skipped optional Docker/mockserver contour."
fi

cat <<'REPORT'
PASS: local exchange tests completed.
Checked callback payloads: authorize, balance, meters, invoice, payment, appeal_start, outages, appointment, organization, emergency, help.
Checked ONEC mock requests: users/start, consents, account-link/start, account-link/confirm, balance, meters, readings, invoice, payment-link, appeals, outages, appointment-topics, appointments, organization, emergency, help.
Checked MAX mock messages: greeting, authorization, account link, balance, meters, reading, invoice, payment, appeal, outages, appointment, organization, emergency, help.
REPORT
