package shared

import "github.com/bwmarrin/discordgo"

// HandlerFunc handles a single interaction (command, component or modal). It
// receives a rich Context and returns an error; returning a *UserError sends a
// friendly message to the user, any other error is logged and reported.
type HandlerFunc func(*Context) error

// Command bundles a Discord application-command definition with its handler and
// optional autocomplete handler.
type Command struct {
	// Def is the command schema registered with Discord.
	Def *discordgo.ApplicationCommand
	// Handler runs when the command is invoked.
	Handler HandlerFunc
	// Autocomplete runs for autocomplete interactions (optional).
	Autocomplete HandlerFunc
}

// Module is the plugin contract every feature implements. Registration is
// declarative: a module exposes its commands, component/modal handlers and
// gateway event handlers, and the router wires them up.
type Module interface {
	// Name is a stable identifier, also used as the custom-ID namespace.
	Name() string
	// Init injects dependencies and performs one-time setup.
	Init(*Deps) error
	// Commands returns the slash/context-menu commands this module provides.
	Commands() []*Command
	// Components maps a custom-ID action prefix to its handler. The router
	// dispatches "<module>:<action>:..." custom IDs to the matching entry.
	Components() map[string]HandlerFunc
	// Modals maps a custom-ID action prefix to its modal-submit handler.
	Modals() map[string]HandlerFunc
	// Events returns raw discordgo event handler funcs (e.g.
	// func(*discordgo.Session, *discordgo.GuildMemberAdd)).
	Events() []interface{}
}

// Base provides no-op implementations of the optional Module methods so a
// concrete module only needs to implement Name plus whatever it actually uses.
type Base struct{}

func (Base) Init(*Deps) error                   { return nil }
func (Base) Commands() []*Command               { return nil }
func (Base) Components() map[string]HandlerFunc { return nil }
func (Base) Modals() map[string]HandlerFunc     { return nil }
func (Base) Events() []interface{}              { return nil }
