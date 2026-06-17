# MAX-бот ЖКХ: Go backend + интеграция с 1С

Backend-сервис принимает webhook-события от MAX, ведет диалог с пользователем, хранит идемпотентность и состояние сессии в PostgreSQL, вызывает HTTP API 1С из папки `cf_billing` и отправляет ответы пользователю через MAX Bot API.

## Что реализовано

- `POST /webhook/max` — прием событий MAX с проверкой секрета из заголовка.
- `cmd/bot-polling` — альтернативный запуск через MAX Long Polling без публичного webhook для локального тестирования.
- Быстрый ответ webhook и асинхронная обработка входящего сообщения.
- Идемпотентность по `message.body.mid` или вычисленному `event_id`.
- PostgreSQL-хранилище для `max_events` и `dialog_sessions`; без `DATABASE_URL` доступен in-memory режим для разработки.
- HTTP-клиент 1С для методов:
  - `POST /max/v1/users/start`
  - `POST /max/v1/consents`
  - `POST /max/v1/account-link/start`
  - `POST /max/v1/account-link/confirm`
  - `GET /max/v1/accounts`
  - `GET /max/v1/accounts/{account_id}/balance`
  - `GET /max/v1/accounts/{account_id}/meters`
  - `POST /max/v1/accounts/{account_id}/meters/{meter_id}/readings`
  - `POST /max/v1/accounts/{account_id}/appeals`
  - `GET /max/v1/reference/help`
- Служебный endpoint для уведомлений из 1С: `POST /internal/notifications/send`.
- Локальный mock-контур для smoke-тестов: PostgreSQL + MockServer для 1С и MAX `/messages`.
- Нативный dev-mock на Go для запуска без Docker и без БД: `go run ./cmd/bot/devmock`.

## Структура проекта

```text
cmd/bot/main.go                 webhook HTTP-сервер
cmd/bot-polling/main.go         long polling запуск без публичного webhook
cmd/bot/devmock/main.go             локальный mock 1С и MAX /messages без Docker
internal/config                 загрузка переменных окружения
internal/httpserver             HTTP endpoints и проверка секретов
internal/service                сценарии бота
internal/clients/max            клиент MAX API
internal/clients/onec           клиент 1С API
internal/model                  DTO MAX и 1С
internal/store                  PostgreSQL и in-memory хранилища
cf_billing                      конфигурация 1С и логика интеграции
init-db.sql                     схема локальной PostgreSQL БД
mock-onec-config.json           mock-ответы 1С и MAX для локального теста
```

## Единая инструкция запуска, активации и проверки

### 1. Подготовьте переменные окружения

Создайте `.env` из шаблона:

```bash
cp .env.example .env
```

Для локального smoke-теста менять значения не нужно: `MAX_BASE_URL` и `ONEC_BASE_URL` уже смотрят в контейнер `mock-onec`, а `ONEC_TOKEN` совпадает с `mock-onec-config.json`.

Для боевого запуска замените минимум:

```dotenv
MAX_BASE_URL=https://platform-api.max.ru
MAX_TOKEN=<реальный токен MAX-бота>
WEBHOOK_SECRET=<длинный секрет webhook>
ONEC_BASE_URL=<публичный или внутренний URL HTTP-сервиса 1С>
ONEC_TOKEN=<токен интеграции с 1С>
INTERNAL_API_TOKEN=<токен для уведомлений из 1С в backend>
```

> Не коммитьте `.env`: он содержит токены и секреты.

### 2. Запустите локальный контур

```bash
docker-compose up -d --build
```

Контур поднимает:

- `max-bot-postgres` — PostgreSQL на порту `5433` хоста;
- `max-bot-mock-onec` — MockServer на порту `1080` хоста;
- `max-bot` — backend на порту `8080` хоста.

Проверьте состояние контейнеров:

```bash
docker-compose ps
```

В логах бота должна быть строка `using PostgreSQL store`:

```bash
docker-compose logs max-bot | tail -50
```
### 2a. Альтернатива: запуск без Docker и без БД

