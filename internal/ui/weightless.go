package ui

// weightless.go — how a weightless relic is told apart on its face: **lifted off the row and
// drifting, a little see-through, with light passing over it** *(owner's call)*.
//
// It is drawn entirely from the relic's own finished card: a soft shadow of its silhouette on the
// row under it, the card itself raised and swaying, and a band of its own picture added onto itself
// sweeping down it. No art of its own, so every relic that can be worn weightless already has its
// weightless face.
//
// **Lifted is the meaning**: a weightless relic takes no slot, so it does not rest on the row the
// slots are counted on. **The see-through is light rather than faded** — a card faded toward the
// ground already means disabled, so the shimmer is what stops a weightless relic reading as one the
// player cannot use.

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// The drift's clock, in ticks. Idle motion, like the hand name's breath, so it is not tied to the
// game's playback speed: a relic does not drift faster because a round is being watched faster.
const (
	weightlessDriftTicks   = 150.0
	weightlessShimmerTicks = 140.0
)

// WeightlessLift is how far a weightless relic stands off its row at tick t: a steady lift with a
// slow drift on top, a small sway and a smaller turn. `phase` keeps a row of them out of step.
func WeightlessLift(t, phase float64) (lift, sway, angle float64) {
	s := math.Sin(2*math.Pi*t/weightlessDriftTicks + phase)
	c := math.Cos(2*math.Pi*t/(weightlessDriftTicks*1.3) + phase)
	return 12 + 4*s, 2.5 * c, 0.02 * c
}

// weightlessGeo is the transform for a card at `scale`, lifted, swayed and turned about its own
// middle, with its resting top-left at `at`.
func weightlessGeo(face *ebiten.Image, at image.Point, scale, dx, dy, angle float64) ebiten.GeoM {
	w := float64(face.Bounds().Dx()) * scale
	h := float64(face.Bounds().Dy()) * scale
	var g ebiten.GeoM
	g.Scale(scale, scale)
	g.Translate(-w/2, -h/2)
	g.Rotate(angle)
	g.Translate(float64(at.X)+w/2+dx, float64(at.Y)+h/2+dy)
	return g
}

// DrawWeightlessShadow is the card's own silhouette, dark and soft, on the row under where it would
// rest. **The softness is a few offset draws**, a blur cheap enough for a row of cards.
func DrawWeightlessShadow(screen, face *ebiten.Image, at image.Point, scale, lift float64) {
	spread := 2 + lift*0.25
	strength := float32(0.28 - lift*0.006)
	for _, o := range [][2]float64{{0, 0}, {spread, 0}, {-spread, 0}, {0, spread}, {0, -spread}} {
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM = weightlessGeo(face, at, scale, 4+o[0], 6+o[1], 0)
		op.ColorScale.Scale(0, 0, 0, strength/2.5)
		screen.DrawImage(face, op)
	}
}

// DrawWeightless draws a finished card face as a weightless relic: its shadow, the card lifted and
// drifting at 85%, and a band of light sweeping down it. `at` is where it would rest, `t` the free
// clock, `phase` this card's offset in its row.
func DrawWeightless(screen, face *ebiten.Image, at image.Point, scale, t, phase float64) {
	if face == nil {
		return
	}
	lift, sway, angle := WeightlessLift(t, phase)
	DrawWeightlessShadow(screen, face, at, scale, lift)

	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM = weightlessGeo(face, at, scale, sway, -lift, angle)
	op.ColorScale.ScaleAlpha(0.85)
	screen.DrawImage(face, op)

	// **Light passing over it**: the card's own strip added onto itself, a bright band and a softer
	// one behind it, sweeping top to bottom every couple of seconds.
	p := math.Mod(t+phase*30, weightlessShimmerTicks) / weightlessShimmerTicks
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
