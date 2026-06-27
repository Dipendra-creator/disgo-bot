-- 0006_economy: per-guild virtual currency (NON-GAMBLING).
--
-- Members earn currency from /daily and /work, hold it in a wallet and bank,
-- transfer it, and spend it in a per-guild shop (optionally granting roles).
-- There are deliberately no gambling mechanics. IDs are BIGINT snowflakes.

CREATE TABLE IF NOT EXISTS economy_settings (
    guild_id             BIGINT      PRIMARY KEY,
    currency_name        TEXT        NOT NULL DEFAULT 'coins',
    currency_symbol      TEXT        NOT NULL DEFAULT '🪙',
    daily_amount         BIGINT      NOT NULL DEFAULT 250,
    work_min             BIGINT      NOT NULL DEFAULT 50,
    work_max             BIGINT      NOT NULL DEFAULT 250,
    work_cooldown_secs   INTEGER     NOT NULL DEFAULT 3600,
    starting_balance     BIGINT      NOT NULL DEFAULT 0,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS economy_users (
    guild_id   BIGINT      NOT NULL,
    user_id    BIGINT      NOT NULL,
    wallet     BIGINT      NOT NULL DEFAULT 0,
    bank       BIGINT      NOT NULL DEFAULT 0,
    last_daily TIMESTAMPTZ,
    last_work  TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (guild_id, user_id)
);

-- Net-worth leaderboard ordering.
CREATE INDEX IF NOT EXISTS idx_economy_users_networth
    ON economy_users (guild_id, (wallet + bank) DESC);

CREATE TABLE IF NOT EXISTS economy_shop (
    id          BIGSERIAL   PRIMARY KEY,
    guild_id    BIGINT      NOT NULL,
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    price       BIGINT      NOT NULL,
    role_id     BIGINT      NOT NULL DEFAULT 0, -- 0 = no role granted
    stock       INTEGER     NOT NULL DEFAULT -1, -- -1 = unlimited
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (guild_id, name)
);

CREATE TABLE IF NOT EXISTS economy_inventory (
    guild_id BIGINT NOT NULL,
    user_id  BIGINT NOT NULL,
    item_id  BIGINT NOT NULL,
    quantity INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (guild_id, user_id, item_id)
);
