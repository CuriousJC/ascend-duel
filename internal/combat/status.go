package combat

import "math/rand"

// Statuses are what elements *do*, and they are the first thing in this package that outlives
// the action that caused it. A chill is read when a turn is taken; a burn ticks at a moment no
// action owns; a shock is rolled against the attack it interrupts.
//
// **Nothing applies a status by itself as of 2026-08-16.** A fire attack is a plain attack with a
// red border unless the attacker is wearing the fire ring. This reverses the position that
// elements carry their statuses inherently, and the reason is that it left rings with nothing to
// *be*: the statuses were already free, so the first three rings had to invent a second mechanic
// to sell. Giving them the statuses instead means the element set is a combo axis on its own
// terms and a ring is the thing that turns a colour into a rule. See Duelist.Rings.
//
// **One lifecycle for all four, so it is learned once**:
//
//   - **Applied by a landed attack whose owner wears the matching ring**, and by nothing else.
//   - **It does not stack; a second hit resets the clock.** *(2026-08-16)* Two ice hits chill for
//     the same one card as one did, and the second simply buys two more round-ends of it. Amounts
//     stacked until then, which made a status something to pile on rather than something to keep
//     up — and with one blow a turn, four stacks of a thing was four cards spent saying one word
//     louder. A ring that *does* stack is a ring that can be designed later; the base rule being
//     "no" is what leaves it somewhere to go.
//   - **Cleared at the end of the round after the one that applied them.** `statusDuration` is
//     counted in round *ends*, and the value is 2 for every element for exactly that reason: a
//     status applied during round N has to survive round N's ending to be felt in round N+1.
//     A duration of 1 would mean a status applied by side B — who acts second — never bit
//     anything at all.
//
// The uniform duration is a starting position, not a discovery. Per-element tuning is one
// constant each away, and `tools/balance` is the thing to run before changing one.

// Status is one element's hold on a duelist: how much, and how much longer.
//
// Amount means a different thing per element and each is documented at its constant below —
// damage per tick for a burn, cards off the turn for a chill, a percentage chance to miss for a
// shock, percentage points of damage for a weight. A generic amount is what lets the four share
// one array and one lifecycle; a generic *meaning* would be a rule nobody could state.
type Status struct {
	Amount int
	Rounds int
}

// Active reports whether this status is doing anything. A zero Status is the absence of one.
func (s Status) Active() bool { return s.Rounds > 0 && s.Amount > 0 }

// statusDuration is how many round-ends a freshly applied status survives. See the file comment
// for why it is 2 rather than 1, and why it is one number rather than four.
const statusDuration = 2

// What a landed attack of each element does to whoever took it, when the attacker wears that
// element's ring. One constant per element, all per *hit* — a Heavy and a Jab apply the same
// status, because what the element does is a property of the ring and what the concept does is
// damage.
//
// That is deliberate and it has a consequence worth stating: **a fire Jab is the cheapest way to
// apply a burn**, at 1 AP. The concept ladder prices damage; the element ladder does not exist.
// If status magnitude should scale with the card, that is a second axis and a design change.
const (
	// burnPctOfDMG is the share of the attacker's DMG a burn deals at the end of each round it
	// survives. It ticks at the end of the round it landed in as well as the one after, so one
	// fire hit is 2 ticks.
	//
	// **It is read off the attacker and frozen when it lands**, not recomputed each tick. A burn
	// is what that blow lit; a duelist whose DMG changes mid-duel does not retroactively burn
	// harder, and the victim carries the number rather than a pointer back to whoever lit it.
	burnPctOfDMG = 10

	// chillCardsPerHit is how many cards come off the front of the target's next turn.
	//
	// **Ice stopped being the AP element on 2026-08-16.** It took a point off the budget until
	// then, which the ring's own text does not describe and which was the quietest possible
	// status: a duelist a point short simply queued a cheaper card and lost nothing they could
	// name. Taking a card is the stagger the combo table already deals in, and a chilled duelist
	// loses a card off every turn it is chilled for rather than one card once.
	chillCardsPerHit = 1

	// shockMissPct is how likely a shocked duelist's attack is to miss, in percentage points. It
	// is the Amount a shock carries, because with stacking gone the magnitude *is* the chance.
	//
	// **It is rolled on every attack the shock outlives** *(2026-08-16)*, where a stack used to be
	// spent on the first attack whether or not the roll landed. Spending was how a stack could be
	// worn down; with nothing to wear down, a shock that consumed itself on contact would be a
	// two-round status that reliably lasted one attack — a duration doing no work.
	//
	// **This is the only randomness in `internal/combat` and it arrives the way CLAUDE.md
	// requires**: an injected `*rand.Rand` on `ResolveRound`, never a package-level source. The
	// costs are real and were accepted: `tools/balance` becomes a distribution rather than an
	// exact answer, and this is a stream advanced per attack phase, so a change early in a duel
	// reshuffles every roll after it.
	//
	// **It can never be a certainty**, which is what one blow per turn demands: a defence that
	// always works deletes a whole opposing turn for the price of one card. 25 is a quarter of
	// them, and the ceiling is the number itself now that four hits cannot add up to anything.
	shockMissPct = 25

	// weightPct is percentage points off the damage the target *deals*. Earth is the only status
	// that reaches forward into what its victim does rather than what happens to them.
	//
	// **25 rather than the 10 it was**, because 10 that could not stack was a status nobody would
	// notice landing. What bounded it before was a cap on four stacks; what bounds it now is that
	// there is only ever one.
	weightPct = 25
)

