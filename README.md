# MAX-бот ЖКХ: Go backend + интеграция с 1С

Backend-сервис принимает webhook-события от MAX, ведет диалог с пользователем, хранит идемпотентность и состояние сессии в PostgreSQL, вызывает HTTP API 1С из папки `cf_billing` и отправляет ответы пользователю через MAX Bot API.

## Что реализовано

- `POST /webhook/max` — прием событий MAX с проверкой секрета из заголовка.
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
- Нативный dev-mock на Go для запуска без Docker и без БД: `go run ./cmd/devmock`.

## Структура проекта

```text
cmd/bot/main.go                 точка входа
cmd/devmock/main.go             локальный mock 1С и MAX /messages без Docker
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
- `cmd/devmock` поднимает простой HTTP mock на `localhost:1080` и переиспользует ответы из `mock-onec-config.json` для API 1С и MAX `/messages`.

Подготовьте отдельный локальный env-файл:

```bash
cp .env.local.example .env.local
```

В первом терминале запустите mock 1С/MAX:

```bash
go run ./cmd/devmock -addr :1080 -config mock-onec-config.json
```

Во втором терминале экспортируйте переменные и запустите бота:

```bash
set -a
source .env.local
set +a
go run ./cmd/bot
```

#### Windows PowerShell

Если команда `go run ./cmd/devmock ...` пишет `stat ...\cmd\devmock: directory not found`, значит в вашей локальной папке нет файлов из актуальной версии проекта. Сначала проверьте наличие файла и обновите репозиторий:

```powershell
Test-Path .\cmd\devmock\main.go
git pull
```

После обновления можно запустить оба процесса одной командой из корня репозитория:

```powershell
.\scripts\run-local.ps1
```

Или вручную в двух окнах PowerShell:

```powershell
# Окно 1: mock 1C/MAX
go run .\cmd\devmock -addr ":1080" -config "mock-onec-config.json"
```

```powershell
# Окно 2: переменные окружения и bot
Copy-Item .env.local.example .env.local -ErrorAction SilentlyContinue
Get-Content .env.local | Where-Object { $_ -and $_ -notmatch '^\s*#' } | ForEach-Object { $name, $value = $_ -split '=', 2; Set-Item -Path "Env:$name" -Value $value }
go run .\cmd\bot
```

В логах бота должна быть строка `using in-memory store (for development only)`. После этого проверки из разделов 3–5 и 8 можно выполнять теми же `curl`-командами. Раздел 6 про PostgreSQL для такого режима не нужен: состояние хранится только в памяти процесса и сбрасывается при перезапуске.


### 2b. Локальный запуск с реальным MAX без webhook: Long Polling

Если бот должен работать на вашем ПК без публичного HTTPS/webhook, включите Long Polling. В этом режиме backend сам опрашивает MAX `GET /updates`, продолжает поднимать локальный HTTP API для 1С и принимает уведомления от 1С через `POST /internal/notifications/send`.

В `.env.local` или `.env` укажите реальные токены и адрес 1С, доступный с вашего ПК:

```dotenv
MAX_BASE_URL=https://platform-api.max.ru
MAX_TOKEN=<реальный токен MAX-бота>
ONEC_BASE_URL=http://localhost:8081/hs/max
ONEC_TOKEN=<токен интеграции с 1С>
INTERNAL_API_TOKEN=<токен для вызовов из 1С в backend>
MAX_POLLING_ENABLED=true
MAX_POLLING_LIMIT=100
MAX_POLLING_TIMEOUT_SECONDS=30
MAX_POLLING_TYPES=message_created,message_callback
```

Запустите backend:

```bash
set -a
source .env.local
set +a
go run ./cmd/bot
```

Для Windows PowerShell:

```powershell
Get-Content .env.local | Where-Object { $_ -and $_ -notmatch '^\s*#' } | ForEach-Object { $name, $value = $_ -split '=', 2; Set-Item -Path "Env:$name" -Value $value }
go run .\cmd\bot
```

Важно: у бота должен быть отключен webhook/subscription в MAX, иначе `GET /updates` не будет использоваться как основной канал доставки событий. Long Polling подходит для локальной разработки и тестирования; для production MAX рекомендует Webhook.

1С может отправлять исходящие уведомления пользователю в локальный backend так:

```bash
curl -X POST http://localhost:8080/internal/notifications/send \
  -H 'Authorization: Bearer <INTERNAL_API_TOKEN>' \
  -H 'Content-Type: application/json' \
  -d '{"chat_id":987654321,"text":"Начисление обновлено","operation_id":"onec-001"}'
```


### 2c. Как связать локальный bot backend с Apache, который публикует HTTP-сервис 1С

Да, локальный сервер из этого репозитория можно связать с Apache, который уже привязан к 1С. Для этого bot backend должен видеть HTTP-публикацию 1С по сети, а Apache/1С должны видеть локальный HTTP API bot backend для исходящих уведомлений. Webhook MAX при этом не нужен: входящие сообщения из MAX забираются через Long Polling.

Рекомендуемая локальная схема:

```text
MAX Bot API <--long polling--> max-bot на вашем ПК:8080
                                  |
                                  | HTTP + Bearer ONEC_TOKEN
                                  v
