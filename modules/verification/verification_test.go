package verification

import "testing"

func TestDefaultSettings(t *testing.T) {
	s := defaultSettings(42)
	if s.GuildID != 42 {
		t.Fatalf("GuildID = %d, want 42", s.GuildID)
	}
	if s.Message == "" || s.ButtonLabel == "" {
		t.Fatalf("defaults must seed message and button label, got %q / %q", s.Message, s.ButtonLabel)
	}
	if s.Configured() {
		t.Fatal("fresh settings must not be Configured (disabled, no role)")
	}
}

func TestConfigured(t *testing.T) {
	cases := []struct {
		name    string
		enabled bool
		roleID  int64
		want    bool
	}{
		{"disabled no role", false, 0, false},
		{"disabled with role", false, 123, false},
		{"enabled no role", true, 0, false},
		{"enabled with role", true, 123, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &Settings{Enabled: tc.enabled, RoleID: tc.roleID}
			if got := s.Configured(); got != tc.want {
				t.Fatalf("Configured() = %v, want %v", got, tc.want)
			}
		})
	}
	var nilS *Settings
	if nilS.Configured() {
		t.Fatal("nil settings must report not Configured")
	}
}

func TestIDRoundTrip(t *testing.T) {
	const raw = "1234567890123456789"
	if got := sid(pid(raw)); got != raw {
		t.Fatalf("sid(pid(%q)) = %q, want %q", raw, got, raw)
	}
	if pid("not-a-number") != 0 {
		t.Fatal("pid of a non-numeric string must be 0")
	}
}
