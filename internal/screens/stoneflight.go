package screens

// A rock shower's stones, on their way to the pouch.
//
// **They used to be a receipt and are now a flight** *(owner's call, 2026-09-06)*. Spending a rock
// shower put a panel up listing what it had handed over, which the player then had to dismiss — a
// dialog with one gesture, standing between them and a fight, saying something that had already
// happened. What it was actually owed was a picture of where the stones went.
//
// **This is the cards-fly-they-never-appear rule applied to something leaving the screen.** Every
// other flight in the game lands a card somewhere the player can see it; these have nowhere on this
// screen to land, because the pouch is read at the shop. So they fly to the duelist card — the one
// thing on this screen that *is* the run — and fade out as they arrive, which says "into your
// keeping" rather than "gone".
//
// **It cannot change an outcome.** The stones are in the run's pouch before the first frame of this
// is drawn; `Session.Grant` has already run and `saveRun` follows it. What is in the air is a ghost
// of something that has happened, which is the same division `spendSelected` keeps and the reason
// playback speed and the debug flags are allowed to exist.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
)

// How the stones travel.
const (
	// stoneFlightStagger is the delay between one stone setting off and the next, so a shower of
	// three reads as three things rather than as one sheet moving.
	stoneFlightStagger = 6

	// stoneFadeFrom is how far through the journey a stone starts fading. **Late**, so it is still
	// solid for most of the crossing and the fade reads as an arrival rather than as the flight
	// being drawn at half strength.
	stoneFadeFrom = 0.6
)

// stoneFlightTicks is how long one stone is in the air: a beat, like every other mover on this
// screen. See clock.go — the game has one speed and everything is a fraction of it.
var stoneFlightTicks = beat(1, 1)

// stoneFlight is one stone crossing the screen.
//
// **It carries its own two seats**, unlike the row movers in cardslide.go, which store none and
// recompute both from a row size. There is no row here and no seat to recompute: the launch pad is
// the parasite's seat in a pane whose contents change the moment the parasite is spent, so a flight
// that recomputed its origin would be reading a seat that now holds something else.
type stoneFlight struct {
	travel
	stone    session.Stone
	from, to image.Point
}

// flyStonesToPouch sets a shower off from the parasite that produced it.
//
// **From the seat the parasite was standing in**, which is the card the player just clicked — so
// the stones come out of the thing that made them. The seat is read before the pane redraws itself
// without that parasite in it, which is why the index is passed in rather than looked up.
func (s *CombatScene) flyStonesToPouch(gs *state.GlobalState, seat int, shown []session.Stone) {
	from := consumableSlotRect(s.consumablePaneRect(gs), seat).Min
	to := duelistCardRect(gs).Min

	for i, st := range shown {
		s.stones = append(s.stones, stoneFlight{
			travel: newTravel(i*stoneFlightStagger, stoneFlightTicks),
			stone:  st,
			from:   from,
			to:     to,
		})
	}
}

// updateStoneFlights advances them and drops the ones that have landed.
func (s *CombatScene) updateStoneFlights() {
	kept := s.stones[:0]
	for i := range s.stones {
		s.stones[i].tick()
		if !s.stones[i].done() {
			kept = append(kept, s.stones[i])
		}
	}
	s.stones = kept
}

// drawStoneFlights draws whatever is in the air.
//
// **Drawn late, over everything**, for the reason the dragged card is: a stone crossing the screen
// passes the ring pane, the table and the hand, and one that went behind any of them would read as
// having been dropped rather than carried.
func (s *CombatScene) drawStoneFlights(gs *state.GlobalState, screen *ebiten.Image) {
	for _, f := range s.stones {
		if f.waiting() {
			continue
		}
		at := lerpPoint(f.from, f.to, easeOut(f.progress()))
		drawStoneCardFading(gs, screen, at, f.stone, fadeOnArrival(f.progress()))
	}
}

// fadeOnArrival is a stone's alpha across its journey: solid, then fading into the card it lands on.
func fadeOnArrival(p float64) float64 {
	if p < stoneFadeFrom {
		return 1
	}
	return 1 - (p-stoneFadeFrom)/(1-stoneFadeFrom)
}

// drawStoneCardFading is drawStoneCard with an alpha on it.
//
// **The card is built by `internal/cards` exactly as a seated one is** and only the blit differs, so
// a stone in the air is the same picture as a stone on a shelf rather than a second drawing of one.
func drawStoneCardFading(gs *state.GlobalState, screen *ebiten.Image, at image.Point,
	st session.Stone, alpha float64) {

	img := cardImage(gs, stoneSpec(gs, st, true), cards.WormStyle)
	if img == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(at.X), float64(at.Y))
	op.ColorScale.ScaleAlpha(float32(alpha))
	screen.DrawImage(img, op)
}
