package automod

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

// saveSettings upserts the full configuration row.
func (r *repo) saveSettings(ctx context.Context, s *Settings) error {
	_, err := r.db.NewInsert().Model(s).
		On("CONFLICT (guild_id) DO UPDATE").
		Set("log_channel_id = EXCLUDED.log_channel_id").
		Set("exempt_role_id = EXCLUDED.exempt_role_id").
		Set("timeout_secs = EXCLUDED.timeout_secs").
		Set("words_enabled = EXCLUDED.words_enabled").
		Set("words_action = EXCLUDED.words_action").
		Set("invites_enabled = EXCLUDED.invites_enabled").
		Set("invites_action = EXCLUDED.invites_action").
		Set("mentions_enabled = EXCLUDED.mentions_enabled").
		Set("mentions_action = EXCLUDED.mentions_action").
		Set("mention_threshold = EXCLUDED.mention_threshold").
		Set("spam_enabled = EXCLUDED.spam_enabled").
		Set("spam_action = EXCLUDED.spam_action").
		Set("spam_count = EXCLUDED.spam_count").
		Set("spam_window_secs = EXCLUDED.spam_window_secs").
		Set("updated_at = now()").
		Exec(ctx)
	return err
}

// --- banned words ---

// addWord inserts a banned term; ok=false means it was already present.
func (r *repo) addWord(ctx context.Context, guildID int64, word string) (ok bool, err error) {
	res, err := r.db.NewRaw(
		`INSERT INTO automod_words (guild_id, word) VALUES (?, ?)
		 ON CONFLICT (guild_id, word) DO NOTHING`,
		guildID, word).Exec(ctx)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// removeWord deletes a banned term; ok=false means it wasn't present.
func (r *repo) removeWord(ctx context.Context, guildID int64, word string) (ok bool, err error) {
	res, err := r.db.NewDelete().Model((*Word)(nil)).
		Where("guild_id = ? AND word = ?", guildID, word).Exec(ctx)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// clearWords removes every banned term for a guild and returns how many.
func (r *repo) clearWords(ctx context.Context, guildID int64) (int, error) {
	res, err := r.db.NewDelete().Model((*Word)(nil)).Where("guild_id = ?", guildID).Exec(ctx)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// listWords returns a guild's banned terms in alphabetical order.
func (r *repo) listWords(ctx context.Context, guildID int64) ([]string, error) {
	var rows []Word
	err := r.db.NewSelect().Model(&rows).
		Where("guild_id = ?", guildID).Order("word ASC").Scan(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(rows))
	for i, w := range rows {
		out[i] = w.Word
	}
	return out, nil
}

// --- violation log ---

// insertViolation appends one enforced-action row.
func (r *repo) insertViolation(ctx context.Context, v *Violation) error {
	_, err := r.db.NewInsert().Model(v).Exec(ctx)
	return err
}

// listViolations returns a page of a guild's violation log (newest first) and
// the total count ignoring the window.
func (r *repo) listViolations(ctx context.Context, guildID int64, limit, offset int) ([]Violation, int, error) {
	var rows []Violation
	total, err := r.db.NewSelect().Model(&rows).
		Where("guild_id = ?", guildID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		ScanAndCount(ctx)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
