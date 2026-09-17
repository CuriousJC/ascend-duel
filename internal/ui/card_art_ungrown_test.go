package ui

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
)

// **What a card's face and its tooltip may say about a worn relic.** It sits here rather than with
// the relic row because ungrown is this package's rule about building a spec, and the row is only
// the first caller that needed it.

// **The card tooltip explains a relic at its record, never at its accumulator** *(owner's call,
// 2026-08-26)*. `ungrown` is the one place that is enforced. The face carries no relic at all — see
// TestNoRelicReachesWhatTheFaceSays.
func TestTheTooltipDropsTheAccumulator(t *testing.T) {
	id, ok := combat.RelicByKey("growth-fire")
	if !ok {
		t.Fatal("growth-fire is in no registry")
	}

	worn := ungrown([]combat.WornRelic{{Relic: id, Grown: 50}})
	if len(worn) != 1 {
		t.Fatalf("ungrown returned %d relics, want 1", len(worn))
	}
	if worn[0].Grown != 0 {
		t.Errorf("the tooltip still carries %d of growth", worn[0].Grown)
	}
	if worn[0].Relic != id {
		t.Error("ungrown changed which relic is worn")
	}
}
