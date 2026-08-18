package combat

import "testing"

// handsFormed returns the hands one side formed. A turn forms at most one, so this is a list only
// so that "none" and "one" are the same shape.
func handsFormed(events []Event, by Side) []HandID {
	var out []HandID
	for _, e := range events {
		if e.Kind != KindCombo || e.Side != by || e.Hand == HandNone {
			continue
		}
		// **The High Card is skipped, because it is not a hand anybody built** *(2026-08-15)*.
		// It is in the catalogue so the feed can name the commonest result in the game, and
		// every assertion below is about what a turn's cards amounted to *beyond* the best of
		// them. `Blow.Formed()` draws exactly the same line for the screen's combo preview.
		if h, ok := HandByID(e.Hand); ok && h.Cards() < 2 {
			continue
		}
		out = append(out, e.Hand)
	}
	return out
}

// comboEventFor is the KindCombo event one side's attack phase raised.
func comboEventFor(events []Event, by Side) (Event, bool) {
	for _, e := range events {
		if e.Kind == KindCombo && e.Side == by {
			return e, true
		}
	}
	return Event{}, false
}

// damageDealtBy totals the damage one side landed on the other.
func damageDealtBy(events []Event, by Side) int {
	total := 0
	for _, e := range events {
		if e.Kind == KindDamage && e.Side == by {
			total += e.Amount
		}
	}
	return total
}

// handByKey finds a catalogue entry by its key rather than by the name it prints, so a rung
// renamed does not silently stop being asserted about.
func handByKey(key string) (Hand, bool) {
	id, ok := HandIDForKey(key)
	if !ok {
		return Hand{}, false
	}
	return HandByID(id)
}

// chilledActions returns the actions one side lost to a chill.
func chilledActions(events []Event, by Side) []ConceptID {
	var out []ConceptID
	for _, e := range events {
		if e.Kind == KindChilled && e.Side == by {
			out = append(out, e.Action)
		}
	}
	return out
}

// sideActions returns the actions one side actually took.
func sideActions(events []Event, by Side) []ConceptID {
	var out []ConceptID
	for _, e := range events {
		if e.Kind == KindAction && e.Side == by {
			out = append(out, e.Action)
		}
	}
	return out
}

// comboCards is the cards one event says formed its hand.
func comboCards(e Event) []int { return e.ComboCards[:e.ComboCardCount] }

// --- one blow per turn --------------------------------------------------------------------

// **The rule the whole model rests on.** However many attack cards a turn queues, exactly one
// figure of damage lands.
func TestATurnDealsDamageExactlyOnce(t *testing.T) {
	a, b := duelist(10, 4, 5000), duelist(10, 4, 5000)

	for _, turn := range [][]Card{
		PlainCards(Strike),
		PlainCards(Strike, Jab),
		PlainCards(Strike, Strike),
		PlainCards(Strike, Strike, Strike),
		PlainCards(Jab, Jab, Jab, Jab, Jab),
	} {
		events, _, _ := resolve(a, b, turn, nil, 1)
		if n := kindCount(events, KindDamage); n != 1 {
			t.Errorf("%v dealt damage %d times, want exactly 1", Concepts(turn), n)
		}
	}
}

// Every attack card is still announced, even the ones that contribute nothing — the screen counts
// one beat per slot to know how far through the round playback is.
func TestEveryAttackCardIsAnnouncedEvenOutsideTheHand(t *testing.T) {
	a, b := duelist(10, 4, 5000), duelist(10, 4, 5000)

	events, _, _ := resolve(a, b, PlainCards(Strike, Jab, Strike), nil, 1)

	if took := sideActions(events, SideA); len(took) != 3 {
		t.Fatalf("three cards were played and %d were announced: %v", len(took), took)
	}
}

// **A card that builds to no hand contributes nothing to the blow.** `Strike, Jab, Strike` is a
// Strike Pair and the Jab is not in it.
func TestACardOutsideTheHandAddsNothing(t *testing.T) {
	a, b := duelist(10, 4, 5000), duelist(10, 4, 5000)

	withJab, _, _ := resolve(a, b, PlainCards(Strike, Jab, Strike), nil, 1)
	without, _, _ := resolve(a, b, PlainCards(Strike, Strike), nil, 1)

	if got, want := damageDealtBy(withJab, SideA), damageDealtBy(without, SideA); got != want {
		t.Errorf("the Jab added %d damage; it is in no hand and should add nothing", got-want)
	}

	e, ok := comboEventFor(withJab, SideA)
	if !ok {
		t.Fatal("no attack phase event")
	}
	if got := comboCards(e); len(got) != 2 {
		t.Errorf("the pair says it was formed from %v, want the two Strikes only", got)
	}
}

