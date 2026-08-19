package combat

import (
	"testing"
)

// The four element statuses, the ring that switches each of them on, and the lifecycle all of
// them share.
//
// **Every test here is written against the rule rather than the constant** where it can be —
// a chill costs `chillPct()` cards, not "one" — so tuning a number does not fail a
// test that was checking the mechanic. The exceptions are the ones pinning a *relationship*
// (a burn ticks twice, a status is gone by the round after, a second hit does not stack), which
// is the thing that must not move without somebody deciding it should.

// **These tests talk in colours and the rules no longer do** *(2026-08-17)*. A status is its own
// record and a ring is what connects the two, so the four rings below are built here — this package
// cannot read `rings.json`, which is parsed in `internal/session` — and every element-shaped helper
// goes through them. What that buys is that a test about the *lifecycle* stays written the way the
// mechanic is discussed, while the decoupling is exercised by the wiring underneath it.

// statusOf is the status the named colour's ring applies, by the pairing `rings.json` ships.
func statusOf(e Element) StatusID {
	switch e {
	case Fire:
		return MustStatus("burning")
	case Ice:
		return MustStatus("chilled")
	case Lightning:
		return MustStatus("shocked")
	case Earth:
		return MustStatus("weighted")
	default:
		return NoStatus
	}
}

// The four figures the tests used to read off constants in status.go. They are file entries now, so
// a test naming the rule rather than the number reads it back out of the registry.
func burnPct() int      { return StatusOf(statusOf(Fire)).Amount }
func chillPct() int     { return StatusOf(statusOf(Ice)).Amount }
func shockPct() int     { return StatusOf(statusOf(Lightning)).Amount }
func statusRounds() int { return StatusOf(statusOf(Fire)).Rounds }
func weightPct() int    { return StatusOf(statusOf(Earth)).Amount }

// testRings is the four elemental rings, registered once for the whole test binary. **Registered
// rather than faked**, so what the tests exercise is the same `RegisterRing` path the game loads
// through — a rule the registry would refuse fails here too.
var testRings = registerTestRings()

func registerTestRings() map[Element]RingID {
	out := map[Element]RingID{}
	for _, e := range []Element{Fire, Ice, Lightning, Earth} {
		id, err := RegisterRing("test."+e.String(), "Test "+e.String(), []RingRule{{
			When: MomentAttackLands,
			If:   RingCondition{Element: e, HasElement: true},
			Then: []RingEffect{{Do: DoApplyStatus, Status: statusOf(e)}},
		}})
		if err != nil {
			panic(err)
		}
		out[e] = id
	}
	return out
}

// wearing returns the duelist with rings for the named elements on. **Every status test needs
// one**, which is the whole point of the 2026-08-16 rule: without a ring an element is a border
// colour and a hand axis and nothing else.
func wearing(d Duelist, es ...Element) Duelist {
	for _, e := range es {
		d = d.Wearing(WornRing{Ring: testRings[e]})
	}
	return d
}

// ringed is a duelist wearing all four, for tests about the lifecycle rather than about rings.
func ringed(d Duelist) Duelist { return wearing(d, Fire, Ice, Lightning, Earth) }

// statusEvents returns the KindStatus events for the status one colour's ring applies.
func statusEvents(events []Event, e Element) []Event {
	var out []Event
	for _, ev := range events {
		if ev.Kind == KindStatus && ev.Status == statusOf(e) {
			out = append(out, ev)
		}
	}
	return out
}

func countKind(events []Event, k EventKind) int {
	n := 0
	for _, e := range events {
		if e.Kind == k {
			n++
		}
	}
	return n
}

// --- the ring gate ---------------------------------------------------------------------------

func TestAnElementAppliesNothingWithoutItsRing(t *testing.T) {
	// **The headline rule** *(2026-08-16)*. A coloured attack from a duelist wearing no ring is a
	// plain attack: it still counts toward the mix multiplier, and it leaves nothing behind.
	for _, e := range []Element{Fire, Ice, Lightning, Earth} {
		a, b := duelist(10, 5, 500), duelist(10, 5, 500)

		events, _, bAfter := resolve(a, b, []Card{Of(Strike, e)}, nil, 1)

		if n := len(statusEvents(events, e)); n != 0 {
			t.Errorf("an unringed %v Strike applied %d statuses, want 0", e, n)
		}
		if bAfter.Statuses[statusOf(e)].Active() {
			t.Errorf("an unringed %v Strike left a %v status behind", e, e)
		}
	}
}

