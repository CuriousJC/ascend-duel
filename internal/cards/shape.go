package cards

import (
	"image"
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/systems"
)

// Rounded-rectangle rasterising, in plain Go.
//
// **The whole game's rounded corners come from here**, for the reason set out in the
// package comment: the screen's old mask-and-blend rounding needed a graphics context and
// this package must not have one, so when the fighters became cards that path lost its
// last caller and went. What is here is deliberately the smallest thing that draws the
// shape.
//
// **Hard-edged, no antialiasing.** The cards were already drawn that way and the glyphs
// on them are 1:1 pixel art, so a soft card edge would put two different rendering
// idioms on one face. It also makes the shape exactly testable — a pixel is either the
// border colour or it is not, so the tests can assert colours rather than tolerances,
// which matters when nobody is looking at the output.

// roundedRect fills a rounded rectangle of the given colour into dst.
//
// The corner test is the ordinary one: inside each corner's radius square, a pixel
// belongs to the shape only if it falls within `radius` of that corner's centre. Every
// other pixel in the bounding box is inside. Comparing squared distances keeps it in
// integers and avoids a square root per pixel.
func roundedRect(dst *image.RGBA, x, y, w, h, radius int, c color.RGBA) {
	if w <= 0 || h <= 0 {
		return
	}
	radius = clampRadius(w, h, radius)

	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			if !insideRounded(w, h, radius, px, py) {
				continue
			}
			dst.SetRGBA(x+px, y+py, c)
		}
	}
}

// insideRounded reports whether (px, py) is part of a w x h rounded rectangle whose top-left
// corner is the origin.
//
// **Its own function because the glyph clips against it too.** The corner glyph is allowed to
// hang off the top-left corner of the card, and anything drawn past the curve would fill in the
// transparent corner and square it off — so the shape test has to be askable from outside the
// fill loop. One copy, so the silhouette a glyph is clipped to cannot drift from the silhouette
// that was drawn.
//
// The test is the ordinary one: inside a corner's radius square, a pixel belongs to the shape
// only if it falls within `radius` of that corner's centre. dx and dy are zero along the
// straight edges, which passes the whole cross of the shape. Squared distances keep it in
// integers and avoid a square root per pixel.
func insideRounded(w, h, radius, px, py int) bool {
	if px < 0 || py < 0 || px >= w || py >= h {
		return false
	}

	dx, dy := 0, 0
	switch {
	case px < radius:
		dx = radius - 1 - px
	case px >= w-radius:
		dx = px - (w - radius)
	}
	switch {
	case py < radius:
		dy = radius - 1 - py
	case py >= h-radius:
		dy = py - (h - radius)
	}
	return dx*dx+dy*dy <= radius*radius
}

// BorderBevel is how much of the border is given over to its light and its shade.
//
// **Two pixels of a six-pixel border**, so four are still the border's own colour: the border is
// what says the card's *state* — resting, selected, unaffordable — and a bevel eating the whole
// ring would leave that signal being read off a lit edge and a shadowed one that are different
// colours from each other. The depth goes on the outside, where the card meets the table.
const BorderBevel = 1

// roundedBorder draws a rounded rectangle of `border` with a `fill` one inset inside it,
// which is a border because the inner shape covers everything but the edge.
//
// Drawing it as two filled shapes rather than as a stroked outline is what keeps the
// inner corner concentric with the outer one. Stroking a path would need the same corner
// arithmetic twice and would leave the two able to disagree.
//
// **The outer BorderBevel pixels are then lit and shadowed** *(2026-08-24)*, from the same
// `systems.BevelEdges` a button's face uses, so a card and a button are lit from the same corner.
// It is done as a pass over the finished ring rather than as two more rounded rects, because the
// light does not follow the rectangle — it follows the *diagonal*, and no stack of concentric
// shapes can put light on a top-left corner and shade on the bottom-right one.
func roundedBorder(dst *image.RGBA, x, y, w, h, radius, width int, border, fill color.RGBA) {
	roundedRect(dst, x, y, w, h, radius, border)
	if width <= 0 {
		return
	}
	iw, ih := w-2*width, h-2*width
	if iw <= 0 || ih <= 0 {
		return
	}
	bevelRing(dst, x, y, w, h, radius, border)
	// The inner radius shrinks by the border width so the two curves stay parallel. A
	// constant radius would leave the border visibly thicker at the corners than along
	// the edges, which is the usual giveaway of a hand-rolled rounded rect.
	roundedRect(dst, x+width, y+width, iw, ih, radius-width, fill)
}

