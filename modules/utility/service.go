package utility

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/pkg/snowflake"
	"github.com/dipu-sharma/disgo-bot/shared"
)

// Service holds the utility module's business logic, kept separate from the
// command handlers so it can be unit-tested in isolation.
type Service struct {
	deps *shared.Deps
}

// NewService constructs the utility service.
func NewService(d *shared.Deps) *Service { return &Service{deps: d} }

// GuildStats is a snapshot of a guild's headline statistics.
type GuildStats struct {
	ID        string
	Name      string
	IconURL   string
	OwnerID   string
	Members   int
	Channels  int
	Roles     int
	Boosts    int
	BoostTier int
	CreatedAt time.Time
}

// GuildStats fetches a guild snapshot via REST (no privileged intents needed).
func (s *Service) GuildStats(sess *discordgo.Session, guildID string) (*GuildStats, error) {
	g, err := sess.GuildWithCounts(guildID)
	if err != nil {
		return nil, fmt.Errorf("fetch guild: %w", err)
	}
	channels, err := sess.GuildChannels(guildID)
	if err != nil {
		return nil, fmt.Errorf("fetch channels: %w", err)
	}
	created, _ := snowflake.Timestamp(g.ID)

	return &GuildStats{
		ID:        g.ID,
		Name:      g.Name,
		IconURL:   g.IconURL("256"),
		OwnerID:   g.OwnerID,
		Members:   g.ApproximateMemberCount,
		Channels:  len(channels),
		Roles:     len(g.Roles),
		Boosts:    g.PremiumSubscriptionCount,
		BoostTier: int(g.PremiumTier),
		CreatedAt: created,
	}, nil
}

// AccountAge returns how long ago an account/entity was created, derived from
// its snowflake ID.
func AccountAge(id string, now time.Time) (time.Duration, error) {
	created, err := snowflake.Timestamp(id)
	if err != nil {
		return 0, err
	}
	return now.Sub(created), nil
}
