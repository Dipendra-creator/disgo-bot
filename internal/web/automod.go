package web

import (
	"net/http"

	"github.com/dipu-sharma/disgo-bot/shared"
	"go.uber.org/zap"
)

// This file adapts the module's shared.AutoMod seam to JSON routes: a banned-word
// editor and a read-only violation log. Mutations re-check CSRF and append to the
// dashboard audit log. The banned word is carried in the request body (rather than
// the path) so arbitrary terms — including ones with slashes — round-trip safely.

// wordRequest is the add/remove body for a banned term.
type wordRequest struct {
	Word string `json:"word"`
}

// wordsResponse is the editor's list payload, returned by list/add/remove so the
// frontend always re-renders from the authoritative set.
type wordsResponse struct {
	Words []string `json:"words"`
}

// handleAutoModWords serves GET /api/guilds/{id}/automod/words.
func (s *Server) handleAutoModWords(w http.ResponseWriter, r *http.Request, _ *Session, guildID string) {
	if s.automod == nil {
		writeErr(w, http.StatusNotFound, "automod not available")
		return
	}
	gid, ok := parseGuildID(w, guildID)
	if !ok {
		return
	}
	words, err := s.automod.ListWords(r.Context(), gid)
	if err != nil {
		s.log.Warn("automod word list failed", zap.Int64("guild", gid), zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "failed to load banned words")
		return
	}
	writeJSON(w, http.StatusOK, wordsResponse{Words: words})
}

// handleAutoModWordAdd serves POST /api/guilds/{id}/automod/words.
func (s *Server) handleAutoModWordAdd(w http.ResponseWriter, r *http.Request, sess *Session, guildID string) {
	if s.automod == nil {
		writeErr(w, http.StatusNotFound, "automod not available")
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
	var body wordRequest
	if !decodeBody(w, r, &body) {
		return
	}
	words, err := s.automod.AddWord(r.Context(), gid, body.Word)
	if err != nil {
		s.writeSeamErr(w, "add banned word", gid, err)
		return
	}
	s.recordAudit(r.Context(), gid, sess, "automod", map[string]any{"word_add": body.Word})
	writeJSON(w, http.StatusOK, wordsResponse{Words: words})
}

// handleAutoModWordRemove serves DELETE /api/guilds/{id}/automod/words.
func (s *Server) handleAutoModWordRemove(w http.ResponseWriter, r *http.Request, sess *Session, guildID string) {
	if s.automod == nil {
		writeErr(w, http.StatusNotFound, "automod not available")
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
	var body wordRequest
	if !decodeBody(w, r, &body) {
		return
	}
	words, err := s.automod.RemoveWord(r.Context(), gid, body.Word)
	if err != nil {
		s.writeSeamErr(w, "remove banned word", gid, err)
		return
	}
	s.recordAudit(r.Context(), gid, sess, "automod", map[string]any{"word_remove": body.Word})
	writeJSON(w, http.StatusOK, wordsResponse{Words: words})
}

// handleAutoModViolations serves GET /api/guilds/{id}/automod/violations.
func (s *Server) handleAutoModViolations(w http.ResponseWriter, r *http.Request, _ *Session, guildID string) {
	if s.automod == nil {
		writeErr(w, http.StatusNotFound, "automod not available")
		return
	}
	gid, ok := parseGuildID(w, guildID)
	if !ok {
		return
	}
	q := r.URL.Query()
	page, err := s.automod.ListViolations(r.Context(), gid, shared.PageQuery{
		Limit:  atoiOr(q.Get("limit"), 25),
		Offset: atoiOr(q.Get("offset"), 0),
	})
	if err != nil {
		s.log.Warn("automod violation list failed", zap.Int64("guild", gid), zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "failed to load violations")
		return
	}
	writeJSON(w, http.StatusOK, page)
}
