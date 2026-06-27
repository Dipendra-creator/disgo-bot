package economy

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/shared"
)

// Shop-item bounds.
const (
	maxItemName = 64
	maxItemDesc = 200
	maxPrice    = 1_000_000_000
	maxStock    = 1_000_000
)

func (m *Module) adminCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:                     "eco-admin",
			Description:              "Manage balances and the shop (admin)",
			DefaultMemberPermissions: permPtr(discordgo.PermissionManageServer),
			Options: []*discordgo.ApplicationCommandOption{
				subCmd("give", "Add currency to a member's wallet",
					userOpt("user", "The member", true), intOpt("amount", "Amount (may be negative)", true)),
				subCmd("set", "Set a member's wallet balance",
					userOpt("user", "The member", true), intOpt("amount", "New wallet balance", true)),
				subCmd("reset", "Clear a member's balance", userOpt("user", "The member", true)),
				subCmd("shop-add", "Add an item to the shop",
					stringOpt("name", "Item name", true),
					intOpt("price", "Price", true),
					stringOpt("description", "Item description", false),
					roleOpt("role", "Role granted on purchase", false),
					intOpt("stock", "Stock (omit for unlimited)", false)),
				subCmd("shop-remove", "Remove a shop item", stringOpt("name", "Item name", true)),
			},
		},
		Handler: m.handleAdmin,
	}
}

func (m *Module) handleAdmin(c *shared.Context) error {
	if c.GuildID() == "" {
		return errGuildOnly
	}
	if err := m.requirePerm(c, discordgo.PermissionManageServer); err != nil {
		return err
	}
	switch subName(c) {
	case "give":
		return m.handleGive(c)
	case "set":
		return m.handleSet(c)
	case "reset":
		return m.handleReset(c)
	case "shop-add":
		return m.handleShopAdd(c)
	case "shop-remove":
		return m.handleShopRemove(c)
	default:
		return shared.UserErr("Unknown subcommand.")
	}
}

func (m *Module) handleGive(c *shared.Context) error {
	user := subUser(c, "user")
	if user == nil || user.Bot {
		return shared.UserErr("Pick a non-bot member.")
	}
	delta := subInt(c, "amount")
	if delta == 0 {
		return shared.UserErr("Amount can't be zero.")
	}
	set, err := m.svc.Settings(c.Ctx, c.GuildID())
	if err != nil {
		return err
	}
	wallet, err := m.svc.GiveBalance(c.Ctx, c.GuildID(), user.ID, delta)
	if err != nil {
		return err
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{
			ui.SuccessEmbed("Balance updated", fmt.Sprintf("%s now has **%s** in their wallet.", user.Mention(), money(set, wallet))),
		},
	}, true)
}

func (m *Module) handleSet(c *shared.Context) error {
	user := subUser(c, "user")
	if user == nil || user.Bot {
		return shared.UserErr("Pick a non-bot member.")
	}
	amount := subInt(c, "amount")
	if amount < 0 || amount > maxPrice {
		return shared.UserErr("Balance must be between 0 and %d.", maxPrice)
	}
	set, err := m.svc.Settings(c.Ctx, c.GuildID())
	if err != nil {
		return err
	}
	if err := m.svc.SetBalance(c.Ctx, c.GuildID(), user.ID, amount); err != nil {
		return err
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{
			ui.SuccessEmbed("Balance set", fmt.Sprintf("%s's wallet is now **%s**.", user.Mention(), money(set, amount))),
		},
	}, true)
}

func (m *Module) handleReset(c *shared.Context) error {
	user := subUser(c, "user")
	if user == nil || user.Bot {
		return shared.UserErr("Pick a non-bot member.")
	}
	if err := m.svc.ResetAccount(c.Ctx, c.GuildID(), user.ID); err != nil {
		return err
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{
			ui.SuccessEmbed("Balance reset", fmt.Sprintf("%s's balance has been cleared.", user.Mention())),
		},
	}, true)
}

func (m *Module) handleShopAdd(c *shared.Context) error {
	name := strings.TrimSpace(subStr(c, "name"))
	if name == "" || len([]rune(name)) > maxItemName {
		return shared.UserErr("Item name must be 1–%d characters.", maxItemName)
	}
	price := subInt(c, "price")
	if price < 0 || price > maxPrice {
		return shared.UserErr("Price must be between 0 and %d.", maxPrice)
	}
	desc := strings.TrimSpace(subStr(c, "description"))
	if len([]rune(desc)) > maxItemDesc {
		return shared.UserErr("Description must be at most %d characters.", maxItemDesc)
	}
	roleID := ""
	if role := subRole(c, "role"); role != nil {
		if role.Managed || role.ID == c.GuildID() {
			return shared.UserErr("That role can't be granted by a shop item.")
		}
		roleID = role.ID
	}
	stock := -1
	if o := subOpt(c, "stock"); o != nil {
		v := o.IntValue()
		if v < 0 || v > maxStock {
			return shared.UserErr("Stock must be between 0 and %d.", maxStock)
		}
		stock = int(v)
	}

	err := m.svc.AddItem(c.Ctx, c.GuildID(), name, desc, price, roleID, stock)
	if errors.Is(err, ErrItemExists) {
		return shared.UserErr("An item named %q already exists.", name)
	}
	if err != nil {
		return err
	}
	set, err := m.svc.Settings(c.Ctx, c.GuildID())
	if err != nil {
		return err
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{
			ui.SuccessEmbed("Item added", fmt.Sprintf("**%s** is now for sale at **%s**.", name, money(set, price))),
		},
	}, true)
}

func (m *Module) handleShopRemove(c *shared.Context) error {
	name := strings.TrimSpace(subStr(c, "name"))
	if name == "" {
		return shared.UserErr("Specify an item name.")
	}
	removed, err := m.svc.RemoveItem(c.Ctx, c.GuildID(), name)
	if err != nil {
		return err
	}
	if !removed {
		return shared.UserErr("No item named %q exists.", name)
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{ui.SuccessEmbed("Item removed", fmt.Sprintf("**%s** is no longer for sale.", name))},
	}, true)
}
