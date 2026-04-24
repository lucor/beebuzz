DROP TABLE IF EXISTS system_notification_settings;

CREATE TABLE notification_events_old (
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

INSERT INTO notification_events_old (
    id, user_id, event_type, device_id, topic, source, encryption_mode,
    attachment_bytes, fail_reason, created_at
)
SELECT
    id, user_id, event_type, device_id, topic,
    CASE source WHEN 'internal' THEN NULL ELSE source END,
    encryption_mode, attachment_bytes, fail_reason, created_at
FROM notification_events;

DROP TABLE notification_events;
ALTER TABLE notification_events_old RENAME TO notification_events;

CREATE INDEX idx_events_user_created_at ON notification_events (user_id, created_at);
CREATE INDEX idx_events_created_at ON notification_events (created_at);
