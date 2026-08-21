package screens

import (
	"image"
	"image/color"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// What a Prepare looks like banking: the points flying out of the card that banked them and into
// the fighter card whose budget they raise, and that card's AP figure going up as they land.
//
// **It is the `KindGathered` row of the theatre table** — `anchorActorSeat` to `anchorActorCard`,
// `gestureFly` — and it is the damage figure's argument applied to the one card in the game whose
// whole effect is a number changing somewhere else. Until this, a Prepare resolved with a lift in
// its seat, a sentence in the log nobody had open, and a budget that silently read two higher at
// the start of the next round. Everything the card did happened off screen.
//
// **The target moved from the strip's AP figure to the fighter card** *(2026-08-19, owner's call)*,
// and the theatre table moved with it. `3/6 AP` under the bar is *this* round's budget being spent,
// and a Prepare does not touch it — what a Prepare changes is the AP line on the fighter card,
// which is the live budget with `BonusAP` in it. A figure landing on the strip would have been a
// number arriving at a total that did not move.
//
// Three things it inherits from combat_hits.go, and they are the rules to keep:
//
// **The model has already moved.** `applyEvent` is where the engine's event is adopted and the
// flight is raised after it, so a figure in the air is a ghost of something that has happened.
// What lags is the *drawing* — `shownBank`, a view over the round's own arithmetic and never a
// second copy of it.
//
// **It stops the playback cursor.** A figure crossing the screen does not fit inside one event's
// dwell, and the alternative is the card's AP figure changing before the number reaches it. It
// changes pacing and cannot change an outcome.
//
// **It cannot change an outcome.** The round was decided before a frame of it was drawn, and what
// the engine banks is `GatheredAP` either way.

// The journey and the pause, as fractions of the one playback speed — see `beat`. They are the
// damage figure's, deliberately: the two gestures are the same shape in opposite directions, and
// two clocks for one shape is two numbers to keep in step.
var (
	bankFlyTicks  = hitFlyTicks
	bankHoldTicks = hitHoldTicks
)

const (
	// bankFigureSize is the type size of a banked figure. **A term's size rather than a total's**:
	// what lands here is one card's contribution to a budget, which is the same kind of quantity a
	// card's figure in the sum is, and the sum's total size would make a Prepare read as loud as a
	// blow.
	bankFigureSize = mathTermSize

	// bankFromScale is how big the figure sets off and bankToScale how big it arrives.
	//
	// **It grows, where a damage figure shrinks, and the difference is the meaning.** A total
	// flying into a card goes *away* into the thing it empties; banked points are being *added* to
	// the fighter, so they arrive at full size on the card they are joining. It is `gestureFly`'s
	// own description — travels and grows into place — where the hit is the documented exception.
	bankFromScale = 0.55
	bankToScale   = 1.0
)

// bankInk is the colour a banked figure is written in: **the AP bar's blue**, because that is
// already what action points are coloured on this screen. The rule the sum box holds — every
// number is drawn in the colour of what produced it — reads here as the colour of what it *is*: a
// Prepare has no element and the figure is not damage, so the budget's own colour is the one thing
// it can wear that says what kind of number it is.
func bankInk() color.RGBA { return apBarColor }

// bankFlight is one Prepare's points on their way from the card that banked them into the fighter
// card whose budget they raise.
//
// **It stores no coordinates**, like every other mover on this screen: both ends are recomputed
// every frame from the geometry functions that own them, so the flight survives the row it left
// re-laying out underneath it.
type bankFlight struct {
	amount int
	side   combat.Side // who banked, which decides both the row the seat is in and the card it flies to

	// seat is the acting card's seat on the table. -1 when the seat cannot be found, which draws
	// nothing rather than flying out of the corner of the screen.
	seat int

	t travel
}

// arrived reports whether the figure has reached the card, which is the moment the AP figure moves.
func (b bankFlight) arrived() bool { return b.t.age >= bankFlyTicks }

// tick advances the figure by a frame. See hitFlight.tick for why it is a method.
func (b *bankFlight) tick() { b.t.tick() }

// done reports whether the whole gesture — flight and hold — is over.
func (b bankFlight) done() bool { return b.t.age >= bankFlyTicks+bankHoldTicks }

// noteBank raises the figure for one banked event.
//
// **The seat is read now rather than at draw time**, exactly as `noteHit` reads the life now: the
// card that banked is the card currently lit, and by the time the figure lands the lit set has
// moved on to whatever resolved next.
func (s *CombatScene) noteBank(e combat.Event) {
	if e.Kind != combat.KindGathered || e.Amount <= 0 {
		return
	}

	s.theatre.banks = append(s.theatre.banks, bankFlight{
		amount: e.Amount,
		side:   e.Side,
		seat:   s.actingSeat(e.Side),
		t:      newTravel(0, bankFlyTicks+bankHoldTicks),
	})
}

// actingSeat is the seat of the card currently lit on the acting side's row — `anchorActorSeat`.
//
// **One card is lit for a prepare whatever else the turn holds**: `noteResolved` seats a prepare
// or a defend on its own, and only an attack hand raises several at once. A prepare is never part
// of that set, so the first lit seat is the card that banked.
func (s *CombatScene) actingSeat(side combat.Side) int {
	seats := s.theatre.enemyFiringSeats
	if side == combat.SideA {
		seats = s.theatre.firingSeats
	}
	if len(seats) > 0 {
		return seats[0]
	}
	return -1
}

// shownBank is what to add to a side's AP figure for points banked this round and already landed.
//
// **The card shows what the budget will be, from the beat the points arrive.** `Duelist.BonusAP`
// is not written until the round's end state is adopted, so without this the figure moves several
// seconds after the card that moved it, with a whole opposing turn in between — which is what made
// a Prepare feel like it did nothing. The engine's arithmetic is untouched: `GatheredAP` is
// already correct underneath and becomes `BonusAP` on the same schedule it always did.
//
// **It is zeroed when that adoption happens**, in `endOfRound`, or the same two points would be
// counted twice — once by this and once by the `BonusAP` the adoption brings.
func (s *CombatScene) shownBank(side combat.Side) int {
	if side < 0 || int(side) >= len(s.theatre.bankShown) {
		return 0
	}
	return s.theatre.bankShown[side]
}

// drawBanks writes every figure at wherever it has got to.
func (s *CombatScene) drawBanks(gs *state.GlobalState, screen *ebiten.Image) {
	for _, b := range s.theatre.banks {
		from, ok := s.bankOrigin(gs, b)
		if !ok {
			continue
		}
		to := s.bankTarget(gs, b)

		// Past the flight the figure sits on the card and fades rather than continuing to move: a
		// number that drifts after landing reads as not having landed.
		p := easeOut(clamp01(float64(b.t.age) / float64(bankFlyTicks)))
		at := image.Pt(
			from.X+int(float64(to.X-from.X)*p),
			from.Y+int(float64(to.Y-from.Y)*p),
		)

		scale := bankFromScale + (bankToScale-bankFromScale)*p
		drawMathText(gs, screen, "+"+strconv.Itoa(b.amount)+" AP", bankFigureSize, bankInk(),
			at, scale, bankAlpha(b), false)
	}
}

// bankAlpha fades the figure out over its hold, so it leaves the card rather than being deleted
// from it. Solid for the whole journey — a figure that is still arriving is not yet history.
func bankAlpha(b bankFlight) float32 {
	if !b.arrived() {
		return 1
	}
	held := float64(b.t.age-bankFlyTicks) / float64(bankHoldTicks)
	return float32(1 - clamp01(held))
}

// bankOrigin is the middle of the card that banked the points, in whichever row it belongs to.
func (s *CombatScene) bankOrigin(gs *state.GlobalState, b bankFlight) (image.Point, bool) {
	if b.seat < 0 {
		return image.Point{}, false
	}
	if b.side == combat.SideB {
		if b.seat >= len(s.theatre.enemyDealt) {
			return image.Point{}, false
		}
		at := lift(enemySeatAt(gs, b.seat, len(s.theatre.enemyDealt), s.enemySplit()), true)
		return image.Pt(at.X+cardWidth/2, at.Y+cardHeight/2), true
	}
	if b.seat >= len(s.theatre.resolved) {
		return image.Point{}, false
	}
	at := lift(playedSeatAt(gs, b.seat, len(s.theatre.resolved), s.playedSplit()), true)
	return image.Pt(at.X+cardWidth/2, at.Y+cardHeight/2), true
}

// bankTarget is the AP line on the banking fighter's own card — the figure the points change.
//
// **The row rather than the card's centre**, because the gesture is only worth drawing if it ends
// on the number it moves. `cards.DuelistStyle` writes its stat rows from `StatsTop` at
// `StatRowPitch`, and AP is the second of them; a style with no stat rows at all — the enemy's,
// which carries a portrait — falls back to the middle of the card, since there is no figure there
// for the points to land on.
func (s *CombatScene) bankTarget(gs *state.GlobalState, b bankFlight) image.Point {
	r, style := s.duelistCardRect(gs), cards.DuelistStyle
	if b.side == combat.SideB {
		r, style = s.enemyCardRect(gs), cards.EnemyStyle
	}

	if style.StatRowPitch <= 0 {
		return image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
	}
	return image.Pt((r.Min.X+r.Max.X)/2,
		r.Min.Y+style.StatsTop+style.StatRowPitch*bankStatRow+style.StatRowPitch/2)
}

// bankStatRow is which stat line the AP figure is: `duelistSpec` fills DMG, AP, Vitae in that
// order. It is written here rather than searched for by label because the spec's rows are a fixed
// array in a known order, and a lookup by string would be a second place the order is asserted.
const bankStatRow = 1
