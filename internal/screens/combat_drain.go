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

// What life arriving looks like: a drain coming back out of the body it was taken from, and a
// regeneration coming out of the ring that made it. One mover, two sources.
//
// **It is the `KindDrained` row of the theater table** — `anchorTargetCard` to `anchorActorCard`,
// `gestureFly` — and it is deliberately the damage figure's journey run backwards. The blow flies
// into the target's card and empties its bar; a beat later the share flies back out of that same
// card into the duelist's, and fills theirs. One object making a round trip says "this is a
// portion of the hit you just watched" in a way a figure appearing beside the health bar cannot.
//
// **Not out of the relic ring**, which is where a status flies from and was the first thing this
// tried. A relic is the *cause*, and `anchorRelic` is the right source for something the relic
// creates out of nothing — a status lands on a body that had none. Life a drain takes is not
// created: it comes off the blow, so the blow is where the player's eye already is.
//
// **A regeneration flies out of the relic instead** — `anchorRelic` to `anchorActorCard` — and that
// is the same distinction the other way round. A relic makes this life out of nothing, exactly as
// it makes a status out of nothing, so the ring is where it comes from and there is no blow for the
// eye to be looking at. The two share everything else: the color, the size, the clock and the bar
// that waits.
//
// **Smaller and in green, where the blow is full size and red.** Both are deliberate and both are
// about rank: the share is a fraction of the figure it came out of, so it reads wrong at the
// figure's own size, and the green is `ui.PlayerSwatch` — the duelist's own color, which the
// screen already uses for their side of everything. No new hue is claimed.
//
// The three rules it inherits: **the model has already moved**, **it cannot change an outcome**,
// and **it holds the playback cursor** — the bar must not rise before the number reaches it,
// which is `hitFlight`'s whole argument pointing the other way.

// drainFlyTicks() is the figure's journey. The damage figure's own, because it retraces it.
func drainFlyTicks() int { return ui.Beat(1, 1) }

// drainHoldTicks() is the pause on the duelist card after landing, with the bar already fuller.
// The overlap is the causal link, exactly as hitHoldTicks() is.
func drainHoldTicks() int { return ui.Beat(7, 10) }

const (
	// drainFigureSize is the type size of the share. **Below the blow's**, which is
	// `hitFigureSize` — the one place on this screen a traveling figure is deliberately not the
	// sum's total size, because this figure *is* a fraction of that one and a matching size would
	// say the two are the same quantity.
	drainFigureSize = mathTotalSize - 10

	// drainFromScale is how big the share starts and drainToScale how big it arrives. It recedes
	// into the card it lands on, like a blow and unlike a term flying into the sum.
	drainFromScale = 1.0
	drainToScale   = 0.74
)

// drainInk is the color the share is written in: **the duelist's own green**, asked for rather
// than restated, so a change to which color the player's side is drawn in is a change to both.
func drainInk() color.RGBA { return ui.PlayerSwatch }

// drainFlight is one relic's share of a blow, on its way back out of the body it was taken from.
//
// **It stores no coordinates**, like every other mover on this screen: both ends are recomputed
// every frame from the geometry that owns them.
type drainFlight struct {
	amount int
	side   combat.Side // who gained the life, and therefore whose card the figure lands on

	// relic is the ring the figure leaves from, and fromRelic says to use it. A drain leaves the
	// other side's fighter card instead — see the source argument above.
	//
	// **A bool beside the id rather than a sentinel**, because the zero RelicID is a real relic,
	// which is the hazard combat.Event.Relic carries its own note about.
	relic     combat.RelicID
	fromRelic bool

	// held is the life the drainer's bar keeps showing until the figure arrives, read back out
	// through shownLife. Same field, same job and the opposite direction to hitFlight's.
	held int

	t ui.Travel
}

// arrived reports whether the figure has reached the card, which is the moment the bar rises.
func (d drainFlight) arrived() bool { return d.t.Age >= drainFlyTicks() }

func (d *drainFlight) Tick()     { d.t.Tick() }
func (d drainFlight) Done() bool { return d.t.Age >= drainFlyTicks()+drainHoldTicks() }

