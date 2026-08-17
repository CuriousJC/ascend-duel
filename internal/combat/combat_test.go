package combat

import (
	"math/rand"
	"testing"

	"github.com/curiousjc/ascend-duel/data"
)

// duelist builds a full-health duelist for tests.
// resolve is ResolveRound with **no randomness at all**, which is what almost every test here
// wants: a nil source means the shock roll never fires, so everything that is still exact stays
// exact and a test about defences is not a test about luck.
//
// The rolled path has its own tests, which pass a decided source deliberately — see below and
// status_test.go.
func resolve(a, b Duelist, aCards, bCards []Card, round int) ([]Event, Duelist, Duelist) {
	return ResolveRound(a, b, aCards, bCards, round, nil)
}

// resolveWith is the same round with a source, for the tests that are about the roll.
func resolveWith(rng *rand.Rand, a, b Duelist, aCards, bCards []Card, round int) ([]Event, Duelist, Duelist) {
	return ResolveRound(a, b, aCards, bCards, round, rng)
}

// fixedSource always hands back the same value, so a test can *say* "this attack misses" instead
// of hunting for a seed that happens to make it. A seed would work and would be unreadable: the
// test would assert a miss while naming a number, and retuning shockMissPct could silently turn
// it into a test about something else.
type fixedSource int64

func (s fixedSource) Int63() int64 { return int64(s) }
func (s fixedSource) Seed(int64)   {}

// The two rolls a test wants. rand.Intn(100) reduces to `int32(Int63()>>32) % 100`, so 0 is
// below any chance the game can produce and 99 is above the shockMissCapPct ceiling — which is
// only expressible because the cap exists, and is one more reason it has to.
func alwaysMisses() *rand.Rand { return rand.New(fixedSource(0)) }
func neverMisses() *rand.Rand  { return rand.New(fixedSource(99 << 32)) }

func duelist(dmg, actions, life int) Duelist {
	return Duelist{DMG: dmg, Actions: actions, MaxLife: life, CurrentLife: life}
}

// firstDamage returns the first damage event dealt by the given side.
func firstDamage(t *testing.T, events []Event, by Side) Event {
	t.Helper()
	for _, e := range events {
		if e.Kind == KindDamage && e.Side == by {
			return e
		}
	}
	t.Fatalf("no damage event from side %s", by)
	return Event{}
}

// damageCount is how many damage events the log holds.
func damageCount(events []Event) int {
	n := 0
	for _, e := range events {
		if e.Kind == KindDamage {
			n++
		}
	}
	return n
}

// actionOrder returns the sides in the order they acted.
func actionOrder(events []Event) []Side {
	var order []Side
	for _, e := range events {
		if e.Kind == KindAction {
			order = append(order, e.Side)
		}
	}
	return order
}

// playedCards returns the cards in the order they resolved, element included — so a test can
// compare a played round against the PlainCards or Of() list it queued, without either side
// having to strip a colour off.
func playedCards(events []Event) []Card {
	var played []Card
	for _, e := range events {
		if e.Kind == KindAction {
			played = append(played, Card{Concept: e.Action, Element: e.Element})
		}
	}
	return played
}

// sidesEqual compares two action orders.
func sidesEqual(got, want []Side) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// cardsEqual compares two played sequences.
func cardsEqual(got, want []Card) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestSideATakesItsWholeTurnFirst(t *testing.T) {
	// A duelist's Spd buys action points, never priority: B is 500 times faster here and
	// still does not act until A has finished. Speed is how *much* you do, and nothing
	// else — initiative was the lever that said otherwise and it no longer exists.
	a := duelist(10, 4, 500)
	b := duelist(10, 54, 500)

	events, _, _ := resolve(a, b,
		PlainCards(Strike, Strike), PlainCards(Strike, Strike), 1)

	want := []Side{SideA, SideA, SideB, SideB}
	if order := actionOrder(events); !sidesEqual(order, want) {
		t.Errorf("action order = %v, want %v even with B far faster", order, want)
	}
}

