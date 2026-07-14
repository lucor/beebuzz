DROP INDEX IF EXISTS idx_billing_subscriptions_user_event_time;

ALTER TABLE billing_subscriptions
DROP COLUMN provider_event_at;
