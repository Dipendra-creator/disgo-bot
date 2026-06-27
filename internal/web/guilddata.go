package web

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/pkg/snowflake"
)

// This file exposes read-only guild metadata — roles, channels and a server
// overview — sourced entirely from the discordgo gateway state cache
// (s.deps.Session.State), never an extra Discord REST call. The frontend uses
// roles/channels to populate real pickers (replacing raw-ID text boxes) and the
// overview to render the landing dashboard. The *discordgo.Guild → view mapping
// is factored into pure functions so it unit-tests without a live session.

// roleView is a guild role as the dashboard needs it.
type roleView struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Color    string `json:"color"` // "#RRGGBB", or "" for the default/inherited color
	Position int    `json:"position"`
	Managed  bool   `json:"managed"`  // managed by an integration (not assignable)
	Hoist    bool   `json:"hoist"`    // displayed separately in the member list
	Everyone bool   `json:"everyone"` // the @everyone role (id == guild id)
}

// channelView is a guild channel as the dashboard needs it.
type channelView struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     int    `json:"type"`                // discordgo.ChannelType
	ParentID string `json:"parent_id,omitempty"` // category the channel sits under
	Position int    `json:"position"`
}

// overview is the server summary shown on the dashboard landing page.
type overview struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Icon        string    `json:"icon,omitempty"`
	OwnerID     string    `json:"owner_id"`
	Members     int       `json:"members"`
	Channels    int       `json:"channels"`
	Roles       int       `json:"roles"`
	PremiumTier int       `json:"premium_tier"`
	Boosts      int       `json:"boosts"`
	CreatedAt   time.Time `json:"created_at"`
}

// colorHex formats a Discord role color integer as "#RRGGBB"; 0 (no color set)
// becomes "" so the frontend can render a neutral swatch.
func colorHex(c int) string {
	if c == 0 {
		return ""
	}
	return fmt.Sprintf("#%06X", c)
}

// rolesView maps a guild's roles to views, highest position first (Discord's own
// ordering). The @everyone role is flagged so the frontend can drop it from
// assignable-role pickers.
func rolesView(g *discordgo.Guild) []roleView {
	out := make([]roleView, 0, len(g.Roles))
	for _, r := range g.Roles {
		if r == nil {
			continue
		}
		out = append(out, roleView{
			ID:       r.ID,
			Name:     r.Name,
			Color:    colorHex(r.Color),
			Position: r.Position,
			Managed:  r.Managed,
			Hoist:    r.Hoist,
			Everyone: r.ID == g.ID,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Position > out[j].Position })
	return out
}

// channelsView maps a guild's channels to views, ordered by position so the
// frontend can group them under their category.
func channelsView(g *discordgo.Guild) []channelView {
	out := make([]channelView, 0, len(g.Channels))
	for _, c := range g.Channels {
		if c == nil {
			continue
		}
		out = append(out, channelView{
			ID:       c.ID,
			Name:     c.Name,
			Type:     int(c.Type),
			ParentID: c.ParentID,
			Position: c.Position,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Position < out[j].Position })
	return out
}

// overviewView builds the server summary. created_at is decoded from the guild
// ID snowflake; a decode failure leaves it zero rather than erroring.
func overviewView(g *discordgo.Guild) overview {
	created, _ := snowflake.Timestamp(g.ID)
	return overview{
		ID:          g.ID,
		Name:        g.Name,
		Icon:        g.Icon,
		OwnerID:     g.OwnerID,
		Members:     g.MemberCount,
		Channels:    len(g.Channels),
		Roles:       len(g.Roles),
		PremiumTier: int(g.PremiumTier),
		Boosts:      g.PremiumSubscriptionCount,
		CreatedAt:   created,
	}
}

// stateGuild returns the cached guild from the gateway state, or false when the
// bot isn't in it / state is unavailable.
func (s *Server) stateGuild(id string) (*discordgo.Guild, bool) {
	sess := s.deps.Session
	if sess == nil || sess.State == nil {
		return nil, false
	}
	g, err := sess.State.Guild(id)
	if err != nil || g == nil {
		return nil, false
	}
	return g, true
}

// handleRoles serves GET /api/guilds/{id}/roles.
func (s *Server) handleRoles(w http.ResponseWriter, _ *http.Request, _ *Session, guildID string) {
	g, ok := s.stateGuild(guildID)
	if !ok {
		writeErr(w, http.StatusNotFound, "server not found")
		return
	}
	writeJSON(w, http.StatusOK, rolesView(g))
}

// handleChannels serves GET /api/guilds/{id}/channels.
func (s *Server) handleChannels(w http.ResponseWriter, _ *http.Request, _ *Session, guildID string) {
	g, ok := s.stateGuild(guildID)
	if !ok {
		writeErr(w, http.StatusNotFound, "server not found")
		return
	}
	writeJSON(w, http.StatusOK, channelsView(g))
}

// handleOverview serves GET /api/guilds/{id}/overview.
func (s *Server) handleOverview(w http.ResponseWriter, _ *http.Request, _ *Session, guildID string) {
	g, ok := s.stateGuild(guildID)
	if !ok {
		writeErr(w, http.StatusNotFound, "server not found")
		return
	}
	writeJSON(w, http.StatusOK, overviewView(g))
}
