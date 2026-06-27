package giveaways

import (
	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/shared"
)

func permPtr(p int64) *int64 { return &p }

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

func channelOpt(name, desc string, required bool) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type:         discordgo.ApplicationCommandOptionChannel,
		Name:         name,
		Description:  desc,
		Required:     required,
		ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText},
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