// statusFor is the element a landed attack applies and how much of it, given the duelist
// throwing it. Basic applies nothing, because it is the absence of an element rather than a
// fifth colour.
//
// **The attacker is a parameter because a burn is a share of their DMG.** The other three are
// flat, and passing the duelist to all four is what keeps "how much" one function rather than a
// special case for fire at the call site.
//
// The bool reports whether anything is applied at all, so a caller does not have to know that a
// zero amount means "no status" as well as meaning "none of this status".
func statusFor(e Element, by Duelist) (int, bool) {
	switch e {
	case Fire:
		// The floor is the same rule Jab's damage follows: a duelist under 10 DMG would otherwise
		// light a burn worth nothing at all, and a status that lands and does nothing is worse
		// than one that does not land.
		burn := by.DMG * burnPctOfDMG / 100
		if burn < 1 {
			burn = 1
		}
		return burn, true
	case Ice:
		return chillCardsPerHit, true
	case Lightning:
		return shockMissPct, true
	case Earth:
		return weightPct, true
	default:
		return 0, false
	}
}

// applyStatus lands one element's status on a duelist: the amount is set and the duration
// refreshed. **Set, not added** — see the file comment on why nothing stacks.
//
// It returns the duelist by value like everything else in this package, and the amount applied,
// so the caller can log what happened without recomputing it.
func applyStatus(d Duelist, e Element, by Duelist) (Duelist, int, bool) {
	amount, ok := statusFor(e, by)
	if !ok {
		return d, 0, false
	}

	d.Statuses[e] = Status{Amount: amount, Rounds: statusDuration}

	return d, amount, true
}

// chillCards is how many actions this duelist loses off the front of its turn to ice. It is read
// where a stagger is read and adds to one, which is what makes "a chilled duelist loses a card"
// one rule rather than a second machinery beside the one the combo table already uses.
func (d Duelist) chillCards() int {
	if !d.Statuses[Ice].Active() {
		return 0
	}
	return d.Statuses[Ice].Amount
}

// weight is the percentage this duelist's outgoing attack damage is blunted by. It is a
// percentage rather than a fraction because MECHANICS.md specified one; the arithmetic below is
// still integer and still truncates.
func (d Duelist) weight() int {
	if !d.Statuses[Earth].Active() {
		return 0
	}
	return d.Statuses[Earth].Amount
}

// blunt applies a weight to one outgoing blow.
//
// **Rounding is toward zero**, matching guardDivisor, the defend reductions and scaleDamage. Earth is the
// first percentage in a package documented as pure integer arithmetic, and the rule being the
// same as every other reduction is what keeps it predictable from the others rather than being
// the one number a player cannot work out.
func blunt(dmg, pct int) int {
	if pct <= 0 {
		return dmg
	}
	return dmg * (100 - pct) / 100
}

// shockMisses rolls a shocked duelist's attack and reports whether it misses.
//
// **Nothing is consumed** *(2026-08-16)*: the shock rolls again on every attack until its rounds
// run out. It therefore takes the duelist by value and gives nothing back — a status that neither
// stacks nor depletes is read, not spent.
//
// The source is passed in rather than drawn from a package global — see the determinism rules in
// CLAUDE.md. A nil source means "no rolls", which is what keeps a caller that has no business
// being random (a preview, a test pinning the deterministic parts) from silently getting one.
func shockMisses(d Duelist, rng *rand.Rand) bool {
	s := d.Statuses[Lightning]
	if !s.Active() || rng == nil {
		return false
	}
	return rng.Intn(100) < s.Amount
}

// tickStatuses counts every status down one round end and clears the ones that run out. The burn
// damage is *not* here — it is a life change that has to produce events, so endRound does it.
func tickStatuses(d Duelist) Duelist {
	for _, e := range AllElements {
		s := d.Statuses[e]
		if s.Rounds <= 0 {
			d.Statuses[e] = Status{}
			continue
		}
		s.Rounds--
		if s.Rounds <= 0 {
			s = Status{}
		}
		d.Statuses[e] = s
	}
	return d
}