Для полностью локального smoke-теста Docker и PostgreSQL не обязательны:

- если `DATABASE_URL` пустой или не задан, backend автоматически использует in-memory хранилище;
- `cmd/bot/devmock` поднимает простой HTTP mock на `localhost:1080` и переиспользует ответы из `mock-onec-config.json` для API 1С и MAX `/messages`.

Подготовьте отдельный локальный env-файл:

```bash
cp .env.local.example .env.local
```

В первом терминале запустите mock 1С/MAX:

```bash
go run ./cmd/bot/devmock -addr :1080 -config mock-onec-config.json
```

Во втором терминале экспортируйте переменные и запустите бота:

```bash
set -a
source .env.local
set +a
go run ./cmd/bot
```

#### Windows PowerShell

Если команда `go run ./cmd/bot/devmock ...` пишет `stat ...\cmd\bot\devmock: directory not found`, значит в вашей локальной папке нет файлов из актуальной версии проекта. Сначала проверьте наличие файла и обновите репозиторий:

```powershell
Test-Path .\cmd\bot\devmock\main.go
git pull
```

После обновления можно запустить оба процесса одной командой из корня репозитория:

```powershell
.\scripts\run-local.ps1
```

Или вручную в двух окнах PowerShell:

```powershell
# Окно 1: mock 1C/MAX
go run .\cmd\bot\devmock -addr ":1080" -config "mock-onec-config.json"
```

```powershell
# Окно 2: переменные окружения и bot
Copy-Item .env.local.example .env.local -ErrorAction SilentlyContinue
Get-Content .env.local | Where-Object { $_ -and $_ -notmatch '^\s*#' } | ForEach-Object { $name, $value = $_ -split '=', 2; Set-Item -Path "Env:$name" -Value $value }
go run .\cmd\bot
```

В логах бота должна быть строка `using in-memory store: state will be lost after restart`. После этого проверки из разделов 3–5 и 8 можно выполнять теми же `curl`-командами. Раздел 6 про PostgreSQL для такого режима не нужен: состояние хранится только в памяти процесса и сбрасывается при перезапуске.

### 2b. Windows: локальная PostgreSQL без Docker

Если PostgreSQL установлен на Windows, но БД/пользователь еще не созданы, выполните из корня репозитория:

```powershell
.\scripts\setup-postgres-local.ps1 -WriteEnvLocal
```

Если PowerShell запрещает запуск `.ps1`, используйте `.cmd`-обертку, она сама запускает PowerShell с `-ExecutionPolicy Bypass` только для текущей команды:

```cmd
.\scripts\setup-postgres-local.cmd -WriteEnvLocal
```

