package web

import (
	"context"
	"strconv"
	"time"

	"github.com/uptrace/bun"
)

// auditLimit caps how many recent entries the dashboard lists per guild.
const auditLimit = 50

// auditEntry is one recorded configuration change made through the dashboard.
// Changes holds the submitted field patch (key -> new value) as JSONB.
type auditEntry struct {
	bun.BaseModel `bun:"table:web_audit_log,alias:wal"`

	ID        int64          `bun:"id,pk,autoincrement"`
	GuildID   int64          `bun:"guild_id,notnull"`
	UserID    int64          `bun:"user_id,notnull"`
	Username  string         `bun:"username,notnull"`
	Module    string         `bun:"module,notnull"`
	Changes   map[string]any `bun:"changes,type:jsonb,nullzero"`
	CreatedAt time.Time      `bun:"created_at,nullzero,notnull,default:now()"`
}

// auditStore appends and reads dashboard change records. It tolerates a nil DB
// (e.g. in unit tests or when the bot runs without persistence) by treating
// recording as a no-op and listing as empty, so the API never fails on it.
type auditStore struct{ db *bun.DB }

func newAuditStore(db *bun.DB) *auditStore { return &auditStore{db: db} }

// record appends one change entry. A nil store/DB is a no-op; callers log but do
// not fail the request on error, since the config write already succeeded.
func (a *auditStore) record(ctx context.Context, e *auditEntry) error {
	if a == nil || a.db == nil {
		return nil
	}
	_, err := a.db.NewInsert().Model(e).Exec(ctx)
	return err
}

// list returns a guild's most recent change entries, newest first.
func (a *auditStore) list(ctx context.Context, guildID int64, limit int) ([]auditEntry, error) {
	if a == nil || a.db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = auditLimit
	}
	var rows []auditEntry
	err := a.db.NewSelect().Model(&rows).
		Where("guild_id = ?", guildID).
		Order("created_at DESC", "id DESC").
		Limit(limit).
		Scan(ctx)
	return rows, err
}

// auditView is the JSON shape returned to the dashboard. Snowflakes are rendered
// as strings to stay safe against JS 2⁵³ precision loss, mirroring the config API.
type auditView struct {
	UserID    string         `json:"user_id"`
	Username  string         `json:"username"`
	Module    string         `json:"module"`
	Changes   map[string]any `json:"changes"`
	CreatedAt time.Time      `json:"created_at"`
}

// toView renders a stored entry for the API.
func (e *auditEntry) toView() auditView {
	return auditView{
		UserID:    strconv.FormatInt(e.UserID, 10),
		Username:  e.Username,
		Module:    e.Module,
		Changes:   e.Changes,
		CreatedAt: e.CreatedAt,
	}
}
