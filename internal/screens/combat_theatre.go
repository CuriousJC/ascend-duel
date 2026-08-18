package screens

import (
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// The theatre: what travels, out of what, into what.
//
// **This file is the map, not the machinery** *(2026-08-18)*. It says, for every kind of event the
// engine can emit, where the thing it produced sets off from and where it lands. The drawings
// themselves live with the code that already owns each region — the sum in `combat_mathbox.go`, the
// seats in `combat_table.go`, the ring row in `combat_rings.go` — because a generic renderer over
// nine genuinely different gestures would be more machinery than the nine gestures.
//
// **The rule it exists to hold is one sentence: everything travels from the thing that caused it to
// the thing it happened to.** A figure appearing in the middle of the screen is a figure that was
// never anywhere else, so the player has to be *told* what it was instead of having watched it
// happen — the same argument that makes every card a journey rather than a reposition. Written down
// as a table the rule is checkable; written into nine call sites it is a habit.
//
// **The checkable part is completeness.** `TestEveryEventKindIsChoreographed` fails when a new
// EventKind arrives without an entry, which is the failure this file is really for: the Resolution
// feed is going behind a button, and after that an event nobody drew is an event the player is
// never told about. A kind that genuinely has no picture says so with `anchorNone` and a reason.
//
// **It is deliberately not JSON** *(2026-08-18, owner asked)*. The five-tuple is real, but every
// anchor below is a geometry *function* that already exists and takes arguments — `ringSlotAt`,
// `enemySeatAt`, `enemyCardRect` — and a file can only name one by string, so the Go table a file
// would need underneath it is this table. And `data/*.json` is `//go:embed`ed, so a beat in a file
// costs the same rebuild as a beat in a constant: the one argument that usually wins for data does
// not apply here. If the *timings* turn out to be what gets edited over and over, lift the timings
// out and leave the anchors. Timings are data; anchors are code.

// anchor names a place on the combat screen that something can leave from or arrive at.
//
// **They are named by role, never by side.** `anchorActorSeat` is the acting side's row whichever
// side that is, because the engine has no idea which duelist is a person and this screen must not
// grow a second opinion about it — the same reason `SoloAttacks` is a flag on a duelist rather than
// a rule about `SideB`.
type anchor int

const (
	// anchorNone is nothing travelling: the event changes no picture, or the picture it changes is
	// already drawn by something else.
	anchorNone anchor = iota

	// anchorActorSeat is the acting card's seat on the table — `playedSeatAt` or `enemySeatAt`
	// depending on the side, which is exactly the choice `seatPlayedCards` already makes.
	anchorActorSeat

	// anchorActorCard and anchorTargetCard are the two fighter cards in the top corners.
	anchorActorCard
	anchorTargetCard

	// anchorActorBadges is the status badge row along the bottom of the acting side's fighter
	// card. It is where a status that is *already standing* acts from, as against one landing.
	anchorActorBadges

	// anchorRing is one worn ring's card in the ring row, named by `Event.Ring`. This is the
	// anchor the engine gained a field for: nothing else on screen can say which ring caused a
	// status.
	anchorRing

	// anchorSumLine is the combo dialog's line of figures, in the feed's collapsed band.
	anchorSumLine

	// anchorBlow is where this turn's damage figure was last seen, and it is the one anchor with a
	// rule rather than a rectangle: **the sum line when the turn scored a hand, the acting card's
	// own seat when it did not**.
	//
	// It exists because `KindDamage` means two different pictures. A player's turn is one blow
	// scored off a hand, so its figure is already on screen in the sum and should travel from
	// there. A solo attacker emits no `KindCombo` at all — every attack lands its own face damage,
	// one card at a time — so there is no sum, and the figure has to come out of the card that
	// swung. `soloAttacker(side)` is the predicate that already knows which.
	anchorBlow

	// anchorAPFigure is the action-point figure on the button strip, under the left end of the AP
	// bar. What a Prepare banks lands on the number it changes.
	anchorAPFigure

	// anchorHandRow is the hand band itself — the row a Plan widens.
	anchorHandRow

	// anchorCount is the width of the enum and is never a place. Keep it last.
	anchorCount
)

func (a anchor) String() string {
	switch a {
	case anchorNone:
		return "nothing"
	case anchorActorSeat:
		return "the acting card's seat"
	case anchorActorCard:
		return "the acting fighter's card"
	case anchorTargetCard:
		return "the target fighter's card"
	case anchorActorBadges:
		return "the acting fighter's badge row"
	case anchorRing:
		return "the ring named on the event"
	case anchorSumLine:
		return "the sum line"
	case anchorBlow:
		return "wherever this turn's figure last was"
	case anchorAPFigure:
		return "the action-point figure"
	case anchorHandRow:
		return "the hand row"
	}
	return "?"
}

// gesture is how a thing gets from its source to its target.
//
// **A flown thing came off something; a popped thing is punctuation the game supplied.** That
// distinction is the whole grammar of the sum box and it generalises: if the player is meant to be
// able to ask "where did that come from", it flies.
type gesture int

const (
	// gestureNone draws nothing.
	gestureNone gesture = iota

	// gestureFly travels from the source to the target and grows into place.
	gestureFly

	// gesturePop is stamped at the target without travelling, for something the game is saying
	// rather than something a card produced — an operator, a multiplier, a label.
	gesturePop

	// gestureStrike marks out something already on screen at the target, for an outcome that is
	// the *absence* of what was about to happen.
	gestureStrike

	// gestureTally is a figure at the target counting to a new value, for a number the player is
	// meant to watch move rather than watch arrive.
	gestureTally

	// gestureCount is the width of the enum and is never a gesture. Keep it last.
	gestureCount
)

func (g gesture) String() string {
	switch g {
	case gestureNone:
		return "nothing"
	case gestureFly:
		return "flies"
	case gesturePop:
		return "pops"
	case gestureStrike:
		return "is struck out"
	case gestureTally:
		return "counts"
	}
	return "?"
}

// flightSpec is one event kind's choreography: what it looks like when it happens.
type flightSpec struct {
	from, to anchor
	how      gesture

	// why is the reason this event is drawn the way it is, in a few words. It is a field rather
	// than a comment beside the entry because a table read as a whole is how this screen's grammar
	// gets checked for consistency, and a reason that has to be hunted for is a reason nobody
	// compares against its neighbours.
	why string
}

// theatre is every event kind and what it does on screen.
//
// **Every kind has an entry, including the ones that draw nothing.** An absent entry and a
// deliberate `anchorNone` read identically and mean completely different things, and the test that
// walks this table cannot tell them apart unless the silence is written down.
var theatre = map[combat.EventKind]flightSpec{
	combat.KindRoundStart: {
		anchorNone, anchorNone, gestureNone,
		"bookkeeping - nothing has happened to anybody yet",
	},
	combat.KindAction: {
		anchorNone, anchorNone, gestureNone,
		"the card lifting in its own seat is the drawing, and tableFireLift already does it",
	},
	combat.KindGathered: {
		anchorActorSeat, anchorAPFigure, gestureFly,
		"banked points land on the budget figure they change",
	},
	combat.KindDrew: {
		anchorActorSeat, anchorHandRow, gestureFly,
		"a Plan widens the next hand, so it flies at the row it widens",
	},
	combat.KindNegated: {
		anchorNone, anchorSumLine, gesturePop,
		"a defence is the reverse of a combo and takes the combo's grammar: x50% on the same line",
	},
	combat.KindDamage: {
		anchorBlow, anchorTargetCard, gestureFly,
		"the figure travels into the card whose bar it empties, and the bar drops on arrival",
	},
	combat.KindDefeated: {
		anchorNone, anchorNone, gestureNone,
		"the bar reaching zero says it; a word as well would say it twice",
	},
	combat.KindCombo: {
		anchorActorSeat, anchorSumLine, gestureFly,
		"each card's own figure flies out of that card into the sum - the dialog that started this",
	},
	combat.KindChilled: {
		anchorActorBadges, anchorActorSeat, gestureFly,
		"the chill stands on the duelist losing the card, so it acts from their own badge row",
	},
	combat.KindStatus: {
		anchorRing, anchorTargetCard, gestureFly,
		"a status has a cause the player is wearing; Event.Ring exists so this can be drawn",
	},
	combat.KindMissed: {
		anchorNone, anchorSumLine, gestureStrike,
		"one shock roll takes the whole turn, so what is struck out is the total, not one card",
	},
	combat.KindBurned: {
		anchorActorBadges, anchorActorCard, gestureFly,
		"a burn is resident on its victim: off their own flame badge into their own bar",
	},
	combat.KindRoundEnd: {
		anchorNone, anchorNone, gestureNone,
		"bookkeeping - the round boundary is a moment, not an event with a victim",
	},
}
