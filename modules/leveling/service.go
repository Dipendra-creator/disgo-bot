package leveling

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/shared"
	"go.uber.org/zap"
)

// awardTimeout bounds the XP award path triggered by a message event.
const awardTimeout = 8 * time.Second

// Service holds leveling business logic with an in-process settings cache.
type Service struct {
	deps *shared.Deps
	repo *repo
	log  *zap.Logger

	mu    sync.RWMutex
	cache map[int64]*Settings
}

// NewService constructs the leveling service.
func NewService(d *shared.Deps) *Service {
	return &Service{deps: d, repo: newRepo(d.DB), log: d.Log, cache: make(map[int64]*Settings)}
}

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

// Settings returns a guild's leveling configuration (defaults when unset).
func (s *Service) Settings(ctx context.Context, guildID string) (*Settings, error) {
	return s.settings(ctx, pid(guildID))
}

// SaveSettings persists settings and refreshes the cache.
func (s *Service) SaveSettings(ctx context.Context, set *Settings) error {
	if err := s.repo.saveSettings(ctx, set); err != nil {
		return err
	}
	s.invalidate(set.GuildID)
	return nil
}

// cooldownKey is the cache key gating a member's XP earning window.
func cooldownKey(guildID, userID string) string {
	return "lvl:cd:" + guildID + ":" + userID
}

// onCooldown reports whether the member earned XP within the cooldown window,
// and otherwise opens a fresh window. Cache failures fail open (award allowed).
func (s *Service) onCooldown(ctx context.Context, guildID, userID string, seconds int) bool {
	key := cooldownKey(guildID, userID)
	if ok, err := s.deps.Cache.Exists(ctx, key); err == nil && ok {
		return true
	}
	_ = s.deps.Cache.Set(ctx, key, "1", time.Duration(seconds)*time.Second)
	return false
}

// HandleMessage awards XP for a qualifying message and processes level-ups.
func (s *Service) HandleMessage(guildID, channelID string, author *discordgo.User) {
	if guildID == "" || author == nil || author.Bot {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), awardTimeout)
	defer cancel()

	set, err := s.settings(ctx, pid(guildID))
	if err != nil || !set.Enabled {
		return
	}
	if s.onCooldown(ctx, guildID, author.ID, set.XPCooldownSeconds) {
		return
	}

	delta := int64(randInt(set.XPMin, set.XPMax))
	newXP, oldLevel, err := s.repo.addXP(ctx, pid(guildID), pid(author.ID), delta)
	if err != nil {
		s.log.Warn("award xp failed", zap.Error(err))
		return
	}
	newLevel := levelForXP(newXP)
	if newLevel <= oldLevel {
		return
	}
	if err := s.repo.setLevel(ctx, pid(guildID), pid(author.ID), newLevel); err != nil {
		s.log.Warn("persist level failed", zap.Error(err))
	}
	s.onLevelUp(ctx, set, guildID, channelID, author, newLevel)
}

// onLevelUp grants reward roles and posts a level-up announcement.
func (s *Service) onLevelUp(ctx context.Context, set *Settings, guildID, channelID string, author *discordgo.User, level int) {
	s.applyRewards(ctx, set, guildID, author.ID, level)

	if !set.AnnounceEnabled {
		return
	}
	target := channelID
	if set.AnnounceChannelID != 0 {
		target = sid(set.AnnounceChannelID)
	}
	if _, err := s.deps.Session.ChannelMessageSendEmbed(target, levelUpEmbed(author, level)); err != nil {
		s.log.Warn("announce level-up failed", zap.Error(err))
	}
}

