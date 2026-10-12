package ui

// The shields a duelist is standing behind, drawn as a heap over whatever raised them.
//
// **A shield forms over the card that made it and stays there** — a defend card, an attack card
// carrying a shield rider, or the ring of a relic that raises one — so where a shield came from is
// never a thing to remember. Several on one card **heap**: each lies a little offset and turned
// from the one under it, so a pile of three is three edges and can be counted at a glance. The
// drawing is `docs/art/shield_art_prompt.MD`; a heavy contour and a light rim are what keep the
// edges apart.
//
// **The scatter is written out, never rolled.** A heap is a picture of a count, and the same count
// lands the same way every time.

import (
	"math"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
)

// ShieldSize is the square a shield is drawn on: two thirds of a card's height, which puts the
// shield itself at about two thirds of the card's width.
const ShieldSize = 280 * 2 / 3

// ShieldKey is the shield drawing for one element, neutral for anything that is not one of the
// five. **Total**, so a shield with no element still draws rather than leaving a hole.
func ShieldKey(e cards.Element) string {
	switch e {
	case cards.Fire, cards.Ice, cards.Lightning, cards.Earth, cards.Arcane:
		return "shield-" + e.String()
	}
	return "shield-neutral"
}

// heapScatter is where each shield in a heap lies relative to the centre of what raised it: a
// nudge across, a nudge down and a turn in degrees. A heap past its length starts round again a
// little lower, so nothing is ever drawn exactly on top of another.
var heapScatter = [...]struct{ dx, dy, deg float64 }{
	{-6, -10, -4},
	{14, 6, 7},
	{-16, 18, -10},
	{10, 30, 12},
	{-4, 40, -2},
}

// HeapPlace is where shield i of a heap of n lies, centred on what raised it whatever n is, so a
// heap of five does not hang lower than a heap of one.
func HeapPlace(i, n int) (dx, dy, deg float64) {
	if n <= 0 {
		return 0, 0, 0
	}
	last := min(n, len(heapScatter)) - 1
	mid := heapScatter[last].dy / 2
	m := heapScatter[i%len(heapScatter)]
	lap := float64(i / len(heapScatter))
	return m.dx, m.dy - mid + lap*12, m.deg
}

// ShieldFormTicks is one shield forming; ShieldStaggerTicks is the gap before the next in the same
// heap starts, so a Block's two arrive as two.
func ShieldFormTicks() int    { return Beat(1, 2) }
func ShieldStaggerTicks() int { return Beat(1, 3) }

// ShieldFormed is how far shield i of a heap has formed, `age` ticks after the heap started: 0
// before it begins, 1 once it has landed.
func ShieldFormed(age, i int) float64 {
	return Clamp01(float64(age-i*ShieldStaggerTicks()) / float64(ShieldFormTicks()))
}

// HeapFormTicks is how long a heap of n takes to form completely.
func HeapFormTicks(n int) int {
	if n <= 0 {
		return 0
	}
	return (n-1)*ShieldStaggerTicks() + ShieldFormTicks()
}

// ShieldPose is one shield's placement on screen: its centre, its turn, its size and how solid it
// is.
type ShieldPose struct {
	X, Y  float64
	Deg   float64
	Scale float64
	Alpha float32
}

// Forming is a pose for a shield `formed` of the way into its heap seat: it drops in from a little
// larger and fades up.
func (p ShieldPose) Forming(formed float64) ShieldPose {
	e := EaseOut(formed)
	p.Scale *= 1.25 - 0.25*e
	p.Alpha *= float32(e)
	return p
}

// HeapPose is shield i of n over a centre, at a scale.
func HeapPose(cx, cy float64, i, n int, scale float64) ShieldPose {
	dx, dy, deg := HeapPlace(i, n)
	return ShieldPose{X: cx + dx*scale, Y: cy + dy*scale, Deg: deg, Scale: scale, Alpha: 1}
}

// DrawShield draws one shield in an element at a pose.
func DrawShield(screen *ebiten.Image, e cards.Element, p ShieldPose) {
	img := systems.ArtMarkImage(ShieldKey(e), ShieldSize, ShieldSize)
	if img == nil || p.Alpha <= 0 {
		return
	}
	half := float64(ShieldSize) / 2
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	op.GeoM.Translate(-half, -half)
	op.GeoM.Scale(p.Scale, p.Scale)
	op.GeoM.Rotate(p.Deg * math.Pi / 180)
	op.GeoM.Translate(p.X, p.Y)
	op.ColorScale.ScaleAlpha(p.Alpha)
	screen.DrawImage(img, op)
}

// LerpPose is the pose `t` of the way from one to another.
func LerpPose(a, b ShieldPose, t float64) ShieldPose {
	return ShieldPose{
		X:     a.X + (b.X-a.X)*t,
		Y:     a.Y + (b.Y-a.Y)*t,
		Deg:   a.Deg + (b.Deg-a.Deg)*t,
		Scale: a.Scale + (b.Scale-a.Scale)*t,
		Alpha: a.Alpha + (b.Alpha-a.Alpha)*float32(t),
	}
}
