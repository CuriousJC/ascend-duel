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

	events, _, _ := ResolveRound(a, b, []ActionKind{Strike, Strike, Strike}, nil, 1)

	got := combosFired(events, SideA)
	if len(got) != 1 || got[0] != FlurryID(Strike) {
		t.Fatalf("three attacks should form one Flurry, got %v", got)
	}
}

func TestTwoAttacksFormNothing(t *testing.T) {
	a, b := duelist(10, 0, 100), duelist(10, 0, 100)

	events, _, _ := ResolveRound(a, b, []ActionKind{Strike, Heavy}, nil, 1)

	if got := combosFired(events, SideA); len(got) != 0 {
		t.Fatalf("two attacks should form nothing, got %v", got)
	}
}

// Three attacks of *different* cards form nothing. The family is per card on purpose: three
// Strikes is a deck you assembled, three attacks is whatever you happened to draw.
func TestThreeDifferentAttacksFormNothing(t *testing.T) {
	a, b := duelist(10, 0, 100), duelist(10, 0, 100)

	events, _, _ := ResolveRound(a, b, []ActionKind{Quick, Strike, Heavy}, nil, 1)

	if got := combosFired(events, SideA); len(got) != 0 {
		t.Fatalf("Quick+Strike+Heavy is not a run of one card, got %v", got)
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
			queue := make([]ActionKind, tc.n)
			for i := range queue {
				queue[i] = card
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
		{FlurryID(Quick), "Quick Flurry"},
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

	events, _, _ := ResolveRound(a, b, []ActionKind{Strike, Strike, Strike, Strike}, nil, 1)

	if got := combosFired(events, SideA); len(got) != 1 {
		t.Fatalf("a card may form at most one combo, so four attacks is one Flurry; got %v", got)
	}
}

func TestFiveAttacksFormOnslaughtRatherThanFlurry(t *testing.T) {
	a, b := duelist(10, 0, 200), duelist(10, 0, 200)

	events, _, _ := ResolveRound(a, b, []ActionKind{Quick, Quick, Quick, Quick, Quick}, nil, 1)

	got := combosFired(events, SideA)
	if len(got) != 1 || got[0] != OnslaughtID(Quick) {
		t.Fatalf("longest run should win, so five attacks is Onslaught; got %v", got)
	}
}

// The whole reason matching happens on the resolved order: the queue here is interrupted by a
// Dodge, but phases move every defense to the end of the turn, so three attacks still land
// back to back. Matching the queue would have missed it and the Resolution pane would have
// shown a combo the engine did not score.
func TestCombosMatchTheResolvedOrderNotTheQueuedOne(t *testing.T) {
	a, b := duelist(10, 0, 100), duelist(10, 0, 100)

	events, _, _ := ResolveRound(a, b, []ActionKind{Strike, Dodge, Strike, Strike}, nil, 1)

	if got := combosFired(events, SideA); len(got) != 1 || got[0] != FlurryID(Strike) {
		t.Fatalf("the Dodge regroups to the end, leaving three consecutive attacks; got %v", got)
	}
}

// MatchCombos is what the screen calls while the player is still planning, and it has to
// agree with the engine or the preview lies.
func TestMatchCombosAgreesWithWhatTheEnginePlays(t *testing.T) {
	queue := []ActionKind{Strike, Dodge, Strike, Strike}

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
	short := Combo{ID: ComboID(90), Name: "short", Run: []Step{Card(Strike), Card(Strike)}}
	long := Combo{ID: ComboID(91), Name: "long", Run: []Step{Card(Strike), Card(Strike), Card(Strike)}}

	// Declared shortest-first, so a matcher that simply took the first hit would be wrong.
	turn := appendTurn(nil, SideA, []ActionKind{Strike, Strike, Strike})
	hits := matchSlots(turn, []Combo{short, long})

	if len(hits) != 1 || hits[0].ID != long.ID {
		t.Fatalf("longest run should win regardless of table order, got %v", hits)
	}
}

func TestAnExactCardPatternDoesNotMatchItsCategory(t *testing.T) {
	only := Combo{ID: ComboID(92), Name: "two strikes", Run: []Step{Card(Strike), Card(Strike)}}

	turn := appendTurn(nil, SideA, []ActionKind{Strike, Heavy})
	if hits := matchSlots(turn, []Combo{only}); len(hits) != 0 {
		t.Fatalf("Card(Strike) should not match a Heavy, got %v", hits)
	}
}

// --- stagger ----------------------------------------------------------------------------

func TestAFlurryCostsTheOpponentTheirNextAction(t *testing.T) {
	a, b := duelist(10, 0, 300), duelist(10, 0, 300)

	events, _, _ := ResolveRound(a, b,
		[]ActionKind{Strike, Strike, Strike},
		[]ActionKind{Quick, Strike}, 1)

	if lost := staggeredActions(events, SideB); len(lost) != 1 || lost[0] != Quick {
		t.Fatalf("B should lose exactly its first action, got %v", lost)
	}
	if took := sideActions(events, SideB); len(took) != 1 || took[0] != Strike {
		t.Fatalf("B should still take the rest of its turn, got %v", took)
	}
}

func TestOnslaughtTakesTheOpponentsWholeTurn(t *testing.T) {
	a, b := duelist(10, 0, 300), duelist(10, 0, 300)

	events, _, _ := ResolveRound(a, b,
		[]ActionKind{Quick, Quick, Quick, Quick, Quick},
		[]ActionKind{Strike, Strike, Guard}, 1)

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
		[]ActionKind{Strike, Strike, Strike},
		[]ActionKind{Gather, Strike}, 1)

	if lost := staggeredActions(events, SideB); len(lost) != 1 || lost[0] != Gather {
		t.Fatalf("the Gather resolves first and so is the action lost, got %v", lost)
	}
	if bAfter.BonusAP != 0 {
		t.Fatalf("a staggered Gather banks nothing, got BonusAP %d", bAfter.BonusAP)
	}

	// The same round without the Flurry, to show the Gather would otherwise have paid.
	_, _, unstaggered := ResolveRound(a, b,
		[]ActionKind{Strike, Strike},
		[]ActionKind{Gather, Strike}, 1)
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
		[]ActionKind{Gather},
		[]ActionKind{Strike, Strike, Strike}, 1)

	if got := combosFired(events, SideB); len(got) != 1 || got[0] != FlurryID(Strike) {
		t.Fatalf("B should form a Flurry, got %v", got)
	}
	if lost := staggeredActions(events, SideA); len(lost) != 0 {
		t.Fatalf("A already acted, so nothing can be taken from it this round, got %v", lost)
	}
	if aAfter.Staggered != 1 {
		t.Fatalf("the stagger should be held on A for next round, got %d", aAfter.Staggered)
	}

	events2, _, _ := ResolveRound(aAfter, bAfter, []ActionKind{Quick, Strike}, nil, 2)
	if lost := staggeredActions(events2, SideA); len(lost) != 1 || lost[0] != Quick {
		t.Fatalf("A should lose its first action next round, got %v", lost)
	}
}

