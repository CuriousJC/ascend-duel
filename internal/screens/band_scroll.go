package screens

// The scroll the USE and SELL tabs are written on: the fight's papyrus, hanging from whatever is
// armed on the top row and unrolling down to show the tabs.
//
// **The same picture and the same two passes as a card's working** — see combat_scrolls.go. The
// scroll is drawn once whole on the pane's backing, behind the cards, so its top is tucked up under
// the armed card; and once again over everything, from the card's bottom edge down, so the tabs and
// the paper under them sit over whatever is below the band. **The tabs show only as far as the paper
// has unrolled**, so arming a card reads as a scroll being let down to the question.
//
// **It is a picture of the question and never the question itself**: what is armed, what a click
// on a tab does and what it pays are all sellTab's, exactly as before.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

// tabScrollTuck is how far the tabs' scroll starts up under the armed card.
const tabScrollTuck = 3

// tabScrollPad is the margin of paper beside and below the tabs.
const tabScrollPad = 8

// tabScroll is the scroll's own state: which armed thing it hangs from, and its clocks.
type tabScroll struct {
	// under is the sale the scroll hangs from — the armed thing, or the one it is rolling up from.
	under shopSale

	unroll  ui.Travel
	rolling bool
	roll    ui.Travel
}

// tickScroll keeps the scroll in step with what is armed: a newly armed thing unrolls one, and a
// disarmed one rolls its scroll back up. **Read off `armed` rather than told at each arming**,
// because a handful of places put the question away by writing the field directly.
func (s *sellTab) tickScroll() {
	sc := &s.scroll
	switch {
	case s.armed.any() && s.armed != sc.under:
		*sc = tabScroll{under: s.armed, unroll: ui.NewTravel(0, scrollUnrollTicks())}
	case !s.armed.any() && sc.under.any() && !sc.rolling:
		sc.rolling, sc.roll = true, ui.NewTravel(0, scrollRollTicks())
	}
	sc.unroll.Tick()
	if sc.rolling {
		sc.roll.Tick()
		if sc.roll.Done() {
			*sc = tabScroll{}
		}
	}
}

// scrollOpen is how far the scroll has unrolled, 0 to 1.
func (s *sellTab) scrollOpen() float64 {
	o := ui.EaseOut(s.scroll.unroll.Progress())
	if s.scroll.rolling {
		o *= 1 - ui.EaseOut(s.scroll.roll.Progress())
	}
	return o
}

// scrollRect is the scroll fully unrolled and the seat it hangs from: the card's own width, from
// tabScrollTuck up under the card to below the last tab with room for the bottom roll. The tabs are
// sized to fit inside it — see sellTabWidth. False with nothing to hang from.
func (s *sellTab) scrollRect(gs *state.GlobalState) (image.Rectangle, image.Rectangle, bool) {
	under := s.scroll.under
	if !under.any() {
		return image.Rectangle{}, image.Rectangle{}, false
	}
	seat, ok := s.seatOf(gs, under)
	if !ok {
		return image.Rectangle{}, image.Rectangle{}, false
	}
	tabs := 1
	if under.kind == saleHeld && s.canUse != nil {
		tabs = 2
	}
	w := seat.Dx()
	cx := (seat.Min.X + seat.Max.X) / 2
	last := seat.Max.Y + shopFigureGap + tabs*sellTabHeight + (tabs-1)*shopFigureGap
	r := image.Rect(cx-w/2, seat.Max.Y-tabScrollTuck, cx+w/2,
		last+tabScrollPad+ui.ScrollBottomHeight(w))
	return r, seat, true
}

// paperBottom is how far down the scroll's paper reaches at its current unroll: the tabs are drawn
// above this line and no further.
func (s *sellTab) paperBottom(r image.Rectangle) int {
	mid := r.Dy() - ui.ScrollBottomHeight(r.Dx())
	return r.Min.Y + int(float64(mid)*s.scrollOpen())
}

// drawScrollBehind is the first pass: the whole scroll, on the backing of the pane the armed thing
// sits in and under its cards. A pane draws it between its backing and its cards, passing which
// kind of thing it holds, so the relic pane never draws a carried card's scroll. Nil draws nothing.
func (s *sellTab) drawScrollBehind(gs *state.GlobalState, screen *ebiten.Image, kind saleKind) {
	if s == nil || s.scroll.under.kind != kind {
		return
	}
	if r, _, ok := s.scrollRect(gs); ok {
		ui.DrawScroll(screen, scrollKey, r, s.scrollOpen(), false)
	}
}

// drawScrollOver is the second pass: the scroll from the armed card's bottom edge down, over
// everything below the band.
func (s *sellTab) drawScrollOver(gs *state.GlobalState, screen *ebiten.Image) {
	r, seat, ok := s.scrollRect(gs)
	if !ok {
		return
	}
	b := screen.Bounds()
	below := image.Rect(b.Min.X, seat.Max.Y, b.Max.X, b.Max.Y)
	ui.DrawScroll(screen.SubImage(below).(*ebiten.Image), scrollKey, r, s.scrollOpen(), false)
}

// tabsShowing is the part of the screen the tabs may be drawn in: above the paper's current edge.
func (s *sellTab) tabsShowing(gs *state.GlobalState, screen image.Rectangle) image.Rectangle {
	r, _, ok := s.scrollRect(gs)
	if !ok {
		return screen
	}
	return image.Rect(screen.Min.X, screen.Min.Y, screen.Max.X, s.paperBottom(r))
}
