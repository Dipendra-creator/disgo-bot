package verification

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/shared"
)

// Bounds on admin-supplied free text (Discord button labels cap at 80 chars).
const (
	maxMessageLen = 1500
	maxLabelLen   = 80
)

func (m *Module) setupCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:                     "verify-setup",
			Description:              "Configure the verification gate for this server",
			DefaultMemberPermissions: permPtr(discordgo.PermissionManageServer),
			Options: []*discordgo.ApplicationCommandOption{
				roleOpt("role", "Role granted to members when they verify", true),
				channelOpt("log_channel", "Channel where verifications are logged", false, discordgo.ChannelTypeGuildText),
				strOpt("message", "Panel description shown above the button", false),
				strOpt("button_label", "Text on the verify button (max 80 chars)", false),
			},
		},
		Handler: m.handleSetup,
	}
}

func (m *Module) handleSetup(c *shared.Context) error {
	if err := m.requirePerm(c, discordgo.PermissionManageServer); err != nil {
		return err
	}

	role := optRole(c, "role")
	if role == nil {
		return shared.UserErr("Pick a role to grant on verification.")
	}
	if role.Managed || role.ID == c.GuildID() {
		return shared.UserErr("That role can't be assigned. Choose a normal, self-assignable role.")
	}

	var logID int64
	if lc := optChannel(c, "log_channel"); lc != nil {
		if lc.Type != discordgo.ChannelTypeGuildText {
			return shared.UserErr("The log channel must be a text channel.")
		}
		logID = pid(lc.ID)
	}

	message := strings.TrimSpace(optStr(c, "message"))
	if message == "" {
		message = defaultMessage
	}
	if len(message) > maxMessageLen {
		return shared.UserErr("The message is too long (max %d characters).", maxMessageLen)
	}
	label := strings.TrimSpace(optStr(c, "button_label"))
	if label == "" {
		label = defaultButtonLabel
	}
	if len(label) > maxLabelLen {
		return shared.UserErr("The button label is too long (max %d characters).", maxLabelLen)
	}

	if err := m.svc.Setup(c.Ctx, c.GuildID(), pid(role.ID), logID, message, label); err != nil {
		return err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Verified role: <@&%s>\n", role.ID)
	if logID != 0 {
		fmt.Fprintf(&b, "Log channel: <#%s>\n", sid(logID))
	} else {
		b.WriteString("Log channel: *none*\n")
	}
	b.WriteString("\nPost a panel with `/verify-panel`.")

	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{ui.SuccessEmbed("Verification configured", b.String())},
	}, true)
}

func (m *Module) panelCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:                     "verify-panel",
			Description:              "Post a verification panel with a verify button",
			DefaultMemberPermissions: permPtr(discordgo.PermissionManageServer),
			Options: []*discordgo.ApplicationCommandOption{
				channelOpt("channel", "Channel to post the panel in (defaults to here)", false, discordgo.ChannelTypeGuildText),
				strOpt("title", "Panel title", false),
				strOpt("description", "Panel description", false),
			},
		},
		Handler: m.handlePanel,
	}
}

func (m *Module) handlePanel(c *shared.Context) error {
	if err := m.requirePerm(c, discordgo.PermissionManageServer); err != nil {
		return err
	}

	set, err := m.svc.Settings(c.Ctx, c.GuildID())
	if err != nil {
		return err
	}

	target := c.Event.ChannelID
	if ch := optChannel(c, "channel"); ch != nil {
		target = ch.ID
	}
	title := strings.TrimSpace(optStr(c, "title"))
	if title == "" {
		title = defaultPanelTitle
	}
	desc := strings.TrimSpace(optStr(c, "description"))
	if desc == "" {
		desc = set.Message
	}

	if err := m.svc.PostPanel(c.Ctx, c.GuildID(), target, title, desc, set.ButtonLabel); err != nil {
		return err
	}

	msg := fmt.Sprintf("Panel posted in <#%s>.", target)
	if !set.Configured() {
		msg += "\n\n⚠️ Verification isn't enabled yet — run `/verify-setup` so the button works."
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{ui.SuccessEmbed("Panel posted", msg)},
	}, true)
}

func (m *Module) disableCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:                     "verify-disable",
			Description:              "Disable the verification gate (config is kept)",
			DefaultMemberPermissions: permPtr(discordgo.PermissionManageServer),
		},
		Handler: m.handleDisable,
	}
}

func (m *Module) handleDisable(c *shared.Context) error {
	if err := m.requirePerm(c, discordgo.PermissionManageServer); err != nil {
		return err
	}
	if err := m.svc.SetEnabled(c.Ctx, c.GuildID(), false); err != nil {
		return err
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{ui.SuccessEmbed("Verification disabled", "Existing verify buttons will stop granting the role. Re-enable with `/verify-setup`.")},
	}, true)
}

func (m *Module) statusCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:                     "verify-status",
			Description:              "Show the current verification configuration",
			DefaultMemberPermissions: permPtr(discordgo.PermissionManageServer),
		},
		Handler: m.handleStatus,
	}
}

func (m *Module) handleStatus(c *shared.Context) error {
	if err := m.requirePerm(c, discordgo.PermissionManageServer); err != nil {
		return err
	}
	set, err := m.svc.Settings(c.Ctx, c.GuildID())
	if err != nil {
		return err
	}
	count, err := m.svc.VerifiedCount(c.Ctx, c.GuildID())
	if err != nil {
		return err
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{statusEmbed(set, count)},
	}, true)
}
