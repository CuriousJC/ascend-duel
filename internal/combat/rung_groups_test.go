package combat

import "testing"

// **A hand reports its groups**, so a screen can mark a Two Pair as two pairs. Two ice Cuts and a
// fire Brace and Bash are an Elemental Two Pair, and each pair is a group of its own; the arcane
// Block is in neither.
func TestATwoPairReportsItsTwoGroups(t *testing.T) {
	turn := slots([]Card{Of(Cut, Ice), Of(Cut, Ice), Of(Brace, Fire), Of(Bash, Fire), Of(Block, Arcane)})
	blow := BlowFor(turn)

	if len(blow.RungGroups) != 2 {
		t.Fatalf("%s reports %d groups %v, want 2", blow.Hand.Name, len(blow.RungGroups), blow.RungGroups)
	}
	seen := 0
	for _, g := range blow.RungGroups {
		if len(g) != 2 {
			t.Errorf("a group holds %v, want two cards", g)
			continue
		}
		if a, b := turn[g[0]].Card.Element, turn[g[1]].Card.Element; a != b {
			t.Errorf("a group mixes %v and %v", a, b)
		}
		seen += len(g)
	}
	if seen != len(blow.Rung) {
		t.Errorf("the groups hold %d cards and the rung %d", seen, len(blow.Rung))
	}
}
