package cards

import (
	"image"
	"image/color"
)

// Rounded-rectangle rasterising, in plain Go.
//
// This is the piece CLAUDE.md's "reuse CreateRoundedRecMask" instruction could not
// cover, for the reason set out in the package comment: that function needs a graphics
// context and this package must not have one. What is here is deliberately the smallest
// thing that draws the same shape.
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
	rr := radius * radius

	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			// Distance from the nearest corner centre, but only when the pixel is
			// actually in a corner square. dx and dy are zero along the straight edges,
			// which makes the test below pass for the whole cross of the shape.
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
			if dx*dx+dy*dy > rr {
				continue
			}
			dst.SetRGBA(x+px, y+py, c)
		}
	}
}

// roundedBorder draws a rounded rectangle of `border` with a `fill` one inset inside it,
// which is a border because the inner shape covers everything but the edge.
//
// Drawing it as two filled shapes rather than as a stroked outline is what keeps the
// inner corner concentric with the outer one. Stroking a path would need the same corner
// arithmetic twice and would leave the two able to disagree.
func roundedBorder(dst *image.RGBA, x, y, w, h, radius, width int, border, fill color.RGBA) {
	roundedRect(dst, x, y, w, h, radius, border)
	if width <= 0 {
		return
	}
	iw, ih := w-2*width, h-2*width
	if iw <= 0 || ih <= 0 {
		return
	}
	// The inner radius shrinks by the border width so the two curves stay parallel. A
	// constant radius would leave the border visibly thicker at the corners than along
	// the edges, which is the usual giveaway of a hand-rolled rounded rect.
	roundedRect(dst, x+width, y+width, iw, ih, radius-width, fill)
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
