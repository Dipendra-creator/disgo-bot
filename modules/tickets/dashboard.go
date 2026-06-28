package tickets

import (
	"context"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/shared"
)

// Tickets implementation — exposes the ticket browser, claim/close actions and a
// live transcript view to the web dashboard. It delegates to the existing
// Service so the dashboard and in-Discord paths share identical persistence and
// the standard close behaviour (transcript log + channel delete).

var _ shared.Tickets = (*Module)(nil)

// Bounds for a single ticket page and the close reason.
const (
	ticketListLimit = 100
	closeReasonMax  = 512
)

// ListTickets returns a page of a guild's tickets, narrowed by status.
func (m *Module) ListTickets(ctx context.Context, guildID int64, q shared.TicketQuery) (shared.TicketPage, error) {
	limit, offset := clampPage(q.Limit, q.Offset, ticketListLimit)
	rows, total, err := m.svc.ListTickets(ctx, guildID, normalizeStatus(q.Status), limit, offset)
	if err != nil {
		return shared.TicketPage{}, err
	}
	gid := sid(guildID)
	views := make([]shared.TicketView, 0, len(rows))
	for i := range rows {
		views = append(views, m.toTicketView(gid, &rows[i]))
	}
	return shared.TicketPage{Tickets: views, Total: total}, nil
}

// ClaimTicket assigns an open ticket to the acting dashboard user.
func (m *Module) ClaimTicket(ctx context.Context, guildID, ticketID, actorID int64) (shared.TicketView, error) {
	t, err := m.svc.ClaimByID(ctx, guildID, ticketID, actorID)
	if err != nil {
		return shared.TicketView{}, mapTicketErr(err)
	}
	return m.toTicketView(sid(guildID), t), nil
}

// CloseTicket closes a ticket on behalf of the acting dashboard user.
func (m *Module) CloseTicket(ctx context.Context, guildID, ticketID, actorID int64, reason string) (shared.TicketView, error) {
	reason = strings.TrimSpace(reason)
	if len(reason) > closeReasonMax {
		return shared.TicketView{}, shared.UserErr("Close reason must be at most %d characters.", closeReasonMax)
	}
	gid := sid(guildID)
	closer := &discordgo.User{ID: sid(actorID), Username: m.memberName(gid, sid(actorID))}
	t, err := m.svc.CloseByID(ctx, guildID, ticketID, closer, reason)
	if err != nil {
		return shared.TicketView{}, mapTicketErr(err)
	}
	return m.toTicketView(gid, t), nil
}

// Transcript returns the recent messages of a still-open ticket's channel.
func (m *Module) Transcript(ctx context.Context, guildID, ticketID int64) (shared.TicketTranscript, error) {
	t, err := m.svc.TicketByID(ctx, guildID, ticketID)
	if err != nil {
		return shared.TicketTranscript{}, mapTicketErr(err)
	}
	if t.Status == StatusClosed {
		return shared.TicketTranscript{}, shared.UserErr("This ticket is closed; its channel no longer exists.")
	}
	gid := sid(guildID)
	view := m.toTicketView(gid, t)

	lines := make([]shared.TranscriptLine, 0, transcriptScan)
	if m.deps != nil && m.deps.Session != nil {
		msgs, ferr := m.deps.Session.ChannelMessages(sid(t.ChannelID), transcriptScan, "", "", "")
		if ferr != nil {
			return shared.TicketTranscript{}, ferr
		}
		// ChannelMessages returns newest-first; emit chronologically.
		for i := len(msgs) - 1; i >= 0; i-- {
			msg := msgs[i]
			author := "unknown"
			if msg.Author != nil {
				author = msg.Author.Username
			}
			content := msg.Content
			if content == "" && len(msg.Embeds) > 0 {
				content = "[embed]"
			}
			if len(msg.Attachments) > 0 {
				content += " [attachment]"
			}
			lines = append(lines, shared.TranscriptLine{
				Timestamp: msg.Timestamp,
				Author:    author,
				Content:   content,
			})
		}
	}
	return shared.TicketTranscript{Ticket: view, Lines: lines}, nil
}

// toTicketView maps an internal Ticket to its transport-agnostic form, resolving
// member display names best-effort from the gateway cache.
func (m *Module) toTicketView(guildID string, t *Ticket) shared.TicketView {
	v := shared.TicketView{
		ID:          t.ID,
		Number:      t.Number,
		ChannelID:   sid(t.ChannelID),
		OpenerID:    sid(t.OpenerID),
		OpenerName:  m.memberName(guildID, sid(t.OpenerID)),
		Subject:     t.Subject,
		Status:      t.Status,
		CloseReason: t.CloseReason,
		CreatedAt:   t.CreatedAt,
		ClosedAt:    t.ClosedAt,
	}
	if t.ClaimerID != 0 {
		v.ClaimerID = sid(t.ClaimerID)
		v.ClaimerName = m.memberName(guildID, v.ClaimerID)
	}
	return v
}

// mapTicketErr converts the repo's not-found sentinel into a UserError; other
// errors pass through unchanged for the web layer to log as a 500.
func mapTicketErr(err error) error {
	if err == ErrNoTicket {
		return shared.UserErr("Ticket not found.")
	}
	return err
}

// normalizeStatus maps a requested status filter to the repo's vocabulary
// ("active", "closed"), collapsing anything else to "" (any).
func normalizeStatus(s string) string {
	switch s {
	case "active", StatusClosed:
		return s
	default:
		return ""
	}
}

// clampPage normalises a limit/offset window: limit into (0, max], offset >= 0.
func clampPage(limit, offset, max int) (int, int) {
	if limit <= 0 || limit > max {
		limit = max
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
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