// --- the damage formula -------------------------------------------------------------------

// **damage = the hand's own cards, plus DMG times the multiplier.** Two Strikes at DMG 10 are 20
// of cards and a 1.5x pair on a DMG of 10, so 35.
func TestDamageIsTheHandsCardsPlusDMGTimesTheMultiplier(t *testing.T) {
	a, b := duelist(10, 4, 5000), duelist(10, 4, 5000)

	pair, ok := handByKey("pair")
	if !ok {
		t.Fatal("the catalogue has no pair")
	}

	events, _, _ := resolve(a, b, PlainCards(Strike, Strike), nil, 1)

	want := Plain(Strike).Damage(10)*2 + Plain(Strike).Damage(10)*pair.Multiplier/multiplierScale
	if got := damageDealtBy(events, SideA); got != want {
		t.Errorf("a plain Strike Pair dealt %d, want %d", got, want)
	}
}

// **The hand is the whole multiplier** *(2026-08-17)*. A second axis counted the distinct colours
// in the formed hand and added its own multiplier on top, so a coloured pair paid more than a plain
// one. Colour buys statuses now and nothing else, and a pair of any two colours is worth exactly
// what the catalogue says a pair is worth.
func TestTheMultiplierIsTheHandsAlone(t *testing.T) {
	a, b := duelist(10, 4, 5000), duelist(10, 4, 5000)

	pair, _ := handByKey("pair")

	for _, tc := range []struct {
		what string
		turn []Card
	}{
		{"two basics", PlainCards(Strike, Strike)},
		{"one colour", []Card{Of(Strike, Ice), Of(Strike, Ice)}},
		{"two colours", []Card{Of(Strike, Fire), Of(Strike, Ice)}},
	} {
		events, _, _ := resolve(a, b, tc.turn, nil, 1)

		e, ok := comboEventFor(events, SideA)
		if !ok {
			t.Fatalf("%s: no attack phase event", tc.what)
		}
		if e.Multiplier != pair.Multiplier {
			t.Errorf("a pair of %s reports x%d, want the pair's own x%d",
				tc.what, e.Multiplier, pair.Multiplier)
		}
	}
}

// --- which hand forms ---------------------------------------------------------------------

// **The best-paying hand wins.** Four Strikes hold a pair and a flurry as well as a barrage; only
// the barrage pays. A fifth Strike changes nothing, since a group matches at least its size.
func TestTheBestPayingHandIsTheOneThatForms(t *testing.T) {
	a, b := duelist(10, 4, 10000), duelist(10, 4, 10000)

	for _, tc := range []struct {
		n    int
		want string
	}{
		{2, "pair"},
		{3, "three-of-a-kind"},
		{4, "four-of-a-kind"},
		{5, "four-of-a-kind"},
	} {
		turn := make([]Card, tc.n)
		for i := range turn {
			turn[i] = Plain(Strike)
		}

		want, ok := handByKey(tc.want)
		if !ok {
			t.Fatalf("the catalogue has no %q", tc.want)
		}
		events, _, _ := resolve(a, b, turn, nil, 1)
		if got := handsFormed(events, SideA); len(got) != 1 || got[0] != want.ID {
			t.Errorf("%d Strikes formed %v, want %s alone", tc.n, got, tc.want)
		}
	}
}

// Two Pair and Full House need two different concepts, so five of one card can be neither.
func TestTheTwoConceptHands(t *testing.T) {
	a, b := duelist(10, 4, 10000), duelist(10, 4, 10000)

	twoPair, _ := handByKey("two-pair")
	fullHouse, _ := handByKey("full-house")
	barrage, _ := handByKey("four-of-a-kind")

	for _, tc := range []struct {
		turn []Card
		want HandID
		what string
	}{
		{PlainCards(Jab, Jab, Strike, Strike), twoPair.ID, "two pairs"},
		{PlainCards(Jab, Jab, Jab, Strike, Strike), fullHouse.ID, "three and two"},
		{PlainCards(Jab, Jab, Jab, Jab, Jab), barrage.ID, "five of one card"},
	} {
		events, _, _ := resolve(a, b, tc.turn, nil, 1)
		if got := handsFormed(events, SideA); len(got) != 1 || got[0] != tc.want {
			t.Errorf("%s formed %v, want one hand", tc.what, got)
		}
	}
}

