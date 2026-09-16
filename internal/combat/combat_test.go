package combat

import (
	"math/rand"
	"testing"
)

// duelist builds a full-health duelist for tests.
// resolve is ResolveRound with **no randomness at all**, which is what almost every test here
// wants: a nil source means the shock roll never fires, so everything that is still exact stays
// exact and a test about defenses is not a test about luck.
//
// The rolled path has its own tests, which pass a decided source deliberately — see below and
// status_test.go.
func resolve(a, b Duelist, aCards, bCards []Card, round int) ([]Event, Duelist, Duelist) {
	return ResolveRound(a, b, aCards, bCards, round, Sources{})
}

// resolveWith is the same round with a source, for the tests that are about the roll.
func resolveWith(rng *rand.Rand, a, b Duelist, aCards, bCards []Card, round int) ([]Event, Duelist, Duelist) {
	return ResolveRound(a, b, aCards, bCards, round, Sources{Roll: rng})
}

// fixedSource always hands back the same value, so a test can *say* "this attack misses" instead
// of hunting for a seed that happens to make it. A seed would work and would be unreadable: the
// test would assert a miss while naming a number, and retuning shockPct() could silently turn
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
// having to strip a color off.
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
		PlainCards(Bash, Bash), PlainCards(Bash, Bash), 1)

	want := []Side{SideA, SideA, SideB, SideB}
	if order := actionOrder(events); !sidesEqual(order, want) {
		t.Errorf("action order = %v, want %v even with B far faster", order, want)
	}
}

func TestATurnResolvesInCategoryOrder(t *testing.T) {
	// **Defenses, then attacks** *(owner's call, 2026-09-15)*, whatever order the cards were
	// queued in — a turn reads as raise the guard, then swing. It ran the other way until then, on
	// an argument that turned out not to depend on within-turn order at all: expireDefenses fires
	// at the start of a side's *own* turn, so a shield raised anywhere in this one is standing
	// through the opponent's either way. See Categories.
	a := duelist(10, 5, 500)
	b := duelist(10, 5, 500)

	queued := PlainCards(Brace, Bash, Block, Guard, Smash)
	events, _, _ := resolve(a, b, queued, nil, 1)

	want := PlainCards(Brace, Block, Guard, Bash, Smash)
	if got := playedCards(events); !cardsEqual(got, want) {
		t.Errorf("played %v, want %v", got, want)
	}
}

func TestQueuedOrderSurvivesInsideACategory(t *testing.T) {
	// Reordering across categories does nothing, but reordering *within* one is the whole
	// remaining point of dragging a card along the row — sequence hands will match on it.
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
	aPlan := PlainCards(Smash, Brace, Block)
	bPlan := PlainCards(Jab, Brace)

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
		PlainCards(Brace, Bash, Block),
		PlainCards(Guard, Smash, Brace))

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
	// Queued attack-first, resolved defend-first, so both slots move and neither Index does.
	order := ResolutionOrder(PlainCards(Jab, Brace), nil)

	if len(order) != 2 {
		t.Fatalf("got %d slots, want 2", len(order))
	}
	if order[0].Card.Concept != Brace || order[0].Index != 1 {
		t.Errorf("first slot = %v index %d, want Brace index 1", order[0].Card.Concept, order[0].Index)
	}
	if order[1].Card.Concept != Jab || order[1].Index != 0 {
		t.Errorf("second slot = %v index %d, want Jab index 0", order[1].Card.Concept, order[1].Index)
	}
}

func TestStrikeDealsDMGAsDamage(t *testing.T) {
	a := duelist(10, 5, 100)
	b := duelist(10, 5, 100)

	events, _, bAfter := resolve(a, b, PlainCards(Bash), nil, 1)

	got := firstDamage(t, events, SideA)
	if got.Amount != 10 {
		t.Errorf("Bash damage = %d, want 10 (attacker DMG)", got.Amount)
	}
	if bAfter.CurrentLife != 90 {
		t.Errorf("target life after Bash = %d, want 90", bAfter.CurrentLife)
	}
}

