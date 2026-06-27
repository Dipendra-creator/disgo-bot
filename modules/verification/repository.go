package verification

import (
	"context"
	"database/sql"
	"errors"

	"github.com/uptrace/bun"
)

type repo struct{ db *bun.DB }

func newRepo(db *bun.DB) *repo { return &repo{db: db} }

// getSettings returns a guild's configuration, or in-memory defaults when no row
// exists yet.
func (r *repo) getSettings(ctx context.Context, guildID int64) (*Settings, error) {
	s := new(Settings)
	err := r.db.NewSelect().Model(s).Where("guild_id = ?", guildID).Limit(1).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return defaultSettings(guildID), nil
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

// saveSettings upserts the core config, leaving panel references untouched.
func (r *repo) saveSettings(ctx context.Context, s *Settings) error {
	_, err := r.db.NewInsert().Model(s).
		On("CONFLICT (guild_id) DO UPDATE").
		Set("enabled = EXCLUDED.enabled").
		Set("role_id = EXCLUDED.role_id").
		Set("log_channel_id = EXCLUDED.log_channel_id").
		Set("message = EXCLUDED.message").
		Set("button_label = EXCLUDED.button_label").
		Set("updated_at = now()").
		Exec(ctx)
	return err
}

// setPanel records where a verification panel was posted.
func (r *repo) setPanel(ctx context.Context, guildID, channelID, messageID int64) error {
	s := &Settings{GuildID: guildID, PanelChannelID: channelID, PanelMessageID: messageID, Message: defaultMessage, ButtonLabel: defaultButtonLabel}
	_, err := r.db.NewInsert().Model(s).
		On("CONFLICT (guild_id) DO UPDATE").
		Set("panel_channel_id = EXCLUDED.panel_channel_id").
		Set("panel_message_id = EXCLUDED.panel_message_id").
		Set("updated_at = now()").
		Exec(ctx)
	return err
}

// recordVerification inserts an audit row. firstTime is false if the member had
// already verified before.
func (r *repo) recordVerification(ctx context.Context, guildID, userID int64) (firstTime bool, err error) {
	res, err := r.db.NewRaw(
		`INSERT INTO verification_records (guild_id, user_id)
		 VALUES (?, ?) ON CONFLICT (guild_id, user_id) DO NOTHING`,
		guildID, userID).Exec(ctx)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// countVerified returns how many members have verified in a guild.
func (r *repo) countVerified(ctx context.Context, guildID int64) (int, error) {
	return r.db.NewSelect().Model((*Record)(nil)).Where("guild_id = ?", guildID).Count(ctx)
}
