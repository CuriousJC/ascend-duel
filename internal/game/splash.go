package game

// splash.go — the studio splash, and the gate that holds the game behind it while it loads.
//
// **It is the frame's rather than a scene's**, because it plays before there is anything a scene
// could be handed: the catalogs, the profile's unlocks and the run are being built on another
// goroutine while it draws. So it is not in the registry and has no ActiveScreen, and while it is
// up Update and Draw do nothing else — no chrome, no score table, no journal, no watcher — which is
// what keeps the loader the only thing touching the state until it has finished. **The music under
// the splash is the default**, music.Default, started by the loader as its first job; score.go
// takes over once the splash is down.
//
// The sequence is dark, a flicker, the bulb lit with the logo for splashLitTicks, and a fade to
// black. **The fade waits for the loader**: a slow machine holds the lit frame, with the light
// still moving, until there is a game to hand over to. Nothing skips it.
//
// **The light is drawn in code on the bulb's own pixel grid** — the rays and the banded glow are
// rasterized at one pixel per art pixel and scaled up whole, so they sit on the same grid as the
// sprite rather than reading as a smooth effect laid over pixel art.

import (
	"image/color"
	"math"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// The beats, in ticks at the game's 60 a second.
const (
	splashDarkTicks = 180
	splashLitTicks  = 300
	splashFadeTicks = 30
)

// splashFlicker is the bulb catching: alternating runs of on and off, starting on, before it holds.
var splashFlicker = []int{5, 7, 3, 10}

// splashPixel is how many screen pixels one art pixel of the bulb takes.
const splashPixel = 6

// The two grounds, taken off the bulb sheet's own frames, and the light's colors.
var (
	splashDark = color.RGBA{22, 22, 22, 255}
	splashLit  = color.RGBA{58, 58, 58, 255}
	splashRay  = color.RGBA{227, 227, 0, 255}
	splashWarm = color.RGBA{255, 200, 60, 255}
)

// The light. splashRays is how many, splashRayStart how far from the glass a ray begins, in art
// pixels, and splashTurn how far they turn a tick, in radians.
const (
	splashRays     = 16
	splashRayStart = 11
	splashTurn     = 2 * math.Pi / splashRays / 240
)

// splashGlowBands are the glow's rings, innermost first: a radius in art pixels and how much light.
var splashGlowBands = []struct {
	radius float64
	alpha  float64
}{{14, 0.22}, {22, 0.14}, {31, 0.08}, {42, 0.04}}

type splash struct {
	tick int

	// loaded is closed by the loader when the game is ready to be handed over.
	loaded chan struct{}

	// after runs on the game's goroutine once the fade is over — what the loader could not do
	// from its own. Nil is nothing to do.
	after func()

	// fadeFrom is the tick the fade to black began on, or zero before it has.
	fadeFrom int

	bulbOff, bulbOn, logo *ebiten.Image

	// canvas is the screen at one pixel per art pixel, where the rays and the glow are drawn
	// before they are scaled up.
	canvas *ebiten.Image
}

// Boot puts the splash up and runs load beside it. **load runs on its own goroutine** and is the
// only thing that may touch the state until it returns; after runs on the game's once the splash
// has finished and load has.
func (g *Game) Boot(load, after func()) {
	off, on, logo := assets.SplashImages()
	s := &splash{
		loaded:  make(chan struct{}),
		after:   after,
		bulbOff: off,
		bulbOn:  on,
		logo:    logo,
		canvas:  ebiten.NewImage(state.ScreenWidth/splashPixel, state.ScreenHeight/splashPixel),
	}
	g.boot = s
	go func() {
		load()
		close(s.loaded)
	}()
}

// ready reports whether the loader has finished.
func (s *splash) ready() bool {
	select {
	case <-s.loaded:
		return true
	default:
		return false
	}
}

// flickerTicks is how long the bulb takes to catch.
func flickerTicks() int {
	n := 0
	for _, run := range splashFlicker {
		n += run
	}
	return n
}

// holdEnd is the earliest tick the fade may begin on. It begins on the first tick at or after it
// on which the loader is done.
func holdEnd() int { return splashDarkTicks + flickerTicks() + splashLitTicks }

// update advances the splash a tick and reports whether it is over.
func (s *splash) update() bool {
	s.tick++
	if s.fadeFrom == 0 && s.tick >= holdEnd() && s.ready() {
		s.fadeFrom = s.tick
	}
	return s.fadeFrom != 0 && s.tick >= s.fadeFrom+splashFadeTicks
}

// lit reports whether the bulb is on this tick.
func (s *splash) lit() bool {
	t := s.tick - splashDarkTicks
	if t < 0 {
		return false
	}
	on := true
	for _, run := range splashFlicker {
		if t < run {
			return on
		}
		t -= run
		on = !on
	}
	return true
}

func (s *splash) draw(screen *ebiten.Image) {
	lit := s.lit()

	bw, bh := s.bulbOn.Bounds().Dx(), s.bulbOn.Bounds().Dy()
	// The bulb hangs from the top edge, centered; the glass is the bottom 16 of its art rows.
	bx := float64(state.ScreenWidth/splashPixel-bw) / 2
	cx, cy := bx+float64(bw)/2, float64(bh)-8

	if !lit {
		screen.Fill(splashDark)
	} else {
		screen.Fill(splashLit)
		s.canvas.Clear()
		s.drawGlow(cx, cy)
		s.drawRays(cx, cy)
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
		op.GeoM.Scale(splashPixel, splashPixel)
		screen.DrawImage(s.canvas, op)
	}

	bulb := s.bulbOff
	if lit {
		bulb = s.bulbOn
	}
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	op.GeoM.Translate(bx, 0)
	op.GeoM.Scale(splashPixel, splashPixel)
	screen.DrawImage(bulb, op)

	if lit {
		lw, lh := s.logo.Bounds().Dx(), s.logo.Bounds().Dy()
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Translate(float64(state.ScreenWidth-lw)/2, float64(state.ScreenHeight-lh-30))
		screen.DrawImage(s.logo, op)
	}

	if s.fadeFrom != 0 {
		a := math.Min(1, float64(s.tick-s.fadeFrom)/splashFadeTicks)
		vector.FillRect(screen, 0, 0, state.ScreenWidth, state.ScreenHeight,
			color.RGBA{0, 0, 0, uint8(a * 255)}, false)
	}
}

// drawRays draws the light's rays onto the canvas, turning slowly, each pulsing a little out of
// step with its neighbours.
func (s *splash) drawRays(cx, cy float64) {
	turn := float64(s.tick) * splashTurn
	reach := float64(s.canvas.Bounds().Dx() + s.canvas.Bounds().Dy())
	for i := range splashRays {
		angle := turn + float64(i)*2*math.Pi/splashRays
		pulse := 0.7 + 0.3*math.Sin(float64(s.tick)*0.08+float64(i)*math.Pi/2)
		ink := color.RGBA{
			uint8(float64(splashRay.R) * pulse), uint8(float64(splashRay.G) * pulse), 0, uint8(255 * pulse),
		}
		dx, dy := math.Cos(angle), math.Sin(angle)
		vector.StrokeLine(s.canvas,
			float32(cx+dx*splashRayStart), float32(cy+dy*splashRayStart),
			float32(cx+dx*reach), float32(cy+dy*reach), 1, ink, false)
	}
}

// drawGlow lays the banded glow round the glass, outermost ring first, breathing slightly.
func (s *splash) drawGlow(cx, cy float64) {
	breath := 0.85 + 0.15*math.Sin(float64(s.tick)*0.05)
	for i := len(splashGlowBands) - 1; i >= 0; i-- {
		b := splashGlowBands[i]
		a := b.alpha * breath
		ink := color.RGBA{
			uint8(float64(splashWarm.R) * a), uint8(float64(splashWarm.G) * a), uint8(float64(splashWarm.B) * a),
			uint8(255 * a),
		}
		vector.FillCircle(s.canvas, float32(cx), float32(cy), float32(b.radius), ink, false)
	}
}
