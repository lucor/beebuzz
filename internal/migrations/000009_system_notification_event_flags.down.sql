ALTER TABLE system_notification_settings
ADD COLUMN signup_created_enabled INTEGER NOT NULL DEFAULT 0 CHECK (signup_created_enabled IN (0, 1));

UPDATE system_notification_settings
SET signup_created_enabled = CASE
    WHEN (event_flags & 1) != 0 THEN 1
    ELSE 0
END;

ALTER TABLE system_notification_settings DROP COLUMN event_flags;
