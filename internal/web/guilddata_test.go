package web

import (
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
)

func TestColorHex(t *testing.T) {
	cases := map[int]string{
		0:        "",        // no color → default
		0x5865F2: "#5865F2", // blurple
		0xFF0000: "#FF0000",
		0x00FF00: "#00FF00",
	}
	for in, want := range cases {
		if got := colorHex(in); got != want {
			t.Errorf("colorHex(%#x) = %q, want %q", in, got, want)
		}
	}
}

func TestRolesView(t *testing.T) {
	const gid = "100000000000000000"
	g := &discordgo.Guild{
		ID: gid,
		Roles: []*discordgo.Role{
			{ID: gid, Name: "@everyone", Position: 0},
			{ID: "2", Name: "Admin", Color: 0xFF0000, Position: 5, Hoist: true},
			{ID: "3", Name: "Bot", Position: 3, Managed: true},
			nil, // defensive: nil entries are skipped
		},
	}
	rv := rolesView(g)
	if len(rv) != 3 {
		t.Fatalf("len = %d, want 3 (nil skipped)", len(rv))
	}
	// Highest position first.
	if rv[0].Name != "Admin" || rv[1].Name != "Bot" || rv[2].Name != "@everyone" {
		t.Fatalf("order wrong: %q %q %q", rv[0].Name, rv[1].Name, rv[2].Name)
	}
	if rv[0].Color != "#FF0000" || !rv[0].Hoist {
		t.Errorf("Admin mapped wrong: %+v", rv[0])
	}
	if !rv[2].Everyone {
		t.Errorf("@everyone not flagged: %+v", rv[2])
	}
	if rv[2].Everyone && rv[0].Everyone {
		t.Errorf("non-everyone role mis-flagged")
	}
	if !rv[1].Managed {
		t.Errorf("Bot should be flagged managed: %+v", rv[1])
	}
}

func TestChannelsView(t *testing.T) {
	g := &discordgo.Guild{
		Channels: []*discordgo.Channel{
			{ID: "10", Name: "general", Type: discordgo.ChannelTypeGuildText, ParentID: "1", Position: 2},
			{ID: "1", Name: "TEXT", Type: discordgo.ChannelTypeGuildCategory, Position: 0},
			{ID: "11", Name: "voice", Type: discordgo.ChannelTypeGuildVoice, ParentID: "1", Position: 1},
		},
	}
	cv := channelsView(g)
	if len(cv) != 3 {
		t.Fatalf("len = %d, want 3", len(cv))
	}
	// Sorted by position ascending.
	if cv[0].Name != "TEXT" || cv[1].Name != "voice" || cv[2].Name != "general" {
		t.Fatalf("order wrong: %q %q %q", cv[0].Name, cv[1].Name, cv[2].Name)
	}
	if cv[2].Type != int(discordgo.ChannelTypeGuildText) || cv[2].ParentID != "1" {
		t.Errorf("general mapped wrong: %+v", cv[2])
	}
}

func TestOverviewView(t *testing.T) {
	// Snowflake 175928847299117063 → 2016-04-30 11:18:25.796 UTC (Discord docs example).
	g := &discordgo.Guild{
		ID:                       "175928847299117063",
		Name:                     "Test Guild",
		Icon:                     "abc",
		OwnerID:                  "42",
		MemberCount:              1234,
		PremiumTier:              discordgo.PremiumTier2,
		PremiumSubscriptionCount: 7,
		Roles:                    []*discordgo.Role{{ID: "1"}, {ID: "2"}},
		Channels:                 []*discordgo.Channel{{ID: "10"}},
	}
	ov := overviewView(g)
	if ov.Name != "Test Guild" || ov.OwnerID != "42" || ov.Members != 1234 {
		t.Errorf("basic fields wrong: %+v", ov)
	}
	if ov.Roles != 2 || ov.Channels != 1 {
		t.Errorf("counts wrong: roles=%d channels=%d", ov.Roles, ov.Channels)
	}
	if ov.PremiumTier != 2 || ov.Boosts != 7 {
		t.Errorf("premium wrong: tier=%d boosts=%d", ov.PremiumTier, ov.Boosts)
	}
	want := time.Date(2016, 4, 30, 11, 18, 25, 796000000, time.UTC)
	if !ov.CreatedAt.Equal(want) {
		t.Errorf("CreatedAt = %v, want %v", ov.CreatedAt, want)
	}
}
