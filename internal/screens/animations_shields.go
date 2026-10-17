package screens

// animations_shields.go — the gallery's shield gestures: a heap forming over the card that raised
// it, one to five in every element, and a hit stopped in the middle by a shield.

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

// galleryInterceptFigure is the gallery's one stopped figure, drawn onto its image once.
var galleryInterceptFigure *ebiten.Image

// interceptGalleryTicks is one stopped hit, meeting, breaking and read, and a beat after.
func interceptGalleryTicks() int {
	return interceptFlyTicks() + interceptBreakTicks() + interceptHoldTicks() + ui.Beat(1, 1)
}

// drawInterceptGallery throws an attack's hit out of its card on the right at the Block card on the
// left, standing in for the duelist, and a fire shield off that card's heap of two meets it halfway,
// so the block is matched and SURGE is written over the break.
func drawInterceptGallery(gs *state.GlobalState, screen *ebiten.Image, at image.Rectangle, age int) {
	from := image.Pt(at.Min.X-460, at.Min.Y+40)
	attack := image.Pt(at.Min.X+460, at.Min.Y)
	ui.BlitCard(gs, screen, from, animFace(gs, combat.Card{Concept: combat.Block, Element: combat.Fire}), cards.Hand)
	ui.BlitCard(gs, screen, attack, animFace(gs, animCardA), cards.Hand)

	cx, cy := float64(from.X)+cardWidth/2, float64(from.Y)+cardHeight/2
	ui.DrawShield(screen, cards.Fire, ui.HeapPose(cx, cy, 0, 2, 1))

	ic := intercept{figure: "-20", ink: ui.ArtFor(animCardA.Element), shield: cards.Fire,
		shieldFrom: ui.HeapPose(cx, cy, 1, 2, 1), label: "SURGE"}
	ic.aim(0.5)
	ic.t.Age = min(age, ic.t.Ticks)
	// **The figure's image is built once for the page**, rather than on every frame of the break.
	if galleryInterceptFigure == nil {
		galleryInterceptFigure = figureImage(gs, ic.figure, hitInkFor(ic.ink))
	}
	ic.figureImg = galleryInterceptFigure
	if ic.Done() {
		return
	}
	drawIntercept(gs, screen, &ic, image.Pt(attack.X+cardWidth/2, attack.Y+cardHeight/2),
		image.Pt(int(cx), int(cy)))
}
