# Публикация 1С через Apache и подключение MAX-бота

Эта инструкция для системного администратора: как опубликовать HTTP-сервис 1С в Apache и как указать его в настройках бота.

## 1. Что уже есть в конфигурации 1С

В конфигурации есть HTTP-сервис `MAXBotHTTP` с корневым URL `max_bot`. Поэтому конечный URL HTTP-сервиса в Apache должен иметь вид:

```text
http://<apache-host>/<publication-name>/hs/max_bot/max/v1/reference/help
```

Где:

- `<apache-host>` — DNS-имя или IP сервера Apache;
- `<publication-name>` — имя публикации информационной базы 1С в Apache, например `billing`;
- `/hs/max_bot` — стандартный префикс HTTP-сервисов 1С + RootURL сервиса `MAXBotHTTP`;
- `/max/v1/reference/help` — один из URL-шаблонов сервиса.

Для бота `ONEC_BASE_URL` должен быть без хвоста `/max/v1/...`:

```dotenv
ONEC_BASE_URL=http://<apache-host>/<publication-name>/hs/max_bot
```

Например:

```dotenv
ONEC_BASE_URL=http://onec-server/billing/hs/max_bot
```

## 2. Что должен опубликовать администратор 1С/Apache

1. Установить компоненту веб-сервера 1С для Apache на сервере 1С/Apache.
2. Убедиться, что Apache запущен и слушает нужный порт, обычно `80` или `443`.
3. В конфигураторе 1С открыть информационную базу, где есть HTTP-сервис `MAXBotHTTP`.
4. Выполнить публикацию на веб-сервере:
   - меню **Администрирование → Публикация на веб-сервере**;
   - веб-сервер: Apache;
   - имя публикации: например `billing`;
   - включить публикацию HTTP-сервисов;
   - убедиться, что HTTP-сервис `MAXBotHTTP` опубликован.
5. После публикации перезапустить Apache, если мастер публикации или политика сервера этого требуют.
6. Проверить, что Apache отдает URL HTTP-сервиса 1С.

## 3. Токен авторизации 1С

Бот отправляет в 1С заголовок:

```http
Authorization: Bearer <ONEC_TOKEN>
```

В конфигурации 1С проверка выполняется через константу `MAXBotToken`. Поэтому значение `ONEC_TOKEN` в `.env.local` бота должно совпадать со значением константы `MAXBotToken` в 1С.

## 4. Проверка публикации с компьютера, где запущен бот

С Windows-ПК, где будет работать бот, проверьте доступ до Apache/1С:

```powershell
$Token = "<ONEC_TOKEN>"
Invoke-WebRequest `
  -Uri "http://<apache-host>/<publication-name>/hs/max_bot/max/v1/reference/help" `
  -Headers @{ Authorization = "Bearer $Token" }
```

Ожидаемый результат: HTTP `200` и JSON-ответ 1С. Если пришел `401`, проверьте `MAXBotToken` в 1С и `ONEC_TOKEN` в `.env.local`. Если URL не открывается, проверьте публикацию, Apache, firewall и сетевой маршрут от ПК с ботом до сервера 1С.

## 5. Настройка бота на Windows

В `.env.local` на Windows-ПК укажите:

```dotenv
HTTP_ADDR=0.0.0.0:8080
MAX_BASE_URL=https://platform-api.max.ru
MAX_TOKEN=<токен MAX-бота>
MAX_POLLING_ENABLED=true
MAX_POLLING_LIMIT=100
MAX_POLLING_TIMEOUT_SECONDS=30
MAX_POLLING_TYPES=message_created,message_callback

ONEC_BASE_URL=http://<apache-host>/<publication-name>/hs/max_bot
ONEC_TOKEN=<значение константы MAXBotToken в 1С>
INTERNAL_API_TOKEN=<токен для обратных вызовов 1С в бот>
DATABASE_URL=
```

Запуск:

```powershell
.\scripts\run-windows-bot.ps1 -OpenFirewall
```

Если не нужен входящий доступ от 1С к боту, флаг `-OpenFirewall` можно не указывать.

## 6. Обратный вызов из 1С в бота

Если 1С должна отправлять уведомления пользователю через MAX-бота, 1С должна вызвать локальный backend бота:

```text
POST http://<windows-pc-ip>:8080/internal/notifications/send
Authorization: Bearer <INTERNAL_API_TOKEN>
Content-Type: application/json
```

Тело запроса:

```json
{
  "chat_id": 987654321,
  "text": "Начисление обновлено",
  "operation_id": "onec-001"
}
```

Проверка с сервера 1С/Apache до Windows-ПК:

```powershell
Invoke-RestMethod http://<windows-pc-ip>:8080/healthz
```

Если не открывается, проверьте `HTTP_ADDR=0.0.0.0:8080`, firewall Windows и доступность ПК из сети сервера 1С.

## 7. Итоговая схема

```text
Пользователь в MAX
   ↓
MAX Bot API
   ↓ long polling
max-bot на Windows-ПК
   ↓ HTTP Authorization: Bearer ONEC_TOKEN
Apache публикация 1С: /<publication-name>/hs/max_bot/max/v1/...
   ↓
HTTP-сервис 1С MAXBotHTTP

1С → http://<windows-pc-ip>:8080/internal/notifications/send → max-bot → MAX Bot API → пользователь
```
