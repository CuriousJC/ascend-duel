package screens

// animations_particles.go — the gallery's particle entries: every element in every sprite set, side
// by side, so the options can be compared before any of them is wired into a fight.
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

// particleSets is the grid's rows.
var particleSets = []struct {
	name string
	set  ui.ParticleSet
}{
	{"shards", ui.ParticleShards},
	{"sparks", ui.ParticleSparks},
	{"mixed", ui.ParticleMixed},
}

// drawWrapGrid draws a resting total in each element, wrapped in its particles, once per sprite set.
// **On the free clock**, because a wrap has no start and no end: it is up for as long as the line is.
func drawWrapGrid(gs *state.GlobalState, screen *ebiten.Image, origin image.Point, motion ui.ParticleMotion, glow bool) {
	for c, el := range particleElements {
		x := origin.X + c*particleCellW + particleCellW/2
		animText(gs, screen, el.String(), x-30, origin.Y-24, floatLabelSize, ui.GroundInk)
	}
	for r, row := range particleSets {
		y := origin.Y + r*particleCellH + particleCellH/2
		animText(gs, screen, row.name, origin.X-90, y, floatLabelSize, animStageInk)
		for c, el := range particleElements {
			center := image.Pt(origin.X+c*particleCellW+particleCellW/2, y)
			em := ui.Emitter{Element: el, Set: row.set, Motion: motion, Glow: glow, Seed: uint32(r*7 + c + 1)}
			ui.DrawParticleWrap(screen, em, center, particleWrapRX, particleWrapRY, -1, gs.Count)
			drawMathText(gs, screen, strconv.Itoa(particleFigure), mathTotalSize, hitInkFor(el), center, 1, 1, false)
		}
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
func drawTrailRows(gs *state.GlobalState, screen *ebiten.Image, origin image.Point, set ui.ParticleSet, age int) {
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

		em := ui.Emitter{Element: el, Set: set, Seed: uint32(r + 11)}
		ui.DrawParticleWrap(screen, em, from, particleWrapRX, particleWrapRY, wrap, age)
		trail := em
		trail.Seed += 100
		trail.Rate = 1.2
		trail.Size = 22
		ui.DrawParticleTrail(screen, trail, path, wrap+int(fly), age)

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
