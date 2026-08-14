package combat

import (
	"math/rand"
	"testing"
)

// duelist builds a full-health duelist for tests.
// resolve is ResolveRound with **no randomness at all**, which is what almost every test here
// wants: a nil source means the shock roll never fires, so everything that is still exact stays
// exact and a test about braces is not a test about luck.
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

func duelist(str, spd, life int) Duelist {
	return Duelist{Str: str, Spd: spd, MaxLife: life, CurrentLife: life}
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
			played = append(played, Card{Action: e.Action, Element: e.Element})
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
	a := duelist(10, 1, 500)
	b := duelist(10, 500, 500)

	events, _, _ := resolve(a, b,
		PlainCards(Strike, Strike), PlainCards(Strike, Strike), 1)

	want := []Side{SideA, SideA, SideB, SideB}
	if order := actionOrder(events); !sidesEqual(order, want) {
		t.Errorf("action order = %v, want %v even with B far faster", order, want)
	}
}

func TestATurnResolvesInCategoryOrder(t *testing.T) {
	// Prepares, then attacks, then defenses, whatever order the cards were queued in. The
	// defenses go last within a turn because the *opponent* moves next, so a defense
	// raised at the end of a turn is up when the blow arrives.
	a := duelist(10, 10, 500)
	b := duelist(10, 10, 500)

	queued := PlainCards(Dodge, Strike, Gather, Guard, Heavy)
	events, _, _ := resolve(a, b, queued, nil, 1)

	want := PlainCards(Gather, Guard, Strike, Heavy, Dodge)
	if got := playedCards(events); !cardsEqual(got, want) {
		t.Errorf("played %v, want %v", got, want)
	}
}

func TestQueuedOrderSurvivesInsideACategory(t *testing.T) {
	// Reordering across categories does nothing, but reordering *within* one is the whole
	// remaining point of dragging a card along the row — sequence combos will match on it.
	a := duelist(10, 10, 500)
	b := duelist(10, 10, 500)

	first, _, _ := resolve(a, b, PlainCards(Heavy, Jab), nil, 1)
	second, _, _ := resolve(a, b, PlainCards(Jab, Heavy), nil, 1)

	if got := playedCards(first); !cardsEqual(got, PlainCards(Heavy, Jab)) {
		t.Errorf("played %v, want [Heavy Jab]", got)
	}
	if got := playedCards(second); !cardsEqual(got, PlainCards(Jab, Heavy)) {
		t.Errorf("played %v, want [Jab Heavy]", got)
	}
}

func TestResolutionOrderIsWhatResolveRoundPlays(t *testing.T) {
	// The Resolution pane draws ResolutionOrder and the engine plays it. If these two
	// ever disagree the pane is lying to the player about their own round, so pin that
	// they are the same sequence rather than trusting the shared call.
	a := duelist(10, 10, 500)
	b := duelist(10, 10, 500)
	aPlan := PlainCards(Heavy, Dodge, Gather)
	bPlan := PlainCards(Jab, Guard)

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
		PlainCards(Dodge, Strike, Gather),
		PlainCards(Guard, Heavy, Riposte))

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
	order := ResolutionOrder(PlainCards(Dodge, Gather), nil)

	if len(order) != 2 {
		t.Fatalf("got %d slots, want 2", len(order))
	}
	if order[0].Card.Action != Gather || order[0].Index != 1 {
		t.Errorf("first slot = %v index %d, want Gather index 1", order[0].Card.Action, order[0].Index)
	}
	if order[1].Card.Action != Dodge || order[1].Index != 0 {
		t.Errorf("second slot = %v index %d, want Dodge index 0", order[1].Card.Action, order[1].Index)
	}
}

func TestStrikeDealsStrengthAsDamage(t *testing.T) {
	a := duelist(10, 10, 100)
	b := duelist(10, 10, 100)

	events, _, bAfter := resolve(a, b, PlainCards(Strike), nil, 1)

	got := firstDamage(t, events, SideA)
	if got.Amount != 10 {
		t.Errorf("Strike damage = %d, want 10 (attacker Str)", got.Amount)
	}
	if bAfter.CurrentLife != 90 {
		t.Errorf("target life after Strike = %d, want 90", bAfter.CurrentLife)
	}
}

