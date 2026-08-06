// Package combat is the duel rules engine. It imports nothing from Ebitengine and
// knows nothing about drawing: ResolveRound takes two duelists and the actions they
// have queued for one round and returns an ordered event log plus the state both
// sides end the round in. The combat screen replays that log; it never computes an
// outcome itself. That split is what makes the rules unit-testable and what would
// let a headless balance sim run thousands of duels with no window.
//
// A duel is a sequence of rounds. Each round both sides spend an action-point budget
// on a set of actions, and those resolve **in phases**: everything side A queued, in
// category order, and then everything side B queued. Control returns to the player to
// re-plan. Nothing here runs a duel to completion — that is the screen's loop, and the
// point is that the player re-evaluates between rounds.
//
// Phase resolution replaced alternation on 2026-08-06, on the grounds that interleaving is
// not graspable by players. See MECHANICS.md. Two consequences run through this file:
//
//   - **Initiative is gone.** With one contiguous turn per side there is no exchange for a
//     faster action to lead, so the whole lever was reporting a distinction the resolver no
//     longer made. See the TODO in TODO.md before bringing it back.
//   - **Defenses cover the opponent's next turn, not the rest of the round.** Side B acts
//     last, so a defense that expired at the round boundary would never protect B from
//     anything. They expire at the start of their owner's own next turn instead, which is
//     the one rule that is symmetric under a resolution order that is not.
package combat

// Side identifies which duelist an event belongs to. The engine is deliberately
// symmetric — it has no notion of "player" — so callers map A and B onto whatever
// they like. Side A takes its whole turn before side B takes any of it.
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
//
// Every field is comparable on purpose: TestRoundIsDeterministic compares two resolved
// duelists with ==, so nothing here may become a slice or a map. Two defenses that need
// counting are two ints rather than a queue for exactly that reason.
type Duelist struct {
	Con         int
	Str         int
	Spd         int
	MaxLife     int
	CurrentLife int

	// Guarded halves every incoming attack. Raised by Guard, and standing until the start
	// of its owner's next turn — long enough to cover the opponent's whole turn once,
	// whichever side raised it. See expireDefenses.
	Guarded bool

	// Pending negations. Each one is spent stopping a single incoming attack dead, and
	// Ripostes are spent before Dodges so their counter-damage lands as early in the
	// opponent's turn as it can — early enough to cut the rest of that turn short if it
	// kills. They expire alongside Guarded.
	Ripostes int
	Dodges   int

	// The two halves of Prepare. PreparedAP is what has been banked *this* round; at the
	// round boundary it becomes BonusAP, which ActionPoints adds to the budget for the
	// round after. Splitting them is what makes the bonus arrive next round rather than
	// funding the turn that bought it.
	//
	// BonusAP is overwritten rather than added to at the boundary, so preparing twice in
	// one round is worth +4 next round while preparing once a round is worth a flat +2.
	// Stacking within a round is deliberate and is what puts the five-attack combo in
	// reach without a ring; carrying across rounds would compound without limit.
	BonusAP    int
	PreparedAP int
}

// Alive reports whether this duelist can still fight.
func (d Duelist) Alive() bool { return d.CurrentLife > 0 }

// Category is which phase of a turn an action resolves in, and the axis the whole round
// is now built on. It is a property of the action, not an independent choice: a fire Guard
// and a plain Guard are both setup.
type Category int

const (
	CategorySetup Category = iota
	CategoryAttack
	CategoryDefend
)

// Categories is every phase in resolution order, and the order a turn is played in.
// Defenses come last within a turn because the *opponent* acts afterwards — a defense
// raised at the end of your turn is up when their blow arrives.
func Categories() []Category {
	return []Category{CategorySetup, CategoryAttack, CategoryDefend}
}

func (c Category) String() string {
	switch c {
	case CategorySetup:
		return "setup"
	case CategoryAttack:
		return "attack"
	case CategoryDefend:
		return "defend"
	default:
		return "?"
	}
}

type ActionKind int

// Declared in category order, so anything that sorts by the raw value — the deck overlay
// does — groups the piles the same way a turn resolves them.
const (
	// Setup.
	Prepare ActionKind = iota
	Guard

	// Attack.
	Quick
	Strike
	Heavy

	// Defend.
	Dodge
	Riposte
)

// AllActions is every action a duelist can queue, in category order.
var AllActions = []ActionKind{Prepare, Guard, Quick, Strike, Heavy, Dodge, Riposte}

