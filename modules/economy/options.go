package economy

import (
	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/shared"
)

func permPtr(p int64) *int64 { return &p }

func userOpt(name, desc string, required bool) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type: discordgo.ApplicationCommandOptionUser, Name: name, Description: desc, Required: required,
	}
}

func intOpt(name, desc string, required bool) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type: discordgo.ApplicationCommandOptionInteger, Name: name, Description: desc, Required: required,
	}
}

func stringOpt(name, desc string, required bool) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type: discordgo.ApplicationCommandOptionString, Name: name, Description: desc, Required: required,
	}
}

func roleOpt(name, desc string, required bool) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type: discordgo.ApplicationCommandOptionRole, Name: name, Description: desc, Required: required,
	}
}

func subCmd(name, desc string, opts ...*discordgo.ApplicationCommandOption) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type: discordgo.ApplicationCommandOptionSubCommand, Name: name, Description: desc, Options: opts,
	}
}

// --- top-level option readers ---

func opt(c *shared.Context, name string) *discordgo.ApplicationCommandInteractionDataOption {
	for _, o := range c.Event.ApplicationCommandData().Options {
		if o.Name == name {
			return o
		}
	}
	return nil
}

func optUser(c *shared.Context, name string) *discordgo.User {
	if o := opt(c, name); o != nil {
		return o.UserValue(c.Session)
	}
	return nil
}

func optInt(c *shared.Context, name string) int64 {
	if o := opt(c, name); o != nil {
		return o.IntValue()
	}
	return 0
}

func optStr(c *shared.Context, name string) string {
	if o := opt(c, name); o != nil {
		return o.StringValue()
	}
	return ""
}

// --- subcommand option readers ---

func subCommand(c *shared.Context) *discordgo.ApplicationCommandInteractionDataOption {
	opts := c.Event.ApplicationCommandData().Options
	if len(opts) == 0 {
		return nil
	}
	return opts[0]
}

func subName(c *shared.Context) string {
	if sc := subCommand(c); sc != nil {
		return sc.Name
	}
	return ""
}

func subOpt(c *shared.Context, name string) *discordgo.ApplicationCommandInteractionDataOption {
	sc := subCommand(c)
	if sc == nil {
		return nil
	}
	for _, o := range sc.Options {
		if o.Name == name {
			return o
		}
	}
	return nil
}

func subInt(c *shared.Context, name string) int64 {
	if o := subOpt(c, name); o != nil {
		return o.IntValue()
	}
	return 0
}

func subStr(c *shared.Context, name string) string {
	if o := subOpt(c, name); o != nil {
		return o.StringValue()
	}
	return ""
}

func subUser(c *shared.Context, name string) *discordgo.User {
	if o := subOpt(c, name); o != nil {
		return o.UserValue(c.Session)
	}
	return nil
}

func subRole(c *shared.Context, name string) *discordgo.Role {
	if o := subOpt(c, name); o != nil {
		return o.RoleValue(c.Session, c.GuildID())
	}
	return nil
}
