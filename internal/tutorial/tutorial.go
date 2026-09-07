package tutorial

// The script, its two closed vocabularies, and the cursor that walks it.

import (
	"fmt"

	"github.com/curiousjc/ascend-duel/data"
)

// Anchor is what a step points at. **Append-only, and never serialized** — the same rule
// `combat.ConceptID` and `systems.GlyphKind` carry, for the same reason: `internal/screens` keys
// its rectangle table by the ordinal, so inserting one mid-enum silently re-points every entry
// after it. What a file writes down is the name.
type Anchor int

const (
	// AnchorNone is a step that points at nothing. The bubble sits in the middle of the screen
	// and the game underneath is untouched. It is the zero value on purpose: a step that said
	// nothing about where to point should aim nowhere rather than at whatever came first.
	AnchorNone Anchor = iota

	// The combat screen.
	AnchorEnemyCard
	AnchorDuelistCard
	AnchorHand

	// AnchorFirstCard is the single leftmost card of the hand, where [AnchorHand] is the whole
	// band.
	//
	// **A gate is only as tight as its anchor, and that is the whole lesson of this one**
	// *(owner's call, 2026-08-25)*. The step that says "click one card to queue it" was anchored on
	// the band, so a player could queue all five — which left the next step saying "one action
	// point gone" when five were, and the step after that ("take the other four") satisfied before
	// it was drawn. Nothing threw; the tutorial simply started describing a game that was no longer
	// on screen.
	//
	// So an anchor names the thing to be clicked, never the region it sits in.
	AnchorFirstCard

	AnchorAPBar
	AnchorDuelButton
	AnchorHandsButton
	AnchorDeckStack
	AnchorTowerPlace
	AnchorMathBand

	// The post-battle screen.
	AnchorRewardWorms

	// AnchorBuildCard is the duelist card in the band both between-fights screens carry, which is
	// where the purse is written. **Answered by two scenes**, uniquely among the anchors, because
	// it is literally the same card in the same place — see `buildCardRect`, which both defer to.
	AnchorBuildCard

	// The shop.
	AnchorShopShelf

	// AnchorMatchingCards is the largest set of cards in the hand that match on the script's
	// [MatchAxis] — the
	// five Jabs of the opening lesson — as one rectangle over the seats they occupy.
	//
	// **It names a set the player has to find, which no other anchor does.** Every other anchor
	// is a control at a known place; this one is a fact about the hand that was dealt, so the
	// scene computes it from the cards rather than from the layout. That is what lets a step say
	// "cards that match make a hand" against a *real* hand instead of against a fixture deck
	// holding nothing else — see the tutorial section of CLAUDE.md.
	//
	// **Pairing it with [CondMatchQueued] is what makes the lesson true by construction**: the
	// lock leaves only these cards clickable, so the hand the player builds is the hand Bob just
	// described. An empty hand, or one with nothing matching in it, reports no rectangle.
	AnchorMatchingCards

	// AnchorShopLeave is the button that ends the shop visit.
	//
	// **A step waiting on an outcome still has to say how to reach it** *(owner's call,
	// 2026-08-25)*. The shop step pointed at the shelf and waited for the run to get back to a
	// fight, which reads as a lock-up: the player buys everything they can afford, and then
	// nothing they do satisfies a step that is pointing somewhere else entirely. An outcome the
	// player reaches by pressing one particular button should have that button lit.
	AnchorShopLeave

	// AnchorLedgerButton is the LEDGER button in the control column — the run's account, and the
	// one anchor that names a control the frame owns rather than a scene.
	//
	// **The rectangle is still a scene's answer**, because `internal/game` places the button by
	// asking `screens.ControlColumnSlot` where the column's last rung is. So the anchor crosses no
	// package line that the button had not already crossed, and the lit square is derived from the
	// same function that positions the thing it lights.
	//
	// **Pairing it with [CondLedgerOpened] costs one frame, deliberately.** The panel takes the
	// whole frame while it is up and the scene under it is not updated at all, so the step cannot
	// advance until the player closes the account again — which is the right moment anyway: a step
	// that gave way while the panel was covering it would have Bob talking to a closed door.
	AnchorLedgerButton

	// AnchorShopWorn is the row of rings the run is actually wearing, in the build band.
	//
	// **The argument against it expired** *(2026-09-06)*. It was deliberately absent because a run
	// reaches its first shop wearing nothing, so a step pointing at the row would have pointed at
	// an empty band — vocabulary ahead of a use. The lesson now makes the player buy two before it
	// says anything about them, so by the time this is pointed at there is something in it. It
	// still reports false for an empty row, which is what keeps that honest.
	AnchorShopWorn

	// AnchorShopDMGPotion is the Draught's own seat in the potions pane — **one card, not the
	// pane**, and that is a lock-up being avoided rather than a preference.
	//
	// The step that names it waits for the Draught to be drunk, and the lit square is the only
	// legal click. An anchor covering all three potions would let the player spend their last
	// vitae on a Salve and then face a step demanding a potion they can no longer afford, with
	// nothing on screen able to satisfy it. See the affordability test in `internal/screens`.
	AnchorShopDMGPotion

	// AnchorRoundTimer is the bar under the tower place: how many of the fight's rounds are gone.
	//
	// **It points at the readout, not at the rule.** The clock kills a duelist still standing at
	// the end of the last round and there is nothing on screen to point at when it does — the death
	// happens inside a resolved round — so the step that teaches it has to be able to name the bar
	// while it is still empty. That is why this is an anchor of its own rather than the step
	// borrowing `tower-place`: the two lines above the bar say where you are, and lighting them to
	// talk about time would point at the wrong sentence.
	AnchorRoundTimer
)

