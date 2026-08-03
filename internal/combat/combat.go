// Package combat is the duel rules engine. It imports nothing from Ebitengine and
// knows nothing about drawing: ResolveRound takes two duelists and the actions they
// have queued for one round and returns an ordered event log plus the state both
// sides end the round in. The combat screen replays that log; it never computes an
// outcome itself. That split is what makes the rules unit-testable and what would
// let a headless balance sim run thousands of duels with no window.
//
// A duel is a sequence of rounds. Each round both sides spend an action-point budget
// on a set of actions, and those resolve alternately — one of A's, one of B's, and so
// on. Control returns to the player to re-plan. Nothing here runs a duel to completion
// — that is the screen's loop, and the point is that the player re-evaluates between
// rounds.
package combat

// Side identifies which duelist an event belongs to. The engine is deliberately
// symmetric — it has no notion of "player" — so callers map A and B onto whatever
// they like. Side A takes the first turn of every round.
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

	// Guarded is a raised guard. It protects from the moment it goes up until its
	// owner's next action, which means it covers every opposing action that falls
	// between the two — including across a round boundary, since a guard raised on the
	// last action of a round is still up when the next one starts.
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

// Initiative is how quickly an action lands, lower being faster. It is a lever entirely
// separate from Cost: cost decides what a plan may contain, initiative decides when the
// pieces of it happen. Guard is deliberately quicker than the Strike it is meant to
// answer, so raising it in the same exchange actually beats the blow to the punch.
const (
	initQuick  = 1
	initGuard  = 2
	initStrike = 3
	initHeavy  = 5
)

// Initiative is where in an exchange this action lands. Lower goes first.
func (a ActionKind) Initiative() int {
	switch a {
	case Quick:
		return initQuick
	case Guard:
		return initGuard
	case Heavy:
		return initHeavy
	default:
		return initStrike
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

// Damage is what an action deals given the attacker's Strength. Guard deals none.
func (a ActionKind) Damage(str int) int {
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

// Slot is one action's place in a round's resolution order: whose it is, where it sits
// in that side's queue, and what it is.
type Slot struct {
	Side   Side
	Index  int
	Action ActionKind
}

// ResolutionOrder is the sequence in which two queued sets resolve, and the single
// authority on that order. ResolveRound plays it and the combat screen's Resolution pane
// draws it; neither works the order out for itself, so the pane and the engine cannot
// drift apart.
//
// The queues alternate one action each, as they always have. What initiative adds is who
// moves first *inside* one exchange: the faster action lands first, lower initiative
// winning, and side A takes a tie. That is what makes the position a card is dragged to
// a real decision — it chooses which of the opponent's actions that card contests, and
// therefore whether it beats that action or answers it.
//
// Whichever queue is longer keeps acting alone once the other runs out. That tail is
// where a speed advantage shows, so it is the point rather than an edge case.
func ResolutionOrder(aActions, bActions []ActionKind) []Slot {
	slots := make([]Slot, 0, len(aActions)+len(bActions))

	for i := 0; i < len(aActions) || i < len(bActions); i++ {
		aHas, bHas := i < len(aActions), i < len(bActions)

		// Side A leads unless B's action in this exchange is strictly faster. A tie going
		// to A preserves the behaviour from before initiative existed and keeps the round
		// reproducible — no comparison here may depend on anything but these two actions.
		aFirst := true
		if aHas && bHas {
			aFirst = aActions[i].Initiative() <= bActions[i].Initiative()
		}

		if aHas && aFirst {
			slots = append(slots, Slot{Side: SideA, Index: i, Action: aActions[i]})
		}
		if bHas {
			slots = append(slots, Slot{Side: SideB, Index: i, Action: bActions[i]})
		}
		if aHas && !aFirst {
			slots = append(slots, Slot{Side: SideA, Index: i, Action: aActions[i]})
		}
	}

	return slots
}

// ResolveRound plays out one round and returns its event log along with the state
// both sides end in. ResolutionOrder decides the order it plays them in.
//
// Alternating rather than resolving one side's whole set and then the other's is what
// makes resolution order a thing the player can reason about and, later, manipulate:
// the order is the visible plan, so speed and other effects have somewhere to bite.
//
// Inputs are taken by value and never mutated, so a caller can re-run a round from
// the same starting state — the returned duelists are the authority on what changed.
func ResolveRound(a, b Duelist, aActions, bActions []ActionKind, round int) (events []Event, aAfter, bAfter Duelist) {
	events = make([]Event, 0, 16)
	events = append(events, Event{Kind: KindRoundStart, Round: round})

	for _, slot := range ResolutionOrder(aActions, bActions) {
		if slot.Side == SideA {
			events, a, b = resolveAction(events, SideA, a, b, slot.Action, round)
			if !b.Alive() {
				break
			}
			continue
		}

		events, b, a = resolveAction(events, SideB, b, a, slot.Action, round)
		if !a.Alive() {
			break
		}
	}

	events = append(events, Event{Kind: KindRoundEnd, Round: round})
	return events, a, b
}

// resolveAction runs a single action by one side against the other. Returning the
// duelists by value keeps ResolveRound free of pointer aliasing between the two
// sides, which is the kind of bug that only shows up when both queue a Guard.
func resolveAction(
	events []Event,
	side Side,
	actor, target Duelist,
	action ActionKind,
	round int,
) ([]Event, Duelist, Duelist) {
	targetSide := SideB
	if side == SideB {
		targetSide = SideA
	}

	// A raised guard has served its purpose by the time its owner comes back around,
	// so it drops here — at the start of the actor's turn, before this action gets the
	// chance to re-raise it. A duelist who queues nothing therefore keeps a guard up:
	// see TestGuardHoldsWhileItsOwnerDoesNothing, which pins that as intended.
	actor.Guarded = false

	events = append(events, Event{
		Kind:   KindAction,
		Side:   side,
		Action: action,
		Round:  round,
	})

	if action == Guard {
		actor.Guarded = true
		return events, actor, target
	}

	dmg := action.Damage(actor.Str)
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
