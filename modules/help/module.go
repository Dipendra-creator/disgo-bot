// Package help implements the /help command: a browsable, example-rich guide to
// every command the bot exposes. It is intentionally self-contained — it does
// not import any other feature module — so the modules stay independent (Clean
// Architecture). The command catalog lives in catalog.go as hand-authored data;
// this file wires the slash command, the category select menu and the
// navigation buttons into the shared.Module contract.
package help

import "github.com/dipu-sharma/disgo-bot/shared"

// moduleName is the custom-ID namespace and select-menu prefix for this module.
const moduleName = "help"

// Module implements shared.Module. It embeds shared.Base for the no-op defaults
// (Modals, Events) it doesn't use.
type Module struct {
	shared.Base
	deps *shared.Deps
}

// New returns an uninitialised help module; call Init before use.
func New() *Module { return &Module{} }

// Name is the module's custom-ID namespace.
func (m *Module) Name() string { return moduleName }

// Init injects dependencies.
func (m *Module) Init(d *shared.Deps) error {
	m.deps = d
	return nil
}

// Commands returns the module's application commands.
func (m *Module) Commands() []*shared.Command {
	return []*shared.Command{m.helpCommand()}
}

// Components maps custom-ID actions to handlers (namespace "help"):
//   - help:cat  → category select menu
//   - help:home → back-to-overview button
func (m *Module) Components() map[string]shared.HandlerFunc {
	return map[string]shared.HandlerFunc{
		"cat":  m.handleCategory,
		"home": m.handleHome,
	}
}
