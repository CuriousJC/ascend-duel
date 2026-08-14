package combat

import "testing"

// combosFired returns the combos one side formed, in the order they fired.
func combosFired(events []Event, by Side) []ComboID {
	var out []ComboID
	for _, e := range events {
		if e.Kind == KindCombo && e.Side == by {
			out = append(out, e.Combo)
		}
	}
	return out
}

// staggeredActions returns the actions one side lost to a stagger.
func staggeredActions(events []Event, by Side) []ActionKind {
	var out []ActionKind
	for _, e := range events {
		if e.Kind == KindStaggered && e.Side == by {
			out = append(out, e.Action)
		}
	}
	return out
}

// sideActions returns the actions one side actually took.
func sideActions(events []Event, by Side) []ActionKind {
	var out []ActionKind
	for _, e := range events {
		if e.Kind == KindAction && e.Side == by {
			out = append(out, e.Action)
		}
	}
	return out
}

// --- matching ---------------------------------------------------------------------------

func TestThreeAttacksFormAFlurry(t *testing.T) {
	a, b := duelist(10, 0, 100), duelist(10, 0, 100)

	events, _, _ := ResolveRound(a, b, PlainCards(Strike, Strike, Strike), nil, 1)

	got := combosFired(events, SideA)
	if len(got) != 1 || got[0] != FlurryID(Strike) {
		t.Fatalf("three attacks should form one Flurry, got %v", got)
	}
}

func TestTwoAttacksFormNothing(t *testing.T) {
	a, b := duelist(10, 0, 100), duelist(10, 0, 100)

	events, _, _ := ResolveRound(a, b, PlainCards(Strike, Heavy), nil, 1)

	if got := combosFired(events, SideA); len(got) != 0 {
		t.Fatalf("two attacks should form nothing, got %v", got)
	}
}

// Three attacks of *different* cards form nothing. The family is per card on purpose: three
// Strikes is a deck you assembled, three attacks is whatever you happened to draw.
func TestThreeDifferentAttacksFormNothing(t *testing.T) {
	a, b := duelist(10, 0, 100), duelist(10, 0, 100)

	events, _, _ := ResolveRound(a, b, PlainCards(Jab, Strike, Heavy), nil, 1)

	if got := combosFired(events, SideA); len(got) != 0 {
		t.Fatalf("Jab+Strike+Heavy is not a run of one card, got %v", got)
	}
}

// Every attack card carries the family, whether or not the budget makes it reachable. Heavy
// is 4 AP, so three of them is 12 and five is 20 — near enough impossible today, and
// deliberately still a rule so that engine-building has something to aim at.
func TestEveryAttackCardHasAFlurryAndAnOnslaught(t *testing.T) {
	for _, card := range AllActions {
		if card.Category() != CategoryAttack {
			if _, ok := ComboByID(FlurryID(card)); ok {
				t.Errorf("%v is not an attack and should carry no flurry", card)
			}
			continue
		}

		for _, tc := range []struct {
			id   ComboID
			n    int
			what string
		}{
			{FlurryID(card), flurryRun, "flurry"},
			{OnslaughtID(card), onslaughtRun, "onslaught"},
		} {
			combo, ok := ComboByID(tc.id)
			if !ok {
				t.Errorf("%v has no %s", card, tc.what)
				continue
			}
			if len(combo.Run) != tc.n {
				t.Errorf("%s: want a run of %d, got %d", combo.Name, tc.n, len(combo.Run))
			}

			a, b := duelist(10, 0, 5000), duelist(10, 0, 5000)
			queue := make([]Card, tc.n)
			for i := range queue {
				queue[i] = Plain(card)
			}

			events, _, _ := ResolveRound(a, b, queue, nil, 1)
			got := combosFired(events, SideA)
			if len(got) != 1 || got[0] != tc.id {
				t.Errorf("%d %v should form %s, got %v", tc.n, card, combo.Name, got)
			}
		}
	}
}

