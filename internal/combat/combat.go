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

import "math/rand"

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
// duelists with ==, so nothing here may become a slice or a map. The defend queue is a fixed
// array plus a count rather than a slice for exactly that reason.
type Duelist struct {
	Con         int
	Str         int
	Spd         int
	MaxLife     int
	CurrentLife int

	// Defends is the plan cards this duelist has raised and not yet spent, and DefendCount is
	// how many of the array is in use.
	//
	// **One card reaches it: Defend** *(2026-08-15)*. It is raised at the end of a turn and stands
	// until the start of its owner's next turn — long enough to cover the opponent's whole turn
	// once, whichever side raised it. See ClearDefenses. It is still a set rather than a flag
	// because two Defends in a round is a legal turn and has to mean something.
	//
	// **Every raised card answers the opponent's one blow, and they compose multiplicatively**
	// *(2026-08-14)*. A turn resolves a single attack, so "which card meets which blow" is a
	// question with no content: a second Defend takes half of what is left after the first has
	// taken half, and the order they were raised in changes nothing. `defendReductionPct` is what
	// each is worth.
	//
	// **A fixed array, not a slice**, because TestRoundIsDeterministic compares two resolved
	// duelists with == and nothing on this struct may stop being comparable. It is a set rather
	// than a queue now; the array is simply how a comparable set of at most five things is held.
	Defends     [maxPendingDefends]PendingDefend
	DefendCount int

	// The two halves of Prepare. GatheredAP is what has been banked *this* round; at the
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

	// The two halves of Plan, and the same shape as Prepare above for the same reason: DrewCards
	// is what has been earned *this* round and BonusDraw is what the round after may draw.
	//
	// **The rules cannot draw a card, and this is how they ask for one** *(2026-08-15)*. There is
	// no deck in this package — the shuffle lives on the combat screen, which is what keeps the
	// engine free of randomness — so a Plan cannot hand its owner two cards. It records that two
	// are owed, and whoever holds the deck honours it when the round is over.
	//
	// **It is a bonus on the refill rather than an immediate draw**, and it has to be. The hand
	// refills to a fixed size at the round boundary, so cards handed over mid-round would simply
	// be two fewer drawn at the boundary and Plan would do nothing at all. Overwritten rather than
	// added to, exactly as BonusAP is: two Plans in one round is a hand of twelve next round, and
	// a Plan every round is a flat +2 rather than a hand that grows forever.
	BonusDraw int
	DrewCards int

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
	// The defences above deliberately stay where they are. Defend is a card effect, not an
	// element status, and filing it in a table indexed by colour would say it was one.
	Statuses [ElementCount]Status
}

// Alive reports whether this duelist can still fight.
func (d Duelist) Alive() bool { return d.CurrentLife > 0 }

// PendingDefend is one raised plan card waiting for the opponent's blow.
//
// **It is just the card.** It carried a charge count until 2026-08-14, when a turn stopped
// resolving more than one attack — counting incoming blows is meaningless when there is only
// ever one.
type PendingDefend struct {
	Card ActionKind
}

// maxPendingDefends bounds the defend set. A turn is capped at MaxActions cards and every one of
// them could be a defence, so this is everything a legal turn can raise.
const maxPendingDefends = baseMaxActions

// defendReductionPct is what one raised Defend takes off the one blow it answers.
//
// **Nothing reduces a blow to zero, and that is a rule rather than a number.** A turn lands one
// figure however many cards went into it, so total negation would be a whole opposing turn deleted
// by a single card — a dominant strategy rather than a decision. Something always lands, so the
// opponent is always still playing. `TestNoDefenceStopsABlowOutright` holds it.
const defendReductionPct = 50

// reductionFor is what one raised card takes off the blow. **Defend is the only card that
// answers a blow at all**, and the switch stays a switch because a second one is a data change
// away rather than a rewrite.
func reductionFor(a ActionKind) int {
	switch a {
	case Defend:
		return defendReductionPct
	default:
		return 0
	}
}

// raiseDefend adds a defend card to the set.
//
// **An overflow is dropped rather than growing the set or panicking.** MaxActions caps a legal
// turn at five actions, so the array holds everything a legal turn can raise; ResolveRound
// deliberately trusts what it is handed so a balance sim can probe outside the rules, and a sim
// that queues six defends should get five of them rather than a crash.
func (d Duelist) raiseDefend(a ActionKind) Duelist {
	if d.DefendCount >= len(d.Defends) {
		return d
	}
	d.Defends[d.DefendCount] = PendingDefend{Card: a}
	d.DefendCount++
	return d
}

