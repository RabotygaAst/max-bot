-- Таблица для отслеживания обработанных событий (идемпотентность)
CREATE TABLE IF NOT EXISTS max_events (
    event_id TEXT PRIMARY KEY,
    status TEXT NOT NULL DEFAULT 'processing',
    operation_id TEXT,
    error_text TEXT,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_max_events_status ON max_events(status);
CREATE INDEX IF NOT EXISTS idx_max_events_processed_at ON max_events(processed_at);

-- Таблица для сессий диалога
CREATE TABLE IF NOT EXISTS dialog_sessions (
    max_user_id BIGINT PRIMARY KEY,
    step TEXT,
    active_account_id TEXT,
    temp JSONB NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dialog_sessions_updated_at ON dialog_sessions(updated_at);

-- Таблица для логирования (опционально, для аудита)
CREATE TABLE IF NOT EXISTS event_logs (
    id SERIAL PRIMARY KEY,
    event_id TEXT NOT NULL,
    max_user_id BIGINT,
    action TEXT,
    details JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_event_logs_event_id ON event_logs(event_id);
CREATE INDEX IF NOT EXISTS idx_event_logs_max_user_id ON event_logs(max_user_id);
CREATE INDEX IF NOT EXISTS idx_event_logs_created_at ON event_logs(created_at);

-- Привязанные лицевые счета пользователей MAX
CREATE TABLE IF NOT EXISTS linked_accounts (
    max_user_id BIGINT NOT NULL,
    account_id TEXT NOT NULL,
    account_number TEXT NOT NULL,
    source TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (max_user_id, account_id)
);

CREATE INDEX IF NOT EXISTS idx_linked_accounts_account_number ON linked_accounts(account_number);

-- Все переданные через бота показания, привязанные к конкретному ЛС
CREATE TABLE IF NOT EXISTS meter_readings (
    id BIGSERIAL PRIMARY KEY,
    max_user_id BIGINT NOT NULL,
    account_id TEXT NOT NULL,
    account_number TEXT NOT NULL,
    meter_id TEXT NOT NULL,
    period TEXT NOT NULL,
    value NUMERIC(14, 3) NOT NULL,
    operation_id TEXT,
    message_id TEXT,
    source TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_meter_readings_account_id ON meter_readings(account_id);
CREATE INDEX IF NOT EXISTS idx_meter_readings_account_number ON meter_readings(account_number);
CREATE INDEX IF NOT EXISTS idx_meter_readings_max_user_id ON meter_readings(max_user_id);

-- Все обращения через бота, привязанные к конкретному ЛС
CREATE TABLE IF NOT EXISTS appeals (
    id BIGSERIAL PRIMARY KEY,
    max_user_id BIGINT NOT NULL,
    account_id TEXT NOT NULL,
    account_number TEXT NOT NULL,
    appeal_id TEXT,
    appeal_number TEXT,
    text TEXT NOT NULL,
    operation_id TEXT,
    message_id TEXT,
    source TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_appeals_account_id ON appeals(account_id);
CREATE INDEX IF NOT EXISTS idx_appeals_account_number ON appeals(account_number);
CREATE INDEX IF NOT EXISTS idx_appeals_max_user_id ON appeals(max_user_id);
