package leveling

import (
	"time"

	"github.com/uptrace/bun"
)

// Settings is per-guild leveling configuration.
type Settings struct {
	bun.BaseModel `bun:"table:leveling_settings,alias:lv"`

	GuildID           int64     `bun:"guild_id,pk"`
	Enabled           bool      `bun:"enabled,notnull"`
	XPCooldownSeconds int       `bun:"xp_cooldown_seconds,notnull"`
	XPMin             int       `bun:"xp_min,notnull"`
	XPMax             int       `bun:"xp_max,notnull"`
	AnnounceChannelID int64     `bun:"announce_channel_id,notnull"` // 0 = reply in the active channel
	AnnounceEnabled   bool      `bun:"announce_enabled,notnull"`
	StackRoles        bool      `bun:"stack_roles,notnull"`
	CreatedAt         time.Time `bun:"created_at,nullzero,notnull,default:now()"`
	UpdatedAt         time.Time `bun:"updated_at,nullzero,notnull,default:now()"`
}

// defaultSettings returns the baseline configuration for an unconfigured guild.
func defaultSettings(guildID int64) *Settings {
	return &Settings{
		GuildID:           guildID,
		Enabled:           true,
		XPCooldownSeconds: 60,
		XPMin:             15,
		XPMax:             25,
		AnnounceEnabled:   true,
	}
}

// UserLevel is one member's XP standing in a guild.
type UserLevel struct {
	bun.BaseModel `bun:"table:leveling_users,alias:lu"`

	GuildID   int64     `bun:"guild_id,pk"`
	UserID    int64     `bun:"user_id,pk"`
	XP        int64     `bun:"xp,notnull"`
	Level     int       `bun:"level,notnull"`
	Messages  int64     `bun:"messages,notnull"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:now()"`
}

// Reward maps a level threshold to a role granted on reaching it.
type Reward struct {
	bun.BaseModel `bun:"table:leveling_rewards,alias:lr"`

	GuildID int64 `bun:"guild_id,pk"`
	Level   int   `bun:"level,pk"`
	RoleID  int64 `bun:"role_id,notnull"`
}
