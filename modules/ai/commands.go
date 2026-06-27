package ai

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/shared"
	"go.uber.org/zap"
)

// askCommand defines /ask, usable by everyone (rate-limited per user).
func (m *Module) askCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:        "ask",
			Description: "Ask the AI assistant a question",
			Options:     []*discordgo.ApplicationCommandOption{strOpt("prompt", "Your question", true)},
		},
		Handler: m.handleAsk,
	}
}

func (m *Module) handleAsk(c *shared.Context) error {
	if !m.svc.Ready() {
		return shared.UserErr("The AI assistant isn't configured on this bot.")
	}
	prompt := strings.TrimSpace(optStr(c, "prompt"))
	if prompt == "" {
		return shared.UserErr("Ask me something.")
	}
	if len([]rune(prompt)) > maxPromptLen {
		return shared.UserErr("That's too long (max %d characters).", maxPromptLen)
	}

	gid := c.GuildID()
	if m.svc.rateLimited(c.Ctx, gid, c.User().ID) {
		return shared.UserErr("You're asking too quickly — give me a few seconds.")
	}

	if err := c.Defer(false); err != nil {
		return err
	}
	answer, err := m.svc.Ask(c.Ctx, gid, prompt)
	if err != nil {
		m.deps.Log.Warn("ask failed", zap.Error(err))
		_, eerr := c.Edit(&discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{ui.ErrorEmbed("The assistant couldn't answer right now. Try again shortly.")},
		})
		return eerr
	}
	_, err = c.Edit(&discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{answerEmbed(prompt, answer, m.svc.Model())},
	})
	return err
}

// aiCommand defines /ai with channel/system/status subcommands (Manage Server).
func (m *Module) aiCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:                     "ai",
			Description:              "Configure the AI assistant",
			DefaultMemberPermissions: permPtr(discordgo.PermissionManageServer),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "channel",
					Description: "Set or clear an opt-in channel where the bot replies to every message",
					Options:     []*discordgo.ApplicationCommandOption{channelOpt("channel", "Assistant channel (omit to disable)", false)},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "system",
					Description: "Set or clear a custom system prompt",
					Options:     []*discordgo.ApplicationCommandOption{strOpt("prompt", "System prompt (omit to reset to default)", false)},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "status",
					Description: "Show the assistant configuration and availability",
				},
			},
		},
		Handler: m.handleAI,
	}
}

func (m *Module) handleAI(c *shared.Context) error {
	if c.GuildID() == "" {
		return shared.UserErr("This command can only be used in a server.")
	}
	if err := m.requirePerm(c, discordgo.PermissionManageServer); err != nil {
		return err
	}
	switch subName(c) {
	case "channel":
		return m.handleSetChannel(c)
	case "system":
		return m.handleSetSystem(c)
	case "status":
		return m.handleStatus(c)
	default:
		return shared.UserErr("Unknown subcommand.")
	}
}

func (m *Module) handleSetChannel(c *shared.Context) error {
	var id int64
	msg := "The assistant channel has been disabled."
	if ch := subChannel(c, "channel"); ch != nil {
		if ch.Type != discordgo.ChannelTypeGuildText {
			return shared.UserErr("The assistant channel must be a text channel.")
		}
		id = pid(ch.ID)
		msg = "I'll now reply to every message in <#" + ch.ID + ">."
		if !m.svc.Ready() {
			msg += "\n\n⚠️ No API key is configured, so replies won't work until the bot owner sets one."
		}
	}
	if err := m.svc.SetChannel(c.Ctx, c.GuildID(), id); err != nil {
		return err
	}
	return reply(c, "Assistant channel updated", msg)
}

func (m *Module) handleSetSystem(c *shared.Context) error {
	prompt := strings.TrimSpace(subStr(c, "prompt"))
	if len([]rune(prompt)) > maxSystemLen {
		return shared.UserErr("That system prompt is too long (max %d characters).", maxSystemLen)
	}
	if err := m.svc.SetSystem(c.Ctx, c.GuildID(), prompt); err != nil {
		return err
	}
	msg := "Custom system prompt set."
	if prompt == "" {
		msg = "System prompt reset to the default."
	}
	return reply(c, "System prompt updated", msg)
}

func (m *Module) handleStatus(c *shared.Context) error {
	set, err := m.svc.Settings(c.Ctx, c.GuildID())
	if err != nil {
		return err
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{statusEmbed(set, m.svc.Ready(), m.svc.Model())},
	}, true)
}

// reply sends a standard ephemeral success reply.
func reply(c *shared.Context, title, msg string) error {
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{ui.SuccessEmbed(title, msg)},
	}, true)
}
