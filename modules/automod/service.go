package automod

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/shared"
	"go.uber.org/zap"
)

// checkTimeout bounds the work done for a single inbound message.
const checkTimeout = 5 * time.Second

// Service holds automod business logic. Per-guild settings and banned-word sets
// are cached in-process (the message path is hot) and invalidated on change. A
// spam tracker keeps a short sliding window of recent message times per member.
type Service struct {
	deps *shared.Deps
	repo *repo
	log  *zap.Logger

	mu       sync.RWMutex
	settings map[int64]*Settings
	words    map[int64]map[string]struct{}

	spamMu sync.Mutex
	spam   map[spamKey][]time.Time
}

type spamKey struct{ guild, user int64 }

// NewService constructs the automod service.
func NewService(d *shared.Deps) *Service {
	return &Service{
		deps:     d,
		repo:     newRepo(d.DB),
		log:      d.Log,
		settings: map[int64]*Settings{},
		words:    map[int64]map[string]struct{}{},
		spam:     map[spamKey][]time.Time{},
	}
}

// --- caches ---

func (s *Service) getSettings(ctx context.Context, guildID int64) (*Settings, error) {
	s.mu.RLock()
	cached, ok := s.settings[guildID]
	s.mu.RUnlock()
	if ok {
		return cached, nil
	}
	set, err := s.repo.getSettings(ctx, guildID)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.settings[guildID] = set
	s.mu.Unlock()
	return set, nil
}

func (s *Service) getWords(ctx context.Context, guildID int64) (map[string]struct{}, error) {
	s.mu.RLock()
	cached, ok := s.words[guildID]
	s.mu.RUnlock()
	if ok {
		return cached, nil
	}
	list, err := s.repo.listWords(ctx, guildID)
	if err != nil {
		return nil, err
	}
	set := make(map[string]struct{}, len(list))
	for _, w := range list {
		set[w] = struct{}{}
	}
	s.mu.Lock()
	s.words[guildID] = set
	s.mu.Unlock()
	return set, nil
}

func (s *Service) invalidate(guildID int64) {
	s.mu.Lock()
	delete(s.settings, guildID)
	s.mu.Unlock()
}

func (s *Service) invalidateWords(guildID int64) {
	s.mu.Lock()
	delete(s.words, guildID)
	s.mu.Unlock()
}

// --- config API (used by command handlers) ---

// Settings returns a guild's configuration (defaults when unset).
func (s *Service) Settings(ctx context.Context, guildID string) (*Settings, error) {
	return s.getSettings(ctx, pid(guildID))
}

// mutate loads a fresh settings row, applies fn, persists it and invalidates the
// cache. Using a fresh row avoids mutating the shared cached pointer.
func (s *Service) mutate(ctx context.Context, guildID string, fn func(*Settings)) error {
	set, err := s.repo.getSettings(ctx, pid(guildID))
	if err != nil {
		return err
	}
	fn(set)
	if err := s.repo.saveSettings(ctx, set); err != nil {
		return err
	}
	s.invalidate(pid(guildID))
	return nil
}

// SaveSettings persists a full settings row and invalidates the cache. Used by
// the web dashboard's partial-patch path.
func (s *Service) SaveSettings(ctx context.Context, set *Settings) error {
	if err := s.repo.saveSettings(ctx, set); err != nil {
		return err
	}
	s.invalidate(set.GuildID)
	return nil
}

func (s *Service) SetLogChannel(ctx context.Context, guildID string, channelID int64) error {
	return s.mutate(ctx, guildID, func(set *Settings) { set.LogChannelID = channelID })
}

func (s *Service) SetExemptRole(ctx context.Context, guildID string, roleID int64) error {
	return s.mutate(ctx, guildID, func(set *Settings) { set.ExemptRoleID = roleID })
}

func (s *Service) SetTimeout(ctx context.Context, guildID string, secs int) error {
	return s.mutate(ctx, guildID, func(set *Settings) { set.TimeoutSecs = secs })
}

func (s *Service) SetWords(ctx context.Context, guildID string, enabled bool, action string) error {
	return s.mutate(ctx, guildID, func(set *Settings) {
		set.WordsEnabled = enabled
		if action != "" {
			set.WordsAction = action
		}
	})
}

