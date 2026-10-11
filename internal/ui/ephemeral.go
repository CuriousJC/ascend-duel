package ui

// ephemeral.go — how an ephemeral relic is told apart on its face: **it fades out toward its
// bottom edge**, opaque at 90% across the top and 10% across the bottom, **and its border is
// broken** *(owner's call)*. A relic that leaves at the end of the fight is drawn already going.
//
// Both are masks over the relic's own finished card, so it composes with everything else a worn
// relic can be: a weightless one is the same faded, broken face, lifted and drifting — see
// DrawWeightless. A cantrip's relic is both.
//
// **The breaks are derived from the outline, never rolled** — the crack pattern's rule. Every
// ephemeral card breaks in the same places, through a resize, a re-entry or a replay.

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// The fade's two ends, as opacity.
const (
	ephemeralTopAlpha    = 0.90
	ephemeralBottomAlpha = 0.10
)

// ephemeralFaces caches a treated copy per finished card face. **Keyed by the face image**, which
// CardImage already caches per spec, so this holds at most one entry per face that cache holds.
var ephemeralFaces = map[*ebiten.Image]*ebiten.Image{}

// ephemeralMasks is one column of the fade per card height, stretched across the card's width.
var ephemeralMasks = map[int]*ebiten.Image{}

// borderMasks is the broken border per card size: the alpha of what is cut away.
var borderMasks = map[image.Point]*ebiten.Image{}

func ephemeralMask(h int) *ebiten.Image {
	if m, ok := ephemeralMasks[h]; ok {
		return m
	}
	px := make([]byte, 4*h)
	for y := 0; y < h; y++ {
		f := 0.0
		if h > 1 {
			f = float64(y) / float64(h-1)
		}
		a := byte(255 * (ephemeralTopAlpha + (ephemeralBottomAlpha-ephemeralTopAlpha)*f))
		// Premultiplied, as every ebiten image is; DestinationIn reads only the alpha.
		px[4*y], px[4*y+1], px[4*y+2], px[4*y+3] = a, a, a, a
	}
	m := ebiten.NewImage(1, h)
	m.WritePixels(px)
	ephemeralMasks[h] = m
	return m
}

// EphemeralFace is a finished relic face as an ephemeral relic: faded and its border broken. Nil in,
// nil out.
func EphemeralFace(face *ebiten.Image) *ebiten.Image {
	if face == nil {
		return nil
	}
	if img, ok := ephemeralFaces[face]; ok {
		return img
	}
	b := face.Bounds()
	out := ebiten.NewImage(b.Dx(), b.Dy())
	out.DrawImage(face, nil)

	fade := &ebiten.DrawImageOptions{Blend: ebiten.BlendDestinationIn}
	fade.GeoM.Scale(float64(b.Dx()), 1)
	out.DrawImage(ephemeralMask(b.Dy()), fade)

	out.DrawImage(borderMask(b.Dx(), b.Dy()), &ebiten.DrawImageOptions{Blend: ebiten.BlendDestinationOut})

	ephemeralFaces[face] = out
	return out
}

// EphemeralIf is the face treated when `on`, and the face itself otherwise — for a caller drawing a
// row where only some relics are ephemeral.
func EphemeralIf(face *ebiten.Image, on bool) *ebiten.Image {
	if !on {
		return face
	}
	return EphemeralFace(face)
}

// DrawEphemeralFloating is DrawFloatingCard for an ephemeral relic: the same idle float, on the
// treated face.
func DrawEphemeralFloating(gs *state.GlobalState, screen *ebiten.Image, at image.Point, seat int,
	spec cards.Spec, st cards.Style) {

	face := EphemeralFace(CardImage(gs, spec, st))
	if face == nil {
		return
	}
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM = IdleFloat.GeoAt(at, st.Width, st.Height, gs.Count, seat)
	screen.DrawImage(face, op)
}

