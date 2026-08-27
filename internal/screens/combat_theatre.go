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

	// anchorSumLine is the hand dialog's line of figures, in the feed's collapsed band.
	anchorSumLine

	// anchorBlow is where this turn's damage figure was last seen, and it is the one anchor with a
	// rule rather than a rectangle: **the sum line when the turn scored a hand, the acting card's
	// own seat when it did not**.
	//
	// It exists because `KindDamage` means two different pictures. A player's turn is one blow
	// scored off a hand, so its figure is already on screen in the sum and should travel from
	// there. A solo attacker emits no `KindHand` at all — every attack lands its own face damage,
	// one card at a time — so there is no sum, and the figure has to come out of the card that
	// swung. `soloAttacker(side)` is the predicate that already knows which.
	anchorBlow

	// anchorAPFigure is the action-point figure on the button strip, under the left end of the AP
	// bar: **this round's budget being spent**.
	//
	// **Nothing targets it, and a Prepare deliberately does not** *(2026-08-19, owner's call)*. It
	// was this row's target on the reasoning that banked points should land on the AP number — but
	// there are two, and the one they change is the fighter card's, which is the budget with
	// `BonusAP` in it. A figure arriving here would be a number landing on a total that does not
	// move. It stays in the enum because it is a real place and the next thing that raises or
	// spends *this* round's budget belongs on it.
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

