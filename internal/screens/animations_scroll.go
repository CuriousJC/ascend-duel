package screens

// animations_scroll.go — the gallery's pane options: the pane the working is written on, in five
// steps of tan, each unrolling under a card and taking a short sum, side by side.
//
// **The pane and the sum are the game's own drawing.** The pane is `drawScrollPane` at
// `scrollRect`'s size, the rows are laid out by `layOutLine` and `mathRowHeight` under the card the
// way `layOutMath` lays them, and every figure is `drawMathText` at the line's own sizes and inks —
// so what is chosen here is the ground and nothing else.

import (
	"fmt"
	"image"
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

// scrollOptionPitch is the distance between one option's column and the next.
const scrollOptionPitch = 240

// scrollDeepTan is where the tan ladder ends: a worn buckskin. The ladder starts at the shipped
// scrollGround and walks toward this in even steps.
var scrollDeepTan = color.RGBA{R: 186, G: 146, B: 98, A: 255}

// scrollTans is the five options, each a fifth further from scrollGround toward scrollDeepTan.
func scrollTans() []color.RGBA {
	out := make([]color.RGBA, 5)
	for i := range out {
		t := float64(i+1) / float64(len(out))
		lerp := func(a, b uint8) uint8 { return uint8(float64(a) + (float64(b)-float64(a))*t + 0.5) }
		out[i] = color.RGBA{
			R: lerp(scrollGround.R, scrollDeepTan.R),
			G: lerp(scrollGround.G, scrollDeepTan.G),
			B: lerp(scrollGround.B, scrollDeepTan.B),
			A: 255,
		}
	}
	return out
}

// scrollSampleRows is the sum each option takes: a card's term, two relics, the answer.
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

// drawScrollOptions draws the five panes in a row, each unrolling under the page's card and the
// sum arriving on it one figure at a time, then holding.
func drawScrollOptions(gs *state.GlobalState, screen *ebiten.Image, at image.Rectangle, p float64) {
	const unrollShare, writeEnd = 0.15, 0.8
	face := animFace(gs, animCardB)
	el := ui.ArtFor(combat.Ice)

	for c, ground := range scrollTans() {
		x := at.Min.X + c*scrollOptionPitch
		cx := x + cardWidth/2

		rows := scrollSampleRows(el)
		y := float64(at.Min.Y + cardHeight + scrollPadY)
		var stack []mathItem
		for _, row := range rows {
			h := mathRowHeight(row)
			layOutLine(gs, row, cx, int(y+h/2))
			y += h + mathRowGap
			stack = append(stack, row...)
		}

		ui.BlitCard(gs, screen, image.Pt(x, at.Min.Y), face, cards.Hand)
		drawScrollPane(screen, scrollRect(gs, stack, cx), ui.EaseOut(min(1, p/unrollShare)), ground)

		// The figures arrive in order across the writing window, each fading up over its share.
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

		animText(gs, screen, fmt.Sprintf("%d  #%02X%02X%02X", c+1, ground.R, ground.G, ground.B),
			x+20, at.Min.Y-34, floatLabelSize, ui.GroundInk)
	}
}

// scrollArtKeys is the scroll art's three colors, lightest first: the tans the flat panes above
// number 2, 3 and 4.
var scrollArtKeys = []string{ui.ScrollLight, ui.ScrollMid, ui.ScrollDeep}

// drawScrollArtOptions draws the scroll art in its three colors side by side, each unrolling under
// the page's card and the sum arriving on it, with or without the top roll. **The rows start below
// the top roll and end above the bottom one**, so nothing is written on a roll.
func drawScrollArtOptions(gs *state.GlobalState, screen *ebiten.Image, at image.Rectangle, p float64, top bool) {
	const unrollShare, writeEnd = 0.2, 0.85
	face := animFace(gs, animCardB)
	el := ui.ArtFor(combat.Ice)
	topH, botH := 0, ui.ScrollBottomHeight(cardWidth)
	if top {
		topH = ui.ScrollTopHeight(cardWidth)
	}

	for c, key := range scrollArtKeys {
		x := at.Min.X + c*scrollOptionPitch
		cx := x + cardWidth/2
		cardBottom := at.Min.Y + cardHeight
		sheetTop := cardBottom
		if !top {
			sheetTop -= scrollTuck
		}

		rows := scrollSampleRows(el)
		y := float64(cardBottom + topH + scrollPadY)
		var stack []mathItem
		for _, row := range rows {
			h := mathRowHeight(row)
			layOutLine(gs, row, cx, int(y+h/2))
			y += h + mathRowGap
			stack = append(stack, row...)
		}
		r := image.Rect(x, sheetTop, x+cardWidth, int(y-mathRowGap)+scrollPadY+botH)

		// **The scroll goes down first and the card over it**, so a scroll tucked under the card
		// comes out from behind it.
		ui.DrawScroll(screen, key, r, ui.EaseOut(min(1, p/unrollShare)), top)
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

		animText(gs, screen, fmt.Sprintf("%d  %s", c+2, key), x+20, at.Min.Y-34, floatLabelSize, ui.GroundInk)
	}
}
