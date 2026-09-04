package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// The worn ring row as something you can drag: what the badge says, and where a drop lands.
//
// Nothing here creates an ebiten.Image, so it runs without a window like the rest of the band's
// tests. What it cannot check is what the drag *looks* like; that is what launching the game is for.

// wornState is a run wearing every named ring, with a record for each.
func wornState(t *testing.T, keys ...string) *state.GlobalState {
	t.Helper()

	gs := &state.GlobalState{
		ScreenWidth: state.ScreenWidth, ScreenHeight: state.ScreenHeight,
		Run:   session.New(session.StartingDeck()),
		Rings: map[string]data.RingData{},
	}
	for _, key := range keys {
		if !gs.Run.Wear(key) {
			t.Fatalf("%s would not go on", key)
		}
		gs.Rings[key] = data.RingData{RingRecord: key, Name: key}
	}
	return gs
}

// **The badge says what the ring is doing, in the units the ring's own effect is written in.** Heart
// grows flat life and Enflamed grows a multiplier; one figure for both would be wrong for one of
// them, which is the whole reason ringCounter reads combat.Scaling rather than assuming.
func TestTheBadgeSaysWhatTheRingIsDoing(t *testing.T) {
	for _, tc := range []struct {
		key   string
		grown int
		want  string
	}{
		{"heart-ring", 0, "+5"},
		{"heart-ring", 45, "+50"},
		{"enflamed-ring", 0, "1.0x"},
		{"enflamed-ring", 50, "1.5x"},
		{"momentum-ring", 60, "1.6x"},
	} {
		id, ok := combat.RingByKey(tc.key)
		if !ok {
			t.Fatalf("%s is in no registry", tc.key)
		}
		if got := ringCounter(combat.WornRing{Ring: id, Grown: tc.grown}); got != tc.want {
			t.Errorf("%s grown %d reads %q, want %q", tc.key, tc.grown, got, tc.want)
		}
	}
}

// **A ring that does not grow carries no badge**, which is most of the catalogue: an empty string is
// what drawRingCard passes through to a card that then draws nothing.
func TestARingThatDoesNotGrowHasNoBadge(t *testing.T) {
	id, ok := combat.RingByKey("keen-ring")
	if !ok {
		t.Fatal("keen-ring is in no registry")
	}
	if got := ringCounter(combat.WornRing{Ring: id, Grown: 50}); got != "" {
		t.Errorf("keen-ring reads %q, want no badge at all", got)
	}
}

// The badges are keyed by record, because the row is about to be dragged into a different order and
// a badge indexed by seat would follow the finger rather than the ring.
func TestTheBadgesAreKeyedByRecord(t *testing.T) {
	gs := wornState(t, "keen-ring", "heart-ring")

	got := runCounters(gs)
	if got["heart-ring"] == "" {
		t.Error("heart-ring has no badge")
	}
	if _, ok := got["keen-ring"]; ok {
		t.Error("keen-ring has a badge, and it does not grow")
	}
}

// **A drop lands on the seat the cursor is over**, and never past the end of the row: nothing is
// being inserted here, so five rings reordered are still five rings.
func TestARingDropLandsOnTheSeatUnderTheCursor(t *testing.T) {
	gs := wornState(t, "keen-ring", "heart-ring", "banker-ring")
	row := buildRingRow(gs, nil)

	for i := 0; i < row.worn; i++ {
		seat := row.rowSlot(gs, i)
		gs.MouseX, gs.MouseY = (seat.Min.X+seat.Max.X)/2, (seat.Min.Y+seat.Max.Y)/2

		if got := row.rowDropIndex(gs); got != i {
			t.Errorf("the middle of seat %d reads as seat %d", i, got)
		}
	}
}

func TestARingDropIsClampedToTheRow(t *testing.T) {
	gs := wornState(t, "keen-ring", "heart-ring", "banker-ring")
	row := buildRingRow(gs, nil)

	gs.MouseX, gs.MouseY = -400, row.rect.Min.Y
	if got := row.rowDropIndex(gs); got != 0 {
		t.Errorf("far left reads as seat %d, want 0", got)
	}

	gs.MouseX = gs.ScreenWidth * 4
	if got := row.rowDropIndex(gs); got != row.worn-1 {
		t.Errorf("far right reads as seat %d, want %d", got, row.worn-1)
	}
}

// **A drop commits to the run**, which is the half of the move every screen shares. The combat
// screen has a second half — see CombatScene.moveRing — and no test here can reach it without a
// fighter.
func TestDroppingARingReordersTheRun(t *testing.T) {
	gs := wornState(t, "keen-ring", "heart-ring", "banker-ring")
	row := buildRingRow(gs, nil)

	row.rowReturn(2, 0)

	want := []string{"banker-ring", "keen-ring", "heart-ring"}
	got := gs.Run.Worn()
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("the row is %v, want %v", got, want)
		}
	}
}

// A cancelled drag passes the same index twice, and every row has to read that as putting the card
// back untouched.
func TestACancelledRingDragChangesNothing(t *testing.T) {
	gs := wornState(t, "keen-ring", "heart-ring")
	row := buildRingRow(gs, nil)

	before := gs.Run.Worn()
	row.rowReturn(1, 1)

	after := gs.Run.Worn()
	for i := range before {
		if before[i] != after[i] {
			t.Fatalf("the row moved: %v -> %v", before, after)
		}
	}
}

// **The card tooltip explains a ring at its record, never at its accumulator** *(owner's call,
// 2026-08-26)*. `ungrown` is the one place that is enforced. The face carries no ring at all — see
// TestNoRingReachesWhatTheFaceSays.
func TestTheTooltipDropsTheAccumulator(t *testing.T) {
	id, ok := combat.RingByKey("enflamed-ring")
	if !ok {
		t.Fatal("enflamed-ring is in no registry")
	}

	worn := ungrown([]combat.WornRing{{Ring: id, Grown: 50}})
	if len(worn) != 1 {
		t.Fatalf("ungrown returned %d rings, want 1", len(worn))
	}
	if worn[0].Grown != 0 {
		t.Errorf("the tooltip still carries %d of growth", worn[0].Grown)
	}
	if worn[0].Ring != id {
		t.Error("ungrown changed which ring is worn")
	}
}

// **The badge shown mid-blow is the one the sum has reached**, so the player watches the number that
// is about to price the next card go up. withGrown is the copy that makes that possible without
// writing back into the fight's own duelist.
func TestTheBadgeFollowsTheSumMidBlow(t *testing.T) {
	id, ok := combat.RingByKey("enflamed-ring")
	if !ok {
		t.Fatal("enflamed-ring is in no registry")
	}

	worn := []combat.WornRing{{Ring: id, Grown: 0}}
	stepped := withGrown(worn, [combat.MaxWornRings]int{20})

	if worn[0].Grown != 0 {
		t.Error("withGrown wrote back into the fight's own worn set")
	}
	if got := ringCounter(stepped[0]); got != "1.2x" {
		t.Errorf("the badge reads %q part way through the sum, want 1.2x", got)
	}
}