// applyRewards grants the role(s) earned at the new level. When roles don't
// stack, only the highest earned reward is kept and lower ones are removed.
func (s *Service) applyRewards(ctx context.Context, set *Settings, guildID, userID string, level int) {
	rewards, err := s.repo.rewardsUpTo(ctx, pid(guildID), level)
	if err != nil || len(rewards) == 0 {
		return
	}
	if set.StackRoles {
		for _, rw := range rewards {
			if err := s.deps.Session.GuildMemberRoleAdd(guildID, userID, sid(rw.RoleID)); err != nil {
				s.log.Warn("grant reward role failed", zap.Error(err), zap.Int("level", rw.Level))
			}
		}
		return
	}
	// Highest reward only (rewards[0] has the greatest level).
	top := rewards[0]
	if err := s.deps.Session.GuildMemberRoleAdd(guildID, userID, sid(top.RoleID)); err != nil {
		s.log.Warn("grant reward role failed", zap.Error(err), zap.Int("level", top.Level))
	}
	for _, rw := range rewards[1:] {
		if rw.RoleID == top.RoleID {
			continue
		}
		if err := s.deps.Session.GuildMemberRoleRemove(guildID, userID, sid(rw.RoleID)); err != nil {
			s.log.Debug("remove superseded reward role failed", zap.Error(err), zap.Int("level", rw.Level))
		}
	}
}

// --- query/admin operations used by commands ---

// Rank returns a member's standing and leaderboard position.
func (s *Service) Rank(ctx context.Context, guildID, userID string) (*UserLevel, progress, int, error) {
	u, err := s.repo.getUser(ctx, pid(guildID), pid(userID))
	if err != nil {
		return nil, progress{}, 0, err
	}
	rank, err := s.repo.rank(ctx, pid(guildID), pid(userID), u.XP)
	if err != nil {
		return nil, progress{}, 0, err
	}
	return u, progressFor(u.XP), rank, nil
}

// Leaderboard returns a page of ranked members plus the total ranked count.
func (s *Service) Leaderboard(ctx context.Context, guildID string, offset, limit int) ([]UserLevel, int, error) {
	rows, err := s.repo.leaderboard(ctx, pid(guildID), offset, limit)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.countRanked(ctx, pid(guildID))
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// GiveXP adds (or removes, when negative) XP and recomputes the level.
func (s *Service) GiveXP(ctx context.Context, guildID, userID string, delta int64) (*UserLevel, error) {
	u, err := s.repo.getUser(ctx, pid(guildID), pid(userID))
	if err != nil {
		return nil, err
	}
	xp := u.XP + delta
	if xp < 0 {
		xp = 0
	}
	return s.SetXP(ctx, guildID, userID, xp)
}

// SetXP overwrites a member's XP and derived level.
func (s *Service) SetXP(ctx context.Context, guildID, userID string, xp int64) (*UserLevel, error) {
	if xp < 0 {
		xp = 0
	}
	level := levelForXP(xp)
	if err := s.repo.setXP(ctx, pid(guildID), pid(userID), xp, level); err != nil {
		return nil, err
	}
	return &UserLevel{GuildID: pid(guildID), UserID: pid(userID), XP: xp, Level: level}, nil
}

// ResetUser clears a member's XP.
func (s *Service) ResetUser(ctx context.Context, guildID, userID string) error {
	return s.repo.resetUser(ctx, pid(guildID), pid(userID))
}

// AddReward maps a level to a reward role.
func (s *Service) AddReward(ctx context.Context, guildID string, level int, roleID string) error {
	return s.repo.addReward(ctx, pid(guildID), level, pid(roleID))
}

// RemoveReward clears the reward at a level.
func (s *Service) RemoveReward(ctx context.Context, guildID string, level int) (bool, error) {
	return s.repo.removeReward(ctx, pid(guildID), level)
}

// Rewards lists a guild's level rewards, lowest level first.
func (s *Service) Rewards(ctx context.Context, guildID string) ([]Reward, error) {
	return s.repo.listRewards(ctx, pid(guildID))
}

// randInt returns a value in [min, max]; it tolerates an inverted or equal range.
func randInt(min, max int) int {
	if max < min {
		min, max = max, min
	}
	if max == min {
		return min
	}
	return min + rand.Intn(max-min+1)
}
