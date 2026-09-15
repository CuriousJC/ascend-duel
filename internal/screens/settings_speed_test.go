package screens

import "testing"

// The speed strip under the game-speed bar. These need no window and draw nothing — they are about
// the table the movement is derived from, which is the half that can be wrong silently.

func TestEverySpeedStripShuffleIsAPermutation(t *testing.T) {
	// **A bad row draws a plausible hand, which is why this exists.** Two cards landing on one seat
	// looks like an overlap and the seat nobody claimed looks like a gap, so the eye reports
	// "slightly odd shuffle" rather than "broken table" — and the strip loops, so it would be seen
	// a hundred times without being diagnosed.
	for n, perm := range speedStripShuffles {
		var seen [speedStripCards]bool
		for card, seat := range perm {
			if seat < 0 || seat >= speedStripCards {
				t.Errorf("shuffle %d sends card %d to seat %d, off the table", n, card, seat)
				continue
			}
			if seen[seat] {
				t.Errorf("shuffle %d sends two cards to seat %d", n, seat)
			}
			seen[seat] = true
		}
		for seat, ok := range seen {
			if !ok {
				t.Errorf("shuffle %d leaves seat %d empty", n, seat)
			}
		}
	}
}

func TestEverySpeedStripShuffleActuallyMovesSomething(t *testing.T) {
	// The identity is a valid permutation and a useless shuffle: the cards would lift, hold and
	// land exactly where they started, which reads as the animation having failed.
	for n, perm := range speedStripShuffles {
		moved := false
		for card, seat := range perm {
			if card != seat {
				moved = true
				break
			}
		}
		if !moved {
			t.Errorf("shuffle %d is the identity — nothing moves", n)
		}
	}
}

func TestTheStripsProgressStaysInsideOneCard(t *testing.T) {
	// staggered shifts each card's start and has to clamp at both ends: a negative progress puts a
	// card off the wrong side of the table and one over 1 overshoots its seat and comes back.
	s := &speedStrip{keys: make([]string, speedStripCards)}
	for step := 0; step <= 100; step++ {
		s.p = float64(step) / 100
		for i := range s.keys {
			if got := s.staggered(i); got < 0 || got > 1 {
				t.Fatalf("card %d at p=%.2f has progress %v", i, s.p, got)
			}
		}
	}
}

func TestTheStripRunsAtThePlayersSpeed(t *testing.T) {
	// **The whole point of the widget**: its phases are fractions of the same beat every card
	// flight in a duel is, so the bar moves the illustration and the thing it illustrates together.
	// This is the clock test's argument applied to the one screen that shows the clock off.
	was := Speed()
	defer SetSpeed(was)

	SetSpeed(1)
	slow := speedInTicks()

	SetSpeed(2)
	if fast := speedInTicks(); fast >= slow {
		t.Errorf("the deal takes %d ticks at 1x and %d at 2x — the strip is not on the speed",
			slow, fast)
	}
}
