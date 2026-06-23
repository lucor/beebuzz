ALTER TABLE webhooks ADD COLUMN title_source TEXT NOT NULL DEFAULT 'path' CHECK(title_source IN ('path', 'static'));
ALTER TABLE webhooks ADD COLUMN title_value TEXT NOT NULL DEFAULT '';
