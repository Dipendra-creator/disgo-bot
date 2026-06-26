package moderation

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/pkg/humanize"
	"github.com/dipu-sharma/disgo-bot/shared"
)

// warnsPerPage bounds how many warnings render on a single /warnings page.
const warnsPerPage = 6

func (m *Module) warnCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:                     "warn",
			Description:              "Warn a member and log the reason",
			DefaultMemberPermissions: permPtr(discordgo.PermissionModerateMembers),
			Options: []*discordgo.ApplicationCommandOption{
				userOpt("user", "The member to warn", true),
				strOpt("reason", "Reason for the warning", true),
			},
		},
		Handler: m.handleWarn,
	}
}

func (m *Module) handleWarn(c *shared.Context) error {
	target := optUser(c, "user")
	if target == nil {
		return shared.UserErr("You must specify a user to warn.")
	}
	if err := m.guard(c, target, optMember(c, "user"), discordgo.PermissionModerateMembers); err != nil {
		return err
	}
	reason := strings.TrimSpace(optStr(c, "reason"))
	if reason == "" {
		return shared.UserErr("A reason is required to warn someone.")
	}

	if err := c.Defer(true); err != nil {
		return err
	}
	guild, _ := m.getGuild(c)
	cs, err := m.svc.Warn(c.Ctx, actionInput{
		GuildID:   c.GuildID(),
		GuildName: guildName(guild),
		Target:    target,
		Mod:       c.User(),
		Reason:    reason,
	})
	if err != nil {
		return err
	}
	return m.editActionResult(c, cs, target)
}

func (m *Module) warningsCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:                     "warnings",
			Description:              "List a member's active warnings",
			DefaultMemberPermissions: permPtr(discordgo.PermissionModerateMembers),
			Options: []*discordgo.ApplicationCommandOption{
				userOpt("user", "The member to inspect", true),
			},
		},
		Handler: m.handleWarnings,
	}
}

func (m *Module) handleWarnings(c *shared.Context) error {
	if err := m.requirePerm(c, discordgo.PermissionModerateMembers); err != nil {
		return err
	}
	target := optUser(c, "user")
	if target == nil {
		return shared.UserErr("You must specify a user.")
	}
	cases, err := m.svc.Warnings(c.Ctx, pid(c.GuildID()), pid(target.ID))
	if err != nil {
		return err
	}
	return c.Reply(m.warningsPage(target, cases, 0), true)
}

// handleWarningsPage re-renders the warnings list on pagination. Args: [userID, page].
func (m *Module) handleWarningsPage(c *shared.Context) error {
	if len(c.Args) < 2 {
		return shared.UserErr("Malformed request.")
	}
	targetID := c.Args[0]
	page, _ := strconv.Atoi(c.Args[1])

	target, err := c.Session.User(targetID)
	if err != nil || target == nil {
		target = &discordgo.User{ID: targetID, Username: targetID}
	}
	cases, err := m.svc.Warnings(c.Ctx, pid(c.GuildID()), pid(targetID))
	if err != nil {
		return err
	}
	return c.Update(m.warningsPage(target, cases, page))
}

// warningsPage builds an ephemeral, paginated warnings embed.
func (m *Module) warningsPage(target *discordgo.User, cases []Case, page int) *discordgo.InteractionResponseData {
	total := (len(cases) + warnsPerPage - 1) / warnsPerPage
	if total == 0 {
		return &discordgo.InteractionResponseData{
			Flags:  discordgo.MessageFlagsEphemeral,
			Embeds: []*discordgo.MessageEmbed{ui.EmptyEmbed("No warnings", target.String()+" has a clean record.")},
		}
	}
	if page < 0 {
		page = 0
	}
	if page >= total {
		page = total - 1
	}
	start := page * warnsPerPage
	end := start + warnsPerPage
	if end > len(cases) {
		end = len(cases)
	}

	e := ui.NewEmbed().
		Color(ui.ColorWarning).
		Author("Warnings • "+target.String(), target.AvatarURL("128")).
		Description(fmt.Sprintf("**%d** active warning(s).", len(cases)))
	for _, w := range cases[start:end] {
		e.Field(
			fmt.Sprintf("Case #%d • %s", w.CaseNumber, humanize.RelativeTag(w.CreatedAt)),
			fmt.Sprintf("%s\nBy <@%s>", reasonOrNone(w.Reason), sid(w.ModeratorID)),
			false,
		)
	}

	data := &discordgo.InteractionResponseData{
		Flags:  discordgo.MessageFlagsEphemeral,
		Embeds: []*discordgo.MessageEmbed{e.Footer("disgo • moderation", "").Timestamp().Build()},
	}
	if total > 1 {
		p := ui.Paginator{Module: m.Name(), Action: "warnings", Token: target.ID, Page: page, Total: total}
		data.Components = []discordgo.MessageComponent{p.Row()}
	}
	return data
}

