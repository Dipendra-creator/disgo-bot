package economy

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/uptrace/bun"
)

// Sentinel errors surfaced to the service layer.
var (
	ErrInsufficient = errors.New("insufficient funds")
	ErrItemNotFound = errors.New("shop item not found")
	ErrItemExists   = errors.New("shop item already exists")
	ErrOutOfStock   = errors.New("shop item out of stock")
)

type repo struct{ db *bun.DB }

func newRepo(db *bun.DB) *repo { return &repo{db: db} }

// --- settings ---

func (r *repo) getSettings(ctx context.Context, guildID int64) (*Settings, error) {
	s := new(Settings)
	err := r.db.NewSelect().Model(s).Where("guild_id = ?", guildID).Limit(1).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return defaultSettings(guildID), nil
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *repo) saveSettings(ctx context.Context, s *Settings) error {
	_, err := r.db.NewInsert().Model(s).
		On("CONFLICT (guild_id) DO UPDATE").
		Set("currency_name = EXCLUDED.currency_name").
		Set("currency_symbol = EXCLUDED.currency_symbol").
		Set("daily_amount = EXCLUDED.daily_amount").
		Set("work_min = EXCLUDED.work_min").
		Set("work_max = EXCLUDED.work_max").
		Set("work_cooldown_secs = EXCLUDED.work_cooldown_secs").
		Set("starting_balance = EXCLUDED.starting_balance").
		Set("updated_at = now()").
		Exec(ctx)
	return err
}

// --- accounts ---

// ensureAccount creates a member's account (seeded with the starting balance)
// if it doesn't already exist.
func (r *repo) ensureAccount(ctx context.Context, db bun.IDB, guildID, userID, starting int64) error {
	_, err := db.NewRaw(
		`INSERT INTO economy_users (guild_id, user_id, wallet)
		 VALUES (?, ?, ?) ON CONFLICT (guild_id, user_id) DO NOTHING`,
		guildID, userID, starting).Exec(ctx)
	return err
}

func (r *repo) getAccount(ctx context.Context, guildID, userID int64) (*Account, error) {
	a := new(Account)
	err := r.db.NewSelect().Model(a).
		Where("guild_id = ? AND user_id = ?", guildID, userID).Limit(1).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return &Account{GuildID: guildID, UserID: userID}, nil
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

// claimEarning adds amount to the wallet and stamps the named cooldown column,
// but only if the column is null or at/older than cutoff. ok=false means the
// member is still on cooldown.
func (r *repo) claimEarning(ctx context.Context, guildID, userID, amount, starting int64, column string, cutoff time.Time) (wallet int64, ok bool, err error) {
	if err = r.ensureAccount(ctx, r.db, guildID, userID, starting); err != nil {
		return 0, false, err
	}
	row := struct {
		Wallet int64 `bun:"wallet"`
	}{}
	// column is a fixed identifier chosen by the caller, never user input.
	q := `UPDATE economy_users SET wallet = wallet + ?, ` + column + ` = now(), updated_at = now()
	      WHERE guild_id = ? AND user_id = ? AND (` + column + ` IS NULL OR ` + column + ` <= ?)
	      RETURNING wallet`
	err = r.db.NewRaw(q, amount, guildID, userID, cutoff).Scan(ctx, &row)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return row.Wallet, true, nil
}

// transfer moves amount between two members' wallets atomically.
func (r *repo) transfer(ctx context.Context, guildID, from, to, amount, starting int64) error {
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := r.ensureAccount(ctx, tx, guildID, from, starting); err != nil {
			return err
		}
		if err := r.ensureAccount(ctx, tx, guildID, to, starting); err != nil {
			return err
		}
		var w int64
		err := tx.NewRaw(
			`UPDATE economy_users SET wallet = wallet - ?, updated_at = now()
			 WHERE guild_id = ? AND user_id = ? AND wallet >= ? RETURNING wallet`,
			amount, guildID, from, amount).Scan(ctx, &w)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInsufficient
		}
		if err != nil {
			return err
		}
		_, err = tx.NewRaw(
			`UPDATE economy_users SET wallet = wallet + ?, updated_at = now()
			 WHERE guild_id = ? AND user_id = ?`, amount, guildID, to).Exec(ctx)
		return err
	})
}

