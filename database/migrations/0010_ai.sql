-- 0010_ai: per-guild settings for the AI assistant module.
--
-- Holds an optional opt-in "assistant channel" where the bot replies to every
-- message, and an optional custom system prompt. The provider credentials and
-- model live in the bot config, not the database. IDs are BIGINT snowflakes.

CREATE TABLE IF NOT EXISTS ai_settings (
    guild_id             BIGINT      PRIMARY KEY,
    assistant_channel_id BIGINT      NOT NULL DEFAULT 0, -- 0 = no opt-in channel
    system_prompt        TEXT        NOT NULL DEFAULT '',
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);
