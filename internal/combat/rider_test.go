package combat

import "testing"

// ridden is a card carrying one heal rider, which is the only rider that exists.
func ridden(id ConceptID, heal int) Card {
	c, ok := Plain(id).AddRider(Rider{Kind: RiderHealOnPlay, Amount: heal})
	if !ok {
		panic("a fresh card had no room for a rider")
	}
	return c
}

// healedBy is the total life a side restored across a whole round.
func healedBy(events []Event, side Side) int {
	total := 0
	for _, e := range events {
		if e.Kind == KindHealed && e.Side == side {
			total += e.Amount
		}
	}
	return total
}

func TestARiddenCardHealsItsOwnerAsItIsPlayed(t *testing.T) {
	a := duelist(10, 3, 100)
	a.CurrentLife = 50
	b := duelist(0, 0, 100)

	events, after, _ := resolve(a, b, []Card{ridden(Strike, 10)}, nil, 1)

	if got := healedBy(events, SideA); got != 10 {
		t.Errorf("a rider worth 10 healed %d", got)
	}
	if after.CurrentLife != 60 {
		t.Errorf("life went %d to %d, wanted 60", 50, after.CurrentLife)
	}
}

func TestTwoRidersOnOneCardBothFire(t *testing.T) {
	// **Riders stack rather than merge** — see Card.AddRider. Two tens are twenty, and the reason
	// to hold it is that it is what keeps the card's face honest about how many parasites have
	// been spent on it.
	c := ridden(Strike, 10)
	c, ok := c.AddRider(Rider{Kind: RiderHealOnPlay, Amount: 10})
	if !ok {
		t.Fatal("a card with one rider had no room for a second")
	}

	a := duelist(10, 3, 100)
	a.CurrentLife = 50

	events, after, _ := resolve(a, duelist(0, 0, 100), []Card{c}, nil, 1)

	if got := healedBy(events, SideA); got != 20 {
		t.Errorf("two riders worth 10 healed %d, wanted 20", got)
	}
	if after.CurrentLife != 70 {
		t.Errorf("life ended at %d, wanted 70", after.CurrentLife)
	}
}

func TestAHealNeverGoesAboveFullLife(t *testing.T) {
	// **Nothing in the game heals above full**, and the event says what actually happened rather
	// than what the rider names — a figure in the log the bar cannot show would be the log lying.
	a := duelist(10, 3, 100)
	a.CurrentLife = 95

	events, after, _ := resolve(a, duelist(0, 0, 100), []Card{ridden(Strike, 10)}, nil, 1)

	if after.CurrentLife != 100 {
		t.Errorf("life ended at %d, wanted the cap at 100", after.CurrentLife)
	}
	if got := healedBy(events, SideA); got != 5 {
		t.Errorf("the event reported %d healed, wanted the 5 that actually landed", got)
	}
}

func TestAHealOnFullLifeIsSilent(t *testing.T) {
	// A rider that fired and changed nothing must not put a line in the feed saying life was
	// restored. The rider is still spent, because it is a property of the card rather than a charge.
	a := duelist(10, 3, 100)

	events, after, _ := resolve(a, duelist(0, 0, 100), []Card{ridden(Strike, 10)}, nil, 1)

	if after.CurrentLife != 100 {
		t.Errorf("life moved to %d on a full-life heal", after.CurrentLife)
	}
	for _, e := range events {
		if e.Kind == KindHealed {
			t.Fatalf("a heal that restored nothing wrote an event: %+v", e)
		}
	}
}

func TestAChilledCardHealsNothing(t *testing.T) {
	// **A card a chill ate was never played**, which is the whole reason riders fire after the
	// chill and before the blow. Getting this backwards would make a rider on the front card of a
	// turn immune to the one thing that can delete it.
	a := duelist(10, 3, 100)
	a.CurrentLife = 50
	a.Statuses[statusOf(Ice)] = Status{Amount: 1, Rounds: 2}

	events, after, _ := resolve(a, duelist(0, 0, 100), []Card{ridden(Strike, 10)}, nil, 1)

	if got := healedBy(events, SideA); got != 0 {
		t.Errorf("a chilled card healed %d", got)
	}
	if after.CurrentLife != 50 {
		t.Errorf("life moved to %d on a turn nothing was played in", after.CurrentLife)
	}
}

