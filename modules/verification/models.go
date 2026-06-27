package verification

import (
	"time"

	"github.com/uptrace/bun"
)

// Defaults applied when a guild has no stored configuration yet.
const (
	defaultMessage     = "Welcome! Click the button below to verify yourself and unlock the rest of the server."
	defaultButtonLabel = "Verify"
	defaultPanelTitle  = "Verification"
)

// Settings is per-guild verification configuration.
type Settings struct {
	bun.BaseModel `bun:"table:verification_settings,alias:vs"`

	GuildID        int64     `bun:"guild_id,pk"`
	Enabled        bool      `bun:"enabled,notnull"`
	RoleID         int64     `bun:"role_id,notnull"` // 0 = unset
	LogChannelID   int64     `bun:"log_channel_id,notnull"`
	Message        string    `bun:"message,notnull"`
	ButtonLabel    string    `bun:"button_label,notnull"`
	PanelChannelID int64     `bun:"panel_channel_id,notnull"`
	PanelMessageID int64     `bun:"panel_message_id,notnull"`
	CreatedAt      time.Time `bun:"created_at,nullzero,notnull,default:now()"`
	UpdatedAt      time.Time `bun:"updated_at,nullzero,notnull,default:now()"`
}

// defaultSettings returns the in-memory defaults for a guild with no stored row.
func defaultSettings(guildID int64) *Settings {
	return &Settings{
		GuildID:     guildID,
		Message:     defaultMessage,
		ButtonLabel: defaultButtonLabel,
	}
}

// Configured reports whether the verify button can grant a role (enabled and a
// role is set).
func (s *Settings) Configured() bool { return s != nil && s.Enabled && s.RoleID != 0 }

// Record audits a single member's verification.
type Record struct {
	bun.BaseModel `bun:"table:verification_records,alias:vr"`

	GuildID    int64     `bun:"guild_id,notnull"`
	UserID     int64     `bun:"user_id,notnull"`
	VerifiedAt time.Time `bun:"verified_at,nullzero,notnull,default:now()"`
}
