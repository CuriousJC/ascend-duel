package screens

import (
	"image"
	"image/color"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
)

// What a hit looks like landing: the figure traveling out of wherever it was worked out, into the
// card whose bar it empties, and the bar waiting for it to arrive.
//
// **It is the `KindDamage` row of the theater table** — `anchorBlow` to `anchorTargetCard`,
// `gestureFly` — and it is the second half of the hand dialog. The dialog answers "where did that
// number come from"; this answers "and what did it do".
//
// **Several can be in the air at once.** A hand-forming turn works every hit out in parallel and
// throws each the moment its line finishes, so five figures leave five lines at five different
// moments and the bar drops as each one arrives.
//
// **The model has already moved.** The life is written before the figure is raised — by
// `applyEvent` for a solo attacker's hit, by `throwColumn` for a hand's — so a figure in the air is
// a ghost of something that has happened, and nothing that asks "how much life is left" gets a
// different answer while it is up. What lags is the *drawing*, through `shownLife`, which is a view
// over the model rather than a second copy of it.
//
// **It cannot change an outcome.** The round was decided before any of it was drawn.
//
// **It can stop the playback cursor**, which the hand dialog was the first thing on this screen
// to do. A figure crossing half the screen does not fit inside one event's dwell, and the
// alternative is the bar dropping before the number reaches it — which is the picture this exists
// to remove. `combatTheater.running` is what `advancePlayback` waits on.

// The figure's journey and the pause it takes on the card, **as fractions of the one playback
// speed** *(2026-08-19)* — see `beat`. They reproduce the 26 and 18 they were tuned to at a speed
// of 25, and they move with it from here: both stop the playback cursor, so a round watched at
// half the speed would otherwise spend twice the share of itself waiting on a number crossing the
// screen.
// hitFlyTicks() is how long the figure takes to reach the card. Longer than a card's flight
// because it crosses more screen and because the bar is waiting on it — a hit that arrives
// before the eye has followed it lands the damage twice as far from its cause as no animation
// at all would.
func hitFlyTicks() int { return ui.Beat(1, 1) }

// hitHoldTicks() is how long the figure stays on the card after landing, before it fades. The
// bar drops at the *start* of this, so there is a beat where the number and the emptier bar
// are on screen together: that overlap is the causal link, and without it the two read as two
// separate events.
func hitHoldTicks() int { return ui.Beat(7, 10) }

const (
	// hitFigureSize is the type size of a landing figure, and it is **`mathTotalSize` on purpose,
	// not a size of its own**. The figure is meant to *be* the hit's total continuing its journey:
	// the line stops drawing its total on the frame this launches, from the same point, and
	// `hitInk` is already the color the total is drawn in — so matching the size is the last of the
	// four things that make one number appear to set off rather than two numbers to swap.
	hitFigureSize = mathTotalSize

	// hitFromScale is how big the figure starts, and hitToScale how big it arrives.
	//
	// **It shrinks, where every item in a hit's line grows, and the difference is the meaning.** A
	// term flying into a line comes *toward* the reader, so it grows; the figure flying into a card
	// goes *away* into it, so it recedes. Starting at exactly 1 is also what makes the first frame
	// of the flight identical to the last frame of the total it replaces.
	hitFromScale = 1.0
	hitToScale   = 0.72
)

// hitInk is the color a landing figure is written in.
//
// **The attack red the log's verbs are marked in, asked for rather than restated.** `verbInkFor`
// decides what an attack is colored in this screen, and a figure that lands damage is the same
// meaning as the verb it marks — so it takes the same answer, and a change to one is a change
// to both. It is deliberately not the screen's old attention yellow: that belonged to the hand,
// and reusing it
// would say the hand fired twice.
func hitInk() color.RGBA { return ui.VerbInkFor(combat.CategoryAttack) }

// hitFlight is one damage figure on its way from where the hit was worked out into the card it
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

	// seat is the acting card's seat on the table when the figure came out of a card, and -1 when it
	// came out of a hit's line. This is `anchorBlow` written down.
	seat int

	// from is where a figure thrown by a line sets off: that line's total. **A point rather than a
	// seat**, and the one mover here that stores one, because a line is laid out once when the box
	// starts and never moves under it.
	from image.Point

	// held is the life the target's bar keeps showing until the figure arrives. It is read back
	// out through `shownLife`, which is why the flight can hold the drawing without anything else
	// on the screen holding a stale life total.
	held int

	t ui.Travel
}

// arrived reports whether the figure has reached the card, which is the moment the bar drops.
func (h hitFlight) arrived() bool { return h.t.Age >= hitFlyTicks() }