func TestATurnResolvesInCategoryOrder(t *testing.T) {
	// Attacks, then plans, whatever order the cards were queued in. The plans go last within a
	// turn because the *opponent* moves next, so a defence raised at the end of a turn is up when
	// the blow arrives.
	a := duelist(10, 5, 500)
	b := duelist(10, 5, 500)

	queued := PlainCards(Plan, Strike, Prepare, Defend, Smash)
	events, _, _ := resolve(a, b, queued, nil, 1)

	want := PlainCards(Strike, Smash, Plan, Prepare, Defend)
	if got := playedCards(events); !cardsEqual(got, want) {
		t.Errorf("played %v, want %v", got, want)
	}
}

func TestQueuedOrderSurvivesInsideACategory(t *testing.T) {
	// Reordering across categories does nothing, but reordering *within* one is the whole
	// remaining point of dragging a card along the row — sequence combos will match on it.
	a := duelist(10, 5, 500)
	b := duelist(10, 5, 500)

	first, _, _ := resolve(a, b, PlainCards(Smash, Jab), nil, 1)
	second, _, _ := resolve(a, b, PlainCards(Jab, Smash), nil, 1)

	if got := playedCards(first); !cardsEqual(got, PlainCards(Smash, Jab)) {
		t.Errorf("played %v, want [Smash Jab]", got)
	}
	if got := playedCards(second); !cardsEqual(got, PlainCards(Jab, Smash)) {
		t.Errorf("played %v, want [Jab Smash]", got)
	}
}

func TestResolutionOrderIsWhatResolveRoundPlays(t *testing.T) {
	// The Resolution pane draws ResolutionOrder and the engine plays it. If these two
	// ever disagree the pane is lying to the player about their own round, so pin that
	// they are the same sequence rather than trusting the shared call.
	a := duelist(10, 5, 500)
	b := duelist(10, 5, 500)
	aPlan := PlainCards(Smash, Plan, Prepare)
	bPlan := PlainCards(Jab, Defend)

	events, _, _ := resolve(a, b, aPlan, bPlan, 1)

	want := make([]Card, 0, len(aPlan)+len(bPlan))
	for _, slot := range ResolutionOrder(aPlan, bPlan) {
		want = append(want, slot.Card)
	}

	if got := playedCards(events); !cardsEqual(got, want) {
		t.Errorf("played order = %v, ResolutionOrder = %v", got, want)
	}
}

func TestResolutionOrderNeverPutsAAfterB(t *testing.T) {
	// ResolveRound expires side B's defenses at the first B slot, which is only the start
	// of B's turn if A's slots never follow it. Pin the block structure that relies on.
	order := ResolutionOrder(
		PlainCards(Plan, Strike, Prepare),
		PlainCards(Defend, Smash, Plan))

	seenB := false
	for _, slot := range order {
		if slot.Side == SideB {
			seenB = true
			continue
		}
		if seenB {
			t.Fatalf("side A acts after side B has started: %v", order)
		}
	}
}

func TestSlotIndexIsThePositionInItsOwnQueue(t *testing.T) {
	// Index is where the card sits in the player's queue, not where it lands in the round.
	// Anything wanting "how far through the round are we" has to count slots instead.
	order := ResolutionOrder(PlainCards(Plan, Jab), nil)

	if len(order) != 2 {
		t.Fatalf("got %d slots, want 2", len(order))
	}
	if order[0].Card.Concept != Jab || order[0].Index != 1 {
		t.Errorf("first slot = %v index %d, want Jab index 1", order[0].Card.Concept, order[0].Index)
	}
	if order[1].Card.Concept != Plan || order[1].Index != 0 {
		t.Errorf("second slot = %v index %d, want Plan index 0", order[1].Card.Concept, order[1].Index)
	}
}

