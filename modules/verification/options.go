package verification

import (
	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/shared"
)

func permPtr(p int64) *int64 { return &p }

func roleOpt(name, desc string, required bool) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionRole,
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

func channelOpt(name, desc string, required bool, types ...discordgo.ChannelType) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type:         discordgo.ApplicationCommandOptionChannel,
		Name:         name,
		Description:  desc,
		Required:     required,
		ChannelTypes: types,
	}
}

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

func optChannel(c *shared.Context, name string) *discordgo.Channel {
	if o := opt(c, name); o != nil {
		return o.ChannelValue(c.Session)
	}
	return nil
}

func optRole(c *shared.Context, name string) *discordgo.Role {
	if o := opt(c, name); o != nil {
		return o.RoleValue(c.Session, c.GuildID())
	}
	return nil
}
