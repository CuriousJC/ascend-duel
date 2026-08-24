package combat

import "testing"

// The 2026-08-15 card set: three attack forms of three tiers, plus the three plans. Each of
// the plans gets the case that says what it does, and the ladder gets the structural cases that
// say the forms are three ways of doing the same thing at the same prices — which is the claim
// MECHANICS.md makes about the deck and the one a retuned cost quietly breaks.

// hasKind reports whether the log holds an event of the given kind.
func hasKind(events []Event, kind EventKind) bool {
	for _, e := range events {
		if e.Kind == kind {
			return true
		}
	}
	return false
}

// kindCount is how many events of a kind the log holds.
func kindCount(events []Event, kind EventKind) int {
	n := 0
	for _, e := range events {
		if e.Kind == kind {
			n++
		}
	}
	return n
}

func TestPrepareBanksForTheRoundAfter(t *testing.T) {
	// **Not the round it is played in.** Those points were committed when the cards were queued,
	// so a bank that funded its own turn would be a free action rather than an investment.
	a := duelist(10, 4, 100)
	b := duelist(10, 4, 100)

	events, aAfter, _ := resolve(a, b, PlainCards(Prepare), nil, 1)

	if !hasKind(events, KindGathered) {
		t.Fatal("no KindGathered event — the Prepare banked nothing")
	}
	bank := ConceptOf(Prepare).Amount
	if aAfter.BonusAP != bank {
		t.Errorf("BonusAP after a Prepare = %d, want %d", aAfter.BonusAP, bank)
	}
	if aAfter.ActionPoints() != a.Actions+bank {
		t.Errorf("next round's budget = %d, want %d", aAfter.ActionPoints(), a.Actions+bank)
	}
}

func TestPrepareIsWorthMoreThanItCosts(t *testing.T) {
	// Two for one is a deliberate profit. What Prepare actually costs is the card slot and the
	// action slot it takes out of the round it is played in, not the point on its face — so if the
	// bank ever stops beating the price, the card is pure loss and nobody would ever play it.
	if net := ConceptOf(Prepare).Amount - ConceptOf(Prepare).Cost; net <= 0 {
		t.Errorf("Prepare nets %+d AP; a bank that does not profit is a card nobody plays", net)
	}
}

// Defend is the whole defensive vocabulary as of 2026-08-15. Under one blow per turn it takes half
// of what arrives and is then spent.
func TestDefendHalvesTheBlowAndIsSpent(t *testing.T) {
	a := duelist(10, 4, 500)
	b := duelist(10, 4, 500)

	// B defends first. B acts second, so its Defend is standing when A's next turn arrives.
	// A Smash and a Thrust are different concepts and different forms, and both are colourless, so
	// they agree on no axis and form no hand: the blow is the High Card — the Smash alone — and the
	// arithmetic is about the Defend rather than about a multiplier.
	_, a, b = resolve(a, b, nil, PlainCards(Defend), 1)
	events, _, bAfter := resolve(a, b, PlainCards(Smash, Thrust), nil, 2)

	if !hasKind(events, KindNegated) {
		t.Fatal("no KindNegated event — the Defend did not apply")
	}
	if want := 500 - Plain(Smash).Damage(10)*(100-ConceptOf(Defend).Amount)/100; bAfter.CurrentLife != want {
		t.Errorf("life after a halved Smash = %d, want %d", bAfter.CurrentLife, want)
	}
	if bAfter.DefendCount != 0 {
		t.Errorf("defends left = %d, want 0 — every defence is spent on the blow it answered", bAfter.DefendCount)
	}
}

// **No card reduces a blow to zero, and that is the design.** A turn lands one figure however many
// cards went into it, so a card taking all of it would delete a whole opposing turn by itself.
// Halving cannot: something always lands, so the opponent is always still playing.
func TestNoDefenceStopsABlowOutright(t *testing.T) {
	dmg := 10
	a := duelist(dmg, 0, 500)
	b := duelist(dmg, 0, 500)

	// Four Jabs is a Barrage, which is the biggest thing a cheap hand can assemble, into every
	// defensive card the game has.
	_, a, b = resolve(a, b, nil, PlainCards(Defend, Defend), 1)
	events, _, bAfter := resolve(a, b, PlainCards(Jab, Jab, Jab, Jab), nil, 2)

	if got := firstDamage(t, events, SideA).Amount; got <= 0 {
		t.Errorf("a Jab Barrage into two Defends dealt %d — nothing may reduce a blow to zero", got)
	}
	if bAfter.CurrentLife >= 500 {
		t.Error("the defender took nothing at all")
	}
}

