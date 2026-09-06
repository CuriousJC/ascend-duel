package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// The offer's arithmetic, which needs no window — the same narrow exception the other tests in
// this package take. Nothing here creates an ebiten.Image.

func testRun() *state.GlobalState {
	return &state.GlobalState{RunSeed: 20260817, Run: session.New(session.StartingDeck())}
}

// TestTheOfferIsAWholeHandOffTheWholeDeck. The offer is a fresh deal, so it may name any card the
// player owns — not just the ones a fight happened to leave somewhere.
func TestTheOfferIsAWholeHandOffTheWholeDeck(t *testing.T) {
	gs := testRun()

	offer := dealOffer(gs)
	if len(offer) != handSize {
		t.Errorf("offered %d cards, want the hand size of %d", len(offer), handSize)
	}

	seen := map[int]bool{}
	for _, i := range offer {
		if i < 0 || i >= gs.Run.Size() {
			t.Errorf("offered index %d against a deck of %d", i, gs.Run.Size())
		}
		if seen[i] {
			t.Errorf("card %d was offered twice", i)
		}
		seen[i] = true
	}
}

// TestTheOfferIsSortedSoTheRowDoesNotJump. Which cards are offered is the random part; where each
// one sits in the row is not.
func TestTheOfferIsSortedSoTheRowDoesNotJump(t *testing.T) {
	offer := dealOffer(testRun())
	for i := 1; i < len(offer); i++ {
		if offer[i] <= offer[i-1] {
			t.Fatalf("offer is not in ascending deck order: %v", offer)
		}
	}
}

// TestTheOfferIsAFunctionOfTheFight — the same fight of the same run offers the same cards however
// many times it is reached, and the next fight offers different ones. This is the property the
// per-fight seeding exists for, and it is what stops a re-entered screen rerolling the reward.
func TestTheOfferIsAFunctionOfTheFight(t *testing.T) {
	gs := testRun()

	first := dealOffer(gs)
	if again := dealOffer(gs); !sameInts(first, again) {
		t.Errorf("the same fight offered %v then %v", first, again)
	}

	gs.Run.WonFight(0, 0)
	if next := dealOffer(gs); sameInts(first, next) {
		t.Errorf("fight 2 offered the same cards as fight 1: %v", next)
	}
}

// TestTheOfferSurvivesAThinnedDeck. Every removal shrinks the deck under the indices, and a deck
// eventually smaller than a hand is reachable by playing — so the offer has to cut to what exists
// rather than assuming there are eight cards to name.
func TestTheOfferSurvivesAThinnedDeck(t *testing.T) {
	gs := testRun()
	for gs.Run.Size() > 3 {
		gs.Run.Remove(0)
	}

	offer := dealOffer(gs)
	if len(offer) != 3 {
		t.Errorf("a deck of 3 offered %d cards", len(offer))
	}
	for _, i := range offer {
		if _, ok := gs.Run.Card(i); !ok {
			t.Errorf("offered index %d, which the deck does not hold", i)
		}
	}
}

// TestAnEmptyRunOffersNothing rather than panicking. Nothing in the game can empty a deck today,
// but the offer is index arithmetic over a list that only ever shrinks.
func TestAnEmptyRunOffersNothing(t *testing.T) {
	gs := &state.GlobalState{RunSeed: 1, Run: session.New(nil)}
	if offer := dealOffer(gs); len(offer) != 0 {
		t.Errorf("an empty deck offered %v", offer)
	}
	if offer := dealOffer(&state.GlobalState{RunSeed: 1}); len(offer) != 0 {
		t.Errorf("a nil run offered %v", offer)
	}
}

// TestThePrizeRowIsTwoWorms. **The money card went on 2026-08-22** — a win pays vitae by itself
// now, read out at the top of the screen — so the offer is the two creatures and nothing else, and
// taking neither is a button rather than a third card.
func TestThePrizeRowIsTwoWorms(t *testing.T) {
	ps := dealPrizes(testRun())

	if len(ps) != wormsOffered {
		t.Fatalf("offered %d prizes, want %d", len(ps), wormsOffered)
	}
	for _, p := range ps {
		if p.worm.Record == "" {
			t.Error("a prize seat is holding something that is not a worm")
		}
	}
}

