package giveaways

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/shared"
	"go.uber.org/zap"
)

const (
	// sweepInterval is how often the sweeper looks for expired giveaways.
	sweepInterval = 20 * time.Second
	// opTimeout bounds a single command/service operation.
	opTimeout = 8 * time.Second
)

// Service holds the giveaway business logic, independent of the interaction
// layer.
type Service struct {
	deps *shared.Deps
	repo *repo
	log  *zap.Logger
}

// NewService constructs the giveaway service.
func NewService(d *shared.Deps) *Service {
	return &Service{deps: d, repo: newRepo(d.DB), log: d.Log}
}

// Create starts a giveaway: it persists the row, posts the entry panel and
// records the message ID.
func (s *Service) Create(ctx context.Context, guildID, channelID, prize string, dur time.Duration, winners int, host *discordgo.User) (*Giveaway, error) {
	g := &Giveaway{
		GuildID:   pid(guildID),
		ChannelID: pid(channelID),
		Prize:     prize,
		Winners:   winners,
		HostID:    pid(host.ID),
		EndsAt:    time.Now().Add(dur),
	}
	if err := s.repo.create(ctx, g); err != nil {
		return nil, err
	}

	msg, err := s.deps.Session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Flags:      discordgo.MessageFlagsIsComponentsV2,
		Components: activePanel(g, 0),
	})
	if err != nil {
		// Roll back the orphaned row so a failed post leaves no ghost giveaway.
		if derr := s.repo.delete(ctx, g.ID); derr != nil {
			s.log.Warn("rollback giveaway failed", zap.Error(derr), zap.Int64("id", g.ID))
		}
		return nil, fmt.Errorf("post giveaway: %w", err)
	}
	g.MessageID = pid(msg.ID)
	if err := s.repo.setMessage(ctx, g.ID, g.MessageID); err != nil {
		s.log.Warn("record giveaway message failed", zap.Error(err), zap.Int64("id", g.ID))
	}
	return g, nil
}

// ToggleEntry adds or removes a member's entry and refreshes the panel count.
func (s *Service) ToggleEntry(ctx context.Context, giveawayID int64, guildID, userID string) (entered bool, count int, err error) {
	g, err := s.repo.byID(ctx, giveawayID)
	if err != nil {
		return false, 0, err
	}
	if g.GuildID != pid(guildID) {
		return false, 0, ErrNotFound
	}
	if g.Ended {
		return false, 0, ErrEnded
	}

	added, err := s.repo.addEntry(ctx, giveawayID, pid(userID))
	if err != nil {
		return false, 0, err
	}
	if !added {
		if _, err := s.repo.removeEntry(ctx, giveawayID, pid(userID)); err != nil {
			return false, 0, err
		}
	}
	count, err = s.repo.countEntries(ctx, giveawayID)
	if err != nil {
		return false, 0, err
	}
	s.editPanel(g, activePanel(g, count))
	return added, count, nil
}

// End closes a running giveaway immediately and draws its winners.
func (s *Service) End(ctx context.Context, guildID string, giveawayID int64) (*Giveaway, []string, error) {
	g, err := s.repo.byID(ctx, giveawayID)
	if err != nil {
		return nil, nil, err
	}
	if g.GuildID != pid(guildID) {
		return nil, nil, ErrNotFound
	}
	if g.Ended {
		return nil, nil, ErrEnded
	}
	winners, err := s.endGiveaway(ctx, g)
	return g, winners, err
}