func TestStrikeDealsDMGAsDamage(t *testing.T) {
	a := duelist(10, 5, 100)
	b := duelist(10, 5, 100)

	events, _, bAfter := resolve(a, b, PlainCards(Strike), nil, 1)

	got := firstDamage(t, events, SideA)
	if got.Amount != 10 {
		t.Errorf("Strike damage = %d, want 10 (attacker DMG)", got.Amount)
	}
	if bAfter.CurrentLife != 90 {
		t.Errorf("target life after Strike = %d, want 90", bAfter.CurrentLife)
	}
}

func TestTheTopTierHitsTwiceAsHardAsTheMiddleOne(t *testing.T) {
	a := duelist(10, 5, 500)
	b := duelist(1, 5, 500)

	strikeLog, _, _ := resolve(a, b, PlainCards(Strike), nil, 1)
	smashLog, _, _ := resolve(a, b, PlainCards(Smash), nil, 1)

	strike := firstDamage(t, strikeLog, SideA)
	smash := firstDamage(t, smashLog, SideA)

	if smash.Amount != strike.Amount*2 {
		t.Errorf("Smash = %d, Strike = %d; want the 3 AP card to be double", smash.Amount, strike.Amount)
	}
}

func TestJabHitsForHalfButNeverZero(t *testing.T) {
	// A DMG of 1 halves to 0 under integer division; a hit that does nothing would make
	// low-DMG duels unresolvable, so Jab floors at 1.
	events, _, _ := resolve(duelist(1, 5, 100), duelist(1, 5, 100), PlainCards(Jab), nil, 1)

	if got := firstDamage(t, events, SideA); got.Amount != 1 {
		t.Errorf("Jab damage at DMG 1 = %d, want 1", got.Amount)
	}
}

func TestOnlyAttacksDealDamage(t *testing.T) {
	// **Nothing in the plan family hits back.** A defence is a wall, not a counter, so a turn made
	// of plans alone is a turn in which nobody is hurt.
	for _, a := range []ConceptID{Prepare, Plan, Defend} {
		events, _, bAfter := resolve(duelist(10, 5, 100), duelist(10, 5, 100),
			PlainCards(a), nil, 1)

		if n := damageCount(events); n != 0 {
			t.Errorf("%v dealt damage %d times unprompted", a, n)
		}
		if bAfter.CurrentLife != 100 {
			t.Errorf("%v took the target to %d, want 100", a, bAfter.CurrentLife)
		}
	}
}

func TestADefendHalvesTheHandRatherThanTheCards(t *testing.T) {
	// What this pins is that the halving lands **after** the combo multiplier — a Defend cuts the
	// hand, not the cards that built it. Halving first would make it strongest against exactly
	// the turns it should be weakest against.
	a := duelist(10, 5, 500)
	b := duelist(10, 5, 500)

	open, _, _ := resolve(a, b, nil, PlainCards(Strike, Strike, Strike), 1)
	shielded, _, _ := resolve(a, b, PlainCards(Defend), PlainCards(Strike, Strike, Strike), 1)

	if n := damageCount(open); n != 1 {
		t.Fatalf("an open turn produced %d damage events, want 1 — a turn is one blow", n)
	}
	if n := damageCount(shielded); n != 1 {
		t.Fatalf("a defended turn produced %d damage events, want 1", n)
	}

	full := firstDamage(t, open, SideB).Amount
	half := firstDamage(t, shielded, SideB).Amount
	if want := full * (100 - ConceptOf(Defend).Amount) / 100; half != want {
		t.Errorf("a flurry through a Defend dealt %d, want %d (half of %d)", half, want, full)
	}
	if half >= full {
		t.Errorf("a defended blow dealt %d against an open %d — the Defend did nothing", half, full)
	}
}