// bevelRing lights the outer BorderBevel pixels of a shape already filled with `border`: the
// top-left side of the card's diagonal takes the lit colour and the bottom-right side the shade.
//
// **The split is the anti-diagonal, not the four edges.** Deciding by edge — top is light, right is
// shade — has to answer for the corners, and every answer is a straight seam somewhere on the
// curve. Comparing a pixel's position across the card against its position down it puts the
// changeover exactly on the two corners the light does not reach, which is where a real bevel's
// is.
func bevelRing(dst *image.RGBA, x, y, w, h, radius int, border color.RGBA) {
	if BorderBevel <= 0 || w <= 2*BorderBevel || h <= 2*BorderBevel {
		return
	}
	light, shade := systems.BevelEdges(border)
	inner := clampRadius(w, h, radius) - BorderBevel

	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			if !insideRounded(w, h, clampRadius(w, h, radius), px, py) {
				continue
			}
			// Anything further in than the bevel is the border proper and keeps its colour.
			if insideRounded(w-2*BorderBevel, h-2*BorderBevel, inner, px-BorderBevel, py-BorderBevel) {
				continue
			}
			c := shade
			if px*h+py*w < w*h {
				c = light
			}
			dst.SetRGBA(x+px, y+py, c)
		}
	}
}

// clampRadius keeps a radius inside the rectangle it is rounding. A radius past half the
// shorter side would have opposite corners overlapping, which renders as a bite out of
// the shape rather than as an error.
func clampRadius(w, h, radius int) int {
	if radius < 0 {
		return 0
	}
	if max := min(w, h) / 2; radius > max {
		return max
	}
	return radius
}

// fillTriangleUp fills an upward-pointing isosceles triangle whose base runs from left to
// left+w, apex centred over it.
//
// Scanline, hard-edged, integer-only — the same idiom as roundedRect and for the same
// reason: a pixel is either the mark or it is not, so a test can assert a colour rather
// than a tolerance. Each row's span grows linearly from the apex to the full base width.
//
// **The apex keeps the base's parity, which is what makes it centred rather than nearly
// centred.** A row is drawn at `left + (w-span)/2`, so a span of the wrong parity puts the
// extra pixel on one side and the whole shape leans by half a pixel — visible on a 44-pixel
// card back, where the apex is most of what is read. An even base therefore comes to a
// two-pixel point and an odd one to a single pixel, rather than every apex being one pixel
// and slightly off.
func fillTriangleUp(dst *image.RGBA, left, top, w, h int, c color.RGBA) {
	if w <= 0 || h < 2 {
		return
	}
	for i := 0; i < h; i++ {
		span := w * i / (h - 1)
		if span < 1 {
			span = 1
		}
		if span%2 != w%2 {
			span++
		}
		fillRect(dst, left+(w-span)/2, top+i, span, 1, c)
	}
}

// fillTriangleDown is fillTriangleUp upside down: the base along the top edge of the box,
// the point at the bottom. It is the lower half of the diamond back, and it keeps the same
// parity rule so the two halves meet on exactly the same columns.
func fillTriangleDown(dst *image.RGBA, left, top, w, h int, c color.RGBA) {
	if w <= 0 || h < 2 {
		return
	}
	for i := 0; i < h; i++ {
		span := w * (h - 1 - i) / (h - 1)
		if span < 1 {
			span = 1
		}
		if span%2 != w%2 {
			span++
		}
		fillRect(dst, left+(w-span)/2, top+i, span, 1, c)
	}
}

// fillRect is an unrounded block, used for the cost dashes.
func fillRect(dst *image.RGBA, x, y, w, h int, c color.RGBA) {
	for py := y; py < y+h; py++ {
		for px := x; px < x+w; px++ {
			dst.SetRGBA(px, py, c)
		}
	}
}
