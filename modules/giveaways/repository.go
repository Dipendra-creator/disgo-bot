package giveaways

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/uptrace/bun"
)

// Sentinel errors surfaced to the service layer.
var (
	ErrNotFound  = errors.New("giveaway not found")
	ErrEnded     = errors.New("giveaway already ended")
	ErrNotEnded  = errors.New("giveaway has not ended yet")
	ErrNoEntries = errors.New("giveaway has no entries")
)

type repo struct{ db *bun.DB }

func newRepo(db *bun.DB) *repo { return &repo{db: db} }

// create inserts a giveaway and populates its generated ID.
func (r *repo) create(ctx context.Context, g *Giveaway) error {
	_, err := r.db.NewInsert().Model(g).Returning("id").Exec(ctx)
	return err
}

// delete removes a giveaway (used to roll back a failed panel post). Entries
// cascade via the foreign key.
func (r *repo) delete(ctx context.Context, id int64) error {
	_, err := r.db.NewDelete().Model((*Giveaway)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

// setMessage records the posted panel's message ID.
func (r *repo) setMessage(ctx context.Context, id, messageID int64) error {
	_, err := r.db.NewUpdate().Model((*Giveaway)(nil)).
		Set("message_id = ?", messageID).Where("id = ?", id).Exec(ctx)
	return err
}

// byID fetches a giveaway by its local ID.
func (r *repo) byID(ctx context.Context, id int64) (*Giveaway, error) {
	g := new(Giveaway)
	err := r.db.NewSelect().Model(g).Where("id = ?", id).Limit(1).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return g, nil
}

// addEntry records a member's entry; added=false means they were already in.
func (r *repo) addEntry(ctx context.Context, giveawayID, userID int64) (added bool, err error) {
	res, err := r.db.NewRaw(
		`INSERT INTO giveaway_entries (giveaway_id, user_id) VALUES (?, ?)
		 ON CONFLICT (giveaway_id, user_id) DO NOTHING`,
		giveawayID, userID).Exec(ctx)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// removeEntry withdraws a member's entry; removed=false means they weren't in.
func (r *repo) removeEntry(ctx context.Context, giveawayID, userID int64) (removed bool, err error) {
	res, err := r.db.NewDelete().Model((*Entry)(nil)).
		Where("giveaway_id = ? AND user_id = ?", giveawayID, userID).Exec(ctx)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// countEntries returns how many members have entered.
func (r *repo) countEntries(ctx context.Context, giveawayID int64) (int, error) {
	return r.db.NewSelect().Model((*Entry)(nil)).Where("giveaway_id = ?", giveawayID).Count(ctx)
}

// entrants returns every entered member's user ID.
func (r *repo) entrants(ctx context.Context, giveawayID int64) ([]int64, error) {
	var ids []int64
	err := r.db.NewSelect().Model((*Entry)(nil)).
		Column("user_id").Where("giveaway_id = ?", giveawayID).Scan(ctx, &ids)
	return ids, err
}

// markEnded closes a giveaway and records its drawn winners.
func (r *repo) markEnded(ctx context.Context, id int64, winnerIDs string) error {
	_, err := r.db.NewUpdate().Model((*Giveaway)(nil)).
		Set("ended = true").Set("winner_ids = ?", winnerIDs).
		Where("id = ?", id).Exec(ctx)
	return err
}

// setWinners overwrites the winner list (used by reroll).
func (r *repo) setWinners(ctx context.Context, id int64, winnerIDs string) error {
	_, err := r.db.NewUpdate().Model((*Giveaway)(nil)).
		Set("winner_ids = ?", winnerIDs).Where("id = ?", id).Exec(ctx)
	return err
}

// listActive returns a guild's running giveaways, soonest-ending first.
func (r *repo) listActive(ctx context.Context, guildID int64) ([]Giveaway, error) {
	var rows []Giveaway
	err := r.db.NewSelect().Model(&rows).
		Where("guild_id = ? AND ended = false", guildID).
		Order("ends_at ASC").Scan(ctx)
	return rows, err
}

// due returns active giveaways whose timer has expired.
func (r *repo) due(ctx context.Context, now time.Time) ([]Giveaway, error) {
	var rows []Giveaway
	err := r.db.NewSelect().Model(&rows).
		Where("ended = false AND ends_at <= ?", now).
		Order("ends_at ASC").Limit(50).Scan(ctx)
	return rows, err
}