// **A Plan banks a wider hand, and the engine can only record that it is owed.** There is no deck
// in this package, so the whole of what the card produces here is BonusDraw plus an event — the
// screen holds the deck and honours it.
func TestPlanBanksItsDrawForTheFollowingRound(t *testing.T) {
	a := duelist(10, 4, 100)
	b := duelist(10, 4, 100)

	events, a1, _ := resolve(a, b, PlainCards(Plan), nil, 1)

	if !hasKind(events, KindDrew) {
		t.Fatal("no KindDrew event — nothing tells the screen to draw")
	}
	drew := ConceptOf(Plan).Amount
	if a1.BonusDraw != drew {
		t.Errorf("BonusDraw = %d after a Plan, want %d", a1.BonusDraw, drew)
	}
	if a1.DrewCards != 0 {
		t.Errorf("DrewCards = %d at the round boundary, want it rolled into BonusDraw", a1.DrewCards)
	}

	// **It lasts exactly one round.** BonusDraw is assigned rather than added to, so a round with
	// no Plan in it puts the hand back to its usual size — which is what makes the card a boost
	// rather than a hand that grows all game.
	_, a2, _ := resolve(a1, b, nil, nil, 2)
	if a2.BonusDraw != 0 {
		t.Errorf("BonusDraw = %d a round later, want 0 — a Plan widens one hand, not every hand", a2.BonusDraw)
	}

	// And two in a round stack, for the same reason two Prepares do.
	_, a3, _ := resolve(a, b, PlainCards(Plan, Plan), nil, 1)
	if a3.BonusDraw != 2*ConceptOf(Plan).Amount {
		t.Errorf("two Plans banked %d, want %d", a3.BonusDraw, 2*ConceptOf(Plan).Amount)
	}
}

func TestEveryRaisedDefenceMeetsTheBlow(t *testing.T) {
	// **Every card fires, not just the front one.** A turn lands one blow, so the choice of which
	// card meets which has nothing to choose between and the whole set answers it. Each card also
	// emits its own event, which is what the Resolution feed narrates.
	//
	// Composed multiplicatively, so two Defends take three quarters rather than the whole thing —
	// which is the point of multiplying rather than adding: cards cannot reach past zero by accident.
	dmg := 100
	a := duelist(dmg, 0, 5000)
	b := duelist(dmg, 0, 5000)

	// The undefended figure comes off a fresh pair, so it is the same blow against nothing.
	open, _, _ := resolve(a, b, PlainCards(Smash, Strike), nil, 2)

	_, a, b = resolve(a, b, nil, PlainCards(Defend, Defend), 1)
	events, _, _ := resolve(a, b, PlainCards(Smash, Strike), nil, 2)

	if got := kindCount(events, KindNegated); got != 2 {
		t.Errorf("%d defences fired, want both of them", got)
	}
	full := firstDamage(t, open, SideA).Amount
	if got, want := firstDamage(t, events, SideA).Amount, full*25/100; got != want {
		t.Errorf("a blow into two Defends dealt %d, want a quarter of %d = %d", got, full, want)
	}
}

func TestDefencesAreSpentWhetherOrNotTheyWereNeeded(t *testing.T) {
	// A defence answers the opponent's turn and then goes, spent or not. It cannot be banked
	// against a turn that queued no attack at all.
	a := duelist(10, 4, 100)
	b := duelist(10, 4, 100)

	// A raises two defences into a turn with nothing to answer.
	_, a1, b1 := resolve(a, b, PlainCards(Defend, Defend), nil, 1)
	if a1.DefendCount != 2 {
		t.Fatalf("A ended round 1 holding %d defends, want 2 unspent", a1.DefendCount)
	}

	// Round two: A queues nothing, so its own turn expires them before B swings.
	events, _, _ := resolve(a1, b1, nil, PlainCards(Strike, Strike), 2)

	open, _, _ := resolve(a, b, nil, PlainCards(Strike, Strike), 1)
	if got, want := firstDamage(t, events, SideB).Amount, firstDamage(t, open, SideB).Amount; got != want {
		t.Errorf("a blow into expired defends dealt %d, want the full %d", got, want)
	}
}

func TestClearDefensesClearsEveryDefensiveField(t *testing.T) {
	// ClearDefenses is the one place that has to know about every defensive field, and a new one
	// left out of it would stand forever — the worst failure mode available here, and a silent
	// one. Pin the whole set rather than only the fields this session added.
	var d Duelist
	for _, k := range []ConceptID{Defend, Defend} {
		d = d.raiseDefend(Plain(k))
	}

	got := ClearDefenses(d)

	if got.DefendCount != 0 || got.Defends != ([maxPendingDefends]PendingDefend{}) {
		t.Errorf("ClearDefenses left something standing: %+v", got)
	}
}

