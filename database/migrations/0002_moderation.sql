-- 0002_moderation: infraction logging and per-guild moderation configuration.
--
-- moderation_cases is the audit trail of every action (ban/kick/timeout/warn/…).
-- case_number is a per-guild, monotonically increasing identifier surfaced to
-- moderators; it is allocated atomically from moderation_case_counters so two
-- concurrent actions never collide. IDs are stored as BIGINT (snowflakes).

CREATE TABLE IF NOT EXISTS moderation_settings (
    guild_id           BIGINT      PRIMARY KEY,
    mod_log_channel_id BIGINT,
    dm_on_action       BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS moderation_case_counters (
    guild_id    BIGINT  PRIMARY KEY,
    last_number BIGINT  NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS moderation_cases (
    id           BIGSERIAL   PRIMARY KEY,
    guild_id     BIGINT      NOT NULL,
    case_number  BIGINT      NOT NULL,
    action       TEXT        NOT NULL,
    target_id    BIGINT      NOT NULL,
    moderator_id BIGINT      NOT NULL DEFAULT 0,
    reason       TEXT        NOT NULL DEFAULT '',
    duration_ms  BIGINT      NOT NULL DEFAULT 0,
    expires_at   TIMESTAMPTZ,
    active       BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (guild_id, case_number)
);

CREATE INDEX IF NOT EXISTS idx_mod_cases_guild_target ON moderation_cases (guild_id, target_id);
CREATE INDEX IF NOT EXISTS idx_mod_cases_guild_action ON moderation_cases (guild_id, action);
-- Supports the tempban sweeper's "active bans due to expire" query.
CREATE INDEX IF NOT EXISTS idx_mod_cases_expiry
    ON moderation_cases (action, active, expires_at)
    WHERE expires_at IS NOT NULL;
