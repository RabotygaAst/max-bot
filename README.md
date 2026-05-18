# MAX-бот ЖКХ: Go backend + интеграция с 1С

Минимальный каркас backend-сервиса для MAX-бота по ТЗ: прием Webhook, проверка секрета, идемпотентность входящих событий, вызовы HTTP API 1С и отправка сообщений пользователю.

## Что реализовано

- `POST /webhook/max` — прием событий MAX.
- Проверка Webhook secret через заголовок, по умолчанию `X-Max-Webhook-Secret`.
- Быстрый ответ Webhook и асинхронная обработка события.
- Идемпотентность по `message.body.mid` или вычисленному `event_id`.
- Вызовы 1С по методам:
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
- JSON-логи через `log/slog`.

## Структура проекта

```text
cmd/bot/main.go                 точка входа
internal/config                 загрузка переменных окружения
internal/httpserver             HTTP endpoints и проверка секретов
internal/service                сценарии бота
internal/clients/max            клиент MAX API
internal/clients/onec           клиент 1С API
internal/model                  DTO MAX и 1С
internal/store                  интерфейс хранилища и in-memory реализация
```

## Быстрый запуск

```bash
cp .env.example .env
# Сгенерируйте WEBHOOK_SECRET и вставьте его в .env
go run ./cmd/bot generate-webhook-secret
# Заполните остальные переменные .env реальными значениями
export $(grep -v '^#' .env | xargs)
go run ./cmd/bot
```

Проверка:

```bash
curl http://localhost:8080/healthz
```

## Генерация Webhook secret

Для безопасного секрета используйте встроенный генератор. Он печатает только секрет в stdout и не требует заполненных `MAX_TOKEN`, `ONEC_BASE_URL`, `ONEC_TOKEN`, `WEBHOOK_SECRET` или `INTERNAL_API_TOKEN`:

```bash
go run ./cmd/bot generate-webhook-secret
# или
go run ./cmd/bot --generate-webhook-secret
```

Скопируйте результат в `WEBHOOK_SECRET` в `.env` и используйте то же значение при регистрации webhook в MAX.

## Пример Webhook-запроса

```bash
curl -X POST http://localhost:8080/webhook/max \
  -H 'Content-Type: application/json' \
  -H 'X-Max-Webhook-Secret: <значение WEBHOOK_SECRET из .env>' \
  -d '{
    "update_type":"message_created",
    "timestamp":1778068800000,
    "message":{
      "sender":{"user_id":123456789,"first_name":"Иван"},
      "recipient":{"chat_id":987654321},
      "body":{"mid":"mid.example.1","text":"/start"}
    }
  }'
```

Ожидаемый ответ Webhook:

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
показание MTR-001 123.456
обращение не убран подъезд
справка
```

## Служебное уведомление из 1С

1С или интеграционный контур может вызвать backend, чтобы отправить пользователю уведомление о квитанции, оплате или статусе обращения.

```bash
curl -X POST http://localhost:8080/internal/notifications/send \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer CHANGE_ME_INTERNAL_TOKEN' \
  -d '{
    "chat_id": 987654321,
    "text": "Статус обращения №123 изменен: в работе.",
    "operation_id": "1c-20260506-000123"
  }'
```

## Переменные окружения

| Переменная | Назначение |
|---|---|
| `HTTP_ADDR` | адрес HTTP-сервера, например `:8080` |
| `REQUEST_TIMEOUT_SECONDS` | таймаут вызовов MAX и 1С |
| `MAX_BASE_URL` | базовый URL MAX API |
| `MAX_TOKEN` | токен MAX Bot API |
| `WEBHOOK_SECRET` | секрет Webhook-подписки; сгенерируйте командой `go run ./cmd/bot generate-webhook-secret` и сохраните в `.env` |
| `WEBHOOK_SECRET_HEADER` | имя заголовка, где ожидается секрет |
| `ONEC_BASE_URL` | базовый URL HTTP API 1С |
| `ONEC_TOKEN` | внутренний токен интеграции с 1С |
| `INTERNAL_API_TOKEN` | токен для служебных вызовов от 1С к backend |

## Как адаптировать под production

1. Заменить `MemoryStore` на PostgreSQL или Redis.
2. Добавить таблицы `max_events`, `dialog_sessions`, `outbox_messages`.
3. Заменить goroutine-обработку на очередь.
4. Добавить retries для технических ошибок 1С и MAX.
5. Добавить метрики Prometheus: количество событий, ошибки 1С, длительность обработки.
6. Настроить reverse proxy с TLS и ограничением доступа.
7. Согласовать с ИБ точный заголовок Webhook secret и правила маскирования ПДн.
8. Сверить `clients/max/client.go` с фактическим форматом отправки сообщений в MAX Bot API.

## Схема таблиц для production-хранилища

```sql
create table max_events (
  event_id text primary key,
  status text not null,
  operation_id text,
  error_text text,
  received_at timestamptz not null default now(),
  processed_at timestamptz
);

create table dialog_sessions (
  max_user_id bigint primary key,
  step text,
  active_account_id text,
  temp jsonb not null default '{}',
  updated_at timestamptz not null default now()
);
```

## Примечания по 1С

Backend ожидает типовой JSON-ответ:

```json
{
  "success": true,
  "code": "OK",
  "message": "Операция выполнена",
  "operation_id": "1c-20260506-000001",
  "actual_at": "2026-05-06T12:00:00+03:00",
  "data": {}
}
```

Для бизнес-ошибок 1С должна возвращать `success=false`, человекочитаемое `message` и код вроде `LINK_REQUIRED`, `READING_PERIOD_CLOSED`, `READING_LESS_THAN_PREVIOUS`.

## Важное ограничение

Это простой каркас для разработки и тестового стенда. Его нельзя считать промышленным решением без постоянного хранилища, очереди, TLS/reverse proxy, мониторинга, регламентов ИБ и полноценной реализации всех сценариев согласия, отзыва согласия, отвязки ЛС, квитанций и вложений.