// ClearDefenses drops every pending defence a turn put up.
//
// Exported because the combat screen resets a duelist between fights and has to be able to
// clear this without knowing what is in it — a screen that listed the fields by hand is how a
// raised defence once survived into the next duel.
func ClearDefenses(d Duelist) Duelist {
	d.Defends = [maxPendingDefends]PendingDefend{}
	d.DefendCount = 0
	return d
}

// Category is which phase of a turn an action resolves in, and the axis the whole round
// is built on. It is a property of the action, not an independent choice: a fire Lunge
// and a plain Lunge are both attacks.
//
// **There are two of them as of 2026-08-15**, down from prepare/attack/defend. The deck is now
// three attack families and one plan family, and the three-way split was describing a deck that
// no longer exists: Prepare, Plan and Defend all sit in the same phase, and what separates them
// is what they do in it rather than when they happen.
type Category int

const (
	CategoryAttack Category = iota
	CategoryPlan
)

// Categories is every phase in resolution order, and the order a turn is played in.
//
// **Attacks first, plans second.** A plan card is either a bank, which pays next round and so
// does not care when it resolves, or a defence — and a defence has to go up at the *end* of your
// turn, because the opponent acts afterwards and that is the blow it answers. Resolving plans
// first would mean every defence expired before anything could be aimed at it.
//
// It is also the order the combat screen lays a turn out in, which is not a coincidence: the row
// on the table reads left to right in exactly this sequence.
func Categories() []Category {
	return []Category{CategoryAttack, CategoryPlan}
}

func (c Category) String() string {
	switch c {
	case CategoryAttack:
		return "attack"
	case CategoryPlan:
		return "plan"
	default:
		return "?"
	}
}

// ParseCategory resolves a category from its name, which is how combos.json writes a combo's
// scope. It reports failure rather than falling back, for the same reason ParseAction does: a
// combo quietly counting the wrong phase is a balance change nobody made.
func ParseCategory(name string) (Category, bool) {
	for _, c := range Categories() {
		if c.String() == name {
			return c, true
		}
	}
	return CategoryAttack, false
}

// Family is which group of cards an action belongs to: three ways of hitting, plus the plans.
//
// **It is what the card's corner says, and it is not the same axis as Category** *(2026-08-15)*.
// Category is when a card resolves and there are two of those; a family is what kind of card it
// is and there are four. Every family but Plan resolves in the attack phase, so the family is the
// finer distinction and the one worth putting on a card face.
//
// **FamilyNone is the zero value and is a real answer**, not a failure: the opponent's cards
// belong to no family. Families are the player's deck axis — three ways of building a pair — and
// an enemy Attack claiming to be a crush would be saying something untrue about a deck the player
// cannot combo with.
type Family int

const (
	FamilyNone Family = iota
	FamilyStab
	FamilySlash
	FamilyCrush
	FamilyPlan
)

// Families is every real family, in a fixed order, for anything that walks them.
func Families() []Family {
	return []Family{FamilyStab, FamilySlash, FamilyCrush, FamilyPlan}
}

func (f Family) String() string {
	switch f {
	case FamilyStab:
		return "stab"
	case FamilySlash:
		return "slash"
	case FamilyCrush:
		return "crush"
	case FamilyPlan:
		return "plan"
	default:
		return "none"
	}
}

// ParseFamily resolves a family from its name, which is how a deck list writes one. It reports
// failure rather than falling back, for the same reason ParseAction does.
func ParseFamily(name string) (Family, bool) {
	for _, f := range Families() {
		if f.String() == name {
			return f, true
		}
	}
	return FamilyNone, false
}

type ActionKind int

