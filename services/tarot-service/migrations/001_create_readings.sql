CREATE TABLE IF NOT EXISTS readings (
    id             TEXT PRIMARY KEY,
    user_id        TEXT NOT NULL,
    mode           VARCHAR(16) NOT NULL,
    spread_type    VARCHAR(32) NOT NULL,
    spread_title   TEXT NOT NULL,
    question       TEXT NOT NULL DEFAULT '',
    cards_json     JSONB NOT NULL,
    interpretation TEXT NOT NULL,
    source         VARCHAR(16) NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_readings_user_created
    ON readings (user_id, created_at DESC);
