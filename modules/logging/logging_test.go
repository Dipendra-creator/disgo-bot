package logging

import (
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestValidCategory(t *testing.T) {
	for _, c := range Categories {
		if !ValidCategory(c) {
			t.Errorf("%q should be valid", c)
		}
	}
	if ValidCategory("nonsense") {
		t.Error("unknown category should be invalid")
	}
}

func TestSettingsChannel(t *testing.T) {
	s := &Settings{MessageChannelID: 11, MemberChannelID: 22, ServerChannelID: 33}
	if s.channel(CategoryMessage) != 11 {
		t.Error("message channel mismatch")
	}
	if s.channel(CategoryMember) != 22 {
		t.Error("member channel mismatch")
	}
	if s.channel(CategoryServer) != 33 {
		t.Error("server channel mismatch")
	}
	if s.channel("bogus") != 0 {
		t.Error("unknown category should resolve to 0")
	}
}

func TestCategoryColumn(t *testing.T) {
	for _, c := range Categories {
		col, err := categoryColumn(c)
		if err != nil || col == "" {
			t.Errorf("category %q should map to a column, got %q err=%v", c, col, err)
		}
	}
	if _, err := categoryColumn("bogus"); err == nil {
		t.Error("unknown category should error")
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate(""); got != "*empty*" {
		t.Errorf("empty content should render placeholder, got %q", got)
	}
	long := strings.Repeat("a", maxFieldLen+50)
	got := truncate(long)
	if !strings.HasSuffix(got, "…") {
		t.Error("over-long content should be truncated with an ellipsis")
	}
	if len([]rune(got)) != maxFieldLen+1 {
		t.Errorf("truncated length = %d, want %d", len([]rune(got)), maxFieldLen+1)
	}
}

func TestStatusEmbed(t *testing.T) {
	e := statusEmbed(&Settings{MessageChannelID: 123})
	if e == nil || len(e.Fields) != len(Categories) {
		t.Fatalf("status embed should have one field per category")
	}
	if !strings.Contains(e.Fields[0].Value, "123") {
		t.Errorf("configured category should mention its channel, got %q", e.Fields[0].Value)
	}
}

func TestMessageDeleteEmbed(t *testing.T) {
	m := &discordgo.Message{ChannelID: "5", Author: &discordgo.User{ID: "9", Username: "bob"}, Content: "hi"}
	e := messageDeleteEmbed(m)
	if e.Title == "" || len(e.Fields) == 0 {
		t.Error("delete embed should have a title and fields")
	}
}

func TestIDs(t *testing.T) {
	if pid("175928847299117063") != 175928847299117063 {
		t.Error("pid round-trip failed")
	}
	if sid(175928847299117063) != "175928847299117063" {
		t.Error("sid round-trip failed")
	}
}
