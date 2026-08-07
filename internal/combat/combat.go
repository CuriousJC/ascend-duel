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

	// Staggered is how many actions are taken off the front of this duelist's *next* turn,
	// or StaggerAll for all of them. Set by an opponent's combo and consumed when that turn
	// comes round.
	//
	// **It has to persist across the round boundary, and that is a consequence of phases
	// rather than a choice.** Side A takes its whole turn first, so a combo A forms bites B
	// in the same round; the identical combo formed by B lands when A has already acted, and
	// has nowhere to go but the round after. Holding it on the duelist is what makes the rule
	// one rule — *a staggered duelist loses actions from its next turn, whenever that is* —
	// instead of two rules that happen to be spelled differently for the two sides.
	Staggered int
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

// baseMaxActions is how many actions one duelist may take in a round, whatever they cost.
const baseMaxActions = 5

// MaxActions is the second of the two bounds on a round. **A round is bounded by cost and
// by count, independently and on purpose**: the budget gates what can be afforded, and this
// gates how much can happen at all — which still bites when discounts have taken cards to
// free, and is what stops a swarm from becoming unbounded as its speed grows.
//
// It is a method rather than the bare constant it used to be, and it lives here rather than
// on the screen where `maxSelected` used to. Both were deliberate: it is a **rule**, so the
// opponent's planner has to obey it exactly as the player's selection does, and making it a
// function of the duelist is what gives a ring or a brand raising the cap somewhere to bite
// without touching a single call site. See MECHANICS.md.
func (d Duelist) MaxActions() int { return baseMaxActions }

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

	// KindCombo says a combo formed, and is emitted at the first card of the run rather than
	// the last — the effects come into force there, so narrating it later would show the
	// player a boosted hit before telling them why.
	KindCombo

	// KindStaggered is one action lost to a stagger. One event per action, so a stagger that
	// takes a whole round narrates as the several things it actually is.
	KindStaggered

	KindRoundEnd
)

// Event is one entry in the replayable log for a single round.
type Event struct {
	Kind   EventKind
	Side   Side       // who acted
	Action ActionKind // set on KindAction, on KindNegated for the defense that stopped it, and on KindStaggered for the action lost
	Amount int        // damage dealt, or action points banked
	Target Side       // who took the damage
	Life   int        // target's life after the event
	Round  int

	// Combo is set on KindCombo, and names which one fired. The screen looks it up with
	// ComboByID rather than being told its name here, so a combo renamed is renamed once.
	Combo ComboID
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
	return resolveRound(a, b, aActions, bActions, round, comboTable)
}

// resolveRound is ResolveRound with the combo table injected. It exists so a test can drive
// a synthetic combo through the whole engine rather than only through the matcher — the
// damage multiplier and the banked points would otherwise be code paths that shipped without
// ever having been run, since the two combos the game currently has both use stagger.
func resolveRound(a, b Duelist, aActions, bActions []ActionKind, round int, table []Combo) (events []Event, aAfter, bAfter Duelist) {
	events = make([]Event, 0, 16)
	events = append(events, Event{Kind: KindRoundStart, Round: round})

	// A defense expires at the start of its owner's next turn, so it covers exactly one
	// opposing turn whichever side raised it. Expiry is a rule about *turns* rather than
	// about the action sequence, which is why it lives here and not in ResolutionOrder —
	// a side that queues nothing still has a turn, and still loses its guard in it.
	//
	// A whole turn each, A then B. This used to be one flat loop over ResolutionOrder with a
	// flag watching for the handover; combos made a turn a thing with its own beginning —
	// stagger is spent at it, and a combo's position is an index *within* it — so the turn
	// became worth naming. ResolutionOrder is still the authority on order: playTurn walks
	// exactly the slots it produced for that side.
	events, a, b = playTurn(events, SideA, a, b, appendTurn(nil, SideA, aActions), round, table)

	// B still loses its standing defenses even in a round it never gets to act in, which is
	// why this is not inside playTurn's early return: expiry is a property of the turn
	// arriving, not of anything happening in it.
	if a.Alive() && b.Alive() {
		events, b, a = playTurn(events, SideB, b, a, appendTurn(nil, SideB, bActions), round, table)
	} else {
		b = expireDefenses(b)
	}

	a, b = endRound(a), endRound(b)

	events = append(events, Event{Kind: KindRoundEnd, Round: round})
	return events, a, b
}

