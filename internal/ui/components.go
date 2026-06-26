package ui

import "github.com/bwmarrin/discordgo"

// Row groups interactive components into an ActionsRow (max 5 components).
func Row(components ...discordgo.MessageComponent) discordgo.ActionsRow {
	return discordgo.ActionsRow{Components: components}
}

// button builds a styled button with an optional unicode emoji glyph.
func button(style discordgo.ButtonStyle, customID, label, emoji string) discordgo.Button {
	b := discordgo.Button{Style: style, CustomID: customID, Label: label}
	if emoji != "" {
		b.Emoji = &discordgo.ComponentEmoji{Name: emoji}
	}
	return b
}

// PrimaryButton is a blurple call-to-action button.
func PrimaryButton(customID, label, emoji string) discordgo.Button {
	return button(discordgo.PrimaryButton, customID, label, emoji)
}

// SecondaryButton is a neutral grey button.
func SecondaryButton(customID, label, emoji string) discordgo.Button {
	return button(discordgo.SecondaryButton, customID, label, emoji)
}

// SuccessButton is a green confirm button.
func SuccessButton(customID, label, emoji string) discordgo.Button {
	return button(discordgo.SuccessButton, customID, label, emoji)
}

// DangerButton is a red destructive button.
func DangerButton(customID, label, emoji string) discordgo.Button {
	return button(discordgo.DangerButton, customID, label, emoji)
}

// LinkButton navigates to an external URL (has no custom ID).
func LinkButton(label, url, emoji string) discordgo.Button {
	b := discordgo.Button{Style: discordgo.LinkButton, Label: label, URL: url}
	if emoji != "" {
		b.Emoji = &discordgo.ComponentEmoji{Name: emoji}
	}
	return b
}

// Disabled returns a copy of the button marked disabled.
func Disabled(b discordgo.Button) discordgo.Button {
	b.Disabled = true
	return b
}

// StringSelect builds a string select menu.
func StringSelect(customID, placeholder string, options ...discordgo.SelectMenuOption) discordgo.SelectMenu {
	return discordgo.SelectMenu{
		MenuType:    discordgo.StringSelectMenu,
		CustomID:    customID,
		Placeholder: placeholder,
		Options:     options,
	}
}

// Option builds a single string-select option.
func Option(label, value, description, emoji string) discordgo.SelectMenuOption {
	o := discordgo.SelectMenuOption{Label: label, Value: value, Description: description}
	if emoji != "" {
		o.Emoji = &discordgo.ComponentEmoji{Name: emoji}
	}
	return o
}

// TextInput builds a modal text input wrapped in its required ActionsRow.
func TextInput(customID, label string, style discordgo.TextInputStyle, required bool) discordgo.ActionsRow {
	return Row(discordgo.TextInput{
		CustomID: customID,
		Label:    label,
		Style:    style,
		Required: &required,
	})
}
