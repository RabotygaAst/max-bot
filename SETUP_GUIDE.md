# 🚀 Полная инструкция: Запуск MAX-бота с Docker, PostgreSQL и Mock 1С API

## Требования

- **Docker** (версия 20.10+) и **docker-compose** (версия 2.0+)
- **ngrok** для expose public URL (если тестируете с реальной MAX API)
- **Git** для управления проектом
- **curl** для тестирования (опционально)

---

## 📦 Шаг 1: Подготовка окружения

### 1.1 Клонируем/готовим проект

```bash
cd /home/dmitrymakov/max-bot
```

### 1.2 Создаем .env файл с реальными credentials

```bash
cp .env.example .env
```

Отредактируйте `.env` и заполните реальные значения:

```bash
cat > .env << 'EOF'
HTTP_ADDR=:8080
REQUEST_TIMEOUT_SECONDS=10

# ✅ Вставьте РЕАЛЬНЫЙ MAX_TOKEN и WEBHOOK_SECRET
MAX_BASE_URL=https://platform-api.max.ru
MAX_TOKEN=f9LHodD0cOIeKW0ZzgnrNXPpGdMltDDemKfWXljiTahRofJYZcDpwhfMunwJNnKBkytsqlIVUSd5XDBLw2FK
WEBHOOK_SECRET=YOUR_REAL_WEBHOOK_SECRET_HERE_2026
WEBHOOK_SECRET_HEADER=X-Max-Webhook-Secret

# Mock 1С API (уже в docker-compose)
ONEC_BASE_URL=http://mock-onec:1080
ONEC_TOKEN=MOCK_ONEC_TOKEN

INTERNAL_API_TOKEN=YOUR_INTERNAL_API_TOKEN_2026

# PostgreSQL (автоматически в docker-compose)
DATABASE_URL=postgres://maxbot:maxbot_password@postgres:5432/maxbot?sslmode=disable
EOF
```

---

## 🐳 Шаг 2: Запуск контейнеров

### 2.1 Стартуем Docker Compose

```bash
docker-compose up -d
```

Это запустит:
- **PostgreSQL** на `localhost:5432`
- **Mock 1С API** (mockserver) на `localhost:1080`
- **MAX-бот** на `localhost:8080`

### 2.2 Проверяем статус контейнеров

```bash
docker-compose ps
```

Вывод должен быть примерно такой:
```
NAME                   STATUS
max-bot-postgres       Up (healthy)
max-bot-mock-onec      Up
max-bot                Up
```

### 2.3 Проверяем логи бота

```bash
docker-compose logs -f max-bot
```

Ищем в логах строку:
```
using PostgreSQL store
```

---

## 🌐 Шаг 3: Expose бота в интернет (для реальной MAX API)

Так как MAX отправляет webhook запросы на ваш backend, нужно expose `localhost:8080` в интернет.

### 3.1 Устанавливаем ngrok

**На Linux/macOS:**
```bash
brew install ngrok
```

Или скачиваем с https://ngrok.com/download

### 3.2 Логинимся в ngrok (если нужно)

```bash
ngrok config add-authtoken YOUR_NGROK_TOKEN
```

### 3.3 Запускаем ngrok tunnel

```bash
ngrok http 8080
```

Вывод:
```
Forwarding                    https://xxx-yyy-zzz.ngrok.io -> http://localhost:8080
```

**Скопируйте URL:** `https://xxx-yyy-zzz.ngrok.io`

---

## ⚙️ Шаг 4: Регистрируем Webhook в MAX

Теперь в кабинете MAX (или через MAX Bot API) регистрируем webhook:

**Endpoint:** `https://xxx-yyy-zzz.ngrok.io/webhook/max` (замените на ваш ngrok URL)
**Secret:** Значение из `WEBHOOK_SECRET` в `.env`

Пример curl-запроса для регистрации webhook (если MAX API это поддерживает):

```bash
curl -X POST https://platform-api.max.ru/max/v1/webhooks \
  -H "Authorization: Bearer YOUR_MAX_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://xxx-yyy-zzz.ngrok.io/webhook/max",
    "secret": "YOUR_WEBHOOK_SECRET_HERE"
  }'
```

---

## 🧪 Шаг 5: Тестирование бота

### 5.1 Проверяем health-check

```bash
curl http://localhost:8080/healthz
```

Ответ:
```json
{"ok": true}
```

### 5.2 Тестируем webhook (имитируя MAX)

```bash
curl -X POST http://localhost:8080/webhook/max \
  -H "Content-Type: application/json" \
  -H "X-Max-Webhook-Secret: YOUR_WEBHOOK_SECRET_HERE_2026" \
  -d '{
    "update_type": "message_created",
    "timestamp": 1778068800000,
    "message": {
      "sender": {
        "user_id": 123456789,
        "first_name": "Иван"
      },
      "recipient": {
        "chat_id": 987654321
      },
      "body": {
        "mid": "mid.example.1",
        "text": "/start"
      }
    }
  }'
```

Ожидаемый ответ:
```json
{"success": true}
```