// tick advances the figure by a frame. **A one-line method rather than the caller reaching for
// `h.t`**, because it is what makes a hitFlight a mover in theater.go's sense and therefore
// something `advance` can drive.
func (h *hitFlight) Tick() { h.t.Tick() }

// done reports whether the whole gesture — flight and hold — is over.
func (h hitFlight) Done() bool { return h.t.Age >= hitFlyTicks()+hitHoldTicks() }

// noteHit raises the figure for one solo attacker's damage event, after `applyEvent` has already
// written the new life. `held` is the life the target's bar was showing a moment earlier, which is
// what it goes on showing until the figure lands.
//
// **The caller reads that life off the combatant before overwriting it, and it may not be derived
// here.** `e.Life` is clamped at zero, so what was there worked back from what is left plus what
// was dealt is the *size of the hit* on a killing blow, and the bar would jump up for the length of
// a flight before emptying.
//
// **A hand-forming side's hits are not flown from here**: their lines throw them. See throwColumn.
func (s *CombatScene) noteHit(e combat.Event, held int) {
	if e.Kind != combat.KindDamage || e.Amount <= 0 || !s.soloAttacker(e.Side) {
		return
	}

	s.Theater.hits = append(s.Theater.hits, hitFlight{
		amount: e.Amount,
		side:   e.Side,
		target: e.Target,
		seat:   s.blowSeat(e),
		held:   held,
		t:      ui.NewTravel(0, hitFlyTicks()+hitHoldTicks()),
	})
}

// throwColumn hands one finished line of the hand dialog on to what became of its hit, read off the
// hit's own event further down the log.
//
//   - **A hit that landed flies**, out of the line's total and into the target's card — unless it
//     was worth nothing, which a defense's usually is, and then its line simply stays. The target's
//     life moves here, when the figure sets off, rather than when playback reaches the event — the
//     lines finish in whatever order their arithmetic allows, so the event is reached later and is
//     walked past. `shownLife` holds the bar until the figure arrives.
//   - **A miss says MISS and a block says BLOCKED** over the line. A miss is walked past like a
//     landing; a block is left for playback, because the shield row is moved by it.
//   - **A hit that was never thrown fades where it stands**: the target fell to an earlier one.
//
// **It decides nothing.** Every figure and every verdict is the resolver's.
func (s *CombatScene) throwColumn(c int) {
	box := &s.Theater.mathBox
	col := &box.columns[c]
	col.thrown = true

	if col.logAt < 0 || col.logAt >= len(s.log) {
		col.unthrown = true
		return
	}
	e := s.log[col.logAt]

	switch e.Kind {
	case combat.KindMissed:
		col.verdict = "MISS"
		col.verdictT = ui.NewTravel(0, mathSymbolTicks())
		s.markShown(col.logAt)

	case combat.KindFizzled:
		col.verdict = "FIZZLE"
		col.verdictT = ui.NewTravel(0, mathSymbolTicks())
		s.markShown(col.logAt)

	case combat.KindBlocked:
		col.verdict = "BLOCKED"
		col.verdictT = ui.NewTravel(0, mathSymbolTicks())

	case combat.KindDamage:
		// **A hit of nothing flies nothing**: a defense's hit lands its statuses and moves no bar, so
		// its line keeps its `= 0` and the event is walked past like any other shown hit.
		if e.Amount <= 0 {
			s.markShown(col.logAt)
			return
		}
		target := s.enemy
		if e.Target == combat.SideA {
			target = s.fighter
		}
		held := target.CurrentLife
		target.CurrentLife = max(0, held-e.Amount)
		s.markShown(col.logAt)

		col.spent = true
		s.Theater.hits = append(s.Theater.hits, hitFlight{
			amount: e.Amount,
			side:   e.Side,
			target: e.Target,
			seat:   -1,
			from:   col.total().at,
			held:   held,
			t:      ui.NewTravel(0, hitFlyTicks()+hitHoldTicks()),
		})
	}
}

// markShown records that playback need not show one event of this round's log again.
func (s *CombatScene) markShown(at int) {
	if s.Theater.walked == nil {
		s.Theater.walked = map[int]bool{}
	}
	s.Theater.walked[at] = true
}

// blowSeat resolves `anchorBlow` for a solo attacker: the card that is lit is the card that is
// hitting — see noteResolved, which seats one card at a time for exactly this reason.
func (s *CombatScene) blowSeat(e combat.Event) int {
	seats := s.Theater.enemyFiringSeats
	if e.Side == combat.SideA {
		seats = s.Theater.firingSeats
	}
	if len(seats) > 0 {
		return seats[0]
	}
	return -1
}

