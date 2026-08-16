package combat

import "testing"

// The 2026-08-15 card set: three attack families of three tiers, plus the three plans. Each of
// the plans gets the case that says what it does, and the ladder gets the structural cases that
// say the families are three ways of doing the same thing at the same prices — which is the claim
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
	a := duelist(10, 0, 100)
	b := duelist(10, 0, 100)

	events, aAfter, _ := resolve(a, b, PlainCards(Prepare), nil, 1)

	if !hasKind(events, KindGathered) {
		t.Fatal("no KindGathered event — the Prepare banked nothing")
	}
	if aAfter.BonusAP != prepareBonusAP {
		t.Errorf("BonusAP after a Prepare = %d, want %d", aAfter.BonusAP, prepareBonusAP)
	}
	if aAfter.ActionPoints() != baseActionPoints+prepareBonusAP {
		t.Errorf("next round's budget = %d, want %d", aAfter.ActionPoints(), baseActionPoints+prepareBonusAP)
	}
}

func TestPrepareIsWorthMoreThanItCosts(t *testing.T) {
	// Two for one is a deliberate profit. What Prepare actually costs is the card slot and the
	// action slot it takes out of the round it is played in, not the point on its face — so if the
	// bank ever stops beating the price, the card is pure loss and nobody would ever play it.
	if net := prepareBonusAP - costPrepare; net <= 0 {
		t.Errorf("Prepare nets %+d AP; a bank that does not profit is a card nobody plays", net)
	}
}