Либо выполните тот же обход вручную:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\setup-postgres-local.ps1 -WriteEnvLocal
```

Скрипт найдет `psql.exe`, попросит пароль администратора PostgreSQL `postgres`, создаст пользователя `maxbot`, базу `maxbot`, применит схему из `init-db.sql` и запишет локальный `DATABASE_URL` в `.env.local`.

Значение по умолчанию:

```text
postgres://maxbot:maxbot_local_2026@localhost:5432/maxbot?sslmode=disable
```

После этого webhook/debug режим на Windows можно запускать с сохранением сессий и обработанных событий в PostgreSQL. Если `.ps1` заблокирован политикой, запускайте `.cmd`:

```cmd
.\scripts\run-local.cmd -UsePostgres
```

Или напрямую через PowerShell, если выполнение `.ps1` разрешено:

```powershell
.\scripts\run-local.ps1 -UsePostgres
```

Polling-режим с реальным MAX и локальным mock 1С тоже может использовать ту же локальную PostgreSQL. Для случая с Execution Policy удобнее так:

```cmd
.\scripts\run-polling-local.cmd -MaxToken "<реальный токен MAX-бота>" -UsePostgres
```

Или напрямую через PowerShell:

```powershell
.\scripts\run-polling-local.ps1 -MaxToken "<реальный токен MAX-бота>" -UsePostgres
```

Если нужно указать другой пароль, порт или имя БД, сначала создайте БД с нужными параметрами:

```powershell
.\scripts\setup-postgres-local.ps1 -Database "maxbot" -AppUser "maxbot" -AppPassword "my_local_password" -WriteEnvLocal
```

А затем передайте DSN при запуске:

```powershell
.\scripts\run-polling-local.ps1 -MaxToken "<реальный токен MAX-бота>" -UsePostgres -DatabaseUrl "postgres://maxbot:my_local_password@localhost:5432/maxbot?sslmode=disable"
```

Linux/Docker-сценарий не меняется: `docker-compose.yml` по-прежнему поднимает PostgreSQL-контейнер и использует `DATABASE_URL` внутри Docker-сети.

Чтобы быстро проверить, что локальная PostgreSQL реально подключена и умеет сохранять записи бота, можно добавить тестовую сессию пользователя:

```cmd
.\scripts\seed-postgres-local.cmd -MaxUserId 123456789 -AccountId "ACC-000123456" -AccountNumber "000123456"
```

Скрипт вставит/обновит запись в `dialog_sessions`, добавит тестовое обработанное событие в `max_events`, создаст audit-запись в `event_logs` и сразу выведет сохраненные строки через `psql`.

### 2c. Альтернатива для реального MAX без публичного webhook: Long Polling

Если публичный HTTPS endpoint недоступен, можно оставить webhook-режим для сервера и запустить отдельный polling-процесс локально. MAX Bot API поддерживает `GET /updates` для разработки и тестирования; при таком запуске бот сам забирает входящие события и переиспользует те же сценарии обработки, что и webhook. Для production предпочтителен webhook.

Перед polling-запуском убедитесь, что у бота нет активной webhook-подписки на тот же токен, иначе MAX может не отдавать события через `/updates`.

В Windows PowerShell для теста с реальным MAX и локальным mock 1С можно запустить готовый скрипт:

```powershell
.\scripts\run-polling-local.ps1 -MaxToken "<реальный токен MAX-бота>"
```

Или вручную в двух окнах PowerShell:

```powershell
# Окно 1: локальный mock 1С для сценариев баланса/показаний/обращений
go run .\cmd\bot\devmock -addr ":1080" -config "mock-onec-config.json"
```

```powershell
# Окно 2: polling-бот, который сам читает сообщения из MAX
$env:MAX_BASE_URL="https://platform-api.max.ru"
$env:MAX_TOKEN="<реальный токен MAX-бота>"
$env:ONEC_BASE_URL="http://localhost:1080"
$env:ONEC_TOKEN="MOCK_ONEC_TOKEN"
$env:INTERNAL_API_TOKEN="CHANGE_ME_INTERNAL_TOKEN"
$env:REQUEST_TIMEOUT_SECONDS="10"
$env:POLLING_LIMIT="100"
$env:POLLING_TIMEOUT_SECONDS="30"
$env:POLLING_RETRY_SECONDS="5"
$env:POLLING_TYPES="message_created,message_callback,bot_started"
Remove-Item Env:DATABASE_URL -ErrorAction SilentlyContinue
go run .\cmd\bot-polling
```

После запуска напишите боту в MAX `/start`, нажмите «Авторизоваться» или отправьте `авторизоваться`, затем `000123456`, `1234`, `баланс`, `показания`, `справка`. Ответы будут отправляться в настоящий чат MAX, а данные 1С будут браться из локального mock.

В Linux/macOS аналогично:

```bash
MAX_BASE_URL=https://platform-api.max.ru \
MAX_TOKEN=<реальный токен MAX-бота> \
ONEC_BASE_URL=http://localhost:1080 \
ONEC_TOKEN=MOCK_ONEC_TOKEN \
INTERNAL_API_TOKEN=CHANGE_ME_INTERNAL_TOKEN \
POLLING_TYPES=message_created,message_callback,bot_started \
go run ./cmd/bot-polling
```

### 3. Проверьте health-check

```bash
curl http://localhost:8080/healthz
```

Ожидаемый ответ:

```json
{"status":"ok"}
```

### 4. Прогоните сценарий диалога без реального MAX

`/debug/send-test-update` имитирует входящее сообщение от MAX и удобен для локальной проверки без публичного webhook:

```bash
curl -s -X POST http://localhost:8080/debug/send-test-update \
  -H 'Content-Type: application/json' \
  -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-001","text":"/start"}'

