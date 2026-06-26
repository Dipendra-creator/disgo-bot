-- 0004_logging: server audit logging.
--
-- A guild routes event categories to log channels. Each category has its own
-- channel column (0 = that category is disabled). Categories:
--   message  - message edits and deletions
--   member   - member joins and leaves (needs the GuildMembers intent)
--   server   - bans/unbans, channel and role create/delete
-- IDs are BIGINT snowflakes.

CREATE TABLE IF NOT EXISTS logging_settings (
    guild_id           BIGINT      PRIMARY KEY,
    message_channel_id BIGINT      NOT NULL DEFAULT 0,
    member_channel_id  BIGINT      NOT NULL DEFAULT 0,
    server_channel_id  BIGINT      NOT NULL DEFAULT 0,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
