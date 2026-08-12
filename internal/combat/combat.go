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
	//
	// A Feint strips one of these without triggering its counter, which is the only way a
	// negation leaves without stopping something.
	Ripostes int
	Dodges   int

	// Braces halve a single incoming attack each and are spent doing it, where Guarded halves
	// every attack and is not. That is the whole difference between the 1-AP defence and the
	// 3-AP prepare: one card, one blow, versus one card, one turn.
	//
	// A brace and a guard both applying quarter the blow. Deliberate — they are different
	// cards bought separately, and a rule that silently ignored one of them would make the
	// cheaper card worthless exactly when the player had committed to both.
	Braces int

	// Mirrored negates every attack in the opponent's next turn and reflects each one back.
	// A bool rather than a count because it is turn-wide like Guarded, not per-blow like a
	// Dodge — and because "every attack" has no number to run out of.
	//
	// It is checked *before* the counted negations, so a Mirror never lets a Dodge or Riposte
	// be spent on a blow it was going to stop for free.
	Mirrored bool

	// The two halves of Gather. GatheredAP is what has been banked *this* round; at the
	// round boundary it becomes BonusAP, which ActionPoints adds to the budget for the
	// round after. Splitting them is what makes the bonus arrive next round rather than
	// funding the turn that bought it.
	//
	// BonusAP is overwritten rather than added to at the boundary, so gathering twice in
	// one round is worth +4 next round while gathering once a round is worth a flat +2.
	// Stacking within a round is deliberate and is what puts the five-attack combo in
	// reach without a ring; carrying across rounds would compound without limit.
	BonusAP    int
	GatheredAP int

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

	// Statuses is what elements have done to this duelist, indexed by the element that did it.
	// See status.go for the lifecycle, which is one rule for all four.
	//
	// **An array indexed by Element rather than four named fields**, and that is the load-bearing
	// choice. It means a fifth element does not grow this struct, and — the reason it is worth
	// doing at all — it makes *"consume the status this element applies"* expressible, which is
	// what MECHANICS.md's Extinguishing Strike needs and what turns four ad-hoc fields into a
	// system. The price is that Element is now append-only: inserting one mid-enum re-points
	// every status a duelist is carrying.
	//
	// The defences above deliberately stay where they are. Guard and Brace are card effects, not
	// element statuses, and filing them in a table indexed by colour would say they were.
	Statuses [ElementCount]Status
}

// Alive reports whether this duelist can still fight.
func (d Duelist) Alive() bool { return d.CurrentLife > 0 }

// Category is which phase of a turn an action resolves in, and the axis the whole round
// is now built on. It is a property of the action, not an independent choice: a fire Guard
// and a plain Guard are both prepares.
type Category int

const (
	CategoryPrepare Category = iota
	CategoryAttack
	CategoryDefend
)

// Categories is every phase in resolution order, and the order a turn is played in.
// Defenses come last within a turn because the *opponent* acts afterwards — a defense
// raised at the end of your turn is up when their blow arrives.
func Categories() []Category {
	return []Category{CategoryPrepare, CategoryAttack, CategoryDefend}
}

func (c Category) String() string {
	switch c {
	case CategoryPrepare:
		return "prepare"
	case CategoryAttack:
		return "attack"
	case CategoryDefend:
		return "defend"
	default:
		return "?"
	}
}

type ActionKind int

// Declared in category order, and **within a category in ascending cost order**, so anything
// that sorts by the raw value — the deck overlay does — groups the piles the same way a turn
// resolves them and ladders them by price inside that.
//
// The twelve are a filled 3x4 grid: three categories by four cost tiers. See MECHANICS.md for
// why twelve rather than however many somebody thought of, and why each tier is required to
// differ in *kind* from the one below it rather than only in magnitude.
//
// **Inserting these five mid-enum shifted every combo ID**, because FlurryID is derived from
// the card's raw value. That was safe exactly once — no profile exists yet, so no discovered
// combo could be re-locked by the shift — and it stops being safe the moment one does. The
// conflict between category ordering here and ID stability there is recorded as an open
// question in MECHANICS.md.
const (
	// Prepare.
	Gather ActionKind = iota
	Sift
	Guard
	Ritual

	// Attack.
	Jab
	Strike
	Feint
	Heavy

	// Defend.
	Brace
	Dodge
	Riposte
	Mirror
)

// AllActions is every action a duelist can queue, in category order.
var AllActions = []ActionKind{
	Gather, Sift, Guard, Ritual,
	Jab, Strike, Feint, Heavy,
	Brace, Dodge, Riposte, Mirror,
}

