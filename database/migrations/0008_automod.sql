-- 0008_automod: content moderation filters (words, invites, mass mentions, spam).
--
-- Each filter has its own enable flag and action (delete the message, or delete
-- and timeout the author). Offending messages are always deleted on a match and
-- optionally mirrored to a log channel. A configurable role and anyone with
-- Manage Messages are exempt. IDs are BIGINT snowflakes.

CREATE TABLE IF NOT EXISTS automod_settings (
    guild_id          BIGINT      PRIMARY KEY,
    log_channel_id    BIGINT      NOT NULL DEFAULT 0, -- 0 = no logging
    exempt_role_id    BIGINT      NOT NULL DEFAULT 0, -- 0 = none
    timeout_secs      INTEGER     NOT NULL DEFAULT 300,

    words_enabled     BOOLEAN     NOT NULL DEFAULT false,
    words_action      TEXT        NOT NULL DEFAULT 'delete',

    invites_enabled   BOOLEAN     NOT NULL DEFAULT false,
    invites_action    TEXT        NOT NULL DEFAULT 'delete',

    mentions_enabled  BOOLEAN     NOT NULL DEFAULT false,
    mentions_action   TEXT        NOT NULL DEFAULT 'delete',
    mention_threshold INTEGER     NOT NULL DEFAULT 5,

    spam_enabled      BOOLEAN     NOT NULL DEFAULT false,
    spam_action       TEXT        NOT NULL DEFAULT 'delete',
    spam_count        INTEGER     NOT NULL DEFAULT 5,
    spam_window_secs  INTEGER     NOT NULL DEFAULT 5,

    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS automod_words (
    guild_id BIGINT NOT NULL,
    word     TEXT   NOT NULL,
    PRIMARY KEY (guild_id, word)
);
