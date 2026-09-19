package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// The offer row when an essence takes more than one card. Nothing here creates an ebiten.Image —
// the same narrow exception the rest of this package's tests take.

// wearingNecklace is a run wearing the relic that widens an essence, with an offer row dealt.
func wearingNecklace(t *testing.T) (*PostBattleScene, *state.GlobalState) {
	t.Helper()

	gs := testRun()
	if !gs.Run.Wear("essence-targets") {
		t.Fatal("the Cloud Necklace would not go on")
	}
	return &PostBattleScene{offer: dealOffer(gs)}, gs
}

// **A relic widens the offer row's requirement, and nothing on a record does.** The reward screen
// reads the run rather than the essence, which is what makes one relic move every essence in the
// catalog at once.
func TestTheOfferAsksForAsManyCardsAsTheRunSays(t *testing.T) {
	gs := testRun()
	bare := &PostBattleScene{offer: dealOffer(gs)}
	if got := bare.targets(gs); got != 1 {
		t.Errorf("a bare run is asked for %d cards, want 1", got)
	}

	s, gs := wearingNecklace(t)
	if got := s.targets(gs); got != 2 {
		t.Errorf("a run wearing the necklace is asked for %d cards, want 2", got)
	}
}

// **Never more cards than the row is holding.** A requirement past the offer would leave every
// essence on the screen unclickable, which is worse than one that reaches less far.
func TestTheOfferNeverAsksForMoreCardsThanItHolds(t *testing.T) {
	s, gs := wearingNecklace(t)
	s.offer = s.offer[:1]
	if got := s.targets(gs); got != 1 {
		t.Errorf("a one-card offer asks for %d cards, want 1", got)
	}
}

// **A full selection replaces its oldest card**, so there is no state the player has to deselect
// their way out of — and with one target that is the pick simply moving, which is what the row
// always did.
func TestAFullSelectionReplacesItsOldestCard(t *testing.T) {
	s, gs := wearingNecklace(t)

	s.selectOffered(gs, 0)
	s.selectOffered(gs, 1)
	if got := s.selectedSlots(); len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Fatalf("two clicks selected %v, want [0 1]", got)
	}

	s.selectOffered(gs, 2)
	if got := s.selectedSlots(); len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Errorf("a third click selected %v, want [1 2] — the oldest replaced", got)
	}

	// Clicking a selected card still puts it back.
	s.selectOffered(gs, 1)
	if got := s.selectedSlots(); len(got) != 1 || got[0] != 2 {
		t.Errorf("deselecting left %v, want [2]", got)
	}
}

// **The order of a selection is the order of the row**, whatever order the cards were clicked in.
// The settled cards land left to right in the order they were sitting in, so there is no separate
// click order to learn.
func TestTheSelectionReadsInRowOrder(t *testing.T) {
	s, gs := wearingNecklace(t)

	s.selectOffered(gs, 3)
	s.selectOffered(gs, 1)
	if got := s.selectedSlots(); len(got) != 2 || got[0] != 1 || got[1] != 3 {
		t.Errorf("clicking 3 then 1 reads as %v, want [1 3]", got)
	}
}

// **A prize is lit exactly when clicking it would take it**, and the reach is a ceiling rather than
// a quota: one card is a legal spend however far the relics let an essence stretch. An essence that
// looked available and did nothing is the failure this predicate exists for, and so is a player
// holding a Cloud Necklace who cannot change a single card.
func TestAPrizeIsDeadUntilACardIsSelectedAndOneIsEnough(t *testing.T) {
	s, gs := wearingNecklace(t)
	s.prizes = []prize{{essence: firstEssence(t)}}

	if s.essenceSpendable(gs, s.prizes[0]) {
		t.Error("the essence was spendable with nothing selected")
	}

	s.selectOffered(gs, 0)
	if !s.essenceSpendable(gs, s.prizes[0]) {
		t.Error("one card was selected and the essence would not take it")
	}

	s.selectOffered(gs, 1)
	if !s.essenceSpendable(gs, s.prizes[0]) {
		t.Error("the essence was dead with both cards it reaches selected")
	}
}

// **The tooltip says what this click would do, not how far the essence could go.** A player who has
// picked one card is about to change one card, and a panel promising two would be describing a
// click they are not making.
func TestTheTooltipCountsTheCardsThatAreActuallyPicked(t *testing.T) {
	s, gs := wearingNecklace(t)

	if got := s.reachNow(gs); got != 2 {
		t.Errorf("with nothing picked the panel says %d cards, want the ceiling of 2", got)
	}

	s.selectOffered(gs, 0)
	if got := s.reachNow(gs); got != 1 {
		t.Errorf("with one card picked the panel says %d cards, want 1", got)
	}

	s.selectOffered(gs, 1)
	if got := s.reachNow(gs); got != 2 {
		t.Errorf("with two cards picked the panel says %d cards, want 2", got)
	}
}

func firstEssence(t *testing.T) session.Essence {
	t.Helper()

	all := session.Essences()
	if len(all) == 0 {
		t.Fatal("the catalog holds no essences")
	}
	return all[0]
}