func (a ActionKind) String() string {
	switch a {
	case Prepare:
		return "Prepare"
	case Guard:
		return "Guard"
	case Quick:
		return "Quick"
	case Strike:
		return "Strike"
	case Heavy:
		return "Heavy"
	case Dodge:
		return "Dodge"
	case Riposte:
		return "Riposte"
	default:
		return "Unknown"
	}
}

// Category is which phase this action resolves in.
func (a ActionKind) Category() Category {
	switch a {
	case Prepare, Guard:
		return CategorySetup
	case Dodge, Riposte:
		return CategoryDefend
	default:
		return CategoryAttack
	}
}

// Action point costs. The budget is the decision: a couple of big swings, or a
// fistful of small ones, or damage traded away for a defense.
const (
	costPrepare = 1
	costGuard   = 3
	costQuick   = 1
	costStrike  = 2
	costHeavy   = 4
	costDodge   = 2
	costRiposte = 3
)

// Cost is what this action takes out of the round's action-point budget.
func (a ActionKind) Cost() int {
	switch a {
	case Prepare:
		return costPrepare
	case Guard:
		return costGuard
	case Quick:
		return costQuick
	case Heavy:
		return costHeavy
	case Dodge:
		return costDodge
	case Riposte:
		return costRiposte
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

// prepareBonusAP is what one Prepare banks for the following round. Two for one is a
// deliberate profit — the cost of Prepare is the card slot and the action slot it takes
// out of the round it is played in, not the point it costs.
const prepareBonusAP = 2

// ActionPoints is how much this duelist has to spend in a round, including anything
// banked by a Prepare in the round before.
func (d Duelist) ActionPoints() int {
	ap := baseActionPoints + d.Spd/speedPerPoint + d.BonusAP
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

// Damage is what an action deals given the attacker's Strength.
//
// Riposte is the odd one: it is a *defend*, and the number below is what it hits back for
// when it stops an attack, not something it deals on its own. Reporting it here is what
// lets the card draw a damage badge for it without the screen knowing the rule.
func (a ActionKind) Damage(str int) int {
	switch a {
	case Heavy:
		return str * 2
	case Strike:
		return str
	case Quick, Riposte:
		d := str / 2
		if d < 1 {
			d = 1
		}
		return d
	default:
		return 0
	}
}

type EventKind int

const (
	KindRoundStart EventKind = iota
	KindAction
	KindPrepared
	KindNegated
	KindGuarded
	KindDamage
	KindDefeated
	KindRoundEnd
)

// Event is one entry in the replayable log for a single round.
type Event struct {
	Kind   EventKind
	Side   Side       // who acted
	Action ActionKind // set on KindAction, and on KindNegated for the defense that stopped it
	Amount int        // damage dealt, or action points banked
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
// **A whole turn each, in category order.** Everything side A queued resolves — setups,
// then attacks, then defenses — and only then does side B begin. Within a category the
// queued order is kept, which is where drag-to-reorder still bites and where sequence
// combos will match.
//
// Index is the action's position in its own side's queue, which is *not* its position
// here: reordering by category is the whole job. Consumers wanting "how far through the
// round are we" should count slots rather than read Index.
func ResolutionOrder(aActions, bActions []ActionKind) []Slot {
	slots := make([]Slot, 0, len(aActions)+len(bActions))
	slots = appendTurn(slots, SideA, aActions)
	slots = appendTurn(slots, SideB, bActions)
	return slots
}

// appendTurn adds one side's whole turn, category by category.
func appendTurn(slots []Slot, side Side, actions []ActionKind) []Slot {
	for _, cat := range Categories() {
		for i, a := range actions {
			if a.Category() == cat {
				slots = append(slots, Slot{Side: side, Index: i, Action: a})
			}
		}
	}
	return slots
}

// ResolveRound plays out one round and returns its event log along with the state
// both sides end in. ResolutionOrder decides the order it plays them in.
//
// Inputs are taken by value and never mutated, so a caller can re-run a round from
// the same starting state — the returned duelists are the authority on what changed.
func ResolveRound(a, b Duelist, aActions, bActions []ActionKind, round int) (events []Event, aAfter, bAfter Duelist) {
	events = make([]Event, 0, 16)
	events = append(events, Event{Kind: KindRoundStart, Round: round})

	// A defense expires at the start of its owner's next turn, so it covers exactly one
	// opposing turn whichever side raised it. Expiry is a rule about *turns* rather than
	// about the action sequence, which is why it lives here and not in ResolutionOrder —
	// a side that queues nothing still has a turn, and still loses its guard in it.
	//
	// Side A's turn starts now; side B's starts the moment A's slots run out.
	a = expireDefenses(a)
	bStarted := false

	for _, slot := range ResolutionOrder(aActions, bActions) {
		if slot.Side == SideB && !bStarted {
			b = expireDefenses(b)
			bStarted = true
		}

		if slot.Side == SideA {
			events, a, b = resolveAction(events, SideA, a, b, slot.Action, round)
		} else {
			events, b, a = resolveAction(events, SideB, b, a, slot.Action, round)
		}

		// Either side can fall here: a Riposte kills the attacker who walked into it.
		if !a.Alive() || !b.Alive() {
			break
		}
	}

	if !bStarted {
		b = expireDefenses(b)
	}

	a, b = endRound(a), endRound(b)

	events = append(events, Event{Kind: KindRoundEnd, Round: round})
	return events, a, b
}

// expireDefenses drops everything the previous turn put up. Called at the start of a
// side's own turn, never at the round boundary — side B acts last, so a defense cleared
// at the boundary would have protected B from nothing at all.
func expireDefenses(d Duelist) Duelist {
	d.Guarded = false
	d.Ripostes = 0
	d.Dodges = 0
	return d
}

// endRound rolls what was banked this round into next round's budget. Assignment rather
// than addition: two Prepares in one round are worth +4 next round, and preparing every
// round is worth a flat +2 rather than compounding forever.
func endRound(d Duelist) Duelist {
	d.BonusAP = d.PreparedAP
	d.PreparedAP = 0
	return d
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
	targetSide := other(side)

	events = append(events, Event{
		Kind:   KindAction,
		Side:   side,
		Action: action,
		Round:  round,
	})

	switch action {
	case Prepare:
		actor.PreparedAP += prepareBonusAP
		events = append(events, Event{
			Kind:   KindPrepared,
			Side:   side,
			Amount: prepareBonusAP,
			Round:  round,
		})
		return events, actor, target

	case Guard:
		actor.Guarded = true
		return events, actor, target

	case Dodge:
		actor.Dodges++
		return events, actor, target

	case Riposte:
		actor.Ripostes++
		return events, actor, target
	}

	return resolveAttack(events, side, targetSide, actor, target, action, round)
}

// resolveAttack lands one attack, or fails to. A pending Riposte or Dodge on the target
// stops it dead; a raised Guard merely halves it.
func resolveAttack(
	events []Event,
	side, targetSide Side,
	actor, target Duelist,
	action ActionKind,
	round int,
) ([]Event, Duelist, Duelist) {
	// Negation first, and Ripostes before Dodges. Both stop the blow completely, so
	// spending the one that hits back first is free — and it gets the counter-damage into
	// the log early enough to end the attacker's turn if it kills.
	if target.Ripostes > 0 {
		target.Ripostes--
		events = append(events, Event{
			Kind:   KindNegated,
			Side:   targetSide,
			Action: Riposte,
			Target: side,
			Round:  round,
		})

		// The counter runs the other way: the defender is the one dealing damage now, so
		// the sides in this event are swapped relative to the attack that provoked it.
		counter := Riposte.Damage(target.Str)
		actor.CurrentLife = reduce(actor.CurrentLife, counter)

		events = append(events, Event{
			Kind:   KindDamage,
			Side:   targetSide,
			Target: side,
			Amount: counter,
			Life:   actor.CurrentLife,
			Round:  round,
		})
		if !actor.Alive() {
			events = append(events, Event{
				Kind:   KindDefeated,
				Side:   targetSide,
				Target: side,
				Round:  round,
			})
		}
		return events, actor, target
	}

	if target.Dodges > 0 {
		target.Dodges--
		events = append(events, Event{
			Kind:   KindNegated,
			Side:   targetSide,
			Action: Dodge,
			Target: side,
			Round:  round,
		})
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

	target.CurrentLife = reduce(target.CurrentLife, dmg)

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

// reduce takes damage off a life total without letting it go negative.
func reduce(life, dmg int) int {
	life -= dmg
	if life < 0 {
		return 0
	}
	return life
}

// other is the side that is not this one.
func other(s Side) Side {
	if s == SideA {
		return SideB
	}
	return SideA
}

// PlanGreedy is a placeholder opponent. It spends the whole budget on the biggest
// attack it can still afford, so it is deterministic and easy to test against. It never
// defends and never prepares — a real AI is its own piece of work, and enemies that fight
// in genuinely different shapes are the next design job. See TODO.md.
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
