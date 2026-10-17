package combat

import (
	"math/rand"
	"testing"
)

// The traits: a card landing twice, a card not counted against the cap, and a wearing's own rules —
// a random shield every turn.

// testTrait registers a relic trait for this package's tests, or hands back the one already
// registered under the key: the registry is the process's, and a test may run twice.
func testTrait(t *testing.T, key string, slot, lastsFight bool, rules []RelicRule) RelicTrait {
	t.Helper()
	if tr, ok := ParseRelicTrait(key); ok {
		return tr
	}
	tr, err := RegisterRelicTrait(key, key, slot, lastsFight, rules)
	if err != nil {
		t.Fatal(err)
	}
	return tr
}

// cardTrait is a card trait out of data/card_traits.json, by key.
func cardTrait(t *testing.T, key string) CardTrait {
	t.Helper()
	tr, ok := ParseCardTrait(key)
	if !ok {
		t.Fatalf("card_traits.json has no %q", key)
	}
	return tr
}

// TestEveryCardTraitNameRoundTrips. A snapshot writes the keys, so each has to come back as itself.
func TestEveryCardTraitNameRoundTrips(t *testing.T) {
	for _, tr := range CardTraits() {
		if got, ok := ParseCardTrait(tr.String()); !ok || got != tr {
			t.Errorf("card trait %q came back as %v, %v", tr, got, ok)
		}
	}
}

// TestATraitThatChangesNothingIsRefused. Counted and landing once is a plain card wearing a word, and
// a wearing taking a slot, lasting the run and firing nothing is a plain wearing.
func TestATraitThatChangesNothingIsRefused(t *testing.T) {
	if _, err := RegisterCardTrait("test-nothing", "Nothing", "NOTHING", true, 1); err == nil {
		t.Error("a card trait that changes nothing registered")
	}
	if _, err := RegisterRelicTrait("test-nothing", "Nothing", true, false, nil); err == nil {
		t.Error("a relic trait that changes nothing registered")
	}
}

// TestATraitsRulesNeverGrow. A growth verb on a trait would feed an accumulator that belongs to the
// relic it is worn on.
func TestATraitsRulesNeverGrow(t *testing.T) {
	_, err := RegisterRelicTrait("test-growing", "Growing", true, false, []RelicRule{{
		When: MomentFightWon,
		Then: []RelicEffect{{Do: DoGrowOnWin, Amount: 1}},
	}})
	if err == nil {
		t.Error("a relic trait carrying a growth verb registered")
	}
}

// TestAMirroredAttackLandsTwiceAtFullStrength. The same Bash, landing twice, throws two hits of the
// plain one's figure.
func TestAMirroredAttackLandsTwiceAtFullStrength(t *testing.T) {
	a, b := duelist(10, 6, 500), duelist(10, 6, 5000)
	mirrored := cardTrait(t, "mirrored")

	plain, _, _ := resolve(a, b, []Card{Of(Bash, Fire)}, nil, 1)
	twice, _, _ := resolve(a, b, []Card{Of(Bash, Fire).WithTrait(mirrored, true)}, nil, 1)

	once := firstDamage(t, plain, SideA).Amount
	if got := damageCount(twice); got != 2 {
		t.Fatalf("a mirrored Bash landed %d hits, want 2", got)
	}
	for _, e := range twice {
		if e.Kind == KindDamage && e.Amount != once {
			t.Errorf("a mirrored landing hit for %d, want the plain card's %d", e.Amount, once)
		}
	}
}

// TestAMirroredDefenseRaisesItsShieldsTwice.
func TestAMirroredDefenseRaisesItsShieldsTwice(t *testing.T) {
	a, b := duelist(10, 6, 500), duelist(10, 6, 5000)
	mirrored := cardTrait(t, "mirrored")

	_, plain, _ := resolve(a, b, []Card{Of(Block, Ice)}, nil, 1)
	_, twice, _ := resolve(a, b, []Card{Of(Block, Ice).WithTrait(mirrored, true)}, nil, 1)

	if got, want := twice.Shields.Count(), 2*plain.Shields.Count(); got != want || want == 0 {
		t.Errorf("a mirrored Block left %d shields standing, want %d", got, want)
	}
}

// TestAnEffortlessCardIsNotCountedAgainstTheCap. Five counted cards fill a turn; an effortless sixth
// still fits, and a sixth counted card does not.
func TestAnEffortlessCardIsNotCountedAgainstTheCap(t *testing.T) {
	d := duelist(10, 20, 500)
	effortless := cardTrait(t, "effortless")
	five := []Card{Of(Jab, Fire), Of(Jab, Fire), Of(Jab, Ice), Of(Jab, Ice), Of(Jab, Earth)}

	if d.RoomFor(five, Of(Jab, Arcane)) {
		t.Error("a sixth counted card fit under the cap")
	}
	if !d.RoomFor(five, Of(Jab, Arcane).WithTrait(effortless, true)) {
		t.Error("an effortless sixth card was refused")
	}
	if got := CountedActions(append(five, Of(Jab, Arcane).WithTrait(effortless, true))); got != 5 {
		t.Errorf("six cards with one effortless counted as %d, want 5", got)
	}
}

