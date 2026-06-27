package economy

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/pkg/humanize"
)

// itemsPerPage bounds how many rows render on a single shop/leaderboard page.
const itemsPerPage = 8

// money formats an amount with the guild's currency symbol and thousands
// separators, e.g. "🪙 1,250".
func money(set *Settings, amount int64) string {
	return fmt.Sprintf("%s %s", set.CurrencySymbol, humanize.Comma(int(amount)))
}

// balanceEmbed renders a member's wallet/bank breakdown.
func balanceEmbed(set *Settings, u *discordgo.User, a *Account) *discordgo.MessageEmbed {
	return ui.NewEmbed().
		Color(ui.ColorBrand).
		Author(u.String(), u.AvatarURL("128")).
		Thumbnail(u.AvatarURL("256")).
		Field("Wallet", money(set, a.Wallet), true).
		Field("Bank", money(set, a.Bank), true).
		Field("Net worth", money(set, a.Net()), true).
		Footer("disgo • economy", "").Timestamp().Build()
}

// earnEmbed reports a successful daily/work claim.
func earnEmbed(set *Settings, title, flavor string, earned, wallet int64) *discordgo.MessageEmbed {
	return ui.NewEmbed().
		Color(ui.ColorSuccess).
		Title(title).
		Description(flavor).
		Field("Earned", money(set, earned), true).
		Field("Wallet", money(set, wallet), true).
		Footer("disgo • economy", "").Timestamp().Build()
}

// cooldownEmbed tells a member when an earning becomes available again.
func cooldownEmbed(title string, retryAt time.Time) *discordgo.MessageEmbed {
	return ui.NewEmbed().
		Color(ui.ColorWarning).
		Title(ui.EmojiLoading + " " + title).
		Description(fmt.Sprintf("You can claim again %s.", humanize.RelativeTag(retryAt))).
		Build()
}

// payEmbed confirms a wallet-to-wallet transfer.
func payEmbed(set *Settings, from, to *discordgo.User, amount int64) *discordgo.MessageEmbed {
	return ui.NewEmbed().
		Color(ui.ColorSuccess).
		Title(ui.EmojiSuccess+" Payment sent").
		Description(fmt.Sprintf("%s paid %s **%s**.", from.Mention(), to.Mention(), money(set, amount))).
		Footer("disgo • economy", "").Timestamp().Build()
}

// moveEmbed confirms a deposit or withdrawal.
func moveEmbed(set *Settings, deposit bool, amount int64, a *Account) *discordgo.MessageEmbed {
	verb := "Withdrew"
	if deposit {
		verb = "Deposited"
	}
	return ui.NewEmbed().
		Color(ui.ColorSuccess).
		Title(ui.EmojiSuccess+" "+verb).
		Description(fmt.Sprintf("%s **%s**.", verb, money(set, amount))).
		Field("Wallet", money(set, a.Wallet), true).
		Field("Bank", money(set, a.Bank), true).
		Footer("disgo • economy", "").Build()
}

// richEmbed renders one page of the net-worth leaderboard.
func richEmbed(set *Settings, guildName string, rows []Account, page, totalPages, total, offset int) *discordgo.MessageEmbed {
	e := ui.NewEmbed().
		Color(ui.ColorBrand).
		Title("💰 " + guildName + " — Richest members")

	if len(rows) == 0 {
		return e.Description("Nobody has any " + set.CurrencyName + " yet.").Build()
	}

	medals := map[int]string{0: "🥇", 1: "🥈", 2: "🥉"}
	body := ""
	for i, r := range rows {
		pos := offset + i
		badge := medals[pos]
		if badge == "" {
			badge = fmt.Sprintf("`#%d`", pos+1)
		}
		body += fmt.Sprintf("%s <@%s> — **%s**\n", badge, sid(r.UserID), money(set, r.Net()))
	}
	e.Description(body)
	e.Footer(fmt.Sprintf("disgo • economy · page %d/%d · %s ranked", page+1, totalPages, humanize.Comma(total)), "")
	return e.Build()
}

// shopEmbed renders one page of the guild shop.
func shopEmbed(set *Settings, guildName string, rows []ShopItem, page, totalPages, total, offset int) *discordgo.MessageEmbed {
	e := ui.NewEmbed().
		Color(ui.ColorInfo).
		Title("🛒 " + guildName + " — Shop")

	if len(rows) == 0 {
		return e.Description("The shop is empty. Admins can add items with `/eco-admin shop-add`.").Build()
	}

	for _, it := range rows {
		stock := "∞"
		if !it.Unlimited() {
			stock = humanize.Comma(it.Stock)
		}
		title := fmt.Sprintf("%s — %s", it.Name, money(set, it.Price))
		desc := it.Description
		if desc == "" {
			desc = "—"
		}
		extra := fmt.Sprintf("\nStock: %s", stock)
		if it.RoleID != 0 {
			extra += fmt.Sprintf(" · grants <@&%s>", sid(it.RoleID))
		}
		e.Field(title, desc+extra, false)
	}
	e.Footer(fmt.Sprintf("disgo • economy · page %d/%d · %s items · buy with /buy", page+1, totalPages, humanize.Comma(total)), "")
	return e.Build()
}

// buyEmbed confirms a purchase.
func buyEmbed(set *Settings, it *ShopItem, wallet int64) *discordgo.MessageEmbed {
	desc := fmt.Sprintf("You bought **%s** for **%s**.", it.Name, money(set, it.Price))
	if it.RoleID != 0 {
		desc += fmt.Sprintf("\nGranted role <@&%s>.", sid(it.RoleID))
	}
	return ui.NewEmbed().
		Color(ui.ColorSuccess).
		Title(ui.EmojiSuccess+" Purchase complete").
		Description(desc).
		Field("Wallet", money(set, wallet), true).
		Footer("disgo • economy", "").Build()
}

// inventoryEmbed lists a member's owned items.
func inventoryEmbed(set *Settings, u *discordgo.User, items []InventoryItem) *discordgo.MessageEmbed {
	e := ui.NewEmbed().
		Color(ui.ColorInfo).
		Author("Inventory • "+u.String(), u.AvatarURL("128"))
	if len(items) == 0 {
		return e.Description("No items yet. Visit the `/shop`.").Build()
	}
	body := ""
	for _, it := range items {
		body += fmt.Sprintf("**%s** ×%d — worth %s each\n", it.Name, it.Quantity, money(set, it.Price))
	}
	return e.Description(body).Footer("disgo • economy", "").Build()
}

// settingsEmbed summarises the economy configuration.
func settingsEmbed(s *Settings) *discordgo.MessageEmbed {
	return ui.NewEmbed().
		Color(ui.ColorInfo).
		Title("⚙️ Economy configuration").
		Field("Currency", fmt.Sprintf("%s %s", s.CurrencySymbol, s.CurrencyName), true).
		Field("Daily reward", humanize.Comma(int(s.DailyAmount)), true).
		Field("Work reward", fmt.Sprintf("%s–%s", humanize.Comma(int(s.WorkMin)), humanize.Comma(int(s.WorkMax))), true).
		Field("Work cooldown", fmt.Sprintf("%ds", s.WorkCooldownSec), true).
		Field("Starting balance", humanize.Comma(int(s.StartingBalance)), true).
		Footer("disgo • economy", "").Build()
}
