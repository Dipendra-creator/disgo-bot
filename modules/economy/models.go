package economy

import (
	"time"

	"github.com/uptrace/bun"
)

// dailyCooldown is the fixed interval between /daily claims.
const dailyCooldown = 24 * time.Hour

// Settings is per-guild economy configuration.
type Settings struct {
	bun.BaseModel `bun:"table:economy_settings,alias:es"`

	GuildID         int64     `bun:"guild_id,pk"`
	CurrencyName    string    `bun:"currency_name,notnull"`
	CurrencySymbol  string    `bun:"currency_symbol,notnull"`
	DailyAmount     int64     `bun:"daily_amount,notnull"`
	WorkMin         int64     `bun:"work_min,notnull"`
	WorkMax         int64     `bun:"work_max,notnull"`
	WorkCooldownSec int       `bun:"work_cooldown_secs,notnull"`
	StartingBalance int64     `bun:"starting_balance,notnull"`
	CreatedAt       time.Time `bun:"created_at,nullzero,notnull,default:now()"`
	UpdatedAt       time.Time `bun:"updated_at,nullzero,notnull,default:now()"`
}

// defaultSettings returns the baseline configuration for an unconfigured guild.
func defaultSettings(guildID int64) *Settings {
	return &Settings{
		GuildID:         guildID,
		CurrencyName:    "coins",
		CurrencySymbol:  "🪙",
		DailyAmount:     250,
		WorkMin:         50,
		WorkMax:         250,
		WorkCooldownSec: 3600,
	}
}

// Account is one member's currency holdings in a guild.
type Account struct {
	bun.BaseModel `bun:"table:economy_users,alias:eu"`

	GuildID   int64     `bun:"guild_id,pk"`
	UserID    int64     `bun:"user_id,pk"`
	Wallet    int64     `bun:"wallet,notnull"`
	Bank      int64     `bun:"bank,notnull"`
	LastDaily time.Time `bun:"last_daily,nullzero"`
	LastWork  time.Time `bun:"last_work,nullzero"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:now()"`
}

// Net returns the account's total net worth (wallet + bank).
func (a *Account) Net() int64 { return a.Wallet + a.Bank }

// ShopItem is a purchasable item in a guild's shop.
type ShopItem struct {
	bun.BaseModel `bun:"table:economy_shop,alias:sh"`

	ID          int64     `bun:"id,pk,autoincrement"`
	GuildID     int64     `bun:"guild_id,notnull"`
	Name        string    `bun:"name,notnull"`
	Description string    `bun:"description,notnull"`
	Price       int64     `bun:"price,notnull"`
	RoleID      int64     `bun:"role_id,notnull"` // 0 = no role granted
	Stock       int       `bun:"stock,notnull"`   // -1 = unlimited
	CreatedAt   time.Time `bun:"created_at,nullzero,notnull,default:now()"`
}

// Unlimited reports whether the item has unlimited stock.
func (i *ShopItem) Unlimited() bool { return i.Stock < 0 }

// InventoryItem is a quantity of a shop item owned by a member, joined with the
// item's display fields for listing.
type InventoryItem struct {
	bun.BaseModel `bun:"table:economy_inventory,alias:inv"`

	GuildID  int64 `bun:"guild_id,pk"`
	UserID   int64 `bun:"user_id,pk"`
	ItemID   int64 `bun:"item_id,pk"`
	Quantity int   `bun:"quantity,notnull"`

	// Populated by the listing join, not stored on this table.
	Name  string `bun:"name,scanonly"`
	Price int64  `bun:"price,scanonly"`
}