### 5.2.1 Локальная проверка без webhook

Если вы хотите проверить обработку без зарегистрированного webhook, используйте новый debug endpoint:

```bash
curl -X POST http://localhost:8080/debug/send-test-update \
  -H "Content-Type: application/json" \
  -d '{"user_id":123456789,"chat_id":987654321,"text":"/start"}'
```

Это удобно для локального тестирования и не требует секретного заголовка.

### 5.3 Проверяем PostgreSQL

```bash
docker-compose exec postgres psql -U maxbot -d maxbot -c "SELECT * FROM max_events LIMIT 5;"
```

Должны увидеть обработанные события.

### 5.4 Тестируем Mock 1С API

```bash
curl -X GET http://localhost:1080/max/v1/accounts \
  -H "Authorization: Bearer MOCK_ONEC_TOKEN"
```

Ответ (mock-ответ):
```json
{
  "success": true,
  "code": "OK",
  "data": {
    "accounts": [...]
  }
}
```

---

## 💬 Шаг 6: Отправляем сообщение в MAX (реальный тест)

Когда все настроено и webhook зарегистрирован в MAX:

1. Откройте чат с ботом в MAX (найдите вашего MAX бота в каталоге)
2. Напишите команду: `/start`
3. Бот должен ответить вам через MAX API

**Если ошибка:**
- Проверьте логи: `docker-compose logs max-bot | tail -50`
- Проверьте `MAX_TOKEN` и `WEBHOOK_SECRET` в `.env`
- Убедитесь, что ngrok URL активен и соответствует зарегистрированному webhook

---

## 🔍 Диагностика и отладка

### Проверяем логи PostgreSQL

```bash
docker-compose logs postgres | grep ERROR
```

### Проверяем логи бота

```bash
docker-compose logs max-bot | grep -E "error|failed|ERROR"
```

### Входим в контейнер бота

```bash
docker-compose exec max-bot sh
```

### Проверяем подключение к БД

```bash
docker-compose exec postgres psql -U maxbot -d maxbot

# Внутри psql:
\d  -- показывает все таблицы
SELECT * FROM max_events;  -- проверяет обработанные события
SELECT * FROM dialog_sessions;  -- проверяет сессии диалогов
\q  -- выход
```

### Перезагружаем контейнеры

```bash
docker-compose restart
```

### Очищаем все контейнеры и данные

```bash
docker-compose down -v  # удалит все, включая БД!
docker-compose up -d  -- пересоздаст с нуля
```

---

## 📝 Шаг 7: Логирование и мониторинг

Все логи пишутся в JSON-формате (через slog).

### Смотрим логи в реальном времени:

```bash
docker-compose logs -f max-bot
```

### Фильтруем логи:

```bash
docker-compose logs max-bot | grep "process event failed"
```

---

## 🚫 Важные ограничения и TODOs

Текущая реализация с Mock 1С API — это **прототип**.

Для production нужно:

1. **Заменить mock на реальный 1С HTTP API endpoint**
   - Обновите `ONEC_BASE_URL` в `.env`
   - Убедитесь, что 1С API возвращает JSON в нужном формате

2. **Добавить retries и exponential backoff**
   - Для случаев, когда MAX или 1С недоступны

3. **Настроить TLS/HTTPS**
   - Добавить самоподписанный сертификат или использовать Let's Encrypt
   - Настроить reverse proxy (nginx)

4. **Добавить метрики Prometheus**
   - Количество обработанных событий
   - Ошибки 1С и MAX
   - Время обработки

5. **Настроить persistent volume для PostgreSQL**
   - Сейчас данные хранятся в `postgres_data/`, это работает, но убедитесь в backup'ах

6. **Добавить очередь обработки событий**
   - Вместо goroutine использовать RabbitMQ, Redis или PostgreSQL outbox pattern

---

## ✅ Финальный чек-лист

- [ ] Docker и docker-compose установлены
- [ ] `.env` файл заполнен реальными credentials
- [ ] `docker-compose up -d` запущен и все контейнеры healthy
- [ ] Webhook зарегистрирован в MAX
- [ ] ngrok tunnel активен (если используется реальная MAX API)
- [ ] `curl http://localhost:8080/healthz` возвращает 200 OK
- [ ] Тестовое сообщение в MAX отправило webhook
- [ ] Логи показывают обработку события

---

## 🆘 Если что-то не работает

1. **Проверьте все .env переменные**
   ```bash
   docker-compose exec max-bot env | grep -E "MAX_|ONEC_|DATABASE_"
   ```

2. **Проверьте сетевое подключение между контейнерами**
   ```bash
   docker-compose exec max-bot ping postgres
   docker-compose exec max-bot ping mock-onec
   ```

3. **Проверьте, что PostgreSQL инициализирована**
   ```bash
   docker-compose logs postgres | grep "CREATE TABLE"
   ```

4. **Выполните полный рестарт**
   ```bash
   docker-compose down -v
   docker-compose up -d
   docker-compose logs -f max-bot
   ```

---

**Успехов в разработке! 🎉**
