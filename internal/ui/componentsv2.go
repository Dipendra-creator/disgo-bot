package ui

import "github.com/bwmarrin/discordgo"

// This file wraps Discord's Components-v2 layout primitives. A v2 message
// replaces the classic content+embeds model with a tree of layout components
// (containers, sections, text displays, separators) and MUST carry the
// MessageFlagsIsComponentsV2 flag — content and embeds are then disallowed.

// Container is a v2 layout block with an optional colored accent bar.
func Container(accentColor int, components ...discordgo.MessageComponent) discordgo.Container {
	c := discordgo.Container{Components: components}
	if accentColor >= 0 {
		ac := accentColor
		c.AccentColor = &ac
	}
	return c
}

// Text is a markdown text-display component.
func Text(content string) discordgo.TextDisplay {
	return discordgo.TextDisplay{Content: content}
}

// Section joins one to three text displays with a trailing accessory (a Button
// or Thumbnail shown to the right).
func Section(accessory discordgo.MessageComponent, lines ...string) discordgo.Section {
	comps := make([]discordgo.MessageComponent, 0, len(lines))
	for _, l := range lines {
		comps = append(comps, Text(l))
	}
	return discordgo.Section{Components: comps, Accessory: accessory}
}

// Separator adds vertical spacing with a divider line between components.
func Separator() discordgo.Separator {
	divider := true
	spacing := discordgo.SeparatorSpacingSizeSmall
	return discordgo.Separator{Divider: &divider, Spacing: &spacing}
}

// Thumbnail is a small image usable as a Section accessory.
func Thumbnail(url string) discordgo.Thumbnail {
	return discordgo.Thumbnail{Media: discordgo.UnfurledMediaItem{URL: url}}
}

// V2 wraps top-level v2 components into interaction response data, setting the
// required Components-v2 flag.
func V2(components ...discordgo.MessageComponent) *discordgo.InteractionResponseData {
	return &discordgo.InteractionResponseData{
		Flags:      discordgo.MessageFlagsIsComponentsV2,
		Components: components,
	}
}

// V2Edit wraps top-level v2 components into a WebhookEdit for editing a
// previously-deferred response, setting the required flag.
func V2Edit(components ...discordgo.MessageComponent) *discordgo.WebhookEdit {
	c := components
	return &discordgo.WebhookEdit{
		Flags:      discordgo.MessageFlagsIsComponentsV2,
		Components: &c,
	}
}
