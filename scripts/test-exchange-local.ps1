$ErrorActionPreference = "Stop"
Set-Location (Join-Path $PSScriptRoot "..")

Write-Host "== Local MAX ↔ Go bot ↔ ONEC mock exchange =="
Write-Host "This script does not require published 1C or a real ONEC_BASE_URL."

go test ./internal/service -run 'TestLocalExchangeFullUserScenario|TestGuestButtonsCallbacks|TestAuthorizedButtonsCallbacks|TestOneCRequestsReachedMock|TestReadingRequestContainsOperationID|TestAccountLinkIsReusedAfterStart' -v
go test ./internal/clients/onec -run TestOneCContractMatchesCfBilling -v

if (Get-Command docker -ErrorAction SilentlyContinue) {
  Write-Host "Docker is available; no Docker contour is required for the httptest-based exchange."
} else {
  Write-Host "Docker is not available; skipped optional Docker/mockserver contour."
}

Write-Host "PASS: local exchange tests completed."
Write-Host "Checked callback payloads: authorize, balance, meters, invoice, payment, appeal_start, outages, appointment, organization, emergency, help."
Write-Host "Checked ONEC mock requests: users/start, consents, account-link/start, account-link/confirm, balance, meters, readings, invoice, payment-link, appeals, outages, appointment-topics, appointments, organization, emergency, help."
Write-Host "Checked MAX mock messages: greeting, authorization, account link, balance, meters, reading, invoice, payment, appeal, outages, appointment, organization, emergency, help."