// The names are generated from the card, so a new attack card names its own combos.
func TestCombosAreNamedForTheCardTheyAreBuiltOn(t *testing.T) {
	for _, tc := range []struct {
		id   ComboID
		want string
	}{
		{FlurryID(Strike), "Strike Flurry"},
		{OnslaughtID(Strike), "Strike Onslaught"},
		{FlurryID(Heavy), "Heavy Flurry"},
		{OnslaughtID(Heavy), "Heavy Onslaught"},
		{FlurryID(Jab), "Jab Flurry"},
	} {
		c, ok := ComboByID(tc.id)
		if !ok {
			t.Errorf("no combo for %d", tc.id)
			continue
		}
		if c.Name != tc.want {
			t.Errorf("want %q, got %q", tc.want, c.Name)
		}
	}
}

// A combo's ID is derived from its card rather than its place in a list, so inserting a card
// cannot renumber the combos already discovered on a profile.
func TestComboIDsDoNotCollide(t *testing.T) {
	seen := map[ComboID]string{}
	for _, c := range Combos() {
		if prev, dup := seen[c.ID]; dup {
			t.Errorf("ID %d is used by both %q and %q", c.ID, prev, c.Name)
		}
		seen[c.ID] = c.Name
	}
}

func TestFourAttacksFormOneFlurryNotTwo(t *testing.T) {
	a, b := duelist(10, 0, 100), duelist(10, 0, 100)

	events, _, _ := ResolveRound(a, b, PlainCards(Strike, Strike, Strike, Strike), nil, 1)

	if got := combosFired(events, SideA); len(got) != 1 {
		t.Fatalf("a card may form at most one combo, so four attacks is one Flurry; got %v", got)
	}
}

func TestFiveAttacksFormOnslaughtRatherThanFlurry(t *testing.T) {
	a, b := duelist(10, 0, 200), duelist(10, 0, 200)

	events, _, _ := ResolveRound(a, b, PlainCards(Jab, Jab, Jab, Jab, Jab), nil, 1)

	got := combosFired(events, SideA)
	if len(got) != 1 || got[0] != OnslaughtID(Jab) {
		t.Fatalf("longest run should win, so five attacks is Onslaught; got %v", got)
	}
}

// The whole reason matching happens on the resolved order: the queue here is interrupted by a
// Dodge, but phases move every defense to the end of the turn, so three attacks still land
// back to back. Matching the queue would have missed it and the Resolution pane would have
// shown a combo the engine did not score.
func TestCombosMatchTheResolvedOrderNotTheQueuedOne(t *testing.T) {
	a, b := duelist(10, 0, 100), duelist(10, 0, 100)

	events, _, _ := ResolveRound(a, b, PlainCards(Strike, Dodge, Strike, Strike), nil, 1)

	if got := combosFired(events, SideA); len(got) != 1 || got[0] != FlurryID(Strike) {
		t.Fatalf("the Dodge regroups to the end, leaving three consecutive attacks; got %v", got)
	}
}

// MatchCombos is what the screen calls while the player is still planning, and it has to
// agree with the engine or the preview lies.
func TestMatchCombosAgreesWithWhatTheEnginePlays(t *testing.T) {
	queue := PlainCards(Strike, Dodge, Strike, Strike)

	preview := MatchCombos(SideA, queue)
	if len(preview) != 1 || preview[0].ID != FlurryID(Strike) {
		t.Fatalf("preview should see one Flurry, got %v", preview)
	}
	if preview[0].Side != SideA {
		t.Fatalf("preview should carry the side it was asked about, got %v", preview[0].Side)
	}

	a, b := duelist(10, 0, 100), duelist(10, 0, 100)
	events, _, _ := ResolveRound(a, b, queue, nil, 1)

	played := combosFired(events, SideA)
	if len(played) != len(preview) || played[0] != preview[0].ID {
		t.Fatalf("preview %v disagrees with what was played %v", preview, played)
	}
}

func TestLongestRunWinsAtTheSamePosition(t *testing.T) {
	short := Combo{ID: ComboID(90), Name: "short", Run: []Step{Exactly(Strike), Exactly(Strike)}}
	long := Combo{ID: ComboID(91), Name: "long", Run: []Step{Exactly(Strike), Exactly(Strike), Exactly(Strike)}}

	// Declared shortest-first, so a matcher that simply took the first hit would be wrong.
	turn := appendTurn(nil, SideA, PlainCards(Strike, Strike, Strike))
	hits := matchSlots(turn, []Combo{short, long})

	if len(hits) != 1 || hits[0].ID != long.ID {
		t.Fatalf("longest run should win regardless of table order, got %v", hits)
	}
}

