package web

import (
	"net/http"
	"strconv"

	"github.com/dipu-sharma/disgo-bot/shared"
	"go.uber.org/zap"
)

// This file adapts the module's shared.Tickets seam to JSON routes: a paginated
// ticket browser, claim/close actions and a live transcript view. Mutations
// re-check CSRF and append to the dashboard audit log; the acting user (claimer /
// closer) is always the server-side session user, never the request body.

// closeTicketRequest is the body for closing a ticket.
type closeTicketRequest struct {
	Reason string `json:"reason"`
}

// handleTickets serves GET /api/guilds/{id}/tickets.
func (s *Server) handleTickets(w http.ResponseWriter, r *http.Request, _ *Session, guildID string) {
	if s.tickets == nil {
		writeErr(w, http.StatusNotFound, "tickets not available")
		return
	}
	gid, ok := parseGuildID(w, guildID)
	if !ok {
		return
	}
	q := r.URL.Query()
	page, err := s.tickets.ListTickets(r.Context(), gid, shared.TicketQuery{
		Status: q.Get("status"),
		Limit:  atoiOr(q.Get("limit"), 25),
		Offset: atoiOr(q.Get("offset"), 0),
	})
	if err != nil {
		s.log.Warn("ticket list failed", zap.Int64("guild", gid), zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "failed to load tickets")
		return
	}
	writeJSON(w, http.StatusOK, page)
}

// handleTicketTranscript serves GET /api/guilds/{id}/tickets/{ticket}/transcript.
func (s *Server) handleTicketTranscript(w http.ResponseWriter, r *http.Request, _ *Session, guildID string) {
	if s.tickets == nil {
		writeErr(w, http.StatusNotFound, "tickets not available")
		return
	}
	gid, ok := parseGuildID(w, guildID)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("ticket"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid ticket id")
		return
	}
	tr, err := s.tickets.Transcript(r.Context(), gid, id)
	if err != nil {
		s.writeSeamErr(w, "ticket transcript", gid, err)
		return
	}
	writeJSON(w, http.StatusOK, tr)
}

// handleTicketClaim serves POST /api/guilds/{id}/tickets/{ticket}/claim.
func (s *Server) handleTicketClaim(w http.ResponseWriter, r *http.Request, sess *Session, guildID string) {
	if s.tickets == nil {
		writeErr(w, http.StatusNotFound, "tickets not available")
		return
	}
	if !s.checkCSRF(r) {
		writeErr(w, http.StatusForbidden, "bad origin")
		return
	}
	gid, ok := parseGuildID(w, guildID)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("ticket"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid ticket id")
		return
	}
	view, err := s.tickets.ClaimTicket(r.Context(), gid, id, sessUID(sess))
	if err != nil {
		s.writeSeamErr(w, "claim ticket", gid, err)
		return
	}
	s.recordAudit(r.Context(), gid, sess, "tickets", map[string]any{"claim": id})
	writeJSON(w, http.StatusOK, view)
}

// handleTicketClose serves POST /api/guilds/{id}/tickets/{ticket}/close.
func (s *Server) handleTicketClose(w http.ResponseWriter, r *http.Request, sess *Session, guildID string) {
	if s.tickets == nil {
		writeErr(w, http.StatusNotFound, "tickets not available")
		return
	}
	if !s.checkCSRF(r) {
		writeErr(w, http.StatusForbidden, "bad origin")
		return
	}
	gid, ok := parseGuildID(w, guildID)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("ticket"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid ticket id")
		return
	}
	var body closeTicketRequest
	if !decodeBody(w, r, &body) {
		return
	}
	view, err := s.tickets.CloseTicket(r.Context(), gid, id, sessUID(sess), body.Reason)
	if err != nil {
		s.writeSeamErr(w, "close ticket", gid, err)
		return
	}
	s.recordAudit(r.Context(), gid, sess, "tickets", map[string]any{"close": id})
	writeJSON(w, http.StatusOK, view)
}

// sessUID parses the session user's snowflake into the int64 the seams use.
func sessUID(sess *Session) int64 {
	n, _ := strconv.ParseInt(sess.UserID, 10, 64)
	return n
}
