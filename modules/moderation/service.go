package moderation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/shared"
	"go.uber.org/zap"
)

// maxTimeout is Discord's hard ceiling on a member timeout.
const maxTimeout = 28 * 24 * time.Hour

// sweepInterval is how often expired temp-bans are reconciled.
const sweepInterval = time.Minute

// errNotWarn is returned when a non-warning case is passed to DeleteWarning.
var errNotWarn = errors.New("case is not a warning")

// Service holds the moderation business logic: it applies the Discord action,
// persists a case, notifies the user and posts to the mod-log. It is kept
// independent of the interaction layer so it can be unit-tested.
type Service struct {
	deps *shared.Deps
	repo *repo
	log  *zap.Logger
}

// NewService constructs the moderation service.
func NewService(d *shared.Deps) *Service {
	return &Service{deps: d, repo: newRepo(d.DB), log: d.Log}
}

// actionInput carries the common inputs for an applied moderation action.
type actionInput struct {
	GuildID    string
	GuildName  string
	Target     *discordgo.User
	Mod        *discordgo.User
	Reason     string
	Duration   time.Duration // 0 = permanent/none
	DeleteDays int           // ban only: days of messages to prune (0-7)
}

func (s *Service) newCase(action string, in actionInput) *Case {
	c := &Case{
		GuildID:     pid(in.GuildID),
		Action:      action,
		TargetID:    pid(in.Target.ID),
		ModeratorID: pid(in.Mod.ID),
		Reason:      in.Reason,
		Active:      true,
	}
	if in.Duration > 0 {
		c.DurationMS = in.Duration.Milliseconds()
		c.ExpiresAt = time.Now().Add(in.Duration)
	}
	return c
}

// auditReason composes the audit-log string Discord stores against the action.
func auditReason(mod *discordgo.User, reason string) string {
	base := reason
	if base == "" {
		base = "No reason provided"
	}
	if mod != nil {
		return fmt.Sprintf("%s | by %s (%s)", base, mod.String(), mod.ID)
	}
	return base
}

// Ban bans a user (optionally for a limited duration) and records the case.
func (s *Service) Ban(ctx context.Context, in actionInput) (*Case, error) {
	set, err := s.repo.getSettings(ctx, pid(in.GuildID))
	if err != nil {
		return nil, err
	}
	c := s.newCase(ActionBan, in)

	// DM before banning: once banned the user may lose the shared-guild path
	// needed to open a DM channel.
	if set.DMOnAction {
		s.dm(in.GuildName, c, in.Target)
	}
	if err := s.deps.Session.GuildBanCreateWithReason(in.GuildID, in.Target.ID, auditReason(in.Mod, in.Reason), in.DeleteDays); err != nil {
		return nil, fmt.Errorf("ban: %w", err)
	}
	if err := s.repo.createCase(ctx, c); err != nil {
		return nil, err
	}
	s.postModLog(set, c, in.Target, in.Mod)
	return c, nil
}

// Unban lifts a ban and records the case.
func (s *Service) Unban(ctx context.Context, in actionInput) (*Case, error) {
	set, err := s.repo.getSettings(ctx, pid(in.GuildID))
	if err != nil {
		return nil, err
	}
	if err := s.deps.Session.GuildBanDelete(in.GuildID, in.Target.ID); err != nil {
		return nil, fmt.Errorf("unban: %w", err)
	}
	c := s.newCase(ActionUnban, in)
	if err := s.repo.createCase(ctx, c); err != nil {
		return nil, err
	}
	s.postModLog(set, c, in.Target, in.Mod)
	return c, nil
}

// Kick removes a member and records the case.
func (s *Service) Kick(ctx context.Context, in actionInput) (*Case, error) {
	set, err := s.repo.getSettings(ctx, pid(in.GuildID))
	if err != nil {
		return nil, err
	}
	c := s.newCase(ActionKick, in)
	if set.DMOnAction {
		s.dm(in.GuildName, c, in.Target)
	}
	if err := s.deps.Session.GuildMemberDeleteWithReason(in.GuildID, in.Target.ID, auditReason(in.Mod, in.Reason)); err != nil {
		return nil, fmt.Errorf("kick: %w", err)
	}
	if err := s.repo.createCase(ctx, c); err != nil {
		return nil, err
	}
	s.postModLog(set, c, in.Target, in.Mod)
	return c, nil
}

