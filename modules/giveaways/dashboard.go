package giveaways

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/shared"
)

// Giveaways implementation — exposes the giveaway manager (list/create/end/
// reroll) to the web dashboard. It delegates to the existing Service so the
// dashboard and in-Discord paths share identical persistence, panels and the
// winner-draw logic.

var _ shared.Giveaways = (*Module)(nil)

// giveawayListLimit bounds a single page of the manager.
const giveawayListLimit = 100

// ListGiveaways returns a page of a guild's giveaways with per-row entry counts.
func (m *Module) ListGiveaways(ctx context.Context, guildID int64, q shared.PageQuery) (shared.GiveawayPage, error) {
	limit, offset := clampPage(q, giveawayListLimit)
	rows, total, err := m.svc.List(ctx, guildID, limit, offset)
	if err != nil {
		return shared.GiveawayPage{}, err
	}
	gid := sid(guildID)
	views := make([]shared.GiveawayView, 0, len(rows))
	for i := range rows {
		g := &rows[i]
		entries, err := m.svc.EntryCount(ctx, g.ID)
		if err != nil {
			return shared.GiveawayPage{}, err
		}
		views = append(views, m.toGiveawayView(gid, g, entries))
	}
	return shared.GiveawayPage{Giveaways: views, Total: total}, nil
}

// CreateGiveaway starts a giveaway hosted by the acting dashboard user.
func (m *Module) CreateGiveaway(ctx context.Context, guildID int64, in shared.GiveawayInput, hostID int64) (shared.GiveawayView, error) {
	prize, dur, winners, err := validateGiveaway(in)
	if err != nil {
		return shared.GiveawayView{}, err
	}
	gid := sid(guildID)
	host := &discordgo.User{ID: sid(hostID), Username: m.memberName(gid, sid(hostID))}
	g, err := m.svc.Create(ctx, gid, in.ChannelID, prize, dur, winners, host)
	if err != nil {
		return shared.GiveawayView{}, mapGiveErr(err)
	}
	return m.toGiveawayView(gid, g, 0), nil
}

// EndGiveaway closes a running giveaway immediately and draws its winners.
func (m *Module) EndGiveaway(ctx context.Context, guildID, giveawayID int64) (shared.GiveawayView, error) {
	gid := sid(guildID)
	g, _, err := m.svc.End(ctx, gid, giveawayID)
	if err != nil {
		return shared.GiveawayView{}, mapGiveErr(err)
	}
	entries, _ := m.svc.EntryCount(ctx, giveawayID)
	return m.toGiveawayView(gid, g, entries), nil
}

// RerollGiveaway draws fresh winners for an already-ended giveaway.
func (m *Module) RerollGiveaway(ctx context.Context, guildID, giveawayID int64, winners int) (shared.GiveawayView, error) {
	gid := sid(guildID)
	g, _, err := m.svc.Reroll(ctx, gid, giveawayID, winners)
	if err != nil {
		return shared.GiveawayView{}, mapGiveErr(err)
	}
	entries, _ := m.svc.EntryCount(ctx, giveawayID)
	return m.toGiveawayView(gid, g, entries), nil
}

// toGiveawayView maps an internal Giveaway to its transport-agnostic form,
// resolving the host's display name best-effort from the gateway cache.
func (m *Module) toGiveawayView(guildID string, g *Giveaway, entries int) shared.GiveawayView {
	winners := g.winnerList()
	if winners == nil {
		winners = []string{}
	}
	return shared.GiveawayView{
		ID:        g.ID,
		ChannelID: sid(g.ChannelID),
		Prize:     g.Prize,
		Winners:   g.Winners,
		HostID:    sid(g.HostID),
		HostName:  m.memberName(guildID, sid(g.HostID)),
		Entries:   entries,
		Ended:     g.Ended,
		WinnerIDs: winners,
		EndsAt:    g.EndsAt,
		CreatedAt: g.CreatedAt,
	}
}

// validateGiveaway normalises and bounds a create payload, returning a UserError
// for anything invalid.
func validateGiveaway(in shared.GiveawayInput) (prize string, dur time.Duration, winners int, err error) {
	if !isSnowflake(strings.TrimSpace(in.ChannelID)) {
		return "", 0, 0, shared.UserErr("A valid channel is required.")
	}
	prize = strings.TrimSpace(in.Prize)
	if prize == "" {
		return "", 0, 0, shared.UserErr("Prize is required.")
	}
	if len(prize) > maxPrizeLen {
		return "", 0, 0, shared.UserErr("Prize must be at most %d characters.", maxPrizeLen)
	}
	winners = in.Winners
	if winners < 1 || winners > maxWinners {
		return "", 0, 0, shared.UserErr("Winners must be between 1 and %d.", maxWinners)
	}
	dur = time.Duration(in.DurationMS) * time.Millisecond
	if dur < minDuration || dur > maxDuration {
		return "", 0, 0, shared.UserErr("Duration must be between 1 minute and 90 days.")
	}
	return prize, dur, winners, nil
}

// mapGiveErr converts the service's sentinels into UserErrors; other errors pass
// through unchanged for the web layer to log as a 500.
func mapGiveErr(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return shared.UserErr("Giveaway not found.")
	case errors.Is(err, ErrEnded):
		return shared.UserErr("This giveaway has already ended.")
	case errors.Is(err, ErrNotEnded):
		return shared.UserErr("This giveaway hasn't ended yet.")
	case errors.Is(err, ErrNoEntries):
		return shared.UserErr("This giveaway had no entries to draw from.")
	default:
		return err
	}
}

// clampPage normalises a PageQuery: limit into (0, max], offset to >= 0.
func clampPage(q shared.PageQuery, max int) (limit, offset int) {
	limit = q.Limit
	if limit <= 0 || limit > max {
		limit = max
	}
	offset = q.Offset
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

// isSnowflake reports whether s is a positive integer id.
func isSnowflake(s string) bool {
	n, err := strconv.ParseInt(s, 10, 64)
	return err == nil && n > 0
}

// memberName returns a display name for a member from the gateway cache, or ""
// when uncached / the session isn't ready. It never makes a Discord REST call.
func (m *Module) memberName(guildID, userID string) string {
	if m.deps == nil || m.deps.Session == nil || m.deps.Session.State == nil {
		return ""
	}
	mem, err := m.deps.Session.State.Member(guildID, userID)
	if err != nil || mem == nil {
		return ""
	}
	if mem.Nick != "" {
		return mem.Nick
	}
	if mem.User != nil {
		return mem.User.Username
	}
	return ""
}
