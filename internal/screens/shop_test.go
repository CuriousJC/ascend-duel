package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// The shelf's arithmetic and the two rows' geometry, both of which need no window — the same narrow
// exception the other tests in this package take. Nothing here creates an ebiten.Image.

func TestTheShelfHoldsThreeRingsTheRunIsNotWearing(t *testing.T) {
	// A ring already on your hand offered back to you is a seat spent saying nothing, and `Buy`
	// would refuse the click anyway.
	gs := testRun()

	shelf := dealShelf(gs)
	if len(shelf) != shelfSize {
		t.Fatalf("the shelf holds %d, want %d", len(shelf), shelfSize)
	}

	worn := map[string]bool{}
	for _, key := range gs.Run.Worn() {
		worn[key] = true
	}

	seen := map[string]bool{}
	for _, item := range shelf {
		if worn[item.key] {
			t.Errorf("%s is for sale and already worn", item.key)
		}
		if seen[item.key] {
			t.Errorf("%s is on the shelf twice", item.key)
		}
		if _, ok := session.RingPrice(item.key); !ok {
			t.Errorf("%s is for sale and is in no record", item.key)
		}
		seen[item.key] = true
	}
}

func TestTheSameFightWalksIntoTheSameShop(t *testing.T) {
	// **Per fight, like the opponent and the worm offer**, so a defeat and a retry meet the same
	// shelf rather than rerolling it. The stream is what makes that true; this is what says so.
	first := shelfKeys(dealShelf(testRun()))
	again := shelfKeys(dealShelf(testRun()))

	if len(first) != len(again) {
		t.Fatalf("two deals of one fight offered %d and %d rings", len(first), len(again))
	}
	for i := range first {
		if first[i] != again[i] {
			t.Fatalf("one fight dealt %v and then %v", first, again)
		}
	}
}

func TestALaterFightIsADifferentShop(t *testing.T) {
	// Not a promise that no two fights ever coincide — three of seventeen will sometimes repeat —
	// but a shelf identical across a whole run would mean the fight index never reached the seed.
	gs := testRun()

	first := shelfKeys(dealShelf(gs))
	for i := 0; i < 6; i++ {
		gs.Run.WonFight(0)
		if !sameKeys(first, shelfKeys(dealShelf(gs))) {
			return
		}
	}
	t.Errorf("seven fights all offered %v", first)
}

func sameKeys(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestTheShelfIsCutToThreeAndNotPaddedToIt(t *testing.T) {
	// **The row is what is left, cut to three**, so a catalogue smaller than the shelf draws a short
	// row rather than one with holes in it. That case is unreachable today — seventeen rings against
	// a cap of five leaves at least twelve unworn — so what is checkable is the cut itself, and the
	// case that *is* reachable: a scene reached before main built a run.
	if got := len(dealShelf(testRun())); got != shelfSize {
		t.Errorf("a fresh run was offered %d rings, want %d", got, shelfSize)
	}
	if len(session.Rings())-combat.MaxWornRings < shelfSize {
		t.Errorf("the catalogue holds %d rings, which is no longer enough to fill a shelf against a "+
			"cap of %d — dealShelf's cut is now load-bearing and wants a test of its own",
			len(session.Rings()), combat.MaxWornRings)
	}

	if got := dealShelf(&state.GlobalState{}); got != nil {
		t.Errorf("a game with no run was offered %v", shelfKeys(got))
	}
}

func TestTheTwoRowsDoNotOverlapAndStayOnScreen(t *testing.T) {
	// Both rows are full-size cards at a fixed pitch, and neither can exceed five. This is what
	// says the fixed pitch actually fits the screen — the hand's compressing pitch exists because
	// eight cards do not.
	gs := &state.GlobalState{ScreenWidth: 1280, ScreenHeight: 960}

	for n := 1; n <= combat.MaxWornRings; n++ {
		left := rowSlot(gs, 0, n, 0)
		right := rowSlot(gs, n-1, n, 0)

		if left.Min.X < 0 || right.Max.X > gs.ScreenWidth {
			t.Errorf("a row of %d runs from %d to %d", n, left.Min.X, right.Max.X)
		}
		for i := 1; i < n; i++ {
			if rowSlot(gs, i, n, 0).Min.X <= rowSlot(gs, i-1, n, 0).Max.X {
				t.Errorf("a row of %d overlaps at seat %d", n, i)
			}
		}
	}

	shelf := rowSlot(gs, 0, shelfSize, gs.PctY(shelfRowPct))
	worn := rowSlot(gs, 0, combat.MaxWornRings, gs.PctY(wornRowPct))
	if shelf.Max.Y >= worn.Min.Y {
		t.Errorf("the shelf ends at %d and the worn row starts at %d", shelf.Max.Y, worn.Min.Y)
	}
	if worn.Max.Y+shopFigureGap+shopFigureSize >= gs.PctY(offerButtonsPct) {
		t.Error("the worn row's sell figures run into the Leave button")
	}
}
