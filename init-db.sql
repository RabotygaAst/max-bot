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

CREATE TABLE IF NOT EXISTS max_users (
    max_user_id BIGINT PRIMARY KEY,
    chat_id BIGINT NOT NULL,
    first_name TEXT,
    source TEXT NOT NULL DEFAULT 'MAX',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS account_links (
    id BIGSERIAL PRIMARY KEY,
    max_user_id BIGINT NOT NULL REFERENCES max_users(max_user_id) ON DELETE CASCADE,
    account_id TEXT NOT NULL,
    account_number TEXT NOT NULL,
    masked_address TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    linked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(max_user_id, account_id)
);
CREATE INDEX IF NOT EXISTS idx_account_links_user_active ON account_links(max_user_id, is_active);
CREATE TABLE IF NOT EXISTS billing_accruals_cache (id BIGSERIAL PRIMARY KEY,max_user_id BIGINT NOT NULL,account_id TEXT NOT NULL,period TEXT NOT NULL,amount NUMERIC(14,2) NOT NULL DEFAULT 0,debt NUMERIC(14,2) NOT NULL DEFAULT 0,overpay NUMERIC(14,2) NOT NULL DEFAULT 0,currency TEXT NOT NULL DEFAULT 'руб.',actual_at TEXT,raw JSONB NOT NULL DEFAULT '{}',updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),UNIQUE(account_id, period));
CREATE TABLE IF NOT EXISTS invoices_cache (id BIGSERIAL PRIMARY KEY,max_user_id BIGINT NOT NULL,account_id TEXT NOT NULL,period TEXT NOT NULL,amount NUMERIC(14,2) NOT NULL DEFAULT 0,currency TEXT NOT NULL DEFAULT 'руб.',document_number TEXT,document_date TEXT,download_url TEXT,raw JSONB NOT NULL DEFAULT '{}',updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),UNIQUE(account_id, period));
CREATE TABLE IF NOT EXISTS payments_cache (id BIGSERIAL PRIMARY KEY,max_user_id BIGINT NOT NULL,account_id TEXT NOT NULL,amount NUMERIC(14,2) NOT NULL DEFAULT 0,currency TEXT NOT NULL DEFAULT 'руб.',payment_url TEXT,status TEXT NOT NULL DEFAULT 'created',operation_id TEXT,expires_at TEXT,raw JSONB NOT NULL DEFAULT '{}',created_at TIMESTAMPTZ NOT NULL DEFAULT NOW());
CREATE TABLE IF NOT EXISTS appointments (id BIGSERIAL PRIMARY KEY,max_user_id BIGINT NOT NULL,account_id TEXT NOT NULL,appointment_id TEXT,number TEXT,topic_id TEXT,topic_title TEXT,office_address TEXT,starts_at TEXT,status TEXT,raw JSONB NOT NULL DEFAULT '{}',created_at TIMESTAMPTZ NOT NULL DEFAULT NOW());
CREATE TABLE IF NOT EXISTS appeals_cache (id BIGSERIAL PRIMARY KEY,max_user_id BIGINT NOT NULL,account_id TEXT NOT NULL,appeal_id TEXT,number TEXT,category TEXT,text TEXT,status TEXT,sla TEXT,raw JSONB NOT NULL DEFAULT '{}',created_at TIMESTAMPTZ NOT NULL DEFAULT NOW());
CREATE TABLE IF NOT EXISTS notification_logs (id BIGSERIAL PRIMARY KEY,chat_id BIGINT NOT NULL,max_user_id BIGINT,account_id TEXT,type TEXT,text TEXT NOT NULL,operation_id TEXT,sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW());
