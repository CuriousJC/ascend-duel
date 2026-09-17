package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
)

// **The one upgrade test that stayed with the screens.** What it checks is the *sentence* a rune's
// rider prints, and the wording lives beside the screen that shows it; everything else about a
// rider's picture went to internal/ui with the card face.

// **Every rider has a sentence.** runeRiderLine had one arm and a "does nothing" default, which
// was a lie about seven of the eight kinds that existed at the time.
func TestEveryRiderKindHasALine(t *testing.T) {
	for _, k := range combat.RiderKinds() {
		if line := runeRiderLine(k); line == "does nothing" {
			t.Errorf("rider %s falls through to the default line", k)
		}
	}
}
