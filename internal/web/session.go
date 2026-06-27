package web

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/dipu-sharma/disgo-bot/internal/cache"
)

// sessionCookie is the name of the cookie holding the opaque session token.
const sessionCookie = "disgo_session"

// errNoSession is returned by the store when a token is absent or expired.
var errNoSession = errors.New("web: no session")

// GuildBrief is the minimal guild identity the dashboard needs. The manageable
// set is captured at login and stored in the session, so per-request handlers
// never need the user's OAuth token (which is discarded after the callback).
type GuildBrief struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Icon string `json:"icon,omitempty"`
}

// Session is the server-side login record for a dashboard user.
type Session struct {
	UserID    string       `json:"user_id"`
	Username  string       `json:"username"`
	Avatar    string       `json:"avatar,omitempty"`
	Guilds    []GuildBrief `json:"guilds"`
	ExpiresAt time.Time    `json:"expires_at"`
}

// manages reports whether the session's user can manage guildID (the set was
// computed at login: MANAGE_GUILD held AND the bot present).
func (s *Session) manages(guildID string) bool {
	for _, g := range s.Guilds {
		if g.ID == guildID {
			return true
		}
	}
	return false
}

// sessionStore persists sessions in the shared cache under "web:sess:<token>".
type sessionStore struct {
	cache cache.Cache
	ttl   time.Duration
}

func newSessionStore(c cache.Cache, ttl time.Duration) *sessionStore {
	return &sessionStore{cache: c, ttl: ttl}
}

func sessionKey(token string) string { return "web:sess:" + token }

// newToken returns a cryptographically-random, URL-safe 256-bit token.
func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// create stores a session and returns its token. ExpiresAt is set from the TTL.
func (st *sessionStore) create(ctx context.Context, sess *Session) (string, error) {
	token, err := newToken()
	if err != nil {
		return "", err
	}
	sess.ExpiresAt = nowUTC().Add(st.ttl)
	raw, err := json.Marshal(sess)
	if err != nil {
		return "", err
	}
	if err := st.cache.Set(ctx, sessionKey(token), string(raw), st.ttl); err != nil {
		return "", err
	}
	return token, nil
}

// load returns the session for token, or errNoSession if absent/expired.
func (st *sessionStore) load(ctx context.Context, token string) (*Session, error) {
	if token == "" {
		return nil, errNoSession
	}
	raw, err := st.cache.Get(ctx, sessionKey(token))
	if errors.Is(err, cache.ErrMiss) {
		return nil, errNoSession
	}
	if err != nil {
		return nil, err
	}
	var sess Session
	if err := json.Unmarshal([]byte(raw), &sess); err != nil {
		return nil, errNoSession
	}
	if nowUTC().After(sess.ExpiresAt) {
		_ = st.cache.Del(ctx, sessionKey(token))
		return nil, errNoSession
	}
	return &sess, nil
}

// destroy removes a session (logout).
func (st *sessionStore) destroy(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return st.cache.Del(ctx, sessionKey(token))
}

// --- cookie helpers ---

func setSessionCookie(w http.ResponseWriter, token string, secure bool, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(ttl.Seconds()),
	})
}

func clearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func cookieValue(r *http.Request, name string) string {
	if c, err := r.Cookie(name); err == nil {
		return c.Value
	}
	return ""
}

// nowUTC is a seam so tests don't depend on wall-clock formatting.
func nowUTC() time.Time { return time.Now().UTC() }