func TestAnExactCardPatternDoesNotMatchItsCategory(t *testing.T) {
	only := Combo{ID: ComboID(92), Name: "two strikes", Run: []Step{Exactly(Strike), Exactly(Strike)}}

	turn := appendTurn(nil, SideA, PlainCards(Strike, Heavy))
	if hits := matchSlots(turn, []Combo{only}); len(hits) != 0 {
		t.Fatalf("Exactly(Strike) should not match a Heavy, got %v", hits)
	}
}

// --- stagger ----------------------------------------------------------------------------

func TestAFlurryCostsTheOpponentTheirNextAction(t *testing.T) {
	a, b := duelist(10, 0, 300), duelist(10, 0, 300)

	events, _, _ := ResolveRound(a, b,
		PlainCards(Strike, Strike, Strike),
		PlainCards(Jab, Strike), 1)

	if lost := staggeredActions(events, SideB); len(lost) != 1 || lost[0] != Jab {
		t.Fatalf("B should lose exactly its first action, got %v", lost)
	}
	if took := sideActions(events, SideB); len(took) != 1 || took[0] != Strike {
		t.Fatalf("B should still take the rest of its turn, got %v", took)
	}
}

func TestOnslaughtTakesTheOpponentsWholeTurn(t *testing.T) {
	a, b := duelist(10, 0, 300), duelist(10, 0, 300)

	events, _, _ := ResolveRound(a, b,
		PlainCards(Jab, Jab, Jab, Jab, Jab),
		PlainCards(Strike, Strike, Guard), 1)

	if lost := staggeredActions(events, SideB); len(lost) != 3 {
		t.Fatalf("Onslaught should take all three of B's actions, got %v", lost)
	}
	if took := sideActions(events, SideB); len(took) != 0 {
		t.Fatalf("B should take no action at all, got %v", took)
	}
}

// Stagger comes off the *front* of the turn, which under phases is the setup phase. Losing a
// Gather rather than an attack is a real consequence and worth pinning: it means a staggered
// duelist is poorer next round as well as slower this one.
func TestStaggerTakesTheFrontOfTheTurnIncludingSetup(t *testing.T) {
	a, b := duelist(10, 0, 300), duelist(10, 0, 300)

	events, _, bAfter := ResolveRound(a, b,
		PlainCards(Strike, Strike, Strike),
		PlainCards(Gather, Strike), 1)

	if lost := staggeredActions(events, SideB); len(lost) != 1 || lost[0] != Gather {
		t.Fatalf("the Gather resolves first and so is the action lost, got %v", lost)
	}
	if bAfter.BonusAP != 0 {
		t.Fatalf("a staggered Gather banks nothing, got BonusAP %d", bAfter.BonusAP)
	}

	// The same round without the Flurry, to show the Gather would otherwise have paid.
	_, _, unstaggered := ResolveRound(a, b,
		PlainCards(Strike, Strike),
		PlainCards(Gather, Strike), 1)
	if unstaggered.BonusAP != gatherBonusAP {
		t.Fatalf("without the stagger the Gather should bank %d, got %d",
			gatherBonusAP, unstaggered.BonusAP)
	}
}

// Side B acts last, so a combo B forms has no turn left to bite in this round. It carries on
// the Duelist and lands next round instead. That delay is the price of phase resolution, and
// it is the one asymmetry the mechanic has.
func TestSideBsFlurryLandsInTheFollowingRound(t *testing.T) {
	a, b := duelist(10, 0, 300), duelist(10, 0, 300)

	events, aAfter, bAfter := ResolveRound(a, b,
		PlainCards(Gather),
		PlainCards(Strike, Strike, Strike), 1)

	if got := combosFired(events, SideB); len(got) != 1 || got[0] != FlurryID(Strike) {
		t.Fatalf("B should form a Flurry, got %v", got)
	}
	if lost := staggeredActions(events, SideA); len(lost) != 0 {
		t.Fatalf("A already acted, so nothing can be taken from it this round, got %v", lost)
	}
	if aAfter.Staggered != 1 {
		t.Fatalf("the stagger should be held on A for next round, got %d", aAfter.Staggered)
	}

	events2, _, _ := ResolveRound(aAfter, bAfter, PlainCards(Jab, Strike), nil, 2)
	if lost := staggeredActions(events2, SideA); len(lost) != 1 || lost[0] != Jab {
		t.Fatalf("A should lose its first action next round, got %v", lost)
	}
}