func TestOnlyTheRingWornSwitchesItsOwnElementOn(t *testing.T) {
	// One ring is one element. A duelist wearing fire and swinging a rainbow lands a burn and
	// nothing else, which is what makes the second and third rings worth buying.
	a := wearing(duelist(10, 8, 500), Fire)
	b := duelist(10, 5, 500)

	_, _, bAfter := resolve(a, b,
		[]Card{Of(Jab, Fire), Of(Jab, Ice), Of(Jab, Lightning), Of(Jab, Earth)}, nil, 1)

	if !bAfter.Statuses[statusOf(Fire)].Active() {
		t.Error("the fire ring's own colour left no burn")
	}
	for _, e := range []Element{Ice, Lightning, Earth} {
		if bAfter.Statuses[statusOf(e)].Active() {
			t.Errorf("a %v card left a status on a duelist wearing no %v ring", e, e)
		}
	}
}

func TestTheRingIsReadOffTheAttackerNotTheVictim(t *testing.T) {
	// Your ring makes your attacks burn. It does nothing about attacks aimed at you — otherwise a
	// ring would be a liability and buying one would be a decision with a wrong answer.
	a := duelist(10, 5, 500)
	b := wearing(duelist(10, 5, 500), Fire)

	_, _, bAfter := resolve(a, b, []Card{Of(Strike, Fire)}, nil, 1)

	if bAfter.Statuses[statusOf(Fire)].Active() {
		t.Error("the victim's own fire ring lit a burn on themselves")
	}
}

func TestABasicAttackAppliesNothingHoweverManyRingsAreWorn(t *testing.T) {
	// Basic is the absence of an element rather than a fifth colour, so no elemental ring can match
	// it. A duelist wearing all four and swinging a plain card leaves nothing behind — which is what
	// keeps "drab lands none" true from the ring's side as well as the card's.
	a, b := ringed(duelist(10, 5, 500)), duelist(10, 5, 500)

	events, _, bAfter := resolve(a, b, []Card{Plain(Strike)}, nil, 1)

	if n := countKind(events, KindStatus); n != 0 {
		t.Errorf("a basic Strike applied %d statuses, want 0", n)
	}
	for _, id := range AllStatuses() {
		if bAfter.Statuses[id].Active() {
			t.Errorf("a basic Strike left %s behind", StatusOf(id).Key)
		}
	}
}

func TestARingNotWornDoesNothing(t *testing.T) {
	// WearsRing is a query over the worn set rather than a flag read, so this is the shape of the
	// gate now: a registered ring nobody put on is a ring that never fires.
	d := wearing(duelist(10, 5, 500), Fire)

	if !d.WearsRing(testRings[Fire]) {
		t.Error("a duelist wearing the fire ring reported not wearing it")
	}
	if d.WearsRing(testRings[Ice]) {
		t.Error("a duelist reported wearing a ring nobody put on")
	}
}

// --- what applies a status -------------------------------------------------------------------

func TestALandedElementalAttackAppliesItsStatus(t *testing.T) {
	// The trigger rule: an attack that connects applies its element, given the ring, and nothing
	// else does.
	for _, e := range []Element{Fire, Ice, Lightning, Earth} {
		a, b := ringed(duelist(10, 5, 500)), duelist(10, 5, 500)
		events, _, bAfter := resolve(a, b, []Card{Of(Strike, e)}, nil, 1)

		if got := statusEvents(events, e); len(got) != 1 {
			t.Errorf("a %v Strike raised %d status events, want 1", e, len(got))
		}
		if !bAfter.Statuses[statusOf(e)].Active() {
			t.Errorf("a %v Strike left no %v status on the target", e, e)
		}
	}
}

func TestOnlyAttacksApplyAStatus(t *testing.T) {
	// **Decided 2026-08-12**: a plan card carries its element for hands and for the ring
	// discount and applies nothing. Otherwise a 1-AP Prepare would be as good a status delivery
	// as a 1-AP Jab, and the plan phase would quietly become the status engine.
	for _, a := range []ConceptID{Prepare, Plan, Defend} {
		attacker, target := ringed(duelist(10, 8, 500)), duelist(10, 5, 500)
		events, _, bAfter := resolve(attacker, target, []Card{Of(a, Fire)}, nil, 1)

		if n := len(statusEvents(events, Fire)); n != 0 {
			t.Errorf("a fire %v applied a status %d times", a, n)
		}
		if bAfter.Statuses[statusOf(Fire)].Active() {
			t.Errorf("a fire %v left a burn on the opponent", a)
		}
	}
}

