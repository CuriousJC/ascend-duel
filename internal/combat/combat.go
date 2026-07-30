// Package combat is the duel rules engine. It imports nothing from Ebitengine and
// knows nothing about drawing: ResolveRound takes two duelists and the actions they
// have queued for one round and returns an ordered event log plus the state both
// sides end the round in. The combat screen replays that log; it never computes an
// outcome itself. That split is what makes the rules unit-testable and what would
// let a headless balance sim run thousands of duels with no window.
//
// A duel is a sequence of rounds. Each round the player spends an action-point
// budget on a set of actions, those resolve in order, then the enemy's set resolves.
// Control returns to the player to re-plan. Nothing here runs a duel to completion —
// that is the screen's loop, and the point is that the player re-evaluates between
// rounds.
package combat

// Side identifies which duelist an event belongs to. The engine is deliberately
// symmetric — it has no notion of "player" — so callers map A and B onto whatever
// they like. Side A resolves first every round.
type Side int

const (
	SideA Side = iota
	SideB
)

func (s Side) String() string {
	if s == SideA {
		return "A"
	}
	return "B"
}

// Duelist is a combatant's stats plus the combat state that persists between rounds.
// entities.Combatant embeds this and adds the sprite, which keeps graphics out of
// the rules entirely.
type Duelist struct {
	Con         int
	Str         int
	Spd         int
	MaxLife     int
	CurrentLife int

	// Guarded is a raised guard carried out of the round it was raised in. Side B
	// raises its guard after side A has already acted, so without this the enemy's
	// Guard could never protect anything.
	Guarded bool
}

// Alive reports whether this duelist can still fight.
func (d Duelist) Alive() bool { return d.CurrentLife > 0 }

type ActionKind int

const (
	Strike ActionKind = iota
	Guard
	Heavy
	Quick
)

// AllActions is every action a duelist can queue, in the order the UI should offer
// them — cheapest first.
var AllActions = []ActionKind{Quick, Strike, Guard, Heavy}

func (a ActionKind) String() string {
	switch a {
	case Strike:
		return "Strike"
	case Guard:
		return "Guard"
	case Heavy:
		return "Heavy"
	case Quick:
		return "Quick"
	default:
		return "Unknown"
	}
}

// Action point costs. The budget is the decision: a couple of big swings, or a
// fistful of small ones, or damage traded away for a Guard.
const (
	costQuick  = 1
	costStrike = 2
	costGuard  = 2
	costHeavy  = 4
)

// Cost is what this action takes out of the round's action-point budget.
func (a ActionKind) Cost() int {
	switch a {
	case Quick:
		return costQuick
	case Guard:
		return costGuard
	case Heavy:
		return costHeavy
	default:
		return costStrike
	}
}

// Budget conversion: everyone gets a usable turn, and speed is a real edge without
// being a landslide.
const (
	baseActionPoints = 4
	speedPerPoint    = 10
)

// ActionPoints is how much this duelist has to spend in a round.
func (d Duelist) ActionPoints() int {
	ap := baseActionPoints + d.Spd/speedPerPoint
	if ap < 1 {
		ap = 1
	}
	return ap
}

// CostOf totals the action-point cost of a queued set.
func CostOf(actions []ActionKind) int {
	total := 0
	for _, a := range actions {
		total += a.Cost()
	}
	return total
}

// CanAfford reports whether a queued set fits inside this duelist's budget. The UI
// enforces this while the player builds a set; ResolveRound trusts what it is given
// so that a balance sim can deliberately probe outside the rules.
func (d Duelist) CanAfford(actions []ActionKind) bool {
	return CostOf(actions) <= d.ActionPoints()
}

// guardDivisor is how much a raised guard cuts incoming damage.
const guardDivisor = 2

// damage is what an action deals given the attacker's Strength. Guard deals none.
func (a ActionKind) damage(str int) int {
	switch a {
	case Guard:
		return 0
	case Heavy:
		return str * 2
	case Quick:
		d := str / 2
		if d < 1 {
			d = 1
		}
		return d
	default:
		return str
	}
}

