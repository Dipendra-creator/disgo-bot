// Package router registers modules and dispatches incoming interactions to the
// right handler, wrapping each call with panic-recovery, metrics and structured
// logging. Slash/context commands route by name; components and modals route by
// the "<module>:<action>" prefix of their custom ID.
package router

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/shared"
	"go.uber.org/zap"
)

// Registry holds all registered handlers and command definitions.
type Registry struct {
	log *zap.Logger

	commands   map[string]*shared.Command    // by application-command name
	components map[string]shared.HandlerFunc // by "module:action"
	modals     map[string]shared.HandlerFunc // by "module:action"
	events     []interface{}                 // raw discordgo handlers
	defs       []*discordgo.ApplicationCommand
}

// NewRegistry creates an empty registry.
func NewRegistry(log *zap.Logger) *Registry {
	return &Registry{
		log:        log,
		commands:   make(map[string]*shared.Command),
		components: make(map[string]shared.HandlerFunc),
		modals:     make(map[string]shared.HandlerFunc),
	}
}

// Register wires a module's commands, component/modal handlers and events into
// the registry. It errors on duplicate command names or handler keys so
// conflicts surface at startup rather than at runtime.
func (r *Registry) Register(m shared.Module) error {
	name := m.Name()

	for _, c := range m.Commands() {
		if c == nil || c.Def == nil {
			return fmt.Errorf("module %q: nil command", name)
		}
		if _, dup := r.commands[c.Def.Name]; dup {
			return fmt.Errorf("module %q: duplicate command %q", name, c.Def.Name)
		}
		r.commands[c.Def.Name] = c
		r.defs = append(r.defs, c.Def)
	}

	for action, h := range m.Components() {
		key := name + ":" + action
		if _, dup := r.components[key]; dup {
			return fmt.Errorf("duplicate component handler %q", key)
		}
		r.components[key] = h
	}

	for action, h := range m.Modals() {
		key := name + ":" + action
		if _, dup := r.modals[key]; dup {
			return fmt.Errorf("duplicate modal handler %q", key)
		}
		r.modals[key] = h
	}

	r.events = append(r.events, m.Events()...)
	r.log.Info("registered module",
		zap.String("module", name),
		zap.Int("commands", len(m.Commands())),
		zap.Int("components", len(m.Components())),
		zap.Int("modals", len(m.Modals())),
	)
	return nil
}

// Events returns the collected raw event handlers for session wiring.
func (r *Registry) Events() []interface{} { return r.events }

// CommandDefs returns the application-command definitions for registration.
func (r *Registry) CommandDefs() []*discordgo.ApplicationCommand { return r.defs }