func (m *Module) caseCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:                     "case",
			Description:              "View a moderation case by number",
			DefaultMemberPermissions: permPtr(discordgo.PermissionModerateMembers),
			Options: []*discordgo.ApplicationCommandOption{
				intOpt("number", "The case number", true),
			},
		},
		Handler: m.handleCaseView,
	}
}

func (m *Module) handleCaseView(c *shared.Context) error {
	if err := m.requirePerm(c, discordgo.PermissionModerateMembers); err != nil {
		return err
	}
	number := optInt(c, "number")
	if number <= 0 {
		return shared.UserErr("Provide a valid case number.")
	}
	cs, err := m.svc.GetCase(c.Ctx, pid(c.GuildID()), number)
	if err != nil {
		if errors.Is(err, ErrCaseNotFound) {
			return shared.UserErr("Case #%d not found.", number)
		}
		return err
	}
	target, _ := c.Session.User(sid(cs.TargetID))
	var mod *discordgo.User
	if cs.ModeratorID != 0 {
		mod, _ = c.Session.User(sid(cs.ModeratorID))
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{caseEmbed(cs, target, mod)},
	}, true)
}

func (m *Module) reasonCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:                     "reason",
			Description:              "Update the reason on an existing case",
			DefaultMemberPermissions: permPtr(discordgo.PermissionModerateMembers),
			Options: []*discordgo.ApplicationCommandOption{
				intOpt("number", "The case number", true),
				strOpt("reason", "The new reason", true),
			},
		},
		Handler: m.handleReason,
	}
}

func (m *Module) handleReason(c *shared.Context) error {
	if err := m.requirePerm(c, discordgo.PermissionModerateMembers); err != nil {
		return err
	}
	number := optInt(c, "number")
	reason := strings.TrimSpace(optStr(c, "reason"))
	if number <= 0 {
		return shared.UserErr("Provide a valid case number.")
	}
	if reason == "" {
		return shared.UserErr("Provide a new reason.")
	}
	cs, err := m.svc.EditReason(c.Ctx, pid(c.GuildID()), number, reason)
	if err != nil {
		if errors.Is(err, ErrCaseNotFound) {
			return shared.UserErr("Case #%d not found.", number)
		}
		return err
	}
	target, _ := c.Session.User(sid(cs.TargetID))
	var mod *discordgo.User
	if cs.ModeratorID != 0 {
		mod, _ = c.Session.User(sid(cs.ModeratorID))
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{caseEmbed(cs, target, mod)},
	}, true)
}

func (m *Module) delwarnCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:                     "delwarn",
			Description:              "Remove (deactivate) a warning case",
			DefaultMemberPermissions: permPtr(discordgo.PermissionModerateMembers),
			Options: []*discordgo.ApplicationCommandOption{
				intOpt("number", "The warning case number", true),
			},
		},
		Handler: m.handleDelwarn,
	}
}

func (m *Module) handleDelwarn(c *shared.Context) error {
	if err := m.requirePerm(c, discordgo.PermissionModerateMembers); err != nil {
		return err
	}
	number := optInt(c, "number")
	if number <= 0 {
		return shared.UserErr("Provide a valid case number.")
	}
	_, err := m.svc.DeleteWarning(c.Ctx, pid(c.GuildID()), number)
	if err != nil {
		switch {
		case errors.Is(err, ErrCaseNotFound):
			return shared.UserErr("Case #%d not found.", number)
		case errors.Is(err, errNotWarn):
			return shared.UserErr("Case #%d isn't a warning.", number)
		default:
			return err
		}
	}
	return c.Reply(&discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{ui.SuccessEmbed("Warning removed", fmt.Sprintf("Case #%d is no longer active.", number))},
	}, true)
}