curl -s -X POST http://localhost:8080/debug/send-test-update \
  -H 'Content-Type: application/json' \
  -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-002","text":"авторизоваться"}'

curl -s -X POST http://localhost:8080/debug/send-test-update \
  -H 'Content-Type: application/json' \
  -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-003","text":"000123456"}'

curl -s -X POST http://localhost:8080/debug/send-test-update \
  -H 'Content-Type: application/json' \
  -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-004","text":"1234"}'

curl -s -X POST http://localhost:8080/debug/send-test-update \
  -H 'Content-Type: application/json' \
  -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-005","text":"баланс"}'

curl -s -X POST http://localhost:8080/debug/send-test-update \
  -H 'Content-Type: application/json' \
  -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-006","text":"показания"}'

curl -s -X POST http://localhost:8080/debug/send-test-update \
  -H 'Content-Type: application/json' \
  -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-007","text":"показание MTR-001 245.678"}'

curl -s -X POST http://localhost:8080/debug/send-test-update \
  -H 'Content-Type: application/json' \
  -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-008","text":"обращение не убран подъезд"}'

curl -s -X POST http://localhost:8080/debug/send-test-update \
  -H 'Content-Type: application/json' \
  -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-009","text":"справка"}'
```

Каждый вызов должен вернуть:

```json
{"success":true}
```

Ответы бота можно увидеть в журнале запросов MockServer к `/messages`:

```bash
curl -s -X PUT http://localhost:1080/mockserver/retrieve \
  -H 'Content-Type: application/json' \
  -d '{"path":"/messages"}'
```
При запуске через `cmd/bot/devmock` вместо MockServer ответы на `/messages` отображаются в логах процесса `go run ./cmd/bot/devmock`.

### 5. Проверьте webhook с секретом

```bash
curl -s -X POST http://localhost:8080/webhook/max \
  -H 'Content-Type: application/json' \
  -H 'X-Max-Webhook-Secret: CHANGE_ME_SECRET_2026' \
  -d '{
    "update_type":"message_created",
    "timestamp":1778068800000,
    "message":{
      "sender":{"user_id":123456789,"first_name":"Иван"},
      "recipient":{"chat_id":987654321},
      "body":{"mid":"smoke-webhook-001","text":"баланс"}
    }
  }'
```

Ожидаемый ответ:

```json
{"success":true}
```

Проверка отрицательного сценария:

```bash
curl -i -s -X POST http://localhost:8080/webhook/max \
  -H 'Content-Type: application/json' \
  -H 'X-Max-Webhook-Secret: WRONG_SECRET' \
  -d '{"update_type":"message_created","message":{"sender":{"user_id":1},"recipient":{"chat_id":1},"body":{"mid":"bad-secret","text":"/start"}}}'
```

Ожидается HTTP `401 Unauthorized`.

### 6. Проверьте БД

```bash
docker-compose exec -T postgres psql -U maxbot -d maxbot -c "SELECT event_id, status, operation_id, error_text FROM max_events ORDER BY received_at DESC LIMIT 10;"
```

Все smoke-события должны иметь статус `processed` и пустой `error_text`.

```bash
docker-compose exec -T postgres psql -U maxbot -d maxbot -c "SELECT max_user_id, active_account_id, temp, updated_at FROM dialog_sessions;"
```

После ввода кода `1234` у пользователя должен быть активный счет `ACC-000123456`.

### 7. Активируйте реальный webhook в MAX

После успешной локальной проверки опубликуйте backend по HTTPS, например через reverse proxy или временно через ngrok:

```bash
ngrok http 8080
```

Укажите в настройках MAX-бота:

- URL webhook: `https://<ваш-домен>/webhook/max`;
- секрет webhook: значение `WEBHOOK_SECRET` из `.env`;
- токен бота: значение `MAX_TOKEN`.

