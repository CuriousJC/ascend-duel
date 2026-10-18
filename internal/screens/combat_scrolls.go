package screens

// The scroll each card's working is written on: unrolled down from under the card as its first line
// starts, and rolled back up as the lines come down.
//
// **It is the scroll art with no top roll** — `ui.DrawScroll`, in its middle tan — so the paper runs
// out from under the card and only the bottom roll moves. **It starts up under the card**, by
// scrollTuck, and is drawn before the table's cards, so the paper comes out from behind the card's
// rounded corners rather than meeting its bottom edge in a seam. Without the art filed it falls
// back to a flat pane.
//
// **It is sized to the whole stack before anything is shown**, every row of every line the card
// throws, its echoes' included — the wrap's measure — so it unrolls once to the space the working
// is about to fill rather than growing as it fills it.
//
// **It may never change an outcome, and nothing waits on it.** It is not in `Running`: the lines
// arrive on their own clock and the pane catches up under them.

import (
	"image"
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

// scrollTuck is how far the scroll starts up under its card.
const scrollTuck = 24

// scrollKey is which of the scroll art's three tans the fight draws.
const scrollKey = ui.ScrollMid

// scrollPadY is the margin of pane above and below the working. **There is no side margin**: a
// pane is its card's width, hanging from it like a tag, and a line wider than the card runs over its
// edges.
const scrollPadY = 12

// scrollGround is the pane's color: a dingy, handled off-white, darker and warmer than the card
// surface so the two read as paper and card.
var scrollGround = color.RGBA{R: 214, G: 206, B: 188, A: 255}

// scrollUnrollTicks is the pane unrolling; scrollRollTicks is it rolling back up, quicker, since
// what it held has already been read.
func scrollUnrollTicks() int { return ui.Beat(1, 2) }
func scrollRollTicks() int   { return ui.Beat(1, 3) }

// mathScroll is one card's pane: the rectangle it unrolls to, and the clocks of its going down and
// coming back up.
type mathScroll struct {
	full   image.Rectangle
	unroll ui.Travel

	// rolling says the box has come down and the pane is rolling up on roll.
	rolling bool
	roll    ui.Travel
}

func (m *mathScroll) Tick() {
	m.unroll.Tick()
	if m.rolling {
		m.roll.Tick()
	}
}

func (m mathScroll) Done() bool { return m.rolling && m.roll.Done() }

// open is the share of the pane showing right now, 0 to 1.
func (m mathScroll) open() float64 {
	o := ui.EaseOut(m.unroll.Progress())
	if m.rolling {
		o *= 1 - ui.EaseOut(m.roll.Progress())
	}
	return o
}

// raiseScrolls unrolls a pane under each card the first time any of its lines begins.
func (s *CombatScene) raiseScrolls(gs *state.GlobalState) {
	b := &s.Theater.mathBox
	for c := range b.columns {
		col := &b.columns[c]
		if !col.begun || col.scrolled {
			continue
		}
		var stack []mathItem
		for k := range b.columns {
			other := &b.columns[k]
			if other.seat != col.seat {
				continue
			}
			other.scrolled = true
			// **A silent line has no scroll**: a card whose every hit comes to 0 unrolls nothing.
			if !other.silent {
				stack = append(stack, other.items...)
			}
		}
		if len(stack) == 0 {
			continue
		}
		card := s.handCardCenter(gs, b.side, col.seat)
		s.Theater.scrolls = append(s.Theater.scrolls, mathScroll{
			full:   scrollUnder(gs, stack, card),
			unroll: ui.NewTravel(0, scrollUnrollTicks()),
		})
	}
}

// scrollRect is the pane under one card's working: the card's width, centered on its column, and
// the stack's height with a margin above and below.
func scrollRect(gs *state.GlobalState, stack []mathItem, cardX int) image.Rectangle {
	center, _, hh := stackExtent(gs, stack)
	return image.Rect(
		cardX-cardWidth/2, center.Y-int(hh)-scrollPadY,
		cardX+cardWidth/2, center.Y+int(hh)+scrollPadY)
}

// scrollUnder is the scroll hanging from one card: the card's width, starting scrollTuck up under
// its bottom edge, and running down past the working far enough for the bottom roll to sit under
// it.
func scrollUnder(gs *state.GlobalState, stack []mathItem, card image.Point) image.Rectangle {
	center, _, hh := stackExtent(gs, stack)
	return image.Rect(
		card.X-cardWidth/2, card.Y+cardHeight/2-scrollTuck,
		card.X+cardWidth/2, center.Y+int(hh)+scrollPadY+ui.ScrollBottomHeight(cardWidth))
}

// closeScrolls rolls every pane back up, for the box coming down.
func (s *CombatScene) closeScrolls() {
	for i := range s.Theater.scrolls {
		m := &s.Theater.scrolls[i]
		if !m.rolling {
			m.rolling = true
			m.roll = ui.NewTravel(0, scrollRollTicks())
		}
	}
}

// drawScrolls draws every scroll at its unrolled share, hanging from its top edge. **Under the
// table's cards**: see the file comment.
func (s *CombatScene) drawScrolls(screen *ebiten.Image) {
	for _, m := range s.Theater.scrolls {
		drawScroll(screen, m)
	}
}

// drawScrollsBelowCards draws every scroll again, **clipped to below the card it hangs from**, with
// the working over the frame's buttons. The scroll has to go under its card, which the table draws
// long before the frame; this second pass puts everything under the card's bottom edge over the
// buttons there as well, and leaves the card — and the shields heaped on it — on top of the tuck.
func (s *CombatScene) drawScrollsBelowCards(screen *ebiten.Image) {
	for _, m := range s.Theater.scrolls {
		drawScroll(screen.SubImage(m.belowCard(screen.Bounds())).(*ebiten.Image), m)
	}
}

// belowCard is the part of the screen a scroll is drawn over the frame in: everything from the
// bottom edge of the card it hangs from down, the full width across so the rolls' overhang is kept.
func (m mathScroll) belowCard(screen image.Rectangle) image.Rectangle {
	return image.Rect(screen.Min.X, m.full.Min.Y+scrollTuck, screen.Max.X, screen.Max.Y)
}

// drawScroll is one scroll: the art with no top roll, or the flat pane while the art is not filed.
func drawScroll(dst *ebiten.Image, m mathScroll) {
	if !ui.DrawScroll(dst, scrollKey, m.full, m.open(), false) {
		drawScrollPane(dst, m.full, m.open(), scrollGround)
	}
}

// drawScrollPane is one pane, `open` of the way down from its top edge, in a ground. Shared with the
// animation gallery's tan options, so what is compared there is the pane the game draws.
func drawScrollPane(screen *ebiten.Image, full image.Rectangle, open float64, ground color.RGBA) {
	h := int(float64(full.Dy()) * open)
	if h <= 0 {
		return
	}
	systems.BevelRect(screen, full.Min.X, full.Min.Y, full.Dx(), h, systems.PaneBevelWidth, ground, false)
}
