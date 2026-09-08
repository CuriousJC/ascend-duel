package combat

import "testing"

// The shield picks the heaviest blow in a creature's turn, whatever order the creature queued it
// in — see shieldedSlots, which holds the argument. These pin the two halves of that: which card
// is chosen, and that the choice survives the queue being rearranged.

// soloTurn builds a creature's turn out of concept keys, in the order given.
func soloTurn(t *testing.T, keys ...string) []Slot {
	t.Helper()
	turn := make([]Slot, len(keys))
	for i, key := range keys {
		id, ok := ConceptByKey(key)
		if !ok {
			t.Fatalf("no concept %q", key)
		}
		turn[i] = Slot{Card: Of(id, Basic), Index: i}
	}
	return turn
}

// eatenKeys names the cards a mask picked, for a failure message that says what happened rather
// than which indices did.
func eatenKeys(turn []Slot, eaten []bool) []string {
	var out []string
	for i, on := range eaten {
		if on {
			out = append(out, ConceptOf(turn[i].Card.Concept).Key)
		}
	}
	return out
}

// TestOneShieldEatsTheHeaviestBlow. The bug this rule replaced: a creature leading with its
// cheapest card spent the player's shield on it and then landed its worst.
func TestOneShieldEatsTheHeaviestBlow(t *testing.T) {
	actor := Duelist{DMG: 10, SoloAttacks: true}
	target := Duelist{Shields: 1}

	// Jab is the cheapest attack the player's own catalogue holds and Lunge the dearest, which is
	// what makes them the two ends to test between. The creature leads with the small one.
	turn := soloTurn(t, "Jab", "Lunge", "Strike")

	eaten := shieldedSlots(actor, target, turn)
	got := eatenKeys(turn, eaten)
	if len(got) != 1 || got[0] != "Lunge" {
		t.Errorf("one shield against Jab, Lunge, Strike ate %v; it should eat the Lunge, which is "+
			"the heaviest of the three", got)
	}
}

// TestTheQueueOrderDoesNotDecideWhatAShieldEats. The whole point of the rule: a shield is worth
// the same against a given set of cards however the creature arranged them.
func TestTheQueueOrderDoesNotDecideWhatAShieldEats(t *testing.T) {
	actor := Duelist{DMG: 10, SoloAttacks: true}
	target := Duelist{Shields: 1}

	first := shieldedSlots(actor, target, soloTurn(t, "Lunge", "Jab", "Strike"))
	last := shieldedSlots(actor, target, soloTurn(t, "Jab", "Strike", "Lunge"))

	if eatenKeys(soloTurn(t, "Lunge", "Jab", "Strike"), first)[0] != "Lunge" ||
		eatenKeys(soloTurn(t, "Jab", "Strike", "Lunge"), last)[0] != "Lunge" {
		t.Error("the same three cards in two orders lost two different cards to one shield")
	}
}

// TestShieldsWorkDownTheOrder. Two shields take the two heaviest, and the survivor is the smallest
// card rather than whichever one happened to be queued last.
func TestShieldsWorkDownTheOrder(t *testing.T) {
	actor := Duelist{DMG: 10, SoloAttacks: true}
	target := Duelist{Shields: 2}

	turn := soloTurn(t, "Jab", "Lunge", "Strike")
	got := eatenKeys(turn, shieldedSlots(actor, target, turn))

	if len(got) != 2 {
		t.Fatalf("two shields ate %d attacks: %v", len(got), got)
	}
	for _, key := range got {
		if key == "Jab" {
			t.Errorf("two shields ate %v, and the Jab is the smallest of the three: the shields "+
				"should be working down from the heaviest", got)
		}
	}
}

// TestATieGoesToTheEarliestCard. Ties have to break somewhere, and the mask has to be a function
// of the turn alone — see shieldedSlots.
func TestATieGoesToTheEarliestCard(t *testing.T) {
	actor := Duelist{DMG: 10, SoloAttacks: true}
	target := Duelist{Shields: 1}

	turn := soloTurn(t, "Strike", "Strike", "Strike")
	eaten := shieldedSlots(actor, target, turn)

	if !eaten[0] {
		t.Errorf("three identical blows against one shield lost %v; the tie breaks on the earliest",
			eatenKeys(turn, eaten))
	}
}

// TestShieldsNeverEatADefence. A creature's plans are not blows, and a shield spent on one would
// be a shield spent on nothing.
func TestShieldsNeverEatADefence(t *testing.T) {
	actor := Duelist{DMG: 10, SoloAttacks: true}
	target := Duelist{Shields: 5}

	turn := soloTurn(t, "Ward", "Jab", "Brace")
	got := eatenKeys(turn, shieldedSlots(actor, target, turn))

	if len(got) != 1 || got[0] != "Jab" {
		t.Errorf("five shields against one attack and two defences ate %v; only the attack is "+
			"something a shield can eat", got)
	}
}

// TestNoShieldsEatNothing, which is every duelist in the game except the player.
func TestNoShieldsEatNothing(t *testing.T) {
	turn := soloTurn(t, "Jab", "Lunge")
	for _, on := range shieldedSlots(Duelist{DMG: 10}, Duelist{}, turn) {
		if on {
			t.Fatal("a duelist holding no shields ate an attack")
		}
	}
}
