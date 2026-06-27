package ai

import (
	"strings"
	"testing"
)

func TestSettingsSystem(t *testing.T) {
	def := defaultSettings(42)
	if def.GuildID != 42 {
		t.Fatalf("GuildID = %d, want 42", def.GuildID)
	}
	if def.AssistantChannelID != 0 || def.SystemPrompt != "" {
		t.Fatalf("defaults must be zero-valued, got %+v", def)
	}
	if got := def.system(); got != defaultSystem {
		t.Fatalf("system() = %q, want the default prompt", got)
	}

	def.SystemPrompt = "Be a pirate."
	if got := def.system(); got != "Be a pirate." {
		t.Fatalf("system() = %q, want the override", got)
	}

	// A nil receiver must still yield the default rather than panicking.
	var nilSet *Settings
	if got := nilSet.system(); got != defaultSystem {
		t.Fatalf("nil system() = %q, want the default prompt", got)
	}
}

func TestExtractText(t *testing.T) {
	r := anthropicResponse{}
	r.Content = append(r.Content, struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}{Type: "text", Text: "Hello "})
	r.Content = append(r.Content, struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}{Type: "thinking", Text: "ignored"})
	r.Content = append(r.Content, struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}{Type: "text", Text: "world"})

	if got := extractText(r); got != "Hello world" {
		t.Fatalf("extractText = %q, want %q", got, "Hello world")
	}
	if got := extractText(anthropicResponse{}); got != "" {
		t.Fatalf("extractText(empty) = %q, want empty", got)
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("short", 100); got != "short" {
		t.Fatalf("truncate no-op = %q", got)
	}
	// Multi-byte runes must not be split mid-character.
	in := strings.Repeat("é", 10)
	got := truncate(in, 4)
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("truncate must append an ellipsis, got %q", got)
	}
	if r := []rune(strings.TrimSuffix(got, "…")); len(r) > 4 {
		t.Fatalf("truncate kept %d runes, want <= 4", len(r))
	}
	for _, c := range got {
		if c != 'é' && c != '…' {
			t.Fatalf("truncate corrupted a rune: %q", got)
		}
	}
}

func TestIDRoundTrip(t *testing.T) {
	const raw = "1234567890123456789"
	if got := sid(pid(raw)); got != raw {
		t.Fatalf("sid(pid(%q)) = %q, want %q", raw, got, raw)
	}
}