func TestABlockedBlowStillAppliesItsStatus(t *testing.T) {
	// **The status lands because the hand formed, not because the blow hurt** *(2026-08-14)*.
	// This reverses the rule that stood while defends *negated*: back then a stopped attack
	// carried nothing in, because nothing arrived. A Defend takes 50% off — so making the status
	// conditional on the final figure would let a defensive card silently un-apply an element the
	// attacker had already paid for, and under one blow per turn that would be every defensive
	// card in the game.
	for _, defence := range []ConceptID{Defend} {
		a, b := ringed(duelist(10, 5, 500)), duelist(10, 8, 500)

		// B raises the defence in round one, A swings into it in round two.
		_, a1, b1 := resolve(a, b, nil, []Card{Plain(defence)}, 1)
		events, _, bAfter := resolve(a1, b1, []Card{Of(Strike, Fire)}, nil, 2)

		if n := len(statusEvents(events, Fire)); n != 1 {
			t.Errorf("a Strike met by a %v applied its burn %d times, want 1", defence, n)
		}
		if !bAfter.Statuses[statusOf(Fire)].Active() {
			t.Errorf("a Strike met by a %v left no burn", defence)
		}
	}
}

func TestOneColourInAHandIsOneStatusHoweverManyCardsCarryIt(t *testing.T) {
	// The mix counts **distinct** colours, not coloured cards, so this is the rule that decides
	// status volume now. Two fire Jabs are a mono fire Pair and land one burn — where under the
	// per-card model they landed two.
	a, b := ringed(duelist(10, 8, 500)), duelist(10, 5, 500)

	events, _, bAfter := resolve(a, b, []Card{Of(Jab, Fire), Of(Jab, Fire)}, nil, 1)

	if n := len(statusEvents(events, Fire)); n != 1 {
		t.Errorf("two fire cards in one hand applied %d burns, want 1", n)
	}
	want := statusAmount(StatusOf(statusOf(Fire)), a)
	if got := bAfter.Statuses[statusOf(Fire)].Amount; got != want {
		t.Errorf("a mono fire hand burned for %d, want %d", got, want)
	}
}

func TestEachColourInTheHandLandsItsOwnStatus(t *testing.T) {
	// The other end of the same rule: a duo hand lands both, which is what the mix multiplier is
	// paying for besides damage — given a ring for each colour.
	a, b := ringed(duelist(10, 8, 500)), duelist(10, 5, 500)

	_, _, bAfter := resolve(a, b, []Card{Of(Jab, Fire), Of(Jab, Ice)}, nil, 1)

	if !bAfter.Statuses[statusOf(Fire)].Active() {
		t.Error("a duo fire/ice hand left no burn")
	}
	if !bAfter.Statuses[statusOf(Ice)].Active() {
		t.Error("a duo fire/ice hand left no chill")
	}
}

func TestACardOutsideTheHandCarriesNoColour(t *testing.T) {
	// Attack cards that build no hand are announced and contribute nothing — not damage and not
	// an element. `Strike, Jab, Strike` is a Strike Pair and the Jab is not in it, so a fire Jab
	// alongside two plain Strikes burns nobody.
	a, b := ringed(duelist(10, 8, 500)), duelist(10, 5, 500)

	events, _, bAfter := resolve(a, b,
		[]Card{Plain(Strike), Of(Jab, Fire), Plain(Strike)}, nil, 1)

	if n := len(statusEvents(events, Fire)); n != 0 {
		t.Errorf("a fire Jab outside the hand applied %d burns, want 0", n)
	}
	if bAfter.Statuses[statusOf(Fire)].Active() {
		t.Error("a card that earned nothing still left its element behind")
	}
}

func TestAHalvedAttackStillAppliesItsStatus(t *testing.T) {
	// **The status lands because the blow did, not because it hurt.** A Defend halves the hit
	// and the hit still connected, so making the status conditional on the final figure would
	// let a defensive card silently un-apply an element the attacker had already paid for.
	a, b := ringed(duelist(10, 5, 500)), duelist(10, 8, 500)

	_, a1, b1 := resolve(a, b, nil, []Card{Plain(Defend)}, 1)
	events, _, bAfter := resolve(a1, b1, []Card{Of(Strike, Ice)}, nil, 2)

	if n := len(statusEvents(events, Ice)); n != 1 {
		t.Errorf("a halved Strike applied its chill %d times, want 1", n)
	}
	if !bAfter.Statuses[statusOf(Ice)].Active() {
		t.Error("a halved ice Strike left no chill")
	}
}

