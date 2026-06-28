package shared

import (
	"context"
	"time"
)

// Tickets is an optional contract a Module implements to expose its support
// tickets to the web dashboard: a paginated ticket browser, claim/close actions,
// and a live transcript view. Like Configurable and Moderation it is a purely
// additive, transport-agnostic seam — the web layer type-asserts each registered
// Module to Tickets, and modules that don't implement it simply aren't surfaced
// as a management console. Implementations never import net/http and exchange
// only the plain types defined here.
type Tickets interface {
	// ListTickets returns a page of a guild's tickets, newest first, narrowed by
	// the query's Status filter and bounded by its Limit/Offset.
	ListTickets(ctx context.Context, guildID int64, q TicketQuery) (TicketPage, error)
	// ClaimTicket assigns an open ticket to the acting dashboard user and returns
	// the updated ticket. It returns a UserError when the ticket is missing,
	// already claimed, or closed.
	ClaimTicket(ctx context.Context, guildID, ticketID, actorID int64) (TicketView, error)
	// CloseTicket closes a ticket on behalf of the acting dashboard user: it logs
	// a transcript (when a log channel is configured), marks the ticket closed and
	// deletes its channel. It returns a UserError when the ticket is missing or
	// already closed.
	CloseTicket(ctx context.Context, guildID, ticketID, actorID int64, reason string) (TicketView, error)
	// Transcript returns the recent messages of a still-open ticket's channel for
	// in-dashboard review. It returns a UserError once the ticket is closed (its
	// channel no longer exists).
	Transcript(ctx context.Context, guildID, ticketID int64) (TicketTranscript, error)
}

// TicketQuery narrows a ticket listing. Status is one of "active" (open or
// claimed), "closed", or "" (any); Limit/Offset drive pagination (the
// implementation clamps unreasonable values).
type TicketQuery struct {
	Status string
	Limit  int
	Offset int
}

// TicketView is one support ticket in the dashboard's transport-agnostic form.
// IDs are strings to preserve snowflake precision across JSON.
type TicketView struct {
	ID          int64     `json:"id"`
	Number      int64     `json:"number"`
	ChannelID   string    `json:"channel_id"`
	OpenerID    string    `json:"opener_id"`
	OpenerName  string    `json:"opener_name,omitempty"` // best-effort from the gateway cache
	ClaimerID   string    `json:"claimer_id"`            // "" = unclaimed
	ClaimerName string    `json:"claimer_name,omitempty"`
	Subject     string    `json:"subject"`
	Status      string    `json:"status"`
	CloseReason string    `json:"close_reason,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	ClosedAt    time.Time `json:"closed_at,omitempty"` // zero when still open
}

// TicketPage is one page of tickets together with the total number of matches
// (ignoring Limit/Offset) so the dashboard can paginate.
type TicketPage struct {
	Tickets []TicketView `json:"tickets"`
	Total   int          `json:"total"`
}

// TranscriptLine is one rendered message in a ticket transcript.
type TranscriptLine struct {
	Timestamp time.Time `json:"timestamp"`
	Author    string    `json:"author"`
	Content   string    `json:"content"`
}

// TicketTranscript is a ticket plus the recent messages from its channel.
type TicketTranscript struct {
	Ticket TicketView       `json:"ticket"`
	Lines  []TranscriptLine `json:"lines"`
}