// **A counted hand does not care what sits between its cards.** Three Strikes with a Jab among
// them is a Flurry; the run-matcher this replaced needed them adjacent and formed nothing.
func TestAHandIgnoresWhatSitsBetweenItsCards(t *testing.T) {
	a, b := duelist(10, 4, 5000), duelist(10, 4, 5000)

	flurry, _ := handByKey("three-of-a-kind")
	events, _, _ := resolve(a, b, PlainCards(Strike, Jab, Strike, Strike), nil, 1)

	if got := handsFormed(events, SideA); len(got) != 1 || got[0] != flurry.ID {
		t.Fatalf("three Strikes around a Jab formed %v, want a flurry", got)
	}
}

// The ladder is scoped to attacks, so three Prepares are three of a card and form nothing.
func TestTheLadderCountsAttacksAlone(t *testing.T) {
	a, b := duelist(10, 4, 5000), duelist(10, 4, 5000)

	events, _, _ := resolve(a, b, PlainCards(Prepare, Prepare, Prepare), nil, 1)

	if got := handsFormed(events, SideA); len(got) != 0 {
		t.Fatalf("three plans formed %v, want nothing", got)
	}
	if n := kindCount(events, KindDamage); n != 0 {
		t.Fatalf("a turn of plans dealt damage %d times, want 0", n)
	}
}

// **With no hand, the biggest single attack is the blow** — the High Card — and it earns no
// multiplier at all, so what lands is exactly what the card's face says.
func TestWithNoHandTheBiggestAttackIsTheBlow(t *testing.T) {
	a, b := duelist(10, 4, 5000), duelist(10, 4, 5000)

	events, _, _ := resolve(a, b, PlainCards(Jab, Smash, Strike), nil, 1)

	if got := handsFormed(events, SideA); len(got) != 0 {
		t.Fatalf("three different attacks formed %v, want no built hand", got)
	}
	if got, want := damageDealtBy(events, SideA), Plain(Smash).Damage(10); got != want {
		t.Errorf("dealt %d, want the Smash's %d and nothing else", got, want)
	}
}

// The High Card is still *named*, which is what lets the feed say what happened on the turn that
// happens most often. A blow the engine could not name is the one failure this model can have.
func TestTheHighCardIsNamedAndPaysNoMultiplier(t *testing.T) {
	a, b := duelist(10, 4, 5000), duelist(10, 4, 5000)

	events, _, _ := resolve(a, b, PlainCards(Jab, Smash, Strike), nil, 1)

	e, ok := comboEventFor(events, SideA)
	if !ok {
		t.Fatal("no KindCombo event — the attack phase said nothing about what it formed")
	}
	high, ok := handByKey("high-card")
	if !ok {
		t.Fatal("the catalogue holds no High Card")
	}
	if e.Hand != high.ID {
		t.Errorf("three different attacks were named %v, want the High Card", e.Hand)
	}
	if e.Multiplier != 0 {
		t.Errorf("the High Card paid a x%d.%02d multiplier, want none",
			e.Multiplier/100, e.Multiplier%100)
	}
	if e.Amount != Plain(Smash).Damage(10) {
		t.Errorf("the High Card came to %d, want the Smash's own %d", e.Amount, Plain(Smash).Damage(10))
	}
}

// --- the hand's colours ---------------------------------------------------------------------

// **The colours in the formed hand are what land, and basic is not one.** This used to be counted
// into a "mix" that paid its own multiplier; what survives is the list, which decides the statuses
// and nothing else.
func TestTheHandsColoursDecideWhichStatusesLand(t *testing.T) {
	for _, tc := range []struct {
		what string
		turn []Card
		want []Element
	}{
		{"two basics", PlainCards(Strike, Strike), nil},
		{"a basic and an ice", []Card{Plain(Strike), Of(Strike, Ice)}, []Element{Ice}},
		{"two ice", []Card{Of(Strike, Ice), Of(Strike, Ice)}, []Element{Ice}},
		{"ice and fire", []Card{Of(Strike, Ice), Of(Strike, Fire)}, []Element{Ice, Fire}},
		{"ice, fire and a basic", []Card{Of(Strike, Ice), Of(Strike, Fire), Plain(Strike)},
			[]Element{Ice, Fire}},
		{"four colours", []Card{
			Of(Strike, Ice), Of(Strike, Fire), Of(Strike, Earth), Of(Strike, Lightning),
		}, []Element{Ice, Fire, Earth, Lightning}},
	} {
		a, b := ringed(duelist(10, 4, 10000)), duelist(10, 4, 10000)
		_, _, bAfter := resolve(a, b, tc.turn, nil, 1)

		for _, e := range AllElements {
			if e == Basic {
				continue
			}
			wanted := false
			for _, w := range tc.want {
				if w == e {
					wanted = true
				}
			}
			if got := bAfter.Statuses[statusOf(e)].Active(); got != wanted {
				t.Errorf("%s: %v active is %v, want %v", tc.what, e, got, wanted)
			}
		}
	}
}