// move shifts amount between a member's wallet and bank. deposit=true moves
// wallet→bank (guarded by wallet ≥ amount); deposit=false moves bank→wallet
// (guarded by bank ≥ amount).
func (r *repo) move(ctx context.Context, guildID, userID, amount, starting int64, deposit bool) (*Account, error) {
	if err := r.ensureAccount(ctx, r.db, guildID, userID, starting); err != nil {
		return nil, err
	}
	// withdraw (bank → wallet)
	q := `UPDATE economy_users SET wallet = wallet + ?, bank = bank - ?, updated_at = now()
	      WHERE guild_id = ? AND user_id = ? AND bank >= ? RETURNING wallet, bank`
	if deposit { // wallet → bank
		q = `UPDATE economy_users SET wallet = wallet - ?, bank = bank + ?, updated_at = now()
		     WHERE guild_id = ? AND user_id = ? AND wallet >= ? RETURNING wallet, bank`
	}
	a := &Account{GuildID: guildID, UserID: userID}
	err := r.db.NewRaw(q, amount, amount, guildID, userID, amount).Scan(ctx, a)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrInsufficient
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

// setBalance overwrites a member's wallet (admin).
func (r *repo) setBalance(ctx context.Context, guildID, userID, wallet, starting int64) error {
	if err := r.ensureAccount(ctx, r.db, guildID, userID, starting); err != nil {
		return err
	}
	_, err := r.db.NewUpdate().Model((*Account)(nil)).
		Set("wallet = ?", wallet).Set("updated_at = now()").
		Where("guild_id = ? AND user_id = ?", guildID, userID).Exec(ctx)
	return err
}

// addWallet adds delta (which may be negative, clamped at 0) to a wallet.
func (r *repo) addWallet(ctx context.Context, guildID, userID, delta, starting int64) (int64, error) {
	if err := r.ensureAccount(ctx, r.db, guildID, userID, starting); err != nil {
		return 0, err
	}
	var w int64
	err := r.db.NewRaw(
		`UPDATE economy_users SET wallet = GREATEST(wallet + ?, 0), updated_at = now()
		 WHERE guild_id = ? AND user_id = ? RETURNING wallet`,
		delta, guildID, userID).Scan(ctx, &w)
	return w, err
}

func (r *repo) resetAccount(ctx context.Context, guildID, userID int64) error {
	_, err := r.db.NewDelete().Model((*Account)(nil)).
		Where("guild_id = ? AND user_id = ?", guildID, userID).Exec(ctx)
	return err
}

func (r *repo) richList(ctx context.Context, guildID int64, offset, limit int) ([]Account, error) {
	var rows []Account
	err := r.db.NewSelect().Model(&rows).
		Where("guild_id = ? AND (wallet + bank) > 0", guildID).
		OrderExpr("(wallet + bank) DESC").
		Offset(offset).Limit(limit).Scan(ctx)
	return rows, err
}

func (r *repo) countRich(ctx context.Context, guildID int64) (int, error) {
	return r.db.NewSelect().Model((*Account)(nil)).
		Where("guild_id = ? AND (wallet + bank) > 0", guildID).Count(ctx)
}

// --- shop ---

func (r *repo) addItem(ctx context.Context, it *ShopItem) error {
	_, err := r.db.NewInsert().Model(it).Exec(ctx)
	if err != nil && strings.Contains(err.Error(), "duplicate") {
		return ErrItemExists
	}
	return err
}

func (r *repo) removeItem(ctx context.Context, guildID int64, name string) (bool, error) {
	res, err := r.db.NewDelete().Model((*ShopItem)(nil)).
		Where("guild_id = ? AND lower(name) = lower(?)", guildID, name).Exec(ctx)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// updateItem overwrites an item's editable fields by id (scoped to the guild).
// It reports whether a row matched.
func (r *repo) updateItem(ctx context.Context, guildID, id int64, name, desc string, price, roleID int64, stock int) (bool, error) {
	res, err := r.db.NewUpdate().Model((*ShopItem)(nil)).
		Set("name = ?", name).
		Set("description = ?", desc).
		Set("price = ?", price).
		Set("role_id = ?", roleID).
		Set("stock = ?", stock).
		Where("id = ? AND guild_id = ?", id, guildID).Exec(ctx)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// removeItemByID deletes a shop item by id (scoped to the guild).
func (r *repo) removeItemByID(ctx context.Context, guildID, id int64) (bool, error) {
	res, err := r.db.NewDelete().Model((*ShopItem)(nil)).
		Where("id = ? AND guild_id = ?", id, guildID).Exec(ctx)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// getItemByID returns a single shop item by id (scoped to the guild).
func (r *repo) getItemByID(ctx context.Context, guildID, id int64) (*ShopItem, error) {
	it := new(ShopItem)
	err := r.db.NewSelect().Model(it).
		Where("id = ? AND guild_id = ?", id, guildID).Limit(1).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrItemNotFound
	}
	if err != nil {
		return nil, err
	}
	return it, nil
}

func (r *repo) listItems(ctx context.Context, guildID int64, offset, limit int) ([]ShopItem, error) {
	var rows []ShopItem
	err := r.db.NewSelect().Model(&rows).
		Where("guild_id = ?", guildID).
		Order("price ASC").Offset(offset).Limit(limit).Scan(ctx)
	return rows, err
}

func (r *repo) countItems(ctx context.Context, guildID int64) (int, error) {
	return r.db.NewSelect().Model((*ShopItem)(nil)).Where("guild_id = ?", guildID).Count(ctx)
}

// buy debits the item's price, decrements limited stock and grants one unit to
// the buyer's inventory, all atomically. It returns the purchased item.
func (r *repo) buy(ctx context.Context, guildID, userID, starting int64, name string) (*ShopItem, int64, error) {
	var item *ShopItem
	var newWallet int64
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		it := new(ShopItem)
		err := tx.NewSelect().Model(it).
			Where("guild_id = ? AND lower(name) = lower(?)", guildID, name).Limit(1).Scan(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrItemNotFound
		}
		if err != nil {
			return err
		}
		item = it

		if err := r.ensureAccount(ctx, tx, guildID, userID, starting); err != nil {
			return err
		}
		err = tx.NewRaw(
			`UPDATE economy_users SET wallet = wallet - ?, updated_at = now()
			 WHERE guild_id = ? AND user_id = ? AND wallet >= ? RETURNING wallet`,
			it.Price, guildID, userID, it.Price).Scan(ctx, &newWallet)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInsufficient
		}
		if err != nil {
			return err
		}

		if !it.Unlimited() {
			var stock int
			err = tx.NewRaw(
				`UPDATE economy_shop SET stock = stock - 1
				 WHERE id = ? AND stock > 0 RETURNING stock`, it.ID).Scan(ctx, &stock)
			if errors.Is(err, sql.ErrNoRows) {
				return ErrOutOfStock
			}
			if err != nil {
				return err
			}
		}

		_, err = tx.NewRaw(
			`INSERT INTO economy_inventory (guild_id, user_id, item_id, quantity)
			 VALUES (?, ?, ?, 1)
			 ON CONFLICT (guild_id, user_id, item_id)
			 DO UPDATE SET quantity = economy_inventory.quantity + 1`,
			guildID, userID, it.ID).Exec(ctx)
		return err
	})
	if err != nil {
		return nil, 0, err
	}
	return item, newWallet, nil
}

func (r *repo) inventory(ctx context.Context, guildID, userID int64) ([]InventoryItem, error) {
	var rows []InventoryItem
	err := r.db.NewSelect().Model(&rows).
		ColumnExpr("inv.*").
		ColumnExpr("sh.name AS name").
		ColumnExpr("sh.price AS price").
		Join("JOIN economy_shop AS sh ON sh.id = inv.item_id").
		Where("inv.guild_id = ? AND inv.user_id = ? AND inv.quantity > 0", guildID, userID).
		Order("sh.name ASC").Scan(ctx)
	return rows, err
}