// borderMask is what the breaks cut out of a w x h relic card's border, as alpha.
func borderMask(w, h int) *ebiten.Image {
	key := image.Pt(w, h)
	if m, ok := borderMasks[key]; ok {
		return m
	}

	r := float64(cards.RelicStyle.CornerRadius)
	bw := float64(cards.RelicStyle.BorderWidth)
	path := newOutline(float64(w), float64(h), r)
	cut := jaggedCut(path.length, bw)

	px := make([]byte, 4*w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			s, d := path.locate(float64(x)+0.5, float64(y)+0.5)
			if d > bw+3 {
				continue
			}
			a := byte(255 * clamp01(cut(s, d)))
			i := 4 * (y*w + x)
			px[i], px[i+1], px[i+2], px[i+3] = a, a, a, a
		}
	}
	m := ebiten.NewImage(w, h)
	m.WritePixels(px)
	borderMasks[key] = m
	return m
}

// jaggedCut is dashes of uneven length with breaks long and short, laid down once along the whole
// outline. Each break bites a little deeper or shallower pixel by pixel, which is what makes its
// inner edge ragged rather than square, and its ends fade over a couple of pixels.
func jaggedCut(length, bw float64) func(s, d float64) float64 {
	type gap struct{ from, to float64 }
	var gaps []gap
	h := uint32(0x9E3779B9)
	next := func(lo, hi int) float64 {
		h ^= h << 13
		h ^= h >> 17
		h ^= h << 5
		return float64(lo + int(h%uint32(hi-lo+1)))
	}
	for s := next(4, 12); s < length; {
		size := next(2, 7)
		if next(0, 4) == 0 {
			size = next(11, 20)
		}
		gaps = append(gaps, gap{s, s + size})
		s += size + next(8, 34)
	}

	return func(s, d float64) float64 {
		for _, g := range gaps {
			if s < g.from-1 || s > g.to+1 {
				continue
			}
			along := clamp01(math.Min(s-g.from+1, g.to+1-s) / 2)
			depth := bw + 1 + 1.5*math.Sin(s*1.7+g.from)
			return along * clamp01(depth-d)
		}
		return 0
	}
}

func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }

// outline is a rounded rectangle's edge, walked clockwise from the end of the top-left corner.
// locate answers, for a pixel, how far round the edge its nearest point is and how far inside it
// the pixel sits — the two figures a pattern along the border needs.
type outline struct {
	w, h, r        float64
	top, side, arc float64
	length         float64
}

func newOutline(w, h, r float64) outline {
	o := outline{w: w, h: h, r: r, top: w - 2*r, side: h - 2*r, arc: math.Pi / 2 * r}
	o.length = 2*o.top + 2*o.side + 4*o.arc
	return o
}

func (o outline) locate(x, y float64) (s, d float64) {
	w, h, r := o.w, o.h, o.r
	corner := func(cx, cy, start, from float64) (float64, float64) {
		t := math.Atan2(y-cy, x-cx) - from
		for t < 0 {
			t += 2 * math.Pi
		}
		return start + t*r, r - math.Hypot(x-cx, y-cy)
	}
	switch {
	case x < r && y < r:
		return corner(r, r, 2*o.top+2*o.side+3*o.arc, -math.Pi)
	case x > w-r && y < r:
		return corner(w-r, r, o.top, -math.Pi/2)
	case x > w-r && y > h-r:
		return corner(w-r, h-r, o.top+o.arc+o.side, 0)
	case x < r && y > h-r:
		return corner(r, h-r, 2*o.top+o.side+2*o.arc, math.Pi/2)
	}

	top, right, bottom, left := y, w-x, h-y, x
	switch math.Min(math.Min(top, bottom), math.Min(left, right)) {
	case top:
		return x - r, top
	case right:
		return o.top + o.arc + (y - r), right
	case bottom:
		return o.top + o.side + 2*o.arc + (w - r - x), bottom
	default:
		return 2*o.top + o.side + 3*o.arc + (h - r - y), left
	}
}
