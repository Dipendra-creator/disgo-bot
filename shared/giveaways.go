package shared

import (
	"context"
	"time"
)

// Giveaways is an optional contract a Module implements to expose its prize
// draws to the web dashboard: list/create giveaways and end or reroll them. Like
// Configurable and Moderation it is a purely additive, transport-agnostic seam —
// the web layer type-asserts each registered Module to Giveaways, and modules
// that don't implement it simply aren't surfaced as a management console.
// Implementations never import net/http and exchange only the plain types here.
type Giveaways interface {
	// ListGiveaways returns a page of a guild's giveaways (active and ended),
	// newest first, with each row's entry count.
	ListGiveaways(ctx context.Context, guildID int64, q PageQuery) (GiveawayPage, error)
	// CreateGiveaway starts a giveaway hosted by the acting dashboard user and
	// returns it. It returns a UserError for invalid input (empty prize, bad
	// duration or winner count, missing channel).
	CreateGiveaway(ctx context.Context, guildID int64, in GiveawayInput, hostID int64) (GiveawayView, error)
	// EndGiveaway closes a running giveaway immediately and draws its winners.
	// It returns a UserError when the giveaway is missing or already ended.
	EndGiveaway(ctx context.Context, guildID, giveawayID int64) (GiveawayView, error)
	// RerollGiveaway draws fresh winners for an already-ended giveaway. A winners
	// value <= 0 reuses the giveaway's configured count. It returns a UserError
	// when the giveaway is missing, not yet ended, or had no entries.
	RerollGiveaway(ctx context.Context, guildID, giveawayID int64, winners int) (GiveawayView, error)
}

// GiveawayInput is the editable payload for creating a giveaway. The host is
// taken from the server-side session, never from this body.
type GiveawayInput struct {
	ChannelID  string // channel the entry panel is posted to
	Prize      string
	DurationMS int64 // time until the draw
	Winners    int
}

// GiveawayView is one giveaway in the dashboard's transport-agnostic form. IDs
// are strings to preserve snowflake precision across JSON.
type GiveawayView struct {
	ID        int64     `json:"id"`
	ChannelID string    `json:"channel_id"`
	Prize     string    `json:"prize"`
	Winners   int       `json:"winners"`
	HostID    string    `json:"host_id"`
	HostName  string    `json:"host_name,omitempty"` // best-effort from the gateway cache
	Entries   int       `json:"entries"`
	Ended     bool      `json:"ended"`
	WinnerIDs []string  `json:"winner_ids"` // populated once ended
	EndsAt    time.Time `json:"ends_at"`
	CreatedAt time.Time `json:"created_at"`
}

// GiveawayPage is one page of giveaways together with the total count (ignoring
// Limit/Offset) so the dashboard can paginate.
type GiveawayPage struct {
	Giveaways []GiveawayView `json:"giveaways"`
	Total     int            `json:"total"`
}