func TestAStaggerIsSpentOnceAndDoesNotLinger(t *testing.T) {
	a, b := duelist(10, 0, 300), duelist(10, 0, 300)

	_, aAfter, bAfter := ResolveRound(a, b,
		PlainCards(Gather),
		PlainCards(Strike, Strike, Strike), 1)

	_, aAfter2, _ := ResolveRound(aAfter, bAfter, PlainCards(Jab, Strike), nil, 2)
	if aAfter2.Staggered != 0 {
		t.Fatalf("the stagger should be consumed by the turn it hit, got %d", aAfter2.Staggered)
	}

	events3, _, _ := ResolveRound(aAfter2, bAfter, PlainCards(Jab, Strike), nil, 3)
	if lost := staggeredActions(events3, SideA); len(lost) != 0 {
		t.Fatalf("round three should be unaffected, got %v", lost)
	}
}

// A combo scored off cards a stagger deleted would let a staggered duelist stagger back with
// a turn it never took.
func TestStaggeredActionsCannotFormACombo(t *testing.T) {
	a := duelist(10, 0, 300)
	b := duelist(10, 0, 300)
	b.Staggered = 3

	events, _, _ := ResolveRound(a, b,
		nil,
		PlainCards(Strike, Strike, Strike, Strike, Strike), 1)

	if got := combosFired(events, SideB); len(got) != 0 {
		t.Fatalf("only two of B's five attacks happened, so nothing should form; got %v", got)
	}
	if took := sideActions(events, SideB); len(took) != 2 {
		t.Fatalf("B should take exactly two actions, got %v", took)
	}
}

func TestStaggerNeverTakesMoreThanTheTurnHolds(t *testing.T) {
	a := duelist(10, 0, 300)
	b := duelist(10, 0, 300)
	b.Staggered = StaggerAll

	events, _, bAfter := ResolveRound(a, b, nil, PlainCards(Strike), 1)

	if lost := staggeredActions(events, SideB); len(lost) != 1 {
		t.Fatalf("a one-action turn can only lose one action, got %v", lost)
	}
	if bAfter.Staggered != 0 {
		t.Fatalf("StaggerAll is spent by the turn it lands on, got %d", bAfter.Staggered)
	}
}

func TestStaggerIsSymmetric(t *testing.T) {
	// The same three attacks, formed by each side in turn, must produce the same stagger.
	a, b := duelist(10, 0, 300), duelist(10, 0, 300)

	_, _, bAfterA := ResolveRound(a, b, PlainCards(Strike, Strike, Strike), nil, 1)
	_, aAfterB, _ := ResolveRound(a, b, nil, PlainCards(Strike, Strike, Strike), 1)

	// B was hit inside the round and spent it there; A holds it for the next one. Both lose
	// exactly one action to it, which is the symmetry that matters.
	if bAfterA.Staggered != 0 {
		t.Fatalf("B's stagger should have been spent in-round, got %d", bAfterA.Staggered)
	}
	if aAfterB.Staggered != 1 {
		t.Fatalf("A should carry one staggered action into the next round, got %d", aAfterB.Staggered)
	}
}

// --- effects other than stagger ---------------------------------------------------------

