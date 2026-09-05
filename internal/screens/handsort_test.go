package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/state"
)

// The sort tabs where they are shared: one preference over every screen that deals a hand, and the
// worm screen's own row arranged by it. The combat screen's half is combat_sort_test.go.
//
// Nothing here creates an ebiten.Image — the same narrow exception the other tests in this package
// take.

// TestTheSortPreferenceIsOneThingForTheWholeGame. It lives on the global state so a player who
// arranges a hand by element does not meet a reward offer arranged by cost. Both scenes keep a
// working copy and load it in Init; this is the load.
func TestTheSortPreferenceIsOneThingForTheWholeGame(t *testing.T) {
	gs := testRun()
	gs.ScreenWidth, gs.ScreenHeight = 1920, 1080

	setHandSort(gs, sortByElement)

	var post PostBattleScene
	post.Init(gs)
	if post.sortMode != sortByElement {
		t.Errorf("the worm screen opened sorting by %v, want %v", post.sortMode, sortByElement)
	}

	if got := handSortOf(gs); got != sortByElement {
		t.Errorf("the state reports %v, want %v", got, sortByElement)
	}
}

// TestAFreshStateSortsByCost. Cost is the zero value on purpose, so a state nobody has touched is
// already arranged the way a fresh screen is — including the one OpeningCards builds with no
// global state at all.
func TestAFreshStateSortsByCost(t *testing.T) {
	if got := handSortOf(&state.GlobalState{}); got != sortByCost {
		t.Errorf("a fresh state sorts by %v, want %v", got, sortByCost)
	}
	if got := handSortOf(nil); got != sortByCost {
		t.Errorf("no state at all sorts by %v, want %v", got, sortByCost)
	}
}

// TestTheWormOfferIsArrangedByTheSortMode. The row a worm is pointed at obeys the same three tabs
// the hand does — which is the whole point of the mode being global.
func TestTheWormOfferIsArrangedByTheSortMode(t *testing.T) {
	gs := testRun()
	gs.ScreenWidth, gs.ScreenHeight = 1920, 1080

	for _, mode := range []handSort{sortByCost, sortByType, sortByElement} {
		setHandSort(gs, mode)

		var s PostBattleScene
		s.Init(gs)
		if len(s.offer) < 2 {
			t.Fatalf("offered %d cards, nothing to order", len(s.offer))
		}

		for i := 1; i < len(s.offer); i++ {
			a, aok := gs.Run.Card(s.offer[i-1])
			b, bok := gs.Run.Card(s.offer[i])
			if !aok || !bok {
				t.Fatalf("offer holds an index the deck does not: %v", s.offer)
			}
			if handLess(mode, b, a) {
				t.Errorf("sorted by %v, card %d comes before card %d in the row", mode, i, i-1)
			}
		}
	}
}

// TestTheWormScreensSortBlockFitsBesideItsRow. The block hangs off the offer row's right edge, and
// the row is centred and eight cards wide — so the two are competing for the same pixels. A block
// running off the screen would be a control the player cannot press.
func TestTheWormScreensSortBlockFitsBesideItsRow(t *testing.T) {
	gs := testRun()
	gs.ScreenWidth, gs.ScreenHeight = 1920, 1080

	var s PostBattleScene
	s.Init(gs)

	row := s.offerRow(gs)
	block := s.sortTabs.rect(gs)

	if block.Min.X < row.Max.X {
		t.Errorf("the block starts at %d, inside a row that ends at %d", block.Min.X, row.Max.X)
	}
	if block.Max.X > gs.ScreenWidth {
		t.Errorf("the block ends at %d, past a screen %d wide", block.Max.X, gs.ScreenWidth)
	}
	if block.Max.Y > gs.ScreenHeight {
		t.Errorf("the block ends at %d, past a screen %d tall", block.Max.Y, gs.ScreenHeight)
	}
}

// TestTheTwoScreensLeaveTheSameAirBesideTheirCards. Both blocks are hung off a row of cards rather
// than off a column of their own, and the same constant is what makes the two rows look alike.
func TestTheTwoScreensLeaveTheSameAirBesideTheirCards(t *testing.T) {
	gs := testRun()
	gs.ScreenWidth, gs.ScreenHeight = 1920, 1080

	var s PostBattleScene
	s.Init(gs)

	if got := s.sortTabs.rect(gs).Min.X - s.offerRow(gs).Max.X; got != sortColumnGap {
		t.Errorf("the worm screen leaves %d beside its row, want %d", got, sortColumnGap)
	}
	if got := sortColumnRect(gs).Min.X - (handBandLeft(gs) + cardBandWidth(gs)); got != 0 {
		t.Errorf("the combat screen's block is %d off the card band's right edge, want 0", got)
	}
}