func TestAnUnriddenCardIsTheZeroValue(t *testing.T) {
	// Nothing may *require* a rider, and the common case has to stay the plain literal every test
	// in this package writes.
	c := Plain(Strike)
	if c.RiderCount() != 0 || c.HealOnPlay() != 0 || len(c.RiderList()) != 0 {
		t.Errorf("a plain card reported riders: %+v", c)
	}
}

func TestACardTakesNoMoreRidersThanTheFaceCanShow(t *testing.T) {
	// MaxCardRiders is a layout number as much as a rules one — a card whose face cannot say what
	// it carries is the failure the alteration mechanic exists to avoid — so the refusal is here
	// rather than a silent drop.
	c := Plain(Strike)
	for i := 0; i < MaxCardRiders; i++ {
		var ok bool
		if c, ok = c.AddRider(Rider{Kind: RiderHealOnPlay, Amount: 1}); !ok {
			t.Fatalf("rider %d of %d was refused", i+1, MaxCardRiders)
		}
	}
	if _, ok := c.AddRider(Rider{Kind: RiderHealOnPlay, Amount: 1}); ok {
		t.Errorf("a card took more than %d riders", MaxCardRiders)
	}
}

func TestEveryRiderKindHasANameThatParsesBack(t *testing.T) {
	// The snapshot writes a rider's *name*, never its ordinal, so a kind whose name does not round
	// trip is a save file that cannot be resumed. See profile.RiderSnapshot.
	for _, k := range RiderKinds() {
		got, ok := ParseRiderKind(k.String())
		if !ok || got != k {
			t.Errorf("rider %d spells itself %q, which parses back as %d/%v", k, k.String(), got, ok)
		}
	}
	if _, ok := ParseRiderKind("no-such-rider"); ok {
		t.Error("an unknown rider name resolved to something")
	}
}

func TestARiderDoesNotStopACardBeingComparable(t *testing.T) {
	// The screen's face cache and TestRoundIsDeterministic both compare cards by value, which is
	// why Riders is a fixed array. A slice here would not compile at all; this is what says so out
	// loud, so the field is not "tidied up" into one later.
	if ridden(Strike, 10) != ridden(Strike, 10) {
		t.Error("two identically ridden cards did not compare equal")
	}
	if ridden(Strike, 10) == ridden(Strike, 20) {
		t.Error("two differently ridden cards compared equal")
	}
}

// carrying is a card with one rider of a named kind, for the six kinds that are not a heal.
func carrying(id ConceptID, kind RiderKind, amount int) Card {
	c, ok := Plain(id).AddRider(Rider{Kind: kind, Amount: amount})
	if !ok {
		panic("a fresh card had no room for a rider")
	}
	return c
}

// holding is a round told what the player kept back, which is the only way to reach the four
// in-hand riders.
func holding(a, b Duelist, aCards, held []Card, round int) ([]Event, Duelist, Duelist) {
	return ResolveRoundHolding(a, b, aCards, nil, held, nil, round, nil)
}

// blowOf is the damage one side dealt across a whole round.
func blowOf(events []Event, side Side) int {
	total := 0
	for _, e := range events {
		if e.Kind == KindDamage && e.Side == side {
			total += e.Amount
		}
	}
	return total
}

func TestACardHeldBackAddsToTheDuelistsDamage(t *testing.T) {
	// **The whole calculation moves, not one term** — see blowDMG. The comparison is the same
	// round played twice, once with the ridden card in hand and once with nothing held, so what
	// is measured is the rider and not the arithmetic around it.
	a := duelist(10, 3, 100)
	b := duelist(0, 0, 1000)

	bare, _, _ := holding(a, b, PlainCards(Strike, Strike), nil, 1)
	held, _, _ := holding(a, b, PlainCards(Strike, Strike),
		[]Card{carrying(Jab, RiderDamageInHand, 10)}, 1)

	if blowOf(held, SideA) <= blowOf(bare, SideA) {
		t.Errorf("a card held back for +10 DMG changed the blow from %d to %d",
			blowOf(bare, SideA), blowOf(held, SideA))
	}
}