// anchorNames is the word each anchor is written as in `data/tutorial.json`.
//
// **One table read in both directions**, by [ParseAnchor] and by [Anchor.String], so a name
// cannot parse as one thing and print as another.
var anchorNames = map[Anchor]string{
	AnchorNone:          "",
	AnchorEnemyCard:     "enemy-card",
	AnchorDuelistCard:   "duelist-card",
	AnchorHand:          "hand",
	AnchorFirstCard:     "first-card",
	AnchorAPBar:         "ap-bar",
	AnchorDuelButton:    "duel-button",
	AnchorHandsButton:   "hands-button",
	AnchorDeckStack:     "deck-stack",
	AnchorTowerPlace:    "tower-place",
	AnchorMathBand:      "math-band",
	AnchorRewardWorms:   "reward-worms",
	AnchorBuildCard:     "build-card",
	AnchorShopShelf:     "shop-shelf",
	AnchorMatchingCards: "matching-cards",
	AnchorShopLeave:     "shop-leave",
	AnchorLedgerButton:  "ledger-button",
	AnchorShopWorn:      "shop-worn",
	AnchorShopDMGPotion: "shop-dmg-potion",
	AnchorRoundTimer:    "round-timer",
}

func (a Anchor) String() string {
	if n, ok := anchorNames[a]; ok {
		return n
	}
	return "unknown"
}

// Anchors is every anchor there is, in declaration order. `internal/screens` walks it to check
// that each one has a rectangle behind it.
func Anchors() []Anchor {
	out := make([]Anchor, 0, len(anchorNames))
	for a := AnchorNone; int(a) < len(anchorNames); a++ {
		out = append(out, a)
	}
	return out
}

// ParseAnchor resolves the word a step wrote. The empty string is [AnchorNone] and is legal;
// anything else the table does not hold is refused.
func ParseAnchor(s string) (Anchor, error) {
	for a, name := range anchorNames {
		if name == s {
			return a, nil
		}
	}
	return AnchorNone, fmt.Errorf("tutorial: %q is not an anchor", s)
}

// Condition is what has to become true before a step gives way to the next one.
//
// Append-only and never serialized, exactly as [Anchor] is.
type Condition int

