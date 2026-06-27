package automod

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/shared"
)

// automodCommand defines /automod with the per-filter and general subcommands,
// gated by Manage Server.
func (m *Module) automodCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:                     "automod",
			Description:              "Configure automatic content moderation",
			DefaultMemberPermissions: permPtr(discordgo.PermissionManageServer),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "status",
					Description: "Show the current automod configuration",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "log",
					Description: "Set or clear the channel automod actions are logged to",
					Options:     []*discordgo.ApplicationCommandOption{channelOpt("channel", "Log channel (omit to disable logging)", false)},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "exempt",
					Description: "Set or clear a role that bypasses every filter",
					Options:     []*discordgo.ApplicationCommandOption{roleOpt("role", "Exempt role (omit to clear)", false)},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "timeout",
					Description: "Set the timeout duration applied by the 'timeout' action",
					Options:     []*discordgo.ApplicationCommandOption{intOpt("seconds", "Timeout length in seconds (10–2419200)", true)},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "words",
					Description: "Toggle the banned-words filter",
					Options:     []*discordgo.ApplicationCommandOption{boolOpt("enabled", "Enable the filter", true), actionOpt()},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "invites",
					Description: "Toggle the invite-link filter",
					Options:     []*discordgo.ApplicationCommandOption{boolOpt("enabled", "Enable the filter", true), actionOpt()},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "mentions",
					Description: "Toggle the mass-mention filter",
					Options: []*discordgo.ApplicationCommandOption{
						boolOpt("enabled", "Enable the filter", true),
						intOpt("threshold", "Trip at this many mentions (min 2)", false),
						actionOpt(),
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "spam",
					Description: "Toggle the spam (message-rate) filter",
					Options: []*discordgo.ApplicationCommandOption{
						boolOpt("enabled", "Enable the filter", true),
						intOpt("count", "Messages allowed in the window (min 2)", false),
						intOpt("per_seconds", "Window length in seconds (1–60)", false),
						actionOpt(),
					},
				},
			},
		},
		Handler: m.handleAutomod,
	}
}

func (m *Module) handleAutomod(c *shared.Context) error {
	if c.GuildID() == "" {
		return shared.UserErr("This command can only be used in a server.")
	}
	if err := m.requirePerm(c, discordgo.PermissionManageServer); err != nil {
		return err
	}
	switch subName(c) {
	case "status":
		return m.handleStatus(c)
	case "log":
		return m.handleLog(c)
	case "exempt":
		return m.handleExempt(c)
	case "timeout":
		return m.handleTimeout(c)
	case "words":
		return m.handleToggleWords(c)
	case "invites":
		return m.handleToggleInvites(c)
	case "mentions":
		return m.handleToggleMentions(c)
	case "spam":
		return m.handleToggleSpam(c)
	default:
		return shared.UserErr("Unknown subcommand.")
	}
}

func (m *Module) handleStatus(c *shared.Context) error {
	set, err := m.svc.Settings(c.Ctx, c.GuildID())
	if err != nil {
		return err
	}
	words, err := m.svc.ListWords(c.Ctx, c.GuildID())
	if err != nil {
		return err
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{statusEmbed(set, len(words))},
	}, true)
}

func (m *Module) handleLog(c *shared.Context) error {
	var id int64
	msg := "AutoMod actions will no longer be logged."
	if ch := subChannel(c, "channel"); ch != nil {
		if ch.Type != discordgo.ChannelTypeGuildText {
			return shared.UserErr("The log channel must be a text channel.")
		}
		id = pid(ch.ID)
		msg = fmt.Sprintf("AutoMod actions will be logged in <#%s>.", ch.ID)
	}
	if err := m.svc.SetLogChannel(c.Ctx, c.GuildID(), id); err != nil {
		return err
	}
	return ok(c, "AutoMod log updated", msg)
}

func (m *Module) handleExempt(c *shared.Context) error {
	var id int64
	msg := "Exempt role cleared."
	if role := subRole(c, "role"); role != nil {
		id = pid(role.ID)
		msg = fmt.Sprintf("Members with <@&%s> now bypass every filter.", role.ID)
	}
	if err := m.svc.SetExemptRole(c.Ctx, c.GuildID(), id); err != nil {
		return err
	}
	return ok(c, "AutoMod exemption updated", msg)
}

func (m *Module) handleTimeout(c *shared.Context) error {
	secs := subInt(c, "seconds")
	if secs < minTimeoutSecs || secs > maxTimeoutSecs {
		return shared.UserErr("Timeout must be between %d and %d seconds.", minTimeoutSecs, maxTimeoutSecs)
	}
	if err := m.svc.SetTimeout(c.Ctx, c.GuildID(), secs); err != nil {
		return err
	}
	return ok(c, "AutoMod timeout updated", fmt.Sprintf("The timeout action will mute for **%ds**.", secs))
}

func (m *Module) handleToggleWords(c *shared.Context) error {
	action, err := actionArg(c)
	if err != nil {
		return err
	}
	if err := m.svc.SetWords(c.Ctx, c.GuildID(), subBool(c, "enabled"), action); err != nil {
		return err
	}
	return ok(c, "Banned-words filter updated", filterState(subBool(c, "enabled"), action)+"\nManage the list with `/automod-words`.")
}

