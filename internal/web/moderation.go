package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/dipu-sharma/disgo-bot/shared"
	"go.uber.org/zap"
)

// This file adapts the module's shared.Moderation seam to JSON routes: a case
// browser, an edit-reason endpoint, and a manual-action endpoint. Mutations
// re-check CSRF and append to the dashboard audit log. The acting moderator is
// always the session user — never trusted from the request body.

// reasonMaxLen bounds a case reason / action reason from the dashboard.
const reasonMaxLen = 1000

// actionRequest is the POST body for a manual moderation action. ModID/ModName
// are intentionally absent: they come from the session, not the client.
type actionRequest struct {
	Action     string `json:"action"`
	TargetID   string `json:"target_id"`
	Reason     string `json:"reason"`
	DurationMS int64  `json:"duration_ms"`
}

// reasonRequest is the PATCH body for editing a case reason.
type reasonRequest struct {
	Reason string `json:"reason"`
}

// handleModCases serves GET /api/guilds/{id}/moderation/cases.
func (s *Server) handleModCases(w http.ResponseWriter, r *http.Request, _ *Session, guildID string) {
	if s.moderation == nil {
		writeErr(w, http.StatusNotFound, "moderation not available")
		return
	}
	gid, err := strconv.ParseInt(guildID, 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid guild id")
		return
	}
	q := r.URL.Query()
	target := strings.TrimSpace(q.Get("target"))
	if target != "" {
		if _, err := strconv.ParseInt(target, 10, 64); err != nil {
			writeErr(w, http.StatusBadRequest, "target must be a user id")
			return
		}
	}
	page, err := s.moderation.ListCases(r.Context(), gid, shared.ModCaseQuery{
		TargetID: target,
		Action:   q.Get("action"),
		Limit:    atoiOr(q.Get("limit"), 25),
		Offset:   atoiOr(q.Get("offset"), 0),
	})
	if err != nil {
		s.log.Warn("list cases failed", zap.Int64("guild", gid), zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "failed to list cases")
		return
	}
	writeJSON(w, http.StatusOK, page)
}

// handleModCaseReason serves PATCH /api/guilds/{id}/moderation/cases/{num}.
func (s *Server) handleModCaseReason(w http.ResponseWriter, r *http.Request, sess *Session, guildID string) {
	if s.moderation == nil {
		writeErr(w, http.StatusNotFound, "moderation not available")
		return
	}
	if !s.checkCSRF(r) {
		writeErr(w, http.StatusForbidden, "bad origin")
		return
	}
	gid, err := strconv.ParseInt(guildID, 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid guild id")
		return
	}
	num, err := strconv.ParseInt(r.PathValue("num"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid case number")
		return
	}
	var body reasonRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	reason := strings.TrimSpace(body.Reason)
	if reason == "" || len(reason) > reasonMaxLen {
		writeErr(w, http.StatusBadRequest, "reason must be 1–1000 characters")
		return
	}
	c, err := s.moderation.EditCaseReason(r.Context(), gid, num, reason)
	if err != nil {
		s.writeModErr(w, "edit case reason", gid, err)
		return
	}
	s.recordAudit(r.Context(), gid, sess, "moderation", map[string]any{
		"edit_reason": num,
	})
	writeJSON(w, http.StatusOK, c)
}

// handleModAction serves POST /api/guilds/{id}/moderation/actions.
func (s *Server) handleModAction(w http.ResponseWriter, r *http.Request, sess *Session, guildID string) {
	if s.moderation == nil {
		writeErr(w, http.StatusNotFound, "moderation not available")
		return
	}
	if !s.checkCSRF(r) {
		writeErr(w, http.StatusForbidden, "bad origin")
		return
	}
	gid, err := strconv.ParseInt(guildID, 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid guild id")
		return
	}
	var body actionRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if len(body.Reason) > reasonMaxLen {
		writeErr(w, http.StatusBadRequest, "reason too long")
		return
	}
	c, err := s.moderation.ApplyAction(r.Context(), body.Action, shared.ModAction{
		GuildID:    guildID,
		TargetID:   strings.TrimSpace(body.TargetID),
		ModID:      sess.UserID,
		ModName:    sess.Username,
		Reason:     strings.TrimSpace(body.Reason),
		DurationMS: body.DurationMS,
	})
	if err != nil {
		s.writeModErr(w, "apply action", gid, err)
		return
	}
	s.recordAudit(r.Context(), gid, sess, "moderation", map[string]any{
		"action":    body.Action,
		"target_id": strings.TrimSpace(body.TargetID),
		"case":      c.Number,
	})
	writeJSON(w, http.StatusOK, c)
}

// writeModErr maps a moderation error to a response: UserError → 400 with its
// message, anything else → logged 500.
func (s *Server) writeModErr(w http.ResponseWriter, op string, gid int64, err error) {
	if ue, ok := shared.AsUserError(err); ok {
		writeErr(w, http.StatusBadRequest, ue.Msg)
		return
	}
	s.log.Warn(op+" failed", zap.Int64("guild", gid), zap.Error(err))
	writeErr(w, http.StatusInternalServerError, "moderation request failed")
}

// atoiOr parses s as an int, returning def on failure or empty input.
func atoiOr(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
