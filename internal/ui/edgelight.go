package ui

// edgelight.go — how an armored relic is told apart at rest: **a line of light running round a shield
// drawn on its face, all the time** *(owner's call)*. Armored's shield only exists in a fight, at the top of
// each turn; between turns, and on every screen that is not a fight, the relic would otherwise look
// like any other.
//
// **A trait that adds rules is the trait that glints**, read off the rules rather than the trait's
// name — see combat.RelicTrait.AddsRules — so a second trait carrying a rule arrives lit.
//
// **Drawn through the card's own transform**, the GeoM its face is drawn with, so the light stays on
// the edge of a card that is bobbing on its idle float. It is a stroke over the finished face rather
// than part of it, because it moves every frame and a face is baked once.
//
// It is code-drawn light, the kind of drawing the authored-art direction is replacing, taken on
// purpose for something that has to move.

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/state"
)

const (
	// edgeTail is how much of the border the light covers, as a share of the whole.
	edgeTail = 0.22

	// edgeSteps is how finely the border is walked; edgeWidth is the stroke at the light's head.
	edgeSteps = 160
	edgeWidth = 3.5
)

// edgeLap is one lap of the border: slow, so it reads as a glint going round rather than as a
// warning.
func edgeLap() int { return Beat(42, 5) }

// edgeInk is the light's color: near-white with a touch of steel, so it reads as a glint off metal
// rather than as a hue claiming an element.
var edgeInk = color.RGBA{0xf2, 0xf6, 0xff, 0xff}

// DrawEdgeLight runs the light round a w x h card whose face is drawn with geo, at tick t. seat
// staggers it, so two armored relics side by side are not in step.
func DrawEdgeLight(screen *ebiten.Image, geo ebiten.GeoM, w, h, radius int, t, seat int) {
	head := math.Mod(float64(t)/float64(edgeLap())+float64(seat)*0.37, 1)
	prev := edgePoint(head, w, h, radius)
	for i := 1; i <= edgeSteps; i++ {
		back := float64(i) / edgeSteps * edgeTail
		at := edgePoint(math.Mod(head-back+1, 1), w, h, radius)
		strength := 1 - float64(i)/edgeSteps
		c := edgeInk
		c.A = uint8(255 * strength * strength)
		x0, y0 := geo.Apply(prev[0], prev[1])
		x1, y1 := geo.Apply(at[0], at[1])
		vector.StrokeLine(screen, float32(x0), float32(y0), float32(x1), float32(y1),
			float32(edgeWidth*(0.35+0.65*strength)), c, true)
		prev = at
	}
}

// DrawEdgeLightFloating is DrawEdgeLight on a card resting on its idle float — the transform
// DrawFloatingCard draws the face with.
func DrawEdgeLightFloating(gs *state.GlobalState, screen *ebiten.Image, at image.Point, seat int, st cards.Style) {
	DrawEdgeLight(screen, IdleFloat.GeoAt(at, st.Width, st.Height, gs.Count, seat),
		st.Width, st.Height, st.CornerRadius, gs.Count, seat)
}

// edgePoint is the point a share u of the way round the shield outline on a w x h card, clockwise
// from the middle of the top edge.
func edgePoint(u float64, w, h, radius int) [2]float64 {
	path := shieldPath(w, h, radius)
	d := u * path.total
	for i := 1; i < len(path.at); i++ {
		if path.along[i] >= d {
			seg := path.along[i] - path.along[i-1]
			f := 0.0
			if seg > 0 {
				f = (d - path.along[i-1]) / seg
			}
			a, b := path.at[i-1], path.at[i]
			return [2]float64{a[0] + (b[0]-a[0])*f, a[1] + (b[1]-a[1])*f}
		}
	}
	return path.at[len(path.at)-1]
}

// shieldOutline is a closed path walked by distance: its points, how far along each one is, and the
// whole length.
type shieldOutline struct {
	at    [][2]float64
	along []float64
	total float64
}

// shieldPaths caches one outline per card size, since the walk is the same every frame.
var shieldPaths = map[[3]int]shieldOutline{}

// shieldPath is a heater shield standing inside a w x h card: **a flat top with rounded shoulders,
// straight sides down a little under half the card, then two curves meeting in a point at the
// foot**. Inset from the border so the light reads as a shape on the card rather than as the
// card's own edge.
func shieldPath(w, h, radius int) shieldOutline {
	key := [3]int{w, h, radius}
	if p, ok := shieldPaths[key]; ok {
		return p
	}
	fw, fh := float64(w), float64(h)
	inset := fw * 0.08
	left, right := inset, fw-inset
	top, foot := fh*0.07, fh-fh*0.05
	mid := fw / 2
	shoulder := math.Max(float64(radius), fw*0.06)
	waist := top + (foot-top)*0.46

	var pts [][2]float64
	add := func(x, y float64) { pts = append(pts, [2]float64{x, y}) }
	bezier := func(p0, p1, p2, p3 [2]float64, n int) {
		for i := 1; i <= n; i++ {
			t := float64(i) / float64(n)
			mt := 1 - t
			x := mt*mt*mt*p0[0] + 3*mt*mt*t*p1[0] + 3*mt*t*t*p2[0] + t*t*t*p3[0]
			y := mt*mt*mt*p0[1] + 3*mt*mt*t*p1[1] + 3*mt*t*t*p2[1] + t*t*t*p3[1]
			add(x, y)
		}
	}
	corner := func(cx, cy, from float64, n int) {
		for i := 1; i <= n; i++ {
			a := from + float64(i)/float64(n)*math.Pi/2
			add(cx+shoulder*math.Cos(a), cy+shoulder*math.Sin(a))
		}
	}

	// Clockwise from the middle of the top.
	add(mid, top)
	add(right-shoulder, top)
	corner(right-shoulder, top+shoulder, -math.Pi/2, 8)
	add(right, waist)
	bezier([2]float64{right, waist}, [2]float64{right, waist + (foot-waist)*0.55},
		[2]float64{mid + (right-mid)*0.45, foot - (foot-waist)*0.18}, [2]float64{mid, foot}, 24)
	bezier([2]float64{mid, foot}, [2]float64{mid - (mid-left)*0.45, foot - (foot-waist)*0.18},
		[2]float64{left, waist + (foot-waist)*0.55}, [2]float64{left, waist}, 24)
	add(left, top+shoulder)
	corner(left+shoulder, top+shoulder, math.Pi, 8)
	add(mid, top)

	out := shieldOutline{at: pts, along: make([]float64, len(pts))}
	for i := 1; i < len(pts); i++ {
		out.total += math.Hypot(pts[i][0]-pts[i-1][0], pts[i][1]-pts[i-1][1])
		out.along[i] = out.total
	}
	shieldPaths[key] = out
	return out
}
