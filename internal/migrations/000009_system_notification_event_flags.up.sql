ALTER TABLE system_notification_settings
ADD COLUMN event_flags INTEGER NOT NULL DEFAULT 0 CHECK (event_flags >= 0);

UPDATE system_notification_settings
SET event_flags = CASE signup_created_enabled
    WHEN 1 THEN 1
    ELSE 0
END;

ALTER TABLE system_notification_settings DROP COLUMN signup_created_enabled;