const (
	// CondNext is the Next button: the player has read the step and says so. The only condition
	// that asks for an acknowledgement rather than an action, and therefore the only one a step
	// can satisfy without learning anything — which is why the interesting steps do not use it.
	CondNext Condition = iota

	// CondCardsQueued is at least one card sitting in the action box.
	CondCardsQueued

	// CondHandEmptied is every card the player is holding sitting in the action box, with none
	// left in the hand.
	//
	// **It is what lets a lesson say "play all of them" and mean it.** `cards-queued` fires on the
	// first card, so a step teaching that matching cards make a hand would give way before the
	// second one was picked up — and the whole point being made is what happens when the set is
	// complete.
	CondHandEmptied

	// CondDuelPressed is a round under way: the player committed their turn.
	CondDuelPressed

	// CondRoundDone is playback finished and the screen back in the player's hands, whether the
	// duel is settled or not.
	CondRoundDone

	// CondMatchQueued is every card of [AnchorMatchingCards] sitting in the action box.
	//
	// **It is `hand-emptied` for a hand with other cards in it** *(2026-08-25)*. That condition
	// wants the *whole* hand queued, which only a fixture deck dealt to exactly the lesson can
	// ever reach: a real hand of eight against a five-card cap and a six-point budget leaves
	// cards behind by the rules of the game, so the step waited forever. This one asks for the
	// set the step is pointing at, which is what it was always asking for.
	CondMatchQueued

	// The three phase conditions: the run has reached that station. They are how a step waits out
	// something open-ended — a fight that takes as many rounds as it takes — without the script
	// having to describe it.
	CondPhaseFight
	CondPhaseReward
	CondPhaseShop

	// CondLedgerOpened is the run's account having been opened since this step came up.
	//
	// **It is measured against a baseline, exactly as [CondRoundDone] is**, because the ledger is
	// reachable from every screen at every moment: a player who opened it out of curiosity three
	// steps earlier must still be asked to open it when the step that teaches it arrives.
	//
	// **It is an action condition**, so the step gates to its anchor — the LEDGER button is the
	// one thing clickable while it stands. That is what makes it safe for the panel to eat the
	// frame: there is nothing else the player could have been doing.
	CondLedgerOpened

	// CondRingsWorn is the run wearing at least the step's [Step.Count] rings.
	//
	// **The only condition that reads a number off the script**, because "buy two" is a fact about
	// the lesson rather than about the game — a second tutorial teaching a single ring would want
	// the same condition with a different figure. Parse refuses it without one.
	CondRingsWorn

	// CondDMGBought is the Draught drunk: the run carrying a damage bonus it did not have.
	//
	// **Measured against a baseline like the ledger's**, so a step asking for it cannot be
	// satisfied by a bonus bought before the step came up. Nothing in the taught run can do that
	// today, and relying on that would be relying on the shape of one script.
	CondDMGBought
)

var conditionNames = map[Condition]string{
	CondNext:         "next",
	CondCardsQueued:  "cards-queued",
	CondHandEmptied:  "hand-emptied",
	CondMatchQueued:  "matching-queued",
	CondDuelPressed:  "duel-pressed",
	CondRoundDone:    "round-done",
	CondPhaseFight:   "phase-fight",
	CondPhaseReward:  "phase-reward",
	CondPhaseShop:    "phase-shop",
	CondLedgerOpened: "ledger-opened",
	CondRingsWorn:    "rings-worn",
	CondDMGBought:    "dmg-bought",
}

func (c Condition) String() string {
	if n, ok := conditionNames[c]; ok {
		return n
	}
	return "unknown"
}

// ParseCondition resolves the word a step wrote. **There is no default**: an `Until` the
// vocabulary does not hold is refused rather than treated as `next`, because a step that quietly
// became a Next-button step is a lesson the player is never actually taught.
func ParseCondition(s string) (Condition, error) {
	for c, name := range conditionNames {
		if name == s {
			return c, nil
		}
	}
	return CondNext, fmt.Errorf("tutorial: %q is not a condition", s)
}

// isAction reports whether a condition is satisfied by the player clicking something on the
// screen, as opposed to by the game arriving somewhere.
//
// **An action condition must gate** — see the check in [Parse]. `cards-queued`, `hand-emptied`,
// `matching-queued` and `duel-pressed` are all satisfied by a click, so a step waiting on one is asking the player to do
// a specific thing; leaving the rest of the screen live lets them do *more* than the step
// describes, and every step after it is then narrating a game that is no longer on screen.
//
// The phase conditions and `round-done` are outcomes rather than clicks. A step waiting for a
// fight to be won cannot gate: winning takes as many clicks as it takes, on controls the step has
// no business naming.
func (c Condition) isAction() bool {
	switch c {
	case CondCardsQueued, CondHandEmptied, CondMatchQueued, CondDuelPressed, CondLedgerOpened,
		CondRingsWorn, CondDMGBought:
		return true
	}
	return false
}