// --- the lifecycle ---------------------------------------------------------------------------

func TestASecondHitResetsTheClockAndDoesNotStack(t *testing.T) {
	// **Nothing stacks as of 2026-08-16.** Two fire hits burn for what one burns for; what the
	// second buys is the clock going back to full. Amounts added until then, which made a status
	// something to pile on rather than something to keep up.
	a, b := ringed(duelist(10, 8, 500)), duelist(10, 5, 500)

	_, a1, b1 := resolve(a, b, []Card{Of(Jab, Fire)}, nil, 1)
	one := b1.Statuses[statusOf(Fire)].Amount

	_, _, b2 := resolve(a1, b1, []Card{Of(Jab, Fire)}, nil, 2)

	if got := b2.Statuses[statusOf(Fire)].Amount; got != one {
		t.Errorf("two fire hits burn for %d against one hit's %d — nothing stacks", got, one)
	}
	// Refreshed rather than added: statusRounds(), less the one round-end that has passed.
	if got, want := b2.Statuses[statusOf(Fire)].Rounds, statusRounds()-1; got != want {
		t.Errorf("two fire hits left %d rounds, want %d — the clock resets, it does not add",
			got, want)
	}
}

func TestAStatusIsGoneByTheEndOfTheRoundAfterItLanded(t *testing.T) {
	// The lifecycle, pinned as a relationship rather than as a number. A status has to survive
	// the round-end of the round that applied it — otherwise one applied by side B, who acts
	// second, would never bite anything at all — and it must not survive the next one.
	a, b := ringed(duelist(10, 5, 500)), duelist(10, 5, 500)

	_, a1, b1 := resolve(a, b, []Card{Of(Strike, Ice)}, nil, 1)
	if !b1.Statuses[statusOf(Ice)].Active() {
		t.Fatal("the chill did not survive the round it was applied in")
	}

	_, _, b2 := resolve(a1, b1, nil, nil, 2)
	if b2.Statuses[statusOf(Ice)].Active() {
		t.Error("the chill outlived the round after the one that applied it")
	}
}

// --- ice ---------------------------------------------------------------------------------------

func TestIceTakesACardOffTheFrontOfTheTurn(t *testing.T) {
	// **Ice takes a card, not a point** *(2026-08-16)*. It reuses the stagger machinery, so what a
	// chilled duelist loses is announced as KindChilled — the front of a turn is its attacks, so
	// what goes is the blow.
	a, b := wearing(duelist(10, 5, 500), Ice), duelist(10, 5, 500)

	plain, _, _ := resolve(a, b, nil, []Card{Plain(Strike), Plain(Strike)}, 1)
	if n := countKind(plain, KindChilled); n != 0 {
		t.Fatalf("an unchilled turn lost %d cards, want 0", n)
	}

	events, _, _ := resolve(a, b, []Card{Of(Jab, Ice)}, []Card{Plain(Strike), Plain(Strike)}, 1)

	if got, want := countKind(events, KindChilled), chillPct(); got != want {
		t.Errorf("a chilled turn lost %d cards, want %d", got, want)
	}
}

func TestAChillBitesEveryTurnItOutlives(t *testing.T) {
	// **The difference between a chill and a stagger**: a stagger is spent when it bites, a chill
	// bites on every turn it is still running for. B is hit in round one and acts after A, so it
	// loses a card that round and again in round two.
	a, b := wearing(duelist(10, 5, 500), Ice), duelist(10, 5, 500)
	bTurn := []Card{Plain(Strike), Plain(Strike)}

	r1, a1, b1 := resolve(a, b, []Card{Of(Jab, Ice)}, bTurn, 1)
	if n := countKind(r1, KindChilled); n != chillPct() {
		t.Fatalf("round 1 lost %d cards to the chill, want %d", n, chillPct())
	}

	r2, a2, b2 := resolve(a1, b1, nil, bTurn, 2)
	if n := countKind(r2, KindChilled); n != chillPct() {
		t.Errorf("round 2 lost %d cards to the chill, want %d — a chill bites while it lasts",
			n, chillPct())
	}

	r3, _, _ := resolve(a2, b2, nil, bTurn, 3)
	if n := countKind(r3, KindChilled); n != 0 {
		t.Errorf("round 3 lost %d cards, want 0 — the chill has expired", n)
	}
}