func TestTheAttackLadderIsThreeFormsByFiveTiers(t *testing.T) {
	// One concept per form per tier, and **the tiers are identical across the forms** — same
	// cost, same damage. **Five rungs, not three** *(2026-08-24)*: the 0 AP and 4 AP ends ship at
	// zero copies and exist only for a worm to walk a card onto, but they are rungs of the same
	// ladder and have to match across the forms exactly as the dealt three do. That is the structural claim MECHANICS.md makes about the deck: a form
	// is which pair you are building, never a better or worse way to build one. It is also the
	// thing that quietly breaks the first time somebody makes a Cleave hit harder than a Lunge.
	//
	// It also catches a concept falling through Cost()'s default arm, which returns a mid-tier
	// price — a mistake that would otherwise hide behind an entirely plausible number.
	const dmg = 10

	attackForms := []Form{FormStab, FormSlash, FormCrush}
	tiers := map[Form]map[int]ConceptID{}

	for _, a := range PlayerConcepts() {
		fam := ConceptOf(a).Form
		if Plain(a).Category() != CategoryAttack || fam == FormNone {
			continue
		}
		if ConceptOf(a).Cost < 0 || ConceptOf(a).Cost > 4 {
			t.Errorf("%v costs %d, outside the 0-4 tiers the ladder is built on", a, ConceptOf(a).Cost)
		}
		if tiers[fam] == nil {
			tiers[fam] = map[int]ConceptID{}
		}
		if other, taken := tiers[fam][ConceptOf(a).Cost]; taken {
			t.Errorf("%v and %v are both %v at %d AP — the ladder holds one concept per rung", other, a, fam, ConceptOf(a).Cost)
		}
		tiers[fam][ConceptOf(a).Cost] = a
	}

	for _, fam := range attackForms {
		if len(tiers[fam]) != 5 {
			t.Errorf("%v has %d of 5 tiers filled", fam, len(tiers[fam]))
		}
	}

	// Every form's rung deals what Stab's rung deals.
	for tier := 0; tier <= 4; tier++ {
		want, ok := tiers[FormStab][tier]
		if !ok {
			continue
		}
		for _, fam := range attackForms[1:] {
			got, ok := tiers[fam][tier]
			if !ok {
				continue
			}
			if Plain(got).Damage(dmg) != Plain(want).Damage(dmg) {
				t.Errorf("at %d AP, %v deals %d and %v deals %d — the forms must ladder identically",
					tier, got, Plain(got).Damage(dmg), want, Plain(want).Damage(dmg))
			}
		}
	}
}

func TestPlanCardsDealNothingAndAttacksDoNot(t *testing.T) {
	// The two categories are what a card *is*, so each has to hold on its own side of the line. A
	// plan that dealt damage would be an attack wearing the wrong verb in the feed.
	const dmg = 10
	for _, a := range PlayerConcepts() {
		switch Plain(a).Category() {
		case CategoryPlan:
			if d := Plain(a).Damage(dmg); d != 0 {
				t.Errorf("%v is a plan and deals %d damage", a, d)
			}
		case CategoryAttack:
			if d := Plain(a).Damage(dmg); d <= 0 {
				t.Errorf("%v is an attack and deals %d damage", a, d)
			}
		}
	}
}

func TestEveryPlanIsInThePlanFormAndViceVersa(t *testing.T) {
	// Category has two values and is derivable from the form, which is exactly why the two must
	// not be able to disagree: the card face says the form and the feed says the category, and a
	// card whose corner and verb told different stories would be the screen contradicting itself.
	for _, a := range PlayerConcepts() {
		if (ConceptOf(a).Form == FormPlan) != (Plain(a).Category() == CategoryPlan) {
			t.Errorf("%v is form %v but category %v", a, ConceptOf(a).Form, Plain(a).Category())
		}
	}
}

func TestEveryPlayerCardHasAForm(t *testing.T) {
	// **FormNone is a real answer, not a fallthrough**, and after 2026-08-16 it belongs entirely
	// to the enemies: a form is the player's deck axis, the thing a pair is counted on and the
	// mark in the card's corner. A player's card landing there would draw with a blank corner and
	// be excluded from the deck panel's sort.
	//
	// It used to name the two shared enemy concepts as the permitted exceptions. There is no shared
	// enemy deck now — every enemy carries its own cards, all of them formless — so the rule is
	// simply that the player's side is fully covered.
	for _, a := range PlayerConcepts() {
		if ConceptOf(a).Form == FormNone {
			t.Errorf("%v has no form; only an enemy's cards may", ConceptOf(a).Label)
		}
	}
}

