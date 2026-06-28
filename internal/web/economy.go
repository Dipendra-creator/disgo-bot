package web

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/dipu-sharma/disgo-bot/shared"
	"go.uber.org/zap"
)

// This file adapts the module's shared.Economy seam to JSON routes: a net-worth
// leaderboard and shop-item CRUD. Mutations re-check CSRF and append to the
// dashboard audit log.

// shopItemRequest is the create/update body for a shop item.
type shopItemRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       int64  `json:"price"`
	RoleID      string `json:"role_id"`
	Stock       int    `json:"stock"`
}

func (b shopItemRequest) input() shared.ShopItemInput {
	return shared.ShopItemInput{
		Name:        b.Name,
		Description: b.Description,
		Price:       b.Price,
		RoleID:      b.RoleID,
		Stock:       b.Stock,
	}
}

// handleEconLeaderboard serves GET /api/guilds/{id}/economy/leaderboard.
func (s *Server) handleEconLeaderboard(w http.ResponseWriter, r *http.Request, _ *Session, guildID string) {
	if s.economy == nil {
		writeErr(w, http.StatusNotFound, "economy not available")
		return
	}
	gid, ok := parseGuildID(w, guildID)
	if !ok {
		return
	}
	q := r.URL.Query()
	page, err := s.economy.RichLeaderboard(r.Context(), gid, shared.PageQuery{
		Limit:  atoiOr(q.Get("limit"), 25),
		Offset: atoiOr(q.Get("offset"), 0),
	})
	if err != nil {
		s.log.Warn("economy leaderboard failed", zap.Int64("guild", gid), zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "failed to load leaderboard")
		return
	}
	writeJSON(w, http.StatusOK, page)
}

// handleEconShop serves GET /api/guilds/{id}/economy/shop.
func (s *Server) handleEconShop(w http.ResponseWriter, r *http.Request, _ *Session, guildID string) {
	if s.economy == nil {
		writeErr(w, http.StatusNotFound, "economy not available")
		return
	}
	gid, ok := parseGuildID(w, guildID)
	if !ok {
		return
	}
	q := r.URL.Query()
	page, err := s.economy.ListShop(r.Context(), gid, shared.PageQuery{
		Limit:  atoiOr(q.Get("limit"), 50),
		Offset: atoiOr(q.Get("offset"), 0),
	})
	if err != nil {
		s.log.Warn("economy shop list failed", zap.Int64("guild", gid), zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "failed to load shop")
		return
	}
	writeJSON(w, http.StatusOK, page)
}

// handleEconShopAdd serves POST /api/guilds/{id}/economy/shop.
func (s *Server) handleEconShopAdd(w http.ResponseWriter, r *http.Request, sess *Session, guildID string) {
	if s.economy == nil {
		writeErr(w, http.StatusNotFound, "economy not available")
		return
	}
	if !s.checkCSRF(r) {
		writeErr(w, http.StatusForbidden, "bad origin")
		return
	}
	gid, ok := parseGuildID(w, guildID)
	if !ok {
		return
	}
	var body shopItemRequest
	if !decodeBody(w, r, &body) {
		return
	}
	item, err := s.economy.AddShopItem(r.Context(), gid, body.input())
	if err != nil {
		s.writeSeamErr(w, "add shop item", gid, err)
		return
	}
	s.recordAudit(r.Context(), gid, sess, "economy", map[string]any{
		"shop_add": item.ID, "name": item.Name,
	})
	writeJSON(w, http.StatusOK, item)
}

// handleEconShopUpdate serves PATCH /api/guilds/{id}/economy/shop/{item}.
func (s *Server) handleEconShopUpdate(w http.ResponseWriter, r *http.Request, sess *Session, guildID string) {
	if s.economy == nil {
		writeErr(w, http.StatusNotFound, "economy not available")
		return
	}
	if !s.checkCSRF(r) {
		writeErr(w, http.StatusForbidden, "bad origin")
		return
	}
	gid, ok := parseGuildID(w, guildID)
	if !ok {
		return
	}
	itemID, err := strconv.ParseInt(r.PathValue("item"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid item id")
		return
	}
	var body shopItemRequest
	if !decodeBody(w, r, &body) {
		return
	}
	item, err := s.economy.UpdateShopItem(r.Context(), gid, itemID, body.input())
	if err != nil {
		s.writeSeamErr(w, "update shop item", gid, err)
		return
	}
	s.recordAudit(r.Context(), gid, sess, "economy", map[string]any{
		"shop_update": item.ID, "name": item.Name,
	})
	writeJSON(w, http.StatusOK, item)
}

// handleEconShopDelete serves DELETE /api/guilds/{id}/economy/shop/{item}.
func (s *Server) handleEconShopDelete(w http.ResponseWriter, r *http.Request, sess *Session, guildID string) {
	if s.economy == nil {
		writeErr(w, http.StatusNotFound, "economy not available")
		return
	}
	if !s.checkCSRF(r) {
		writeErr(w, http.StatusForbidden, "bad origin")
		return
	}
	gid, ok := parseGuildID(w, guildID)
	if !ok {
		return
	}
	itemID, err := strconv.ParseInt(r.PathValue("item"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid item id")
		return
	}
	if err := s.economy.RemoveShopItem(r.Context(), gid, itemID); err != nil {
		s.writeSeamErr(w, "remove shop item", gid, err)
		return
	}
	s.recordAudit(r.Context(), gid, sess, "economy", map[string]any{"shop_remove": itemID})
	w.WriteHeader(http.StatusNoContent)
}

// --- shared helpers for the economy/leveling seam handlers ---

// parseGuildID parses the {id} path value, writing a 400 and returning false on
// failure.
func parseGuildID(w http.ResponseWriter, guildID string) (int64, bool) {
	gid, err := strconv.ParseInt(guildID, 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid guild id")
		return 0, false
	}
	return gid, true
}

// decodeBody decodes a bounded JSON request body, writing a 400 and returning
// false on failure.
func decodeBody(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(v); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

// writeSeamErr maps a dashboard-seam error to a response: UserError → 400 with
// its message, anything else → logged 500.
func (s *Server) writeSeamErr(w http.ResponseWriter, op string, gid int64, err error) {
	if ue, ok := shared.AsUserError(err); ok {
		writeErr(w, http.StatusBadRequest, ue.Msg)
		return
	}
	s.log.Warn(op+" failed", zap.Int64("guild", gid), zap.Error(err))
	writeErr(w, http.StatusInternalServerError, "request failed")
}