// **A second hit resets the clock rather than deepening the chill** *(2026-08-17)*. Nothing stacks,
// so a duelist hit twice still loses one card a turn — for longer.
func TestASecondIceHitDoesNotDeepenTheChill(t *testing.T) {
	a, b := wearing(duelist(10, 5, 500), Ice), duelist(10, 5, 500)
	bTurn := []Card{Plain(Strike), Plain(Strike), Plain(Strike)}

	_, a1, b1 := resolve(a, b, []Card{Of(Jab, Ice)}, nil, 1)

	events, _, _ := resolve(a1, b1, []Card{Of(Jab, Ice)}, bTurn, 2)

	if got, want := countKind(events, KindChilled), chillPct(); got != want {
		t.Errorf("a twice-chilled turn lost %d cards, want %d", got, want)
	}
}

func TestAStatusNoLongerTouchesTheBudget(t *testing.T) {
	// Ice cut the action-point budget until 2026-08-16. Nothing does now, and the check is here
	// rather than deleted because a duelist whose budget quietly moved is the failure this rule
	// change could reintroduce without anyone noticing.
	a, b := ringed(duelist(10, 5, 500)), duelist(10, 5, 500)
	before := b.ActionPoints()

	_, _, bAfter := resolve(a, b, []Card{Of(Strike, Ice)}, nil, 1)

	if got := bAfter.ActionPoints(); got != before {
		t.Errorf("a chilled duelist has %d AP, want %d — statuses do not touch the budget", got, before)
	}
}

// --- lightning ---------------------------------------------------------------------------------

func TestAShockIsARollAndTheSourceDecidesIt(t *testing.T) {
	// **A roll again as of 2026-08-14**, reversing the deterministic version taken two days
	// earlier. One blow per turn is what forced it: a certain miss used to delete one attack out
	// of several and now deletes the whole turn, so a 1 AP lightning Jab could erase an 8 AP
	// Barrage outright.
	//
	// The same shocked duelist and the same turn, twice, with the two rolls decided rather than
	// seeded — see fixedSource.
	a, b := wearing(duelist(10, 5, 500), Lightning), duelist(10, 5, 500)

	_, a1, b1 := resolve(a, b, []Card{Of(Strike, Lightning)}, nil, 1)
	if !b1.Statuses[statusOf(Lightning)].Active() {
		t.Fatal("the lightning Strike left no shock")
	}

	missed, aMissed, _ := resolveWith(alwaysMisses(), a1, b1, nil, []Card{Plain(Strike)}, 2)
	landed, aLanded, _ := resolveWith(neverMisses(), a1, b1, nil, []Card{Plain(Strike)}, 2)

	if n := countKind(missed, KindMissed); n != 1 {
		t.Errorf("a losing roll missed %d times, want 1", n)
	}
	if aMissed.CurrentLife != a1.CurrentLife {
		t.Errorf("a missed attack still dealt %d damage", a1.CurrentLife-aMissed.CurrentLife)
	}

	if n := countKind(landed, KindMissed); n != 0 {
		t.Errorf("a winning roll missed %d times, want 0", n)
	}
	if aLanded.CurrentLife >= a1.CurrentLife {
		t.Error("a shocked attack that passed its roll dealt no damage")
	}
}

func TestAShockIsAFlatChanceThatCanNeverBeCertain(t *testing.T) {
	// **The chance is the Amount now that nothing stacks**, and the cap that used to hold four
	// stacks under a certainty went with the stacking. What has to stay true is the reason the cap
	// existed: a defence that always works deletes a whole opposing turn for one card.
	if shockPct() >= 100 {
		t.Errorf("a shock misses %d%% of the time, which is the certain miss this replaced",
			shockPct())
	}
	if shockPct() <= 0 {
		t.Errorf("a shock misses %d%% of the time, so lightning does nothing", shockPct())
	}

	d := duelist(10, 5, 500)
	if attackMisses(d, alwaysMisses()) {
		t.Error("an unshocked duelist missed")
	}

	d.Statuses[statusOf(Lightning)] = Status{Amount: shockPct(), Rounds: 1}
	if !attackMisses(d, alwaysMisses()) {
		t.Error("a shocked duelist passed a roll it could not pass")
	}
	if attackMisses(d, neverMisses()) {
		t.Error("a shocked duelist failed a roll it could not fail")
	}
	if attackMisses(d, nil) {
		t.Error("a nil source produced a roll")
	}
}

