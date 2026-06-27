package web

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/dipu-sharma/disgo-bot/shared"
	"go.uber.org/zap"
)

// meResponse is the payload of GET /api/me.
type meResponse struct {
	UserID   string       `json:"user_id"`
	Username string       `json:"username"`
	Avatar   string       `json:"avatar,omitempty"`
	Guilds   []GuildBrief `json:"guilds"`
}

// moduleConfig is one module's schema plus the guild's current values.
type moduleConfig struct {
	Module string         `json:"module"`
	Title  string         `json:"title"`
	Fields []shared.Field `json:"fields"`
	Values map[string]any `json:"values"`
}

// handleMe returns the logged-in user and their manageable guilds.
func (s *Server) handleMe(w http.ResponseWriter, _ *http.Request, sess *Session) {
	writeJSON(w, http.StatusOK, meResponse{
		UserID:   sess.UserID,
		Username: sess.Username,
		Avatar:   sess.Avatar,
		Guilds:   sess.Guilds,
	})
}

// handleGuilds returns just the manageable guild list.
func (s *Server) handleGuilds(w http.ResponseWriter, _ *http.Request, sess *Session) {
	writeJSON(w, http.StatusOK, sess.Guilds)
}

// handleModules lists every configurable module with its schema and the guild's
// current values.
func (s *Server) handleModules(w http.ResponseWriter, r *http.Request, _ *Session, guildID string) {
	gid, err := strconv.ParseInt(guildID, 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid guild id")
		return
	}
	out := make([]moduleConfig, 0, len(s.order))
	for _, name := range s.order {
		mc, err := s.moduleConfigFor(r, name, gid)
		if err != nil {
			s.log.Warn("module config read failed", zap.String("module", name), zap.Error(err))
			writeErr(w, http.StatusInternalServerError, "failed to read configuration")
			return
		}
		out = append(out, mc)
	}
	writeJSON(w, http.StatusOK, out)
}

// handleModuleGet returns one module's schema + values.
func (s *Server) handleModuleGet(w http.ResponseWriter, r *http.Request, _ *Session, guildID string) {
	name := r.PathValue("mod")
	if _, ok := s.mods[name]; !ok {
		writeErr(w, http.StatusNotFound, "unknown module")
		return
	}
	gid, err := strconv.ParseInt(guildID, 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid guild id")
		return
	}
	mc, err := s.moduleConfigFor(r, name, gid)
	if err != nil {
		s.log.Warn("module config read failed", zap.String("module", name), zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "failed to read configuration")
		return
	}
	writeJSON(w, http.StatusOK, mc)
}

// handleModulePatch validates and applies a partial config patch.
func (s *Server) handleModulePatch(w http.ResponseWriter, r *http.Request, sess *Session, guildID string) {
	if !s.checkCSRF(r) {
		writeErr(w, http.StatusForbidden, "bad origin")
		return
	}
	name := r.PathValue("mod")
	mod, ok := s.mods[name]
	if !ok {
		writeErr(w, http.StatusNotFound, "unknown module")
		return
	}
	gid, err := strconv.ParseInt(guildID, 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid guild id")
		return
	}

	var patch map[string]any
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&patch); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if len(patch) == 0 {
		writeErr(w, http.StatusBadRequest, "empty patch")
		return
	}

	if err := mod.SetConfig(r.Context(), gid, patch); err != nil {
		if ue, ok := shared.AsUserError(err); ok {
			writeErr(w, http.StatusBadRequest, ue.Msg)
			return
		}
		s.log.Warn("module config write failed", zap.String("module", name), zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "failed to save configuration")
		return
	}

	// Record the change for the audit log. The config write already succeeded,
	// so a recording failure is logged but never fails the request.
	s.recordAudit(r.Context(), gid, sess, name, patch)

	// Return the fresh state so the client reflects what was actually stored.
	mc, err := s.moduleConfigFor(r, name, gid)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	writeJSON(w, http.StatusOK, mc)
}

// recordAudit appends an entry describing an accepted config change. It is
// best-effort: any error is logged, not surfaced, since the change is committed.
func (s *Server) recordAudit(ctx context.Context, gid int64, sess *Session, module string, patch map[string]any) {
	uid, _ := strconv.ParseInt(sess.UserID, 10, 64) // 0 on parse failure
	err := s.audit.record(ctx, &auditEntry{
		GuildID:  gid,
		UserID:   uid,
		Username: sess.Username,
		Module:   module,
		Changes:  patch,
	})
	if err != nil {
		s.log.Warn("audit record failed",
			zap.String("module", module), zap.Int64("guild", gid), zap.Error(err))
	}
}

// handleAudit returns a guild's most recent dashboard config changes.
func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request, _ *Session, guildID string) {
	gid, err := strconv.ParseInt(guildID, 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid guild id")
		return
	}
	rows, err := s.audit.list(r.Context(), gid, auditLimit)
	if err != nil {
		s.log.Warn("audit list failed", zap.Int64("guild", gid), zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "failed to read audit log")
		return
	}
	views := make([]auditView, 0, len(rows))
	for i := range rows {
		views = append(views, rows[i].toView())
	}
	writeJSON(w, http.StatusOK, views)
}

// moduleConfigFor reads one module's schema and current values for a guild.
func (s *Server) moduleConfigFor(r *http.Request, name string, gid int64) (moduleConfig, error) {
	mod := s.mods[name]
	schema := mod.ConfigSchema()
	values, err := mod.GetConfig(r.Context(), gid)
	if err != nil {
		return moduleConfig{}, err
	}
	return moduleConfig{
		Module: schema.Module,
		Title:  schema.Title,
		Fields: schema.Fields,
		Values: values,
	}, nil
}