// playTurn runs one side's whole turn: expiry, then whatever a stagger has taken off the
// front of it, then the actions that survive, with any combo they form in force.
//
// **Combos are matched against what is left after a stagger, not against the queue.** The
// player queued five attacks; a stagger that ate two means three happened, and a combo
// scored off cards that never resolved would let a staggered duelist stagger back with a
// turn they did not take.
func playTurn(
	events []Event,
	side Side,
	actor, target Duelist,
	turn []Slot,
	round int,
	table []Combo,
) ([]Event, Duelist, Duelist) {
	actor = expireDefenses(actor)

	// Stagger comes off the front, which needs no tie-break and so is the only pick that is
	// deterministic without inventing a rule. Under phases the front of a turn is its setups,
	// so being staggered costs a Prepare before it costs an attack — a real consequence, and
	// the one that makes stagger worth planning around rather than merely suffering.
	//
	// **The action points are not refunded.** They were committed when the cards were queued,
	// and letting them come back would make stagger pure tempo; keeping them spent makes it
	// tempo and economy both, which is the price a combo costing a whole round's budget to
	// set up should command. Recorded as the open question it was in TODO.md.
	lost := actor.Staggered
	actor.Staggered = 0
	if lost == StaggerAll || lost > len(turn) {
		lost = len(turn)
	}
	for i := 0; i < lost; i++ {
		events = append(events, Event{
			Kind:   KindStaggered,
			Side:   side,
			Action: turn[i].Action,
			Round:  round,
		})
	}
	turn = turn[lost:]

	// One running multiplier for the turn. Combos compose by multiplying, so two overlapping
	// rewards cannot be made to disagree about which one wins.
	num, den := 1, 1
	hits := matchSlots(turn, table)

	for i, slot := range turn {
		for _, h := range hits {
			if h.Start != i {
				continue
			}
			events = append(events, Event{
				Kind:  KindCombo,
				Side:  side,
				Combo: h.ID,
				Round: round,
			})

			if h.Effect.DamageNum > 0 && h.Effect.DamageDen > 0 {
				num, den = num*h.Effect.DamageNum, den*h.Effect.DamageDen
			}
			if h.Effect.BankAP != 0 {
				actor.PreparedAP += h.Effect.BankAP
				events = append(events, Event{
					Kind:   KindPrepared,
					Side:   side,
					Amount: h.Effect.BankAP,
					Round:  round,
				})
			}
			if h.Effect.Stagger != 0 {
				target.Staggered = addStagger(target.Staggered, h.Effect.Stagger)
			}
		}

		events, actor, target = resolveAction(events, side, actor, target, slot.Action, round, num, den)

		// Either side can fall here: a Riposte kills the attacker who walked into it.
		if !actor.Alive() || !target.Alive() {
			break
		}
	}

	return events, actor, target
}

