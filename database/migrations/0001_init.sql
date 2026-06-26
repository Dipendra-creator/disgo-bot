-- 0001_init: foundational schema.
-- Guild-scoped configuration and a generic key/value settings store that
-- feature modules build upon. Every table is namespaced by guild_id to enforce
-- tenant isolation at the data layer.

CREATE TABLE IF NOT EXISTS guilds (
    id          BIGINT      PRIMARY KEY,            -- Discord guild (snowflake)
    name        TEXT        NOT NULL DEFAULT '',
    locale      TEXT        NOT NULL DEFAULT 'en-US',
    joined_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    left_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Per-guild, per-module settings. value is JSON so each module owns its shape.
CREATE TABLE IF NOT EXISTS guild_settings (
    guild_id    BIGINT      NOT NULL REFERENCES guilds(id) ON DELETE CASCADE,
    module      TEXT        NOT NULL,
    value       JSONB       NOT NULL DEFAULT '{}'::jsonb,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (guild_id, module)
);

CREATE INDEX IF NOT EXISTS idx_guild_settings_module ON guild_settings (module);
