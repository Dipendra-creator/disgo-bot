package router

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/observability"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/shared"
	"go.uber.org/zap"
)

// handlerTimeout bounds a handler's lifetime; Discord allows 15 minutes after a
// deferral, so we stop slightly short.
const handlerTimeout = 14 * time.Minute

// Dispatch returns the single InteractionCreate handler that routes every
// interaction. Deps supplies the per-interaction Context.
func (r *Registry) Dispatch(deps *shared.Deps) func(*discordgo.Session, *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if deps.Metrics != nil {
			deps.Metrics.CountInteraction(int(i.Type))
		}
		switch i.Type {
		case discordgo.InteractionApplicationCommand, discordgo.InteractionApplicationCommandAutocomplete:
			r.handleCommand(deps, i)
		case discordgo.InteractionMessageComponent:
			r.handleComponent(deps, i)
		case discordgo.InteractionModalSubmit:
			r.handleModal(deps, i)
		default:
			r.log.Debug("unhandled interaction type", zap.Int("type", int(i.Type)))
		}
	}
}

func (r *Registry) handleCommand(deps *shared.Deps, i *discordgo.InteractionCreate) {
	name := i.ApplicationCommandData().Name
	cmd, ok := r.commands[name]
	if !ok {
		r.log.Warn("unknown command", zap.String("command", name))
		return
	}

	if i.Type == discordgo.InteractionApplicationCommandAutocomplete {
		if cmd.Autocomplete != nil {
			r.run(deps, i, "autocomplete:"+name, cmd.Autocomplete, nil)
		}
		return
	}
	r.run(deps, i, name, cmd.Handler, nil)
}

func (r *Registry) handleComponent(deps *shared.Deps, i *discordgo.InteractionCreate) {
	module, action, args := shared.ParseID(i.MessageComponentData().CustomID)
	key := module + ":" + action
	h, ok := r.components[key]
	if !ok {
		r.log.Warn("unknown component", zap.String("custom_id", i.MessageComponentData().CustomID))
		return
	}
	r.run(deps, i, key, h, args)
}

func (r *Registry) handleModal(deps *shared.Deps, i *discordgo.InteractionCreate) {
	module, action, args := shared.ParseID(i.ModalSubmitData().CustomID)
	key := module + ":" + action
	h, ok := r.modals[key]
	if !ok {
		r.log.Warn("unknown modal", zap.String("custom_id", i.ModalSubmitData().CustomID))
		return
	}
	r.run(deps, i, key, h, args)
}

// run executes a handler with the full middleware chain: timeout context,
// panic recovery, error surfacing and metrics.
func (r *Registry) run(deps *shared.Deps, i *discordgo.InteractionCreate, name string, h shared.HandlerFunc, args []string) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), handlerTimeout)
	defer cancel()

	log := r.log.With(
		zap.String("handler", name),
		zap.String("guild_id", i.GuildID),
	)
	c := &shared.Context{
		Ctx:     ctx,
		Session: deps.Session,
		Event:   i,
		Deps:    deps,
		Log:     log,
		Args:    args,
	}

	ok := true
	defer func() {
		if rec := recover(); rec != nil {
			ok = false
			log.Error("handler panic", zap.Any("panic", rec))
			observability.CaptureError(fmt.Errorf("panic in %s: %v", name, rec))
			r.surfaceError(c, "An unexpected error occurred.")
		}
		if deps.Metrics != nil {
			deps.Metrics.ObserveCommand(name, ok, time.Since(start))
		}
	}()

	if err := h(c); err != nil {
		ok = false
		if ue, isUser := shared.AsUserError(err); isUser {
			r.surfaceError(c, ue.Msg)
			return
		}
		log.Error("handler error", zap.Error(err))
		observability.CaptureError(err)
		r.surfaceError(c, "An unexpected error occurred.")
	}
}

// surfaceError shows a friendly ephemeral error, falling back to a followup if
// the interaction was already acknowledged.
func (r *Registry) surfaceError(c *shared.Context, msg string) {
	if err := c.Reply(ui.ErrorReply(msg), true); err != nil {
		if _, ferr := c.Followup(&discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{ui.ErrorEmbed(msg)},
		}, true); ferr != nil {
			c.Log.Warn("failed to surface error to user", zap.Error(ferr))
		}
	}
}
