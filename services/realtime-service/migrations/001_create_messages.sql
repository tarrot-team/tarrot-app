CREATE TABLE IF NOT EXISTS chat_messages (
    id           TEXT PRIMARY KEY,
    room         VARCHAR(64) NOT NULL,
    from_user_id TEXT NOT NULL,
    text         TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_chat_messages_room_created
    ON chat_messages (room, created_at DESC);