func TestADefendCoversExactlyOneOpposingTurn(t *testing.T) {
	// Raised in round 1, still up for B's round-1 turn, gone by the time A's round-2 turn
	// begins. A defense expires at the start of its owner's next turn — not at the round
	// boundary, which would leave side B's defenses protecting it from nothing.
	a := duelist(10, 5, 500)
	b := duelist(10, 5, 500)

	round1, a1, b1 := resolve(a, b, PlainCards(Defend), PlainCards(Strike), 1)
	if hit := firstDamage(t, round1, SideB); hit.Amount != 5 {
		t.Errorf("round 1 hit into a fresh Defend = %d, want 5", hit.Amount)
	}
	// **Spent, not standing.** A defence answers exactly one blow and goes with it, so B's Strike
	// is what consumed it — which is the same reason round two below arrives at full strength.
	if a1.DefendCount != 0 {
		t.Fatalf("A ended the round holding %d defences, want the Defend spent on B's blow", a1.DefendCount)
	}

	round2, _, _ := resolve(a1, b1, PlainCards(Jab), PlainCards(Strike), 2)
	if hit := firstDamage(t, round2, SideB); hit.Amount != 10 {
		t.Errorf("round 2 hit after the Defend expired = %d, want full 10", hit.Amount)
	}
}

func TestSideBsDefenceProtectsItInTheFollowingRound(t *testing.T) {
	// The asymmetry the expiry rule exists for. B acts last, so its defence cannot cover
	// anything in the round it was raised — it has to survive the boundary and cover A's
	// next turn, or the card would be worthless in B's hands.
	a := duelist(10, 5, 500)
	b := duelist(10, 5, 500)

	_, a1, b1 := resolve(a, b, nil, PlainCards(Defend), 1)
	if b1.DefendCount != 1 {
		t.Fatal("side B's Defend did not survive the round it was raised in")
	}

	round2, _, _ := resolve(a1, b1, PlainCards(Strike), nil, 2)
	if hit := firstDamage(t, round2, SideA); hit.Amount != 5 {
		t.Errorf("A's hit into B's carried Defend = %d, want 5", hit.Amount)
	}
}

func TestAnIdleDuelistLosesItsDefence(t *testing.T) {
	// A turn happens whether or not anything is queued into it, and a defence expires at
	// the start of its owner's turn. Standing still therefore does not bank one.
	a := duelist(10, 5, 500)
	b := duelist(10, 5, 500)

	_, a1, b1 := resolve(a, b, PlainCards(Defend), nil, 1)

	round2, _, _ := resolve(a1, b1, nil, PlainCards(Strike), 2)
	if hit := firstDamage(t, round2, SideB); hit.Amount != 10 {
		t.Errorf("hit in round 2 = %d, want full 10 — an idle turn still expires a defence", hit.Amount)
	}
}

func TestTheOrderDefencesWereRaisedInChangesNothing(t *testing.T) {
	// **The raise-order rule is retired** *(2026-08-14)*. It stood for a day, and one attack per
	// turn removed its content: there is no "first blow" for the first card to answer, so every
	// raised card meets the same one and they compose.
	//
	// Both orderings are checked, because a test that queued only one would pass against a rule
	// that still read the queue. **The reductions truncate**, so commutativity is a property of
	// these two percentages rather than a theorem — if a retune breaks this, that is worth
	// knowing rather than worth silently allowing.
	var took []int
	for _, raised := range [][]Card{
		PlainCards(Defend, Prepare),
		PlainCards(Prepare, Defend),
	} {
		events, _, _ := resolve(duelist(10, 5, 500), duelist(10, 5, 500),
			raised, PlainCards(Strike, Strike), 1)
		took = append(took, firstDamage(t, events, SideB).Amount)
	}

	if took[0] != took[1] {
		t.Errorf("Defend-then-Prepare let %d through and Prepare-then-Defend %d — order must not matter",
			took[0], took[1])
	}
}

func TestDefensesExpireWithTheTurnTheyCovered(t *testing.T) {
	// An unspent Defend does not bank. It covers the opponent's next turn and then goes.
	a := duelist(10, 5, 500)
	b := duelist(10, 5, 500)

	_, a1, b1 := resolve(a, b, PlainCards(Defend), nil, 1)
	if a1.DefendCount != 1 || a1.Defends[0].Card != Defend {
		t.Fatalf("A ended round 1 holding %d defends (%v), want one unspent Defend",
			a1.DefendCount, a1.Defends[0].Card)
	}

	round2, _, _ := resolve(a1, b1, PlainCards(Jab), PlainCards(Strike), 2)
	if n := damageCount(round2); n != 2 {
		t.Errorf("damage events in round 2 = %d, want 2 — the defence expired at A's turn", n)
	}
}

