CREATE TABLE IF NOT EXISTS profiles (
                                        user_id     TEXT PRIMARY KEY,
                                        username    VARCHAR(32) NOT NULL,
    bio         TEXT NOT NULL DEFAULT '',
    avatar_url  TEXT NOT NULL DEFAULT '',
    birth_date  VARCHAR(10) NOT NULL DEFAULT '',
    level       INT NOT NULL DEFAULT 1,
    reputation  INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL
    );

CREATE UNIQUE INDEX IF NOT EXISTS idx_profiles_username_lower
    ON profiles (LOWER(username));