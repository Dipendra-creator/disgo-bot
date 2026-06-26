package ui

import "github.com/bwmarrin/discordgo"

// Standardised "state" embeds give every module a consistent look for the
// common outcomes: success, error, empty, loading.

// SuccessEmbed renders a green success message.
func SuccessEmbed(title, msg string) *discordgo.MessageEmbed {
	return NewEmbed().Color(ColorSuccess).Title(EmojiSuccess + " " + title).Description(msg).Build()
}

// ErrorEmbed renders a red error message.
func ErrorEmbed(msg string) *discordgo.MessageEmbed {
	return NewEmbed().Color(ColorDanger).Title(EmojiError + " Something went wrong").Description(msg).Build()
}

// WarningEmbed renders a yellow warning message.
func WarningEmbed(title, msg string) *discordgo.MessageEmbed {
	return NewEmbed().Color(ColorWarning).Title(EmojiWarning + " " + title).Description(msg).Build()
}

// LoadingEmbed renders a neutral "working…" placeholder for deferred responses.
func LoadingEmbed(msg string) *discordgo.MessageEmbed {
	if msg == "" {
		msg = "Working on it…"
	}
	return NewEmbed().Color(ColorMuted).Title(EmojiLoading + " Please wait").Description(msg).Build()
}

// EmptyEmbed renders a muted empty-state message.
func EmptyEmbed(title, msg string) *discordgo.MessageEmbed {
	return NewEmbed().Color(ColorMuted).Title(EmojiEmpty + " " + title).Description(msg).Build()
}

// ErrorReply wraps ErrorEmbed in ephemeral interaction response data.
func ErrorReply(msg string) *discordgo.InteractionResponseData {
	return &discordgo.InteractionResponseData{
		Flags:  discordgo.MessageFlagsEphemeral,
		Embeds: []*discordgo.MessageEmbed{ErrorEmbed(msg)},
	}
}
