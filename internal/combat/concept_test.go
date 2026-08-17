package combat

import (
	"testing"

	"github.com/curiousjc/ascend-duel/data"
)

// mustTestConcept registers a concept a test needs and hands back its ID, reusing it if a previous
// test in the same run already registered that key.
//
// **The registry is package state and additive**, so a test that registers a card leaves it there
// for every test after it. That is deliberately harmless — nothing walks the whole registry to
// decide an outcome, and `PlayerConcepts` counts only the deck list — but it does mean a key has
// to be unique across the package and re-registration has to be tolerated rather than fatal.
func mustTestConcept(key string, c data.CardData) ConceptID {
	if id, ok := ConceptByKey(key); ok {
		return id
	}
	id, err := RegisterConcept("", c)
	if err != nil {
		panic(err)
	}
	return id
}

func TestTheRegistryRefusesACardTheRulesCannotResolve(t *testing.T) {
	// **The check that replaced CheckCostTiers.** That one compared a declared cost against the
	// rules and had nothing left to compare once the file became the rules; what is worth checking
	// now is that a record makes sense at all, because a card the loader shrugged at would be a
	// card that silently did nothing in a duel.
	cases := []struct {
		name string
		card data.CardData
	}{
		{"no label", data.CardData{Verb: "attack", Amount: 100, Cost: 1}},
		{"unknown verb", data.CardData{Label: "Bad1", Verb: "smite", Amount: 100, Cost: 1}},
		{"unknown target", data.CardData{Label: "Bad2", Verb: "attack", Amount: 100, Cost: 1, Target: "everyone"}},
		{"unknown family", data.CardData{Label: "Bad3", Verb: "attack", Amount: 100, Cost: 1, Family: "punch"}},
		{"zero amount", data.CardData{Label: "Bad4", Verb: "attack", Amount: 0, Cost: 1}},
		{"negative cost", data.CardData{Label: "Bad5", Verb: "attack", Amount: 100, Cost: -1}},

		// Nothing reduces a blow to zero — a card that did would delete a whole opposing turn.
		{"total defence", data.CardData{Label: "Bad6", Verb: "defend", Amount: 100, Cost: 3}},

		// Draining and milling are designed and unbuilt. A card asking for one would silently act
		// on its own duelist instead, which is the quiet failure this refuses.
		{"bank at the opponent", data.CardData{Label: "Bad7", Verb: "bank", Amount: 2, Cost: 1, Target: "opponent"}},
		{"draw at the opponent", data.CardData{Label: "Bad8", Verb: "draw", Amount: 2, Cost: 2, Target: "opponent"}},
	}

	for _, c := range cases {
		if _, err := RegisterConcept("", c.card); err == nil {
			t.Errorf("%s: registered without complaint", c.name)
		}
	}
}

func TestAKeyIsScopedToItsOwner(t *testing.T) {
	// Forty creatures will want a card called Bite and they will not all want it at the same cost.
	// The label collides freely; the key must not.
	bite := data.CardData{Label: "Bite", Verb: "attack", Amount: 100, Cost: 2, Copies: 1}

	wolf, err := RegisterConcept("TestWolf", bite)
	if err != nil {
		t.Fatalf("registering TestWolf.Bite: %v", err)
	}
	rat, err := RegisterConcept("TestRat", bite)
	if err != nil {
		t.Fatalf("registering TestRat.Bite: %v", err)
	}

	if wolf == rat {
		t.Fatal("two enemies' Bites share one concept, so a turn holding both would count a pair")
	}
	if ConceptOf(wolf).Label != "Bite" || ConceptOf(rat).Label != "Bite" {
		t.Error("scoping the key changed what the card is called on screen")
	}

	// And the same owner may not say it twice.
	if _, err := RegisterConcept("TestWolf", bite); err == nil {
		t.Error("TestWolf registered two cards called Bite")
	}
}

func TestTheDamageLadderIsTheCardsMultiplier(t *testing.T) {
	// The three rungs used to be a switch statement. They are three numbers in a file now, and the
	// arithmetic is the same for a card sitting between them.
	cases := []struct {
		amount, dmg, want int
	}{
		{50, 10, 5},
		{100, 10, 10},
		{200, 10, 20},
		{150, 10, 15}, // a rung the old ladder could not express
		{50, 1, 1},    // the floor: a cheap card at low DMG still lands something
	}

	for i, c := range cases {
		id := mustTestConcept("Ladder"+string(rune('a'+i)), data.CardData{
			Label: "Ladder" + string(rune('a'+i)), Verb: "attack", Amount: c.amount, Cost: 1,
		})
		if got := Plain(id).Damage(c.dmg); got != c.want {
			t.Errorf("%d%% of DMG %d = %d, want %d", c.amount, c.dmg, got, c.want)
		}
	}
}

func TestAPlanCardDealsNothing(t *testing.T) {
	for _, id := range []ConceptID{Prepare, Plan, Defend} {
		if got := Plain(id).Damage(100); got != 0 {
			t.Errorf("%v deals %d", ConceptOf(id).Label, got)
		}
	}
}
