// Package database opens the PostgreSQL connection (via Bun + pgdriver) and
// applies schema migrations.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/dipu-sharma/disgo-bot/internal/config"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
)

// New opens a Bun DB over PostgreSQL, configures the pool and verifies
// connectivity. In development it attaches a query-logging hook.
func New(ctx context.Context, cfg *config.Config) (*bun.DB, error) {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(cfg.Postgres.ConnString())))
	sqldb.SetMaxOpenConns(cfg.Postgres.PoolMax)
	sqldb.SetMaxIdleConns(cfg.Postgres.PoolMax)
	sqldb.SetConnMaxLifetime(30 * time.Minute)

	db := bun.NewDB(sqldb, pgdialect.New())
	if cfg.Env == "development" {
		db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(false)))
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return db, nil
}
