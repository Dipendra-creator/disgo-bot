package help

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/dipu-sharma/disgo-bot/shared"
)

// Components-v2 has a hard ~4000-character budget across all text displays in a
// single message. The catalog is authored to stay well under this, but category
// bodies are still trimmed defensively so a future edit can never produce a
// message Discord rejects at runtime.
const maxBodyChars = 3500

// trunc shortens s to n characters with an ellipsis, respecting rune-ish bounds.
func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}

// categorySelect builds the menu used to jump between categories. When current
// is non-empty that category is marked as the default selection.
func categorySelect(current string) discordgo.ActionsRow {
	opts := make([]discordgo.SelectMenuOption, 0, len(Catalog))
	for i := range Catalog {
		c := &Catalog[i]
		o := ui.Option(c.Label, c.Key, trunc(c.Blurb, 100), c.Emoji)
		if c.Key == current {
			o.Default = true
		}
		opts = append(opts, o)
	}
	return ui.Row(ui.StringSelect(shared.BuildID(moduleName, "cat"), "Browse a category…", opts...))
}

// homeRow is the navigation row shown on detail views: back to the overview.
func homeRow() discordgo.ActionsRow {
	return ui.Row(ui.SecondaryButton(shared.BuildID(moduleName, "home"), "Overview", "🏠"))
}

// renderOverview is the landing view: what the bot does, the category list, a
// couple of starter examples, and the category picker.
func renderOverview() *discordgo.InteractionResponseData {
	var list strings.Builder
	for i := range Catalog {
		c := &Catalog[i]
		fmt.Fprintf(&list, "%s **%s** — %s\n", c.Emoji, c.Label, c.Blurb)
	}

	header := ui.Text(fmt.Sprintf(
		"## 📖 Help\n%d commands across %d modules. Pick a category below to see commands and examples.",
		commandCount(), len(Catalog),
	))
	tip := ui.Text(
		"**Quick start**\n" +
			"↳ `/help command:ban` — jump straight to one command's examples\n" +
			"↳ `/serverinfo` — try a no-permission command right now\n" +
			"↳ Most setup commands need **Manage Server**.",
	)

	container := ui.Container(ui.ColorBrand,
		header,
		ui.Separator(),
		ui.Text(trunc(list.String(), maxBodyChars)),
		ui.Separator(),
		tip,
		categorySelect(""),
	)
	return ui.V2(container)
}

// commandBlock renders one command as a compact multi-line entry with a single
// representative example.
func commandBlock(b *strings.Builder, cmd *Command) {
	perm := ""
	if cmd.Perm != "" {
		perm = "  ·  🔒 " + cmd.Perm
	}
	fmt.Fprintf(b, "**`%s`**%s\n%s\n", cmd.Usage, perm, cmd.Desc)
	if len(cmd.Examples) > 0 {
		ex := cmd.Examples[0]
		fmt.Fprintf(b, "↳ `%s` — %s\n", ex.Usage, ex.Note)
	}
	b.WriteString("\n")
}

// renderCategory lists every command in one category, each with one example,
// plus the picker (with this category pre-selected) and an Overview button.
func renderCategory(key string) *discordgo.InteractionResponseData {
	c := category(key)
	if c == nil {
		return renderOverview()
	}

	var body strings.Builder
	for i := range c.Commands {
		commandBlock(&body, &c.Commands[i])
	}

	header := ui.Text(fmt.Sprintf("## %s %s\n%s", c.Emoji, c.Label, c.Blurb))
	footer := ui.Text(fmt.Sprintf("Use `/help command:<name>` for full examples of a single command. • %d commands", len(c.Commands)))

	container := ui.Container(ui.ColorBrand,
		header,
		ui.Separator(),
		ui.Text(trunc(strings.TrimRight(body.String(), "\n"), maxBodyChars)),
		ui.Separator(),
		footer,
		categorySelect(key),
		homeRow(),
	)
	return ui.V2(container)
}

// renderCommand is the deep view for a single command: full description, every
// example, and navigation back.
func renderCommand(cat *Category, cmd *Command) *discordgo.InteractionResponseData {
	var b strings.Builder
	fmt.Fprintf(&b, "## %s `/%s`\n%s\n", cat.Emoji, cmd.Name, cmd.Desc)
	if cmd.Perm != "" {
		fmt.Fprintf(&b, "**Requires:** %s\n", cmd.Perm)
	} else {
		b.WriteString("**Requires:** anyone can use this.\n")
	}

	var ex strings.Builder
	ex.WriteString("**Usage**\n`" + cmd.Usage + "`\n\n**Examples**\n")
	if len(cmd.Examples) == 0 {
		fmt.Fprintf(&ex, "↳ `/%s`\n", cmd.Name)
	}
	for _, e := range cmd.Examples {
		fmt.Fprintf(&ex, "↳ `%s`\n%s\n\n", e.Usage, e.Note)
	}

	container := ui.Container(ui.ColorBrand,
		ui.Text(trunc(b.String(), 1000)),
		ui.Separator(),
		ui.Text(trunc(strings.TrimRight(ex.String(), "\n"), maxBodyChars)),
		ui.Separator(),
		categorySelect(cat.Key),
		homeRow(),
	)
	return ui.V2(container)
}
