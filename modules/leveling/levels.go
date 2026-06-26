package leveling

// XP curve. The XP required to advance from level L to L+1 is
//
//	5*L^2 + 50*L + 100
//
// (the widely-used MEE6 curve), so each level costs progressively more. All
// helpers operate on cumulative XP totals.

// xpForNext returns the XP needed to go from level to level+1.
func xpForNext(level int) int64 {
	l := int64(level)
	return 5*l*l + 50*l + 100
}

// xpForLevel returns the cumulative XP required to reach a level (level 0 = 0).
func xpForLevel(level int) int64 {
	var total int64
	for i := 0; i < level; i++ {
		total += xpForNext(i)
	}
	return total
}

// levelForXP returns the level a cumulative XP total corresponds to.
func levelForXP(xp int64) int {
	if xp <= 0 {
		return 0
	}
	level := 0
	for xp >= xpForLevel(level+1) {
		level++
	}
	return level
}

// progress describes a member's standing within their current level.
type progress struct {
	Level     int
	Into      int64 // XP earned into the current level
	Need      int64 // XP span of the current level
	Total     int64 // lifetime XP
	NextTotal int64 // cumulative XP needed for the next level
}

// progressFor computes the per-level progress for a cumulative XP total.
func progressFor(totalXP int64) progress {
	level := levelForXP(totalXP)
	base := xpForLevel(level)
	need := xpForNext(level)
	return progress{
		Level:     level,
		Into:      totalXP - base,
		Need:      need,
		Total:     totalXP,
		NextTotal: base + need,
	}
}
