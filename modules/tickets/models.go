package tickets

import (
	"time"

	"github.com/uptrace/bun"
)

// Ticket lifecycle states.
const (
	StatusOpen    = "open"
	StatusClaimed = "claimed"
	StatusClosed  = "closed"
)

// Ticket is one support ticket: a private channel tracked from open to close.
type Ticket struct {
	bun.BaseModel `bun:"table:tickets,alias:tk"`

	ID          int64     `bun:"id,pk,autoincrement"`
	GuildID     int64     `bun:"guild_id,notnull"`
	Number      int64     `bun:"number,notnull"`
	ChannelID   int64     `bun:"channel_id,notnull"`
	OpenerID    int64     `bun:"opener_id,notnull"`
	ClaimerID   int64     `bun:"claimer_id,notnull"` // 0 = unclaimed
	Subject     string    `bun:"subject,notnull"`
	Status      string    `bun:"status,notnull"`
	ClosedBy    int64     `bun:"closed_by,notnull"` // 0 = not closed
	CloseReason string    `bun:"close_reason,notnull"`
	CreatedAt   time.Time `bun:"created_at,nullzero,notnull,default:now()"`
	ClosedAt    time.Time `bun:"closed_at,nullzero"`
}

// Settings is per-guild ticket configuration.
type Settings struct {
	bun.BaseModel `bun:"table:ticket_settings,alias:ts"`

	GuildID        int64     `bun:"guild_id,pk"`
	CategoryID     int64     `bun:"category_id,nullzero"`
	StaffRoleID    int64     `bun:"staff_role_id,nullzero"`
	LogChannelID   int64     `bun:"log_channel_id,nullzero"`
	PanelChannelID int64     `bun:"panel_channel_id,nullzero"`
	PanelMessageID int64     `bun:"panel_message_id,nullzero"`
	WelcomeMessage string    `bun:"welcome_message,notnull"`
	CreatedAt      time.Time `bun:"created_at,nullzero,notnull,default:now()"`
	UpdatedAt      time.Time `bun:"updated_at,nullzero,notnull,default:now()"`
}

// Configured reports whether tickets can be opened (a category is set).
func (s *Settings) Configured() bool { return s != nil && s.CategoryID != 0 }
