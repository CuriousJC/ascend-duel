package screens

// animations_particles.go — the gallery's particle entries: the wrap round a resting total and the
// trail behind a flying one, in every element, off the game's own emitters.
//
// **The figure is the game's own.** The number is `drawMathText` at the total's size in
// `hitInkFor`'s color, and the flight is `drawHits`' easing, scale and fade — so what is being judged
// is the particles round the number the game actually draws.

import (
	"image"
	"strconv"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

// The particle grid's shape.
const (
	particleCellW    = 240
	particleCellH    = 190
	particleWrapRX   = 31
	particleWrapRY   = 20
	particleTrailRun = 1050
	particleTrailRow = 120
	particleFigure   = 42
)

// particleElements is the five, in the order the grid draws them.
var particleElements = []cards.Element{cards.Fire, cards.Ice, cards.Lightning, cards.Earth, cards.Arcane}

// drawWrapRow draws a resting total in each element, wrapped in its particles. **On the free
// clock**, because a wrap has no start and no end: it is up for as long as the line is.
func drawWrapRow(gs *state.GlobalState, screen *ebiten.Image, origin image.Point) {
	y := origin.Y + particleCellH/2
	for c, el := range particleElements {
		x := origin.X + c*particleCellW + particleCellW/2
		animText(gs, screen, el.String(), x-30, origin.Y-24, floatLabelSize, ui.GroundInk)
		center := image.Pt(x, y)
		ui.DrawParticleWrap(screen, wrapEmitter(el, uint32(c+1), 0), center, particleWrapRX, particleWrapRY, -1, gs.Count)
		drawMathText(gs, screen, strconv.Itoa(particleFigure), mathTotalSize, hitInkFor(el), center, 1, 1, false)
	}
}

// trailWrapTicks is how long the total rests, wrapped, before it sets off.
func trailWrapTicks() int { return ui.Beat(1, 1) }

// trailGalleryTicks is one whole run of a trail entry: the rest, the flight, the hold, and the
// trail dying away behind it.
func trailGalleryTicks() int {
	return trailWrapTicks() + hitFlyTicks() + hitHoldTicks() + 48
}

// drawTrailRows flies a hit's figure across the stage once per element, wrapped while it rests and
// trailing its particles as it goes. `age` is the gallery's own clock, so the trail outlives the hold.
func drawTrailRows(gs *state.GlobalState, screen *ebiten.Image, origin image.Point, age int) {
	wrap := trailWrapTicks()
	fly := float64(hitFlyTicks())
	for r, el := range particleElements {
		y := origin.Y + r*particleTrailRow + particleTrailRow/2
		animText(gs, screen, el.String(), origin.X-110, y, floatLabelSize, animStageInk)

		from := image.Pt(origin.X+40, y)
		to := image.Pt(origin.X+particleTrailRun, y)
		path := func(t float64) (float64, float64) {
			p := ui.EaseOut(ui.Clamp01((t - float64(wrap)) / fly))
			return float64(from.X) + float64(to.X-from.X)*p, float64(from.Y) + float64(to.Y-from.Y)*p
		}

		ui.DrawParticleWrap(screen, wrapEmitter(el, uint32(r+11), 0), from, particleWrapRX, particleWrapRY, wrap, age)
		ui.DrawParticleTrail(screen, trailEmitter(el, uint32(r+111)), path, wrap+int(fly), age)

		// The figure, drawn as drawHits draws it.
		if age < wrap {
			drawMathText(gs, screen, strconv.Itoa(particleFigure), mathTotalSize, hitInkFor(el), from, 1, 1, false)
			continue
		}
		h := hitFlight{t: ui.NewTravel(0, hitFlyTicks()+hitHoldTicks())}
		h.t.Age = age - wrap
		if h.Done() {
			continue
		}
		p := ui.EaseOut(ui.Clamp01(float64(h.t.Age) / fly))
		x, yy := path(float64(age))
		scale := hitFromScale + (hitToScale-hitFromScale)*p
		drawMathText(gs, screen, "-"+strconv.Itoa(particleFigure), hitFigureSize, hitInkFor(el),
			image.Pt(int(x), int(yy)), scale, hitAlpha(h), false)
	}
}
