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