Apache на сервере 1С --> публикация HTTP-сервиса 1С, например /hs/max

1С/Apache --HTTP + Bearer INTERNAL_API_TOKEN--> max-bot на вашем ПК:8080/internal/notifications/send
```

Что нужно подготовить:

1. На ПК с ботом установите Go 1.22+ или Docker, склонируйте репозиторий и создайте локальный `.env.local` из `.env.local.example`.
2. На сервере 1С проверьте публикацию HTTP-сервиса через Apache. Например, если Apache доступен как `http://onec-server`, а публикация называется `hs/max`, то в боте укажите `ONEC_BASE_URL=http://onec-server/hs/max`.
3. В Apache/1С должен быть включен доступ к методам, которые вызывает бот:
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
4. Apache/1С должен принимать заголовок `Authorization: Bearer <ONEC_TOKEN>` от бота. Значение `<ONEC_TOKEN>` должно совпадать с `ONEC_TOKEN` в `.env.local`.
5. Если 1С будет сама отправлять сообщения пользователю через бота, сервер 1С должен иметь сетевой доступ к ПК с ботом по адресу вида `http://<ip-вашего-пк>:8080/internal/notifications/send`, а запрос должен содержать `Authorization: Bearer <INTERNAL_API_TOKEN>`.

Минимальный `.env.local` для связки с Apache/1С:

```dotenv
HTTP_ADDR=:8080
MAX_BASE_URL=https://platform-api.max.ru
MAX_TOKEN=<токен MAX-бота>
MAX_POLLING_ENABLED=true
MAX_POLLING_LIMIT=100
MAX_POLLING_TIMEOUT_SECONDS=30
MAX_POLLING_TYPES=message_created,message_callback

ONEC_BASE_URL=http://<apache-host-or-ip>/hs/max
ONEC_TOKEN=<токен, который проверяет HTTP-сервис 1С>
INTERNAL_API_TOKEN=<токен, с которым 1С вызывает backend бота>
DATABASE_URL=
```

Проверки перед запуском с реальным MAX:

```bash
# ПК с ботом должен видеть Apache/1С
curl -i -H "Authorization: Bearer <ONEC_TOKEN>" http://<apache-host-or-ip>/hs/max/max/v1/reference/help

# Сервер 1С должен видеть bot backend на ПК
curl -i http://<ip-вашего-пк>:8080/healthz
```

Если второй curl с сервера 1С не проходит, откройте порт `8080` в firewall Windows/Linux или запустите backend на адресе всех интерфейсов (`HTTP_ADDR=0.0.0.0:8080`). Если Apache и бот находятся на одном ПК, можно использовать `ONEC_BASE_URL=http://localhost/hs/max` или порт Apache, например `http://localhost:8081/hs/max`.

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
  -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-002","text":"согласен"}'

curl -s -X POST http://localhost:8080/debug/send-test-update \
  -H 'Content-Type: application/json' \
  -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-003","text":"привязать 000123456"}'

curl -s -X POST http://localhost:8080/debug/send-test-update \
  -H 'Content-Type: application/json' \
  -d '{"user_id":123456789,"chat_id":987654321,"mid":"smoke-004","text":"код 000123456 1234"}'

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
При запуске через `cmd/devmock` вместо MockServer ответы на `/messages` отображаются в логах процесса `go run ./cmd/devmock`.

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

После команды `код 000123456 1234` у пользователя должен быть активный счет `ACC-000123456`.

### 7. Активируйте реальный webhook в MAX

После успешной локальной проверки опубликуйте backend по HTTPS, например через reverse proxy или временно через ngrok:

```bash
ngrok http 8080
```

Укажите в настройках MAX-бота:

- URL webhook: `https://<ваш-домен>/webhook/max`;
- секрет webhook: значение `WEBHOOK_SECRET` из `.env`;
- токен бота: значение `MAX_TOKEN`.

Если регистрация webhook выполняется через API MAX, используйте официальный метод платформы MAX для управления webhook и передайте тот же URL и секрет. После активации отправьте боту в MAX команды `/start`, `согласен`, `привязать 000123456`, `код 000123456 1234`, `баланс`, `показания`, `справка` и проверьте логи backend.

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
согласен
привязать 000123456
код 000123456 1234
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
| `MAX_POLLING_ENABLED` | Включает локальное получение событий через MAX Long Polling без публичного webhook. |
| `MAX_POLLING_LIMIT` | Максимальное число событий за один запрос `GET /updates`, от 1 до 1000. |
| `MAX_POLLING_TIMEOUT_SECONDS` | Таймаут long polling запроса к MAX, от 0 до 90 секунд. |
| `MAX_POLLING_TYPES` | Список типов событий для long polling, например `message_created,message_callback`. |
| `ONEC_BASE_URL` | Базовый URL HTTP API 1С. |
| `ONEC_TOKEN` | Токен интеграции с 1С. |
| `INTERNAL_API_TOKEN` | Токен для служебных вызовов от 1С к backend. |
| `DATABASE_URL` | PostgreSQL DSN. Если не задан, используется in-memory хранилище. |

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