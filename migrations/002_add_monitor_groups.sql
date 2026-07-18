-- Add recursive monitor grouping support.

ALTER TABLE monitors
    ADD COLUMN IF NOT EXISTS group_id INTEGER REFERENCES monitors(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_monitors_group ON monitors(group_id);

ALTER TABLE monitors
    DROP CONSTRAINT IF EXISTS monitors_type_check;

ALTER TABLE monitors
    ADD CONSTRAINT monitors_type_check CHECK (type IN ('http', 'tcp', 'ping', 'dns', 'push', 'group'));