func TestAStaggerIsSpentOnceAndDoesNotLinger(t *testing.T) {
	a, b := duelist(10, 0, 300), duelist(10, 0, 300)

	_, aAfter, bAfter := ResolveRound(a, b,
		[]ActionKind{Gather},
		[]ActionKind{Strike, Strike, Strike}, 1)

	_, aAfter2, _ := ResolveRound(aAfter, bAfter, []ActionKind{Quick, Strike}, nil, 2)
	if aAfter2.Staggered != 0 {
		t.Fatalf("the stagger should be consumed by the turn it hit, got %d", aAfter2.Staggered)
	}

	events3, _, _ := ResolveRound(aAfter2, bAfter, []ActionKind{Quick, Strike}, nil, 3)
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
		[]ActionKind{Strike, Strike, Strike, Strike, Strike}, 1)

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

	events, _, bAfter := ResolveRound(a, b, nil, []ActionKind{Strike}, 1)

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

	_, _, bAfterA := ResolveRound(a, b, []ActionKind{Strike, Strike, Strike}, nil, 1)
	_, aAfterB, _ := ResolveRound(a, b, nil, []ActionKind{Strike, Strike, Strike}, 1)

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
		Run:    []Step{Card(Strike), Card(Strike)},
		Effect: Effect{DamageNum: 2, DamageDen: 1},
	}}

	a, b := duelist(10, 0, 100), duelist(10, 0, 100)
	_, _, bAfter := resolveRound(a, b, []ActionKind{Strike, Strike, Strike}, nil, 1, table)

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
		Run:    []Step{Card(Strike), Card(Strike)},
		Effect: Effect{DamageNum: 3, DamageDen: 2},
	}}

	a := duelist(3, 0, 100)
	b := duelist(3, 0, 100)
	b.Guarded = true

	_, _, bAfter := resolveRound(a, b, []ActionKind{Strike, Strike, Strike}, nil, 1, table)

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

	events, _, _ := ResolveRound(a, b, []ActionKind{Strike, Strike, Strike}, nil, 1)

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
	b.Ripostes = 2

	events, aAfter, bAfter := ResolveRound(a, b, []ActionKind{Strike, Strike, Strike}, nil, 1)

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
		Run:    []Step{Card(Quick), Card(Quick)},
		Effect: Effect{BankAP: 3},
	}}

	a, b := duelist(10, 0, 100), duelist(10, 0, 100)
	before := a.ActionPoints()

	_, aAfter, _ := resolveRound(a, b, []ActionKind{Quick, Quick}, nil, 1, table)

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
		Run:    []Step{Card(Strike), Card(Strike)},
		Effect: Effect{DamageNum: 2, DamageDen: 1},
	}}

	a, b := duelist(10, 0, 200), duelist(10, 0, 200)
	_, aAfter, _ := resolveRound(a, b, []ActionKind{Strike, Strike}, []ActionKind{Strike}, 1, table)

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
	aPlan := []ActionKind{Strike, Strike, Strike}
	bPlan := []ActionKind{Gather, Quick, Strike, Dodge}

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
		if beats[i] != slot.Action {
			t.Fatalf("beat %d is %v, but slot %d is %v", i, beats[i], i, slot.Action)
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
	actions := []ActionKind{Strike, Strike, Strike}

	e1, a1, b1 := ResolveRound(a, b, actions, []ActionKind{Quick, Strike}, 1)
	e2, a2, b2 := ResolveRound(a, b, actions, []ActionKind{Quick, Strike}, 1)

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
