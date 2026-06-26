-- 0005_leveling: per-guild XP and levels.
--
-- Members earn XP for chatting (rate-limited by a per-user cooldown enforced in
-- the cache, not here). Cumulative XP maps to a level via a fixed curve in code.
-- Optional level-reward roles are granted on level-up. IDs are BIGINT snowflakes.

CREATE TABLE IF NOT EXISTS leveling_settings (
    guild_id            BIGINT      PRIMARY KEY,
    enabled             BOOLEAN     NOT NULL DEFAULT TRUE,
    xp_cooldown_seconds INTEGER     NOT NULL DEFAULT 60,
    xp_min              INTEGER     NOT NULL DEFAULT 15,
    xp_max              INTEGER     NOT NULL DEFAULT 25,
    announce_channel_id BIGINT      NOT NULL DEFAULT 0, -- 0 = announce in-channel
    announce_enabled    BOOLEAN     NOT NULL DEFAULT TRUE,
    stack_roles         BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS leveling_users (
    guild_id   BIGINT      NOT NULL,
    user_id    BIGINT      NOT NULL,
    xp         BIGINT      NOT NULL DEFAULT 0,
    level      INTEGER     NOT NULL DEFAULT 0,
    messages   BIGINT      NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (guild_id, user_id)
);

-- Leaderboard ordering and rank lookups.
CREATE INDEX IF NOT EXISTS idx_leveling_users_rank
    ON leveling_users (guild_id, xp DESC);

CREATE TABLE IF NOT EXISTS leveling_rewards (
    guild_id BIGINT  NOT NULL,
    level    INTEGER NOT NULL,
    role_id  BIGINT  NOT NULL,
    PRIMARY KEY (guild_id, level)
);