// Lock is how much of the screen a step takes away from the player.
//
// **It is derived from the condition, never authored** *(owner's call, 2026-08-25)*. It was a
// `Gate` field in the file for a few hours and that was the bug: a step about which room you are
// standing in said `Gate: false`, meaning "do not lock anything", and a player read it while
// queueing two cards the step had not mentioned. What the rule actually is — lock everything that
// is not the thing the tutorial is asking for — has exactly three cases, and the condition already
// says which one a step is in. A field could only ever disagree with it.
type Lock int

const (
	// LockNone leaves the whole screen live. **Only the outcome conditions get it**: a step waiting
	// for a fight to be won, a reward to be taken or a shop to be left cannot say which controls
	// that will need, and locking one would deadlock the tutorial against its own condition.
	LockNone Lock = iota

	// LockAll leaves nothing live but Bob's own two buttons. It is what a step that simply says
	// something gets, because the only thing it is asking for is to be read.
	LockAll

	// LockToAnchor leaves Bob's buttons and the anchor. It is what a step asking for a specific
	// click gets, and the anchor is that click — see [Anchor] on why it names the control rather
	// than the region around it.
	LockToAnchor
)

var lockNames = map[Lock]string{LockNone: "none", LockAll: "all", LockToAnchor: "to-anchor"}

func (l Lock) String() string {
	if n, ok := lockNames[l]; ok {
		return n
	}
	return "unknown"
}

// lockFor is the three-way split, and the one place it is decided.
//
// **[CondRoundDone] locks everything, which is what separates it from the phase conditions**
// *(2026-09-06)*. They share a shape — an outcome, not a click — and were treated alike, so a step
// narrating a round's playback left the whole screen live and the player queued the next turn while
// Bob was still describing this one. The distinction is whether the player has anything to *do*:
// winning a fight or leaving a shop takes clicks on controls the step has no business naming, where
// a round plays itself out and there is nothing to press. So it gets what a Next-button step gets.
func lockFor(c Condition) Lock {
	switch {
	case c.isAction():
		return LockToAnchor
	case c == CondNext, c == CondRoundDone:
		return LockAll
	default:
		return LockNone
	}
}

// Step is one thing Bob says, with the file's strings already resolved.
type Step struct {
	Key    string
	Text   string
	Anchor Anchor
	Lock   Lock
	Until  Condition

	// Count is how many of something the step is waiting for, and **only the counting conditions
	// read it**. It is the one number the script hands the state machine.
	//
	// **Refused where it means nothing and required where it is needed**, per Parse: a step
	// carrying a count its condition cannot read is a figure nobody will ever act on, which is how
	// a script comes to say something it is not doing.
	Count int
}

// Facts is what a scene says is true this frame, and it is the whole of what a condition may
// read. See the package doc for why the traffic goes this way rather than as events.
//
// **The zero value is a scene that has published nothing**, and it satisfies no condition except
// [CondNext]. That is deliberate: a screen that forgets to publish stalls the tutorial on its
// first real step, which is loud, rather than skipping steps, which is not.
type Facts struct {
	// Phase is where the run is — `session.Phase.String()`, passed as a string so that this
	// package does not import `internal/session` to read one word off it.
	Phase string

	// Queued is how many cards are sitting in the action box.
	Queued int

	// Unqueued is how many cards the player is holding that they have **not** put in the queue.
	//
	// **It is not the size of the hand, and that distinction cost a bug** *(2026-08-25)*. Queueing
	// a card does not take it out of the hand — the combat screen marks it `selected` and leaves it
	// in the row, and `fighterActions` is the selected subset — so a step waiting for the hand to
	// be *emptied* waited forever against a hand that never shrinks. The field is named for what it
	// counts rather than for where the cards are.
	//
	// [CondHandEmptied] reads this and Queued together, so that a hand with nothing in it because
	// nothing was ever dealt does not read as one the player has finished queueing.
	Unqueued int

	// Matching is how many cards of the hand form its largest matching set, and MatchingQueued
	// how many of those the player has put in the queue. [CondMatchQueued] reads both.
	//
	// **Two fields rather than a bool**, so the condition is a comparison a test can drive to
	// either side, and so a scene that publishes neither reports the zero value and satisfies
	// nothing — the same property the rest of Facts has.
	Matching, MatchingQueued int

	// Resolving is whether a round is playing back rather than being planned.
	Resolving bool

	// LedgerOpens is `state.LedgerOpens`: how many times the run's account has been opened this
	// session. [CondLedgerOpened] reads it against a baseline, for the reason RoundsPlayed is read
	// that way — see below.
	LedgerOpens int

	// RingsWorn is how many rings the run has on, and DMGBonus what its potions have added to the
	// duelist's damage. [CondRingsWorn] and [CondDMGBought] read them — the second against a
	// baseline, the first as a total, because a ring can be sold again and a step asking for two
	// rings wants two rings on the hand rather than two purchases ever made.
	RingsWorn int
	DMGBonus  int

	// RoundsPlayed is how many rounds this fight has resolved. [CondRoundDone] reads it rather
	// than watching `Resolving` fall, because a step that arrived *during* playback would see
	// Resolving go false at the end of the round it did not start and count that as its own.
	RoundsPlayed int
}