func TestHeavyHitsTwiceAsHardAsStrike(t *testing.T) {
	a := duelist(10, 10, 500)
	b := duelist(1, 10, 500)

	strikeLog, _, _ := resolve(a, b, PlainCards(Strike), nil, 1)
	heavyLog, _, _ := resolve(a, b, PlainCards(Heavy), nil, 1)

	strike := firstDamage(t, strikeLog, SideA)
	heavy := firstDamage(t, heavyLog, SideA)

	if heavy.Amount != strike.Amount*2 {
		t.Errorf("Heavy = %d, Strike = %d; want Heavy to be double", heavy.Amount, strike.Amount)
	}
}

func TestJabHitsForHalfButNeverZero(t *testing.T) {
	// Str 1 halves to 0 under integer division; a hit that does nothing would make
	// low-Strength duels unresolvable, so Jab floors at 1.
	events, _, _ := resolve(duelist(1, 10, 100), duelist(1, 10, 100), PlainCards(Jab), nil, 1)

	if got := firstDamage(t, events, SideA); got.Amount != 1 {
		t.Errorf("Jab damage at Str 1 = %d, want 1", got.Amount)
	}
}

func TestOnlyAttacksDealDamageOnTheirOwn(t *testing.T) {
	// Riposte reports a damage figure — it is what the counter hits for — but it only ever hits
	// back at something. Played into an opponent who does nothing, it deals nothing.
	for _, a := range []ActionKind{Gather, Guard, Dodge, Riposte} {
		events, _, bAfter := resolve(duelist(10, 10, 100), duelist(10, 10, 100),
			PlainCards(a), nil, 1)

		if n := damageCount(events); n != 0 {
			t.Errorf("%v dealt damage %d times unprompted", a, n)
		}
		if bAfter.CurrentLife != 100 {
			t.Errorf("%v took the target to %d, want 100", a, bAfter.CurrentLife)
		}
	}
}

func TestAGuardHalvesTheOpponentsWholeTurn(t *testing.T) {
	// Guard is a prepare and it is broad: it covers the opposing turn rather than one card. Under
	// one blow per turn "every attack" and "the blow" are the same thing, so what this pins now is
	// that the halving lands **after** the combo multiplier — a guard cuts the hand, not the cards
	// that built it. Halving first would make Guard stronger against exactly the turns it should
	// be weakest against.
	a := duelist(10, 10, 500)
	b := duelist(10, 10, 500)

	open, _, _ := resolve(a, b, nil, PlainCards(Strike, Strike, Strike), 1)
	guarded, _, _ := resolve(a, b, PlainCards(Guard), PlainCards(Strike, Strike, Strike), 1)

	if n := damageCount(open); n != 1 {
		t.Fatalf("an open turn produced %d damage events, want 1 — a turn is one blow", n)
	}
	if n := damageCount(guarded); n != 1 {
		t.Fatalf("a guarded turn produced %d damage events, want 1", n)
	}

	full := firstDamage(t, open, SideB).Amount
	half := firstDamage(t, guarded, SideB).Amount
	if want := full / guardDivisor; half != want {
		t.Errorf("a flurry through a guard dealt %d, want %d (half of %d)", half, want, full)
	}
	if half >= full {
		t.Errorf("a guarded blow dealt %d against an unguarded %d — the guard did nothing", half, full)
	}
}