func TestEveryPlayerConceptIsFoundByItsLabel(t *testing.T) {
	// duelist_cards.json names cards by label, and the deck the screen builds looks each one up in
	// the registry the rules built from that same file. A label that does not round-trip is four
	// cards missing from the deck.
	for _, a := range PlayerConcepts() {
		label := ConceptOf(a).Label
		got, ok := ConceptByKey(label)
		if !ok {
			t.Errorf("no concept registered under %q", label)
			continue
		}
		if got != a {
			t.Errorf("ConceptByKey(%q) = %v, want %v", label, got, a)
		}
	}

	if _, ok := ConceptByKey("Cleeve"); ok {
		t.Error("the registry answered to a typo — it must report failure so a bad deck fails loudly")
	}
}

func TestParseFormRoundTripsEveryForm(t *testing.T) {
	// The deck lists declare a form and `CheckCostTiers` holds it against the rules, so the join
	// has to work in both directions exactly as ParseAction's does.
	for _, f := range Forms() {
		got, ok := ParseForm(f.String())
		if !ok {
			t.Errorf("ParseForm(%q) failed for a form the rules define", f.String())
			continue
		}
		if got != f {
			t.Errorf("ParseForm(%q) = %v, want %v", f.String(), got, f)
		}
	}

	if _, ok := ParseForm("stabby"); ok {
		t.Error("ParseForm accepted a typo")
	}
	// And FormNone is not in Forms(), so it cannot be parsed into by accident — but it still
	// names itself, because a deck list writes "none" for the opponent's cards.
	if got := FormNone.String(); got != "none" {
		t.Errorf("FormNone is named %q, want %q", got, "none")
	}
}

// TestAWormsBoundsHold. The two per-card modifiers are the only way a card's numbers move, and
// both ends of each are a rule rather than a convenience.
func TestAWormsBoundsHold(t *testing.T) {
	// Cost floors at zero and does not go negative. A free card is bounded by the count cap
	// instead of the budget, which is the trade that was taken deliberately.
	cheap := Card{Concept: Strike, CostDelta: -99}
	if got := cheap.Cost(); got != 0 {
		t.Errorf("a card cheapened past zero costs %d, want 0", got)
	}

	// Nothing stops a blow outright, however many worms are stacked on a Defend.
	wall := Card{Concept: Defend, AmountPct: 10000}
	if got := wall.Amount(); got >= 100 {
		t.Errorf("a defence scaled up reduces by %d%%, and nothing may reach 100", got)
	}

	// An amount cannot be scaled away to nothing: a reward that left a card doing zero would be
	// a punishment wearing a gift's clothes.
	crushed := Card{Concept: Prepare, AmountPct: 1}
	if got := crushed.Amount(); got < 1 {
		t.Errorf("a scaled-down card banks %d", got)
	}

	// The zero value is unmodified, which is what keeps every existing Card literal working.
	plain := Card{Concept: Prepare}
	if plain.Amount() != ConceptOf(Prepare).Amount {
		t.Errorf("an unmodified card reports %d against the concept's %d",
			plain.Amount(), ConceptOf(Prepare).Amount)
	}
}

// TestTheLadderWalksItsOwnForm. Promote and demote are derived from what duelist_cards.json
// declares rather than from a table beside it, so this pins that the derivation finds the right
// neighbour and stops at both ends.
func TestTheLadderWalksItsOwnForm(t *testing.T) {
	up, ok := Neighbour(Jab, 1)
	if !ok {
		t.Fatal("Jab cannot be promoted, and it is a middle rung of its ladder")
	}
	if ConceptOf(up).Form != ConceptOf(Jab).Form {
		t.Errorf("promoting a %v produced a %v", ConceptOf(Jab).Form, ConceptOf(up).Form)
	}
	if ConceptOf(up).Tier() != ConceptOf(Jab).Tier()+1 {
		t.Errorf("promoting moved from tier %d to %d", ConceptOf(Jab).Tier(), ConceptOf(up).Tier())
	}

	if down, ok := Neighbour(up, -1); !ok || down != Jab {
		t.Errorf("demoting the promotion gave %v, want Jab", down)
	}

	// A Jab now demotes, because the ladder grew an end below it. That is the whole point of the
	// zero-copy rungs: the worm reaches a card it used to be refused on.
	if down, ok := Neighbour(Jab, -1); !ok || down != Poke {
		t.Errorf("demoting a Jab gave %v, want Poke", down)
	}

	// Both ends still stop, one rung further out than they used to. A card at the top of its form
	// cannot be promoted, and the screen asks before it offers so the player is never shown a worm
	// that would do nothing.
	if _, ok := Neighbour(Poke, -1); ok {
		t.Error("the bottom of a ladder was demoted")
	}
	if _, ok := Neighbour(Impale, 1); ok {
		t.Error("the top of a ladder was promoted")
	}

	// A plan has no form and therefore no ladder, and neither has any enemy card.
	if _, ok := Neighbour(Prepare, 1); ok {
		t.Error("a plan card was promoted")
	}
}
