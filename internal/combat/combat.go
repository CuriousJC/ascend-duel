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
//
// **Three stats, and every one of them is the number it sounds like** *(2026-08-16)*. Constitution
// and Speed went with the same argument that took Strength the day before: each existed only to be
// converted into something else — `Con * 5` was life and `4 + Spd/10` was the action-point budget
// — so the player had to learn a number they could never act on directly. Speed was the clearer
// case of the two: twenty-four distinct values across the roster produced three distinct budgets.
type Duelist struct {
	// DMG is what a 1x attack deals in this duelist's hands, and it is the figure on the fighter
	// card. The ladder scales off it: a card declares its multiplier and the arithmetic is
	// `DMG * Amount / 100`.
	DMG int

	// Actions is this duelist's action-point budget before anything banked. It is what a round is
	// spent out of, and cards cost 1 to 3 of it.
	Actions int

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
	// taken half, and the order they were raised in changes nothing. `reductionFor` is what each is
	// worth, and it reads the card's own declared Amount.
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

	// Rings is which elements this duelist wears a ring for, indexed the same way Statuses is.
	//
	// **It is what makes an element do anything at all** *(2026-08-16)*. A fire attack from a
	// duelist with no fire ring is a plain attack with a red border: it counts toward a mix, it
	// is discounted by nothing, and it applies no burn. See status.go for the argument, which is
	// that statuses given away free left the first three rings with no mechanic of their own.
	//
	// **The ring is read off the attacker, never the victim.** Your fire ring makes *your* fire
	// attacks burn; it does nothing when a fire attack is aimed at you.
	//
	// **An array of bools rather than a set**, because Duelist has to stay comparable — see the
	// note at the top of this struct — and because Element is already the index of the status it
	// switches on. It is the seat the ring discount sits in as well: `Card.Cost()` has the other
	// half of that and neither is wired to the other yet.
	//
	// **Enemies never wear one.** The zero value is what an enemy is hydrated with and nothing
	// sets it for them, so an enemy's elements are inert by construction rather than by a rule
	// written down somewhere else. Statuses reaching the player by some other route later is
	// expected; it will not be by an enemy putting on jewellery.
	Rings [ElementCount]bool

	// SoloAttacks makes this duelist's attack cards resolve **one at a time, in the order they
	// were queued**, each landing its own blow — instead of being read as a set and scored
	// through the combo table.
	//
	// **It is what an enemy is** *(2026-08-17, owner's call)*. Combos are the player's mechanic:
	// the hands are counted off concepts and the mixes off colours, and an enemy has neither axis
	// to play with — every enemy card in `data/enemies.json` is authored `basic` and
	// `FamilyNone`, so an opponent's "hand" was whatever its planner happened to afford. Now an
	// enemy holding three cards swings three times and the player can read the round off the
	// cards on the table.
	//
	// **The default is false, so a plain `Duelist{}` combos.** Combos are the norm and this is the
	// exception, which is why the field is named for the exception rather than for the norm: a
	// `Combos bool` would have made every existing literal — the whole test suite, the balance
	// tool's fighter — quietly stop comboing.
	//
	// **It is a flag on the duelist rather than a rule about SideB.** The engine has no idea which
	// side is a person, and it must not learn: the balance tool plays both sides headlessly, and
	// a rule keyed on the side would be a rule that cannot be tested from the other end.
	//
	// The alternative considered and rejected was deriving it from the cards — an enemy card is
	// `FamilyNone`, so "no family, no combo" needs no field at all. It was rejected because it
	// couples two things that are not the same thing: affixes are designed to *transform* an
	// enemy deck, and a card that gained a family would silently gain combos with it.
	SoloAttacks bool
}

// WearsRing reports whether this duelist's ring makes the named element do something. Basic is
// never worn — it is the absence of an element, so there is nothing for a ring to point at.
func (d Duelist) WearsRing(e Element) bool {
	if e <= Basic || int(e) >= ElementCount {
		return false
	}
	return d.Rings[e]
}

