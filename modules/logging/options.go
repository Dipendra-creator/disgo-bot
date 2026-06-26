package logging

import (
	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/shared"
)

func permPtr(p int64) *int64 { return &p }

// categoryChoices is the shared option-choice set for the category argument.
func categoryChoices() []*discordgo.ApplicationCommandOptionChoice {
	return []*discordgo.ApplicationCommandOptionChoice{
		{Name: "Message events (edits, deletions)", Value: CategoryMessage},
		{Name: "Member events (joins, leaves)", Value: CategoryMember},
		{Name: "Server events (bans, channels, roles)", Value: CategoryServer},
	}
}

func categoryOpt() *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        "category",
		Description: "Which group of events to route",
		Required:    true,
		Choices:     categoryChoices(),
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

// subCommand returns the invoked subcommand option, or nil.
func subCommand(c *shared.Context) *discordgo.ApplicationCommandInteractionDataOption {
	opts := c.Event.ApplicationCommandData().Options
	if len(opts) == 0 {
		return nil
	}
	return opts[0]
}

// subName returns the invoked subcommand's name ("" when none).
func subName(c *shared.Context) string {
	if sc := subCommand(c); sc != nil {
		return sc.Name
	}
	return ""
}

// subOpt reads a named option from within the invoked subcommand.
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

func subChannel(c *shared.Context, name string) *discordgo.Channel {
	if o := subOpt(c, name); o != nil {
		return o.ChannelValue(c.Session)
	}
	return nil
}