func TestAGuardCoversExactlyOneOpposingTurn(t *testing.T) {
	// Raised in round 1, still up for B's round-1 turn, gone by the time A's round-2 turn
	// begins. A defense expires at the start of its owner's next turn — not at the round
	// boundary, which would leave side B's defenses protecting it from nothing.
	a := duelist(10, 10, 500)
	b := duelist(10, 10, 500)

	round1, a1, b1 := resolve(a, b, PlainCards(Guard), PlainCards(Strike), 1)
	if hit := firstDamage(t, round1, SideB); hit.Amount != 5 {
		t.Errorf("round 1 hit into a fresh guard = %d, want 5", hit.Amount)
	}
	if !a1.Guarded {
		t.Fatal("A's guard should still be standing at the round boundary")
	}

	round2, _, _ := resolve(a1, b1, PlainCards(Jab), PlainCards(Strike), 2)
	if hit := firstDamage(t, round2, SideB); hit.Amount != 10 {
		t.Errorf("round 2 hit after the guard expired = %d, want full 10", hit.Amount)
	}
}

func TestSideBsGuardProtectsItInTheFollowingRound(t *testing.T) {
	// The asymmetry the expiry rule exists for. B acts last, so its guard cannot cover
	// anything in the round it was raised — it has to survive the boundary and cover A's
	// next turn, or the card would be worthless in B's hands.
	a := duelist(10, 10, 500)
	b := duelist(10, 10, 500)

	_, a1, b1 := resolve(a, b, nil, PlainCards(Guard), 1)
	if !b1.Guarded {
		t.Fatal("side B's guard did not survive the round it was raised in")
	}

	round2, _, _ := resolve(a1, b1, PlainCards(Strike), nil, 2)
	if hit := firstDamage(t, round2, SideA); hit.Amount != 5 {
		t.Errorf("A's hit into B's carried guard = %d, want 5", hit.Amount)
	}
}

func TestAnIdleDuelistLosesItsGuard(t *testing.T) {
	// A turn happens whether or not anything is queued into it, and a defense expires at
	// the start of its owner's turn. Standing still therefore does not bank a guard —
	// which the old until-you-act rule allowed, deliberately, and phases dissolve.
	a := duelist(10, 10, 500)
	b := duelist(10, 10, 500)

	_, a1, b1 := resolve(a, b, PlainCards(Guard), nil, 1)

	round2, _, _ := resolve(a1, b1, nil, PlainCards(Strike), 2)
	if hit := firstDamage(t, round2, SideB); hit.Amount != 10 {
		t.Errorf("hit in round 2 = %d, want full 10 — an idle turn still expires a guard", hit.Amount)
	}
}

func TestADodgeTakesItsShareOffTheOneBlow(t *testing.T) {
	// **Reductions, not negations** *(2026-08-14)*. A turn resolves one attack, so an all-or-
	// nothing Dodge erased a whole hand for 2 AP — and "halve the first of five" and "stop three
	// of five" both collapsed to "the blow", which took Brace's and Retreat's reason to exist
	// with them. Each card takes its percentage instead.
	a := duelist(10, 10, 500)
	b := duelist(10, 10, 500)

	open, _, _ := resolve(a, b, nil, PlainCards(Heavy, Strike), 1)
	dodged, aAfter, _ := resolve(a, b, PlainCards(Dodge), PlainCards(Heavy, Strike), 1)

	full := firstDamage(t, open, SideB).Amount
	got := firstDamage(t, dodged, SideB).Amount

	if want := full * (100 - dodgeReductionPct) / 100; got != want {
		t.Errorf("a blow through one Dodge dealt %d, want %d (%d less %d%%)",
			got, want, full, dodgeReductionPct)
	}
	if got <= 0 {
		t.Error("a Dodge stopped the blow dead; only Retreat may reach zero")
	}
	if aAfter.CurrentLife != a.CurrentLife-got {
		t.Errorf("A ended on %d, want %d", aAfter.CurrentLife, a.CurrentLife-got)
	}
}

