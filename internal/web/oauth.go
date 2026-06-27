package web

import (
	"net/http"
	"time"

	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

// OAuth constants for Discord's authorization-code flow.
const (
	oauthStateCookie = "disgo_oauth_state"
	oauthStateTTL    = 10 * time.Minute
)

// discordEndpoint is Discord's OAuth2 endpoint (not in oauth2/endpoints).
var discordEndpoint = oauth2.Endpoint{
	AuthURL:  "https://discord.com/oauth2/authorize",
	TokenURL: "https://discord.com/api/oauth2/token",
}

func oauthKey(state string) string { return "web:oauth:" + state }

// handleLogin starts the OAuth flow: it generates a PKCE verifier and a CSRF
// state, stashes the verifier server-side keyed by state, sets the state in a
// short-lived cookie (double-submit), and redirects to Discord's consent page.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	state, err := newToken()
	if err != nil {
		s.log.Error("oauth state gen", zap.Error(err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	verifier := oauth2.GenerateVerifier()
	if err := s.deps.Cache.Set(r.Context(), oauthKey(state), verifier, oauthStateTTL); err != nil {
		s.log.Error("oauth state store", zap.Error(err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(oauthStateTTL.Seconds()),
	})
	url := s.oauth.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))
	http.Redirect(w, r, url, http.StatusFound)
}

// handleCallback completes the flow: it verifies state, exchanges the code (with
// the PKCE verifier), captures the user's identity and manageable guilds, stores
// a session and sets the session cookie. The OAuth token is used only here and
// then discarded — never persisted or exposed to the browser.
func (s *Server) handleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	state, code := q.Get("state"), q.Get("code")
	cookieState := cookieValue(r, oauthStateCookie)

	// Clear the state cookie regardless of outcome.
	http.SetCookie(w, &http.Cookie{Name: oauthStateCookie, Path: "/", MaxAge: -1, HttpOnly: true})

	if state == "" || code == "" || state != cookieState {
		http.Error(w, "invalid OAuth state", http.StatusBadRequest)
		return
	}
	verifier, err := s.deps.Cache.Get(ctx, oauthKey(state))
	if err != nil {
		http.Error(w, "OAuth state expired, please retry", http.StatusBadRequest)
		return
	}
	_ = s.deps.Cache.Del(ctx, oauthKey(state))

	token, err := s.oauth.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		s.log.Warn("oauth exchange failed", zap.Error(err))
		http.Error(w, "login failed", http.StatusBadGateway)
		return
	}

	client := s.oauth.Client(ctx, token)
	user, err := fetchUser(ctx, client)
	if err != nil {
		s.log.Warn("fetch user failed", zap.Error(err))
		http.Error(w, "login failed", http.StatusBadGateway)
		return
	}
	guilds, err := fetchGuilds(ctx, client)
	if err != nil {
		s.log.Warn("fetch guilds failed", zap.Error(err))
		http.Error(w, "login failed", http.StatusBadGateway)
		return
	}

	sess := &Session{
		UserID:   user.ID,
		Username: user.name(),
		Avatar:   user.Avatar,
		Guilds:   manageableGuilds(guilds, s.botInGuild),
	}
	st, err := s.sess.create(ctx, sess)
	if err != nil {
		s.log.Error("session create failed", zap.Error(err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	setSessionCookie(w, st, s.cfg.CookieSecure, s.sess.ttl)
	http.Redirect(w, r, "/", http.StatusFound)
}

// handleLogout destroys the session and clears the cookie.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	token := cookieValue(r, sessionCookie)
	if err := s.sess.destroy(r.Context(), token); err != nil {
		s.log.Warn("session destroy failed", zap.Error(err))
	}
	clearSessionCookie(w, s.cfg.CookieSecure)
	w.WriteHeader(http.StatusNoContent)
}
