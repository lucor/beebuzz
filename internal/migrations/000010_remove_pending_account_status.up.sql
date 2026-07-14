-- MUST be run with NoTxWrap=true (set in migrations.go). The driver does NOT
-- wrap this migration in a transaction, so we control PRAGMA and transactions
-- explicitly. This is required because DROP TABLE on a parent table with FK
-- references from child tables needs PRAGMA foreign_keys = OFF, which is a
-- no-op when set inside a transaction.

PRAGMA foreign_keys = OFF;

BEGIN;

CREATE TABLE users_new (
    id          TEXT PRIMARY KEY,
    email       TEXT NOT NULL,
    is_admin    INTEGER NOT NULL DEFAULT 0 CHECK (is_admin IN (0, 1)),
    account_status TEXT NOT NULL DEFAULT 'active'
        CHECK (account_status IN ('active', 'blocked')),
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL,
    plan TEXT NOT NULL DEFAULT 'free' CHECK (plan IN ('free', 'hosted')),
    plan_expires_at INTEGER,
    CONSTRAINT users_email_unique UNIQUE (email)
);

INSERT INTO users_new (
    id,
    email,
    is_admin,
    account_status,
    created_at,
    updated_at,
    plan,
    plan_expires_at
)
SELECT
    id,
    email,
    is_admin,
    account_status,
    created_at,
    updated_at,
    plan,
    plan_expires_at
FROM users;

DROP TABLE users;
ALTER TABLE users_new RENAME TO users;

CREATE INDEX idx_users_created_at ON users (created_at);
CREATE INDEX idx_users_account_status ON users(account_status);

COMMIT;

PRAGMA foreign_keys = ON;