func TestOnlyRetreatStopsABlowOutright(t *testing.T) {
	// The one card that reaches 100%, and the reason it is allowed to: it is the most expensive
	// defend in the game and 4 of a 6-point budget is most of a round spent doing nothing else.
	// If a second card ever reaches zero, this is the test that should be argued with first.
	for _, card := range []ActionKind{Brace, Dodge, Riposte, Retreat} {
		a := duelist(10, 10, 500)
		b := duelist(10, 10, 500)

		events, _, _ := resolve(a, b, PlainCards(card), PlainCards(Heavy, Strike), 1)
		got := firstDamage(t, events, SideB).Amount

		if card == Retreat && got != 0 {
			t.Errorf("a Retreat let %d through, want 0", got)
		}
		if card != Retreat && got <= 0 {
			t.Errorf("a %v let nothing through; only Retreat may", card)
		}
	}
}

func TestARiposteReducesTheBlowAndHitsBack(t *testing.T) {
	// **The counter fires after the blow rather than instead of it** *(2026-08-14)*, which is
	// what a Riposte reducing rather than negating implies: the attacker connects for a quarter
	// and takes a Jab back for it. Under negation the counter replaced the attack; now it answers
	// one that happened.
	a := duelist(10, 10, 500)
	b := duelist(10, 10, 500)

	open, _, _ := resolve(a, b, nil, PlainCards(Heavy), 1)
	full := firstDamage(t, open, SideB).Amount

	events, aAfter, bAfter := resolve(a, b, PlainCards(Riposte), PlainCards(Heavy), 1)

	through := full * (100 - riposteReductionPct) / 100
	if got := a.CurrentLife - aAfter.CurrentLife; got != through {
		t.Errorf("A took %d through a riposte, want %d", got, through)
	}

	// The counter is dealt by the defender, so its event belongs to side A.
	counter := firstDamage(t, events, SideA)
	if want := Riposte.Damage(a.Str); counter.Amount != want {
		t.Errorf("counter damage = %d, want %d (a Jab's worth)", counter.Amount, want)
	}
	if got := b.CurrentLife - bAfter.CurrentLife; got != counter.Amount {
		t.Errorf("B ended down %d, want %d", got, counter.Amount)
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
		PlainCards(Dodge, Brace),
		PlainCards(Brace, Dodge),
	} {
		events, _, _ := resolve(duelist(10, 10, 500), duelist(10, 10, 500),
			raised, PlainCards(Strike, Strike), 1)
		took = append(took, firstDamage(t, events, SideB).Amount)
	}

	if took[0] != took[1] {
		t.Errorf("Dodge-then-Brace let %d through and Brace-then-Dodge %d — order must not matter",
			took[0], took[1])
	}
}

func TestARiposteCanKillTheAttacker(t *testing.T) {
	// Defenses converting into damage is the theory the category split rests on, so the counter
	// has to be able to finish a duel.
	//
	// **What it can no longer do is cut the turn short.** Every attack card is announced before
	// the blow is worked out, so both Strikes get a beat even though the counter kills B. That is
	// one blow per turn showing through in the log, and the screen depends on it — one beat per
	// slot, taken or lost.
	a := duelist(10, 10, 500)
	b := duelist(10, 10, 3)

	events, _, bAfter := resolve(a, b,
		PlainCards(Riposte), PlainCards(Strike, Strike), 1)

	if bAfter.Alive() {
		t.Fatalf("B survived its own attack into a riposte with %d life", bAfter.CurrentLife)
	}
	if order := actionOrder(events); !sidesEqual(order, []Side{SideA, SideB, SideB}) {
		t.Errorf("actions = %v, want A's riposte and both of B's announced strikes", order)
	}
}

func TestDefensesExpireWithTheTurnTheyCovered(t *testing.T) {
	// An unspent Dodge does not bank. It covers the opponent's next turn and then goes,
	// the same rule Guard follows.
	a := duelist(10, 10, 500)
	b := duelist(10, 10, 500)

	_, a1, b1 := resolve(a, b, PlainCards(Dodge), nil, 1)
	if a1.DefendCount != 1 || a1.Defends[0].Card != Dodge {
		t.Fatalf("A ended round 1 holding %d defends (%v), want one unspent Dodge",
			a1.DefendCount, a1.Defends[0].Card)
	}

	round2, _, _ := resolve(a1, b1, PlainCards(Jab), PlainCards(Strike), 2)
	if n := damageCount(round2); n != 2 {
		t.Errorf("damage events in round 2 = %d, want 2 — the dodge expired at A's turn", n)
	}
}