func (s *Service) SetInvites(ctx context.Context, guildID string, enabled bool, action string) error {
	return s.mutate(ctx, guildID, func(set *Settings) {
		set.InvitesEnabled = enabled
		if action != "" {
			set.InvitesAction = action
		}
	})
}

func (s *Service) SetMentions(ctx context.Context, guildID string, enabled bool, threshold int, action string) error {
	return s.mutate(ctx, guildID, func(set *Settings) {
		set.MentionsEnabled = enabled
		if threshold > 0 {
			set.MentionThreshold = threshold
		}
		if action != "" {
			set.MentionsAction = action
		}
	})
}

func (s *Service) SetSpam(ctx context.Context, guildID string, enabled bool, count, windowSecs int, action string) error {
	return s.mutate(ctx, guildID, func(set *Settings) {
		set.SpamEnabled = enabled
		if count > 0 {
			set.SpamCount = count
		}
		if windowSecs > 0 {
			set.SpamWindowSecs = windowSecs
		}
		if action != "" {
			set.SpamAction = action
		}
	})
}

// AddWord adds a banned term (already trimmed/lowercased by the caller).
func (s *Service) AddWord(ctx context.Context, guildID string, word string) (bool, error) {
	ok, err := s.repo.addWord(ctx, pid(guildID), word)
	if err == nil {
		s.invalidateWords(pid(guildID))
	}
	return ok, err
}

func (s *Service) RemoveWord(ctx context.Context, guildID string, word string) (bool, error) {
	ok, err := s.repo.removeWord(ctx, pid(guildID), word)
	if err == nil {
		s.invalidateWords(pid(guildID))
	}
	return ok, err
}

func (s *Service) ClearWords(ctx context.Context, guildID string) (int, error) {
	n, err := s.repo.clearWords(ctx, pid(guildID))
	if err == nil {
		s.invalidateWords(pid(guildID))
	}
	return n, err
}

func (s *Service) ListWords(ctx context.Context, guildID string) ([]string, error) {
	return s.repo.listWords(ctx, pid(guildID))
}

// --- message pipeline ---

// violation describes a matched filter and the action to take.
type violation struct {
	filter string
	action string
	detail string
}

