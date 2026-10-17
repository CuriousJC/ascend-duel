package ui

// aloft.go — how a thing that **takes no seat** is told apart on its face: **lifted off its row and
// drifting, a little see-through, with light passing over it** *(owner's call)*. A weightless relic
// takes no slot on the band, a weightless consumable no seat in the pane, and an effortless card no
// place in the turn's five; all three are drawn this way, so the one picture means the one thing.
//
// It is drawn entirely from the thing's own finished card: a soft shadow of its silhouette on the
// row under it, the card itself raised and swaying, and a band of its own picture added onto itself
// sweeping down it. No art of its own, so every card that can be aloft already has its aloft face.
//
// **Lifted is the meaning**: it is not counted on the row the seats are counted on, so it does not
// rest there. **The see-through is light rather than faded** — a card faded toward the ground
// already means disabled, so the shimmer is what stops an aloft card reading as one the player
// cannot use.

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// The drift's clock, in ticks. Idle motion, like the hand name's breath, so it is not tied to the
// game's playback speed: a card does not drift faster because a round is being watched faster.
const (
	aloftDriftTicks   = 150.0
	aloftShimmerTicks = 140.0
)

// AloftLift is how far a aloft card stands off its row at tick t: a steady lift with a
// slow drift on top, a small sway and a smaller turn. `phase` keeps a row of them out of step.
func AloftLift(t, phase float64) (lift, sway, angle float64) {
	s := math.Sin(2*math.Pi*t/aloftDriftTicks + phase)
	c := math.Cos(2*math.Pi*t/(aloftDriftTicks*1.3) + phase)
	return 12 + 4*s, 2.5 * c, 0.02 * c
}

// aloftGeo is the transform for a card at `scale`, lifted, swayed and turned about its own
// middle, with its resting top-left at `at`.
func aloftGeo(face *ebiten.Image, at image.Point, scale, dx, dy, angle float64) ebiten.GeoM {
	w := float64(face.Bounds().Dx()) * scale
	h := float64(face.Bounds().Dy()) * scale
	var g ebiten.GeoM
	g.Scale(scale, scale)
	g.Translate(-w/2, -h/2)
	g.Rotate(angle)
	g.Translate(float64(at.X)+w/2+dx, float64(at.Y)+h/2+dy)
	return g
}

// DrawAloftShadow is the card's own silhouette, dark and soft, on the row under where it would
// rest. **The softness is a few offset draws**, a blur cheap enough for a row of cards.
func DrawAloftShadow(screen, face *ebiten.Image, at image.Point, scale, lift float64) {
	spread := 2 + lift*0.25
	strength := float32(0.28 - lift*0.006)
	for _, o := range [][2]float64{{0, 0}, {spread, 0}, {-spread, 0}, {0, spread}, {0, -spread}} {
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM = aloftGeo(face, at, scale, 4+o[0], 6+o[1], 0)
		op.ColorScale.Scale(0, 0, 0, strength/2.5)
		screen.DrawImage(face, op)
	}
}

// DrawSeatedCard is a card in its seat on a row: aloft when `aloft` — it takes no seat — and floating
// like every other card in the row otherwise. `seat` keeps a row out of step with itself either way.
func DrawSeatedCard(gs *state.GlobalState, screen *ebiten.Image, at image.Point, seat int,
	spec cards.Spec, st cards.Style, aloft bool) {

	if aloft {
		DrawAloft(screen, CardImage(gs, spec, st), at, 1, float64(gs.Count), float64(seat)*2.1)
		return
	}
	DrawFloatingCard(gs, screen, at, seat, spec, st)
}

// DrawAloft draws a finished card face as an aloft card: its shadow, the card lifted and
// drifting at 85%, and a band of light sweeping down it. `at` is where it would rest, `t` the free
// clock, `phase` this card's offset in its row.
func DrawAloft(screen, face *ebiten.Image, at image.Point, scale, t, phase float64) {
	if face == nil {
		return
	}
	lift, sway, angle := AloftLift(t, phase)
	DrawAloftShadow(screen, face, at, scale, lift)

	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM = aloftGeo(face, at, scale, sway, -lift, angle)
	op.ColorScale.ScaleAlpha(0.85)
	screen.DrawImage(face, op)

	// **Light passing over it**: the card's own strip added onto itself, a bright band and a softer
	// one behind it, sweeping top to bottom every couple of seconds.
	p := math.Mod(t+phase*30, aloftShimmerTicks) / aloftShimmerTicks
	h := face.Bounds().Dy()
	band := h / 5
	y0 := int(p*float64(h+band)) - band
	for _, k := range []float32{0.35, 0.2} {
		lo, hi := max(0, y0), min(h, y0+band)
		if hi <= lo {
			break
		}
		strip := face.SubImage(image.Rect(0, lo, face.Bounds().Dx(), hi)).(*ebiten.Image)
		add := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear, Blend: ebiten.BlendLighter}
		var g ebiten.GeoM
		g.Translate(0, float64(lo))
		g.Concat(op.GeoM)
		add.GeoM = g
		add.ColorScale.ScaleAlpha(k)
		screen.DrawImage(strip, add)
		y0 += band / 3
		band /= 2
	}
}
