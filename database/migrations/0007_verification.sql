-- 0007_verification: member gate via a one-click verify button.
--
-- An admin configures a verified role (and optional log channel) and posts a
-- panel; members click the button to receive the role. A records table audits
-- who verified and when. IDs are BIGINT snowflakes.

CREATE TABLE IF NOT EXISTS verification_settings (
    guild_id          BIGINT      PRIMARY KEY,
    enabled           BOOLEAN     NOT NULL DEFAULT false,
    role_id           BIGINT      NOT NULL DEFAULT 0, -- 0 = unset
    log_channel_id    BIGINT      NOT NULL DEFAULT 0, -- 0 = no logging
    message           TEXT        NOT NULL DEFAULT '',
    button_label      TEXT        NOT NULL DEFAULT 'Verify',
    panel_channel_id  BIGINT      NOT NULL DEFAULT 0,
    panel_message_id  BIGINT      NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS verification_records (
    guild_id    BIGINT      NOT NULL,
    user_id     BIGINT      NOT NULL,
    verified_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (guild_id, user_id)
);
