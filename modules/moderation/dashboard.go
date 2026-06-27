package moderation

import (
	"context"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/shared"
)

// Moderation implementation — exposes the case history and manual actions to the
// web dashboard. It delegates to the existing Service so the dashboard path and
// the slash-command path share identical persistence, DM and mod-log behaviour.

var _ shared.Moderation = (*Module)(nil)

// caseListLimit bounds a single page of the case browser.
const caseListLimit = 100

// toModCase maps an internal Case to the transport-agnostic shared.ModCase.
func toModCase(c *Case) shared.ModCase {
	mc := shared.ModCase{
		Number:      c.CaseNumber,
		Action:      c.Action,
		TargetID:    sid(c.TargetID),
		Reason:      c.Reason,
		Active:      c.Active,
		DurationMS:  c.DurationMS,
		CreatedAt:   c.CreatedAt,
		ModeratorID: idString(c.ModeratorID),
	}
	if !c.ExpiresAt.IsZero() {
		mc.ExpiresAt = c.ExpiresAt
	}
	return mc
}

// ListCases returns a page of the guild's cases for the dashboard browser.
func (m *Module) ListCases(ctx context.Context, guildID int64, q shared.ModCaseQuery) (shared.ModCasePage, error) {
	limit := q.Limit
	if limit <= 0 || limit > caseListLimit {
		limit = caseListLimit
	}
	offset := q.Offset
	if offset < 0 {
		offset = 0
	}
	cases, total, err := m.svc.ListCases(ctx, guildID, pid(q.TargetID), q.Action, limit, offset)
	if err != nil {
		return shared.ModCasePage{}, err
	}
	out := make([]shared.ModCase, 0, len(cases))
	for i := range cases {
		out = append(out, toModCase(&cases[i]))
	}
	return shared.ModCasePage{Cases: out, Total: total}, nil
}

// EditCaseReason rewrites a case's reason via the service.
func (m *Module) EditCaseReason(ctx context.Context, guildID, number int64, reason string) (shared.ModCase, error) {
	c, err := m.svc.EditReason(ctx, guildID, number, reason)
	if err != nil {
		if err == ErrCaseNotFound {
			return shared.ModCase{}, shared.UserErr("Case #%d not found.", number)
		}
		return shared.ModCase{}, err
	}
	return toModCase(c), nil
}

// ApplyAction performs a manual ban/kick/timeout/warn requested from the
// dashboard. The acting dashboard user is recorded as the case moderator. Guard
// rules block self-, owner- and bot-targeting before any Discord call; the
// Discord API still enforces bot-vs-target role hierarchy on top.
func (m *Module) ApplyAction(ctx context.Context, action string, a shared.ModAction) (shared.ModCase, error) {
	switch action {
	case ActionBan, ActionKick, ActionTimeout, ActionWarn:
	default:
		return shared.ModCase{}, shared.UserErr("Unsupported action %q.", action)
	}
	if !isSnowflake(a.TargetID) {
		return shared.ModCase{}, shared.UserErr("A valid target user ID is required.")
	}
	if a.TargetID == a.ModID {
		return shared.ModCase{}, shared.UserErr("You can't action yourself.")
	}

	guildName := "the server"
	if g := m.stateGuild(a.GuildID); g != nil {
		if g.Name != "" {
			guildName = g.Name
		}
		if a.TargetID == g.OwnerID {
			return shared.ModCase{}, shared.UserErr("You can't action the server owner.")
		}
	}
	if a.TargetID == m.selfID() {
		return shared.ModCase{}, shared.UserErr("You can't action the bot.")
	}

	in := actionInput{
		GuildID:   a.GuildID,
		GuildName: guildName,
		Target:    m.resolveUser(a.TargetID),
		Mod:       &discordgo.User{ID: a.ModID, Username: a.ModName},
		Reason:    a.Reason,
		Duration:  time.Duration(a.DurationMS) * time.Millisecond,
	}

	var (
		c   *Case
		err error
	)
	switch action {
	case ActionBan:
		c, err = m.svc.Ban(ctx, in)
	case ActionKick:
		c, err = m.svc.Kick(ctx, in)
	case ActionTimeout:
		c, err = m.svc.Timeout(ctx, in)
	case ActionWarn:
		c, err = m.svc.Warn(ctx, in)
	}
	if err != nil {
		return shared.ModCase{}, err
	}
	return toModCase(c), nil
}

// stateGuild reads a guild from the gateway cache, or nil when unavailable.
func (m *Module) stateGuild(id string) *discordgo.Guild {
	if m.deps == nil || m.deps.Session == nil || m.deps.Session.State == nil {
		return nil
	}
	g, err := m.deps.Session.State.Guild(id)
	if err != nil {
		return nil
	}
	return g
}

// selfID returns the bot user's ID, or "" when the session isn't ready.
func (m *Module) selfID() string {
	if m.deps == nil || m.deps.Session == nil || m.deps.Session.State == nil || m.deps.Session.State.User == nil {
		return ""
	}
	return m.deps.Session.State.User.ID
}

// resolveUser fetches the full user for nicer DM/mod-log embeds, falling back to
// an ID-only stub when the lookup fails or no session is present.
func (m *Module) resolveUser(id string) *discordgo.User {
	if m.deps != nil && m.deps.Session != nil {
		if u, err := m.deps.Session.User(id); err == nil && u != nil {
			return u
		}
	}
	return &discordgo.User{ID: id}
}
