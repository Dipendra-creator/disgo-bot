// Package utility is the sample feature module that proves the framework
// end-to-end. It implements the shared.Module contract with slash commands
// (/ping, /serverinfo, /userinfo, /avatar), a user context-menu command, and
// component handlers (serverinfo refresh button, avatar size buttons).
//
// New feature modules (moderation, tickets, …) follow this exact shape.
package utility

import "github.com/dipu-sharma/disgo-bot/shared"

// Module implements shared.Module. It embeds shared.Base for no-op defaults of
// the methods it doesn't need (Modals, Events).
type Module struct {
	shared.Base
	deps *shared.Deps
	svc  *Service
}

// New returns an uninitialised utility module; call Init before use.
func New() *Module { return &Module{} }

// Name is the module's custom-ID namespace.
func (m *Module) Name() string { return "utility" }

// Init injects dependencies.
func (m *Module) Init(d *shared.Deps) error {
	m.deps = d
	m.svc = NewService(d)
	return nil
}

// Commands returns the module's application commands.
func (m *Module) Commands() []*shared.Command {
	return []*shared.Command{
		m.pingCommand(),
		m.serverinfoCommand(),
		m.userinfoCommand(),
		m.userinfoContextCommand(),
		m.avatarCommand(),
	}
}

// Components maps custom-ID actions to handlers (namespace "utility").
func (m *Module) Components() map[string]shared.HandlerFunc {
	return map[string]shared.HandlerFunc{
		"refresh": m.handleServerinfoRefresh,
		"avatar":  m.handleAvatarSize,
	}
}
