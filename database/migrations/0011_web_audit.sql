-- 0011_web_audit: audit trail of configuration changes made via the web
-- dashboard.
--
-- Every accepted PATCH to a module's config records who changed it (the Discord
-- user from the dashboard session), which guild and module, and the submitted
-- field changes as JSONB. This is append-only; the dashboard reads the most
-- recent rows per guild. IDs are stored as BIGINT snowflakes, matching the rest
-- of the schema; user_id is 0 only if a session token could not be parsed.

CREATE TABLE IF NOT EXISTS web_audit_log (
    id          BIGSERIAL   PRIMARY KEY,
    guild_id    BIGINT      NOT NULL,
    user_id     BIGINT      NOT NULL DEFAULT 0,
    username    TEXT        NOT NULL DEFAULT '',
    module      TEXT        NOT NULL,
    changes     JSONB       NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- The dashboard lists a guild's changes newest-first; index for that access path.
CREATE INDEX IF NOT EXISTS web_audit_log_guild_created_idx
    ON web_audit_log (guild_id, created_at DESC);