func (a ActionKind) String() string {
	switch a {
	case Gather:
		return "Gather"
	case Sift:
		return "Sift"
	case Guard:
		return "Guard"
	case Ritual:
		return "Ritual"
	case Jab:
		return "Jab"
	case Strike:
		return "Strike"
	case Feint:
		return "Feint"
	case Heavy:
		return "Heavy"
	case Brace:
		return "Brace"
	case Dodge:
		return "Dodge"
	case Riposte:
		return "Riposte"
	case Mirror:
		return "Mirror"
	default:
		return "Unknown"
	}
}

// ParseAction resolves an action from its name, which is how a data record names a concept.
// The bool reports whether it was found rather than falling back to something playable —
// unlike ParsePlanStyle, where a typo should still produce a fightable enemy. A deck quietly
// short a concept is a balance change nobody made, so the caller is expected to fail loudly.
func ParseAction(name string) (ActionKind, bool) {
	for _, a := range AllActions {
		if a.String() == name {
			return a, true
		}
	}
	return Gather, false
}

// Category is which phase this action resolves in.
func (a ActionKind) Category() Category {
	switch a {
	case Gather, Sift, Guard, Ritual:
		return CategoryPrepare
	case Brace, Dodge, Riposte, Mirror:
		return CategoryDefend
	default:
		return CategoryAttack
	}
}

// Action point costs. The budget is the decision: a couple of big swings, or a
// fistful of small ones, or damage traded away for a defense.
// The four tiers are 1/2/3/4 in every category, which is what makes the grid in MECHANICS.md a
// grid rather than a table of coincidences.
const (
	costGather = 1
	costSift   = 2
	costGuard  = 3
	costRitual = 4

	costJab    = 1
	costStrike = 2
	costFeint  = 3
	costHeavy  = 4

	costBrace   = 1
	costDodge   = 2
	costRiposte = 3
	costMirror  = 4
)