// choreography is every event kind and what it does on screen.
//
// **It is the table, not the stage.** What is currently *moving* is `combatTheatre` below; this
// says what each kind of event travels out of and into when it does.
//
// **Every kind has an entry, including the ones that draw nothing.** An absent entry and a
// deliberate `anchorNone` read identically and mean completely different things, and the test that
// walks this table cannot tell them apart unless the silence is written down.
var choreography = map[combat.EventKind]flightSpec{
	combat.KindRoundStart: {
		anchorNone, anchorNone, gestureNone,
		"bookkeeping - nothing has happened to anybody yet",
	},
	combat.KindAction: {
		anchorNone, anchorNone, gestureNone,
		"the card lifting in its own seat is the drawing, and tableFireLift already does it",
	},
	combat.KindGathered: {
		anchorActorSeat, anchorActorCard, gestureFly,
		"banked points land on the AP line they raise, which is the fighter card's and not the strip's",
	},
	combat.KindDrew: {
		anchorActorSeat, anchorHandRow, gestureFly,
		"a Plan widens the next hand, so it flies at the row it widens",
	},
	combat.KindNegated: {
		anchorNone, anchorSumLine, gesturePop,
		"a defence is the reverse of a hand and takes the hand's grammar: x50% on the same line",
	},
	combat.KindDamage: {
		anchorBlow, anchorTargetCard, gestureFly,
		"the figure travels into the card whose bar it empties, and the bar drops on arrival",
	},
	combat.KindDefeated: {
		anchorNone, anchorNone, gestureNone,
		"the bar reaching zero says it; a word as well would say it twice",
	},
	combat.KindHand: {
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
	combat.KindHealed: {
		anchorActorSeat, anchorActorCard, gestureFly,
		"a rider is a property of the card, so the life flies out of that card's seat into the bar it fills",
	},
	combat.KindRoundEnd: {
		anchorNone, anchorNone, gestureNone,
		"bookkeeping - the round boundary is a moment, not an event with a victim",
	},
}

// combatTheatre is everything the combat screen has moving on it.
//
// **The screen uses a theatre rather than being one** *(2026-08-21)*: the rules that apply to all
// of it are in theatre.go and are shared, and what is *in* it is this screen's own. A between-fight
// screen with things to move declares its own and implements the same three methods.
//
// The fields kept their comments when they moved off `CombatScene`, because each one still says
// something about that particular mover that nothing else does. What they no longer each say is
// the part that was true of all of them — that it is presentation, that it runs on the game's one
// speed, and that it is taken down together. That is theatre.go's now.
type combatTheatre struct {
	// Cards currently travelling to or from the draw pile. Purely something to look at:
	// every one of them is a ghost of a card that has already moved. See combat_flight.go.
	flights []cardFlight

	// Cards currently moving from one slot in the hand to another — a sort, or the row
	// closing up after cards were spent. Separate from flights rather than a fourth flag on
	// one, because a slide is the only mover whose journey begins and ends in the row, and it
	// is the only one that needs a row size at each end.
	slides []handSlide

	// The player's side of the table: the cards played this round, in resolution order, flying
	// out of the hand and into a row on the left facing the opponent's. Dealt in full the
	// moment the round starts — see seatPlayedCards — and what a hand narrows to the cards it
	// was made of.
	//
	// Cleared when the hand is spent, which is the moment those cards actually leave.
	resolved []resolvedCard

	// The opponent's side of the table: their whole queue for the coming round, flying in from
	// their own card in the top-right corner and settling into a row on the right.
	//
	// **It is laid out when the opponent plans, which is the start of the planning phase**, so
	// the player picks their round against a hand they can see. Re-seated once per round; see
	// planEnemyRound.
	enemyDealt []dealtCard

	// firingSeats and enemyFiringSeats are which cards on each side of the table are resolving
	// right now, empty for none. **Playback drives which cards are lit, not which cards exist**:
	// both hands are on the table from the moment DUEL! is pressed.
	//
	// Two fields rather than a side-plus-seat pair, because both rows are drawn independently
	// and each only ever asks about itself. `noteResolved` writes both on every action, so only
	// one side is ever lit at a time.
	//
	// **A list rather than one seat, because the attack phase is one blow** *(2026-08-14)*. The
	// cards announce one after another and stay up as they do, so the whole hand is raised by the
	// time the hand lands on it — and the hand then drops whichever of them earned nothing. A
	// single lit seat could only ever say "this card", which is the reading the one-blow rule
	// exists to stop.
	firingSeats      []int
	enemyFiringSeats []int

	// hits are the damage figures currently travelling into a fighter card, and the reason a
	// health bar can lag the life behind it. See combat_hits.go — the model is already correct
	// while one of these is up; what waits is the drawing.
	hits []hitFlight

	// banks are the `+2 AP` figures a Prepare sends to the fighter card whose budget it raises,
	// and bankShown is what they have already delivered — the points a card's AP line is drawing
	// on top of `Duelist.BonusAP`, which the engine does not write until the round's end state is
	// adopted. Indexed by side. See combat_bank.go.
	banks     []bankFlight
	bankShown [2]int

	// banner is the name of the hand the player committed, on its way from the planning seat to
	// the hand row or resting there. **It is raised at DUEL! and lives until the round is over**,
	// which is what makes the planned name and the fired one one word that moves rather than two
	// that appear. See combat_mathbox.go.
	banner handBanner

	// mathBox is the hand dialog: the blow's arithmetic acted out across the band above the hand
	// on the beat the hand fires. See combat_mathbox.go.
	//
	// **It is the one thing on this screen that can stop the playback cursor.** Every other
	// animation runs on its own clock beside the log; this one is a beat *of* the log, because a
	// sum revealed a figure at a time does not fit inside a single event's dwell. It still
	// decides nothing — `ResolveRound` settled the round before any of it was drawn.
	mathBox handMathBox
}

// combatTheatre answers the shared contract. **The assertion is the point of the interface**:
// nothing takes a `theatre` as a parameter, and what this line buys is that a second scene's
// theatre cannot quietly implement two of the three.
var _ theatre = (*combatTheatre)(nil)

// tick advances everything on stage by a frame and drops whatever has finished.
//
// **The hand dialog is deliberately not here.** It is the one mover that is a beat *of* playback
// rather than something running alongside it — the cursor waits for it — so `advancePlayback`
// drives it and this does not. See mathBox and combat_mathbox.go.
func (t *combatTheatre) tick() {
	t.flights = advance(t.flights)
	t.slides = advance(t.slides)
	t.hits = advance(t.hits)
	t.tickBanks()

	// The two rows on the table never expire: cards arrive and stay until the round is spent, so
	// they are advanced in place rather than filtered.
	for i := range t.resolved {
		t.resolved[i].tick()
	}
	for i := range t.enemyDealt {
		t.enemyDealt[i].tick()
	}

	// The committed hand's name. It sets off at DUEL!, before the first event is reached, and must
	// not be held up by the dialog that stops the cursor — which is why it ticks here with
	// everything else rather than inside playback. See handBanner.
	t.banner.tick()
}

// tickBanks is the one mover with its own loop, because it does something on the frame a figure
// *arrives* rather than on the frame it finishes.
//
// **The credit happens on arrival**, which is the whole point of the gesture: the number reaching
// the card is what raises the card's figure. It is kept in `bankShown` rather than read back off
// the live flights because a flight is dropped when its hold expires and the points have to stay
// on the card until the round's end state is adopted — see shownBank.
func (t *combatTheatre) tickBanks() {
	live := t.banks[:0]
	for i := range t.banks {
		b := &t.banks[i]
		was := b.arrived()
		b.tick()
		if !was && b.arrived() {
			t.bankShown[b.side] += b.amount
		}
		if !b.done() {
			live = append(live, *b)
		}
	}
	t.banks = live
}

// running reports whether anything the playback cursor waits on is still going.
//
// **It is the figures, not the cards.** A card flying to its seat runs alongside playback and never
// holds it up; a damage figure crossing to a health bar does, because the bar must not drop before
// the number reaches it. Adding a mover here is deciding that the round should wait for it.
func (t *combatTheatre) running() bool {
	return running(t.hits) || running(t.banks)
}

// clear takes the whole stage down, view state included.
//
// **This is the method the grouping exists for.** It used to be six statements and two calls in
// `Init`, each added after something was found still on screen at the start of the next fight: a
// figure in the air belongs to the fight that raised it, and a settled duel freezes rather than
// spending its hand, so anything tidied up only by the end-of-round spend was still there. Zeroing
// the struct cannot miss one, and a mover added tomorrow is covered without anyone remembering.
func (t *combatTheatre) clear() { *t = combatTheatre{} }
