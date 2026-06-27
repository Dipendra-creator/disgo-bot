package verification

import (
	"context"
	"fmt"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/shared"
	"go.uber.org/zap"
)

// Service holds the verification business logic, independent of the interaction
// layer. Per-guild settings are cached in-process and invalidated on change.
type Service struct {
	deps  *shared.Deps
	repo  *repo
	log   *zap.Logger
	mu    sync.RWMutex
	cache map[int64]*Settings
}

// NewService constructs the verification service.
func NewService(d *shared.Deps) *Service {
	return &Service{deps: d, repo: newRepo(d.DB), log: d.Log, cache: map[int64]*Settings{}}
}

// settings returns a guild's configuration, reading through an in-process cache.
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

func (s *Service) invalidate(guildID int64) {
	s.mu.Lock()
	delete(s.cache, guildID)
	s.mu.Unlock()
}

// Settings returns a guild's configuration (defaults when unset).
func (s *Service) Settings(ctx context.Context, guildID string) (*Settings, error) {
	return s.settings(ctx, pid(guildID))
}

// VerifiedCount returns how many members have verified in a guild.
func (s *Service) VerifiedCount(ctx context.Context, guildID string) (int, error) {
	return s.repo.countVerified(ctx, pid(guildID))
}

// Setup persists the verification configuration and enables it.
func (s *Service) Setup(ctx context.Context, guildID string, roleID, logChannelID int64, message, buttonLabel string) error {
	set := &Settings{
		GuildID:      pid(guildID),
		Enabled:      true,
		RoleID:       roleID,
		LogChannelID: logChannelID,
		Message:      message,
		ButtonLabel:  buttonLabel,
	}
	if err := s.repo.saveSettings(ctx, set); err != nil {
		return err
	}
	s.invalidate(pid(guildID))
	return nil
}

// SaveSettings persists the core config (excluding panel references) and
// refreshes the cache. Used by the web dashboard's partial-patch path.
func (s *Service) SaveSettings(ctx context.Context, set *Settings) error {
	if err := s.repo.saveSettings(ctx, set); err != nil {
		return err
	}
	s.invalidate(set.GuildID)
	return nil
}

// SetEnabled toggles verification without touching the rest of the config.
func (s *Service) SetEnabled(ctx context.Context, guildID string, enabled bool) error {
	set, err := s.repo.getSettings(ctx, pid(guildID))
	if err != nil {
		return err
	}
	set.Enabled = enabled
	if err := s.repo.saveSettings(ctx, set); err != nil {
		return err
	}
	s.invalidate(pid(guildID))
	return nil
}

// PostPanel publishes a verification panel (a Components-v2 message with a verify
// button) to a channel and records its location.
func (s *Service) PostPanel(ctx context.Context, guildID, channelID, title, desc, buttonLabel string) error {
	msg, err := s.deps.Session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Flags:      discordgo.MessageFlagsIsComponentsV2,
		Components: panelComponents(title, desc, buttonLabel),
	})
	if err != nil {
		return fmt.Errorf("post panel: %w", err)
	}
	if err := s.repo.setPanel(ctx, pid(guildID), pid(channelID), pid(msg.ID)); err != nil {
		s.log.Warn("record panel failed", zap.Error(err))
	}
	s.invalidate(pid(guildID))
	return nil
}

// VerifyResult reports the outcome of a verify click.
type VerifyResult struct {
	AlreadyHadRole bool
	RoleID         int64
}

// Verify grants the configured role to a member (idempotently) and audits the
// first time they do so.
func (s *Service) Verify(ctx context.Context, guildID string, member *discordgo.Member, user *discordgo.User) (*VerifyResult, error) {
	set, err := s.settings(ctx, pid(guildID))
	if err != nil {
		return nil, err
	}
	if !set.Configured() {
		return nil, shared.UserErr("Verification isn't set up on this server yet.")
	}

	roleStr := sid(set.RoleID)
	if member != nil {
		for _, r := range member.Roles {
			if r == roleStr {
				return &VerifyResult{AlreadyHadRole: true, RoleID: set.RoleID}, nil
			}
		}
	}

	if err := s.deps.Session.GuildMemberRoleAdd(guildID, user.ID, roleStr); err != nil {
		s.log.Warn("grant verified role failed", zap.Error(err),
			zap.String("guild", guildID), zap.String("user", user.ID))
		return nil, shared.UserErr("I couldn't assign the verified role. Make sure my role is above it and I have the Manage Roles permission.")
	}

	first, err := s.repo.recordVerification(ctx, pid(guildID), pid(user.ID))
	if err != nil {
		s.log.Warn("record verification failed", zap.Error(err))
	}
	if first && set.LogChannelID != 0 {
		s.postLog(set.LogChannelID, user)
	}
	return &VerifyResult{RoleID: set.RoleID}, nil
}

func (s *Service) postLog(logChannelID int64, user *discordgo.User) {
	_, err := s.deps.Session.ChannelMessageSendEmbed(sid(logChannelID), verifiedLogEmbed(user))
	if err != nil {
		s.log.Warn("post verification log failed", zap.Error(err), zap.Int64("channel", logChannelID))
	}
}
