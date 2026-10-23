package screens

// animations_scroll.go — the gallery's pane entry: the scroll the working is written on, unrolling
// out from under a card and taking a short sum.
//
// **The scroll and the sum are the game's own drawing.** The scroll is `ui.DrawScroll` in
// `scrollKey`, the rows are laid out by `layOutLine` and `mathRowHeight` under the card the way
// `layOutMath` lays them, and every figure is `drawMathText` at the line's own sizes and inks.

import (
	"image"
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

// scrollSampleRows is the sum the scroll takes: a card's term, two relics, the answer.
func scrollSampleRows(el cards.Element) [][]mathItem {
	fig := func(s string, size float64, ink color.RGBA) mathItem {
		return mathItem{text: s, size: size, tint: ink}
	}
	op := func(s string) mathItem { return mathItem{text: s, size: mathSymbolSize, tint: mathOperatorInk()} }
	card := hitInkFor(el)
	return [][]mathItem{
		{fig("10", mathTermSize, card), op("x"), fig("0.5", mathTermSize, card)},
		{op("x"), fig("4", mathGrowthSize, ui.BoostInk)},
		{op("x"), fig("2", mathGrowthSize, ui.BoostInk)},
		{op("="), fig("40", mathTotalSize, card)},
	}
}

// drawScrollArt draws the scroll unrolling under the page's card, as the combat screen draws it —
// no top roll, tucked under the card — and the sum arriving on it. **The rows end above the bottom
// roll**, so nothing is written on it.
func drawScrollArt(gs *state.GlobalState, screen *ebiten.Image, at image.Rectangle, p float64) {
	const unrollShare, writeEnd = 0.2, 0.85
	face := animFace(gs, animCardB)
	el := ui.ArtFor(combat.Ice)
	botH := ui.ScrollBottomHeight(cardWidth)

	x := at.Min.X
	cx := x + cardWidth/2
	cardBottom := at.Min.Y + cardHeight

	rows := scrollSampleRows(el)
	y := float64(cardBottom + scrollPadY)
	var stack []mathItem
	for _, row := range rows {
		h := mathRowHeight(row)
		layOutLine(gs, row, cx, int(y+h/2))
		y += h + mathRowGap
		stack = append(stack, row...)
	}
	r := image.Rect(x, cardBottom-scrollTuck, x+cardWidth, int(y-mathRowGap)+scrollPadY+botH)

	// **The scroll goes down first and the card over it**, so the scroll comes out from behind it.
	ui.DrawScroll(screen, scrollKey, r, ui.EaseOut(min(1, p/unrollShare)), false)
	ui.BlitCard(gs, screen, image.Pt(x, at.Min.Y), face, cards.Hand)

	n := len(stack)
	for i, it := range stack {
		start := unrollShare + (writeEnd-unrollShare)*float64(i)/float64(n)
		share := (writeEnd - unrollShare) / float64(n)
		a := min(1, max(0, (p-start)/share))
		if a <= 0 {
			continue
		}
		drawMathText(gs, screen, it.text, it.size, it.tint, it.at, 1, float32(a), false)
	}
}
