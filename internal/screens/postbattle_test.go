package screens

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

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

// TestThePrizeRowIsTwoEssences. **The money card went on 2026-08-22** — a win pays vitae by itself
// now, read out at the top of the screen — so the offer is the two creatures and nothing else, and
// taking neither is a button rather than a third card.
func TestThePrizeRowIsTwoEssences(t *testing.T) {
	ps := dealPrizes(testRun())

	if len(ps) != essencesOffered {
		t.Fatalf("offered %d prizes, want %d", len(ps), essencesOffered)
	}
	for _, p := range ps {
		if p.essence.Record == "" {
			t.Error("a prize seat is holding something that is not an essence")
		}
	}
}

// TestTwoDistinctEssencesAreOffered. Two so the choice is a comparison; distinct because being
// offered the same essence twice is a choice that is not one, and it is a property of shuffling the
// catalog rather than drawing from it twice.
func TestTwoDistinctEssencesAreOffered(t *testing.T) {
	gs := testRun()

	offer := dealEssences(gs)
	if len(offer) != essencesOffered {
		t.Fatalf("offered %d essences, want %d", len(offer), essencesOffered)
	}
	if offer[0].Record == offer[1].Record {
		t.Errorf("both options are %s", offer[0].Record)
	}
}

// TestTheEssenceOfferIsAFunctionOfTheFight, like the cards: re-entering the screen must not reroll
// the reward, and the next fight must not repeat it verbatim.
func TestTheEssenceOfferIsAFunctionOfTheFight(t *testing.T) {
	gs := testRun()

	first := dealEssences(gs)
	again := dealEssences(gs)
	if first[0].Record != again[0].Record || first[1].Record != again[1].Record {
		t.Errorf("the same fight offered %v then %v", prizeNames(toPrizes(first)), prizeNames(toPrizes(again)))
	}

	gs.Run.WonFight(0, 0)
	next := dealEssences(gs)
	if first[0].Record == next[0].Record && first[1].Record == next[1].Record {
		t.Errorf("fight 2 offered the same pair as fight 1: %v", prizeNames(toPrizes(next)))
	}
}

// TestTheEssencesAndTheCardsDoNotShareAStream. Adding an essence to the catalog must not change which
// *cards* a fight offers — that is the failure the salts exist to prevent, and it is checkable
// here because both offers are functions of the same run and fight.
func TestTheEssencesAndTheCardsDoNotShareAStream(t *testing.T) {
	gs := testRun()

	if seeds.ForFight(gs.RunSeed, seeds.EssenceOffer, gs.Run.Fight()) ==
		seeds.ForFight(gs.RunSeed, seeds.RewardHand, gs.Run.Fight()) {
		t.Error("the essence offer and the card offer are seeded identically")
	}
}

