package leveling

import (
	"context"
	"database/sql"
	"errors"

	"github.com/uptrace/bun"
)

type repo struct{ db *bun.DB }

func newRepo(db *bun.DB) *repo { return &repo{db: db} }

// --- settings ---

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

// saveSettings upserts the full settings row.
func (r *repo) saveSettings(ctx context.Context, s *Settings) error {
	_, err := r.db.NewInsert().Model(s).
		On("CONFLICT (guild_id) DO UPDATE").
		Set("enabled = EXCLUDED.enabled").
		Set("xp_cooldown_seconds = EXCLUDED.xp_cooldown_seconds").
		Set("xp_min = EXCLUDED.xp_min").
		Set("xp_max = EXCLUDED.xp_max").
		Set("announce_channel_id = EXCLUDED.announce_channel_id").
		Set("announce_enabled = EXCLUDED.announce_enabled").
		Set("stack_roles = EXCLUDED.stack_roles").
		Set("updated_at = now()").
		Exec(ctx)
	return err
}

// --- user XP ---

// addXP atomically adds delta XP (and one message) to a member, returning the
// new XP total and the level stored before this change.
func (r *repo) addXP(ctx context.Context, guildID, userID, delta int64) (newXP int64, oldLevel int, err error) {
	row := struct {
		XP    int64 `bun:"xp"`
		Level int   `bun:"level"`
	}{}
	err = r.db.NewRaw(
		`INSERT INTO leveling_users (guild_id, user_id, xp, level, messages)
		 VALUES (?, ?, ?, 0, 1)
		 ON CONFLICT (guild_id, user_id) DO UPDATE
		 SET xp = leveling_users.xp + ?, messages = leveling_users.messages + 1, updated_at = now()
		 RETURNING xp, level`,
		guildID, userID, delta, delta).Scan(ctx, &row)
	return row.XP, row.Level, err
}

// setLevel records a member's recomputed level.
func (r *repo) setLevel(ctx context.Context, guildID, userID int64, level int) error {
	_, err := r.db.NewUpdate().Model((*UserLevel)(nil)).
		Set("level = ?", level).
		Set("updated_at = now()").
		Where("guild_id = ? AND user_id = ?", guildID, userID).
		Exec(ctx)
	return err
}

func (r *repo) getUser(ctx context.Context, guildID, userID int64) (*UserLevel, error) {
	u := new(UserLevel)
	err := r.db.NewSelect().Model(u).
		Where("guild_id = ? AND user_id = ?", guildID, userID).Limit(1).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return &UserLevel{GuildID: guildID, UserID: userID}, nil
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

// setXP overwrites a member's XP and level (admin action).
func (r *repo) setXP(ctx context.Context, guildID, userID, xp int64, level int) error {
	u := &UserLevel{GuildID: guildID, UserID: userID, XP: xp, Level: level}
	_, err := r.db.NewInsert().Model(u).
		On("CONFLICT (guild_id, user_id) DO UPDATE").
		Set("xp = EXCLUDED.xp").
		Set("level = EXCLUDED.level").
		Set("updated_at = now()").
		Exec(ctx)
	return err
}

func (r *repo) resetUser(ctx context.Context, guildID, userID int64) error {
	_, err := r.db.NewDelete().Model((*UserLevel)(nil)).
		Where("guild_id = ? AND user_id = ?", guildID, userID).Exec(ctx)
	return err
}

// rank returns a member's 1-based leaderboard position by XP (0 when unranked).
func (r *repo) rank(ctx context.Context, guildID, userID, xp int64) (int, error) {
	if xp <= 0 {
		return 0, nil
	}
	ahead, err := r.db.NewSelect().Model((*UserLevel)(nil)).
		Where("guild_id = ? AND xp > ?", guildID, xp).Count(ctx)
	if err != nil {
		return 0, err
	}
	return ahead + 1, nil
}

func (r *repo) leaderboard(ctx context.Context, guildID int64, offset, limit int) ([]UserLevel, error) {
	var rows []UserLevel
	err := r.db.NewSelect().Model(&rows).
		Where("guild_id = ? AND xp > 0", guildID).
		Order("xp DESC").
		Offset(offset).Limit(limit).Scan(ctx)
	return rows, err
}

func (r *repo) countRanked(ctx context.Context, guildID int64) (int, error) {
	return r.db.NewSelect().Model((*UserLevel)(nil)).
		Where("guild_id = ? AND xp > 0", guildID).Count(ctx)
}

// --- rewards ---

func (r *repo) addReward(ctx context.Context, guildID int64, level int, roleID int64) error {
	rw := &Reward{GuildID: guildID, Level: level, RoleID: roleID}
	_, err := r.db.NewInsert().Model(rw).
		On("CONFLICT (guild_id, level) DO UPDATE").
		Set("role_id = EXCLUDED.role_id").
		Exec(ctx)
	return err
}

func (r *repo) removeReward(ctx context.Context, guildID int64, level int) (bool, error) {
	res, err := r.db.NewDelete().Model((*Reward)(nil)).
		Where("guild_id = ? AND level = ?", guildID, level).Exec(ctx)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (r *repo) listRewards(ctx context.Context, guildID int64) ([]Reward, error) {
	var rows []Reward
	err := r.db.NewSelect().Model(&rows).
		Where("guild_id = ?", guildID).Order("level ASC").Scan(ctx)
	return rows, err
}

// rewardsUpTo returns rewards whose level threshold is at or below level,
// highest level first (so the top reward is element 0).
func (r *repo) rewardsUpTo(ctx context.Context, guildID int64, level int) ([]Reward, error) {
	var rows []Reward
	err := r.db.NewSelect().Model(&rows).
		Where("guild_id = ? AND level <= ?", guildID, level).
		Order("level DESC").Scan(ctx)
	return rows, err
}