func TestGatherFundsTheFollowingRound(t *testing.T) {
	d := duelist(10, 11, 100) // 5 AP before any bonus

	_, aAfter, _ := resolve(d, duelist(10, 10, 100), PlainCards(Gather), nil, 1)

	if aAfter.ActionPoints() != 7 {
		t.Errorf("budget after one Gather = %d, want 7 (5 + 2)", aAfter.ActionPoints())
	}
}

func TestGatherDoesNotFundTheRoundItIsPlayedIn(t *testing.T) {
	// The whole shape of the card: you pay now and you are paid later, which is what makes
	// it an investment rather than a discount.
	d := duelist(10, 11, 100)

	events, _, _ := resolve(d, duelist(10, 10, 100), PlainCards(Gather), nil, 1)

	for _, e := range events {
		if e.Kind == KindGathered && e.Amount != gatherBonusAP {
			t.Errorf("gathered %d AP, want %d", e.Amount, gatherBonusAP)
		}
	}
	if d.ActionPoints() != 5 {
		t.Errorf("the duelist's own budget moved to %d mid-round, want 5", d.ActionPoints())
	}
}

func TestGathersStackWithinARound(t *testing.T) {
	// Two in one round are worth +4. Deliberate: it is what puts a five-attack round in
	// reach without a ring discount, and that trade — a whole round spent setting up — is
	// the price of getting there.
	d := duelist(10, 11, 100) // 5 AP

	_, aAfter, _ := resolve(d, duelist(10, 10, 100),
		PlainCards(Gather, Gather), nil, 1)

	if aAfter.ActionPoints() != 9 {
		t.Errorf("budget after two Gathers = %d, want 9 (5 + 4)", aAfter.ActionPoints())
	}
}

func TestGatherDoesNotCompoundAcrossRounds(t *testing.T) {
	// Gathering every round is worth a flat +2, not +2 then +4 then +6. The bonus is
	// replaced at the boundary rather than added to, which is what bounds the ramp.
	a := duelist(10, 11, 500) // 5 AP
	b := duelist(10, 10, 500)

	_, a1, b1 := resolve(a, b, PlainCards(Gather), nil, 1)
	_, a2, _ := resolve(a1, b1, PlainCards(Gather), nil, 2)

	if a2.ActionPoints() != 7 {
		t.Errorf("budget after gathering twice in a row = %d, want a flat 7", a2.ActionPoints())
	}
}

func TestABonusLapsesIfItIsNotRenewed(t *testing.T) {
	a := duelist(10, 11, 500)
	b := duelist(10, 10, 500)

	_, a1, b1 := resolve(a, b, PlainCards(Gather), nil, 1)
	_, a2, _ := resolve(a1, b1, PlainCards(Strike), nil, 2)

	if a2.ActionPoints() != 5 {
		t.Errorf("budget after spending the bonus round = %d, want 5", a2.ActionPoints())
	}
}

func TestActionPointsFromSpeed(t *testing.T) {
	cases := []struct {
		spd, want int
	}{
		{0, 4},
		{15, 5}, // the shipped Monster1
		{20, 6}, // the shipped Fighter1
		{100, 14},
	}

	for _, c := range cases {
		if got := (Duelist{Spd: c.spd}).ActionPoints(); got != c.want {
			t.Errorf("ActionPoints at Spd %d = %d, want %d", c.spd, got, c.want)
		}
	}
}

func TestCanAffordEnforcesTheBudget(t *testing.T) {
	d := duelist(10, 11, 100) // 5 AP

	if !d.CanAfford(PlainCards(Heavy, Gather)) { // 4 + 1
		t.Error("Heavy + Gather costs 5 and should fit a 5 AP budget")
	}
	if d.CanAfford(PlainCards(Guard, Riposte)) { // 3 + 3
		t.Error("Guard + Riposte costs 6 and should not fit a 5 AP budget")
	}
}

