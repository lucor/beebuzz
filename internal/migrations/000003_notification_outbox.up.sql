-- Short-lived notification outbox for Hive HTTPS recovery.
-- Ordering and pagination use the UUIDv7 id, which is time-sortable and carries
-- a millisecond timestamp; no separate sent_at column is needed.
CREATE TABLE IF NOT EXISTS notification_outbox (
    id            TEXT PRIMARY KEY,
    user_id       TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    topic_id      TEXT NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
    topic         TEXT NOT NULL,
    delivery_mode TEXT NOT NULL CHECK (delivery_mode IN ('server_trusted', 'e2e')),
    payload_json  TEXT NOT NULL,
    expires_at    INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS notification_outbox_recipients (
    notification_id TEXT NOT NULL REFERENCES notification_outbox(id) ON DELETE CASCADE,
    device_id       TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    created_at      INTEGER NOT NULL,
    PRIMARY KEY (notification_id, device_id)
);

CREATE INDEX IF NOT EXISTS idx_notification_outbox_expires_at ON notification_outbox(expires_at);
CREATE INDEX IF NOT EXISTS idx_notification_outbox_recipients_device_id ON notification_outbox_recipients(device_id);