// noteDrain raises the figure for one drain or one regeneration, after applyEvent has already
// written the new life. `held` is the life the gaining bar was showing a moment earlier, which is
// what it goes on showing until the figure lands.
//
// **One function over both kinds**, because everything about the gesture is shared and only the
// source differs — two would be two clocks and two chances for the pair to drift apart.
func (s *CombatScene) noteDrain(e combat.Event, held int) {
	regen := e.Kind == combat.KindRegenerated
	if e.Kind != combat.KindDrained && !regen {
		return
	}
	if e.Amount <= 0 {
		return
	}

	s.Theater.drains = append(s.Theater.drains, drainFlight{
		amount:    e.Amount,
		side:      e.Side,
		relic:     e.Relic,
		fromRelic: regen,
		held:      held,
		t:         ui.NewTravel(0, drainFlyTicks()+drainHoldTicks()),
	})
}

// drawDrains writes every share at wherever it has got to.
func (s *CombatScene) drawDrains(gs *state.GlobalState, screen *ebiten.Image) {
	for _, d := range s.Theater.drains {
		from, ok := s.drainOrigin(gs, d)
		if !ok {
			continue
		}
		to := s.fighterCardMid(gs, d.side)

		p := ui.EaseOut(ui.Clamp01(float64(d.t.Age) / float64(drainFlyTicks())))
		at := image.Pt(
			from.X+int(float64(to.X-from.X)*p),
			from.Y+int(float64(to.Y-from.Y)*p),
		)

		scale := drainFromScale + (drainToScale-drainFromScale)*p
		drawMathText(gs, screen, "+"+strconv.Itoa(d.amount), drainFigureSize, drainInk(),
			at, scale, drainAlpha(d), false)
	}
}

// drainAlpha holds the figure solid for the flight and fades it over the hold, like hitAlpha.
func drainAlpha(d drainFlight) float32 {
	if !d.arrived() {
		return 1
	}
	held := float64(d.t.Age-drainFlyTicks()) / float64(drainHoldTicks())
	return float32(ui.Clamp01(1 - held))
}

// drainOrigin is where the figure sets off from: the ring that made it, or the body it was taken
// out of.
//
// **A ring the row is no longer drawing raises nothing**, which is hitOrigin's rule — a figure
// leaving a seat that is not there would leave the corner of the screen. It can happen: the relic
// row is reorderable and a scene can be re-entered under a running flight.
func (s *CombatScene) drainOrigin(gs *state.GlobalState, d drainFlight) (image.Point, bool) {
	if !d.fromRelic {
		return s.fighterCardMid(gs, other(d.side)), true
	}

	seat, ok := wornSeatOf(gs, d.relic)
	if !ok {
		return image.Point{}, false
	}
	r := s.relicRow(gs).RowSlot(gs, seat)
	return image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2), true
}

// wornSeatOf is where in the relic row one relic is standing, by the key the rules know it as.
//
// **The run is the authority on worn order**, exactly as wornRelics says — the screen is looking
// a position up, not deciding one.
func wornSeatOf(gs *state.GlobalState, id combat.RelicID) (int, bool) {
	if gs.Run == nil {
		return 0, false
	}
	key := combat.RelicOf(id).Key
	for i, worn := range gs.Run.Worn() {
		if worn == key {
			return i, true
		}
	}
	return 0, false
}

// fighterCardMid is the middle of one side's fighter card — `anchorActorCard` and
// `anchorTargetCard`, which are the same place asked about two different sides.
func (s *CombatScene) fighterCardMid(gs *state.GlobalState, side combat.Side) image.Point {
	r := ui.EnemyCardRect(gs)
	if side == combat.SideA {
		r = ui.DuelistCardRect(gs)
	}
	return image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
}

// other is the side a drain takes its life out of.
func other(s combat.Side) combat.Side {
	if s == combat.SideA {
		return combat.SideB
	}
	return combat.SideA
}

// shownDrain is the life a fighter card is still waiting on, and it is shownLife's own idea: while
// a share is in the air the bar keeps showing what it had before, so the rise and the arrival are
// one event rather than two.
//
// **The earliest share still owed is the one that decides**, exactly as shownLife's walk does.
func (s *CombatScene) shownDrain(side combat.Side) (int, bool) {
	for _, d := range s.Theater.drains {
		if d.side == side && !d.arrived() {
			return d.held, true
		}
	}
	return 0, false
}
