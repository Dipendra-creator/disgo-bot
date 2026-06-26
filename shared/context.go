package shared

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

// Context carries everything a handler needs for one interaction and provides
// ergonomic response helpers over the raw discordgo session. It is created fresh
// per interaction by the router.
type Context struct {
	// Ctx is the request-scoped context (carries cancellation/timeout).
	Ctx context.Context
	// Session is the active gateway session.
	Session *discordgo.Session
	// Event is the incoming interaction.
	Event *discordgo.InteractionCreate
	// Deps grants access to shared dependencies.
	Deps *Deps
	// Log is a logger pre-tagged with interaction metadata.
	Log *zap.Logger
	// Args holds the custom-ID arguments (segments after "<module>:<action>")
	// for component and modal handlers. Nil for slash commands.
	Args []string
}

// interaction returns the underlying *discordgo.Interaction.
func (c *Context) interaction() *discordgo.Interaction { return c.Event.Interaction }

// GuildID returns the guild the interaction originated in ("" in DMs).
func (c *Context) GuildID() string { return c.Event.GuildID }

// User returns the invoking user, whether in a guild (Member) or DM.
func (c *Context) User() *discordgo.User {
	if c.Event.Member != nil && c.Event.Member.User != nil {
		return c.Event.Member.User
	}
	return c.Event.User
}

// Member returns the invoking guild member, or nil in DMs.
func (c *Context) Member() *discordgo.Member { return c.Event.Member }

// Respond sends an immediate response to the interaction.
func (c *Context) Respond(resp *discordgo.InteractionResponse) error {
	return c.Session.InteractionRespond(c.interaction(), resp)
}

// Reply sends a channel message in response to the interaction. Set ephemeral to
// make it visible only to the invoker.
func (c *Context) Reply(data *discordgo.InteractionResponseData, ephemeral bool) error {
	if ephemeral {
		data.Flags |= discordgo.MessageFlagsEphemeral
	}
	return c.Respond(&discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: data,
	})
}

// Defer acknowledges the interaction, giving up to 15 minutes to follow up with
// the real response via Edit. Set ephemeral for a private deferral.
func (c *Context) Defer(ephemeral bool) error {
	resp := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{},
	}
	if ephemeral {
		resp.Data.Flags = discordgo.MessageFlagsEphemeral
	}
	return c.Respond(resp)
}

// DeferUpdate acknowledges a component interaction without changing the message
// (the handler may edit it later).
func (c *Context) DeferUpdate() error {
	return c.Respond(&discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
}

// Update edits the message a component is attached to, in place.
func (c *Context) Update(data *discordgo.InteractionResponseData) error {
	return c.Respond(&discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: data,
	})
}

// Edit replaces the (possibly deferred) original interaction response.
func (c *Context) Edit(edit *discordgo.WebhookEdit) (*discordgo.Message, error) {
	return c.Session.InteractionResponseEdit(c.interaction(), edit)
}

// Followup posts an additional message after the initial response.
func (c *Context) Followup(params *discordgo.WebhookParams, ephemeral bool) (*discordgo.Message, error) {
	if ephemeral {
		params.Flags |= discordgo.MessageFlagsEphemeral
	}
	return c.Session.FollowupMessageCreate(c.interaction(), true, params)
}

// Modal opens a modal in response to the interaction.
func (c *Context) Modal(customID, title string, rows ...discordgo.MessageComponent) error {
	return c.Respond(&discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID:   customID,
			Title:      title,
			Components: rows,
		},
	})
}
