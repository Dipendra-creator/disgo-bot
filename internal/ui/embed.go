package ui

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

// Embed is a fluent builder around *discordgo.MessageEmbed that yields polished,
// consistent embeds with minimal boilerplate.
type Embed struct {
	e *discordgo.MessageEmbed
}

// NewEmbed starts a new embed with the brand primary color.
func NewEmbed() *Embed {
	return &Embed{e: &discordgo.MessageEmbed{Color: ColorPrimary}}
}

// Title sets the embed title.
func (b *Embed) Title(t string) *Embed { b.e.Title = t; return b }

// Description sets the embed description/body.
func (b *Embed) Description(d string) *Embed { b.e.Description = d; return b }

// Color overrides the accent color.
func (b *Embed) Color(c int) *Embed { b.e.Color = c; return b }

// URL makes the title a hyperlink.
func (b *Embed) URL(u string) *Embed { b.e.URL = u; return b }

// Field appends a field. Inline groups fields side-by-side.
func (b *Embed) Field(name, value string, inline bool) *Embed {
	b.e.Fields = append(b.e.Fields, &discordgo.MessageEmbedField{
		Name:   name,
		Value:  value,
		Inline: inline,
	})
	return b
}

// Author sets the small author line with an optional icon.
func (b *Embed) Author(name, iconURL string) *Embed {
	b.e.Author = &discordgo.MessageEmbedAuthor{Name: name, IconURL: iconURL}
	return b
}

// Thumbnail sets the top-right thumbnail image.
func (b *Embed) Thumbnail(url string) *Embed {
	if url != "" {
		b.e.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: url}
	}
	return b
}

// Image sets the large bottom image.
func (b *Embed) Image(url string) *Embed {
	if url != "" {
		b.e.Image = &discordgo.MessageEmbedImage{URL: url}
	}
	return b
}

// Footer sets the footer text and optional icon.
func (b *Embed) Footer(text, iconURL string) *Embed {
	b.e.Footer = &discordgo.MessageEmbedFooter{Text: text, IconURL: iconURL}
	return b
}

// Timestamp stamps the embed with the current time.
func (b *Embed) Timestamp() *Embed {
	b.e.Timestamp = time.Now().Format(time.RFC3339)
	return b
}

// Build returns the underlying discordgo embed.
func (b *Embed) Build() *discordgo.MessageEmbed { return b.e }

// Reply wraps the embed in interaction response data ready for Context.Reply.
func (b *Embed) Reply() *discordgo.InteractionResponseData {
	return &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{b.e}}
}