// HandleMessage evaluates an inbound message against the guild's filters and
// enforces the first one that matches. It is called from the gateway event
// handler (already panic-guarded).
func (s *Service) HandleMessage(mc *discordgo.MessageCreate) {
	if mc.GuildID == "" || mc.Author == nil || mc.Author.Bot || mc.Author.System {
		return
	}
	if mc.Author.ID == s.botUserID() {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
	defer cancel()

	set, err := s.getSettings(ctx, pid(mc.GuildID))
	if err != nil || !set.AnyEnabled() {
		return
	}
	if s.isExempt(mc, set) {
		return
	}

	v := s.evaluate(ctx, mc, set)
	if v == nil {
		return
	}
	s.enforce(mc, set, v)
	s.recordViolation(ctx, mc, v)
}

// recordViolation appends the enforced action to the audit log. It is
// best-effort: a write failure is logged but never interrupts enforcement.
func (s *Service) recordViolation(ctx context.Context, mc *discordgo.MessageCreate, v *violation) {
	name := ""
	if mc.Author != nil {
		name = mc.Author.Username
	}
	row := &Violation{
		GuildID:   pid(mc.GuildID),
		UserID:    pid(mc.Author.ID),
		UserName:  name,
		ChannelID: pid(mc.ChannelID),
		Filter:    v.filter,
		Action:    v.action,
		Detail:    v.detail,
	}
	if err := s.repo.insertViolation(ctx, row); err != nil {
		s.log.Warn("automod violation log failed", zap.Error(err), zap.String("guild", mc.GuildID))
	}
}

// ListViolations returns a page of the guild's violation log plus the total.
func (s *Service) ListViolations(ctx context.Context, guildID int64, limit, offset int) ([]Violation, int, error) {
	return s.repo.listViolations(ctx, guildID, limit, offset)
}

// evaluate runs the enabled filters in priority order and returns the first hit.
func (s *Service) evaluate(ctx context.Context, mc *discordgo.MessageCreate, set *Settings) *violation {
	content := mc.Content

	if set.WordsEnabled {
		if words, err := s.getWords(ctx, pid(mc.GuildID)); err == nil {
			if hit, ok := matchBannedWord(content, words); ok {
				return &violation{FilterWords, set.WordsAction, "banned word: " + hit}
			}
		}
	}
	if set.InvitesEnabled && hasInvite(content) {
		return &violation{FilterInvites, set.InvitesAction, "invite link"}
	}
	if set.MentionsEnabled {
		n := len(mc.Mentions) + len(mc.MentionRoles)
		if mc.MentionEveryone {
			n++
		}
		if n >= set.MentionThreshold {
			return &violation{FilterMentions, set.MentionsAction, fmt.Sprintf("%d mentions", n)}
		}
	}
	if set.SpamEnabled && s.recordSpam(pid(mc.GuildID), pid(mc.Author.ID), set.SpamCount, set.SpamWindowSecs) {
		return &violation{FilterSpam, set.SpamAction, fmt.Sprintf("%d msgs / %ds", set.SpamCount, set.SpamWindowSecs)}
	}
	return nil
}

// enforce deletes the offending message, applies any escalation and logs it.
func (s *Service) enforce(mc *discordgo.MessageCreate, set *Settings, v *violation) {
	if err := s.deps.Session.ChannelMessageDelete(mc.ChannelID, mc.ID); err != nil {
		s.log.Warn("automod delete failed", zap.Error(err), zap.String("channel", mc.ChannelID))
	}
	if v.action == ActionTimeout {
		until := time.Now().Add(time.Duration(set.TimeoutSecs) * time.Second)
		if err := s.deps.Session.GuildMemberTimeout(mc.GuildID, mc.Author.ID, &until); err != nil {
			s.log.Warn("automod timeout failed", zap.Error(err),
				zap.String("guild", mc.GuildID), zap.String("user", mc.Author.ID))
		}
	}
	if set.LogChannelID != 0 {
		embed := violationEmbed(mc, v, set)
		if _, err := s.deps.Session.ChannelMessageSendEmbed(sid(set.LogChannelID), embed); err != nil {
			s.log.Warn("automod log failed", zap.Error(err), zap.Int64("channel", set.LogChannelID))
		}
	}
}

// isExempt reports whether a message's author bypasses automod: they hold the
// exempt role, or have Manage Messages.
func (s *Service) isExempt(mc *discordgo.MessageCreate, set *Settings) bool {
	member := mc.Member
	if member == nil {
		return false
	}
	if set.ExemptRoleID != 0 {
		want := sid(set.ExemptRoleID)
		for _, r := range member.Roles {
			if r == want {
				return true
			}
		}
	}
	// Populate User so the owner short-circuit in MemberPermissions can fire.
	member.User = mc.Author
	guild := s.guild(mc.GuildID)
	return shared.HasPermission(guild, member, discordgo.PermissionManageMessages)
}

// recordSpam appends now to the member's sliding window, prunes expired entries
// and reports whether the count has reached the configured threshold.
func (s *Service) recordSpam(guildID, userID int64, count, windowSecs int) bool {
	now := time.Now()
	cutoff := now.Add(-time.Duration(windowSecs) * time.Second)
	key := spamKey{guildID, userID}

	s.spamMu.Lock()
	defer s.spamMu.Unlock()

	times := s.spam[key]
	kept := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	kept = append(kept, now)
	s.spam[key] = kept
	return len(kept) >= count
}

func (s *Service) guild(guildID string) *discordgo.Guild {
	if s.deps.Session.State != nil {
		if g, err := s.deps.Session.State.Guild(guildID); err == nil && g != nil {
			return g
		}
	}
	g, _ := s.deps.Session.Guild(guildID)
	return g
}

func (s *Service) botUserID() string {
	if s.deps.Session.State != nil && s.deps.Session.State.User != nil {
		return s.deps.Session.State.User.ID
	}
	return ""
}
