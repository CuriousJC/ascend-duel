package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/profile"
)

// The speed setting: the bar's arithmetic, and the one clock it moves.
//
// **What is being defended is that speed is a view.** clock.go is the game's one pace and every
// duration in the game is a fraction of it, so scaling it moves pictures and nothing else — a
// whole round is resolved before playback begins. A test that a slider maps 0..1 onto 0.5..2 is
// half of that; the other half is that the number the rest of the package reads actually changes.

// atSpeed runs f with the game speed set, and puts it back afterwards. The scale is package state,
// so a test that left it moved would change the pace of every test after it.
func atSpeed(scale float64, f func()) {
	was := Speed()
	SetSpeed(scale)
	defer SetSpeed(was)
	f()
}

func TestTheBarCoversExactlyTheAllowedSpeeds(t *testing.T) {
	if got := speedFor(0); got != profile.SpeedMin {
		t.Errorf("the left end of the bar is %v, want the slowest allowed %v", got, profile.SpeedMin)
	}
	if got := speedFor(1); got != profile.SpeedMax {
		t.Errorf("the right end of the bar is %v, want the fastest allowed %v", got, profile.SpeedMax)
	}
}

func TestAPositionSurvivesBeingReadBack(t *testing.T) {
	// Init puts the knob where the game already is, so the two directions have to agree or the
	// bar jumps the first time the screen is opened.
	for _, v := range []float64{0, 0.25, 0.5, 1} {
		if got := speedValue(speedFor(v)); got != v {
			t.Errorf("a bar at %v read back as %v", v, got)
		}
	}
}

func TestTheTunedSpeedIsOnTheBar(t *testing.T) {
	// 1 is what every duration in the game was written against, so it has to be a position the
	// player can actually return to rather than something between two of them.
	v := speedValue(1)
	if v < 0 || v > 1 {
		t.Fatalf("the tuned speed sits at %v, off the bar entirely", v)
	}
	if got := speedFor(v); got != 1 {
		t.Errorf("the position for the tuned speed gives %v, want 1", got)
	}
}

func TestASpeedOfZeroIsIgnoredRatherThanApplied(t *testing.T) {
	// **A game speed of zero is not a speed** — it stops every clock in the game. profile has
	// already normalised anything off disk, so this is the last guard rather than the only one.
	atSpeed(1, func() {
		SetSpeed(0)
		if Speed() != 1 {
			t.Errorf("a speed of zero was applied: the game is at %v", Speed())
		}
		SetSpeed(-2)
		if Speed() != 1 {
			t.Errorf("a negative speed was applied: the game is at %v", Speed())
		}
	})
}

func TestFasterMeansFewerTicks(t *testing.T) {
	var slow, tuned, fast int
	atSpeed(profile.SpeedMin, func() { slow = speedTicks() })
	atSpeed(1, func() { tuned = speedTicks() })
	atSpeed(profile.SpeedMax, func() { fast = speedTicks() })

	if !(slow > tuned && tuned > fast) {
		t.Errorf("the beat runs %d slow, %d tuned, %d fast — want it strictly shortening", slow, tuned, fast)
	}
	if tuned != beatTicks {
		t.Errorf("at the tuned speed the beat is %d, want the constant %d it was written against",
			tuned, beatTicks)
	}
}

func TestTheSpeedMovesEveryClockAndNotJustTheBeat(t *testing.T) {
	// **The failure this exists for is a second clock**, which the package has had before: a
	// duration written as a raw number rather than as a fraction of the beat is one the setting
	// cannot reach. `beat` is how every clock that is not the speed itself is written, so it is
	// what has to move.
	var slow, fast int
	atSpeed(profile.SpeedMin, func() { slow = beat(1, 1) })
	atSpeed(profile.SpeedMax, func() { fast = beat(1, 1) })

	if slow <= fast {
		t.Errorf("a whole beat is %d slow and %d fast — the setting is not reaching beat()", slow, fast)
	}
}

func TestANeverShorterThanATick(t *testing.T) {
	// A small enough fraction of a fast enough speed becomes a movement that does not happen.
	atSpeed(profile.SpeedMax, func() {
		if got := beat(1, 1000); got < 1 {
			t.Errorf("a thousandth of a beat is %d ticks, want at least 1", got)
		}
	})
}
