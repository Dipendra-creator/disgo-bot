package moderation

import (
	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/shared"
)

// Option/command construction helpers keep the command definitions terse.

func permPtr(p int64) *int64 { return &p }

func floatPtr(f float64) *float64 { return &f }

func userOpt(name, desc string, required bool) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionUser,
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

func intOpt(name, desc string, required bool) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionInteger,
		Name:        name,
		Description: desc,
		Required:    required,
	}
}

// Option readers operate on the invoked command's options.

func opt(c *shared.Context, name string) *discordgo.ApplicationCommandInteractionDataOption {
	for _, o := range c.Event.ApplicationCommandData().Options {
		if o.Name == name {
			return o
		}
	}
	return nil
}

func optStr(c *shared.Context, name string) string {
	if o := opt(c, name); o != nil {
		return o.StringValue()
	}
	return ""
}

func optInt(c *shared.Context, name string) int64 {
	if o := opt(c, name); o != nil {
		return o.IntValue()
	}
	return 0
}

func optUser(c *shared.Context, name string) *discordgo.User {
	o := opt(c, name)
	if o == nil {
		return nil
	}
	return o.UserValue(c.Session)
}

// optMember returns the resolved guild member for a user option, or nil when the
// user isn't a member of the guild.
func optMember(c *shared.Context, name string) *discordgo.Member {
	o := opt(c, name)
	if o == nil {
		return nil
	}
	data := c.Event.ApplicationCommandData()
	if data.Resolved == nil || data.Resolved.Members == nil {
		return nil
	}
	id, _ := o.Value.(string)
	mem := data.Resolved.Members[id]
	if mem != nil && mem.User == nil {
		mem.User = o.UserValue(c.Session)
	}
	return mem
}

func optChannel(c *shared.Context, name string) *discordgo.Channel {
	o := opt(c, name)
	if o == nil {
		return nil
	}
	return o.ChannelValue(c.Session)
}
