package automod

import (
	"testing"
	"time"
)

func TestDefaultSettings(t *testing.T) {
	s := defaultSettings(7)
	if s.GuildID != 7 {
		t.Fatalf("GuildID = %d, want 7", s.GuildID)
	}
	if s.AnyEnabled() {
		t.Fatal("fresh settings must have no filters enabled")
	}
	if !validAction(s.WordsAction) || !validAction(s.SpamAction) {
		t.Fatal("default actions must be valid")
	}
	if s.MentionThreshold < minMentionThreshold || s.SpamCount < minSpamCount {
		t.Fatal("default thresholds must respect their minimums")
	}
}

func TestAnyEnabled(t *testing.T) {
	s := defaultSettings(1)
	s.InvitesEnabled = true
	if !s.AnyEnabled() {
		t.Fatal("AnyEnabled must be true when a filter is on")
	}
	var nilS *Settings
	if nilS.AnyEnabled() {
		t.Fatal("nil settings must report AnyEnabled false")
	}
}

func TestValidAction(t *testing.T) {
	for _, a := range []string{ActionDelete, ActionTimeout} {
		if !validAction(a) {
			t.Fatalf("%q should be valid", a)
		}
	}
	if validAction("ban") {
		t.Fatal("'ban' is not a valid automod action")
	}
}

func TestHasInvite(t *testing.T) {
	hits := []string{
		"join here discord.gg/abc123",
		"https://discord.com/invite/Xy-9",
		"DISCORD.GG/UPPER",
		"discordapp.com/invite/legacy",
	}
	for _, s := range hits {
		if !hasInvite(s) {
			t.Errorf("expected invite match in %q", s)
		}
	}
	misses := []string{"no link here", "discord.gg", "visit discord.com/terms"}
	for _, s := range misses {
		if hasInvite(s) {
			t.Errorf("unexpected invite match in %q", s)
		}
	}
}

func TestMatchBannedWord(t *testing.T) {
	words := map[string]struct{}{"badword": {}, "two words": {}}

	if _, ok := matchBannedWord("this is a BadWord here", words); !ok {
		t.Error("should match a whole-token banned word, case-insensitively")
	}
	if _, ok := matchBannedWord("contains two words inline", words); !ok {
		t.Error("should match a multi-word phrase as a substring")
	}
	if _, ok := matchBannedWord("badwords plural embedded", words); ok {
		t.Error("single-word terms must match whole tokens, not substrings")
	}
	if _, ok := matchBannedWord("totally clean", words); ok {
		t.Error("clean content must not match")
	}
	if _, ok := matchBannedWord("anything", map[string]struct{}{}); ok {
		t.Error("empty word set must never match")
	}
}

func TestRecordSpam(t *testing.T) {
	s := &Service{spam: map[spamKey][]time.Time{}}
	// count=3 within a wide window: first two are fine, third trips.
	if s.recordSpam(1, 2, 3, 60) {
		t.Fatal("1st message must not trip")
	}
	if s.recordSpam(1, 2, 3, 60) {
		t.Fatal("2nd message must not trip")
	}
	if !s.recordSpam(1, 2, 3, 60) {
		t.Fatal("3rd message must trip the threshold")
	}
	// A different user is tracked independently.
	if s.recordSpam(1, 99, 3, 60) {
		t.Fatal("a different user must not be affected")
	}
}

func TestNormalizeWord(t *testing.T) {
	if got := normalizeWord("  FooBar  "); got != "foobar" {
		t.Fatalf("normalizeWord = %q, want \"foobar\"", got)
	}
}

func TestIDRoundTrip(t *testing.T) {
	const raw = "1234567890123456789"
	if got := sid(pid(raw)); got != raw {
		t.Fatalf("sid(pid(%q)) = %q, want %q", raw, got, raw)
	}
}