func TestTheTopTierHitsThreeTimesAsHardAsTheMiddleOne(t *testing.T) {
	a := duelist(10, 5, 500)
	b := duelist(1, 5, 500)

	strikeLog, _, _ := resolve(a, b, PlainCards(Bash), nil, 1)
	smashLog, _, _ := resolve(a, b, PlainCards(Smash), nil, 1)

	strike := firstDamage(t, strikeLog, SideA)
	smash := firstDamage(t, smashLog, SideA)

	// **3x, not the 2x it was until 2026-09-01** *(owner's call)*. The 3 AP cards were a point
	// dearer than the 2 AP ones for double the figure, which is the same damage per point and
	// therefore no reason to play one; at triple they buy something the budget cannot get by
	// spending the same points on cheaper cards.
	if smash.Amount != strike.Amount*3 {
		t.Errorf("Smash = %d, Bash = %d; want the 3 AP card to be triple", smash.Amount, strike.Amount)
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
	// **Nothing in the plan form hits back.** A defense is a wall, not a counter, so a turn made
	// of plans alone is a turn in which nobody is hurt.
	for _, a := range []ConceptID{Block, Brace, Guard} {
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

func TestSideBsDefenseProtectsItInTheFollowingRound(t *testing.T) {
	// The asymmetry the expiry rule exists for. B acts last, so its defense cannot cover
	// anything in the round it was raised — it has to survive the boundary and cover A's
	// next turn, or the card would be worthless in B's hands.
	//
	// **It was a percentage guard until 2026-09-16 and is a shield now.** The verb went; the
	// asymmetry did not, and it is the reason expireDefenses fires at the start of a side's own
	// turn rather than at the round boundary.
	a := duelist(10, 5, 500)
	b := duelist(10, 5, 500)

	_, a1, b1 := resolve(a, b, nil, PlainCards(Brace), 1)
	if b1.Shields != 1 {
		t.Fatalf("side B holds %d shields after raising a Brace, want it to survive the round", b1.Shields)
	}

	round2, _, _ := resolve(a1, b1, PlainCards(Bash), nil, 2)
	if !hasKind(round2, KindBlocked) {
		t.Error("A's blow was not eaten — B's carried shield did not cover A's next turn")
	}
}

func TestAnIdleDuelistLosesItsDefense(t *testing.T) {
	// A turn happens whether or not anything is queued into it, and a defense expires at
	// the start of its owner's turn. Standing still therefore does not bank one.
	a := duelist(10, 5, 500)
	b := duelist(10, 5, 500)

	_, a1, b1 := resolve(a, b, PlainCards(Brace), nil, 1)

	round2, _, _ := resolve(a1, b1, nil, PlainCards(Bash), 2)
	if hit := firstDamage(t, round2, SideB); hit.Amount != 10 {
		t.Errorf("hit in round 2 = %d, want full 10 — an idle turn still expires a defense", hit.Amount)
	}
	if hasKind(round2, KindBlocked) {
		t.Error("the shield ate a blow in round 2 — an idle turn should have expired it")
	}
}

func TestDefensesExpireWithTheTurnTheyCovered(t *testing.T) {
	// An unspent shield does not bank. It covers the opponent's next turn and then goes.
	a := duelist(10, 5, 500)
	b := duelist(10, 5, 500)

	_, a1, b1 := resolve(a, b, PlainCards(Brace), nil, 1)
	if a1.Shields != 1 {
		t.Fatalf("A ended round 1 holding %d shields, want the one it raised", a1.Shields)
	}

	round2, _, _ := resolve(a1, b1, PlainCards(Jab), PlainCards(Bash), 2)
	if a2 := damageCount(round2); a2 != 2 {
		t.Errorf("damage events in round 2 = %d, want 2 — the defense expired at A's turn", a2)
	}
}

// **A round's budget is the stat and nothing else** *(2026-08-31)*, so this is a short test about
// a number that is now a constant for the whole fight. It stays because ActionPoints is a method
// rather than a field read, and a method is somewhere a relic can one day bite.
func TestActionPointsIsTheStat(t *testing.T) {
	for _, actions := range []int{4, 5, 6} {
		d := Duelist{Actions: actions}
		if got := d.ActionPoints(); got != actions {
			t.Errorf("ActionPoints at Actions %d = %d, want %d", actions, got, actions)
		}
	}
}

func TestCanAffordEnforcesTheBudget(t *testing.T) {
	d := duelist(10, 5, 100) // 5 AP

	if !d.CanAfford(PlainCards(Smash, Thrust)) { // 3 + 2
		t.Error("Smash + Thrust costs 5 and should fit a 5 AP budget")
	}
	if d.CanAfford(PlainCards(Guard, Skewer)) { // 3 + 3
		t.Error("Guard + Skewer costs 6 and should not fit a 5 AP budget")
	}
}

func TestCategoriesCoverEveryPlayerConcept(t *testing.T) {
	// A card whose verb has no category would silently fall into attack and resolve in the wrong
	// phase. Pin the whole player deck instead — the enemy decks are checked by the same rule, but
	// they are data and this test would become a copy of enemies.json.
	want := map[ConceptID]Category{
		Jab:    CategoryAttack,
		Thrust: CategoryAttack,
		Skewer: CategoryAttack,
		Poke:   CategoryAttack,
		Impale: CategoryAttack,

		Cut:    CategoryAttack,
		Slice:  CategoryAttack,
		Cleave: CategoryAttack,
		Nick:   CategoryAttack,
		Sever:  CategoryAttack,

		Thump:     CategoryAttack,
		Bash:      CategoryAttack,
		Smash:     CategoryAttack,
		Tap:       CategoryAttack,
		Pulverize: CategoryAttack,

		Flinch: CategoryDefend,
		Brace:  CategoryDefend,
		Block:  CategoryDefend,
		Guard:  CategoryDefend,
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
	// A queues three bashes into an enemy that cannot survive the hand they form. B must never
	// get its reply.
	//
	// **The three Bashes are one blow, and they are still three beats.** They are announced
	// before the hand is scored, so A's turn contributes three actions and B's contributes none —
	// which is the honest reading of "the round stopped at the kill" now that a turn cannot be
	// cut off partway through its own attack.
	a := duelist(100, 5, 100)
	b := duelist(10, 5, 10)

	events, _, bAfter := resolve(a, b,
		PlainCards(Bash, Bash, Bash), PlainCards(Bash), 1)

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
		PlainCards(Bash), nil, 1)

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

	resolve(a, b, PlainCards(Bash), PlainCards(Bash), 1)

	if a.CurrentLife != 100 || b.CurrentLife != 100 {
		t.Errorf("inputs mutated: a=%d b=%d, want both 100", a.CurrentLife, b.CurrentLife)
	}
}

func TestBothSidesDefendingDoesNotAlias(t *testing.T) {
	// resolveAction passes duelists by value; if that ever became pointer-based, a mutual
	// defense is where an aliasing bug would surface — one side's raised shields leaking
	// onto the other. **Two different counts on purpose**, so a leak reads as a wrong number
	// rather than as a coincidence.
	a := duelist(10, 5, 100)
	b := duelist(10, 5, 100)

	_, a1, b1 := resolve(a, b, PlainCards(Block), PlainCards(Brace), 1)
	if a1.Shields != 2 {
		t.Errorf("A raised a Block and ended the round holding %d shields, want 2", a1.Shields)
	}
	if b1.Shields != 1 {
		t.Errorf("B raised a Brace and ended the round holding %d shields, want 1", b1.Shields)
	}
}

func TestRoundIsDeterministic(t *testing.T) {
	a := duelist(7, 5, 300)
	b := duelist(9, 5, 300)

	aPlan := PlainCards(Bash, Guard, Block)
	bPlan := PlainCards(Jab, Brace)

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
		hand = append(hand, PlainCards(Block, Guard, Jab, Bash, Smash)...)
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
		PlainCards(Guard, Block, Brace), // no attacks at all
		PlainCards(Guard, Block),
		PlainCards(Smash, Smash, Jab, Guard),
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
	hand := PlainCards(Smash, Jab, Bash, Guard, Jab, Block, Bash, Smash)

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
	// Swept across a wide budget range, because the budget moves with the stat while the action
	// cap does not — which is the corner where an unbounded planner would overrun.
	for actions := 0; actions <= 16; actions++ {
		d := duelist(10, actions, 100)
		plan := PlanFor(d, stockHand())

		if !d.CanAfford(plan) {
			t.Errorf("at %d actions: plan costs %d, budget is %d",
				actions, d.CostOf(plan), d.ActionPoints())
		}
		if len(plan) > d.MaxActions() {
			t.Errorf("at %d actions: %d actions, cap is %d", actions, len(plan), d.MaxActions())
		}
		if d.ActionPoints() > 0 && len(plan) == 0 {
			t.Errorf("at %d actions: plan is empty", actions)
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
	// that only maximized damage would never raise a shield, so every defensive card authored into
	// the roster would sit in a discard pile forever.
	d := duelist(10, 5, 100) // 2 AP of attack, 3 left over
	hand := PlainCards(Bash, Guard)

	plan := PlanFor(d, hand)

	attacks, shields := 0, 0
	for _, c := range plan {
		switch c.Spec().Verb {
		case VerbAttack:
			attacks++
		case VerbShield:
			shields++
		}
	}
	if attacks != 1 || shields != 1 {
		t.Errorf("planned %v, want the Bash and the Guard", planKey(plan))
	}
}
