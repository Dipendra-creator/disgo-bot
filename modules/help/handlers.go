package help

import (
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/shared"
)

// maxChoices is Discord's cap on autocomplete suggestions per response.
const maxChoices = 25

// helpCommand defines /help with an optional, autocompleting `command` argument.
// With no argument it opens the browsable overview; with a command name it jumps
// straight to that command's full examples.
func (m *Module) helpCommand() *shared.Command {
	return &shared.Command{
		Def: &discordgo.ApplicationCommand{
			Name:        "help",
			Description: "Browse every command with examples",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:         discordgo.ApplicationCommandOptionString,
					Name:         "command",
					Description:  "Jump to a specific command (e.g. ban, daily, giveaway)",
					Required:     false,
					Autocomplete: true,
				},
			},
		},
		Handler:      m.handleHelp,
		Autocomplete: m.handleAutocomplete,
	}
}

// handleHelp answers the slash command. All help responses are ephemeral so the
// guide never clutters the channel.
func (m *Module) handleHelp(c *shared.Context) error {
	query := ""
	for _, o := range c.Event.ApplicationCommandData().Options {
		if o.Name == "command" {
			query = strings.TrimSpace(strings.ToLower(o.StringValue()))
		}
	}
	query = strings.TrimPrefix(query, "/")

	if query == "" {
		return c.Reply(renderOverview(), true)
	}
	if cat, cmd := findCommand(query); cmd != nil {
		return c.Reply(renderCommand(cat, cmd), true)
	}
	if category(query) != nil {
		return c.Reply(renderCategory(query), true)
	}
	// Unknown token: fall back to the overview rather than erroring.
	return c.Reply(renderOverview(), true)
}

// handleCategory re-renders the chosen category in place when a select option is
// picked.
func (m *Module) handleCategory(c *shared.Context) error {
	values := c.Event.MessageComponentData().Values
	if len(values) == 0 {
		return c.Update(renderOverview())
	}
	return c.Update(renderCategory(values[0]))
}

// handleHome returns to the overview in place.
func (m *Module) handleHome(c *shared.Context) error {
	return c.Update(renderOverview())
}

// handleAutocomplete suggests command names matching what the user has typed,
// preferring prefix matches and falling back to substring matches.
func (m *Module) handleAutocomplete(c *shared.Context) error {
	typed := ""
	for _, o := range c.Event.ApplicationCommandData().Options {
		if o.Focused {
			typed = strings.ToLower(strings.TrimSpace(o.StringValue()))
			break
		}
	}
	typed = strings.TrimPrefix(typed, "/")

	type scored struct {
		name  string
		label string
		rank  int // 0 = prefix match, 1 = substring match
	}
	var matches []scored
	for i := range Catalog {
		cat := &Catalog[i]
		for j := range cat.Commands {
			name := cat.Commands[j].Name
			switch {
			case typed == "" || strings.HasPrefix(name, typed):
				matches = append(matches, scored{name, "/" + name + " — " + cat.Label, 0})
			case strings.Contains(name, typed):
				matches = append(matches, scored{name, "/" + name + " — " + cat.Label, 1})
			}
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].rank != matches[j].rank {
			return matches[i].rank < matches[j].rank
		}
		return matches[i].name < matches[j].name
	})

	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0, maxChoices)
	for _, mt := range matches {
		if len(choices) >= maxChoices {
			break
		}
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  trunc(mt.label, 100),
			Value: mt.name,
		})
	}

	return c.Respond(&discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{Choices: choices},
	})
}
