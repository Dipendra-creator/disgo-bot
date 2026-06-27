package automod

import (
	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/shared"
)

func permPtr(p int64) *int64 { return &p }

func boolOpt(name, desc string, required bool) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionBoolean,
		Name:        name,
		Description: desc,
		Required:    required,
	}
}

func intOpt(name, desc string, required bool) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionInteger,
		Name:        name,
		Description: desc,
		Required:    required,
	}
}

func strOpt(name, desc string, required bool) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        name,
		Description: desc,
		Required:    required,
	}
}

func roleOpt(name, desc string, required bool) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionRole,
		Name:        name,
		Description: desc,
		Required:    required,
	}
}

func channelOpt(name, desc string, required bool) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type:         discordgo.ApplicationCommandOptionChannel,
		Name:         name,
		Description:  desc,
		Required:     required,
		ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText},
	}
}

// actionOpt is the optional escalation-action choice (delete / timeout).
func actionOpt() *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        "action",
		Description: "What to do on a match (default: delete)",
		Required:    false,
		Choices: []*discordgo.ApplicationCommandOptionChoice{
			{Name: "Delete the message", Value: ActionDelete},
			{Name: "Delete and timeout the author", Value: ActionTimeout},
		},
	}
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

func subStr(c *shared.Context, name string) string {
	if o := subOpt(c, name); o != nil {
		return o.StringValue()
	}
	return ""
}

func subBool(c *shared.Context, name string) bool {
	if o := subOpt(c, name); o != nil {
		return o.BoolValue()
	}
	return false
}

// subInt returns the integer option, or 0 when absent.
func subInt(c *shared.Context, name string) int {
	if o := subOpt(c, name); o != nil {
		return int(o.IntValue())
	}
	return 0
}

func subChannel(c *shared.Context, name string) *discordgo.Channel {
	if o := subOpt(c, name); o != nil {
		return o.ChannelValue(c.Session)
	}
	return nil
}

func subRole(c *shared.Context, name string) *discordgo.Role {
	if o := subOpt(c, name); o != nil {
		return o.RoleValue(c.Session, c.GuildID())
	}
	return nil
}
