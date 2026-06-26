package moderation

import (
	"time"

	"github.com/uptrace/bun"
)

// Action identifies the kind of moderation action recorded in a case. A
// temporary ban is stored as ActionBan with a non-zero duration/expiry.
const (
	ActionBan       = "ban"
	ActionUnban     = "unban"
	ActionKick      = "kick"
	ActionTimeout   = "timeout"
	ActionUntimeout = "untimeout"
	ActionWarn      = "warn"
)

// Case is one logged moderation action against a user in a guild. CaseNumber is
// a per-guild, monotonically increasing identifier surfaced to moderators.
type Case struct {
	bun.BaseModel `bun:"table:moderation_cases,alias:mc"`

	ID          int64     `bun:"id,pk,autoincrement"`
	GuildID     int64     `bun:"guild_id,notnull"`
	CaseNumber  int64     `bun:"case_number,notnull"`
	Action      string    `bun:"action,notnull"`
	TargetID    int64     `bun:"target_id,notnull"`
	ModeratorID int64     `bun:"moderator_id,notnull"` // 0 = system/automatic
	Reason      string    `bun:"reason,notnull"`
	DurationMS  int64     `bun:"duration_ms,notnull"` // 0 = permanent/none
	ExpiresAt   time.Time `bun:"expires_at,nullzero"`
	Active      bool      `bun:"active,notnull"`
	CreatedAt   time.Time `bun:"created_at,nullzero,notnull,default:now()"`
	UpdatedAt   time.Time `bun:"updated_at,nullzero,notnull,default:now()"`
}

// Duration returns the action's configured length (0 when permanent/none).
func (c *Case) Duration() time.Duration {
	return time.Duration(c.DurationMS) * time.Millisecond
}

// Temporary reports whether the case represents a time-limited action.
func (c *Case) Temporary() bool { return c.DurationMS > 0 }

// Settings is per-guild moderation configuration.
type Settings struct {
	bun.BaseModel `bun:"table:moderation_settings,alias:ms"`

	GuildID         int64     `bun:"guild_id,pk"`
	ModLogChannelID int64     `bun:"mod_log_channel_id,nullzero"` // 0 = unset
	DMOnAction      bool      `bun:"dm_on_action,notnull"`
	CreatedAt       time.Time `bun:"created_at,nullzero,notnull,default:now()"`
	UpdatedAt       time.Time `bun:"updated_at,nullzero,notnull,default:now()"`
}
