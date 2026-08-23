package pyramid

import "testing"

// The ascent curve: every room grows an opponent's HP and DMG by AscentGrowthPct, compounding, so
// winning is what makes the next fight harder. See ScaleToFight.
//
// This package has no test file until now and needs no window — it imports `data` and
// `internal/combat` and nothing else, which is the property that lets anything headless read
// the curve.

func TestTheFirstFightIsTheBaseline(t *testing.T) {
	// Floor 1's outer room is fight 0 and takes the record's own numbers. If this ever scales, a
	// roster tuned by hand is being read through a multiplier nobody applied on purpose.
	for _, base := range []int{0, 1, 5, 100, 400} {
		if got := ScaleToFight(base, 0); got != base {
			t.Errorf("fight 0 scaled %d to %d, want it untouched", base, got)
		}
	}
}

func TestEachRoomGrowsOnTheOneBeforeIt(t *testing.T) {
	// True compounding — 100 x 1.1^n, truncated once — rather than a stat re-truncated every room.
	// Written out rather than computed so the test cannot restate the implementation, and these are
	// the figures a calculator gives: the fixed-point multiplier is an implementation detail and
	// must not be visible in the answers.
	const base = 100
	want := []int{100, 110, 121, 133, 146, 161, 177, 194, 214, 235}

	for fight, w := range want {
		if got := ScaleToFight(base, fight); got != w {
			t.Errorf("fight %d grew %d to %d, want %d", fight, base, got, w)
		}
	}
}

func TestTheCurveOnlyEverGrows(t *testing.T) {
	// A stat that went down a room would be a difficulty curve with a dip in it, which is worse
	// than a flat one: the player would learn that some rooms are free.
	for _, base := range []int{1, 5, 9, 10, 11, 80, 400} {
		prev := ScaleToFight(base, 0)
		for fight := 1; fight < 24; fight++ {
			got := ScaleToFight(base, fight)
			if got < prev {
				t.Errorf("base %d shrank from %d to %d at fight %d", base, prev, got, fight)
			}
			prev = got
		}
	}
}

func TestASmallStatStillClimbs(t *testing.T) {
	// **This is the test that caught the first implementation.** Compounding the *stat* rather than
	// the multiplier truncates `5 * 110 / 100` back to 5, so every stat below 10 was frozen for the
	// whole ascent — and half the roster opens on DMG 5 or 6, which is exactly the band the curve
	// exists to lift. It looks correct on a 100 HP enemy and does nothing at all on a Giant Bat.
	for _, base := range []int{1, 4, 5, 6, 9} {
		if got := ScaleToFight(base, 8); got <= base {
			t.Errorf("a stat of %d is still %d eight rooms in — the curve rounds it away", base, got)
		}
	}

	// Slow is fine and expected: 5 grows by half a point a room, so the first room cannot move it.
	// What must not happen is never moving.
	if got := ScaleToFight(5, 1); got != 5 {
		t.Errorf("DMG 5 reached %d after one room, want 5 — truncation is the intended rounding", got)
	}
}

func TestAFloorIsThreeRooms(t *testing.T) {
	if FightsPerFloor != 3 {
		t.Fatalf("a floor holds %d fights — the tower is 8 floors x 3", FightsPerFloor)
	}
	for floor, want := range map[int]int{1: 0, 2: 3, 3: 6, 8: 21} {
		if got := FirstFightOnFloor(floor); got != want {
			t.Errorf("floor %d opens on fight %d, want %d", floor, got, want)
		}
	}
	// Floors count from one, so anything below that is the first fight rather than a negative
	// index into the ascent.
	if got := FirstFightOnFloor(0); got != 0 {
		t.Errorf("floor 0 mapped to fight %d, want 0", got)
	}
}
