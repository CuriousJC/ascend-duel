package screens

// animations_shields.go — the gallery's two shield gestures: a heap forming over the card that
// raised it, one to five in every element, and a shield thrown onto an attack it ate.

import (
	"fmt"
	"image"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

// The heap page's shape: five Block cards side by side, one more shield on each.
const (
	heapMax         = 5
	heapColumnPitch = 236
)

// heapElements is the element each column is drawn in, so the page shows every color.
var heapElements = [heapMax]combat.Element{
	combat.Fire, combat.Ice, combat.Lightning, combat.Earth, combat.Arcane,
}

// heapGalleryTicks is a heap of five formed, and a beat to look at it.
func heapGalleryTicks() int { return ui.HeapFormTicks(heapMax) + ui.Beat(2, 1) }

// drawHeapGallery forms one to five shields over Block cards, a column per count.
func drawHeapGallery(gs *state.GlobalState, screen *ebiten.Image, origin image.Point, age int) {
	for n := 1; n <= heapMax; n++ {
		el := heapElements[n-1]
		at := image.Pt(origin.X+(n-1)*heapColumnPitch, origin.Y)
		ui.BlitCard(gs, screen, at, animFace(gs, combat.Card{Concept: combat.Block, Element: el}), cards.Hand)

		cx, cy := float64(at.X)+cardWidth/2, float64(at.Y)+cardHeight/2
		for i := 0; i < n; i++ {
			formed := ui.ShieldFormed(age, i)
			if formed <= 0 {
				continue
			}
			ui.DrawShield(screen, ui.ArtFor(el), ui.HeapPose(cx, cy, i, n, 1).Forming(formed))
		}
		animText(gs, screen, fmt.Sprintf("x%d", n), at.X+cardWidth/2-12, at.Y+cardHeight+18,
			animNoteSize, animStageInk)
	}
}

// smotherGalleryTicks is one blocked hit shown, a shield thrown onto it, and a beat on its card.
func smotherGalleryTicks() int { return smotherTotalTicks() + ui.Beat(1, 1) }

// drawSmotherGallery shows a fire attack's blocked hit, then throws a fire shield off a heap of two
// onto it, so the block is matched and SURGE is written across the card.
func drawSmotherGallery(gs *state.GlobalState, screen *ebiten.Image, at image.Rectangle, age int) {
	from := image.Pt(at.Min.X-460, at.Min.Y+40)
	ui.BlitCard(gs, screen, from, animFace(gs, combat.Card{Concept: combat.Block, Element: combat.Fire}), cards.Hand)
	ui.BlitCard(gs, screen, at.Min, animFace(gs, animCardA), cards.Hand)

	cx, cy := float64(from.X)+cardWidth/2, float64(from.Y)+cardHeight/2
	ui.DrawShield(screen, cards.Fire, ui.HeapPose(cx, cy, 0, 2, 1))

	m := smother{element: cards.Fire, from: ui.HeapPose(cx, cy, 1, 2, 1), surged: true,
		would: 20, ink: ui.ArtFor(animCardA.Element), t: ui.NewTravel(0, smotherTotalTicks())}
	m.t.Age = min(age, smotherTotalTicks())
	drawSmother(gs, screen, m, at)
}
