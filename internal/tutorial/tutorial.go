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

	// AnchorShopLeave is the button that ends the shop visit.
	//
	// **A step waiting on an outcome still has to say how to reach it** *(owner's call,
	// 2026-08-25)*. The shop step pointed at the shelf and waited for the run to get back to a
	// fight, which reads as a lock-up: the player buys everything they can afford, and then
	// nothing they do satisfies a step that is pointing somewhere else entirely. An outcome the
	// player reaches by pressing one particular button should have that button lit.
	AnchorShopLeave
)

// anchorNames is the word each anchor is written as in `data/tutorial.json`.
//
// **One table read in both directions**, by [ParseAnchor] and by [Anchor.String], so a name
// cannot parse as one thing and print as another.
var anchorNames = map[Anchor]string{
	AnchorNone:        "",
	AnchorEnemyCard:   "enemy-card",
	AnchorDuelistCard: "duelist-card",
	AnchorHand:        "hand",
	AnchorFirstCard:   "first-card",
	AnchorAPBar:       "ap-bar",
	AnchorDuelButton:  "duel-button",
	AnchorHandsButton: "hands-button",
	AnchorDeckStack:   "deck-stack",
	AnchorTowerPlace:  "tower-place",
	AnchorMathBand:    "math-band",
	AnchorRewardWorms: "reward-worms",
	AnchorBuildCard:   "build-card",
	AnchorShopShelf:   "shop-shelf",
	AnchorShopLeave:   "shop-leave",
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

	// The three phase conditions: the run has reached that station. They are how a step waits out
	// something open-ended — a fight that takes as many rounds as it takes — without the script
	// having to describe it.
	CondPhaseFight
	CondPhaseReward
	CondPhaseShop
)

var conditionNames = map[Condition]string{
	CondNext:        "next",
	CondCardsQueued: "cards-queued",
	CondHandEmptied: "hand-emptied",
	CondDuelPressed: "duel-pressed",
	CondRoundDone:   "round-done",
	CondPhaseFight:  "phase-fight",
	CondPhaseReward: "phase-reward",
	CondPhaseShop:   "phase-shop",
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
// **An action condition must gate** — see the check in [Parse]. `cards-queued`, `hand-emptied` and
// `duel-pressed` are all satisfied by a click, so a step waiting on one is asking the player to do
// a specific thing; leaving the rest of the screen live lets them do *more* than the step
// describes, and every step after it is then narrating a game that is no longer on screen.
//
// The phase conditions and `round-done` are outcomes rather than clicks. A step waiting for a
// fight to be won cannot gate: winning takes as many clicks as it takes, on controls the step has
// no business naming.
func (c Condition) isAction() bool {
	switch c {
	case CondCardsQueued, CondHandEmptied, CondDuelPressed:
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
func lockFor(c Condition) Lock {
	switch {
	case c.isAction():
		return LockToAnchor
	case c == CondNext:
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

	// Resolving is whether a round is playing back rather than being planned.
	Resolving bool

	// RoundsPlayed is how many rounds this fight has resolved. [CondRoundDone] reads it rather
	// than watching `Resolving` fall, because a step that arrived *during* playback would see
	// Resolving go false at the end of the round it did not start and count that as its own.
	RoundsPlayed int
}

// Script is a parsed `data/tutorial.json`.
type Script []Step

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
func Parse(records []data.TutorialStepData) (Script, error) {
	if len(records) == 0 {
		return nil, fmt.Errorf("the script is empty")
	}

	out := make(Script, 0, len(records))
	seen := map[string]bool{}

	for i, r := range records {
		if r.StepRecord == "" {
			return nil, fmt.Errorf("step %d has no StepRecord", i)
		}
		if seen[r.StepRecord] {
			return nil, fmt.Errorf("step %q appears twice", r.StepRecord)
		}
		seen[r.StepRecord] = true

		if r.Text == "" {
			return nil, fmt.Errorf("step %q says nothing", r.StepRecord)
		}

		anchor, err := ParseAnchor(r.Anchor)
		if err != nil {
			return nil, fmt.Errorf("step %q: %w", r.StepRecord, err)
		}
		until, err := ParseCondition(r.Until)
		if err != nil {
			return nil, fmt.Errorf("step %q: %w", r.StepRecord, err)
		}

		lock := lockFor(until)

		// **A step that asks for a click must have something to point the click at.** It is refused
		// rather than downgraded: silently turning it into a fully locked step would leave the
		// player with no legal click at all and a condition only they could satisfy.
		if lock == LockToAnchor && anchor == AnchorNone {
			return nil, fmt.Errorf(
				"step %q waits for the player to click something but names nothing to click",
				r.StepRecord)
		}

		out = append(out, Step{
			Key: r.StepRecord, Text: r.Text, Anchor: anchor, Lock: lock, Until: until,
		})
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
}

// NewRun starts the script at its first step.
func NewRun(s Script) *Run { return &Run{script: s} }

// Active reports whether there is still a step to show.
func (r *Run) Active() bool { return r != nil && !r.done && r.step < len(r.script) }

// Current is the step being shown.
func (r *Run) Current() (Step, bool) {
	if !r.Active() {
		return Step{}, false
	}
	return r.script[r.step], true
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
	if r.step >= len(r.script) {
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
	if r.satisfied(step.Until, f, nextPressed) {
		r.Advance(f)
	}
}

func (r *Run) satisfied(c Condition, f Facts, nextPressed bool) bool {
	switch c {
	case CondNext:
		return nextPressed
	case CondCardsQueued:
		return f.Queued > 0
	case CondHandEmptied:
		return f.Queued > 0 && f.Unqueued == 0
	case CondDuelPressed:
		return f.Resolving
	case CondRoundDone:
		return f.RoundsPlayed > r.baseRounds && !f.Resolving
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