func (m *Module) handleToggleInvites(c *shared.Context) error {
	action, err := actionArg(c)
	if err != nil {
		return err
	}
	if err := m.svc.SetInvites(c.Ctx, c.GuildID(), subBool(c, "enabled"), action); err != nil {
		return err
	}
	return ok(c, "Invite-link filter updated", filterState(subBool(c, "enabled"), action))
}

func (m *Module) handleToggleMentions(c *shared.Context) error {
	action, err := actionArg(c)
	if err != nil {
		return err
	}
	threshold := subInt(c, "threshold")
	if threshold != 0 && threshold < minMentionThreshold {
		return shared.UserErr("The mention threshold must be at least %d.", minMentionThreshold)
	}
	if err := m.svc.SetMentions(c.Ctx, c.GuildID(), subBool(c, "enabled"), threshold, action); err != nil {
		return err
	}
	return ok(c, "Mass-mention filter updated", filterState(subBool(c, "enabled"), action))
}

func (m *Module) handleToggleSpam(c *shared.Context) error {
	action, err := actionArg(c)
	if err != nil {
		return err
	}
	count := subInt(c, "count")
	if count != 0 && count < minSpamCount {
		return shared.UserErr("The spam message count must be at least %d.", minSpamCount)
	}
	window := subInt(c, "per_seconds")
	if window != 0 && (window < minSpamWindowSecs || window > maxSpamWindowSecs) {
		return shared.UserErr("The spam window must be between %d and %d seconds.", minSpamWindowSecs, maxSpamWindowSecs)
	}
	if err := m.svc.SetSpam(c.Ctx, c.GuildID(), subBool(c, "enabled"), count, window, action); err != nil {
		return err
	}
	return ok(c, "Spam filter updated", filterState(subBool(c, "enabled"), action))
}

// wordsCommand defines /automod-words for managing the banned-word list.
func (m *Module) wordsCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:                     "automod-words",
			Description:              "Manage the automod banned-word list",
			DefaultMemberPermissions: permPtr(discordgo.PermissionManageServer),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "add",
					Description: "Add a banned word or phrase",
					Options:     []*discordgo.ApplicationCommandOption{strOpt("word", "The word or phrase to ban", true)},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "remove",
					Description: "Remove a banned word or phrase",
					Options:     []*discordgo.ApplicationCommandOption{strOpt("word", "The word or phrase to unban", true)},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "list",
					Description: "List the banned words",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "clear",
					Description: "Remove every banned word",
				},
			},
		},
		Handler: m.handleWords,
	}
}

func (m *Module) handleWords(c *shared.Context) error {
	if c.GuildID() == "" {
		return shared.UserErr("This command can only be used in a server.")
	}
	if err := m.requirePerm(c, discordgo.PermissionManageServer); err != nil {
		return err
	}
	switch subName(c) {
	case "add":
		return m.handleWordAdd(c)
	case "remove":
		return m.handleWordRemove(c)
	case "list":
		return m.handleWordList(c)
	case "clear":
		return m.handleWordClear(c)
	default:
		return shared.UserErr("Unknown subcommand.")
	}
}

func (m *Module) handleWordAdd(c *shared.Context) error {
	word := normalizeWord(subStr(c, "word"))
	if word == "" {
		return shared.UserErr("Provide a word to ban.")
	}
	if len(word) > maxWordLen {
		return shared.UserErr("That word is too long (max %d characters).", maxWordLen)
	}
	added, err := m.svc.AddWord(c.Ctx, c.GuildID(), word)
	if err != nil {
		return err
	}
	if !added {
		return shared.UserErr("`%s` is already banned.", word)
	}
	return ok(c, "Word banned", fmt.Sprintf("`%s` will now be filtered.", word))
}

func (m *Module) handleWordRemove(c *shared.Context) error {
	word := normalizeWord(subStr(c, "word"))
	removed, err := m.svc.RemoveWord(c.Ctx, c.GuildID(), word)
	if err != nil {
		return err
	}
	if !removed {
		return shared.UserErr("`%s` wasn't on the list.", word)
	}
	return ok(c, "Word unbanned", fmt.Sprintf("`%s` is no longer filtered.", word))
}

func (m *Module) handleWordList(c *shared.Context) error {
	words, err := m.svc.ListWords(c.Ctx, c.GuildID())
	if err != nil {
		return err
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{wordsEmbed(words)},
	}, true)
}

func (m *Module) handleWordClear(c *shared.Context) error {
	n, err := m.svc.ClearWords(c.Ctx, c.GuildID())
	if err != nil {
		return err
	}
	return ok(c, "Word list cleared", fmt.Sprintf("Removed **%d** banned words.", n))
}

// --- helpers ---

// actionArg reads and validates the optional action choice.
func actionArg(c *shared.Context) (string, error) {
	action := subStr(c, "action")
	if action != "" && !validAction(action) {
		return "", shared.UserErr("Unknown action.")
	}
	return action, nil
}

// normalizeWord trims and lowercases a banned term so matching is consistent.
func normalizeWord(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// filterState renders a short enable/action summary for a toggle reply.
func filterState(enabled bool, action string) string {
	if !enabled {
		return "Filter **disabled**."
	}
	if action == "" {
		action = "the configured action"
	}
	return fmt.Sprintf("Filter **enabled** — action: **%s**.", action)
}

// ok sends a standard ephemeral success reply.
func ok(c *shared.Context, title, msg string) error {
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{ui.SuccessEmbed(title, msg)},
	}, true)
}
