# ⚡ Quick Start (5 минут)

## Если вы спешите — вот самые важные команды:

### 1️⃣ Подготовка
```bash
cd /home/dmitrymakov/max-bot
cp .env.example .env
# Сгенерируйте WEBHOOK_SECRET и скопируйте результат в .env
go run ./cmd/bot generate-webhook-secret
# Отредактируйте .env и вставьте ваш MAX_TOKEN, WEBHOOK_SECRET и остальные значения
nano .env
```

### 2️⃣ Запуск (Docker)
```bash
docker-compose up -d
docker-compose logs -f max-bot
```

Дождитесь строки:
```
using PostgreSQL store
```

### 3️⃣ Expose в интернет (для реальной MAX)
```bash
ngrok http 8080
# Скопируйте URL вроде: https://xxx-yyy-zzz.ngrok.io
```

### 4️⃣ Регистрируем webhook в MAX
В кабинете MAX зарегистрируйте:
- **Endpoint:** `https://xxx-yyy-zzz.ngrok.io/webhook/max`
- **Secret:** значение `WEBHOOK_SECRET` из `.env`, сгенерированное командой `go run ./cmd/bot generate-webhook-secret`

### 5️⃣ Тестируем
```bash
# Health check
curl http://localhost:8080/healthz

# Имитируем webhook (подставьте сгенерированный WEBHOOK_SECRET из .env)
curl -X POST http://localhost:8080/webhook/max \
  -H "Content-Type: application/json" \
  -H "X-Max-Webhook-Secret: <значение WEBHOOK_SECRET из .env>" \
  -d '{"update_type":"message_created","timestamp":1778068800000,"message":{"sender":{"user_id":123456789,"first_name":"Тест"},"recipient":{"chat_id":987654321},"body":{"mid":"test.1","text":"/start"}}}'
```

### 5.1 Локальная проверка без webhook
```bash
curl -X POST http://localhost:8080/debug/send-test-update \
  -H "Content-Type: application/json" \
  -d '{"user_id":123456789,"chat_id":987654321,"text":"/start"}'
```

Этот endpoint работает локально и позволяет проверить обработку команд без регистрации webhook.

### 6️⃣ Пишем боту в MAX 💬
Откройте чат с ботом в MAX и напишите: `/start`

---

## 🔧 Для разработки (если нужно дебажить локально без Docker)

```bash
# 1. Установите зависимости
go mod download

# 2. Запустите PostgreSQL отдельно (или используйте docker-compose postgres только)
docker run --name max-bot-pg -e POSTGRES_DB=maxbot -e POSTGRES_USER=maxbot \
  -e POSTGRES_PASSWORD=maxbot_password -p 5432:5432 postgres:16-alpine

# 3. Установите переменные
export $(grep -v '^#' .env | xargs)

# 4. Запустите бота
go run ./cmd/bot
```

---

## ⚙️ Что было изменено:

✅ Добавлена поддержка **PostgreSQL** Store  
✅ Обновлен `docker-compose.yml` с PostgreSQL и Mock 1С  
✅ Создан `init-db.sql` для инициализации схемы  
✅ Создана `postgres.go` реализация Store  
✅ Обновлен `main.go` для работы с БД  
✅ `.env.example` обновлен с DATABASE_URL  
✅ Добавлена зависимость `github.com/lib/pq`  

---

**→ Полная инструкция:** см. [SETUP_GUIDE.md](SETUP_GUIDE.md)
