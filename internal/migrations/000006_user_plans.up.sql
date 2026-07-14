ALTER TABLE users ADD COLUMN plan TEXT NOT NULL DEFAULT 'free' CHECK (plan IN ('free', 'hosted'));
ALTER TABLE users ADD COLUMN plan_expires_at INTEGER;