// MatchAxis is which axis [AnchorMatchingCards] counts a set on — the same three axes a hand is
// scored on, because a "set" the tutorial points at has to be a set the matcher would pay for.
//
// **It is authored rather than derived** *(2026-08-25)*. The anchor counted concepts and nothing
// else while the lesson was five Jabs; teaching an elemental four of a kind meant the square and
// the sentence had to agree about which axis they were talking about, and no rule in the script
// says which one a lesson is about. It is one word in the file.
//
// **Closed and not defaulted**, like every other vocabulary here: a script that uses the matching
// anchor without naming an axis is refused at load, because an axis silently defaulting to concept
// is a lesson pointing confidently at the wrong cards.
type MatchAxis int

const (
	// MatchNone is a script that never points at a matching set.
	MatchNone MatchAxis = iota
	MatchConcept
	MatchForm
	MatchElement
)

var matchNames = map[MatchAxis]string{
	MatchConcept: "concept",
	MatchForm:    "form",
	MatchElement: "element",
}

func (m MatchAxis) String() string {
	if n, ok := matchNames[m]; ok {
		return n
	}
	return "none"
}

// ParseMatchAxis resolves the word a script writes. Empty is [MatchNone], which is legal only for
// a script with no matching step in it — Parse is what enforces that.
func ParseMatchAxis(s string) (MatchAxis, error) {
	if s == "" {
		return MatchNone, nil
	}
	for m, n := range matchNames {
		if n == s {
			return m, nil
		}
	}
	return MatchNone, fmt.Errorf("%q is not an axis a hand is scored on: concept, form or element", s)
}

// Script is a parsed `data/tutorial.json`: the steps, and the run the lesson needs to be true.
//
// **A script carries its own preconditions** *(2026-08-25)*. Bob promises a hand of five matching
// cards and a fight ended in one blow, and neither is a fact about the game — they are facts about
// one deal against one creature. They were pinned by `internal/scenario` while a fixture was the
// only way to start the tutorial, and the day the profile became a real trigger the lesson ran on
// whatever the clock had rolled and promised five Jabs over a hand holding two. A promise and the
// thing that makes it true have to travel together.
//
// **This package resolves neither of them.** A seed is `internal/seeds`' business and an opponent
// is the climb's; both are strings here and `main` and `internal/session` are what act on them —
// the same line every other vocabulary in this file draws between naming a thing and being it.
type Script struct {
	// Seed is the run code the lesson is written against, or empty for a lesson that does not care
	// what it is dealt.
	Seed string

	// Enemy is the record key the first room should stand, or empty for whoever the climb put there.
	Enemy string

	// Match is the axis [AnchorMatchingCards] counts on. Required by any script that uses that
	// anchor or [CondMatchQueued]; see Parse.
	Match MatchAxis

	// Steps is what Bob says, in order.
	Steps []Step
}

// Len is how many steps the script holds.
func (s Script) Len() int { return len(s.Steps) }

