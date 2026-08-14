package combat

import "math/rand"

// Statuses are what elements *do*, and they are the first thing in this package that outlives
// the action that caused it. `Guarded` and the counted negations all expire at the start of
// their owner's next turn; a burn ticks at a moment no action owns, and a chill is read when a
// budget is worked out rather than when a card resolves.
//
// **One lifecycle for all four, so it is learned once** *(decided 2026-08-12)*:
//
//   - **Applied by a landed attack**, and by nothing else. See element.go for why.
//   - **Amount stacks, duration refreshes.** Two ice hits chill for two; the second does not
//     make the first last longer than the second does.
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
// action points for a chill, missed attacks for a shock, damage per tick for a burn, percentage
// points for a weight. A generic amount is what lets the four share one array and one lifecycle;
// a generic *meaning* would be a rule nobody could state.
type Status struct {
	Amount int
	Rounds int
}

// Active reports whether this status is doing anything. A zero Status is the absence of one.
func (s Status) Active() bool { return s.Rounds > 0 && s.Amount > 0 }

// statusDuration is how many round-ends a freshly applied status survives. See the file comment
// for why it is 2 rather than 1, and why it is one number rather than four.
const statusDuration = 2

// What a landed attack of each element does to whoever took it. One constant per element, all
// per *hit* — a Heavy and a Jab apply the same status, because what the element does is a
// property of the element and what the concept does is damage.
//
// That is deliberate and it has a consequence worth stating: **a fire Jab is the cheapest way to
// apply a burn**, at 1 AP. The concept ladder prices damage; the element ladder does not exist.
// If status magnitude should scale with the card, that is a second axis and a design change.
const (
	// chillPerHit is action points taken off the target's next budget. Ice is the AP element in
	// both directions — the ring discounts your ice cards, the status cuts their budget.
	chillPerHit = 1

	// shockPerHit is how many *chances to miss* a landed lightning hit puts on the target.
	//
	// **It is a roll again as of 2026-08-14**, reversing the deterministic version taken on
	// 2026-08-12. The reason it had to change: a turn now resolves one attack, so a certain miss
	// deleted a whole turn — up to a 250-damage Onslaught — for the price of one 1 AP fire Jab.
	// The reason a roll was chosen over the deterministic alternatives (breaking the hand,
	// cutting the multiplier) is that lightning should feel unreliable.
	//
	// **This is the first randomness in `internal/combat` and it arrives the way CLAUDE.md
	// requires**: an injected `*rand.Rand` on `ResolveRound`, never a package-level source. The
	// costs are real and were accepted: `tools/balance` becomes a distribution rather than an
	// exact answer, and this is a stream advanced per attack phase, so a change early in a duel
	// reshuffles every roll after it.
	shockPerHit = 1

	// shockMissPct is how likely one stack of shock is to make the turn's attack miss, and
	// shockMissCapPct is the most any number of stacks can reach.
	//
	// **The cap exists so shock can never be a certainty.** Without it four lightning hits would
	// be the deterministic rule this replaced, arrived at by a different route — and a defence
	// that always works is the thing one blow per turn makes intolerable.
	shockMissPct    = 25
	shockMissCapPct = 75

	// burnPerHit is damage dealt at the end of each round the burn survives. It ticks at the end
	// of the round it landed in as well as the one after, so one fire hit is 2 ticks.
	burnPerHit = 2

	// weightPerHit is percentage points off the damage the target *deals*. Earth is the only
	// status that reaches forward into what its victim does rather than what happens to them.
	weightPerHit = 10

	// weightCapPct is the most a weight can blunt, however many earth hits land. Without it four
	// earth Jabs make an opponent harmless for 4 AP, which is a cheaper answer to a duel than
	// winning it.
	weightCapPct = 50
)

// statusFor is the element a landed attack applies, and how much of it. Basic applies nothing,
// because it is the absence of an element rather than a fifth colour.
//
// The bool reports whether anything is applied at all, so a caller does not have to know that a
// zero amount means "no status" as well as meaning "none of this status".
func statusFor(e Element) (int, bool) {
	switch e {
	case Fire:
		return burnPerHit, true
	case Ice:
		return chillPerHit, true
	case Lightning:
		return shockPerHit, true
	case Earth:
		return weightPerHit, true
	default:
		return 0, false
	}
}

// applyStatus lands one element's status on a duelist: the amount stacks, the duration refreshes.
// It returns the duelist by value like everything else in this package, and the amount applied,
// so the caller can log what happened without recomputing it.
func applyStatus(d Duelist, e Element) (Duelist, int, bool) {
	amount, ok := statusFor(e)
	if !ok {
		return d, 0, false
	}

	s := d.Statuses[e]
	s.Amount += amount
	s.Rounds = statusDuration
	d.Statuses[e] = s

	return d, amount, true
}

// chill is the action points this duelist has lost to ice.
func (d Duelist) chill() int {
	if !d.Statuses[Ice].Active() {
		return 0
	}
	return d.Statuses[Ice].Amount
}

// weightPct is the percentage this duelist's outgoing attack damage is blunted by, already
// capped. It is a percentage rather than a fraction because MECHANICS.md specified one; the
// arithmetic below is still integer and still truncates.
func (d Duelist) weightPct() int {
	if !d.Statuses[Earth].Active() {
		return 0
	}
	if w := d.Statuses[Earth].Amount; w < weightCapPct {
		return w
	}
	return weightCapPct
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

// shockChancePct is how likely a shocked duelist's attack is to miss, given the stacks it is
// carrying. Stacks add and the total is capped, so more lightning is better without ever being
// certain.
func (d Duelist) shockChancePct() int {
	s := d.Statuses[Lightning]
	if !s.Active() {
		return 0
	}
	if pct := s.Amount * shockMissPct; pct < shockMissCapPct {
		return pct
	}
	return shockMissCapPct
}

// spendShock consumes one stack of a shock and rolls it, reporting whether the attack it was
// checked against misses.
//
// **One stack is spent whether or not the roll lands.** A shock that only burned itself on a
// success would make the status last until it worked, which is a guarantee wearing a
// probability's clothes.
//
// The source is passed in rather than drawn from a package global — see the determinism rules in
// CLAUDE.md. A nil source means "no rolls", which is what keeps a caller that has no business
// being random (a preview, a test pinning the deterministic parts) from silently getting one.
func spendShock(d Duelist, rng *rand.Rand) (Duelist, bool) {
	chance := d.shockChancePct()
	if chance <= 0 {
		return d, false
	}

	s := d.Statuses[Lightning]
	s.Amount--
	if s.Amount <= 0 {
		s = Status{}
	}
	d.Statuses[Lightning] = s

	if rng == nil {
		return d, false
	}
	return d, rng.Intn(100) < chance
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
