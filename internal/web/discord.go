package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// discordAPI is the base URL for Discord's REST API.
const discordAPI = "https://discord.com/api"

// permManageGuild is the MANAGE_GUILD permission bit; holders (and owners) may
// configure a guild in the dashboard.
const permManageGuild int64 = 0x20

// apiUser is the subset of GET /users/@me we use.
type apiUser struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	GlobalName string `json:"global_name"`
	Avatar     string `json:"avatar"`
}

// name prefers the global display name, falling back to the username.
func (u apiUser) name() string {
	if u.GlobalName != "" {
		return u.GlobalName
	}
	return u.Username
}

// apiGuild is the subset of GET /users/@me/guilds we use. Permissions is a
// stringified bitfield (Discord serialises it as a string to dodge JS integer
// precision limits).
type apiGuild struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	Owner       bool   `json:"owner"`
	Permissions string `json:"permissions"`
}

// canManage reports whether the user may configure this guild.
func (g apiGuild) canManage() bool {
	if g.Owner {
		return true
	}
	perms, err := strconv.ParseInt(g.Permissions, 10, 64)
	if err != nil {
		return false
	}
	return perms&permManageGuild != 0
}

// fetchUser loads the authenticated user's profile.
func fetchUser(ctx context.Context, client *http.Client) (*apiUser, error) {
	var u apiUser
	if err := getJSON(ctx, client, discordAPI+"/users/@me", &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// fetchGuilds loads the guilds the authenticated user belongs to.
func fetchGuilds(ctx context.Context, client *http.Client) ([]apiGuild, error) {
	var gs []apiGuild
	if err := getJSON(ctx, client, discordAPI+"/users/@me/guilds", &gs); err != nil {
		return nil, err
	}
	return gs, nil
}

// manageableGuilds filters to guilds the user can manage AND where the bot is
// present (so configuring them actually does something). botPresent decouples
// the bot-membership check for testing.
func manageableGuilds(guilds []apiGuild, botPresent func(id string) bool) []GuildBrief {
	out := make([]GuildBrief, 0, len(guilds))
	for _, g := range guilds {
		if !g.canManage() || !botPresent(g.ID) {
			continue
		}
		out = append(out, GuildBrief{ID: g.ID, Name: g.Name, Icon: g.Icon})
	}
	return out
}

// botInGuild reports whether the bot is a member of guildID, using the gateway
// state cache (populated for every guild the bot has joined).
func (s *Server) botInGuild(id string) bool {
	sess := s.deps.Session
	if sess == nil || sess.State == nil {
		return false
	}
	if g, err := sess.State.Guild(id); err == nil && g != nil {
		return true
	}
	return false
}

// getJSON performs a GET and decodes a JSON response, capping the body size.
func getJSON(ctx context.Context, client *http.Client, url string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("discord GET %s: status %d", url, resp.StatusCode)
	}
	return json.Unmarshal(data, dst)
}
