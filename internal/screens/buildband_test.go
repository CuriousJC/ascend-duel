package screens

import (
	"image"
	"testing"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// The build band's ring row, which needs no window — nothing here creates an ebiten.Image.

// bandState is a run wearing one ring, with a record for it, on a 1280x960 screen.
func bandState(t *testing.T) *state.GlobalState {
	t.Helper()

	gs := &state.GlobalState{
		ScreenWidth: 1280, ScreenHeight: 960,
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