func TestAShockRollsAgainOnEveryAttackItOutlives(t *testing.T) {
	// **Nothing is spent** *(2026-08-16)*. A stack used to be consumed by the first attack whether
	// or not the roll landed; with nothing to wear down, that would make a two-round status one
	// that reliably lasted one attack.
	// B is shocked during A's turn and acts later the same round, so it gets one attack that
	// round and one more in the round after — two rolls out of one hit.
	a, b := wearing(duelist(10, 5, 500), Lightning), duelist(10, 5, 500)

	r1, a1, b1 := resolveWith(alwaysMisses(), a, b,
		[]Card{Of(Jab, Lightning)}, []Card{Plain(Strike)}, 1)
	if n := countKind(r1, KindMissed); n != 1 {
		t.Fatalf("round 1 missed %d times, want 1", n)
	}
	if !b1.Statuses[statusOf(Lightning)].Active() {
		t.Fatal("the roll consumed the shock")
	}

	r2, _, _ := resolveWith(alwaysMisses(), a1, b1, nil, []Card{Plain(Strike)}, 2)
	if n := countKind(r2, KindMissed); n != 1 {
		t.Errorf("round 2 missed %d times, want 1 — a shock rolls while it lasts", n)
	}
}

func TestAShockDeletesTheWholeTurnBecauseATurnIsOneBlow(t *testing.T) {
	// Under the multi-blow model a shock cancelled one attack out of several. A turn now resolves
	// a single blow, so a landed roll deletes all of it — which is the whole reason the certain
	// miss had to become a roll. See MECHANICS.md.
	a, b := wearing(duelist(10, 5, 500), Lightning), duelist(10, 8, 500)

	_, a1, b1 := resolve(a, b, []Card{Of(Jab, Lightning)}, nil, 1)
	events, aAfter, _ := resolveWith(alwaysMisses(), a1, b1, nil, []Card{Plain(Jab), Plain(Jab)}, 2)

	if n := countKind(events, KindMissed); n != 1 {
		t.Errorf("%d misses, want exactly 1 — a turn has one attack to miss", n)
	}
	if n := countKind(events, KindDamage); n != 0 {
		t.Errorf("%d damage events, want 0 — the blow that missed was the whole turn", n)
	}
	if aAfter.CurrentLife != a1.CurrentLife {
		t.Errorf("a missed turn still dealt %d damage", a1.CurrentLife-aAfter.CurrentLife)
	}
}

func TestAMissedAttackDoesNothingElseEither(t *testing.T) {
	// The miss happens before any defence is spent and before any status is applied. The attack
	// did not occur.
	//
	// **What it does not undo is the hand's own reward** — a stagger is paid on forming the hand,
	// not on connecting. That is deliberate and is pinned in hand_test.go.
	a := wearing(duelist(10, 5, 500), Lightning)
	b := wearing(duelist(10, 8, 500), Fire)

	_, a1, b1 := resolve(a, b, []Card{Of(Jab, Lightning)}, nil, 1)

	// B is shocked and swings a fire Strike; A is holding a Defend for it.
	events, _, bAfter := resolveWith(alwaysMisses(), a1, b1,
		[]Card{Plain(Defend)}, []Card{Of(Strike, Fire)}, 2)

	if n := countKind(events, KindNegated); n != 0 {
		t.Error("a missed attack still spent the defence that was waiting for it")
	}
	if n := len(statusEvents(events, Fire)); n != 0 {
		t.Error("a missed attack still applied its burn")
	}
	if bAfter.Statuses[statusOf(Fire)].Active() {
		t.Error("a missed attack left its element on somebody")
	}
}

// --- fire ---------------------------------------------------------------------------------------

