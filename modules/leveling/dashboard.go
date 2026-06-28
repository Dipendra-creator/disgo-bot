package leveling

import (
	"context"
	"strconv"
	"strings"

	"github.com/dipu-sharma/disgo-bot/shared"
)

// Leveling implementation — exposes the XP leaderboard and level→role rewards to
// the web dashboard. It delegates to the existing Service so the dashboard and
// slash-command paths share identical persistence and reward semantics.

var _ shared.Leveling = (*Module)(nil)

// leaderboardLimit bounds a single page of the XP leaderboard.
const leaderboardLimit = 100

// maxRewardLevel bounds the level a reward can be attached to.
const maxRewardLevel = 1000

// XPLeaderboard returns a page of ranked members (highest XP first).
func (m *Module) XPLeaderboard(ctx context.Context, guildID int64, q shared.PageQuery) (shared.LevelLeaderboard, error) {
	limit := q.Limit
	if limit <= 0 || limit > leaderboardLimit {
		limit = leaderboardLimit
	}
	offset := q.Offset
	if offset < 0 {
		offset = 0
	}
	gid := sid(guildID)
	rows, total, err := m.svc.Leaderboard(ctx, gid, offset, limit)
	if err != nil {
		return shared.LevelLeaderboard{}, err
	}
	members := make([]shared.LevelMember, 0, len(rows))
	for i := range rows {
		u := &rows[i]
		uid := sid(u.UserID)
		members = append(members, shared.LevelMember{
			UserID:   uid,
			Username: m.memberName(gid, uid),
			XP:       u.XP,
			Level:    u.Level,
			Messages: u.Messages,
		})
	}
	return shared.LevelLeaderboard{Members: members, Total: total}, nil
}

// ListRewards returns the guild's level→role rewards, lowest level first.
func (m *Module) ListRewards(ctx context.Context, guildID int64) ([]shared.LevelReward, error) {
	rows, err := m.svc.Rewards(ctx, sid(guildID))
	if err != nil {
		return nil, err
	}
	out := make([]shared.LevelReward, 0, len(rows))
	for i := range rows {
		out = append(out, shared.LevelReward{Level: rows[i].Level, RoleID: sid(rows[i].RoleID)})
	}
	return out, nil
}

// SetReward maps a level threshold to a reward role (upsert).
func (m *Module) SetReward(ctx context.Context, guildID int64, level int, roleID string) error {
	if level < 1 || level > maxRewardLevel {
		return shared.UserErr("Level must be between 1 and %d.", maxRewardLevel)
	}
	role := strings.TrimSpace(roleID)
	if !isSnowflake(role) {
		return shared.UserErr("A valid reward role is required.")
	}
	return m.svc.AddReward(ctx, sid(guildID), level, role)
}

// RemoveReward clears the reward at a level.
func (m *Module) RemoveReward(ctx context.Context, guildID int64, level int) error {
	ok, err := m.svc.RemoveReward(ctx, sid(guildID), level)
	if err != nil {
		return err
	}
	if !ok {
		return shared.UserErr("No reward is configured at level %d.", level)
	}
	return nil
}

// isSnowflake reports whether s is a positive integer id.
func isSnowflake(s string) bool {
	n, err := strconv.ParseInt(s, 10, 64)
	return err == nil && n > 0
}

// memberName returns a display name for a member from the gateway cache, or ""
// when uncached / the session isn't ready. It never makes a Discord REST call.
func (m *Module) memberName(guildID, userID string) string {
	if m.deps == nil || m.deps.Session == nil || m.deps.Session.State == nil {
		return ""
	}
	mem, err := m.deps.Session.State.Member(guildID, userID)
	if err != nil || mem == nil {
		return ""
	}
	if mem.Nick != "" {
		return mem.Nick
	}
	if mem.User != nil {
		return mem.User.Username
	}
	return ""
}