// Declared in family order, and **within a family in ascending cost order**, so anything
// that sorts by the raw value — the deck overlay does — groups the piles the same way a turn
// resolves them and ladders them by price inside that.
//
// **The twelve are three attack families by three tiers, plus three plans** *(2026-08-15)*. Every
// family runs 1 AP for half damage, 2 AP for one, 3 AP for two, so a family is *which* pair you
// are building rather than a better or worse way to build one. See MECHANICS.md.
//
// **Attack and Heavy are the opponent's, and they are deliberately not in a family.** An enemy
// draws from its own short list — see `data/enemy_cards.json` — and giving those cards a family
// would put them in a deck axis the player cannot reach.
//
// **Rewriting this enum shifted every combo ID**, because an expanded hand's ID is derived from
// the card's raw value. That was safe because no profile exists yet, so no discovered combo could
// be re-locked by the shift, and it stops being safe the moment one does. The conflict between
// grouping here and ID stability there is recorded as an open question in MECHANICS.md.
const (
	// Stab.
	Jab ActionKind = iota
	Thrust
	Lunge

	// Slash.
	Cut
	Slash
	Cleave

	// Crush.
	Bash
	Strike
	Smash

	// Plan. The concept named Plan is one card of the family named plan, which is a collision
	// of words rather than of meanings: the family is what the card's corner says and this is
	// the card that draws.
	Prepare
	Plan
	Defend

	// The opponent's, which belong to no family.
	Attack
	Heavy
)

// AllActions is every action either side can queue, in family order.
var AllActions = []ActionKind{
	Jab, Thrust, Lunge,
	Cut, Slash, Cleave,
	Bash, Strike, Smash,
	Prepare, Plan, Defend,
	Attack, Heavy,
}

func (a ActionKind) String() string {
	switch a {
	case Jab:
		return "Jab"
	case Thrust:
		return "Thrust"
	case Lunge:
		return "Lunge"
	case Cut:
		return "Cut"
	case Slash:
		return "Slash"
	case Cleave:
		return "Cleave"
	case Bash:
		return "Bash"
	case Strike:
		return "Strike"
	case Smash:
		return "Smash"
	case Prepare:
		return "Prepare"
	case Plan:
		return "Plan"
	case Defend:
		return "Defend"
	case Attack:
		return "Attack"
	case Heavy:
		return "Heavy"
	default:
		return "Unknown"
	}
}