func TestABurnIsAShareOfTheAttackersDMG(t *testing.T) {
	// **The tick is read off whoever lit it, and frozen when it lands.** A stronger duelist burns
	// harder; the victim carries the number rather than a pointer back to the attacker.
	strong := wearing(duelist(100, 5, 5000), Fire)
	weak := wearing(duelist(10, 5, 5000), Fire)
	target := duelist(10, 5, 5000)

	_, _, hot := resolve(strong, target, []Card{Of(Jab, Fire)}, nil, 1)
	_, _, mild := resolve(weak, target, []Card{Of(Jab, Fire)}, nil, 1)

	if want := strong.DMG * burnPct() / 100; hot.Statuses[statusOf(Fire)].Amount != want {
		t.Errorf("a DMG %d duelist burned for %d, want %d",
			strong.DMG, hot.Statuses[statusOf(Fire)].Amount, want)
	}
	if hot.Statuses[statusOf(Fire)].Amount <= mild.Statuses[statusOf(Fire)].Amount {
		t.Errorf("a DMG %d burn of %d did not beat a DMG %d burn of %d",
			strong.DMG, hot.Statuses[statusOf(Fire)].Amount, weak.DMG, mild.Statuses[statusOf(Fire)].Amount)
	}
}

func TestABurnAlwaysTicksForSomething(t *testing.T) {
	// The floor is the rule Jab's damage already follows: a duelist under 10 DMG would light a
	// burn worth nothing, and a status that lands and does nothing is worse than one that does not
	// land.
	feeble := wearing(duelist(1, 5, 500), Fire)

	if got := statusAmount(StatusOf(statusOf(Fire)), feeble); got < 1 {
		t.Errorf("a DMG %d duelist lights a burn of %d, want at least 1", feeble.DMG, got)
	}
}

func TestFireTicksAtTheEndOfEveryRoundItSurvives(t *testing.T) {
	// The DoT: it lands at end of round, including the end of the round it was applied in, and
	// it persists across the boundary. Two ticks from one hit at the current duration.
	a, b := wearing(duelist(10, 5, 500), Fire), duelist(10, 5, 500)

	r1, a1, b1 := resolve(a, b, []Card{Of(Jab, Fire)}, nil, 1)
	if n := countKind(r1, KindBurned); n != 1 {
		t.Errorf("round 1 burned %d times, want 1 — a DoT ticks at the end of the round it lands in", n)
	}

	r2, _, b2 := resolve(a1, b1, nil, nil, 2)
	if n := countKind(r2, KindBurned); n != 1 {
		t.Errorf("round 2 burned %d times, want 1 — the burn persists across the boundary", n)
	}

	r3, _, _ := resolve(a1, b2, nil, nil, 3)
	if n := countKind(r3, KindBurned); n != 0 {
		t.Errorf("round 3 burned %d times, want 0 — the burn has expired", n)
	}
}

func TestABurnCanKill(t *testing.T) {
	// Fire is the one thing in the game that ends a duel without an action, so the log has to
	// say so — the screen reads KindDefeated to end the fight and would otherwise leave a dead
	// duelist standing.
	a := wearing(duelist(10, 5, 500), Fire)
	b := duelist(10, 5, 500)

	// Enough life to survive the Jab and not the tick, so it is unambiguously the fire that
	// finished it rather than the blow that lit it.
	b.CurrentLife = Plain(Jab).Damage(a.DMG) + 1

	events, _, bAfter := resolve(a, b, []Card{Of(Jab, Fire)}, nil, 1)

	if bAfter.Alive() {
		t.Fatalf("the burn left the target on %d life", bAfter.CurrentLife)
	}
	if countKind(events, KindDefeated) == 0 {
		t.Error("a duelist killed by a burn produced no KindDefeated event")
	}
}

func TestADeadDuelistDoesNotBurn(t *testing.T) {
	// **A corpse does not tick, and the reason is the log rather than the arithmetic.** The
	// first version burned regardless: a duelist killed on the opposing turn took a fire tick
	// afterwards and the Resolution feed read "falls / burns for 2 / falls". Whether a duelist
	// is dead is settled before either side's round-end runs, so skipping the tick introduces
	// no order dependence.
	a, b := wearing(duelist(10, 8, 500), Fire), duelist(10, 5, 500)

	// A fire Pair rather than a fire Jab beside a plain Strike: the pair is a *hand*, so both
	// cards count and the mix is fire. A mixed pile would resolve as its single biggest attack —
	// the plain Strike — and light nothing at all.
	turn := []Card{Of(Jab, Fire), Of(Jab, Fire)}

	// Learn what the hand deals rather than writing the arithmetic down a second time; the
	// multipliers are expected to be retuned and this test is not about them.
	probe, _, _ := resolve(a, b, turn, nil, 1)
	blow := firstDamage(t, probe, SideA).Amount

	// Light B and kill B with the same turn.
	b.CurrentLife = blow
	events, _, bAfter := resolve(a, b, turn, nil, 1)

	if bAfter.Alive() {
		t.Fatalf("the target survived on %d life; this test needs it dead", bAfter.CurrentLife)
	}
	if !bAfter.Statuses[statusOf(Fire)].Active() {
		t.Fatal("the fire hand left no burn, so this test proves nothing")
	}
	if n := countKind(events, KindBurned); n != 0 {
		t.Errorf("a dead duelist burned %d times", n)
	}
	if n := countKind(events, KindDefeated); n != 1 {
		t.Errorf("%d KindDefeated events, want exactly 1 — a duelist falls once", n)
	}
}