// **Only the cards in the hand carry colour.** An off-colour card that contributed to no hand
// cannot put its status on anybody.
func TestACardOutsideTheHandDoesNotColourIt(t *testing.T) {
	a, b := ringed(duelist(10, 4, 5000)), duelist(10, 4, 5000)

	_, _, bAfter := resolve(a, b, []Card{Of(Strike, Ice), Of(Jab, Fire), Of(Strike, Ice)}, nil, 1)

	if !bAfter.Statuses[statusOf(Ice)].Active() {
		t.Error("the ice pair is the hand and should have chilled")
	}
	if bAfter.Statuses[statusOf(Fire)].Active() {
		t.Error("the fire Jab is in no hand, so it should have burned nobody")
	}
}

// **One status per colour in the hand**, so one colour lands one and four land four — for a
// duelist wearing all four rings, which is what a status needs since 2026-08-16.
func TestEveryColourInTheHandLandsItsStatus(t *testing.T) {
	a, b := ringed(duelist(10, 4, 10000)), duelist(10, 4, 10000)

	events, _, bAfter := resolve(a, b, []Card{
		Of(Strike, Fire), Of(Strike, Ice), Of(Strike, Earth), Of(Strike, Lightning),
	}, nil, 1)

	if n := kindCount(events, KindStatus); n != 4 {
		t.Errorf("a rainbow landed %d statuses, want 4", n)
	}
	for _, e := range []Element{Fire, Ice, Earth, Lightning} {
		if !bAfter.Statuses[statusOf(e)].Active() {
			t.Errorf("%v did not land", e)
		}
	}
}

func TestAColourlessHandLandsNoStatus(t *testing.T) {
	a, b := duelist(10, 4, 5000), duelist(10, 4, 5000)

	events, _, _ := resolve(a, b, PlainCards(Strike, Strike), nil, 1)

	if n := kindCount(events, KindStatus); n != 0 {
		t.Errorf("a colourless pair landed %d statuses, want 0 — basic is not a colour", n)
	}
}

// A lone attack that formed no hand still applies its own element, which is the rule that
// predates hands and was deliberately kept.
func TestALoneAttackStillAppliesItsElement(t *testing.T) {
	a, b := ringed(duelist(10, 4, 5000)), duelist(10, 4, 5000)

	events, _, bAfter := resolve(a, b, []Card{Of(Strike, Ice), Plain(Jab)}, nil, 1)

	if got := handsFormed(events, SideA); len(got) != 0 {
		t.Fatalf("a Strike and a Jab formed %v, want no hand", got)
	}
	if !bAfter.Statuses[statusOf(Ice)].Active() {
		t.Error("the ice Strike was the blow and should still have chilled")
	}
}

// --- a chilled turn -------------------------------------------------------------------------

// **A hand is scored off what survives the chill, not off the queue.** Scoring the queue would let
// a chilled duelist swing with a turn it never took. Four Strikes survive out of five, so a four of
// a kind forms rather than whatever five would have been.
func TestChilledCardsCannotFormAHand(t *testing.T) {
	a := wearing(duelist(10, 4, 20000), Ice)
	b := duelist(10, 4, 20000)

	// A's ice Jab chills B before B's own turn is read.
	events, _, _ := resolve(a, b,
		[]Card{Of(Jab, Ice)},
		PlainCards(Strike, Strike, Strike, Strike, Strike), 1)

	if lost := chilledActions(events, SideB); len(lost) != chillPct() {
		t.Fatalf("B should lose %d card to the chill, got %v", chillPct(), lost)
	}

	fourOfAKind, _ := handByKey("four-of-a-kind")
	if got := handsFormed(events, SideB); len(got) != 1 || got[0] != fourOfAKind.ID {
		t.Fatalf("four surviving Strikes should form a four of a kind, got %v", got)
	}
}

