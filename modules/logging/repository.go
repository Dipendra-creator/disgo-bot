package logging

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/uptrace/bun"
)

type repo struct{ db *bun.DB }

func newRepo(db *bun.DB) *repo { return &repo{db: db} }

// getSettings returns a guild's logging configuration, or zero-valued defaults
// (all categories disabled) when unset.
func (r *repo) getSettings(ctx context.Context, guildID int64) (*Settings, error) {
	s := new(Settings)
	err := r.db.NewSelect().Model(s).
		Where("guild_id = ?", guildID).
		Limit(1).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return &Settings{GuildID: guildID}, nil
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

// setChannel upserts the channel for a single category (0 disables it).
func (r *repo) setChannel(ctx context.Context, guildID int64, category string, channelID int64) error {
	col, err := categoryColumn(category)
	if err != nil {
		return err
	}
	s := &Settings{GuildID: guildID}
	switch category {
	case CategoryMessage:
		s.MessageChannelID = channelID
	case CategoryMember:
		s.MemberChannelID = channelID
	case CategoryServer:
		s.ServerChannelID = channelID
	}
	_, err = r.db.NewInsert().Model(s).
		On("CONFLICT (guild_id) DO UPDATE").
		Set(col + " = EXCLUDED." + col).
		Set("updated_at = now()").
		Exec(ctx)
	return err
}

// categoryColumn maps a category to its settings column name.
func categoryColumn(category string) (string, error) {
	switch category {
	case CategoryMessage:
		return "message_channel_id", nil
	case CategoryMember:
		return "member_channel_id", nil
	case CategoryServer:
		return "server_channel_id", nil
	default:
		return "", fmt.Errorf("unknown logging category %q", category)
	}
}
