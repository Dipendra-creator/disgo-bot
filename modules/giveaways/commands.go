package giveaways

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/pkg/duration"
	"github.com/dipu-sharma/disgo-bot/shared"
)

// giveawayCommand defines /giveaway with start/end/reroll/list subcommands,
// gated by Manage Server.
func (m *Module) giveawayCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:                     "giveaway",
			Description:              "Create and manage prize giveaways",
			DefaultMemberPermissions: permPtr(discordgo.PermissionManageServer),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "start",
					Description: "Start a new giveaway",
					Options: []*discordgo.ApplicationCommandOption{
						strOpt("prize", "What's being given away", true),
						strOpt("duration", "How long it runs (e.g. 30m, 2h, 7d)", true),
						intOpt("winners", "Number of winners (default 1)", false),
						channelOpt("channel", "Channel to post in (defaults to here)", false),
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "end",
					Description: "End a running giveaway now and draw winners",
					Options:     []*discordgo.ApplicationCommandOption{intOpt("id", "Giveaway ID (see /giveaway list)", true)},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "reroll",
					Description: "Draw fresh winners for an ended giveaway",
					Options: []*discordgo.ApplicationCommandOption{
						intOpt("id", "Giveaway ID", true),
						intOpt("winners", "How many to reroll (default: the original count)", false),
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "list",
					Description: "List the running giveaways",
				},
			},
		},
		Handler: m.handleGiveaway,
	}
}

func (m *Module) handleGiveaway(c *shared.Context) error {
	if c.GuildID() == "" {
		return shared.UserErr("This command can only be used in a server.")
	}
	if err := m.requirePerm(c, discordgo.PermissionManageServer); err != nil {
		return err
	}
	switch subName(c) {
	case "start":
		return m.handleStart(c)
	case "end":
		return m.handleEnd(c)
	case "reroll":
		return m.handleReroll(c)
	case "list":
		return m.handleList(c)
	default:
		return shared.UserErr("Unknown subcommand.")
	}
}

func (m *Module) handleStart(c *shared.Context) error {
	prize := strings.TrimSpace(subStr(c, "prize"))
	if prize == "" {
		return shared.UserErr("Provide a prize.")
	}
	if len(prize) > maxPrizeLen {
		return shared.UserErr("That prize is too long (max %d characters).", maxPrizeLen)
	}

	dur, err := duration.Parse(strings.TrimSpace(subStr(c, "duration")))
	if err != nil {
		return shared.UserErr("Invalid duration. Try formats like 30m, 2h or 7d.")
	}
	if dur < minDuration || dur > maxDuration {
		return shared.UserErr("Duration must be between 1 minute and 90 days.")
	}

	winners := subInt(c, "winners")
	if winners == 0 {
		winners = 1
	}
	if winners < 1 || winners > maxWinners {
		return shared.UserErr("Winners must be between 1 and %d.", maxWinners)
	}

	channelID := c.Event.ChannelID
	if ch := subChannel(c, "channel"); ch != nil {
		if ch.Type != discordgo.ChannelTypeGuildText {
			return shared.UserErr("Giveaways must be posted in a text channel.")
		}
		channelID = ch.ID
	}

	g, err := m.svc.Create(c.Ctx, c.GuildID(), channelID, prize, dur, winners, c.User())
	if err != nil {
		return err
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{ui.SuccessEmbed("Giveaway started",
			fmt.Sprintf("`#%d` **%s** is live in <#%s>, ending %s.", g.ID, g.Prize, channelID, durationHint(dur)))},
	}, true)
}

func (m *Module) handleEnd(c *shared.Context) error {
	id := int64(subInt(c, "id"))
	g, winners, err := m.svc.End(c.Ctx, c.GuildID(), id)
	if err != nil {
		return mapErr(err)
	}
	msg := fmt.Sprintf("`#%d` **%s** ended.", g.ID, g.Prize)
	if len(winners) == 0 {
		msg += " There weren't enough entries to pick a winner."
	} else {
		msg += " Winners: " + mentions(winners)
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{ui.SuccessEmbed("Giveaway ended", msg)},
	}, true)
}

func (m *Module) handleReroll(c *shared.Context) error {
	id := int64(subInt(c, "id"))
	g, winners, err := m.svc.Reroll(c.Ctx, c.GuildID(), id, subInt(c, "winners"))
	if err != nil {
		return mapErr(err)
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{ui.SuccessEmbed("Giveaway rerolled",
			fmt.Sprintf("`#%d` **%s** — new winner(s): %s", g.ID, g.Prize, mentions(winners)))},
	}, true)
}

func (m *Module) handleList(c *shared.Context) error {
	rows, err := m.svc.ListActive(c.Ctx, c.GuildID())
	if err != nil {
		return err
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{listEmbed(m.guildName(c), rows)},
	}, true)
}

// mapErr converts repository sentinels into friendly user-facing errors.
func mapErr(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return shared.UserErr("No giveaway with that ID in this server.")
	case errors.Is(err, ErrEnded):
		return shared.UserErr("That giveaway has already ended.")
	case errors.Is(err, ErrNotEnded):
		return shared.UserErr("That giveaway is still running — end it first to reroll.")
	case errors.Is(err, ErrNoEntries):
		return shared.UserErr("That giveaway had no entries to draw from.")
	default:
		return err
	}
}

// durationHint renders an approximate end time for the start confirmation.
func durationHint(d time.Duration) string { return "in " + duration.Human(d) }
