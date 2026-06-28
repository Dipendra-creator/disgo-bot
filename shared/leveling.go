package shared

import "context"

// Leveling is an optional contract a Module implements to expose its XP data to
// the web dashboard: a ranked leaderboard and level→role reward management.
// Like the other dashboard seams it is purely additive and transport-agnostic —
// implementations never import net/http and exchange only the types here.
type Leveling interface {
	// XPLeaderboard returns a page of ranked members (highest XP first) plus
	// the total ranked count.
	XPLeaderboard(ctx context.Context, guildID int64, q PageQuery) (LevelLeaderboard, error)
	// ListRewards returns the guild's level→role rewards, lowest level first.
	ListRewards(ctx context.Context, guildID int64) ([]LevelReward, error)
	// SetReward maps a level threshold to a reward role (upsert). It returns a
	// UserError for an invalid level or role id.
	SetReward(ctx context.Context, guildID int64, level int, roleID string) error
	// RemoveReward clears the reward at a level. It returns a UserError when no
	// reward is configured at that level.
	RemoveReward(ctx context.Context, guildID int64, level int) error
}

// LevelMember is one ranked member on the XP leaderboard. IDs are strings to
// preserve snowflake precision across JSON.
type LevelMember struct {
	UserID   string `json:"user_id"`
	Username string `json:"username,omitempty"` // best-effort from the gateway cache
	XP       int64  `json:"xp"`
	Level    int    `json:"level"`
	Messages int64  `json:"messages"`
}

// LevelLeaderboard is one page of ranked members with the total ranked count.
type LevelLeaderboard struct {
	Members []LevelMember `json:"members"`
	Total   int           `json:"total"`
}

// LevelReward maps a level threshold to the role granted on reaching it.
type LevelReward struct {
	Level  int    `json:"level"`
	RoleID string `json:"role_id"`
}
