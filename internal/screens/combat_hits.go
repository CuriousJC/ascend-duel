package screens

import (
	"image"
	"image/color"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// What a blow looks like landing: the figure travelling out of wherever it was worked out, into
// the card whose bar it empties, and the bar waiting for it to arrive.
//
// **It is the `KindDamage` row of the theatre table** — `anchorBlow` to `anchorTargetCard`,
// `gestureFly` — and it is the second half of the sum box. The dialog answered "where did that
// number come from"; this answers "and what did it do", which used to be a bar that dropped while
// the total was still sitting in the middle of the screen with no connection drawn between them.
//
// Two rules it inherits and one it adds.
//
// **The model has already moved.** `applyEvent` writes the new life the instant the event is
// reached, exactly as it always did, and the flight is raised afterwards — so a figure in the air
// is a ghost of something that has happened, and nothing that asks "how much life is left" gets a
// different answer while it is up. What lags is the *drawing*, through `shownLife`, which is a
// view over the model rather than a second copy of it. This is the same division `spendSelected`
// keeps: the animation never owns the state.
//
// **It cannot change an outcome.** The round was decided before any of it was drawn.
//
// **It can stop the playback cursor**, which the hand dialog was the first thing on this screen
// to do. A figure crossing half the screen does not fit inside one event's dwell, and the
// alternative is the bar dropping before the number reaches it — which is the picture this exists
// to remove. `combatTheatre.running` is what `advancePlayback` waits on.

// The figure's journey and the pause it takes on the card, **as fractions of the one playback
// speed** *(2026-08-19)* — see `beat`. They reproduce the 26 and 18 they were tuned to at a speed
// of 25, and they move with it from here: both stop the playback cursor, so a round watched at
// half the speed would otherwise spend twice the share of itself waiting on a number crossing the
// screen.
var (
	// hitFlyTicks is how long the figure takes to reach the card. Longer than a card's flight
	// because it crosses more screen and because the bar is waiting on it — a hit that arrives
	// before the eye has followed it lands the damage twice as far from its cause as no animation
	// at all would.
	hitFlyTicks = beat(1, 1)

	// hitHoldTicks is how long the figure stays on the card after landing, before it fades. The
	// bar drops at the *start* of this, so there is a beat where the number and the emptier bar
	// are on screen together: that overlap is the causal link, and without it the two read as two
	// separate events.
	hitHoldTicks = beat(7, 10)
)

const (
	// hitFigureSize is the type size of a landing figure, and it is **`mathTotalSize` on purpose,
	// not a size of its own**. The figure is meant to *be* the sum's total continuing its journey:
	// `advancePlayback` clears the box on the same frame this launches, at the same point, and
	// `hitInk` is already the colour the total is drawn in — so matching the size is the last of
	// the four things that make one number appear to set off rather than two numbers to swap.
	// A size of its own here is a figure that visibly is not the total.
	hitFigureSize = mathTotalSize

	// hitFromScale is how big the figure starts, and hitToScale how big it arrives.
	//
	// **It shrinks, where every item in the sum box grows, and the difference is the meaning.** A
	// term flying into the sum comes *toward* the reader, so it grows; the total flying into a card
	// goes *away* into it, so it recedes. Starting at exactly 1 is also what makes the first frame
	// of the flight identical to the last frame of the total it replaces.
	hitFromScale = 1.0
	hitToScale   = 0.72
)

// hitInk is the colour a landing figure is written in.
//
// **The attack red the log's verbs are marked in, asked for rather than restated.** `verbInkFor`
// decides what an attack is coloured in this screen, and a figure that lands damage is the same
// meaning as the verb it marks — so it takes the same answer, and a change to one is a change
// to both. It is deliberately not the screen's old attention yellow: that belonged to the hand,
// and reusing it
// would say the hand fired twice.
func hitInk() color.RGBA { return verbInkFor(combat.CategoryAttack) }

// hitFlight is one damage figure on its way from where the blow was worked out into the card it
// empties.
//
// **It stores no coordinates**, like every other mover on this screen: both ends are recomputed
// every frame from the geometry functions that own them, so the flight survives the row it left
// re-laying out underneath it. What it stores instead is enough to *find* both ends — which side
// acted, who is being hit, and the seat the figure came out of when there was no sum on screen.
type hitFlight struct {
	amount int
	side   combat.Side // who acted, which decides the row a seat is measured in
	target combat.Side // whose card the figure is flying into

	// seat is the acting card's seat on the table when the figure came out of a card rather than
	// out of the sum, and -1 when it came out of the sum. This is `anchorBlow` written down: a
	// scored hand already has its total on screen and the figure travels from there, while a solo
	// attacker has no sum at all and every attack lands its own face damage out of its own card.
	seat int

	// held is the life the target's bar keeps showing until the figure arrives. It is read back
	// out through `shownLife`, which is why the flight can hold the drawing without anything else
	// on the screen holding a stale life total.
	held int

	t travel
}

// arrived reports whether the figure has reached the card, which is the moment the bar drops.
func (h hitFlight) arrived() bool { return h.t.age >= hitFlyTicks }

// tick advances the figure by a frame. **A one-line method rather than the caller reaching for
// `h.t`**, because it is what makes a hitFlight a mover in theatre.go's sense and therefore
// something `advance` can drive.
func (h *hitFlight) tick() { h.t.tick() }

// done reports whether the whole gesture — flight and hold — is over.
func (h hitFlight) done() bool { return h.t.age >= hitFlyTicks+hitHoldTicks }

// noteHit raises the figure for one damage event, after `applyEvent` has already written the new
// life. `held` is the life the target's bar was showing a moment earlier, which is what it goes on
// showing until the figure lands.
//
// **The caller reads that life off the combatant before overwriting it, and it may not be derived
// here** *(2026-08-19)*. It was `e.Life + e.Amount` — what was there, worked back from what is
// left plus what was dealt — and that is right for every blow except the one that ends a duel:
// `e.Life` is clamped at zero, so the arithmetic returns the *size of the blow* instead. A pair of
// Cleaves for 60 on an enemy holding 30 drew 60, so the bar jumped up for the length of a flight
// and then emptied. **Overkill is exactly the case the reconstruction cannot see**, which is why
// it only ever showed on a killing blow.
func (s *CombatScene) noteHit(e combat.Event, held int) {
	if e.Kind != combat.KindDamage || e.Amount <= 0 {
		return
	}

	s.theatre.hits = append(s.theatre.hits, hitFlight{
		amount: e.Amount,
		side:   e.Side,
		target: e.Target,
		seat:   s.blowSeat(e),
		held:   held,
		t:      newTravel(0, hitFlyTicks+hitHoldTicks),
	})
}

// blowSeat resolves `anchorBlow`: where this turn's figure already is.
//
// **The sum line when the turn scored a hand, the acting card's own seat when it did not.** A
// player's turn is one blow read off a hand, so the total is on screen in the sum box and the
// figure should leave from there — anything else would put two different figures on screen for one
// blow. A solo attacker emits no `KindHand` at all, so there is no sum, and the figure has to come
// out of the card that swung.
//
// It returns -1 for the sum, which is what `hitFlight.seat` means by it.
func (s *CombatScene) blowSeat(e combat.Event) int {
	if !s.soloAttacker(e.Side) {
		return -1
	}
	// The card that is lit is the card that is hitting, for a solo attacker — see noteResolved,
	// which seats one card at a time for exactly this reason.
	seats := s.theatre.enemyFiringSeats
	if e.Side == combat.SideA {
		seats = s.theatre.firingSeats
	}
	if len(seats) > 0 {
		return seats[0]
	}
	return -1
}

// shownLife is the life a fighter card should draw, which is not always the life it has.
//
// **A bar waits for the figure aimed at it.** While a hit is in the air the card keeps showing what
// it had before the blow, so the drop and the arrival are one event rather than two — the number
// reaching the card is what empties it. The model is already correct underneath; this is a view.
//
// **The earliest figure still owed is the one that decides**, which is what the walk returns:
// hits are appended in log order, so the first one that has not arrived is the oldest blow the bar
// has yet to show. Everything before it has already landed and is already in the model.
//
// **There is only ever one, and the loop is deliberately written as though there might be several.**
// `advancePlayback` holds the cursor for the whole of a figure's flight and hold, so a second
// KindDamage cannot be reached until the first is finished and dropped. Relying on that here would
// mean this function quietly breaks the day the hold is shortened or a kind starts flying two
// figures — and it would break as a bar showing a life nobody has, which is hard to attribute.
func (s *CombatScene) shownLife(side combat.Side, actual int) int {
	for _, h := range s.theatre.hits {
		if h.target == side && !h.arrived() {
			return h.held
		}
	}
	return actual
}

// drawHits writes every figure at wherever it has got to.
func (s *CombatScene) drawHits(gs *state.GlobalState, screen *ebiten.Image) {
	for _, h := range s.theatre.hits {
		from, ok := s.hitOrigin(gs, h)
		if !ok {
			continue
		}
		to := s.hitTarget(gs, h)

		// Past the flight the figure sits on the card and fades, rather than continuing to move —
		// a number that drifts after landing reads as not having landed.
		p := easeOut(clamp01(float64(h.t.age) / float64(hitFlyTicks)))
		at := image.Pt(
			from.X+int(float64(to.X-from.X)*p),
			from.Y+int(float64(to.Y-from.Y)*p),
		)

		scale := hitFromScale + (hitToScale-hitFromScale)*p
		// Not bold: the sum's own total is not, and the handoff between the two depends on the
		// figure setting off looking exactly like the number it left.
		drawMathText(gs, screen, "-"+strconv.Itoa(h.amount), hitFigureSize, hitInk(),
			at, scale, hitAlpha(h), false)
	}
}

// hitAlpha holds the figure solid for the whole flight and fades it out over the hold.
//
// **It does not fade *in*, and that is the handoff again.** A figure that faded up over its first
// frames would blink where the sum's total had been fully opaque a frame earlier, which is exactly
// the seam the matched size, colour and position exist to remove. What it fades out of is the card
// it landed on, after the bar has already dropped — so the last thing to go is the number, and the
// emptier bar is what is left.
func hitAlpha(h hitFlight) float32 {
	if !h.arrived() {
		return 1
	}
	held := float64(h.t.age-hitFlyTicks) / float64(hitHoldTicks)
	return float32(clamp01(1 - held))
}

// hitOrigin is where a figure sets off from — `anchorBlow`, resolved to a point.
func (s *CombatScene) hitOrigin(gs *state.GlobalState, h hitFlight) (image.Point, bool) {
	if h.seat < 0 {
		r := s.handMathRect(gs)
		return image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2), true
	}

	// **Recomputed from the same seat functions the rows are drawn with**, never cached, so a
	// figure in the air survives the row re-laying out under it — the rule every mover on this
	// screen keeps. `slotAt` and friends give a card's top-left, so the middle is half a card in.
	var at image.Point
	if h.side == combat.SideA {
		if h.seat >= len(s.theatre.resolved) {
			return image.Point{}, false
		}
		at = playedSeatAt(gs, h.seat, len(s.theatre.resolved), s.playedSplit())
	} else {
		if h.seat >= len(s.theatre.enemyDealt) {
			return image.Point{}, false
		}
		at = enemySeatAt(gs, h.seat, len(s.theatre.enemyDealt), s.enemySplit())
	}
	return image.Pt(at.X+cardWidth/2, at.Y+cardHeight/2), true
}

// hitTarget is the middle of the card the figure is flying into — `anchorTargetCard`.
func (s *CombatScene) hitTarget(gs *state.GlobalState, h hitFlight) image.Point {
	r := s.enemyCardRect(gs)
	if h.target == combat.SideA {
		r = s.duelistCardRect(gs)
	}
	return image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
}

// clamp01 holds a progress figure inside its range. The flight's own clock runs past the flight so
// the hold can be measured off it, so the travelled fraction has to be clamped rather than trusted.
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
