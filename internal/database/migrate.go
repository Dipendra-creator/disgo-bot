package database

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	rootdb "github.com/dipu-sharma/disgo-bot/database"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// schemaMigrationsDDL tracks which migration files have been applied.
const schemaMigrationsDDL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    name        TEXT        PRIMARY KEY,
    applied_at  TIMESTAMPTZ NOT NULL DEFAULT now()
)`

// Migrate applies every embedded migration not yet recorded in
// schema_migrations, in lexical filename order, each within its own
// transaction. It is idempotent and safe to run on every startup.
func Migrate(ctx context.Context, db *bun.DB, log *zap.Logger) error {
	if _, err := db.ExecContext(ctx, schemaMigrationsDDL); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	applied, err := appliedSet(ctx, db)
	if err != nil {
		return err
	}

	files, err := migrationFiles()
	if err != nil {
		return err
	}

	var ran int
	for _, name := range files {
		if applied[name] {
			continue
		}
		sqlBytes, err := rootdb.Migrations.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %q: %w", name, err)
		}
		if err := applyOne(ctx, db, name, string(sqlBytes)); err != nil {
			return err
		}
		log.Info("applied migration", zap.String("migration", name))
		ran++
	}

	if ran == 0 {
		log.Info("database schema up to date")
	} else {
		log.Info("database migrations complete", zap.Int("applied", ran))
	}
	return nil
}

func applyOne(ctx context.Context, db *bun.DB, name, body string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx for %q: %w", name, err)
	}
	if _, err := tx.ExecContext(ctx, body); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("exec migration %q: %w", name, err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (name) VALUES (?)", name); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("record migration %q: %w", name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %q: %w", name, err)
	}
	return nil
}

func appliedSet(ctx context.Context, db *bun.DB) (map[string]bool, error) {
	var names []string
	if err := db.NewSelect().
		ColumnExpr("name").
		TableExpr("schema_migrations").
		Scan(ctx, &names); err != nil {
		return nil, fmt.Errorf("load applied migrations: %w", err)
	}
	set := make(map[string]bool, len(names))
	for _, n := range names {
		set[n] = true
	}
	return set, nil
}

func migrationFiles() ([]string, error) {
	entries, err := fs.ReadDir(rootdb.Migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("list migrations: %w", err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	return files, nil
}
