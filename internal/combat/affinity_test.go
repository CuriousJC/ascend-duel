package combat

import "testing"

// A creature's own element: a hit of it fizzles, and a shield of it banks an action point when it
// eats one. See fizzles and blockedByShield.

// creature is a solo attacker dealt as one element.
func creature(e Element, dmg, actions, life int) Duelist {
	d := duelist(dmg, actions, life)
	d.SoloAttacks = true
	d.Element = e
	return d
}

// TestAHitOfTheTargetsOwnElementLandsNothing. An ice card thrown at an ice goblin is a wasted
// attack; the fire card beside it lands as it always did.
func TestAHitOfTheTargetsOwnElementLandsNothing(t *testing.T) {
	goblin := creature(Ice, 10, 6, 5000)
	events, _, after := resolve(duelist(10, 8, 5000), goblin, []Card{Of(Bash, Ice), Of(Bash, Fire)}, nil, 1)

	hits := hitEvents(events, SideA)
	if len(hits) != 2 {
		t.Fatalf("threw %d hits, want two", len(hits))
	}
	if hits[0].Kind != KindFizzled || hits[0].Element != Ice {
		t.Errorf("the ice Bash came to %v, want a KindFizzled", hits[0].Kind)
	}
	if hits[1].Kind != KindDamage {
		t.Errorf("the fire Bash came to %v, want it to land", hits[1].Kind)
	}
	if lost := goblin.CurrentLife - after.CurrentLife; lost != hits[1].Amount {
		t.Errorf("the goblin lost %d, want only the fire hit's %d", lost, hits[1].Amount)
	}
}

// TestAFizzledCardStillFormsTheHand. The hand is read off the turn before a hit is thrown, so two
// ice Bashes at an ice goblin are still a pair — they just land nothing.
func TestAFizzledCardStillFormsTheHand(t *testing.T) {
	two := []Card{Of(Bash, Ice), Of(Bash, Ice)}
	plain, _, _ := resolve(duelist(10, 8, 5000), duelist(10, 6, 5000), two, nil, 1)
	fizzled, _, after := resolve(duelist(10, 8, 5000), creature(Ice, 10, 6, 5000), two, nil, 1)

	if handOf(t, plain).Hand != handOf(t, fizzled).Hand {
		t.Errorf("the fizzled turn named a different hand from the plain one")
	}
	if after.CurrentLife != 5000 {
		t.Errorf("the goblin lost %d life to two fizzled hits", 5000-after.CurrentLife)
	}
}

// TestAFizzledHitDoesNothingElse. No status, no drain: the whole hit is wasted, not just its
// figure.
func TestAFizzledHitDoesNothingElse(t *testing.T) {
	burning := firstStatusOf(t, EffectDamageOverTime)
	lit := relic(t, "affinity-kindling", RelicRule{
		When: MomentAttackLands,
		If:   RelicCondition{Element: Fire, HasElement: true},
		Then: []RelicEffect{{Do: DoApplyStatus, Status: burning}},
	})
	a := duelist(10, 8, 5000).Wearing(WornRelic{Relic: lit})

	events, _, _ := resolve(a, creature(Fire, 10, 6, 5000), []Card{Of(Bash, Fire)}, nil, 1)
	if n := kindCount(events, KindStatus); n != 0 {
		t.Errorf("a fizzled fire hit landed %d statuses", n)
	}
}

// TestAWildcardNeverFizzles. It counts as every element, so a fizzle on its own would waste it
// against every creature in the game.
func TestAWildcardNeverFizzles(t *testing.T) {
	events, _, _ := resolve(duelist(10, 8, 5000), creature(Ice, 10, 6, 5000), []Card{wild(Bash, Ice)}, nil, 1)
	if n := kindCount(events, KindFizzled); n != 0 {
		t.Errorf("a wild ice Bash fizzled on an ice creature")
	}
	if damageCount(events) != 1 {
		t.Errorf("a wild ice Bash did not land on an ice creature")
	}
}

// TestTheDuelistHasNoElementToFizzleOn. One-way: a creature's ice hits land on the player.
func TestTheDuelistHasNoElementToFizzleOn(t *testing.T) {
	events, _, _ := resolve(duelist(10, 8, 5000), creature(Ice, 10, 6, 5000), nil, []Card{Of(Bash, Ice)}, 1)
	if n := kindCount(events, KindFizzled); n != 0 {
		t.Errorf("an ice creature's hit fizzled on the duelist")
	}
}

