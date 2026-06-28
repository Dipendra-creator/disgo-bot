-- 0012_automod_violations: an audit trail of enforced automod actions.
--
-- Every time a filter matches and the bot deletes a message (optionally timing
-- out the author), a row is appended here so moderators can review activity from
-- the dashboard. The author's display name is captured at write time because the
-- offending message — and sometimes the member — may be gone by the time the log
-- is read. IDs are BIGINT snowflakes; user_name is best-effort.

CREATE TABLE IF NOT EXISTS automod_violations (
    id         BIGSERIAL   PRIMARY KEY,
    guild_id   BIGINT      NOT NULL,
    user_id    BIGINT      NOT NULL,
    user_name  TEXT        NOT NULL DEFAULT '',
    channel_id BIGINT      NOT NULL DEFAULT 0,
    filter     TEXT        NOT NULL, -- words | invites | mentions | spam
    action     TEXT        NOT NULL, -- delete | timeout
    detail     TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_automod_violations_guild
    ON automod_violations (guild_id, created_at DESC);