// TestTwoDistinctWormsAreOffered. Two so the choice is a comparison; distinct because being
// offered the same worm twice is a choice that is not one, and it is a property of shuffling the
// catalogue rather than drawing from it twice.
func TestTwoDistinctWormsAreOffered(t *testing.T) {
	gs := testRun()

	offer := dealWorms(gs)
	if len(offer) != wormsOffered {
		t.Fatalf("offered %d worms, want %d", len(offer), wormsOffered)
	}
	if offer[0].Record == offer[1].Record {
		t.Errorf("both options are %s", offer[0].Record)
	}
}

// TestTheWormOfferIsAFunctionOfTheFight, like the cards: re-entering the screen must not reroll
// the reward, and the next fight must not repeat it verbatim.
func TestTheWormOfferIsAFunctionOfTheFight(t *testing.T) {
	gs := testRun()

	first := dealWorms(gs)
	again := dealWorms(gs)
	if first[0].Record != again[0].Record || first[1].Record != again[1].Record {
		t.Errorf("the same fight offered %v then %v", prizeNames(toPrizes(first)), prizeNames(toPrizes(again)))
	}

	gs.Run.WonFight(0, 0)
	next := dealWorms(gs)
	if first[0].Record == next[0].Record && first[1].Record == next[1].Record {
		t.Errorf("fight 2 offered the same pair as fight 1: %v", prizeNames(toPrizes(next)))
	}
}

// TestTheWormsAndTheCardsDoNotShareAStream. Adding a worm to the catalogue must not change which
// *cards* a fight offers — that is the failure the salts exist to prevent, and it is checkable
// here because both offers are functions of the same run and fight.
func TestTheWormsAndTheCardsDoNotShareAStream(t *testing.T) {
	gs := testRun()

	if seeds.ForFight(gs.RunSeed, seeds.WormOffer, gs.Run.Fight()) ==
		seeds.ForFight(gs.RunSeed, seeds.RewardHand, gs.Run.Fight()) {
		t.Error("the worm offer and the card offer are seeded identically")
	}
}

func toPrizes(ws []session.Worm) []prize {
	out := make([]prize, 0, len(ws))
	for _, w := range ws {
		out = append(out, prize{worm: w})
	}
	return out
}

func sameInts(a, b []int) bool {
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

// The reward screen puts both rows on one screen, and the two ways that goes wrong are a control
// under the cards and a selection nobody can see. Arithmetic, so it is checkable without a window.
func TestTheSkipButtonStandsBetweenTheWorms(t *testing.T) {
	gs := testState()
	prizes, button := wormRowSeats(gs, 2)

	if len(prizes) != 2 {
		t.Fatalf("laid out %d prize seats, want 2", len(prizes))
	}
	if button.Min.X < prizes[0].Max.X || button.Max.X > prizes[1].Min.X {
		t.Errorf("the button runs %d..%d, not between worms ending at %d and starting at %d",
			button.Min.X, button.Max.X, prizes[0].Max.X, prizes[1].Min.X)
	}

	// **Centred on the row it stands in**, or it reads as a control that happens to be near the
	// cards rather than as the third answer beside them.
	rowMid, buttonMid := (prizes[0].Min.Y+prizes[0].Max.Y)/2, (button.Min.Y+button.Max.Y)/2
	if rowMid != buttonMid {
		t.Errorf("the button is centred at y=%d against a card row centred at y=%d", buttonMid, rowMid)
	}
}

// **A lifted card must not reach the worms above it.** The lift is what says which card is
// selected, and the row it lifts into is the one the worms stand in — see offerRowPct, which moved
// to buy this clearance.
func TestASelectedOfferCardClearsTheWormRow(t *testing.T) {
	gs := testState()
	prizes, _ := wormRowSeats(gs, 2)
	row := offerRowOf(gs, handSize)

	if top := row.Min.Y - offerSelectedNudge; top <= prizes[0].Max.Y {
		t.Errorf("a lifted offer card reaches y=%d, inside a worm row ending at y=%d",
			top, prizes[0].Max.Y)
	}
	if row.Max.Y > gs.ScreenHeight {
		t.Errorf("the offer row reaches y=%d, past the %d-pixel screen", row.Max.Y, gs.ScreenHeight)
	}
}
