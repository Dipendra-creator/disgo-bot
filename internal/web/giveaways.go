package web

import (
	"net/http"
	"strconv"

	"github.com/dipu-sharma/disgo-bot/shared"
	"go.uber.org/zap"
)

// This file adapts the module's shared.Giveaways seam to JSON routes: a giveaway
// manager that lists, creates, ends and rerolls draws. Mutations re-check CSRF
// and append to the dashboard audit log; the host of a created giveaway is always
// the server-side session user, never the request body.

// createGiveawayRequest is the body for starting a giveaway.
type createGiveawayRequest struct {
	ChannelID  string `json:"channel_id"`
	Prize      string `json:"prize"`
	DurationMS int64  `json:"duration_ms"`
	Winners    int    `json:"winners"`
}

func (b createGiveawayRequest) input() shared.GiveawayInput {
	return shared.GiveawayInput{
		ChannelID:  b.ChannelID,
		Prize:      b.Prize,
		DurationMS: b.DurationMS,
		Winners:    b.Winners,
	}
}

// rerollRequest is the body for rerolling a giveaway (winners <= 0 reuses count).
type rerollRequest struct {
	Winners int `json:"winners"`
}

// handleGiveaways serves GET /api/guilds/{id}/giveaways.
func (s *Server) handleGiveaways(w http.ResponseWriter, r *http.Request, _ *Session, guildID string) {
	if s.giveaways == nil {
		writeErr(w, http.StatusNotFound, "giveaways not available")
		return
	}
	gid, ok := parseGuildID(w, guildID)
	if !ok {
		return
	}
	q := r.URL.Query()
	page, err := s.giveaways.ListGiveaways(r.Context(), gid, shared.PageQuery{
		Limit:  atoiOr(q.Get("limit"), 25),
		Offset: atoiOr(q.Get("offset"), 0),
	})
	if err != nil {
		s.log.Warn("giveaway list failed", zap.Int64("guild", gid), zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "failed to load giveaways")
		return
	}
	writeJSON(w, http.StatusOK, page)
}

// handleGiveawayCreate serves POST /api/guilds/{id}/giveaways.
func (s *Server) handleGiveawayCreate(w http.ResponseWriter, r *http.Request, sess *Session, guildID string) {
	if s.giveaways == nil {
		writeErr(w, http.StatusNotFound, "giveaways not available")
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
	var body createGiveawayRequest
	if !decodeBody(w, r, &body) {
		return
	}
	view, err := s.giveaways.CreateGiveaway(r.Context(), gid, body.input(), sessUID(sess))
	if err != nil {
		s.writeSeamErr(w, "create giveaway", gid, err)
		return
	}
	s.recordAudit(r.Context(), gid, sess, "giveaways", map[string]any{
		"create": view.ID, "prize": view.Prize,
	})
	writeJSON(w, http.StatusOK, view)
}

// handleGiveawayEnd serves POST /api/guilds/{id}/giveaways/{gw}/end.
func (s *Server) handleGiveawayEnd(w http.ResponseWriter, r *http.Request, sess *Session, guildID string) {
	if s.giveaways == nil {
		writeErr(w, http.StatusNotFound, "giveaways not available")
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
	id, err := strconv.ParseInt(r.PathValue("gw"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid giveaway id")
		return
	}
	view, err := s.giveaways.EndGiveaway(r.Context(), gid, id)
	if err != nil {
		s.writeSeamErr(w, "end giveaway", gid, err)
		return
	}
	s.recordAudit(r.Context(), gid, sess, "giveaways", map[string]any{"end": id})
	writeJSON(w, http.StatusOK, view)
}

// handleGiveawayReroll serves POST /api/guilds/{id}/giveaways/{gw}/reroll.
func (s *Server) handleGiveawayReroll(w http.ResponseWriter, r *http.Request, sess *Session, guildID string) {
	if s.giveaways == nil {
		writeErr(w, http.StatusNotFound, "giveaways not available")
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
	id, err := strconv.ParseInt(r.PathValue("gw"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid giveaway id")
		return
	}
	var body rerollRequest
	if !decodeBody(w, r, &body) {
		return
	}
	view, err := s.giveaways.RerollGiveaway(r.Context(), gid, id, body.Winners)
	if err != nil {
		s.writeSeamErr(w, "reroll giveaway", gid, err)
		return
	}
	s.recordAudit(r.Context(), gid, sess, "giveaways", map[string]any{"reroll": id})
	writeJSON(w, http.StatusOK, view)
}