func TestPrepareFundsTheFollowingRound(t *testing.T) {
	d := duelist(10, 5, 100) // 5 AP before any bonus

	_, aAfter, _ := resolve(d, duelist(10, 5, 100), PlainCards(Prepare), nil, 1)

	if aAfter.ActionPoints() != 7 {
		t.Errorf("budget after one Prepare = %d, want 7 (5 + 2)", aAfter.ActionPoints())
	}
}

func TestPrepareDoesNotFundTheRoundItIsPlayedIn(t *testing.T) {
	// The whole shape of the card: you pay now and you are paid later, which is what makes
	// it an investment rather than a discount.
	d := duelist(10, 5, 100)

	events, _, _ := resolve(d, duelist(10, 5, 100), PlainCards(Prepare), nil, 1)

	for _, e := range events {
		if e.Kind == KindGathered && e.Amount != ConceptOf(Prepare).Amount {
			t.Errorf("gathered %d AP, want %d", e.Amount, ConceptOf(Prepare).Amount)
		}
	}
	if d.ActionPoints() != 5 {
		t.Errorf("the duelist's own budget moved to %d mid-round, want 5", d.ActionPoints())
	}
}

func TestPreparesStackWithinARound(t *testing.T) {
	// Two in one round are worth +4. Deliberate: it is what puts a five-attack round in
	// reach without a ring discount, and that trade — a whole round spent setting up — is
	// the price of getting there.
	d := duelist(10, 5, 100) // 5 AP

	_, aAfter, _ := resolve(d, duelist(10, 5, 100),
		PlainCards(Prepare, Prepare), nil, 1)

	if aAfter.ActionPoints() != 9 {
		t.Errorf("budget after two Prepares = %d, want 9 (5 + 4)", aAfter.ActionPoints())
	}
}

func TestPrepareDoesNotCompoundAcrossRounds(t *testing.T) {
	// Preparing every round is worth a flat +2, not +2 then +4 then +6. The bonus is
	// replaced at the boundary rather than added to, which is what bounds the ramp.
	a := duelist(10, 5, 500) // 5 AP
	b := duelist(10, 5, 500)

	_, a1, b1 := resolve(a, b, PlainCards(Prepare), nil, 1)
	_, a2, _ := resolve(a1, b1, PlainCards(Prepare), nil, 2)

	if a2.ActionPoints() != 7 {
		t.Errorf("budget after preparing twice in a row = %d, want a flat 7", a2.ActionPoints())
	}
}

func TestABonusLapsesIfItIsNotRenewed(t *testing.T) {
	a := duelist(10, 5, 500)
	b := duelist(10, 5, 500)

	_, a1, b1 := resolve(a, b, PlainCards(Prepare), nil, 1)
	_, a2, _ := resolve(a1, b1, PlainCards(Strike), nil, 2)

	if a2.ActionPoints() != 5 {
		t.Errorf("budget after spending the bonus round = %d, want 5", a2.ActionPoints())
	}
}

func TestActionPointsFromSpeed(t *testing.T) {
	cases := []struct {
		actions, bonus, want int
	}{
		{4, 0, 4},
		{5, 0, 5},
		{6, 0, 6},
		{6, 2, 8},  // a Prepare banked last round
		{4, -2, 2}, // a sim probing below the floor is not the engine's problem
	}

	for _, c := range cases {
		d := Duelist{Actions: c.actions, BonusAP: c.bonus}
		if got := d.ActionPoints(); got != c.want {
			t.Errorf("ActionPoints at Actions %d bonus %d = %d, want %d",
				c.actions, c.bonus, got, c.want)
		}
	}
}

