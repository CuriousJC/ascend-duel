package screens

import (
	"image"
	"testing"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// The build band's ring row, which needs no window — nothing here creates an ebiten.Image.

// bandState is a run wearing one ring, with a record for it, at the game’s internal resolution.
func bandState(t *testing.T) *state.GlobalState {
	t.Helper()

	gs := &state.GlobalState{
		ScreenWidth: state.ScreenWidth, ScreenHeight: state.ScreenHeight,
		Run:   session.New(session.StartingDeck()),
		Rings: map[string]data.RingData{},
	}

	// **A run opens bare**, so the fixture puts a ring on rather than skipping — a skipped test
	// says nothing, and this one is standing in for a tooltip that went missing without failing
	// anything. `Wear` refuses a key the catalogue does not hold, so a rename fails the test
	// rather than quietly emptying it.
	for _, key := range session.Rings() {
		if gs.Run.Wear(key) {
			gs.Rings[key] = data.RingData{Name: "Test Ring", Text: "does a thing"}
			break
		}
	}
	if len(gs.Run.Worn()) == 0 {
		t.Fatal("the fixture could not put a single ring on")
	}
	return gs
}

// **A ring in the band explains itself, on every screen that draws one.** The reward screen drew
// the row and hovered nothing for a day, so the row a player reads their build off was silent on
// the screen where they choose what to do to that build. One helper now, and this is what says it
// still answers.
func TestHoveringAWornRingExplainsIt(t *testing.T) {
	gs := bandState(t)

	worn := gs.Run.Worn()
	seat := ringSlotRect(buildRingRect(gs), 0, len(worn))
	at := image.Pt((seat.Min.X+seat.Max.X)/2, (seat.Min.Y+seat.Max.Y)/2)

	var tip models.Tooltip
	if !hoverBuildRings(gs, at, &tip) {
		t.Fatalf("the cursor at %v found no ring, and seat 0 is %v", at, seat)
	}
	if !tip.Pointed() {
		t.Error("a ring was found and the tooltip was not pointed at it")
	}
}

// The cursor away from the row finds nothing, so a tooltip cannot answer for a ring the pointer is
// not on.
func TestHoveringOffTheRowFindsNothing(t *testing.T) {
	gs := bandState(t)

	var tip models.Tooltip
	if hoverBuildRings(gs, image.Pt(gs.PctX(50), gs.PctY(90)), &tip) {
		t.Error("a cursor at the bottom of the screen found a ring in the band")
	}
}

// **The seat that is drawn is the seat that is hit-tested.** ringSlotAt answers with the point a
// card is drawn from and ringSlotRect turns it into an area; deriving that area a second time
// anywhere is the drawn-here-clicked-there bug this shape exists to prevent.
func TestTheRingSeatIsDrawnWhereItIsClicked(t *testing.T) {
	gs := bandState(t)
	row := buildRingRect(gs)

	for n := 1; n <= 5; n++ {
		for i := 0; i < n; i++ {
			if got, want := ringSlotRect(row, i, n).Min, ringSlotAt(row, i, n); got != want {
				t.Errorf("row of %d, seat %d: rect starts at %v, card is drawn at %v",
					n, i, got, want)
			}
		}
	}
}

// **The row is centred and grows outwards, rather than pinned to both edges.** A run wearing two
// rings on a band with no enemy card to end it put one beside the duelist card and the other in
// the far corner; the pitch is capped now, so the slack sits at the two ends of the row.
func TestTheRingRowIsCentredAndGrowsOutwards(t *testing.T) {
	gs := bandState(t)
	row := buildRingRect(gs)

	var last int
	for n := 1; n <= maxRings; n++ {
		left := ringSlotAt(row, 0, n).X
		right := ringSlotRect(row, n-1, n).Max.X

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

// The pitch never leaves more than ringSlotMaxGap of bare table between two rings, which is the
// cap that makes the row above grow rather than spread.
func TestTwoRingsSitBesideEachOtherRatherThanApart(t *testing.T) {
	row := buildRingRect(bandState(t))

	for n := 2; n <= maxRings; n++ {
		if gap := ringSlotPitch(row, n) - cards.RingStyle.Width; gap > ringSlotMaxGap {
			t.Errorf("a row of %d leaves %dpx between rings, past the %dpx cap",
				n, gap, ringSlotMaxGap)
		}
	}
}

// **A full row sits inside the pane and centred on it.**
//
// **It used to say the row filled the pane exactly** *(until 2026-09-04)*, because at 1280 wide it
// did — the cap was read off the gap five rings left in this pane. At 1920 the pane is wider than
// five capped rings need, so the row centres in it with slack at both ends, and asserting a flush
// left edge would be asserting that the cap must be re-derived from whatever pane it is handed.
// That is the behaviour ringSlotMaxGap exists to prevent; see the note on it.
//
// What is still worth holding is that the row is centred and that five of them fit, which is the
// pair the cap and the centring are between them responsible for.
func TestAFullRowStillFillsTheCombatPane(t *testing.T) {
	gs := testState()
	s := &CombatScene{}
	pane := s.ringPaneRect(gs)

	first := ringSlotAt(pane, 0, maxRings).X
	last := ringSlotRect(pane, maxRings-1, maxRings).Max.X

	if first < pane.Min.X {
		t.Errorf("the first of five rings sits at x=%d, left of the pane's edge x=%d", first, pane.Min.X)
	}
	if last > pane.Max.X {
		t.Errorf("the last of five rings ends at x=%d, past the pane's x=%d", last, pane.Max.X)
	}
	if before, after := first-pane.Min.X, pane.Max.X-last; before != after {
		t.Errorf("the row is not centred: %dpx before it and %dpx after", before, after)
	}
}
