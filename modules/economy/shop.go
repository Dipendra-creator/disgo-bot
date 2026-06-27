package economy

import (
	"errors"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/shared"
)

func (m *Module) shopCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:        "shop",
			Description: "Browse the server shop",
		},
		Handler: m.handleShop,
	}
}

func (m *Module) handleShop(c *shared.Context) error {
	if c.GuildID() == "" {
		return errGuildOnly
	}
	data, err := m.shopPage(c, 0)
	if err != nil {
		return err
	}
	return c.Reply(data, false)
}

// handleShopPage re-renders the shop on pagination. Args: [token, page].
func (m *Module) handleShopPage(c *shared.Context) error {
	page := pageArg(c)
	data, err := m.shopPage(c, page)
	if err != nil {
		return err
	}
	return c.Update(data)
}

func (m *Module) shopPage(c *shared.Context, page int) (*discordgo.InteractionResponseData, error) {
	if page < 0 {
		page = 0
	}
	set, err := m.svc.Settings(c.Ctx, c.GuildID())
	if err != nil {
		return nil, err
	}
	rows, total, err := m.svc.Shop(c.Ctx, c.GuildID(), page*itemsPerPage, itemsPerPage)
	if err != nil {
		return nil, err
	}
	totalPages := pageCount(total)
	data := &discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{shopEmbed(set, m.guildName(c), rows, page, totalPages, total, page*itemsPerPage)},
	}
	if totalPages > 1 {
		p := ui.Paginator{Module: m.Name(), Action: "shop", Token: "0", Page: page, Total: totalPages}
		data.Components = []discordgo.MessageComponent{p.Row()}
	}
	return data, nil
}

func (m *Module) buyCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:        "buy",
			Description: "Buy an item from the server shop",
			Options:     []*discordgo.ApplicationCommandOption{stringOpt("item", "The item name", true)},
		},
		Handler: m.handleBuy,
	}
}

func (m *Module) handleBuy(c *shared.Context) error {
	if c.GuildID() == "" {
		return errGuildOnly
	}
	name := optStr(c, "item")
	if name == "" {
		return shared.UserErr("Specify an item name.")
	}
	set, err := m.svc.Settings(c.Ctx, c.GuildID())
	if err != nil {
		return err
	}
	item, wallet, err := m.svc.Buy(c.Ctx, c.GuildID(), c.User().ID, name)
	switch {
	case errors.Is(err, ErrItemNotFound):
		return shared.UserErr("No item named %q is for sale.", name)
	case errors.Is(err, ErrInsufficient):
		return shared.UserErr("You can't afford that.")
	case errors.Is(err, ErrOutOfStock):
		return shared.UserErr("That item is out of stock.")
	case err != nil:
		return err
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{buyEmbed(set, item, wallet)},
	}, false)
}

func (m *Module) inventoryCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:        "inventory",
			Description: "Show your owned items (or another member's)",
			Options:     []*discordgo.ApplicationCommandOption{userOpt("user", "The member to inspect", false)},
		},
		Handler: m.handleInventory,
	}
}

func (m *Module) handleInventory(c *shared.Context) error {
	if c.GuildID() == "" {
		return errGuildOnly
	}
	target := optUser(c, "user")
	if target == nil {
		target = c.User()
	}
	set, err := m.svc.Settings(c.Ctx, c.GuildID())
	if err != nil {
		return err
	}
	items, err := m.svc.Inventory(c.Ctx, c.GuildID(), target.ID)
	if err != nil {
		return err
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{inventoryEmbed(set, target, items)},
	}, false)
}

func (m *Module) richCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:        "rich",
			Description: "Show the richest members",
		},
		Handler: m.handleRich,
	}
}

func (m *Module) handleRich(c *shared.Context) error {
	if c.GuildID() == "" {
		return errGuildOnly
	}
	data, err := m.richPage(c, 0)
	if err != nil {
		return err
	}
	return c.Reply(data, false)
}

// handleRichPage re-renders the leaderboard on pagination. Args: [token, page].
func (m *Module) handleRichPage(c *shared.Context) error {
	page := pageArg(c)
	data, err := m.richPage(c, page)
	if err != nil {
		return err
	}
	return c.Update(data)
}

func (m *Module) richPage(c *shared.Context, page int) (*discordgo.InteractionResponseData, error) {
	if page < 0 {
		page = 0
	}
	set, err := m.svc.Settings(c.Ctx, c.GuildID())
	if err != nil {
		return nil, err
	}
	rows, total, err := m.svc.Rich(c.Ctx, c.GuildID(), page*itemsPerPage, itemsPerPage)
	if err != nil {
		return nil, err
	}
	totalPages := pageCount(total)
	data := &discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{richEmbed(set, m.guildName(c), rows, page, totalPages, total, page*itemsPerPage)},
	}
	if totalPages > 1 {
		p := ui.Paginator{Module: m.Name(), Action: "rich", Token: "0", Page: page, Total: totalPages}
		data.Components = []discordgo.MessageComponent{p.Row()}
	}
	return data, nil
}

// --- helpers ---

func pageArg(c *shared.Context) int {
	if len(c.Args) >= 2 {
		page, _ := strconv.Atoi(c.Args[1])
		return page
	}
	return 0
}

func pageCount(total int) int {
	n := (total + itemsPerPage - 1) / itemsPerPage
	if n == 0 {
		return 1
	}
	return n
}

func (m *Module) guildName(c *shared.Context) string {
	g := m.getGuild(c)
	if g != nil && g.Name != "" {
		return g.Name
	}
	return "Server"
}