// Neither shipping combo uses a damage multiplier, so this drives one through the whole
// engine with an injected table rather than leaving that path unexercised.
//
// **A multiplier boosts what comes after the run, not the run itself** — the consequence of
// firing a combo on completion rather than on its first card. See playTurn.
func TestADamageMultiplierBoostsWhatFollowsTheCombo(t *testing.T) {
	table := []Combo{{
		ID:     ComboID(93),
		Name:   "double up",
		Run:    []Step{Exactly(Strike), Exactly(Strike)},
		Effect: Effect{DamageNum: 2, DamageDen: 1},
	}}

	a, b := duelist(10, 0, 100), duelist(10, 0, 100)
	_, _, bAfter := resolveRound(a, b, PlainCards(Strike, Strike, Strike), nil, 1, table)

	// The first two Strikes form the combo and land for 10 each. The third is doubled to 20.
	if got := 100 - bAfter.CurrentLife; got != 40 {
		t.Fatalf("want 10 + 10 + 20 = 40 damage, got %d", got)
	}
}

// A Guard should be worth the same fraction against a boosted blow as against a plain one.
// The numbers are chosen so the two orderings genuinely differ: at Str 3, boosting to 4 then
// halving gives 2, while halving to 1 then boosting gives 1.
func TestAMultiplierAppliesBeforeAGuardHalvesIt(t *testing.T) {
	table := []Combo{{
		ID:     ComboID(94),
		Name:   "half again",
		Run:    []Step{Exactly(Strike), Exactly(Strike)},
		Effect: Effect{DamageNum: 3, DamageDen: 2},
	}}

	a := duelist(3, 0, 100)
	b := duelist(3, 0, 100)
	b.Guarded = true

	_, _, bAfter := resolveRound(a, b, PlainCards(Strike, Strike, Strike), nil, 1, table)

	// Two unboosted Strikes at 3 are halved to 1 each. The third is boosted to 4 first and
	// halved to 2 — where halving first would have given 1, and 1 + 1 + 1 = 3.
	if got := 100 - bAfter.CurrentLife; got != 4 {
		t.Fatalf("want 1 + 1 + 2 = 4 damage, got %d", got)
	}
}

// --- when a combo fires -------------------------------------------------------------------

// The combo is announced *after* the cards that earned it, which is how it reads on screen:
// three strikes land, and then the flurry is recognised.
func TestAComboIsAnnouncedAfterTheCardsThatFormedIt(t *testing.T) {
	a, b := duelist(10, 0, 300), duelist(10, 0, 300)

	events, _, _ := ResolveRound(a, b, PlainCards(Strike, Strike, Strike), nil, 1)

	actionsBefore, comboAt := 0, -1
	for i, e := range events {
		switch e.Kind {
		case KindAction:
			if comboAt < 0 {
				actionsBefore++
			}
		case KindCombo:
			if comboAt < 0 {
				comboAt = i
			}
		}
	}

	if comboAt < 0 {
		t.Fatal("no combo fired")
	}
	if actionsBefore != 3 {
		t.Fatalf("all three attacks should resolve before the combo is announced, got %d", actionsBefore)
	}
}

// **A run that never finishes pays nothing.** Matching happens up front against the whole
// turn, so a turn cut short used to fire a combo off cards that never resolved. Firing on
// completion makes that impossible rather than needing a check.
func TestAComboCutShortByDeathDoesNotFire(t *testing.T) {
	// B is holding two Ripostes, each of which negates a Strike and hits back for Str/2 = 2.
	// A has 4 life, so the second counter kills it partway through a three-Strike run.
	a := duelist(10, 0, 4)
	b := duelist(4, 0, 300)
	b = b.raiseDefend(Riposte).raiseDefend(Riposte)

	events, aAfter, bAfter := ResolveRound(a, b, PlainCards(Strike, Strike, Strike), nil, 1)

	if aAfter.Alive() {
		t.Fatal("this fixture is meant to kill side A partway through its run")
	}
	if got := combosFired(events, SideA); len(got) != 0 {
		t.Fatalf("a run cut short should form nothing, got %v", got)
	}
	if bAfter.Staggered != 0 {
		t.Fatalf("and it should certainly not stagger anyone, got %d", bAfter.Staggered)
	}
}

