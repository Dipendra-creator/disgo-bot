package web

import (
	"net/http"
	"strconv"

	"github.com/dipu-sharma/disgo-bot/shared"
	"go.uber.org/zap"
)

// This file adapts the module's shared.Leveling seam to JSON routes: an XP
// leaderboard and level→role reward management. Reward mutations re-check CSRF
// and append to the dashboard audit log.

// rewardRoleRequest is the body for setting a level reward role.
type rewardRoleRequest struct {
	RoleID string `json:"role_id"`
}

// handleLevelLeaderboard serves GET /api/guilds/{id}/leveling/leaderboard.
func (s *Server) handleLevelLeaderboard(w http.ResponseWriter, r *http.Request, _ *Session, guildID string) {
	if s.leveling == nil {
		writeErr(w, http.StatusNotFound, "leveling not available")
		return
	}
	gid, ok := parseGuildID(w, guildID)
	if !ok {
		return
	}
	q := r.URL.Query()
	page, err := s.leveling.XPLeaderboard(r.Context(), gid, shared.PageQuery{
		Limit:  atoiOr(q.Get("limit"), 25),
		Offset: atoiOr(q.Get("offset"), 0),
	})
	if err != nil {
		s.log.Warn("leveling leaderboard failed", zap.Int64("guild", gid), zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "failed to load leaderboard")
		return
	}
	writeJSON(w, http.StatusOK, page)
}

// handleLevelRewards serves GET /api/guilds/{id}/leveling/rewards.
func (s *Server) handleLevelRewards(w http.ResponseWriter, r *http.Request, _ *Session, guildID string) {
	if s.leveling == nil {
		writeErr(w, http.StatusNotFound, "leveling not available")
		return
	}
	gid, ok := parseGuildID(w, guildID)
	if !ok {
		return
	}
	rewards, err := s.leveling.ListRewards(r.Context(), gid)
	if err != nil {
		s.log.Warn("leveling rewards failed", zap.Int64("guild", gid), zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "failed to load rewards")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rewards": rewards})
}

// handleLevelRewardSet serves PUT /api/guilds/{id}/leveling/rewards/{level}.
func (s *Server) handleLevelRewardSet(w http.ResponseWriter, r *http.Request, sess *Session, guildID string) {
	if s.leveling == nil {
		writeErr(w, http.StatusNotFound, "leveling not available")
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
	level, err := strconv.Atoi(r.PathValue("level"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid level")
		return
	}
	var body rewardRoleRequest
	if !decodeBody(w, r, &body) {
		return
	}
	if err := s.leveling.SetReward(r.Context(), gid, level, body.RoleID); err != nil {
		s.writeSeamErr(w, "set level reward", gid, err)
		return
	}
	s.recordAudit(r.Context(), gid, sess, "leveling", map[string]any{
		"reward_set": level, "role_id": body.RoleID,
	})
	w.WriteHeader(http.StatusNoContent)
}

// handleLevelRewardDelete serves DELETE /api/guilds/{id}/leveling/rewards/{level}.
func (s *Server) handleLevelRewardDelete(w http.ResponseWriter, r *http.Request, sess *Session, guildID string) {
	if s.leveling == nil {
		writeErr(w, http.StatusNotFound, "leveling not available")
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
	level, err := strconv.Atoi(r.PathValue("level"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid level")
		return
	}
	if err := s.leveling.RemoveReward(r.Context(), gid, level); err != nil {
		s.writeSeamErr(w, "remove level reward", gid, err)
		return
	}
	s.recordAudit(r.Context(), gid, sess, "leveling", map[string]any{"reward_remove": level})
	w.WriteHeader(http.StatusNoContent)
}
