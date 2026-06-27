package ai

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

// saveSettings upserts the configuration row.
func (r *repo) saveSettings(ctx context.Context, s *Settings) error {
	_, err := r.db.NewInsert().Model(s).
		On("CONFLICT (guild_id) DO UPDATE").
		Set("assistant_channel_id = EXCLUDED.assistant_channel_id").
		Set("system_prompt = EXCLUDED.system_prompt").
		Set("updated_at = now()").
		Exec(ctx)
	return err
}