// addStagger combines an existing stagger with a new one. StaggerAll absorbs everything —
// there is nothing above "the whole turn" for a count to add to.
func addStagger(existing, add int) int {
	if existing == StaggerAll || add == StaggerAll {
		return StaggerAll
	}
	return existing + add
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
// num/den is the damage multiplier any combo in force has put on this turn, carried down to
// resolveAttack. 1/1 when nothing is boosting.
func resolveAction(
	events []Event,
	side Side,
	actor, target Duelist,
	action ActionKind,
	round int,
	num, den int,
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

	return resolveAttack(events, side, targetSide, actor, target, action, round, num, den)
}

// resolveAttack lands one attack, or fails to. A pending Riposte or Dodge on the target
// stops it dead; a raised Guard merely halves it.
//
// **A combo multiplier scales the blow before the Guard halves it**, so a guard is worth the
// same fraction against a boosted attack as against a plain one. Halving first and then
// multiplying would make Guard progressively worse the bigger the hit, which is the opposite
// of what a defensive card is for. The counter-damage from a Riposte is deliberately outside
// this: it is the *defender* hitting back, and it is not part of the attacker's combo.
func resolveAttack(
	events []Event,
	side, targetSide Side,
	actor, target Duelist,
	action ActionKind,
	round int,
	num, den int,
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

	dmg := scaleDamage(action.Damage(actor.Str), num, den)
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

// PlanStyle is how an opponent fights. It exists because "negates one attack" is priced
// against how many attacks arrive, and for as long as every enemy spent its whole budget
// on two big swings, two defensive cards bought total immunity — a duel that ran three
// rounds taking 0, 0 and 2 damage. That is not a fault in Dodge's cost. It is one opponent
// wearing one shape, and the shape is what the player is really buying answers to.
//
// Each style is a pure function of a Duelist, so an enemy's whole plan is reproducible from
// its stats and what it banked last round. No randomness, no clock, no memory beyond the
// Duelist itself.
//
// **These are behaviours, not the enemy model.** MECHANICS.md decides that enemies get a
// deck and that an affix transforms it, which subjects the opponent to the same "what did I
// draw" pressure the player faces. That is a bigger build needing its own shuffle stream; a
// deck-driven planner will arrive as one more style beside these rather than replacing the
// idea of a style.
type PlanStyle int

const (
	// StyleBrute spends everything on the biggest attack it can afford. Few, heavy blows —
	// the shape every enemy used to have, kept because it is a fine *first* opponent and a
	// useful baseline. Dodge is strong against it, which is the point of it not being alone.
	StyleBrute PlanStyle = iota

	// StyleSwarm attacks as many times as the round allows. This is the answer to the
	// immunity problem: a negation stops one blow, so five cheap ones walk straight through
	// a pair of them, and Guard's flat halving is suddenly worth its 3 points.
	StyleSwarm

	// StyleWarden opens with a Guard and attacks with the rest, so the player's damage is
	// halved until they can punch through it. This is what makes Heavy worth 4.
	StyleWarden

	// StyleTactician banks action points and then unloads them. It alternates a light
	// setup round against an oversized one, which is a rhythm the player can read and
	// answer — guard the spike, punish the setup.
	StyleTactician
)

func (p PlanStyle) String() string {
	switch p {
	case StyleSwarm:
		return "swarm"
	case StyleWarden:
		return "warden"
	case StyleTactician:
		return "tactician"
	default:
		return "brute"
	}
}

// PlanStyles is every style, in a fixed order, for anything that has to walk them.
func PlanStyles() []PlanStyle {
	return []PlanStyle{StyleBrute, StyleSwarm, StyleWarden, StyleTactician}
}

// ParsePlanStyle reads a style out of a data record, falling back to brute so a typo in
// JSON produces a fightable enemy rather than a panic or a duelist that stands still.
func ParsePlanStyle(s string) (PlanStyle, bool) {
	for _, p := range PlanStyles() {
		if p.String() == s {
			return p, true
		}
	}
	return StyleBrute, false
}

// PlanFor builds one round's plan in the given style. Every style is bounded by both the
// action-point budget and MaxActions, and none may return a plan its duelist cannot pay
// for — TestEveryStyleObeysBothBounds holds all of them to that.
func PlanFor(style PlanStyle, d Duelist) []ActionKind {
	budget, slots := d.ActionPoints(), d.MaxActions()

	switch style {
	case StyleSwarm:
		return planSwarm(budget, slots)
	case StyleWarden:
		return planWarden(budget, slots)
	case StyleTactician:
		return planTactician(d, budget, slots)
	default:
		return planBrute(budget, slots)
	}
}

// planBrute fills with the most expensive attack that still fits.
func planBrute(budget, slots int) []ActionKind {
	plan := make([]ActionKind, 0, slots)

	for len(plan) < slots {
		switch {
		case budget >= costHeavy:
			plan, budget = append(plan, Heavy), budget-costHeavy
		case budget >= costStrike:
			plan, budget = append(plan, Strike), budget-costStrike
		case budget >= costQuick:
			plan, budget = append(plan, Quick), budget-costQuick
		default:
			return plan
		}
	}
	return plan
}

// planSwarm takes as many separate attacks as the round allows, then spends whatever is
// left over making those attacks bigger rather than adding a sixth it has no slot for.
//
// Widening first and upgrading second is the whole character of the style: a swarm with
// more points does not become a brute, it becomes a swarm that hurts. Without the upgrade
// pass a fast swarm would simply waste the points its speed bought it.
func planSwarm(budget, slots int) []ActionKind {
	plan := make([]ActionKind, 0, slots)
	for len(plan) < slots && budget >= costQuick {
		plan, budget = append(plan, Quick), budget-costQuick
	}

	// Quick -> Strike costs 1 more, Strike -> Heavy costs 2 more. Upgrade along the plan
	// rather than pouring everything into the first slot, so the round stays wide.
	for _, step := range []struct {
		from, to ActionKind
	}{{Quick, Strike}, {Strike, Heavy}} {
		gap := step.to.Cost() - step.from.Cost()
		for i := range plan {
			if budget < gap {
				break
			}
			if plan[i] == step.from {
				plan[i], budget = step.to, budget-gap
			}
		}
	}
	return plan
}

// planWarden puts a Guard up first and attacks with what is left. The Guard is a setup, so
// resolution moves it to the front of the turn regardless of where it sits here — it is
// first in the slice only because that is how the plan reads.
func planWarden(budget, slots int) []ActionKind {
	if budget < costGuard || slots < 1 {
		return planBrute(budget, slots)
	}
	return append([]ActionKind{Guard}, planBrute(budget-costGuard, slots-1)...)
}

// planTactician alternates between banking and spending, reading which round it is in off
// its own BonusAP: anything banked means last round was the setup, so this one is the
// payoff. It needs no memory of its own because Prepare already leaves the evidence.
func planTactician(d Duelist, budget, slots int) []ActionKind {
	if d.BonusAP > 0 {
		// The payoff round. Everything into the biggest attacks that fit.
		return planBrute(budget, slots)
	}

	// The setup round: bank what a second Prepare is worth, keep enough back to stay
	// dangerous, and never spend so much preparing that the round is a free hit for the
	// player. Two Prepares is the whole point — it is what makes the next round oversized.
	plan := make([]ActionKind, 0, slots)
	for len(plan) < slots-1 && budget >= costPrepare*2 && len(plan) < 2 {
		plan, budget = append(plan, Prepare), budget-costPrepare
	}
	return append(plan, planBrute(budget, slots-len(plan))...)
}