// Alive reports whether this duelist can still fight.
func (d Duelist) Alive() bool { return d.CurrentLife > 0 }

// PendingDefend is one raised plan card waiting for the opponent's blow.
//
// **It is just the card.** It carried a charge count until 2026-08-14, when a turn stopped
// resolving more than one attack — counting incoming blows is meaningless when there is only
// ever one.
type PendingDefend struct {
	// Card is the whole card rather than its concept, **because what a defence is worth is a
	// property of the card** *(2026-08-17)*: a worm can scale one Defend without touching the
	// others. Storing the ID lost that the moment it was raised.
	Card Card
}

// maxPendingDefends bounds the defend set. A turn is capped at MaxActions cards and every one of
// them could be a defence, so this is everything a legal turn can raise.
const maxPendingDefends = baseMaxActions

// reductionFor is what one raised card takes off the blow: its own declared Amount, as a
// percentage.
//
// **Nothing reduces a blow to zero, and that is a rule rather than a number.** A turn lands one
// figure however many cards went into it, so total negation would be a whole opposing turn deleted
// by a single card — a dominant strategy rather than a decision. Something always lands, so the
// opponent is always still playing. `RegisterConcept` refuses a card declaring 100 or more, and
// `TestNoDefenceStopsABlowOutright` holds the resolver to it.
func reductionFor(card Card) int {
	if card.Spec().Verb != VerbDefend {
		return 0
	}
	return card.Amount()
}