// Load parses and validates the script, and **panics on anything it cannot resolve**.
//
// A panic rather than an error for the reason every other catalogue in this game panics at load:
// it fails when the binary starts rather than at the step nobody reached in testing. A tutorial
// is the one feature whose whole audience is a player who does not yet know what the game is
// supposed to look like, so a broken one is worse than none.
func Load() Script {
	s, err := Parse(data.LoadTutorial())
	if err != nil {
		panic("tutorial: " + err.Error())
	}
	return s
}

// Parse is Load without the panic, so a test can assert on a bad record rather than recover from
// one.
func Parse(in data.TutorialData) (Script, error) {
	records := in.Steps
	if len(records) == 0 {
		return Script{}, fmt.Errorf("the script is empty")
	}

	axis, err := ParseMatchAxis(in.Match)
	if err != nil {
		return Script{}, fmt.Errorf("Match: %w", err)
	}

	out := Script{Seed: in.Seed, Enemy: in.Enemy, Match: axis, Steps: make([]Step, 0, len(records))}
	seen := map[string]bool{}
	usesMatching := false

	for i, r := range records {
		if r.StepRecord == "" {
			return Script{}, fmt.Errorf("step %d has no StepRecord", i)
		}
		if seen[r.StepRecord] {
			return Script{}, fmt.Errorf("step %q appears twice", r.StepRecord)
		}
		seen[r.StepRecord] = true

		if r.Text == "" {
			return Script{}, fmt.Errorf("step %q says nothing", r.StepRecord)
		}

		anchor, err := ParseAnchor(r.Anchor)
		if err != nil {
			return Script{}, fmt.Errorf("step %q: %w", r.StepRecord, err)
		}
		until, err := ParseCondition(r.Until)
		if err != nil {
			return Script{}, fmt.Errorf("step %q: %w", r.StepRecord, err)
		}

		lock := lockFor(until)

		// **A step that asks for a click must have something to point the click at.** It is refused
		// rather than downgraded: silently turning it into a fully locked step would leave the
		// player with no legal click at all and a condition only they could satisfy.
		if lock == LockToAnchor && anchor == AnchorNone {
			return Script{}, fmt.Errorf(
				"step %q waits for the player to click something but names nothing to click",
				r.StepRecord)
		}

		if anchor == AnchorMatchingCards || until == CondMatchQueued {
			usesMatching = true
		}

		// **A count belongs to exactly the conditions that read one.** Required where it is read,
		// because a step waiting for "at least zero rings" is satisfied before it is drawn;
		// refused everywhere else, because a number nothing acts on is a script saying something
		// it is not doing.
		if until == CondRingsWorn && r.Count < 1 {
			return Script{}, fmt.Errorf(
				"step %q waits on %v and names no Count, so it is satisfied before it is shown",
				r.StepRecord, until)
		}
		if until != CondRingsWorn && r.Count != 0 {
			return Script{}, fmt.Errorf("step %q carries Count %d, which %v does not read",
				r.StepRecord, r.Count, until)
		}

		out.Steps = append(out.Steps, Step{
			Key: r.StepRecord, Text: r.Text, Anchor: anchor, Lock: lock, Until: until,
			Count: r.Count,
		})
	}

	// **A matching step with no axis is refused rather than defaulted.** The square the player is
	// shown and the condition that lets them past it are the same set of cards, and which cards
	// those are depends entirely on the axis — so a script that does not say is a script whose two
	// halves could disagree.
	if usesMatching && axis == MatchNone {
		return Script{}, fmt.Errorf(
			"the script points at a matching set but names no Match axis: concept, form or element")
	}
	return out, nil
}

// Run is how far through the script one player is.
//
// **It is a cursor and a script, and nothing about a screen.** Where it *lives* is
// `session.Session`, because the script spans the whole run loop and no scene outlives a fight —
// see the run-state rule in CLAUDE.md.
type Run struct {
	script Script
	step   int
	done   bool

	// baseRounds is RoundsPlayed as it stood when the current step came up, so [CondRoundDone]
	// measures a round this step actually watched.
	baseRounds int

	// baseLedger is the same trick for [CondLedgerOpened]: the account can be opened from any
	// screen at any moment, so the step asks for an opening of its own rather than for the tally
	// to be non-zero.
	baseLedger int

	// baseDMG is the same again for [CondDMGBought].
	baseDMG int
}

// NewRun starts the script at its first step.
func NewRun(s Script) *Run { return &Run{script: s} }

