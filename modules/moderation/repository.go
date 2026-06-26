package moderation

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/uptrace/bun"
)

// ErrCaseNotFound is returned when a case number doesn't exist in a guild.
var ErrCaseNotFound = errors.New("case not found")

// repo is the moderation module's data-access layer over bun/Postgres.
type repo struct{ db *bun.DB }

func newRepo(db *bun.DB) *repo { return &repo{db: db} }

// createCase allocates the next per-guild case number and inserts the case in a
// single transaction, so concurrent actions never share a number.
func (r *repo) createCase(ctx context.Context, c *Case) error {
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var number int64
		if err := tx.NewRaw(
			`INSERT INTO moderation_case_counters (guild_id, last_number)
			 VALUES (?, 1)
			 ON CONFLICT (guild_id)
			 DO UPDATE SET last_number = moderation_case_counters.last_number + 1
			 RETURNING last_number`, c.GuildID).Scan(ctx, &number); err != nil {
			return err
		}
		c.CaseNumber = number
		_, err := tx.NewInsert().Model(c).Exec(ctx)
		return err
	})
}

// getCase fetches a single case by its guild-scoped number.
func (r *repo) getCase(ctx context.Context, guildID, number int64) (*Case, error) {
	c := new(Case)
	err := r.db.NewSelect().Model(c).
		Where("guild_id = ? AND case_number = ?", guildID, number).
		Limit(1).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCaseNotFound
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

// listByTarget returns a user's cases, newest first, optionally filtered by
// action and to active rows only.
func (r *repo) listByTarget(ctx context.Context, guildID, targetID int64, action string, activeOnly bool) ([]Case, error) {
	var cs []Case
	q := r.db.NewSelect().Model(&cs).
		Where("guild_id = ? AND target_id = ?", guildID, targetID)
	if action != "" {
		q = q.Where("action = ?", action)
	}
	if activeOnly {
		q = q.Where("active = TRUE")
	}
	if err := q.Order("case_number DESC").Scan(ctx); err != nil {
		return nil, err
	}
	return cs, nil
}

// updateReason rewrites a case's reason, returning ErrCaseNotFound if absent.
func (r *repo) updateReason(ctx context.Context, guildID, number int64, reason string) error {
	res, err := r.db.NewUpdate().Model((*Case)(nil)).
		Set("reason = ?", reason).
		Set("updated_at = now()").
		Where("guild_id = ? AND case_number = ?", guildID, number).
		Exec(ctx)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrCaseNotFound
	}
	return nil
}

// deactivate marks a case inactive (e.g. a pardoned warning or expired ban).
func (r *repo) deactivate(ctx context.Context, guildID, number int64) error {
	res, err := r.db.NewUpdate().Model((*Case)(nil)).
		Set("active = FALSE").
		Set("updated_at = now()").
		Where("guild_id = ? AND case_number = ?", guildID, number).
		Exec(ctx)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrCaseNotFound
	}
	return nil
}

// dueTempbans returns active ban cases whose expiry has passed.
func (r *repo) dueTempbans(ctx context.Context, now time.Time) ([]Case, error) {
	var cs []Case
	err := r.db.NewSelect().Model(&cs).
		Where("action = ?", ActionBan).
		Where("active = TRUE").
		Where("expires_at IS NOT NULL").
		Where("expires_at <= ?", now).
		Order("expires_at ASC").
		Limit(100).
		Scan(ctx)
	return cs, err
}

// getSettings returns a guild's settings, or sensible defaults when unset.
func (r *repo) getSettings(ctx context.Context, guildID int64) (*Settings, error) {
	s := new(Settings)
	err := r.db.NewSelect().Model(s).
		Where("guild_id = ?", guildID).
		Limit(1).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return &Settings{GuildID: guildID, DMOnAction: true}, nil
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

// setModLogChannel upserts the mod-log channel, preserving other settings.
func (r *repo) setModLogChannel(ctx context.Context, guildID, channelID int64) error {
	s := &Settings{GuildID: guildID, ModLogChannelID: channelID, DMOnAction: true}
	_, err := r.db.NewInsert().Model(s).
		On("CONFLICT (guild_id) DO UPDATE").
		Set("mod_log_channel_id = EXCLUDED.mod_log_channel_id").
		Set("updated_at = now()").
		Exec(ctx)
	return err
}