// Timeout applies a Discord communication timeout and records the case.
func (s *Service) Timeout(ctx context.Context, in actionInput) (*Case, error) {
	if in.Duration <= 0 {
		return nil, shared.UserErr("Provide a timeout duration, e.g. 10m or 2h.")
	}
	if in.Duration > maxTimeout {
		return nil, shared.UserErr("Timeouts can't exceed 28 days.")
	}
	set, err := s.repo.getSettings(ctx, pid(in.GuildID))
	if err != nil {
		return nil, err
	}
	until := time.Now().Add(in.Duration)
	if err := s.deps.Session.GuildMemberTimeout(in.GuildID, in.Target.ID, &until); err != nil {
		return nil, fmt.Errorf("timeout: %w", err)
	}
	c := s.newCase(ActionTimeout, in)
	if set.DMOnAction {
		s.dm(in.GuildName, c, in.Target)
	}
	if err := s.repo.createCase(ctx, c); err != nil {
		return nil, err
	}
	s.postModLog(set, c, in.Target, in.Mod)
	return c, nil
}

// Untimeout clears a member's timeout and records the case.
func (s *Service) Untimeout(ctx context.Context, in actionInput) (*Case, error) {
	set, err := s.repo.getSettings(ctx, pid(in.GuildID))
	if err != nil {
		return nil, err
	}
	if err := s.deps.Session.GuildMemberTimeout(in.GuildID, in.Target.ID, nil); err != nil {
		return nil, fmt.Errorf("untimeout: %w", err)
	}
	c := s.newCase(ActionUntimeout, in)
	if err := s.repo.createCase(ctx, c); err != nil {
		return nil, err
	}
	s.postModLog(set, c, in.Target, in.Mod)
	return c, nil
}

// Warn records a warning (no Discord side-effect) and notifies the user.
func (s *Service) Warn(ctx context.Context, in actionInput) (*Case, error) {
	set, err := s.repo.getSettings(ctx, pid(in.GuildID))
	if err != nil {
		return nil, err
	}
	c := s.newCase(ActionWarn, in)
	if err := s.repo.createCase(ctx, c); err != nil {
		return nil, err
	}
	if set.DMOnAction {
		s.dm(in.GuildName, c, in.Target)
	}
	s.postModLog(set, c, in.Target, in.Mod)
	return c, nil
}

// Warnings returns a user's active warnings, newest first.
func (s *Service) Warnings(ctx context.Context, guildID, targetID int64) ([]Case, error) {
	return s.repo.listByTarget(ctx, guildID, targetID, ActionWarn, true)
}

// GetCase fetches a single case by number.
func (s *Service) GetCase(ctx context.Context, guildID, number int64) (*Case, error) {
	return s.repo.getCase(ctx, guildID, number)
}

// ListCases returns a page of a guild's cases (newest first) plus the total
// match count, for the dashboard case browser.
func (s *Service) ListCases(ctx context.Context, guildID, targetID int64, action string, limit, offset int) ([]Case, int, error) {
	return s.repo.listCases(ctx, guildID, targetID, action, limit, offset)
}

// EditReason rewrites a case's reason and returns the updated case.
func (s *Service) EditReason(ctx context.Context, guildID, number int64, reason string) (*Case, error) {
	if err := s.repo.updateReason(ctx, guildID, number, reason); err != nil {
		return nil, err
	}
	return s.repo.getCase(ctx, guildID, number)
}

// DeleteWarning deactivates an active warning case.
func (s *Service) DeleteWarning(ctx context.Context, guildID, number int64) (*Case, error) {
	c, err := s.repo.getCase(ctx, guildID, number)
	if err != nil {
		return nil, err
	}
	if c.Action != ActionWarn {
		return nil, errNotWarn
	}
	if !c.Active {
		return c, nil
	}
	if err := s.repo.deactivate(ctx, guildID, number); err != nil {
		return nil, err
	}
	c.Active = false
	return c, nil
}

// SetModLog points the guild's mod-log at a channel.
func (s *Service) SetModLog(ctx context.Context, guildID, channelID int64) error {
	return s.repo.setModLogChannel(ctx, guildID, channelID)
}

// Settings returns a guild's moderation configuration (defaults when unset).
func (s *Service) Settings(ctx context.Context, guildID string) (*Settings, error) {
	return s.repo.getSettings(ctx, pid(guildID))
}