// Side B acts last, so ice B lands finds A has already acted, and bites in the round after. That is
// the one asymmetry phases impose, and the status is what carries it across the boundary.
func TestIceLandedByBBitesInTheFollowingRound(t *testing.T) {
	a, b := duelist(10, 4, 20000), wearing(duelist(10, 4, 20000), Ice)

	r1, a1, b1 := resolve(a, b, PlainCards(Prepare), []Card{Of(Strike, Ice)}, 1)

	if lost := chilledActions(r1, SideA); len(lost) != 0 {
		t.Fatalf("A already acted, so nothing can be taken from it this round, got %v", lost)
	}
	if !a1.Statuses[statusOf(Ice)].Active() {
		t.Fatal("A should be carrying the chill into the next round")
	}

	r2, _, _ := resolve(a1, b1, PlainCards(Strike, Strike), nil, 2)
	if lost := chilledActions(r2, SideA); len(lost) != chillPct() {
		t.Fatalf("A should lose %d card in the round after, got %v", chillPct(), lost)
	}
}

// --- the event ----------------------------------------------------------------------------

// **A counted hand is not contiguous**, which is the case a start-and-length bracket could not
// describe: Two Pair is two cards, a card that earned nothing, and two more.
func TestTheEventNamesScatteredCards(t *testing.T) {
	a, b := duelist(10, 4, 20000), duelist(10, 4, 20000)

	events, _, _ := resolve(a, b, PlainCards(Jab, Jab, Smash, Strike, Strike), nil, 1)

	e, ok := comboEventFor(events, SideA)
	if !ok {
		t.Fatal("no attack phase event")
	}
	got := comboCards(e)
	if len(got) != 4 || got[0] != 0 || got[1] != 1 || got[2] != 3 || got[3] != 4 {
		t.Fatalf("two pair says it was formed from %v, want [0 1 3 4] around the Smash", got)
	}
}

// The attack phase is announced before the blow lands, so a boosted figure never arrives before
// the reason for it.
func TestTheHandIsAnnouncedBeforeTheDamage(t *testing.T) {
	a, b := duelist(10, 4, 5000), duelist(10, 4, 5000)

	events, _, _ := resolve(a, b, PlainCards(Strike, Strike), nil, 1)

	comboAt, damageAt := -1, -1
	for i, e := range events {
		if e.Kind == KindCombo && comboAt < 0 {
			comboAt = i
		}
		if e.Kind == KindDamage && damageAt < 0 {
			damageAt = i
		}
	}
	if comboAt < 0 || damageAt < 0 {
		t.Fatal("expected both a combo and a damage event")
	}
	if comboAt > damageAt {
		t.Error("the damage landed before the hand that explains it was announced")
	}
}

// --- housekeeping -------------------------------------------------------------------------

func TestARoundWithNoRandomnessIsDeterministic(t *testing.T) {
	a, b := duelist(10, 4, 20000), duelist(10, 4, 20000)
	turn := PlainCards(Strike, Strike, Strike)

	e1, a1, b1 := resolve(a, b, turn, PlainCards(Jab, Strike), 1)
	e2, a2, b2 := resolve(a, b, turn, PlainCards(Jab, Strike), 1)

	if a1 != a2 || b1 != b2 {
		t.Fatal("the same round resolved twice must end in the same state")
	}
	if len(e1) != len(e2) {
		t.Fatalf("event logs differ in length: %d vs %d", len(e1), len(e2))
	}
	for i := range e1 {
		if e1[i] != e2[i] {
			t.Fatalf("event %d differs: %+v vs %+v", i, e1[i], e2[i])
		}
	}
}

// TestEverySlotIsEitherTakenOrChilled is the invariant the screen's highlight rests on:
// CombatScene.currentSlot counts one beat per slot, taken or lost, and would light the wrong card
// for the rest of the round if a slot went unaccounted for.
func TestEverySlotIsEitherTakenOrChilled(t *testing.T) {
	a, b := wearing(duelist(10, 4, 20000), Ice), duelist(10, 4, 20000)
	aPlan := []Card{Of(Strike, Ice), Of(Strike, Ice), Of(Strike, Ice)}
	bPlan := PlainCards(Prepare, Jab, Strike, Defend)

	events, _, _ := resolve(a, b, aPlan, bPlan, 1)
	order := ResolutionOrder(aPlan, bPlan)

	var beats []ConceptID
	for _, e := range events {
		if e.Kind == KindAction || e.Kind == KindChilled {
			beats = append(beats, e.Action)
		}
	}

	if len(beats) != len(order) {
		t.Fatalf("every slot needs exactly one beat: %d slots, %d beats", len(order), len(beats))
	}
	for i, slot := range order {
		if beats[i] != slot.Card.Concept {
			t.Fatalf("beat %d is %v, but slot %d is %v", i, beats[i], i, slot.Card.Concept)
		}
	}

	if lost := chilledActions(events, SideB); len(lost) == 0 {
		t.Fatal("this fixture is meant to chill side B")
	}
}