// Active reports whether there is still a step to show.
func (r *Run) Active() bool { return r != nil && !r.done && r.step < r.script.Len() }

// Current is the step being shown.
func (r *Run) Current() (Step, bool) {
	if !r.Active() {
		return Step{}, false
	}
	return r.script.Steps[r.step], true
}

// Enemy is the opponent the script asks for, or empty — for a run that is not being taught, for a
// script that does not care, and for a nil receiver, which is the ordinary case.
//
// **It does not stop answering when the script ends**, deliberately. A player who skips the lesson
// mid-fight must not have the creature in front of them swapped out, and a defeat re-enters the
// same room and should meet the same opponent — which is the rule everywhere else in the climb.
func (r *Run) Enemy() string {
	if r == nil {
		return ""
	}
	return r.script.Enemy
}

// Match is the axis this run's script counts a matching set on, or [MatchNone] for a run that is
// not being taught — which is the ordinary case, and why the nil receiver answers.
func (r *Run) Match() MatchAxis {
	if r == nil {
		return MatchNone
	}
	return r.script.Match
}

// Skip ends the tutorial outright. It is what the Skip button does, and it is deliberately
// irreversible for the session: a player who dismissed Bob does not want him back three clicks
// later, and there is nowhere to put a control that would restore him.
func (r *Run) Skip() {
	if r != nil {
		r.done = true
	}
}

// Advance moves to the next step, ending the run when the script is spent.
func (r *Run) Advance(f Facts) {
	if !r.Active() {
		return
	}
	r.step++
	r.baseRounds = f.RoundsPlayed
	r.baseLedger = f.LedgerOpens
	r.baseDMG = f.DMGBonus
	if r.step >= r.script.Len() {
		r.done = true
	}
}

// Update is the once-a-frame call: it takes what the scene says is true and advances the step if
// that satisfies its condition. `nextPressed` is the Next button, which only [CondNext] reads.
//
// **It advances at most one step per frame**, deliberately. Chaining — letting a step whose
// condition is already satisfied fall straight through to the next — would skip past whatever the
// player was meant to read on the way, and the case is real: `phase-shop` is true for every frame
// the shop is on screen, so a following `phase-shop` step would never be seen at all.
func (r *Run) Update(f Facts, nextPressed bool) {
	step, ok := r.Current()
	if !ok {
		return
	}
	if r.satisfied(step, f, nextPressed) {
		r.Advance(f)
	}
}

func (r *Run) satisfied(step Step, f Facts, nextPressed bool) bool {
	switch step.Until {
	case CondNext:
		return nextPressed
	case CondCardsQueued:
		return f.Queued > 0
	case CondHandEmptied:
		return f.Queued > 0 && f.Unqueued == 0
	case CondMatchQueued:
		return f.Matching > 0 && f.MatchingQueued == f.Matching
	case CondDuelPressed:
		return f.Resolving
	case CondRoundDone:
		return f.RoundsPlayed > r.baseRounds && !f.Resolving
	case CondLedgerOpened:
		return f.LedgerOpens > r.baseLedger
	case CondRingsWorn:
		// **A step with no count is never satisfied**, rather than satisfied immediately. Parse
		// refuses one in a real script, but the zero value has to stall like everything else here
		// — a condition that sails through on Facts nobody published is the failure the whole of
		// this switch is written to avoid.
		return step.Count > 0 && f.RingsWorn >= step.Count
	case CondDMGBought:
		return f.DMGBonus > r.baseDMG
	case CondPhaseFight:
		return f.Phase == "fight"
	case CondPhaseReward:
		return f.Phase == "reward"
	case CondPhaseShop:
		return f.Phase == "shop"
	}
	return false
}

// Gate says how much of the screen the current step is holding, and which anchor it leaves open
// if it leaves one.
//
// **Three answers, not two.** `LockNone` means restrict nothing; `LockAll` means restrict
// everything, and the returned anchor is meaningless; `LockToAnchor` means restrict everything
// except that anchor. A caller collapsing the middle case into the first is the bug this replaced
// — see [Lock].
func (r *Run) Gate() (Anchor, Lock) {
	step, ok := r.Current()
	if !ok {
		return AnchorNone, LockNone
	}
	return step.Anchor, step.Lock
}
