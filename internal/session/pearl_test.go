package session

import (
	"math/rand"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/seeds"
)

// The Eternity Pearl, and the weightless consumables it makes.

const pearl = "eternity-pearl"

// TestThePearlCopiesOneCarriedThingWeightlessOncePerFight. A copy goes in weightless, so the pane
// is no fuller than it was; a second call in the same fight makes nothing, and the next fight makes
// another.
func TestThePearlCopiesOneCarriedThingWeightlessOncePerFight(t *testing.T) {
	run := runWith(combat.Plain(combat.Bash))
	if !run.Wear(pearl) {
		t.Fatal("the run would not wear the pearl")
	}
	filler := anyWithRider(t, combat.RiderHealOnPlay).Record
	for i := 0; i < MaxConsumables; i++ {
		if !run.Hold(filler) {
			t.Fatalf("the sack refused rune %d", i+1)
		}
	}

	rng := rand.New(rand.NewSource(1))
	made := run.CopyAtFightStart(rng)
	if len(made) != 1 || run.HoldCount() != MaxConsumables+1 {
		t.Fatalf("the pearl made %d copies and the sack holds %d, want one copy over %d",
			len(made), run.HoldCount(), MaxConsumables)
	}
	if run.WeightedConsumables() != MaxConsumables {
		t.Errorf("the copy weighs something: %d weighted", run.WeightedConsumables())
	}
	held := run.Consumables()
	if !held[len(held)-1].Weightless {
		t.Errorf("the copy is not marked weightless")
	}

	if again := run.CopyAtFightStart(rng); len(again) != 0 {
		t.Errorf("the pearl copied twice in one fight")
	}
	run.WonFight(10, 10)
	if next := run.CopyAtFightStart(rng); len(next) != 1 {
		t.Errorf("the pearl made %d copies in the next fight, want 1", len(next))
	}
}

// TestAWeightlessCopyLeavesRoomToBuy. A full pane with a weightless copy in it is full; spending
// the copy does not open a seat, and spending a weighted one does.
func TestAWeightlessCopyLeavesRoomToBuy(t *testing.T) {
	run := runWith(combat.Plain(combat.Bash))
	run.Wear(pearl)
	filler := anyWithRider(t, combat.RiderHealOnPlay).Record
	run.Hold(filler)
	run.CopyAtFightStart(rand.New(rand.NewSource(1)))

	if run.ConsumablesFull() || run.WeightedConsumables() != 1 {
		t.Fatalf("one rune and its copy weigh %d", run.WeightedConsumables())
	}
	if !run.Drop(1) || run.WeightedConsumables() != 1 {
		t.Errorf("dropping the copy changed what weighs: %d", run.WeightedConsumables())
	}
}

// TestNothingCarriedIsNothingCopied. An empty pane takes no roll and makes no copy.
func TestNothingCarriedIsNothingCopied(t *testing.T) {
	run := runWith(combat.Plain(combat.Bash))
	run.Wear(pearl)
	if made := run.CopyAtFightStart(rand.New(rand.NewSource(1))); len(made) != 0 {
		t.Errorf("an empty pane copied %d things", len(made))
	}
}

// TestWeightlessCopiesSurviveAResume. Which entries weigh nothing, and which fight the pearl last
// fired in, both come back — or a resumed fight would copy a second time.
func TestWeightlessCopiesSurviveAResume(t *testing.T) {
	motifs, shape := rosters(t)
	seed, err := seeds.Parse(theSeed)
	if err != nil {
		t.Fatal(err)
	}
	s := Start(motifs, shape, seed)
	s.Wear(pearl)
	filler := anyWithRider(t, combat.RiderHealOnPlay).Record
	s.Hold(filler)
	s.Hold(filler)
	s.CopyAtFightStart(rand.New(rand.NewSource(1)))

	back, _, err := Resume(motifs, shape, s.Snapshot(seed))
	if err != nil {
		t.Fatalf("a snapshot this build wrote must resume: %v", err)
	}
	want, got := s.Consumables(), back.Consumables()
	if len(got) != len(want) {
		t.Fatalf("resumed carrying %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Weightless != want[i].Weightless {
			t.Errorf("entry %d came back weightless=%v, want %v", i, got[i].Weightless, want[i].Weightless)
		}
	}
	if made := back.CopyAtFightStart(rand.New(rand.NewSource(1))); len(made) != 0 {
		t.Errorf("a resumed run copied again in the fight the pearl already fired in")
	}
}
