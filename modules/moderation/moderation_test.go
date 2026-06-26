package moderation

import (
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dipu-sharma/disgo-bot/internal/ui"
)

func TestStyleFor(t *testing.T) {
	if got := styleFor(ActionBan, false); got.verb != "Banned" || got.color != ui.ColorDanger {
		t.Errorf("ban style = %+v", got)
	}
	if got := styleFor(ActionBan, true); got.verb != "Temporarily Banned" {
		t.Errorf("temp ban verb = %q", got.verb)
	}
	if got := styleFor(ActionWarn, false); got.verb != "Warned" || got.color != ui.ColorWarning {
		t.Errorf("warn style = %+v", got)
	}
	if got := styleFor(ActionUnban, false); got.color != ui.ColorSuccess {
		t.Errorf("unban color = %x", got.color)
	}
}

func TestAuditReason(t *testing.T) {
	mod := &discordgo.User{ID: "42", Username: "mod", Discriminator: "0"}
	got := auditReason(mod, "spam")
	if got == "" || got == "spam" {
		t.Errorf("expected composed reason, got %q", got)
	}
	if r := auditReason(nil, ""); r != "No reason provided" {
		t.Errorf("empty/no-mod reason = %q", r)
	}
}

func TestReasonOrNone(t *testing.T) {
	if reasonOrNone("") == "" {
		t.Error("empty reason should yield placeholder")
	}
	if reasonOrNone("  ") == "  " {
		t.Error("whitespace reason should yield placeholder")
	}
	if got := reasonOrNone("rude"); got != "rude" {
		t.Errorf("reason = %q", got)
	}
}

func sampleGuild() *discordgo.Guild {
	return &discordgo.Guild{
		ID:      "guild",
		OwnerID: "owner",
		Roles: []*discordgo.Role{
			{ID: "everyone", Position: 0},
			{ID: "member", Position: 1},
			{ID: "mod", Position: 5},
			{ID: "admin", Position: 9},
		},
	}
}

func TestHighestRole(t *testing.T) {
	g := sampleGuild()
	cases := []struct {
		roles []string
		want  int
	}{
		{nil, -1},
		{[]string{"member"}, 1},
		{[]string{"member", "mod"}, 5},
		{[]string{"admin", "member"}, 9},
		{[]string{"unknown"}, -1},
	}
	for _, c := range cases {
		got := highestRole(g, &discordgo.Member{Roles: c.roles})
		if got != c.want {
			t.Errorf("highestRole(%v) = %d, want %d", c.roles, got, c.want)
		}
	}
}

func TestCheckTarget(t *testing.T) {
	g := sampleGuild()
	if err := checkTarget(g, "a", "a", "bot"); err == nil {
		t.Error("self-moderation should error")
	}
	if err := checkTarget(g, "a", "bot", "bot"); err == nil {
		t.Error("moderating the bot should error")
	}
	if err := checkTarget(g, "a", "owner", "bot"); err == nil {
		t.Error("moderating the owner should error")
	}
	if err := checkTarget(g, "a", "b", "bot"); err != nil {
		t.Errorf("valid target should pass, got %v", err)
	}
}

func TestCheckHierarchy(t *testing.T) {
	g := sampleGuild()
	mod := &discordgo.Member{User: &discordgo.User{ID: "m"}, Roles: []string{"mod"}}     // pos 5
	admin := &discordgo.Member{User: &discordgo.User{ID: "a"}, Roles: []string{"admin"}} // pos 9
	bot := &discordgo.Member{User: &discordgo.User{ID: "bot"}, Roles: []string{"admin"}} // pos 9

	// Mod (5) cannot action admin (9).
	if err := checkHierarchy(g, mod, admin, bot); err == nil {
		t.Error("mod acting on higher role should error")
	}
	// Admin invoker (9) over mod target (5), bot admin (9) over mod (5): ok.
	if err := checkHierarchy(g, admin, mod, bot); err != nil {
		t.Errorf("admin over mod should pass, got %v", err)
	}
	// Bot too low: weakBot (1) cannot act on mod (5).
	weakBot := &discordgo.Member{User: &discordgo.User{ID: "bot"}, Roles: []string{"member"}}
	if err := checkHierarchy(g, admin, mod, weakBot); err == nil {
		t.Error("bot below target should error")
	}
	// Owner invoker bypasses the invoker check.
	owner := &discordgo.Member{User: &discordgo.User{ID: "owner"}, Roles: nil}
	if err := checkHierarchy(g, owner, mod, bot); err != nil {
		t.Errorf("owner should bypass invoker hierarchy, got %v", err)
	}
	// Nil guild/target is best-effort no-op.
	if err := checkHierarchy(nil, mod, admin, bot); err != nil {
		t.Errorf("nil guild should be no-op, got %v", err)
	}
}

func TestIDs(t *testing.T) {
	if pid("175928847299117063") != 175928847299117063 {
		t.Error("pid round-trip failed")
	}
	if sid(175928847299117063) != "175928847299117063" {
		t.Error("sid round-trip failed")
	}
	if pid("not-a-number") != 0 {
		t.Error("pid of garbage should be 0")
	}
	if !isSnowflake("175928847299117063") {
		t.Error("valid snowflake rejected")
	}
	for _, bad := range []string{"", "123", "abc", "12345678901234567890123"} {
		if isSnowflake(bad) {
			t.Errorf("isSnowflake(%q) should be false", bad)
		}
	}
}

func TestCaseTemporary(t *testing.T) {
	perm := &Case{Action: ActionBan}
	if perm.Temporary() {
		t.Error("zero-duration case should be permanent")
	}
	temp := &Case{Action: ActionBan, DurationMS: (2 * time.Hour).Milliseconds()}
	if !temp.Temporary() {
		t.Error("non-zero-duration case should be temporary")
	}
	if temp.Duration() != 2*time.Hour {
		t.Errorf("Duration() = %v", temp.Duration())
	}
}