func TestCanAffordEnforcesTheBudget(t *testing.T) {
	d := duelist(10, 5, 100) // 5 AP

	if !d.CanAfford(PlainCards(Smash, Thrust)) { // 3 + 2
		t.Error("Smash + Thrust costs 5 and should fit a 5 AP budget")
	}
	if d.CanAfford(PlainCards(Defend, Lunge)) { // 3 + 3
		t.Error("Defend + Lunge costs 6 and should not fit a 5 AP budget")
	}
}

func TestCategoriesCoverEveryPlayerConcept(t *testing.T) {
	// A card whose verb has no category would silently fall into attack and resolve in the wrong
	// phase. Pin the whole player deck instead — the enemy decks are checked by the same rule, but
	// they are data and this test would become a copy of enemies.json.
	want := map[ConceptID]Category{
		Jab:    CategoryAttack,
		Thrust: CategoryAttack,
		Lunge:  CategoryAttack,

		Cut:    CategoryAttack,
		Slash:  CategoryAttack,
		Cleave: CategoryAttack,

		Bash:   CategoryAttack,
		Strike: CategoryAttack,
		Smash:  CategoryAttack,

		Prepare: CategoryPlan,
		Plan:    CategoryPlan,
		Defend:  CategoryPlan,
	}

	got := PlayerConcepts()
	if len(got) != len(want) {
		t.Fatalf("the player has %d concepts, the category table %d", len(got), len(want))
	}
	for _, a := range got {
		if c := Plain(a).Category(); c != want[a] {
			t.Errorf("%v is %v, want %v", ConceptOf(a).Label, c, want[a])
		}
	}
}

func TestDefeatStopsTheRoundEarly(t *testing.T) {
	// A queues three strikes into an enemy that cannot survive the hand they form. B must never
	// get its reply.
	//
	// **The three Strikes are one blow, and they are still three beats.** They are announced
	// before the hand is scored, so A's turn contributes three actions and B's contributes none —
	// which is the honest reading of "the round stopped at the kill" now that a turn cannot be
	// cut off partway through its own attack.
	a := duelist(100, 5, 100)
	b := duelist(10, 5, 10)

	events, _, bAfter := resolve(a, b,
		PlainCards(Strike, Strike, Strike), PlainCards(Strike), 1)

	if bAfter.Alive() {
		t.Fatalf("side B survived with %d life, want defeated", bAfter.CurrentLife)
	}
	if n := damageCount(events); n != 1 {
		t.Errorf("damage events = %d, want 1 — a turn lands one blow", n)
	}
	if order := actionOrder(events); !sidesEqual(order, []Side{SideA, SideA, SideA}) {
		t.Errorf("actions = %v, want only A's three — B is dead and cannot reply", order)
	}
}

func TestLifeNeverGoesNegative(t *testing.T) {
	events, _, bAfter := resolve(duelist(1000, 5, 100), duelist(1, 5, 5),
		PlainCards(Strike), nil, 1)

	for _, e := range events {
		if e.Life < 0 {
			t.Fatalf("event %v reported negative life %d", e.Kind, e.Life)
		}
	}
	if bAfter.CurrentLife != 0 {
		t.Errorf("life after overkill = %d, want 0", bAfter.CurrentLife)
	}
}

func TestResolveRoundDoesNotMutateItsInputs(t *testing.T) {
	// The screen holds the live combatants; ResolveRound must not drain them as a
	// side effect of building the log.
	a := duelist(10, 5, 100)
	b := duelist(10, 5, 100)

	resolve(a, b, PlainCards(Strike), PlainCards(Strike), 1)

	if a.CurrentLife != 100 || b.CurrentLife != 100 {
		t.Errorf("inputs mutated: a=%d b=%d, want both 100", a.CurrentLife, b.CurrentLife)
	}
}

func TestBothSidesDefendingDoesNotAlias(t *testing.T) {
	// resolveAction passes duelists by value; if that ever became pointer-based, a mutual
	// defense is where an aliasing bug would surface — one side's raised guard leaking
	// onto the other.
	a := duelist(10, 5, 100)
	b := duelist(10, 5, 100)

	_, a1, b1 := resolve(a, b, PlainCards(Defend), PlainCards(Defend), 1)
	if a1.DefendCount != 1 || a1.Defends[0].Card != Defend {
		t.Errorf("A raised a Defend and ended the round holding %d (%v)",
			a1.DefendCount, a1.Defends[0].Card)
	}
	if b1.DefendCount != 1 || b1.Defends[0].Card != Defend {
		t.Errorf("B raised a Defend and ended the round holding %d (%v)",
			b1.DefendCount, b1.Defends[0].Card)
	}
}