Если регистрация webhook выполняется через API MAX, используйте официальный метод платформы MAX для управления webhook и передайте тот же URL и секрет. После активации отправьте боту в MAX команды `/start`, нажмите «Авторизоваться» или отправьте `авторизоваться`, затем `000123456`, `1234`, `баланс`, `показания`, `справка` и проверьте логи backend.

### 8. Отправьте служебное уведомление из 1С

```bash
curl -s -X POST http://localhost:8080/internal/notifications/send \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer CHANGE_ME_INTERNAL_TOKEN_2026' \
  -d '{
    "chat_id":987654321,
    "text":"Статус обращения №ОБР-000001 изменен: в работе.",
    "operation_id":"1c-20260601-000001"
  }'
```

Ожидаемый ответ:

```json
{"success":true}
```

## Команды пользователя

```text
/start
авторизоваться
000123456
1234
баланс
показания
показание MTR-001 245.678
обращение не убран подъезд
справка
```

## Переменные окружения

| Переменная | Назначение |
|---|---|
| `HTTP_ADDR` | Адрес HTTP-сервера, например `:8080`. |
| `REQUEST_TIMEOUT_SECONDS` | Таймаут вызовов MAX и 1С. |
| `MAX_BASE_URL` | Базовый URL MAX API или локального mock-сервера. |
| `MAX_TOKEN` | Токен MAX Bot API. |
| `WEBHOOK_SECRET` | Секрет webhook-подписки. Если пустой, бот сгенерирует временный секрет при старте и выведет его в лог. |
| `WEBHOOK_SECRET_HEADER` | Имя заголовка с секретом, по умолчанию `X-Max-Webhook-Secret`. |
| `ONEC_BASE_URL` | Базовый URL HTTP API 1С. |
| `ONEC_TOKEN` | Токен интеграции с 1С. |
| `INTERNAL_API_TOKEN` | Токен для служебных вызовов от 1С к backend. |
| `DATABASE_URL` | PostgreSQL DSN. Если не задан, используется in-memory хранилище. |
| `POLLING_LIMIT` | Размер пачки `GET /updates`, по умолчанию `100`, максимум `1000`. |
| `POLLING_TIMEOUT_SECONDS` | Long polling timeout для `GET /updates`, по умолчанию `30`, максимум `90`. |
| `POLLING_RETRY_SECONDS` | Пауза перед повтором после ошибки polling, по умолчанию `5`. |
| `POLLING_TYPES` | CSV-список типов событий для polling, по умолчанию `message_created,message_callback,bot_started`. |

## Диагностика

```bash
docker-compose logs max-bot | tail -100
docker-compose logs postgres | tail -100
docker-compose logs mock-onec | tail -100
```

Проверка ошибок в обработке событий:

```bash
docker-compose exec -T postgres psql -U maxbot -d maxbot -c "SELECT * FROM max_events WHERE status <> 'processed' ORDER BY received_at DESC;"
```

Повторная чистая проверка с удалением данных:

```bash
docker-compose down -v
docker-compose up -d --build
```

## Production-рекомендации

1. Использовать постоянный HTTPS endpoint и стабильную регистрацию webhook в MAX.
2. Хранить токены в secret manager, а не в файлах на сервере.
3. Добавить очередь/outbox и retries для технических ошибок MAX и 1С.
4. Настроить метрики и алерты: ошибки 1С, ошибки MAX, длительность обработки, дубликаты webhook.
5. Согласовать с ИБ маскирование ПДн в логах и текстах ответов.
6. Сверить `internal/clients/max/client.go` с актуальным форматом отправки сообщений в MAX Bot API перед боевым включением.
## Расширенные ЖКХ-сценарии MAX

