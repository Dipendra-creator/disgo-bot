package shared

import "context"

// Economy is an optional contract a Module implements to expose its currency
// data to the web dashboard: a net-worth leaderboard and shop-item management.
// Like Configurable and Moderation it is a purely additive, transport-agnostic
// seam — the web layer type-asserts each registered Module to Economy, and
// modules that don't implement it simply aren't surfaced as a management
// console. Implementations never import net/http and exchange only the plain
// types defined here.
type Economy interface {
	// RichLeaderboard returns a page of the net-worth leaderboard (highest
	// first) together with the guild's currency labels for display.
	RichLeaderboard(ctx context.Context, guildID int64, q PageQuery) (EconLeaderboard, error)
	// ListShop returns a page of the guild's shop items, plus the total count.
	ListShop(ctx context.Context, guildID int64, q PageQuery) (ShopPage, error)
	// AddShopItem creates a purchasable item and returns it. It returns a
	// UserError for invalid input (empty name, negative price, bad role id).
	AddShopItem(ctx context.Context, guildID int64, in ShopItemInput) (ShopItemView, error)
	// UpdateShopItem overwrites an existing item by id and returns the result.
	// It returns a UserError when the item doesn't exist or input is invalid.
	UpdateShopItem(ctx context.Context, guildID, itemID int64, in ShopItemInput) (ShopItemView, error)
	// RemoveShopItem deletes a shop item by id. It returns a UserError when no
	// item with that id exists in the guild.
	RemoveShopItem(ctx context.Context, guildID, itemID int64) error
}

// PageQuery is a generic limit/offset window shared by the paginated dashboard
// seams. The implementation clamps unreasonable values.
type PageQuery struct {
	Limit  int
	Offset int
}

// EconMember is one ranked account on the net-worth leaderboard. IDs are
// strings to preserve snowflake precision across JSON.
type EconMember struct {
	UserID   string `json:"user_id"`
	Username string `json:"username,omitempty"` // best-effort from the gateway cache
	Wallet   int64  `json:"wallet"`
	Bank     int64  `json:"bank"`
	Net      int64  `json:"net"`
}

// EconLeaderboard is one page of the net-worth leaderboard with the total
// ranked count (ignoring Limit/Offset) and the guild's currency labels.
type EconLeaderboard struct {
	Members  []EconMember `json:"members"`
	Total    int          `json:"total"`
	Currency string       `json:"currency"`
	Symbol   string       `json:"symbol"`
}

// ShopItemView is a shop item in the dashboard's transport-agnostic form.
type ShopItemView struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       int64  `json:"price"`
	RoleID      string `json:"role_id"` // "" = no role granted
	Stock       int    `json:"stock"`   // -1 = unlimited
}

// ShopPage is one page of shop items with the total item count.
type ShopPage struct {
	Items []ShopItemView `json:"items"`
	Total int            `json:"total"`
}

// ShopItemInput is the editable payload for creating or updating a shop item.
type ShopItemInput struct {
	Name        string
	Description string
	Price       int64
	RoleID      string // "" = no role granted
	Stock       int    // -1 = unlimited
}
