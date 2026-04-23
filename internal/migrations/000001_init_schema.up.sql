-- BeeBuzz Complete Schema Setup

-- ========== USERS AND AUTHENTICATION ==========

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id          TEXT PRIMARY KEY,
    email       TEXT NOT NULL,
    is_admin    INTEGER NOT NULL DEFAULT 0 CHECK (is_admin IN (0, 1)),
    account_status TEXT NOT NULL DEFAULT 'active'
        CHECK (account_status IN ('pending', 'active', 'blocked')),
    trial_started_at INTEGER,
    signup_reason TEXT,
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL,

    CONSTRAINT users_email_unique UNIQUE (email)
);

CREATE INDEX idx_users_created_at ON users (created_at);
CREATE INDEX idx_users_account_status ON users(account_status);

-- Auth challenges for login/signup
CREATE TABLE IF NOT EXISTS auth_challenges (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	state TEXT NOT NULL,
	otp_hash TEXT NOT NULL UNIQUE,
	expires_at INTEGER NOT NULL,
	used_at INTEGER,
	attempt_count INTEGER NOT NULL DEFAULT 0,
	created_at INTEGER NOT NULL,
	FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_auth_challenges_user_id ON auth_challenges(user_id);
CREATE INDEX IF NOT EXISTS idx_auth_challenges_state ON auth_challenges(state);
CREATE INDEX IF NOT EXISTS idx_auth_challenges_expires_at ON auth_challenges(expires_at);

-- User sessions for authentication
CREATE TABLE IF NOT EXISTS sessions (
    token_hash TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    last_seen_at INTEGER NOT NULL,

    CONSTRAINT sessions_user_id_fk
        FOREIGN KEY (user_id) REFERENCES users (id)
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id    ON sessions (user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions (expires_at);

-- ========== API TOKENS ==========

-- API tokens for programmatic access
CREATE TABLE IF NOT EXISTS api_tokens (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id),
	token_hash TEXT NOT NULL UNIQUE,
	name TEXT DEFAULT '',
	description TEXT,
	is_active INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
	last_used_at INTEGER,
	expires_at INTEGER,
	revoked_at INTEGER,
	created_at INTEGER NOT NULL
);

-- ========== DEVICES ==========

-- Devices registered by users
CREATE TABLE IF NOT EXISTS devices (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    is_active   INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
    pairing_status TEXT NOT NULL DEFAULT 'pending'
        CHECK (pairing_status IN ('pending', 'paired', 'unpaired', 'subscription_gone')),
    device_token_hash TEXT,
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);

CREATE INDEX idx_devices_user_id ON devices(user_id);

-- Device pairing codes for device registration
CREATE TABLE IF NOT EXISTS device_pairing_codes (
    code_hash   TEXT PRIMARY KEY,
    device_id   TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    expires_at  INTEGER NOT NULL,
    used_at     INTEGER,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    created_at  INTEGER NOT NULL
);

CREATE INDEX idx_device_pairing_codes_device_id ON device_pairing_codes(device_id);
CREATE INDEX idx_device_pairing_codes_expires_at ON device_pairing_codes(expires_at);

-- Push subscriptions (Web Push Protocol details)
CREATE TABLE IF NOT EXISTS push_subscriptions (
    device_id      TEXT PRIMARY KEY REFERENCES devices(id) ON DELETE CASCADE,
    endpoint       TEXT NOT NULL UNIQUE,
    p256dh         TEXT NOT NULL,
    auth           TEXT NOT NULL,
    age_recipient  TEXT NOT NULL,
    created_at     INTEGER NOT NULL,
    updated_at     INTEGER NOT NULL
);

-- ========== TOPICS ==========

-- Topics for organizing notifications (PER-USER)
CREATE TABLE IF NOT EXISTS topics (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	description TEXT,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL,
	UNIQUE(user_id, name)
);

-- Many-to-many: API tokens to topics
CREATE TABLE IF NOT EXISTS api_token_topics (
	api_token_id TEXT NOT NULL REFERENCES api_tokens(id) ON DELETE CASCADE,
	topic_id TEXT NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
	created_at INTEGER NOT NULL,
	PRIMARY KEY (api_token_id, topic_id)
);

-- Many-to-many: Devices to topics
CREATE TABLE IF NOT EXISTS device_topics (
    device_id   TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    topic_id    TEXT NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
    created_at  INTEGER NOT NULL,
    PRIMARY KEY (device_id, topic_id)
);

-- ========== WEBHOOKS ==========

-- Webhooks for JSON payload notifications
CREATE TABLE IF NOT EXISTS webhooks (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	token_hash TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL,
	description TEXT,
	title_path TEXT NOT NULL,
	body_path TEXT NOT NULL,
	payload_type TEXT NOT NULL DEFAULT 'beebuzz' CHECK(payload_type IN ('beebuzz', 'custom')),
	priority TEXT NOT NULL DEFAULT 'normal' CHECK (priority IN ('normal', 'high')),
	is_active INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
	revoked_at INTEGER,
	last_used_at INTEGER,
	created_at INTEGER NOT NULL
);

-- Many-to-many: Webhooks to topics
CREATE TABLE IF NOT EXISTS webhook_topics (
	webhook_id TEXT NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
	topic_id TEXT NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
	created_at INTEGER NOT NULL,
	PRIMARY KEY (webhook_id, topic_id)
);

-- ========== ATTACHMENTS ==========

-- BeeBuzz-hosted attachments
CREATE TABLE IF NOT EXISTS attachments (
	id TEXT PRIMARY KEY,
	token TEXT NOT NULL UNIQUE,
	topic_id TEXT NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
	mime_type TEXT NOT NULL,
	file_size_bytes INTEGER NOT NULL,
	created_at INTEGER NOT NULL,
	expires_at INTEGER NOT NULL
);

-- ========== INDEXES ==========

-- API token indexes
CREATE INDEX IF NOT EXISTS idx_api_tokens_user_id ON api_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_api_tokens_expires_at ON api_tokens(expires_at);

-- Topic indexes (reverse lookup on junction tables)
CREATE INDEX IF NOT EXISTS idx_api_token_topics_topic_id ON api_token_topics(topic_id);
CREATE INDEX IF NOT EXISTS idx_device_topics_topic_id ON device_topics(topic_id);

-- Webhook indexes
CREATE INDEX IF NOT EXISTS idx_webhooks_user_id ON webhooks(user_id);
CREATE INDEX IF NOT EXISTS idx_webhook_topics_topic_id ON webhook_topics(topic_id);

-- Attachment indexes
CREATE INDEX IF NOT EXISTS idx_attachments_topic_id ON attachments(topic_id);
CREATE INDEX IF NOT EXISTS idx_attachments_expires_at ON attachments(expires_at);

-- ========== ANALYTICS ========== 

-- Notification event tracking for analytics and daily usage rollups.
CREATE TABLE IF NOT EXISTS notification_events (
    id               TEXT PRIMARY KEY,
    user_id          TEXT NOT NULL REFERENCES users(id),
    event_type       TEXT NOT NULL CHECK (event_type IN (
        'notification_created',
        'notification_delivered',
        'notification_failed'
    )),
    device_id        TEXT,
    topic            TEXT,
    source           TEXT CHECK (source IS NULL OR source IN ('cli', 'webhook', 'api')),
    encryption_mode  TEXT CHECK (encryption_mode IS NULL OR encryption_mode IN ('server_trusted', 'e2e')),
    attachment_bytes INTEGER CHECK (attachment_bytes IS NULL OR attachment_bytes >= 0),
    fail_reason      TEXT,
    created_at       INTEGER NOT NULL
);

CREATE INDEX idx_events_user_created_at ON notification_events (user_id, created_at);
CREATE INDEX idx_events_created_at ON notification_events (created_at);

CREATE TABLE IF NOT EXISTS daily_usage_summary (
    user_id                       TEXT NOT NULL REFERENCES users(id),
    day_start_ms                  INTEGER NOT NULL,
    notifications_total           INTEGER NOT NULL DEFAULT 0,
    notifications_delivered       INTEGER NOT NULL DEFAULT 0,
    notifications_failed          INTEGER NOT NULL DEFAULT 0,
    notifications_server_trusted  INTEGER NOT NULL DEFAULT 0,
    notifications_e2e             INTEGER NOT NULL DEFAULT 0,
    attachments_count             INTEGER NOT NULL DEFAULT 0,
    attachments_bytes_total       INTEGER NOT NULL DEFAULT 0,
    sources_cli                   INTEGER NOT NULL DEFAULT 0,
    sources_webhook               INTEGER NOT NULL DEFAULT 0,
    sources_api                   INTEGER NOT NULL DEFAULT 0,
    devices_lost                  INTEGER NOT NULL DEFAULT 0,
    created_at                    INTEGER NOT NULL,
    updated_at                    INTEGER NOT NULL,
    PRIMARY KEY (user_id, day_start_ms)
);

CREATE INDEX idx_daily_usage_summary_day_start_ms ON daily_usage_summary (day_start_ms);
