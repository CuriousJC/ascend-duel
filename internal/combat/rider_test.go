package combat

import "testing"

// ridden is a card carrying one heal rider.
func ridden(id ConceptID, heal int) Card {
	return Plain(id).SetRider(Rider{Kind: RiderHealOnPlay, Amount: heal})
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

	events, after, _ := resolve(a, b, []Card{ridden(Bash, 10)}, nil, 1)

	if got := healedBy(events, SideA); got != 10 {
		t.Errorf("a rider worth 10 healed %d", got)
	}
	if after.CurrentLife != 60 {
		t.Errorf("life went %d to %d, wanted 60", 50, after.CurrentLife)
	}
}

func TestASecondRiderReplacesTheFirst(t *testing.T) {
	// **Last one wins** — see Card.SetRider. A card holds one upgrade, so a second heal is not a
	// second ten: it is the card forgetting the first one. This is the test that would go red if
	// stacking came back in.
	c := ridden(Bash, 10).SetRider(Rider{Kind: RiderHealOnPlay, Amount: 7})

	a := duelist(10, 3, 100)
	a.CurrentLife = 50

	events, after, _ := resolve(a, duelist(0, 0, 100), []Card{c}, nil, 1)

	if got := healedBy(events, SideA); got != 7 {
		t.Errorf("the replacing rider healed %d, wanted 7", got)
	}
	if after.CurrentLife != 57 {
		t.Errorf("life ended at %d, wanted 57", after.CurrentLife)
	}
}

func TestAHealNeverGoesAboveFullLife(t *testing.T) {
	// **Nothing in the game heals above full**, and the event says what actually happened rather
	// than what the rider names — a figure in the log the bar cannot show would be the log lying.
	a := duelist(10, 3, 100)
	a.CurrentLife = 95

	events, after, _ := resolve(a, duelist(0, 0, 100), []Card{ridden(Bash, 10)}, nil, 1)

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

	events, after, _ := resolve(a, duelist(0, 0, 100), []Card{ridden(Bash, 10)}, nil, 1)

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

	events, after, _ := resolve(a, duelist(0, 0, 100), []Card{ridden(Bash, 10)}, nil, 1)

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
	c := Plain(Bash)
	if c.RiderCount() != 0 || c.HealOnPlay() != 0 || len(c.RiderList()) != 0 {
		t.Errorf("a plain card reported riders: %+v", c)
	}
}

