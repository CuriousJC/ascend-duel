package screens

import (
	"image"
	"testing"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// The build band's relic row, which needs no window — nothing here creates an ebiten.Image.

// bandState is a run wearing one relic, with a record for it, at the game’s internal resolution.
func bandState(t *testing.T) *state.GlobalState {
	t.Helper()

	gs := &state.GlobalState{
		ScreenWidth: state.ScreenWidth, ScreenHeight: state.ScreenHeight,
		Run:    session.New(session.StartingDeck()),
		Relics: map[string]data.RelicData{},
	}

	// **A run opens bare**, so the fixture puts a relic on rather than skipping — a skipped test
	// says nothing, and this one is standing in for a tooltip that went missing without failing
	// anything. `Wear` refuses a key the catalogue does not hold, so a rename fails the test
	// rather than quietly emptying it.
	for _, key := range session.Relics() {
		if gs.Run.Wear(key) {
			gs.Relics[key] = data.RelicData{Name: "Test Relic", Text: "does a thing"}
			break
		}
	}
	if len(gs.Run.Worn()) == 0 {
		t.Fatal("the fixture could not put a single relic on")
	}
	return gs
}

// **A relic in the band explains itself, on every screen that draws one.** The reward screen drew
// the row and hovered nothing for a day, so the row a player reads their build off was silent on
// the screen where they choose what to do to that build. One helper now, and this is what says it
// still answers.
func TestHoveringAWornRelicExplainsIt(t *testing.T) {
	gs := bandState(t)

	worn := gs.Run.Worn()
	seat := relicSlotRect(buildRelicRect(gs), 0, len(worn))
	at := image.Pt((seat.Min.X+seat.Max.X)/2, (seat.Min.Y+seat.Max.Y)/2)

	var tip models.Tooltip
	if !hoverBuildRelics(gs, at, &tip) {
		t.Fatalf("the cursor at %v found no relic, and seat 0 is %v", at, seat)
	}
	if !tip.Pointed() {
		t.Error("a relic was found and the tooltip was not pointed at it")
	}
}

// The cursor away from the row finds nothing, so a tooltip cannot answer for a relic the pointer is
// not on.
func TestHoveringOffTheRowFindsNothing(t *testing.T) {
	gs := bandState(t)

	var tip models.Tooltip
	if hoverBuildRelics(gs, image.Pt(gs.PctX(50), gs.PctY(90)), &tip) {
		t.Error("a cursor at the bottom of the screen found a relic in the band")
	}
}

// **The seat that is drawn is the seat that is hit-tested.** relicSlotAt answers with the point a
// card is drawn from and relicSlotRect turns it into an area; deriving that area a second time
// anywhere is the drawn-here-clicked-there bug this shape exists to prevent.
func TestTheRelicSeatIsDrawnWhereItIsClicked(t *testing.T) {
	gs := bandState(t)
	row := buildRelicRect(gs)

	for n := 1; n <= 5; n++ {
		for i := 0; i < n; i++ {
			if got, want := relicSlotRect(row, i, n).Min, relicSlotAt(row, i, n); got != want {
				t.Errorf("row of %d, seat %d: rect starts at %v, card is drawn at %v",
					n, i, got, want)
			}
		}
	}
}

// **The row is centred and grows outwards, rather than pinned to both edges.** A run wearing two
// relics on a band with no enemy card to end it put one beside the duelist card and the other in
// the far corner; the pitch is capped now, so the slack sits at the two ends of the row.
func TestTheRelicRowIsCentredAndGrowsOutwards(t *testing.T) {
	gs := bandState(t)
	row := buildRelicRect(gs)

	var last int
	for n := 1; n <= combat.DefaultRelicSlots; n++ {
		left := relicSlotAt(row, 0, n).X
		right := relicSlotRect(row, n-1, n).Max.X

		if l, r := left-row.Min.X, row.Max.X-right; l-r > 1 || r-l > 1 {
			t.Errorf("a row of %d leaves %dpx on the left and %dpx on the right", n, l, r)
		}
		if width := right - left; n > 1 && width <= last {
			t.Errorf("a row of %d is %dpx wide, no wider than the %dpx row of %d",
				n, width, last, n-1)
		}
		last = right - left
	}
}

// The pitch never leaves more than relicSlotMaxGap of bare table between two relics, which is the
// cap that makes the row above grow rather than spread.
func TestTwoRelicsSitBesideEachOtherRatherThanApart(t *testing.T) {
	row := buildRelicRect(bandState(t))

	for n := 2; n <= maxRelics; n++ {
		if gap := relicSlotPitch(row, n) - cards.RelicStyle.Width; gap > relicSlotMaxGap {
			t.Errorf("a row of %d leaves %dpx between relics, past the %dpx cap",
				n, gap, relicSlotMaxGap)
		}
	}
}

// **A full row sits inside the pane and centred on it.**
//
// **It used to say the row filled the pane exactly** *(until 2026-09-04)*, because at 1280 wide it
// did — the cap was read off the gap five relics left in this pane. At 1920 the pane is wider than
// five capped relics need, so the row centres in it with slack at both ends, and asserting a flush
// left edge would be asserting that the cap must be re-derived from whatever pane it is handed.
// That is the behaviour relicSlotMaxGap exists to prevent; see the note on it.
//
// What is still worth holding is that the row is centred and that five of them fit, which is the
// pair the cap and the centring are between them responsible for.
func TestAFullRowStillFillsTheCombatPane(t *testing.T) {
	gs := testState()
	s := &CombatScene{}
	pane := s.relicPaneRect(gs)

	slots := combat.DefaultRelicSlots
	first := relicSlotAt(pane, 0, slots).X
	last := relicSlotRect(pane, slots-1, slots).Max.X

	if first < pane.Min.X {
		t.Errorf("the first of five relics sits at x=%d, left of the pane's edge x=%d", first, pane.Min.X)
	}
	if last > pane.Max.X {
		t.Errorf("the last of five relics ends at x=%d, past the pane's x=%d", last, pane.Max.X)
	}
	if before, after := first-pane.Min.X, pane.Max.X-last; before != after {
		t.Errorf("the row is not centred: %dpx before it and %dpx after", before, after)
	}
}

// **The widest row the array allows still lands inside the pane**, which is the weaker half of the
// pair above and the half that has to hold past the shipped cap.
//
// The two are split because they are different claims *(2026-09-11)*. Exact centring and strictly
// outward growth are properties of the row a player can actually reach — five — and they come apart
// by a pixel further up, where `relicSlotPitch` divides the pane by one more seat and the remainder
// has nowhere to go. What must hold at any width is that nothing is drawn off the end of the pane,
// because `combat.MaxWornRelics` is how many relics a fixture may put on and a row drawn past the
// table would be the screen lying about what the duelist is wearing.
func TestTheWidestPossibleRelicRowStaysInsideThePane(t *testing.T) {
	gs := testState()
	s := &CombatScene{}
	pane := s.relicPaneRect(gs)

	for n := combat.DefaultRelicSlots; n <= maxRelics; n++ {
		first := relicSlotAt(pane, 0, n).X
		last := relicSlotRect(pane, n-1, n).Max.X

		if first < pane.Min.X {
			t.Errorf("a row of %d starts at x=%d, left of the pane's x=%d", n, first, pane.Min.X)
		}
		if last > pane.Max.X {
			t.Errorf("a row of %d ends at x=%d, past the pane's x=%d", n, last, pane.Max.X)
		}
		if before, after := first-pane.Min.X, pane.Max.X-last; before-after > 1 || after-before > 1 {
			t.Errorf("a row of %d is off centre: %dpx before it and %dpx after", n, before, after)
		}
	}
}

// TestTheBandEndsBelowBothOfItsHalves. Everything on the two between-fight screens is placed under
// the band, so this figure being short by ten pixels is a worn relic drawn through a line of type.
// The relic row is dropped below the duelist card's top and is the same height, so the row is what
// ends last — reading the card alone is the mistake this pins.
func TestTheBandEndsBelowBothOfItsHalves(t *testing.T) {
	gs := testState()

	bottom := buildBandBottom(gs)
	if card := buildCardRect(gs).Max.Y; bottom < card {
		t.Errorf("the band ends at %d and the duelist card at %d", bottom, card)
	}
	if relics := buildRelicRect(gs).Max.Y; bottom < relics {
		t.Errorf("the band ends at %d and the relic row at %d", bottom, relics)
	}
}

// TestNothingUnderTheBandIsDrawnInsideIt. Both between-fight screens write type below the build
// band, and both of them did it in absolute pixels until 2026-09-05 — figures that were right when
// the screen was 1280x960 and the cards were a quarter smaller. The reward screen's narration ended
// up struck through by a worn relic, and nothing failed.
func TestNothingUnderTheBandIsDrawnInsideIt(t *testing.T) {
	gs := testState()
	bottom := buildBandBottom(gs)

	for _, c := range []struct {
		what string
		top  int
	}{
		{"the reward screen's title", offerTitleTop(gs)},
		{"the reward screen's narration", proseTop(gs)},
		{"the reward screen's hint", offerHintTop(gs)},
		{"the shop's narration", shopProseTop},
	} {
		if c.top <= bottom {
			t.Errorf("%s starts at %d, inside a band that ends at %d", c.what, c.top, bottom)
		}
	}
}
