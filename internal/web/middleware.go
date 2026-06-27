package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"go.uber.org/zap"
)

// authedHandler is an HTTP handler that also receives the resolved Session.
type authedHandler func(w http.ResponseWriter, r *http.Request, sess *Session)

// requireAuth loads the session from the cookie and rejects unauthenticated
// requests with 401.
func (s *Server) requireAuth(h authedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, err := s.sess.load(r.Context(), cookieValue(r, sessionCookie))
		if err != nil {
			if !errors.Is(err, errNoSession) {
				s.log.Warn("session load failed", zap.Error(err))
			}
			writeErr(w, http.StatusUnauthorized, "not logged in")
			return
		}
		h(w, r, sess)
	}
}

// requireGuildManage wraps an authed handler for a guild-scoped route. It
// re-verifies, server-side, that the session's user may manage the path's guild
// — the client's claims are never trusted. The verified guild ID is passed on.
func (s *Server) requireGuildManage(h func(w http.ResponseWriter, r *http.Request, sess *Session, guildID string)) authedHandler {
	return func(w http.ResponseWriter, r *http.Request, sess *Session) {
		guildID := r.PathValue("id")
		if guildID == "" || !sess.manages(guildID) {
			writeErr(w, http.StatusForbidden, "you do not manage this server")
			return
		}
		h(w, r, sess, guildID)
	}
}

// checkCSRF guards state-changing requests. With SameSite=Lax cookies the main
// defense is in place; this adds an Origin/Referer same-host check as defense in
// depth. Returns true when the request is allowed.
func (s *Server) checkCSRF(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = r.Header.Get("Referer")
	}
	if origin == "" {
		// No Origin/Referer (e.g. same-origin fetch in some clients) — rely on
		// the SameSite cookie. Permit.
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Host, s.publicHost)
}

// --- response helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// recoverLog wraps the whole mux with panic recovery and request logging.
func (s *Server) recoverLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.log.Error("web handler panic",
					zap.Any("panic", rec), zap.String("path", r.URL.Path))
				writeErr(w, http.StatusInternalServerError, "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