// Family is which group this action belongs to. The opponent's two belong to none.
func (a ActionKind) Family() Family {
	switch a {
	case Jab, Thrust, Lunge:
		return FamilyStab
	case Cut, Slash, Cleave:
		return FamilySlash
	case Bash, Strike, Smash:
		return FamilyCrush
	case Prepare, Plan, Defend:
		return FamilyPlan
	default:
		return FamilyNone
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
	return Jab, false
}

// Category is which phase this action resolves in.
func (a ActionKind) Category() Category {
	switch a {
	case Prepare, Plan, Defend:
		return CategoryPlan
	default:
		return CategoryAttack
	}
}

// Action point costs. The budget is the decision: a couple of big swings, or a
// fistful of small ones, or a whole round traded away for a defence.
//
// **The three attack tiers cost the same in every family** — 1, 2, 3 — which is what makes a
// family a choice of *which* pair to build rather than a stronger or weaker way to build one.
//
// **The three plans sit on the same 1/2/3 ladder**, so the whole game prices in three tiers and
// nothing costs four. Prepare pays in points, Plan pays in cards and Defend pays in survival, and
// they cost more in that order — three of a four-to-six point budget is most of a round, which
// is what halving a blow is meant to cost.
const (
	costLight = 1 // Jab, Cut, Bash
	costMid   = 2 // Thrust, Slash, Strike
	costHeavy = 3 // Lunge, Cleave, Smash

	costPrepare = 1
	costPlan    = 2
	costDefend  = 3

	// The opponent's two, priced against the player's tiers so an enemy Attack costs what a
	// Strike does and an enemy Heavy costs what a Smash does.
	costEnemyAttack = 2
	costEnemyHeavy  = 3
)

// Cost is what this action takes out of the round's action-point budget.
func (a ActionKind) Cost() int {
	switch a {
	case Jab, Cut, Bash:
		return costLight
	case Lunge, Cleave, Smash:
		return costHeavy
	case Prepare:
		return costPrepare
	case Plan:
		return costPlan
	case Defend:
		return costDefend
	case Heavy:
		return costEnemyHeavy
	case Attack:
		return costEnemyAttack
	default:
		return costMid
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

// planDrawCards is how many extra cards one Plan puts in the following round's hand. Two for two
// points, and **for one round only** — see Duelist.BonusDraw, which is assigned rather than added
// to at the round boundary. What it buys is choice rather than cards: the deck is not thinned and
// nothing is gained permanently, so a Plan is a wider hand once and then a hand of eight again.
const planDrawCards = 2

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
// banked by a Prepare in the round before and less anything an ice hit has taken off.
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

// Damage is what an action deals given the attacker's Strength.
//
// **One ladder, three families** *(2026-08-15)*: half strength at 1 AP, strength at 2, double at
// 3, in Stab and Slash and Crush alike. A family says which pair a card builds toward and nothing
// about how hard it hits, which is what makes the choice between them a *plan* rather than an
// arithmetic comparison.
//
// The opponent's Attack and Heavy sit on the same ladder at the same prices. A plan card deals
// nothing.
func (a ActionKind) Damage(str int) int {
	switch a {
	case Lunge, Cleave, Smash, Heavy:
		return str * 2
	case Thrust, Slash, Strike, Attack:
		return str
	case Jab, Cut, Bash:
		// The floor keeps the cheapest cards from rounding away to nothing at low Strength,
		// which is where a duel starts.
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

	// KindDrew is a Plan banking cards for the following round's hand. Amount is how many.
	//
	// **It is the engine asking rather than doing.** This package has no deck, so the event is
	// the whole of what a Plan produces here — the screen honours it when it refills. It is a
	// separate kind from KindGathered because the two bank different currencies and the feed has
	// to be able to say which.
	KindDrew

	// KindNegated is the blow meeting a raised defence: Action is the card that answered it and
	// Amount is what is left of the blow afterwards.
	//
	// **One kind, one card.** It was three kinds for three cards that differed only in percentage;
	// `Action` already names which card it was, which is the whole distinction the feed was drawing.
	// Only Defend reaches it today, and the kind keeps its general name because what it describes is
	// a blow being reduced rather than one particular card doing it.
	KindNegated

	KindDamage
	KindDefeated

	// KindCombo says a combo formed. **Every one a turn forms is emitted before that turn's
	// first KindAction**, because the combo phase resolves before the cards do — so a boosted
	// hit is never shown before the reason for it. Several can arrive together and their runs
	// may overlap; see matchSlots.
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
	Action ActionKind // set on KindAction, on KindNegated for the defense that stopped it, on KindStaggered for the action lost, on KindMissed for the attack that never landed, and on KindCombo for the card the blow led with
	Amount int        // damage dealt, action points banked, status applied, or on KindCombo what the hand adds up to
	Target Side       // who took the damage
	Life   int        // target's life after the event
	Round  int

	// Element is the card's element on KindAction and KindMissed, and which status is meant on
	// KindStatus and KindBurned. Basic everywhere else, which is also the zero value — an event
	// with nothing to say about colour says `basic`, exactly as a plain card does.
	Element Element

	// Hand and Mix are set on KindCombo and name what the attack phase formed. The screen looks
	// them up with HandByID and MixByID rather than being told their names here, so a hand
	// renamed is renamed once.
	//
	// **Hand is HandNone when no hand formed** and the blow is a lone attack. Mix is still set in
	// that case, because one card is still one colour.
	Hand HandID
	Mix  MixID

	// Multiplier is the turn's damage multiplier in percent — `Hand.Multiplier + Mix.Multiplier`,
	// so 350 is the 3.5x a duo pair earns. It is on the event because the screen has no business
	// re-deriving a number the resolver already worked out.
	Multiplier int

	// Base and Swing are the other two terms of the blow's arithmetic on KindCombo, and they are
	// here for the same reason Multiplier is: the Resolution feed prints the sum — `20 + 10 x 3.5
	// = 55` — and a screen working a damage figure out for itself would be a second resolver.
	//
	// Base is what the hand's own cards carry; Swing is what one Strike deals at this attacker's
	// strength, which is the DMG on their fighter card. `Amount` is what the three come to.
	//
	// **Amount is the blow before the attacker's weight and before anything the defender raised**,
	// so it is what the hand was worth rather than what landed. What landed is the KindDamage
	// after it, and the gap between the two figures is exactly what the defence was worth.
	Base  int
	Swing int

	// ComboCards and ComboCardCount are set on KindCombo alongside Combo: **which cards of this
	// side's turn formed it**, as indices into the turn *as it was played*.
	//
	// **They are here so a screen never has to work out which cards earned a combo.** The
	// matcher already knows, and re-deriving it from the combo's pattern would be a second
	// matcher — the drift ResolutionOrder exists to prevent. It would also be wrong: a counted
	// hand is not contiguous, so Two Pair can be two cards, a card that earned nothing, and two
	// more.
	//
	// **A fixed array rather than a slice, because Event has to stay comparable** —
	// TestCombosDoNotBreakDeterminism compares two logs entry by entry with ==. It is sized to
	// baseMaxActions, which is every card a legal turn can hold; a balance sim deliberately
	// queueing more gets its extra cards dropped from the *bracket* rather than from the combo,
	// the same posture raiseDefend takes on an over-long defend list.
	//
	// The indices count the actions that actually resolved, staggered ones already removed,
	// which is the same sequence as this side's KindAction events — **events that have not
	// happened yet when this one arrives**, since the combo phase runs first. The screen seats
	// the whole turn at DUEL! rather than a card at a time, so the cards are there to bracket.
	ComboCards     [baseMaxActions]int
	ComboCardCount int
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
// **`rng` is the round's randomness and may be nil.** It is the seat CLAUDE.md's determinism
// rules require — an injected source, never a package global — and today the only thing that
// draws from it is a shock roll. A nil source means no roll ever lands, which is what a caller
// with no business being random should pass: a preview, or a test pinning the parts of the
// engine that are still exact.
func ResolveRound(a, b Duelist, aCards, bCards []Card, round int, rng *rand.Rand) (events []Event, aAfter, bAfter Duelist) {
	return resolveRound(a, b, aCards, bCards, round, handTable, mixTable, rng)
}

// resolveRound is ResolveRound with the catalogue injected. It exists so a test can drive a
// synthetic hand through the whole engine rather than only through the matcher.
func resolveRound(a, b Duelist, aCards, bCards []Card, round int, hands []Hand, mixes []Mix, rng *rand.Rand) (events []Event, aAfter, bAfter Duelist) {
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
	events, a, b = playTurn(events, SideA, a, b, appendTurn(nil, SideA, aCards), round, hands, mixes, rng)

	// B still loses its standing defenses even in a round it never gets to act in, which is
	// why this is not inside playTurn's early return: expiry is a property of the turn
	// arriving, not of anything happening in it.
	if a.Alive() && b.Alive() {
		events, b, a = playTurn(events, SideB, b, a, appendTurn(nil, SideB, bCards), round, hands, mixes, rng)
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
// front of it, then **every combo the surviving cards form**, and only then the cards
// themselves.
//
// **Combos are matched against what is left after a stagger, not against the queue.** The
// player queued five attacks; a stagger that ate two means three happened, and a combo
// scored off cards a stagger deleted would let a staggered duelist stagger back with a turn
// they did not take. That ordering is the reason the combo phase sits *inside* a turn rather
// than at the top of the round: a round-wide combo phase would score B's combos before A's
// stagger had taken anything off B.
func playTurn(
	events []Event,
	side Side,
	actor, target Duelist,
	turn []Slot,
	round int,
	hands []Hand,
	mixes []Mix,
	rng *rand.Rand,
) ([]Event, Duelist, Duelist) {
	actor = expireDefenses(actor)

	// Stagger comes off the front, which needs no tie-break and so is the only pick that is
	// deterministic without inventing a rule. Under phases the front of a turn is its prepares,
	// so being staggered costs a Prepare before it costs an attack — a real consequence, and
	// the one that makes stagger worth planning around rather than merely suffering.
	//
	// **The action points are not refunded.** They were committed when the cards were queued,
	// and letting them come back would make stagger pure tempo; keeping them spent makes it
	// tempo and economy both, which is the price a combo costing a whole round's budget to
	// set up should command.
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

	// **The attack phase is one blow, whatever it was made of.** Every attack card queued is
	// announced, then the hand they form is announced, then a single figure of damage lands. Five
	// Strikes are not five hits; they are one Strike Barrage.
	events, actor, target = resolveAttackPhase(events, side, actor, target, turn, round, hands, mixes, rng)

	// **The plan phase comes second, and that is what a defence needs** *(2026-08-15)*. A Defend
	// answers the *opponent's* blow, and the opponent acts after this turn ends — so a defence
	// raised at the end of a turn is the only one that is standing when anything is aimed at it.
	// Prepare and Plan both bank for the round after and do not care where they sit.
	//
	// **It is skipped if either side fell**, since a corpse raising a shield is a line in the log
	// nobody wants and a duel that is over does not need one.
	if !actor.Alive() || !target.Alive() {
		return events, actor, target
	}
	for _, slot := range turn {
		if slot.Card.Category() != CategoryPlan {
			continue
		}
		events, actor, target = resolvePlan(events, side, actor, target, slot.Card, round)
	}

	return events, actor, target
}

// resolvePlan runs one plan card. **The plans are the only cards that still resolve one at a
// time**, because each does something to its own duelist rather than contributing to a shared
// blow.
func resolvePlan(
	events []Event,
	side Side,
	actor, target Duelist,
	card Card,
	round int,
) ([]Event, Duelist, Duelist) {
	events = append(events, Event{
		Kind:    KindAction,
		Side:    side,
		Action:  card.Action,
		Element: card.Element,
		Round:   round,
	})

	switch card.Action {
	case Prepare:
		actor.GatheredAP += prepareBonusAP
		events = append(events, Event{
			Kind:   KindGathered,
			Side:   side,
			Amount: prepareBonusAP,
			Round:  round,
		})

	case Plan:
		// **Recorded, not drawn.** There is no deck in this package; what is banked here is honoured
		// by whoever holds one when the round is over. See Duelist.BonusDraw.
		actor.DrewCards += planDrawCards
		events = append(events, Event{
			Kind:   KindDrew,
			Side:   side,
			Amount: planDrawCards,
			Round:  round,
		})

	case Defend:
		// Raised, not spent. What it is worth is `reductionFor`, and it is read when the opponent's
		// blow arrives — see resolveAttackPhase.
		actor = actor.raiseDefend(card.Action)
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
//
// The clearing itself is ClearDefenses, which the combat screen also calls between fights; the
// timing rule is what lives here.
func expireDefenses(d Duelist) Duelist { return ClearDefenses(d) }

// endRound rolls what was banked this round into next round's budget, ticks a burn, and counts
// every status down one. Assignment rather than addition for the bank: two Prepares in one round
// are worth +4 next round, and preparing every round is worth a flat +2 rather than compounding
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

	// Same shape and the same reason: assignment, so a Plan widens exactly one hand.
	d.BonusDraw = d.DrewCards
	d.DrewCards = 0
	return events, d
}

// resolveAttackPhase is the whole of one side's offence: every attack card it queued, the hand
// they form, and the single blow that follows.
//
// **One blow per turn** *(2026-08-14)*. Attack cards no longer resolve one at a time; they are
// announced, and then `BlowFor` reads them as a set and says what they amount to. Cards that
// contribute to no hand are announced and then ignored — `Strike, Jab, Strike` is a Strike Pair
// and the Jab is not in it, so it adds nothing to the figure.
//
// The order inside the blow is: shock roll, base damage from the hand's own cards, the hand and
// mix multiplier, the attacker's earth weight, then the defender's raised cards. **Weight sits
// where it does because it is a property of the attacker** — it says how hard they can still
// swing — so everything the defender does happens to a blow that has already been blunted.
func resolveAttackPhase(
	events []Event,
	side Side,
	actor, target Duelist,
	turn []Slot,
	round int,
	hands []Hand,
	mixes []Mix,
	rng *rand.Rand,
) ([]Event, Duelist, Duelist) {
	targetSide := other(side)

	// Every attack card is announced whether or not it ends up in the hand. **A slot that
	// resolved has to produce a beat**, because the screen counts one per slot to know how far
	// through the round playback is — see TestEverySlotIsEitherTakenOrStaggered.
	attacks := 0
	for _, slot := range turn {
		if slot.Card.Category() != CategoryAttack {
			continue
		}
		attacks++
		events = append(events, Event{
			Kind:    KindAction,
			Side:    side,
			Action:  slot.Card.Action,
			Element: slot.Card.Element,
			Round:   round,
		})
	}
	if attacks == 0 {
		return events, actor, target
	}

	blow := blowFor(turn, hands, mixes)
	if len(blow.Cards) == 0 {
		return events, actor, target
	}

	// The hand is announced before the blow lands, so a boosted figure never arrives before the
	// reason for it. A lone attack that formed nothing carries HandNone and says only its colour.
	//
	// **It also carries the sum**, which is what the damage below is taken from — see comboEvent.
	swung := comboEvent(side, blow, turn, actor.Str, round)
	events = append(events, swung)

	// **What the hand buys besides damage is paid on forming it, not on connecting.** A shock
	// that makes the blow miss does not undo a stagger the player assembled five cards to earn —
	// the hand is scored off the queue, and the queue was committed when DUEL! was pressed.
	if blow.Hand.Effect.BankAP != 0 {
		actor.GatheredAP += blow.Hand.Effect.BankAP
		events = append(events, Event{
			Kind:   KindGathered,
			Side:   side,
			Amount: blow.Hand.Effect.BankAP,
			Round:  round,
		})
	}
	if blow.Hand.Effect.Stagger != 0 {
		target.Staggered = addStagger(target.Staggered, blow.Hand.Effect.Stagger)
	}

	// A shocked attacker may miss outright, and misses before anything else happens — no defence
	// spent, no status applied. The attack did not occur.
	//
	// **This is a roll**, and the only one in the package. See shockMissPct.
	if shocked, missed := spendShock(actor, rng); missed {
		actor = shocked
		events = append(events, Event{
			Kind:    KindMissed,
			Side:    side,
			Action:  turn[blow.Cards[0]].Card.Action,
			Element: turn[blow.Cards[0]].Card.Element,
			Target:  targetSide,
			Round:   round,
		})
		return events, actor, target
	} else {
		actor = shocked
	}

	// **Base damage is the cards in the hand, and the multiplier is DMG on top.** DMG is what one
	// Strike deals at this duelist's strength, which is the figure the duelist card shows — so
	// `20 + 10 x 1.5 = 35` for a pair of Strikes at Str 10, exactly as the design states it.
	//
	// That sum is the announcement's `Amount`, taken rather than repeated: the feed prints the
	// arithmetic, and a second copy of it here is the one way the printed sum could be wrong.
	dmg := blunt(swung.Amount, actor.weightPct())

	// **Every raised defence answers the blow, and they compose multiplicatively.** Order is not
	// read: a turn has one attack, so "which card meets which blow" has no content. Multiplying
	// what is left rather than adding the percentages is what stops two cards reaching zero by
	// accident while keeping each one worth something — two Defends take three quarters rather
	// than the whole thing, and a third takes seven eighths, which is a curve that never arrives.
	for i := 0; i < target.DefendCount; i++ {
		card := target.Defends[i].Card
		pct := reductionFor(card)
		if pct <= 0 {
			continue
		}
		dmg = dmg * (100 - pct) / 100

		events = append(events, Event{
			Kind:   KindNegated,
			Side:   targetSide,
			Action: card,
			Target: side,
			Amount: dmg,
			Round:  round,
		})
	}

	// Every defence is spent on the one blow it answered.
	target.Defends = [maxPendingDefends]PendingDefend{}
	target.DefendCount = 0

	target.CurrentLife = reduce(target.CurrentLife, dmg)
	events = append(events, Event{
		Kind:   KindDamage,
		Side:   side,
		Target: targetSide,
		Amount: dmg,
		Life:   target.CurrentLife,
		Round:  round,
	})

	// **The mix lands its colours' statuses, and it does so because the hand was formed rather
	// than because the blow hurt.** A hand halved by a Defend still connected, and making the
	// status conditional on the final figure would mean a defensive card silently un-applied an
	// element the attacker had already paid for.
	//
	// One status per distinct colour, so mono lands one and rainbow lands four. Drab lands none,
	// which is what "basic is not a colour" means at the other end.
	for _, e := range blow.Elements {
		applied, amount, ok := applyStatus(target, e)
		if !ok {
			continue
		}
		target = applied
		events = append(events, Event{
			Kind:    KindStatus,
			Side:    side,
			Target:  targetSide,
			Element: e,
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

// comboEvent packages what the attack phase formed for the screen: which hand, which mix, the
// multiplier, which cards of the turn earned it, and the arithmetic they come to.
//
// **It is the attack phase's one line in the feed** *(2026-08-14)*, so it carries everything that
// line has to say. The individual attack cards are still announced — a slot that resolved has to
// produce a beat — but the screen draws no sentence for them: five cards making one blow read as
// five blows, which is the thing one-blow-per-turn was meant to stop saying.
func comboEvent(side Side, blow Blow, turn []Slot, str, round int) Event {
	// **The blow is added up here and nowhere else.** The attack phase takes its damage figure off
	// this event rather than recomputing it, so the sentence the feed prints and the damage that
	// lands cannot be two different sums.
	base := 0
	for _, i := range blow.Cards {
		base += turn[i].Card.Damage(str)
	}
	swing := Strike.Damage(str)

	lead := turn[blow.Cards[0]].Card

	e := Event{
		Kind:       KindCombo,
		Side:       side,
		Action:     lead.Action,
		Element:    lead.Element,
		Amount:     base + scaleDamage(swing, blow.Multiplier),
		Hand:       blow.Hand.ID,
		Mix:        blow.Mix.ID,
		Multiplier: blow.Multiplier,
		Base:       base,
		Swing:      swing,
		Round:      round,
	}
	for _, i := range blow.Cards {
		if e.ComboCardCount >= len(e.ComboCards) {
			break
		}
		e.ComboCards[e.ComboCardCount] = i
		e.ComboCardCount++
	}
	return e
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

// PlanStyle is how an opponent fights. It exists because a defence is priced against how many
// attacks arrive, and for as long as every enemy spent its whole budget on two big swings, two
// defensive cards bought total immunity — a duel that ran three rounds taking 0, 0 and 2 damage.
// That was not a fault in what a defence cost. It is one opponent wearing one shape, and the
// shape is what the player is really buying answers to.
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
	// useful baseline. A Defend is at its best against it, which is the point of it not being
	// alone: one big swing is exactly the blow halving is worth most against.
	StyleBrute PlanStyle = iota

	// StyleSwarm attacks as many times as the round allows. It used to be the answer to the
	// immunity problem, back when a defence stopped one blow of several. **One blow per turn has
	// taken that away**: a swarm's five cheap attacks arrive as one figure like everything else,
	// so what the style now buys is a *hand* — five cards is a combo and two is not.
	StyleSwarm

	// StyleWarden opens with a Defend and attacks with the rest, so the player's damage is
	// halved until they can punch through it.
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

// planWarden puts a Defend up first and attacks with what is left. Defend is a plan, so
// resolution moves it to the *end* of the turn regardless of where it sits here — it is
// first in the slice only because that is how the plan reads.
//
// **No Defend in hand and it fights like a brute**, which is the deck showing through the
// style: a warden is a duelist that shields when it can, not one that always has a shield.
//
// **Today it is always a brute**, because `data/enemy_cards.json` deals nothing but Attack and
// Heavy — an opponent has no plan cards at all as of 2026-08-15. This is written against the deck
// it will have rather than the deck it has, exactly as it was before the rework.
func planWarden(p *pool, budget, slots int) []Card {
	if budget < costDefend || slots < 1 {
		return planBrute(p, budget, slots)
	}
	shield, ok := p.take(Defend)
	if !ok {
		return planBrute(p, budget, slots)
	}
	return append([]Card{shield}, planBrute(p, budget-costDefend, slots-1)...)
}

// planTactician alternates between banking and spending, reading which round it is in off
// its own BonusAP: anything banked means last round was the setup, so this one is the
// payoff. It needs no memory of its own because Prepare already leaves the evidence.
//
// **Also always a brute today**, and for the same reason as the warden: no enemy deck holds a
// Prepare.
func planTactician(d Duelist, p *pool, budget, slots int) []Card {
	if d.BonusAP > 0 {
		// The payoff round. Everything into the biggest attacks that fit.
		return planBrute(p, budget, slots)
	}

	// The setup round: bank what a second Prepare is worth, keep enough back to stay
	// dangerous, and never spend so much banking that the round is a free hit for the
	// player. Two Prepares is the whole point — it is what makes the next round oversized,
	// and a hand holding one is a setup round that only half works.
	plan := make([]Card, 0, slots)
	for len(plan) < slots-1 && budget >= costPrepare*2 && len(plan) < 2 {
		bank, ok := p.take(Prepare)
		if !ok {
			break
		}
		plan, budget = append(plan, bank), budget-costPrepare
	}
	return append(plan, planBrute(p, budget, slots-len(plan))...)
}