func TestACardCarriesOneUpgradeAndNoMore(t *testing.T) {
	// A card has a form, an element and an action, and then **one** upgrade. This is the rules
	// half of that; upgradeOf in internal/screens is the drawing half, and both would have to be
	// changed together for a card to wear two.
	if MaxCardRiders != 1 {
		t.Fatalf("a card holds %d riders; the whole upgrade grammar assumes one", MaxCardRiders)
	}
	c := Plain(Bash).
		SetRider(Rider{Kind: RiderHealOnPlay, Amount: 1}).
		SetRider(Rider{Kind: RiderWildElement})
	if c.RiderCount() != 1 {
		t.Errorf("a card ridden twice carries %d riders", c.RiderCount())
	}
	if c.HealOnPlay() != 0 {
		t.Errorf("the replaced heal still pays %d", c.HealOnPlay())
	}
	if !c.Wild(AxisElement) {
		t.Error("the replacing rider did not take")
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
	//
	// **Through variables rather than as two literal calls**, so that the comparison is not a
	// tautology a reader has to re-derive — and so staticcheck does not read it as one.
	same, alsoSame := ridden(Bash, 10), ridden(Bash, 10)
	other := ridden(Bash, 20)
	if same != alsoSame {
		t.Error("two identically ridden cards did not compare equal")
	}
	if same == other {
		t.Error("two differently ridden cards compared equal")
	}
}

// carrying is a card with one rider of a named kind, for the kinds that are not a heal.
func carrying(id ConceptID, kind RiderKind, amount int) Card {
	return Plain(id).SetRider(Rider{Kind: kind, Amount: amount})
}

// holding is a round told what the player kept back, which is the only way to reach the four
// in-hand riders.
func holding(a, b Duelist, aCards, held []Card, round int) ([]Event, Duelist, Duelist) {
	return ResolveRoundHolding(a, b, aCards, nil, held, nil, round, Sources{})
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

	bare, _, _ := holding(a, b, PlainCards(Bash, Bash), nil, 1)
	held, _, _ := holding(a, b, PlainCards(Bash, Bash),
		[]Card{carrying(Jab, RiderDamageInHand, 10)}, 1)

	if blowOf(held, SideA) <= blowOf(bare, SideA) {
		t.Errorf("a card held back for +10 DMG changed the blow from %d to %d",
			blowOf(bare, SideA), blowOf(held, SideA))
	}
}

func TestACardHeldBackIsNotPlayedAndDoesNotFormTheHand(t *testing.T) {
	// The held card is a Bash and so are the two played ones. If holding it reached the matcher
	// it would make trips out of a pair, and the multiplier would move — which would be the
	// resolver treating a card nobody played as one that was.
	a := duelist(10, 3, 100)
	b := duelist(0, 0, 1000)

	pair, _, _ := holding(a, b, PlainCards(Bash, Bash), nil, 1)
	withHeld, _, _ := holding(a, b, PlainCards(Bash, Bash), PlainCards(Bash), 1)

	one, ok := handEventFor(pair, SideA)
	if !ok {
		t.Fatal("a pair formed no hand")
	}
	two, ok := handEventFor(withHeld, SideA)
	if !ok {
		t.Fatal("a pair with a card held back formed no hand")
	}
	if one.Hand != two.Hand || one.Multiplier != two.Multiplier {
		t.Errorf("holding a third Bash changed the hand from %d (x%d) to %d (x%d)",
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
		events, _, _ := holding(a, b, PlainCards(Bash), held, round)

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
	// A Jab is an attack card. The point of the rider is that it does a defense's job as well,
	// which no card in the catalog does.
	a := duelist(10, 3, 100)
	events, after, _ := resolve(a, duelist(0, 0, 1000),
		[]Card{carrying(Jab, RiderShieldOnPlay, 1)}, nil, 1)

	if after.Shields.Count() != 1 {
		t.Errorf("an attack carrying a shield rider left %d shields, wanted 1", after.Shields.Count())
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

func TestARiderThatScalesInComboDoublesOnlyItsOwnHit(t *testing.T) {
	// **A played card's rider prices its own hit and nothing else** *(owner's call, 2026-09-26)*.
	// Two Bashes form a pair and the ridden Bash is one of them: its hit doubles, and the other
	// Bash's hit is exactly what it is without the rider in the turn.
	a := duelist(10, 3, 100)
	b := duelist(0, 0, 1000)

	ridden, _, _ := resolve(a, b, []Card{carrying(Bash, RiderScaleInCombo, 200), Plain(Bash)}, nil, 1)
	bare, _, _ := resolve(a, b, PlainCards(Bash, Bash), nil, 1)
	re, be := handEventOf(t, ridden, SideA), handEventOf(t, bare, SideA)

	if want := scaleDamage(be.HandAmounts[0]*2, be.Multiplier); re.HitAmounts[0] != want {
		t.Errorf("the ridden Bash's hit came to %d, want its term doubled under the hand = %d",
			re.HitAmounts[0], want)
	}
	if re.HitAmounts[1] != be.HitAmounts[1] {
		t.Errorf("the other Bash's hit came to %d, want the %d it deals with no rider in the turn",
			re.HitAmounts[1], be.HitAmounts[1])
	}
	if re.HandDMG != be.HandDMG {
		t.Errorf("the turn was swung at %d DMG, want the bare %d: a played card's rider is not the turn's",
			re.HandDMG, be.HandDMG)
	}
	if re.HandPlayPct[0] != 200 || re.HandPlayPct[1] != 100 {
		t.Errorf("the working says %d%% and %d%%, want 200 on the ridden hit and 100 on the other",
			re.HandPlayPct[0], re.HandPlayPct[1])
	}
}

func TestAScaleOnPlayRiderPaysWithoutMakingTheHand(t *testing.T) {
	// **Played is enough** *(owner's call, 2026-09-26)*: a Bash beside a pair of Jabs rides along on
	// their Pair without making it, and its own hit still doubles.
	a := duelist(10, 3, 100)
	b := duelist(0, 0, 1000)

	ridden, _, _ := resolve(a, b, []Card{Plain(Jab), Plain(Jab), carrying(Bash, RiderScaleInCombo, 200)}, nil, 1)
	bare, _, _ := resolve(a, b, PlainCards(Jab, Jab, Bash), nil, 1)
	re, be := handEventOf(t, ridden, SideA), handEventOf(t, bare, SideA)

	last := re.HandCardCount - 1
	if want := scaleDamage(be.HandAmounts[last]*2, be.Multiplier); re.HitAmounts[last] != want {
		t.Errorf("the ridden Bash outside the hand came to %d, want its term doubled under the hand = %d",
			re.HitAmounts[last], want)
	}
	for n := 0; n < last; n++ {
		if re.HitAmounts[n] != be.HitAmounts[n] {
			t.Errorf("Jab %d came to %d, want the bare %d", n, re.HitAmounts[n], be.HitAmounts[n])
		}
	}
}

func TestACardsRidersGoOnBeforeItsRelics(t *testing.T) {
	// **DUELIST, CARD, RELICS, HAND** *(owner's call, 2026-09-26)*: a relic prices what the card came
	// to after its own riders, so a +10 under a 2x relic is worth 20.
	keen := relic(t, "order-keen", RelicRule{
		When: MomentCardDamage,
		If:   RelicCondition{},
		Then: []RelicEffect{{Do: DoScaleDamage, Amount: 200}},
	})
	a := duelist(10, 3, 100).Wearing(WornRelic{Relic: keen})
	b := duelist(0, 0, 1000)

	ridden, _, _ := resolve(a, b, []Card{carrying(Bash, RiderDamageOnPlay, 10)}, nil, 1)
	e := handEventOf(t, ridden, SideA)
	// Bash is 1x: (10 x 1 + 10) x 2 = 40 before the hand.
	if e.HandAmounts[0] != 40 {
		t.Errorf("a Bash with +10 under a 2x relic came to %d before the hand, want (10+10) x 2 = 40",
			e.HandAmounts[0])
	}
}

func TestARidersDamageBonusDoesNotOutliveTheBlow(t *testing.T) {
	// **The DMG is put back before the duelist is returned** — see handEvent. A bonus that stuck
	// would turn a one-turn rider into a permanent upgrade, silently.
	a := duelist(10, 3, 100)
	_, after, _ := resolve(a, duelist(0, 0, 1000),
		[]Card{carrying(Bash, RiderDamageOnPlay, 10)}, nil, 1)

	if after.DMG != 10 {
		t.Errorf("the duelist came out of the round at %d DMG, wanted 10", after.DMG)
	}
}

func TestADamageOnPlayRiderAddsToOnlyItsOwnHit(t *testing.T) {
	// **The +10 is the card's, not the turn's** *(owner's call, 2026-09-26)*: it goes on the ridden
	// card's term before the hand multiplies it, and the other card's hit does not see it.
	a := duelist(10, 3, 100)
	b := duelist(0, 0, 1000)

	ridden, _, _ := resolve(a, b, []Card{carrying(Bash, RiderDamageOnPlay, 10), Plain(Bash)}, nil, 1)
	bare, _, _ := resolve(a, b, PlainCards(Bash, Bash), nil, 1)
	re, be := handEventOf(t, ridden, SideA), handEventOf(t, bare, SideA)

	if want := scaleDamage(be.HandAmounts[0]+10, be.Multiplier); re.HitAmounts[0] != want {
		t.Errorf("the ridden Bash's hit came to %d, want its term plus 10 under the hand = %d",
			re.HitAmounts[0], want)
	}
	if re.HitAmounts[1] != be.HitAmounts[1] {
		t.Errorf("the other Bash's hit came to %d, want the %d it deals with no rider in the turn",
			re.HitAmounts[1], be.HitAmounts[1])
	}
	if re.HandPlayAdd[0] != 10 || re.HandPlayAdd[1] != 0 {
		t.Errorf("the working says +%d and +%d, want +10 on the ridden hit and nothing on the other",
			re.HandPlayAdd[0], re.HandPlayAdd[1])
	}
}