// Reroll draws fresh winners for an already-ended giveaway.
func (s *Service) Reroll(ctx context.Context, guildID string, giveawayID int64, winners int) (*Giveaway, []string, error) {
	g, err := s.repo.byID(ctx, giveawayID)
	if err != nil {
		return nil, nil, err
	}
	if g.GuildID != pid(guildID) {
		return nil, nil, ErrNotFound
	}
	if !g.Ended {
		return nil, nil, ErrNotEnded
	}
	if winners <= 0 {
		winners = g.Winners
	}
	entrants, err := s.repo.entrants(ctx, giveawayID)
	if err != nil {
		return nil, nil, err
	}
	if len(entrants) == 0 {
		return nil, nil, ErrNoEntries
	}

	won := drawWinners(entrants, winners)
	if err := s.repo.setWinners(ctx, g.ID, joinIDs(won)); err != nil {
		return nil, nil, err
	}
	g.WinnerIDs = joinIDs(won)
	wons := g.winnerList()
	s.announce(g.ChannelID, rerollText(g, wons))
	return g, wons, nil
}

// ListActive returns a guild's running giveaways.
func (s *Service) ListActive(ctx context.Context, guildID string) ([]Giveaway, error) {
	return s.repo.listActive(ctx, pid(guildID))
}

// List returns a page of a guild's giveaways (active and ended) with the total
// count. Used by the web dashboard's giveaway manager.
func (s *Service) List(ctx context.Context, guildID int64, limit, offset int) ([]Giveaway, int, error) {
	return s.repo.listForGuild(ctx, guildID, limit, offset)
}

// EntryCount returns how many members have entered a giveaway.
func (s *Service) EntryCount(ctx context.Context, giveawayID int64) (int, error) {
	return s.repo.countEntries(ctx, giveawayID)
}

// endGiveaway draws winners, persists the result, updates the panel and pings
// the winners in the channel. Shared by manual End and the sweeper.
func (s *Service) endGiveaway(ctx context.Context, g *Giveaway) ([]string, error) {
	entrants, err := s.repo.entrants(ctx, g.ID)
	if err != nil {
		return nil, err
	}
	won := drawWinners(entrants, g.Winners)
	if err := s.repo.markEnded(ctx, g.ID, joinIDs(won)); err != nil {
		return nil, err
	}
	g.Ended = true
	g.WinnerIDs = joinIDs(won)

	wons := g.winnerList()
	s.editPanel(g, endedPanel(g, len(entrants), wons))
	s.announce(g.ChannelID, announceText(g, wons))
	return wons, nil
}

// editPanel rewrites the giveaway's panel message (no-op if it wasn't recorded).
func (s *Service) editPanel(g *Giveaway, components []discordgo.MessageComponent) {
	if g.MessageID == 0 {
		return
	}
	flags := discordgo.MessageFlagsIsComponentsV2
	if _, err := s.deps.Session.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    sid(g.ChannelID),
		ID:         sid(g.MessageID),
		Flags:      flags,
		Components: &components,
	}); err != nil {
		s.log.Warn("edit giveaway panel failed", zap.Error(err), zap.Int64("giveaway", g.ID))
	}
}

// announce posts a winner ping that is allowed to mention the winning users.
func (s *Service) announce(channelID int64, content string) {
	_, err := s.deps.Session.ChannelMessageSendComplex(sid(channelID), &discordgo.MessageSend{
		Content: content,
		AllowedMentions: &discordgo.MessageAllowedMentions{
			Parse: []discordgo.AllowedMentionType{discordgo.AllowedMentionTypeUsers},
		},
	})
	if err != nil {
		s.log.Warn("announce giveaway failed", zap.Error(err), zap.Int64("channel", channelID))
	}
}

// runSweeper ends giveaways whose timers have expired until ctx is canceled.
func (s *Service) runSweeper(ctx context.Context) {
	ticker := time.NewTicker(sweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sweep(ctx)
		}
	}
}

func (s *Service) sweep(ctx context.Context) {
	cctx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()

	due, err := s.repo.due(cctx, time.Now())
	if err != nil {
		s.log.Warn("giveaway sweep query failed", zap.Error(err))
		return
	}
	for i := range due {
		g := &due[i]
		if _, err := s.endGiveaway(cctx, g); err != nil {
			s.log.Warn("giveaway auto-end failed", zap.Error(err), zap.Int64("giveaway", g.ID))
		}
	}
}
