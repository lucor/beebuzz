CREATE TABLE IF NOT EXISTS debug_reports (
    report_id   TEXT PRIMARY KEY,
    device_id   TEXT NOT NULL,
    created_at  INTEGER NOT NULL,
    payload_json TEXT NOT NULL,

    FOREIGN KEY (device_id) REFERENCES devices(id)
);

CREATE INDEX IF NOT EXISTS idx_debug_reports_device_id ON debug_reports(device_id);
CREATE INDEX IF NOT EXISTS idx_debug_reports_created_at ON debug_reports(created_at);
