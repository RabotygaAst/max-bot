param(
  [string]$ONEC_BASE_URL = $env:ONEC_BASE_URL,
  [string]$ONEC_TOKEN = $env:ONEC_TOKEN,
  [string]$MAX_USER_ID = $env:MAX_USER_ID,
  [string]$CHAT_ID = $env:CHAT_ID,
  [string]$ACCOUNT_NUMBER = $env:ACCOUNT_NUMBER
)
$ErrorActionPreference = 'Stop'
if (!$ONEC_BASE_URL -or !$ONEC_TOKEN -or !$MAX_USER_ID -or !$CHAT_ID -or !$ACCOUNT_NUMBER) { throw 'ONEC_BASE_URL, ONEC_TOKEN, MAX_USER_ID, CHAT_ID, ACCOUNT_NUMBER are required' }
$Base = $ONEC_BASE_URL.TrimEnd('/')
$Headers = @{ Authorization = "Bearer $ONEC_TOKEN"; 'Content-Type' = 'application/json' }
$Findings = New-Object System.Collections.Generic.List[string]
function Invoke-Step($Method, $Path, $Body = $null) {
  Write-Host "`n>>> $Method $Base$Path"
  if ($Body) { Write-Host $Body }
  try { $script:Response = Invoke-RestMethod -Method $Method -Uri "$Base$Path" -Headers $Headers -Body $Body; $script:Response | ConvertTo-Json -Depth 20 | Write-Host }
  catch { $Findings.Add("$Method $Path failed: $($_.Exception.Message)"); if ($_.ErrorDetails.Message) { Write-Host $_.ErrorDetails.Message }; $script:Response = $null }
}
Invoke-Step POST '/max/v1/users/start' (@{max_user_id=[int64]$MAX_USER_ID;chat_id=[int64]$CHAT_ID;first_name='Smoke';source='MAX'} | ConvertTo-Json)
Invoke-Step POST '/max/v1/consents' (@{max_user_id=[int64]$MAX_USER_ID;consent_version='smoke-real-1c';source='MAX'} | ConvertTo-Json)
Invoke-Step POST '/max/v1/account-link/start' (@{max_user_id=[int64]$MAX_USER_ID;account_number=$ACCOUNT_NUMBER;source='MAX'} | ConvertTo-Json)
$Code = Read-Host 'Введите код подтверждения, полученный штатным каналом 1С'
Invoke-Step POST '/max/v1/account-link/confirm' (@{max_user_id=[int64]$MAX_USER_ID;account_number=$ACCOUNT_NUMBER;code=$Code;source='MAX'} | ConvertTo-Json)
$AccountID = $Response.data.account_id
if (!$AccountID) { throw 'account_id пустой: 1С не вернула идентификатор ЛС' }
Invoke-Step GET "/max/v1/accounts?max_user_id=$MAX_USER_ID"
Invoke-Step GET "/max/v1/accounts/$AccountID/balance?max_user_id=$MAX_USER_ID"
Invoke-Step GET "/max/v1/accounts/$AccountID/meters?max_user_id=$MAX_USER_ID"
$MeterID = $Response.data[0].meter_id
if (!$MeterID) { $Findings.Add("meters пустой: проверьте, есть ли в базе приборы по ЛС $AccountID") } else { Invoke-Step POST "/max/v1/accounts/$AccountID/meters/$MeterID/readings" (@{period=(Get-Date -Format 'yyyy-MM');value=1;source='MAX';max_user_id=[int64]$MAX_USER_ID;message_id='smoke-real-1c';operation_id="smoke-real-1c-$([DateTimeOffset]::Now.ToUnixTimeSeconds())"} | ConvertTo-Json) }
Invoke-Step GET "/max/v1/accounts/$AccountID/invoice?period=$(Get-Date -Format 'yyyy-MM')&max_user_id=$MAX_USER_ID"
Invoke-Step POST "/max/v1/accounts/$AccountID/appeals" (@{max_user_id=[int64]$MAX_USER_ID;category='smoke';text='Smoke test';source='MAX';message_id='smoke-real-1c';operation_id="smoke-real-1c-appeal-$([DateTimeOffset]::Now.ToUnixTimeSeconds())"} | ConvertTo-Json)
Write-Host "`n=== Подозрительные ответы / findings ==="
if ($Findings.Count -eq 0) { Write-Host 'Не обнаружены скриптом' } else { $Findings | ForEach-Object { Write-Host $_ } }
