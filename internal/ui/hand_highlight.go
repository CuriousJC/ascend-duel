package ui

// hand_highlight.go — the plate the cards of a hand sit on.
//
// **One picture stretched to any run of cards.** A hand's cards are not always side by side — a
// Four of a Kind with a dead card in the middle is two pairs of neighbours — so a run of one card, of
// two, of five is one plate each, and two runs are two plates. The picture is two end caps and a
// middle that stretches, the bar cell's shape, so a single drawing covers every width.
//
// **It is authored art, and until it is filed a plain plate stands in**: a flat light fill with a
// dark edge, which is the backlog drawing every authored piece replaces. See
// docs/art/hand_highlight_art_prompt.MD.

import (
	"image"
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// The plate's art: its key, the size it is authored at, and the width of each end cap in its own
// pixels. Every column between the caps is the same, so the middle stretches.
const (
	HandHighlightKey = "hand-highlight"
	handHighlightW   = 512
	handHighlightH   = 256
	handHighlightCap = 96
)

// The stand-in plate, drawn while no art is filed.
var (
	handHighlightFill = color.RGBA{R: 0xF2, G: 0xF3, B: 0xF5, A: 0xFF}
	handHighlightEdge = color.RGBA{R: 0x15, G: 0x17, B: 0x1C, A: 0xFF}
)

const handHighlightEdgeWidth = 4

// DrawHandHighlight draws one plate filling r, at an alpha.
func DrawHandHighlight(dst *ebiten.Image, r image.Rectangle, alpha float32) {
	if r.Dx() <= 0 || r.Dy() <= 0 || alpha <= 0 {
		return
	}
	src := systems.ArtMarkImage(HandHighlightKey, handHighlightW, handHighlightH)
	if src == nil {
		drawStandInHighlight(dst, r, alpha)
		return
	}

	capW := handHighlightCap * r.Dy() / handHighlightH
	if 2*capW > r.Dx() {
		capW = r.Dx() / 2
	}
	mid := r.Dx() - 2*capW
	piece := func(sx0, sx1, dx, dw int) {
		if dw <= 0 || sx1 <= sx0 {
			return
		}
		sub := src.SubImage(image.Rect(sx0, 0, sx1, handHighlightH)).(*ebiten.Image)
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Scale(float64(dw)/float64(sx1-sx0), float64(r.Dy())/handHighlightH)
		op.GeoM.Translate(float64(dx), float64(r.Min.Y))
		op.ColorScale.ScaleAlpha(alpha)
		dst.DrawImage(sub, op)
	}
	piece(0, handHighlightCap, r.Min.X, capW)
	piece(handHighlightCap, handHighlightW-handHighlightCap, r.Min.X+capW, mid)
	piece(handHighlightW-handHighlightCap, handHighlightW, r.Min.X+capW+mid, capW)
}

// drawStandInHighlight is the plate until its art is filed: a light fill with a dark edge.
func drawStandInHighlight(dst *ebiten.Image, r image.Rectangle, alpha float32) {
	x, y, w, h := float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy())
	fill := handHighlightFill
	fill.A = uint8(float32(fill.A) * alpha)
	edge := handHighlightEdge
	edge.A = uint8(float32(edge.A) * alpha)
	vector.FillRect(dst, x, y, w, h, premultiply(fill), true)
	vector.StrokeRect(dst, x, y, w, h, handHighlightEdgeWidth, premultiply(edge), true)
}

// premultiply scales a straight-alpha color's channels by its alpha, which is what Ebitengine's
// color.RGBA expects.
func premultiply(c color.RGBA) color.RGBA {
	a := uint32(c.A)
	return color.RGBA{R: uint8(uint32(c.R) * a / 255), G: uint8(uint32(c.G) * a / 255),
		B: uint8(uint32(c.B) * a / 255), A: c.A}
}
