package screens

import (
	"image"
	"sort"

	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/trace"
	"github.com/hajimehoshi/ebiten/v2"
)

// How the *hand* is arranged: what the combat screen does when a sort tab is pressed.
//
// **The modes, the comparisons and the tab block itself are in handsort.go** *(2026-09-05)*, which
// belongs to no screen — the worm screen sorts its offer with the same three tabs. What is left
// here is the half only a duel has: a queue to resync and a row of cards to send sliding.
//
// **The order of the queue is a rule as of 2026-08-26** *(owner's call)*. It was not one for the
// first three months of this screen, and this file was written under the old rule: cross-category
// order is still regrouped by `combat.ResolutionOrder` and a hand is still *counted* rather than
// read in sequence, but a growing ring now steps between the cards of a single blow, so the card
// that goes first pays for the card behind it. Sorting a queued hand therefore re-prices it.
//
// **That is accepted rather than guarded** *(owner's call)*. These buttons stay live and a bad sort
// can cost the player damage — the intent is that the order of the cards is something worth paying
// attention to, and a control that went dead the moment it mattered would be teaching the opposite.
//
// What the buttons are still *for* is reading eight overlapping cards: cost tells you what you can
// afford, type tells you what the round is made of, element tells you what a mix is worth.
//
// **The sort re-applies on every refill**, not only when a button is pressed, so a hand dealt
// at the end of a round arrives already arranged and a newly drawn card lands where it belongs
// instead of on the right-hand end. A drag still moves a card and still survives until the next
// refill, at which point the sort reclaims the row.
//
// **The mode survives Init**, unlike everything else on this screen — and it now survives leaving
// the screen entirely, being `gs.HandSort` rather than a field here. It is a reading preference
// rather than a fact about a duel.

// The three sort tabs are the top three rungs of the control column — see controlcolumn.go,
// which owns the geometry the frame's ledger button shares.

// sortColumnRect is what the block occupies, for anything measuring against it.
func sortColumnRect(gs *state.GlobalState) image.Rectangle {
	first := sortTabRect(gs, 0)
	last := sortTabRect(gs, len(sortButtonSpecs)-1)
	return image.Rect(first.Min.X, first.Min.Y, first.Max.X, last.Max.Y)
}

// setSort switches the arrangement, rearranges the hand and sends every card that moved
// sliding to its new place.
//
// Pressing the active mode again re-sorts rather than doing nothing, which is what makes it
// the way to undo a drag: the button the player is looking at is already the one describing
// the order they want back.
//
// **The cards have already moved by the time the slides exist**, exactly as they have for a
// discard or a deal — see spendSelected. The hand is in its new order the instant sortHand
// returns and every slide is a ghost of a card that is already where it is going.
func (s *CombatScene) setSort(mode handSort) {
	s.sortMode = mode

	s.theatre.slides = slidesFor(s.theatre.slides, s.sortHand(),
		func(i int) actionCard { return s.hand[i].actionCard },
		func(i int) int { return selectedLift(s.hand[i].selected) })

	trace.Logf("input", "hand sorted by %v -> %s", mode, handLabel(s.hand))
}

// sortHand rearranges the hand into the current mode.
//
// **Stable**, so two genuinely identical cards keep the order they were in — one of them may
// be selected and therefore lifted out of the row, and a card jumping sideways because its twin
// was dealt is a movement with no cause on screen.
//
// **It resyncs the queue, and that is not housekeeping.** The list is the authority on the
// queue's order as well as its membership, and `handIndexForQueue` is the inverse of that one
// walk — so a hand rearranged under a stale `fighterActions` would leave the hand preview
// naming a hand the cards it now holds do not make. Nothing about the *round*
// changes: the queue holds the same cards, and order is not something the engine reads.
//
// **It returns the permutation it applied** — for each new position, the index that card came
// from — because a card sliding to its new place has to know where it set off from, and two
// identical cards cannot be told apart afterwards by looking at them. Sorting a slice of
// indices and rebuilding the hand from it is what makes that answer available at all; sorting
// the cards in place throws it away.
//
// **It reads `s.sortMode`, which is this screen's working copy of `gs.HandSort`.** The
// preference is global — see that field — but neither of this function's callers is handed a
// `*GlobalState`: a refill is reached from a button callback, and `OpeningCards` builds a bare
// scene with no state at all to answer what a seed deals. Threading one in would put the whole
// deck path in the business of knowing about a screen, and a button reaching global state is
// exactly what this package's callbacks do not do. `Init` loads the copy and
// `updateSortButtons` writes it back each tick, so the two cannot disagree for a frame.
func (s *CombatScene) sortHand() []int {
	mode := s.sortMode

	order := make([]int, len(s.hand))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool {
		return handLess(mode, s.hand[order[i]].actionCard, s.hand[order[j]].actionCard)
	})

	sorted := make([]paletteCard, len(s.hand))
	for to, from := range order {
		sorted[to] = s.hand[from]
	}
	s.hand = sorted

	s.syncQueue()
	return order
}

// updateSortButtons runs the column and latches whichever mode is active.
//
// **All three go dead outside planning**, and that is a rule rather than tidiness: a card that
// has resolved is drawn from the hand slot it flew out of — see resolvedCard.handIndex — so
// rearranging the hand mid-round would light the wrong card on the table. The deck overlay
// takes them out for the reason it takes out everything else: it is a dialog.
func (s *CombatScene) updateSortButtons(gs *state.GlobalState) {
	s.sortTabs.update(gs, s.planning() && !s.modalUp())

	// **The write back is here rather than in the callback**, because a button's OnClick reaches
	// no global state on any screen in this package — see PostBattleScene.skipping for the same
	// shape. The scene's copy is what a press moves; this is where it becomes the preference every
	// other screen reads.
	setHandSort(gs, s.sortMode)
}

// drawSortButtons draws the column.
func (s *CombatScene) drawSortButtons(gs *state.GlobalState, screen *ebiten.Image) {
	s.sortTabs.draw(gs, screen)
}

// buildSortButtons builds the column. A method on the scene rather than a free function because
// the callback has to reach the scene's own state, which is the same reason every other widget on
// this screen is built here.
func (s *CombatScene) buildSortButtons() {
	s.sortTabs = newSortTabs(sortTabRect, s.setSort)
}