Добавлены пользовательские команды: `организация`, `об организации`, `аварийная`, `диспетчерская`, `квитанция [YYYY-MM]`, `счет [YYYY-MM]`, `оплатить`, `оплата`, `отключения`, `перерывы`, `нет воды`, `нет света`, `запись`, `запись <topic_id>`, а также категории обращений `обращение`, `заявка`, `авария`, `жалоба`.

### Новые ожидаемые endpoint 1С

- `GET /max/v1/reference/organization`
- `GET /max/v1/reference/emergency`
- `GET /max/v1/accounts/{account_id}/invoice?period=YYYY-MM&max_user_id={max_user_id}`
- `POST /max/v1/accounts/{account_id}/payment-link`
- `GET /max/v1/accounts/{account_id}/outages?max_user_id={max_user_id}`
- `GET /max/v1/reference/appointment-topics`
- `POST /max/v1/accounts/{account_id}/appointments`

Endpoint `POST /internal/notifications/send` сохранил обратную совместимость и дополнительно принимает optional-поля `type` и `account_id`.

### Smoke-test новых сценариев через debug endpoint

После запуска `docker-compose up -d --build` можно отправить события:

```bash
curl -s -X POST http://localhost:8080/debug/send-test-update -H 'Content-Type: application/json' -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-org-001","text":"организация"}'
curl -s -X POST http://localhost:8080/debug/send-test-update -H 'Content-Type: application/json' -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-emergency-001","text":"аварийная"}'
curl -s -X POST http://localhost:8080/debug/send-test-update -H 'Content-Type: application/json' -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-start-001","text":"/start"}'
curl -s -X POST http://localhost:8080/debug/send-test-update -H 'Content-Type: application/json' -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-auth-001","text":"авторизоваться"}'
curl -s -X POST http://localhost:8080/debug/send-test-update -H 'Content-Type: application/json' -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-auth-002","text":"000123456"}'
curl -s -X POST http://localhost:8080/debug/send-test-update -H 'Content-Type: application/json' -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-auth-003","text":"1234"}'
curl -s -X POST http://localhost:8080/debug/send-test-update -H 'Content-Type: application/json' -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-invoice-001","text":"квитанция 2026-05"}'
curl -s -X POST http://localhost:8080/debug/send-test-update -H 'Content-Type: application/json' -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-payment-001","text":"оплатить"}'
curl -s -X POST http://localhost:8080/debug/send-test-update -H 'Content-Type: application/json' -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-outages-001","text":"отключения"}'
curl -s -X POST http://localhost:8080/debug/send-test-update -H 'Content-Type: application/json' -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-appointment-001","text":"запись"}'
curl -s -X POST http://localhost:8080/debug/send-test-update -H 'Content-Type: application/json' -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-appointment-002","text":"запись billing"}'
```

Каждый запрос должен вернуть `{"success":true}`, а ответы бота отправляются в mock MAX `/messages`.

## Постоянное состояние бота

PostgreSQL — основной режим хранения состояния. Таблицы `max_users` и `account_links` хранят постоянную авторизацию MAX-пользователя и активный лицевой счет; `dialog_sessions` используется только для временных шагов диалога. Без `DATABASE_URL` запускается только development fallback in-memory, состояние в нем теряется после перезапуска.

## PostgreSQL как основной режим

Docker Compose поднимает PostgreSQL и применяет `init-db.sql`. Для ручного запуска задайте `DATABASE_URL`, например:

```bash
export DATABASE_URL='postgres://maxbot:maxbot@localhost:5432/maxbot?sslmode=disable'
go run ./cmd/bot
```

Добавлены таблицы: `max_users`, `account_links`, `billing_accruals_cache`, `invoices_cache`, `payments_cache`, `appointments`, `appeals_cache`, `notification_logs`.

## Windows без Docker

1. Установите PostgreSQL 15+ и убедитесь, что `psql.exe` доступен в `PATH` либо находится в `C:\Program Files\PostgreSQL\*\bin`.
2. Выполните:

