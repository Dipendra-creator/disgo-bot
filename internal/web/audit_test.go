package web

import (
	"context"
	"testing"
	"time"
)

// A store with no database must degrade to a safe no-op so the dashboard keeps
// working without persistence (and unit tests need no DB).
func TestAuditStoreNilDBSafe(t *testing.T) {
	ctx := context.Background()

	for _, st := range []*auditStore{nil, newAuditStore(nil)} {
		if err := st.record(ctx, &auditEntry{GuildID: 1, Module: "x"}); err != nil {
			t.Fatalf("record on nil-db store: %v", err)
		}
		rows, err := st.list(ctx, 1, 10)
		if err != nil {
			t.Fatalf("list on nil-db store: %v", err)
		}
		if len(rows) != 0 {
			t.Fatalf("expected no rows, got %d", len(rows))
		}
	}
}

// toView renders snowflakes as strings (JS precision safety) and carries the
// change set through unchanged.
func TestAuditEntryToView(t *testing.T) {
	at := time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)
	e := auditEntry{
		GuildID:   42,
		UserID:    123456789012345678,
		Username:  "ada",
		Module:    "leveling",
		Changes:   map[string]any{"enabled": true, "xp_min": float64(5)},
		CreatedAt: at,
	}
	v := e.toView()
	if v.UserID != "123456789012345678" {
		t.Fatalf("user_id = %q, want string snowflake", v.UserID)
	}
	if v.Username != "ada" || v.Module != "leveling" {
		t.Fatalf("identity fields not carried: %+v", v)
	}
	if v.Changes["enabled"] != true || v.Changes["xp_min"] != float64(5) {
		t.Fatalf("changes not carried: %+v", v.Changes)
	}
	if !v.CreatedAt.Equal(at) {
		t.Fatalf("created_at = %v, want %v", v.CreatedAt, at)
	}
}
