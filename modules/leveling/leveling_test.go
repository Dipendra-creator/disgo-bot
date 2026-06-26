package leveling

import "testing"

func TestXPForNext(t *testing.T) {
	// 5*L^2 + 50*L + 100
	cases := map[int]int64{0: 100, 1: 155, 2: 220, 10: 1100}
	for level, want := range cases {
		if got := xpForNext(level); got != want {
			t.Errorf("xpForNext(%d) = %d, want %d", level, got, want)
		}
	}
}

func TestXPForLevelMonotonic(t *testing.T) {
	if xpForLevel(0) != 0 {
		t.Errorf("xpForLevel(0) = %d, want 0", xpForLevel(0))
	}
	for l := 1; l <= 50; l++ {
		if xpForLevel(l) <= xpForLevel(l-1) {
			t.Fatalf("cumulative XP not increasing at level %d", l)
		}
	}
	// xpForLevel(1) is the cost of the first level.
	if xpForLevel(1) != xpForNext(0) {
		t.Errorf("xpForLevel(1) = %d, want %d", xpForLevel(1), xpForNext(0))
	}
}

func TestLevelForXP(t *testing.T) {
	if levelForXP(0) != 0 || levelForXP(-5) != 0 {
		t.Error("non-positive XP should be level 0")
	}
	if levelForXP(99) != 0 {
		t.Errorf("99 XP should still be level 0, got %d", levelForXP(99))
	}
	if levelForXP(100) != 1 {
		t.Errorf("100 XP should be level 1, got %d", levelForXP(100))
	}
	// Round-trip: the XP threshold of level L resolves back to L.
	for l := 0; l <= 60; l++ {
		if got := levelForXP(xpForLevel(l)); got != l {
			t.Errorf("levelForXP(xpForLevel(%d)) = %d", l, got)
		}
		// One XP short of the threshold is the previous level.
		if l > 0 {
			if got := levelForXP(xpForLevel(l) - 1); got != l-1 {
				t.Errorf("levelForXP(xpForLevel(%d)-1) = %d, want %d", l, got, l-1)
			}
		}
	}
}

func TestProgressFor(t *testing.T) {
	// Exactly at the level-3 threshold: 0 into the level.
	base := xpForLevel(3)
	p := progressFor(base)
	if p.Level != 3 || p.Into != 0 {
		t.Errorf("at threshold: level=%d into=%d, want 3/0", p.Level, p.Into)
	}
	if p.Need != xpForNext(3) {
		t.Errorf("Need = %d, want %d", p.Need, xpForNext(3))
	}
	// Halfway-ish into the level.
	p = progressFor(base + 10)
	if p.Level != 3 || p.Into != 10 {
		t.Errorf("mid-level: level=%d into=%d, want 3/10", p.Level, p.Into)
	}
}

func TestRandInt(t *testing.T) {
	for i := 0; i < 1000; i++ {
		v := randInt(15, 25)
		if v < 15 || v > 25 {
			t.Fatalf("randInt(15,25) out of range: %d", v)
		}
	}
	if randInt(7, 7) != 7 {
		t.Error("equal bounds should return the bound")
	}
	// Inverted range tolerated.
	v := randInt(25, 15)
	if v < 15 || v > 25 {
		t.Fatalf("inverted range out of bounds: %d", v)
	}
}

func TestDefaultSettings(t *testing.T) {
	s := defaultSettings(42)
	if !s.Enabled || s.XPMin != 15 || s.XPMax != 25 || s.XPCooldownSeconds != 60 {
		t.Errorf("unexpected defaults: %+v", s)
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
