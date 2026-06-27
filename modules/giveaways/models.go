package giveaways

import (
	"strings"
	"time"

	"github.com/uptrace/bun"
)

// Bounds on giveaway parameters.
const (
	maxPrizeLen = 256
	maxWinners  = 50
	minDuration = time.Minute
	maxDuration = 90 * 24 * time.Hour // 90 days
)

// Giveaway is a single timed prize draw.
type Giveaway struct {
	bun.BaseModel `bun:"table:giveaways,alias:gw"`

	ID        int64     `bun:"id,pk,autoincrement"`
	GuildID   int64     `bun:"guild_id,notnull"`
	ChannelID int64     `bun:"channel_id,notnull"`
	MessageID int64     `bun:"message_id,notnull"`
	Prize     string    `bun:"prize,notnull"`
	Winners   int       `bun:"winners,notnull"`
	HostID    int64     `bun:"host_id,notnull"`
	EndsAt    time.Time `bun:"ends_at,notnull"`
	Ended     bool      `bun:"ended,notnull"`
	WinnerIDs string    `bun:"winner_ids,notnull"`
	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:now()"`
}

// winnerList splits the stored comma-separated winner IDs.
func (g *Giveaway) winnerList() []string {
	if g.WinnerIDs == "" {
		return nil
	}
	return strings.Split(g.WinnerIDs, ",")
}

// joinIDs renders winner IDs into the stored comma-separated form.
func joinIDs(ids []int64) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = sid(id)
	}
	return strings.Join(parts, ",")
}

// Entry is one member's participation in a giveaway.
type Entry struct {
	bun.BaseModel `bun:"table:giveaway_entries,alias:ge"`

	GiveawayID int64     `bun:"giveaway_id,notnull"`
	UserID     int64     `bun:"user_id,notnull"`
	EnteredAt  time.Time `bun:"entered_at,nullzero,notnull,default:now()"`
}