// TestAnEffortlessCardJoinsTheHand. Four fire cards and an effortless fifth fire card past a full
// turn form the five-card rung the counted four could not.
func TestAnEffortlessCardJoinsTheHand(t *testing.T) {
	d := duelist(10, 20, 500)
	effortless := cardTrait(t, "effortless")
	counted := []Card{Of(Jab, Fire), Of(Jab, Fire), Of(Jab, Fire), Of(Jab, Fire), Of(Brace, Ice)}
	sixth := Of(Jab, Fire).WithTrait(effortless, true)

	four := BlowFor(ResolutionOrder(counted, nil))
	five := BlowFor(ResolutionOrder(append(counted, sixth), nil))
	if four.Hand.ID == five.Hand.ID || five.Hand.Cards() <= four.Hand.Cards() {
		t.Errorf("an effortless fifth fire card left the hand at %q (%d cards), from %q (%d)",
			five.Hand.Name, five.Hand.Cards(), four.Hand.Name, four.Hand.Cards())
	}

	// **Every card of the turn throws a hit**, the Brace included, so six cards are six hits.
	events, _, _ := resolve(d, duelist(10, 6, 5000), append(counted, sixth), nil, 1)
	if got := damageCount(events); got != 6 {
		t.Errorf("a six-card turn landed %d hits, want 6", got)
	}
}

// TestATraitsRandomShieldIsRaisedEveryTurn. A trait carrying a random raise-shield raises one shield a
// turn from the relic it is worn on, in one of the five colors, the same one for the same seed — and
// none with no source to roll from.
func TestATraitsRandomShieldIsRaisedEveryTurn(t *testing.T) {
	armored := testTrait(t, "test-armored", true, false, []RelicRule{{
		When: MomentTurnStart,
		Then: []RelicEffect{{Do: DoRaiseShield, RandomElement: true, Amount: 1}},
	}})
	id := relic(t, "armored.plain", RelicRule{
		When: MomentEquipped,
		Then: []RelicEffect{{Do: DoAddDMG, Amount: 1}},
	})
	a := duelist(10, 6, 500).Wearing(WornRelic{Relic: id, Traits: armored})
	b := duelist(10, 6, 5000)

	ward := func(seed int64) []Event {
		events, _, _ := ResolveRound(a, b, nil, nil, 1, Sources{Shield: rand.New(rand.NewSource(seed))})
		var out []Event
		for _, e := range events {
			if e.Kind == KindWarded {
				out = append(out, e)
			}
		}
		return out
	}

	got := ward(7)
	if len(got) != 1 || got[0].Amount != 1 || got[0].Relic != id {
		t.Fatalf("an armored relic warded %+v, want one shield from that relic", got)
	}
	colored := false
	for _, e := range ShieldElements {
		colored = colored || got[0].Element == e
	}
	if !colored {
		t.Errorf("the armored shield was %v, want one of the five colors", got[0].Element)
	}
	if again := ward(7); len(again) != 1 || again[0].Element != got[0].Element {
		t.Error("the same seed rolled a different armored shield")
	}

	if events, _, _ := resolve(a, b, nil, nil, 1); count(events, KindWarded) != 0 {
		t.Error("an armored relic raised a shield with no source to roll from")
	}
}

// TestATraitsRulesDoNotGrowWithTheRelic. A growing relic worn with a trait raising one shield still
// raises one, however far the relic has grown.
func TestATraitsRulesDoNotGrowWithTheRelic(t *testing.T) {
	ward := testTrait(t, "test-ward", true, false, []RelicRule{{
		When: MomentTurnStart,
		Then: []RelicEffect{{Do: DoRaiseShield, Element: Fire, Amount: 1}},
	}})
	id := relic(t, "grown.plain", RelicRule{
		When: MomentEquipped,
		Then: []RelicEffect{{Do: DoAddDMG, Amount: 1}},
	})
	a := duelist(10, 6, 500).Wearing(WornRelic{Relic: id, Grown: 40, Traits: ward})

	events, _, _ := resolve(a, duelist(10, 6, 5000), nil, nil, 1)
	for _, e := range events {
		if e.Kind == KindWarded && e.Amount != 1 {
			t.Errorf("a trait's one shield came up as %d on a relic grown by 40", e.Amount)
		}
	}
}
