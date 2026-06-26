package tickets

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestBuildOverwrites(t *testing.T) {
	ow := buildOverwrites("guild", "opener", "bot", 555)
	if len(ow) != 4 {
		t.Fatalf("expected 4 overwrites (everyone, opener, bot, staff), got %d", len(ow))
	}

	// @everyone is denied view.
	if ow[0].ID != "guild" || ow[0].Type != discordgo.PermissionOverwriteTypeRole {
		t.Errorf("first overwrite should be @everyone role, got %+v", ow[0])
	}
	if ow[0].Deny&discordgo.PermissionViewChannel == 0 {
		t.Error("@everyone should be denied ViewChannel")
	}

	// Opener can view and send.
	if ow[1].ID != "opener" || ow[1].Type != discordgo.PermissionOverwriteTypeMember {
		t.Errorf("second overwrite should be opener member, got %+v", ow[1])
	}
	if ow[1].Allow&discordgo.PermissionSendMessages == 0 {
		t.Error("opener should be allowed SendMessages")
	}

	// Bot gets Manage Channels (to delete the channel on close).
	if ow[2].ID != "bot" || ow[2].Allow&discordgo.PermissionManageChannels == 0 {
		t.Errorf("bot overwrite should grant ManageChannels, got %+v", ow[2])
	}

	// Staff role present by ID.
	if ow[3].ID != "555" || ow[3].Type != discordgo.PermissionOverwriteTypeRole {
		t.Errorf("staff overwrite should be role 555, got %+v", ow[3])
	}
}

func TestBuildOverwritesMinimal(t *testing.T) {
	ow := buildOverwrites("guild", "opener", "", 0)
	if len(ow) != 2 {
		t.Fatalf("expected 2 overwrites without bot/staff, got %d", len(ow))
	}
}

func TestSettingsConfigured(t *testing.T) {
	var nilSet *Settings
	if nilSet.Configured() {
		t.Error("nil settings should not be configured")
	}
	if (&Settings{}).Configured() {
		t.Error("settings without a category should not be configured")
	}
	if !(&Settings{CategoryID: 1}).Configured() {
		t.Error("settings with a category should be configured")
	}
}

func TestPanelComponents(t *testing.T) {
	comps := panelComponents("Title", "Desc")
	if len(comps) != 1 {
		t.Fatalf("panel should be a single top-level container, got %d", len(comps))
	}
	if _, ok := comps[0].(discordgo.Container); !ok {
		t.Errorf("panel top-level component should be a Container, got %T", comps[0])
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