func TestRoundIsDeterministic(t *testing.T) {
	a := duelist(7, 5, 300)
	b := duelist(9, 5, 300)

	aPlan := PlainCards(Strike, Defend, Prepare)
	bPlan := PlainCards(Jab, Defend)

	first, a1, b1 := resolve(a, b, aPlan, bPlan, 1)
	second, a2, b2 := resolve(a, b, aPlan, bPlan, 1)

	if len(first) != len(second) {
		t.Fatalf("log lengths differ: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("event %d differs: %+v vs %+v", i, first[i], second[i])
		}
	}
	if a1 != a2 || b1 != b2 {
		t.Error("resulting duelists differ between identical runs")
	}
}

func TestEmptyQueueIsAHarmlessRound(t *testing.T) {
	a := duelist(10, 5, 100)
	b := duelist(10, 5, 100)

	_, aAfter, bAfter := resolve(a, b, nil, nil, 1)

	if aAfter.CurrentLife != 100 || bAfter.CurrentLife != 100 {
		t.Errorf("nobody acted but life changed: a=%d b=%d", aAfter.CurrentLife, bAfter.CurrentLife)
	}
}

// stockHand holds enough of everything the planner could want, so the tests below stay tests of
// *preference* rather than of the draw. What the deck actually does to a plan is
// TestThePlannerNeverPlaysACardItWasNotDealt and the balance tool.
func stockHand() []Card {
	var hand []Card
	for i := 0; i < 6; i++ {
		hand = append(hand, PlainCards(Prepare, Defend, Jab, Strike, Smash)...)
	}
	return hand
}

func TestThePlannerNeverPlaysACardItWasNotDealt(t *testing.T) {
	// **The whole point of enemies having a deck.** The planner decides how a hand is spent, not
	// what is in it, so one handed three Jabs may only ever queue Jabs — and one handed nothing may
	// only queue nothing. Before decks landed a brute produced Heavies out of thin air, which made
	// the enemy card list a file that could never matter.
	hands := [][]Card{
		nil,
		PlainCards(Jab),
		PlainCards(Jab, Jab, Jab),
		PlainCards(Defend, Prepare, Plan), // no attacks at all
		PlainCards(Defend, Prepare),
		PlainCards(Smash, Smash, Jab, Defend),
	}

	for _, hand := range hands {
		d := duelist(10, 9, 100) // deliberately rich: the budget must not be the limit
		plan := PlanFor(d, hand)

		left := append([]Card(nil), hand...)
		for _, a := range plan {
			found := -1
			for i, c := range left {
				if c == a {
					found = i
					break
				}
			}
			if found < 0 {
				t.Fatalf("planned %v from a hand of %v", a, hand)
			}
			left = append(left[:found], left[found+1:]...)
		}
	}
}

func TestPlanningIsReproducible(t *testing.T) {
	// The determinism rule, at the planner. Nothing here may consult a map's iteration order
	// or a clock, so the same hand must plan the same round every time — which is what lets a
	// seeded run be replayed and what the balance tool depends on.
	hand := PlainCards(Smash, Jab, Strike, Defend, Jab, Prepare, Strike, Smash)

	d := duelist(10, 4, 100)
	want := planKey(PlanFor(d, hand))
	for i := 0; i < 50; i++ {
		if got := planKey(PlanFor(d, hand)); got != want {
			t.Fatalf("planned %s then %s from the same hand", want, got)
		}
	}
}