// --- earth -------------------------------------------------------------------------------------

func TestEarthBluntsWhatItsVictimDeals(t *testing.T) {
	// Earth is the only status that reaches forward into what its victim *does*. It applies
	// attacker-side, before any of the defender's cards touch the blow.
	a, b := wearing(duelist(10, 5, 500), Earth), duelist(10, 5, 500)

	plain, _, _ := resolve(a, b, nil, []Card{Plain(Strike)}, 1)
	base := firstDamage(t, plain, SideB).Amount

	_, a1, b1 := resolve(a, b, []Card{Of(Strike, Earth)}, nil, 1)
	weighted, _, _ := resolve(a1, b1, nil, []Card{Plain(Strike)}, 2)

	got := firstDamage(t, weighted, SideB).Amount
	want := blunt(base, weightPct())
	if got != want {
		t.Errorf("a weighted Strike dealt %d, want %d (%d blunted by %d%%)",
			got, want, base, weightPct())
	}
	if got >= base {
		t.Errorf("a weighted Strike dealt %d against an unweighted %d — earth did nothing", got, base)
	}
}

func TestAWeightCannotBluntEverything(t *testing.T) {
	// **What bounded this was a cap on four stacks; what bounds it now is that there is only ever
	// one.** The rule it protects is unchanged: nothing takes a blow to nothing, because a turn
	// lands one figure and total negation is a whole turn deleted by one card.
	if weightPct() >= 100 {
		t.Errorf("a weight of %d%% can erase a blow outright", weightPct())
	}

	d := duelist(10, 5, 500)
	d.Statuses[statusOf(Earth)] = Status{Amount: weightPct(), Rounds: 1}
	if blunt(100, d.weight()) <= 0 {
		t.Error("a weighted blow was reduced to nothing")
	}
}

func TestBluntingRoundsTowardZeroLikeEveryOtherReduction(t *testing.T) {
	// Earth is the first percentage in a package documented as pure integer arithmetic. The
	// rounding rule matters more than the direction: it has to match guardDivisor and
	// scaleDamage so a player can predict it from the reductions they already know.
	if got, want := blunt(15, 10), 13; got != want { // 13.5
		t.Errorf("15 blunted by 10%% = %d, want %d", got, want)
	}
	if got, want := blunt(1, 50), 0; got != want {
		t.Errorf("1 blunted by 50%% = %d, want %d", got, want)
	}
	if got := blunt(20, 0); got != 20 {
		t.Errorf("an unweighted blow was changed to %d", got)
	}
}

// --- determinism ---------------------------------------------------------------------------------

func TestStatusesLeaveARoundStillDeterministic(t *testing.T) {
	// The rule the whole package is built on, re-checked against the one feature added since
	// that could plausibly have broken it. Nothing in a status consults a clock or a map.
	a, b := ringed(duelist(10, 6, 500)), ringed(duelist(10, 6, 500))
	aPlan := []Card{Of(Strike, Fire), Of(Jab, Ice)}
	bPlan := []Card{Of(Jab, Lightning), Of(Jab, Earth)}

	first, a1, b1 := resolve(a, b, aPlan, bPlan, 1)
	for i := 0; i < 20; i++ {
		got, a2, b2 := resolve(a, b, aPlan, bPlan, 1)
		if len(got) != len(first) {
			t.Fatalf("run %d produced %d events, first run produced %d", i, len(got), len(first))
		}
		for j := range got {
			if got[j] != first[j] {
				t.Fatalf("run %d event %d = %+v, first run = %+v", i, j, got[j], first[j])
			}
		}
		if a2 != a1 || b2 != b1 {
			t.Fatalf("run %d ended in a different state", i)
		}
	}
}
