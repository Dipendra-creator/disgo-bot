package leveling

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/pkg/humanize"
	"github.com/dipu-sharma/disgo-bot/shared"
)

func (m *Module) xpCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:                     "xp",
			Description:              "Adjust a member's XP (admin)",
			DefaultMemberPermissions: permPtr(discordgo.PermissionManageServer),
			Options: []*discordgo.ApplicationCommandOption{
				subCmd("give", "Add XP to a member",
					userOpt("user", "The member", true), intOpt("amount", "XP to add (may be negative)", true)),
				subCmd("set", "Set a member's total XP",
					userOpt("user", "The member", true), intOpt("amount", "New XP total", true)),
				subCmd("reset", "Clear a member's XP", userOpt("user", "The member", true)),
			},
		},
		Handler: m.handleXP,
	}
}

func (m *Module) handleXP(c *shared.Context) error {
	if c.GuildID() == "" {
		return shared.UserErr("This command can only be used in a server.")
	}
	if err := m.requirePerm(c, discordgo.PermissionManageServer); err != nil {
		return err
	}
	user := subUser(c, "user")
	if user == nil {
		return shared.UserErr("You must specify a member.")
	}
	if user.Bot {
		return shared.UserErr("Bots don't earn XP.")
	}

	switch subName(c) {
	case "give":
		ul, err := m.svc.GiveXP(c.Ctx, c.GuildID(), user.ID, subInt(c, "amount"))
		if err != nil {
			return err
		}
		return m.xpResult(c, user, ul, "XP updated")
	case "set":
		amount := subInt(c, "amount")
		if amount < 0 {
			return shared.UserErr("XP can't be negative.")
		}
		ul, err := m.svc.SetXP(c.Ctx, c.GuildID(), user.ID, amount)
		if err != nil {
			return err
		}
		return m.xpResult(c, user, ul, "XP set")
	case "reset":
		if err := m.svc.ResetUser(c.Ctx, c.GuildID(), user.ID); err != nil {
			return err
		}
		return c.Reply(&discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{ui.SuccessEmbed("XP reset", fmt.Sprintf("%s is back to 0 XP.", user.Mention()))},
		}, true)
	default:
		return shared.UserErr("Unknown subcommand.")
	}
}

func (m *Module) xpResult(c *shared.Context, user *discordgo.User, ul *UserLevel, title string) error {
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{
			ui.SuccessEmbed(title, fmt.Sprintf("%s is now **level %d** with %s XP.", user.Mention(), ul.Level, humanize.Comma(int(ul.XP)))),
		},
	}, true)
}