type EventKind int

const (
	KindRoundStart EventKind = iota
	KindVolleyStart
	KindAction
	KindGuarded
	KindDamage
	KindDefeated
	KindRoundEnd
)

// Event is one entry in the replayable log for a single round.
type Event struct {
	Kind   EventKind
	Side   Side       // who acted
	Action ActionKind // set on KindAction
	Amount int        // damage dealt, on KindDamage
	Target Side       // who took the damage
	Life   int        // target's life after the event
	Round  int
}

// ResolveRound plays out one round and returns its event log along with the state
// both sides end in. Side A's whole set resolves first, then side B's.
//
// Inputs are taken by value and never mutated, so a caller can re-run a round from
// the same starting state — the returned duelists are the authority on what changed.
func ResolveRound(a, b Duelist, aActions, bActions []ActionKind, round int) (events []Event, aAfter, bAfter Duelist) {
	events = make([]Event, 0, 16)
	events = append(events, Event{Kind: KindRoundStart, Round: round})

	// Side A first. A's guard from last round has already served its purpose against
	// B's reply, so it drops as A comes back around.
	a.Guarded = false
	events, a, b = resolveVolley(events, SideA, a, b, aActions, round)

	if !b.Alive() {
		events = append(events, Event{Kind: KindRoundEnd, Round: round})
		return events, a, b
	}

	// Side B replies. Its guard from last round protected it against the volley
	// above, and drops now.
	b.Guarded = false
	events, b, a = resolveVolley(events, SideB, b, a, bActions, round)

	events = append(events, Event{Kind: KindRoundEnd, Round: round})
	return events, a, b
}

// resolveVolley runs one side's whole queued set against the other. Returning the
// duelists by value keeps ResolveRound free of pointer aliasing between the two
// sides, which is the kind of bug that only shows up when both queue a Guard.
func resolveVolley(
	events []Event,
	side Side,
	actor, target Duelist,
	actions []ActionKind,
	round int,
) ([]Event, Duelist, Duelist) {
	targetSide := SideB
	if side == SideB {
		targetSide = SideA
	}

	events = append(events, Event{Kind: KindVolleyStart, Side: side, Round: round})

	for _, action := range actions {
		events = append(events, Event{
			Kind:   KindAction,
			Side:   side,
			Action: action,
			Round:  round,
		})

		if action == Guard {
			actor.Guarded = true
			continue
		}

		dmg := action.damage(actor.Str)
		if target.Guarded {
			dmg /= guardDivisor
			events = append(events, Event{
				Kind:   KindGuarded,
				Side:   side,
				Target: targetSide,
				Amount: dmg,
				Life:   target.CurrentLife,
				Round:  round,
			})
		}

		target.CurrentLife -= dmg
		if target.CurrentLife < 0 {
			target.CurrentLife = 0
		}

		events = append(events, Event{
			Kind:   KindDamage,
			Side:   side,
			Target: targetSide,
			Amount: dmg,
			Life:   target.CurrentLife,
			Round:  round,
		})

		if !target.Alive() {
			events = append(events, Event{
				Kind:   KindDefeated,
				Side:   side,
				Target: targetSide,
				Round:  round,
			})
			break
		}
	}

	return events, actor, target
}

// PlanGreedy is a placeholder opponent. It spends the whole budget on the biggest
// attack it can still afford, so it is deterministic and easy to test against. It
// never guards — a real AI is its own piece of work.
func PlanGreedy(d Duelist) []ActionKind {
	remaining := d.ActionPoints()
	plan := make([]ActionKind, 0, remaining)

	for {
		switch {
		case remaining >= costHeavy:
			plan = append(plan, Heavy)
			remaining -= costHeavy
		case remaining >= costStrike:
			plan = append(plan, Strike)
			remaining -= costStrike
		case remaining >= costQuick:
			plan = append(plan, Quick)
			remaining -= costQuick
		default:
			return plan
		}
	}
}
