package screens

// The discard folding back into the draw pile, made visible.
//
// **When a deal runs the pile dry, the discard is shuffled back in and the deal carries on** —
// `drawHand` does that the moment it happens, so the hand and the piles are right before a frame of
// this is drawn. What this file owns is the picture: the pile empties as its last card leaves, every
// card in the discard flies in face down from the right edge of the screen into the gap, and only
// then does the deal go on out of the refilled pile.
//
// **It may never change an outcome**, the flights' rule. The shuffle has already happened; the
// cards still to be dealt are held back by their own delay and nothing else waits on it.

import (
	"math"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

// returnSpreadTicks is how long the returning cards take to set off, first to last, whatever their
// number — so a discard of forty arrives as a stream rather than as forty beats.
func returnSpreadTicks() int { return ui.Beat(6, 5) }

// returnLift is how high a returning card arcs over the hand on its way to the pile.
const returnLift = 60

// reshuffleDelays fills a deal's returns, if drawHand folded the discard back in, and hands back
// the delay each dealt card sets off at: the deal's own stagger for the cards before the reshuffle,
// and the same stagger after the last returning card has landed for the cards after it.
func (s *CombatScene) reshuffleDelays(d *handDeal, dealt int) func(i int) int {
	stagger := flightStaggerPer()
	at, n := s.reshuffledAt, s.reshuffled
	if at < 0 || at > dealt || n <= 0 {
		return func(i int) int { return i * stagger }
	}

	// **The returns set off on the beat the next card would have left**, so the pile stands empty
	// for one stagger before the first one arrives.
	start := at * stagger
	spread := returnSpreadTicks()
	d.returns = make([]ui.Travel, n)
	for k := range d.returns {
		off := 0
		if n > 1 {
			off = k * spread / (n - 1)
		}
		d.returns[k] = ui.NewTravel(start+off, flightTicks())
	}
	d.emptyAfter = at - 1
	landed := start + spread + flightTicks()

	return func(i int) int {
		if i < at {
			return i * stagger
		}
		return landed + (i-at)*stagger
	}
}

// pileShown is how many cards the pile is drawn as holding.
//
// **During a reshuffle it is the picture's count, not the model's.** The model refilled the pile
// before the deal began, so until the first returning card lands the pile holds only the cards
// before the reshuffle that have not yet left — and none once the last of them has. From the first
// landing on it is the refilled pile plus the cards still waiting to leave it.
func (s *CombatScene) pileShown() int {
	d := s.Theater.deal
	if len(d.returns) == 0 {
		return len(s.deck)
	}

	waiting := func(from, to int) int {
		n := 0
		for i := from; i < to && i < len(d.cards); i++ {
			if d.cards[i].flight.Waiting() {
				n++
			}
		}
		return n
	}
	at := d.emptyAfter + 1
	if !d.returns[0].Done() {
		return waiting(0, at)
	}
	return len(s.deck) + waiting(at, len(d.cards))
}

// drawReturns draws the discard flying back into the pile: face down, in off the right edge of the
// screen along the hand row, arcing over it, and into the pile's place.
func (s *CombatScene) drawReturns(gs *state.GlobalState, screen *ebiten.Image) {
	d := s.Theater.deal
	if len(d.returns) == 0 {
		return
	}
	to := deckStackRect(gs).Min
	from := to
	from.X = gs.ScreenWidth + cardWidth/2
	back := s.backSpec(gs)

	for _, r := range d.returns {
		if r.Waiting() || r.Done() {
			continue
		}
		drawReturnCard(gs, screen, from.X, to.X, to.Y, r.Progress(), back)
	}
}

// drawReturnCard is one returning card at raw progress along its journey.
func drawReturnCard(gs *state.GlobalState, screen *ebiten.Image, fromX, toX, y int, raw float64, back cards.Spec) {
	t := ui.EaseOut(raw)
	x := float64(fromX) + (float64(toX)-float64(fromX))*t
	lift := math.Sin(raw*math.Pi) * returnLift

	var geo ebiten.GeoM
	geo.Translate(x, float64(y)-lift)
	ui.DrawFlyingCard(gs, screen, back, cards.Hand, geo)
}
