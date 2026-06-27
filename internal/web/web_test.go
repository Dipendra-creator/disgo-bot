package web

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/dipu-sharma/disgo-bot/internal/cache"
	"github.com/dipu-sharma/disgo-bot/internal/config"
	"github.com/dipu-sharma/disgo-bot/shared"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

func memCache(t *testing.T) cache.Cache {
	t.Helper()
	return cache.New(context.Background(), config.RedisConfig{Enabled: false}, zap.NewNop())
}

func TestSessionRoundTrip(t *testing.T) {
	ctx := context.Background()
	store := newSessionStore(memCache(t), time.Hour)

	in := &Session{
		UserID:   "42",
		Username: "ada",
		Guilds:   []GuildBrief{{ID: "1", Name: "Guild One"}},
	}
	token, err := store.create(ctx, in)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if token == "" {
		t.Fatal("empty token")
	}

	got, err := store.load(ctx, token)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.UserID != "42" || got.Username != "ada" || !got.manages("1") {
		t.Fatalf("loaded session mismatch: %+v", got)
	}
	if got.manages("999") {
		t.Fatal("manages must be false for an unlisted guild")
	}

	if err := store.destroy(ctx, token); err != nil {
		t.Fatalf("destroy: %v", err)
	}
	if _, err := store.load(ctx, token); err == nil {
		t.Fatal("expected errNoSession after destroy")
	}
}

func TestSessionExpiry(t *testing.T) {
	ctx := context.Background()
	c := memCache(t)
	store := newSessionStore(c, time.Hour)

	// Hand-place a session whose ExpiresAt is already in the past.
	expired := &Session{UserID: "7", ExpiresAt: nowUTC().Add(-time.Minute)}
	raw, _ := json.Marshal(expired)
	if err := c.Set(ctx, sessionKey("stale"), string(raw), time.Hour); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := store.load(ctx, "stale"); err == nil {
		t.Fatal("expected expired session to be rejected")
	}
}

func TestManageableGuilds(t *testing.T) {
	guilds := []apiGuild{
		{ID: "1", Name: "Owner", Owner: true, Permissions: "0"},
		{ID: "2", Name: "Manager", Permissions: "32"},      // 0x20 MANAGE_GUILD
		{ID: "3", Name: "Member", Permissions: "0"},        // no perm
		{ID: "4", Name: "ManagerNoBot", Permissions: "32"}, // perm but bot absent
	}
	botPresent := func(id string) bool { return id != "4" }

	got := manageableGuilds(guilds, botPresent)
	ids := map[string]bool{}
	for _, g := range got {
		ids[g.ID] = true
	}
	if !ids["1"] || !ids["2"] {
		t.Fatalf("owner and manager must be included: %v", got)
	}
	if ids["3"] {
		t.Fatal("member without MANAGE_GUILD must be excluded")
	}
	if ids["4"] {
		t.Fatal("guild without the bot must be excluded")
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
}

func TestNewBuildsOAuth(t *testing.T) {
	cfg := config.Default()
	cfg.Discord.AppID = "appid123"
	cfg.Web.Enabled = true
	cfg.Web.PublicURL = "https://dash.example.com"
	cfg.Web.ClientSecret = "secret"

	deps := &shared.Deps{Config: &cfg, Log: zap.NewNop(), Cache: memCache(t)}
	s, err := New(deps, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if s.publicHost != "dash.example.com" {
		t.Fatalf("publicHost = %q", s.publicHost)
	}
	if s.oauth.RedirectURL != "https://dash.example.com/auth/callback" {
		t.Fatalf("RedirectURL = %q", s.oauth.RedirectURL)
	}

	url := s.oauth.AuthCodeURL("st4te", oauth2.S256ChallengeOption(oauth2.GenerateVerifier()))
	for _, want := range []string{"state=st4te", "code_challenge=", "code_challenge_method=S256", "scope=identify+guilds", "client_id=appid123"} {
		if !strings.Contains(url, want) {
			t.Errorf("auth URL missing %q\n  got: %s", want, url)
		}
	}
}

func TestNewRejectsBadPublicURL(t *testing.T) {
	cfg := config.Default()
	cfg.Web.PublicURL = "" // invalid
	deps := &shared.Deps{Config: &cfg, Log: zap.NewNop(), Cache: memCache(t)}
	if _, err := New(deps, nil); err == nil {
		t.Fatal("expected error for empty public_url")
	}
}