func toPrizes(ws []session.Essence) []prize {
	out := make([]prize, 0, len(ws))
	for _, w := range ws {
		out = append(out, prize{essence: w})
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
//
// **The essences touch and the way out stands off the end of them** *(owner's call, 2026-09-18)*.
// The pair is the question and is compared across, so a control cutting between them is a third
// card that is not one.
func TestTheSkipButtonStandsBesideTheEssences(t *testing.T) {
	gs := testState()
	prizes, button := essenceRowSeats(gs, 2)

	if len(prizes) != 2 {
		t.Fatalf("laid out %d prize seats, want 2", len(prizes))
	}
	if button.Min.X < prizes[1].Max.X {
		t.Errorf("the button starts at %d, inside a row whose last essence ends at %d",
			button.Min.X, prizes[1].Max.X)
	}
	if gap := prizes[1].Min.X - prizes[0].Max.X; gap != essenceRowGap {
		t.Errorf("the essences are %d apart, want the row's own gap of %d", gap, essenceRowGap)
	}

	// **Bottom-aligned to the cards**, or a control two fifths of a card tall floats in the row it
	// is standing in.
	if button.Max.Y != prizes[1].Max.Y {
		t.Errorf("the button ends at y=%d and the essences at y=%d", button.Max.Y, prizes[1].Max.Y)
	}
	if button.Max.Y != essenceRowBottom(gs) {
		t.Errorf("the row ends at y=%d and the band at y=%d", button.Max.Y, essenceRowBottom(gs))
	}
}

// **A lifted card must not reach the essences above it.** The lift is what says which card is
// selected, and the row it lifts into is the one the essences stand in — see offerRowPct, which moved
// to buy this clearance.
func TestASelectedOfferCardClearsTheEssenceRow(t *testing.T) {
	gs := testState()
	prizes, _ := essenceRowSeats(gs, 2)
	row := offerRowOf(gs, handSize)

	if top := row.Min.Y - offerSelectedNudge; top <= prizes[0].Max.Y {
		t.Errorf("a lifted offer card reaches y=%d, inside an essence row ending at y=%d",
			top, prizes[0].Max.Y)
	}
	if row.Max.Y > gs.ScreenHeight {
		t.Errorf("the offer row reaches y=%d, past the %d-pixel screen", row.Max.Y, gs.ScreenHeight)
	}
}

// **The payout and the offer share one screen and two columns** *(owner's call, 2026-09-18)*. The
// failure this pins is the one the merge could produce silently: a row centered on the screen
// standing half in the column the payout is being read in, so the two are read as one thing.
func TestTheEssenceRowStaysOutOfThePayoutsColumn(t *testing.T) {
	gs := testState()
	prizes, button := essenceRowSeats(gs, 2)

	split := gs.PctX(payoutColumnPct)
	for i, seat := range prizes {
		if seat.Min.X < split {
			t.Errorf("essence %d starts at x=%d, inside a payout column ending at %d",
				i, seat.Min.X, split)
		}
	}
	if button.Min.X < split {
		t.Errorf("the way out starts at x=%d, inside a payout column ending at %d",
			button.Min.X, split)
	}
	if mid := proseColumnMid(gs); mid >= split {
		t.Errorf("the payout is centered at x=%d, outside its own column ending at %d", mid, split)
	}
}

// **The payout ends on the edge the essences end on** *(owner's call, 2026-09-18)*, whatever the
// script's length — a fight paying no interest is one sentence shorter, and a block hung from the
// top of its column would float by exactly that sentence.
//
// **And it still has to fit the band it is read in**, between the relics above and the cards below.
// The wording is authored, so a longer script is a layout change and this is what says so.
func TestThePayoutIsAlignedToTheBottomOfTheEssenceRow(t *testing.T) {
	gs := testState()
	gs.Run = session.New(session.StartingDeck())
	gs.Run.WonFight(3, 40)

	lines := payoutLines(gs)
	if len(lines) < 2 {
		t.Fatalf("a won fight narrated %d lines", len(lines))
	}

	for _, n := range []int{len(lines), len(lines) - 1, 1} {
		bottom := proseTop(gs, n) + (n-1)*proseLineGap + proseLineHeight
		if want := essenceRowBottom(gs); bottom != want {
			t.Errorf("%d lines end at y=%d and the essence row at y=%d", n, bottom, want)
		}
	}

	if top := proseTop(gs, len(lines)); top <= buildBandBottom(gs) {
		t.Errorf("%d narrated lines start at y=%d, inside a build band ending at y=%d",
			len(lines), top, buildBandBottom(gs))
	}
	if bottom := essenceRowBottom(gs); bottom > offerRowOf(gs, handSize).Min.Y-offerSelectedNudge {
		t.Errorf("the payout's band ends at y=%d and a lifted offer card reaches y=%d",
			bottom, offerRowOf(gs, handSize).Min.Y-offerSelectedNudge)
	}
}

// **proseLineHeight has to be the font's own** *(2026-09-18)*, because the payout's block is laid
// out from its last line's bottom edge and that edge is what the essences beside it are aligned to.
// A pitch is the distance between two lines and says nothing about where the last one ends, so the
// figure cannot be derived from proseLineGap — it is measured, and this is what stops it drifting
// when the type size moves.
//
// **It also checks the widest line fits the column**, which is the other half of a payout that has
// been given a third of the width: the wording is authored, so a longer sentence is a layout
// change. It uses the mathbox's font state for the reason that one exists — parsing bytes creates
// no `ebiten.Image` and the package already links Ebitengine.
func TestThePayoutsTypeFitsItsColumn(t *testing.T) {
	gs := mathTestState(t)
	gs.Run = session.New(session.StartingDeck())
	gs.Run.WonFight(3, 40)

	face := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: proseTextSize}

	_, h := text.Measure("Ay", face, 0)
	if got := int(h); got != proseLineHeight {
		t.Errorf("kubasta at %d sets a %d-pixel line and proseLineHeight is %d",
			proseTextSize, got, proseLineHeight)
	}

	mid, split := proseColumnMid(gs), gs.PctX(payoutColumnPct)
	for _, line := range payoutLines(gs) {
		w, _ := text.Measure(line.plain(), face, 0)
		if left, right := mid-int(w)/2, mid+int(w)/2; left < 0 || right > split {
			t.Errorf("%q runs %d..%d, outside a column of 0..%d", line.plain(), left, right, split)
		}
	}
}
