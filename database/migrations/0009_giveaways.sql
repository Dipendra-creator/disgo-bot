-- 0009_giveaways: timed prize giveaways with a one-click entry button.
--
-- A host starts a giveaway (prize, duration, winner count); members enter via a
-- button; an in-process sweeper draws winners when the timer expires. Winners
-- can be rerolled afterwards. IDs are BIGINT snowflakes; the giveaway id is a
-- local BIGSERIAL.

CREATE TABLE IF NOT EXISTS giveaways (
    id          BIGSERIAL   PRIMARY KEY,
    guild_id    BIGINT      NOT NULL,
    channel_id  BIGINT      NOT NULL,
    message_id  BIGINT      NOT NULL DEFAULT 0,
    prize       TEXT        NOT NULL,
    winners     INTEGER     NOT NULL DEFAULT 1,
    host_id     BIGINT      NOT NULL,
    ends_at     TIMESTAMPTZ NOT NULL,
    ended       BOOLEAN     NOT NULL DEFAULT false,
    winner_ids  TEXT        NOT NULL DEFAULT '', -- comma-separated after the draw
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Sweeper lookup: active giveaways past their end time.
CREATE INDEX IF NOT EXISTS idx_giveaways_due ON giveaways (ended, ends_at);

CREATE TABLE IF NOT EXISTS giveaway_entries (
    giveaway_id BIGINT      NOT NULL REFERENCES giveaways (id) ON DELETE CASCADE,
    user_id     BIGINT      NOT NULL,
    entered_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (giveaway_id, user_id)
);
