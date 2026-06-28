// Package web serves the optional dashboard: Discord OAuth2 login, a session
// store backed by the shared cache, and a REST API that exposes each module's
// per-guild configuration via the shared.Configurable seam. It is started only
// when config.Web.Enabled is set and the OAuth credentials are present.
package web

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dipu-sharma/disgo-bot/internal/config"
	"github.com/dipu-sharma/disgo-bot/shared"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

//go:embed static
var staticFS embed.FS

// Server is the dashboard HTTP server.
type Server struct {
	deps *shared.Deps
	log  *zap.Logger
	cfg  config.WebConfig

	oauth      *oauth2.Config
	sess       *sessionStore
	audit      *auditStore
	publicHost string

	mods  map[string]shared.Configurable // configurable modules by name
	order []string                       // module names in registration order

	moderation shared.Moderation // moderation console seam, if a module exposes it
	economy    shared.Economy    // economy console seam, if a module exposes it
	leveling   shared.Leveling   // leveling console seam, if a module exposes it

	http *http.Server
}

// New builds the dashboard server from the dependency container and the
// registered modules (those implementing shared.Configurable become editable).
// It returns an error if the OAuth configuration is incomplete.
func New(deps *shared.Deps, modules []shared.Module) (*Server, error) {
	cfg := deps.Config.Web

	pub, err := url.Parse(cfg.PublicURL)
	if err != nil || pub.Host == "" {
		return nil, &shared.UserError{Msg: "web.public_url must be a valid absolute URL"}
	}

	oauth := &oauth2.Config{
		ClientID:     deps.Config.Discord.AppID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     discordEndpoint,
		RedirectURL:  strings.TrimRight(cfg.PublicURL, "/") + "/auth/callback",
		Scopes:       []string{"identify", "guilds"},
	}

	ttl := time.Duration(cfg.SessionHours) * time.Hour
	if ttl <= 0 {
		ttl = 168 * time.Hour
	}

	s := &Server{
		deps:       deps,
		log:        deps.Log,
		cfg:        cfg,
		oauth:      oauth,
		sess:       newSessionStore(deps.Cache, ttl),
		audit:      newAuditStore(deps.DB),
		publicHost: pub.Host,
		mods:       map[string]shared.Configurable{},
	}
	for _, m := range modules {
		if c, ok := m.(shared.Configurable); ok {
			s.mods[m.Name()] = c
			s.order = append(s.order, m.Name())
		}
		if mod, ok := m.(shared.Moderation); ok {
			s.moderation = mod
		}
		if eco, ok := m.(shared.Economy); ok {
			s.economy = eco
		}
		if lvl, ok := m.(shared.Leveling); ok {
			s.leveling = lvl
		}
	}
	s.http = &http.Server{
		Addr:              cfg.Addr,
		Handler:           s.recoverLog(s.routes()),
		ReadHeaderTimeout: 5 * time.Second,
	}
	return s, nil
}

// routes wires the mux using Go 1.22+ method+wildcard patterns.
func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /auth/login", s.handleLogin)
	mux.HandleFunc("GET /auth/callback", s.handleCallback)
	mux.HandleFunc("POST /auth/logout", s.handleLogout)

	mux.HandleFunc("GET /api/me", s.requireAuth(s.handleMe))
	mux.HandleFunc("GET /api/guilds", s.requireAuth(s.handleGuilds))
	mux.HandleFunc("GET /api/guilds/{id}/overview", s.requireAuth(s.requireGuildManage(s.handleOverview)))
	mux.HandleFunc("GET /api/guilds/{id}/roles", s.requireAuth(s.requireGuildManage(s.handleRoles)))
	mux.HandleFunc("GET /api/guilds/{id}/channels", s.requireAuth(s.requireGuildManage(s.handleChannels)))
	mux.HandleFunc("GET /api/guilds/{id}/modules", s.requireAuth(s.requireGuildManage(s.handleModules)))
	mux.HandleFunc("GET /api/guilds/{id}/modules/{mod}", s.requireAuth(s.requireGuildManage(s.handleModuleGet)))
	mux.HandleFunc("PATCH /api/guilds/{id}/modules/{mod}", s.requireAuth(s.requireGuildManage(s.handleModulePatch)))
	mux.HandleFunc("GET /api/guilds/{id}/audit", s.requireAuth(s.requireGuildManage(s.handleAudit)))

	// Moderation console.
	mux.HandleFunc("GET /api/guilds/{id}/moderation/cases", s.requireAuth(s.requireGuildManage(s.handleModCases)))
	mux.HandleFunc("PATCH /api/guilds/{id}/moderation/cases/{num}", s.requireAuth(s.requireGuildManage(s.handleModCaseReason)))
	mux.HandleFunc("POST /api/guilds/{id}/moderation/actions", s.requireAuth(s.requireGuildManage(s.handleModAction)))

	// Economy console — leaderboard + shop CRUD.
	mux.HandleFunc("GET /api/guilds/{id}/economy/leaderboard", s.requireAuth(s.requireGuildManage(s.handleEconLeaderboard)))
	mux.HandleFunc("GET /api/guilds/{id}/economy/shop", s.requireAuth(s.requireGuildManage(s.handleEconShop)))
	mux.HandleFunc("POST /api/guilds/{id}/economy/shop", s.requireAuth(s.requireGuildManage(s.handleEconShopAdd)))
	mux.HandleFunc("PATCH /api/guilds/{id}/economy/shop/{item}", s.requireAuth(s.requireGuildManage(s.handleEconShopUpdate)))
	mux.HandleFunc("DELETE /api/guilds/{id}/economy/shop/{item}", s.requireAuth(s.requireGuildManage(s.handleEconShopDelete)))

	// Leveling console — XP leaderboard + level rewards.
	mux.HandleFunc("GET /api/guilds/{id}/leveling/leaderboard", s.requireAuth(s.requireGuildManage(s.handleLevelLeaderboard)))
	mux.HandleFunc("GET /api/guilds/{id}/leveling/rewards", s.requireAuth(s.requireGuildManage(s.handleLevelRewards)))
	mux.HandleFunc("PUT /api/guilds/{id}/leveling/rewards/{level}", s.requireAuth(s.requireGuildManage(s.handleLevelRewardSet)))
	mux.HandleFunc("DELETE /api/guilds/{id}/leveling/rewards/{level}", s.requireAuth(s.requireGuildManage(s.handleLevelRewardDelete)))

	// Feature discovery — which management consoles the dashboard should show.
	mux.HandleFunc("GET /api/guilds/{id}/features", s.requireAuth(s.requireGuildManage(s.handleFeatures)))

	// Static dashboard at "/".
	sub, _ := fs.Sub(staticFS, "static")
	mux.Handle("GET /", http.FileServer(http.FS(sub)))

	return mux
}

// ModuleCount reports how many configurable modules are exposed (for logging).
func (s *Server) ModuleCount() int { return len(s.order) }

// Start runs the HTTP server in the background.
func (s *Server) Start() {
	go func() {
		s.log.Info("dashboard listening",
			zap.String("addr", s.cfg.Addr), zap.Int("modules", len(s.order)))
		if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log.Error("dashboard server error", zap.Error(err))
		}
	}()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) {
	if err := s.http.Shutdown(ctx); err != nil {
		s.log.Warn("dashboard shutdown", zap.Error(err))
	}
}
