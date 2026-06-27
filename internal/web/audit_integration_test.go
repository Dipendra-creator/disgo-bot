package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dipu-sharma/disgo-bot/internal/config"
	idb "github.com/dipu-sharma/disgo-bot/internal/database"
	"github.com/dipu-sharma/disgo-bot/shared"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"go.uber.org/zap"
)

// openTestDB connects to DISGO_TEST_DATABASE_URL and applies all migrations, or
// skips the test when no disposable database is configured.
//
//	DISGO_TEST_DATABASE_URL=postgres://postgres@127.0.0.1:5432/disgo?sslmode=disable \
//	    go test ./internal/web/ -run Postgres -v
func openTestDB(t *testing.T) *bun.DB {
	t.Helper()
	dsn := os.Getenv("DISGO_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set DISGO_TEST_DATABASE_URL to run the Postgres integration tests")
	}
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New())
	t.Cleanup(func() { _ = db.Close() })
	if err := db.PingContext(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}
	if err := idb.Migrate(context.Background(), db, zap.NewNop()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// TestAuditStorePostgres round-trips entries through the JSONB column directly
// against the store (proving migration 0011_web_audit and the bun model tags).
func TestAuditStorePostgres(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	store := newAuditStore(db)
	const guild = int64(99001)
	if _, err := db.NewDelete().Model((*auditEntry)(nil)).Where("guild_id = ?", guild).Exec(ctx); err != nil {
		t.Fatalf("cleanup: %v", err)
	}

	first := &auditEntry{
		GuildID: guild, UserID: 123456789012345678, Username: "ada",
		Module: "leveling", Changes: map[string]any{"enabled": true, "xp_min": float64(5)},
		CreatedAt: time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC),
	}
	second := &auditEntry{
		GuildID: guild, UserID: 987654321098765432, Username: "grace",
		Module: "automod", Changes: map[string]any{"words_action": "timeout"},
		CreatedAt: time.Date(2026, 6, 27, 11, 0, 0, 0, time.UTC),
	}
	if err := store.record(ctx, first); err != nil {
		t.Fatalf("record first: %v", err)
	}
	if err := store.record(ctx, second); err != nil {
		t.Fatalf("record second: %v", err)
	}

	rows, err := store.list(ctx, guild, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len = %d, want 2", len(rows))
	}
	if rows[0].Module != "automod" || rows[1].Module != "leveling" { // newest first
		t.Fatalf("ordering wrong: %q then %q", rows[0].Module, rows[1].Module)
	}
	if rows[1].Changes["enabled"] != true || rows[1].Changes["xp_min"] != float64(5) {
		t.Fatalf("jsonb changes not preserved: %+v", rows[1].Changes)
	}
	if rows[0].UserID != 987654321098765432 {
		t.Fatalf("user_id round-trip wrong: %d", rows[0].UserID)
	}
	if rows[0].ID == 0 || rows[0].CreatedAt.IsZero() {
		t.Fatalf("autoincrement/timestamp not populated: %+v", rows[0])
	}
}

// stubModule is a minimal Configurable used to exercise the HTTP chain without a
// real module. It records the last patch it received.
type stubModule struct {
	shared.Base
	last map[string]any
}

func (m *stubModule) Name() string { return "demo" }
func (m *stubModule) ConfigSchema() shared.ConfigSchema {
	return shared.ConfigSchema{
		Module: "demo", Title: "Demo",
		Fields: []shared.Field{{Key: "enabled", Label: "Enabled", Type: shared.FieldBool}},
	}
}
func (m *stubModule) GetConfig(context.Context, int64) (map[string]any, error) {
	return map[string]any{"enabled": false}, nil
}
func (m *stubModule) SetConfig(_ context.Context, _ int64, patch map[string]any) error {
	m.last = patch
	return nil
}

// TestAuditHTTPFlowPostgres drives the real router + middleware end to end
// (auth gate, guild-manage gate, PATCH → record → GET audit) against live
// Postgres, with a seeded session standing in for the Discord OAuth login.
func TestAuditHTTPFlowPostgres(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	const guild = "99002"
	if _, err := db.NewDelete().Model((*auditEntry)(nil)).Where("guild_id = 99002").Exec(ctx); err != nil {
		t.Fatalf("cleanup: %v", err)
	}

	cfg := config.Default()
	cfg.Discord.AppID = "app"
	cfg.Web.Enabled = true
	cfg.Web.PublicURL = "http://localhost"
	cfg.Web.ClientSecret = "secret"
	deps := &shared.Deps{Config: &cfg, Log: zap.NewNop(), Cache: memCache(t), DB: db}

	s, err := New(deps, []shared.Module{&stubModule{}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	h := s.routes()

	// Seed a logged-in session that manages guild 99002.
	token, err := s.sess.create(ctx, &Session{
		UserID:   "123456789012345678",
		Username: "ada",
		Guilds:   []GuildBrief{{ID: guild, Name: "Test Guild"}},
	})
	if err != nil {
		t.Fatalf("seed session: %v", err)
	}
	authed := func(method, path, body string) *httptest.ResponseRecorder {
		var r *http.Request
		if body != "" {
			r = httptest.NewRequest(method, path, strings.NewReader(body))
		} else {
			r = httptest.NewRequest(method, path, nil)
		}
		r.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		return rec
	}

	// Unauthenticated request is rejected.
	{
		r := httptest.NewRequest("GET", "/api/guilds/"+guild+"/audit", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("no-cookie audit: code = %d, want 401", rec.Code)
		}
	}
	// A guild the user does not manage is forbidden.
	if rec := authed("GET", "/api/guilds/55555/audit", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("unmanaged guild: code = %d, want 403", rec.Code)
	}

	// PATCH the stub module's config — should succeed and record an audit row.
	if rec := authed("PATCH", "/api/guilds/"+guild+"/modules/demo", `{"enabled":true}`); rec.Code != http.StatusOK {
		t.Fatalf("patch: code = %d body = %s", rec.Code, rec.Body.String())
	}

	// GET the audit log — the change just made must appear.
	rec := authed("GET", "/api/guilds/"+guild+"/audit", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("audit get: code = %d", rec.Code)
	}
	var views []auditView
	if err := json.Unmarshal(rec.Body.Bytes(), &views); err != nil {
		t.Fatalf("decode audit: %v (%s)", err, rec.Body.String())
	}
	if len(views) != 1 {
		t.Fatalf("audit len = %d, want 1: %s", len(views), rec.Body.String())
	}
	v := views[0]
	if v.Module != "demo" || v.Username != "ada" || v.UserID != "123456789012345678" {
		t.Fatalf("audit identity wrong: %+v", v)
	}
	if v.Changes["enabled"] != true {
		t.Fatalf("audit changes wrong: %+v", v.Changes)
	}
}
