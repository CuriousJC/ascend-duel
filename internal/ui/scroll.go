package ui

// scroll.go — the papyrus a card's working is written on.
//
// **One picture stretched to any length.** The art is a top roll, a middle every row of which is the
// same, and a bottom roll, so a single drawing covers a sum of two lines and one that runs off the
// screen: the rolls are drawn at their own height and the middle stretches between them. It unrolls
// by the bottom roll travelling down, the paper appearing behind it.
//
// **The sheet is the card's width**: the art is scaled so its paper, not its rolls, spans the
// rectangle it is given, and the rolls overhang it a little either side as a real scroll's do.
//
// **It may be drawn without its top roll**, the paper then running straight out from under the card
// with only the bottom roll — one of the two shapes the gallery puts side by side.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
)

// The scroll art's three files, lightest first.
const (
	ScrollLight = "scroll-light"
	ScrollMid   = "scroll-mid"
	ScrollDeep  = "scroll-deep"
)

// The art's own measurements, in its pixels: the canvas, where the paper runs across it, and the
// two rolls' bands. Every row between the bands is the same, so the middle stretches.
const (
	scrollArtW      = 512
	scrollArtH      = 1024
	scrollSheetLeft = 32
	scrollSheetW    = 448
	scrollTopRoll   = 128
	scrollBottomAt  = 896
)

// ScrollScale is the art's scale when its paper is w pixels wide.
func ScrollScale(w int) float64 { return float64(w) / scrollSheetW }

// ScrollTopHeight and ScrollBottomHeight are the two rolls' heights when the paper is w wide: what
// the working has to keep clear of at either end.
func ScrollTopHeight(w int) int { return int(float64(scrollTopRoll)*ScrollScale(w) + 0.5) }
func ScrollBottomHeight(w int) int {
	return int(float64(scrollArtH-scrollBottomAt)*ScrollScale(w) + 0.5)
}

// DrawScroll draws one scroll whose paper spans r across and, fully unrolled, fills r down — the
// bottom roll's foot on r.Max.Y. `open` is how far it has unrolled, 0 to 1: at 0 the rolls sit
// together at the top of r. `top` says whether the top roll is drawn. Reports false when the art is
// not filed, so the caller can draw its stand-in.
func DrawScroll(dst *ebiten.Image, key string, r image.Rectangle, open float64, top bool) bool {
	src := systems.ArtMarkImage(key, scrollArtW, scrollArtH)
	if src == nil {
		return false
	}
	s := ScrollScale(r.Dx())
	x := float64(r.Min.X) - scrollSheetLeft*s
	topH, botH := 0.0, float64(scrollArtH-scrollBottomAt)*s
	if top {
		topH = scrollTopRoll * s
	}
	mid := (float64(r.Dy()) - topH - botH) * Clamp01(open)
	if mid < 0 {
		mid = 0
	}

	piece := func(sy0, sy1 int, y, h float64) {
		if h <= 0 {
			return
		}
		sub := src.SubImage(image.Rect(0, sy0, scrollArtW, sy1)).(*ebiten.Image)
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Scale(s, h/float64(sy1-sy0))
		op.GeoM.Translate(x, y)
		dst.DrawImage(sub, op)
	}
	y := float64(r.Min.Y)
	if top {
		piece(0, scrollTopRoll, y, topH)
	}
	piece(scrollTopRoll, scrollBottomAt, y+topH, mid)
	piece(scrollBottomAt, scrollArtH, y+topH+mid, botH)
	return true
}