// Cost is what this action takes out of the round's action-point budget.
func (a ActionKind) Cost() int {
	switch a {
	case Gather:
		return costGather
	case Sift:
		return costSift
	case Guard:
		return costGuard
	case Ritual:
		return costRitual
	case Jab:
		return costJab
	case Feint:
		return costFeint
	case Heavy:
		return costHeavy
	case Brace:
		return costBrace
	case Dodge:
		return costDodge
	case Riposte:
		return costRiposte
	case Mirror:
		return costMirror
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

// gatherBonusAP is what one Gather banks for the following round. Two for one is a
// deliberate profit — the cost of Gather is the card slot and the action slot it takes
// out of the round it is played in, not the point it costs.
const gatherBonusAP = 2

// ritualBonusAP is Ritual's bank. **Deliberately the same net rate as Gather** — 1 AP for +2 and
// 4 AP for +5 are both net +1 per point spent — so Ritual is not a better Gather, it is a Gather
// that does not eat the round. Four Gathers bank more (+8) but spend four of five action slots;
// Ritual banks +5 and leaves four slots to fight with. The difference it sells is *slots*.
//
// An earlier draft granted +2 AP and +2 to the action cap instead, to reach six- and seven-card
// combo runs. That was cut when the cap was frozen at five permanently: a growable cap dilutes
// every named hand shape as it grows. See MECHANICS.md.
const ritualBonusAP = 5

// braceDivisor is how much a spent brace cuts one incoming attack. Its own constant rather than
// a shared one with guardDivisor: the two are the same number today and are different rules, so
// tuning one must not silently move the other.
const braceDivisor = 2

// mirrorReflectNum/Den is the fraction of a stopped blow a Mirror sends back, as a fraction so
// this package stays integer arithmetic. **1/1 — full reflection, as designed.**
//
// **`tools/balance` says this is too strong, and the number is here so the fix is one line.**
// With full reflection the `mirroring` posture beats all four shipped enemies and finishes every
// one of them on 60/60 life, which is the "strong against everything" condition that means a
// card is mispriced rather than good. It is also, accidentally, the only posture that beats
// Swarm1 — an enemy MECHANICS.md records as unbeatable by all three original postures — so it
// papers over a known balance hole by being overpowered instead of by being right.
//
// Left at 1/1 deliberately: full reflection is the card as specified, and quietly halving a
// design decision to make a table look better is the wrong way round. The two candidate levers,
// in the order worth trying:
//
//   - **Halve the reflection** (1/2 here). Keeps the card's character — it still scales with what
//     the opponent committed — and stops it being a better Dodge in every matchup.
//   - **Cap what it stops.** Negating a whole turn for 4 AP is the other half of the problem;
//     "negates and reflects the first two attacks" is a different card but a priceable one.
const (
	mirrorReflectNum = 1
	mirrorReflectDen = 1
)

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
// banked by a Gather in the round before and less anything an ice hit has taken off.
//
// **A chill is read here rather than subtracted when it lands**, which is what makes ice bite
// the round after the blow rather than the round it was struck in — the budget for the round in
// progress has already been committed by the time an attack resolves.
func (d Duelist) ActionPoints() int {
	ap := baseActionPoints + d.Spd/speedPerPoint + d.BonusAP - d.chill()
	if ap < 1 {
		ap = 1
	}
	return ap
}

// CostOf totals the action-point cost of a queued set.
func CostOf(cards []Card) int {
	total := 0
	for _, c := range cards {
		total += c.Cost()
	}
	return total
}

// CanAfford reports whether a queued set fits inside this duelist's budget. The UI
// enforces this while the player builds a set; ResolveRound trusts what it is given
// so that a balance sim can deliberately probe outside the rules.
func (d Duelist) CanAfford(cards []Card) bool {
	return CostOf(cards) <= d.ActionPoints()
}

// guardDivisor is how much a raised guard cuts incoming damage.
const guardDivisor = 2

// Damage is what an action deals given the attacker's Strength.
//
// Riposte is the odd one: it is a *defend*, and the number below is what it hits back for
// when it stops an attack, not something it deals on its own. Reporting it here is what
// lets the card draw a damage badge for it without the screen knowing the rule.
// Feint deals a Strike's damage and pays its extra point for the negation it strips, not for a
// bigger number — see resolveAttack.
func (a ActionKind) Damage(str int) int {
	switch a {
	case Heavy:
		return str * 2
	case Strike, Feint:
		return str
	case Jab, Riposte:
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
	KindGathered
	KindNegated
	KindGuarded

	// KindBraced is one attack halved by a spent brace, and reports the reduced figure exactly
	// as KindGuarded does. Separate from KindGuarded because they are different cards with
	// different costs, and a log that called both "guarded" would make the 1-AP defence
	// invisible in the one place the player looks to find out what happened.
	KindBraced

	// KindStripped is a negation removed by a Feint without being spent on anything. Action is
	// the negation that went; Target is whose it was. It is not a KindNegated because nothing
	// was negated — the whole point of the card is that the defence leaves without stopping a
	// blow and without firing its counter.
	KindStripped

	KindDamage
	KindDefeated

	// KindCombo says a combo formed, and is emitted at the first card of the run rather than
	// the last — the effects come into force there, so narrating it later would show the
	// player a boosted hit before telling them why.
	KindCombo

	// KindStaggered is one action lost to a stagger. One event per action, so a stagger that
	// takes a whole round narrates as the several things it actually is.
	KindStaggered

	// The three element events, added 2026-08-12 with the statuses.

	// KindStatus is one element status landing on a duelist. Element says which, Amount says how
	// much was added by this hit, Target is who is carrying it.
	//
	// It is a separate event rather than a field on KindDamage because a status is not the blow:
	// a chill that lands is felt a round later and against a completely different card, and a
	// Resolution feed that folded it into the damage line would announce it at the one moment it
	// does nothing.
	KindStatus

	// KindMissed is an attack that never happened because its owner was shocked. Action is the
	// attack that was lost and Side is whose it was, which makes it the lightning counterpart of
	// KindStaggered — a slot that resolves into nothing.
	//
	// It is deliberately not a KindNegated: nothing of the defender's stopped it, and a log
	// saying a blow was "stopped cold" by a defence that was never raised would send the player
	// looking for a card that is not there.
	KindMissed

	// KindBurned is a fire tick at the end of a round. Target is who burned; Side is the same,
	// because nobody acted.
	KindBurned

	KindRoundEnd
)

// Event is one entry in the replayable log for a single round.
type Event struct {
	Kind   EventKind
	Side   Side       // who acted
	Action ActionKind // set on KindAction, on KindNegated for the defense that stopped it, on KindStaggered for the action lost, and on KindMissed for the attack that never landed
	Amount int        // damage dealt, action points banked, or status applied
	Target Side       // who took the damage
	Life   int        // target's life after the event
	Round  int

	// Element is the card's element on KindAction and KindMissed, and which status is meant on
	// KindStatus and KindBurned. Basic everywhere else, which is also the zero value — an event
	// with nothing to say about colour says `basic`, exactly as a plain card does.
	Element Element

	// Combo is set on KindCombo, and names which one fired. The screen looks it up with
	// ComboByID rather than being told its name here, so a combo renamed is renamed once.
	Combo ComboID

	// ComboStart and ComboLength are set on KindCombo alongside Combo: which run of this
	// side's turn formed it, as an offset and a count into the turn *as it was played*.
	//
	// **They are here so a screen never has to work out which cards earned a combo.** The
	// matcher already knows — ComboHit carries exactly these two numbers — and re-deriving
	// them from the combo's pattern length would be a second matcher, which is the drift
	// ResolutionOrder exists to prevent. Matching is greedy and longest-first, so a run of
	// five Strikes forms one Onslaught and no Flurries; a screen counting cards backwards
	// from the event would confidently bracket the wrong three.
	//
	// The offset counts the actions that actually resolved, staggered ones already removed,
	// which is the same sequence as this side's KindAction events.
	ComboStart, ComboLength int
}

// Slot is one card's place in a round's resolution order: whose it is, where it sits
// in that side's queue, and what it is.
//
// **It holds a whole Card rather than a bare ActionKind** since 2026-08-12. A slot is what both
// the engine and the screen walk, so anything that has to know a card's element while a round is
// being ordered — a combo matching on colour, a row drawing a border — reads it here.
type Slot struct {
	Side  Side
	Index int
	Card  Card
}

// ResolutionOrder is the sequence in which two queued sets resolve, and the single
// authority on that order. ResolveRound plays it and the combat screen's Resolution pane
// draws it; neither works the order out for itself, so the pane and the engine cannot
// drift apart.
//
// **A whole turn each, in category order.** Everything side A queued resolves — prepares,
// then attacks, then defenses — and only then does side B begin. Within a category the
// queued order is kept, which is where drag-to-reorder still bites and where sequence
// combos will match.
//
// Index is the action's position in its own side's queue, which is *not* its position
// here: reordering by category is the whole job. Consumers wanting "how far through the
// round are we" should count slots rather than read Index.
func ResolutionOrder(aCards, bCards []Card) []Slot {
	slots := make([]Slot, 0, len(aCards)+len(bCards))
	slots = appendTurn(slots, SideA, aCards)
	slots = appendTurn(slots, SideB, bCards)
	return slots
}

// appendTurn adds one side's whole turn, category by category.
func appendTurn(slots []Slot, side Side, cards []Card) []Slot {
	for _, cat := range Categories() {
		for i, c := range cards {
			if c.Category() == cat {
				slots = append(slots, Slot{Side: side, Index: i, Card: c})
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
func ResolveRound(a, b Duelist, aCards, bCards []Card, round int) (events []Event, aAfter, bAfter Duelist) {
	return resolveRound(a, b, aCards, bCards, round, comboTable)
}

// resolveRound is ResolveRound with the combo table injected. It exists so a test can drive
// a synthetic combo through the whole engine rather than only through the matcher — the
// damage multiplier and the banked points would otherwise be code paths that shipped without
// ever having been run, since the two combos the game currently has both use stagger.
func resolveRound(a, b Duelist, aCards, bCards []Card, round int, table []Combo) (events []Event, aAfter, bAfter Duelist) {
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
	events, a, b = playTurn(events, SideA, a, b, appendTurn(nil, SideA, aCards), round, table)

	// B still loses its standing defenses even in a round it never gets to act in, which is
	// why this is not inside playTurn's early return: expiry is a property of the turn
	// arriving, not of anything happening in it.
	if a.Alive() && b.Alive() {
		events, b, a = playTurn(events, SideB, b, a, appendTurn(nil, SideB, bCards), round, table)
	} else {
		b = expireDefenses(b)
	}

	// **A always burns before B**, which is the same order the turns were played in and needs no
	// tie-break.
	events, a = endRound(events, SideA, a, round)
	events, b = endRound(events, SideB, b, round)

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
	// deterministic without inventing a rule. Under phases the front of a turn is its prepares,
	// so being staggered costs a Gather before it costs an attack — a real consequence, and
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
			Kind:    KindStaggered,
			Side:    side,
			Action:  turn[i].Card.Action,
			Element: turn[i].Card.Element,
			Round:   round,
		})
	}
	turn = turn[lost:]

	// One running multiplier for the turn. Combos compose by multiplying, so two overlapping
	// rewards cannot be made to disagree about which one wins.
	num, den := 1, 1
	hits := matchSlots(turn, table)

	for i, slot := range turn {
		// The action first, then any combo it *completes*. **A combo lands after the cards
		// that formed it** — changed 2026-08-07, having first been built the other way round.
		//
		// Firing at the run's first card let a damage multiplier boost the very cards that
		// earned it, which is why it was built that way. Two things beat that:
		//
		//   - **It read backwards.** "COMBO! Strike Flurry" arrived above three strikes that
		//     had not happened yet, announcing a thing before its cause.
		//   - **A run that never finished still paid out.** Matching happens up front, so a
		//     turn cut short by a Riposte kill on the second of three strikes fired the combo
		//     anyway. Firing on completion makes that impossible by construction rather than
		//     by a check somebody has to remember.
		//
		// The cost, recorded honestly: a damage multiplier can now only boost what comes
		// *after* the run. "Complete the combo, get a bonus for the rest of your turn" is a
		// coherent reading, but it is not the one the multiplier was designed against, and
		// neither shipping combo uses one — so this is untested by anything real yet.
		events, actor, target = resolveAction(events, side, actor, target, slot.Card, round, num, den)

		// Either side can fall here: a Riposte kills the attacker who walked into it. A combo
		// completing on this slot is skipped along with the rest of the turn, which is the
		// point above.
		if !actor.Alive() || !target.Alive() {
			break
		}

		for _, h := range hits {
			if i != h.Start+h.Length-1 {
				continue
			}
			events = append(events, Event{
				Kind:        KindCombo,
				Side:        side,
				Combo:       h.ID,
				ComboStart:  h.Start,
				ComboLength: h.Length,
				Round:       round,
			})

			if h.Effect.DamageNum > 0 && h.Effect.DamageDen > 0 {
				num, den = num*h.Effect.DamageNum, den*h.Effect.DamageDen
			}
			if h.Effect.BankAP != 0 {
				actor.GatheredAP += h.Effect.BankAP
				events = append(events, Event{
					Kind:   KindGathered,
					Side:   side,
					Amount: h.Effect.BankAP,
					Round:  round,
				})
			}
			if h.Effect.Stagger != 0 {
				target.Staggered = addStagger(target.Staggered, h.Effect.Stagger)
			}
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
	d.Braces = 0
	d.Mirrored = false
	return d
}

// endRound rolls what was banked this round into next round's budget, ticks a burn, and counts
// every status down one. Assignment rather than addition for the bank: two Gathers in one round
// are worth +4 next round, and gathering every round is worth a flat +2 rather than compounding
// forever.
//
// **The burn ticks before the countdown**, so a fire hit lands damage at the end of the round it
// was struck in as well as the round after. MECHANICS.md says a DoT "lands at end of round" and
// this is the end of the round it was applied in; making it wait would mean a fire attack did
// nothing at all in a duel that ended on the round it was played.
//
// **A dead duelist does not burn.** The first version ticked regardless, on the grounds that
// skipping a corpse would make the order of two deaths matter — it does not, because whether a
// duelist is dead is settled before either side's round-end runs. What it did instead was
// announce a second `KindDefeated` over a body, and the Resolution feed duly read
// "Goblin falls / Goblin burns for 2 / Goblin falls". Statuses still tick down, so a duelist
// somehow revived does not wake up carrying an expired burn.
func endRound(events []Event, side Side, d Duelist, round int) ([]Event, Duelist) {
	if burn := d.Statuses[Fire]; burn.Active() && d.Alive() {
		d.CurrentLife = reduce(d.CurrentLife, burn.Amount)

		// Side and Target are both this duelist, because nobody acted. The fire was applied by an
		// attack rounds ago and the duelist that lit it may not even be the one alive to see this.
		events = append(events, Event{
			Kind:    KindBurned,
			Side:    side,
			Target:  side,
			Element: Fire,
			Amount:  burn.Amount,
			Life:    d.CurrentLife,
			Round:   round,
		})

		if !d.Alive() {
			events = append(events, Event{
				Kind:   KindDefeated,
				Side:   side,
				Target: side,
				Round:  round,
			})
		}
	}

	d = tickStatuses(d)

	d.BonusAP = d.GatheredAP
	d.GatheredAP = 0
	return events, d
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
	card Card,
	round int,
	num, den int,
) ([]Event, Duelist, Duelist) {
	targetSide := other(side)
	action := card.Action

	events = append(events, Event{
		Kind:    KindAction,
		Side:    side,
		Action:  action,
		Element: card.Element,
		Round:   round,
	})

	switch action {
	case Gather, Ritual:
		bank := gatherBonusAP
		if action == Ritual {
			bank = ritualBonusAP
		}
		actor.GatheredAP += bank
		events = append(events, Event{
			Kind:   KindGathered,
			Side:   side,
			Amount: bank,
			Round:  round,
		})
		return events, actor, target

	case Sift:
		// **Sift has no effect in the rules, and that is the design rather than a gap.** It
		// manipulates the hand, and the deck deliberately lives on the scene so this package
		// stays free of a shuffle — see CLAUDE.md on determinism. The screen does the work at
		// the round boundary; the engine's only job is to charge the action points and give the
		// card a slot in the resolution order. It is also the one concept tools/balance is
		// structurally unable to see.
		return events, actor, target

	case Guard:
		actor.Guarded = true
		return events, actor, target

	case Brace:
		actor.Braces++
		return events, actor, target

	case Dodge:
		actor.Dodges++
		return events, actor, target

	case Riposte:
		actor.Ripostes++
		return events, actor, target

	case Mirror:
		actor.Mirrored = true
		return events, actor, target
	}

	return resolveAttack(events, side, targetSide, actor, target, card, round, num, den)
}

// resolveAttack lands one attack, or fails to. A pending Riposte or Dodge on the target
// stops it dead; a raised Guard merely halves it.
//
// **A combo multiplier scales the blow before the Guard halves it**, so a guard is worth the
// same fraction against a boosted attack as against a plain one. Halving first and then
// multiplying would make Guard progressively worse the bigger the hit, which is the opposite
// of what a defensive card is for. The counter-damage from a Riposte is deliberately outside
// this: it is the *defender* hitting back, and it is not part of the attacker's combo.
//
// **The order of the whole calculation is: concept damage, combo multiplier, the attacker's own
// earth weight, then the defender's brace and guard.** Weight sits where it does because it is a
// property of the *attacker* — it says how hard they can still swing — so everything the
// defender does happens to a blow that has already been blunted. That is also why the reflection
// a Mirror sends back is the blunted figure: it is what the attacker actually committed.
func resolveAttack(
	events []Event,
	side, targetSide Side,
	actor, target Duelist,
	card Card,
	round int,
	num, den int,
) ([]Event, Duelist, Duelist) {
	action := card.Action

	// A shocked attacker misses outright, and misses before anything else happens — no Feint
	// strip, no negation spent, no status applied. The attack did not occur.
	//
	// **It is certain rather than a roll.** See shockPerHit for why this package has no
	// randomness in it and is not about to acquire any.
	if shocked, missed := spendShock(actor); missed {
		actor = shocked
		events = append(events, Event{
			Kind:    KindMissed,
			Side:    side,
			Action:  action,
			Element: card.Element,
			Target:  targetSide,
			Round:   round,
		})
		return events, actor, target
	}

	// A Feint strips one counted negation before anything else resolves, and takes no counter
	// for it. That is what the extra point over a Strike buys: attacking into a Riposte
	// normally costs the attacker str/2, and this is the card that clears one safely.
	//
	// **Unconditional, and deliberately so.** It fires even when the blow is about to be
	// stopped by a Mirror, because a strip that silently did nothing depending on a card the
	// player cannot see would be a hidden interaction in a game whose whole point is reading
	// the opponent. Ripostes before Dodges, matching the order they are spent in — the Riposte
	// is the one worth removing.
	//
	// Guarded and Mirrored are turn-wide states, not pending negations, and a Feint does not
	// touch either.
	if action == Feint && (target.Ripostes > 0 || target.Dodges > 0) {
		stripped := Riposte
		if target.Ripostes > 0 {
			target.Ripostes--
		} else {
			target.Dodges--
			stripped = Dodge
		}
		events = append(events, Event{
			Kind:   KindStripped,
			Side:   side,
			Action: stripped,
			Target: targetSide,
			Round:  round,
		})
	}

	// A Mirror stops everything and sends it back, and it is checked before the counted
	// negations so it never lets a Dodge or Riposte be spent on a blow that was going to be
	// stopped for free.
	//
	// **The reflected figure is the damage after any combo multiplier and before any guard.**
	// After the multiplier because what it reflects is what the attacker actually committed —
	// which is what makes Mirror the answer to a telegraphed spike rather than a flat damage
	// reduction. Before any guard because the blow never lands, so nothing of the defender's
	// ever reduces it; and the defender's own Guard is irrelevant to what the attacker threw.
	// This follows Riposte's counter, which is likewise raw.
	if target.Mirrored {
		reflected := scaleDamage(
			blunt(scaleDamage(card.Damage(actor.Str), num, den), actor.weightPct()),
			mirrorReflectNum, mirrorReflectDen)

		events = append(events, Event{
			Kind:   KindNegated,
			Side:   targetSide,
			Action: Mirror,
			Target: side,
			Round:  round,
		})

		// The reflection runs the other way, so the sides here are swapped relative to the
		// attack that provoked it — the same shape as a Riposte's counter.
		actor.CurrentLife = reduce(actor.CurrentLife, reflected)
		events = append(events, Event{
			Kind:   KindDamage,
			Side:   targetSide,
			Target: side,
			Amount: reflected,
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

	// The attacker's own earth weight blunts the blow before any of the defender's cards touch
	// it. A Riposte's counter is deliberately outside this, exactly as it is outside the combo
	// multiplier: it is the defender hitting back, not a card the weighted duelist played.
	dmg := blunt(scaleDamage(card.Damage(actor.Str), num, den), actor.weightPct())

	// A brace is spent on this blow; a guard is not spent at all. Both can apply, quartering
	// the hit — deliberate, since they are two cards bought separately and a rule that ignored
	// one of them would make the cheaper card worthless exactly when the player committed to
	// both. Brace goes first so the log reads cheap-then-broad, in cost order.
	if target.Braces > 0 {
		target.Braces--
		dmg /= braceDivisor
		events = append(events, Event{
			Kind:   KindBraced,
			Side:   side,
			Target: targetSide,
			Amount: dmg,
			Life:   target.CurrentLife,
			Round:  round,
		})
	}

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

	// **The status lands because the blow did, not because it hurt.** A Jab guarded down to zero
	// still connected, and making the status conditional on the final figure would mean a
	// defensive card silently un-applied an element the attacker had already paid for — the same
	// shape of hidden interaction the unconditional Feint strip above exists to avoid.
	//
	// After the damage event so the log reads blow-then-consequence, and after the death check
	// below would be wrong: a duelist who dies to the hit is not left carrying a chill.
	if applied, amount, ok := applyStatus(target, card.Element); ok {
		target = applied
		events = append(events, Event{
			Kind:    KindStatus,
			Side:    side,
			Target:  targetSide,
			Element: card.Element,
			Amount:  amount,
			Life:    target.CurrentLife,
			Round:   round,
		})
	}

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

// PlanFor builds one round's plan in the given style, **chosen from the hand the opponent
// was dealt**. Every style is bounded by the action-point budget and by MaxActions, and none
// may return a plan its duelist cannot pay for or a card it was not holding —
// TestEveryStyleObeysBothBounds holds all of them to that.
//
// **A style used to conjure its cards** *(changed 2026-08-11)*. A brute produced Heavies
// whether or not a Heavy existed anywhere in the game, which made an enemy deck a thing that
// could be authored and never read — and made MECHANICS.md's affixes, which *transform* an
// enemy's deck, unable to change anything. A style is now how a hand is played rather than
// what is played, and an enemy that draws no attacks does nothing that round.
//
// **The shuffle stays outside this package.** What arrives is a hand, already dealt, exactly
// as the player's hand reaches the screen — `internal/combat` keeps no randomness and no
// clock, which is what `TestRoundIsDeterministic` pins and what lets the balance tool run
// whole duels headlessly.
// **A planner reasons about concepts and carries elements along.** Every choice below is made
// on cost and category; the element rides on the card it was dealt on and reaches the round
// untouched. That is deliberate and it is currently free — `data/enemy_cards.json` is all basic,
// per MECHANICS.md's plan that an affix transforms an enemy's deck rather than the file naming
// colours. A style that *preferred* an element would be a fifth style, not a change here.
func PlanFor(style PlanStyle, d Duelist, hand []Card) []Card {
	budget, slots := d.ActionPoints(), d.MaxActions()
	p := newPool(hand)

	switch style {
	case StyleSwarm:
		return planSwarm(p, budget, slots)
	case StyleWarden:
		return planWarden(p, budget, slots)
	case StyleTactician:
		return planTactician(d, p, budget, slots)
	default:
		return planBrute(p, budget, slots)
	}
}

// pool is a hand being spent: the cards dealt, and which of them a plan has taken.
//
// **A slice with used flags rather than a count per kind.** A map of counts would be the
// obvious shape and would make every choice below depend on Go's randomised map iteration —
// which the determinism rules forbid outright, and which would show up as an enemy that
// planned differently on identical input. Order here is the order the cards were dealt in,
// and every scan below runs front to back so ties resolve the same way twice.
type pool struct {
	cards []Card
	used  []bool
}

func newPool(hand []Card) *pool {
	return &pool{cards: hand, used: make([]bool, len(hand))}
}

// take removes one card of a concept and hands it back whole, element included, reporting
// whether the hand held one. It matches on the concept alone: no style asks for a colour.
func (p *pool) take(k ActionKind) (Card, bool) {
	for i, c := range p.cards {
		if !p.used[i] && c.Action == k {
			p.used[i] = true
			return c, true
		}
	}
	return Card{}, false
}

// put returns a card to the hand. Used by the swarm's upgrade pass, which swaps a small
// attack out for a bigger one and has to make the small one available again.
//
// It matches the whole card, so a fire Jab put back frees a fire Jab. Two identical cards are
// interchangeable and it does not matter which of them is marked unused; two Jabs of different
// elements are not, and matching on the concept alone would quietly recolour a hand.
func (p *pool) put(card Card) {
	for i, c := range p.cards {
		if p.used[i] && c == card {
			p.used[i] = false
			return
		}
	}
}

// takeAttack removes the attack this style wants and reports it: the most expensive one
// inside the budget, or the cheapest, depending on which way the style leans. Ties go to the
// card dealt first, which is what makes the choice reproducible.
//
// `above` is a floor the cost has to beat, so the swarm's upgrade pass can ask for something
// strictly better than what it already has.
func (p *pool) takeAttack(budget int, dearest bool, above int) (Card, bool) {
	best, bestAt, found := Card{}, -1, false

	for i, c := range p.cards {
		if p.used[i] || c.Category() != CategoryAttack {
			continue
		}
		cost := c.Cost()
		if cost > budget || cost <= above {
			continue
		}
		switch {
		case !found:
			best, bestAt, found = c, i, true
		case dearest && cost > best.Cost():
			best, bestAt = c, i
		case !dearest && cost < best.Cost():
			best, bestAt = c, i
		}
	}

	if !found {
		return Card{}, false
	}
	p.used[bestAt] = true
	return best, true
}

// planBrute fills with the most expensive attack in hand that still fits.
func planBrute(p *pool, budget, slots int) []Card {
	plan := make([]Card, 0, slots)

	for len(plan) < slots {
		a, ok := p.takeAttack(budget, true, 0)
		if !ok {
			return plan
		}
		plan, budget = append(plan, a), budget-a.Cost()
	}
	return plan
}

// planSwarm takes as many separate attacks as the round allows, then spends whatever is
// left over making those attacks bigger rather than adding a sixth it has no slot for.
//
// Widening first and upgrading second is the whole character of the style: a swarm with
// more points does not become a brute, it becomes a swarm that hurts. Without the upgrade
// pass a fast swarm would simply waste the points its speed bought it.
//
// **The upgrade is now limited by the hand as well as the budget.** It swaps a planned
// attack for a dearer one still in the pool and puts the small one back, so a swarm holding
// six Jabs and nothing else stays six Jabs however many points it has spare — which is the
// deck doing its job rather than the planner failing.
func planSwarm(p *pool, budget, slots int) []Card {
	plan := make([]Card, 0, slots)
	for len(plan) < slots {
		a, ok := p.takeAttack(budget, false, 0)
		if !ok {
			break
		}
		plan, budget = append(plan, a), budget-a.Cost()
	}

	// Upgrade along the plan rather than pouring everything into the first slot, so the
	// round stays wide.
	for i := range plan {
		have := plan[i].Cost()
		better, ok := p.takeAttack(budget+have, true, have)
		if !ok {
			continue
		}
		p.put(plan[i])
		budget -= better.Cost() - have
		plan[i] = better
	}
	return plan
}

// planWarden puts a Guard up first and attacks with what is left. The Guard is a prepare, so
// resolution moves it to the front of the turn regardless of where it sits here — it is
// first in the slice only because that is how the plan reads.
//
// **No Guard in hand and it fights like a brute**, which is the deck showing through the
// style: a warden is a duelist that guards when it can, not one that always has a shield.
func planWarden(p *pool, budget, slots int) []Card {
	if budget < costGuard || slots < 1 {
		return planBrute(p, budget, slots)
	}
	guard, ok := p.take(Guard)
	if !ok {
		return planBrute(p, budget, slots)
	}
	return append([]Card{guard}, planBrute(p, budget-costGuard, slots-1)...)
}

// planTactician alternates between banking and spending, reading which round it is in off
// its own BonusAP: anything banked means last round was the setup, so this one is the
// payoff. It needs no memory of its own because Gather already leaves the evidence.
func planTactician(d Duelist, p *pool, budget, slots int) []Card {
	if d.BonusAP > 0 {
		// The payoff round. Everything into the biggest attacks that fit.
		return planBrute(p, budget, slots)
	}

	// The setup round: bank what a second Gather is worth, keep enough back to stay
	// dangerous, and never spend so much gathering that the round is a free hit for the
	// player. Two Gathers is the whole point — it is what makes the next round oversized,
	// and a hand holding one Gather is a setup round that only half works.
	plan := make([]Card, 0, slots)
	for len(plan) < slots-1 && budget >= costGather*2 && len(plan) < 2 {
		gather, ok := p.take(Gather)
		if !ok {
			break
		}
		plan, budget = append(plan, gather), budget-costGather
	}
	return append(plan, planBrute(p, budget, slots-len(plan))...)
}
