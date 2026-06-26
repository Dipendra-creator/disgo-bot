-- 0003_tickets: support-ticket system.
--
-- Each guild configures a category (where ticket channels are created), an
-- optional staff role (granted access to every ticket) and a log channel
-- (where transcripts are posted on close). Ticket numbers are per-guild and
-- allocated atomically from ticket_counters. IDs are BIGINT snowflakes.

CREATE TABLE IF NOT EXISTS ticket_settings (
    guild_id         BIGINT      PRIMARY KEY,
    category_id      BIGINT,
    staff_role_id    BIGINT,
    log_channel_id   BIGINT,
    panel_channel_id BIGINT,
    panel_message_id BIGINT,
    welcome_message  TEXT        NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS ticket_counters (
    guild_id    BIGINT PRIMARY KEY,
    last_number BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS tickets (
    id           BIGSERIAL   PRIMARY KEY,
    guild_id     BIGINT      NOT NULL,
    number       BIGINT      NOT NULL,
    channel_id   BIGINT      NOT NULL,
    opener_id    BIGINT      NOT NULL,
    claimer_id   BIGINT      NOT NULL DEFAULT 0,
    subject      TEXT        NOT NULL DEFAULT '',
    status       TEXT        NOT NULL DEFAULT 'open',
    closed_by    BIGINT      NOT NULL DEFAULT 0,
    close_reason TEXT        NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    closed_at    TIMESTAMPTZ,
    UNIQUE (guild_id, number)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tickets_channel ON tickets (channel_id);
-- Supports the "does this user already have an open ticket?" check.
CREATE INDEX IF NOT EXISTS idx_tickets_open_by_opener
    ON tickets (guild_id, opener_id)
    WHERE status <> 'closed';