// Points cannot be handed back into the round that committed them, so a combo that banks AP
// rides the same path a Gather does and arrives in the round after.
func TestComboBankedPointsArriveInTheFollowingRound(t *testing.T) {
	table := []Combo{{
		ID:     ComboID(95),
		Name:   "second wind",
		Run:    []Step{Exactly(Jab), Exactly(Jab)},
		Effect: Effect{BankAP: 3},
	}}

	a, b := duelist(10, 0, 100), duelist(10, 0, 100)
	before := a.ActionPoints()

	_, aAfter, _ := resolveRound(a, b, PlainCards(Jab, Jab), nil, 1, table)

	if aAfter.BonusAP != 3 {
		t.Fatalf("want 3 points banked, got %d", aAfter.BonusAP)
	}
	if aAfter.ActionPoints() != before+3 {
		t.Fatalf("want %d points next round, got %d", before+3, aAfter.ActionPoints())
	}
}

func TestACombosEffectsDoNotLeakIntoTheOpponentsTurn(t *testing.T) {
	table := []Combo{{
		ID:     ComboID(96),
		Name:   "double up",
		Run:    []Step{Exactly(Strike), Exactly(Strike)},
		Effect: Effect{DamageNum: 2, DamageDen: 1},
	}}

	a, b := duelist(10, 0, 200), duelist(10, 0, 200)
	_, aAfter, _ := resolveRound(a, b, PlainCards(Strike, Strike), PlainCards(Strike), 1, table)

	// B's single Strike is not part of A's combo and must land for its plain 10.
	if got := 200 - aAfter.CurrentLife; got != 10 {
		t.Fatalf("B's attack should be unboosted: want 10, got %d", got)
	}
}

// TestResolutionOrderIsWhatResolveRoundPlays pins that the pane and the engine agree on the
// order. Stagger weakens that: a staggered slot is drawn as a row but never resolves, so the
// actions alone no longer account for every row.
//
// **This is the invariant the screen's highlight now rests on** — CombatScene.currentSlot
// counts one beat per slot, taken or lost, and would light the wrong card for the rest of the
// round if a slot went unaccounted for.
func TestEverySlotIsEitherTakenOrStaggered(t *testing.T) {
	a, b := duelist(10, 0, 500), duelist(10, 0, 500)
	aPlan := PlainCards(Strike, Strike, Strike)
	bPlan := PlainCards(Gather, Jab, Strike, Dodge)

	events, _, _ := ResolveRound(a, b, aPlan, bPlan, 1)

	order := ResolutionOrder(aPlan, bPlan)

	var beats []ActionKind
	for _, e := range events {
		if e.Kind == KindAction || e.Kind == KindStaggered {
			beats = append(beats, e.Action)
		}
	}

	if len(beats) != len(order) {
		t.Fatalf("every slot needs exactly one beat: %d slots, %d beats", len(order), len(beats))
	}
	for i, slot := range order {
		if beats[i] != slot.Card.Action {
			t.Fatalf("beat %d is %v, but slot %d is %v", i, beats[i], i, slot.Card.Action)
		}
	}

	// And the round really did stagger something, or this proves nothing.
	if lost := staggeredActions(events, SideB); len(lost) == 0 {
		t.Fatal("this fixture is meant to stagger side B")
	}
}

// --- housekeeping -------------------------------------------------------------------------

func TestEveryComboIsNamedAndFindable(t *testing.T) {
	for _, c := range Combos() {
		if c.Name == "" {
			t.Errorf("combo %d has no name", c.ID)
		}
		if len(c.Run) == 0 {
			t.Errorf("combo %q matches nothing", c.Name)
		}
		if c.ID == ComboNone {
			t.Errorf("combo %q uses the zero ID, which means 'no combo'", c.Name)
		}
		if found, ok := ComboByID(c.ID); !ok || found.Name != c.Name {
			t.Errorf("combo %q is not findable by its own ID", c.Name)
		}
	}
}

func TestComboByIDReportsAnUnknownIDRatherThanInventingOne(t *testing.T) {
	if _, ok := ComboByID(ComboID(9999)); ok {
		t.Fatal("an unknown ID should not resolve to a combo")
	}
}

