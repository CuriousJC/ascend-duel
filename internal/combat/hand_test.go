package combat

import (
	"reflect"
	"testing"
)

// handsFormed returns the hands one side formed. A turn forms at most one, so this is a list only
// so that "none" and "one" are the same shape.
func handsFormed(events []Event, by Side) []HandID {
	var out []HandID
	for _, e := range events {
		if e.Kind != KindHand || e.Side != by || e.Hand == HandNone {
			continue
		}
		// **The No Hand is skipped here, because every assertion below is about what a turn's
		// cards amounted to *beyond* the best of them** — a helper for the matcher's tests, not a
		// statement about what the game shows. **The screens draw no such line any more**
		// *(2026-08-19)*: a lone attack is named, previewed and shouted like any other hand, and
		// `Blow.Formed()` went with that change, having been the predicate for the distinction.
		if h, ok := HandByID(e.Hand); ok && h.Cards() < 2 {
			continue
		}
		out = append(out, e.Hand)
	}
	return out
}

// handEventFor is the KindHand event one side's attack phase raised.
func handEventFor(events []Event, by Side) (Event, bool) {
	for _, e := range events {
		if e.Kind == KindHand && e.Side == by {
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

// handByKey finds a catalog entry by its key rather than by the name it prints, so a rung
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

// handCards is the cards one event says formed its hand.
func handCards(e Event) []int { return e.HandCards[:e.HandCardCount] }

// --- a hit per card -----------------------------------------------------------------------

// **The rule the whole model rests on.** However many attack cards a turn queues, each of them
// lands its own figure of damage.
func TestATurnDealsDamageOncePerAttackCard(t *testing.T) {
	a, b := duelist(10, 4, 5000), duelist(10, 4, 5000)

	for _, turn := range [][]Card{
		PlainCards(Bash),
		PlainCards(Bash, Jab),
		PlainCards(Bash, Bash),
		PlainCards(Bash, Bash, Bash),
		PlainCards(Jab, Jab, Jab, Jab, Jab),
	} {
		events, _, _ := resolve(a, b, turn, nil, 1)
		if n := kindCount(events, KindDamage); n != len(turn) {
			t.Errorf("%v dealt damage %d times, want once per card", Concepts(turn), n)
		}
	}
}

// Every attack card is still announced, even the ones that contribute nothing — the screen counts
// one beat per slot to know how far through the round playback is.
func TestEveryAttackCardIsAnnouncedEvenOutsideTheHand(t *testing.T) {
	a, b := duelist(10, 4, 5000), duelist(10, 4, 5000)

	events, _, _ := resolve(a, b, PlainCards(Bash, Jab, Bash), nil, 1)

	if took := sideActions(events, SideA); len(took) != 3 {
		t.Fatalf("three cards were played and %d were announced: %v", len(took), took)
	}
}

// **An attack that builds to no hand still pays into the blow.** `Bash, Jab, Bash` is a Pair the
// Jab does not make, and the Jab's damage lands anyway, at the Pair's multiplier, because action
// points were spent on it.
func TestAnAttackOutsideTheHandStillPaysIntoTheBlow(t *testing.T) {
	a, b := duelist(10, 4, 5000), duelist(10, 4, 5000)

	withJab, _, _ := resolve(a, b, PlainCards(Bash, Jab, Bash), nil, 1)
	without, _, _ := resolve(a, b, PlainCards(Bash, Bash), nil, 1)

	// The Pair is the identity, so what the Jab is worth here is its own face damage.
	want := Plain(Jab).Damage(10)
	if got := damageDealtBy(withJab, SideA) - damageDealtBy(without, SideA); got != want {
		t.Errorf("the Jab added %d damage, want its own %d", got, want)
	}

	e, ok := handEventFor(withJab, SideA)
	if !ok {
		t.Fatal("no attack phase event")
	}
	if got := handCards(e); len(got) != 3 {
		t.Errorf("the blow says it was paid by %v, want all three cards", got)
	}
}

// **The rung is still the cards that made it**, which is the question `RiderScaleInCombo` asks and
// the one the scoring set stopped answering. The two are different sets on the same turn.
func TestTheRungIsNarrowerThanTheScoringSet(t *testing.T) {
	turn := ResolutionOrder(PlainCards(Bash, Jab, Bash), nil)
	blow := BlowFor(turn)

	if len(blow.Cards) != 3 {
		t.Errorf("the scoring set is %v, want all three cards", blow.Cards)
	}
	if len(blow.Rung) != 2 {
		t.Errorf("the rung is %v, want the two Bashes only", blow.Rung)
	}
	for _, i := range blow.Rung {
		if turn[i].Card.Concept != Bash {
			t.Errorf("the rung holds %v, which is not one of the Bashes", turn[i].Card.Concept)
		}
	}
}

// **A defense is in the blow only by making the hand.** It deals nothing either way, so what this
// pins is the membership: an attack joins because it was paid for, a shield does not.
func TestADefenseOutsideTheHandIsNotInTheBlow(t *testing.T) {
	// Resolution order puts the defenses first, so the turn is Brace, Block, Bash, Bash. The ice
	// Brace joins the two ice Bashes for an Elemental Three of a Kind; the fire Block makes
	// nothing and is left out of a scoring set that holds the cards either side of it.
	turn := ResolutionOrder([]Card{
		Of(Brace, Ice), Of(Block, Fire), Of(Bash, Ice), Of(Bash, Ice),
	}, nil)
	blow := BlowFor(turn)

	got := blow.Cards
	if len(got) != 3 || got[0] != 0 || got[1] != 2 || got[2] != 3 {
		t.Fatalf("the blow was paid by %v, want [0 2 3] around the fire Block", got)
	}
}

// **The event carries both sets, and they come apart on exactly this turn.** Two shields make the
// Pair; every card throws a hit, and the attack beside them made no part of the rung.
// The screen raises the rung on the announcement and works the hits out, so an event carrying only
// one of the two would have to guess at the other.
func TestTheEventNamesTheRungApartFromTheBlow(t *testing.T) {
	a, b := duelist(10, 6, 5000), duelist(10, 6, 5000)

	events, _, _ := resolve(a, b, PlainCards(Brace, Block, Bash), nil, 1)
	e, ok := handEventFor(events, SideA)
	if !ok {
		t.Fatal("no attack phase event")
	}

	// Brace, Block, Bash once resolved: the two defenses are the Pair, and every card throws a hit.
	if got := handCards(e); len(got) != 3 {
		t.Errorf("the hits were thrown by %v, want all three cards", got)
	}
	got := e.RungCards[:e.RungCardCount]
	if len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Fatalf("the rung is %v, want the two defenses", got)
	}
}

// **A pair of shields beside one attack lands the attack.** The two defenses form the Pair on
// themselves and deal nothing; the Bash made no part of it and swings anyway, because the action
// points were spent on it. The trap is a blow summed from the rung alone, which is that pair's own
// zero.
func TestAnAttackBesideAPairOfShieldsStillLands(t *testing.T) {
	a, b := duelist(10, 6, 5000), duelist(10, 6, 5000)

	events, _, after := resolve(a, b, PlainCards(Brace, Block, Bash), nil, 1)

	if got := damageDealtBy(events, SideA); got <= 0 {
		t.Fatalf("the Bash dealt %d beside two shields, want its own damage", got)
	}
	if after.CurrentLife >= b.CurrentLife {
		t.Errorf("the target is on %d life, want less than the %d it started with",
			after.CurrentLife, b.CurrentLife)
	}
}

// --- the damage formula -------------------------------------------------------------------

// **damage = the hand's own cards, times the multiplier** *(2026-08-18)*. Two Bashes at DMG 10
// are 20 of cards, and a 1.5x pair takes that to 30. There is no third term: the multiplier used
// to be applied to a separate swing of one 1x attack at the attacker's DMG and added on top, which
// made a hand's percent worth a fixed figure rather than a proportion of the cards that formed it.
func TestDamageIsTheHandsCardsTimesTheMultiplier(t *testing.T) {
	a, b := duelist(10, 4, 5000), duelist(10, 4, 5000)

	pair, ok := handByKey("pair")
	if !ok {
		t.Fatal("the catalog has no pair")
	}

	events, _, _ := resolve(a, b, PlainCards(Bash, Bash), nil, 1)

	want := Plain(Bash).Damage(10) * 2 * pair.Multiplier / multiplierScale
	if got := damageDealtBy(events, SideA); got != want {
		t.Errorf("a plain Bash Pair dealt %d, want %d", got, want)
	}
}

// **A hand is worth a proportion of its own cards, so the same hand pays more on bigger cards.**
// This is the property the swing term did not have — it paid the same figure whatever was played,
// so a Four of a Kind was worth 2.5x the base on the cheapest cards and 0.6x on the dearest.
func TestTheSameHandPaysMoreOnBiggerCards(t *testing.T) {
	a, b := duelist(10, 6, 5000), duelist(10, 6, 5000)

	jabs, _, _ := resolve(a, b, PlainCards(Jab, Jab), nil, 1)
	lunges, _, _ := resolve(a, b, PlainCards(Skewer, Skewer), nil, 1)

	jabPair, lungePair := damageDealtBy(jabs, SideA), damageDealtBy(lunges, SideA)

	// **The multiple is read off the cards, not written down here** *(2026-09-01)*. It was a
	// literal 4 while Jab dealt half DMG and Skewer double; the day the 3 AP cards went to triple
	// it failed, having pinned a tuning decision inside a test about proportionality. What this
	// is here to catch is a term added *outside* the multiplier, which shows up as the pairs
	// being a different multiple apart than the cards are — whatever that multiple currently is.
	//
	// **Within the rounding, and that slack is not slop.** The blow is integer arithmetic, so a
	// multiplier that does not divide the small figure exactly loses up to a point on the Jab
	// pair — and scaling that up multiplies the loss with it. A gap wider than the truncation is
	// the term this test exists to find.
	const dmg = 10
	mult := Plain(Skewer).Damage(dmg) / Plain(Jab).Damage(dmg)
	want := jabPair * mult
	if lungePair < want || lungePair > want+mult {
		t.Errorf("a Skewer Pair dealt %d against a Jab Pair's %d; the cards are %dx apart, so want %d (+ up to %d of rounding)",
			lungePair, jabPair, mult, want, mult)
	}
}

// **Every multiplier the catalog holds is worth what it says against every hit.** The ladder is
// tuned by editing hands.json alone, which is only true while nothing in the resolver adds to the
// figure the file's percent is applied to.
func TestEveryHandIsWorthItsCatalogMultiplier(t *testing.T) {
	a, b := duelist(10, 8, 5000), duelist(10, 8, 5000)

	for _, tc := range []struct {
		key  string
		turn []Card
	}{
		{"no-hand", PlainCards(Bash)},
		{"pair", PlainCards(Bash, Bash)},
		{"concept-three-of-a-kind", PlainCards(Bash, Bash, Bash)},
		{"concept-four-of-a-kind", PlainCards(Bash, Bash, Bash, Bash)},
	} {
		h, ok := handByKey(tc.key)
		if !ok {
			t.Fatalf("the catalog has no %q", tc.key)
		}

		events, _, _ := resolve(a, b, tc.turn, nil, 1)
		e, ok := handEventFor(events, SideA)
		if !ok {
			t.Fatalf("%s: no KindHand event", tc.key)
		}

		card := Plain(Bash).Damage(10)
		if got := cardTerms(e); got != card*len(tc.turn) {
			t.Errorf("%s: the cards' terms came to %d, want the %d cards' own %d",
				tc.key, got, len(tc.turn), card*len(tc.turn))
		}
		// **Each hit is multiplied and rounded on its own.**
		if want := len(tc.turn) * scaleDamage(card, h.Multiplier); e.Amount != want {
			t.Errorf("%s: came to %d, want %d hits of %d x %d%% = %d",
				tc.key, e.Amount, len(tc.turn), card, h.Multiplier, want)
		}
	}
}

// **Every attack played throws a hit, and the hits add up to the hand's figure.** The hand dialog
// works each one out on screen, so a hit whose figure disagreed with the total would be arithmetic
// the player can see is wrong.
func TestEveryAttackThrowsAHitAndTheHitsAddUp(t *testing.T) {
	a, b := duelist(10, 8, 5000), duelist(10, 8, 5000)

	events, _, _ := resolve(a, b, PlainCards(Bash, Jab, Bash, Bash), nil, 1)
	e, ok := handEventFor(events, SideA)
	if !ok {
		t.Fatal("no KindHand event")
	}
	// The Jab makes no trips and throws a hit anyway, so there are four.
	if e.HandCardCount != 4 {
		t.Fatalf("the hand carries %d hits, want the four cards played", e.HandCardCount)
	}

	sum := 0
	for i := 0; i < e.HandCardCount; i++ {
		if e.HitAmounts[i] <= 0 {
			t.Errorf("hit %d carries a figure of %d", i, e.HitAmounts[i])
		}
		sum += e.HitAmounts[i]
	}
	if sum != e.Amount {
		t.Errorf("the hits come to %d between them, but the hand says %d", sum, e.Amount)
	}
	if want := Plain(Bash).Damage(10)*3 + Plain(Jab).Damage(10); cardTerms(e) != want {
		t.Errorf("the cards' terms are %d, want the four cards' own %d", cardTerms(e), want)
	}
}

// **The hand is the whole multiplier** *(2026-08-17)*. A second axis counted the distinct colors
// in the formed hand and added its own multiplier on top, so a colored pair paid more than a plain
// one. Color buys statuses now and nothing else, and a pair of any two colors is worth exactly
// what the catalog says a pair is worth.
func TestTheMultiplierIsTheHandsAlone(t *testing.T) {
	a, b := duelist(10, 4, 5000), duelist(10, 4, 5000)

	pair, _ := handByKey("pair")

	for _, tc := range []struct {
		what string
		turn []Card
	}{
		{"two basics", PlainCards(Bash, Bash)},
		{"one color", []Card{Of(Bash, Ice), Of(Bash, Ice)}},
		{"two colors", []Card{Of(Bash, Fire), Of(Bash, Ice)}},
	} {
		events, _, _ := resolve(a, b, tc.turn, nil, 1)

		e, ok := handEventFor(events, SideA)
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

// **The best-paying hand wins.** Four Bashes hold a pair and a flurry as well as a barrage; only
// the barrage pays.
//
// **A fifth Bash is its own rung** *(2026-08-19)*. It used to change nothing — a group matches at
// least its size, so five of one card was still the four — and the ladder now goes one further on
// every axis. Five copies of a concept could not be dealt from the 48-card deck of the time, which
// shipped four of each; arcane made a concept five cards on 2026-08-25, so the rung is dealable and
// the `duplicate` essence is no longer the only way to it.
func TestTheBestPayingHandIsTheOneThatForms(t *testing.T) {
	a, b := duelist(10, 4, 10000), duelist(10, 4, 10000)

	for _, tc := range []struct {
		n    int
		want string
	}{
		{2, "pair"},
		{3, "concept-three-of-a-kind"},
		{4, "concept-four-of-a-kind"},
		{5, "concept-five-of-a-kind"},
	} {
		turn := make([]Card, tc.n)
		for i := range turn {
			turn[i] = Plain(Bash)
		}

		want, ok := handByKey(tc.want)
		if !ok {
			t.Fatalf("the catalog has no %q", tc.want)
		}
		events, _, _ := resolve(a, b, turn, nil, 1)
		if got := handsFormed(events, SideA); len(got) != 1 || got[0] != want.ID {
			t.Errorf("%d Bashes formed %v, want %s alone", tc.n, got, tc.want)
		}
	}
}

// Two Pair and Full House need two different concepts, so five of one card can be neither.
func TestTheTwoConceptHands(t *testing.T) {
	a, b := duelist(10, 4, 10000), duelist(10, 4, 10000)

	twoPair, _ := handByKey("concept-two-pair")
	fullHouse, _ := handByKey("concept-full-house")
	fiveOfAKind, _ := handByKey("concept-five-of-a-kind")

	for _, tc := range []struct {
		turn []Card
		want HandID
		what string
	}{
		{PlainCards(Jab, Jab, Bash, Bash), twoPair.ID, "two pairs"},
		{PlainCards(Jab, Jab, Jab, Bash, Bash), fullHouse.ID, "three and two"},
		{PlainCards(Jab, Jab, Jab, Jab, Jab), fiveOfAKind.ID, "five of one card"},
	} {
		events, _, _ := resolve(a, b, tc.turn, nil, 1)
		if got := handsFormed(events, SideA); len(got) != 1 || got[0] != tc.want {
			t.Errorf("%s formed %v, want one hand", tc.what, got)
		}
	}
}

// **A counted hand does not care what sits between its cards.** Three Bashes with a Jab among
// them is a Flurry; the run-matcher this replaced needed them adjacent and formed nothing.
func TestAHandIgnoresWhatSitsBetweenItsCards(t *testing.T) {
	a, b := duelist(10, 4, 5000), duelist(10, 4, 5000)

	flurry, _ := handByKey("concept-three-of-a-kind")
	events, _, _ := resolve(a, b, PlainCards(Bash, Jab, Bash, Bash), nil, 1)

	if got := handsFormed(events, SideA); len(got) != 1 || got[0] != flurry.ID {
		t.Fatalf("three Bashes around a Jab formed %v, want a flurry", got)
	}
}

// **A turn of nothing but defenses is a hand, and it lands nothing** *(owner's call)*. Every card
// carries a form and an element and every card is counted, so three Blocks are three of a kind —
// the ladder can see a shield build. Each throws a hit, and every hit is worth nothing: no damage,
// and nothing of the target's is spent.
func TestATurnOfDefensesFormsAHandAndLandsNothing(t *testing.T) {
	a, b := duelist(10, 4, 5000), duelist(10, 4, 5000)

	events, _, after := resolve(a, b, PlainCards(Block, Block, Block), nil, 1)

	if got := handsFormed(events, SideA); len(got) != 1 {
		t.Fatalf("three Blocks formed %v, want one hand", got)
	}
	for _, e := range events {
		if e.Kind == KindDamage && e.Amount != 0 {
			t.Fatalf("a turn of defenses hit for %d, want nothing", e.Amount)
		}
	}
	if after.CurrentLife != b.CurrentLife {
		t.Errorf("the target is on %d life, want the %d it started with",
			after.CurrentLife, b.CurrentLife)
	}
}

// **With no hand, every attack still lands and none of them is multiplied.** The No Hand means *no
// multiplier* rather than *the biggest attack alone*, so what lands is exactly what the three faces
// say, added up.
func TestWithNoHandEveryAttackLandsAtTheIdentity(t *testing.T) {
	a, b := duelist(10, 4, 5000), duelist(10, 4, 5000)

	events, _, _ := resolve(a, b, PlainCards(Jab, Cut, Smash), nil, 1)

	if got := handsFormed(events, SideA); len(got) != 0 {
		t.Fatalf("three different attacks formed %v, want no built hand", got)
	}
	want := Plain(Jab).Damage(10) + Plain(Cut).Damage(10) + Plain(Smash).Damage(10)
	if got := damageDealtBy(events, SideA); got != want {
		t.Errorf("dealt %d, want the three faces' own %d", got, want)
	}
	e, ok := handEventFor(events, SideA)
	if !ok {
		t.Fatal("no attack phase event")
	}
	if cardTerms(e) != want {
		t.Errorf("the cards' terms are %d, want the three cards' %d", cardTerms(e), want)
	}
}

// The No Hand is still *named*, which is what lets the feed say what happened on the turn that
// happens most often. A blow the engine could not name is the one failure this model can have.
func TestTheNoHandIsNamedAndPaysTheIdentityMultiplier(t *testing.T) {
	a, b := duelist(10, 4, 5000), duelist(10, 4, 5000)

	events, _, _ := resolve(a, b, PlainCards(Jab, Cut, Smash), nil, 1)

	e, ok := handEventFor(events, SideA)
	if !ok {
		t.Fatal("no KindHand event — the attack phase said nothing about what it formed")
	}
	high, ok := handByKey("no-hand")
	if !ok {
		t.Fatal("the catalog holds no No Hand")
	}
	if e.Hand != high.ID {
		t.Errorf("three different attacks were named %v, want the No Hand", e.Hand)
	}
	// **The No Hand sits at the identity** *(2026-08-18)*. It was 0 while the multiplier applied
	// to a swing added on top of the cards; now that it multiplies the cards, 0 would be an attack
	// phase that dealt nothing, and 100 is what makes a lone attack land its own face damage.
	if e.Multiplier != multiplierScale {
		t.Errorf("the No Hand paid a x%d.%02d multiplier, want the identity",
			e.Multiplier/100, e.Multiplier%100)
	}
	want := Plain(Jab).Damage(10) + Plain(Cut).Damage(10) + Plain(Smash).Damage(10)
	if e.Amount != want {
		t.Errorf("the No Hand came to %d, want the three cards' own %d", e.Amount, want)
	}
}

// --- the hand's colors ---------------------------------------------------------------------

// **The colors in the formed hand are what land, and basic is not one.** This used to be counted
// into a "mix" that paid its own multiplier; what survives is the list, which decides the statuses
// and nothing else.
func TestTheHandsColorsDecideWhichStatusesLand(t *testing.T) {
	for _, tc := range []struct {
		what string
		turn []Card
		want []Element
	}{
		{"two basics", PlainCards(Bash, Bash), nil},
		{"a basic and an ice", []Card{Plain(Bash), Of(Bash, Ice)}, []Element{Ice}},
		{"two ice", []Card{Of(Bash, Ice), Of(Bash, Ice)}, []Element{Ice}},
		{"ice and fire", []Card{Of(Bash, Ice), Of(Bash, Fire)}, []Element{Ice, Fire}},
		{"ice, fire and a basic", []Card{Of(Bash, Ice), Of(Bash, Fire), Plain(Bash)},
			[]Element{Ice, Fire}},
		{"five colors", []Card{
			Of(Bash, Ice), Of(Bash, Fire), Of(Bash, Earth), Of(Bash, Lightning),
			Of(Bash, Arcane),
		}, []Element{Ice, Fire, Earth, Lightning, Arcane}},
	} {
		a, b := reliced(duelist(10, 4, 10000)), duelist(10, 4, 10000)
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

// **Every card that pays into the blow carries its color.** The fire Jab makes no hand and swings
// anyway, so it burns — a card that visibly hit and left nothing behind would read as a bug rather
// than as a rule.
func TestAnAttackOutsideTheHandStillColorsTheBlow(t *testing.T) {
	a, b := reliced(duelist(10, 4, 5000)), duelist(10, 4, 5000)

	_, _, bAfter := resolve(a, b, []Card{Of(Bash, Ice), Of(Jab, Fire), Of(Bash, Ice)}, nil, 1)

	if !bAfter.Statuses[statusOf(Ice)].Active() {
		t.Error("the ice pair is the hand and should have chilled")
	}
	if !bAfter.Statuses[statusOf(Fire)].Active() {
		t.Error("the fire Jab paid into the blow and should have burned")
	}
}

// **A defense that made no hand still lands its color**: every card throws a hit, and a hit lands
// its card's statuses.
func TestADefenseOutsideTheHandStillLandsItsColor(t *testing.T) {
	a, b := reliced(duelist(10, 6, 5000)), duelist(10, 6, 5000)

	// Brace, Block, Bash, Bash once resolved. The ice Brace makes the elemental trips with the two
	// ice Bashes; the fire Block makes nothing.
	_, _, bAfter := resolve(a, b, []Card{
		Of(Brace, Ice), Of(Block, Fire), Of(Bash, Ice), Of(Bash, Ice),
	}, nil, 1)

	if !bAfter.Statuses[statusOf(Ice)].Active() {
		t.Error("the ice trips are the hand and should have chilled")
	}
	if !bAfter.Statuses[statusOf(Fire)].Active() {
		t.Error("the fire Block threw a hit and should have burned")
	}
}

// **One status per color in the hand**, so one color lands one and four land four — for a
// duelist wearing all four relics, which is what a status needs since 2026-08-16.
func TestEveryColorInTheHandLandsItsStatus(t *testing.T) {
	a, b := reliced(duelist(10, 4, 10000)), duelist(10, 4, 10000)

	events, _, bAfter := resolve(a, b, []Card{
		Of(Bash, Fire), Of(Bash, Ice), Of(Bash, Earth), Of(Bash, Lightning),
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

func TestAColorlessHandLandsNoStatus(t *testing.T) {
	a, b := duelist(10, 4, 5000), duelist(10, 4, 5000)

	events, _, _ := resolve(a, b, PlainCards(Bash, Bash), nil, 1)

	if n := kindCount(events, KindStatus); n != 0 {
		t.Errorf("a colorless pair landed %d statuses, want 0 — basic is not a color", n)
	}
}

// A lone attack that formed no hand still applies its own element, which is the rule that
// predates hands and was deliberately kept.
func TestALoneAttackStillAppliesItsElement(t *testing.T) {
	a, b := reliced(duelist(10, 4, 5000)), duelist(10, 4, 5000)

	events, _, bAfter := resolve(a, b, []Card{Of(Bash, Ice), Plain(Jab)}, nil, 1)

	if got := handsFormed(events, SideA); len(got) != 0 {
		t.Fatalf("a Bash and a Jab formed %v, want no hand", got)
	}
	if !bAfter.Statuses[statusOf(Ice)].Active() {
		t.Error("the ice Bash was the blow and should still have chilled")
	}
}

// --- a chilled turn -------------------------------------------------------------------------

// **A hand is scored off what survives the chill, not off the queue.** Scoring the queue would let
// a chilled duelist swing with a turn it never took. Four Bashes survive out of five, so a four of
// a kind forms rather than whatever five would have been.
func TestChilledCardsCannotFormAHand(t *testing.T) {
	a := wearing(duelist(10, 4, 20000), Ice)
	b := duelist(10, 4, 20000)

	// A's ice Jab chills B before B's own turn is read.
	events, _, _ := resolve(a, b,
		[]Card{Of(Jab, Ice)},
		PlainCards(Bash, Bash, Bash, Bash, Bash), 1)

	if lost := chilledActions(events, SideB); len(lost) != chillPct() {
		t.Fatalf("B should lose %d card to the chill, got %v", chillPct(), lost)
	}

	fourOfAKind, _ := handByKey("concept-four-of-a-kind")
	if got := handsFormed(events, SideB); len(got) != 1 || got[0] != fourOfAKind.ID {
		t.Fatalf("four surviving Bashes should form a four of a kind, got %v", got)
	}
}

// Side B acts last, so ice B lands finds A has already acted, and bites in the round after. That is
// the one asymmetry phases impose, and the status is what carries it across the boundary.
func TestIceLandedByBBitesInTheFollowingRound(t *testing.T) {
	a, b := duelist(10, 4, 20000), wearing(duelist(10, 4, 20000), Ice)

	// **A queues nothing**, deliberately: a shield would eat B's Bash whole and the ice would
	// never land, which is a test about shields rather than about when a chill bites.
	r1, a1, b1 := resolve(a, b, nil, []Card{Of(Bash, Ice)}, 1)

	if lost := chilledActions(r1, SideA); len(lost) != 0 {
		t.Fatalf("A already acted, so nothing can be taken from it this round, got %v", lost)
	}
	if !a1.Statuses[statusOf(Ice)].Active() {
		t.Fatal("A should be carrying the chill into the next round")
	}

	r2, _, _ := resolve(a1, b1, PlainCards(Bash, Bash), nil, 2)
	if lost := chilledActions(r2, SideA); len(lost) != chillPct() {
		t.Fatalf("A should lose %d card in the round after, got %v", chillPct(), lost)
	}
}

// --- the event ----------------------------------------------------------------------------

// **A rung is not contiguous**, which is the case a start-and-length bracket could not describe:
// resolution order puts the defenses first, so a shield that made no hand can sit between two
// cards that did.
func TestTheEventNamesScatteredCards(t *testing.T) {
	a, b := duelist(10, 6, 20000), duelist(10, 6, 20000)

	// Brace, Block, Bash, Bash once resolved: the ice Brace and the two ice Bashes are an
	// Elemental Three of a Kind and the fire Block is in nothing.
	events, _, _ := resolve(a, b, []Card{
		Of(Brace, Ice), Of(Block, Fire), Of(Bash, Ice), Of(Bash, Ice),
	}, nil, 1)

	e, ok := handEventFor(events, SideA)
	if !ok {
		t.Fatal("no attack phase event")
	}
	got := e.RungCards[:e.RungCardCount]
	if len(got) != 3 || got[0] != 0 || got[1] != 2 || got[2] != 3 {
		t.Fatalf("the rung is %v, want [0 2 3] around the fire Block", got)
	}
	// **Every card throws a hit**, the fire Block included.
	if hits := handCards(e); len(hits) != 4 {
		t.Errorf("the hits were thrown by %v, want all four cards", hits)
	}
}

// The attack phase is announced before the blow lands, so a boosted figure never arrives before
// the reason for it.
func TestTheHandIsAnnouncedBeforeTheDamage(t *testing.T) {
	a, b := duelist(10, 4, 5000), duelist(10, 4, 5000)

	events, _, _ := resolve(a, b, PlainCards(Bash, Bash), nil, 1)

	handAt, damageAt := -1, -1
	for i, e := range events {
		if e.Kind == KindHand && handAt < 0 {
			handAt = i
		}
		if e.Kind == KindDamage && damageAt < 0 {
			damageAt = i
		}
	}
	if handAt < 0 || damageAt < 0 {
		t.Fatal("expected both a hand and a damage event")
	}
	if handAt > damageAt {
		t.Error("the damage landed before the hand that explains it was announced")
	}
}

// --- housekeeping -------------------------------------------------------------------------

func TestARoundWithNoRandomnessIsDeterministic(t *testing.T) {
	a, b := duelist(10, 4, 20000), duelist(10, 4, 20000)
	turn := PlainCards(Bash, Bash, Bash)

	e1, a1, b1 := resolve(a, b, turn, PlainCards(Jab, Bash), 1)
	e2, a2, b2 := resolve(a, b, turn, PlainCards(Jab, Bash), 1)

	// DeepEqual: a duelist holds a relic slice now, so it is no longer comparable. See
	// TestRoundIsDeterministic, which says the same thing about the log.
	if !reflect.DeepEqual(a1, a2) || !reflect.DeepEqual(b1, b2) {
		t.Fatal("the same round resolved twice must end in the same state")
	}
	if len(e1) != len(e2) {
		t.Fatalf("event logs differ in length: %d vs %d", len(e1), len(e2))
	}
	for i := range e1 {
		if !reflect.DeepEqual(e1[i], e2[i]) {
			t.Fatalf("event %d differs: %+v vs %+v", i, e1[i], e2[i])
		}
	}
}

// TestEverySlotIsEitherTakenOrChilled is the invariant the screen's highlight rests on:
// CombatScene.currentSlot counts one beat per slot, taken or lost, and would light the wrong card
// for the rest of the round if a slot went unaccounted for.
func TestEverySlotIsEitherTakenOrChilled(t *testing.T) {
	a, b := wearing(duelist(10, 4, 20000), Ice), duelist(10, 4, 20000)
	aPlan := []Card{Of(Bash, Ice), Of(Bash, Ice), Of(Bash, Ice)}
	bPlan := PlainCards(Block, Jab, Bash, Guard)

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

// **A blow of nothing may not spend anything of the target's** *(owner's call, 2026-09-02)*. A
// shield eats one attack whole and shields are cleared by the turn they answer, so a turn of
// shields that counted as an attack would strip an opponent's shields for free — which is the
// whole reason the zero blow is counted and not thrown.
func TestAZeroBlowSpendsNothingOfTheTargets(t *testing.T) {
	a, b := duelist(10, 4, 5000), duelist(10, 4, 5000)
	b.Shields = 2

	events, _, after := resolve(a, b, PlainCards(Block, Block, Block), nil, 1)

	// The target's own turn expires what they were holding, so what proves the blow never touched
	// it is the expiry announcing both shields still standing.
	held := 0
	for _, e := range events {
		if e.Kind == KindExpired && e.Target == SideB {
			held = e.Amount
		}
		if e.Kind == KindBlocked {
			t.Errorf("a zero blow was blocked: it never happened")
		}
	}
	if held != 2 {
		t.Errorf("%d of the target's shields survived the turn, want both", held)
	}
	if after.CurrentLife != b.CurrentLife {
		t.Errorf("the target is on %d life, want the %d they started with",
			after.CurrentLife, b.CurrentLife)
	}
}
