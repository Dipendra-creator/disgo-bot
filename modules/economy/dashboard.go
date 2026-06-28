package economy

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/dipu-sharma/disgo-bot/shared"
)

// Economy implementation — exposes the net-worth leaderboard and shop-item
// management to the web dashboard. It delegates to the existing Service so the
// dashboard and slash-command paths share identical persistence and the
// in-process settings cache.

var _ shared.Economy = (*Module)(nil)

// Bounds for a single leaderboard / shop page and editable item fields.
const (
	leaderboardLimit = 100
	shopListLimit    = 100
	itemNameMax      = 100
	itemDescMax      = 280
)

// toShopView maps an internal ShopItem to its transport-agnostic form.
func toShopView(it *ShopItem) shared.ShopItemView {
	return shared.ShopItemView{
		ID:          it.ID,
		Name:        it.Name,
		Description: it.Description,
		Price:       it.Price,
		RoleID:      roleString(it.RoleID),
		Stock:       it.Stock,
	}
}

// RichLeaderboard returns a page of the net-worth leaderboard with currency labels.
func (m *Module) RichLeaderboard(ctx context.Context, guildID int64, q shared.PageQuery) (shared.EconLeaderboard, error) {
	limit, offset := clampPage(q, leaderboardLimit)
	gid := sid(guildID)

	rows, total, err := m.svc.Rich(ctx, gid, offset, limit)
	if err != nil {
		return shared.EconLeaderboard{}, err
	}
	set, err := m.svc.Settings(ctx, gid)
	if err != nil {
		return shared.EconLeaderboard{}, err
	}

	members := make([]shared.EconMember, 0, len(rows))
	for i := range rows {
		a := &rows[i]
		uid := sid(a.UserID)
		members = append(members, shared.EconMember{
			UserID:   uid,
			Username: m.memberName(gid, uid),
			Wallet:   a.Wallet,
			Bank:     a.Bank,
			Net:      a.Net(),
		})
	}
	return shared.EconLeaderboard{
		Members:  members,
		Total:    total,
		Currency: set.CurrencyName,
		Symbol:   set.CurrencySymbol,
	}, nil
}

// ListShop returns a page of the guild's shop items.
func (m *Module) ListShop(ctx context.Context, guildID int64, q shared.PageQuery) (shared.ShopPage, error) {
	limit, offset := clampPage(q, shopListLimit)
	rows, total, err := m.svc.Shop(ctx, sid(guildID), offset, limit)
	if err != nil {
		return shared.ShopPage{}, err
	}
	items := make([]shared.ShopItemView, 0, len(rows))
	for i := range rows {
		items = append(items, toShopView(&rows[i]))
	}
	return shared.ShopPage{Items: items, Total: total}, nil
}

// AddShopItem creates a purchasable item after validating its input.
func (m *Module) AddShopItem(ctx context.Context, guildID int64, in shared.ShopItemInput) (shared.ShopItemView, error) {
	clean, err := validateItem(in)
	if err != nil {
		return shared.ShopItemView{}, err
	}
	it, err := m.svc.CreateItem(ctx, sid(guildID), clean.Name, clean.Description, clean.Price, clean.RoleID, clean.Stock)
	if err != nil {
		if errors.Is(err, ErrItemExists) {
			return shared.ShopItemView{}, shared.UserErr("A shop item named %q already exists.", clean.Name)
		}
		return shared.ShopItemView{}, err
	}
	return toShopView(it), nil
}

// UpdateShopItem overwrites an existing item by id.
func (m *Module) UpdateShopItem(ctx context.Context, guildID, itemID int64, in shared.ShopItemInput) (shared.ShopItemView, error) {
	clean, err := validateItem(in)
	if err != nil {
		return shared.ShopItemView{}, err
	}
	it, err := m.svc.EditItem(ctx, sid(guildID), itemID, clean.Name, clean.Description, clean.Price, clean.RoleID, clean.Stock)
	if err != nil {
		if errors.Is(err, ErrItemNotFound) {
			return shared.ShopItemView{}, shared.UserErr("Shop item not found.")
		}
		return shared.ShopItemView{}, err
	}
	return toShopView(it), nil
}

// RemoveShopItem deletes a shop item by id.
func (m *Module) RemoveShopItem(ctx context.Context, guildID, itemID int64) error {
	ok, err := m.svc.RemoveItemByID(ctx, sid(guildID), itemID)
	if err != nil {
		return err
	}
	if !ok {
		return shared.UserErr("Shop item not found.")
	}
	return nil
}

// validateItem normalises and bounds a shop-item payload, returning a UserError
// for anything invalid.
func validateItem(in shared.ShopItemInput) (shared.ShopItemInput, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return in, shared.UserErr("Item name is required.")
	}
	if len(name) > itemNameMax {
		return in, shared.UserErr("Item name must be at most %d characters.", itemNameMax)
	}
	desc := strings.TrimSpace(in.Description)
	if len(desc) > itemDescMax {
		return in, shared.UserErr("Item description must be at most %d characters.", itemDescMax)
	}
	if in.Price < 0 {
		return in, shared.UserErr("Price can't be negative.")
	}
	if in.Stock < -1 {
		return in, shared.UserErr("Stock must be -1 (unlimited) or a count of 0 or more.")
	}
	role := strings.TrimSpace(in.RoleID)
	if role != "" && !isSnowflake(role) {
		return in, shared.UserErr("Reward role must be a valid role id.")
	}
	return shared.ShopItemInput{Name: name, Description: desc, Price: in.Price, RoleID: role, Stock: in.Stock}, nil
}

// clampPage normalises a PageQuery: limit into (0, max], offset to >= 0.
func clampPage(q shared.PageQuery, max int) (limit, offset int) {
	limit = q.Limit
	if limit <= 0 || limit > max {
		limit = max
	}
	offset = q.Offset
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

// roleString formats a stored role id, mapping the 0 sentinel to "" (no role).
func roleString(id int64) string {
	if id == 0 {
		return ""
	}
	return sid(id)
}

// isSnowflake reports whether s is a positive integer id.
func isSnowflake(s string) bool {
	n, err := strconv.ParseInt(s, 10, 64)
	return err == nil && n > 0
}

// memberName returns a display name for a member from the gateway cache, or ""
// when uncached / the session isn't ready. It never makes a Discord REST call.
func (m *Module) memberName(guildID, userID string) string {
	if m.deps == nil || m.deps.Session == nil || m.deps.Session.State == nil {
		return ""
	}
	mem, err := m.deps.Session.State.Member(guildID, userID)
	if err != nil || mem == nil {
		return ""
	}
	if mem.Nick != "" {
		return mem.Nick
	}
	if mem.User != nil {
		return mem.User.Username
	}
	return ""
}