func TestCombosDoNotBreakDeterminism(t *testing.T) {
	a, b := duelist(10, 0, 300), duelist(10, 0, 300)
	actions := PlainCards(Strike, Strike, Strike)

	e1, a1, b1 := ResolveRound(a, b, actions, PlainCards(Jab, Strike), 1)
	e2, a2, b2 := ResolveRound(a, b, actions, PlainCards(Jab, Strike), 1)

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

// TestComboEventNamesTheCardsThatFormedIt pins the two fields the combat screen brackets a
// combo with. Deriving them from the pattern instead would be a second matcher, and matching
// is greedy and longest-first — so the derived answer and the real one disagree exactly when
// a longer combo swallowed a shorter one.
func TestComboEventNamesTheCardsThatFormedIt(t *testing.T) {
	// Three Strikes, then a Jab that is not part of the run.
	log, _, _ := ResolveRound(
		Duelist{MaxLife: 200, CurrentLife: 200, Str: 5, Spd: 20},
		Duelist{MaxLife: 200, CurrentLife: 200, Str: 5, Spd: 20},
		PlainCards(Strike, Strike, Strike, Jab),
		nil,
		1,
	)

	var combos []Event
	for _, e := range log {
		if e.Kind == KindCombo {
			combos = append(combos, e)
		}
	}
	if len(combos) != 1 {
		t.Fatalf("%d combos fired, want the one Strike Flurry", len(combos))
	}

	c := combos[0]
	if c.ComboStart != 0 || c.ComboLength != flurryRun {
		t.Errorf("the flurry says it spans [%d,%d), want [0,%d)",
			c.ComboStart, c.ComboStart+c.ComboLength, flurryRun)
	}

	// And the span has to name real actions: the run must sit inside the turn that played.
	var played int
	for _, e := range log {
		if e.Kind == KindAction && e.Side == SideA {
			played++
		}
	}
	if c.ComboStart+c.ComboLength > played {
		t.Errorf("the combo spans past the end of a turn that played %d actions", played)
	}
}

// TestLongerComboReportsItsOwnSpan is the case a screen deriving the span would get wrong:
// five Strikes are one Onslaught, not a Flurry plus leftovers.
func TestLongerComboReportsItsOwnSpan(t *testing.T) {
	log, _, _ := ResolveRound(
		Duelist{MaxLife: 400, CurrentLife: 400, Str: 5, Spd: 40},
		Duelist{MaxLife: 400, CurrentLife: 400, Str: 5, Spd: 40},
		PlainCards(Strike, Strike, Strike, Strike, Strike),
		nil,
		1,
	)

	for _, e := range log {
		if e.Kind != KindCombo {
			continue
		}
		if e.Combo != OnslaughtID(Strike) {
			t.Errorf("five strikes fired combo %d, want the Strike Onslaught", e.Combo)
		}
		if e.ComboLength != onslaughtRun {
			t.Errorf("the onslaught claims %d cards, want %d", e.ComboLength, onslaughtRun)
		}
		return
	}
	t.Fatal("five strikes fired no combo at all")
}

// The element axis on a Step, added 2026-08-12 with elements themselves. Nothing in the
// shipping table uses it yet — the flurry family matches concepts and ignores colour — so
// these drive it through matchSlots with tables of their own, which is exactly what the table
// parameter on matchSlots exists for.

func TestAFlurryDoesNotCareWhatColourItIs(t *testing.T) {
	// **The default has to stay colour-blind.** A Strike Flurry is three Strikes however they
	// are painted; requiring one element would silently turn every flurry in the game into an
	// element combo, and a 60-card deck holds only 5 Strikes of each colour.
	a, b := duelist(10, 60, 5000), duelist(10, 60, 5000)

	events, _, _ := ResolveRound(a, b, []Card{
		Of(Strike, Fire), Of(Strike, Ice), Of(Strike, Earth),
	}, nil, 1)

	got := combosFired(events, SideA)
	if len(got) != 1 || got[0] != FlurryID(Strike) {
		t.Errorf("three differently coloured Strikes fired %v, want one Strike Flurry", got)
	}
}

func TestOfElementMatchesAnyConceptOfOneColour(t *testing.T) {
	// The shape the five-of-a-colour combo is made of: five steps that name a colour and
	// nothing else.
	table := []Combo{{
		ID:   ComboID(300),
		Name: "monochrome",
		Run:  []Step{OfElement(Ice), OfElement(Ice), OfElement(Ice)},
	}}

	hits := matchSlots(appendTurn(nil, SideA, []Card{
		Of(Gather, Ice), Of(Strike, Ice), Of(Brace, Ice),
	}), table)
	if len(hits) != 1 {
		t.Fatalf("three ice cards of three concepts fired %d combos, want 1", len(hits))
	}

	// One off-colour card breaks the run outright, which is the property that makes the
	// five-of-a-colour combo an all-in round rather than a near-miss.
	hits = matchSlots(appendTurn(nil, SideA, []Card{
		Of(Gather, Ice), Of(Strike, Fire), Of(Brace, Ice),
	}), table)
	if len(hits) != 0 {
		t.Errorf("a run broken by one fire card still fired %v", hits)
	}
}

func TestWithElementPinsAStepToOneColour(t *testing.T) {
	// `Exactly(Strike).WithElement(Ice)` is an ice Strike and nothing else. Both halves have to
	// bite: a fire Strike fails the colour, an ice Jab fails the concept.
	table := []Combo{{
		ID:   ComboID(301),
		Name: "cold pair",
		Run:  []Step{Exactly(Strike).WithElement(Ice), Exactly(Strike).WithElement(Ice)},
	}}

	for _, tc := range []struct {
		what string
		turn []Card
		want int
	}{
		{"two ice Strikes", []Card{Of(Strike, Ice), Of(Strike, Ice)}, 1},
		{"two fire Strikes", []Card{Of(Strike, Fire), Of(Strike, Fire)}, 0},
		{"two ice Jabs", []Card{Of(Jab, Ice), Of(Jab, Ice)}, 0},
		{"one of each", []Card{Of(Strike, Ice), Of(Strike, Fire)}, 0},
		{"two plain Strikes", PlainCards(Strike, Strike), 0},
	} {
		if got := len(matchSlots(appendTurn(nil, SideA, tc.turn), table)); got != tc.want {
			t.Errorf("%s fired %d combos, want %d", tc.what, got, tc.want)
		}
	}
}

func TestASequenceComboReadsTheOrderTheCardsWereQueuedIn(t *testing.T) {
	// **This is what earns drag-to-reorder its place back.** Phases regroup a turn by category,
	// but within a category the queued order survives — so an ice Strike before a fire Strike is
	// a different round from the reverse, off the same two cards.
	table := []Combo{{
		ID:   ComboID(302),
		Name: "burnt icecube",
		Run:  []Step{Exactly(Strike).WithElement(Ice), Exactly(Strike).WithElement(Fire)},
	}}

	forwards := matchSlots(appendTurn(nil, SideA,
		[]Card{Of(Strike, Ice), Of(Strike, Fire)}), table)
	if len(forwards) != 1 {
		t.Fatalf("ice then fire fired %d combos, want 1", len(forwards))
	}

	backwards := matchSlots(appendTurn(nil, SideA,
		[]Card{Of(Strike, Fire), Of(Strike, Ice)}), table)
	if len(backwards) != 0 {
		t.Errorf("fire then ice fired %v; the same two cards in the other order is a different round",
			backwards)
	}
}

func TestBasicIsAColourAStepCanName(t *testing.T) {
	// `hasElement` is a separate flag rather than Basic meaning "unset", because an all-plain
	// run is a pattern somebody may legitimately want and `element: 0` could not say whether it
	// had been asked for.
	table := []Combo{{
		ID:   ComboID(303),
		Name: "plain pair",
		Run:  []Step{Exactly(Jab).WithElement(Basic), Exactly(Jab).WithElement(Basic)},
	}}

	if got := len(matchSlots(appendTurn(nil, SideA, PlainCards(Jab, Jab)), table)); got != 1 {
		t.Errorf("two plain Jabs fired %d combos, want 1", got)
	}
	if got := len(matchSlots(appendTurn(nil, SideA,
		[]Card{Of(Jab, Fire), Of(Jab, Fire)}), table)); got != 0 {
		t.Errorf("two fire Jabs matched a pattern that named basic")
	}
}
