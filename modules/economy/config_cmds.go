package economy

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/shared"
)

// Configuration bounds keep settings sane.
const (
	maxName        = 32
	maxSymbol      = 16
	maxReward      = 1_000_000
	maxCooldownSec = 86400
)

func (m *Module) configCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:                     "eco-config",
			Description:              "Configure the economy system",
			DefaultMemberPermissions: permPtr(discordgo.PermissionManageServer),
			Options: []*discordgo.ApplicationCommandOption{
				subCmd("status", "Show the current configuration"),
				subCmd("currency", "Set the currency name and symbol",
					stringOpt("name", "Currency name (e.g. coins)", true),
					stringOpt("symbol", "Currency symbol/emoji", true)),
				subCmd("daily", "Set the daily reward amount", intOpt("amount", "Reward per /daily", true)),
				subCmd("work", "Set the work reward range and cooldown",
					intOpt("min", "Minimum reward", true),
					intOpt("max", "Maximum reward", true),
					intOpt("cooldown", "Cooldown in seconds", true)),
				subCmd("starting", "Set the balance new members start with", intOpt("amount", "Starting balance", true)),
			},
		},
		Handler: m.handleConfig,
	}
}

func (m *Module) handleConfig(c *shared.Context) error {
	if c.GuildID() == "" {
		return errGuildOnly
	}
	if err := m.requirePerm(c, discordgo.PermissionManageServer); err != nil {
		return err
	}
	set, err := m.svc.Settings(c.Ctx, c.GuildID())
	if err != nil {
		return err
	}

	switch subName(c) {
	case "status":
		// no change
	case "currency":
		if err := applyCurrency(c, set); err != nil {
			return err
		}
	case "daily":
		amount := subInt(c, "amount")
		if amount < 1 || amount > maxReward {
			return shared.UserErr("Daily amount must be between 1 and %d.", maxReward)
		}
		set.DailyAmount = amount
	case "work":
		if err := applyWork(c, set); err != nil {
			return err
		}
	case "starting":
		amount := subInt(c, "amount")
		if amount < 0 || amount > maxReward {
			return shared.UserErr("Starting balance must be between 0 and %d.", maxReward)
		}
		set.StartingBalance = amount
	default:
		return shared.UserErr("Unknown subcommand.")
	}

	if subName(c) != "status" {
		if err := m.svc.SaveSettings(c.Ctx, set); err != nil {
			return err
		}
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{settingsEmbed(set)},
	}, true)
}

func applyCurrency(c *shared.Context, set *Settings) error {
	name := strings.TrimSpace(subStr(c, "name"))
	symbol := strings.TrimSpace(subStr(c, "symbol"))
	if name == "" || len([]rune(name)) > maxName {
		return shared.UserErr("Currency name must be 1–%d characters.", maxName)
	}
	if symbol == "" || len([]rune(symbol)) > maxSymbol {
		return shared.UserErr("Currency symbol must be 1–%d characters.", maxSymbol)
	}
	set.CurrencyName, set.CurrencySymbol = name, symbol
	return nil
}

func applyWork(c *shared.Context, set *Settings) error {
	min, max, cooldown := subInt(c, "min"), subInt(c, "max"), subInt(c, "cooldown")
	if min < 0 || max < min || max > maxReward {
		return shared.UserErr("Provide 0 ≤ min ≤ max ≤ %d.", maxReward)
	}
	if cooldown < 0 || cooldown > maxCooldownSec {
		return shared.UserErr("Cooldown must be between 0 and %d seconds.", maxCooldownSec)
	}
	set.WorkMin, set.WorkMax, set.WorkCooldownSec = min, max, int(cooldown)
	return nil
}
