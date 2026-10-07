package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
)

// The sort tabs where they are shared: one preference over every screen that deals a hand, and the
// essence screen's own row arranged by it. The combat screen's half is combat_sort_test.go.
//
// Nothing here creates an ebiten.Image — the same narrow exception the other tests in this package
// take.

// TestTheSortPreferenceIsOneThingForTheWholeGame. It lives on the global state so a player who
// arranges a hand by element does not meet a reward offer arranged by cost. Both scenes keep a
// working copy and load it in Init; this is the load.
func TestTheSortPreferenceIsOneThingForTheWholeGame(t *testing.T) {
	gs := testRun()
	gs.ScreenWidth, gs.ScreenHeight = 1920, 1080

	ui.SetHandSort(gs, ui.SortByElement)

	var post PostBattleScene
	post.Init(gs)
	if post.mode != ui.SortByElement {
		t.Errorf("the essence screen opened sorting by %v, want %v", post.mode, ui.SortByElement)
	}

	if got := ui.HandSortOf(gs); got != ui.SortByElement {
		t.Errorf("the state reports %v, want %v", got, ui.SortByElement)
	}
}

// TestAFreshStateSortsByCost. Cost is the zero value on purpose, so a state nobody has touched is
// already arranged the way a fresh screen is — including the one OpeningCards builds with no
// global state at all.
func TestAFreshStateSortsByCost(t *testing.T) {
	if got := ui.HandSortOf(&state.GlobalState{}); got != ui.SortByCost {
		t.Errorf("a fresh state sorts by %v, want %v", got, ui.SortByCost)
	}
	if got := ui.HandSortOf(nil); got != ui.SortByCost {
		t.Errorf("no state at all sorts by %v, want %v", got, ui.SortByCost)
	}
}

// TestTheEssenceOfferIsArrangedByTheSortMode. The row an essence is pointed at obeys the same three tabs
// the hand does — which is the whole point of the mode being global.
func TestTheEssenceOfferIsArrangedByTheSortMode(t *testing.T) {
	gs := testRun()
	gs.ScreenWidth, gs.ScreenHeight = 1920, 1080

	for _, mode := range []ui.HandSort{ui.SortByCost, ui.SortByForm, ui.SortByElement} {
		ui.SetHandSort(gs, mode)

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
			if ui.HandLess(mode, b, a, gs.Run.WornRelics()) {
				t.Errorf("sorted by %v, card %d comes before card %d in the row", mode, i, i-1)
			}
		}
	}
}

// TestTheEssenceScreensSortBlockFitsBesideItsRow. The block hangs off the offer row's right edge, and
// the row is centered and eight cards wide — so the two are competing for the same pixels. A block
// running off the screen would be a control the player cannot press.
func TestTheEssenceScreensSortBlockFitsBesideItsRow(t *testing.T) {
	gs := testRun()
	gs.ScreenWidth, gs.ScreenHeight = 1920, 1080

	var s PostBattleScene
	s.Init(gs)

	row := s.offerRow(gs)
	block := s.tabs.Rect(gs)

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

// TestTheRewardRowStandsWhereTheHandDoes. The bottom third is one layout: the reward screen's row
// and its sort block are the combat screen's hand band and block, to the pixel.
func TestTheRewardRowStandsWhereTheHandDoes(t *testing.T) {
	gs := testRun()
	gs.ScreenWidth, gs.ScreenHeight = 1920, 1080

	var s PostBattleScene
	s.Init(gs)

	for i := range ui.SortButtonSpecs {
		if got, want := s.tabRect(gs, i), sortTabRect(gs, i); got != want {
			t.Errorf("the reward screen's tab %d is at %v, the combat screen's at %v", i, got, want)
		}
	}
	if got, want := s.offerRow(gs), handBand(gs, len(s.offer)); got != want {
		t.Errorf("the reward row is at %v, the combat hand of %d at %v", got, len(s.offer), want)
	}
}