// TestAMatchingShieldEatsItsOwnElementFirst. One ice shield against a light ice hit and a heavy
// fire one takes the ice: matching is the first pass, heaviest the second.
func TestAMatchingShieldEatsItsOwnElementFirst(t *testing.T) {
	actor := Duelist{DMG: 10, SoloAttacks: true}
	target := Duelist{Shields: ShieldStack{Ice: 1}}
	jab, _ := ConceptByKey("Jab")
	skewer, _ := ConceptByKey("Skewer")
	turn := []Slot{{Card: Of(jab, Ice), Index: 0}, {Card: Of(skewer, Fire), Index: 1}}

	eaten := shieldedSlots(actor, target, turn)
	if eaten[0] != Ice || eaten[1] >= 0 {
		t.Errorf("one ice shield against an ice Jab and a fire Skewer ate %v, want the Jab", eaten)
	}
}

// TestLeftoverShieldsEatTheHeaviestOfWhatIsLeft. Two shields, one matching: the ice shield takes
// the ice hit, and the fire shield takes the heaviest of the rest whatever its element.
func TestLeftoverShieldsEatTheHeaviestOfWhatIsLeft(t *testing.T) {
	actor := Duelist{DMG: 10, SoloAttacks: true}
	target := Duelist{Shields: ShieldStack{Ice: 1, Fire: 1}}
	jab, _ := ConceptByKey("Jab")
	skewer, _ := ConceptByKey("Skewer")
	turn := []Slot{
		{Card: Of(jab, Ice), Index: 0},
		{Card: Of(jab, Earth), Index: 1},
		{Card: Of(skewer, Earth), Index: 2},
	}

	eaten := shieldedSlots(actor, target, turn)
	if eaten[0] != Ice || eaten[1] >= 0 || eaten[2] != Fire {
		t.Errorf("ice and fire shields against ice Jab, earth Jab, earth Skewer ate %v", eaten)
	}
}

// TestAMatchedBlockBanksAnActionPointForTheNextTurnOnly. Three ice shields eat three of an ice
// creature's hits and buy three points; the turn after spends them and they are gone.
func TestAMatchedBlockBanksAnActionPointForTheNextTurnOnly(t *testing.T) {
	a := duelist(10, 6, 5000)
	goblin := creature(Ice, 10, 6, 5000)

	events, a1, _ := resolve(a, goblin, []Card{Of(Guard, Ice)}, []Card{Of(Bash, Ice), Of(Bash, Ice), Of(Bash, Ice)}, 1)
	surged := 0
	for _, e := range events {
		if e.Kind == KindBlocked && e.Surged {
			surged++
		}
	}
	if surged != 3 {
		t.Errorf("three ice shields blocked %d ice hits with a surge, want 3", surged)
	}
	if a1.Surge != 3 || a1.ActionPoints() != a.Actions+3 {
		t.Errorf("after three matched blocks the duelist has %d AP (surge %d), want %d",
			a1.ActionPoints(), a1.Surge, a.Actions+3)
	}

	_, a2, _ := resolve(a1, goblin, nil, nil, 2)
	if a2.Surge != 0 || a2.ActionPoints() != a.Actions {
		t.Errorf("the surge outlived the turn it bought: %d AP after the next round", a2.ActionPoints())
	}
}

// TestAnUnmatchedBlockBanksNothing. A fire shield still eats an ice hit whole; it just buys no AP.
func TestAnUnmatchedBlockBanksNothing(t *testing.T) {
	_, a1, _ := resolve(duelist(10, 6, 5000), creature(Ice, 10, 6, 5000),
		[]Card{Of(Brace, Fire)}, []Card{Of(Bash, Ice)}, 1)
	if a1.Surge != 0 {
		t.Errorf("a fire shield eating an ice hit banked %d AP", a1.Surge)
	}
	if a1.CurrentLife != 5000 {
		t.Errorf("the fire shield did not eat the ice hit")
	}
}

// handOf is the one hand event a round's log holds.
func handOf(t *testing.T, events []Event) Event {
	t.Helper()
	for _, e := range events {
		if e.Kind == KindHand {
			return e
		}
	}
	t.Fatal("no hand event in the log")
	return Event{}
}
