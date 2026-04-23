-- Rollback: Drop all tables in reverse dependency order

DROP TABLE IF EXISTS daily_usage_summary;
DROP TABLE IF EXISTS notification_events;
DROP TABLE IF EXISTS webhook_topics;
DROP TABLE IF EXISTS webhooks;
DROP TABLE IF EXISTS device_topics;
DROP TABLE IF EXISTS api_token_topics;
DROP TABLE IF EXISTS attachments;
DROP TABLE IF EXISTS topics;
DROP TABLE IF EXISTS push_subscriptions;
DROP TABLE IF EXISTS device_pairing_codes;
DROP TABLE IF EXISTS devices;
DROP TABLE IF EXISTS api_tokens;
DROP TABLE IF EXISTS auth_challenges;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
