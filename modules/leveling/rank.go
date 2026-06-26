package leveling

import (
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/shared"
)

func (m *Module) rankCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:        "rank",
			Description: "Show your level and rank (or another member's)",
			Options:     []*discordgo.ApplicationCommandOption{userOpt("user", "The member to inspect", false)},
		},
		Handler: m.handleRank,
	}
}

func (m *Module) handleRank(c *shared.Context) error {
	if c.GuildID() == "" {
		return shared.UserErr("This command can only be used in a server.")
	}
	target := optUser(c, "user")
	if target == nil {
		target = c.User()
	}
	if target.Bot {
		return shared.UserErr("Bots don't earn XP.")
	}
	u, prog, rank, err := m.svc.Rank(c.Ctx, c.GuildID(), target.ID)
	if err != nil {
		return err
	}
	_, total, err := m.svc.Leaderboard(c.Ctx, c.GuildID(), 0, 1)
	if err != nil {
		return err
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{rankEmbed(target, u, prog, rank, total)},
	}, false)
}

func (m *Module) leaderboardCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:        "leaderboard",
			Description: "Show the server XP leaderboard",
		},
		Handler: m.handleLeaderboard,
	}
}

func (m *Module) handleLeaderboard(c *shared.Context) error {
	if c.GuildID() == "" {
		return shared.UserErr("This command can only be used in a server.")
	}
	data, err := m.leaderboardPage(c, 0)
	if err != nil {
		return err
	}
	return c.Reply(data, false)
}

// handleLeaderboardPage re-renders the leaderboard on pagination. Args: [token, page].
func (m *Module) handleLeaderboardPage(c *shared.Context) error {
	page := 0
	if len(c.Args) >= 2 {
		page, _ = strconv.Atoi(c.Args[1])
	}
	data, err := m.leaderboardPage(c, page)
	if err != nil {
		return err
	}
	return c.Update(data)
}

func (m *Module) leaderboardPage(c *shared.Context, page int) (*discordgo.InteractionResponseData, error) {
	if page < 0 {
		page = 0
	}
	rows, total, err := m.svc.Leaderboard(c.Ctx, c.GuildID(), page*usersPerPage, usersPerPage)
	if err != nil {
		return nil, err
	}
	totalPages := (total + usersPerPage - 1) / usersPerPage
	if totalPages == 0 {
		totalPages = 1
	}

	guild := m.getGuild(c)
	name := "Server"
	if guild != nil && guild.Name != "" {
		name = guild.Name
	}

	data := &discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{leaderboardEmbed(name, rows, page, totalPages, total, page*usersPerPage)},
	}
	if totalPages > 1 {
		p := ui.Paginator{Module: m.Name(), Action: "lb", Token: "0", Page: page, Total: totalPages}
		data.Components = []discordgo.MessageComponent{p.Row()}
	}
	return data, nil
}
