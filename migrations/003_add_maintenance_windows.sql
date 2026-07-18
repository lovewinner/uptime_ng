-- Add maintenance windows for notification/check suppression.

CREATE TABLE IF NOT EXISTS maintenance_windows (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    monitor_id INTEGER REFERENCES monitors(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description VARCHAR(500),
    start_at TIMESTAMP NOT NULL,
    end_at TIMESTAMP NOT NULL,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_maintenance_windows_user ON maintenance_windows(user_id);
CREATE INDEX IF NOT EXISTS idx_maintenance_windows_monitor ON maintenance_windows(monitor_id);
CREATE INDEX IF NOT EXISTS idx_maintenance_windows_time ON maintenance_windows(start_at, end_at);
