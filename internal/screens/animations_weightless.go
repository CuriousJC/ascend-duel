package screens

// animations_weightless.go — the gallery's comparison of ways a weightless relic could be told apart
// on its face: lifted off the row and drifting, a ghost of itself, and lifted with a shimmer — the
// last being the one the band draws (`ui.DrawWeightless`). Every one is drawn from the relic's own
// finished card over a relic pane's backing, with an untreated relic first in each row to read the
// others against.

import (
	"image"
	"math"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// The relics the comparison is drawn on, picked to look unlike each other: an orb, a weapon and a
// jar, plus the ring every row starts with untreated.
var (
	weightlessReference = "dmg-pair-hand"
	weightlessSamples   = []string{"flip-fire-to-ice", "dmg-all-slash", "jar-fire"}
)

// The comparison's shape. The cards are drawn at a little over half size so three rows fit.
const (
	weightlessScale    = 0.62
	weightlessRowPitch = 215
	weightlessCardGap  = 150
	weightlessRefGap   = 60
)

// weightlessOption is one way of drawing a weightless relic.
type weightlessOption struct {
	name, what string
	draw       func(screen, face *ebiten.Image, at image.Point, t float64, phase float64)
}

var weightlessOptions = []weightlessOption{
	{"float", "lifted off the row, a soft shadow under it, drifting", drawWeightlessFloat},
	{"ghost", "see-through, with a faint pulsing glow", drawWeightlessGhost},
	{"float + shimmer (the band's)", "lifted and drifting, slightly see-through, light passing over it",
		func(screen, face *ebiten.Image, at image.Point, t, phase float64) {
			ui.DrawWeightless(screen, face, at, weightlessScale, t, phase)
		}},
}

// drawWeightlessComparison is the page: one row per option.
func drawWeightlessComparison(gs *state.GlobalState, screen *ebiten.Image, origin image.Point) {
	face := func(key string) *ebiten.Image {
		r, ok := gs.Relics[key]
		if !ok {
			return nil
		}
		return ui.CardImage(gs, ui.RelicSpec(gs, r, "", true, false), cards.RelicStyle)
	}
	ref := face(weightlessReference)
	t := float64(gs.Count)

	w := float64(cards.RelicStyle.Width) * weightlessScale
	h := float64(cards.RelicStyle.Height) * weightlessScale
	for row, opt := range weightlessOptions {
		y := origin.Y + row*weightlessRowPitch
		animText(gs, screen, opt.name+" - "+opt.what, origin.X, y-14, animNoteSize, ui.GroundInk)

		// The pane's backing, one step off the ground, as the band draws it.
		width := float32(w) + weightlessRefGap + float32(len(weightlessSamples))*weightlessCardGap + 20
		vector.DrawFilledRect(screen, float32(origin.X-10), float32(y+6), width, float32(h)+28,
			relicPaneBackColor, false)

		if ref != nil {
			op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
			op.GeoM.Scale(weightlessScale, weightlessScale)
			op.GeoM.Translate(float64(origin.X), float64(y+20))
			screen.DrawImage(ref, op)
		}
		for i, key := range weightlessSamples {
			at := image.Pt(origin.X+int(w)+weightlessRefGap+i*weightlessCardGap, y+20)
			if img := face(key); img != nil {
				opt.draw(screen, img, at, t, float64(i)*2.1)
			}
		}
	}
}

// liftedGeo is the comparison's transform for a card lifted, swayed and turned about its middle.
func liftedGeo(face *ebiten.Image, at image.Point, dx, dy, angle float64) ebiten.GeoM {
	w := float64(face.Bounds().Dx()) * weightlessScale
	h := float64(face.Bounds().Dy()) * weightlessScale
	var g ebiten.GeoM
	g.Scale(weightlessScale, weightlessScale)
	g.Translate(-w/2, -h/2)
	g.Rotate(angle)
	g.Translate(float64(at.X)+w/2+dx, float64(at.Y)+h/2+dy)
	return g
}

func drawWeightlessFloat(screen, face *ebiten.Image, at image.Point, t, phase float64) {
	lift, sway, angle := ui.WeightlessLift(t, phase)
	ui.DrawWeightlessShadow(screen, face, at, weightlessScale, lift)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM = liftedGeo(face, at, sway, -lift, angle)
	screen.DrawImage(face, op)
}

func drawWeightlessGhost(screen, face *ebiten.Image, at image.Point, t, phase float64) {
	pulse := 0.5 + 0.5*math.Sin(2*math.Pi*t/120+phase)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM = liftedGeo(face, at, 0, 0, 0)
	op.ColorScale.ScaleAlpha(0.62)
	screen.DrawImage(face, op)

	glow := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear, Blend: ebiten.BlendLighter}
	glow.GeoM = op.GeoM
	glow.ColorScale.ScaleAlpha(float32(0.08 + 0.14*pulse))
	screen.DrawImage(face, glow)
}