func TestACardHeldBackIsNotPlayedAndDoesNotFormTheHand(t *testing.T) {
	// The held card is a Strike and so are the two played ones. If holding it reached the matcher
	// it would make trips out of a pair, and the multiplier would move — which would be the
	// resolver treating a card nobody played as one that was.
	a := duelist(10, 3, 100)
	b := duelist(0, 0, 1000)

	pair, _, _ := holding(a, b, PlainCards(Strike, Strike), nil, 1)
	withHeld, _, _ := holding(a, b, PlainCards(Strike, Strike), PlainCards(Strike), 1)

	one, ok := handEventFor(pair, SideA)
	if !ok {
		t.Fatal("a pair formed no hand")
	}
	two, ok := handEventFor(withHeld, SideA)
	if !ok {
		t.Fatal("a pair with a card held back formed no hand")
	}
	if one.Hand != two.Hand || one.Multiplier != two.Multiplier {
		t.Errorf("holding a third Strike changed the hand from %d (x%d) to %d (x%d)",
			one.Hand, one.Multiplier, two.Hand, two.Multiplier)
	}
}

func TestACardHeldBackPaysVitaeEveryTurnItIsHeld(t *testing.T) {
	// **It is announced, never applied** — the rules have no purse. What this proves is that the
	// announcement arrives, with the figure the rider names, on every turn the card is still held.
	a := duelist(10, 3, 100)
	b := duelist(0, 0, 1000)
	held := []Card{carrying(Jab, RiderVitaeInHand, 3)}

	for round := 1; round <= 3; round++ {
		events, _, _ := holding(a, b, PlainCards(Strike), held, round)

		paid := 0
		for _, e := range events {
			if e.Kind == KindVitae && e.Side == SideA {
				paid += e.Amount
			}
		}
		if paid != 3 {
			t.Errorf("round %d paid %d vitae, wanted 3", round, paid)
		}
	}
}

func TestARiddenCardRaisesShieldsAsItIsPlayed(t *testing.T) {
	// A Jab is an attack card. The point of the rider is that it does a defence's job as well,
	// which no card in the catalogue does.
	a := duelist(10, 3, 100)
	events, after, _ := resolve(a, duelist(0, 0, 1000),
		[]Card{carrying(Jab, RiderShieldOnPlay, 1)}, nil, 1)

	if after.Shields != 1 {
		t.Errorf("an attack carrying a shield rider left %d shields, wanted 1", after.Shields)
	}
	raised := false
	for _, e := range events {
		if e.Kind == KindRaised && e.Side == SideA {
			raised = true
		}
	}
	if !raised {
		t.Error("shields went up with nothing in the log saying so")
	}
}

func TestARiderThatScalesInComboNeedsTheCardToMakeTheHand(t *testing.T) {
	// **`Blow.Cards` is the scoring set, not the turn** — so a ridden card played *outside* the
	// hand pays nothing, and the same card played into it doubles the blow. That is the whole
	// distinction the rider is written to make.
	a := duelist(10, 3, 100)
	b := duelist(0, 0, 1000)

	// Two Strikes form a pair; the ridden Strike is one of them.
	inHand, _, _ := resolve(a, b, []Card{carrying(Strike, RiderScaleInCombo, 200), Plain(Strike)}, nil, 1)
	bare, _, _ := resolve(a, b, PlainCards(Strike, Strike), nil, 1)

	// **Within a point, because the multiplier truncates** *(2026-09-05)*. scaleDamage is integer
	// arithmetic rounding toward zero, so doubling the DMG and *then* scaling is not always the
	// same as scaling and then doubling — `40 * 114 / 100` is 45 where `(20 * 114 / 100) * 2` is
	// 44. What this pins is that the ridden card doubled the blow; the odd point is the ladder's
	// rounding and pinning it exactly would make the test fail on any multiplier that is not a
	// factor of the sum, which is a tuning constraint nobody agreed to.
	got, want := blowOf(inHand, SideA), blowOf(bare, SideA)*2
	if got < want || got > want+1 {
		t.Errorf("a 2x rider in the combo dealt %d, wanted %d (or one more, for the truncation)", got, want)
	}
}

func TestARidersDamageBonusDoesNotOutliveTheBlow(t *testing.T) {
	// **The DMG is put back before the duelist is returned** — see handEvent. A bonus that stuck
	// would turn a one-turn rider into a permanent upgrade, silently.
	a := duelist(10, 3, 100)
	_, after, _ := resolve(a, duelist(0, 0, 1000),
		[]Card{carrying(Strike, RiderDamageOnPlay, 10)}, nil, 1)

	if after.DMG != 10 {
		t.Errorf("the duelist came out of the round at %d DMG, wanted 10", after.DMG)
	}
}