// shownLife is the life a fighter card should draw, which is not always the life it has.
//
// **A bar waits for the figures aimed at it.** While hits are in the air the card keeps showing what
// it had before them, less whatever has already arrived, so the drop and the arrival are one event
// rather than two — each number reaching the card is what empties that much of it. The model is
// already correct underneath; this is a view.
//
// **The life shown is the model's plus every figure still owed, and never more than the most any of
// them was holding.** Figures in parallel land in any order, so the sum of what is still in the air
// is what the bar has yet to lose; the cap is what keeps a killing hit's overkill from drawing a bar
// above the life that was there.
//
// **A rider's grant is added to whichever answer comes back.** Life a heal or a golden card put on
// is not in `actual` yet — the screen's copy of the duelist does not take it up until `endOfRound`
// — so it rides on top of both branches, and it is zero until the figure that carries it has
// landed. See signalShown, which is this idea pointing the other way.
func (s *CombatScene) shownLife(side combat.Side, actual int) int {
	granted := s.Theater.shownFor(side).life
	owed, top := 0, -1
	for _, h := range s.Theater.hits {
		if h.target == side && !h.arrived() {
			owed += h.amount
			top = max(top, h.held)
		}
	}
	if top >= 0 {
		return min(actual+owed, top) + granted
	}
	// **A drain holds the bar the same way a hit does, in the other direction.** The life is
	// already on the model; what waits is the rise, so the bar fills as the figure lands rather
	// than a beat before it sets off. See combat_drain.go.
	if held, waiting := s.shownDrain(side); waiting {
		return held + granted
	}
	return actual + granted
}

// drawHits writes every figure at wherever it has got to.
func (s *CombatScene) drawHits(gs *state.GlobalState, screen *ebiten.Image) {
	for _, h := range s.Theater.hits {
		from, ok := s.hitOrigin(gs, h)
		if !ok {
			continue
		}
		to := s.hitTarget(gs, h)

		// Past the flight the figure sits on the card and fades, rather than continuing to move —
		// a number that drifts after landing reads as not having landed.
		p := ui.EaseOut(ui.Clamp01(float64(h.t.Age) / float64(hitFlyTicks())))
		at := image.Pt(
			from.X+int(float64(to.X-from.X)*p),
			from.Y+int(float64(to.Y-from.Y)*p),
		)

		scale := hitFromScale + (hitToScale-hitFromScale)*p
		// Not bold: a line's own total is not, and the handoff between the two depends on the
		// figure setting off looking exactly like the number it left.
		drawMathText(gs, screen, "-"+strconv.Itoa(h.amount), hitFigureSize, hitInk(),
			at, scale, hitAlpha(h), false)
	}
}

// hitAlpha holds the figure solid for the whole flight and fades it out over the hold.
//
// **It does not fade *in*, and that is the handoff again.** A figure that faded up over its first
// frames would blink where the line's total had been fully opaque a frame earlier, which is exactly
// the seam the matched size, color and position exist to remove. What it fades out of is the card
// it landed on, after the bar has already dropped — so the last thing to go is the number, and the
// emptier bar is what is left.
func hitAlpha(h hitFlight) float32 {
	if !h.arrived() {
		return 1
	}
	held := float64(h.t.Age-hitFlyTicks()) / float64(hitHoldTicks())
	return float32(ui.Clamp01(1 - held))
}

// hitOrigin is where a figure sets off from — `anchorBlow`, resolved to a point.
func (s *CombatScene) hitOrigin(gs *state.GlobalState, h hitFlight) (image.Point, bool) {
	if h.seat < 0 {
		return h.from, true
	}

	// **Recomputed from the same seat functions the rows are drawn with**, never cached, so a
	// figure in the air survives the row re-laying out under it — the rule every mover on this
	// screen keeps. `slotAt` and friends give a card's top-left, so the middle is half a card in.
	var at image.Point
	if h.side == combat.SideA {
		if h.seat >= len(s.Theater.resolved) {
			return image.Point{}, false
		}
		at = playedSeatAt(gs, h.seat, len(s.Theater.resolved), s.playedSplit())
	} else {
		if h.seat >= len(s.Theater.enemyDealt) {
			return image.Point{}, false
		}
		at = enemySeatAt(gs, h.seat, len(s.Theater.enemyDealt), s.enemySplit())
	}
	return image.Pt(at.X+cardWidth/2, at.Y+cardHeight/2), true
}

// hitTarget is the middle of the card the figure is flying into — `anchorTargetCard`.
func (s *CombatScene) hitTarget(gs *state.GlobalState, h hitFlight) image.Point {
	r := ui.EnemyCardRect(gs)
	if h.target == combat.SideA {
		r = ui.DuelistCardRect(gs)
	}
	return image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
}