func TestThePlannerObeysBothBounds(t *testing.T) {
	// The two bounds on a round apply to the opponent exactly as they apply to the player.
	// Swept across a wide budget range and across banked points, because both bounds move:
	// AP grows with the stat and with BonusAP, while the action cap does not, which is the
	// corner where an unbounded planner would overrun.
	for actions := 0; actions <= 16; actions++ {
		for _, bonus := range []int{0, 2, 4, 10} {
			d := duelist(10, actions, 100)
			d.BonusAP = bonus
			plan := PlanFor(d, stockHand())

			if !d.CanAfford(plan) {
				t.Errorf("at %d actions bonus %d: plan costs %d, budget is %d",
					actions, bonus, CostOf(plan), d.ActionPoints())
			}
			if len(plan) > d.MaxActions() {
				t.Errorf("at %d actions bonus %d: %d actions, cap is %d",
					actions, bonus, len(plan), d.MaxActions())
			}
			if d.ActionPoints() > 0 && len(plan) == 0 {
				t.Errorf("at %d actions bonus %d: plan is empty", actions, bonus)
			}
		}
	}
}

// planKey renders a plan so two of them can be compared as strings.
func planKey(plan []Card) string {
	out := ""
	for i, c := range plan {
		if i > 0 {
			out += "+"
		}
		out += c.String()
	}
	return out
}

func TestThePlannerBuildsTheHardestHittingHand(t *testing.T) {
	// **The rule the four styles collapsed into.** A greedy planner takes the dearest card that
	// fits and stops; this one scores whole combinations through the same BlowFor the resolver
	// uses, so it finds that three Jabs forming a Flurry beat one Smash.
	d := duelist(10, 3, 100) // 3 AP: one Smash, or three Jabs
	hand := PlainCards(Smash, Jab, Jab, Jab)

	plan := PlanFor(d, hand)
	if len(plan) != 3 {
		t.Fatalf("planned %v, want the three Jabs", planKey(plan))
	}
	for _, c := range plan {
		if c.Concept != Jab {
			t.Fatalf("planned %v, want the three Jabs", planKey(plan))
		}
	}
}

func TestThePlannerSpendsWhatTheAttacksDidNotWant(t *testing.T) {
	// **This is what keeps a non-attack card in an enemy deck from being dead content.** A planner
	// that only maximised damage would never raise a shield, so every defensive card authored into
	// the roster would sit in a discard pile forever.
	d := duelist(10, 5, 100) // 2 AP of attack, 3 left over
	hand := PlainCards(Strike, Defend)

	plan := PlanFor(d, hand)

	attacks, shields := 0, 0
	for _, c := range plan {
		switch c.Spec().Verb {
		case VerbAttack:
			attacks++
		case VerbDefend:
			shields++
		}
	}
	if attacks != 1 || shields != 1 {
		t.Errorf("planned %v, want the Strike and the Defend", planKey(plan))
	}
}

func TestThePlannerPrefersAShieldToABank(t *testing.T) {
	// The one tie-break among the leftovers, and the reason for it: a defence is the leftover that
	// decides whether the enemy is alive to use the next one.
	d := duelist(10, 5, 100)
	hand := PlainCards(Strike, Prepare, Defend)

	plan := PlanFor(d, hand)
	for _, c := range plan {
		if c.Concept == Defend {
			return
		}
	}
	t.Errorf("planned %v with 3 spare AP and a Defend in hand", planKey(plan))
}

func TestThePlannerNeverSpendsAnAttackOnItself(t *testing.T) {
	// Recoil costs its owner life and contributes nothing to the hand, so a planner reaching for
	// one would be choosing to hurt itself for no gain. Nothing reads recoil yet — see MECHANICS.md
	// — and this is what pins that the search does not stumble into it.
	recoil := mustTestConcept("PlannerRecoil", data.CardData{
		Label: "PlannerRecoil", Verb: "attack", Amount: 200, Cost: 1, Target: "self",
	})

	d := duelist(10, 5, 100)
	for _, c := range PlanFor(d, PlainCards(recoil, Strike)) {
		if c.Concept == recoil {
			t.Error("the planner queued a recoil card, which only costs it life")
		}
	}
}