// SaveSettings upserts the guild's moderation configuration. Used by the web
// dashboard's partial-patch path.
func (s *Service) SaveSettings(ctx context.Context, set *Settings) error {
	return s.repo.saveSettings(ctx, set)
}

// Purge bulk-deletes up to count recent messages in a channel, optionally
// limited to a single author. Messages older than 14 days (which Discord
// refuses to bulk-delete) are skipped. Returns the number deleted.
func (s *Service) Purge(channelID string, count int, filterUserID string) (int, error) {
	msgs, err := s.deps.Session.ChannelMessages(channelID, count, "", "", "")
	if err != nil {
		return 0, fmt.Errorf("fetch messages: %w", err)
	}
	cutoff := time.Now().Add(-14 * 24 * time.Hour)
	ids := make([]string, 0, len(msgs))
	for _, m := range msgs {
		if filterUserID != "" && (m.Author == nil || m.Author.ID != filterUserID) {
			continue
		}
		if m.Timestamp.Before(cutoff) {
			continue
		}
		ids = append(ids, m.ID)
	}
	switch {
	case len(ids) == 0:
		return 0, nil
	case len(ids) == 1:
		if err := s.deps.Session.ChannelMessageDelete(channelID, ids[0]); err != nil {
			return 0, err
		}
		return 1, nil
	default:
		if err := s.deps.Session.ChannelMessagesBulkDelete(channelID, ids); err != nil {
			return 0, err
		}
		return len(ids), nil
	}
}

// dm best-effort delivers a private notice to the affected user; failures
// (closed DMs, no mutual guild) are logged at debug and otherwise ignored.
func (s *Service) dm(guildName string, c *Case, target *discordgo.User) {
	if target == nil {
		return
	}
	ch, err := s.deps.Session.UserChannelCreate(target.ID)
	if err != nil {
		s.log.Debug("dm channel create failed", zap.Error(err), zap.String("user", target.ID))
		return
	}
	if _, err := s.deps.Session.ChannelMessageSendEmbed(ch.ID, dmEmbed(guildName, c)); err != nil {
		s.log.Debug("dm send failed", zap.Error(err), zap.String("user", target.ID))
	}
}

// postModLog publishes a case to the configured mod-log channel, if any.
func (s *Service) postModLog(set *Settings, c *Case, target, mod *discordgo.User) {
	if set == nil || set.ModLogChannelID == 0 {
		return
	}
	if _, err := s.deps.Session.ChannelMessageSendEmbed(sid(set.ModLogChannelID), caseEmbed(c, target, mod)); err != nil {
		s.log.Warn("post mod-log failed", zap.Error(err), zap.Int64("channel", set.ModLogChannelID))
	}
}

// runTempbanSweeper periodically lifts expired temp-bans until ctx is canceled.
func (s *Service) runTempbanSweeper(ctx context.Context) {
	ticker := time.NewTicker(sweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sweepTempbans(ctx)
		}
	}
}

func (s *Service) sweepTempbans(ctx context.Context) {
	cases, err := s.repo.dueTempbans(ctx, time.Now())
	if err != nil {
		s.log.Warn("tempban sweep query failed", zap.Error(err))
		return
	}
	for i := range cases {
		c := &cases[i]
		guildID, userID := sid(c.GuildID), sid(c.TargetID)

		if err := s.deps.Session.GuildBanDelete(guildID, userID); err != nil {
			// Log but still deactivate so a permanently-failing case (e.g.
			// already manually unbanned) doesn't loop every tick.
			s.log.Warn("tempban auto-unban failed",
				zap.Error(err), zap.String("guild", guildID), zap.String("user", userID))
		}
		if err := s.repo.deactivate(ctx, c.GuildID, c.CaseNumber); err != nil {
			s.log.Warn("tempban deactivate failed", zap.Error(err), zap.Int64("case", c.CaseNumber))
			continue
		}

		set, err := s.repo.getSettings(ctx, c.GuildID)
		if err != nil {
			continue
		}
		unban := &Case{
			GuildID:  c.GuildID,
			Action:   ActionUnban,
			TargetID: c.TargetID,
			Reason:   "Temp-ban expired",
			Active:   true,
		}
		if err := s.repo.createCase(ctx, unban); err != nil {
			s.log.Warn("tempban expiry case failed", zap.Error(err))
			continue
		}
		target, _ := s.deps.Session.User(userID)
		s.postModLog(set, unban, target, nil)
	}
}
