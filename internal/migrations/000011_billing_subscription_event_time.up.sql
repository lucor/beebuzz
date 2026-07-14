ALTER TABLE billing_subscriptions
ADD COLUMN provider_event_at INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_billing_subscriptions_user_event_time
    ON billing_subscriptions(user_id, provider_event_at DESC);
