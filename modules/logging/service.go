package logging

import (
	"context"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/shared"
	"go.uber.org/zap"
)

// emitTimeout bounds a single settings lookup performed from an event handler.
const emitTimeout = 5 * time.Second

// Service holds the logging business logic and an in-process settings cache.
// Gateway events fire frequently, so settings are cached per guild and
// invalidated on change rather than read from the database every event.
type Service struct {
	deps *shared.Deps
	repo *repo
	log  *zap.Logger

	mu    sync.RWMutex
	cache map[int64]*Settings
}

// NewService constructs the logging service.
func NewService(d *shared.Deps) *Service {
	return &Service{deps: d, repo: newRepo(d.DB), log: d.Log, cache: make(map[int64]*Settings)}
}

// settings returns a guild's settings, consulting the in-process cache first.
func (s *Service) settings(ctx context.Context, guildID int64) (*Settings, error) {
	s.mu.RLock()
	cached, ok := s.cache[guildID]
	s.mu.RUnlock()
	if ok {
		return cached, nil
	}
	set, err := s.repo.getSettings(ctx, guildID)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.cache[guildID] = set
	s.mu.Unlock()
	return set, nil
}

// invalidate drops a guild's cached settings after a change.
func (s *Service) invalidate(guildID int64) {
	s.mu.Lock()
	delete(s.cache, guildID)
	s.mu.Unlock()
}

// Settings returns a guild's logging configuration (defaults when unset).
func (s *Service) Settings(ctx context.Context, guildID string) (*Settings, error) {
	return s.settings(ctx, pid(guildID))
}

// SetChannel routes a category to a channel (channelID "" disables it).
func (s *Service) SetChannel(ctx context.Context, guildID, category, channelID string) error {
	if err := s.repo.setChannel(ctx, pid(guildID), category, pid(channelID)); err != nil {
		return err
	}
	s.invalidate(pid(guildID))
	return nil
}

// emit posts an event embed to the channel configured for a category. It is a
// no-op when the guild has no channel set for that category. Called from event
// handlers, so it manages its own short-lived context and never blocks the
// caller meaningfully.
func (s *Service) emit(guildID, category string, embed *discordgo.MessageEmbed) {
	if guildID == "" || embed == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), emitTimeout)
	defer cancel()

	set, err := s.settings(ctx, pid(guildID))
	if err != nil {
		s.log.Warn("logging settings lookup failed", zap.Error(err), zap.String("guild", guildID))
		return
	}
	chID := set.channel(category)
	if chID == 0 {
		return
	}
	if _, err := s.deps.Session.ChannelMessageSendEmbed(sid(chID), embed); err != nil {
		s.log.Warn("post log embed failed",
			zap.Error(err), zap.String("category", category), zap.Int64("channel", chID))
	}
}

// botUserID returns the bot's own user ID, or "" if the session isn't ready.
func (s *Service) botUserID() string {
	if s.deps.Session.State != nil && s.deps.Session.State.User != nil {
		return s.deps.Session.State.User.ID
	}
	return ""
}