func TestCategoriesCoverEveryAction(t *testing.T) {
	// A new action with no category would silently fall into attack and resolve in the
	// wrong phase. Pin the whole table instead.
	want := map[ActionKind]Category{
		Gather: CategoryPrepare,
		Sift:   CategoryPrepare,
		Guard:  CategoryPrepare,
		Ritual: CategoryPrepare,

		Jab:    CategoryAttack,
		Strike: CategoryAttack,
		Feint:  CategoryAttack,
		Heavy:  CategoryAttack,

		Brace:   CategoryDefend,
		Dodge:   CategoryDefend,
		Riposte: CategoryDefend,
		Retreat: CategoryDefend,
	}

	if len(AllActions) != len(want) {
		t.Fatalf("AllActions holds %d actions, the category table %d", len(AllActions), len(want))
	}
	for _, a := range AllActions {
		if got := a.Category(); got != want[a] {
			t.Errorf("%v is %v, want %v", a, got, want[a])
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
	a := duelist(100, 10, 100)
	b := duelist(10, 10, 10)

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
	events, _, bAfter := resolve(duelist(1000, 10, 100), duelist(1, 10, 5),
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
	a := duelist(10, 10, 100)
	b := duelist(10, 10, 100)

	resolve(a, b, PlainCards(Strike), PlainCards(Strike), 1)

	if a.CurrentLife != 100 || b.CurrentLife != 100 {
		t.Errorf("inputs mutated: a=%d b=%d, want both 100", a.CurrentLife, b.CurrentLife)
	}
}

func TestBothSidesDefendingDoesNotAlias(t *testing.T) {
	// resolveAction passes duelists by value; if that ever became pointer-based, a mutual
	// defense is where an aliasing bug would surface — one side's raised guard leaking
	// onto the other.
	a := duelist(10, 10, 100)
	b := duelist(10, 10, 100)

	_, a1, b1 := resolve(a, b, PlainCards(Guard), PlainCards(Dodge), 1)
	if !a1.Guarded {
		t.Error("A raised a guard and did not end the round with one")
	}
	if b1.Guarded {
		t.Error("B's dodge left it guarded; A's flag has leaked across")
	}
	if a1.DefendCount != 0 || b1.DefendCount != 1 || b1.Defends[0].Card != Dodge {
		t.Errorf("pending defends: a=%d b=%d (%v), want none and one Dodge",
			a1.DefendCount, b1.DefendCount, b1.Defends[0].Card)
	}
}

func TestRoundIsDeterministic(t *testing.T) {
	a := duelist(7, 13, 300)
	b := duelist(9, 17, 300)

	aPlan := PlainCards(Strike, Guard, Gather)
	bPlan := PlainCards(Jab, Riposte)

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
	a := duelist(10, 10, 100)
	b := duelist(10, 10, 100)

	_, aAfter, bAfter := resolve(a, b, nil, nil, 1)

	if aAfter.CurrentLife != 100 || bAfter.CurrentLife != 100 {
		t.Errorf("nobody acted but life changed: a=%d b=%d", aAfter.CurrentLife, bAfter.CurrentLife)
	}
}

// stockHand holds enough of everything a style could want, so the tests below stay tests of
// *preference* rather than of the draw.
//
// **Planners took no hand until 2026-08-11** — every style conjured its cards — and these
// assertions were written against that. Handing them a hand nothing can run short of keeps
// them asking the question they were written to ask: given the choice, what does a brute do
// differently from a swarm. What the deck actually does to a style is
// TestNoStylePlaysACardItWasNotDealt and the balance tool.
func stockHand() []Card {
	var hand []Card
	for i := 0; i < 6; i++ {
		hand = append(hand, PlainCards(Gather, Guard, Jab, Strike, Heavy)...)
	}
	return hand
}

func TestNoStylePlaysACardItWasNotDealt(t *testing.T) {
	// **The whole point of enemies having a deck.** A style is how a hand is played, not what
	// is played, so a planner handed three Jabs may only ever queue Jabs — and a planner
	// handed nothing may only queue nothing. Before decks landed a brute produced Heavies out
	// of thin air, which made `enemy_cards.json` a file that could never matter.
	hands := [][]Card{
		nil,
		PlainCards(Jab),
		PlainCards(Jab, Jab, Jab),
		PlainCards(Brace, Dodge, Retreat), // no attacks at all
		PlainCards(Guard, Gather),
		PlainCards(Heavy, Heavy, Jab, Brace),
	}

	for _, style := range PlanStyles() {
		for _, hand := range hands {
			d := duelist(10, 60, 100) // deliberately rich: the budget must not be the limit
			plan := PlanFor(style, d, hand)

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
					t.Fatalf("%v played %v from a hand of %v", style, a, hand)
				}
				left = append(left[:found], left[found+1:]...)
			}
		}
	}
}

func TestPlanningIsReproducible(t *testing.T) {
	// The determinism rule, at the planner. Nothing here may consult a map's iteration order
	// or a clock, so the same style and the same hand must plan the same round every time —
	// which is what lets a seeded run be replayed and what the balance tool depends on.
	hand := PlainCards(Heavy, Jab, Strike, Guard, Jab, Gather, Strike, Heavy)

	for _, style := range PlanStyles() {
		d := duelist(10, 30, 100)
		want := planKey(PlanFor(style, d, hand))
		for i := 0; i < 50; i++ {
			if got := planKey(PlanFor(style, d, hand)); got != want {
				t.Fatalf("%v planned %s then %s from the same hand", style, want, got)
			}
		}
	}
}

func TestEveryStyleObeysBothBounds(t *testing.T) {
	// The two bounds on a round apply to the opponent exactly as they apply to the player.
	// Swept across a wide speed range and across banked points, because both bounds move:
	// AP grows with Spd and with BonusAP, while the action cap does not, which is the
	// corner where an unbounded planner would overrun.
	for _, style := range PlanStyles() {
		for spd := 0; spd <= 120; spd += 3 {
			for _, bonus := range []int{0, 2, 4, 10} {
				d := duelist(10, spd, 100)
				d.BonusAP = bonus
				plan := PlanFor(style, d, stockHand())

				if !d.CanAfford(plan) {
					t.Errorf("%v at Spd %d bonus %d: plan costs %d, budget is %d",
						style, spd, bonus, CostOf(plan), d.ActionPoints())
				}
				if len(plan) > d.MaxActions() {
					t.Errorf("%v at Spd %d bonus %d: %d actions, cap is %d",
						style, spd, bonus, len(plan), d.MaxActions())
				}
				if len(plan) == 0 {
					t.Errorf("%v at Spd %d bonus %d: plan is empty", style, spd, bonus)
				}
			}
		}
	}
}

func TestEveryStyleFightsInADifferentShape(t *testing.T) {
	// The whole reason styles exist. If two of them produce the same round against the same
	// duelist then one of them is not earning its place, and the player has nothing new to
	// find an answer to.
	d := duelist(10, 15, 100) // the shipped Monster1: 5 AP

	seen := map[string]PlanStyle{}
	for _, style := range PlanStyles() {
		key := planKey(PlanFor(style, d, stockHand()))
		if prev, clash := seen[key]; clash {
			t.Errorf("%v and %v both plan %s", prev, style, key)
		}
		seen[key] = style
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

func TestSwarmAttacksMoreOftenThanBrute(t *testing.T) {
	// The specific thing styles were added to fix: a negation stops one blow, so an opponent
	// that attacks five times walks through the pair of them that shut a brute out
	// completely. Width is the mechanic, not damage.
	d := duelist(10, 15, 100)

	brute := PlanFor(StyleBrute, d, stockHand())
	swarm := PlanFor(StyleSwarm, d, stockHand())

	if len(swarm) <= len(brute) {
		t.Errorf("swarm plans %d actions, brute %d; swarm must be the wider round",
			len(swarm), len(brute))
	}
}

func TestSwarmSpendsSpareBudgetOnBiggerAttacksNotMoreOfThem(t *testing.T) {
	// A fast swarm has more points than it has slots. It must not waste them, and it must
	// not stop being a swarm to use them.
	d := duelist(10, 100, 100) // 14 AP against a cap of 5 actions
	plan := PlanFor(StyleSwarm, d, stockHand())

	if len(plan) != d.MaxActions() {
		t.Errorf("plan is %d actions, want the full %d", len(plan), d.MaxActions())
	}
	if left := d.ActionPoints() - CostOf(plan); left > 0 {
		t.Errorf("left %d points unspent with upgrades still available", left)
	}
	for _, a := range plan {
		if a.Category() != CategoryAttack {
			t.Errorf("swarm queued %v, which is %v — every slot should be an attack", a, a.Category())
		}
	}
}

func TestWardenGuardsAndStillAttacks(t *testing.T) {
	d := duelist(10, 15, 100) // 5 AP: Guard is 3, leaving a Strike
	plan := PlanFor(StyleWarden, d, stockHand())

	guards, attacks := 0, 0
	for _, a := range plan {
		switch a.Category() {
		case CategoryPrepare:
			guards++
		case CategoryAttack:
			attacks++
		}
	}
	if guards != 1 {
		t.Errorf("warden queued %d prepares, want exactly 1 Guard", guards)
	}
	if attacks == 0 {
		t.Error("warden queued no attacks; a pure turtle cannot win and is not a fight")
	}
}

func TestWardenFallsBackToAttackingWhenItCannotAffordAGuard(t *testing.T) {
	// A duelist too slow to pay 3 for a Guard must still take a turn rather than stand there.
	d := duelist(10, 0, 100)
	d.BonusAP = -2 // floors ActionPoints at 1

	if plan := PlanFor(StyleWarden, d, stockHand()); len(plan) == 0 {
		t.Error("warden with a 1 AP budget planned nothing")
	}
}

func TestTacticianBanksThenUnloads(t *testing.T) {
	// The rhythm is the point: a light round the player can punish, then an oversized one
	// they have to answer. It reads which round it is in off its own BonusAP, so it needs no
	// memory beyond what Gather already leaves behind.
	d := duelist(10, 15, 100) // 5 AP

	setup := PlanFor(StyleTactician, d, stockHand())
	prepares := 0
	for _, c := range setup {
		if c.Action == Gather {
			prepares++
		}
	}
	if prepares == 0 {
		t.Fatalf("setup round planned no Gather: %v", setup)
	}

	// Play the setup round through so the bank is real rather than assumed.
	_, after, _ := resolve(d, duelist(10, 10, 500), setup, nil, 1)
	if after.BonusAP == 0 {
		t.Fatal("the setup round banked nothing")
	}

	payoff := PlanFor(StyleTactician, after, stockHand())
	for _, c := range payoff {
		if c.Action == Gather {
			t.Errorf("payoff round is still gathering: %v", payoff)
		}
	}
	if CostOf(payoff) <= CostOf(setup) {
		t.Errorf("payoff round costs %d, setup round %d; the spike must be the bigger one",
			CostOf(payoff), CostOf(setup))
	}
}

func TestParsePlanStyleRoundTripsAndFallsBack(t *testing.T) {
	// Styles arrive as strings out of combatants.json, so a typo must produce a fightable
	// enemy rather than one that stands still.
	for _, style := range PlanStyles() {
		got, ok := ParsePlanStyle(style.String())
		if !ok || got != style {
			t.Errorf("%q parsed to %v (ok=%v), want %v", style.String(), got, ok, style)
		}
	}

	got, ok := ParsePlanStyle("wardne")
	if ok {
		t.Error("a typo reported itself as a known style")
	}
	if got != StyleBrute {
		t.Errorf("unknown style fell back to %v, want brute", got)
	}
}
