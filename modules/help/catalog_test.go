package help

import (
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
)

// textLen sums the characters across every TextDisplay in a v2 component tree.
// Components-v2 caps total text content at 4000 characters per message.
func textLen(comps []discordgo.MessageComponent) int {
	n := 0
	for _, comp := range comps {
		switch c := comp.(type) {
		case discordgo.TextDisplay:
			n += len(c.Content)
		case discordgo.Container:
			n += textLen(c.Components)
		case discordgo.Section:
			n += textLen(c.Components)
		}
	}
	return n
}

// TestCatalogIntegrity guards the hand-authored catalog against the structural
// mistakes that would surface only at runtime in Discord.
func TestCatalogIntegrity(t *testing.T) {
	if len(Catalog) == 0 {
		t.Fatal("catalog is empty")
	}
	seen := map[string]bool{}
	for _, c := range Catalog {
		if c.Key == "" || c.Label == "" || c.Emoji == "" || c.Blurb == "" {
			t.Errorf("category %q has empty metadata", c.Key)
		}
		if seen[c.Key] {
			t.Errorf("duplicate category key %q", c.Key)
		}
		seen[c.Key] = true
		if len(c.Blurb) > 100 {
			t.Errorf("category %q blurb %d chars > 100 (select-option description cap)", c.Key, len(c.Blurb))
		}
		if len(c.Commands) == 0 {
			t.Errorf("category %q has no commands", c.Key)
		}
		for _, cmd := range c.Commands {
			if cmd.Name == "" || cmd.Usage == "" || cmd.Desc == "" {
				t.Errorf("command %q in %q missing required field", cmd.Name, c.Key)
			}
			if strings.Contains(cmd.Name, " ") || strings.HasPrefix(cmd.Name, "/") {
				t.Errorf("command name %q must be a bare token (no slash, no spaces)", cmd.Name)
			}
			if !strings.HasPrefix(cmd.Usage, "/"+cmd.Name) {
				t.Errorf("command %q usage %q should start with /%s", cmd.Name, cmd.Usage, cmd.Name)
			}
			if len(cmd.Examples) == 0 {
				t.Errorf("command %q has no examples — every command must show at least one", cmd.Name)
			}
			for _, ex := range cmd.Examples {
				if !strings.HasPrefix(ex.Usage, "/") || ex.Note == "" {
					t.Errorf("command %q example %q malformed (usage must start with /, note required)", cmd.Name, ex.Usage)
				}
			}
		}
	}
}

// TestLookups exercises the catalog access helpers.
func TestLookups(t *testing.T) {
	if got := commandCount(); got < len(Catalog) {
		t.Fatalf("commandCount = %d, want >= %d", got, len(Catalog))
	}
	if category("moderation") == nil {
		t.Fatal("category(moderation) = nil")
	}
	if category("does-not-exist") != nil {
		t.Fatal("category(does-not-exist) should be nil")
	}
	cat, cmd := findCommand("ban")
	if cat == nil || cmd == nil {
		t.Fatal("findCommand(ban) returned nil")
	}
	if cat.Key != "moderation" || cmd.Name != "ban" {
		t.Fatalf("findCommand(ban) = %q/%q, want moderation/ban", cat.Key, cmd.Name)
	}
	if _, miss := findCommand("nope"); miss != nil {
		t.Fatal("findCommand(nope) should miss")
	}
}

// TestRenderersWithinBudget proves every view stays under the Components-v2
// character budget and produces non-empty output, including the deep command
// view for every catalogued command.
func TestRenderersWithinBudget(t *testing.T) {
	const budget = 4000
	if d := renderOverview(); d == nil || len(d.Components) == 0 {
		t.Fatal("overview rendered empty")
	} else if n := textLen(d.Components); n > budget {
		t.Fatalf("overview text %d > %d budget", n, budget)
	}
	for _, c := range Catalog {
		d := renderCategory(c.Key)
		if d == nil || len(d.Components) == 0 {
			t.Fatalf("category %q rendered empty", c.Key)
		}
		if n := textLen(d.Components); n > budget {
			t.Fatalf("category %q text %d > %d budget", c.Key, n, budget)
		}
		for i := range c.Commands {
			cmd := &c.Commands[i]
			cat := category(c.Key)
			cd := renderCommand(cat, cmd)
			if cd == nil || len(cd.Components) == 0 {
				t.Fatalf("command %q rendered empty", cmd.Name)
			}
			if n := textLen(cd.Components); n > budget {
				t.Fatalf("command %q text %d > %d budget", cmd.Name, n, budget)
			}
		}
	}
	// Unknown category falls back to the overview rather than panicking.
	if d := renderCategory("bogus"); d == nil || len(d.Components) == 0 {
		t.Fatal("renderCategory(bogus) should fall back to overview")
	}
}

// TestTrunc covers the defensive text trimming.
func TestTrunc(t *testing.T) {
	if got := trunc("hello", 10); got != "hello" {
		t.Errorf("trunc no-op = %q", got)
	}
	if got := trunc("hello world", 5); got != "hell…" {
		t.Errorf("trunc = %q, want hell…", got)
	}
}