// Defend is the whole defensive vocabulary as of 2026-08-15. Under one blow per turn it takes half
// of what arrives and is then spent.
func TestDefendHalvesTheBlowAndIsSpent(t *testing.T) {
	a := duelist(10, 0, 500)
	b := duelist(10, 0, 500)

	// B defends first. B acts second, so its Defend is standing when A's next turn arrives.
	// A Smash and a Strike are different concepts and form no hand, so the blow is the High Card
	// — the Smash alone — and the arithmetic is about the Defend rather than about a multiplier.
	_, a, b = resolve(a, b, nil, PlainCards(Defend), 1)
	events, _, bAfter := resolve(a, b, PlainCards(Smash, Strike), nil, 2)

	if !hasKind(events, KindNegated) {
		t.Fatal("no KindNegated event — the Defend did not apply")
	}
	if want := 500 - Smash.Damage(10)*(100-defendReductionPct)/100; bAfter.CurrentLife != want {
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
	a := duelist(10, 0, 100)
	b := duelist(10, 0, 100)

	events, a1, _ := resolve(a, b, PlainCards(Plan), nil, 1)

	if !hasKind(events, KindDrew) {
		t.Fatal("no KindDrew event — nothing tells the screen to draw")
	}
	if a1.BonusDraw != planDrawCards {
		t.Errorf("BonusDraw = %d after a Plan, want %d", a1.BonusDraw, planDrawCards)
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
	if a3.BonusDraw != 2*planDrawCards {
		t.Errorf("two Plans banked %d, want %d", a3.BonusDraw, 2*planDrawCards)
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
	a := duelist(10, 0, 100)
	b := duelist(10, 0, 100)

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
	for _, k := range []ActionKind{Defend, Defend} {
		d = d.raiseDefend(k)
	}

	got := ClearDefenses(d)

	if got.DefendCount != 0 || got.Defends != ([maxPendingDefends]PendingDefend{}) {
		t.Errorf("ClearDefenses left something standing: %+v", got)
	}
}

func TestTheAttackLadderIsThreeFamiliesByThreeTiers(t *testing.T) {
	// One concept per family per tier, and **the tiers are identical across the families** — same
	// cost, same damage. That is the structural claim MECHANICS.md makes about the deck: a family
	// is which pair you are building, never a better or worse way to build one. It is also the
	// thing that quietly breaks the first time somebody makes a Cleave hit harder than a Lunge.
	//
	// It also catches a concept falling through Cost()'s default arm, which returns a mid-tier
	// price — a mistake that would otherwise hide behind an entirely plausible number.
	const dmg = 10

	attackFamilies := []Family{FamilyStab, FamilySlash, FamilyCrush}
	tiers := map[Family]map[int]ActionKind{}

	for _, a := range AllActions {
		fam := a.Family()
		if a.Category() != CategoryAttack || fam == FamilyNone {
			continue
		}
		if a.Cost() < 1 || a.Cost() > 3 {
			t.Errorf("%v costs %d, outside the 1-3 tiers the ladder is built on", a, a.Cost())
		}
		if tiers[fam] == nil {
			tiers[fam] = map[int]ActionKind{}
		}
		if other, taken := tiers[fam][a.Cost()]; taken {
			t.Errorf("%v and %v are both %v at %d AP — the ladder holds one concept per rung", other, a, fam, a.Cost())
		}
		tiers[fam][a.Cost()] = a
	}

	for _, fam := range attackFamilies {
		if len(tiers[fam]) != 3 {
			t.Errorf("%v has %d of 3 tiers filled", fam, len(tiers[fam]))
		}
	}

	// Every family's rung deals what Stab's rung deals.
	for tier := 1; tier <= 3; tier++ {
		want, ok := tiers[FamilyStab][tier]
		if !ok {
			continue
		}
		for _, fam := range attackFamilies[1:] {
			got, ok := tiers[fam][tier]
			if !ok {
				continue
			}
			if got.Damage(dmg) != want.Damage(dmg) {
				t.Errorf("at %d AP, %v deals %d and %v deals %d — the families must ladder identically",
					tier, got, got.Damage(dmg), want, want.Damage(dmg))
			}
		}
	}
}

func TestPlanCardsDealNothingAndAttacksDoNot(t *testing.T) {
	// The two categories are what a card *is*, so each has to hold on its own side of the line. A
	// plan that dealt damage would be an attack wearing the wrong verb in the feed.
	const dmg = 10
	for _, a := range AllActions {
		switch a.Category() {
		case CategoryPlan:
			if d := a.Damage(dmg); d != 0 {
				t.Errorf("%v is a plan and deals %d damage", a, d)
			}
		case CategoryAttack:
			if d := a.Damage(dmg); d <= 0 {
				t.Errorf("%v is an attack and deals %d damage", a, d)
			}
		}
	}
}

func TestEveryPlanIsInThePlanFamilyAndViceVersa(t *testing.T) {
	// Category has two values and is derivable from the family, which is exactly why the two must
	// not be able to disagree: the card face says the family and the feed says the category, and a
	// card whose corner and verb told different stories would be the screen contradicting itself.
	for _, a := range AllActions {
		if (a.Family() == FamilyPlan) != (a.Category() == CategoryPlan) {
			t.Errorf("%v is family %v but category %v", a, a.Family(), a.Category())
		}
	}
}

func TestOnlyTheOpponentsCardsHaveNoFamily(t *testing.T) {
	// **FamilyNone is a real answer, not a fallthrough.** It means "not in the player's deck", and
	// the two enemy concepts are the only cards it may apply to — a player's card landing there
	// would draw with a blank corner and be excluded from the deck panel's sort.
	familyless := map[ActionKind]bool{Attack: true, Heavy: true}
	for _, a := range AllActions {
		if a.Family() == FamilyNone && !familyless[a] {
			t.Errorf("%v has no family; only the opponent's cards may", a)
		}
		if a.Family() != FamilyNone && familyless[a] {
			t.Errorf("%v is the opponent's and should belong to no family, but says %v", a, a.Family())
		}
	}
}

func TestParseActionRoundTripsEveryConcept(t *testing.T) {
	// duelist_cards.json names concepts by string, so this is the join between the deck data and
	// the rules. A concept whose name does not round-trip is five cards missing from the deck.
	for _, a := range AllActions {
		got, ok := ParseAction(a.String())
		if !ok {
			t.Errorf("ParseAction(%q) failed for a concept the rules define", a.String())
			continue
		}
		if got != a {
			t.Errorf("ParseAction(%q) = %v, want %v", a.String(), got, a)
		}
	}

	if _, ok := ParseAction("Cleeve"); ok {
		t.Error("ParseAction accepted a typo — it must report failure so a bad deck fails loudly")
	}
}

func TestParseFamilyRoundTripsEveryFamily(t *testing.T) {
	// The deck lists declare a family and `CheckCostTiers` holds it against the rules, so the join
	// has to work in both directions exactly as ParseAction's does.
	for _, f := range Families() {
		got, ok := ParseFamily(f.String())
		if !ok {
			t.Errorf("ParseFamily(%q) failed for a family the rules define", f.String())
			continue
		}
		if got != f {
			t.Errorf("ParseFamily(%q) = %v, want %v", f.String(), got, f)
		}
	}

	if _, ok := ParseFamily("stabby"); ok {
		t.Error("ParseFamily accepted a typo")
	}
	// And FamilyNone is not in Families(), so it cannot be parsed into by accident — but it still
	// names itself, because a deck list writes "none" for the opponent's cards.
	if got := FamilyNone.String(); got != "none" {
		t.Errorf("FamilyNone is named %q, want %q", got, "none")
	}
}
