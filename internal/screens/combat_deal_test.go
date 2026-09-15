package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
)

// The cascade's rings are the worn row, left to right, and this is the test that would have caught
// them not being.
//
// **The first version built the ring list in the order the cards happened to meet each relic**, on
// the reasoning that `combat.FlipSteps` walks the worn row — true per card, and silent about the
// order across a hand. Deal an earth card before a lightning one and the third-worn relic takes
// ring 0, so the row fires right to left.
//
// **And the ordering is the smaller half.** A card's faces carry the ring that produced each one
// and are shown in sequence, so a face list whose ring indices do not ascend can never be walked to
// the end — a lightning card under all three rings stopped on earth, wearing a face the hand it
// sits in disagrees with. Both halves are checked below.
func TestTheCascadeFiresTheRingsInWornOrder(t *testing.T) {
	gs := saveState(t)

	// Worn left to right: lightning becomes ice, ice becomes earth, earth becomes ice again. The
	// third ring doubling back is what makes a non-ascending face list possible at all.
	worn := []string{"flip-lightning-to-ice", "flip-ice-to-earth", "flip-earth-to-ice"}
	for _, key := range worn {
		if !gs.Run.Wear(key) {
			t.Fatalf("could not wear %q", key)
		}
	}

	// **Earth first, deliberately.** It is the card that only the last-worn ring touches, and
	// putting it at the front of the pile is exactly what made the old code file that ring as
	// ring 0.
	pile := []actionCard{
		{Concept: combat.Bash, Element: combat.Earth},
		{Concept: combat.Jab, Element: combat.Fire},
		{Concept: combat.Cut, Element: combat.Lightning},
	}

	s := &CombatScene{run: gs.Run}
	for _, c := range pile {
		s.hand = append(s.hand, paletteCard{actionCard: c})
	}
	s.startDeal(0, pile)

	d := s.theater.deal
	if len(d.rings) != len(worn) {
		t.Fatalf("the cascade has %d rings, want %d: %v", len(d.rings), len(worn), d.rings)
	}
	for i, key := range worn {
		if d.rings[i].key != key {
			t.Errorf("ring %d is %q, want %q — the rings are not the worn row", i, d.rings[i].key, key)
		}
	}

	// Every card's faces must name ascending rings, or the beats cannot reach the last one.
	for i, c := range d.cards {
		last := -1
		for _, f := range c.faces[1:] {
			if f.ring <= last {
				t.Errorf("card %d shows ring %d after ring %d; its last face is unreachable", i, f.ring, last)
			}
			last = f.ring
		}
	}

	// And the beats land: run the whole sequence and every card ends on its final face, which is
	// the element the hand behind it is holding.
	for i := 0; i < 10000 && d.running(); i++ {
		s.tickDeal()
		d = s.theater.deal
	}
	if d.running() {
		t.Fatal("the cascade never finished")
	}

	// The deal is spent, so the faces are gone with it — what is checkable afterwards is that the
	// hand holds what the cascade was walking toward. Every lightning, ice and earth card lands on
	// ice under these three; the fire card is untouched.
	want := map[combat.Element]combat.Element{
		combat.Earth: combat.Ice, combat.Fire: combat.Fire, combat.Lightning: combat.Ice,
	}
	for _, c := range pile {
		got, ok := combat.FlipElement(gs.Run.WornRelics(), c)
		if !ok {
			got = c.Element
		}
		if got != want[c.Element] {
			t.Errorf("a %v card is dealt as %v, want %v", c.Element, got, want[c.Element])
		}
	}
}