// raiseDefend adds a defend card to the set.
//
// **An overflow is dropped rather than growing the set or panicking.** MaxActions caps a legal
// turn at five actions, so the array holds everything a legal turn can raise; ResolveRound
// deliberately trusts what it is handed so a balance sim can probe outside the rules, and a sim
// that queues six defends should get five of them rather than a crash.
func (d Duelist) raiseDefend(card Card) Duelist {
	if d.DefendCount >= len(d.Defends) {
		return d
	}
	d.Defends[d.DefendCount] = PendingDefend{Card: card}
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
//
// **It is the stat plus the bank, and nothing else** *(2026-08-16)*. It used to be
// `4 + Spd/10 + BonusAP`, a conversion whose only observable effect was to flatten
// twenty-four distinct Speed values into three budgets. Now the file says what the budget is.
//
// **No status touches it.** A chill did until 2026-08-16, and it is now a card off the front of
// the turn instead — see playTurn. What that costs is the one thing the old version had going for
// it: an AP cut was felt while the player was still choosing, and a card taken off a committed
// turn is felt after they have. What it buys is a status a player can name.
func (d Duelist) ActionPoints() int {
	return d.Actions + d.BonusAP
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
	Side   Side      // who acted
	Action ConceptID // set on KindAction, on KindNegated for the defense that stopped it, on KindStaggered for the action lost, on KindMissed for the attack that never landed, and on KindCombo for the card the blow led with
	Amount int       // damage dealt, action points banked, status applied, or on KindCombo what the hand adds up to
	Target Side      // who took the damage
	Life   int       // target's life after the event
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
// **It holds a whole Card rather than a bare concept** since 2026-08-12. A slot is what both
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
	// deterministic without inventing a rule. **The front of a turn is its attacks** — the phase
	// order puts them before the plans — so what a stagger costs first is the blow, which is what
	// makes it worth planning around rather than merely suffering.
	//
	// **The action points are not refunded.** They were committed when the cards were queued,
	// and letting them come back would make stagger pure tempo; keeping them spent makes it
	// tempo and economy both, which is the price a combo costing a whole round's budget to
	// set up should command.
	//
	// **A chill is added to the stagger rather than counted separately** *(2026-08-16)*. Ice takes
	// a card off the front of a turn, which is precisely what a stagger is, so the two add and one
	// loop announces both. The difference between them is where they come from and how long they
	// last: a stagger is spent when it bites, a chill bites on every turn it outlives.
	//
	// The consequence, stated: a card lost to ice is announced as `KindStaggered`, so the feed
	// says "staggered" for something the ring calls a chill. One event kind is what keeps the
	// playback's one-beat-per-slot invariant true without a second thing to remember — see
	// `currentSlot` on the screen, which counts these events to find the lit row.
	lost := addStagger(actor.Staggered, actor.chillCards())
	actor.Staggered = 0
	if lost == StaggerAll || lost > len(turn) {
		lost = len(turn)
	}
	for i := 0; i < lost; i++ {
		events = append(events, Event{
			Kind:    KindStaggered,
			Side:    side,
			Action:  turn[i].Card.Concept,
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
		Action:  card.Concept,
		Element: card.Element,
		Round:   round,
	})

	// **The switch is on the verb, not on the card** *(2026-08-16)*. It used to name Prepare, Plan
	// and Defend one at a time, which meant an enemy's `Congeal` could not shield anything however
	// obviously it was a defence. Three verbs, any number of cards.
	spec := card.Spec()
	switch spec.Verb {
	case VerbBank:
		actor.GatheredAP += card.Amount()
		events = append(events, Event{
			Kind:   KindGathered,
			Side:   side,
			Amount: card.Amount(),
			Round:  round,
		})

	case VerbDraw:
		// **Recorded, not drawn.** There is no deck in this package; what is banked here is honoured
		// by whoever holds one when the round is over. See Duelist.BonusDraw.
		actor.DrewCards += card.Amount()
		events = append(events, Event{
			Kind:   KindDrew,
			Side:   side,
			Amount: card.Amount(),
			Round:  round,
		})

	case VerbDefend:
		// Raised, not spent. What it is worth is `reductionFor`, and it is read when the opponent's
		// blow arrives — see resolveAttackPhase.
		actor = actor.raiseDefend(card)
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

	// **A solo attacker takes a different phase entirely, not a special case inside this one.**
	// The two are different shapes: one announces everything and then lands a single figure, the
	// other resolves each card completely before the next one starts. Threading a flag through
	// the blow, the multiplier and the combo event would leave a function whose every step had
	// two readings.
	if actor.SoloAttacks {
		return resolveSoloAttacks(events, side, actor, target, turn, round, rng)
	}

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
			Action:  slot.Card.Concept,
			Element: slot.Card.Element,
			Round:   round,
		})
	}
	if attacks == 0 {
		return events, actor, target
	}

	// **Recoil is paid before the blow, and it is not part of it.** An attack aimed at self costs
	// its owner life and contributes nothing to the hand — `formsBlow` is what keeps it out of the
	// matcher, so a Blood Rite beside two Strikes is a Strike Pair rather than a Strike Flurry.
	//
	// It resolves first because it is the wind-up rather than the follow-through: a duelist who
	// tears themselves open to swing has already paid when the swing lands, and a recoil that
	// killed its owner after their blow connected would read as a corpse having hit something.
	for _, slot := range turn {
		if !slot.Card.Spec().Recoils() {
			continue
		}
		actor.CurrentLife = reduce(actor.CurrentLife, slot.Card.Damage(actor.DMG))
		events = append(events, Event{
			Kind:    KindDamage,
			Side:    side,
			Target:  side,
			Element: slot.Card.Element,
			Amount:  slot.Card.Damage(actor.DMG),
			Life:    actor.CurrentLife,
			Round:   round,
		})
	}
	if !actor.Alive() {
		events = append(events, Event{
			Kind:   KindDefeated,
			Side:   side,
			Target: side,
			Round:  round,
		})
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
	swung := comboEvent(side, blow, turn, actor.DMG, round)
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
	// **This is a roll**, and the only one in the package. See shockMissPct. Nothing is consumed
	// by it: a shock rolls on every attack it outlives, so the duelist comes back unchanged.
	if shockMisses(actor, rng) {
		events = append(events, Event{
			Kind:    KindMissed,
			Side:    side,
			Action:  turn[blow.Cards[0]].Card.Concept,
			Element: turn[blow.Cards[0]].Card.Element,
			Target:  targetSide,
			Round:   round,
		})
		return events, actor, target
	}

	// **Base damage is the cards in the hand, and the multiplier is DMG on top.** DMG is what one
	// Strike deals at this duelist's strength, which is the figure the duelist card shows — so
	// `20 + 10 x 1.5 = 35` for a pair of Strikes at Str 10, exactly as the design states it.
	//
	// That sum is the announcement's `Amount`, taken rather than repeated: the feed prints the
	// arithmetic, and a second copy of it here is the one way the printed sum could be wrong.
	dmg := blunt(swung.Amount, actor.weight())

	events, dmg = applyDefends(events, side, target, dmg, round)

	// Every defence is spent on the turn it answered.
	target = ClearDefenses(target)

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
	//
	// **And every one of them is gated on a ring the attacker is wearing** *(2026-08-16)*. A
	// rainbow thrown by a duelist wearing two rings lands two statuses; thrown by an enemy it
	// lands none. The colours still count toward the mix multiplier either way — what the ring
	// buys is the status, not the combo.
	for _, e := range blow.Elements {
		if !actor.WearsRing(e) {
			continue
		}
		applied, amount, ok := applyStatus(target, e, actor)
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

// applyDefends runs every card the target has raised over one incoming blow and reports what is
// left of it, announcing each as it bites.
//
// **It does not spend them, and the caller clears them once the turn is over.** A defence covers
// exactly one opposing *turn* — see expireDefenses — which is one blow from a comboing duelist and
// several from a solo one. Spending them on the first blow would make a Defend nearly worthless
// against the very opponents that swing more than once.
//
// **They compose multiplicatively and the order is not read.** Multiplying what is left rather
// than adding the percentages is what stops two cards reaching zero by accident while keeping each
// one worth something: two Defends take three quarters rather than the whole thing, and a third
// takes seven eighths, which is a curve that never arrives.
func applyDefends(events []Event, side Side, target Duelist, dmg, round int) ([]Event, int) {
	for i := 0; i < target.DefendCount; i++ {
		card := target.Defends[i].Card
		pct := reductionFor(card)
		if pct <= 0 {
			continue
		}
		dmg = dmg * (100 - pct) / 100

		events = append(events, Event{
			Kind:   KindNegated,
			Side:   other(side),
			Action: card.Concept,
			Target: side,
			Amount: dmg,
			Round:  round,
		})
	}
	return events, dmg
}

// resolveSoloAttacks is the attack phase of a duelist whose cards do not combo: **every attack
// resolves completely, in queue order, before the next one starts**.
//
// **No hand is read and no combo event is emitted** *(2026-08-17)*. That is the whole of the
// difference — there is no set to score, so there is no multiplier, no mix, no banked AP and no
// stagger. What lands is the sum of what was played, one figure at a time, and the screen writes a
// sentence per card because there is no phase line to carry them.
//
// Three things it keeps deliberately in step with the comboing phase, because they are rules about
// attacking rather than rules about combos:
//
//   - **One beat per slot.** Every attack card announces itself with a KindAction, so playback can
//     still count how far through the round it is — see TestEverySlotIsEitherTakenOrStaggered.
//   - **One shock roll for the turn, not one per card.** A shock is "the turn's attack misses", and
//     rolling per card would both change what the status means and advance the one random stream in
//     the package a different number of times per round. A shocked solo attacker misses with
//     everything and says so on each card.
//   - **Weight, then defences, then statuses**, in that order, for the reason the other phase gives:
//     weight is a property of the attacker, so everything the defender does happens to a blow that
//     has already been blunted.
func resolveSoloAttacks(
	events []Event,
	side Side,
	actor, target Duelist,
	turn []Slot,
	round int,
	rng *rand.Rand,
) ([]Event, Duelist, Duelist) {
	targetSide := other(side)

	attacked := false
	missed := false
	rolled := false

	for _, slot := range turn {
		if slot.Card.Category() != CategoryAttack {
			continue
		}
		attacked = true

		events = append(events, Event{
			Kind:    KindAction,
			Side:    side,
			Action:  slot.Card.Concept,
			Element: slot.Card.Element,
			Round:   round,
		})

		// **Recoil is the card hurting its own owner**, and it is not a blow: it lands whatever the
		// shock did, because a duelist tearing themselves open has already paid by the time the
		// swing would have missed.
		if slot.Card.Spec().Recoils() {
			actor.CurrentLife = reduce(actor.CurrentLife, slot.Card.Damage(actor.DMG))
			events = append(events, Event{
				Kind:    KindDamage,
				Side:    side,
				Target:  side,
				Element: slot.Card.Element,
				Amount:  slot.Card.Damage(actor.DMG),
				Life:    actor.CurrentLife,
				Round:   round,
			})
			if !actor.Alive() {
				events = append(events, Event{Kind: KindDefeated, Side: side, Target: side, Round: round})
				return events, actor, target
			}
			continue
		}

		// The roll happens on the first card that could actually swing, and once only. Rolling
		// before the loop would advance the stream for a turn of nothing but recoil.
		if !rolled {
			missed, rolled = shockMisses(actor, rng), true
		}
		if missed {
			events = append(events, Event{
				Kind:    KindMissed,
				Side:    side,
				Action:  slot.Card.Concept,
				Element: slot.Card.Element,
				Target:  targetSide,
				Round:   round,
			})
			continue
		}

		dmg := blunt(slot.Card.Damage(actor.DMG), actor.weight())
		events, dmg = applyDefends(events, side, target, dmg, round)

		target.CurrentLife = reduce(target.CurrentLife, dmg)
		events = append(events, Event{
			Kind:   KindDamage,
			Side:   side,
			Target: targetSide,
			Amount: dmg,
			Life:   target.CurrentLife,
			Round:  round,
		})

		// One card, one colour, and the same ring gate the other phase applies. An enemy wears no
		// rings, so this does nothing for the only duelists that are solo attackers today — it is
		// here because the rule belongs to attacking, not to comboing.
		if actor.WearsRing(slot.Card.Element) {
			if applied, amount, ok := applyStatus(target, slot.Card.Element, actor); ok {
				target = applied
				events = append(events, Event{
					Kind:    KindStatus,
					Side:    side,
					Target:  targetSide,
					Element: slot.Card.Element,
					Amount:  amount,
					Life:    target.CurrentLife,
					Round:   round,
				})
			}
		}

		if !target.Alive() {
			events = append(events, Event{Kind: KindDefeated, Side: side, Target: targetSide, Round: round})
			return events, actor, target
		}
	}

	// **The defences are spent only if something was swung at them**, which is the comboing
	// phase's rule too: a turn with no attacks in it returns before clearing, and expireDefenses
	// takes them at the start of their owner's next turn instead.
	if attacked {
		target = ClearDefenses(target)
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
func comboEvent(side Side, blow Blow, turn []Slot, dmg, round int) Event {
	// **The blow is added up here and nowhere else.** The attack phase takes its damage figure off
	// this event rather than recomputing it, so the sentence the feed prints and the damage that
	// lands cannot be two different sums.
	base := 0
	for _, i := range blow.Cards {
		base += turn[i].Card.Damage(dmg)
	}
	swing := referenceSwing(dmg)

	lead := turn[blow.Cards[0]].Card

	e := Event{
		Kind:       KindCombo,
		Side:       side,
		Action:     lead.Concept,
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

// referenceSwing is the blow a hand's multiplier is applied to: one 1x attack at this duelist's
// DMG, which is exactly their DMG.
//
// **It was `Strike.Damage(dmg)` until 2026-08-16**, and it named a particular card because the
// ladder was a switch statement with Strike sitting on its middle rung. A card declares its own
// multiplier now, so 1x is the definition rather than one card's entry — and the multiplier a hand
// pays should not move because the player's Strike was retuned.
func referenceSwing(dmg int) int { return dmg }

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

// PlanFor builds one round's plan out of the hand an opponent was dealt.
//
// **One planner as of 2026-08-16, and the deck is what makes an enemy itself.** There were four
// styles — brute, swarm, warden, tactician — chosen by a string on the enemy record, and they went
// with the shared enemy deck. An opponent that holds six cheap copies of one card *is* a swarm; one
// holding three expensive ones *is* a brute. The personality moved into the thing the player can
// actually read, which is what the cards do rather than which branch a switch took.
//
// Two things went wrong with styles that this cannot repeat. Three of the four were unreachable —
// the warden asked for a Defend by name and the shared list held none, so every enemy in the game
// fought as a brute. And none of them was rewritten when a turn stopped resolving several blows, so
// the shape the roster treated as weakest (a swarm's four cheap attacks, now a Barrage at 5x) had
// quietly become the strongest.
//
// **The rule is: hit as hard as this hand can, then spend what is left over.** The attack half is
// exact rather than greedy — it scores every affordable combination through `blowFor`, the same
// function that resolves the round, so the plan the opponent plays is the plan the engine will
// score. A greedy "take the dearest" pass cannot see that three Ooze beat one Dissolve.
//
// **Bounded by the budget and by MaxActions**, both, and it may never return a card it was not
// dealt — TestThePlannerObeysBothBounds holds it.
//
// **The shuffle stays outside this package.** What arrives is a hand, already dealt, exactly as the
// player's hand reaches the screen: `internal/combat` keeps no randomness and no clock, which is
// what TestRoundIsDeterministic pins and what lets the balance tool run whole duels headlessly.
//
// **The planner reasons about concepts and carries elements along.** Every choice is made on cost
// and damage; the element rides on the card it was dealt on and reaches the round untouched. An
// enemy's colours do nothing until an affix attunes them, so preferring one would be preferring a
// border.
func PlanFor(d Duelist, hand []Card) []Card {
	return planFor(d, hand, handTable, mixTable)
}

// planFor is PlanFor with the catalogue injected, so a test can drive a planner against a
// synthetic ladder.
func planFor(d Duelist, hand []Card, hands []Hand, mixes []Mix) []Card {
	budget, slots := d.ActionPoints(), d.MaxActions()

	chosen, spent := bestAttacks(d, hand, budget, slots, hands, mixes)
	return append(chosen, spareCards(hand, chosen, budget-spent, slots-len(chosen))...)
}

// bestAttacks is the hardest-hitting affordable combination of attacks in the hand, and what it
// cost. It returns the cards in the order they were dealt.
//
// **Exhaustive over subsets, which is affordable because a hand is small.** EnemyHandSize is 7, so
// this is at most 128 candidates, each scored by the real `blowFor`. Above `maxSearchableAttacks`
// it falls back to taking the biggest cards that fit — a balance sim deliberately handing an
// opponent twenty attacks should get a plan rather than a hung process.
func bestAttacks(d Duelist, hand []Card, budget, slots int, hands []Hand, mixes []Mix) ([]Card, int) {
	var offence []int
	for i, c := range hand {
		s := c.Spec()
		if s.Verb == VerbAttack && s.Target == TargetOpponent {
			offence = append(offence, i)
		}
	}
	if len(offence) == 0 {
		return nil, 0
	}
	if len(offence) > maxSearchableAttacks {
		return greedyAttacks(hand, offence, budget, slots)
	}

	bestScore, bestCost := -1, 0
	var best []int

	// Ascending masks, and a strictly-better test, so a tie goes to the combination whose cards
	// were dealt earliest. That is deterministic without inventing a rule — the same tie-break
	// `matchCountOf` and `biggestAttack` take.
	for mask := 1; mask < 1<<len(offence); mask++ {
		var pick []int
		cost := 0
		for bit, idx := range offence {
			if mask&(1<<bit) == 0 {
				continue
			}
			pick = append(pick, idx)
			cost += hand[idx].Cost()
		}
		if len(pick) > slots || cost > budget {
			continue
		}
		if score := blowScore(d, hand, pick, hands, mixes); score > bestScore {
			bestScore, bestCost, best = score, cost, pick
		}
	}

	out := make([]Card, 0, len(best))
	for _, i := range best {
		out = append(out, hand[i])
	}
	return out, bestCost
}

// maxSearchableAttacks bounds the exhaustive search. Seven is a full enemy hand; this is well
// above it and exists only so a sim probing outside the rules degrades rather than hangs.
const maxSearchableAttacks = 14

// blowScore is what one candidate turn would actually land, run through the same matcher the
// resolver uses. It is the blow before any defence, which is all a planner can know — it cannot
// see what the other side has raised.
func blowScore(d Duelist, hand []Card, pick []int, hands []Hand, mixes []Mix) int {
	// **A solo attacker's turn is worth the sum of its cards and nothing else.** No hand is read,
	// so there is no multiplier to chase and no card that earns nothing — which also means the
	// search is now looking for the most damage the budget buys rather than the best combination.
	if d.SoloAttacks {
		total := 0
		for _, idx := range pick {
			total += hand[idx].Damage(d.DMG)
		}
		return total
	}

	turn := make([]Slot, 0, len(pick))
	for i, idx := range pick {
		turn = append(turn, Slot{Index: i, Card: hand[idx]})
	}

	blow := blowFor(turn, hands, mixes)
	if len(blow.Cards) == 0 {
		return 0
	}

	base := 0
	for _, i := range blow.Cards {
		base += turn[i].Card.Damage(d.DMG)
	}
	return base + scaleDamage(referenceSwing(d.DMG), blow.Multiplier)
}

// greedyAttacks is the fallback for a hand too big to search: the dearest cards that fit, which is
// what the old brute did.
func greedyAttacks(hand []Card, offence []int, budget, slots int) ([]Card, int) {
	used := make([]bool, len(hand))
	var out []Card
	spent := 0

	for len(out) < slots {
		best, bestCost := -1, 0
		for _, i := range offence {
			if used[i] {
				continue
			}
			if cost := hand[i].Cost(); cost <= budget-spent && cost > bestCost {
				best, bestCost = i, cost
			}
		}
		if best < 0 {
			break
		}
		used[best] = true
		out = append(out, hand[best])
		spent += bestCost
	}
	return out, spent
}

// spareCards fills the slots and points the attacks did not want with whatever else the hand holds.
//
// **This is what keeps a non-attack card in an enemy deck from being dead content.** A planner that
// only maximised damage would never raise a shield or bank a point, so every `Congeal` authored
// into the roster would sit in a discard pile forever. Attacking is still the whole of the plan;
// this is the change left over.
//
// **Defences go up first, then the hand's own order.** A defence is the one leftover that changes
// whether the enemy is alive to use the next one, so it earns the tie-break; past that the deck
// author decides by what they put in, and the draw decides which of it turned up.
func spareCards(hand []Card, chosen []Card, budget, slots int) []Card {
	if slots <= 0 || budget <= 0 {
		return nil
	}

	used := make([]bool, len(hand))
	for _, c := range chosen {
		for i, h := range hand {
			if !used[i] && h == c {
				used[i] = true
				break
			}
		}
	}

	var out []Card
	for _, wantDefend := range []bool{true, false} {
		for i, c := range hand {
			if used[i] || len(out) >= slots {
				continue
			}
			s := c.Spec()
			if s.Verb == VerbAttack {
				continue // an attack the search already declined is an attack that did not help
			}
			if (s.Verb == VerbDefend) != wantDefend {
				continue
			}
			if s.Cost > budget {
				continue
			}
			used[i] = true
			out = append(out, c)
			budget -= s.Cost
		}
	}
	return out
}