```powershell
Copy-Item .env.local.example .env.local -ErrorAction SilentlyContinue
.\scripts\setup-postgres-local.ps1 -WriteEnvLocal
.\scripts\seed-postgres-local.ps1
.\scripts\run-local.ps1 -UsePostgres
```

CMD-обертки:

```cmd
.\scripts\setup-postgres-local.cmd -WriteEnvLocal
.\scripts\seed-postgres-local.cmd
.\scripts\run-local.cmd -UsePostgres
```

## Seed тестовых данных

```powershell
.\scripts\seed-postgres-local.ps1
```

Seed создает пользователя `123456789`, чат `987654321`, лицевой счет `000123456`, кэш баланса за `2026-05`, квитанцию, платеж, обращение и запись на прием.

## Новые команды пользователя

```text
/start
меню
помощь
справка
организация
об организации
контакты
аварийная
диспетчерская
авторизоваться
привязать <номер ЛС>
код <номер ЛС> <код>
<номер ЛС>
<код>
баланс
квитанция
квитанция 2026-05
счет 2026-05
оплатить
оплата
показания
показание MTR-001 245.678
обращение
обращение не убран подъезд
заявка не работает домофон
авария прорвало трубу
жалоба некачественная уборка
отключения
перерывы
нет воды
нет света
запись
запись billing
прием
```

## Новые endpoint 1С

```text
GET  /max/v1/reference/organization
GET  /max/v1/reference/emergency
GET  /max/v1/accounts/{account_id}/invoice?period=YYYY-MM&max_user_id={max_user_id}
POST /max/v1/accounts/{account_id}/payment-link
GET  /max/v1/accounts/{account_id}/outages?max_user_id={max_user_id}
GET  /max/v1/reference/appointment-topics
POST /max/v1/accounts/{account_id}/appointments
```

## Smoke-тесты

После запуска `scripts/run-local.ps1 -UsePostgres` debug endpoint должен возвращать `{"success":true}`:

```bash
curl -s -X POST http://localhost:8080/debug/send-test-update -H 'Content-Type: application/json' -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-org-001","text":"организация"}'
curl -s -X POST http://localhost:8080/debug/send-test-update -H 'Content-Type: application/json' -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-emergency-001","text":"аварийная"}'
curl -s -X POST http://localhost:8080/debug/send-test-update -H 'Content-Type: application/json' -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-start-001","text":"/start"}'
curl -s -X POST http://localhost:8080/debug/send-test-update -H 'Content-Type: application/json' -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-auth-001","text":"авторизоваться"}'
curl -s -X POST http://localhost:8080/debug/send-test-update -H 'Content-Type: application/json' -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-balance-001","text":"баланс"}'
curl -s -X POST http://localhost:8080/debug/send-test-update -H 'Content-Type: application/json' -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-invoice-001","text":"квитанция 2026-05"}'
curl -s -X POST http://localhost:8080/debug/send-test-update -H 'Content-Type: application/json' -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-payment-001","text":"оплатить"}'
curl -s -X POST http://localhost:8080/debug/send-test-update -H 'Content-Type: application/json' -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-outages-001","text":"отключения"}'
curl -s -X POST http://localhost:8080/debug/send-test-update -H 'Content-Type: application/json' -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-appointment-001","text":"запись"}'
curl -s -X POST http://localhost:8080/debug/send-test-update -H 'Content-Type: application/json' -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-appointment-002","text":"запись billing"}'
```

## Проверка сохранения после перезапуска

1. Запустите бота с PostgreSQL.
2. Авторизуйте пользователя (`авторизоваться` → `000123456` → `1234`).
3. Остановите бота.
4. Запустите снова с тем же `DATABASE_URL`.
5. Отправьте `/start`.
6. Бот должен ответить, что пользователь уже авторизован, без повторного ввода ЛС.

SQL-проверка:

```sql
SELECT * FROM max_users;
SELECT * FROM account_links;
SELECT * FROM billing_accruals_cache;
SELECT * FROM invoices_cache;
SELECT * FROM appointments;
```
