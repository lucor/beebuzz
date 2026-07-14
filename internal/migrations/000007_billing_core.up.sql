-- Provider-neutral billing foundations.

CREATE TABLE IF NOT EXISTS billing_customers (
    id                   TEXT PRIMARY KEY,
    user_id              TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider             TEXT NOT NULL,
    provider_customer_id TEXT,
    created_at           INTEGER NOT NULL,
    updated_at           INTEGER NOT NULL,
    UNIQUE(user_id, provider)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_billing_customers_provider_customer
    ON billing_customers(provider, provider_customer_id)
    WHERE provider_customer_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS billing_subscriptions (
    id                       TEXT PRIMARY KEY,
    user_id                  TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    customer_id              TEXT REFERENCES billing_customers(id) ON DELETE SET NULL,
    provider                 TEXT NOT NULL,
    provider_subscription_id TEXT NOT NULL,
    plan                     TEXT NOT NULL CHECK (plan IN ('hosted')),
    status                   TEXT NOT NULL CHECK (status IN (
        'incomplete',
        'active',
        'scheduled_cancel',
        'past_due',
        'canceled',
        'expired'
    )),
    current_period_end       INTEGER,
    cancel_at_period_end     INTEGER NOT NULL DEFAULT 0 CHECK (cancel_at_period_end IN (0, 1)),
    created_at               INTEGER NOT NULL,
    updated_at               INTEGER NOT NULL,
    UNIQUE(provider, provider_subscription_id)
);

CREATE INDEX IF NOT EXISTS idx_billing_subscriptions_user_id
    ON billing_subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_billing_subscriptions_status
    ON billing_subscriptions(status);

CREATE TABLE IF NOT EXISTS billing_events (
    id                  TEXT PRIMARY KEY,
    provider            TEXT NOT NULL,
    provider_event_id   TEXT NOT NULL,
    event_type          TEXT NOT NULL,
    payload_sha256      TEXT NOT NULL,
    processed_at        INTEGER NOT NULL,
    created_at          INTEGER NOT NULL,
    UNIQUE(provider, provider_event_id)
);
